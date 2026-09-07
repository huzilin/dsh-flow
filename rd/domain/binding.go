// Package domain 状态机与纯逻辑（spec §1.3）：类别→流程类型绑定、
// 需求/版本状态与枚举校验——全部无 IO，可表驱动单测。
//
// 类别→流程类型绑定规则出处（证据最充分的显式定义）：
//   - docs-tmp/ui-interaction-final.md §2「类别↔流程类型绑定」定稿：
//     需求任务→business/tech/task（scope 三值）、规划审批任务/版本规划任务→plan（无 scope）；
//   - docs/v1.0/arch_design.md §8.2：按需求 category/reqType 选择模板（business/tech/task/plan）。
//   - 需求表（spec §0.3）无 reqType 列，需求任务在 {business, tech, task} 内的子选择
//     由创建入参 flow_defs_id 显式指定，缺省取 business（保守默认）。
package domain

import perr "github.com/pingcap/errors"

// 需求类别（requirements.category，决定流程类型绑定）。
const (
	CategoryRequirement = "需求任务"   // 需求任务
	CategoryApproval    = "规划审批任务" // 规划审批任务
	CategoryPlanning    = "版本规划任务" // 版本规划任务
)

// 流程类型（flow_defs.flow_defs_id，seed 四值）。
const (
	FlowDefsBusiness = "business"
	FlowDefsTech     = "tech"
	FlowDefsTask     = "task"
	FlowDefsPlan     = "plan"
)

// 需求状态（requirements.status）。
const (
	ReqStatusPending  = "待启动"
	ReqStatusRunning  = "进行中"
	ReqStatusDone     = "已完成"
	ReqStatusCanceled = "已取消"
)

// 版本状态（versions.state，单向：待启动→进行中→已发布；已废弃=任意阶段可达终态）。
const (
	VerStatePending   = "待启动"
	VerStateRunning   = "进行中"
	VerStatePublished = "已发布"
	VerStateDeleted   = "已废弃"
)

// 需求优先级（requirements.priority）。
const (
	PriorityP0 = "P0"
	PriorityP1 = "P1"
	PriorityP2 = "P2"
	PriorityP3 = "P3"
)

// 默认优先级（创建入参空值时）。
const DefaultPriority = PriorityP2

// CategoryDefinitions 类别全集（表驱动：类别 → 允许流程集 + 默认流程）。
// 顺序即展示序；校验与派生一律查本表，不写 if-else 分支。
var CategoryDefinitions = []CategoryDef{
	{
		Name:         CategoryRequirement,
		AllowedFlows: []string{FlowDefsBusiness, FlowDefsTech, FlowDefsTask},
		DefaultFlow:  FlowDefsBusiness,
		DefaultScope: "", // scope 由各流程 scope_options 承载（T4 实例化消费）
	},
	{
		Name:         CategoryApproval,
		AllowedFlows: []string{FlowDefsPlan},
		DefaultFlow:  FlowDefsPlan,
		DefaultScope: "",
	},
	{
		Name:         CategoryPlanning,
		AllowedFlows: []string{FlowDefsPlan},
		DefaultFlow:  FlowDefsPlan,
		DefaultScope: "",
	},
}

// CategoryDef 类别绑定定义表条目。
type CategoryDef struct {
	// Name 类别标签（requirements.category 原值）
	Name string
	// AllowedFlows 允许绑定的流程类型集（flow_defs_id）
	AllowedFlows []string
	// DefaultFlow 缺省派生的流程类型
	DefaultFlow string
	// DefaultScope 默认 scope（预留；T4 实例化消费）
	DefaultScope string
}

// CategoryDefByName 按类别名查定义；未命中返回 false（未知类别）。
func CategoryDefByName(category string) (CategoryDef, bool) {
	for _, def := range CategoryDefinitions {
		if def.Name == category {
			return def, true
		}
	}
	return CategoryDef{}, false
}

// DeriveFlowDefsID 类别→流程类型派生绑定。
//
// explicit 为空 → 取类别默认流程；非空 → 必须在类别允许集内（越集返回错误，
// 错误信息带允许集便于前端提示）。未知类别返回错误。
//
// 错误集：入参非法（未知类别 / flow_defs_id 越出类别允许集）。
// 调用方以 errs.ErrInvalidArgument 为 cause 包裹（见 service 层）。
func DeriveFlowDefsID(category string, explicit string) (string, error) {
	def, ok := CategoryDefByName(category)
	if !ok {
		return "", perr.Errorf("未知类别 %q（允许：需求任务/规划审批任务/版本规划任务）", category)
	}
	if explicit == "" {
		return def.DefaultFlow, nil
	}
	for _, f := range def.AllowedFlows {
		if f == explicit {
			return explicit, nil
		}
	}
	return "", perr.Errorf("类别 %q 不允许流程类型 %q（允许集：%v）", category, explicit, def.AllowedFlows)
}

// ValidRequirementStatus 需求状态校验。
func ValidRequirementStatus(status string) bool {
	switch status {
	case ReqStatusPending, ReqStatusRunning, ReqStatusDone, ReqStatusCanceled:
		return true
	}
	return false
}

// ValidPriority 优先级校验（P0-P3）。
func ValidPriority(priority string) bool {
	switch priority {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3:
		return true
	}
	return false
}

// ValidVersionState 版本状态校验。
func ValidVersionState(state string) bool {
	switch state {
	case VerStatePending, VerStateRunning, VerStatePublished, VerStateDeleted:
		return true
	}
	return false
}

// RequirementUnfinished 需求是否「未完成」（版本发布软约束 / 废弃引用判定口径：
// status 非 已完成/已取消 即 unfinished）。
func RequirementUnfinished(status string) bool {
	return status != ReqStatusDone && status != ReqStatusCanceled
}
