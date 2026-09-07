// repo 包本文件：扫描器读侧查询（T10）——在途任务清单与催办/介入通知查重。
// 通知行载体口径：issues 表 issue_type=notification（spec 14 表无独立通知表）；
// 查重前缀由调用方（service 层）以 domain.ScannerNotificationTextPrefix 装配传入
// （repo 不反向依赖 domain，惯例同 issue.go 状态常量镜像注）。
package repo

import (
	"context"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// issueTypeNotification issues.issue_type 通知值镜像（与 domain.IssueTypeNotification
// 同源——spec §0 列注释为单一口径；repo 不反向依赖 domain）。
const issueTypeNotification = "notification"

// ListInflightTasks 读全部在途任务（status 不在终态集，id 升序）——扫描器候选集
// 数据源。任务量级 MVP 全量装载后由 domain.ScanTask 逐行判定（阈值判定在应用层，
// spec §3 性能考量口径）；终态任务不装载。
func (r *Repo) ListInflightTasks(ctx context.Context) ([]*Task, error) {
	items := make([]*Task, 0)
	err := r.g.WithContext(ctx).
		Where("status NOT IN ?", TerminalTaskStatuses).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// FindOpenScannerNotification 查该需求该节点下、指定任务与 kind 的未闭合通知行
// （issue_type=notification 且 state 不在闭合集；textPrefix 前缀匹配 task_id+kind）。
// 命中即「同任务同轮已有同类催办」——扫描器跳过不重复建；未命中返回 (nil, nil)。
//
// text LIKE 前缀匹配不走向索引，扫描集为「该节点未闭合 notification 行」，MVP
// 量级可接受（索引优化留待量级证据出现后再做，不在本工单扩 DDL）。
func (r *Repo) FindOpenScannerNotification(ctx context.Context, requirementID string, nodeID string, textPrefix string) (*Issue, error) {
	row := &Issue{}
	err := r.g.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ?", requirementID, nodeID).
		Where("issue_type = ? AND state NOT IN ?", issueTypeNotification, issueOpenStates).
		Where("text LIKE ?", textPrefix+"%").
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
