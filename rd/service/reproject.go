// service 包本文件：T9 打回链路——pending_scope 范围重定与下游重投影。
//
// 职责（spec §4.1 打回链路 AC + capability-model-final §1.4「pending_scope 支持重定
// 范围」）：打回后目标节点持有 pending_scope 待重定标记（RollbackForRejectTx 落值），
// 本端点按重定 scope 重投影该节点下游节点集（复用 T4 投影函数 domain.
// ProjectFlowNodes：被裁节点删行不占位、新增定义节点插入新实例行、node_order
// 全量重排、评审位/激活清单按新 scope 重裁剪），随后清 pending_scope 继续编排。
//
// 事务边界：单事务——下游节点行锁读（ListFlowNodesByRequirementLockTx）→ 重投影
// 写（行级乐观锁更新/删除/插入）+ 逐字段审计（by_who=user）；flows.scope 与
// pending_scope 同事务落值。
package service

import (
	"context"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// Reproject wb_reproject（POST /api/requirements/:reqId/reproject）：范围重定投影。
// 幂等口径：无 pending_scope 待重定标记（未打回/已重投影）→ noop=true 库不变；
// scope 空=按当前 scope 重投影（仅清标记，节点集不变——同范围重排）。
//
// 上游保护：重定 scope 不得裁剪任何上游已渲染节点（含未激活行）与持有
// pending_scope 的节点自身——否则 blocked（打回重定只作用于下游节点集）。
//
// 入参：requirement_id 必填（reqId 取路径）；scope ∈ flow_defs.scope_options
// （空=当前 scope）。
// 出参：生效 scope / noop / 持有标记的节点 id / 重投影覆盖的下游节点行数。
//
// 错误集：ErrInvalidArgument（必填缺失/scope 越出 scope_options）；
// ErrNotFound（需求/流程实例/流程定义未命中）；ErrGateBlocked（重定 scope 裁剪
// 上游节点或标记节点自身）；ErrOptimisticLock；底层 IO/JSON 错误经 Trace 上抛。
func (s *FlowService) Reproject(ctx context.Context, req *api.ReprojectScopeReq) (*api.ReprojectScopeResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}

	resp := &api.ReprojectScopeResp{}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		flow, err := repo.GetFlowByRequirementIDTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}
		if flow == nil {
			return notFound("需求 %q 无流程实例", req.RequirementId)
		}

		nodes, err := repo.ListFlowNodesByRequirementLockTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}

		// scope 校验前置：越出 scope_options 属入参非法，与是否持有 pending_scope
		// 无关——先于幂等 no-op 判定拦截，非法入参不得借道 no-op 溜过
		newScope := req.Scope
		if newScope != "" {
			flowDef, err := s.repo.GetFlowDefByID(ctx, flow.FlowDefsID)
			if err != nil {
				return err
			}
			if flowDef == nil {
				return notFound("流程定义 %q 未命中", flow.FlowDefsID)
			}
			options, err := domain.ParseScopeOptions(flowDef.ScopeOptions)
			if err != nil {
				return err
			}
			valid := false
			for _, option := range options {
				if option == newScope {
					valid = true
					break
				}
			}
			if !valid {
				return invalidArg("scope %q 越出可选集 %v", newScope, options)
			}
		}

		var carrier *repo.FlowNode
		for _, node := range nodes {
			if node.PendingScope != nil {
				carrier = node
				break
			}
		}
		if carrier == nil {
			// 无待重定标记：幂等 no-op（重复重投影/未打回）
			resp.Ok = true
			resp.Scope = flow.Scope
			resp.Noop = true
			return nil
		}

		if newScope == "" {
			newScope = flow.Scope
		}

		defPtrs, err := s.repo.ListFlowDefNodesByFlowDefsID(ctx, flow.FlowDefsID)
		if err != nil {
			return err
		}
		defs := make([]repo.FlowDefNode, 0, len(defPtrs))
		for _, def := range defPtrs {
			defs = append(defs, *def)
		}
		projected, err := domain.ProjectFlowNodes(flow.FlowDefsID, newScope, defs)
		if err != nil {
			return err
		}
		byDefID := make(map[string]domain.ProjectedNode, len(projected))
		for _, p := range projected {
			byDefID[p.DefNodeID] = p
		}
		carrierProjectedOrder, err := validateReprojectScope(nodes, carrier, projected)
		if err != nil {
			return err
		}

		// 下游重投影（含 carrier）：保留重排 / 删除裁剪；上游行不动（前缀稳定）
		count := int64(0)
		for _, node := range nodes {
			if node.NodeOrder < carrier.NodeOrder {
				continue
			}
			count++
			p, stillRendered := byDefID[node.DefNodeID]
			if !stillRendered {
				if err := s.deleteFlowNodeTx(tx, node); err != nil {
					return err
				}
				continue
			}
			updates := make(map[string]any)
			audits := make([]repo.AuditLog, 0, 3)
			if node.NodeOrder != p.NodeOrder {
				updates["node_order"] = p.NodeOrder
				audits = append(audits, flowNodeAudit(node, "nodeOrder", formatMillis(node.NodeOrder), formatMillis(p.NodeOrder)))
			}
			if node.Reviewers != p.Reviewers {
				updates["reviewers"] = p.Reviewers
				audits = append(audits, flowNodeAudit(node, "reviewers", node.Reviewers, p.Reviewers))
			}
			if len(updates) > 0 {
				if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.FlowNode{}, node.ID, node.UpdatedAt, updates); err != nil {
					return err
				}
				for i := range audits {
					if err := s.repo.InsertAuditLogWith(tx, &audits[i]); err != nil {
						return err
					}
				}
			}
		}

		// 新增渲染的定义节点（重定扩展，如 rd → full 补 fe/QA 节点）：新实例行
		newDefs := make([]domain.ProjectedNode, 0)
		for _, p := range projected {
			if p.NodeOrder < carrierProjectedOrder {
				continue
			}
			exists := false
			for _, node := range nodes {
				if node.NodeOrder >= carrier.NodeOrder && node.DefNodeID == p.DefNodeID {
					exists = true
					break
				}
			}
			if !exists {
				newDefs = append(newDefs, p)
			}
		}
		if len(newDefs) > 0 {
			base, err := s.repo.AllocNodeIDBaseTx(ctx, tx)
			if err != nil {
				return err
			}
			for i, p := range newDefs {
				nodeID := repo.FormatNodeID(base + int64(i) + 1)
				if err := tx.Create(&repo.FlowNode{
					RequirementID: req.RequirementId,
					NodeID:        nodeID,
					DefNodeID:     p.DefNodeID,
					NodeOrder:     p.NodeOrder,
					NodeKey:       p.NodeKey,
					State:         repo.PhaseEmpty,
					ReviewResults: "[]",
					Reviewers:     p.Reviewers,
				}).Error; err != nil {
					return errs.Trace(err)
				}
				if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
					Table:  "flow_nodes",
					RowID:  nodeID,
					Action: "INSERT",
					ByWho:  auditByWhoUser,
				}); err != nil {
					return err
				}
				count++
			}
		}

		// 清 pending_scope 标记 + flows.scope 落值（变更时）
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.FlowNode{}, carrier.ID, carrier.UpdatedAt, map[string]any{
			"pending_scope": nil,
		}); err != nil {
			return err
		}
		if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:    "flow_nodes",
			RowID:    carrier.NodeID,
			Action:   "UPDATE",
			Field:    utils.StrPtr("pendingScope"),
			OldValue: utils.StrPtr(derefString(carrier.PendingScope)),
			NewValue: utils.StrPtr(""),
			ByWho:    auditByWhoUser,
		}); err != nil {
			return err
		}
		if newScope != flow.Scope {
			if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Flow{}, flow.ID, flow.UpdatedAt, map[string]any{
				"scope": newScope,
			}); err != nil {
				return err
			}
			if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
				Table:    "flows",
				RowID:    flow.RequirementID,
				Action:   "UPDATE",
				Field:    utils.StrPtr("scope"),
				OldValue: utils.StrPtr(flow.Scope),
				NewValue: utils.StrPtr(newScope),
				ByWho:    auditByWhoUser,
			}); err != nil {
				return err
			}
		}

		resp.Ok = true
		resp.Scope = newScope
		resp.NodeId = carrier.NodeID
		resp.ReprojectedCount = count
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return resp, nil
}

// validateReprojectScope 上游保护（前缀稳定性校验）：重定 scope 后
//
//   - 持有 pending_scope 的节点自身与全部上游已渲染节点必须仍然渲染（否则打回
//     回退链断裂）；
//   - 上游定义节点集不得新增（投影前缀与既有上游行 def 集一致——上游行不动、
//     node_order 前缀稳定的保证）。
//
// 返回 carrier 在新投影下的渲染序（=既有 node_order，前缀稳定下相等）。
//
// 错误集：ErrGateBlocked。
func validateReprojectScope(nodes []*repo.FlowNode, carrier *repo.FlowNode, projected []domain.ProjectedNode) (int64, error) {
	byDefID := make(map[string]domain.ProjectedNode, len(projected))
	for _, p := range projected {
		byDefID[p.DefNodeID] = p
	}
	carrierProjected, ok := byDefID[carrier.DefNodeID]
	if !ok {
		return 0, gateBlocked("重定 scope 裁剪回退目标节点 %q（%s），拒绝重投影", carrier.NodeID, carrier.DefNodeID)
	}
	for _, node := range nodes {
		if node.NodeOrder >= carrier.NodeOrder {
			continue
		}
		if _, still := byDefID[node.DefNodeID]; !still {
			return 0, gateBlocked("重定 scope 裁剪上游节点 %q（%s），拒绝重投影", node.NodeID, node.DefNodeID)
		}
	}
	// 前缀新增校验：新投影中 carrier 之前的 def 集 = 既有上游行 def 集
	upstreamDefs := make(map[string]bool, carrier.NodeOrder)
	for _, node := range nodes {
		if node.NodeOrder < carrier.NodeOrder {
			upstreamDefs[node.DefNodeID] = true
		}
	}
	for _, p := range projected {
		if p.NodeOrder < carrierProjected.NodeOrder && !upstreamDefs[p.DefNodeID] {
			return 0, gateBlocked("重定 scope 在回退目标 %q 上游新增节点 %q（%s），拒绝重投影", carrier.NodeID, p.DefNodeID, p.DefNodeID)
		}
	}
	return carrierProjected.NodeOrder, nil
}

// deleteFlowNodeTx 裁剪节点删除（乐观锁语义：updated_at 匹配才删；审计 DELETE）。
func (s *FlowService) deleteFlowNodeTx(tx *gormio.DB, node *repo.FlowNode) error {
	res := tx.Where("id = ? AND updated_at = ?", node.ID, node.UpdatedAt).
		Delete(&repo.FlowNode{})
	if res.Error != nil {
		return errs.Trace(res.Error)
	}
	if res.RowsAffected == 0 {
		return errs.Trace(errs.ErrOptimisticLock)
	}
	return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
		Table:  "flow_nodes",
		RowID:  node.NodeID,
		Action: "DELETE",
		ByWho:  auditByWhoUser,
	})
}

// flowNodeAudit flow_nodes UPDATE 审计条目（重投影 by_who=user 口径）。
func flowNodeAudit(node *repo.FlowNode, field string, oldVal string, newVal string) repo.AuditLog {
	return repo.AuditLog{
		Table:    "flow_nodes",
		RowID:    node.NodeID,
		Action:   "UPDATE",
		Field:    utils.StrPtr(field),
		OldValue: utils.StrPtr(oldVal),
		NewValue: utils.StrPtr(newVal),
		ByWho:    auditByWhoUser,
	}
}
