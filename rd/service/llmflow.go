// service 包本文件：LLM 通路执行器（spec 注记 -L4 + MIG-08/MIG-09 执行拍板）
// ——wbflow 编排引擎的真执行层：produce 通路（产出文档入库）与 review 通路
// （读被评审产物→意见评论挂锚→wb_review 表态）。
//
// 执行形态（拍板 B 异步）：节点激活=wb_assign_task 派发 → 后台 goroutine 调 LLM
// → 产物入库（wb_add_artifact，-K kind=document）/ 意见评论（-L3 range|document
// 锚）→ wb_task_update complete 走既有判据链（exec 族 complete 内聚触发编排推进，
// HTTP 全程不阻塞）。派发与驱动由扫描器每轮 SweepLLMTasks 收口：
//
//   - 派发：节点处于 entry/exec/review 相位且动作在准入集（config llm.actions，
//     entry=align 类节点激活相位，BUG-018；样板
//     先行=to-brd + arch.prd-review.review，机制通用扩面改清单）、无同键在途任务
//     → wb_assign_task 派发并启动执行；
//   - 恢复（拍板「scanner 轮询发现异常中断任务→继续任务」）：在途 LLM 任务进程内
//     无执行协程登记（服务重启/进程崩溃后必然成立，镜像 outbox sweep 模式）→
//     重新触发执行（继续而非回退）；「超阈值无心跳」语义由扫描间隔承载；
//   - 停摆护栏：节点存在未闭合 llm-fail 通知（任务回退后置位，User 处理后在
//     收件箱解决）→ 该节点暂停派发与驱动，防耗尽后无限重派。
//
// 失败策略（拍板 B+）：预检 fail fast（输入缺失直接失败不空调）→ 确定性错误不
// 重试；LLM 调用走指数退避重试器（llm.retry_max 参数化默认 6）→ 耗尽=任务回退
// （exec 重入语义：exec 族任务置 rejected + 节点 rolled_back_count++/reopen_source
// =rework-return 预留词表）+ need_intervene 置位 + 告警通知（复用通知族，报 User）。
package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/llm"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// llmFailNotifPrefix LLM 失败通知 text 前缀（节点级停摆护栏查重键——与扫描器
// [followup]/[intervene] 前缀族同构，FindOpenScannerNotification LIKE 前缀匹配）。
const llmFailNotifPrefix = "[llm-fail] "

// llmRetryBase/llmRetryCap 指数退避序列参数（1s 起步、30s 封顶；总次数由
// LLMFlowOptions.RetryMax 装配，拍板默认 6）。
const (
	llmRetryBase = time.Second
	llmRetryCap  = 30 * time.Second
)

// maxLLMReviewComments 单轮评审自动落评论上限（防模型 findings 失控刷库；
// 超出部分截断并在意见正文标注）。
const maxLLMReviewComments = 20

// LLMFlowOptions LLM 执行器参数（config llm 段装配）。
type LLMFlowOptions struct {
	// Enabled 总开关（false=SweepLLMTasks 空转）
	Enabled bool
	// RetryMax LLM 调用总尝试次数（含首次；拍板默认 6）
	RetryMax int
	// Actions 自动执行准入动作集（动作定位全名；空集=不派发）
	Actions []string
}

// LLMFlowService LLM 通路执行器：扫描器每轮派发+驱动，后台协程执行 produce/
// review 两通路。client 为装配层注入的 Completer（base_url 空=FakeClient 演示态）。
type LLMFlowService struct {
	repo      *repo.Repo
	tasks     *TaskService
	artifacts *ArtifactService
	issues    *IssueService
	reviews   *ReviewService
	notifier  *Notifier
	client    llm.Completer
	opts      LLMFlowOptions
	allowed   map[string]bool
	// running 进程内执行协程登记（taskID → 执行中）：双驱动去重 + 停滞恢复
	// 判据（扫描轮发现「在途且无登记」即视为异常中断，重新触发=继续任务）
	mu      sync.Mutex
	running map[string]bool
	// backoffSink 单测采集退避序列注入点（nil=生产等待）
	backoffSink func(d time.Duration)
	now         func() int64
	log         *zap.Logger
}

// NewLLMFlowService 构造 LLM 执行器（client 必填，演示态传 FakeClient）。
func NewLLMFlowService(r *repo.Repo, tasks *TaskService, artifacts *ArtifactService, issues *IssueService, reviews *ReviewService, client llm.Completer, opts LLMFlowOptions) *LLMFlowService {
	allowed := make(map[string]bool, len(opts.Actions))
	for _, id := range opts.Actions {
		allowed[id] = true
	}
	return &LLMFlowService{
		repo:      r,
		tasks:     tasks,
		artifacts: artifacts,
		issues:    issues,
		reviews:   reviews,
		notifier:  NewNotifier(r),
		client:    client,
		opts:      opts,
		allowed:   allowed,
		running:   make(map[string]bool),
		now:       repo.NowMillis,
		log:       zap.NewNop(),
	}
}

// WithLogger 注入日志（缺省 nop）。
func (s *LLMFlowService) WithLogger(lg *zap.Logger) *LLMFlowService {
	if lg != nil {
		s.log = lg
	}
	return s
}

// WithBackoffSink 注入退避采集（单测/IT 免真实等待的时钟接缝；生产保持 nil 走
// 真实 timer 等待）。fn 逐次收到指数退避间隔。
func (s *LLMFlowService) WithBackoffSink(fn func(d time.Duration)) *LLMFlowService {
	s.backoffSink = fn
	return s
}

// LLMSweepReport 单轮 LLM 巡检报告（扫描报告与测试断言载体）。
type LLMSweepReport struct {
	// Dispatched 本轮新派发任务（wb_assign_task）
	Dispatched []string
	// Continued 本轮恢复驱动（停滞续跑/崩溃续跑）的任务
	Continued []string
}

// llmAction LLM 候选动作（派发定位载体）。
type llmAction struct {
	// ActionID 动作定位（wb_assign_task 入参口径）
	ActionID string
	// Assignee 执行角色
	Assignee string
}

// SweepLLMTasks 单轮 LLM 巡检：逐编排中流程扫 exec/review 相位节点，派发缺失的
// 准入动作任务、恢复驱动在途任务（继续语义）。节点存在未闭合 llm-fail 通知时
// 整节点停摆（等 User 处理后解决通知恢复）。scanner.RunOnce 消费。
//
// 错误集：底层 IO/JSON 错误经 Trace 上抛（单流程失败整体上抛，由扫描循环容错）。
func (s *LLMFlowService) SweepLLMTasks(ctx context.Context) (*LLMSweepReport, error) {
	report := &LLMSweepReport{Dispatched: []string{}, Continued: []string{}}
	if s.client == nil || !s.opts.Enabled {
		return report, nil
	}
	flows, err := s.repo.ListFlowsInOrchestration(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}
	for _, flow := range flows {
		nodes, err := s.repo.ListFlowNodesByRequirement(ctx, flow.RequirementID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		for _, node := range nodes {
			var candidates []llmAction
			switch node.State {
			case repo.PhaseEntry, repo.PhaseExec:
				// entry：align 类节点（开场白挂 entry）的激活相位且终生驻留——
				// 挂载其上的 exec 动作（to-brd 样板宿主）经 execCandidates 内部
				// 过滤（on=exec+subagent+produce 族+准入）派发；program 执行者与
				// 非 exec 条目自然排除。BUG-018（2026-09-15 QA 轮）：原闸仅
				// exec/review，align 宿主动作永不自动派发（IT 造数置 exec 掩盖）。
				candidates, err = s.execCandidates(ctx, flow, node)
			case repo.PhaseReview:
				candidates, err = s.reviewCandidates(ctx, node)
			}
			if err != nil {
				return nil, err
			}
			if len(candidates) == 0 {
				continue
			}
			// 停摆护栏：节点存在未闭合 llm-fail 通知（回退后置位）→ 派发与驱动
			// 全停，等 User 处理后在收件箱解决通知恢复（防耗尽后无限重派）
			halted, err := s.repo.FindOpenScannerNotification(ctx, flow.RequirementID, node.NodeID, llmFailNotifPrefix)
			if err != nil {
				return nil, errs.Trace(err)
			}
			if halted != nil {
				continue
			}
			tasks, err := s.repo.ListTasksByRequirementAndNode(ctx, flow.RequirementID, node.NodeID)
			if err != nil {
				return nil, errs.Trace(err)
			}
			for _, candidate := range candidates {
				active := findInflightTask(tasks, candidate)
				if active == nil {
					if hasCompletedTask(tasks, candidate) {
						// 同动作已有完成任务（推进受阻滞留 exec 相位场景）不再重产——
						// 重复派发会把 exec 判据重新拉回「存在在途任务」阻塞推进
						continue
					}
					assigned, assignErr := s.tasks.Assign(ctx, &api.AssignTaskReq{
						RequirementId: flow.RequirementID,
						NodeId:        node.NodeID,
						ActionId:      candidate.ActionID,
						Assignee:      candidate.Assignee,
					})
					if assignErr != nil {
						return nil, assignErr
					}
					report.Dispatched = append(report.Dispatched, assigned.Task.TaskId)
					active = &repo.Task{TaskID: assigned.Task.TaskId, ActionID: candidate.ActionID, RoleID: candidate.Assignee}
				} else if s.continueKey(active) != "" {
					report.Continued = append(report.Continued, active.TaskID)
				} else {
					continue // 进程内执行中（running 登记），不双驱动
				}
				s.launch(active.TaskID)
			}
		}
	}
	return report, nil
}

// continueKey 在途任务是否需要恢复驱动：进程内无执行协程登记=异常中断（服务
// 重启/崩溃后必然成立）→ 返回 taskID 表「继续」；执行中返回空串跳过。
func (s *LLMFlowService) continueKey(task *repo.Task) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[task.TaskID] {
		return ""
	}
	return task.TaskID
}

// launch 启动执行协程（同步登记 running 防扫描轮与启动间隙的双驱动竞态；
// 协程退出时解除登记）。
func (s *LLMFlowService) launch(taskID string) {
	s.mu.Lock()
	if s.running[taskID] {
		s.mu.Unlock()
		return
	}
	s.running[taskID] = true
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, taskID)
			s.mu.Unlock()
			// panic 兜底（novel worker 教训）：协程异常退出不留在 running 登记，
			// 下一扫描轮按停滞恢复继续
			if r := recover(); r != nil {
				s.log.Error("LLM 执行协程 panic", zap.String("task", taskID), zap.Any("panic", r))
			}
		}()
		if err := s.runTask(context.Background(), taskID); err != nil {
			s.log.Error("LLM 任务执行失败（回退+介入）", zap.String("task", taskID), zap.Error(err))
			if failErr := s.failTask(context.Background(), taskID, err); failErr != nil {
				s.log.Error("LLM 任务失败落账异常", zap.String("task", taskID), zap.Error(failErr))
			}
		}
	}()
}

// findInflightTask 在途任务中定位候选动作的活跃任务（幂等键 action_id+role_id
// 同 wb_assign_task 派发键；未命中 nil）。
func findInflightTask(tasks []*repo.Task, candidate llmAction) *repo.Task {
	for _, task := range tasks {
		if task.ActionID == candidate.ActionID && task.RoleID == candidate.Assignee && !isTerminalTask(task) {
			return task
		}
	}
	return nil
}

// hasCompletedTask 候选动作是否已有完成任务（重派护栏，推进受阻场景防反复重产）。
func hasCompletedTask(tasks []*repo.Task, candidate llmAction) bool {
	for _, task := range tasks {
		if task.ActionID == candidate.ActionID && task.RoleID == candidate.Assignee && task.Status == repo.TaskStatusCompleted {
			return true
		}
	}
	return false
}

// execCandidates exec 相位 LLM 候选动作集：激活清单（scope 裁剪后）× 重入配置
// 选择（rolled_back_count 首产/重入二选一，domain 单一口径）× 准入集 ∩ 家族
// （produce/dev/accept，create_reqs 无产物判据不入）× 素材存在（skill/inline）。
// kind=node 动作须 executor=subagent（program=内置动作非 LLM 通路）。
//
// 错误集：底层 IO/JSON 错误经 Trace 上抛。
func (s *LLMFlowService) execCandidates(ctx context.Context, flow *repo.Flow, node *repo.FlowNode) ([]llmAction, error) {
	defNodes, err := s.repo.ListFlowDefNodesByTemplateIDs(ctx, []string{node.DefNodeID})
	if err != nil {
		return nil, errs.Trace(err)
	}
	if len(defNodes) == 0 {
		return nil, nil
	}
	filtered, err := domain.FilterNodeActions(defNodes[0].NodeActions, flow.Scope)
	if err != nil {
		return nil, errs.Trace(err)
	}
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(filtered, &entries); err != nil {
		return nil, errs.Trace(err)
	}
	_, infos, err := loadCapabilityInfos(ctx, s.repo)
	if err != nil {
		return nil, err
	}
	// 期望产物 kind 对照（-K）：LLM 只能产入库文档，期望 link 的动作集不派发
	kindMap, err := s.tasks.expectedKindMap(ctx, node)
	if err != nil {
		return nil, err
	}
	candidates := make([]llmAction, 0, 1)
	for _, entry := range entries {
		switch entry.Kind {
		case repo.NodeActionKindNode:
			if entry.On != repo.PhaseExec || entry.Executor != repo.ExecutorSubagent {
				continue
			}
			if entry.Skill == "" && entry.Inline == "" {
				continue
			}
			if taskType, known := domain.TaskTypeForAction(entry.ActionID); !known || taskType != repo.TaskTypeProduce {
				continue
			}
			if s.allowed[entry.ActionID] {
				candidates = append(candidates, llmAction{ActionID: entry.ActionID, Assignee: entry.ExecutorRole})
			}
		case repo.NodeActionKindCapability:
			info := infos[entry.CapabilityID]
			active := make([]domain.ActiveAction, 0, len(entry.Actions))
			for _, actionID := range entry.Actions {
				for _, def := range info.Actions {
					if def.ActionID == actionID {
						active = append(active, domain.ActiveAction{
							ActionID:     entry.CapabilityID + "." + def.ActionID,
							Name:         def.Name,
							Kind:         repo.NodeActionKindCapability,
							CapabilityID: entry.CapabilityID,
							On:           def.On,
							Skill:        def.Skill,
							Inline:       def.Inline,
						})
					}
				}
			}
			kept := domain.FilterExecActionsByReentry(active, node.RolledBackCount)
			for _, action := range kept {
				bare := action.ActionID
				if _, id, isCapability := domain.SplitCapabilityAction(action.ActionID); isCapability {
					bare = id
				}
				taskType, known := domain.TaskTypeForAction(bare)
				if !known || taskType == repo.TaskTypeCreateReqs || !domain.IsExecAdvanceFamily(taskType) {
					continue
				}
				if action.Skill == "" && action.Inline == "" {
					continue
				}
				// -K 硬约束：动作期望产物须全为入库文档（含 link 期望的开发节点
				// 不入 LLM 通路——外链产物 LLM 无法真实产出）
				if !llmProducible(bare, info, kindMap) {
					continue
				}
				if s.allowed[action.ActionID] {
					candidates = append(candidates, llmAction{ActionID: action.ActionID, Assignee: info.RoleID})
				}
			}
		}
	}
	return candidates, nil
}

// llmProducible 动作产物可入库判定（bare=能力内裸动作 id）：outputProductTypes
// 逐类型查节点期望 kind 对照——声明为 link 的类型不可 LLM 产出；未声明（期望外
// 补充）按可产出；动作未命中词表（防御）按可产出。
func llmProducible(bare string, info domain.CapabilityInfo, kindMap map[string]string) bool {
	for _, def := range info.Actions {
		if def.ActionID != bare {
			continue
		}
		for _, productType := range def.OutputProductTypes {
			if kind, declared := kindMap[productType]; declared && kind != domain.ArtifactKindDocument {
				return false
			}
		}
		return true
	}
	return true
}

// reviewCandidates review 相位 LLM 候选评审位集：评审位声明 × 评审能力 review
// 动作（skill/inline 素材存在）× 准入集 × 当轮未表态（reviewing）。
//
// 错误集：底层 IO/JSON 错误经 Trace 上抛。
func (s *LLMFlowService) reviewCandidates(ctx context.Context, node *repo.FlowNode) ([]llmAction, error) {
	// 门禁已落定（approved/rejected）→ 评审收敛完成，不再派发表态
	if node.LastDecision != nil {
		return nil, nil
	}
	reviewers, err := domain.ParseReviewers(node.Reviewers)
	if err != nil {
		return nil, errs.Trace(err)
	}
	_, infos, err := loadCapabilityInfos(ctx, s.repo)
	if err != nil {
		return nil, err
	}
	var entries []repo.ReviewResult
	if err := utils.UnmarshalJSONText(node.ReviewResults, &entries); err != nil {
		return nil, errs.Trace(err)
	}
	round := domain.CurrentRound(entries)
	stated := make(map[string]bool, len(entries))
	for _, result := range entries {
		if result.Round == round && result.State != repo.ReviewStateReviewing {
			stated[result.Role] = true
		}
	}
	candidates := make([]llmAction, 0, len(reviewers))
	for _, reviewer := range reviewers {
		if stated[reviewer.Role] {
			continue
		}
		actionID := reviewer.CapabilityID + ".review"
		info := infos[reviewer.CapabilityID]
		material := false
		for _, def := range info.Actions {
			if def.ActionID == "review" && def.On == repo.PhaseReview && (def.Skill != "" || def.Inline != "") {
				material = true
				break
			}
		}
		if !material {
			continue
		}
		if s.allowed[actionID] {
			candidates = append(candidates, llmAction{ActionID: actionID, Assignee: reviewer.Role})
		}
	}
	return candidates, nil
}

// runTask 单任务执行主体（后台协程消费；错误统一上抛由 launch 落回退链）：
// ack → 按任务族分流 produce/review 通路 → complete 走既有判据链。
func (s *LLMFlowService) runTask(ctx context.Context, taskID string) error {
	task, err := s.repo.GetTaskByTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil || isTerminalTask(task) {
		return nil
	}
	// ack：todo → in-progress（重复 ack 幂等 no-op；断点恢复口径）
	if _, err := s.tasks.Update(ctx, &api.TaskUpdateReq{TaskId: taskID, Event: "ack"}); err != nil {
		return err
	}
	flow, err := s.repo.GetFlowByRequirementID(ctx, task.RequirementID)
	if err != nil {
		return err
	}
	if flow == nil {
		return notFound("需求 %q 无流程实例", task.RequirementID)
	}
	node, err := s.repo.GetFlowNodeByRequirementAndNodeID(ctx, task.RequirementID, task.NodeID)
	if err != nil {
		return err
	}
	if node == nil {
		return notFound("节点 %q 不存在于需求 %q", task.NodeID, task.RequirementID)
	}
	switch task.TaskType {
	case repo.TaskTypeProduce, repo.TaskTypeDev, repo.TaskTypeAccept:
		return s.runProduce(ctx, task, flow, node)
	case repo.TaskTypeReview:
		return s.runReview(ctx, task, flow, node)
	default:
		// 非 LLM 族任务（人工/agent 域）不由本执行器处理
		return nil
	}
}

// heartbeat 任务进度心跳（每次尝试后落 last_progress——扫描器停滞判定与收件箱
// 呈现的活性证据；失败不阻断主链路）。
func (s *LLMFlowService) heartbeat(ctx context.Context, taskID string, text string) {
	_, err := s.tasks.Update(ctx, &api.TaskUpdateReq{TaskId: taskID, Event: "progress", Progress: text})
	if err != nil {
		s.log.Warn("LLM 心跳写入失败（不阻断）", zap.String("task", taskID), zap.Error(err))
	}
}

// completeLLMText 带重试与心跳的 LLM 补全（两通路共用）：指数退避重试器 +
// OnAttempt 心跳。单测经 backoffSink 采集退避序列（免真实等待）。
func (s *LLMFlowService) completeLLMText(ctx context.Context, taskID string, system string, user string) (string, error) {
	out, _, err := llm.CompleteWithRetry(ctx, s.client, system, user, llm.RetryOptions{
		Max:        s.opts.RetryMax,
		Base:       llmRetryBase,
		MaxBackoff: llmRetryCap,
		Sleep: func(ctx context.Context, d time.Duration) bool {
			if s.backoffSink != nil {
				s.backoffSink(d)
				return true
			}
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return false
			case <-timer.C:
				return true
			}
		},
		OnAttempt: func(attempt int, attemptErr error) {
			status := "成功"
			if attemptErr != nil {
				status = "失败：" + truncateLLMText(attemptErr.Error(), 160)
			}
			s.heartbeat(ctx, taskID, fmt.Sprintf("llm: 第 %d/%d 次调用%s", attempt, s.opts.RetryMax, status))
		},
	})
	return out, err
}

// runProduce produce 通路：素材组装（动作素材 skill/inline + 上游入库产物正文）
// → 预检 fail fast → LLM 产文档 → 写 docs root → wb_add_artifact（-K
// kind=document）→ complete 判据链（exec 族内聚触发编排推进）。
func (s *LLMFlowService) runProduce(ctx context.Context, task *repo.Task, flow *repo.Flow, node *repo.FlowNode) error {
	var taskCtx repo.TaskContext
	if err := utils.UnmarshalJSONText(task.Context, &taskCtx); err != nil {
		return err
	}
	// 预检（fail fast，不空调）：期望产物集非空且全为入库文档期望
	expected, err := s.tasks.expectedProductTypes(ctx, task.ActionID, node)
	if err != nil {
		return err
	}
	if len(expected) == 0 {
		return precheckFailure("动作 %q 无期望产物集，无法确定产出物", task.ActionID)
	}
	kindMap, err := s.tasks.expectedKindMap(ctx, node)
	if err != nil {
		return err
	}
	for _, productType := range expected {
		if kind, declared := kindMap[productType]; declared && kind != domain.ArtifactKindDocument {
			return precheckFailure("期望产物 %q 为 %s 期望，LLM 通路仅支持入库文档（kind=document）", productType, kind)
		}
	}
	// 上游素材：直接上游连续 produce 段已登记产物正文（-L4「读输入=上游入库产物」）
	docs, err := s.tasks.upstreamProduceDocs(ctx, flow, node)
	if err != nil {
		return err
	}
	upstream, err := s.renderUpstreamDocs(ctx, task.RequirementID, docs)
	if err != nil {
		return err
	}
	labelByType, err := s.expectedLabelMap(ctx, node)
	if err != nil {
		return err
	}
	user := buildProduceUserPrompt(taskCtx, node, task.Goal, expected, labelByType, upstream)
	out, err := s.completeLLMText(ctx, task.TaskID, llm.ProduceSystemPrompt(task.RoleID, actionMaterial(taskCtx)), user)
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return precheckFailure("LLM 产出为空文本")
	}
	// 产出入库：写 docs root + wb_add_artifact（幂等：同 href 重复登记命中既有行）
	for _, productType := range expected {
		href := llmDocHref(task.RequirementID, node.NodeID, productType)
		if err := s.artifacts.WriteDocument(href, out); err != nil {
			return err
		}
		if _, err := s.artifacts.Add(ctx, &api.AddArtifactReq{
			RequirementId: task.RequirementID,
			NodeId:        node.NodeID,
			ProductType:   productType,
			Label:         labelOf(productType, labelByType),
			Href:          href,
			Kind:          domain.ArtifactKindDocument,
			Note:          "LLM 产出（-L4 通路自动登记）",
			AddedBy:       task.RoleID,
		}); err != nil {
			return err
		}
	}
	// complete 走既有判据链（exec 族判据通过 → completed + 编排内聚推进）
	if _, err := s.tasks.Update(ctx, &api.TaskUpdateReq{TaskId: task.TaskID, Event: "complete"}); err != nil {
		return err
	}
	return nil
}

// runReview review 通路：读被评审产物（依赖文档正文，行号标注）→ LLM 按 JSON
// 形状契约出 findings → 逐条意见评论（-L3 锚点 range|document 挂被评审产物）→
// wb_review 表态（有意见=unagree 走澄清循环，无意见=agree）→ complete 判据链。
func (s *LLMFlowService) runReview(ctx context.Context, task *repo.Task, flow *repo.Flow, node *repo.FlowNode) error {
	var taskCtx repo.TaskContext
	if err := utils.UnmarshalJSONText(task.Context, &taskCtx); err != nil {
		return err
	}
	// 预检（fail fast）：被评审产物（dependencyDocs）必须存在且可读
	if len(taskCtx.DependencyDocs) == 0 {
		return precheckFailure("评审任务无被评审产物（dependencyDocs 空），拒绝空调模型")
	}
	var user strings.Builder
	for _, doc := range taskCtx.DependencyDocs {
		content, href, err := s.artifactText(ctx, task.RequirementID, doc.ArtifactID)
		if err != nil {
			return err
		}
		user.WriteString(fmt.Sprintf("── 被评审产物《%s》（%s，artifactId=%s，%s）──\n", doc.Label, doc.ProductType, doc.ArtifactID, href))
		user.WriteString(llm.LineNumbered(content))
		user.WriteString("\n\n")
	}
	capabilityID := taskCtx.Action.CapabilityID
	if capabilityID == "" {
		return precheckFailure("评审任务缺 capabilityId，无法定位评审能力")
	}
	prompt := user.String()
	out, err := s.completeLLMText(ctx, task.TaskID, llm.ReviewSystemPrompt(task.RoleID, capabilityID, actionMaterial(taskCtx)), prompt)
	if err != nil {
		return err
	}
	findings := llm.ParseReviewFindings(out)
	// 意见评论（-L3 锚点：行级→range 划词，全篇级→document；挂被评审产物）
	truncated := false
	posted := 0
	for _, finding := range findings {
		if posted >= maxLLMReviewComments {
			truncated = true
			break
		}
		anchor := s.reviewAnchor(finding, taskCtx.DependencyDocs)
		if _, err := s.issues.Comment(ctx, &api.CommentReq{
			RequirementId: task.RequirementID,
			NodeId:        node.NodeID,
			Author:        task.RoleID,
			RoleId:        task.RoleID,
			IssueType:     domain.IssueTypeIssue,
			Text:          finding.Desc,
			Anchor:        anchor,
		}); err != nil {
			return err
		}
		posted++
	}
	// 表态：有意见=unagree（意见须澄清闭环后才能 agree，注记-L1 同构）；无意见=agree
	state := repo.ReviewStateAgree
	summary := fmt.Sprintf("LLM 评审通过（findings=%d）", len(findings))
	if len(findings) > 0 {
		state = repo.ReviewStateUnagree
		summary = fmt.Sprintf("LLM 评审提出 %d 条意见（已挂锚评论）", len(findings))
	}
	if truncated {
		summary += fmt.Sprintf("（超出 %d 条上限截断，详见通知）", maxLLMReviewComments)
	}
	s.heartbeat(ctx, task.TaskID, "llm: "+summary)
	if _, err := s.reviews.Review(ctx, &api.ReviewReq{
		RequirementId: task.RequirementID,
		NodeId:        node.NodeID,
		Role:          task.RoleID,
		CapabilityId:  capabilityID,
		State:         state,
	}); err != nil {
		return err
	}
	if _, err := s.tasks.Update(ctx, &api.TaskUpdateReq{TaskId: task.TaskID, Event: "complete"}); err != nil {
		return err
	}
	return nil
}

// reviewAnchor findings → 评论锚点装配（-L3 两级）：startLine/endLine 合法 →
// range（划词）；否则 document（文档级）。artifactId 取首个被评审产物（单产物
// 评审为主形态；多产物时意见挂首个——宽口径 MVP）。
func (s *LLMFlowService) reviewAnchor(finding llm.ReviewFinding, docs []repo.TaskContextDependencyDoc) *api.CommentAnchor {
	if len(docs) == 0 {
		return nil
	}
	artifactID := docs[0].ArtifactID
	if finding.StartLine >= 1 && finding.EndLine >= finding.StartLine {
		return &api.CommentAnchor{
			Kind:       domain.IssueAnchorRange,
			ArtifactId: artifactID,
			StartLine:  finding.StartLine,
			EndLine:    finding.EndLine,
		}
	}
	return &api.CommentAnchor{Kind: domain.IssueAnchorDocument, ArtifactId: artifactID}
}

// artifactText 产物正文读取（评审素材）：document 读正文；link 取 href 标注
// （外链正文不可达，评审按元信息给全篇意见）。
func (s *LLMFlowService) artifactText(ctx context.Context, requirementID string, artifactID string) (string, string, error) {
	resp, err := s.artifacts.GetContent(ctx, &api.GetArtifactContentReq{RequirementId: requirementID, ArtifactId: artifactID})
	if err != nil {
		return "", "", err
	}
	if resp.Kind == domain.ArtifactKindLink {
		return fmt.Sprintf("（外链产物，正文不可达：%s）", resp.Href), resp.Href, nil
	}
	return resp.Content, resp.Href, nil
}

// renderUpstreamDocs produce 素材的上游产物段装配。
func (s *LLMFlowService) renderUpstreamDocs(ctx context.Context, requirementID string, docs []repo.TaskContextDependencyDoc) (string, error) {
	if len(docs) == 0 {
		return "", nil
	}
	var b strings.Builder
	for _, doc := range docs {
		content, href, err := s.artifactText(ctx, requirementID, doc.ArtifactID)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "── 上游产物《%s》（%s，artifactId=%s，%s）──\n%s\n\n", doc.Label, doc.ProductType, doc.ArtifactID, href, content)
	}
	return b.String(), nil
}

// buildProduceUserPrompt produce 通路 user 段装配（素材组装段单一口径）：
// 需求/节点/目标摘要 + 期望产物 + 上游入库产物正文 + 动作素材。
func buildProduceUserPrompt(taskCtx repo.TaskContext, node *repo.FlowNode, goal string, expected []string, labelByType map[string]string, upstream string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "需求：%s《%s》（状态 %s，scope %s）\n", taskCtx.Requirement.RequirementID, taskCtx.Requirement.Title, taskCtx.Requirement.Status, taskCtx.Scope)
	fmt.Fprintf(&b, "节点：%s（%s，相位 %s）\n", node.NodeKey, node.NodeID, node.State)
	if goal != "" {
		fmt.Fprintf(&b, "任务目标：%s\n", goal)
	}
	labels := make([]string, 0, len(expected))
	for _, productType := range expected {
		labels = append(labels, labelOf(productType, labelByType))
	}
	fmt.Fprintf(&b, "期望产物：%s\n", strings.Join(labels, "、"))
	if upstream != "" {
		b.WriteString("\n上游产物正文：\n")
		b.WriteString(upstream)
	} else {
		b.WriteString("\n（无上游产物——本节点为首产出节点，依据需求摘要直接产出。）\n")
	}
	b.WriteString("\n请输出期望产物的完整文档正文（Markdown），仅正文，无任何解释。")
	return b.String()
}

// actionMaterial 动作素材文本（capability 动作 skill/inline=prompt 素材，-L4）：skill
// 名（DSH skill 挂载引用）与 inline 定义拼装；均空时回退动作名。
func actionMaterial(taskCtx repo.TaskContext) string {
	parts := make([]string, 0, 2)
	if taskCtx.Action.Name != "" {
		parts = append(parts, "动作："+taskCtx.Action.Name)
	}
	if taskCtx.Action.Skill != "" {
		parts = append(parts, "产出方法参考（skill）："+taskCtx.Action.Skill)
	}
	if taskCtx.Action.Inline != "" {
		parts = append(parts, "要点："+taskCtx.Action.Inline)
	}
	return strings.Join(parts, "；")
}

// expectedLabelMap 节点期望产物 productType → label（登记 Label 装配用；定义
// 缺失=空表回落 productType）。
func (s *LLMFlowService) expectedLabelMap(ctx context.Context, node *repo.FlowNode) (map[string]string, error) {
	expected, err := defExpectedOutputs(ctx, s.repo, node.DefNodeID)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(expected))
	for _, item := range expected {
		labels[item.ProductType] = item.Label
	}
	return labels, nil
}

// labelOf productType 显示名（期望声明优先，回落类型名）。
func labelOf(productType string, labelByType map[string]string) string {
	if label, ok := labelByType[productType]; ok && label != "" {
		return label
	}
	return productType
}

// llmDocHref LLM 产出入库 href（docs root 相对路径，确定性可重入：同任务恢复
// 重跑覆盖同文件，产物登记幂等命中既有行）。
func llmDocHref(requirementID string, nodeID string, productType string) string {
	return fmt.Sprintf("llm/%s/%s/%s.md", requirementID, nodeID, productType)
}

// truncateLLMText 文本截断（通知/心跳载体长度护栏）。
func truncateLLMText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// precheckFailure 预检 fail fast（确定性错误，不进重试器直接走回退链）。
func precheckFailure(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("LLM 预检未通过: %s", fmt.Sprintf(format, args...)))
}

// failTask 任务失败落账（回退链，launch 消费；幂等：已终态任务跳过）：
//   - 任务置 rejected（done_at/last_progress 落因，by_who=system 审计）；
//   - exec 族任务：节点 exec 重入语义回退（rolled_back_count++ / reopen_source=
//     rework-return 预留词表）——重新派发时按重入配置选择（revise 等）；
//   - need_intervene 置位 + 告警通知（复用通知族，报 User；未闭合期间该节点
//     派发/驱动停摆，User 处理后解决通知恢复自动执行）。
//
// 事务边界：回退写与通知 enqueue 同事务（outbox 原子口径），提交后即时推送。
func (s *LLMFlowService) failTask(ctx context.Context, taskID string, cause error) error {
	reason := truncateLLMText(cause.Error(), 400)
	now := repo.NowMillis()
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := repo.GetTaskByIDTx(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if row == nil || isTerminalTask(row) {
			return nil
		}
		updates := map[string]any{
			"status":        repo.TaskStatusRejected,
			"done_at":       now,
			"last_progress": reason,
		}
		// 失败落账审计 by_who=system（系统回退口径，非执行角色上报）
		audits := []repo.AuditLog{
			{Table: "tasks", RowID: row.TaskID, Action: "UPDATE", Field: utils.StrPtr("status"), OldValue: utils.StrPtr(row.Status), NewValue: utils.StrPtr(repo.TaskStatusRejected), ByWho: auditByWhoSystem},
			{Table: "tasks", RowID: row.TaskID, Action: "UPDATE", Field: utils.StrPtr("lastProgress"), OldValue: utils.StrPtr(utils.DerefStr(row.LastProgress)), NewValue: utils.StrPtr(reason), ByWho: auditByWhoSystem},
			{Table: "tasks", RowID: row.TaskID, Action: "UPDATE", Field: utils.StrPtr("doneAt"), OldValue: utils.StrPtr(""), NewValue: utils.StrPtr(utils.FormatMillis(now)), ByWho: auditByWhoSystem},
		}
		baseline := row.UpdatedAt
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Task{}, row.ID, baseline, updates); err != nil {
			return err
		}
		for i := range audits {
			if err := s.repo.InsertAuditLogWith(tx, &audits[i]); err != nil {
				return err
			}
		}
		// exec 族：节点 exec 重入语义回退（review 族不动 rolled_back_count——
		// 该计数在评审节点承载打回轮护栏，误加会破坏 collect round 口径）
		if domain.IsExecAdvanceFamily(row.TaskType) {
			node, nodeErr := repo.GetFlowNodeByRequirementAndNodeIDLockTx(ctx, tx, row.RequirementID, row.NodeID)
			if nodeErr != nil {
				return nodeErr
			}
			if node != nil {
				if err := repo.ApplyNodeChangeTx(ctx, s.repo, tx, node, auditByWhoSystem,
					repo.RolledBackCount(node.RolledBackCount+1),
					repo.ReopenSource(domain.ReopenSourceReworkReturn),
				); err != nil {
					return err
				}
			}
		}
		// need_intervene 置位 + 告警通知（报 User；text 前缀=节点级停摆护栏查重键）
		return EnqueueNotificationTx(ctx, tx, NotificationSpec{
			Kind:          NotificationKindNeedIntervene,
			RequirementID: row.RequirementID,
			NodeID:        row.NodeID,
			TargetID:      row.TaskID,
			RecipientRole: scanAssigneeUser,
			Sender:        auditByWhoSystem,
			SenderRole:    auditByWhoSystem,
			Text: fmt.Sprintf("%stask=%s node=%s LLM 任务失败已回退，需人工介入：%s（处理后请在收件箱解决本通知以恢复自动执行）",
				llmFailNotifPrefix, row.TaskID, row.NodeID, reason),
		})
	})
	if txErr != nil {
		return txErr
	}
	s.notifier.FlushPending(ctx)
	return nil
}
