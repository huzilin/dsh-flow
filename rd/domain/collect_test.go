// T8 domain collect 聚合判定表驱动全分支单测（AC，无 IO 全覆盖）：
// collect 三段判定（先齐后判）× 同轮同角色覆写取最后 × 评审轮初始化/当前轮过滤 ×
// 激活相位（空转规则）× 结论协议装配（last_decision JSON schema）。
// 定稿出处：docs-tmp/capability-model-final.md §1.2/§1.2a/§1.3/§6 rule4。
package domain

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// reviewersJSON3 三评审位（bus-03 full scope 同构：arch/rd/fe）。
const reviewersJSON3 = `[{"role":"arch","capabilityId":"arch.prd-review"},{"role":"rd","capabilityId":"rd.prd-review"},{"role":"fe","capabilityId":"fe.prd-review"}]`

// entry 条目快捷构造。
func entry(role string, state string, round int64) repo.ReviewResult {
	return repo.ReviewResult{Role: role, CapabilityID: role + "-cap", State: state, Round: round, At: 100 + round}
}

// entriesJSON 条目集序列化。
func entriesJSON(t *testing.T, entries []repo.ReviewResult) string {
	t.Helper()
	text, err := utils.MarshalJSONText(entries)
	assert.NilError(t, err)
	return text
}

// TestUnit_CollectVerdict 全分支：未齐 pending（含个别 unagree 未齐场景）/
// 全 agree approved / 存在 unagree clarify / 空评审位空转直过 / 多轮历史不参与判定。
func TestUnit_CollectVerdict(t *testing.T) {
	cases := []struct {
		name      string
		reviewers string
		entries   []repo.ReviewResult
		want      CollectOutcome
	}{
		{"空评审位=空转直过 approved", `[]`, nil, CollectApproved},
		{"无人表态 pending", reviewersJSON3, nil, CollectPending},
		{"1/3 表态 pending", reviewersJSON3, []repo.ReviewResult{entry("arch", repo.ReviewStateAgree, 0)}, CollectPending},
		{"2/3 表态 pending", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateAgree, 0),
			entry("rd", repo.ReviewStateUnagree, 0),
		}, CollectPending},
		{"3/3 全 agree approved", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateAgree, 0),
			entry("rd", repo.ReviewStateAgree, 0),
			entry("fe", repo.ReviewStateAgree, 0),
		}, CollectApproved},
		{"3/3 存在 unagree clarify", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateAgree, 0),
			entry("rd", repo.ReviewStateUnagree, 0),
			entry("fe", repo.ReviewStateAgree, 0),
		}, CollectClarify},
		{"3/3 全 unagree clarify", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateUnagree, 0),
			entry("rd", repo.ReviewStateUnagree, 0),
			entry("fe", repo.ReviewStateUnagree, 0),
		}, CollectClarify},
		{"历史轮全 agree 不参与判定（当前轮未表态）", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateAgree, 0),
			entry("rd", repo.ReviewStateAgree, 0),
			entry("fe", repo.ReviewStateAgree, 0),
			entry("arch", repo.ReviewStateReviewing, 1),
			entry("rd", repo.ReviewStateReviewing, 1),
			entry("fe", repo.ReviewStateReviewing, 1),
		}, CollectPending},
		{"当前轮全 agree approved（重入轮）", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateUnagree, 0),
			entry("rd", repo.ReviewStateAgree, 0),
			entry("fe", repo.ReviewStateAgree, 0),
			entry("arch", repo.ReviewStateAgree, 1),
			entry("rd", repo.ReviewStateAgree, 1),
			entry("fe", repo.ReviewStateAgree, 1),
		}, CollectApproved},
		{"reviewing 条目不计入已表态", reviewersJSON3, []repo.ReviewResult{
			entry("arch", repo.ReviewStateAgree, 0),
			entry("rd", repo.ReviewStateReviewing, 0),
			entry("fe", repo.ReviewStateReviewing, 0),
		}, CollectPending},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CollectVerdict(tc.reviewers, entriesJSON(t, tc.entries))
			assert.NilError(t, err)
			assert.Equal(t, got, tc.want)
		})
	}

	// 空数组字段（实例化初始值 "[]"）与非法 JSON 两分支
	got, err := CollectVerdict(reviewersJSON3, "[]")
	assert.NilError(t, err)
	assert.Equal(t, got, CollectPending)
	_, err = CollectVerdict(reviewersJSON3, `{bad`)
	assert.Assert(t, err != nil, "review_results 非法 JSON 应报错")
	_, err = CollectVerdict(`{bad`, "[]")
	assert.Assert(t, err != nil, "reviewers 非法 JSON 应报错")
}

// TestUnit_UpsertReviewResult 同轮同角色覆写取最后：命中 (role, round) 覆写
// state/capabilityId/at；未命中追加；跨轮不互覆。
func TestUnit_UpsertReviewResult(t *testing.T) {
	entries := []repo.ReviewResult{entry("arch", repo.ReviewStateReviewing, 0)}
	entries = UpsertReviewResult(entries, repo.ReviewResult{
		Role: "arch", CapabilityID: "arch.prd-review", State: repo.ReviewStateAgree, Round: 0, At: 200,
	})
	assert.Equal(t, len(entries), 1, "同轮同角色覆写不追加")
	assert.Equal(t, entries[0].State, repo.ReviewStateAgree)
	assert.Equal(t, entries[0].At, int64(200))
	assert.Equal(t, entries[0].CapabilityID, "arch.prd-review")

	entries = UpsertReviewResult(entries, repo.ReviewResult{
		Role: "arch", CapabilityID: "arch.prd-review", State: repo.ReviewStateUnagree, Round: 0, At: 300,
	})
	assert.Equal(t, len(entries), 1, "覆写取最后")
	assert.Equal(t, entries[0].State, repo.ReviewStateUnagree)

	entries = UpsertReviewResult(entries, entry("rd", repo.ReviewStateAgree, 0))
	assert.Equal(t, len(entries), 2, "不同角色追加")

	entries = UpsertReviewResult(entries, entry("arch", repo.ReviewStateReviewing, 1))
	assert.Equal(t, len(entries), 3, "跨轮新条目追加（重入轮初始化）")
	current := CurrentRoundEntries(entries)
	assert.Equal(t, len(current), 1, "当前轮过滤")
	assert.Equal(t, current[0].Round, int64(1))
}

// TestUnit_InitReviewRound 评审轮初始化：位置全集 reviewing 同轮同刻。
func TestUnit_InitReviewRound(t *testing.T) {
	entries, err := InitReviewRound(reviewersJSON3, 2, 555)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 3)
	for _, item := range entries {
		assert.Equal(t, item.State, repo.ReviewStateReviewing)
		assert.Equal(t, item.Round, int64(2))
		assert.Equal(t, item.At, int64(555))
	}
	switch entries[0].Role {
	case "arch":
		assert.Equal(t, entries[0].CapabilityID, "arch.prd-review")
	default:
		t.Fatalf("条目顺序应随声明序，实测首角色 %q", entries[0].Role)
	}
	_, err = InitReviewRound(`{bad`, 0, 0)
	assert.Assert(t, err != nil)
}

// TestUnit_CurrentRound 当前轮推导：空集轮 0 / 最大 round。
func TestUnit_CurrentRound(t *testing.T) {
	assert.Equal(t, CurrentRound(nil), int64(0))
	assert.Equal(t, CurrentRound([]repo.ReviewResult{entry("a", "agree", 0), entry("b", "agree", 3)}), int64(3))
}

// TestUnit_ActivationPhase 激活相位：gate=entry / review=review /
// produce/align 按 entry 挂载空转（§6 rule4）。
func TestUnit_ActivationPhase(t *testing.T) {
	cases := []struct {
		name        string
		nodeType    string
		nodeActions string
		want        string
	}{
		{"gate 驻留 entry（拍板卡）", "gate", "[]", repo.PhaseEntry},
		{"review 直入评审相位", "review", "[]", repo.PhaseReview},
		{"produce entry 无挂载空转 exec", "produce",
			`[{"kind":"capability","capabilityId":"product-design","actions":["produce","revise"]}]`, repo.PhaseExec},
		{"align entry 挂载开场白驻留", "align",
			`[{"kind":"node","actionId":"align.opening","on":"entry","executor":"program","executorRole":"pm"}]`, repo.PhaseEntry},
		{"align entry 无挂载空转 exec", "align",
			`[{"kind":"node","actionId":"to-brd","on":"exec","executor":"subagent","executorRole":"pm"}]`, repo.PhaseExec},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ActivationPhase(tc.nodeType, tc.nodeActions)
			assert.NilError(t, err)
			assert.Equal(t, got, tc.want)
		})
	}
	_, err := ActivationPhase("produce", `{bad`)
	assert.Assert(t, err != nil)
}

// TestUnit_GateDecisionProtocol 结论协议装配与解析（migration 0003 schema）：
// branch 空省略键 / 非空携带；空串=未判定零值。
func TestUnit_GateDecisionProtocol(t *testing.T) {
	text, err := BuildGateDecision(GateVerdictApproved, "全 agree，collect 通过", 1234, "")
	assert.NilError(t, err)
	assert.Equal(t, text, `{"verdict":"approved","reason":"全 agree，collect 通过","at":1234}`)

	text, err = BuildGateDecision(GateVerdictRejected, "打回理由", 2345, "feat/x")
	assert.NilError(t, err)
	assert.Equal(t, text, `{"verdict":"rejected","reason":"打回理由","at":2345,"branch":"feat/x"}`)

	decision, err := ParseGateDecision(text)
	assert.NilError(t, err)
	assert.Equal(t, decision.Verdict, GateVerdictRejected)
	assert.Equal(t, decision.Reason, "打回理由")
	assert.Equal(t, decision.At, int64(2345))
	assert.Equal(t, decision.Branch, "feat/x")

	decision, err = ParseGateDecision("")
	assert.NilError(t, err)
	assert.Equal(t, decision, GateDecision{}, "空串=未判定零值")
}

// TestUnit_ReenterRoundEntries 澄清后重入新轮：agree 携结论进新轮（不重复打扰）、
// unagree 复位 reviewing、同刻同轮；全量替换=当前轮快照。
func TestUnit_ReenterRoundEntries(t *testing.T) {
	previous := []repo.ReviewResult{
		entry("arch", repo.ReviewStateAgree, 0),
		entry("rd", repo.ReviewStateUnagree, 0),
		entry("fe", repo.ReviewStateAgree, 0),
	}
	entries, err := ReenterRoundEntries(reviewersJSON3, entriesJSON(t, previous), 1, 999)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 3)
	byRole := make(map[string]repo.ReviewResult, 3)
	for _, item := range entries {
		assert.Equal(t, item.Round, int64(1))
		assert.Equal(t, item.At, int64(999))
		byRole[item.Role] = item
	}
	assert.Equal(t, byRole["arch"].State, repo.ReviewStateAgree, "agree 携结论进新轮")
	assert.Equal(t, byRole["fe"].State, repo.ReviewStateAgree)
	assert.Equal(t, byRole["rd"].State, repo.ReviewStateReviewing, "unagree 复位重新表态")
	assert.Equal(t, byRole["rd"].CapabilityID, "rd.prd-review", "复位条目能力回落声明位")

	// collect 闭环：复位后未齐 pending，rd 重报 agree 后 approved
	got, err := CollectVerdict(reviewersJSON3, entriesJSON(t, entries))
	assert.NilError(t, err)
	assert.Equal(t, got, CollectPending)
	entries = UpsertReviewResult(entries, repo.ReviewResult{
		Role: "rd", CapabilityID: "rd.prd-review", State: repo.ReviewStateAgree, Round: 1, At: 1000,
	})
	got, err = CollectVerdict(reviewersJSON3, entriesJSON(t, entries))
	assert.NilError(t, err)
	assert.Equal(t, got, CollectApproved, "重入轮全 agree approved")

	// 空历史（理论不达，防御）与非法 JSON
	entries, err = ReenterRoundEntries(reviewersJSON3, "[]", 1, 999)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 3)
	for _, item := range entries {
		assert.Equal(t, item.State, repo.ReviewStateReviewing)
	}
	_, err = ReenterRoundEntries(reviewersJSON3, `{bad`, 1, 999)
	assert.Assert(t, err != nil)
}
