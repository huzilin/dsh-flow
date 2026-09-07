// repo 包本文件：requirements 表业务查询与 req-NNN 业务 id 事务内取号（spec §0.3）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// requirement_id 前缀与序号位宽（示例 req-001，三位零填充，超三位自然进位）。
const requirementIDPrefix = "req-"

// ListRequirementFilter 需求列表筛选条件（零值字段=不过滤）。
type ListRequirementFilter struct {
	// VersionID 精确版本筛选；nil=不过滤
	VersionID *string
	// Backlog true=仅待分配池（version_id IS NULL）；与 VersionID 同给时 Backlog 优先
	Backlog bool
	// Priority P0-P3；空=不过滤
	Priority string
	// Status 需求状态；空=不过滤
	Status string
	// Page 页码，1 起（调用方归一化）
	Page int64
	// PageSize 页大小（调用方归一化）
	PageSize int64
}

// ListRequirements 分页查询需求（强制分页：LIMIT/OFFSET 由 filter 承载）+ 筛选后总数。
// 排序 id DESC（新需求在前）。
func (r *Repo) ListRequirements(ctx context.Context, filter ListRequirementFilter) ([]*Requirement, int64, error) {
	// count 与 find 各自独立构建查询（防 gorm 条件跨执行复用污染）
	conds := func(q *gorm.DB) *gorm.DB {
		switch {
		case filter.Backlog:
			q = q.Where("version_id IS NULL")
		case filter.VersionID != nil:
			q = q.Where("version_id = ?", *filter.VersionID)
		}
		if filter.Priority != "" {
			q = q.Where("priority = ?", filter.Priority)
		}
		if filter.Status != "" {
			q = q.Where("status = ?", filter.Status)
		}
		return q
	}

	total := int64(0)
	if err := conds(r.g.WithContext(ctx).Model(&Requirement{})).Count(&total).Error; err != nil {
		return nil, 0, errs.Trace(err)
	}
	items := make([]*Requirement, 0, filter.PageSize)
	offset := (filter.Page - 1) * filter.PageSize
	err := conds(r.g.WithContext(ctx).Model(&Requirement{})).
		Order("id DESC").
		Limit(int(filter.PageSize)).
		Offset(int(offset)).
		Find(&items).Error
	if err != nil {
		return nil, 0, errs.Trace(err)
	}
	return items, total, nil
}

// GetRequirementByID 按业务 id 读需求行；未命中返回 (nil, nil)（由 service 层转 ErrNotFound）。
func (r *Repo) GetRequirementByID(ctx context.Context, requirementID string) (*Requirement, error) {
	row := &Requirement{}
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListFlowsByRequirement 读需求全部流程实例（T4 前为空集）。
func (r *Repo) ListFlowsByRequirement(ctx context.Context, requirementID string) ([]*Flow, error) {
	items := make([]*Flow, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// ListFlowNodesByRequirement 读需求全部实例节点（渲染序）；T4 前为空集。
func (r *Repo) ListFlowNodesByRequirement(ctx context.Context, requirementID string) ([]*FlowNode, error) {
	items := make([]*FlowNode, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		Order("node_order").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// ListArtifactsByRequirement 读需求全部产物登记；T6 前为空集。
func (r *Repo) ListArtifactsByRequirement(ctx context.Context, requirementID string) ([]*Artifact, error) {
	items := make([]*Artifact, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// ListIssuesByRequirement 读需求全部评论；T7 前为空集。
// 通知行（issue_type=notification，T11 收件箱载体）不属于评论流——聚合快照/
// 澄清判据等通用消费方一律排除，收件箱经 ListNotifications 专读。
func (r *Repo) ListIssuesByRequirement(ctx context.Context, requirementID string) ([]*Issue, error) {
	items := make([]*Issue, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ?", requirementID).
		Where("issue_type <> ?", IssueTypeNotification).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// DeleteRequirementByIDTx 事务内按主键删除需求行（先 GetRequirementByID 命中后调用）。
func (r *Repo) DeleteRequirementByIDTx(tx *gorm.DB, id int64) error {
	res := tx.Delete(&Requirement{}, id)
	if res.Error != nil {
		return errs.Trace(res.Error)
	}
	if res.RowsAffected == 0 {
		return errs.Trace(errs.ErrOptimisticLock)
	}
	return nil
}

// AllocRequirementIDTx 事务内取下一个需求业务 id（req-NNN 递增，spec §0.3 示例 req-001）。
//
// 取号 SQL：MAX(数字后缀) FOR UPDATE 锁索引范围（事务内取号防重）；并发场景下
// 双方同取号时由 uk_requirements_rid 兜底（1062）或锁竞争（1213/1205），
// 调用方（service 层）整体重试事务。
func (r *Repo) AllocRequirementIDTx(ctx context.Context, tx *gorm.DB) (string, error) {
	seq, err := maxIDNumberSuffixTx(ctx, tx, "requirements", "requirement_id", requirementIDPrefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%03d", requirementIDPrefix, seq+1), nil
}

// maxIDNumberSuffixTx 事务内计算业务 id 数字后缀最大值：
// SELECT COALESCE(MAX(CAST(SUBSTRING(col, <prefixLen+1>) AS UNSIGNED)), 0)
//
//	FROM <table> WHERE <col> LIKE '<prefix>%' FOR UPDATE
//
// FOR UPDATE 锁住扫描到的索引范围，序列化同前缀取号（幻影由唯一键兜底）。
func maxIDNumberSuffixTx(ctx context.Context, tx *gorm.DB, table string, col string, prefix string) (int64, error) {
	substrStart := len(prefix) + 1 // SUBSTRING 1 起
	stmt := fmt.Sprintf(
		"SELECT COALESCE(MAX(CAST(SUBSTRING(%s, ?) AS UNSIGNED)), 0) FROM %s WHERE %s LIKE ? FOR UPDATE",
		col, table, col,
	)
	var maxSeq int64
	err := tx.WithContext(ctx).Raw(stmt, substrStart, prefix+"%").Scan(&maxSeq).Error
	if err != nil {
		return 0, errs.Trace(err)
	}
	if maxSeq < 0 {
		return 0, errs.Trace(perr.Errorf("%s.%s 序号非法（负数）", table, col))
	}
	return maxSeq, nil
}

// isRecordNotFound 判定 gorm ErrRecordNotFound。
func isRecordNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
