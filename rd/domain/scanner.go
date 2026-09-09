// Package domain 本文件：扫描器纯逻辑（T10）——在途任务催办/介入判定（spec §2
// 异常处理表「agent 无响应」行 + §4.1 P2 AC「5min 催办 / 10min 介入按时触发」）。
// 无 IO：时间以参数注入（now int64），判定事实（已有同类未闭合通知 / 绑定 session
// 是否 active）由 service 层查询后传入，可全分支表驱动单测。
//
// 本期降级口径（无真实 DSH subagent，嵌入适配器 out of scope）：「绑定 subagent
// agent/status=running → 等待」以「该任务键（requirement+node+role）存在 active
// 会话绑定」为 running 代理判据；「已结束 5min 未表态」降级为任务级时限——未 ack
// 以创建时刻为基线、已 ack 以最近上报（updated_at）为基线，闲置时长越限即触发。
package domain

import (
	"fmt"

	"github.com/huzilin/wbflow/rd/repo"
)

// 扫描器通知 kind（v3 通知规程「悬置催办 / 需你介入」两类，ui-interaction-final
// §3.1）。载体口径：spec 14 表无独立通知表，issues 表 issue_type=notification 行
// 承载；kind 以 text 稳定前缀编码（ScannerNotificationText 单一口径），T11 收件箱
// 读端按前缀分类。task_id 形态 t-<数字>-<数字>，不含 LIKE 通配符，前缀匹配安全。
const (
	NotifKindFollowup  = "followup"  // 悬置催办：催执行角色（遇卡点上报 user）
	NotifKindIntervene = "intervene" // 需你介入：升级通知 User
)

// ScanVerdict 扫描判定输出（Action 取 NotifKind* 或 ""=不动）。
type ScanVerdict struct {
	// Action 待执行动作：NotifKindFollowup / NotifKindIntervene / ""=跳过
	Action string
	// IdleMillis 判定基线的闲置时长（毫秒）
	IdleMillis int64
	// BaselineMillis 判定基线：未 ack=创建时刻；已 ack=最近上报时刻
	BaselineMillis int64
}

// ScanInput 单任务扫描判定输入。
type ScanInput struct {
	// Now 当前时刻（注入时钟，unix 毫秒）
	Now int64
	// FollowupAfterMillis 催办阈值（默认 5min，参数化配置）
	FollowupAfterMillis int64
	// InterveneAfterMillis 介入升级阈值（默认 10min；须 ≥ 催办阈值，否则催办
	// 恒被介入取代——config 默认值保证）
	InterveneAfterMillis int64
	// OpenFollowup 该任务已有未闭合催办通知（同轮不重复催）
	OpenFollowup bool
	// OpenIntervene 该任务已有未闭合介入通知
	OpenIntervene bool
	// SessionActive 任务键存在 active 会话绑定（running 代理判据 → 等待）
	SessionActive bool
}

// ScanTask 单在途任务催办/介入判定（spec §2「agent 无响应」行降级口径）：
//
//  1. 终态任务（completed/rejected/closed）不扫；
//  2. 绑定 session active（running 代理）→ 等待，不催；
//  3. 闲置 ≥ 介入阈值且无未闭合介入通知 → 介入升级（通知 user）；
//  4. 闲置 ≥ 催办阈值且无未闭合催办/介入通知 → 催办（挂 requirement/node/role）；
//  5. 其余不动。
//
// 闲置基线：未 ack（LastAck NULL）=创建时刻（派发即起算）；已 ack=最近上报时刻
// （ack/progress/complete 尝试均刷新 updated_at，作为「最近表态/推进」单一代理）。
func ScanTask(task *repo.Task, in ScanInput) ScanVerdict {
	if isTerminalStatus(task.Status) {
		return ScanVerdict{}
	}
	if in.SessionActive {
		return ScanVerdict{}
	}
	baseline := task.CreatedAt
	if task.LastAck != nil {
		baseline = task.UpdatedAt
	}
	idle := in.Now - baseline
	if idle < 0 {
		idle = 0
	}
	verdict := ScanVerdict{IdleMillis: idle, BaselineMillis: baseline}
	switch {
	case idle >= in.InterveneAfterMillis && !in.OpenIntervene:
		verdict.Action = NotifKindIntervene
	case idle >= in.FollowupAfterMillis && !in.OpenFollowup && !in.OpenIntervene:
		verdict.Action = NotifKindFollowup
	}
	return verdict
}

// isTerminalStatus 任务终态判定（与 repo.TerminalTaskStatuses 同源集；domain 侧
// 独立常量避免遍历切片，取值以 repo 列注释为单一口径）。
func isTerminalStatus(status string) bool {
	switch status {
	case repo.TaskStatusCompleted, repo.TaskStatusRejected, repo.TaskStatusClosed:
		return true
	}
	return false
}

// ScannerNotificationText 扫描器通知行 text 装配（单一口径）：
// `[<kind>] task=<task_id> idle_ms=<n> <文案>`。前缀段供查重与 T11 分类，
// 文案按 v3 规程：催办提示「遇卡点上报」；介入为 User 升级口径。
func ScannerNotificationText(kind string, taskID string, idleMillis int64) string {
	switch kind {
	case NotifKindFollowup:
		return fmt.Sprintf("[followup] task=%s idle_ms=%d 任务超时未推进，请尽快 ack/progress/complete；遇卡点请上报 user", taskID, idleMillis)
	case NotifKindIntervene:
		return fmt.Sprintf("[intervene] task=%s idle_ms=%d 任务长时间未推进，需用户介入", taskID, idleMillis)
	}
	return fmt.Sprintf("[%s] task=%s idle_ms=%d", kind, taskID, idleMillis)
}

// ScannerNotificationTextPrefix 通知查重前缀（repo 层 LIKE 前缀匹配用，
// 与 ScannerNotificationText 同源推导）。
func ScannerNotificationTextPrefix(kind string, taskID string) string {
	return fmt.Sprintf("[%s] task=%s ", kind, taskID)
}
