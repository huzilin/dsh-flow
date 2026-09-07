// repo 包本文件：基线 helper——应用层时间戳、updated_at 乐观锁更新、append-only 审计写入。
package repo

import (
	"context"
	"time"

	"gorm.io/gorm"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// Repo GORM MySQL 访问句柄（模型与基线 helper 的收拢点；
// 各表业务查询由后续工单按需在 repo 包内添加）。
type Repo struct {
	g *gorm.DB
}

// NewRepo 构造 Repo（g 为 database.Open 产出的业务读写句柄）。
func NewRepo(g *gorm.DB) *Repo {
	return &Repo{g: g}
}

// Gorm 暴露底层句柄（写路径显式事务 + 行锁时使用）。
func (r *Repo) Gorm() *gorm.DB {
	return r.g
}

// NowMillis 应用层时间戳统一入口：unix 毫秒（spec §0——双时间戳由应用层维护）。
func NowMillis() int64 {
	return time.Now().UnixMilli()
}

// UpdateWithOptimisticLock updated_at 乐观锁更新（spec §0.2）。
//
// UPDATE <model 表> SET <updates> WHERE id = ? AND updated_at = <读取时值>；
// 0 行命中（值已被并发修改或行不存在）返回 errs.ErrOptimisticLock，由上层重试或报错。
//
// 入参约定：
//   - model：任意 repo model 值（表名与主键由 GORM 解析），如 &Task{}；
//   - updates：键=列名（snake_case），值=新值；updated_at 由本 helper 自动写入新时间戳，
//     调用方不得自带 updated_at 键。
func (r *Repo) UpdateWithOptimisticLock(ctx context.Context, model any, id int64, readUpdatedAt int64, updates map[string]any) error {
	return UpdateWithOptimisticLockTx(ctx, r.g, model, id, readUpdatedAt, updates)
}

// UpdateWithOptimisticLockTx 事务内乐观锁更新：语义同 UpdateWithOptimisticLock，
// 在指定句柄（业务事务）内执行，保证「更新 + 审计」同事务原子。
func UpdateWithOptimisticLockTx(ctx context.Context, tx *gorm.DB, model any, id int64, readUpdatedAt int64, updates map[string]any) error {
	if _, ok := updates["updated_at"]; ok {
		return errs.Trace(perr.Errorf("updates 不得携带 updated_at（由 helper 维护乐观锁时间戳）"))
	}
	updates["updated_at"] = NowMillis()
	res := tx.WithContext(ctx).Model(model).
		Where("id = ? AND updated_at = ?", id, readUpdatedAt).
		Updates(updates)
	if res.Error != nil {
		return errs.Trace(res.Error)
	}
	if res.RowsAffected == 0 {
		return errs.Trace(errs.ErrOptimisticLock)
	}
	return nil
}

// InsertAuditLog append-only 审计写入 helper（spec §0 表 #11）。
//
// created_at=updated_at 恒同值由本 helper 强制覆盖（调用方传入值不生效），
// 保证审计行不可变语义；审计与业务同库同事务的用法：
//
//	tx := repo.Gorm().WithContext(ctx)  // 业务事务内
//	repo.InsertAuditLogWith(tx, entry)
//
// 业务事务路径用 InsertAuditLogWith（传入事务句柄）；独立写入用本方法。
func (r *Repo) InsertAuditLog(ctx context.Context, entry *AuditLog) error {
	return r.InsertAuditLogWith(r.g.WithContext(ctx), entry)
}

// InsertAuditLogWith 在指定句柄（含业务事务）内写入审计行，created_at=updated_at 恒同值。
func (r *Repo) InsertAuditLogWith(tx *gorm.DB, entry *AuditLog) error {
	now := NowMillis()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	if err := tx.Create(entry).Error; err != nil {
		return errs.Trace(err)
	}
	return nil
}
