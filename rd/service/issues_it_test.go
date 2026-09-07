// T7 涉库集成测试（AC）：评论/issue 五端点全链路——正常 / 拦截（400 入参锚点 /
// 404 未命中 / 409 非法迁移与操作者越权）/ 状态幂等分支 / loop_count 双读法与
// 阈值自动 escalate / assignee 推导（L1 锚点产物登记人）/ my_task relatedIssues
// 注入 / 审计留痕 / 并发 reply 追加不丢行（事务 + 行锁）。
// env 门控与 scratch 库口径沿用 T2/T3/T4：WBFLOW_MYSQL_IT=1 + WBFLOW_MYSQL_* 注入，
// 独立 scratch 库（wbflow_it_*）测后 DROP——禁写 wbflow 库。密码仅经 env 零落地。
package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
)

// createBusinessFullReq 创建 business full 需求（13 节点全渲染）。
func createBusinessFullReq(t *testing.T, router *gin.Engine) string {
	t.Helper()
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements", map[string]any{
		"title": "评论链路需求", "category": "需求任务", "scope": "full", "createdBy": "user-a",
	})
	assert.Equal(t, resp.code, http.StatusOK, "创建失败: %v", resp.body)
	return item(t, resp, "requirement").str("requirementId")
}

// parseReplies 解析出参 replies JSON 文本。
func parseReplies(t *testing.T, tr testResp) []map[string]any {
	t.Helper()
	text := tr.str("replies")
	if text == "" {
		text = "[]"
	}
	out := make([]map[string]any, 0)
	assert.NilError(t, json.Unmarshal([]byte(text), &out))
	return out
}

// TestIntegration_IssueEndpoints_RealMySQL 五端点全链路：澄清循环 → 修复循环 →
// 复核闭环 + 双路径 resolve + 越权/非法迁移拦截 + my_task relatedIssues。
func TestIntegration_IssueEndpoints_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	reqID := createBusinessFullReq(t, router)
	node := reqID + "/comments"

	// 夹具：bus-02（n-03 产品设计）产物，登记人 pm——L1 归属推导依据
	art := &repo.Artifact{
		ArtifactID: "art-it-doc", RequirementID: reqID, NodeID: "n-03",
		ProductType: "product_doc", Label: "《产品设计文档》", Href: "docs/product.md",
		Kind: "document", AddedBy: "pm",
	}
	assert.NilError(t, handles.Gorm.Create(art).Error)

	// 1. wb_comment：line 锚点（产物必锚 + L1 assignee=登记人 pm + reviewer_role=arch 评审位命中）
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-04",
		"anchor": map[string]any{"kind": "line", "artifactId": "art-it-doc", "line": 42},
		"author": "arch", "roleId": "arch", "text": "第 42 行时序图缺超时分支",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issue := item(t, resp, "issue")
	issueID := issue.str("issueId")
	assert.Assert(t, issueID != "", "issue_id 应取号 cm-<ts>")
	assert.Equal(t, issue.str("state"), "open")
	assert.Equal(t, issue.str("anchorKind"), "line")
	assert.Equal(t, issue.str("anchorArtifactId"), "art-it-doc")
	assert.Equal(t, issue.number(t, "anchorLine"), int64(42))
	assert.Equal(t, issue.str("reviewerRole"), "arch", "arch ∈ n-04 评审位，评审归属推导")
	assert.Equal(t, issue.str("assignee"), "pm", "L1 锚点产物登记人推导")
	assert.Equal(t, issue.str("issueType"), "issue")
	assert.Equal(t, issue.number(t, "loopCount"), int64(0))
	assert.Equal(t, auditCount(t, handles, "issues", issueID, "INSERT"), int64(1))

	// 2. 入参拦截：锚点 kind 与字段一致性（400）
	for name, anchor := range map[string]map[string]any{
		"line 缺产物":      {"kind": "line", "line": 3},
		"line 缺行号":      {"kind": "line", "artifactId": "art-it-doc"},
		"block 缺 lang":  {"kind": "block", "artifactId": "art-it-doc", "blockStart": 1, "blockEnd": 2},
		"range 倒挂":      {"kind": "range", "artifactId": "art-it-doc", "startLine": 9, "endLine": 3},
		"未知 kind":       {"kind": "heading"},
		"document 冗余行号": {"kind": "document", "line": 5},
	} {
		resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
			"nodeId": "n-04", "anchor": anchor, "author": "arch", "roleId": "arch", "text": "x",
		})
		assert.Equal(t, resp.code, http.StatusBadRequest, "%s 应 400: %v", name, resp.body)
	}
	// 入参拦截：issue_type 越集 / 必填缺失（400）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-04", "author": "arch", "roleId": "arch", "text": "x", "issueType": "bug",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-04", "roleId": "arch", "text": "缺 author",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 3. 404：节点归属校验 / 评论未命中
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-99", "author": "arch", "roleId": "arch", "text": "x",
	})
	assert.Equal(t, resp.code, http.StatusNotFound)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/cm-999/replies", map[string]any{
		"by": "pm", "roleId": "pm", "text": "x",
	})
	assert.Equal(t, resp.code, http.StatusNotFound)

	// 4. wb_reply：澄清回复完成 open→pending-confirm，roundCount++、last_reply_at
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "pm", "roleId": "pm", "text": "超时分支已补 §4.3",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "pending-confirm", "CONTEXT：已处理=回复完成待复核")
	assert.Equal(t, issue.number(t, "roundCount"), int64(1))
	assert.Assert(t, issue.number(t, "lastReplyAt") > 0)
	assert.Equal(t, len(parseReplies(t, issue)), 1)

	// 5. wb_reply：待复核态追加回复，状态停留不回退（多轮澄清幂等）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "arch", "roleId": "arch", "text": "补充追问：重试策略？",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "pending-confirm")
	assert.Equal(t, issue.number(t, "roundCount"), int64(2))

	// 6. wb_resolve path=fix：越权（fe 非 assignee）409；assignee pm 定性需修复 → to-fix
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/resolve", map[string]any{
		"path": "fix", "operator": "fe", "note": "越权",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "resolve 非 assignee 应 409: %v", resp.body)
	assert.Equal(t, resp.str("code"), "INVALID_TRANSITION")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/resolve", map[string]any{
		"path": "fix", "operator": "pm", "note": "需改时序图并补分支",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "to-fix")
	assert.Equal(t, issue.str("resolvePath"), "fix")
	assert.Equal(t, issue.str("resolveBy"), "pm")
	assert.Assert(t, issue.number(t, "resolveAt") > 0)
	assert.Equal(t, issue.str("resolveNote"), "需改时序图并补分支")
	// path 越集 400
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/resolve", map[string]any{
		"path": "close", "operator": "pm",
	})
	assert.Equal(t, resp.code, http.StatusBadRequest)

	// 7. wb_reply：修复完成申报 to-fix→fixed
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "pm", "roleId": "pm", "text": "时序图已补超时分支",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "fixed")

	// 8. wb_reopen：越权（rd 非提评者）409；提评者 arch 修复 revise → to-fix，loop_count++（修复期读=revise 次数）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/reopen", map[string]any{
		"operator": "rd", "text": "越权",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "reopen 非提评者应 409")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/reopen", map[string]any{
		"operator": "arch", "text": "超时分支重试策略仍缺失",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	assert.Equal(t, resp.number(t, "loopCount"), int64(1), "修复 revise 计数")
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "to-fix")

	// 9. 修复→复核→resolved：reply（fixed）→ confirm（提评者）
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
		"by": "pm", "roleId": "pm", "text": "重试策略已补三次退避",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "fixed")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/confirm", map[string]any{
		"operator": "rd",
	})
	assert.Equal(t, resp.code, http.StatusConflict, "confirm 非提评者应 409")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/confirm", map[string]any{
		"operator": "arch",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "resolved", "CONTEXT：resolved=评审者确认后达成")
	assert.Equal(t, issue.number(t, "resolveReviewConfirmed"), int64(1))
	assert.Equal(t, issue.str("resolvedBy"), "arch")
	assert.Assert(t, issue.number(t, "resolvedAt") > 0)
	assert.Assert(t, auditCount(t, handles, "issues", issueID, "UPDATE") >= 1, "迁移应有 UPDATE 审计")

	// 10. 终态拦截：resolved 上四类动作全 409（非法迁移哨兵）
	for name, path := range map[string]string{
		"reply":   "/api/comments/" + issueID + "/replies",
		"resolve": "/api/comments/" + issueID + "/resolve",
		"confirm": "/api/comments/" + issueID + "/confirm",
		"reopen":  "/api/comments/" + issueID + "/reopen",
	} {
		body := map[string]any{"operator": "arch", "path": "agree", "by": "arch", "roleId": "arch", "text": "终态"}
		resp = reqJSON(t, router, http.MethodPost, path, body)
		assert.Equal(t, resp.code, http.StatusConflict, "终态 %s 应 409", name)
	}

	// 11. wb_resolve path=agree：协商一致关闭（确认内嵌，直落 resolved）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-04", "anchor": map[string]any{"kind": "document"},
		"author": "rd", "roleId": "rd", "issueType": "discussion", "assignee": "qa",
		"text": "验收口径双方对齐即可关闭", "escalate": true,
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	agree := item(t, resp, "issue")
	agreeID := agree.str("issueId")
	assert.Equal(t, agree.str("state"), "open")
	assert.Equal(t, agree.str("reviewerRole"), "rd")
	assert.Equal(t, agree.str("assignee"), "qa", "显式指派优先于推导")
	assert.Equal(t, agree.str("issueType"), "discussion")
	assert.Equal(t, agree.number(t, "escalate"), int64(1), "建评 +escalate 置位")
	assert.Equal(t, agree.str("anchorKind"), "document")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+agreeID+"/resolve", map[string]any{
		"path": "agree", "operator": "qa", "note": "已对齐",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	agree = item(t, resp, "issue")
	assert.Equal(t, agree.str("state"), "resolved")
	assert.Equal(t, agree.str("resolvePath"), "agree")
	assert.Equal(t, agree.number(t, "resolveReviewConfirmed"), int64(1), "协商一致=确认内嵌")
	assert.Equal(t, agree.str("resolvedBy"), "qa")
	assert.Equal(t, agree.number(t, "loopCount"), int64(0))

	// 12. 状态幂等无害：fixed 态重申报 resolve path=fix 停留 fixed（重复申报无害）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-04", "anchor": map[string]any{"kind": "document"},
		"author": "fe", "roleId": "fe", "assignee": "fe", "text": "样式偏差",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	fixID := item(t, resp, "issue").str("issueId")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+fixID+"/resolve", map[string]any{"path": "fix", "operator": "fe"})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, item(t, resp, "issue").str("state"), "to-fix")
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+fixID+"/replies", map[string]any{
		"by": "fe", "roleId": "fe", "text": "已修",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+fixID+"/resolve", map[string]any{
		"path": "fix", "operator": "fe", "note": "修复说明更新",
	})
	assert.Equal(t, resp.code, http.StatusOK, "fixed 态重申报应无害: %v", resp.body)
	issue = item(t, resp, "issue")
	assert.Equal(t, issue.str("state"), "fixed", "重申报状态幂等")
	assert.Equal(t, issue.number(t, "roundCount"), int64(1))

	// 13. 澄清不收敛：reopen 达阈值（5）自动标 escalate（spec §2，T12 消费钩子）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+node, map[string]any{
		"nodeId": "n-04", "anchor": map[string]any{"kind": "document"},
		"author": "arch", "roleId": "arch", "assignee": "pm", "text": "拉锯议题",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	loopID := item(t, resp, "issue").str("issueId")
	for i := 1; i <= 5; i++ {
		resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+loopID+"/replies", map[string]any{
			"by": "pm", "roleId": "pm", "text": fmt.Sprintf("澄清第 %d 轮", i),
		})
		assert.Equal(t, resp.code, http.StatusOK)
		issue = item(t, resp, "issue")
		assert.Equal(t, issue.str("state"), "pending-confirm")
		resp = reqJSON(t, router, http.MethodPost, "/api/comments/"+loopID+"/reopen", map[string]any{
			"operator": "arch", "text": fmt.Sprintf("不满意第 %d 轮", i),
		})
		assert.Equal(t, resp.code, http.StatusOK)
		assert.Equal(t, resp.number(t, "loopCount"), int64(i), "澄清 reopen 计数（澄清期读=reopen 次数）")
	}
	assert.Equal(t, item(t, resp, "issue").number(t, "escalate"), int64(1), "reopen≥5 自动标 escalate")
	assert.Equal(t, item(t, resp, "issue").str("state"), "open")

	// 14. my_task relatedIssues：目标节点未闭合 issue 摘要（resolved 不返回）
	//     此时 n-04 上：issueID=resolved、agreeID=resolved、fixID=fixed（未闭合）、loopID=open（未闭合）
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{
		"role": "pm", "nodeId": "n-04",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	related := arr(t, resp, "relatedIssues")
	assert.Equal(t, len(related), 2, "resolved 不注入，实测 %v", resp.body["relatedIssues"])
	seen := map[string]bool{}
	for _, r := range related {
		seen[r.str("issueId")] = true
		assert.Assert(t, r.str("anchor") != "", "锚点摘要应填充")
		assert.Assert(t, r.str("assignee") != "")
	}
	assert.Assert(t, seen[fixID] && seen[loopID], "未闭合两条都应返回")
	// 无 issue 节点空集
	resp = reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/my-task", map[string]any{
		"role": "pm", "nodeId": "n-03",
	})
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Equal(t, len(arr(t, resp, "relatedIssues")), 0)

	// 15. 快照 issues 集合字段接线（GetRequirementResp.issues）
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	assert.Equal(t, resp.code, http.StatusOK)
	assert.Assert(t, len(arr(t, resp, "issues")) >= 4, "快照应含全部评论行")
}

// TestIntegration_IssueConcurrentReply_RealMySQL 并发 reply 追加不丢行：
// 行锁 + 单事务串行化读改写，N 路并发全部成功且 replies/round_count 恰好 N。
func TestIntegration_IssueConcurrentReply_RealMySQL(t *testing.T) {
	router, handles := openScratch(t)
	reqID := createBusinessFullReq(t, router)
	art := &repo.Artifact{
		ArtifactID: "art-it-cc", RequirementID: reqID, NodeID: "n-03",
		ProductType: "product_doc", Label: "《产品设计文档》", Href: "docs/product.md",
		Kind: "document", AddedBy: "pm",
	}
	assert.NilError(t, handles.Gorm.Create(art).Error)
	resp := reqJSON(t, router, http.MethodPost, "/api/requirements/"+reqID+"/comments", map[string]any{
		"nodeId": "n-04", "anchor": map[string]any{"kind": "document"},
		"author": "arch", "roleId": "arch", "assignee": "pm", "text": "并发回复基线",
	})
	assert.Equal(t, resp.code, http.StatusOK, "%v", resp.body)
	issueID := item(t, resp, "issue").str("issueId")

	const n = 5
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := reqJSON(t, router, http.MethodPost, "/api/comments/"+issueID+"/replies", map[string]any{
				"by": fmt.Sprintf("pm-%d", i), "roleId": "pm", "text": fmt.Sprintf("并发回复 %d", i),
			})
			if r.code != http.StatusOK {
				errs[i] = fmt.Errorf("reply #%d HTTP %d: %v", i, r.code, r.body)
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NilError(t, err, "并发 reply #%d", i)
	}
	resp = reqJSON(t, router, http.MethodGet, "/api/requirements/"+reqID, nil)
	assert.Equal(t, resp.code, http.StatusOK)
	var target map[string]any
	for _, it := range arr(t, resp, "issues") {
		if it.str("issueId") == issueID {
			target = it.body
		}
	}
	assert.Assert(t, target != nil, "快照应含并发 issue")
	replies := make([]map[string]any, 0)
	assert.NilError(t, json.Unmarshal([]byte(target["replies"].(string)), &replies))
	assert.Equal(t, len(replies), n, "行锁串行化下 N 条回复不丢行")
	assert.Equal(t, int64(len(replies)), mustRoundCount(t, target), "round_count 与回复数一致")
}

// mustRoundCount 从快照 issue 行取 round_count（数字字符串）。
func mustRoundCount(t *testing.T, row map[string]any) int64 {
	t.Helper()
	switch v := row["roundCount"].(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		assert.NilError(t, err)
		return n
	case float64:
		return int64(v)
	}
	t.Fatalf("roundCount 缺失: %v", row["roundCount"])
	return 0
}
