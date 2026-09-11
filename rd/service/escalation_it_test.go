// T12 涉库集成测试（AC，spec §4.1 冲突上升 AC）：escalate → 裁决 session →
// 结论回填 issue → reviewer 重表态 → 恢复评审轮全链路 + 阈值自动 escalate
// （注入小阈值）+ 通知位（issues notification 行）+ 非法路径 blocked 分支
// （未 escalate 拉裁决 / 重复裁决 / 终态回填）。
// env 门控与 scratch 库口径沿用 T2：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库，密码仅经 env 零落地。
//
// 实例节点 id 与需求序对应（T4 取号规则）：第 k 条 business×full 需求节点 =
// n-(13(k-1)+1)..n-13k ↔ bus-00..bus-12（bus-03 评审位 arch/rd/fe，user_gate=1）。
package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/database"
)

// escalationITThreshold IT 注入小阈值（AC「注入小阈值跑通」，避免 5 轮 reopen 拉锯）。
const escalationITThreshold = 2

// escalationIssue 建一条 open issue（arch 提评、显式指派 pm；2026-09-10 BUG-013
// 拍板：评审上下文 issue 评论必须锚产物——夹具登记上游产物并携 document 锚）。
func escalationIssue(t *testing.T, router *gin.Engine, handles *database.DB, reqID string, text string) (string, testResp) {
	t.Helper()
	seedUpstreamArtifact(t, handles, reqID, "art-it-esc", "n-03")
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-04", "author": "arch", "roleId": "arch", "assignee": "pm", "text": text,
		"anchor": map[string]any{"kind": "document", "artifactId": "art-it-esc"},
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	return item(t, resp, "issue").str("issueId"), resp
}

// escalationNoticeAnchor 读该需求最新一条通知行的 anchor_kind 与跳转锚
// （BUG-008 投影口径断言数据源）。
func escalationNoticeAnchor(t *testing.T, handles *database.DB, reqID string, sourceIssueID string) (string, string) {
	t.Helper()
	var kind, anchor string
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).Raw(
		"SELECT anchor_kind, COALESCE(anchor_artifact_id, '') FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND text LIKE ? ORDER BY id DESC LIMIT 1",
		reqID, "%"+sourceIssueID+"%").Row().Scan(&kind, &anchor))
	return kind, anchor
}

// jsonInt64 protojson 出参 int64 字段读取（字符串/数字双形态）。
func jsonInt64(t *testing.T, row map[string]any, key string) int64 {
	t.Helper()
	switch v := row[key].(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		assert.NilError(t, err, "字段 %s 应为数字字符串，实测 %v", key, row[key])
		return n
	case float64:
		return int64(v)
	}
	t.Fatalf("字段 %s 缺失或类型非法: %v", key, row[key])
	return 0
}

// sessionRows scratch 库读该需求全部 session 行关键列（断言数据源）。
func sessionRows(t *testing.T, handles *database.DB, reqID string) []map[string]string {
	t.Helper()
	type row struct {
		SessionID string
		NodeID    string
		RoleID    string
		DshRef    string
		Status    string
	}
	rows := make([]row, 0)
	err := handles.Gorm.WithContext(context.Background()).
		Raw("SELECT session_id, node_id, role_id, dsh_ref, status FROM sessions WHERE requirement_id = ? ORDER BY id", reqID).
		Scan(&rows).Error
	assert.NilError(t, err)
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]string{
			"sessionId": r.SessionID, "nodeId": r.NodeID, "roleId": r.RoleID,
			"dshRef": r.DshRef, "status": r.Status,
		})
	}
	return out
}

// repliesEntries 解析 issue 行 replies JSON（map 切片）。
func repliesEntries(t *testing.T, raw string) []map[string]any {
	t.Helper()
	out := make([]map[string]any, 0)
	assert.NilError(t, json.Unmarshal([]byte(raw), &out))
	return out
}

// TestIntegration_EscalationChain_RealMySQL 冲突上升 AC 全链路（review 相位拉锯）：
// reopen 达注入阈值自动 escalate + 通知位 → wb_escalate 拉起裁决 session（sessions
// 建绑 role=user、dsh_ref 占位、幂等 noop）→ wb_adjudicate override 结论回填
// （replies 裁决条目 + resolve_path=override + resolve_note 引用结论 + resolved 终态
// + escalate 复位 + session 终局）→ reviewer 重表态（评审轮重入新轮）→ 恢复评审轮
// （重入轮全 agree → agreed → 拍板推进，正常判据链恢复）。
func TestIntegration_EscalationChain_RealMySQL(t *testing.T) {
	router, handles := openScratchWithThreshold(t, escalationITThreshold)
	reqID := setupReviewNode(t, router, handles, "1", "n-04")
	issueID, _ := escalationIssue(t, router, handles, reqID, "边界场景未覆盖，需补测试用例")

	// 1. 阈值自动 escalate（注入阈值 2）：reopen 计数=2 时置位 + 通知位行
	// （完整拉锯：reply→reopen×2——reopen 操作者=提评者 arch）
	for i := 1; i <= escalationITThreshold; i++ {
		resp := reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/clarified", map[string]any{
			"by": "pm", "roleId": "pm", "text": "澄清回复",
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
		resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/reopen", map[string]any{
			"operator": "arch", "text": "不满意",
		})
		assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
		assert.Equal(t, resp.number(t, "loopCount"), int64(i))
	}
	resp := reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	assert.Equal(t, resp.code, http.StatusOK)
	var issueRow map[string]any
	for _, it := range arr(t, resp, "issues") {
		if it.str("issueId") == issueID {
			issueRow = it.body
		}
	}
	assert.Assert(t, issueRow != nil)
	assert.Equal(t, jsonInt64(t, issueRow, "escalate"), int64(1), "reopen 达阈值自动标 escalate")
	assert.Equal(t, jsonInt64(t, issueRow, "loopCount"), int64(escalationITThreshold))

	// 通知位：issues notification 行（assignee=user，同事务落库）
	noticeCount := int64(0)
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).Raw(
		"SELECT COUNT(*) FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND assignee = 'user' AND text LIKE ?",
		reqID, "%"+issueID+"%").Scan(&noticeCount).Error)
	assert.Equal(t, noticeCount, int64(1), "自动上升应写 notification 通知位行")
	// 通知行投影口径（BUG-008）：anchor_kind 落七类词表 escalation + anchor=源
	// issueId（跳转锚）——收件箱「冲突待裁决←escalation」投影不断链
	noticeKind, noticeAnchor := escalationNoticeAnchor(t, handles, reqID, issueID)
	assert.Equal(t, noticeKind, "escalation", "自动上升通知行 anchor_kind 应为 escalation")
	assert.Equal(t, noticeAnchor, issueID, "跳转锚=源 issueId")
	assert.Equal(t, auditCount(t, handles, "issues", issueID, "UPDATE") >= 1, true, "escalate 置位审计")

	// 2. wb_escalate 拉起裁决 session：sessions 建绑（node 关联 / role=user / dsh_ref 占位）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/escalate", map[string]any{
		"operator": "arch", "reason": "双方拉锯不收敛",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.boolean("ok"), true)
	assert.Equal(t, resp.boolean("escalated"), false, "自动上升已置位，本调用不重复置位")
	assert.Equal(t, resp.boolean("noop"), false)
	sessionID := resp.str("sessionId")
	assert.Assert(t, strings.HasPrefix(sessionID, "ad-"), "裁决 session 业务 id: %s", sessionID)
	sessions := sessionRows(t, handles, reqID)
	assert.Equal(t, len(sessions), 1, "裁决 session 落 sessions 表")
	assert.Equal(t, sessions[0]["nodeId"], "n-04", "session 与节点关联")
	assert.Equal(t, sessions[0]["roleId"], "user", "role=user 仲裁语义")
	assert.Equal(t, sessions[0]["dshRef"], "pending-dsh-adjudication", "dsh_ref 占位")
	assert.Equal(t, sessions[0]["status"], "active")
	// replies 上升条目（issue↔session 关联载体；出参取端点 issue 全量——快照
	// issues 投影不含 replies 与 resolve 族列，T7 已知口径，勿在其上断言）
	entries := repliesEntries(t, item(t, resp, "issue").body["replies"].(string))
	assert.Assert(t, len(entries) > 0 && entries[len(entries)-1]["kind"] == "escalation" &&
		entries[len(entries)-1]["sessionId"] == sessionID, "上升条目携 sessionId: %v", entries)

	// 3. 幂等：进行中 session 重复 escalate noop（不重复建绑、不重复留痕）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/escalate", map[string]any{
		"operator": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.boolean("noop"), true)
	assert.Equal(t, resp.str("sessionId"), sessionID, "幂等回显既有 session")
	assert.Equal(t, len(sessionRows(t, handles, reqID)), 1, "不重复建绑")

	// 4. wb_adjudicate verdict=override：仲裁推翻（wb_override 语义收编）全链回填
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/adjudicate", map[string]any{
		"verdict": "override", "conclusion": "议题不成立，按现状通过", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("verdict"), "override")
	assert.Equal(t, resp.str("sessionId"), sessionID)
	assert.Equal(t, resp.boolean("reentered"), true, "reviewer 重表态轮拉起")
	assert.Equal(t, resp.number(t, "round"), int64(1), "重入新轮")
	assert.Equal(t, nodeCol(t, handles, "n-04", "state"), "review", "恢复评审轮")
	reentered := nodeCol(t, handles, "n-04", "review_results")
	assert.Assert(t, strings.Contains(reentered, `"round":1`), "新轮快照: %s", reentered)
	assert.Assert(t, !strings.Contains(reentered, `"round":0`), "全量替换=当前轮快照")

	// issue 终态回填断言（resolved + resolve 族 + escalate 复位 + 裁决条目；
	// 出参取 adjudicate resp.issue 全量；09-12 裁决 overridden 态废除——
	// override 动作置 resolved 且 resolve_path=override）
	issue := item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "resolved", "仲裁推翻=resolved 终态（09-12 裁决废除 overridden）")
	assert.Equal(t, issue.str("resolvePath"), "override", "resolve_path=强权仲裁关闭")
	assert.Equal(t, issue.str("resolveNote"), "议题不成立，按现状通过", "resolve_note 引用结论")
	assert.Equal(t, issue.str("resolvedBy"), "user")
	assert.Equal(t, issue.number(t, "resolveReviewerConfirmed"), int64(1), "仲裁终局确认内嵌")
	assert.Equal(t, issue.number(t, "escalate"), int64(0), "escalate 复位（裁决完成即挂起解除）")
	entries = repliesEntries(t, issue.body["replies"].(string))
	var escalationEntry, adjudicationEntry map[string]any
	for _, e := range entries {
		if e["kind"] == "escalation" {
			escalationEntry = e
		}
		if e["kind"] == "adjudication" {
			adjudicationEntry = e
		}
	}
	assert.Assert(t, escalationEntry != nil && escalationEntry["sessionId"] == sessionID, "上升条目携 sessionId")
	assert.Assert(t, adjudicationEntry != nil && adjudicationEntry["verdict"] == "override", "裁决条目留痕")

	// session 终局
	sessions = sessionRows(t, handles, reqID)
	assert.Equal(t, sessions[0]["status"], "ended", "裁决 session 终局")

	// 5. 恢复评审轮：重入轮全 agree → agreed → 拍板 approved 推进（正常判据链恢复）
	review := func(role string, capabilityID string) testResp {
		return reqJSON(t, router, http.MethodPost,
			"/api/requirements/"+reqID+"/nodes/n-04/review",
			map[string]any{"role": role, "capabilityId": capabilityID, "state": "agree"})
	}
	assert.Equal(t, review("arch", "arch.prd-review").code, http.StatusOK)
	assert.Equal(t, review("rd", "rd.prd-review").code, http.StatusOK)
	resp = review("fe", "fe.prd-review")
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, resp.str("collectOutcome"), "approved")
	assert.Equal(t, resp.str("gateState"), "agreed", "user_gate=1 唤醒拍板")
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/nodes/n-04/decision",
		map[string]any{"verdict": "approved", "reason": "裁决后通过"})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("nextNodeId"), "n-05", "正常判据链恢复推进")

	// 审计留痕：issue UPDATE（escalate/replies/state）+ sessions INSERT/UPDATE
	assert.Assert(t, auditFieldCount(t, handles, "issues", issueID, "escalate") >= 2, "自动置位+复位审计")
	assert.Assert(t, auditFieldCount(t, handles, "issues", issueID, "state") >= 1, "resolved 迁移审计（override 动作置 resolved）")
	assert.Assert(t, auditCount(t, handles, "sessions", sessionID, "INSERT") == 1, "session INSERT 审计")
	assert.Assert(t, auditFieldCount(t, handles, "sessions", sessionID, "status") == 1, "session 终局审计")
}

// TestIntegration_EscalationUpholdAndBlocked_RealMySQL uphold 路径（维持议题：
// 状态保持、escalate 复位、回正常链路）+ 非法路径 blocked 分支（未 escalate 拉裁决 /
// 重复裁决 / 终态回填 / verdict 越集 / 404）。
func TestIntegration_EscalationUpholdAndBlocked_RealMySQL(t *testing.T) {
	router, handles := openScratchWithThreshold(t, escalationITThreshold)
	reqID := setupReviewNode(t, router, handles, "1", "n-04")

	// 0. 未 escalate 拉裁决 → blocked（escalate=0 无 session）
	plainID, _ := escalationIssue(t, router, handles, reqID, "普通议题")
	resp := reqJSON(t, router, http.MethodPost, "/api/comments/"+plainID+"/adjudicate", map[string]any{
		"verdict": "uphold", "conclusion": "x", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "未 escalate 拉裁决应 blocked: %v", resp.body)
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")
	assert.Equal(t, resp.boolean("blocked"), true)

	// 1. verdict 越集 400 / 入参缺失 400 / 404
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+plainID+"/adjudicate", map[string]any{
		"verdict": "approved", "conclusion": "x", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest, "verdict 越集（拍板词表不通用）")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+plainID+"/escalate", map[string]any{
		"operator": "",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest, "operator 必填")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/cm-999/escalate", map[string]any{
		"operator": "arch",
	})
	assert.Equal(t, resp.code, http.StatusNotFound, "评论未命中")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/cm-999/adjudicate", map[string]any{
		"verdict": "uphold", "conclusion": "x", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusNotFound)

	// 2. 终态不可上升：resolved issue 上 escalate → 409
	agreeID, _ := escalationIssue(t, router, handles, reqID, "协商一致议题")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+agreeID+"/settle", map[string]any{
		"operator": "pm", "note": "已对齐",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+agreeID+"/escalate", map[string]any{
		"operator": "arch",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "终态不可上升裁决")
	assert.Equal(t, resp.str("code"), "INVALID_TRANSITION")
	// 终态回填 → 409（即便伪造 escalate 置位也拒绝——状态机先行）
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).Exec(
		"UPDATE issues SET escalate = 1 WHERE issue_id = ?", agreeID).Error)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+agreeID+"/adjudicate", map[string]any{
		"verdict": "uphold", "conclusion": "x", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "终态回填拒绝")
	assert.Equal(t, resp.str("code"), "INVALID_TRANSITION")

	// 3. uphold 全链：手动 escalate（未达阈值）→ session → 裁决维持 → 状态保持
	upholdID, _ := escalationIssue(t, router, handles, reqID, "拉锯议题（维持）")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+upholdID+"/escalate", map[string]any{
		"operator": "arch", "reason": "手动上升",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.boolean("escalated"), true, "手动上升新置位")
	assert.Equal(t, resp.boolean("noop"), false)
	sessionID := resp.str("sessionId")
	// 通知位（手动路径同样落通知行）
	noticeCount := int64(0)
	assert.NilError(t, handles.Gorm.WithContext(context.Background()).Raw(
		"SELECT COUNT(*) FROM issues WHERE requirement_id = ? AND issue_type = 'notification' AND text LIKE ?",
		reqID, "%"+upholdID+"%").Scan(&noticeCount).Error)
	assert.Equal(t, noticeCount, int64(1), "手动上升写通知位行")
	// 通知行投影口径（BUG-008）：手动路径 anchor_kind 落 escalation（旧口径 document
	// 使收件箱 escalation 投影断链）+ anchor=源 issueId
	noticeKind, noticeAnchor := escalationNoticeAnchor(t, handles, reqID, upholdID)
	assert.Equal(t, noticeKind, "escalation", "手动上升通知行 anchor_kind 应为 escalation")
	assert.Equal(t, noticeAnchor, upholdID, "跳转锚=源 issueId")

	// 未拉起 session 的上升 issue 裁决 → blocked（无上升条目场景由终态/越集分支覆盖；
	// 此处直接验证重复裁决分支：先首裁 uphold）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+upholdID+"/adjudicate", map[string]any{
		"verdict": "uphold", "conclusion": "议题成立，owner 补充说明", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.str("verdict"), "uphold")
	assert.Equal(t, resp.boolean("reentered"), true)
	assert.Equal(t, resp.number(t, "round"), int64(1))
	assert.Equal(t, nodeCol(t, handles, "n-04", "state"), "review")

	// issue 断言：状态保持 open、escalate 复位、裁决条目、resolve 族保持 NULL
	// （出参取 adjudicate resp.issue 全量）
	issue := item(t, resp, "issue")
	assert.Equal(t, issue.str("issueId"), upholdID)
	assert.Equal(t, issue.str("state"), "open", "维持议题=状态保持")
	assert.Equal(t, issue.number(t, "escalate"), int64(0), "escalate 复位")
	assert.Equal(t, issue.str("resolvePath"), "", "resolve 族保持 NULL=未解决语义")
	entries := repliesEntries(t, issue.body["replies"].(string))
	found := false
	for _, e := range entries {
		if e["kind"] == "adjudication" && e["verdict"] == "uphold" {
			found = true
		}
	}
	assert.Assert(t, found, "裁决条目留痕")
	// session 终局 + 重复裁决 blocked
	assert.Equal(t, sessionRows(t, handles, reqID)[0]["status"], "ended")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+upholdID+"/adjudicate", map[string]any{
		"verdict": "uphold", "conclusion": "重复裁决", "decided_by": "user",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "重复裁决应 blocked")
	assert.Equal(t, resp.str("code"), "GATE_BLOCKED")

	// 4. 已终局 session 上重复 escalate → 新 session 允许（二次拉锯口径）：
	// escalate 幂等只挡「进行中」session，终局后可再次拉起
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+upholdID+"/escalate", map[string]any{
		"operator": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.boolean("noop"), false, "终局后二次拉起新 session")
	assert.Assert(t, resp.str("sessionId") != sessionID, "新 session id")
	assert.Equal(t, len(sessionRows(t, handles, reqID)), 2, "sessions 2 行（uphold 首拉/二次拉起）")
}
