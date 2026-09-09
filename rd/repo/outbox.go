package repo

import (
	"context"
	"fmt"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"gorm.io/gorm"
)

// Outbox 状态机取值（spec §1.2 注记-H）：
//
//	pending（业务事务内 enqueue）→ sending（flush/scanner 认领推送中）
//	→ sent（issues 通知行落库成功）| failed（推送失败，retry_count++）；
//	failed 且 retry_count<OutboxRetryLimit 可再认领；耗尽置 dead（终态+audit）；
//	sending 停滞超 OutboxSendingStaleMs 视为进程崩溃残留，scanner 恢复认领。
const (
	OutboxStatusPending = "pending"
	OutboxStatusSending = "sending"
	OutboxStatusSent    = "sent"
	OutboxStatusFailed  = "failed"
	OutboxStatusDead    = "dead"

	// OutboxRetryLimit 补投重试上限（retry_count 达限置 dead 终态+audit 告警）。
	OutboxRetryLimit = 5
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
// + last_error）。返回失败路径自增后的 retry_count（调用方判定 retry 耗尽置 dead）；
// 成功路径返回 0。
func (r *Repo) SettleOutbox(ctx context.Context, id int64, success bool, sendErr string) (int64, error) {
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
		return 0, errs.Trace(err)
	}
	if success {
		return 0, nil
	}
	var row Outbox
	if err := r.g.WithContext(ctx).Select("retry_count").Where("id = ?", id).
		First(&row).Error; err != nil {
		return 0, errs.Trace(err)
	}
	return row.RetryCount, nil
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

// ListRetryableOutbox 补投候选（scanner 周期消费）：
//   - failed 且 retry_count<OutboxRetryLimit（重试预算内）；
//   - pending 停滞（immediate flush 未触达——业务提交后进程即崩的残留）；
//   - sending 停滞（认领后进程崩溃残留）。
//
// staleBeforeMs=now-stale 窗口（毫秒绝对值），避免与进行中的 immediate flush 抢行
// （抢到也有认领条件兜底，但减少无谓竞争）。
func (r *Repo) ListRetryableOutbox(ctx context.Context, staleBeforeMs int64, limit int) ([]*Outbox, error) {
	var rows []*Outbox
	if err := r.g.WithContext(ctx).
		Where(
			"(status = ? AND retry_count < ?) OR (status = ? AND updated_at <= ?) OR (status = ? AND updated_at <= ?)",
			OutboxStatusFailed, OutboxRetryLimit, OutboxStatusPending, staleBeforeMs, OutboxStatusSending, staleBeforeMs,
		).
		Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, errs.Trace(err)
	}
	return rows, nil
}

// DeadOutboxWithAlert retry 耗尽终态化（dead，不再被任何认领/补投选取）+ audit
// 告警留痕一次（audit_log：table=notify_outbox，field=status failed→dead）。
func (r *Repo) DeadOutboxWithAlert(ctx context.Context, id int64) error {
	txErr := r.g.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.WithContext(ctx).Model(&Outbox{}).
			Where("id = ? AND status = ?", id, OutboxStatusFailed).
			Updates(map[string]any{"status": OutboxStatusDead, "updated_at": NowMillis()})
		if res.Error != nil {
			return errs.Trace(res.Error)
		}
		if res.RowsAffected == 0 {
			return nil
		}
		return r.InsertAuditLogWith(tx, &AuditLog{
			Table:    "notify_outbox",
			RowID:    fmt.Sprintf("%d", id),
			Action:   "UPDATE",
			Field:    utils.StrPtr("status"),
			OldValue: utils.StrPtr(OutboxStatusFailed),
			NewValue: utils.StrPtr(OutboxStatusDead),
			ByWho:    "system",
		})
	})
	if txErr != nil {
		return errs.Trace(txErr)
	}
	return nil
}

// GetOutbox 单行读（dead 判定/测试用）；不存在返回 (nil, nil)。
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
