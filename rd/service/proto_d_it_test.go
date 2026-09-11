// 原型补齐增补（spec §1.2 注记-D）涉库集成测试：
//
//	-D1 retest 通知（resolve path=fix → fixed 后派发 re_review，target_id=issueId，
//	   收件=reviewer_role；澄清子型 target_id 恒空不受影响）
//	-D2 快照 sessions 投影（GetRequirementResp.sessions）
//	-D3 history 端点（GET /api/requirements/:reqId/history 读 audit_log）
//	-D4 列表/快照 dependencies 填充（flows.dependencies 透传）
//
// env 门控与 scratch 库口径沿用 reqver_it_test.go：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_*
// 注入；独立 scratch 库测后 DROP。密码仅经 env 零落地。
package service_test

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// TestIntegration_ProtoD_SnapshotDepsSessionsHistory 快照/列表增补字段 + history 端点。
func TestIntegration_ProtoD_SnapshotDepsSessionsHistory(t *testing.T) {
	router, handles := openScratch(t)
	reqID := createBusinessFullReq(t, router)

	// 1. 创建即实例化：快照 requirement.dependencies 透传 flows.dependencies（初始空集）
	resp := reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	req := item(t, resp, "requirement")
	assert.Equal(t, len(strArr(t, req, "dependencies")), 0, "初始依赖为空集")
	assert.Equal(t, len(arr(t, resp, "sessions")), 0, "新建需求无会话绑定，空集合法")

	// 2. 依赖填充：直更 flows.dependencies 后快照透传（解析路径覆盖）
	db := handles.Gorm.WithContext(t.Context())
	res := db.Exec("UPDATE flows SET dependencies = ? WHERE requirement_id = ?",
		`["req-900","req-901"]`, reqID)
	assert.NilError(t, res.Error)
	assert.Equal(t, res.RowsAffected, int64(1), "创建即实例化，flows 行应存在")
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	req = item(t, resp, "requirement")
	// 快照 dependencies 应透传 flows.dependencies
	assert.DeepEqual(t, strArr(t, req, "dependencies"), []string{"req-900", "req-901"})

	// 3. 列表 dependencies 同源填充（IN 批量路径）
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements?page_size=100", nil)
	items := arr(t, resp, "items")
	assert.Assert(t, len(items) >= 1)
	var found *testResp
	for i := range items {
		if items[i].str("requirementId") == reqID {
			found = &items[i]
			break
		}
	}
	assert.Assert(t, found != nil, "列表应含本需求")
	// 列表行 dependencies 同源
	assert.DeepEqual(t, strArr(t, *found, "dependencies"), []string{"req-900", "req-901"})

	// 4. sessions 投影：直插绑定行后快照呈现（裁决链落行形态）
	sess := &repo.Session{
		SessionID:     "s-it-adj-1",
		RequirementID: reqID,
		NodeID:        "n-04",
		RoleID:        "arch",
		DshRef:        `{"placeholder":"it"}`,
		Status:        "active",
	}
	assert.NilError(t, handles.Gorm.Create(sess).Error)
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	sessions := arr(t, resp, "sessions")
	assert.Equal(t, len(sessions), 1, "快照应含 sessions 投影: %v", resp.body["sessions"])
	assert.Equal(t, sessions[0].str("sessionId"), "s-it-adj-1")
	assert.Equal(t, sessions[0].str("roleId"), "arch")
	assert.Equal(t, sessions[0].str("status"), "active")

	// 5. history：INSERT（创建）+ UPDATE（改字段）审计倒序透出
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	baseline := item(t, resp, "requirement").number(t, "updatedAt")
	resp = reqJSON(t, router, http.MethodPut, "/api/requirements/"+reqID, map[string]any{
		"title": "评论链路需求 v2", "description": "", "category": "需求任务",
		"priority": "P2", "status": "待启动", "owner": "",
		"versionId": "", "updatedAt": baseline,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)

	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID+"/history", nil)
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	history := arr(t, resp, "items")
	assert.Assert(t, len(history) >= 2, "至少 INSERT+UPDATE 两条: %v", resp.body)
	assert.Equal(t, history[0].str("action"), "UPDATE", "created_at 倒序，最新在前")
	assert.Equal(t, history[0].str("field"), "title")
	assert.Equal(t, history[0].str("oldValue"), "评论链路需求")
	assert.Equal(t, history[0].str("newValue"), "评论链路需求 v2")
	assert.Equal(t, history[len(history)-1].str("action"), "INSERT")

	// 6. history 404：需求未命中
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/req-999/history", nil)
	assert.Equal(t, resp.code, http.StatusNotFound)
}

// TestIntegration_ProtoD_RetestNotification 修复提交（resolve path=fix → fixed）
// 派发 re_review retest 子型通知（target_id=issueId，收件=reviewer_role）。
func TestIntegration_ProtoD_RetestNotification(t *testing.T) {
	router, handles := openScratch(t)
	reqID := createBusinessFullReq(t, router)

	// 夹具：产物（L1 assignee=登记人 pm）+ 评论（reviewer_role=arch 评审位命中）
	art := &repo.Artifact{
		ArtifactID: "art-it-retest", RequirementID: reqID, NodeID: "n-03",
		ProductType: "product_doc", Label: "《产品设计文档》", Href: "docs/product.md",
		Kind: "document", AddedBy: "pm",
	}
	assert.NilError(t, handles.Gorm.Create(art).Error)
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-04",
		"anchor": map[string]any{"kind": "range", "artifactId": "art-it-retest", "startLine": 6, "endLine": 7},
		"author": "arch", "roleId": "arch", "text": "第 7 行缺超时分支",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issueID := item(t, resp, "issue").str("issueId")

	// 修复闭环：wb_clarified（open→pending-confirm）→ wb_qualify 定性需修复（to-fix）→
	// 再 qualify 修复关闭申报（fixed → retest 通知，N1「to-fix→fixed」触发口径）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/clarified", map[string]any{
		"by": "pm", "roleId": "pm", "text": "超时分支已补",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/qualify", map[string]any{
		"operator": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "to-fix", "pending-confirm 定性需修复 → to-fix")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/qualify", map[string]any{
		"operator": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "fixed")

	// 收件箱：提评者 arch 应收 re_review 且 target_id=issueId（retest 子型判据）
	resp = reqJSON(t, router, http.MethodGet, "/api/inbox?role=arch&kind=re_review", nil)
	assert.Equal(t, resp.code, http.StatusOK)
	notifs := arr(t, resp, "notifications")
	assert.Assert(t, len(notifs) >= 1, "提评者应收 retest 通知: %v", resp.body)
	var hit *testResp
	for i := range notifs {
		if notifs[i].str("targetId") == issueID {
			hit = &notifs[i]
			break
		}
	}
	assert.Assert(t, hit != nil, "retest 子型 target_id=issueId: %v", resp.body)
	assert.Equal(t, hit.str("recipientRole"), "arch")

	// 幂等：fixed 态重复 qualify（Fixed→Fixed 重入）不重发 retest（CR B1 守卫）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/qualify", map[string]any{
		"operator": "pm",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "fixed")
	resp = reqJSON(t, router, http.MethodGet, "/api/inbox?role=arch&kind=re_review", nil)
	recount := 0
	for _, n := range arr(t, resp, "notifications") {
		if n.str("targetId") == issueID {
			recount++
		}
	}
	assert.Equal(t, recount, 1, "Fixed→Fixed 重入不重发 retest（幂等守卫）: %v", resp.body)

	// 非评审评论（reviewer_role 空、assignee 显式指派）：qualify 不派发 retest
	plain := &repo.Issue{
		IssueID: "cm-it-plain", RequirementID: reqID, NodeID: "n-02",
		AnchorKind: "document", Author: "user", RoleID: "user",
		IssueType: "issue", Text: "整体补充性能章节", State: repo.IssueStateOpen,
		Replies: "[]", Assignee: utils.StrPtr("rd"),
	}
	assert.NilError(t, handles.Gorm.Create(plain).Error)
	plainID := plain.IssueID
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+plainID+"/qualify", map[string]any{
		"operator": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "to-fix")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+plainID+"/qualify", map[string]any{
		"operator": "rd",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, item(t, resp, "issue").str("state"), "fixed")
	assert.Equal(t, item(t, resp, "issue").str("reviewerRole"), "")

	// n-02 无其他 re_review 来源：arch 收件清单不含该 plain issue（target 未新增）
	resp = reqJSON(t, router, http.MethodGet, "/api/inbox?role=arch&kind=re_review", nil)
	for _, n := range arr(t, resp, "notifications") {
		assert.Assert(t, n.str("targetId") != plainID,
			"非评审评论 resolve(fix) 不应派发 retest（reviewer_role 空）")
	}
}

// strArr 字符串数组字段断言数据源（dependencies 等 repeated string 出参）。
func strArr(t *testing.T, tr testResp, key string) []string {
	t.Helper()
	raw, ok := tr.body[key].([]any)
	assert.Assert(t, ok, "字段 %s 应为数组，实测 %v", key, tr.body[key])
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		s, ok := e.(string)
		assert.Assert(t, ok, "元素应为字符串，实测 %v", e)
		out = append(out, s)
	}
	return out
}
