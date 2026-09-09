// service 包本文件：扫描器（T10）——在途任务催办/介入扫描循环（spec §2 异常处理
// 表「agent 无响应」行 + §4.1 P2 AC「5min 催办 / 10min 介入按时触发（注入时钟模拟）」
// +「MySQL 不可用恢复后扫描器续跑」行）。判定纯逻辑在 domain.ScanTask（注入时钟
// 以参数传入，禁止 time.Now 散落判定）；本层只做事实查询装配、通知写入与循环容错。
package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/repo"
)

// scanAssigneeUser 需你介入通知报送责任人（seed roles 内置 user 角色直报 User）。
const scanAssigneeUser = "user"

// ScannerPinger 扫描器 DB 连通探测接缝（healthz 同款分层探测的最小面；
// *database.DB 实现之，便于 IT 打桩）。
type ScannerPinger interface {
	PingContext(ctx context.Context) error
}

// ScannerOptions 扫描器参数化配置（spec §2「参数化配置」+ T10 AC「阈值参数化：
// 配置改小即可加速演练」）。
type ScannerOptions struct {
	// Interval 扫描间隔
	Interval time.Duration
	// FollowupAfter 催办阈值（默认 5min）
	FollowupAfter time.Duration
	// InterveneAfter 介入升级阈值（默认 10min；须 ≥ FollowupAfter，否则催办恒被
	// 介入取代——config 默认值保证，此处不重复校验）
	InterveneAfter time.Duration
	// RetryBackoff MySQL 不可用退避间隔
	RetryBackoff time.Duration
}

// Scanner 扫描器 service：RunOnce 单轮扫描（可注入时钟直接驱动），Run 周期循环
// （MySQL 不可用退避不崩，恢复续跑；无内存态依赖，进程重启从账面状态续扫）。
type Scanner struct {
	repo     *repo.Repo
	db       ScannerPinger
	notifier *Notifier
	adv      *AdvanceService
	opts     ScannerOptions
	now      func() int64
	log      *zap.Logger
}

// NewScanner 构造扫描器（now 缺省 repo.NowMillis；log 缺省 nop）。
func NewScanner(r *repo.Repo, db ScannerPinger, opts ScannerOptions) *Scanner {
	return &Scanner{
		repo:     r,
		db:       db,
		notifier: NewNotifier(r),
		adv:      NewAdvanceService(r),
		opts:     opts,
		now:      repo.NowMillis,
		log:      zap.NewNop(),
	}
}

// WithClock 注入时钟（单测模拟时间推进用；生产用默认 repo.NowMillis）。
func (s *Scanner) WithClock(now func() int64) *Scanner {
	s.now = now
	return s
}

// WithLogger 注入日志（缺省 nop）。
func (s *Scanner) WithLogger(lg *zap.Logger) *Scanner {
	if lg != nil {
		s.log = lg
	}
	return s
}

// ScanReport 单轮扫描报告（main 日志与测试断言载体）。
type ScanReport struct {
	// Degraded true=MySQL 不可用跳过本轮（退避后续扫，进程不崩）
	Degraded bool
	// Error 降级原因摘要
	Error string
	// Scanned 参与判定的在途任务数
	Scanned int64
	// Followups 本轮新建催办通知（task_id）
	Followups []string
	// Intervenes 本轮新建介入通知（task_id）
	Intervenes []string
	// ParallelPushes 本轮并行组兜底补推命中的需求 id 集（2026-09-09 拍板-C③：
	// 双侧均 complete 但组未推进的漏推自愈）
	ParallelPushes []string
}

// RunOnce 单轮扫描：DB ping 失败降级返回（Degraded=true，无错误上抛——调用方
// 决定退避节奏）；在途任务逐行 domain.ScanTask 判定，命中阈值且无同类未闭合通知
// 时经 Notifier 写催办/介入通知行（挂 requirement/node/role）。
//
// 幂等：同一任务同轮已有同类未闭合 notification（state 不在 resolved/overridden
// 闭合集）不重复建（spec §2 幂等口径 + T10 AC「催办幂等」）。
//
// 错误集：查询/写入 IO 错误经 Trace 上抛（ping 探测已隔离主链路不可用场景；
// 扫描中途 IO 失败由 Run 循环捕获记日志、下轮续扫）。
func (s *Scanner) RunOnce(ctx context.Context) (*ScanReport, error) {
	report := &ScanReport{
		Followups:      make([]string, 0),
		Intervenes:     make([]string, 0),
		ParallelPushes: make([]string, 0),
	}
	if s.db != nil {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := s.db.PingContext(pingCtx)
		cancel()
		if err != nil {
			report.Degraded = true
			report.Error = err.Error()
			return report, nil
		}
	}
	tasks, err := s.repo.ListInflightTasks(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}
	report.Scanned = int64(len(tasks))
	now := s.now()
	for _, task := range tasks {
		// 事实装配：同类未闭合通知查重 + active 会话绑定（running 代理判据）
		openFollowup, err := s.repo.FindOpenScannerNotification(ctx, task.RequirementID, task.NodeID,
			domain.ScannerNotificationTextPrefix(domain.NotifKindFollowup, task.TaskID))
		if err != nil {
			return nil, errs.Trace(err)
		}
		openIntervene, err := s.repo.FindOpenScannerNotification(ctx, task.RequirementID, task.NodeID,
			domain.ScannerNotificationTextPrefix(domain.NotifKindIntervene, task.TaskID))
		if err != nil {
			return nil, errs.Trace(err)
		}
		session, err := s.repo.FindActiveSessionByTaskKey(ctx, task.RequirementID, task.NodeID, task.RoleID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		verdict := domain.ScanTask(task, domain.ScanInput{
			Now:                  now,
			FollowupAfterMillis:  s.opts.FollowupAfter.Milliseconds(),
			InterveneAfterMillis: s.opts.InterveneAfter.Milliseconds(),
			OpenFollowup:         openFollowup != nil,
			OpenIntervene:        openIntervene != nil,
			SessionActive:        session != nil,
		})
		if verdict.Action == "" {
			continue
		}
		if err := s.notifyVerdict(ctx, task, verdict); err != nil {
			return nil, err
		}
		if verdict.Action == domain.NotifKindFollowup {
			report.Followups = append(report.Followups, task.TaskID)
		} else {
			report.Intervenes = append(report.Intervenes, task.TaskID)
		}
	}
	// 并行组兜底巡检（2026-09-09 拍板-C③）：「双侧均 complete 但组未推进」的
	// 流程经同一推进核补推（同 flows 行锁 + 审计 by_who=system）——漏推自愈
	pushed, err := s.adv.SweepParallelJoins(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}
	report.ParallelPushes = pushed
	// 通知 outbox 补投（spec §1.2 注记-H）：failed（retry 预算内）+ pending/sending
	// 停滞（业务提交后 immediate flush 未触达/进程崩溃残留）周期兜底
	s.notifier.SweepOutbox(ctx, now)
	s.notifier.FlushPending(ctx)
	return report, nil
}

// notifyVerdict 判定动作落通知行（悬置催办报送执行角色；需你介入报送 user——
// v3 §3.1 报送责任人列）。
func (s *Scanner) notifyVerdict(ctx context.Context, task *repo.Task, verdict domain.ScanVerdict) error {
	assignee := task.RoleID
	if verdict.Action == domain.NotifKindIntervene {
		assignee = scanAssigneeUser
	}
	// 归一写入口（T11 七类 Notify 经 outbox 通道——spec §1.2 注记-H）：扫描器 kind
	// 落 stall_followup/need_intervene 枚举（anchor_kind=类别）；派发者/角色保持
	// 扫描器 system 口径（T10 行为不变，文本前缀幂等查重不变）；task 类目标 id 落
	// anchor_artifact_id（收件箱跳转）。enqueue 用 autocommit 单语句（scanner 无
	// 业务事务），随 RunOnce 尾部 SweepOutbox/FlushPending 推送。
	kind := NotificationKindStallFollowup
	if verdict.Action == domain.NotifKindIntervene {
		kind = NotificationKindNeedIntervene
	}
	err := EnqueueNotificationTx(ctx, s.repo.Gorm(), NotificationSpec{
		Kind:          kind,
		RequirementID: task.RequirementID,
		NodeID:        task.NodeID,
		TargetID:      task.TaskID,
		RecipientRole: assignee,
		Sender:        auditByWhoSystem,
		SenderRole:    auditByWhoSystem,
		Text:          domain.ScannerNotificationText(verdict.Action, task.TaskID, verdict.IdleMillis),
	})
	return err
}

// Run 周期扫描循环：启动即扫一轮；此后每 Interval 一轮；单轮降级（MySQL 不可用）
// 按 RetryBackoff 退避后继续（恢复后续跑，账面状态无内存依赖）；stop 关闭或 ctx
// 取消即退出（main 优雅退出联动）。
func (s *Scanner) Run(ctx context.Context, stop <-chan struct{}) {
	for {
		report, err := s.RunOnce(ctx)
		if err != nil {
			s.log.Warn("扫描器单轮失败（下轮续扫）", zap.Error(err))
		} else if report.Degraded {
			s.log.Warn("MySQL 不可用，扫描器降级退避", zap.String("error", report.Error))
			if !s.sleep(ctx, stop, s.opts.RetryBackoff) {
				return
			}
		} else if len(report.Followups) > 0 || len(report.Intervenes) > 0 {
			s.log.Info("扫描器产出通知",
				zap.Strings("followups", report.Followups),
				zap.Strings("intervenes", report.Intervenes),
			)
		}
		if !s.sleep(ctx, stop, s.opts.Interval) {
			return
		}
	}
}

// sleep 等待 d 或 stop/ctx 触发；false=应退出。
func (s *Scanner) sleep(ctx context.Context, stop <-chan struct{}, d time.Duration) bool {
	if d <= 0 {
		d = time.Second
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-stop:
		return false
	case <-timer.C:
		return true
	}
}
