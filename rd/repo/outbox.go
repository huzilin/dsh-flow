package repo

import (
	"context"
	"fmt"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"gorm.io/gorm"
)

// Outbox 状态机取值（spec §1.2 注记-H；2026-09-10 User 追加拍板 semantica
// `c25c0552`——至少一次投递，无限补投，无放弃态）：
//
//	pending（业务事务内 enqueue）→ sending（flush/scanner 认领推送中）
//	→ sent（issues 通知行落库成功）| failed（推送失败，retry_count++）；
//	failed 永久处于补投候选（scanner 周期重推直至成功）；
//	sending 停滞超 OutboxSendingStaleMs 视为进程崩溃残留，scanner 恢复认领。
const (
	OutboxStatusPending = "pending"
	OutboxStatusSending = "sending"
	OutboxStatusSent    = "sent"
	OutboxStatusFailed  = "failed"
)

// InsertOutboxTx 事务内 enqueue 一条通知意图（与业务写路径同事务，意图与业务
// 提交原子——「Dispatch 未被调用即丢」窗口由此消除）。
func InsertOutboxTx(ctx context.Context, tx *gorm.DB, row *Outbox) error {
	if err := tx.WithContext(ctx).Create(row).Error; err != nil {
		return errs.Trace(err)
	}
	return nil
}

// ClaimOutbox 原子认领一条待推进行（status → sending，乐观条件防双推并发）。
// 影响行数=0 表示已被并发方认领或状态已迁移，调用方跳过该行。
func (r *Repo) ClaimOutbox(ctx context.Context, id int64, fromStatuses []string) (bool, error) {
	q := r.g.WithContext(ctx).Model(&Outbox{}).
		Where("id = ? AND status IN ?", id, fromStatuses)
	res := q.Updates(map[string]any{"status": OutboxStatusSending, "updated_at": NowMillis()})
	if res.Error != nil {
		return false, errs.Trace(res.Error)
	}
	return res.RowsAffected == 1, nil
}

// SettleOutbox 推送落账：sent（成功，记 sent_at）或 failed（失败，retry_count++
// + last_error；retry_count 仅观测，无上限——无限补投）。
func (r *Repo) SettleOutbox(ctx context.Context, id int64, success bool, sendErr string) error {
	updates := map[string]any{"updated_at": NowMillis()}
	if success {
		updates["status"] = OutboxStatusSent
		updates["sent_at"] = NowMillis()
	} else {
		updates["status"] = OutboxStatusFailed
		updates["retry_count"] = gorm.Expr("retry_count + 1")
		updates["last_error"] = sendErr
	}
	if err := r.g.WithContext(ctx).Model(&Outbox{}).Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return errs.Trace(err)
	}
	return nil
}

// ListPendingOutbox 待推进行（status=pending，业务事务提交后 immediate flush 消费，
// 不做 stale 过滤——越新越该推）。
func (r *Repo) ListPendingOutbox(ctx context.Context, limit int) ([]*Outbox, error) {
	var rows []*Outbox
	if err := r.g.WithContext(ctx).
		Where("status = ?", OutboxStatusPending).
		Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, errs.Trace(err)
	}
	return rows, nil
}

// ListRetryableOutbox 补投候选（scanner 周期消费，无限补投——failed 永久在选）：
//   - failed（retry_count 仅观测，不设上限）；
//   - pending 停滞（immediate flush 未触达——业务提交后进程即崩的残留）；
//   - sending 停滞（认领后进程崩溃残留）。
//
// staleBeforeMs=now-stale 窗口（毫秒绝对值），避免与进行中的 immediate flush 抢行
// （抢到也有认领条件兜底，但减少无谓竞争）。
func (r *Repo) ListRetryableOutbox(ctx context.Context, staleBeforeMs int64, limit int) ([]*Outbox, error) {
	var rows []*Outbox
	if err := r.g.WithContext(ctx).
		Where(
			"(status = ?) OR (status = ? AND updated_at <= ?) OR (status = ? AND updated_at <= ?)",
			OutboxStatusFailed, OutboxStatusPending, staleBeforeMs, OutboxStatusSending, staleBeforeMs,
		).
		Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, errs.Trace(err)
	}
	return rows, nil
}

// GetOutbox 单行读（测试用）；不存在返回 (nil, nil)。
func (r *Repo) GetOutbox(ctx context.Context, id int64) (*Outbox, error) {
	var row Outbox
	if err := r.g.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return &row, nil
}

// OutboxStringErr 错误文本截断（last_error 列位 TEXT，防御性截断 2k）。
func OutboxStringErr(format string, args ...any) string {
	s := fmt.Sprintf(format, args...)
	if len(s) > 2048 {
		s = s[:2048]
	}
	return s
}
