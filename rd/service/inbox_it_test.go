// T11 涉库集成测试（AC）：wb_list_inbox 收件箱 + TODO 投影——
//   - 七类通知触发与载体断言：gate_pending（评审 agreed）/ clarify_task（collect
//     判 unagree）/ re_review（澄清闭环重入轮）/ escalation（escalate 置位）经真实
//     写路径触发；stall_followup / need_intervene（扫描器触发点 T10，枚举位）与
//     session_message（会话通道预留位）经收敛写入口 Notify 直投断言载体与投影；
//   - 跳转定位字段（requirement/node/target 类型与 id）逐类断言；
//   - 收件角色过滤 / 已读未读分支 / 分页 / 未读优先排序；
//   - TODO 投影一致性：tasks 表同源（不双写）、终态不残留；
//   - 通知落库审计 by_who=plugin。
//
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库，密码仅经 env 零落地。
//
// 实例节点 id 与需求序对应（T4 取号规则）：第 k 条 business×full 需求节点 =
// n-(13(k-1)+1)..n-13k ↔ bus-00..bus-12（第 1 条 bus-03=n-04、第 2 条 bus-03=n-17）。
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// inbox GET /api/inbox 薄封装（query 原样拼接）。
func inbox(t *testing.T, router *gin.Engine, query string) testResp {
	t.Helper()
	return reqJSON(t, router, http.MethodGet, "/api/inbox"+query, nil)
}

// findNotification 按类别取通知条目（未命中返回 ok=false）。
func findNotification(t *testing.T, tr testResp, kind string) (testResp, bool) {
	t.Helper()
	for _, entry := range arr(t, tr, "notifications") {
		if entry.str("kind") == kind {
			return entry, true
		}
	}
	return testResp{}, false
}

// notificationIDs 全部通知条目 id 集（分页/过滤断言用）。
func notificationIDs(t *testing.T, tr testResp) map[string]bool {
	t.Helper()
	ids := make(map[string]bool)
	for _, entry := range arr(t, tr, "notifications") {
		ids[entry.str("issueId")] = true
	}
	return ids
}

// TestIntegration_InboxRealMySQL 收件箱七类 + TODO 投影全链真库验证。
func TestIntegration_InboxRealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	review := func(reqID string, nodeId string, role string, capabilityID string, state string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/"+nodeId+"/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": state})
	}
	directNotify := func(spec service.NotificationSpec) string {
		t.Helper()
		notifier := service.NewNotifier(repo.NewRepo(handles.Gorm))
		row, err := notifier.Notify(context.Background(), spec)
		assert.NilError(t, err, "直投通知: %v", spec)
		return row.IssueID
	}

	// ---- 需求 1：bus-03（n-04）user_gate=1；owner=pm（被评审方收件口径） ----
	reqID := setupReviewNode(t, router, handles, "1", "n-04")
	execSQL(t, handles, "UPDATE requirements SET owner = 'pm' WHERE requirement_id = ?", reqID)

	// 1. gate_pending：三轮 agree → collect approved + user_gate=1 → 待你拍板
	for _, position := range []struct{ role, capabilityID string }{
		{"arch", "arch.prd-review"}, {"rd", "rd.prd-review"}, {"fe", "fe.prd-review"},
	} {
		resp := review(reqID, "n-04", position.role, position.capabilityID, "agree")
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	}
	resp := inbox(t, router, "?role=pm&read_filter=unread")
	assert.Equal(t, resp.code, http.StatusOK)
	gate, ok := findNotification(t, resp, "gate_pending")
	assert.Assert(t, ok, "待你拍板通知应在 pm 未读收件箱: %v", resp.body)
	assert.Equal(t, gate.str("requirementId"), reqID)
	assert.Equal(t, gate.str("nodeId"), "n-04")
	assert.Equal(t, gate.str("targetType"), "node", "拍板跳转=节点详情")
	assert.Equal(t, gate.str("targetId"), "", "node 类定位取 nodeId")
	assert.Equal(t, gate.str("recipientRole"), "pm", "收件=owner（被评审方）")
	assert.Equal(t, gate.str("senderRole"), "plugin", "系统投递口径")
	assert.Equal(t, gate.str("state"), "open", "未读")
	assert.Assert(t, gate.str("issueId") != "", "载体行 id")
	assert.Assert(t, gate.number(t, "createdAt") > 0)
	// 角色过滤：rd 收件箱无拍板通知
	_, ok = findNotification(t, inbox(t, router, "?role=rd"), "gate_pending")
	assert.Assert(t, !ok, "拍板通知只发 owner")

	// 通知落库审计 by_who=plugin（T13 全路径审计的写路径侧验证）
	var byWho string
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).
		Raw("SELECT by_who FROM audit_log WHERE table_name='issues' AND row_id=? AND action='INSERT'", gate.str("issueId")).
		Scan(&byWho).Error)
	assert.Equal(t, byWho, "plugin", "通知 INSERT 审计 by_who=plugin")

	// 2. clarify_task + re_review：需求 2（bus-03=n-17）澄清循环全链
	req2 := setupReviewNode(t, router, handles, "2", "n-17")
	execSQL(t, handles, "UPDATE requirements SET owner = 'pm' WHERE requirement_id = ?", req2)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+req2+"/comments", map[string]any{
		"nodeId": "n-17", "author": "arch", "roleId": "arch", "text": "边界未覆盖", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issueID := item(t, resp, "issue").str("issueId")
	assert.Equal(t, review(req2, "n-17", "arch", "arch.prd-review", "agree").code, http.StatusOK)
	assert.Equal(t, review(req2, "n-17", "rd", "rd.prd-review", "unagree").code, http.StatusOK)
	resp = review(req2, "n-17", "fe", "fe.prd-review", "agree")
	assert.Equal(t, resp.str("collectOutcome"), "clarify", "全表态存在 unagree → 澄清")
	resp = inbox(t, router, "?role=pm")
	clarify, ok := findNotification(t, resp, "clarify_task")
	assert.Assert(t, ok, "澄清任务通知应发 owner: %v", resp.body)
	assert.Equal(t, clarify.str("nodeId"), "n-17")
	assert.Equal(t, clarify.str("targetType"), "node")
	assert.Equal(t, clarify.str("recipientRole"), "pm")

	// 澄清闭环：respond complete → review 重入新轮 → re_review 通知复位评审位（rd）
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": req2, "nodeId": "n-17", "actionId": "product-design.clarify", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	clarifyTask := item(t, resp, "task").str("taskId")
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+clarifyTask,
		map[string]any{"event": "ack"}).code, http.StatusOK)
	assert.Equal(t, reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies",
		map[string]any{"by": "pm", "roleId": "pm", "text": "附录 C 已覆盖"}).code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+clarifyTask, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = inbox(t, router, "?role=rd")
	reReview, ok := findNotification(t, resp, "re_review")
	assert.Assert(t, ok, "重新表态请求应发复位评审位（rd）: %v", resp.body)
	assert.Equal(t, reReview.str("nodeId"), "n-17")
	assert.Equal(t, reReview.str("targetType"), "node")
	assert.Equal(t, reReview.str("recipientRole"), "rd")
	// 复位评审位唯一：arch/fe 携结论进新轮，仅 rd 收通知
	resp = inbox(t, router, "?role=arch")
	_, ok = findNotification(t, resp, "re_review")
	assert.Assert(t, !ok, "携结论进新轮的评审位不收重表态通知")

	// 3. escalation：评论携 escalate 置位 → 冲突上升裁决通知（reviewer 主持）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+req2+"/comments", map[string]any{
		"nodeId": "n-17", "author": "arch", "roleId": "arch", "text": "口径冲突，上升裁决", "escalate": true,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	escIssueID := item(t, resp, "issue").str("issueId")
	resp = inbox(t, router, "?role=arch")
	esc, ok := findNotification(t, resp, "escalation")
	assert.Assert(t, ok, "冲突上升裁决通知应发 reviewer 主持（arch）: %v", resp.body)
	assert.Equal(t, esc.str("nodeId"), "n-17")
	assert.Equal(t, esc.str("targetType"), "issue", "裁决跳转=冲突 issue")
	assert.Equal(t, esc.str("targetId"), escIssueID)
	assert.Equal(t, esc.str("recipientRole"), "arch")
	// 非评审上下文（reviewer_role 空）回落 User 仲裁（n-02 属需求 1，无评审位声明）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-02", "author": "pm", "roleId": "pm", "text": "对齐分歧上升", "escalate": true,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	_, ok = findNotification(t, inbox(t, router, "?role=user"), "escalation")
	assert.Assert(t, ok, "无评审归属的 escalate 通知回落 User")

	// 4. 枚举位三类直投（stall_followup/need_intervene 触发点=扫描器 T10；
	// session_message 触发点=会话通道 T12+）——载体与投影断言
	stallID := directNotify(service.NotificationSpec{
		Kind: service.NotificationKindStallFollowup, RequirementID: reqID, NodeID: "n-04",
		TargetID: "t-1730000000000-001", RecipientRole: "rd", Sender: "scanner",
		Text: "已结束 5min 未表态，请跟进（遇卡点上报）",
	})
	directNotify(service.NotificationSpec{
		Kind: service.NotificationKindNeedIntervene, RequirementID: reqID, NodeID: "n-04",
		TargetID: "t-1730000000000-001", RecipientRole: "user", Sender: "scanner",
		Text: "10min 仍未推进，需你介入",
	})
	directNotify(service.NotificationSpec{
		Kind: service.NotificationKindSessionMessage, RequirementID: reqID, NodeID: "n-04",
		TargetID: "sess-1", RecipientRole: "pm", Text: "对齐会话有新消息",
	})
	resp = inbox(t, router, "?role=rd")
	stall, ok := findNotification(t, resp, "stall_followup")
	assert.Assert(t, ok, "悬置催办枚举位: %v", resp.body)
	assert.Equal(t, stall.str("issueId"), stallID)
	assert.Equal(t, stall.str("targetType"), "task", "催办跳转=任务")
	assert.Equal(t, stall.str("targetId"), "t-1730000000000-001")
	_, ok = findNotification(t, inbox(t, router, "?role=user"), "need_intervene")
	assert.Assert(t, ok, "需你介入枚举位（收件 User）")
	sess, ok := findNotification(t, inbox(t, router, "?role=pm"), "session_message")
	assert.Assert(t, ok, "会话消息枚举位")
	assert.Equal(t, sess.str("targetType"), "session")
	assert.Equal(t, sess.str("targetId"), "sess-1")

	// 5. kind 筛选 + 越集拦截
	resp = inbox(t, router, "?role=pm&kind=session_message")
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, len(arr(t, resp, "notifications")), 1, "kind 单值筛选")
	assert.Equal(t, arr(t, resp, "notifications")[0].str("kind"), "session_message")
	resp = inbox(t, router, "?read_filter=no_such")
	assert.Equal(t, resp.code, http.StatusBadRequest)
	resp = inbox(t, router, "?kind=no_such")
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 6. 已读/未读分支：标记 gate_pending 已读 → read/read_filter 双向成立
	execSQL(t, handles, "UPDATE issues SET state = 'resolved' WHERE issue_id = ?", gate.str("issueId"))
	ids := notificationIDs(t, inbox(t, router, "?role=pm&read_filter=unread"))
	assert.Assert(t, !ids[gate.str("issueId")], "已读不进未读分支")
	ids = notificationIDs(t, inbox(t, router, "?role=pm&read_filter=read"))
	assert.Assert(t, ids[gate.str("issueId")], "已读分支命中")
	assert.Assert(t, notificationIDs(t, inbox(t, router, "?role=pm"))[gate.str("issueId")], "全量分支含已读")

	// 7. 分页 + 未读优先排序：qa 直投 3 条，标记最早 1 条已读
	qaIDs := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		qaIDs = append(qaIDs, directNotify(service.NotificationSpec{
			Kind: service.NotificationKindStallFollowup, RequirementID: reqID, NodeID: "n-04",
			TargetID: fmt.Sprintf("t-qa-%d", i), RecipientRole: "qa", Text: "催办",
		}))
	}
	execSQL(t, handles, "UPDATE issues SET state = 'resolved' WHERE issue_id = ?", qaIDs[0])
	resp = inbox(t, router, "?role=qa&page=1&page_size=2")
	assert.Equal(t, resp.number(t, "total"), int64(3), "筛选后总数")
	assert.Equal(t, len(arr(t, resp, "notifications")), 2)
	assert.Equal(t, arr(t, resp, "notifications")[0].str("issueId"), qaIDs[2], "未读优先且新→旧")
	assert.Equal(t, arr(t, resp, "notifications")[1].str("issueId"), qaIDs[1])
	resp = inbox(t, router, "?role=qa&page=2&page_size=2")
	assert.Equal(t, resp.number(t, "total"), int64(3))
	assert.Equal(t, len(arr(t, resp, "notifications")), 1)
	assert.Equal(t, arr(t, resp, "notifications")[0].str("issueId"), qaIDs[0], "已读垫底")

	// 8. TODO 投影一致性：tasks 表同源、终态不残留
	resp = reqJSON(t, router, http.MethodPost, "/api/tasks", map[string]any{
		"requirementId": reqID, "nodeId": "n-03", "actionId": "product-design.produce", "assignee": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	produceTask := item(t, resp, "task").str("taskId")
	resp = inbox(t, router, "?role=pm")
	found := false
	for _, todo := range arr(t, resp, "todos") {
		if todo.str("taskId") == produceTask {
			found = true
			assert.Equal(t, todo.str("status"), "todo")
			assert.Equal(t, todo.str("requirementId"), reqID, "投影跳转定位")
			assert.Equal(t, todo.str("nodeId"), "n-03")
		}
	}
	assert.Assert(t, found, "在途任务进 TODO 投影")
	// 同源断言：投影行数=tasks 表该角色在途行数（无独立投影表、不双写）
	var inflight int64
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).Raw(
		"SELECT COUNT(*) FROM tasks WHERE role_id = 'pm' AND status NOT IN ('completed','rejected','closed')").
		Scan(&inflight).Error)
	assert.Equal(t, int64(len(arr(t, resp, "todos"))), inflight, "TODO 投影与 tasks 表同源")
	// 任务终态迁移后投影同步（completed 不残留）
	execSQL(t, handles, `INSERT INTO artifacts
		(artifact_id, requirement_id, node_id, product_type, label, href, added_by, created_at, updated_at)
		VALUES ('art-inbox-1', ?, 'n-03', 'product_doc', '《产品设计文档》', 'docs/x.md', 'pm', 1, 1)`, reqID)
	assert.Equal(t, reqJSON(t, router, http.MethodPatch, "/api/tasks/"+produceTask,
		map[string]any{"event": "ack"}).code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPatch, "/api/tasks/"+produceTask, map[string]any{"event": "complete"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = inbox(t, router, "?role=pm")
	for _, todo := range arr(t, resp, "todos") {
		assert.Assert(t, todo.str("taskId") != produceTask, "终态任务不残留投影")
	}
	// 角色过滤：rd 无在途任务
	assert.Equal(t, len(arr(t, inbox(t, router, "?role=rd"), "todos")), 0, "按角色过滤 TODO")

	// 9. 未带 role：全量视图（跨角色）
	assert.Assert(t, len(arr(t, inbox(t, router, ""), "notifications")) > 0, "空 role=全量")
	// 401：无 token 拒绝
	assert.Equal(t, reqJSONHeader(t, router, http.MethodGet, "/api/inbox", nil, nil).code, http.StatusUnauthorized)
}
