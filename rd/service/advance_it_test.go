// wb_advance 编排推进端点涉库集成测试（2026-09-09 拍板-B/C）：三类推进
// （①跨节点 / ②节点内 CR 轮转 / ③机械收口）+ 幂等 + 并行 join 三层
// （提交瞬间检查 / flows 行锁并发防护 / scanner 兜底巡检补推）。
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/config"
	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/handler"
	"github.com/huzilin/wbflow/rd/migration"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// advanceCtx IT 通用 ctx（context.Background 别名，长函数体少一层导入噪音）。
func advanceCtx() context.Context { return context.Background() }

// openAdvanceScratch advance IT 专用装配：每测试独立 scratch 库 + 全量端点装配
// （tasks/reviews/advance 为推进链路被测面；artifacts 挂临时 docs root 供执行步
// 登记产物），测后 DROP。装配项复用 openScratchWithDocs 的口径，不共用实例
// （该 helper 仅装配 artifact 面，task 端点 404）。
func openAdvanceScratch(t *testing.T) (*gin.Engine, *database.DB, string) {
	t.Helper()
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	cfg := &config.MySQLConfig{
		Host:               envOr("WBFLOW_MYSQL_HOST", "192.168.3.100"),
		Port:               3306,
		User:               envOr("WBFLOW_MYSQL_USER", "root"),
		Password:           os.Getenv("WBFLOW_MYSQL_PASSWORD"),
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetimeSec: 3600,
	}
	if p := os.Getenv("WBFLOW_MYSQL_PORT"); p != "" {
		port, err := strconv.ParseInt(p, 10, 64)
		assert.NilError(t, err)
		cfg.Port = port
	}
	cfg.DBName = fmt.Sprintf("wbflow_it_%d", time.Now().UnixNano())
	t.Logf("scratch 库: %s", cfg.DBName)

	handles, err := migration.Bootstrap(context.Background(), cfg)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = handles.Close()
		dropScratch(t, cfg)
	})
	report, err := migration.Run(context.Background(), handles.Gorm)
	assert.NilError(t, err)
	assert.Equal(t, report.ToVersion, migration.LatestVersion())

	docsRoot := t.TempDir()
	r := repo.NewRepo(handles.Gorm)
	gin.SetMode(gin.TestMode)
	router := handler.NewRouter(&handler.Deps{
		Logger:      zap.NewNop(),
		Health:      service.NewHealth(handles),
		Token:       itToken,
		Reqs:        service.NewRequirementService(r),
		Vers:        service.NewVersionService(r),
		Flows:       service.NewFlowService(r),
		Artifacts:   service.NewArtifactService(r, docsRoot),
		Tasks:       service.NewTaskService(r),
		Comments:    service.NewIssueService(r, domain.DefaultEscalateLoopThreshold),
		Reviews:     service.NewReviewService(r),
		Advance:     service.NewAdvanceService(r),
		Escalations: service.NewEscalationService(r),
		Inbox:       service.NewInboxService(r),
	})
	return router, handles, docsRoot
}

// advanceNode wb_advance HTTP 调用助手。
func advanceNode(t *testing.T, router *gin.Engine, reqID string, nodeID string) testResp {
	t.Helper()
	return reqJSON(t, router, http.MethodPost,
		"/api/requirements/"+reqID+"/nodes/"+nodeID+"/advance",
		map[string]any{"requirementId": reqID, "nodeId": nodeID})
}

// createFullReq 建业务 full 需求（13 实例节点），返回需求 id。
func createFullReq(t *testing.T, router *gin.Engine, title string) string {
	t.Helper()
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": title, "category": "需求任务", "flowDefsId": "business", "scope": "full",
		"createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	return item(t, resp, "requirement").str("requirementId")
}

// startFlowAt 编排接缝播种：current 指到 nodeID 且该节点激活到 phase
// （advance IT 的推演起点，与既有 IT 播种口径一致）。
func startFlowAt(t *testing.T, handles *database.DB, reqID string, nodeID string, phase string) {
	t.Helper()
	execSQL(t, handles, "UPDATE flows SET current_node_id = ? WHERE requirement_id = ?", nodeID, reqID)
	execSQL(t, handles, "UPDATE flow_nodes SET state = ?, entered_at = ? WHERE node_id = ?", phase, repo.NowMillis(), nodeID)
}

// assignAckAndRegister 开发节点执行步：派发动作 + ack + 登记期望产物
// （dev_notes/code_mr，link 类最简），返回 task id。
func assignAckAndRegister(t *testing.T, router *gin.Engine, reqID string, nodeID string, actionID string, assignee string) string {
	t.Helper()
	resp := reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": nodeID, "actionId": actionID, "assignee": assignee,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	taskID := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "ack"}).code, http.StatusOK)
	for _, ptype := range []string{"dev_notes", "code_mr"} {
		resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/"+nodeID+"/artifacts", map[string]any{
			"productType": ptype, "label": "《IT-" + ptype + "》",
			"href": "https://docs.example.local/" + nodeID + "-" + ptype, "kind": "link", "addedBy": assignee,
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	}
	return taskID
}

// TestIntegration_Advance_Phases_RealMySQL 三类推进与幂等/拦截分支：
// 未进入编排 blocked / exec 判据未达 blocked / ①跨节点推进（review 相位播种轮 0）/
// 重复推进幂等 noop / ②dev 节点内 CR 轮转 / 404。
func TestIntegration_Advance_Phases_RealMySQL(t *testing.T) {
	router, handles, _ := openAdvanceScratch(t)
	reqID := createFullReq(t, router, "推进端点需求")

	// 1. 未进入编排（state=""）→ blocked
	resp := advanceNode(t, router, reqID, "n-03")
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.boolean("blocked"), true)

	// 2. exec 相位但判据未达（无任务）→ blocked
	startFlowAt(t, handles, reqID, "n-03", "exec")
	resp = advanceNode(t, router, reqID, "n-03")
	assert.Equal(t, resp.code, http.StatusConflict, "无完成 exec 任务应 blocked: %v", resp.body)

	// 3. ①跨节点推进：n-03（bus-02 产品设计）produce 任务完成 → advance →
	//    n-04（bus-03 评审）激活 review 相位 + 评审轮 0 播种 + current 前移
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	taskID := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "ack"}).code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-03/artifacts", map[string]any{
		"productType": "product_doc", "label": "《产品设计》", "href": "https://docs.example.local/pd", "kind": "link", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	// complete 判据通过即内聚触发推进（拍板-B）：current 应已前移 n-04
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-04", "complete 内聚触发应已跨节点推进")
	assert.Equal(t, nodeCol(t, handles, "n-04", "state"), "review", "下一节点应激活 review 相位")
	// 显式 wb_advance 幂等命中（noop）：不重复播种、current 不再动
	resp = advanceNode(t, router, reqID, "n-03")
	assert.Equal(t, resp.code, http.StatusOK, "已完成推进应 200 幂等: %v", resp.body)
	assert.Equal(t, resp.str("outcome"), "noop")
	assert.Equal(t, resp.boolean("noop"), true)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-04")
	assert.Equal(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId"), int64(1), "幂等重放不重复写推进审计")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-04", "reviewResults"), int64(1), "评审轮只播种一次")
	reviewResults := nodeCol(t, handles, "n-04", "review_results")
	assert.Assert(t, strings.Contains(reviewResults, `"state":"reviewing"`), "评审轮 0 应全位 reviewing: %s", reviewResults)

	// 4. ②节点内 CR 轮转：n-09（bus-08 RD 开发）dev 完成 → advance → 本节点
	//    state=review、轮 0 播种、current 不动（重复调用幂等 noop）
	startFlowAt(t, handles, reqID, "n-09", "exec")
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-09", "actionId": "backend-dev.dev", "assignee": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	taskID = item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "ack"}).code, http.StatusOK)
	for _, ptype := range []string{"dev_notes", "code_mr"} {
		resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-09/artifacts", map[string]any{
			"productType": ptype, "label": "《RD-" + ptype + "》", "href": "https://docs.example.local/rd-" + ptype, "kind": "link", "addedBy": "rd",
		})
		assert.Equal(t, resp.code, http.StatusOK)
	}
	resp = advanceNode(t, router, reqID, "n-09")
	assert.Equal(t, resp.code, http.StatusConflict, "判据未达（未 complete）应 blocked: %v", resp.body)
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "complete"}).code, http.StatusOK)
	assert.Equal(t, nodeCol(t, handles, "n-09", "state"), "review", "dev complete 应内聚触发节点内 CR 轮转")
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-09", "轮转不动 current")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-09", "state"), int64(1), "轮转 state 审计一次（plugin）")
	resp = advanceNode(t, router, reqID, "n-09")
	assert.Equal(t, resp.code, http.StatusOK, "轮转后重复 advance 应幂等 200: %v", resp.body)
	assert.Equal(t, resp.str("outcome"), "noop", "review 未齐重放应 noop 当前态")
	assert.Equal(t, nodeCol(t, handles, "n-09", "review_results"), nodeCol(t, handles, "n-09", "review_results"))

	// 5. respond / entry / 404 拦截
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'respond' WHERE node_id = 'n-04'")
	resp = advanceNode(t, router, reqID, "n-04")
	assert.Equal(t, resp.code, http.StatusConflict, "respond 相位应 blocked")
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'entry' WHERE node_id = 'n-04'")
	resp = advanceNode(t, router, reqID, "n-04")
	assert.Equal(t, resp.code, http.StatusConflict, "entry 相位应 blocked（走 wb_decision）")
	resp = advanceNode(t, router, reqID, "n-99")
	assert.Equal(t, resp.code, http.StatusNotFound, "未知节点应 404")
}

// TestIntegration_Advance_Join_RealMySQL 并行 join（拍板-C①）：RD∥FE 并行开发组
// （n-09/n-10，bus-08/bus-09）真并行驱动——双侧同时 exec（编排播种）：
// rd 先 CR approved → 激活对侧交接不越组；fe 后 CR approved → 对侧已 complete →
// 整组推进 bus-10（QA）。
func TestIntegration_Advance_Join_RealMySQL(t *testing.T) {
	router, handles, _ := openAdvanceScratch(t)
	reqID := createFullReq(t, router, "并行 join 需求")

	// 双侧并行播种（真并行驱动：同时 exec、current 驻留 rd 侧）
	startFlowAt(t, handles, reqID, "n-09", "exec")
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'exec', entered_at = ? WHERE node_id = 'n-10'", repo.NowMillis())

	// rd 侧完成 → CR 轮转 → cr agree 机械 approved → 激活 fe 侧交接（不越组）
	rdTask := assignAckAndRegister(t, router, reqID, "n-09", "backend-dev.dev", "rd")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+rdTask, map[string]any{"event": "complete"}).code, http.StatusOK)
	assert.Equal(t, nodeCol(t, handles, "n-09", "state"), "review")
	assert.Equal(t, reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-09/review",
		map[string]any{"role": "cr", "capabilityId": "code-review", "state": "agree"}).code, http.StatusOK)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-10", "rd 完成应激活 fe 侧（待对侧交接）")
	assert.Equal(t, nodeCol(t, handles, "n-10", "state"), "exec", "fe 侧已激活（并行驱动语义）")
	assert.Assert(t, nodeCol(t, handles, "n-11", "state") == "", "整组未推进：QA 节点应未激活")

	// fe 侧完成 → CR 轮转 → cr agree 机械 approved → 对侧已 complete → 整组推进
	feTask := assignAckAndRegister(t, router, reqID, "n-10", "frontend-dev.dev", "fe")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+feTask, map[string]any{"event": "complete"}).code, http.StatusOK)
	assert.Equal(t, nodeCol(t, handles, "n-10", "state"), "review")
	assert.Equal(t, reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-10/review",
		map[string]any{"role": "cr", "capabilityId": "code-review", "state": "agree"}).code, http.StatusOK)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-11", "双侧均 complete 应整组推进 QA")
	assert.Equal(t, nodeCol(t, handles, "n-11", "state"), "exec", "QA 节点应激活 exec")

	// 整组推进后重复 advance 幂等：双方均 noop，不重复激活/前移
	for _, nid := range []string{"n-09", "n-10"} {
		resp := advanceNode(t, router, reqID, nid)
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
		assert.Equal(t, resp.str("outcome"), "noop", "%s 组已推进应 noop", nid)
	}
	assert.Equal(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId"), int64(2), "前移审计恰两次（交接 + 整组）")
}

// TestIntegration_Advance_JoinWait_RealMySQL 并行 join wait 分支（拍板-C①对称面）：
// fe 侧先 CR approved 而对侧 rd 未完成 → 本侧完成待对侧（current 不动、QA 不激活）；
// rd 后 CR approved → 其 advance 对称判定整组推进。
func TestIntegration_Advance_JoinWait_RealMySQL(t *testing.T) {
	router, handles, _ := openAdvanceScratch(t)
	reqID := createFullReq(t, router, "join 待对侧需求")

	// 双侧并行播种，current 驻留 rd 侧
	startFlowAt(t, handles, reqID, "n-09", "exec")
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'exec', entered_at = ? WHERE node_id = 'n-10'", repo.NowMillis())

	// fe 侧（n-10）先完成 → CR 轮转 → cr agree → 对侧未完成 → wait（current 不动）
	feTask := assignAckAndRegister(t, router, reqID, "n-10", "frontend-dev.dev", "fe")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+feTask, map[string]any{"event": "complete"}).code, http.StatusOK)
	assert.Equal(t, nodeCol(t, handles, "n-10", "state"), "review")
	assert.Equal(t, reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-10/review",
		map[string]any{"role": "cr", "capabilityId": "code-review", "state": "agree"}).code, http.StatusOK)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-09", "对侧未完成 current 不动")
	assert.Equal(t, nodeCol(t, handles, "n-11", "state"), "", "对侧未完成整组不推进")

	// rd 侧（n-09）后完成 → CR 轮转 → cr agree → 对侧已 complete → 整组推进
	rdTask := assignAckAndRegister(t, router, reqID, "n-09", "backend-dev.dev", "rd")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+rdTask, map[string]any{"event": "complete"}).code, http.StatusOK)
	assert.Equal(t, nodeCol(t, handles, "n-09", "state"), "review")
	assert.Equal(t, reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-09/review",
		map[string]any{"role": "cr", "capabilityId": "code-review", "state": "agree"}).code, http.StatusOK)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-11", "双侧均 complete 应整组推进")
	assert.Equal(t, nodeCol(t, handles, "n-11", "state"), "exec")
}

// TestIntegration_Advance_JoinConcurrent_RealMySQL 并发双 advance 幂等（拍板-C②）：
// 双侧均 complete 但组未推进（卡点播种），goroutine 双调用同打推进面——flows 行锁
// 串行化下仅一次推进生效（前移审计恰一次、QA 节点激活恰一次），双方均 200。
func TestIntegration_Advance_JoinConcurrent_RealMySQL(t *testing.T) {
	router, handles, _ := openAdvanceScratch(t)
	reqID := createFullReq(t, router, "并发 join 需求")

	// 播放卡点：双侧 CR 已 approved、current 驻留 n-09、QA 未激活
	startFlowAt(t, handles, reqID, "n-09", "review")
	startFlowAt(t, handles, reqID, "n-10", "review")
	execSQL(t, handles, "UPDATE flow_nodes SET last_decision = ? WHERE node_id IN ('n-09','n-10')",
		`{"verdict":"approved","reason":"cr","at":1730000000000}`)

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	outcomes := make([]string, 0, 2)
	for _, nodeID := range []string{"n-09", "n-10"} {
		wg.Add(1)
		go func(nid string) {
			defer wg.Done()
			resp := advanceNode(t, router, reqID, nid)
			assert.Equal(t, resp.code, http.StatusOK, "并发双 advance 均 200: %v", resp.body)
			mu.Lock()
			defer mu.Unlock()
			outcomes = append(outcomes, resp.str("outcome"))
		}(nodeID)
	}
	wg.Wait()

	crosses := 0
	for _, outcome := range outcomes {
		if outcome == "cross" {
			crosses++
		}
	}
	assert.Equal(t, crosses, 1, "仅一次整组推进生效（行锁串行化）: %v", outcomes)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-11")
	assert.Equal(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId"), int64(1), "前移审计恰一次")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-11", "state"), int64(1), "QA 激活审计恰一次")
}

// TestIntegration_Advance_Sweep_RealMySQL scanner 兜底巡检补推（拍板-C③）：
// 双侧均 complete 但组未推进（卡点播种）→ RunOnce 命中补推（by_who=system 审计）；
// 重复巡检幂等不再推进。
func TestIntegration_Advance_Sweep_RealMySQL(t *testing.T) {
	router, handles, _ := openAdvanceScratch(t)
	reqID := createFullReq(t, router, "巡检补推需求")

	startFlowAt(t, handles, reqID, "n-09", "review")
	startFlowAt(t, handles, reqID, "n-10", "review")
	execSQL(t, handles, "UPDATE flow_nodes SET last_decision = ? WHERE node_id IN ('n-09','n-10')",
		`{"verdict":"approved","reason":"cr","at":1730000000000}`)

	scanner := service.NewScanner(repo.NewRepo(handles.Gorm), handles, scannerTestOpts())
	report, err := scanner.RunOnce(advanceCtx())
	assert.NilError(t, err)
	assert.Equal(t, len(report.ParallelPushes), 1, "应命中并行组卡点补推: %v", report.ParallelPushes)
	assert.Equal(t, report.ParallelPushes[0], reqID)
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-11")
	assert.Equal(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId"), int64(1))
	assert.Assert(t, rowCount(t, handles,
		"SELECT COUNT(*) FROM audit_log WHERE table_name = 'flows' AND row_id = ? AND field = 'currentNodeId' AND by_who = 'system'", reqID) == 1,
		"巡检补推进审计 by_who=system")

	// 重复巡检幂等：组已推进，不再命中
	report, err = scanner.RunOnce(advanceCtx())
	assert.NilError(t, err)
	assert.Equal(t, len(report.ParallelPushes), 0, "组已推进不应再命中")
	assert.Equal(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId"), int64(1))
}
