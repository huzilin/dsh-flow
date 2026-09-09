// 任务族判据校验表驱动单测（T5 AC：produce/dev/review/respond/accept/create_reqs
// 有产物/表态与缺失两分支）+ 动作定位解析与任务族归派 + 泳道派生。
// domain 纯逻辑无 IO；env 门控涉库集成见 service/task_it_test.go。
package domain

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// TestCompleteCriteria_ProductFamily produce/dev/accept 族：期望产物逐一登记→放行，
// 部分缺失→逐类型列缺（artifact:<productType>）。
func TestCompleteCriteria_ProductFamily(t *testing.T) {
	expected := []string{"dev_notes", "code_mr"}
	// 全登记（多余类型无害）→ 满足
	got := CompleteCriteria(repo.TaskTypeDev, TaskFacts{
		Role:                   "rd",
		ExpectedProductTypes:   expected,
		RegisteredProductTypes: []string{"dev_notes", "code_mr", "rd_design"},
	})
	assert.DeepEqual(t, got, []string{})
	// 全缺 → 逐类型列出
	got = CompleteCriteria(repo.TaskTypeProduce, TaskFacts{
		Role:                   "pm",
		ExpectedProductTypes:   []string{"product_doc"},
		RegisteredProductTypes: []string{},
	})
	assert.DeepEqual(t, got, []string{"artifact:product_doc"})
	// 部分缺 → 只缺项列出
	got = CompleteCriteria(repo.TaskTypeDev, TaskFacts{
		Role:                   "rd",
		ExpectedProductTypes:   expected,
		RegisteredProductTypes: []string{"dev_notes"},
	})
	assert.DeepEqual(t, got, []string{"artifact:code_mr"})
	// accept 同表驱动分支（acceptance-test accept/retest → test_result/defect）
	got = CompleteCriteria(repo.TaskTypeAccept, TaskFacts{
		Role:                   "qa",
		ExpectedProductTypes:   []string{"test_result", "defect"},
		RegisteredProductTypes: []string{"defect"},
	})
	assert.DeepEqual(t, got, []string{"artifact:test_result"})
}

// TestCompleteCriteria_ReviewFamily review 族：该角色已表态（agree/unagree）→放行；
// reviewing/未表态 → 列缺 review_statement:<role>。
func TestCompleteCriteria_ReviewFamily(t *testing.T) {
	stated := func(state string) TaskFacts {
		return TaskFacts{
			Role:              "rd",
			ReviewStateByRole: map[string]string{"rd": state, "arch": "agree"},
		}
	}
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeReview, stated("agree")), []string{})
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeReview, stated("unagree")), []string{})
	// reviewing 不算表态
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeReview, stated("reviewing")), []string{"review_statement:rd"})
	// 无表态记录列缺
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeReview, TaskFacts{Role: "fe", ReviewStateByRole: map[string]string{}}),
		[]string{"review_statement:fe"})
}

// TestCompleteCriteria_RespondAndCreateReqs respond 族最小判据（该角色该节点
// open 清零）；create_reqs 最小判据 v1 恒放行；未知任务族 fail-closed。
func TestCompleteCriteria_RespondAndCreateReqs(t *testing.T) {
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeRespond, TaskFacts{Role: "pm", OpenIssueCount: 0}), []string{})
	// 有未闭环意见列缺
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeRespond, TaskFacts{Role: "pm", OpenIssueCount: 2}),
		[]string{"open_issues:pm(2)"})
	assert.DeepEqual(t, CompleteCriteria(repo.TaskTypeCreateReqs, TaskFacts{Role: "leader"}), []string{})
	// 未知任务族 fail-closed
	assert.DeepEqual(t, CompleteCriteria("unknown", TaskFacts{Role: "rd"}), []string{"unknown_task_type:unknown"})
}

// TestTaskTypeForAction 动作 id → 任务族映射（seed 21 能力动作词表归族抽样全表）。
func TestTaskTypeForAction(t *testing.T) {
	cases := []struct {
		actionID string
		want     string
	}{
		{"produce", repo.TaskTypeProduce},
		{"revise", repo.TaskTypeProduce},
		{"to-brd", repo.TaskTypeProduce},
		{"to-alignment", repo.TaskTypeProduce},
		{"dev", repo.TaskTypeDev},
		{"diagnosing-bugs", repo.TaskTypeDev},
		{"review", repo.TaskTypeReview},
		{"clarify", repo.TaskTypeRespond},
		{"fix_on_review", repo.TaskTypeRespond},
		{"confirm", repo.TaskTypeRespond},
		{"accept", repo.TaskTypeAccept},
		{"retest", repo.TaskTypeAccept},
		{"create_reqs", repo.TaskTypeCreateReqs},
	}
	for _, c := range cases {
		got, ok := TaskTypeForAction(c.actionID)
		assert.Assert(t, ok, "动作 %s 应在映射", c.actionID)
		assert.Equal(t, got, c.want, "动作 %s 归族", c.actionID)
	}
	_, ok := TaskTypeForAction("align.opening")
	assert.Assert(t, !ok, "program 类 node action 不在任务族映射")
}

// TestBranchForScope 泳道派生：rd/fe 直取，full 双泳道与 plan 空。
func TestBranchForScope(t *testing.T) {
	assert.Equal(t, BranchForScope(repo.ScopeRD), "rd")
	assert.Equal(t, BranchForScope(repo.ScopeFE), "fe")
	assert.Equal(t, BranchForScope(repo.ScopeFull), "")
	assert.Equal(t, BranchForScope(""), "")
}

// capabilityFixtures 单测能力装配（backend-dev/rd.prd-review 两能力抽样）。
func capabilityFixtures() map[string]CapabilityInfo {
	return map[string]CapabilityInfo{
		"backend-dev": {
			RoleID: "rd",
			Actions: []repo.ActionDef{
				{ActionID: "dev", On: "exec", Skill: "implement-techdesign", OutputProductTypes: []string{"dev_notes", "code_mr"}},
				{ActionID: "clarify", On: "respond"},
			},
		},
		"rd.prd-review": {
			RoleID: "rd",
			Actions: []repo.ActionDef{
				{ActionID: "review", On: "review", Skill: "prd-review-rd-skill"},
			},
		},
	}
}

// TestResolveTaskAction_CapabilityActivated 激活清单命中：role/task_type/产物期望派生。
func TestResolveTaskAction_CapabilityActivated(t *testing.T) {
	resolved, err := ResolveTaskAction("backend-dev.dev", "rd",
		`[{"kind":"capability","capabilityId":"backend-dev","actions":["dev","diagnosing-bugs"]}]`,
		`[]`, capabilityFixtures())
	assert.NilError(t, err)
	assert.Equal(t, resolved.ActionID, "backend-dev.dev")
	assert.Equal(t, resolved.CapabilityID, "backend-dev")
	assert.Equal(t, resolved.RoleID, "rd")
	assert.Equal(t, resolved.TaskType, repo.TaskTypeDev)
	assert.Equal(t, resolved.Source, ActionSourceNodeActions)
	assert.DeepEqual(t, resolved.OutputProductTypes, []string{"dev_notes", "code_mr"})
}

// TestResolveTaskAction_ReviewerActivated 评审位声明命中（review 能力合法激活源）。
func TestResolveTaskAction_ReviewerActivated(t *testing.T) {
	resolved, err := ResolveTaskAction("rd.prd-review.review", "rd",
		`[{"kind":"capability","capabilityId":"product-design","actions":["clarify"]}]`,
		`[{"role":"rd","capabilityId":"rd.prd-review"}]`, capabilityFixtures())
	assert.NilError(t, err)
	assert.Equal(t, resolved.TaskType, repo.TaskTypeReview)
	assert.Equal(t, resolved.Source, ActionSourceReviewers)
}

// TestResolveTaskAction_Rejected 拦截分支：未激活 / 角色不符 / 未知能力 / node action
// 响应者不符 —— 全部 ErrInvalidArgument（HTTP 400）。
func TestResolveTaskAction_Rejected(t *testing.T) {
	nodeActions := `[{"kind":"capability","capabilityId":"backend-dev","actions":["dev"]}]`
	reviewers := `[{"role":"cr","capabilityId":"code-review"}]`

	_, err := ResolveTaskAction("backend-dev.dev", "fe", nodeActions, reviewers, capabilityFixtures())
	assert.ErrorContains(t, err, "归属角色")
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "应 400")

	_, err = ResolveTaskAction("rd.prd-review.review", "rd", nodeActions, reviewers, capabilityFixtures())
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "评审位未声明该能力应未激活")

	_, err = ResolveTaskAction("backend-dev.clarify", "rd", nodeActions, reviewers, capabilityFixtures())
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "动作在能力词表但未激活（dev 才激活）——clarify 未激活")

	_, err = ResolveTaskAction("no-such-cap.dev", "rd", nodeActions, reviewers, capabilityFixtures())
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "未知能力 400")

	_, err = ResolveTaskAction("backend-dev.dev", "rd",
		`[{"kind":"node","actionId":"to-brd","executorRole":"pm"}]`, `[]`, capabilityFixtures())
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "capability 动作绕道 node 声明应未知能力 400")
}

// TestResolveTaskAction_NodeAction node action 定位：executor_role 校验与缺省。
func TestResolveTaskAction_NodeAction(t *testing.T) {
	resolved, err := ResolveTaskAction("to-brd", "pm",
		`[{"kind":"node","actionId":"to-brd","on":"exec","executorRole":"pm","skill":"to-brd"}]`,
		`[]`, capabilityFixtures())
	assert.NilError(t, err)
	assert.Equal(t, resolved.RoleID, "pm")
	assert.Equal(t, resolved.TaskType, repo.TaskTypeProduce)
	assert.Equal(t, resolved.Skill, "to-brd")
	assert.Equal(t, resolved.Source, ActionSourceNodeActions)

	_, err = ResolveTaskAction("to-brd", "rd",
		`[{"kind":"node","actionId":"to-brd","executorRole":"pm"}]`, `[]`, capabilityFixtures())
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "响应者角色不符 400")
}

// TestSplitCapabilityAction 能力×动作切分（capabilityId 可含点，首个点切分）。
func TestSplitCapabilityAction(t *testing.T) {
	capID, actID, ok := SplitCapabilityAction("rd.prd-review.review")
	assert.Assert(t, ok)
	assert.Equal(t, capID, "rd.prd-review")
	assert.Equal(t, actID, "review")

	_, _, ok = SplitCapabilityAction("to-brd")
	assert.Assert(t, !ok, "无点号=node action")

	_, _, ok = SplitCapabilityAction(".dev")
	assert.Assert(t, !ok, "首点非法")
}
