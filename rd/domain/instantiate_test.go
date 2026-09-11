// domain 单测（TestUnit_ 前缀约定）：scope 判定 / 创建 scope 归一 / 实例化投影
// （过滤 + 重排 + 评审位与激活清单裁剪）/ 依赖环检测 / my_task 三向投影。
package domain_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// TestUnit_ScopeMatches 渲染集合表达式判定：”=any / 集合命中 / 空 scope 仅空表达式。
func TestUnit_ScopeMatches(t *testing.T) {
	cases := []struct {
		expr  string
		scope string
		want  bool
	}{
		{"", "full", true},
		{"", "rd", true},
		{"", "fe", true},
		{"", "", true},
		{"full", "full", true},
		{"full", "rd", false},
		{"rd|full", "rd", true},
		{"rd|full", "full", true},
		{"rd|full", "fe", false},
		{"fe|full", "fe", true},
		{"fe|full", "rd", false},
		{"rd|fe", "rd", true},
		{"rd|fe", "fe", true},
		{"rd|fe", "full", false},
		{"full", "", false},
		{"rd|full", "", false},
	}
	for _, tc := range cases {
		assert.Equal(t, domain.ScopeMatches(tc.expr, tc.scope), tc.want,
			"ScopeMatches(%q, %q)", tc.expr, tc.scope)
	}
}

// TestUnit_ResolveScope 创建 scope 归一：空=首选项 / 越集报错 / 无 scope 流程。
func TestUnit_ResolveScope(t *testing.T) {
	business := []string{"full", "rd", "fe"}
	task := []string{"rd", "fe"}

	got, err := domain.ResolveScope(business, "")
	assert.NilError(t, err)
	assert.Equal(t, got, "full", "空=scope_options 首值")

	got, err = domain.ResolveScope(task, "")
	assert.NilError(t, err)
	assert.Equal(t, got, "rd", "task 默认首值 rd")

	got, err = domain.ResolveScope(business, "fe")
	assert.NilError(t, err)
	assert.Equal(t, got, "fe")

	_, err = domain.ResolveScope(business, "be")
	assert.ErrorContains(t, err, "越出可选集", "越集非法")

	got, err = domain.ResolveScope(nil, "")
	assert.NilError(t, err)
	assert.Equal(t, got, "", "plan 无 scope")

	_, err = domain.ResolveScope(nil, "full")
	assert.ErrorContains(t, err, "无 scope", "plan 携带显式 scope 非法")
}

// defNode 投影测试构造器（只含投影消费字段）。
func defNode(flow, tnid string, order int64, scope string, reviewers []repo.ReviewerDef, actions []repo.NodeActionDef) repo.FlowDefNode {
	return repo.FlowDefNode{
		FlowDefsID:     flow,
		TemplateNodeID: tnid,
		NodeOrder:      order,
		NodeKey:        "key-" + tnid,
		NodeType:       repo.NodeTypeProduce,
		ScopeExpr:      scope,
		Reviewers:      mustJSON(reviewers),
		NodeActions:    mustJSON(actions),
	}
}

// TestUnit_ProjectFlowNodes 过滤 + 重排 + 评审位/激活清单同函数裁剪。
func TestUnit_ProjectFlowNodes(t *testing.T) {
	defs := []repo.FlowDefNode{
		defNode("business", "bus-00", 1, "", nil, nil),
		defNode("business", "bus-01", 2, "", nil, nil),
		defNode("business", "bus-04", 5, "rd|full", nil, nil),
		defNode("business", "bus-06", 7, "fe|full", nil, nil),
		defNode("business", "bus-10", 11, "full", nil, nil),
		// 他流程定义节点不参与投影
		defNode("plan", "pln-d0", 1, "", nil, nil),
	}
	// 评审位与条目级 scope（bus-03 形态复刻）
	rewritten := defNode("business", "bus-03", 4, "",
		[]repo.ReviewerDef{
			{Role: "arch", CapabilityID: "arch.prd-review"},
			{Role: "rd", CapabilityID: "rd.prd-review", ScopeExpr: "rd|full"},
			{Role: "fe", CapabilityID: "fe.prd-review", ScopeExpr: "full"},
		},
		[]repo.NodeActionDef{
			{Kind: repo.NodeActionKindCapability, CapabilityID: "product-design", Actions: []string{"clarify"}},
			{Kind: repo.NodeActionKindCapability, CapabilityID: "backend-design", ScopeExpr: "rd|full", Actions: []string{"produce"}},
			{Kind: repo.NodeActionKindCapability, CapabilityID: "frontend-design", ScopeExpr: "fe|full", Actions: []string{"produce"}},
		})
	defs = append(defs, rewritten)

	// full：全部渲染（bus-00,01,03,04,06,10），重排 1..6，被裁不占位
	projected, err := domain.ProjectFlowNodes("business", "full", defs)
	assert.NilError(t, err)
	assert.DeepEqual(t, projectedIDs(projected), []string{"bus-00", "bus-01", "bus-03", "bus-04", "bus-06", "bus-10"})
	for i, node := range projected {
		assert.Equal(t, node.NodeOrder, int64(i+1), "full 序号连续: %s", node.DefNodeID)
	}
	// full 下评审位全保留、激活清单全保留
	assert.DeepEqual(t, projected[2].Reviewers, mustJSON([]repo.ReviewerDef{
		{Role: "arch", CapabilityID: "arch.prd-review"},
		{Role: "rd", CapabilityID: "rd.prd-review", ScopeExpr: "rd|full"},
		{Role: "fe", CapabilityID: "fe.prd-review", ScopeExpr: "full"},
	}))
	assert.Equal(t, countActions(projected[2].NodeActions), 3)

	// rd：fe 侧被裁（bus-06/bus-10 不渲染），序号 1..4 连续
	projected, err = domain.ProjectFlowNodes("business", "rd", defs)
	assert.NilError(t, err)
	assert.DeepEqual(t, projectedIDs(projected), []string{"bus-00", "bus-01", "bus-03", "bus-04"})
	for i, node := range projected {
		assert.Equal(t, node.NodeOrder, int64(i+1), "rd 序号连续: %s", node.DefNodeID)
	}
	// rd 评审位：arch + rd（fe|full 位被裁）
	assert.DeepEqual(t, projected[2].Reviewers, mustJSON([]repo.ReviewerDef{
		{Role: "arch", CapabilityID: "arch.prd-review"},
		{Role: "rd", CapabilityID: "rd.prd-review", ScopeExpr: "rd|full"},
	}))
	// rd 激活清单：随节点条目 + rd 条目，fe 条目被裁
	assert.Equal(t, countActions(projected[2].NodeActions), 2)

	// fe：对称（bus-04 不渲染），rd 条目被裁
	projected, err = domain.ProjectFlowNodes("business", "fe", defs)
	assert.NilError(t, err)
	assert.DeepEqual(t, projectedIDs(projected), []string{"bus-00", "bus-01", "bus-03", "bus-06"})
	assert.Equal(t, countActions(projected[2].NodeActions), 2)
	assert.Assert(t, !containsStr(projected[2].Reviewers, `"role":"rd"`), "fe 侧 rd 评审位应被裁")
}

// TestUnit_FindDependencyCycle 依赖环检测：无环 / 自环 / 双节点环 / 菱形（无环）。
func TestUnit_FindDependencyCycle(t *testing.T) {
	assert.Assert(t, domain.FindDependencyCycle("r", domain.DependencyEdges{
		"r": {"a", "b"}, "a": {"c"}, "b": {"c"},
	}) == nil, "菱形依赖非环")

	cycle := domain.FindDependencyCycle("r", domain.DependencyEdges{
		"r": {"a"}, "a": {"b"}, "b": {"r"},
	})
	assert.DeepEqual(t, cycle, []string{"r", "a", "b", "r"})

	cycle = domain.FindDependencyCycle("r", domain.DependencyEdges{"r": {"r"}})
	assert.DeepEqual(t, cycle, []string{"r", "r"})

	assert.Assert(t, domain.FindDependencyCycle("r", domain.DependencyEdges{"r": {"x"}}) == nil,
		"悬挂依赖（无 flow 行）不构成环")
}

// TestUnit_ActiveActionsFor 三向投影：角色 × 相位 × 节点激活清单。
func TestUnit_ActiveActionsFor(t *testing.T) {
	capabilities := map[string]domain.CapabilityInfo{
		"product-design": {
			RoleID: "pm",
			Actions: []repo.ActionDef{
				{ActionID: "produce", Name: "产出产品设计", On: repo.PhaseExec, Skill: "to-prd"},
				{ActionID: "revise", Name: "修订产品设计", On: repo.PhaseExec},
				{ActionID: "clarify", Name: "澄清", On: repo.PhaseRespond},
			},
		},
		"backend-design": {RoleID: "rd", Actions: []repo.ActionDef{{ActionID: "produce", Name: "x", On: repo.PhaseExec}}},
	}
	nodeActions := mustJSON([]repo.NodeActionDef{
		{Kind: repo.NodeActionKindCapability, CapabilityID: "product-design", Actions: []string{"produce", "revise", "clarify"}},
		{Kind: repo.NodeActionKindNode, ActionID: "align.opening", On: repo.PhaseEntry, Executor: repo.ExecutorProgram, ExecutorRole: "pm"},
	})

	// pm × exec × bus-02 形态：produce+revise（clarify on=respond 不命中）
	active, err := domain.ActiveActionsFor("pm", repo.PhaseExec, nodeActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(active), 2)
	assert.Equal(t, active[0].ActionID, "product-design.produce", "BUG-006 口径：capability 动作全名回显")
	assert.Equal(t, active[0].Skill, "to-prd")

	// 相位不命中（entry 无 capability 动作）→ node action 命中
	active, err = domain.ActiveActionsFor("pm", repo.PhaseEntry, nodeActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(active), 1)
	assert.Equal(t, active[0].Kind, repo.NodeActionKindNode)
	assert.Equal(t, active[0].ExecutorRole, "pm")

	// 角色不命中 → 空集（不报错）
	active, err = domain.ActiveActionsFor("rd", repo.PhaseExec, nodeActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(active), 0)

	// 未进入编排 → 恒空集
	active, err = domain.ActiveActionsFor("pm", repo.PhaseEmpty, nodeActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(active), 0)

	// 激活清单未含的动作不投影（backend-design 未激活）
	rdActions := mustJSON([]repo.NodeActionDef{
		{Kind: repo.NodeActionKindCapability, CapabilityID: "backend-design", Actions: []string{"produce"}},
	})
	active, err = domain.ActiveActionsFor("rd", repo.PhaseExec, rdActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(active), 1)
}

// TestUnit_SummarizeReviewResults 表态聚合计数与明细。
func TestUnit_SummarizeReviewResults(t *testing.T) {
	summary, err := domain.SummarizeReviewResults(`[
		{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":0,"at":1},
		{"role":"rd","capabilityId":"rd.prd-review","state":"unagree","round":0,"at":2},
		{"role":"fe","capabilityId":"fe.prd-review","state":"reviewing","round":0,"at":3}
	]`)
	assert.NilError(t, err)
	assert.Equal(t, summary.Agree, int64(1))
	assert.Equal(t, summary.Unagree, int64(1))
	assert.Equal(t, summary.Reviewing, int64(1))
	assert.Equal(t, len(summary.Details), 3)

	summary, err = domain.SummarizeReviewResults("")
	assert.NilError(t, err, "空文本=未填充，零值摘要")
	assert.Equal(t, summary.Agree+summary.Unagree+summary.Reviewing, int64(0))
}

// TestUnit_RelatedNodesFor 相关节点三路判定：评审位 / node action 响应者 / 能力归属。
func TestUnit_RelatedNodesFor(t *testing.T) {
	capabilityRoles := map[string]string{"product-design": "pm", "backend-design": "rd"}
	nodes := []domain.NodeRef{
		{NodeID: "n-01", NodeOrder: 1, Reviewers: mustJSON([]repo.ReviewerDef{{Role: "arch", CapabilityID: "arch.x"}}), NodeActions: "[]"},
		{NodeID: "n-02", NodeOrder: 2, Reviewers: "[]", NodeActions: mustJSON([]repo.NodeActionDef{
			{Kind: repo.NodeActionKindNode, ActionID: "align.opening", On: "entry", ExecutorRole: "pm"},
		})},
		{NodeID: "n-03", NodeOrder: 3, Reviewers: "[]", NodeActions: mustJSON([]repo.NodeActionDef{
			{Kind: repo.NodeActionKindCapability, CapabilityID: "product-design", Actions: []string{"produce"}},
		})},
		{NodeID: "n-04", NodeOrder: 4, Reviewers: "[]", NodeActions: "[]"},
	}
	related, err := domain.RelatedNodesFor("pm", capabilityRoles, nodes)
	assert.NilError(t, err)
	assert.DeepEqual(t, nodeIDs(related), []string{"n-02", "n-03"})

	related, err = domain.RelatedNodesFor("arch", capabilityRoles, nodes)
	assert.NilError(t, err)
	assert.DeepEqual(t, nodeIDs(related), []string{"n-01"})
}

// ---- 测试 helper ----

func mustJSON(v any) string {
	out, err := utils.MarshalJSONText(v)
	if err != nil {
		panic(err)
	}
	return out
}

func projectedIDs(nodes []domain.ProjectedNode) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.DefNodeID)
	}
	return out
}

func nodeIDs(nodes []domain.NodeRef) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.NodeID)
	}
	return out
}

func countActions(actionsJSON string) int {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(actionsJSON, &entries); err != nil {
		panic(err)
	}
	return len(entries)
}

func containsStr(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
