// repo 包本文件：流程实例（flows/flow_nodes）与定义侧（flow_defs/flow_def_nodes/capabilities）
// 业务查询 + n-NN 实例节点 id 事务内取号（T4 流程实例化消费）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

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
