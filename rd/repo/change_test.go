// 变更同源派生核单测（纯函数，免库）：构造器映射、旧值文本提取、同值不落审计、
// Silent/Who 修饰、Null 变体、空集/重复字段装配缺陷。
package repo

import (
	"testing"

	"gotest.tools/v3/assert"
)

// changeNode 派生核基线行（覆盖全部可变字段的非零/零值组合）。
func changeNode() *FlowNode {
	return &FlowNode{
		NodeID:          "n-01",
		State:           PhaseExec,
		ReviewResults:   `[{"role":"rd","capabilityId":"rd.dev","state":"agree","round":0,"at":100}]`,
		RolledBackCount: 2,
		ReopenSource:    "",
		Reviewers:       `[{"role":"rd","capabilityId":"rd.dev"}]`,
		NodeOrder:       5,
		EnteredAt:       int64Ptr(1234),
		LastDecision:    strPtr(`{"verdict":"rejected"}`),
		PendingScope:    strPtr("full"),
		FeatureBranch:   strPtr("feat/a"),
	}
}

func strPtr(s string) *string { return &s }

func int64Ptr(v int64) *int64 { return &v }

// TestUnit_NodeChangeDerive_MappingAndOrder 构造器 → 列名/审计字段名映射正确，
// 审计行序=声明序，updates 与审计同源产出。
func TestUnit_NodeChangeDerive_MappingAndOrder(t *testing.T) {
	node := changeNode()
	node.LastDecision = nil
	node.PendingScope = nil
	node.EnteredAt = nil

	updates, audits, err := deriveNodeChange(node, "plugin",
		NodeState(PhaseReview),
		EnteredAt(9000),
		ReviewResults("[]"),
		RolledBackCount(3),
		ReopenSource("downstream-reject"),
		LastDecision(`{"verdict":"approved"}`),
		PendingScope("rd"),
		Reviewers(`[{"role":"qa","capabilityId":"qa.accept"}]`),
		NodeOrder(7),
		FeatureBranch("feat/b"),
	)
	assert.NilError(t, err)

	// updates：全部声明字段无条件写入（列名口径），不含 updated_at（由乐观锁 helper 维护）
	assert.Equal(t, len(updates), 10, "%v", updates)
	assert.Equal(t, updates["state"], PhaseReview)
	assert.Equal(t, updates["entered_at"], int64(9000))
	assert.Equal(t, updates["review_results"], "[]")
	assert.Equal(t, updates["rolled_back_count"], int64(3))
	assert.Equal(t, updates["reopen_source"], "downstream-reject")
	assert.Equal(t, updates["last_decision"], `{"verdict":"approved"}`)
	assert.Equal(t, updates["pending_scope"], "rd")
	assert.Equal(t, updates["reviewers"], `[{"role":"qa","capabilityId":"qa.accept"}]`)
	assert.Equal(t, updates["node_order"], int64(7))
	assert.Equal(t, updates["feature_branch"], "feat/b")

	// 审计：行序=声明序，字段名=camelCase，旧值取自行基线
	assert.Equal(t, len(audits), 10)
	wantOrder := []struct {
		field, old, new string
	}{
		{"state", PhaseExec, PhaseReview},
		{"enteredAt", "", "9000"},
		{"reviewResults", node.ReviewResults, "[]"},
		{"rolledBackCount", "2", "3"},
		{"reopenSource", "", "downstream-reject"},
		{"lastDecision", "", `{"verdict":"approved"}`},
		{"pendingScope", "", "rd"},
		{"reviewers", node.Reviewers, `[{"role":"qa","capabilityId":"qa.accept"}]`},
		{"nodeOrder", "5", "7"},
		{"featureBranch", "feat/a", "feat/b"},
	}
	for i, w := range wantOrder {
		assert.Equal(t, audits[i].Table, "flow_nodes")
		assert.Equal(t, audits[i].RowID, "n-01")
		assert.Equal(t, audits[i].Action, "UPDATE")
		assert.Equal(t, *audits[i].Field, w.field, "audit[%d]", i)
		assert.Equal(t, *audits[i].OldValue, w.old, "audit[%d] %s", i, w.field)
		assert.Equal(t, *audits[i].NewValue, w.new, "audit[%d] %s", i, w.field)
		assert.Equal(t, audits[i].ByWho, "plugin", "audit[%d]", i)
	}
}

// TestUnit_NodeChangeDerive_SameValueNoAudit 同值写：updates 照写、审计不落
// （替代原手写 if old != new 条件）。
func TestUnit_NodeChangeDerive_SameValueNoAudit(t *testing.T) {
	node := changeNode() // State=exec、RolledBackCount=2、FeatureBranch=feat/a
	updates, audits, err := deriveNodeChange(node, "plugin",
		NodeState(PhaseExec),
		RolledBackCount(2),
		FeatureBranch("feat/a"),
		ReviewResults(node.ReviewResults),
	)
	assert.NilError(t, err)
	assert.Equal(t, len(updates), 4, "同值字段 updates 照写: %v", updates)
	assert.Equal(t, updates["state"], PhaseExec)
	assert.Equal(t, updates["rolled_back_count"], int64(2))
	assert.Equal(t, len(audits), 0, "同值审计不落: %v", audits)
}

// TestUnit_NodeChangeDerive_SilentAndWho Silent=写而不审计；Who=单条署名覆盖，
// 默认署名仍为 Apply 的 byWho。
func TestUnit_NodeChangeDerive_SilentAndWho(t *testing.T) {
	node := changeNode()
	node.LastDecision = strPtr(`{"verdict":"rejected"}`)
	node.PendingScope = nil

	updates, audits, err := deriveNodeChange(node, "plugin",
		Silent(LastDecisionNull()),     // 写 NULL 不审计（被拒结论已由拍板审计回放）
		Silent(PendingScope("full")),   // 写标记不审计（清标记时由 reproject 审计）
		Who("rd", ReviewResults("[]")), // 表态审计署名=表态角色
		NodeState(PhaseExec),           // 默认署名=plugin（old=exec 同值也不落）
	)
	assert.NilError(t, err)
	assert.Equal(t, updates["last_decision"], nil)
	assert.Equal(t, updates["pending_scope"], "full")
	assert.Equal(t, updates["review_results"], "[]")
	assert.Equal(t, len(audits), 1)
	assert.Equal(t, *audits[0].Field, "reviewResults")
	assert.Equal(t, audits[0].ByWho, "rd")
}

// TestUnit_NodeChangeDerive_NullVariants Null 变体：列写 NULL、审计新值=空串；
// 旧值非空时照常落审计。
func TestUnit_NodeChangeDerive_NullVariants(t *testing.T) {
	node := changeNode() // EnteredAt=1234、LastDecision/PendingScope 非空
	updates, audits, err := deriveNodeChange(node, "system",
		EnteredAtNull(),
		LastDecisionNull(),
		PendingScopeNull(),
	)
	assert.NilError(t, err)
	assert.Equal(t, updates["entered_at"], nil)
	assert.Equal(t, updates["last_decision"], nil)
	assert.Equal(t, updates["pending_scope"], nil)
	assert.Equal(t, len(audits), 3)
	assert.Equal(t, *audits[0].OldValue, "1234")
	assert.Equal(t, *audits[0].NewValue, "")
	assert.Equal(t, *audits[1].OldValue, `{"verdict":"rejected"}`)
	assert.Equal(t, *audits[1].NewValue, "")
	assert.Equal(t, *audits[2].OldValue, "full")
	assert.Equal(t, *audits[2].NewValue, "")
}

// TestUnit_NodeChangeDerive_EmptyAndDuplicate 空变更集与重复字段=装配缺陷，报错。
func TestUnit_NodeChangeDerive_EmptyAndDuplicate(t *testing.T) {
	node := changeNode()

	_, _, err := deriveNodeChange(node, "plugin")
	assert.Assert(t, err != nil, "空变更集应报错")

	_, _, err = deriveNodeChange(node, "plugin", NodeState(PhaseExec), NodeState(PhaseReview))
	assert.Assert(t, err != nil, "重复字段应报错")
}
