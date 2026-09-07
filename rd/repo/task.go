// repo 包本文件：任务总线（tasks 表）业务查询与 t-<ts>-<seq> 业务 id 事务内取号
// （spec §0 表 #7 + T5 工单）。写路径消费方为 service.TaskService；
// 判据校验的事实读取（artifacts/review_results/issues）复用 requirement.go 既有查询。
package repo

import (
	"context"
	"fmt"

	gormio "gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// taskIDPrefix 任务业务 id 前缀（示例 t-1730000000000-001；seq 全局递增三位零填充，
// 超三位自然进位；ts=取号时刻 unix 毫秒）。
const taskIDPrefix = "t-"

// 任务状态族（tasks.status 六值 + 终态集）。
const (
	TaskStatusTodo          = "todo"
	TaskStatusInProgress    = "in-progress"
	TaskStatusAwaitingReply = "awaiting-reply"
	TaskStatusCompleted     = "completed"
	TaskStatusRejected      = "rejected"
	TaskStatusClosed        = "closed"
)

// TerminalTaskStatuses 任务终态集：重复派发对同键不再幂等命中（可再派），
// 终态任务再上报一律幂等 no-op（spec §2 异常处理表）。
var TerminalTaskStatuses = []string{TaskStatusCompleted, TaskStatusRejected, TaskStatusClosed}

// GetTaskByTaskID 按业务 id 读任务行；未命中返回 (nil, nil)（由 service 层转 ErrNotFound）。
func (r *Repo) GetTaskByTaskID(ctx context.Context, taskID string) (*Task, error) {
	row := &Task{}
	err := r.g.WithContext(ctx).
		Where("task_id = ?", taskID).
		First(row).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// GetTaskByIDTx 事务内按业务 id 加锁读任务行（SELECT … FOR UPDATE，防并发上报
// 读脏/双写）；未命中返回 (nil, nil)。
func GetTaskByIDTx(ctx context.Context, tx *gormio.DB, taskID string) (*Task, error) {
	row := &Task{}
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("task_id = ?", taskID).
		First(row).Error
	if err != nil {
		if err == gormio.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// FindActiveTaskByKeyTx 事务内按幂等键查未终态任务（派发幂等命中数据源）：
// requirement_id + node_id + action_id + role_id 四键全等且 status 不在终态集；
// 未命中返回 (nil, nil)。调用前须先持有同节点 flow_nodes 行锁，防并发双建幻影。
func FindActiveTaskByKeyTx(ctx context.Context, tx *gormio.DB, requirementID string, nodeID string, actionID string, roleID string) (*Task, error) {
	row := &Task{}
	err := tx.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ? AND action_id = ? AND role_id = ?", requirementID, nodeID, actionID, roleID).
		Where("status NOT IN ?", TerminalTaskStatuses).
		Order("id").
		First(row).Error
	if err != nil {
		if err == gormio.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errs.Trace(err)
	}
	return row, nil
}

// ListInflightTasksByRequirementAndNode 读该需求该节点的在途任务（status 不在终态
// 集，id 升序；wb_my_task relatedTasks 数据源，T6/T7 亦消费）。
func (r *Repo) ListInflightTasksByRequirementAndNode(ctx context.Context, requirementID string, nodeID string) ([]*Task, error) {
	items := make([]*Task, 0)
	err := r.g.WithContext(ctx).
		Where("requirement_id = ? AND node_id = ?", requirementID, nodeID).
		Where("status NOT IN ?", TerminalTaskStatuses).
		Order("id").
		Find(&items).Error
	if err != nil {
		return nil, errs.Trace(err)
	}
	return items, nil
}

// AllocTaskIDTx 事务内取下一个任务业务 id（t-<ts>-<seq>，seq 全局递增）。
//
// 取号 SQL：MAX(SUBSTRING_INDEX(task_id, '-', -1) 数字后缀) FOR UPDATE，锁
// uk_tasks_tid 索引扫描范围序列化取号（与 req-NNN 同口径）；并发同号由唯一键
// 兜底（1062）或锁竞争（1213/1205），调用方（service 层）整体重试事务。
func (r *Repo) AllocTaskIDTx(ctx context.Context, tx *gormio.DB) (string, error) {
	stmt := "SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(task_id, '-', -1) AS UNSIGNED)), 0) FROM tasks WHERE task_id LIKE ? FOR UPDATE"
	var maxSeq int64
	if err := tx.WithContext(ctx).Raw(stmt, taskIDPrefix+"%").Scan(&maxSeq).Error; err != nil {
		return "", errs.Trace(err)
	}
	if maxSeq < 0 {
		return "", errs.Trace(fmt.Errorf("tasks.task_id 序号非法（负数）"))
	}
	return FormatTaskID(NowMillis(), maxSeq+1), nil
}

// FormatTaskID 时间戳与序号转任务业务 id（t-<ts>-<seq>，seq 三位零填充）。
func FormatTaskID(ts int64, seq int64) string {
	return fmt.Sprintf("%s%d-%03d", taskIDPrefix, ts, seq)
}
