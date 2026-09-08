package domain

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/repo"
)

// TestUnit_FilterExecActionsByReentry exec 首产/返工配置选择表驱动（capability-model
// §5.1：0=首产 produce/dev/accept，≥1=返工 revise/diagnosing-bugs/retest；
// 集合外动作如 submit 恒保留）。
func TestUnit_FilterExecActionsByReentry(t *testing.T) {
	actions := []ActiveAction{
		{ActionID: "produce", Kind: repo.NodeActionKindCapability, On: repo.PhaseExec},
		{ActionID: "submit", Kind: repo.NodeActionKindCapability, On: repo.PhaseExec},
		{ActionID: "revise", Kind: repo.NodeActionKindCapability, On: repo.PhaseExec},
	}
	// 首次：保留 produce/submit，剔除 revise
	kept := FilterExecActionsByReentry(actions, 0)
	assert.Equal(t, len(kept), 2)
	assert.Equal(t, kept[0].ActionID, "produce")
	assert.Equal(t, kept[1].ActionID, "submit")

	// 重入：保留 submit/revise（保持入参顺序），剔除 produce
	kept = FilterExecActionsByReentry(actions, 1)
	assert.Equal(t, len(kept), 2)
	assert.Equal(t, kept[0].ActionID, "submit")
	assert.Equal(t, kept[1].ActionID, "revise")
	kept = FilterExecActionsByReentry(actions, 3)
	assert.Equal(t, len(kept), 2)
	assert.Equal(t, kept[1].ActionID, "revise")

	// 非 exec 集动作集原样语义：空集进空集出
	assert.Equal(t, len(FilterExecActionsByReentry(nil, 0)), 0)
	assert.Equal(t, len(FilterExecActionsByReentry(nil, 2)), 0)

	// dev/diagnosing-bugs 与 accept/retest 同表驱动（开发与 QA 节点同口径）
	devSet := []ActiveAction{
		{ActionID: "dev", On: repo.PhaseExec},
		{ActionID: "diagnosing-bugs", On: repo.PhaseExec},
	}
	kept = FilterExecActionsByReentry(devSet, 0)
	assert.Equal(t, len(kept), 1)
	assert.Equal(t, kept[0].ActionID, "dev")
	kept = FilterExecActionsByReentry(devSet, 1)
	assert.Equal(t, len(kept), 1)
	assert.Equal(t, kept[0].ActionID, "diagnosing-bugs")

	qaSet := []ActiveAction{
		{ActionID: "accept", On: repo.PhaseExec},
		{ActionID: "retest", On: repo.PhaseExec},
	}
	kept = FilterExecActionsByReentry(qaSet, 0)
	assert.Equal(t, len(kept), 1)
	assert.Equal(t, kept[0].ActionID, "accept")
	kept = FilterExecActionsByReentry(qaSet, 1)
	assert.Equal(t, len(kept), 1)
	assert.Equal(t, kept[0].ActionID, "retest")
}

// TestUnit_RollbackTarget 回退目标选取：渲染序上一节点；无上游=nil；乱序入参
// 按 node_order 判定（内部按有序断言使用）。
func TestUnit_RollbackTarget(t *testing.T) {
	nodes := []*repo.FlowNode{
		{NodeID: "n-01", NodeOrder: 1},
		{NodeID: "n-02", NodeOrder: 2},
		{NodeID: "n-03", NodeOrder: 3},
	}
	target := RollbackTarget(nodes, 3)
	assert.Assert(t, target != nil)
	assert.Equal(t, target.NodeID, "n-02")

	target = RollbackTarget(nodes, 1)
	assert.Assert(t, target == nil, "首节点无上游")

	target = RollbackTarget(nil, 5)
	assert.Assert(t, target == nil)
}

// TestUnit_RollbackGuard 回退幂等护栏：同轮重放 no-op；clarify 升轮不打回不受影响。
func TestUnit_RollbackGuard(t *testing.T) {
	assert.Assert(t, RollbackNeeded(0, 0), "首次打回：0<=0 应用")
	assert.Assert(t, RollbackNeeded(0, 2), "clarify 升轮后打回：0<=2 应用")
	assert.Assert(t, !RollbackNeeded(1, 0), "同轮重放：1>0 no-op")
	assert.Assert(t, !RollbackNeeded(2, 1), "后续轮已处理：2>1 no-op")
	assert.Equal(t, NextRollbackCount(0), int64(1))
	assert.Equal(t, NextRollbackCount(2), int64(3))
}
