// repo 包本文件：issues 表业务查询、cm-<ts> 业务 id 事务内取号、
// 行锁读取（评论写路径「事务 + 行锁 + 乐观锁更新」基线，T7）。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// issueIDPrefix 评论业务 id 前缀（spec §0：cm-<ts>；取号=毫秒时间戳与序号基值取大，
// 并发同毫秒由 FOR UPDATE 基值递增兜底，形态仍为纯数字后缀时间戳）。
const issueIDPrefix = "cm-"

// 未闭合状态集（my_task relatedIssues 口径：state 非 resolved）。
// SQL NOT IN 直接内联常量切片。
// 修复（T8）：原实现误传 open 族集合致 NOT IN 语义反转——查回的恰是已闭合行，
// 与本注释口径及 issues_it_test「resolved 不注入」断言冲突（T7 遗留缺陷，
// 阻碍 T8 澄清循环「owner 收 clarify 上下文」AC，最小修复对齐注释口径）。
var issueOpenStates = []string{IssueStateResolved}

// issue 状态常量镜像（domain 为纯逻辑包，repo 不反向依赖，取值与
// domain.IssueState* 同源——spec §0 列注释为单一口径；overridden 态已按
// 09-12 裁决废除，override 动作置 resolved，semantica 920030de）。
const (
	IssueStateOpen           = "open"
	IssueStatePendingConfirm = "pending-confirm"
	IssueStateToFix          = "to-fix"
	IssueStateFixed          = "fixed"
	IssueStateResolved       = "resolved"
)

// GetIssueByIssueID 按评论业务 id 读评论行；未命中返回 (nil, nil)（service 转 ErrNotFound）。
func (r *Repo) GetIssueByIssueID(ctx context.Context, issueID string) (*Issue, error) {
	row := &Issue{}
	err := r.g.WithContext(ctx).
		Where("issue_id = ?", issueID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetIssueByIssueIDLockTx 事务内行锁读取评论行（SELECT … FOR UPDATE：
// 并发 reply 追加不丢行——读改写原子基线）；未命中返回 (nil, nil)。
func (r *Repo) GetIssueByIssueIDLockTx(ctx context.Context, tx *gorm.DB, issueID string) (*Issue, error) {
	row := &Issue{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("issue_id = ?", issueID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListOpenIssuesByNode 读需求下指定节点的未闭合评论（my_task relatedIssues 数据源，
// state NOT IN (resolved)）；nodeID 为空返回空集。
// 通知行（issue_type=notification，T11 收件箱载体）不属于评论流，排除。
func (r *Repo) ListOpenIssuesByNode(ctx context.Context, requirementID string, nodeID string) ([]*Issue, error) {
	if nodeID == "" {
		return make([]*Issue, 0), nil
	}
	items := make([]*Issue, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ? AND state NOT IN ?", requirementID, nodeID, issueOpenStates).
		Where("issue_type <> ?", IssueTypeNotification).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// AllocIssueIDTx 事务内取下一个评论业务 id（cm-<ts>）：
// MAX(数字后缀) FOR UPDATE 与当前毫秒时间戳取大 + 1（时间戳形态优先；
// 同毫秒并发/回拨场景由序号基值单调递增兜底），uk_issues_iid 唯一键最终兜底。
func (r *Repo) AllocIssueIDTx(ctx context.Context, tx *gorm.DB) (string, error) {
	base, err := maxIDNumberSuffixTx(ctx, tx, "issues", "issue_id", issueIDPrefix)
	if err != nil {
		return "", err
	}
	next := NowMillis()
	if base >= next {
		next = base + 1
	}
	return fmt.Sprintf("%s%d", issueIDPrefix, next), nil
}
