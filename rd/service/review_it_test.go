// T8 涉库集成测试（AC）：per-agent 表态（单字段读写一致/同轮覆写取最后）+
// collect 聚合判定三段（pending/clarify/approved）+ 并发双 collect 行锁仅一次推进 +
// User 拍板（user_gate 校验/幂等/改判冲突）+ 澄清循环消费 T7 issue 链（owner respond
// 后 review 重入新轮，my_task 摘要 round 轮转回归）。
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库，密码仅经 env 零落地。
//
// 实例节点 id 与需求序对应（T4 取号规则）：第 k 条 business×full 需求节点 =
// n-(13(k-1)+1)..n-13k ↔ bus-00..bus-12（如第 2 条需求的 bus-03 = n-17、bus-11 = n-25）。
package service_test

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/database"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// setupReviewNode 建需求并把 nodeId 置为当前节点 + review 相位（exec→review 提评
// 迁移属编排接缝，scratch 库内 SQL 模拟——T4 IT 同口径；nodeId 按需求序传入）。
// 相位激活同时播种评审轮（全位 reviewing 轮 0——与 advanceApprovedNode 激活口径一致）。
func setupReviewNode(t *testing.T, router *gin.Engine, handles *database.DB, label string, nodeId string) string {
	t.Helper()
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "评审需求" + label, "category": "需求任务", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	reqID := item(t, resp, "requirement").str("requirementId")
	seeded, err := domain.InitReviewRound(nodeCol(t, handles, nodeId, "reviewers"), 0, repo.NowMillis())
	assert.NilError(t, err)
	seededJSON, err := utils.MarshalJSONText(seeded)
	assert.NilError(t, err)
	execSQL(t, handles, "UPDATE flows SET current_node_id = ? WHERE requirement_id = ?", nodeId, reqID)
	execSQL(t, handles, "UPDATE flow_nodes SET state = 'review', review_results = ? WHERE requirement_id = ? AND node_id = ?",
		seededJSON, reqID, nodeId)
	return reqID
}

// seedUpstreamArtifact 评审链夹具：指定节点登记产物行（BUG-011/013 评审上下文
// 断言数据源——评审评论锚产物强制的锚定目标）。同 id 已存在时跳过（幂等，
// 同需求多 issue 夹具共用）。
func seedUpstreamArtifact(t *testing.T, handles *database.DB, reqID string, artifactID string, nodeID string) {
	t.Helper()
	var count int64
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM artifacts WHERE artifact_id = ?", artifactID).Scan(&count).Error)
	if count > 0 {
		return
	}
	art := &repo.Artifact{
		ArtifactID: artifactID, RequirementID: reqID, NodeID: nodeID,
		ProductType: "product_doc", Label: "《产品设计文档》", Href: "docs/product.md",
		Kind: "document", AddedBy: "pm",
	}
	assert.NilError(t, handles.Gorm.Create(art).Error)
}

// nodeCol scratch 库读实例节点单列文本（NULL=空串；col 限内部字面量，不收外部输入）。
func nodeCol(t *testing.T, handles *database.DB, nodeId string, col string) string {
	t.Helper()
	var out sql.NullString
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT "+col+" FROM flow_nodes WHERE node_id = ?", nodeId).Scan(&out).Error
	assert.NilError(t, err)
	return out.String
}

// flowCol scratch 库读流程实例单列文本（NULL=空串）。
func flowCol(t *testing.T, handles *database.DB, reqID string, col string) string {
	t.Helper()
	var out sql.NullString
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT "+col+" FROM flows WHERE requirement_id = ?", reqID).Scan(&out).Error
	assert.NilError(t, err)
	return out.String
}

// taskCol scratch 库读任务单列文本。
func taskCol(t *testing.T, handles *database.DB, taskID string, col string) string {
	t.Helper()
	var out sql.NullString
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT "+col+" FROM tasks WHERE task_id = ?", taskID).Scan(&out).Error
	assert.NilError(t, err)
	return out.String
}

// auditFieldCount 按字段计审计行（action=UPDATE 精细化断言）。
func auditFieldCount(t *testing.T, handles *database.DB, table string, rowID string, field string) int64 {
	t.Helper()
	count := int64(0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT COUNT(*) FROM audit_log WHERE table_name = ? AND row_id = ? AND action = 'UPDATE' AND field = ?",
			table, rowID, field).Scan(&count).Error
	assert.NilError(t, err)
	return count
}

// TestIntegration_ReviewCollectDecision_RealMySQL 表态→collect→agreed→拍板→推进全链：
// user_gate=1 节点 agreed 停留等拍板；拍板 approved 前移 + 下一节点激活相位 + entered_at；
// 重复拍板幂等 / 改判冲突；user_gate=0 节点 decision blocked；gate 型节点直拍（空转）+
// entry 激活驻留；rejected 落 verdict 不推进（T9 重入接缝）。
func TestIntegration_ReviewCollectDecision_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	review := func(reqID string, nodeId string, role string, capabilityID string, state string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": state})
	}
	decide := func(reqID string, nodeId string, body map[string]any) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/decision", body)
	}

	reqID := setupReviewNode(t, router, handles, "1", "n-04") // bus-03 评审位 arch/rd/fe，ug=1

	// 拦截分支：state 越集 400 / 非评审位 409 blocked / 能力位不符 409 / 非 review 相位 409
	resp := review(reqID, "n-04", "arch", "arch.prd-review", "reviewing")
	assert.Equal(t, resp.code, http.StatusBadRequest, "reviewing 为系统态不接受上报")
	assert.Equal(t, resp.str("code"), "INVALID_ARGUMENT")
	resp = review(reqID, "n-04", "cr", "code-review", "agree")
	assert.Equal(t, resp.code, http.StatusConflict, "cr 非该节点评审位")
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")
	assert.Equal(t, resp.boolean("blocked"), true)
	resp = review(reqID, "n-04", "arch", "rd.prd-review", "agree")
	assert.Equal(t, resp.code, http.StatusConflict, "arch 角色挂错评审能力")
	resp = review(reqID, "n-03", "pm", "product-design", "agree")
	assert.Equal(t, resp.code, http.StatusConflict, "exec 相位节点不接受表态")

	// per-agent 表态：1/3 → 2/3 pending（review_results 单字段累积，读写一致）
	resp = review(reqID, "n-04", "arch", "arch.prd-review", "agree")
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("collectOutcome"), "pending")
	assert.Equal(t, resp.number(t, "round"), int64(0))
	resp = review(reqID, "n-04", "rd", "rd.prd-review", "agree")
	assert.Equal(t, resp.str("collectOutcome"), "pending")
	// 同轮同角色覆写取最后：arch 重报条目不增（3 位置条目恒在，2 agree + 1 reviewing）
	resp = review(reqID, "n-04", "arch", "arch.prd-review", "agree")
	assert.Equal(t, resp.code, http.StatusOK)
	results := nodeCol(t, handles, "n-04", "review_results")
	assert.Equal(t, strings.Count(results, `"state":"agree"`), 2, "两票 agree: %s", results)
	assert.Equal(t, strings.Count(results, `"role"`), 3, "评审位条目不增不删: %s", results)

	// my_task 摘要回归：agree=2 reviewing=1（T4 口径，round 轮转前）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "fe", "nodeId": "n-04"})
	assert.Equal(t, resp.code, http.StatusOK)
	summary := item(t, resp, "reviewSummary")
	assert.Equal(t, summary.number(t, "agree"), int64(2))
	assert.Equal(t, summary.number(t, "reviewing"), int64(1))

	// 全 agree → user_gate=1：agreed 停留（不写 last_decision、不推进）
	resp = review(reqID, "n-04", "fe", "fe.prd-review", "agree")
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("collectOutcome"), "approved")
	assert.Equal(t, resp.str("gateState"), "agreed", "user_gate=1 唤醒 User 终审")
	assert.Equal(t, resp.str("nextNodeId"), "")
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-04", "agreed 不前移")
	assert.Equal(t, nodeCol(t, handles, "n-04", "last_decision"), "", "last_decision 留待拍板")

	// 非 user_gate 位拍板拦截（bus-04 user_gate=0）
	resp = decide(reqID, "n-05", map[string]any{"verdict": "approved", "reason": "越权拍板"})
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")
	assert.Equal(t, resp.boolean("blocked"), true)

	// User 拍板 approved：gate 落定 + 前移 n-05 + 下一节点激活（produce 无 entry 挂载空转 exec）
	resp = decide(reqID, "n-04", map[string]any{"verdict": "approved", "reason": "评审通过", "branch": "feat/prd"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("verdict"), "approved")
	assert.Equal(t, resp.str("nextNodeId"), "n-05")
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-05", "current_node_id 前移")
	assert.Equal(t, nodeCol(t, handles, "n-05", "state"), "exec", "空转直过 entry")
	assert.Assert(t, nodeCol(t, handles, "n-05", "entered_at") != "", "entered_at 落值")
	lastDecision := nodeCol(t, handles, "n-04", "last_decision")
	assert.Assert(t, strings.Contains(lastDecision, `"verdict":"approved"`), "结论协议落库: %s", lastDecision)
	assert.Assert(t, strings.Contains(lastDecision, `"branch":"feat/prd"`), "branch 落库")

	// 重复拍板幂等 no-op（不重复推进、不改库）
	resp = decide(reqID, "n-04", map[string]any{"verdict": "approved", "reason": "重复"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("noop"), true)
	assert.Equal(t, resp.str("verdict"), "approved")
	assert.Equal(t, nodeCol(t, handles, "n-04", "last_decision"), lastDecision, "幂等不改库")

	// 改判冲突
	resp = decide(reqID, "n-04", map[string]any{"verdict": "rejected", "reason": "改判"})
	assert.Equal(t, resp.code, http.StatusConflict)
	assert.Equal(t, resp.str("code"), "IDEMPOTENT_CONFLICT")

	// 审计：表态+落定留痕 / 激活 state+enteredAt / 前移审计一次
	assert.Assert(t, auditCount(t, handles, "flow_nodes", "n-04", "UPDATE") >= 4, "表态与落定审计留痕")
	assert.Assert(t, auditFieldCount(t, handles, "flow_nodes", "n-05", "state") == 1, "激活 state 审计一次")
	assert.Assert(t, auditFieldCount(t, handles, "flow_nodes", "n-05", "enteredAt") == 1, "enteredAt 审计一次")
	assert.Assert(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId") == 1, "前移审计一次")
	assert.Assert(t, auditFieldCount(t, handles, "flow_nodes", "n-04", "lastDecision") == 1, "拍板审计一次")

	// gate 型节点（第 2 条需求 bus-11=n-25）无评审位直拍（空转 approved）+ 门禁类 entry 驻留
	req2 := setupReviewNode(t, router, handles, "2", "n-25")
	resp = decide(req2, "n-25", map[string]any{"verdict": "approved", "reason": "用户验收通过"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("nextNodeId"), "n-26")
	assert.Equal(t, nodeCol(t, handles, "n-26", "state"), "entry", "gate 型下一节点驻留 entry")
	assert.Assert(t, nodeCol(t, handles, "n-26", "entered_at") != "", "entered_at 落值")

	// rejected 拍板（第 3 条需求 bus-03=n-30）：verdict 落定 + T9 回退重入同事务生效
	// （目标 bus-02=n-29 exec 重入；评审节点清相位待重新提评；全链路细粒度断言见
	// reject_it_test.go）
	req3 := setupReviewNode(t, router, handles, "3", "n-30")
	for _, position := range []struct{ role, capabilityID string }{
		{"arch", "arch.prd-review"}, {"rd", "rd.prd-review"}, {"fe", "fe.prd-review"},
	} {
		resp = review(req3, "n-30", position.role, position.capabilityID, "agree")
		assert.Equal(t, resp.code, http.StatusOK)
	}
	resp = decide(req3, "n-30", map[string]any{"verdict": "rejected", "reason": "设计有误，打回"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("verdict"), "rejected")
	assert.Equal(t, resp.str("nextNodeId"), "", "rejected 不推进")
	assert.Equal(t, resp.str("rollbackNodeId"), "n-29", "回退目标=上一渲染节点")
	assert.Equal(t, flowCol(t, handles, req3, "current_node_id"), "n-29", "current_node_id 回退")
	assert.Equal(t, nodeCol(t, handles, "n-30", "state"), "", "评审节点清相位待重新提评")
	assert.Equal(t, nodeCol(t, handles, "n-30", "rolled_back_count"), "1", "打回计数=round+1")
	assert.Equal(t, nodeCol(t, handles, "n-29", "state"), "exec", "回退目标 exec 重入")
	assert.Equal(t, nodeCol(t, handles, "n-29", "reopen_source"), "downstream-reject", "目标重开来源")
	assert.Assert(t, nodeCol(t, handles, "n-29", "pending_scope") != "", "pending_scope 待重定标记")
	assert.Assert(t, auditFieldCount(t, handles, "flow_nodes", "n-30", "lastDecision") == 1, "rejected 结论审计留痕")
}

// TestIntegration_ReviewClarifyLoop_RealMySQL 澄清循环消费 T7 issue 链最小闭环：
// unagree → clarify（节点 respond，owner my_task 拿 clarify 上下文与 issue 清单）→
// respond complete 判据拦截（open issue 未清）→ wb_clarified 澄清 → complete 通过 →
// review 重入新轮（agree 携结论、unagree 复位 reviewing，round+1）→ 重入轮全 agree
// → 拍板推进。
func TestIntegration_ReviewClarifyLoop_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	review := func(reqID string, nodeId string, role string, capabilityID string, state string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": state})
	}

	reqID := setupReviewNode(t, router, handles, "1", "n-04")

	// reviewer 先提 issue（open；评审节点无 exec 能力归属，按 T7 口径显式指派 owner=pm；
	// BUG-013 口径：评审上下文 issue 评论必须锚产物——夹具登记上游产物并携锚），
	// 再报 agree
	seedUpstreamArtifact(t, handles, reqID, "art-it-review", "n-03")
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-04", "author": "arch", "roleId": "arch", "text": "边界场景未覆盖", "assignee": "pm",
		"anchor": map[string]any{"kind": "document", "artifactId": "art-it-review"},
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issueID := item(t, resp, "issue").str("issueId")
	// 注记-L1：agree 前置=名下评论闭环——owner 协商一致关闭后再表态
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/settle", map[string]any{
		"commentId": issueID, "operator": "pm", "note": "补充边界场景",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)

	assert.Equal(t, review(reqID, "n-04", "arch", "arch.prd-review", "agree").code, http.StatusOK)
	// rd 另提 open issue（unagree 提异议不受闭环限制——-L1 只约束 agree）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-04", "author": "rd", "roleId": "rd", "text": "接口超时策略未定义", "assignee": "pm",
		"anchor": map[string]any{"kind": "document", "artifactId": "art-it-review"},
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issue2 := item(t, resp, "issue").str("issueId")
	// 存在 unagree 且未齐 → pending（先齐后判，§1.2a ①→②）
	resp = review(reqID, "n-04", "rd", "rd.prd-review", "unagree")
	assert.Equal(t, resp.str("collectOutcome"), "pending", "未齐不触发澄清")

	// 全表态且存在 unagree → clarify：节点进 respond 相位
	resp = review(reqID, "n-04", "fe", "fe.prd-review", "agree")
	assert.Equal(t, resp.str("collectOutcome"), "clarify")
	assert.Equal(t, nodeCol(t, handles, "n-04", "state"), "respond", "owner 澄清相位激活")

	// owner（pm）my_task：respond 相位拿 clarify 上下文 + 未闭合 issue 清单
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "pm", "nodeId": "n-04"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("currentPhase"), "respond")
	active := arr(t, resp, "activeActions")
	assert.Equal(t, len(active), 1)
	assert.Equal(t, active[0].str("actionId"), "product-design.clarify", "BUG-006 口径：全名回显")
	issues := arr(t, resp, "relatedIssues")
	assert.Equal(t, len(issues), 1, "clarify 上下文携带 open issue（rd 的超时议题）")
	assert.Equal(t, issues[0].str("issueId"), issue2)

	// 派发 respond 任务并上报：open issue 未清 → complete 判据拦截回 in-progress
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-04", "actionId": "product-design.clarify", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	taskID := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID,
		map[string]any{"event": "ack"}).code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusConflict, "open issue 未清，判据拦截")
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")
	assert.Equal(t, taskCol(t, handles, taskID, "status"), "in-progress", "拦截后任务回 in-progress")
	assert.Equal(t, nodeCol(t, handles, "n-04", "state"), "respond", "节点停留 respond")

	// owner 澄清回复：open→pending-confirm（open 清零）→ complete 通过 → 评审轮重入
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issue2+"/clarified", map[string]any{
		"by": "pm", "roleId": "pm", "text": "超时策略已在 §6 约定",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+taskID, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, nodeCol(t, handles, "n-04", "state"), "review", "review 重入新轮")
	reentered := nodeCol(t, handles, "n-04", "review_results")
	assert.Assert(t, strings.Contains(reentered, `"round":1`), "新轮条目: %s", reentered)
	assert.Assert(t, !strings.Contains(reentered, `"round":0`), "全量替换=当前轮快照: %s", reentered)
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-04", "state") >= 1, true, "respond→review 编排审计")

	// my_task 摘要 round 轮转回归：agree=2（arch/fe 携结论）reviewing=1（rd 复位）全 round=1
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task",
		map[string]any{"role": "arch", "nodeId": "n-04"})
	assert.Equal(t, resp.code, http.StatusOK)
	summary := item(t, resp, "reviewSummary")
	assert.Equal(t, summary.number(t, "agree"), int64(2))
	assert.Equal(t, summary.number(t, "unagree"), int64(0))
	assert.Equal(t, summary.number(t, "reviewing"), int64(1))
	details := arr(t, summary, "details")
	assert.Equal(t, len(details), 3)
	for _, detail := range details {
		assert.Equal(t, detail.number(t, "round"), int64(1), "摘要明细全为新轮")
	}

	// 重入轮全 agree → agreed → 拍板 approved 推进（闭环）
	assert.Equal(t, review(reqID, "n-04", "rd", "rd.prd-review", "agree").code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-04/decision",
		map[string]any{"verdict": "approved", "reason": "澄清后通过"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("nextNodeId"), "n-05")
	assert.Equal(t, nodeCol(t, handles, "n-05", "state"), "exec")
}

// TestIntegration_ReviewConcurrentCollect_RealMySQL 并发双 collect 竞争（真库行锁验证）：
// 两表态并发触发 collect——行锁串行化下仅一次推进生效（flows 前移审计恰一次、
// last_decision 审计恰一次），败者无副作用且响应合法。
func TestIntegration_ReviewConcurrentCollect_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)

	reqID := setupReviewNode(t, router, handles, "1", "n-04")
	// scratch 编排模拟：bus-03 降为 user_gate=0（机械 approved 即时推进路径）
	execSQL(t, handles, "UPDATE flow_def_nodes SET user_gate = 0 WHERE template_node_id = 'bus-03'")

	reviewResp := func(role string, capabilityID string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/n-04/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": "agree"})
	}
	// arch 先表态（1/3），rd/fe 并发同发（各自触发 collect）
	assert.Equal(t, reviewResp("arch", "arch.prd-review").code, http.StatusOK)

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	codes := make([]int, 0, 2)
	nexts := make([]string, 0, 2)
	for _, position := range []struct{ role, capabilityID string }{
		{"rd", "rd.prd-review"}, {"fe", "fe.prd-review"},
	} {
		wg.Add(1)
		go func(role string, capabilityID string) {
			defer wg.Done()
			resp := reviewResp(role, capabilityID)
			mu.Lock()
			defer mu.Unlock()
			codes = append(codes, resp.code)
			nexts = append(nexts, resp.str("nextNodeId"))
		}(position.role, position.capabilityID)
	}
	wg.Wait()

	for _, code := range codes {
		assert.Equal(t, code, http.StatusOK, "并发表态均应成功（行锁串行化，无 500/409）")
	}
	advanced := 0
	for _, next := range nexts {
		if next == "n-05" {
			advanced++
		}
	}
	assert.Equal(t, advanced, 1, "仅一次推进生效")

	// 终态一致：三票齐、gate 落定、前移一次、下一节点激活
	assert.Equal(t, flowCol(t, handles, reqID, "current_node_id"), "n-05")
	assert.Equal(t, nodeCol(t, handles, "n-05", "state"), "exec")
	assert.Assert(t, strings.Contains(nodeCol(t, handles, "n-04", "last_decision"), `"verdict":"approved"`))
	results := nodeCol(t, handles, "n-04", "review_results")
	assert.Equal(t, strings.Count(results, `"role"`), 3, "三评审位条目齐: %s", results)
	assert.Equal(t, auditFieldCount(t, handles, "flows", reqID, "currentNodeId"), int64(1), "前移审计恰一次")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-04", "lastDecision"), int64(1), "落定审计恰一次")
	assert.Equal(t, auditFieldCount(t, handles, "flow_nodes", "n-05", "state"), int64(1), "激活审计恰一次")
}
