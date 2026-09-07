// category→flow 绑定表驱动单测（domain 纯逻辑，无 IO）：
// 全类别默认派生 / 允许集内显式指定 / 越集拦截 / 未知类别拦截 / 枚举校验。
package domain

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// TestDeriveFlowDefsID_Defaults 全类别默认派生（出处：ui-interaction-final §2 定稿绑定）。
func TestDeriveFlowDefsID_Defaults(t *testing.T) {
	cases := []struct {
		category string
		want     string
	}{
		{CategoryRequirement, FlowDefsBusiness},
		{CategoryApproval, FlowDefsPlan},
		{CategoryPlanning, FlowDefsPlan},
	}
	for _, c := range cases {
		got, err := DeriveFlowDefsID(c.category, "")
		assert.NilError(t, err, "category=%s", c.category)
		assert.Equal(t, got, c.want, "category=%s", c.category)
	}
}

// TestDeriveFlowDefsID_ExplicitInAllowedSet 需求任务允许集 {business, tech, task} 内显式指定。
func TestDeriveFlowDefsID_ExplicitInAllowedSet(t *testing.T) {
	for _, flow := range []string{FlowDefsBusiness, FlowDefsTech, FlowDefsTask} {
		got, err := DeriveFlowDefsID(CategoryRequirement, flow)
		assert.NilError(t, err, "flow=%s", flow)
		assert.Equal(t, got, flow)
	}
	// 审批/规划类别：唯一允许 plan，显式 plan 合法
	got, err := DeriveFlowDefsID(CategoryApproval, FlowDefsPlan)
	assert.NilError(t, err)
	assert.Equal(t, got, FlowDefsPlan)
}

// TestDeriveFlowDefsID_Rejections 越集与未知类别拦截。
func TestDeriveFlowDefsID_Rejections(t *testing.T) {
	// plan 不在需求任务允许集
	_, err := DeriveFlowDefsID(CategoryRequirement, FlowDefsPlan)
	assert.Assert(t, err != nil, "需求任务绑 plan 应被拦")

	// business/tech/task 不在审批类别允许集
	for _, flow := range []string{FlowDefsBusiness, FlowDefsTech, FlowDefsTask} {
		_, err := DeriveFlowDefsID(CategoryApproval, flow)
		assert.Assert(t, err != nil, "规划审批任务绑 %s 应被拦", flow)
		_, err = DeriveFlowDefsID(CategoryPlanning, flow)
		assert.Assert(t, err != nil, "版本规划任务绑 %s 应被拦", flow)
	}

	// 未知类别
	_, err = DeriveFlowDefsID("未知类别", "")
	assert.Assert(t, err != nil, "未知类别应被拦")
	assert.Assert(t, strings.Contains(err.Error(), "未知类别"))
}

// TestCategoryDefinitions_TableIntegrity 绑定定义表完整性：类别名唯一、
// 默认流程在允许集内、非空允许集。
func TestCategoryDefinitions_TableIntegrity(t *testing.T) {
	seen := make(map[string]bool)
	for _, def := range CategoryDefinitions {
		assert.Assert(t, !seen[def.Name], "类别 %s 重复定义", def.Name)
		seen[def.Name] = true
		assert.Assert(t, len(def.AllowedFlows) > 0, "类别 %s 允许集非空", def.Name)
		defaultInSet := false
		for _, f := range def.AllowedFlows {
			if f == def.DefaultFlow {
				defaultInSet = true
			}
		}
		assert.Assert(t, defaultInSet, "类别 %s 默认流程须在允许集内", def.Name)
	}
	assert.Equal(t, len(seen), 3, "三类别全集")
}

// TestEnumValidators 枚举校验与 unfinished 口径。
func TestEnumValidators(t *testing.T) {
	assert.Assert(t, ValidRequirementStatus(ReqStatusPending))
	assert.Assert(t, ValidRequirementStatus(ReqStatusRunning))
	assert.Assert(t, ValidRequirementStatus(ReqStatusDone))
	assert.Assert(t, ValidRequirementStatus(ReqStatusCanceled))
	assert.Assert(t, !ValidRequirementStatus("unknown"))

	for _, p := range []string{PriorityP0, PriorityP1, PriorityP2, PriorityP3} {
		assert.Assert(t, ValidPriority(p))
	}
	assert.Assert(t, !ValidPriority("P4"))

	assert.Assert(t, ValidVersionState(VerStatePending))
	assert.Assert(t, ValidVersionState(VerStateRunning))
	assert.Assert(t, ValidVersionState(VerStatePublished))
	assert.Assert(t, ValidVersionState(VerStateDeleted))
	assert.Assert(t, !ValidVersionState("unknown"))

	assert.Assert(t, RequirementUnfinished(ReqStatusPending), "待启动=unfinished")
	assert.Assert(t, RequirementUnfinished(ReqStatusRunning), "进行中=unfinished")
	assert.Assert(t, !RequirementUnfinished(ReqStatusDone), "已完成≠unfinished")
	assert.Assert(t, !RequirementUnfinished(ReqStatusCanceled), "已取消≠unfinished")
}
