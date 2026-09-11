// Package domain 本文件：任务总线纯逻辑（T5）——动作定位解析（capability×action →
// role/task_type，含激活校验）、泳道派生、任务目标缺省装配与 complete 判据校验框架
// （表驱动，spec §2「agent 谎报完成」行）。无 IO，可单测。
//
// 定稿出处：
//   - spec §0 表 #7 tasks.task_type 六族 / §1.2 wb_assign_task·wb_task_update 两行 /
//     §2 异常处理表前两行（谎报完成判据拦截、派发幂等键）；
//   - capability-actions-final §2：动作=能力内可独立调度单元，任务族按动作语义归族；
//   - 评审位声明（flow_def_nodes.reviewers）为评审能力的合法激活源
//     （payload.go ReviewerDef「审批 skills map 派生源」口径）——review 族任务
//     依此激活，与 node_actions 激活清单并列。
package domain

import (
	"fmt"
	"strings"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// actionTaskTypes 动作 id → 任务族映射（表驱动单一口径）。
// 依据 seed 21 能力动作词表归族：产出/修订类=produce、开发/缺陷类=dev、
// 表态=review、响应类（澄清/修复/复核）=respond、验收复测=accept、规划建需=create_reqs。
var actionTaskTypes = map[string]string{
	// produce：主产出与修订（含 node action 文档产出类）
	"produce":      repo.TaskTypeProduce,
	"revise":       repo.TaskTypeProduce,
	"to-brd":       repo.TaskTypeProduce,
	"to-alignment": repo.TaskTypeProduce,
	// dev：开发实现与缺陷修复
	"dev":             repo.TaskTypeDev,
	"diagnosing-bugs": repo.TaskTypeDev,
	// review：评审表态
	"review": repo.TaskTypeReview,
	// respond：owner 响应（澄清/评审修复/复核）
	"clarify":       repo.TaskTypeRespond,
	"fix_on_review": repo.TaskTypeRespond,
	"confirm":       repo.TaskTypeRespond,
	// accept：验收与复验重测
	"accept": repo.TaskTypeAccept,
	"retest": repo.TaskTypeAccept,
	// create_reqs：规划建需·启动版本
	"create_reqs": repo.TaskTypeCreateReqs,
}

// TaskTypeForAction 动作 id → 任务族（表驱动）；未知动作返回 ("", false)。
func TaskTypeForAction(actionID string) (string, bool) {
	taskType, ok := actionTaskTypes[actionID]
	return taskType, ok
}

// IsExecAdvanceFamily exec 阶段标志任务族（单一来源，2026-09-10 CR 复审收敛）：
// produce/dev/accept/create_reqs 完成 = exec 出口放行的唯一判据集。消费方：
// wb_advance execAdvanceReady 与 complete 内聚推进触发（两处必须同集，
// 防扩族时单点漏改行为分叉）。
func IsExecAdvanceFamily(taskType string) bool {
	switch taskType {
	case repo.TaskTypeProduce, repo.TaskTypeDev, repo.TaskTypeAccept, repo.TaskTypeCreateReqs:
		return true
	}
	return false
}

// BranchForScope 泳道派生：rd/fe 单泳道按 scope 直取；full（双泳道）与 plan（无
// scope）任务不锚定单侧泳道，branch 空（DDL 列注释「rd/fe/空」第三值）。
func BranchForScope(scope string) string {
	switch scope {
	case repo.ScopeRD:
		return repo.ScopeRD
	case repo.ScopeFE:
		return repo.ScopeFE
	default:
		return ""
	}
}

// ResolvedAction 任务派发的动作定位结果（wb_assign_task 装配输入）。
type ResolvedAction struct {
	// ActionID 动作定位（入参回显）
	ActionID string
	// CapabilityID 所属能力；kind=node 动作为空
	CapabilityID string
	// RoleID 执行角色（capability 归属角色 / node action executor_role）
	RoleID string
	// TaskType 任务族（表驱动归族）
	TaskType string
	// Name 动作显示名（node action 无独立名，取 action_id）
	Name string
	// On 挂载相位；未知为空
	On string
	// Skill 挂载 skill；空=无
	Skill string
	// Inline 一句话内联定义；空=无
	Inline string
	// OutputProductTypes 产出产物类型（complete 判据 produce/dev/accept 族期望集）
	OutputProductTypes []string
	// Source 激活来源：node_actions 激活清单 | reviewers 评审位声明
	Source string
}

// 动作激活来源（ResolvedAction.Source 取值）。
const (
	ActionSourceNodeActions = "node_actions"
	ActionSourceReviewers   = "reviewers"
)

// ResolveTaskAction 动作定位解析：action_id 为 "<capabilityId>.<actionId>" 时
// 走能力动作解析——capabilityId 须在该节点 node_actions 激活清单中且动作在激活
// 子集内（spec §1.2：未激活 400），或该能力为节点评审位声明（review 能力合法
// 激活源）；assignee 须与 capabilities.role_id 一致。无点号视为 node action——
// 在节点 node_actions kind=node 条目中定位，executor_role 非空时须与 assignee 一致。
//
// 错误集：ErrInvalidArgument（未知动作/未激活/assignee 与角色归属不符）；
// node_actions JSON 非法（实例数据损坏，原样上抛）。
func ResolveTaskAction(actionID string, assignee string, nodeActionsJSON string, reviewersJSON string, capabilities map[string]CapabilityInfo) (*ResolvedAction, error) {
	if capID, actID, isCapability := splitCapabilityAction(actionID); isCapability {
		return resolveCapabilityAction(capID, actID, assignee, nodeActionsJSON, reviewersJSON, capabilities)
	}
	return resolveNodeAction(actionID, assignee, nodeActionsJSON)
}

// SplitCapabilityAction 拆分 "<capabilityId>.<actionId>"：id 约定——action_id 不含点
// （capability_id 可含点，如 rd.prd-review），故按**最后一个**点切分；无点号（或首尾点）
// 为 node action，ok=false。脆弱点声明：若 action_id 自身含点（如 node 动作
// align.opening）会被误切为 capability/action 对——防线在其后的解析链：误切出的
// "capability" 不在能力注册表即拒（未知能力），即便命中，动作 id 也无任务族映射
// （TaskTypeForAction 未知动作，「不可派发」拦截）。导出供 service 层 complete 判据
// 期望产物解析复用（单一口径）。
func SplitCapabilityAction(actionID string) (string, string, bool) {
	idx := strings.LastIndex(actionID, ".")
	if idx <= 0 || idx == len(actionID)-1 {
		return "", "", false
	}
	return actionID[:idx], actionID[idx+1:], true
}

// splitCapabilityAction 拆分 "<capabilityId>.<actionId>"（内部转发导出口径）。
func splitCapabilityAction(actionID string) (string, string, bool) {
	return SplitCapabilityAction(actionID)
}

// resolveCapabilityAction 能力动作定位：激活清单命中或评审位声明命中，角色归属校验。
func resolveCapabilityAction(capabilityID string, actionID string, assignee string, nodeActionsJSON string, reviewersJSON string, capabilities map[string]CapabilityInfo) (*ResolvedAction, error) {
	info, ok := capabilities[capabilityID]
	if !ok {
		return nil, invalidAction("未知能力 %q", capabilityID)
	}
	if info.RoleID != assignee {
		return nil, invalidAction("assignee %q 与能力 %q 归属角色 %q 不符", assignee, capabilityID, info.RoleID)
	}
	var action *repo.ActionDef
	for i := range info.Actions {
		if info.Actions[i].ActionID == actionID {
			action = &info.Actions[i]
			break
		}
	}
	if action == nil {
		return nil, invalidAction("能力 %q 无动作 %q", capabilityID, actionID)
	}

	activated, err := capabilityActivatedInNodeActions(capabilityID, actionID, nodeActionsJSON)
	if err != nil {
		return nil, err
	}
	source := ""
	if activated {
		source = ActionSourceNodeActions
	} else {
		reviewerHit, err := capabilityActivatedInReviewers(capabilityID, reviewersJSON)
		if err != nil {
			return nil, err
		}
		if reviewerHit {
			source = ActionSourceReviewers
		}
	}
	if source == "" {
		return nil, invalidAction("动作 %s.%s 未在该节点激活（激活清单/评审位均未声明）", capabilityID, actionID)
	}

	taskType, known := TaskTypeForAction(actionID)
	if !known {
		return nil, invalidAction("动作 %q 不在任务族映射（不可派发）", actionID)
	}
	return &ResolvedAction{
		ActionID:           capabilityID + "." + actionID,
		CapabilityID:       capabilityID,
		RoleID:             info.RoleID,
		TaskType:           taskType,
		Name:               action.Name,
		On:                 action.On,
		Skill:              action.Skill,
		Inline:             action.Inline,
		OutputProductTypes: action.OutputProductTypes,
		Source:             source,
	}, nil
}

// capabilityActivatedInNodeActions 激活清单判定：kind=capability 条目命中能力且
// 动作在激活子集（条目级 scope 已由实例化/my_task 同函数裁剪，此处不再裁）。
func capabilityActivatedInNodeActions(capabilityID string, actionID string, nodeActionsJSON string) (bool, error) {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(nodeActionsJSON, &entries); err != nil {
		return false, perr.Trace(err)
	}
	for _, entry := range entries {
		if entry.Kind != repo.NodeActionKindCapability || entry.CapabilityID != capabilityID {
			continue
		}
		for _, activated := range entry.Actions {
			if activated == actionID {
				return true, nil
			}
		}
	}
	return false, nil
}

// capabilityActivatedInReviewers 评审位声明判定：节点评审位声明该能力即激活
// （评审能力动作不在 node_actions 激活清单——per-agent 表态经评审位派生）。
func capabilityActivatedInReviewers(capabilityID string, reviewersJSON string) (bool, error) {
	var reviewers []repo.ReviewerDef
	if err := utils.UnmarshalJSONText(reviewersJSON, &reviewers); err != nil {
		return false, perr.Trace(err)
	}
	for _, reviewer := range reviewers {
		if reviewer.CapabilityID == capabilityID {
			return true, nil
		}
	}
	return false, nil
}

// resolveNodeAction node action 定位（kind=node 条目）。
func resolveNodeAction(actionID string, assignee string, nodeActionsJSON string) (*ResolvedAction, error) {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(nodeActionsJSON, &entries); err != nil {
		return nil, perr.Trace(err)
	}
	for _, entry := range entries {
		if entry.Kind != repo.NodeActionKindNode || entry.ActionID != actionID {
			continue
		}
		taskType, known := TaskTypeForAction(actionID)
		if !known {
			return nil, invalidAction("node action %q 不在任务族映射（不可派发）", actionID)
		}
		roleID := assignee
		if entry.ExecutorRole != "" {
			if entry.ExecutorRole != assignee {
				return nil, invalidAction("assignee %q 与 node action %q 响应者角色 %q 不符", assignee, actionID, entry.ExecutorRole)
			}
			roleID = entry.ExecutorRole
		}
		return &ResolvedAction{
			ActionID:     actionID,
			CapabilityID: "",
			RoleID:       roleID,
			TaskType:     taskType,
			Name:         actionID,
			On:           entry.On,
			Skill:        entry.Skill,
			Inline:       entry.Inline,
			Source:       ActionSourceNodeActions,
		}, nil
	}
	return nil, invalidAction("node action %q 未在该节点声明", actionID)
}

// invalidAction 动作定位非法错误（哨兵 errs.ErrInvalidArgument 为 cause）。
func invalidAction(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrInvalidArgument, fmt.Sprintf(format, args...)))
}

// DefaultTaskGoal 任务目标缺省装配（AssignTaskReq.goal 空=确定性缺省）。
func DefaultTaskGoal(nodeKey string, resolved *ResolvedAction) string {
	return fmt.Sprintf("在节点「%s」执行动作 %s（%s，任务族 %s）", nodeKey, resolved.ActionID, resolved.Name, resolved.TaskType)
}

// TaskFacts complete 判据校验的事实输入（service 层经 repo 查询装配，无 IO 纯判定）。
type TaskFacts struct {
	// Role 任务执行角色（review/respond 族判据主体）
	Role string
	// RegisteredProductTypes 该需求该节点已登记产物类型集（artifacts 直查）
	RegisteredProductTypes []string
	// ExpectedProductTypes 期望产物类型集（动作 outputProductTypes；空=节点
	// expected_outputs 兜底）
	ExpectedProductTypes []string
	// ReviewStateByRole 该节点 per-agent 最新表态（role → state）
	ReviewStateByRole map[string]string
	// OpenIssueCount 该节点指派给执行角色的未闭环 issue 数（state=open）
	OpenIssueCount int64
}

// CompleteCriteria complete 判据校验框架（表驱动，spec §2「agent 谎报完成」行）：
// 按任务族返回缺项清单（空=满足，可 completed）。
//
//   - produce / dev / accept：期望产物类型逐一查已登记（缺 → "artifact:<productType>"）；
//   - review：该角色在该节点已有表态（agree/unagree；缺 → "review_statement:<role>"）；
//   - respond：该角色该节点无未闭环 issue（有 → "open_issues:<role>"）——最小判据 v1：
//     澄清/修复/复核的闭环以 open 清零为门槛，精细锚定（pending-confirm 复核链）
//     留 T7/T8 接入后收紧；
//   - create_reqs：最小判据 v1=无硬判据（建需产物为 flows.dependencies 边与子需求，
//     由 T8/plan 流程接入后收紧），恒放行；
//   - 未知任务族：fail-closed（缺项 "unknown_task_type:<族>"），不静默放行。
func CompleteCriteria(taskType string, facts TaskFacts) []string {
	switch taskType {
	case repo.TaskTypeProduce, repo.TaskTypeDev, repo.TaskTypeAccept:
		registered := make(map[string]bool, len(facts.RegisteredProductTypes))
		for _, productType := range facts.RegisteredProductTypes {
			registered[productType] = true
		}
		missing := make([]string, 0)
		for _, expected := range facts.ExpectedProductTypes {
			if !registered[expected] {
				missing = append(missing, "artifact:"+expected)
			}
		}
		return missing
	case repo.TaskTypeReview:
		state, stated := facts.ReviewStateByRole[facts.Role]
		if !stated || (state != repo.ReviewStateAgree && state != repo.ReviewStateUnagree) {
			return []string{"review_statement:" + facts.Role}
		}
		return []string{}
	case repo.TaskTypeRespond:
		if facts.OpenIssueCount > 0 {
			return []string{fmt.Sprintf("open_issues:%s(%d)", facts.Role, facts.OpenIssueCount)}
		}
		return []string{}
	case repo.TaskTypeCreateReqs:
		return []string{}
	default:
		return []string{"unknown_task_type:" + taskType}
	}
}
