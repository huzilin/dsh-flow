// LLM 通路执行器涉库集成测试（-L4，env 门控沿用既有 IT 口径：WBFLOW_MYSQL_IT=1
// + WBFLOW_MYSQL_* 注入，独立 scratch 库测后 DROP——禁写 wbflow 库）：
//   - produce 通路样板（bus-01 to-brd）：派发→FakeClient 产文档→入库（-K
//     kind=document）→complete 判据链→编排内聚推进 bus-02；
//   - review 通路样板（bus-03 arch.prd-review.review）：读被评审产物→findings
//     评论挂锚（-L3 range|document）→wb_review unagree→complete；
//   - 失败策略 B+：重试耗尽→任务回退（exec 重入语义）+need_intervene 告警通知
//     →停摆护栏（不重派）。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/llm"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// ctxBackground IT 通用 ctx。
func ctxBackground() context.Context { return context.Background() }

// errsTraceLLM 注入型 LLM 错误（ErrLLM 哨兵链，重试器逐次回传）。
func errsTraceLLM() error {
	return errs.Trace(fmt.Errorf("%w: 替身注入持续失败", errs.ErrLLM))
}

// llmFlowEnv LLM IT 装配：scratch 库 + 双通路样板准入集 + 即时退避（免真实等待）。
func llmFlowEnv(t *testing.T, fake *llm.FakeClient) (*gin.Engine, *database.DB, string, *service.LLMFlowService) {
	t.Helper()
	router, handles, docsRoot := openAdvanceScratch(t)
	r := repo.NewRepo(handles.Gorm)
	flowSvc := service.NewLLMFlowService(r,
		service.NewTaskService(r),
		service.NewArtifactService(r, docsRoot),
		service.NewIssueService(r, domain.DefaultEscalateLoopThreshold),
		service.NewReviewService(r),
		fake,
		service.LLMFlowOptions{
			Enabled:  true,
			RetryMax: 3,
			Actions:  []string{"to-brd", "arch.prd-review.review"},
		}).WithLogger(zap.NewNop()).WithBackoffSink(func(time.Duration) {})
	return router, handles, docsRoot, flowSvc
}

// waitForTaskStatus 轮询任务状态至期望值（后台协程异步执行，10s 截止）。
func waitForTaskStatus(t *testing.T, handles *database.DB, taskID string, want string) repo.Task {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		row := &repo.Task{}
		err := handles.Gorm.Where("task_id = ?", taskID).First(row).Error
		assert.NilError(t, err)
		if row.Status == want {
			return *row
		}
		time.Sleep(50 * time.Millisecond)
	}
	row := &repo.Task{}
	_ = handles.Gorm.Where("task_id = ?", taskID).First(row).Error
	progress := ""
	if row.LastProgress != nil {
		progress = *row.LastProgress
	}
	t.Fatalf("任务 %s 未在期限内到达状态 %s（实测 %s，last_progress=%q）", taskID, want, row.Status, progress)
	return repo.Task{}
}

// waitForCount 轮询标量计数至期望值（事务提交后异步收尾动作的断言接缝，10s 截止）。
func waitForCount(t *testing.T, handles *database.DB, want int64, label string, query string, args ...any) int64 {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var got int64
	for time.Now().Before(deadline) {
		got = 0
		assert.NilError(t, handles.Gorm.Raw(query, args...).Scan(&got).Error)
		if got == want {
			return got
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s 未达期望 %d（实测 %d）", label, want, got)
	return got
}

// instanceNodeID 模板节点 id → 实例节点 id（flow_nodes.node_id 为实例 id n-XX；
// def_node_id 才是 bus-XX 模板引用）。
func instanceNodeID(t *testing.T, handles *database.DB, reqID string, templateID string) string {
	t.Helper()
	id := ""
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT node_id FROM flow_nodes WHERE requirement_id = ? AND def_node_id = ?", reqID, templateID).Scan(&id).Error)
	assert.Assert(t, id != "", "实例节点 %s 应存在（req %s）", templateID, reqID)
	return id
}

// TestIntegration_LLMFlow_Produce_RealMySQL produce 通路样板（bus-01 to-brd）：
// 派发→产文档→入库（kind=document）→complete 判据链→编排内聚推进 bus-02；
// 同动作已有完成任务不再重派。
func TestIntegration_LLMFlow_Produce_RealMySQL(t *testing.T) {
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	fake := &llm.FakeClient{ChatText: "# 业务需求文档\n\nLLM 产出的需求正文。", Usage: llm.Usage{PromptTokens: 9, CompletionTokens: 21}}
	router, handles, docsRoot, flowSvc := llmFlowEnv(t, fake)
	ctx := context.Background()

	reqID := createFullReq(t, router, "LLM produce IT")
	bus01 := instanceNodeID(t, handles, reqID, "bus-01")
	bus02 := instanceNodeID(t, handles, reqID, "bus-02")
	startFlowAt(t, handles, reqID, bus01, "entry")

	report, err := flowSvc.SweepLLMTasks(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report.Dispatched), 1, "应派发 to-brd 任务: %v", report)
	assert.Equal(t, len(report.Continued), 0, "首次派发无恢复")

	taskID := report.Dispatched[0]
	waitForTaskStatus(t, handles, taskID, repo.TaskStatusCompleted)

	// 产物入库（-K）：brd=期望 document，kind=document，登记在 bus-01
	artCount := int64(0)
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT COUNT(*) FROM artifacts WHERE requirement_id = ? AND node_id = ? AND product_type = 'brd' AND kind = 'document'",
		reqID, bus01).Scan(&artCount).Error)
	assert.Equal(t, artCount, int64(1), "brd 产物应入库且 kind=document")
	// 产文写入 docs root
	_, statErr := os.Stat(filepath.Join(docsRoot, "llm", reqID, bus01, "brd.md"))
	assert.NilError(t, statErr, "LLM 产文应写入 docs root")
	// align 节点 user_gate=1 拍板位：complete 判据过后停等拍板（entry 相位推进由
	// wb_decision 驱动，BUG-018 真实相位链修正），拍板后 current 前移 bus-02
	current := ""
	assert.NilError(t, handles.Gorm.Raw("SELECT current_node_id FROM flows WHERE requirement_id = ?", reqID).Scan(&current).Error)
	assert.Equal(t, current, bus01, "align 拍板位应停等 wb_decision（不内聚推进）")
	decideResp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/"+bus01+"/decision",
		map[string]any{"verdict": "approved", "reason": "对齐结论用户确认"})
	assert.Equal(t, decideResp.code, 200, "拍板应通过: %v", decideResp.body)
	assert.NilError(t, handles.Gorm.Raw("SELECT current_node_id FROM flows WHERE requirement_id = ?", reqID).Scan(&current).Error)
	assert.Equal(t, current, bus02, "拍板后应推进 bus-02")
	state := ""
	assert.NilError(t, handles.Gorm.Raw("SELECT state FROM flow_nodes WHERE requirement_id = ? AND node_id = ?", reqID, bus01).Scan(&state).Error)
	assert.Equal(t, state, repo.PhaseDone, "bus-01 应离场 done")

	// 重复巡检：同动作已有完成任务 + 节点 done → 不再派发
	report2, err := flowSvc.SweepLLMTasks(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report2.Dispatched), 0, "同动作已完成不应重派: %v", report2)
}

// TestIntegration_LLMFlow_Review_RealMySQL review 通路样板（bus-03 架构评审位）：
// 读被评审产物→findings 评论挂锚（range 划词 + document 文档级）→wb_review
// unagree→complete 判据通过（评审位表态落 review_results）。
func TestIntegration_LLMFlow_Review_RealMySQL(t *testing.T) {
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	fake := &llm.FakeClient{ChatText: `{"findings":[{"desc":"第二行目标描述不明确","startLine":2,"endLine":2},{"desc":"全文缺少验收标准"}]}`}
	router, handles, docsRoot, flowSvc := llmFlowEnv(t, fake)
	ctx := context.Background()

	reqID := createFullReq(t, router, "LLM review IT")
	bus02 := instanceNodeID(t, handles, reqID, "bus-02")
	bus03 := instanceNodeID(t, handles, reqID, "bus-03")
	// 上游 bus-02 先产被评审产物（dependencyDocs 数据源=直接上游连续 produce 段）
	startFlowAt(t, handles, reqID, bus02, "exec")
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/"+bus02+"/artifacts", map[string]any{
		"productType": "product_doc", "label": "《IT-产品设计》",
		"href": writeITDoc(t, docsRoot, "llm-it-prd.md"), "kind": "document", "addedBy": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	artifactID := item(t, resp, "artifact").str("artifactId")

	// bus-03 进入 review 相位（评审位按 seed 声明：arch/rd/fe）
	startFlowAt(t, handles, reqID, bus03, "review")
	execSQL(t, handles,
		"UPDATE flow_nodes SET review_results = ? WHERE requirement_id = ? AND node_id = ?",
		`[{"role":"arch","capabilityId":"arch.prd-review","state":"reviewing","round":0,"at":0},{"role":"rd","capabilityId":"rd.prd-review","state":"reviewing","round":0,"at":0},{"role":"fe","capabilityId":"fe.prd-review","state":"reviewing","round":0,"at":0}]`,
		reqID, bus03)

	report, err := flowSvc.SweepLLMTasks(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report.Dispatched), 1, "应派发 arch 评审任务（准入集仅 arch 位）: %v", report)
	taskID := report.Dispatched[0]
	waitForTaskStatus(t, handles, taskID, repo.TaskStatusCompleted)

	// 意见评论落 issues：range 锚（startLine=2）+ document 锚各一条，挂被评审产物
	anchorRange := int64(0)
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT COUNT(*) FROM issues WHERE requirement_id = ? AND node_id = ? AND issue_type = 'issue' AND anchor_kind = 'range' AND anchor_artifact_id = ? AND anchor_start_line = 2",
		reqID, bus03, artifactID).Scan(&anchorRange).Error)
	assert.Equal(t, anchorRange, int64(1), "划词锚意见应落库")
	anchorDoc := int64(0)
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT COUNT(*) FROM issues WHERE requirement_id = ? AND node_id = ? AND issue_type = 'issue' AND anchor_kind = 'document' AND anchor_artifact_id = ?",
		reqID, bus03, artifactID).Scan(&anchorDoc).Error)
	assert.Equal(t, anchorDoc, int64(1), "文档级锚意见应落库")

	// wb_review 表态：有意见=unagree；其余评审位仍 reviewing（collect pending）
	results := ""
	assert.NilError(t, handles.Gorm.Raw("SELECT review_results FROM flow_nodes WHERE requirement_id = ? AND node_id = ?", reqID, bus03).Scan(&results).Error)
	assert.Assert(t, strings.Contains(results, `"role":"arch"`), "arch 表态应落 review_results: %s", results)
	assert.Assert(t, strings.Contains(results, `"state":"unagree"`), "有意见应 unagree: %s", results)
	state := ""
	assert.NilError(t, handles.Gorm.Raw("SELECT state FROM flow_nodes WHERE requirement_id = ? AND node_id = ?", reqID, bus03).Scan(&state).Error)
	assert.Equal(t, state, repo.PhaseReview, "评审未齐应停留 review 相位")
}

// TestIntegration_LLMFlow_FailRollback_RealMySQL 失败策略 B+：重试耗尽→任务回退
// （rejected + 节点 exec 重入 rolled_back_count++/rework-return）+ need_intervene
// 告警通知 → 停摆护栏（不重派）。
func TestIntegration_LLMFlow_FailRollback_RealMySQL(t *testing.T) {
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	fake := &llm.FakeClient{ChatErr: errsTraceLLM()}
	router, handles, _, flowSvc := llmFlowEnv(t, fake)
	ctx := context.Background()

	reqID := createFullReq(t, router, "LLM fail IT")
	bus01 := instanceNodeID(t, handles, reqID, "bus-01")
	startFlowAt(t, handles, reqID, bus01, "entry")

	report, err := flowSvc.SweepLLMTasks(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report.Dispatched), 1, "应派发 to-brd 任务: %v", report)
	taskID := report.Dispatched[0]
	waitForTaskStatus(t, handles, taskID, repo.TaskStatusRejected)

	// 节点 exec 重入语义回退
	nodeRow := struct {
		RolledBackCount int64
		ReopenSource    string
	}{}
	assert.NilError(t, handles.Gorm.Raw(
		"SELECT rolled_back_count, reopen_source FROM flow_nodes WHERE requirement_id = ? AND node_id = ?",
		reqID, bus01).Scan(&nodeRow).Error)
	assert.Equal(t, nodeRow.RolledBackCount, int64(1), "rolled_back_count 应 +1（exec 重入）")
	assert.Equal(t, nodeRow.ReopenSource, domain.ReopenSourceReworkReturn, "reopen_source=rework-return")

	// need_intervene 告警通知（未闭合；outbox 推送为提交后异步收尾，轮询断言）
	waitForCount(t, handles, 1, "need_intervene 告警通知",
		"SELECT COUNT(*) FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND anchor_kind = 'need_intervene' AND state = 'open' AND text LIKE '[llm-fail] task=%'",
		reqID)

	// 停摆护栏：未闭合 llm-fail 通知期间不重派
	report2, err := flowSvc.SweepLLMTasks(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(report2.Dispatched), 0, "停摆期不应重派: %v", report2)
}

// TestIntegration_LLMFlow_ChatClientTransport_RealMySQL 传输内核真 HTTP 演练：
// produce 通路装配 ChatClient（httptest 起 OpenAI 形状网关替身）走完整管线——
// 真演练口径（真 DSH supplier 不可用时same路径可平替，仅端点不同）。
func TestIntegration_LLMFlow_ChatClientTransport_RealMySQL(t *testing.T) {
	if os.Getenv("WBFLOW_MYSQL_IT") == "" {
		t.Skip("真实 MySQL 集成：设 WBFLOW_MYSQL_IT=1 且注入 WBFLOW_MYSQL_PASSWORD 后运行")
	}
	// OpenAI 形状网关替身（/v1/chat/completions + usage 段）
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"# 业务需求文档（真传输）\n\n正文。"}}],"usage":{"prompt_tokens":12,"completion_tokens":34,"cost_yuan":0.5}}`))
	}))
	t.Cleanup(srv.Close)

	router, handles, docsRoot := openAdvanceScratch(t)
	r := repo.NewRepo(handles.Gorm)
	client := llm.NewChatClient(srv.URL+"/v1", "sk-it", "deepseek-v4-flash", 10*time.Second, nil)
	flowSvc := service.NewLLMFlowService(r,
		service.NewTaskService(r),
		service.NewArtifactService(r, docsRoot),
		service.NewIssueService(r, domain.DefaultEscalateLoopThreshold),
		service.NewReviewService(r),
		client,
		service.LLMFlowOptions{Enabled: true, RetryMax: 2, Actions: []string{"to-brd"}},
	).WithLogger(zap.NewNop()).WithBackoffSink(func(time.Duration) {})

	reqID := createFullReq(t, router, "LLM chat transport IT")
	bus01 := instanceNodeID(t, handles, reqID, "bus-01")
	startFlowAt(t, handles, reqID, bus01, "entry")

	report, err := flowSvc.SweepLLMTasks(ctxBackground())
	assert.NilError(t, err)
	assert.Equal(t, len(report.Dispatched), 1, "应派发 to-brd: %v", report)
	waitForTaskStatus(t, handles, report.Dispatched[0], repo.TaskStatusCompleted)

	content, err := os.ReadFile(filepath.Join(docsRoot, "llm", reqID, bus01, "brd.md"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(content), "真传输"), "产文应为网关回传正文: %q", string(content))
}
