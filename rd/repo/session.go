// repo 包本文件：sessions 表业务查询与 s-<ts>-<seq> 业务 id 事务内取号（spec §0
// 表 #10 + T10 工单）。写路径消费方为 service.SessionService（事务 + 审计在上层）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// sessionIDPrefix 会话业务 id 前缀（形态对齐 cm- 家族：s-<ts>，ts=取号时刻
// unix 毫秒；同毫秒并发由基值递增兜底）。
const sessionIDPrefix = "s-"

// GetSessionBySessionID 按业务 id 读会话行；未命中返回 (nil, nil)（service 转 ErrNotFound）。
func (r *Repo) GetSessionBySessionID(ctx context.Context, sessionID string) (*Session, error) {
	row := &Session{}
	err := r.g.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetSessionBySessionIDLockTx 事务内行锁读取会话行（SELECT … FOR UPDATE：状态
// 迁移读改写原子基线）；未命中返回 (nil, nil)。
func (r *Repo) GetSessionBySessionIDLockTx(ctx context.Context, tx *gorm.DB, sessionID string) (*Session, error) {
	row := &Session{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("session_id = ?", sessionID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListSessionsByRequirement 读需求下全部会话绑定（id 升序）；requirementID 为空
// 返回空集（DDL 语义：requirement_id 空串=无关联）。
func (r *Repo) ListSessionsByRequirement(ctx context.Context, requirementID string) ([]*Session, error) {
	if requirementID == "" {
		return make([]*Session, 0), nil
	}
	items := make([]*Session, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// FindActiveSessionByTaskKey 按任务键（requirement_id+node_id+role_id）查 active
// 会话绑定（扫描器 running 代理判据数据源）；未命中返回 (nil, nil)。
func (r *Repo) FindActiveSessionByTaskKey(ctx context.Context, requirementID string, nodeID string, roleID string) (*Session, error) {
	row := &Session{}
	err := r.g.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ? AND role_id = ? AND status = ?", requirementID, nodeID, roleID, "active").
		Order("id").
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// AllocSessionIDTx 事务内取下一个会话业务 id（s-<ts>）：MAX(数字后缀) FOR UPDATE
// 与当前毫秒时间戳取大 + 1（时间戳形态优先；同毫秒并发/回拨由序号基值单调递增兜底），
// uk_sessions_sid 唯一键最终兜底。口径与 AllocIssueIDTx 同构。
func (r *Repo) AllocSessionIDTx(ctx context.Context, tx *gorm.DB) (string, error) {
	base, err := maxIDNumberSuffixTx(ctx, tx, "sessions", "session_id", sessionIDPrefix)
	if err != nil {
		return "", err
	}
	next := NowMillis()
	if base >= next {
		next = base + 1
	}
	return fmt.Sprintf("%s%d", sessionIDPrefix, next), nil
}
