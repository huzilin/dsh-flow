// repo 包本文件：sessions 表业务查询、ad-<ts> 裁决 session 业务 id 事务内取号、
// 行锁读取（T12 冲突上升裁决链——裁决 session 建绑与终局「事务 + 行锁 + 乐观锁更新」
// 基线，与 issues 取号同构）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// sessionIDPrefix 裁决 session 业务 id 前缀（spec §0 sessions.session_id「会话业务 id」
// 未定字面格式，T12 裁决通道取 ad-<ts>（adjudication），取号口径与 issues cm-<ts> 同构：
// 毫秒时间戳与序号基值取大，并发同毫秒由 FOR UPDATE 基值递增兜底）。
const sessionIDPrefix = "ad-"

// session 状态常量镜像（sessions.status 列注释：active/ended/disposed；
// T12 裁决链消费 active/ended 两值，disposed 留 DSH 会话资源回收工单）。
const (
	SessionStatusActive   = "active"
	SessionStatusEnded    = "ended"
	SessionStatusDisposed = "disposed"
)

// 裁决 session 绑定角色（spec §0 sessions.role_id「绑定角色」；T12 仲裁语义恒 user
// ——User 出结论，reviewer 主持呈报，capability-model-final §6 rule6/8 口径）。
const SessionRoleAdjudicator = "user"

// dshRefPlaceholder DSH 会话句柄占位（T12 裁决 session 建绑先行，DSH 泛用会话通道
// 接线属后续工单——capability-model-final §6 rule7「按需会话通道」plugin 侧接线）。
const dshRefPlaceholder = "pending-dsh-adjudication"

// GetSessionBySessionID 按会话业务 id 读会话行；未命中返回 (nil, nil)（service 转 ErrNotFound）。
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

// GetSessionBySessionIDLockTx 事务内行锁读取会话行（SELECT … FOR UPDATE：裁决终局
// 判定与状态迁移原子——并发双 adjudicate 串行化，仅一次生效）；未命中返回 (nil, nil)。
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

// CreateSessionTx 事务内建会话绑定行（T12 裁决 session：requirement_id/node_id 关联
// 冲突 issue 挂载节点，role_id=user 仲裁语义，dsh_ref 占位）。
func (r *Repo) CreateSessionTx(ctx context.Context, tx *gorm.DB, row *Session) error {
	if err := tx.WithContext(ctx).Create(row).Error; err != nil {
		return errs.Trace(err)
	}
	return nil
}

// AllocSessionIDTx 事务内取下一个裁决 session 业务 id（ad-<ts>）：
// MAX(数字后缀) FOR UPDATE 与当前毫秒时间戳取大 + 1（uk_sessions_sid 唯一键最终兜底，
// 锁竞争由调用方整体重试）。
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

// NewAdjudicationSession 裁决 session 绑定行装配（建绑字段单一口径：
// role=user 仲裁语义、dsh_ref 占位、status=active；sessionID 由 AllocSessionIDTx 取号）。
func NewAdjudicationSession(sessionID string, requirementID string, nodeID string) *Session {
	return &Session{
		SessionID:     sessionID,
		RequirementID: requirementID,
		NodeID:        nodeID,
		RoleID:        SessionRoleAdjudicator,
		DshRef:        dshRefPlaceholder,
		Status:        SessionStatusActive,
	}
}
