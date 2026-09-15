// Package contract 跨语言契约快照单一真相源（架构评审 C3 深化，提案
// .plan/proposals/contract-golden-fixture-plan.md）——把 FE 手写镜像的 payload
// 形状、判定函数与词表，从 Go 真相源装配为一份 golden JSON（fe/src/api/contract/
// golden.json，提交进库），Go 自检测试与 FE vitest 消费同一份：任一侧漂移即红灯。
//
// 再生命令（改判定/词表/payload 后必须重跑并提交两侧）：
//
//	cd rd && go test ./contract/ -run TestUnit_ContractSnapshot -update
//	cd fe && npx vitest run src/api/contract
package contract

import (
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
	"github.com/huzilin/wbflow/rd/service"
)

// Build 装配契约快照（三节：payload 形状样例 / verdicts 判定真值表 / tables 词表）。
// 输出为可 JSON 序列化的 map（键序不敏感——比对前归一化）。
func Build() map[string]any {
	return map[string]any{
		"_readme": "跨语言契约 golden（C3）：Go 真相源生成，Go 自检测试与 FE vitest 消费同一份；" +
			"再生命令见 rd/contract/contract.go 头注。改判定/词表/payload 后必须 -update 重生成并提交。",
		"payload":  payloadSection(),
		"verdicts": verdictsSection(),
		"tables":   tablesSection(),
	}
}

// payloadSection TEXT 列 JSON 载荷样例（FE flowview.ts parse 函数族的形状锁；
// 含 omitempty/边界键位覆盖）。
func payloadSection() map[string]any {
	return map[string]any{
		"reviewerDef": []repo.ReviewerDef{
			{Role: "rd", CapabilityID: "rd.prd-review"},
			{Role: "qa", CapabilityID: "qa.accept", ScopeExpr: "full"},
		},
		"reviewResult": repo.ReviewResult{
			Role: "rd", CapabilityID: "rd.prd-review", State: repo.ReviewStateAgree,
			Round: 1, At: 1757875200000,
		},
		"issueReply": map[string]repo.IssueReply{
			// 普通回复（kind 空=普通）
			"plain": {ReplyID: "r-1", By: "pm", RoleID: "pm", Text: "澄清说明", At: 1757875200000},
			// 误发作废（voided=true）
			"voided": {ReplyID: "r-2", By: "rd", RoleID: "rd", Text: "（误发）", Voided: true, At: 1757875200001},
			// 冲突上升条目（T12：kind=escalation + sessionId）
			"escalation": {ReplyID: "r-3", By: "arch", RoleID: "arch", Text: "冲突上升理由",
				At: 1757875200002, Kind: domain.ReplyKindEscalation, SessionID: "sess-adj-1"},
			// 仲裁裁决条目（T12：kind=adjudication + verdict + sessionId）
			"adjudication": {ReplyID: "r-4", By: "user", RoleID: "user", Text: "裁决结论",
				At: 1757875200003, Kind: domain.ReplyKindAdjudication, Verdict: "uphold", SessionID: "sess-adj-1"},
		},
		"gateDecision": map[string]domain.GateDecision{
			"approved": {Verdict: "approved", Reason: "评审通过", At: 1757875200000, Branch: "feat/audit"},
			"rejected": {Verdict: "rejected", Reason: "打回修改", At: 1757875200001},
		},
	}
}

// verdictsSection 判定函数真值表（输入样例 → Go 真相源输出；FE 逐例断言同输入同结果）。
func verdictsSection() map[string]any {
	return map[string]any{
		"summarizeReviewResults": []map[string]any{
			{
				"name": "三态齐备",
				"reviewResults": mustJSON([]repo.ReviewResult{
					{Role: "rd", CapabilityID: "rd.prd-review", State: repo.ReviewStateAgree, Round: 0, At: 1},
					{Role: "fe", CapabilityID: "fe.prd-review", State: repo.ReviewStateUnagree, Round: 0, At: 2},
					{Role: "arch", CapabilityID: "arch.prd-review", State: repo.ReviewStateReviewing, Round: 0, At: 3},
				}),
				"reviewers": mustJSON([]repo.ReviewerDef{
					{Role: "rd", CapabilityID: "rd.prd-review"},
					{Role: "fe", CapabilityID: "fe.prd-review"},
					{Role: "arch", CapabilityID: "arch.prd-review"},
				}),
				"go": goSummary(`[{"role":"rd","capabilityId":"rd.prd-review","state":"agree","round":0,"at":1},{"role":"fe","capabilityId":"fe.prd-review","state":"unagree","round":0,"at":2},{"role":"arch","capabilityId":"arch.prd-review","state":"reviewing","round":0,"at":3}]`),
			},
		},
		"inReviewContext": []map[string]any{
			{"name": "review 相位", "state": repo.PhaseReview, "reviewers": `[]`, "role": "rd", "go": goInReview(repo.PhaseReview, `[]`, "rd")},
			{"name": "respond 相位", "state": repo.PhaseRespond, "reviewers": `[]`, "role": "rd", "go": goInReview(repo.PhaseRespond, `[]`, "rd")},
			{"name": "exec 相位评审者身份", "state": repo.PhaseExec,
				"reviewers": `[{"role":"rd","capabilityId":"rd.prd-review"}]`, "role": "rd",
				"go": goInReview(repo.PhaseExec, `[{"role":"rd","capabilityId":"rd.prd-review"}]`, "rd")},
			{"name": "exec 相位非评审者", "state": repo.PhaseExec,
				"reviewers": `[{"role":"rd","capabilityId":"rd.prd-review"}]`, "role": "fe",
				"go": goInReview(repo.PhaseExec, `[{"role":"rd","capabilityId":"rd.prd-review"}]`, "fe")},
			{"name": "空相位评审者身份", "state": repo.PhaseEmpty,
				"reviewers": `[{"role":"qa","capabilityId":"qa.accept","scopeExpr":"full"}]`, "role": "qa",
				"go": goInReview(repo.PhaseEmpty, `[{"role":"qa","capabilityId":"qa.accept","scopeExpr":"full"}]`, "qa")},
		},
		"reviewAnchorGuard": []map[string]any{
			{"name": "评审对话期 document 未挂产物=拦截", "issueType": domain.IssueTypeIssue,
				"anchorKind": domain.IssueAnchorDocument, "artifactId": "", "reviewCtx": true, "go": service.ReviewAnchorGuardBlocked(domain.IssueTypeIssue, domain.IssueAnchorDocument, "", true)},
			{"name": "已挂产物=放行", "issueType": domain.IssueTypeIssue,
				"anchorKind": domain.IssueAnchorDocument, "artifactId": "art-1", "reviewCtx": true, "go": service.ReviewAnchorGuardBlocked(domain.IssueTypeIssue, domain.IssueAnchorDocument, "art-1", true)},
			{"name": "range 锚天然挂产物=放行", "issueType": domain.IssueTypeIssue,
				"anchorKind": domain.IssueAnchorRange, "artifactId": "", "reviewCtx": true, "go": service.ReviewAnchorGuardBlocked(domain.IssueTypeIssue, domain.IssueAnchorRange, "", true)},
			{"name": "discussion 类不受限=放行", "issueType": domain.IssueTypeDiscussion,
				"anchorKind": domain.IssueAnchorDocument, "artifactId": "", "reviewCtx": true, "go": service.ReviewAnchorGuardBlocked(domain.IssueTypeDiscussion, domain.IssueAnchorDocument, "", true)},
			{"name": "非评审上下文=放行", "issueType": domain.IssueTypeIssue,
				"anchorKind": domain.IssueAnchorDocument, "artifactId": "", "reviewCtx": false, "go": service.ReviewAnchorGuardBlocked(domain.IssueTypeIssue, domain.IssueAnchorDocument, "", false)},
		},
		"laneSideExecCR": laneSideCases(),
	}
}

// laneSideCases 泳道侧成员判定真值表。feSkip=true 的样例为「已知双端分歧」：
// FE isLaneSideExecCR 不做 scope 单侧校验（以相邻成对约束替代，flowview.ts:359 注），
// scope 双侧命中时 Go=false / FE=true——FE 测试跳过此类样例并引用本注。
func laneSideCases() []map[string]any {
	def := func(nodeType string, gate int64, reviewers string, scopeExpr string) *repo.FlowDefNode {
		return &repo.FlowDefNode{NodeType: nodeType, Gate: gate, Reviewers: reviewers, ScopeExpr: scopeExpr}
	}
	rdReviewers := `[{"role":"rd","capabilityId":"rd.dev"}]`
	cases := []map[string]any{
		{"name": "rd 泳道 exec-CR（bus-08 形）", "nodeType": repo.NodeTypeProduce, "gate": int64(1),
			"reviewers": rdReviewers, "scopeExpr": repo.ScopeRD, "lane": repo.ScopeRD,
			"go": service.LaneSideExecCR(def(repo.NodeTypeProduce, 1, rdReviewers, repo.ScopeRD), repo.ScopeRD)},
		{"name": "同节点 fe 视角不成组", "nodeType": repo.NodeTypeProduce, "gate": int64(1),
			"reviewers": rdReviewers, "scopeExpr": repo.ScopeRD, "lane": repo.ScopeFE,
			"go": service.LaneSideExecCR(def(repo.NodeTypeProduce, 1, rdReviewers, repo.ScopeRD), repo.ScopeFE),
			// 已知双端分歧：FE 无 scope 校验（lane 归属由相邻成对承载），此样例 FE 跳过
			"feSkip": "FE isLaneSideExecCR 无 scope 校验（flowview.ts:359 注——相邻成对约束替代单侧校验）"},
		{"name": "fe 泳道 exec-CR（bus-09 形）", "nodeType": repo.NodeTypeProduce, "gate": int64(1),
			"reviewers": rdReviewers, "scopeExpr": repo.ScopeFE, "lane": repo.ScopeFE,
			"go": service.LaneSideExecCR(def(repo.NodeTypeProduce, 1, rdReviewers, repo.ScopeFE), repo.ScopeFE)},
		{"name": "无评审位不成组", "nodeType": repo.NodeTypeProduce, "gate": int64(1),
			"reviewers": `[]`, "scopeExpr": repo.ScopeRD, "lane": repo.ScopeRD,
			"go": service.LaneSideExecCR(def(repo.NodeTypeProduce, 1, `[]`, repo.ScopeRD), repo.ScopeRD)},
		{"name": "非 produce 型不成组", "nodeType": repo.NodeTypeReview, "gate": int64(1),
			"reviewers": rdReviewers, "scopeExpr": repo.ScopeRD, "lane": repo.ScopeRD,
			"go": service.LaneSideExecCR(def(repo.NodeTypeReview, 1, rdReviewers, repo.ScopeRD), repo.ScopeRD)},
		{
			"name": "scope 双侧命中（full）", "nodeType": repo.NodeTypeProduce, "gate": int64(1),
			"reviewers": rdReviewers, "scopeExpr": repo.ScopeFull, "lane": repo.ScopeRD,
			"go": service.LaneSideExecCR(def(repo.NodeTypeProduce, 1, rdReviewers, repo.ScopeFull), repo.ScopeRD),
			// 已知双端分歧：FE 不校验 scope（相邻成对约束替代），此样例 FE 跳过
			"feSkip": "FE isLaneSideExecCR 无 scope 校验（flowview.ts:359 注——相邻成对约束替代单侧校验）",
		},
	}
	return cases
}

// tablesSection 词表快照（FE constants.ts 手抄表的单一样例源）。
func tablesSection() map[string]any {
	categories := make([]map[string]any, 0, len(domain.CategoryDefinitions))
	for _, def := range domain.CategoryDefinitions {
		categories = append(categories, map[string]any{
			"name":         def.Name,
			"allowedFlows": def.AllowedFlows,
			"defaultFlow":  def.DefaultFlow,
			"defaultScope": def.DefaultScope,
		})
	}
	return map[string]any{
		"categories":              categories,
		"notificationTargetTypes": service.NotificationTargetTypes(),
		"requirementStatuses": []string{
			domain.ReqStatusPending, domain.ReqStatusRunning, domain.ReqStatusDone, domain.ReqStatusCanceled,
		},
	}
}

// mustJSON 判据输入样例的 JSON 文本化（真值表内嵌原样字符串，FE 直接解析消费）。
func mustJSON(v any) string {
	b, err := utils.MarshalJSONText(v)
	if err != nil {
		panic(err) // 生成期装配缺陷，宁可失败暴露
	}
	return b
}

// goSummary SummarizeReviewResults 真值（FE summarizeReviews 计数镜像的锁）。
func goSummary(reviewResults string) map[string]any {
	summary, err := domain.SummarizeReviewResults(reviewResults)
	if err != nil {
		panic(err)
	}
	return map[string]any{
		"agree":     summary.Agree,
		"unagree":   summary.Unagree,
		"reviewing": summary.Reviewing,
	}
}

// goInReview InReviewContext 真值（FE inReviewCommentContext 镜像的锁）。
func goInReview(state string, reviewers string, role string) bool {
	ok, err := service.InReviewContext(&repo.FlowNode{State: state, Reviewers: reviewers}, role)
	if err != nil {
		panic(err)
	}
	return ok
}
