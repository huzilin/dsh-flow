// repo 包本文件：通知载体（issues 表 issue_type=notification 行）业务查询与写入
// （T11 wb_list_inbox 收件箱 + notifier 收敛写入口的数据源）。
//
// 列复用口径（工单 T11 定稿，issues 行 issue_type=notification 分区内）：
//   - anchor_kind = 通知类别（七类枚举，见 service.NotificationKind*）；
//   - anchor_artifact_id = 跳转目标业务 id（target_type=node 时空 NULL，定位取 node_id 列）；
//   - assignee = 收件角色（idx_issues_assignee 索引支撑收件箱角色过滤）；
//   - state = 读标记：open=未读 / resolved=已读（本工单只做读过滤读标记，
//     「标记已读」端点不在 spec §1.2 v1.0 路由表，为 v1.1/T16 预留）；
//   - author/role_id = 派发者/派发者角色（plugin 投递固定 plugin）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// 通知读标记状态（issues.state 列在 notification 分区内的两值口径）。
const (
	// NotificationStateUnread 未读（建行初值）
	NotificationStateUnread = "open"
	// NotificationStateRead 已读
	NotificationStateRead = "resolved"
)

// IssueTypeNotification 通知行类型（issues.issue_type 列值；与 domain.IssueType*
// 同源镜像——repo 不反向依赖 domain，spec §0 列注释为单一口径）。
const IssueTypeNotification = "notification"

// ReadFilter 值集（ListInboxReq.read_filter 越集由 service 拦截）。
const (
	ReadFilterUnread = "unread"
	ReadFilterRead   = "read"
)

// NotificationFilter 收件箱通知筛选（wb_list_inbox 数据源入参；分页强制）。
type NotificationFilter struct {
	// RecipientRole 收件角色（issues.assignee）；空=不过滤
	RecipientRole string
	// ReadFilter 已读过滤：""=全部 / unread / read
	ReadFilter string
	// Kind 通知类别（七类枚举单值）；空=全部
	Kind string
	// RequirementID 按需求过滤；空=跨需求
	RequirementID string
	// Page 页码（1 起，调用方归一化）
	Page int64
	// PageSize 页大小（调用方归一化）
	PageSize int64
}

// InsertNotification 通知行写入（notifier 收敛写入口的落库通道）：单事务
// INSERT issues（issue_type=notification）+ INSERT 审计（by_who=plugin——
// capability-model-final §6.8「收件箱通知=系统层 plugin 投递」口径）。
// 调用方（service.Notifier）保证事务外派发：失败返回错误、不牵连业务事务。
func (r *Repo) InsertNotification(ctx context.Context, row *Issue) error {
	txErr := r.g.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(row).Error; err != nil {
			return errs.Trace(err)
		}
		return r.InsertAuditLogWith(tx, &AuditLog{
			Table:  "issues",
			RowID:  row.IssueID,
			Action: "INSERT",
			ByWho:  "plugin",
		})
	})
	if txErr != nil {
		return errs.Trace(txErr)
	}
	return nil
}

// AllocIssueID 通知行取号（非事务版 AllocIssueIDTx：事务外派发通道用）。
// autocommit 下 FOR UPDATE 锁即取即放，同毫秒并发极端场景可能撞唯一键——
// 通知为非关键路径，冲突由 Dispatch 吞错记日志兜底（业务不受影响）。
func (r *Repo) AllocIssueID(ctx context.Context) (string, error) {
	return r.AllocIssueIDTx(ctx, r.g)
}

// ListNotifications 收件箱通知分页查询（未读优先：state=open 组置前，组内 id DESC
// 新→旧）+ 筛选后总数。count 与 find 各自独立构建查询（防 gorm 条件跨执行复用污染，
// 与 ListRequirements 同口径）。
func (r *Repo) ListNotifications(ctx context.Context, filter NotificationFilter) ([]*Issue, int64, error) {
	conds := func(q *gorm.DB) *gorm.DB {
		q = q.Where("issue_type = ?", IssueTypeNotification)
		if filter.RecipientRole != "" {
			q = q.Where("assignee = ?", filter.RecipientRole)
		}
		switch filter.ReadFilter {
		case ReadFilterUnread:
			q = q.Where("state = ?", NotificationStateUnread)
		case ReadFilterRead:
			q = q.Where("state <> ?", NotificationStateUnread)
		}
		if filter.Kind != "" {
			// 类别落 anchor_kind 复用列（本文件头注口径）
			q = q.Where("anchor_kind = ?", filter.Kind)
		}
		if filter.RequirementID != "" {
			q = q.Where("requirement_id = ?", filter.RequirementID)
		}
		return q
	}

	total := int64(0)
	if err := conds(r.g.WithContext(ctx).Model(&Issue{})).Count(&total).Error; err != nil {
		return nil, 0, errs.Trace(err)
	}
	items := make([]*Issue, 0, filter.PageSize)
	offset := (filter.Page - 1) * filter.PageSize
	err := conds(r.g.WithContext(ctx).Model(&Issue{})).
		// 未闭合（未读）优先 + 同组内新→旧（MySQL 布尔表达式排序；状态值取自
		// 本包常量，无外部输入，Raw 排序表达式无注入面）
		Order(fmt.Sprintf("state = '%s' DESC", NotificationStateUnread)).
		Order("id DESC").
		Limit(int(filter.PageSize)).
		Offset(int(offset)).
		Find(&items).Error
	if err != nil {
		return nil, 0, errs.Trace(err)
	}
	return items, total, nil
}

// ListInflightTasksByRole TODO 投影数据源（wb_list_inbox todos 字段）：该角色全部
// 在途任务（status NOT IN 终态集，终态 completed/rejected/closed 不残留），
// id 升序。roleID 空=全角色（User 全量视图口径）。
func (r *Repo) ListInflightTasksByRole(ctx context.Context, roleID string) ([]*Task, error) {
	items := make([]*Task, 0)
	q := r.g.WithContext(ctx).Model(&Task{}).Where("status NOT IN ?", TerminalTaskStatuses)
	if roleID != "" {
		q = q.Where("role_id = ?", roleID)
	}
	err := q.Order("id").Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}
