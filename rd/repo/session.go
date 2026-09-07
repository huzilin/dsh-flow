// repo 包本文件：sessions 表业务查询、s-<ts> 会话业务 id 事务内取号（spec §0
// 表 #10 + T10 工单，写路径消费方为 service.SessionService，事务 + 审计在上层）、
// ad-<ts> 裁决 session 取号与建绑（T12 冲突上升裁决链——裁决 session 建绑与终局
// 「事务 + 行锁 + 乐观锁更新」基线，与 issues 取号同构）。
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

// adjudicationSessionIDPrefix 裁决 session 业务 id 前缀（spec §0 sessions.session_id
// 「会话业务 id」未定字面格式，T12 裁决通道取 ad-<ts>（adjudication），取号口径与
// issues cm-<ts> 同构：毫秒时间戳与序号基值取大，并发同毫秒由 FOR UPDATE 基值递增兜底）。
const adjudicationSessionIDPrefix = "ad-"

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
// 迁移读改写原子——并发双 adjudicate 终局判定串行化，仅一次生效）；未命中返回 (nil, nil)。
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

// AllocAdjudicationSessionIDTx 事务内取下一个裁决 session 业务 id（ad-<ts>）：
// MAX(数字后缀) FOR UPDATE 与当前毫秒时间戳取大 + 1（uk_sessions_sid 唯一键最终兜底，
// 锁竞争由调用方整体重试）。
func (r *Repo) AllocAdjudicationSessionIDTx(ctx context.Context, tx *gorm.DB) (string, error) {
	base, err := maxIDNumberSuffixTx(ctx, tx, "sessions", "session_id", adjudicationSessionIDPrefix)
	if err != nil {
		return "", err
	}
	next := NowMillis()
	if base >= next {
		next = base + 1
	}
	return fmt.Sprintf("%s%d", adjudicationSessionIDPrefix, next), nil
}

// NewAdjudicationSession 裁决 session 绑定行装配（建绑字段单一口径：
// role=user 仲裁语义、dsh_ref 占位、status=active；sessionID 由
// AllocAdjudicationSessionIDTx 取号）。
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
