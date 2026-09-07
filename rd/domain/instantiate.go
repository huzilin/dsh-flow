// Package domain 状态机与纯逻辑（spec §1.3）。本文件：流程实例化投影——
// scope 渲染集合表达式判定、定义节点 scope 过滤与序号重排、评审位/激活清单裁剪。
//
// 定稿出处：
//   - scope 表达式语义（data-tools-final §1.2）：”=any（任何 scope 渲染）｜'rd|full'=rd 侧
//     节点｜'fe|full'=fe 侧节点｜'full'=仅 full（QA 验收=联调职责）｜'rd|fe'=task 全渲染
//     （task 无 full）；需求 scope 三值 {full, rd, fe}；实例化按表达式 ∩ requirement.scope 判定；
//   - 节点裁剪与评审位裁剪由同一套 scope 判定函数给出（arch_design v0.29 §B 遗漏修复点）；
//   - 渲染重排（ui-interaction-final §1.1）：过滤后重新分组排布 + 重编号（被裁节点不占位）；
//   - 渲染示例：full=13 全渲染（business）；rd=9 节点单主链（0→1→2→3→4→5→8→11→12）；fe 对称。
package domain

import (
	"sort"
	"strings"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// ScopeMatches scope 渲染集合表达式判定（节点裁剪/评审位裁剪/激活清单裁剪共用同一函数）。
//
// scopeExpr=="" → 恒渲染（any）；否则按 "|" 拆分为渲染 scope 集合，
// 需求 scope 须命中其一。需求 scope 为空（plan）时仅空表达式命中。
func ScopeMatches(scopeExpr string, scope string) bool {
	if scopeExpr == "" {
		return true
	}
	if scope == "" {
		return false
	}
	for _, part := range strings.Split(scopeExpr, "|") {
		if part == scope {
			return true
		}
	}
	return false
}

// ParseScopeOptions 解析 flow_defs.scope_options（JSON 字符串数组，如 ["full","rd","fe"]；
// plan 为 []）。
//
// 错误集：JSON 非法（seed 数据损坏）。
func ParseScopeOptions(scopeOptionsJSON string) ([]string, error) {
	options := make([]string, 0)
	if err := utils.UnmarshalJSONText(scopeOptionsJSON, &options); err != nil {
		return nil, perr.Trace(err)
	}
	return options, nil
}

// ResolveScope 创建入参 scope 归一：空=取 scope_options 首值（business/tech 默认 full、
// task 默认 rd——表驱动确定性缺省）；显式值须在 scope_options 内（越集非法）；
// 无 scope 流程（plan，options 空）显式值必须为空串。
//
// 返回归一后的 scope（plan 为 ""）。
//
// 错误集：入参非法（scope 越出 scope_options / 无 scope 流程携带显式值）。
func ResolveScope(scopeOptions []string, explicit string) (string, error) {
	if len(scopeOptions) == 0 {
		if explicit != "" {
			return "", perr.Errorf("该流程无 scope（scope_options=[]），不接受显式值 %q", explicit)
		}
		return "", nil
	}
	if explicit == "" {
		return scopeOptions[0], nil
	}
	for _, option := range scopeOptions {
		if option == explicit {
			return explicit, nil
		}
	}
	return "", perr.Errorf("scope %q 越出可选集 %v", explicit, scopeOptions)
}

// ProjectedNode 实例化投影结果（flow_nodes 行前态；node_id/审计由 service 层装配）。
type ProjectedNode struct {
	// DefNodeID 定义节点引用（flow_def_nodes.template_node_id）
	DefNodeID string
	// NodeOrder 渲染序号（scope 过滤后重排，1 起连续、被裁不占位）
	NodeOrder int64
	// NodeKey 节点名（从定义复制）
	NodeKey string
	// NodeType 节点类型：produce/review/align/gate
	NodeType string
	// Gate 是否门禁
	Gate int64
	// UserGate User 终审拍板位
	UserGate int64
	// Reviewers 实例评审位声明 JSON 文本（按 scope 裁剪后）
	Reviewers string
	// NodeActions 动作激活清单 JSON 文本（条目级 scope 裁剪后）
	NodeActions string
}

// ProjectFlowNodes 流程实例化投影：按 flow_defs_id 取定义节点，scope_expr ∩ scope 过滤，
// 按定义 node_order 排序后重排（1 起连续）；评审位按 ReviewerDef.ScopeExpr、
// 激活清单按 NodeActionDef.ScopeExpr 用同一判定函数裁剪（arch_design v0.29 同源口径）。
//
// 入参：defs 为该流程全部定义节点（顺序不敏感，内部按 NodeOrder 稳定排序）。
//
// 错误集：reviewers/node_actions JSON 非法（seed 数据损坏）。
func ProjectFlowNodes(flowDefsID string, scope string, defs []repo.FlowDefNode) ([]ProjectedNode, error) {
	matched := make([]repo.FlowDefNode, 0, len(defs))
	for _, def := range defs {
		if def.FlowDefsID != flowDefsID {
			continue
		}
		if !ScopeMatches(def.ScopeExpr, scope) {
			continue
		}
		matched = append(matched, def)
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].NodeOrder < matched[j].NodeOrder
	})

	projected := make([]ProjectedNode, 0, len(matched))
	for i, def := range matched {
		reviewers, err := filterReviewers(def.Reviewers, scope)
		if err != nil {
			return nil, perr.Errorf("定义节点 %s reviewers 裁剪失败: %s", def.TemplateNodeID, err)
		}
		actions, err := filterNodeActions(def.NodeActions, scope)
		if err != nil {
			return nil, perr.Errorf("定义节点 %s node_actions 裁剪失败: %s", def.TemplateNodeID, err)
		}
		projected = append(projected, ProjectedNode{
			DefNodeID:   def.TemplateNodeID,
			NodeOrder:   int64(i + 1),
			NodeKey:     def.NodeKey,
			NodeType:    def.NodeType,
			Gate:        def.Gate,
			UserGate:    def.UserGate,
			Reviewers:   reviewers,
			NodeActions: actions,
		})
	}
	return projected, nil
}

// filterReviewers 评审位裁剪：ScopeExpr 未命中 scope 的位不进入实例
// （fe 位在 rd 需求零评审位——arch_design v0.29「FE 角色零节点零任务零评审位」）。
func filterReviewers(reviewersJSON string, scope string) (string, error) {
	var reviewers []repo.ReviewerDef
	if err := utils.UnmarshalJSONText(reviewersJSON, &reviewers); err != nil {
		return "", err
	}
	kept := make([]repo.ReviewerDef, 0, len(reviewers))
	for _, reviewer := range reviewers {
		if ScopeMatches(reviewer.ScopeExpr, scope) {
			kept = append(kept, reviewer)
		}
	}
	return utils.MarshalJSONText(kept)
}

// FilterNodeActions 激活清单裁剪（导出）：条目级 ScopeExpr（task 流程 by-scope 激活）
// 未命中 scope 的条目不参与投影；空=随节点恒在。实例化与 my_task 三向投影共用。
func FilterNodeActions(actionsJSON string, scope string) (string, error) {
	return filterNodeActions(actionsJSON, scope)
}

// filterNodeActions 激活清单裁剪：条目级 ScopeExpr（task 流程 by-scope 激活）
// 未命中 scope 的条目不进入实例；空=随节点恒在。
func filterNodeActions(actionsJSON string, scope string) (string, error) {
	var actions []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(actionsJSON, &actions); err != nil {
		return "", err
	}
	kept := make([]repo.NodeActionDef, 0, len(actions))
	for _, action := range actions {
		if ScopeMatches(action.ScopeExpr, scope) {
			kept = append(kept, action)
		}
	}
	return utils.MarshalJSONText(kept)
}
