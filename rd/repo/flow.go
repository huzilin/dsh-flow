// repo 包本文件：流程实例（flows/flow_nodes）与定义侧（flow_defs/flow_def_nodes/capabilities）
// 业务查询 + n-NN 实例节点 id 事务内取号（T4 流程实例化消费）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// flowNodeIDPrefix 实例节点 id 前缀（示例 n-05，两位零填充，超两位自然进位）。
const flowNodeIDPrefix = "n-"

// GetFlowByRequirementID 按需求 id 读流程实例（requirement 唯一，uk_flows_req）；
// 未命中返回 (nil, nil)（由 service 层转 ErrNotFound）。
func (r *Repo) GetFlowByRequirementID(ctx context.Context, requirementID string) (*Flow, error) {
	row := &Flow{}
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetFlowNodeByRequirementAndNodeID 读需求下指定实例节点；未命中返回 (nil, nil)。
func (r *Repo) GetFlowNodeByRequirementAndNodeID(ctx context.Context, requirementID string, nodeID string) (*FlowNode, error) {
	row := &FlowNode{}
	err := r.g.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ?", requirementID, nodeID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetFlowNodeByRequirementAndNodeIDLockTx 事务内行锁读取实例节点（SELECT … FOR UPDATE：
// collect 聚合判定读改写原子基线——wb_review 同事务写表态，并发双 collect 被行锁
// 串行化，spec §1.3/§2）；未命中返回 (nil, nil)。
func GetFlowNodeByRequirementAndNodeIDLockTx(ctx context.Context, tx *gorm.DB, requirementID string, nodeID string) (*FlowNode, error) {
	row := &FlowNode{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("requirement_id = ? AND node_id = ?", requirementID, nodeID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListFlowNodesByRequirementLockTx 事务内行锁读取需求全部实例节点（渲染序；
// SELECT … FOR UPDATE——T9 打回回退的批量读改写基线：回退目标与下游联动重置
// 单事务行锁，并发回退仅一次生效）；空集返回空切片。
func ListFlowNodesByRequirementLockTx(ctx context.Context, tx *gorm.DB, requirementID string) ([]*FlowNode, error) {
	items := make([]*FlowNode, 0)
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("requirement_id = ?", requirementID).
		Order("node_order").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// GetFlowDefByID 按流程类型读流程定义行；未命中返回 (nil, nil)。
func (r *Repo) GetFlowDefByID(ctx context.Context, flowDefsID string) (*FlowDef, error) {
	row := &FlowDef{}
	err := r.g.WithContext(ctx).
		Where("flow_defs_id = ?", flowDefsID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListFlowDefNodesByFlowDefsID 读流程全部定义节点（node_order 升序）。
func (r *Repo) ListFlowDefNodesByFlowDefsID(ctx context.Context, flowDefsID string) ([]*FlowDefNode, error) {
	items := make([]*FlowDefNode, 0)
	err := r.g.WithContext(ctx).
		Where("flow_defs_id = ?", flowDefsID).
		Order("node_order").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// ListFlowDefNodesByTemplateIDs 按定义节点 id 集合批量读定义节点
// （快照/my_task 装配 node_type/gate/user_gate 用）。
func (r *Repo) ListFlowDefNodesByTemplateIDs(ctx context.Context, templateNodeIDs []string) ([]*FlowDefNode, error) {
	if len(templateNodeIDs) == 0 {
		return make([]*FlowDefNode, 0), nil
	}
	items := make([]*FlowDefNode, 0, len(templateNodeIDs))
	err := r.g.WithContext(ctx).
		Where("template_node_id IN ?", templateNodeIDs).
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// ListCapabilities 全量读能力注册表（21 行量级，my_task 装配能力归属与动作词表用）。
func (r *Repo) ListCapabilities(ctx context.Context) ([]*Capability, error) {
	items := make([]*Capability, 0)
	err := r.g.WithContext(ctx).
		Order("capability_id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// GetCapabilityByID 按能力 id 读能力行（complete 判据期望产物解析用）；
// 未命中返回 (nil, nil)。
func (r *Repo) GetCapabilityByID(ctx context.Context, capabilityID string) (*Capability, error) {
	row := &Capability{}
	err := r.g.WithContext(ctx).
		Where("capability_id = ?", capabilityID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetFlowByRequirementIDTx 事务内读流程实例（collect 推进同事务更新
// current_node_id 前的基线读取；未命中返回 (nil, nil)）。
func GetFlowByRequirementIDTx(ctx context.Context, tx *gorm.DB, requirementID string) (*Flow, error) {
	row := &Flow{}
	err := tx.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetFlowByRequirementIDLockTx 事务内行锁读取流程实例（SELECT … FOR UPDATE）。
// 编排推进路径（wb_advance / complete 内聚触发 / 巡检补推 / review·decision 推进）
// 的并发防护基线（spec §1.2 拍板注记-C：flows 行锁串行化组推进，防双侧同时
// complete 的并发双推进；RC + 单 master 行锁已足）。锁序约定：全部推进路径
// 先锁 flows 行再锁 flow_nodes 行，避免交叉死锁。未命中返回 (nil, nil)。
func GetFlowByRequirementIDLockTx(ctx context.Context, tx *gorm.DB, requirementID string) (*Flow, error) {
	row := &Flow{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("requirement_id = ?", requirementID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListFlowsInOrchestration 读编排已启动的流程实例（current_node_id 非空）——
// 扫描器并行组兜底巡检的候选集。空集返回空切片。
func (r *Repo) ListFlowsInOrchestration(ctx context.Context) ([]*Flow, error) {
	items := make([]*Flow, 0)
	err := r.g.WithContext(ctx).
		Where("current_node_id IS NOT NULL").
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// ListFlowDefNodesByTemplateIDsTx 事务内按定义节点 id 集合批量读定义节点
// （推进路径的激活相位/并行组判定静态读取，与实例读同事务一致）。
func ListFlowDefNodesByTemplateIDsTx(ctx context.Context, tx *gorm.DB, templateNodeIDs []string) ([]*FlowDefNode, error) {
	if len(templateNodeIDs) == 0 {
		return make([]*FlowDefNode, 0), nil
	}
	items := make([]*FlowDefNode, 0, len(templateNodeIDs))
	err := tx.WithContext(ctx).
		Where("template_node_id IN ?", templateNodeIDs).
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// GetFlowDefNodeByTemplateIDTx 事务内读定义节点（激活相位/门禁位判定的静态 seed 读取）；
// 未命中返回 (nil, nil)（seed 异常由 service 层转错误）。
func GetFlowDefNodeByTemplateIDTx(ctx context.Context, tx *gorm.DB, templateNodeID string) (*FlowDefNode, error) {
	row := &FlowDefNode{}
	err := tx.WithContext(ctx).
		Where("template_node_id = ?", templateNodeID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// AllocNodeIDBaseTx 事务内取实例节点 id 序号基值：
// SELECT MAX(数字后缀) FROM flow_nodes WHERE node_id LIKE 'n-%' FOR UPDATE。
// node_id 全局唯一（uk_flow_nodes_nid），同事务内多节点由调用方本地递增派生
// （基值+n，不逐行回查）；并发创建由范围锁 + 唯一键兜底。
func (r *Repo) AllocNodeIDBaseTx(ctx context.Context, tx *gorm.DB) (int64, error) {
	base, err := maxIDNumberSuffixTx(ctx, tx, "flow_nodes", "node_id", flowNodeIDPrefix)
	if err != nil {
		return 0, err
	}
	return base, nil
}

// FormatNodeID 序号转实例节点 id（n-NN，两位零填充）。
func FormatNodeID(seq int64) string {
	return fmt.Sprintf("%s%02d", flowNodeIDPrefix, seq)
}
