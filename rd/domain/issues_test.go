// T7 domain 状态机表驱动全分支单测（AC：合法迁移逐条 + 非法迁移逐条，
// 无 IO 全覆盖）：迁移表 × loop_count 双读法 × 操作者角色约束 × 锚点四级校验 ×
// escalate 置位 × reviewer_role/assignee 推导 × 锚点摘要。
package domain

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// issueStates 全状态集（枚举遍历用；overridden 态已按 09-12 裁决废除）。
var issueStates = []string{
	repo.IssueStateOpen,
	repo.IssueStatePendingConfirm,
	repo.IssueStateToFix,
	repo.IssueStateFixed,
	repo.IssueStateResolved,
}

// TestUnit_IssueTransition_Legal 合法迁移逐条：表内每条键断言目标态与副作用
// （与 issueTransitions 定义一一对应，防手误漂移）。
func TestUnit_IssueTransition_Legal(t *testing.T) {
	legal := []struct {
		name   string
		action string
		path   string
		from   string
		want   IssueEffect
	}{
		// reply：澄清回复完成→待复核；修复完成→已修复待复核；待复核态追加不回退。
		{"reply open→pending-confirm", IssueActionReply, "", repo.IssueStateOpen, IssueEffect{NextState: repo.IssueStatePendingConfirm, RoundCountDelta: 1, AppendReply: true, TouchLastReplyAt: true}},
		{"reply to-fix→fixed", IssueActionReply, "", repo.IssueStateToFix, IssueEffect{NextState: repo.IssueStateFixed, RoundCountDelta: 1, AppendReply: true, TouchLastReplyAt: true}},
		{"reply pending-confirm 停留", IssueActionReply, "", repo.IssueStatePendingConfirm, IssueEffect{NextState: repo.IssueStatePendingConfirm, RoundCountDelta: 1, AppendReply: true, TouchLastReplyAt: true}},
		{"reply fixed 停留", IssueActionReply, "", repo.IssueStateFixed, IssueEffect{NextState: repo.IssueStateFixed, RoundCountDelta: 1, AppendReply: true, TouchLastReplyAt: true}},
		// resolve path=fix：定性需修复 / 修复关闭申报。
		{"resolve fix open→to-fix", IssueActionResolve, IssueResolvePathFix, repo.IssueStateOpen, IssueEffect{NextState: repo.IssueStateToFix, SetResolveFamily: true}},
		{"resolve fix pending-confirm→to-fix", IssueActionResolve, IssueResolvePathFix, repo.IssueStatePendingConfirm, IssueEffect{NextState: repo.IssueStateToFix, SetResolveFamily: true}},
		{"resolve fix to-fix→fixed", IssueActionResolve, IssueResolvePathFix, repo.IssueStateToFix, IssueEffect{NextState: repo.IssueStateFixed, SetResolveFamily: true}},
		{"resolve fix fixed→fixed", IssueActionResolve, IssueResolvePathFix, repo.IssueStateFixed, IssueEffect{NextState: repo.IssueStateFixed, SetResolveFamily: true}},
		// resolve path=agree：协商一致关闭（确认内嵌）。
		{"resolve agree open→resolved", IssueActionResolve, IssueResolvePathAgree, repo.IssueStateOpen, IssueEffect{NextState: repo.IssueStateResolved, SetResolveFamily: true, ConfirmClose: true}},
		{"resolve agree pending-confirm→resolved", IssueActionResolve, IssueResolvePathAgree, repo.IssueStatePendingConfirm, IssueEffect{NextState: repo.IssueStateResolved, SetResolveFamily: true, ConfirmClose: true}},
		{"resolve agree to-fix→resolved", IssueActionResolve, IssueResolvePathAgree, repo.IssueStateToFix, IssueEffect{NextState: repo.IssueStateResolved, SetResolveFamily: true, ConfirmClose: true}},
		{"resolve agree fixed→resolved", IssueActionResolve, IssueResolvePathAgree, repo.IssueStateFixed, IssueEffect{NextState: repo.IssueStateResolved, SetResolveFamily: true, ConfirmClose: true}},
		// confirm：提评者复核确认（澄清侧/修复侧）。
		{"confirm pending-confirm→resolved", IssueActionConfirm, "", repo.IssueStatePendingConfirm, IssueEffect{NextState: repo.IssueStateResolved, ConfirmClose: true}},
		{"confirm fixed→resolved", IssueActionConfirm, "", repo.IssueStateFixed, IssueEffect{NextState: repo.IssueStateResolved, ConfirmClose: true}},
		// reopen：澄清 reopen / 修复 revise——loop_count 双读法共用列。
		{"reopen pending-confirm→open（澄清）", IssueActionReopen, "", repo.IssueStatePendingConfirm, IssueEffect{NextState: repo.IssueStateOpen, LoopCountDelta: 1, AppendReply: true}},
		{"reopen fixed→to-fix（revise）", IssueActionReopen, "", repo.IssueStateFixed, IssueEffect{NextState: repo.IssueStateToFix, LoopCountDelta: 1, AppendReply: true}},
	}
	for _, tc := range legal {
		effect, err := IssueTransition(tc.action, tc.path, tc.from)
		assert.NilError(t, err, "%s", tc.name)
		assert.DeepEqual(t, effect, tc.want)
	}
}

// TestUnit_IssueTransition_Illegal 非法迁移逐条：全组合（动作 × 路径 × 状态）中
// 不在合法迁移表内的每一格都必须命中 ErrInvalidTransition（409 映射哨兵）；
// resolve 缺 path / path 越集为入参非法语义（同样拒绝，错误信息带 path 要求）。
func TestUnit_IssueTransition_Illegal(t *testing.T) {
	actions := []string{IssueActionReply, IssueActionResolve, IssueActionConfirm, IssueActionReopen, "unknown-action"}
	paths := []string{"", IssueResolvePathFix, IssueResolvePathAgree, "bogus"}
	for _, action := range actions {
		for _, path := range paths {
			for _, from := range issueStates {
				if _, ok := issueTransitions[IssueTransitionKey{Action: action, ResolvePath: path, From: from}]; ok {
					continue // 合法格已在 Legal 用例逐条断言
				}
				effect, err := IssueTransition(action, path, from)
				if action == IssueActionResolve && path == "" {
					assert.Assert(t, err != nil, "resolve 缺 path 应拒绝 from=%s", from)
					continue // 入参缺失单独断言（不带迁移哨兵）
				}
				assert.Assert(t, err != nil, "应拒绝 %s(path=%s) from=%s", action, path, from)
				assert.Assert(t, strings.Contains(err.Error(), errs.ErrInvalidTransition.Error()),
					"%s(path=%s) from=%s 应以 ErrInvalidTransition 为 cause，实测 %v", action, path, from, err)
				assert.DeepEqual(t, effect, IssueEffect{})
			}
		}
	}
	// resolve 缺 path：入参语义错误（单独断言，错误信息要求 fix|agree）。
	_, err := IssueTransition(IssueActionResolve, "", repo.IssueStateOpen)
	assert.Assert(t, err != nil && strings.Contains(err.Error(), "fix|agree"))
}

// TestUnit_IssueLoopCount_DualReading loop_count 双读法（AC）：
// 同一列澄清期读=reopen 次数、修复期读=revise 次数——两处 reopen 各 +1，
// 其余动作不增减。
func TestUnit_IssueLoopCount_DualReading(t *testing.T) {
	loop := int64(0)
	// 澄清期：open→(reply)→pending-confirm→(reopen)→open 计 reopen 次数
	effect, err := IssueTransition(IssueActionReply, "", repo.IssueStateOpen)
	assert.NilError(t, err)
	loop += effect.LoopCountDelta
	assert.Equal(t, loop, int64(0))
	effect, err = IssueTransition(IssueActionReopen, "", repo.IssueStatePendingConfirm)
	assert.NilError(t, err)
	loop += effect.LoopCountDelta
	assert.Equal(t, loop, int64(1), "澄清 reopen 计数 +1")
	// 修复期：to-fix→(reply)→fixed→(reopen)→to-fix 计 revise 次数
	effect, err = IssueTransition(IssueActionReply, "", repo.IssueStateToFix)
	assert.NilError(t, err)
	loop += effect.LoopCountDelta
	effect, err = IssueTransition(IssueActionReopen, "", repo.IssueStateFixed)
	assert.NilError(t, err)
	loop += effect.LoopCountDelta
	assert.Equal(t, loop, int64(2), "修复 revise 计数 +1（同列双读法）")
	// 其余动作 loop_count 恒 0
	for _, tc := range []struct {
		action string
		path   string
		from   string
	}{
		{IssueActionReply, "", repo.IssueStateOpen},
		{IssueActionResolve, IssueResolvePathFix, repo.IssueStatePendingConfirm},
		{IssueActionResolve, IssueResolvePathAgree, repo.IssueStateToFix},
		{IssueActionConfirm, "", repo.IssueStateFixed},
	} {
		effect, err := IssueTransition(tc.action, tc.path, tc.from)
		assert.NilError(t, err)
		assert.Equal(t, effect.LoopCountDelta, int64(0), "%s 不动 loop_count", tc.action)
	}
}

// TestUnit_ValidateIssueActor 操作者角色约束全分支：resolve=assignee（未指派拒绝、
// 他人拒绝）；confirm/reopen=提评者（reviewer_role 优先，缺省 author）；reply 不设限。
func TestUnit_ValidateIssueActor(t *testing.T) {
	view := IssueView{State: repo.IssueStateOpen, Author: "arch", ReviewerRole: "arch", Assignee: "rd"}
	// resolve：assignee 本人允许
	assert.NilError(t, ValidateIssueActor(IssueActionResolve, "rd", view))
	// resolve：非 assignee / 未指派 → ErrInvalidTransition
	err := ValidateIssueActor(IssueActionResolve, "fe", view)
	assert.Assert(t, errs.Is(err, errs.ErrInvalidTransition), "非 assignee 应拒绝，实测 %v", err)
	err = ValidateIssueActor(IssueActionResolve, "rd", IssueView{State: repo.IssueStateOpen, Author: "arch"})
	assert.Assert(t, errs.Is(err, errs.ErrInvalidTransition), "未指派 assignee 应拒绝，实测 %v", err)
	// confirm/reopen：reviewer_role 优先命中
	assert.NilError(t, ValidateIssueActor(IssueActionConfirm, "arch", view))
	assert.NilError(t, ValidateIssueActor(IssueActionReopen, "arch", view))
	// confirm/reopen：reviewer_role 缺省回退 author
	noReviewer := IssueView{State: repo.IssueStatePendingConfirm, Author: "qa", Assignee: "rd"}
	assert.NilError(t, ValidateIssueActor(IssueActionConfirm, "qa", noReviewer))
	assert.NilError(t, ValidateIssueActor(IssueActionReopen, "qa", noReviewer))
	// confirm/reopen：他人拒绝
	err = ValidateIssueActor(IssueActionConfirm, "rd", view)
	assert.Assert(t, errs.Is(err, errs.ErrInvalidTransition))
	err = ValidateIssueActor(IssueActionReopen, "pm", noReviewer)
	assert.Assert(t, errs.Is(err, errs.ErrInvalidTransition))
	// reply：任何角色允许（对话双方追加语义）
	for _, role := range []string{"arch", "rd", "fe", "qa", "user"} {
		assert.NilError(t, ValidateIssueActor(IssueActionReply, role, view), "reply 不设限: %s", role)
	}
}

// TestUnit_ValidateIssueAnchor 锚点四级校验表驱动：kind 与字段一致性、
// block 必带 lang、范围倒挂、冗余字段、未知 kind（AC）。
func TestUnit_ValidateIssueAnchor(t *testing.T) {
	valid := []struct {
		name                                           string
		kind, artifactID, lang                         string
		line, startLine, endLine, blockStart, blockEnd int64
	}{
		{"document 无锚字段", IssueAnchorDocument, "", "", 0, 0, 0, 0, 0},
		{"range", IssueAnchorRange, "art-1", "", 0, 10, 20, 0, 0},
		{"range 单行等价（划词单行）", IssueAnchorRange, "art-1", "", 0, 7, 7, 0, 0},
	}
	for _, tc := range valid {
		err := ValidateIssueAnchor(tc.kind, tc.artifactID, tc.line, tc.startLine, tc.endLine, tc.blockStart, tc.blockEnd, tc.lang)
		assert.NilError(t, err, "%s", tc.name)
	}
	invalid := []struct {
		name                                           string
		kind, artifactID, lang                         string
		line, startLine, endLine, blockStart, blockEnd int64
	}{
		{"未知 kind", "heading", "art-1", "", 0, 0, 0, 0, 0},
		{"line 缺 artifact", IssueAnchorLine, "", "", 42, 0, 0, 0, 0},
		{"line 缺行号", IssueAnchorLine, "art-1", "", 0, 0, 0, 0, 0},
		{"line 行号 0", IssueAnchorLine, "art-1", "", 0, 0, 0, 0, 0},
		{"line 冗余 range 字段", IssueAnchorLine, "art-1", "", 42, 1, 2, 0, 0},
		{"line 冗余 block 字段", IssueAnchorLine, "art-1", "sql", 42, 0, 0, 1, 2},
		{"range 缺 artifact", IssueAnchorRange, "", "", 0, 1, 2, 0, 0},
		{"range 缺 end", IssueAnchorRange, "art-1", "", 0, 1, 0, 0, 0},
		{"range 倒挂", IssueAnchorRange, "art-1", "", 0, 20, 10, 0, 0},
		{"range 冗余 line 字段", IssueAnchorRange, "art-1", "", 3, 1, 2, 0, 0},
		{"range 冗余 lang", IssueAnchorRange, "art-1", "sql", 0, 1, 2, 0, 0},
		{"block 缺 artifact", IssueAnchorBlock, "", "sql", 0, 0, 0, 1, 2},
		{"block 缺 lang", IssueAnchorBlock, "art-2", "", 0, 0, 0, 42, 58},
		{"block 倒挂", IssueAnchorBlock, "art-2", "go", 0, 0, 0, 58, 42},
		{"block 冗余 range 字段", IssueAnchorBlock, "art-2", "go", 0, 1, 2, 42, 58},
		{"document 冗余行号", IssueAnchorDocument, "", "", 5, 0, 0, 0, 0},
		{"document 冗余 lang", IssueAnchorDocument, "", "sql", 0, 0, 0, 0, 0},
	}
	for _, tc := range invalid {
		err := ValidateIssueAnchor(tc.kind, tc.artifactID, tc.line, tc.startLine, tc.endLine, tc.blockStart, tc.blockEnd, tc.lang)
		assert.Assert(t, err != nil, "%s 应拒绝", tc.name)
	}
}

// TestUnit_AnchorKindDeprecated 锚点词表收紧（注记-L3，2026-09-10 拍板：仅
// 划词 range + 文档级 document）——line/block 一律拒绝并提示废弃。
func TestUnit_AnchorKindDeprecated(t *testing.T) {
	err := ValidateIssueAnchor(IssueAnchorLine, "art-1", 42, 0, 0, 0, 0, "")
	assert.Assert(t, err != nil && strings.Contains(err.Error(), "已废弃"), "line 锚应报废弃: %v", err)
	err = ValidateIssueAnchor(IssueAnchorBlock, "art-2", 0, 0, 0, 42, 58, "go")
	assert.Assert(t, err != nil && strings.Contains(err.Error(), "已废弃"), "block 锚应报废弃: %v", err)
}

// TestUnit_CommentsClosure 评论闭环判定表驱动（注记-L1：collect/approve 前置；
// 09-12 裁决 overridden 态废除，override 动作置 resolved）——
// 角色过滤（reviewer_role）、类型过滤（notification/discussion 不参与）、
// 闭环状态集（resolved；open/to-fix/fixed/pending-confirm 未闭环）、
// 空集=闭环。
func TestUnit_CommentsClosure(t *testing.T) {
	role := func(r string) *string { return &r }
	issues := []repo.Issue{
		{IssueType: IssueTypeIssue, ReviewerRole: role("arch"), State: repo.IssueStateOpen},
		{IssueType: IssueTypeIssue, ReviewerRole: role("arch"), State: repo.IssueStateResolved},
		{IssueType: IssueTypeIssue, ReviewerRole: role("rd"), State: repo.IssueStateToFix},
		{IssueType: IssueTypeIssue, ReviewerRole: role("qa"), State: repo.IssueStateFixed},
		{IssueType: IssueTypeNotification, ReviewerRole: role("arch"), State: repo.IssueStateOpen},
		{IssueType: IssueTypeDiscussion, ReviewerRole: role("arch"), State: repo.IssueStatePendingConfirm},
		{IssueType: IssueTypeIssue, State: repo.IssueStateOpen},
	}
	closed, n := RoleCommentsClosure(issues, "arch")
	assert.Assert(t, !closed && n == 1, "arch=resolved+open，1 条未闭环，got %d", n)
	closed, n = RoleCommentsClosure(issues, "rd")
	assert.Assert(t, !closed && n == 1, "rd to-fix 未闭环，got %d", n)
	closed, n = RoleCommentsClosure(issues, "qa")
	assert.Assert(t, !closed && n == 1, "qa fixed 未复核=未闭环，got %d", n)
	closed, n = RoleCommentsClosure(issues, "fe")
	assert.Assert(t, closed && n == 0, "fe 空集=闭环（vacuous）")
	closed, n = NodeCommentsClosure(issues)
	assert.Assert(t, !closed && n == 4, "节点级未闭环=arch open+rd to-fix+qa fixed+无归属 open（notification/discussion 排除），got %d", n)
}

// TestUnit_Escalate escalate 置位（幂等）与阈值判定分支（spec §2 reopen≥5 口径）。
func TestUnit_Escalate(t *testing.T) {
	assert.Equal(t, MarkEscalate(), int64(1))
	assert.Assert(t, AutoEscalateNeeded(5, DefaultEscalateLoopThreshold), "reopen≥5 达阈值")
	assert.Assert(t, AutoEscalateNeeded(6, DefaultEscalateLoopThreshold), "超阈值仍上升")
	assert.Assert(t, !AutoEscalateNeeded(4, DefaultEscalateLoopThreshold), "未达阈值")
	assert.Assert(t, !AutoEscalateNeeded(5, 0), "阈值 0=判定关闭（防御）")
}

// TestUnit_DeriveReviewerRole 评审归属推导：发表者在评审位命中/不在为空/JSON 损坏报错。
func TestUnit_DeriveReviewerRole(t *testing.T) {
	reviewers := `[{"role":"arch","capabilityId":"arch.prd-review"},{"role":"rd","capabilityId":"backend-dev.rd-review","scopeExpr":"rd|full"}]`
	got, err := DeriveReviewerRole("arch", reviewers)
	assert.NilError(t, err)
	assert.Equal(t, got, "arch")
	got, err = DeriveReviewerRole("qa", reviewers)
	assert.NilError(t, err)
	assert.Equal(t, got, "", "非评审位=未推导")
	_, err = DeriveReviewerRole("arch", "{broken")
	assert.Assert(t, err != nil, "reviewers JSON 损坏应报错")
}

// TestUnit_DeriveAssignee 解决责任人推导优先级：显式 > L1 锚点产物登记人 > L2 节点产出角色 > 未指派。
func TestUnit_DeriveAssignee(t *testing.T) {
	assert.Equal(t, DeriveAssignee("fe", "rd", "rd"), "fe", "显式指派优先")
	assert.Equal(t, DeriveAssignee("", "rd", "rd"), "rd", "L1 锚点产物作者")
	assert.Equal(t, DeriveAssignee("", "", "rd"), "rd", "L2 节点产出角色")
	assert.Equal(t, DeriveAssignee("", "", ""), "", "全落空=未指派")
}

// TestUnit_NodeExecRole 节点产出角色推导：exec 能力归属命中/无 exec/JSON 损坏。
func TestUnit_NodeExecRole(t *testing.T) {
	actions := `[
		{"kind":"capability","capabilityId":"backend-design","actions":["revise","produce"],"scopeExpr":""},
		{"kind":"node","actionId":"align.opening","on":"entry","executor":"subagent"}
	]`
	vocab := map[string]CapabilityInfo{
		"backend-design":  {RoleID: "rd", Actions: []repo.ActionDef{{ActionID: "produce", On: "exec"}, {ActionID: "revise", On: "exec"}, {ActionID: "clarify", On: "respond"}}},
		"acceptance-test": {RoleID: "qa", Actions: []repo.ActionDef{{ActionID: "accept", On: "exec"}}},
	}
	got, err := NodeExecRole(actions, vocab)
	assert.NilError(t, err)
	assert.Equal(t, got, "rd")
	noExec := `[{"kind":"capability","capabilityId":"backend-dev","actions":["review"]}]`
	got, err = NodeExecRole(noExec, vocab)
	assert.NilError(t, err)
	assert.Equal(t, got, "", "无 exec 能力=空")
	_, err = NodeExecRole("[broken", vocab)
	assert.Assert(t, err != nil)
}

// TestUnit_SummarizeIssueAnchor 锚点摘要四级 + 产物前缀。
func TestUnit_SummarizeIssueAnchor(t *testing.T) {
	assert.Equal(t, SummarizeIssueAnchor(IssueAnchorDocument, "", 0, 0, 0, 0, 0, ""), "document")
	assert.Equal(t, SummarizeIssueAnchor(IssueAnchorLine, "art-1", 42, 0, 0, 0, 0, ""), "art-1 line 42")
	assert.Equal(t, SummarizeIssueAnchor(IssueAnchorRange, "art-1", 0, 10, 20, 0, 0, ""), "art-1 range 10-20")
	assert.Equal(t, SummarizeIssueAnchor(IssueAnchorBlock, "art-2", 0, 0, 0, 42, 58, "sql"), "art-2 block 42-58(sql)")
}

// TestUnit_ValidIssueTypeState 枚举校验：issue_type 三值+空默认；state 五值
// （09-12 裁决废除 overridden，override 动作置 resolved）。
func TestUnit_ValidIssueTypeState(t *testing.T) {
	for _, ok := range []string{"", IssueTypeIssue, IssueTypeNotification, IssueTypeDiscussion} {
		assert.Assert(t, ValidIssueType(ok), "issue_type %q 合法", ok)
	}
	assert.Assert(t, !ValidIssueType("bug"))
	for _, ok := range issueStates {
		assert.Assert(t, ValidIssueState(ok), "state %q 合法", ok)
	}
	assert.Assert(t, !ValidIssueState("overridden"), "overridden 态已废除（09-12 裁决）")
	assert.Assert(t, !ValidIssueState("closed"))
}

// TestUnit_OwnerRoleWithFallback owner 推导收口（C5-D2）：exec 能力归属角色优先，
// 空=回落需求 owner。
func TestUnit_OwnerRoleWithFallback(t *testing.T) {
	if got := OwnerRoleWithFallback("rd", "pm"); got != "rd" {
		t.Fatalf("exec 角色应优先: got %q", got)
	}
	if got := OwnerRoleWithFallback("", "pm"); got != "pm" {
		t.Fatalf("空 exec 应回落需求 owner: got %q", got)
	}
	if got := OwnerRoleWithFallback("", ""); got != "" {
		t.Fatalf("双空应返回空串（调用方跳过派发）: got %q", got)
	}
}
