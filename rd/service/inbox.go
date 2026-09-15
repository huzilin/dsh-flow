// service 包本文件：wb_list_inbox 收件箱 + TODO 投影读端点（spec §1.2 路由表
// GET /api/inbox 行，通知枚举 v3 七类）。DTO 以 protoc 生成的 api 包类型承载
// （proto 为契约真相源，禁手写漂移）。
//
// 双通道同源口径（ui-interaction-final §3/§4）：
//   - 通知 = issues 行 issue_type=notification 投影（含跳转定位）；
//   - TODO = tasks 表在途任务同源投影（无独立投影表、不双写——任务状态迁移后
//     投影即时同步，completed/rejected/closed 不残留）。
package service

import (
	"context"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// 默认分页口径（ListInboxReq.page_size=0 时）。
const inboxDefaultPageSize = int64(20)

// InboxService 收件箱 service（wb_list_inbox）。
type InboxService struct {
	repo *repo.Repo
}

// NewInboxService 构造收件箱 service。
func NewInboxService(r *repo.Repo) *InboxService {
	return &InboxService{repo: r}
}

// List wb_list_inbox：七类通知（收件角色过滤 + 未读优先 + 分页）+ TODO 投影
// （同响应双通道）。
//
// 入参：role 收件角色过滤（通知按 issues.assignee、TODO 按 tasks.role_id；空=
// 不过滤全量）；read_filter ∈ {""（全部）, unread（未读）, read（已读）}，越集 400；
// kind 七类枚举单值筛选（空=全部，越集 400）；requirement_id 可选按需求过滤；
// page/page_size 分页（0 归一化为 1/默认 20）。
// 出参：notifications（未读优先、同组内新→旧，含跳转定位 requirement_id/node_id/
// target_type/target_id）、total（筛选后总数）、todos（该角色全部在途任务投影，
// 不参与通知分页）。
//
// 错误集：ErrInvalidArgument（read_filter/kind 越集）；底层 IO 错误经 Trace 上抛。
func (s *InboxService) List(ctx context.Context, req *api.ListInboxReq) (*api.ListInboxResp, error) {
	switch req.ReadFilter {
	case "", repo.ReadFilterUnread, repo.ReadFilterRead:
	default:
		return nil, invalidArg("read_filter 须为空|unread|read，实测 %q", req.ReadFilter)
	}
	if req.Kind != "" && !validNotificationKind(req.Kind) {
		return nil, invalidArg("kind %q 越集（通知七类枚举）", req.Kind)
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = inboxDefaultPageSize
	}

	rows, total, err := s.repo.ListNotifications(ctx, repo.NotificationFilter{
		RecipientRole: req.Role,
		ReadFilter:    req.ReadFilter,
		Kind:          req.Kind,
		RequirementID: req.RequirementId,
		Page:          page,
		PageSize:      pageSize,
	})
	if err != nil {
		return nil, errs.Trace(err)
	}
	items := make([]*api.InboxItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, inboxItem(row))
	}

	// TODO 投影：tasks 表在途任务同源投影（终态不残留；不双写）
	taskRows, err := s.repo.ListInflightTasksByRole(ctx, req.Role)
	if err != nil {
		return nil, errs.Trace(err)
	}
	todos := make([]*api.TodoItem, 0, len(taskRows))
	for _, row := range taskRows {
		todos = append(todos, todoItem(row))
	}

	return &api.ListInboxResp{
		Ok:            true,
		Notifications: items,
		Total:         total,
		Todos:         todos,
	}, nil
}

// inboxItem repo 通知行 → proto 出参（跳转定位口径见 notifier.go 七类映射表：
// target_type 由 kind 确定性映射；target_type=node 时 target_id 空串、定位取 node_id）。
func inboxItem(row *repo.Issue) *api.InboxItem {
	return &api.InboxItem{
		IssueId:       row.IssueID,
		Kind:          row.AnchorKind,
		RequirementId: row.RequirementID,
		NodeId:        row.NodeID,
		TargetType:    notificationTargetTypes[row.AnchorKind],
		TargetId:      utils.DerefStr(row.AnchorArtifactID),
		Text:          row.Text,
		Sender:        row.Author,
		SenderRole:    row.RoleID,
		RecipientRole: utils.DerefStr(row.Assignee),
		State:         row.State,
		CreatedAt:     row.CreatedAt,
	}
}

// todoItem repo 任务行 → proto TODO 投影出参（在途任务最小集）。
func todoItem(row *repo.Task) *api.TodoItem {
	return &api.TodoItem{
		TaskId:        row.TaskID,
		RequirementId: row.RequirementID,
		NodeId:        row.NodeID,
		ActionId:      row.ActionID,
		TaskType:      row.TaskType,
		Status:        row.Status,
		Goal:          row.Goal,
		CreatedAt:     row.CreatedAt,
	}
}
