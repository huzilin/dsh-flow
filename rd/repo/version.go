// repo 包本文件：versions 表业务查询与 v-NNN 业务 id 事务内取号（spec §0.3）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// version_id 前缀（示例 v-1，无零填充，纯递增）。
const versionIDPrefix = "v-"

// ListVersions 全量版本（含待分配池标记行），排序 id ASC（创建序）。
func (r *Repo) ListVersions(ctx context.Context) ([]*Version, error) {
	items := make([]*Version, 0)
	err := r.g.WithContext(ctx).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// GetVersionByID 按业务 id 读版本行；未命中返回 (nil, nil)（由 service 层转 ErrNotFound）。
func (r *Repo) GetVersionByID(ctx context.Context, versionID string) (*Version, error) {
	row := &Version{}
	err := r.g.WithContext(ctx).
		Where("version_id = ?", versionID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListRequirementsByVersionStatus 读版本内需求，排除 excludeStatuses 列出的状态
// （口径由调用方给：版本发布软约束 / 废弃引用判定 = 非 已完成/已取消）。
func (r *Repo) ListRequirementsByVersionStatus(ctx context.Context, versionID string, excludeStatuses []string) ([]*Requirement, error) {
	items := make([]*Requirement, 0)
	q := r.g.WithContext(ctx).Where("version_id = ?", versionID)
	if len(excludeStatuses) > 0 {
		q = q.Where("status NOT IN ?", excludeStatuses)
	}
	err := q.Order("id").Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// AllocVersionIDTx 事务内取下一个版本业务 id（v-NNN 递增，spec §0.3 示例 v-1）。
// 防重口径同 AllocRequirementIDTx（uk_versions_vid + 锁竞争由调用方整体重试）。
func (r *Repo) AllocVersionIDTx(ctx context.Context, tx *gorm.DB) (string, error) {
	seq, err := maxIDNumberSuffixTx(ctx, tx, "versions", "version_id", versionIDPrefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", versionIDPrefix, seq+1), nil
}
