// service 包本文件：wb_list / wb_create / wb_update_version、wb_advance_version
// （发布软约束）、wb_deprecate_version（有引用 blocked）五端点业务编排。
package service

import (
	"context"

	gormio "gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// VersionService 版本管理面 service（versions 五端点）。
type VersionService struct {
	repo *repo.Repo
}

// NewVersionService 构造版本 service。
func NewVersionService(r *repo.Repo) *VersionService {
	return &VersionService{repo: r}
}

// List wb_list_versions：全量版本列表（含待分配池标记行），创建序（id ASC）。
//
// 错误集：无业务错误；底层 IO 错误经 Trace 上抛（HTTP 侧 INTERNAL）。
func (s *VersionService) List(ctx context.Context, req *api.ListVersionsReq) (*api.ListVersionsResp, error) {
	rows, err := s.repo.ListVersions(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}
	items := make([]*api.VersionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, versionItem(row))
	}
	return &api.ListVersionsResp{Ok: true, Items: items}, nil
}

// Create wb_create_version：version_id 取号（v-NNN 递增，事务内取号防重，
// 竞态整体重试）+ 默认 state=待启动；创建动作同事务写 audit_log（INSERT）。
//
// 入参：name 必填；is_backlog 标记待分配池。
// 出参：创建后的版本行（含生成的 version_id）。
//
// 错误集：ErrInvalidArgument（name 缺失）；ErrIdempotentConflict（取号重试耗尽）。
func (s *VersionService) Create(ctx context.Context, req *api.CreateVersionReq) (*api.CreateVersionResp, error) {
	if req.Name == "" {
		return nil, invalidArg("name 必填")
	}
	row, err := withIDAllocRetry(func() (*repo.Version, bool, error) {
		return s.createOnce(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return &api.CreateVersionResp{Ok: true, Version: versionItem(row)}, nil
}

// createOnce 单次创建尝试（事务：取号 → 插入 → 审计）。
func (s *VersionService) createOnce(ctx context.Context, req *api.CreateVersionReq) (*repo.Version, bool, error) {
	row := &repo.Version{
		Name:      req.Name,
		State:     domain.VerStatePending,
		IsBacklog: boolToInt64(req.IsBacklog),
	}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		versionID, err := s.repo.AllocVersionIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.VersionID = versionID
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "versions",
			RowID:  versionID,
			Action: "INSERT",
			ByWho:  auditByWhoManagement,
		})
	})
	if txErr != nil {
		if repo.IsDuplicateEntry(txErr) || repo.IsLockRace(txErr) {
			return nil, true, errs.Trace(txErr)
		}
		return nil, false, errs.Trace(txErr)
	}
	return row, false, nil
}

// Update wb_update_version：改名（乐观锁；无变更 no-op 幂等，不写审计）。
// 状态迁移不走本端点——推进=wb_advance_version、废弃=wb_deprecate_version。
//
// 入参：version_id 必填；name 必填；updated_at=读取基线（0=以当前库值为基线）。
// 出参：更新后的版本行（新 updated_at 基线）。
//
// 错误集：ErrInvalidArgument（必填缺失）；ErrNotFound（版本不存在）；
// ErrOptimisticLock（基线不匹配，409）。
func (s *VersionService) Update(ctx context.Context, req *api.UpdateVersionReq) (*api.UpdateVersionResp, error) {
	if req.VersionId == "" {
		return nil, invalidArg("version_id 必填")
	}
	if req.Name == "" {
		return nil, invalidArg("name 必填")
	}
	var updated *repo.Version
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := getVersionTx(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		if row == nil {
			return notFound("版本 %q 不存在", req.VersionId)
		}
		if row.Name == req.Name {
			updated = row
			return nil
		}
		baseline := row.UpdatedAt
		if req.UpdatedAt != 0 {
			baseline = req.UpdatedAt
		}
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Version{}, row.ID, baseline, map[string]any{
			"name": req.Name,
		}); err != nil {
			return err
		}
		if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:    "versions",
			RowID:    row.VersionID,
			Action:   "UPDATE",
			Field:    strPtr("name"),
			OldValue: strPtr(row.Name),
			NewValue: strPtr(req.Name),
			ByWho:    auditByWhoManagement,
		}); err != nil {
			return err
		}
		fresh, err := getVersionTx(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		updated = fresh
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return &api.UpdateVersionResp{Ok: true, Version: versionItem(updated)}, nil
}

// Advance wb_advance_version：发布（软约束口径）——版本内 unfinished
// （status 非 已完成/已取消，domain.RequirementUnfinished）需求**仅提示不拦截**，
// 响应 ok + unfinished 清单；版本 state→已发布。已发布重复发布幂等无害
// （no-op 不写审计，仍返回 unfinished 清单）；已废弃=终态 blocked。
// 状态迁移同事务写 audit_log（UPDATE state 旧值→新值）。
//
// 入参：version_id 必填；updated_at=读取基线（0=以当前库值为基线，宽松模式）。
// 出参：更新后的版本行 + unfinished 需求清单（仅提示）。
//
// 错误集：ErrInvalidArgument（id 缺失）；ErrNotFound（版本不存在）；
// ErrGateBlocked（已废弃终态）；ErrOptimisticLock（基线不匹配，409）。
func (s *VersionService) Advance(ctx context.Context, req *api.AdvanceVersionReq) (*api.AdvanceVersionResp, error) {
	if req.VersionId == "" {
		return nil, invalidArg("version_id 必填")
	}
	var (
		updated    *repo.Version
		unfinished []*api.RequirementRef
	)
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := getVersionTx(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		if row == nil {
			return notFound("版本 %q 不存在", req.VersionId)
		}
		if row.State == domain.VerStateDeleted {
			return gateBlocked("版本 %q 已废弃（终态），不可发布", req.VersionId)
		}
		refs, err := s.unfinishedRefs(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		if row.State == domain.VerStatePublished {
			// 幂等：重复发布无害，仅回 unfinished 提示
			updated = row
			unfinished = refs
			return nil
		}
		baseline := row.UpdatedAt
		if req.UpdatedAt != 0 {
			baseline = req.UpdatedAt
		}
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Version{}, row.ID, baseline, map[string]any{
			"state": domain.VerStatePublished,
		}); err != nil {
			return err
		}
		if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:    "versions",
			RowID:    row.VersionID,
			Action:   "UPDATE",
			Field:    strPtr("state"),
			OldValue: strPtr(row.State),
			NewValue: strPtr(domain.VerStatePublished),
			ByWho:    auditByWhoManagement,
		}); err != nil {
			return err
		}
		fresh, err := getVersionTx(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		updated = fresh
		unfinished = refs
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return &api.AdvanceVersionResp{Ok: true, Version: versionItem(updated), Unfinished: unfinished}, nil
}

// Deprecate wb_deprecate_version ⚠️：废弃=终态。有引用（version_id 引用且
// 状态非 已取消/已完成 的需求存在）→ ErrGateBlocked（HTTP 侧 {ok:false,
// blocked:true}），不执行废弃；已废弃重复废弃幂等无害（no-op 不写审计）。
// 废弃动作同事务写 audit_log（UPDATE state 旧值→新值）。
//
// 入参：version_id 必填；updated_at=读取基线（0=以当前库值为基线，宽松模式）。
// 出参：更新后的版本行（state=已废弃）。
//
// 错误集：ErrInvalidArgument（id 缺失）；ErrNotFound（版本不存在）；
// ErrGateBlocked（存在未完成需求引用）；ErrOptimisticLock（基线不匹配，409）。
func (s *VersionService) Deprecate(ctx context.Context, req *api.DeprecateVersionReq) (*api.DeprecateVersionResp, error) {
	if req.VersionId == "" {
		return nil, invalidArg("version_id 必填")
	}
	var updated *repo.Version
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := getVersionTx(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		if row == nil {
			return notFound("版本 %q 不存在", req.VersionId)
		}
		if row.State == domain.VerStateDeleted {
			updated = row
			return nil
		}
		refs, err := s.unfinishedRefs(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		if len(refs) > 0 {
			return gateBlocked("版本 %q 存在 %d 条未完成需求引用，废弃被拦截（先取消/完成或移出版本）",
				req.VersionId, len(refs))
		}
		baseline := row.UpdatedAt
		if req.UpdatedAt != 0 {
			baseline = req.UpdatedAt
		}
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Version{}, row.ID, baseline, map[string]any{
			"state": domain.VerStateDeleted,
		}); err != nil {
			return err
		}
		if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:    "versions",
			RowID:    row.VersionID,
			Action:   "UPDATE",
			Field:    strPtr("state"),
			OldValue: strPtr(row.State),
			NewValue: strPtr(domain.VerStateDeleted),
			ByWho:    auditByWhoManagement,
		}); err != nil {
			return err
		}
		fresh, err := getVersionTx(ctx, tx, req.VersionId)
		if err != nil {
			return err
		}
		updated = fresh
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return &api.DeprecateVersionResp{Ok: true, Version: versionItem(updated)}, nil
}

// unfinishedRefs 版本内 unfinished 需求清单（发布软约束提示 / 废弃引用拦截共用；
// 口径 domain：status 非 已完成/已取消）。
func (s *VersionService) unfinishedRefs(ctx context.Context, tx *gormio.DB, versionID string) ([]*api.RequirementRef, error) {
	rows := make([]*repo.Requirement, 0)
	err := tx.WithContext(ctx).
		Where("version_id = ? AND status NOT IN ?", versionID,
			[]string{domain.ReqStatusDone, domain.ReqStatusCanceled}).
		Order("id").
		Find(&rows).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	refs := make([]*api.RequirementRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, &api.RequirementRef{
			RequirementId: row.RequirementID,
			Title:         row.Title,
			Status:        row.Status,
		})
	}
	return refs, nil
}

// getVersionTx 事务内按业务 id 读版本行（加锁读，防状态迁移竞态）。
func getVersionTx(ctx context.Context, tx *gormio.DB, versionID string) (*repo.Version, error) {
	row := &repo.Version{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("version_id = ?", versionID).
		First(row).Error
	if err != nil {
		if err == gormio.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// boolToInt64 TINYINT(1) 列取值转换。
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// strPtr 字符串指针（NULL 保留清单列 / 审计文本构造）。
func strPtr(s string) *string { return &s }
