// service 包本文件：wb_my_task（三向校验 role × node × 相位）与依赖树只读端点
// （分析师视图，动态加载）——spec §1.2 路由表两行。DTO 以 protoc 生成的 api 包类型承载。
package service

import (
	"context"
	"fmt"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// FlowService 流程实例读端点 service（wb_my_task + 依赖树）。
type FlowService struct {
	repo *repo.Repo
}

// NewFlowService 构造流程实例 service。
func NewFlowService(r *repo.Repo) *FlowService {
	return &FlowService{repo: r}
}

// MyTask wb_my_task：三向校验（role × node × 相位）的动作投影。
// 不属于该节点/该相位的角色返回空动作集而非报错（spec §1.2 口径）。
//
// 入参：req.RequirementId 必填；req.Role 必填（未知角色=空动作集）；
// req.NodeId 空=当前节点（flows.current_node_id；NULL=未进入编排）。
// 出参：node（目标节点摘要；未进入编排且未指定 nodeId 时未填充）、
// current_phase（=node.state，""=未进入编排）、active_actions（节点激活清单 ×
// 相位 on × 角色归属过滤）、review_summary（per-agent 聚合计数与明细）、
// related_nodes（该角色相关节点最小集：评审位 / node action 响应者 / 能力归属）、
// related_artifacts（产物侧上下文注入：本节点已登记产物摘要 + 评审节点的被评审
// 产物（直接上游 produce，BUG-011 口径）置前；无目标节点=空集）、
// related_tasks（该需求该节点在途任务 task_id/action/status 摘要，T5 任务侧注入）、
// related_issues（目标节点未闭合 issue 摘要：issueId/state/issueType/assignee/anchor，T7）。
//
// 错误集：ErrInvalidArgument（requirement_id/role 缺失）；ErrNotFound（需求 /
// 流程实例 / 节点未命中）。
func (s *FlowService) MyTask(ctx context.Context, req *api.MyTaskReq) (*api.MyTaskResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.Role == "" {
		return nil, invalidArg("role 必填")
	}
	requirement, err := s.repo.GetRequirementByID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if requirement == nil {
		return nil, notFound("需求 %q 不存在", req.RequirementId)
	}
	flow, err := s.repo.GetFlowByRequirementID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if flow == nil {
		return nil, notFound("需求 %q 无流程实例", req.RequirementId)
	}

	node, err := s.resolveTargetNode(ctx, req.RequirementId, flow, req.NodeId)
	if err != nil {
		return nil, err
	}

	nodeRows, err := s.repo.ListFlowNodesByRequirement(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	// 定义节点批量装配：激活清单在定义侧（flow_def_nodes.node_actions），
	// 按实例 scope 裁剪条目级 scope 后参与三向投影（与实例化同函数口径）
	defByTmplID, err := s.loadDefNodesByRequirement(ctx, nodeRows)
	if err != nil {
		return nil, errs.Trace(err)
	}
	capabilityRoles, capabilityInfos, err := s.loadCapabilityMaps(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}

	resp := &api.MyTaskResp{
		Ok:            true,
		RequirementId: req.RequirementId,
		Role:          req.Role,
	}

	// relatedNodes：全实例节点上的角色相关性（评审位 / node action 响应者 / 能力归属）
	refs := make([]domain.NodeRef, 0, len(nodeRows))
	for _, n := range nodeRows {
		nodeActions, filterErr := s.scopedNodeActions(defByTmplID, n, flow.Scope)
		if filterErr != nil {
			return nil, errs.Trace(filterErr)
		}
		refs = append(refs, domain.NodeRef{
			NodeID:      n.NodeID,
			DefNodeID:   n.DefNodeID,
			NodeOrder:   n.NodeOrder,
			NodeKey:     n.NodeKey,
			State:       n.State,
			Reviewers:   n.Reviewers,
			NodeActions: nodeActions,
		})
	}
	related, err := domain.RelatedNodesFor(req.Role, capabilityRoles, refs)
	if err != nil {
		return nil, errs.Trace(err)
	}
	relatedItems := make([]*api.RelatedNodeItem, 0, len(related))
	for _, ref := range related {
		relatedItems = append(relatedItems, &api.RelatedNodeItem{
			NodeId:    ref.NodeID,
			NodeKey:   ref.NodeKey,
			NodeOrder: ref.NodeOrder,
			State:     ref.State,
		})
	}
	resp.RelatedNodes = relatedItems

	// 无目标节点（未进入编排且未指定 nodeId）：相位空、动作空集，摘要零值
	if node == nil {
		resp.CurrentPhase = repo.PhaseEmpty
		resp.ActiveActions = make([]*api.MyTaskActionItem, 0)
		resp.ReviewSummary = &api.ReviewSummary{}
		resp.RelatedArtifacts = make([]*api.RelatedArtifactItem, 0)
		return resp, nil
	}

	// 产物侧上下文注入：目标节点已登记产物摘要（无登记=空集，不报错）；
	// 评审节点追加被评审产物（2026-09-10 BUG-011 拍板：直接上游 produce 节点
	// 已登记产物=提评来源，多上游全取）
	relatedArtifacts, err := s.loadRelatedArtifacts(ctx, req.RequirementId, node, nodeRows, defByTmplID)
	if err != nil {
		return nil, errs.Trace(err)
	}
	resp.RelatedArtifacts = relatedArtifacts

	nodeType := ""
	nodeActions := ""
	if defNode, ok := defByTmplID[node.DefNodeID]; ok {
		nodeType = defNode.NodeType
		filtered, filterErr := domain.FilterNodeActions(defNode.NodeActions, flow.Scope)
		if filterErr != nil {
			return nil, errs.Trace(filterErr)
		}
		nodeActions = filtered
	}
	resp.Node = &api.MyTaskNode{
		NodeId:    node.NodeID,
		DefNodeId: node.DefNodeID,
		NodeType:  nodeType,
		State:     node.State,
		Reviewers: node.Reviewers,
		// T9 重入上下文：exec 首产/返工配置选择依据 + 重开来源 + 评审敲定分支
		RolledBackCount: node.RolledBackCount,
		ReopenSource:    node.ReopenSource,
		FeatureBranch:   derefString(node.FeatureBranch),
	}
	resp.CurrentPhase = node.State

	active, err := domain.ActiveActionsFor(req.Role, node.State, nodeActions, capabilityInfos)
	if err != nil {
		return nil, errs.Trace(err)
	}
	// exec 相位首产/返工配置选择（T9）：rolled_back_count=0 → 首产集（produce/dev/
	// accept）；≥1 → 返工集（revise/diagnosing-bugs/retest）；集合外动作不受影响
	if node.State == repo.PhaseExec {
		active = domain.FilterExecActionsByReentry(active, node.RolledBackCount)
	}
	activeItems := make([]*api.MyTaskActionItem, 0, len(active))
	for _, action := range active {
		activeItems = append(activeItems, &api.MyTaskActionItem{
			ActionId:     action.ActionID,
			Name:         action.Name,
			Kind:         action.Kind,
			CapabilityId: action.CapabilityID,
			On:           action.On,
			Executor:     action.Executor,
			ExecutorRole: action.ExecutorRole,
			Skill:        action.Skill,
			Inline:       action.Inline,
		})
	}
	resp.ActiveActions = activeItems

	summary, err := domain.SummarizeReviewResults(node.ReviewResults)
	if err != nil {
		return nil, errs.Trace(err)
	}
	details := make([]*api.ReviewResultItem, 0, len(summary.Details))
	for _, result := range summary.Details {
		details = append(details, &api.ReviewResultItem{
			Role:         result.Role,
			CapabilityId: result.CapabilityID,
			State:        result.State,
			Round:        result.Round,
			At:           result.At,
		})
	}
	resp.ReviewSummary = &api.ReviewSummary{
		Agree:     summary.Agree,
		Unagree:   summary.Unagree,
		Reviewing: summary.Reviewing,
		Details:   details,
	}
	if err := s.appendRelatedTasks(ctx, resp, req.RequirementId, node.NodeID); err != nil {
		return nil, err
	}

	// relatedIssues（意见侧上下文注入，T7）：目标节点未闭合 issue 摘要
	// （state 非 resolved/overridden），集中独立函数。
	if err := s.injectRelatedIssues(ctx, resp, req.RequirementId, node.NodeID); err != nil {
		return nil, errs.Trace(err)
	}
	return resp, nil
}

// appendRelatedTasks wb_my_task 响应补 relatedTasks 摘要（T5 任务侧注入点，
// 独立函数便于 T6/T7 同文件改动合并）：该需求该节点在途任务（status 非
// completed/rejected/closed）的 task_id/action/status 最小集。
//
// 错误集：底层 IO 错误经 Trace 上抛。
func (s *FlowService) appendRelatedTasks(ctx context.Context, resp *api.MyTaskResp, requirementID string, nodeID string) error {
	tasks, err := s.repo.ListInflightTasksByRequirementAndNode(ctx, requirementID, nodeID)
	if err != nil {
		return errs.Trace(err)
	}
	items := make([]*api.RelatedTaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, &api.RelatedTaskItem{
			TaskId:   task.TaskID,
			ActionId: task.ActionID,
			Status:   task.Status,
		})
	}
	resp.RelatedTasks = items
	return nil
}

// injectRelatedIssues my_task 意见侧上下文注入：读目标节点未闭合评论
// （state NOT IN (resolved, overridden)），投影 RelatedIssueItem 最小摘要
// （issueId/state/issueType/assignee/anchor 锚点摘要）。查询失败仅阻断本次
// my_task（与主链路同错误口径，不降级吞错）。
//
// 错误集：底层 IO 错误经 Trace 上抛。
func (s *FlowService) injectRelatedIssues(ctx context.Context, resp *api.MyTaskResp, requirementID string, nodeID string) error {
	rows, err := s.repo.ListOpenIssuesByNode(ctx, requirementID, nodeID)
	if err != nil {
		return errs.Trace(err)
	}
	items := make([]*api.RelatedIssueItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &api.RelatedIssueItem{
			IssueId:   row.IssueID,
			State:     row.State,
			IssueType: row.IssueType,
			Assignee:  derefString(row.Assignee),
			Anchor: domain.SummarizeIssueAnchor(
				row.AnchorKind,
				derefString(row.AnchorArtifactID),
				derefInt64(row.AnchorLine),
				derefInt64(row.AnchorStartLine),
				derefInt64(row.AnchorEndLine),
				derefInt64(row.AnchorBlockStart),
				derefInt64(row.AnchorBlockEnd),
				derefString(row.AnchorLang),
			),
		})
	}
	resp.RelatedIssues = items
	return nil
}

// DependencyTree 依赖树只读端点：以当前需求为根，沿 flows.dependencies 递归展开
// （含对方 title/status/version；动态加载语义=分析师按需拉取，非常驻——
// ui-interaction-final §2）。依赖图成环时拒绝（ErrDependencyCycle）——树不可构建。
//
// 入参：req.RequirementId 根需求业务 id。
// 出参：root 子树（found=false=悬挂依赖：无对应需求行，不再展开）。
//
// 错误集：ErrInvalidArgument（requirement_id 缺失）；ErrNotFound（根需求不存在）；
// ErrDependencyCycle（依赖环）。
func (s *FlowService) DependencyTree(ctx context.Context, req *api.DependencyTreeReq) (*api.DependencyTreeResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	rootRow, err := s.repo.GetRequirementByID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if rootRow == nil {
		return nil, notFound("需求 %q 不存在", req.RequirementId)
	}

	// 沿依赖边收集可达子图（BFS；悬挂 id 无 flow 行，不展开）
	edges := domain.DependencyEdges{}
	reqCache := map[string]*repo.Requirement{req.RequirementId: rootRow}
	queue := []string{req.RequirementId}
	queued := map[string]bool{req.RequirementId: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		flow, err := s.repo.GetFlowByRequirementID(ctx, current)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if flow == nil {
			continue
		}
		var deps []string
		if err := utils.UnmarshalJSONText(flow.Dependencies, &deps); err != nil {
			return nil, errs.Trace(err)
		}
		edges[current] = deps
		for _, dep := range deps {
			if queued[dep] {
				continue
			}
			queued[dep] = true
			row, err := s.repo.GetRequirementByID(ctx, dep)
			if err != nil {
				return nil, errs.Trace(err)
			}
			if row != nil {
				reqCache[dep] = row
			}
			queue = append(queue, dep)
		}
	}

	if cycle := domain.FindDependencyCycle(req.RequirementId, edges); cycle != nil {
		return nil, fmt.Errorf("%w: 依赖环 %v", errs.ErrDependencyCycle, cycle)
	}

	resp := &api.DependencyTreeResp{
		Ok:   true,
		Root: s.buildDependencyNode(req.RequirementId, edges, reqCache, map[string]bool{req.RequirementId: true}),
	}
	return resp, nil
}

// loadRelatedArtifacts 目标节点产物摘要装配（my_task 产物侧上下文注入，T6）：
// 本节点已登记产物 + 评审节点的被评审产物（2026-09-10 BUG-011 拍板：node_type=
// review 的节点追加直接上游 produce 节点已登记产物=提评来源，被评审产物置前
// ——评审者先读被评审文档，零跳转）。登记集为空=无产物（空集不报错）；改动
// 收敛于本独立函数，便于并行工单对 flow.go 的多方合并。
func (s *FlowService) loadRelatedArtifacts(ctx context.Context, requirementID string, node *repo.FlowNode, nodeRows []*repo.FlowNode, defByTmplID map[string]*repo.FlowDefNode) ([]*api.RelatedArtifactItem, error) {
	items := make([]*api.RelatedArtifactItem, 0)
	if defNode, ok := defByTmplID[node.DefNodeID]; ok && defNode.NodeType == repo.NodeTypeReview {
		for _, upstream := range upstreamProduceNodes(defByTmplID, nodeRows, node) {
			if err := appendArtifactItems(ctx, s.repo, &items, requirementID, upstream.NodeID); err != nil {
				return nil, err
			}
		}
	}
	if err := appendArtifactItems(ctx, s.repo, &items, requirementID, node.NodeID); err != nil {
		return nil, err
	}
	return items, nil
}

// appendArtifactItems 单节点已登记产物摘要追加（登记序；loadRelatedArtifacts
// 装配子步）。
//
// 错误集：底层 IO 错误经 Trace 上抛。
func appendArtifactItems(ctx context.Context, r *repo.Repo, items *[]*api.RelatedArtifactItem, requirementID string, nodeID string) error {
	rows, err := r.ListArtifactsByRequirementAndNode(ctx, requirementID, nodeID)
	if err != nil {
		return errs.Trace(err)
	}
	for _, row := range rows {
		*items = append(*items, &api.RelatedArtifactItem{
			ArtifactId:  row.ArtifactID,
			ProductType: row.ProductType,
			Label:       row.Label,
			Kind:        row.Kind,
		})
	}
	return nil
}

// upstreamProduceNodes 评审节点的直接上游 produce 实例节点集（BUG-011 被评审
// 产物口径）：自紧邻前驱（node_order 降序）向前连续收集 produce 型节点，遇首个
// 非 produce 前驱停止——实例链线性，紧邻连续 produce 段即全部直接上游提评来源
// （多上游全取）；紧邻前驱即非 produce（纯对齐/门禁前驱）返回空集。
func upstreamProduceNodes(defByTmplID map[string]*repo.FlowDefNode, nodeRows []*repo.FlowNode, node *repo.FlowNode) []*repo.FlowNode {
	out := make([]*repo.FlowNode, 0)
	for i := len(nodeRows) - 1; i >= 0; i-- {
		candidate := nodeRows[i]
		if candidate.NodeOrder >= node.NodeOrder {
			continue
		}
		defNode, ok := defByTmplID[candidate.DefNodeID]
		if !ok || defNode.NodeType != repo.NodeTypeProduce {
			break
		}
		out = append([]*repo.FlowNode{candidate}, out...)
	}
	return out
}

// resolveTargetNode 解析 my_task 目标节点：显式 nodeId 优先；缺省取当前节点
// （NULL=未进入编排，返回 (nil, nil)）。
func (s *FlowService) resolveTargetNode(ctx context.Context, requirementID string, flow *repo.Flow, nodeID string) (*repo.FlowNode, error) {
	target := nodeID
	if target == "" {
		if flow.CurrentNodeID == nil {
			return nil, nil
		}
		target = *flow.CurrentNodeID
	}
	node, err := s.repo.GetFlowNodeByRequirementAndNodeID(ctx, requirementID, target)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if node == nil {
		return nil, notFound("节点 %q 不存在于需求 %q", target, requirementID)
	}
	return node, nil
}

// loadDefNodesByRequirement 需求实例行引用的定义节点批量装配
// （template_node_id → 定义行）。
func (s *FlowService) loadDefNodesByRequirement(ctx context.Context, nodeRows []*repo.FlowNode) (map[string]*repo.FlowDefNode, error) {
	seen := make(map[string]bool, len(nodeRows))
	ids := make([]string, 0, len(nodeRows))
	for _, n := range nodeRows {
		if !seen[n.DefNodeID] {
			seen[n.DefNodeID] = true
			ids = append(ids, n.DefNodeID)
		}
	}
	defNodes, err := s.repo.ListFlowDefNodesByTemplateIDs(ctx, ids)
	if err != nil {
		return nil, errs.Trace(err)
	}
	out := make(map[string]*repo.FlowDefNode, len(defNodes))
	for _, defNode := range defNodes {
		out[defNode.TemplateNodeID] = defNode
	}
	return out, nil
}

// scopedNodeActions 单节点激活清单取用：定义侧 JSON 按实例 scope 裁剪条目级 scope
// （定义缺失=seed 异常，返回空清单）。
func (s *FlowService) scopedNodeActions(defByTmplID map[string]*repo.FlowDefNode, node *repo.FlowNode, scope string) (string, error) {
	defNode, ok := defByTmplID[node.DefNodeID]
	if !ok {
		return "[]", nil
	}
	return domain.FilterNodeActions(defNode.NodeActions, scope)
}

// loadCapabilityMaps 能力装配：capability_id → 归属角色 / 投影信息（动作词表）。
func (s *FlowService) loadCapabilityMaps(ctx context.Context) (map[string]string, map[string]domain.CapabilityInfo, error) {
	caps, err := s.repo.ListCapabilities(ctx)
	if err != nil {
		return nil, nil, errs.Trace(err)
	}
	roles := make(map[string]string, len(caps))
	for _, cap := range caps {
		roles[cap.CapabilityID] = cap.RoleID
	}
	infos, err := buildCapabilityInfoMap(caps)
	if err != nil {
		return nil, nil, err
	}
	return roles, infos, nil
}

// buildDependencyNode 递归构建依赖树节点（环已由 FindDependencyCycle 拦截，
// path 集为纵深防御）。
func (s *FlowService) buildDependencyNode(requirementID string, edges domain.DependencyEdges, reqCache map[string]*repo.Requirement, path map[string]bool) *api.DependencyNodeItem {
	item := &api.DependencyNodeItem{
		RequirementId: requirementID,
		Found:         false,
		Children:      make([]*api.DependencyNodeItem, 0),
	}
	if row, ok := reqCache[requirementID]; ok {
		item.Found = true
		item.Title = row.Title
		item.Status = row.Status
		if row.VersionID != nil {
			item.VersionId = *row.VersionID
		}
	}
	for _, dep := range edges[requirementID] {
		if path[dep] {
			continue
		}
		childPath := make(map[string]bool, len(path)+1)
		for k, v := range path {
			childPath[k] = v
		}
		childPath[dep] = true
		item.Children = append(item.Children, s.buildDependencyNode(dep, edges, reqCache, childPath))
	}
	return item
}
