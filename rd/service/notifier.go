// service 包本文件：通知写入口收敛点（T10）——扫描器催办/介入通知落 issues 表
// （issue_type=notification 行；spec 14 表无独立通知表，T11 收件箱读端复用本入口
// 扩展其余通知类型）。v3 七类通知规程出处：docs-tmp/ui-interaction-final.md §3.1/§6.8。
package service

import (
	"context"

	"gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// Notifier 通知写入口（issues notification 行独立 notifier，写路径收敛便于 T11
// 复用与审计口径统一）。
type Notifier struct {
	repo *repo.Repo
}

// NewNotifier 构造通知 service。
func NewNotifier(r *repo.Repo) *Notifier {
	return &Notifier{repo: r}
}

// NotificationInput 通知写入入参。
type NotificationInput struct {
	// RequirementID 关联需求（通知挂载；空串=无，DDL 允许）
	RequirementID string
	// NodeID 关联节点（通知挂载；空串=无）
	NodeID string
	// RoleID 发表者角色（扫描器口径=system；v3 其余类型由触发写路径自定）
	RoleID string
	// Assignee 报送责任人：悬置催办=执行角色 / 需你介入=user（v3 §3.1 列）
	Assignee string
	// Text 通知文本（扫描器 kind 行以 domain.ScannerNotificationText 前缀编码）
	Text string
	// ByWho 审计操作者（扫描器=system）
	ByWho string
}

// Notify 写入一条通知行（issues：issue_type=notification、state=open、anchor=document、
// escalate=0——escalate 列保留给冲突上升语义 T12）。
// 写路径单事务：issue_id 事务内取号（cm-<ts> 家族）+ INSERT + 审计（by_who=in.ByWho）。
//
// 入参：Text 必填；ByWho 空串默认 system。
// 出参：落库后通知行。
//
// 错误集：ErrInvalidArgument（Text 缺失）；底层 IO/取号错误经 Trace 上抛。
func (n *Notifier) Notify(ctx context.Context, in NotificationInput) (*repo.Issue, error) {
	if in.Text == "" {
		return nil, invalidArg("text 必填")
	}
	byWho := in.ByWho
	if byWho == "" {
		byWho = "system"
	}
	row := &repo.Issue{
		RequirementID: in.RequirementID,
		NodeID:        in.NodeID,
		AnchorKind:    domain.IssueAnchorDocument,
		Author:        byWho,
		RoleID:        in.RoleID,
		IssueType:     domain.IssueTypeNotification,
		Assignee:      strPtrIfSet(in.Assignee),
		Text:          in.Text,
		State:         repo.IssueStateOpen,
		Replies:       "[]",
	}
	txErr := n.repo.Gorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		issueID, err := n.repo.AllocIssueIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.IssueID = issueID
		if err := tx.Create(row).Error; err != nil {
			return errs.Trace(err)
		}
		return n.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "issues",
			RowID:  issueID,
			Action: "INSERT",
			ByWho:  byWho,
		})
	})
	if txErr != nil {
		return nil, txErr
	}
	return row, nil
}
