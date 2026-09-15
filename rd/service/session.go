// service 包本文件：会话绑定 service（T10）——flow↔session 绑定 CRUD 与 status
// 生命周期（spec §0 表 #10 sessions）。DSH 嵌入适配器 out of scope（spec「Out of
// Scope」），dsh_ref 以句柄占位承载；角色说明（note）按节点配置传入以解析器接口
// 占位，真实接线与验收在 T15+。路由表（spec §1.2）无 session 端点：本期为 service 层
// 能力（扫描器与后续 plugin 泛用拉起接口消费），不出 HTTP。
package service

import (
	"context"
	"fmt"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// 会话绑定审计 by_who 口径：拉起方=plugin（v3 会话通道由 plugin 泛用拉起接口驱动，
// ui-interaction-final §4「会话 tab/按钮」行）；生命周期流转默认 system。
const (
	auditByWhoSessionBind = "plugin"
	auditByWhoSystem      = "system"
)

// RoleNoteResolver 角色说明（note）按节点配置解析接口（capability-model-final：
// subagent 角色说明按节点配置传入；「persona」为已退役旧名，08-18 拍板改称说明/注释）。
// 本期占位：无实现注册时按空说明处理；真实数据源（节点配置侧）与验收在 T15+ 接线。
type RoleNoteResolver interface {
	// NoteFor 返回该需求该节点该角色的角色说明文本；未配置返回 ("", nil)。
	NoteFor(ctx context.Context, requirementID string, nodeID string, roleID string) (string, error)
}

// SessionService 会话绑定 service（绑定 CRUD + status 生命周期 + 角色说明占位）。
type SessionService struct {
	repo     *repo.Repo
	noteRes  RoleNoteResolver
	endByWho string
}

// NewSessionService 构造会话绑定 service（resolver 可为 nil=无角色说明占位）。
func NewSessionService(r *repo.Repo, resolver RoleNoteResolver) *SessionService {
	return &SessionService{repo: r, noteRes: resolver, endByWho: auditByWhoSystem}
}

// SessionBindInput 绑定入参。
type SessionBindInput struct {
	// RequirementID 关联需求；空串=无（DDL 语义）
	RequirementID string
	// NodeID 关联节点；空串=无
	NodeID string
	// RoleID 绑定角色（角色说明按节点配置传入的归属角色，必填）
	RoleID string
	// DshRef DSH 会话句柄；空串=占位句柄（本期无真实 DSH，落 placeholder JSON）
	DshRef string
	// ByWho 审计操作者；空串默认 plugin（拉起方口径）
	ByWho string
}

// Bind 建立会话绑定：session_id 事务内取号（s-<ts>）+ INSERT（status=active）+
// 审计。dsh_ref 空串时落占位句柄 JSON（{"kind":"placeholder","note":…}，
// note 经 RoleNoteResolver 按节点配置解析——真实接线 T15+，本期默认空）。
// 同任务键重复绑定不去重（多轮会话各占一行，按 session_id 区分；active 代理判据
// 取最早一行，口径见 repo.FindActiveSessionByTaskKey）。
//
// 入参：RoleID 必填。
// 出参：落库后绑定行。
//
// 错误集：ErrInvalidArgument（role_id 缺失）；底层 IO/取号错误经 Trace 上抛。
func (s *SessionService) Bind(ctx context.Context, in SessionBindInput) (*repo.Session, error) {
	if in.RoleID == "" {
		return nil, invalidArg("role_id 必填")
	}
	byWho := in.ByWho
	if byWho == "" {
		byWho = auditByWhoSessionBind
	}
	dshRef := in.DshRef
	if dshRef == "" {
		note := ""
		if s.noteRes != nil {
			text, err := s.noteRes.NoteFor(ctx, in.RequirementID, in.NodeID, in.RoleID)
			if err != nil {
				return nil, errs.Trace(err)
			}
			note = text
		}
		dshRef = placeholderDshRef(note)
	}
	row := &repo.Session{
		RequirementID: in.RequirementID,
		NodeID:        in.NodeID,
		RoleID:        in.RoleID,
		DshRef:        dshRef,
		Status:        domain.SessionStatusActive,
	}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		sessionID, err := s.repo.AllocSessionIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.SessionID = sessionID
		if err := tx.Create(row).Error; err != nil {
			return errs.Trace(err)
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "sessions",
			RowID:  sessionID,
			Action: "INSERT",
			ByWho:  byWho,
		})
	})
	if txErr != nil {
		return nil, txErr
	}
	return row, nil
}

// Get 按业务 id 读绑定行；未命中返回 ErrNotFound。
//
// 错误集：ErrNotFound（session_id 未命中）；底层 IO 错误经 Trace 上抛。
func (s *SessionService) Get(ctx context.Context, sessionID string) (*repo.Session, error) {
	row, err := s.repo.GetSessionBySessionID(ctx, sessionID)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if row == nil {
		return nil, notFound("会话 %q 不存在", sessionID)
	}
	return row, nil
}

// ListByRequirement 读需求下全部绑定行（id 升序）。
//
// 错误集：底层 IO 错误经 Trace 上抛。
func (s *SessionService) ListByRequirement(ctx context.Context, requirementID string) ([]*repo.Session, error) {
	return s.repo.ListSessionsByRequirement(ctx, requirementID)
}

// End 结束会话（active→ended）；同态幂等 no-op（noop=true）。
//
// 错误集：ErrNotFound（未命中）；ErrInvalidTransition（非法迁移，409）；
// ErrOptimisticLock（基线冲突，409）；底层 IO 错误经 Trace 上抛。
func (s *SessionService) End(ctx context.Context, sessionID string) (*repo.Session, bool, error) {
	return s.transition(ctx, sessionID, domain.SessionStatusEnded)
}

// Dispose 处置会话（ended→disposed，回收句柄终态）；同态幂等 no-op（noop=true）。
//
// 错误集：ErrNotFound（未命中）；ErrInvalidTransition（非法迁移，409）；
// ErrOptimisticLock（基线冲突，409）；底层 IO 错误经 Trace 上抛。
func (s *SessionService) Dispose(ctx context.Context, sessionID string) (*repo.Session, bool, error) {
	return s.transition(ctx, sessionID, domain.SessionStatusDisposed)
}

// transition 状态迁移单事务（行读 → domain 迁移表判定 → 乐观锁更新 + 审计）：
// 同态 no-op 不写库不写审计；非法迁移 ErrInvalidTransition（active→disposed 跳段、
// 回退、disposed 后再迁一律拦截）。
func (s *SessionService) transition(ctx context.Context, sessionID string, to string) (*repo.Session, bool, error) {
	var (
		row  *repo.Session
		noop bool
	)
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		locked, err := s.repo.GetSessionBySessionIDLockTx(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if locked == nil {
			return notFound("会话 %q 不存在", sessionID)
		}
		allowed, changed := domain.TransitionSessionStatus(locked.Status, to)
		if !allowed {
			return errs.Trace(fmt.Errorf("%w: 会话 %q 状态 %q 不可迁移至 %q（合法链 active→ended→disposed）",
				errs.ErrInvalidTransition, sessionID, locked.Status, to))
		}
		if !changed {
			row = locked
			noop = true
			return nil
		}
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Session{}, locked.ID, locked.UpdatedAt, map[string]any{
			"status": to,
		}); err != nil {
			return err
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:    "sessions",
			RowID:    sessionID,
			Action:   "UPDATE",
			Field:    utils.StrPtr("status"),
			OldValue: utils.StrPtr(locked.Status),
			NewValue: utils.StrPtr(to),
			ByWho:    s.endByWho,
		})
	})
	if txErr != nil {
		return nil, false, txErr
	}
	if noop {
		return row, true, nil
	}
	fresh, err := s.repo.GetSessionBySessionID(ctx, sessionID)
	if err != nil {
		return nil, false, errs.Trace(err)
	}
	if fresh == nil {
		return nil, false, notFound("会话 %q 不存在", sessionID)
	}
	return fresh, false, nil
}

// placeholderDshRef 占位句柄装配（本期无真实 DSH）：{"kind":"placeholder",
// "note":"<按节点配置解析>"}；真实 DSH 句柄接入后由嵌入适配器替换（T15+）。
func placeholderDshRef(note string) string {
	return fmt.Sprintf(`{"kind":"placeholder","note":%q}`, note)
}
