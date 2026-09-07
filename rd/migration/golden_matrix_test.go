// 黄金数据 scope 实例化矩阵遍历（T4 AC）：40 定义节点定稿口径上，四流程 × 各自
// scope_options 全组合投影——节点集与序号符合渲染矩阵（ui-interaction-final §1.1 渲染示例
// + business-flow-derivation / flow-derivation-tech-task-plan 定稿），被裁节点不占位。
// 纯逻辑（domain 投影函数 × goldenNodes），不触库；DB 侧实例化另有集成测试。
package migration_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// flowScopeMatrix 四流程 scope_options（与 seed flow_defs 一致）。
var flowScopeMatrix = map[string][]string{
	"business": {"full", "rd", "fe"},
	"tech":     {"full", "rd", "fe"},
	"task":     {"rd", "fe"},
	"plan":     {},
}

// wantMatrix 渲染矩阵期望：flow × scope → 定义节点 id 序列（投影后按渲染序）。
var wantMatrix = map[string]map[string][]string{
	"business": {
		"full": {"bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-06", "bus-07", "bus-08", "bus-09", "bus-10", "bus-11", "bus-12"},
		"rd":   {"bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-08", "bus-11", "bus-12"},
		"fe":   {"bus-00", "bus-01", "bus-02", "bus-03", "bus-06", "bus-07", "bus-09", "bus-11", "bus-12"},
	},
	"tech": {
		"full": {"tec-b0", "tec-b1", "tec-b2", "tec-b3", "tec-b4", "tec-b5", "tec-b8a", "tec-b8b", "tec-b9", "tec-b10", "tec-b11"},
		"rd":   {"tec-b0", "tec-b1", "tec-b2", "tec-b3", "tec-b8a", "tec-b10", "tec-b11"},
		"fe":   {"tec-b0", "tec-b1", "tec-b4", "tec-b5", "tec-b8b", "tec-b10", "tec-b11"},
	},
	"task": {
		"rd": {"tsk-c0", "tsk-c1", "tsk-c2", "tsk-c3", "tsk-c4", "tsk-c6", "tsk-c7"},
		"fe": {"tsk-c0", "tsk-c1", "tsk-c2", "tsk-c3", "tsk-c4", "tsk-c6", "tsk-c7"},
	},
	"plan": {
		"": {"pln-d0", "pln-d1", "pln-d2", "pln-d3", "pln-d4", "pln-d5", "pln-d6", "pln-d7", "pln-d8"},
	},
}

// TestUnit_GoldenScopeInstantiationMatrix 40 定义节点 × 全 scope 组合遍历：
// 节点集恒等、序号 1 起连续（被裁不占位）、business rd=9 节点（FE 零节点）、
// task 无 QA 不裁行（rd|fe 全渲染）、plan 无 scope 全渲染。
func TestUnit_GoldenScopeInstantiationMatrix(t *testing.T) {
	golden := goldenNodes()
	assert.Equal(t, len(golden), 40, "定义节点黄金数据应 40 行")

	for flow, scopes := range flowScopeMatrix {
		for _, scope := range scopes {
			projected, err := domain.ProjectFlowNodes(flow, scope, golden)
			assert.NilError(t, err, "投影失败: %s/%s", flow, scope)
			want := wantMatrix[flow][scope]
			assert.Equal(t, len(projected), len(want), "节点数: %s/%s", flow, scope)
			for i, node := range projected {
				assert.Equal(t, node.DefNodeID, want[i], "节点序: %s/%s[%d]", flow, scope, i)
				assert.Equal(t, node.NodeOrder, int64(i+1), "渲染序号连续（被裁不占位）: %s/%s[%d]", flow, scope, i)
			}
		}
	}
}

// TestUnit_GoldenScopeInstantiationPrune 评审位/激活清单裁剪抽核（同函数判定口径）：
// business rd 实例 FE 零评审位；bus-03 rd 保留 arch+rd 位；task tsk-c3 单评审位按侧取一。
func TestUnit_GoldenScopeInstantiationPrune(t *testing.T) {
	golden := goldenNodes()

	businessRD, err := domain.ProjectFlowNodes("business", "rd", golden)
	assert.NilError(t, err)
	byID := projectedByID(businessRD)

	// bus-03（序 4）：arch + rd 位保留，fe|full 位被裁
	assert.Equal(t, byID["bus-03"].Reviewers, j([]repo.ReviewerDef{
		{Role: "arch", CapabilityID: "arch.prd-review"},
		{Role: "rd", CapabilityID: "rd.prd-review", ScopeExpr: "rd|full"},
	}))
	// bus-10（QA 验收，full 专属）不在 rd 实例
	_, ok := byID["bus-10"]
	assert.Assert(t, !ok, "QA 验收仅 full 渲染")
	// bus-05 评审位：fe|full 的 fe 位被裁（arch/qa/pm 恒在）
	assert.Assert(t, !containsReviewerRole(byID["bus-05"].Reviewers, "fe"), "bus-05 fe 位应被裁")
	assert.Assert(t, containsReviewerRole(byID["bus-05"].Reviewers, "qa"), "bus-05 qa 位保留")

	// task rd/fe：tsk-c3 激活清单单侧取一
	taskRD, err := domain.ProjectFlowNodes("task", "rd", golden)
	assert.NilError(t, err)
	taskByID := projectedByID(taskRD)
	assert.Equal(t, countEntryActions(t, taskByID["tsk-c3"].NodeActions), 1, "task rd 技术评审只激活 backend-design")
	assert.Assert(t, containsStr(taskByID["tsk-c3"].NodeActions, "backend-design"), "task rd 激活 backend-design")

	taskFE, err := domain.ProjectFlowNodes("task", "fe", golden)
	assert.NilError(t, err)
	feByID := projectedByID(taskFE)
	assert.Assert(t, containsStr(feByID["tsk-c3"].NodeActions, "frontend-design"), "task fe 激活 frontend-design")
}

// ---- helper ----

func projectedByID(nodes []domain.ProjectedNode) map[string]domain.ProjectedNode {
	out := make(map[string]domain.ProjectedNode, len(nodes))
	for _, node := range nodes {
		out[node.DefNodeID] = node
	}
	return out
}

func countEntryActions(t *testing.T, actionsJSON string) int {
	t.Helper()
	var entries []repo.NodeActionDef
	assert.NilError(t, utils.UnmarshalJSONText(actionsJSON, &entries))
	return len(entries)
}

func containsReviewerRole(reviewersJSON string, role string) bool {
	var reviewers []repo.ReviewerDef
	if err := utils.UnmarshalJSONText(reviewersJSON, &reviewers); err != nil {
		return false
	}
	for _, reviewer := range reviewers {
		if reviewer.Role == role {
			return true
		}
	}
	return false
}

func containsStr(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
