// Package domain 本文件：ActiveActionsFor 动作标识口径单测（2026-09-10 BUG-006
// 拍板——kind=capability 条目回显 "<capabilityId>.<action>" 全名，与 wb_assign_task
// 入参同口径；kind=node 保持原 id）。纯逻辑，无 IO。
package domain

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
)

// TestUnit_ActiveActionsFor_CapabilityActionFullName BUG-006：my-task 激活动作
// 标识与 assign 入参同口径——capability 动作回显全名（可直接回填派发），
// node action 保持原 id（本就带点）。
func TestUnit_ActiveActionsFor_CapabilityActionFullName(t *testing.T) {
	nodeActions := `[{"kind":"capability","capabilityId":"product-design","actions":["produce"]},
	                 {"kind":"node","actionId":"align.opening","on":"entry","executorRole":"pm"}]`
	capabilities := map[string]CapabilityInfo{
		"product-design": {
			RoleID:  "pm",
			Actions: []repo.ActionDef{{ActionID: "produce", Name: "产出", On: "exec"}},
		},
	}

	// kind=capability：全名回显（旧口径为裸名 produce，与 assign 解析断链）
	active, err := ActiveActionsFor("pm", "exec", nodeActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(active), 1)
	assert.Equal(t, active[0].ActionID, "product-design.produce")
	assert.Equal(t, active[0].Kind, repo.NodeActionKindCapability)
	assert.Equal(t, active[0].CapabilityID, "product-design")

	// 回填验证：全名经 SplitCapabilityAction（assign 解析同函数）切回能力×动作
	capID, actID, ok := SplitCapabilityAction(active[0].ActionID)
	assert.Equal(t, ok, true)
	assert.Equal(t, capID, "product-design")
	assert.Equal(t, actID, "produce")

	// kind=node 条目保持原 id（align.opening 本就带点同口径）
	nodeActive, err := ActiveActionsFor("pm", "entry", nodeActions, capabilities)
	assert.NilError(t, err)
	assert.Equal(t, len(nodeActive), 1)
	assert.Equal(t, nodeActive[0].ActionID, "align.opening")
	assert.Equal(t, nodeActive[0].Kind, repo.NodeActionKindNode)
}
