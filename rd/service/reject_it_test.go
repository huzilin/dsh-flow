// T9 打回链路涉库集成测试（AC §4.1 打回链路）：评审 unagree→to-fix → rejected →
// exec 重入（revise）→ 重新提测 → QA retest 复验 → resolved 全链路 + pending_scope
// 重定范围重投影（full→rd 裁剪/上游保护）+ 回退幂等（同轮重放 no-op）+ 并发回退
// 行锁仅一次生效。
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库，密码仅经 env 零落地。
//
// 实例节点 id 与需求序对应（T4 取号规则）：第 k 条 business×full 需求节点 =
// n-(13(k-1)+1)..n-13k ↔ bus-00..bus-12（bus-04 RD 技术设计=第 5 位、bus-05 RD 技术
// 评审=第 6 位、bus-10 QA 验收=第 11 位）。exec→review 提评迁移属编排接缝，scratch
// 库内 SQL 模拟（与 T4/T8 IT 同口径）；重新提评播种轮=评审节点 rolled_back_count
// （data-tools-final「collect 读取 round == rolled_back_count」）。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// openRejectScratch 全量路由装配（含 artifacts/tasks/comments/reviews——打回链路
// 横跨五端点）+ 独立 scratch 库，测后 DROP。自包含 helper（不动既有 openScratch*
// 签名，便于并行工单三方合并）。
func openRejectScratch(t *testing.T) (*gin.Engine, *database.DB, string) {
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

	r := repo.NewRepo(handles.Gorm)
	docsRoot := t.TempDir()
	gin.SetMode(gin.TestMode)
	router := handler.NewRouter(&handler.Deps{
		Logger:    zap.NewNop(),
		Health:    service.NewHealth(handles),
		Token:     itToken,
		Reqs:      service.NewRequirementService(r),
		Vers:      service.NewVersionService(r),
		Flows:     service.NewFlowService(r),
		Artifacts: service.NewArtifactService(r, docsRoot),
		Tasks:     service.NewTaskService(r),
		Comments:  service.NewIssueService(r, domain.DefaultEscalateLoopThreshold),
		Reviews:   service.NewReviewService(r),
	})
	return router, handles, docsRoot
}

// seedReviewRoundAt播种评审相位（重入轮=round；与 advanceApprovedNode 激活口径一致）
// 并把 current_node_id 指到该节点——重新提测的编排接缝模拟。
func seedReviewRoundAt(t *testing.T, handles *database.DB, reqID string, nodeId string, round int64) {
	t.Helper()
	seeded, err := domain.InitReviewRound(nodeCol(t, handles, nodeId, "reviewers"), round, repo.NowMillis())
	assert.NilError(t, err)
	seededJSON, err := utils.MarshalJSONText(seeded)
	assert.NilError(t, err)
	execSQL(t, handles, "UPDATE flows SET current_node_id = ? WHERE requirement_id = ?", nodeId, reqID)
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'review', review_results = ? WHERE node_id = ?",
		seededJSON, nodeId)
}

// nodeIntCol 读实例节点整型单列（col 限内部字面量）。
func nodeIntCol(t *testing.T, handles *database.DB, nodeId string, col string) int64 {
	t.Helper()
	var out int64
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT "+col+" FROM flow_nodes WHERE node_id = ?", nodeId).Scan(&out).Error
	assert.NilError(t, err)
	return out
}

// rowCount 通用计数断言数据源。
func rowCount(t *testing.T, handles *database.DB, query string, args ...any) int64 {
	t.Helper()
	count := int64(0)
	err := handles.Gorm.WithContext(context.Background()).Raw(query, args...).Scan(&count).Error
	assert.NilError(t, err)
	return count
}

// TestIntegration_RejectRollbackLoop_RealMySQL 打回链路全链（AC）：
// unagree→clarify→to-fix→rejected→exec 重入（revise 多配置选择）→reproject 清
// pending_scope→重新提测（round=rolled_back_count）→approved（feature_branch 落值）
// →下游重入配置（revise）→QA retest（仅 full 节点，rbc≥1 选择）→issue resolved。
func TestIntegration_RejectRollbackLoop_RealMySQL(t *testing.T) {
	router, handles, docsRoot := openRejectScratch(t)
	review := func(reqID string, nodeId string, role string, capabilityID string, state string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": state})
	}
	positions := []struct{ role, capabilityID string }{
		{"arch", "arch.rd-design-review"}, {"fe", "fe.rd-design-review"},
		{"qa", "qa.rd-design-review"}, {"pm", "pm.rd-design-review"},
	}

	reqID := setupReviewNode(t, router, handles, "1", "n-06") // bus-05 RD 技术评审 ug=1

	// —— 评审 unagree → clarify（owner 澄清循环，T8 语义回归）——
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-06", "author": "arch", "roleId": "arch", "text": "技术设计未覆盖回滚场景", "assignee": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issueID := item(t, resp, "issue").str("issueId")

	assert.Equal(t, review(reqID, "n-06", "arch", "arch.rd-design-review", "unagree").code, http.StatusOK)
	for _, p := range positions[1:] {
		assert.Equal(t, review(reqID, "n-06", p.role, p.capabilityID, "agree").code, http.StatusOK)
	}
	assert.Equal(t, nodeCol(t, handles, "n-06", "state"), "respond", "clarify 激活")

	// owner 澄清：open→pending-confirm（回复）→ complete 通过 → review 重入轮 1
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-06", "actionId": "backend-design.clarify", "assignee": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	clarifyTask := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+clarifyTask, map[string]any{"event": "ack"}).code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+clarifyTask, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusConflict, "open issue 未清，判据拦截")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "rd", "roleId": "rd", "text": "回滚场景已在 4.3 节补充说明",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+clarifyTask, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, nodeCol(t, handles, "n-06", "state"), "review", "澄清后 review 重入")

	// —— 确认真问题（to-fix）→ 全 agree → User 拍板 rejected ——
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/resolve", map[string]any{
		"commentId": issueID, "path": "fix", "operator": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "to-fix", "确认真问题")
	assert.Equal(t, review(reqID, "n-06", "arch", "arch.rd-design-review", "agree").code, http.StatusOK)
	for _, p := range positions[1:] {
		assert.Equal(t, review(reqID, "n-06", p.role, p.capabilityID, "agree").code, http.StatusOK)
	}
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-06/decision",
		map[string]any{"verdict": "rejected", "reason": "回滚场景设计有误，打回重做"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("verdict"), "rejected")
	assert.Equal(t, resp.str("rollbackNodeId"), "n-05", "回退目标=上一渲染节点 bus-04")

	// 回退重入落值断言：目标 exec 重入 + reopen_source + rolled_back_count + pending_scope
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-05", "current_node_id 回退")
	assert.Equal(t, nodeCol(t, handles, "n-05", "state"), "exec")
	assert.Equal(t, nodeIntCol(t, handles, "n-05", "rolled_back_count"), int64(1))
	assert.Equal(t, nodeCol(t, handles, "n-05", "reopen_source"), "downstream-reject")
	assert.Equal(t, nodeCol(t, handles, "n-05", "pending_scope"), "full", "待重定标记=当前 scope")
	assert.Equal(t, nodeCol(t, handles, "n-05", "review_results"), "[]", "目标门禁态清零")
	assert.Equal(t, nodeCol(t, handles, "n-05", "last_decision"), "", "目标 last_decision 清零待重新 collect")
	assert.Assert(t, nodeCol(t, handles, "n-05", "entered_at") != "", "重入时间落值")
	// 评审节点：打回计数=round+1（round 1=澄清重入轮）、相位/评审轮/结论清零（等待重新提评）
	assert.Equal(t, nodeCol(t, handles, "n-06", "state"), "")
	assert.Equal(t, nodeIntCol(t, handles, "n-06", "rolled_back_count"), int64(2), "打回计数=round+1")
	assert.Equal(t, nodeCol(t, handles, "n-06", "review_results"), "[]")
	assert.Equal(t, nodeCol(t, handles, "n-06", "last_decision"), "", "结论清零（新轮重新拍板）")
	// 下游联动重置：upstream-reject + rolled_back_count++（QA retest 选择依据）
	for _, downstream := range []string{"n-07", "n-08", "n-09", "n-10", "n-11", "n-12", "n-13"} {
		assert.Equal(t, nodeCol(t, handles, downstream, "reopen_source"), "upstream-reject", downstream)
		assert.Equal(t, nodeIntCol(t, handles, downstream, "rolled_back_count"), int64(1), downstream)
		assert.Equal(t, nodeCol(t, handles, downstream, "state"), "", downstream)
	}

	// 回退重复上报幂等无害：重复拍板被门禁拦截（评审轮已清）+ 带外重放 no-op 不改库
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-06/decision",
		map[string]any{"verdict": "rejected", "reason": "重复打回"})
	assert.Equal(t, resp.code, http.StatusConflict, "评审轮已清，重复拍板不触发二次回退")
	auditBefore := auditFieldCount(t, handles, "flow_nodes", "n-05", "rolledBackCount")
	reviewSvc := service.NewReviewService(repo.NewRepo(handles.Gorm))
	targetID, err := reviewSvc.RollbackForReject(context.Background(), reqID, "n-06")
	assert.NilError(t, err)
	assert.Equal(t, targetID, "", "同轮重放 no-op")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-05", "rolledBackCount"), auditBefore, "重放零副作用")

	// —— exec 重入（revise 多配置选择）：my_task 按 rolled_back_count 出返工集 ——
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "rd", "nodeId": "n-05"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("currentPhase"), "exec")
	nodeItem := item(t, resp, "node")
	assert.Equal(t, nodeItem.number(t, "rolledBackCount"), int64(1))
	assert.Equal(t, nodeItem.str("reopenSource"), "downstream-reject")
	actions := arr(t, resp, "activeActions")
	actionIDs := make([]string, 0, len(actions))
	for _, a := range actions {
		actionIDs = append(actionIDs, a.str("actionId"))
	}
	assert.Assert(t, contains(actionIDs, "revise"), "重入出返工配置: %v", actionIDs)
	assert.Assert(t, !contains(actionIDs, "produce"), "首产配置被剔除: %v", actionIDs)

	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-05", "actionId": "backend-design.revise", "assignee": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	reviseTask := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+reviseTask, map[string]any{"event": "ack"}).code, http.StatusOK)
	assert.NilError(t, os.MkdirAll(filepath.Join(docsRoot, "docs/v1/rd"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(docsRoot, "docs/v1/rd/design-v2.md"), []byte("# RD 技术设计（修订）\n"), 0o644))
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-05/artifacts", map[string]any{
		"productType": "rd_design", "label": "《RD 技术设计（修订）》", "href": "docs/v1/rd/design-v2.md", "addedBy": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+reviseTask, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "task").str("status"), "completed", "revise 完成判据通过")

	// —— pending_scope 重定范围：同范围重投影清标记继续 ——
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/reproject", map[string]any{"scope": ""})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.boolean("noop"), false)
	assert.Equal(t, resp.str("scope"), "full")
	assert.Equal(t, resp.str("nodeId"), "n-05")
	assert.Equal(t, resp.number(t, "reprojectedCount"), int64(9), "下游 n-05..n-13 全量重排")
	assert.Equal(t, nodeCol(t, handles, "n-05", "pending_scope"), "", "标记清除")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/reproject", map[string]any{"scope": ""})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("noop"), true, "重复重投影幂等 no-op")

	// —— 重新提测：播种新评审轮 = rolled_back_count（编排接缝模拟）——
	seedReviewRoundAt(t, handles, reqID, "n-06", 2)
	for _, p := range positions {
		resp = review(reqID, "n-06", p.role, p.capabilityID, "agree")
		assert.Equal(t, resp.code, http.StatusOK)
		assert.Equal(t, resp.number(t, "round"), int64(2), "新轮与打回计数对齐")
	}
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-06/decision",
		map[string]any{"verdict": "approved", "reason": "修订后通过", "branch": "feat/rd-rework"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("nextNodeId"), "n-07")
	assert.Equal(t, nodeCol(t, handles, "n-06", "feature_branch"), "feat/rd-rework", "评审敲定分支落值")

	// 下游重入配置选择：bus-06 已被 upstream-reject 波及（rbc=1）→ my_task 出 revise
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "fe", "nodeId": "n-07"})
	assert.Equal(t, resp.code, http.StatusOK)
	actions = arr(t, resp, "activeActions")
	actionIDs = actionIDs[:0]
	for _, a := range actions {
		actionIDs = append(actionIDs, a.str("actionId"))
	}
	assert.Assert(t, contains(actionIDs, "revise"), "下游重入出返工配置: %v", actionIDs)
	assert.Assert(t, !contains(actionIDs, "produce"), "下游首产配置被剔除: %v", actionIDs)

	// —— QA retest（仅 full scope 节点；rbc≥1 选 retest）——
	now := repo.NowMillis()
	execSQL(t, handles, "UPDATE flows SET current_node_id = 'n-11' WHERE requirement_id = ?", reqID)
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'exec', entered_at = ? WHERE node_id = 'n-11'", now)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "qa", "nodeId": "n-11"})
	assert.Equal(t, resp.code, http.StatusOK)
	actions = arr(t, resp, "activeActions")
	assert.Equal(t, len(actions), 1, "QA 重入仅出 retest: %v", actions)
	assert.Equal(t, actions[0].str("actionId"), "retest")
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-11", "actionId": "acceptance-test.retest", "assignee": "qa",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	retestTask := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+retestTask, map[string]any{"event": "ack"}).code, http.StatusOK)
	assert.NilError(t, os.MkdirAll(filepath.Join(docsRoot, "docs/v1/qa"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(docsRoot, "docs/v1/qa/test-v2.md"), []byte("# 复验记录\n"), 0o644))
	assert.NilError(t, os.WriteFile(filepath.Join(docsRoot, "docs/v1/qa/defect-v2.md"), []byte("# 缺陷文档\n"), 0o644))
	for _, art := range []struct{ ptype, label, href string }{
		{"test_result", "《复验记录》", "docs/v1/qa/test-v2.md"},
		{"defect", "《缺陷文档（复验更新）》", "docs/v1/qa/defect-v2.md"},
	} {
		resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-11/artifacts", map[string]any{
			"productType": art.ptype, "label": art.label, "href": art.href, "addedBy": "qa",
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	}
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+retestTask, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "task").str("status"), "completed", "retest 完成判据通过")

	// —— 复验通过 → issue resolved（消费 T7 confirm 路径：to-fix→fixed→resolved）——
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "rd", "roleId": "rd", "text": "回滚场景已修复并复验通过",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/confirm", map[string]any{
		"commentId": issueID, "operator": "arch",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "resolved", "复验确认闭环")
	assert.Equal(t, rowCount(t, handles,
		"SELECT COUNT(*) FROM issues WHERE issue_id = ? AND state = 'resolved' AND resolved_by = 'arch' AND resolved_at IS NOT NULL",
		issueID), int64(1))

	// QA retest 仅 full scope 节点：rd 需求不渲染 bus-10（结构上无 retest 触发面）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "RD 单泳道需求", "category": "需求任务", "scope": "rd", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	rdReqID := item(t, resp, "requirement").str("requirementId")
	assert.Equal(t, rowCount(t, handles,
		"SELECT COUNT(*) FROM flow_nodes WHERE requirement_id = ? AND def_node_id = 'bus-10'", rdReqID), int64(0),
		"rd 需求无 QA 节点")
}

// TestIntegration_ReprojectScopeChange_RealMySQL 重定范围重投影：full→rd 裁剪下游
// （fe/QA 节点删行不占位、重排连续）+ 上游保护（裁剪回退目标/上游 → blocked）+
// 重复重投影幂等。
func TestIntegration_ReprojectScopeChange_RealMySQL(t *testing.T) {
	router, handles, _ := openRejectScratch(t)
	review := func(reqID string, nodeId string, role string, capabilityID string, state string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": state})
	}

	reqID := setupReviewNode(t, router, handles, "1", "n-06")
	for _, p := range []struct{ role, capabilityID string }{
		{"arch", "arch.rd-design-review"}, {"fe", "fe.rd-design-review"},
		{"qa", "qa.rd-design-review"}, {"pm", "pm.rd-design-review"},
	} {
		assert.Equal(t, review(reqID, "n-06", p.role, p.capabilityID, "agree").code, http.StatusOK)
	}
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-06/decision",
		map[string]any{"verdict": "rejected", "reason": "打回重定范围"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("rollbackNodeId"), "n-05")

	// 上游保护：fe 范围裁剪回退目标 bus-04（rd|full）→ blocked
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/reproject", map[string]any{"scope": "fe"})
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")
	assert.Equal(t, resp.boolean("blocked"), true)

	// scope 越出可选集 → 400
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/reproject", map[string]any{"scope": "be"})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	assert.Equal(t, resp.str("code"), "INVALID_ARGUMENT")

	// full→rd：下游裁剪（bus-06/07/09/10 删行）+ 重排（bus-08:9→7、bus-11:12→8、bus-12:13→9）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/reproject", map[string]any{"scope": "rd"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("scope"), "rd")
	assert.Equal(t, resp.str("nodeId"), "n-05")
	assert.Equal(t, flowCol(t, handles, reqID, "scope"), "rd", "flows.scope 落值")
	assert.Equal(t, nodeCol(t, handles, "n-05", "pending_scope"), "", "标记清除")
	for _, trimmed := range []string{"n-07", "n-08", "n-10", "n-11"} {
		assert.Equal(t, nodeCol(t, handles, trimmed, "node_id"), "", "被裁节点删行: "+trimmed)
	}
	assert.Equal(t, nodeIntCol(t, handles, "n-09", "node_order"), int64(7), "bus-08 重排")
	assert.Equal(t, nodeIntCol(t, handles, "n-12", "node_order"), int64(8), "bus-11 重排")
	assert.Equal(t, nodeIntCol(t, handles, "n-13", "node_order"), int64(9), "bus-12 重排")
	for i, upstream := range []string{"n-01", "n-02", "n-03", "n-04", "n-05"} {
		assert.Equal(t, nodeIntCol(t, handles, upstream, "node_order"), int64(i+1), "上游前缀不动: "+upstream)
	}

	// 重复重投影：无 pending_scope → 幂等 no-op
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/reproject", map[string]any{"scope": "fe"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("noop"), true, "无待重定标记不生效")
}

// TestIntegration_RollbackConcurrent_RealMySQL 并发回退仅一次生效（真库行锁验证）：
// 两路 RollbackForReject 并发——行锁串行化下恰一次应用（rolled_back_count 单次递增、
// reopen_source 审计一次），另一路护栏 no-op 零副作用。
func TestIntegration_RollbackConcurrent_RealMySQL(t *testing.T) {
	router, handles, _ := openRejectScratch(t)
	review := func(reqID string, nodeId string, role string, capabilityID string, state string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": state})
	}

	reqID := setupReviewNode(t, router, handles, "1", "n-06")
	for _, p := range []struct{ role, capabilityID string }{
		{"arch", "arch.rd-design-review"}, {"fe", "fe.rd-design-review"},
		{"qa", "qa.rd-design-review"}, {"pm", "pm.rd-design-review"},
	} {
		assert.Equal(t, review(reqID, "n-06", p.role, p.capabilityID, "agree").code, http.StatusOK)
	}
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-06/decision",
		map[string]any{"verdict": "rejected", "reason": "首次打回"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("rollbackNodeId"), "n-05")

	// 复位到「待回退」基线，构造并发竞争窗口
	seedReviewRoundAt(t, handles, reqID, "n-06", 0)
	execSQL(t, handles, "UPDATE flow_nodes SET rolled_back_count = 0 WHERE node_id = 'n-06'")
	execSQL(t, handles, "UPDATE flow_nodes SET rolled_back_count = 0, state = '', reopen_source = '', pending_scope = NULL WHERE node_id = 'n-05'")
	execSQL(t, handles, "UPDATE flow_nodes SET rolled_back_count = 0, state = '', reopen_source = '' WHERE node_id IN ('n-07','n-08','n-09','n-10','n-11','n-12','n-13')")
	execSQL(t, handles, "DELETE FROM audit_log WHERE table_name = 'flow_nodes' AND row_id IN ('n-05','n-06') AND action = 'UPDATE'")

	reviewSvc := service.NewReviewService(repo.NewRepo(handles.Gorm))
	const racers = 2
	type result struct {
		target string
		err    error
	}
	results := make(chan result, racers)
	var wg sync.WaitGroup
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target, err := reviewSvc.RollbackForReject(context.Background(), reqID, "n-06")
			results <- result{target: target, err: err}
		}()
	}
	wg.Wait()
	close(results)

	applied := 0
	for r := range results {
		assert.NilError(t, r.err)
		if r.target == "n-05" {
			applied++
		}
	}
	assert.Equal(t, applied, 1, "并发回退恰一次生效")
	assert.Equal(t, nodeIntCol(t, handles, "n-05", "rolled_back_count"), int64(1), "计数单次递增")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-05", "reopenSource"), int64(1), "重开来源审计一次")
}

// contains 字符串切片包含判定。
func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
