// service 包本文件：编排会话（架构评审 C1 深化，提案
// .plan/proposals/orchestration-session-plan.md）——编排事务生命周期、锁序纪律与
// 提交后通知 flush 的一处定义。包内私有（全部调用方在本包，不立对外接缝）。
//
// 口径：
//   - 锁序：全部编排路径先锁 flows 行再锁 flow_nodes 行（repo/flow.go 拍板注记-C②，
//     RC + 单 master 行锁已足；防双侧同时 complete 的并发双推进/交叉死锁）——
//     约定由本 module 落入类型，调用点不再人守（reject 历史颠倒点已随迁移改序）；
//   - 未命中：flow/node 未命中统一转 ErrNotFound（随事务回滚）；
//   - 提交后 flush：run 提交成功后恒 FlushPending——通知意图与业务提交原子
//     （EnqueueNotificationTx/outbox），推送失败不阻断（scanner SweepOutbox 补投）；
//   - 锁定读不缓存：session 的 nodes/node 每次调用即现锁现读（review needAdvance
//     等路径需要 Apply 之后的最新基线）。
package service

import (
	"context"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// orchestrator 编排会话入口（repo+notifier 无状态，各 service 构造期自建）。
type orchestrator struct {
	repo     *repo.Repo
	notifier *Notifier
}

// newOrchestrator 构造编排会话入口。
func newOrchestrator(r *repo.Repo) *orchestrator {
	return &orchestrator{repo: r, notifier: NewNotifier(r)}
}

// flowLockMode flow 行锁定模式。
type flowLockMode int

const (
	// flowLock 行锁（推进/回退/表态的并发基线——拍板注记-C②）
	flowLock flowLockMode = iota
	// flowRead 非锁读（reproject：flow 行无并发写面，锁由 nodes 清单承载）
	flowRead
)

// run 编排事务顶层入口：开事务 → 锁序基线（flows→flow_nodes）→ fn → 提交 →
// FlushPending。fn 内守卫拦截/未命中以错误返回随事务回滚。
func (o *orchestrator) run(ctx context.Context, requirementID string, fn func(*orchSession) error) error {
	return o.runMode(ctx, requirementID, flowLock, fn)
}

// runFlowRead run 的 flow 非锁读变体（reproject 专用，D3）。
func (o *orchestrator) runFlowRead(ctx context.Context, requirementID string, fn func(*orchSession) error) error {
	return o.runMode(ctx, requirementID, flowRead, fn)
}

func (o *orchestrator) runMode(ctx context.Context, requirementID string, mode flowLockMode, fn func(*orchSession) error) error {
	txErr := o.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		sess, err := o.open(ctx, tx, requirementID, mode)
		if err != nil {
			return err
		}
		return fn(sess)
	})
	if txErr != nil {
		return txErr
	}
	o.notifier.FlushPending(ctx)
	return nil
}

// lock 已开事务的中段接管（task complete 判据通过后的内嵌推进形状——其事务先锁
// 任务行，锁序不得换 run 的 flow 先锁，故事务生命周期归调用方、锁序与 run 同核）。
// flow 未命中返回 ErrNotFound（调用方决定容错口径，如 complete 跳过推进）。
func (o *orchestrator) lock(ctx context.Context, tx *gormio.DB, requirementID string) (*orchSession, error) {
	return o.open(ctx, tx, requirementID, flowLock)
}

func (o *orchestrator) open(ctx context.Context, tx *gormio.DB, requirementID string, mode flowLockMode) (*orchSession, error) {
	var flow *repo.Flow
	var err error
	if mode == flowRead {
		flow, err = repo.GetFlowByRequirementIDTx(ctx, tx, requirementID)
	} else {
		flow, err = repo.GetFlowByRequirementIDLockTx(ctx, tx, requirementID)
	}
	if err != nil {
		return nil, err
	}
	if flow == nil {
		return nil, notFound("需求 %q 无流程实例", requirementID)
	}
	return &orchSession{o: o, tx: tx, flowRow: flow}, nil
}

// orchSession 编排事务内基线与写入口（生命周期=单次事务；锁定读不缓存）。
type orchSession struct {
	o       *orchestrator
	tx      *gormio.DB
	flowRow *repo.Flow
}

// flow 编排流程实例基线（锁定读或非锁读，按会话模式）。
func (s *orchSession) flow() *repo.Flow { return s.flowRow }

// tx 事务句柄逃生舱（advanceNodeLocked 等既有收口函数消费；新写路径优先用会话方法）。
func (s *orchSession) txHandle() *gormio.DB { return s.tx }

// nodes 锁定读节点全清单（node_order 序；每次调用即当前基线）。
func (s *orchSession) nodes(ctx context.Context) ([]*repo.FlowNode, error) {
	return repo.ListFlowNodesByRequirementLockTx(ctx, s.tx, s.flowRow.RequirementID)
}

// node 锁定读单节点（未命中转 ErrNotFound）。
func (s *orchSession) node(ctx context.Context, nodeID string) (*repo.FlowNode, error) {
	row, err := repo.GetFlowNodeByRequirementAndNodeIDLockTx(ctx, s.tx, s.flowRow.RequirementID, nodeID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, notFound("节点 %q 不存在于需求 %q", nodeID, s.flowRow.RequirementID)
	}
	return row, nil
}

// apply 节点变更同源派生落库（C2 ApplyNodeChangeTx 转调；乐观锁基线=节点锁定行）。
func (s *orchSession) apply(ctx context.Context, node *repo.FlowNode, byWho string, changes ...repo.NodeChange) error {
	return repo.ApplyNodeChangeTx(ctx, s.o.repo, s.tx, node, byWho, changes...)
}

// moveCurrent flows.current_node_id 前移（乐观锁 + 审计）。已驻留目标=false 幂等
// 不写（推进核据此返回 noop；回退核的无条件前移在幂等护栏下不会重复到达）。
func (s *orchSession) moveCurrent(ctx context.Context, target *repo.FlowNode, byWho string) (bool, error) {
	flow := s.flowRow
	if flow.CurrentNodeID != nil && *flow.CurrentNodeID == target.NodeID {
		return false, nil
	}
	if err := repo.UpdateWithOptimisticLockTx(ctx, s.tx, &repo.Flow{}, flow.ID, flow.UpdatedAt, map[string]any{
		"current_node_id": target.NodeID,
	}); err != nil {
		return false, err
	}
	if err := s.o.repo.InsertAuditLogWith(s.tx, &repo.AuditLog{
		Table:    "flows",
		RowID:    flow.RequirementID,
		Action:   "UPDATE",
		Field:    utils.StrPtr("currentNodeId"),
		OldValue: utils.StrPtr(utils.DerefStr(flow.CurrentNodeID)),
		NewValue: utils.StrPtr(target.NodeID),
		ByWho:    byWho,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// notify 通知意图随业务同事务 enqueue（outbox 原子口径；推送失败不阻断）。
func (s *orchSession) notify(ctx context.Context, spec NotificationSpec) error {
	return EnqueueNotificationTx(ctx, s.tx, spec)
}
