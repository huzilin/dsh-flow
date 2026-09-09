// Package domain 本文件：wb_my_task 三向校验（role × node × 相位）的动作投影、
// 评审表态摘要与角色相关节点集——纯逻辑，无 IO。
//
// 定稿出处：
//   - docs-tmp/data-tools-final.md §2.2 wb_my_task 升级契约：入参 {reqId, role, nodeId?}
//     （nodeId 缺省=当前节点）；返回 currentPhase / activeActions（按节点激活清单 + 当前
//     相位过滤）/ relatedNodes 最小返回集 / review_results 摘要；
//   - docs-tmp/ui-interaction-final.md §4「节点状态卡」消费口径：当前挂载动作 =
//     node_actions ∩ state；
//   - spec §1.2 wb_my_task：不属于该节点/该相位的角色返回空动作集而非报错；
//   - capability-model-final §1.1：exec 期 reviewState 恒无评审语义——activeActions 只看
//     node_actions 条目 on 字段与相位一致。
package domain

import (
	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// CapabilityInfo my_task 投影所需的能力最小信息（service 层从 capabilities 表装配）。
type CapabilityInfo struct {
	// RoleID 能力归属角色（三向校验 role 维度的判定依据）
	RoleID string
	// Actions 能力动作词表（ActionDef：actionId/on/name/skill/inline）
	Actions []repo.ActionDef
}

// ActiveAction 激活动作投影（activeActions 条目）。
type ActiveAction struct {
	// ActionID 动作标识（能力内动作 id 或 node action id）
	ActionID string
	// Name 动作显示名（node action 无独立名，取 action_id）
	Name string
	// Kind capability | node
	Kind string
	// CapabilityID 所属能力（kind=capability 有效）
	CapabilityID string
	// On 挂载相位（=过滤相位）
	On string
	// Executor 执行者（kind=node 有效）
	Executor string
	// ExecutorRole 响应者角色（kind=node 有效）
	ExecutorRole string
	// Skill 挂载 skill；空=无
	Skill string
	// Inline 一句话内联定义；空=无
	Inline string
}

// ActiveActionsFor 三向校验动作投影：实例节点激活清单（实例化已按条目级 scope 裁剪）
// × 相位（条目 on / 能力动作 on == phase）× 角色（kind=capability 按 capabilities.role_id、
// kind=node 按 executor_role）过滤。任一维不命中即空集（不报错——spec §1.2 口径）。
// phase 为空（未进入编排）恒空集。
//
// 错误集：node_actions JSON 非法（实例数据损坏）。
func ActiveActionsFor(role string, phase string, nodeActionsJSON string, capabilities map[string]CapabilityInfo) ([]ActiveAction, error) {
	if phase == "" {
		return []ActiveAction{}, nil
	}
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(nodeActionsJSON, &entries); err != nil {
		return nil, perr.Trace(err)
	}
	active := make([]ActiveAction, 0)
	for _, entry := range entries {
		switch entry.Kind {
		case repo.NodeActionKindCapability:
			info, ok := capabilities[entry.CapabilityID]
			if !ok || info.RoleID != role {
				continue
			}
			activated := make(map[string]bool, len(entry.Actions))
			for _, actionID := range entry.Actions {
				activated[actionID] = true
			}
			for _, action := range info.Actions {
				if activated[action.ActionID] && action.On == phase {
					active = append(active, ActiveAction{
						ActionID:     action.ActionID,
						Name:         action.Name,
						Kind:         repo.NodeActionKindCapability,
						CapabilityID: entry.CapabilityID,
						On:           action.On,
						Skill:        action.Skill,
						Inline:       action.Inline,
					})
				}
			}
		case repo.NodeActionKindNode:
			if entry.On == phase && entry.ExecutorRole == role {
				active = append(active, ActiveAction{
					ActionID:     entry.ActionID,
					Name:         entry.ActionID,
					Kind:         repo.NodeActionKindNode,
					CapabilityID: "",
					On:           entry.On,
					Executor:     entry.Executor,
					ExecutorRole: entry.ExecutorRole,
					Skill:        entry.Skill,
					Inline:       entry.Inline,
				})
			}
		}
	}
	return active, nil
}

// ReviewSummary 评审表态摘要（per-agent 聚合计数 + 明细）。
type ReviewSummary struct {
	// Agree 已同意计数
	Agree int64
	// Unagree 已反对计数
	Unagree int64
	// Reviewing 待表态计数
	Reviewing int64
	// Details 表态明细（flow_nodes.review_results 条目原样）
	Details []repo.ReviewResult
}

// SummarizeReviewResults 解析实例节点 review_results 并聚合计数。
//
// 错误集：review_results JSON 非法（实例数据损坏）。
func SummarizeReviewResults(reviewResultsJSON string) (ReviewSummary, error) {
	var results []repo.ReviewResult
	if err := utils.UnmarshalJSONText(reviewResultsJSON, &results); err != nil {
		return ReviewSummary{}, perr.Trace(err)
	}
	summary := ReviewSummary{Details: results}
	for _, result := range results {
		switch result.State {
		case repo.ReviewStateAgree:
			summary.Agree++
		case repo.ReviewStateUnagree:
			summary.Unagree++
		case repo.ReviewStateReviewing:
			summary.Reviewing++
		}
	}
	return summary, nil
}

// NodeRef 节点引用（relatedNodes 条目与投影输入的最小节点视图）。
type NodeRef struct {
	// NodeID 实例节点 id
	NodeID string
	// DefNodeID 定义节点引用
	DefNodeID string
	// NodeOrder 渲染序号
	NodeOrder int64
	// NodeKey 节点名
	NodeKey string
	// State 可配相位
	State string
	// Reviewers 实例评审位声明 JSON 文本（scope 裁剪后）
	Reviewers string
	// NodeActions 激活清单 JSON 文本（条目级 scope 裁剪后）
	NodeActions string
}

// RelatedNodesFor 角色相关节点集（relatedNodes 最小返回集，三路判定）：
// 实例评审位出现该角色，或 node action 响应者角色为该角色，或激活清单能力归属角色为该角色
// （capabilityRoles 为 capability_id → 归属角色装配表）。按传入顺序保留
// （调用方以 node_order 排序传入）。
//
// 错误集：reviewers/node_actions JSON 非法（实例数据损坏）。
func RelatedNodesFor(role string, capabilityRoles map[string]string, nodes []NodeRef) ([]NodeRef, error) {
	related := make([]NodeRef, 0)
	for _, node := range nodes {
		hit, err := nodeRelatesToRole(role, node)
		if err != nil {
			return nil, err
		}
		if !hit {
			hit = nodeRelatesViaCapability(role, capabilityRoles, node)
		}
		if hit {
			related = append(related, node)
		}
	}
	return related, nil
}

// nodeRelatesToRole 单节点判定：评审位出现该角色，或 node action 响应者角色为该角色。
func nodeRelatesToRole(role string, node NodeRef) (bool, error) {
	var reviewers []repo.ReviewerDef
	if err := utils.UnmarshalJSONText(node.Reviewers, &reviewers); err != nil {
		return false, perr.Trace(err)
	}
	for _, reviewer := range reviewers {
		if reviewer.Role == role {
			return true, nil
		}
	}
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(node.NodeActions, &entries); err != nil {
		return false, perr.Trace(err)
	}
	for _, entry := range entries {
		if entry.Kind == repo.NodeActionKindNode && entry.ExecutorRole == role {
			return true, nil
		}
	}
	return false, nil
}

// nodeRelatesViaCapability 能力归属维度判定（kind=capability 条目 → capabilities.role_id）。
func nodeRelatesViaCapability(role string, capabilityRoles map[string]string, node NodeRef) bool {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(node.NodeActions, &entries); err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.Kind == repo.NodeActionKindCapability && capabilityRoles[entry.CapabilityID] == role {
			return true
		}
	}
	return false
}
