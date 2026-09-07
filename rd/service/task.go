// service 包本文件：wb_assign_task（幂等派发）与 wb_task_update（ack/progress/
// complete 上报）任务总线端点（spec §1.2 路由表两行 + §2 异常处理表前两行）。
// DTO 以 protoc 生成的 api 包类型承载（proto 为契约真相源）。
package service

import (
	"context"
	"fmt"
	"strconv"

	gormio "gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// TaskService 任务总线 service（wb_assign_task / wb_task_update）。
type TaskService struct {
	repo *repo.Repo
	// notifier 通知收敛写入口（T11：事务外派发，失败不阻断业务）
	notifier *Notifier
}

// NewTaskService 构造任务总线 service。
func NewTaskService(r *repo.Repo) *TaskService {
	return &TaskService{repo: r, notifier: NewNotifier(r)}
}

// dispatchReReviewNotifications 重新表态请求派发（事务提交后调用，事务外落库）：
// 澄清闭环重入新轮后，被复位为 reviewing 的评审位逐角色收 re_review 通知
// （§6.8「重新表态请求」——reviewer followup/复用通道；跳转=节点详情）。
func (s *TaskService) dispatchReReviewNotifications(ctx context.Context, requirementID string, nodeID string, roles []string) {
	for _, role := range roles {
		Dispatch(s.notifier, NotificationSpec{
			Kind:          NotificationKindReReview,
			RequirementID: requirementID,
			NodeID:        nodeID,
			RecipientRole: role,
			Text:          fmt.Sprintf("节点 %s 澄清已回应，请重新表态（新评审轮）", nodeID),
		})
	}
}

// Assign wb_assign_task：幂等派发——幂等键 requirement_id+node_id+action_id+
// assignee（Further Notes 口径），同键存在未终态任务（status 非 completed/
// rejected/closed）时返回既有任务（no-op，不重复建、不写审计）；终态后同键可再派。
// role_id/task_type 按 action 定位（domain.ResolveTaskAction：capability 动作须在
// 节点激活清单或评审位声明中，node action 须在节点声明；未激活 400）；branch 按
// 需求 scope 派生（rd/fe/空）；goal/context 确定性上下文装配（产物引用/意见清单
// 等注入点留待 T6/T7）。写路径单事务：flow_nodes 行锁（序列化同节点派发，防幂等
// 幻影双建）+ task_id 事务内取号（t-<ts>-<seq>）+ INSERT 审计（by_who=user，管理面
// 派发口径）；取号竞态整体重试。新建任务 status=todo、dispatch_at/last_ack NULL
// （ack 落值，断点恢复口径）。
//
// 入参：requirement_id/node_id/action_id/assignee 必填（幂等键成员）；assignee 须与
// 能力归属角色（或 node action executor_role）一致；context 提供时须为合法 JSON。
// 出参：task 任务行；reused=true=幂等命中既有任务。
//
// 错误集：ErrInvalidArgument（必填缺失/未知动作/未激活/角色不符/context 非法 JSON）；
// ErrNotFound（需求/流程实例/节点未命中）；ErrIdempotentConflict（取号重试耗尽）。
func (s *TaskService) Assign(ctx context.Context, req *api.AssignTaskReq) (*api.AssignTaskResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.NodeId == "" {
		return nil, invalidArg("node_id 必填")
	}
	if req.ActionId == "" {
		return nil, invalidArg("action_id 必填")
	}
	if req.Assignee == "" {
		return nil, invalidArg("assignee 必填")
	}
	if req.Context != "" {
		var probe map[string]any
		if err := utils.UnmarshalJSONText(req.Context, &probe); err != nil {
			return nil, invalidArg("context 非合法 JSON: %v", err)
		}
	}

	requirement, err := s.repo.GetRequirementByID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if requirement == nil {
		return nil, notFound("需求 %q 不存在", req.RequirementId)
	}
	flow, err := s.repo.GetFlowByRequirementID(ctx, req.RequirementId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if flow == nil {
		return nil, notFound("需求 %q 无流程实例", req.RequirementId)
	}
	node, err := s.repo.GetFlowNodeByRequirementAndNodeID(ctx, req.RequirementId, req.NodeId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if node == nil {
		return nil, notFound("节点 %q 不存在于需求 %q", req.NodeId, req.RequirementId)
	}

	// 激活清单按实例 scope 裁剪后定位动作（与 my_task 三向投影同函数口径）
	nodeActions, err := s.scopedNodeActions(ctx, node, flow.Scope)
	if err != nil {
		return nil, err
	}
	_, capabilityInfos, err := loadCapabilityInfos(ctx, s.repo)
	if err != nil {
		return nil, errs.Trace(err)
	}
	resolved, err := domain.ResolveTaskAction(req.ActionId, req.Assignee, nodeActions, node.Reviewers, capabilityInfos)
	if err != nil {
		return nil, err
	}

	taskCtx, err := buildTaskContext(requirement, flow, node, resolved, req.Context)
	if err != nil {
		return nil, err
	}
	goal := req.Goal
	if goal == "" {
		goal = domain.DefaultTaskGoal(node.NodeKey, resolved)
	}
	branch := domain.BranchForScope(flow.Scope)

	for attempt := 0; attempt < idAllocRetryLimit; attempt++ {
		row, reused, retryable, assignErr := s.assignOnce(ctx, req, node.ID, resolved, branch, goal, taskCtx)
		if assignErr != nil {
			if retryable {
				continue
			}
			return nil, assignErr
		}
		return &api.AssignTaskResp{Ok: true, Task: taskItem(row), Reused: reused}, nil
	}
	return nil, errs.Trace(errs.ErrIdempotentConflict)
}

// assignOnce 单次派发尝试（事务：flow_nodes 行锁 → 幂等命中检查 → 取号插入 →
// 审计）。行锁序列化同节点派发——幂等检查在锁内，并发同键幻影双建被阻断；
// 返回 retryable=true 表示取号唯一键冲突/锁竞争可整体重试。
func (s *TaskService) assignOnce(ctx context.Context, req *api.AssignTaskReq, nodeRowID int64, resolved *domain.ResolvedAction, branch string, goal string, taskCtx string) (*repo.Task, bool, bool, error) {
	row := &repo.Task{
		RequirementID: req.RequirementId,
		NodeID:        req.NodeId,
		ActionID:      resolved.ActionID,
		Branch:        branch,
		RoleID:        resolved.RoleID,
		TaskType:      resolved.TaskType,
		Goal:          goal,
		Context:       taskCtx,
		Status:        repo.TaskStatusTodo,
	}
	reused := false
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		// 行锁：同节点派发串行化（幂等检查在锁内，幻影双建被阻断）
		locked := &repo.FlowNode{}
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("id = ?", nodeRowID).
			First(locked).Error; err != nil {
			return errs.Trace(err)
		}
		existing, err := repo.FindActiveTaskByKeyTx(ctx, tx, req.RequirementId, req.NodeId, resolved.ActionID, resolved.RoleID)
		if err != nil {
			return err
		}
		if existing != nil {
			row = existing
			reused = true
			return nil
		}
		taskID, err := s.repo.AllocTaskIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.TaskID = taskID
		if err := tx.Create(row).Error; err != nil {
			return errs.Trace(err)
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "tasks",
			RowID:  taskID,
			Action: "INSERT",
			ByWho:  auditByWhoManagement,
		})
	})
	if txErr != nil {
		if repo.IsDuplicateEntry(txErr) || repo.IsLockRace(txErr) {
			return nil, false, true, errs.Trace(txErr)
		}
		return nil, false, false, errs.Trace(txErr)
	}
	return row, reused, false, nil
}

// buildTaskContext 确定性上下文装配（tasks.context TEXT 列单一口径 repo.TaskContext）：
// my_task 同源的节点/需求摘要 + 动作定位摘要；产物引用/意见清单等留待 T6/T7 注入。
// supplement 为调用方原样透传 JSON 文本（入参已校验合法）。
func buildTaskContext(requirement *repo.Requirement, flow *repo.Flow, node *repo.FlowNode, resolved *domain.ResolvedAction, supplement string) (string, error) {
	taskContext := repo.TaskContext{
		Requirement: repo.TaskContextRequirement{
			RequirementID: requirement.RequirementID,
			Title:         requirement.Title,
			Status:        requirement.Status,
		},
		Node: repo.TaskContextNode{
			NodeID:    node.NodeID,
			DefNodeID: node.DefNodeID,
			NodeKey:   node.NodeKey,
			State:     node.State,
		},
		Action: repo.TaskContextAction{
			ActionID:     resolved.ActionID,
			CapabilityID: resolved.CapabilityID,
			Name:         resolved.Name,
			On:           resolved.On,
			Skill:        resolved.Skill,
			Inline:       resolved.Inline,
		},
		Scope:      flow.Scope,
		Branch:     domain.BranchForScope(flow.Scope),
		Supplement: supplement,
	}
	return utils.MarshalJSONText(taskContext)
}

// Update wb_task_update：event 三值上报——
//
//   - ack：dispatch_at（NULL 时落值）/last_ack 落值，status todo→in-progress；
//     重复 ack 幂等 no-op（noop=true，库不变、不写审计）；
//   - progress：last_progress 覆写（progress 必填非空）；逐变更字段 UPDATE 审计；
//   - complete：判据校验框架（domain.CompleteCriteria 表驱动，事实经 repo 直查
//     artifacts/flow_nodes.review_results/issues，不依赖 T6/T7 写端点）——不满足
//     返回 ErrGateBlocked（errs.WithMissing 载体，HTTP 出参 blocked:true+missing）
//     且任务回 in-progress（事务内持久化，diff 变更才写审计）；满足 → completed +
//     done_at；未 ack 先 complete 为时序违规（ErrStaleReport，409）。
//
// 已终态任务（completed/rejected/closed）再上报一律幂等 no-op（noop=true）。
// 写路径单事务：tasks 行锁读（FOR UPDATE）+ 乐观锁更新（UpdateWithOptimisticLockTx）
// + 审计（by_who=执行角色，上报者口径）。
//
// 入参：task_id 必填；event ∈ {ack, progress, complete}；updated_at=读取基线
// （0=以当前库值为基线，宽松模式，与 wb_update_requirement 同口径）。
// 出参：更新后任务行；noop=true=幂等 no-op。
//
// 错误集：ErrInvalidArgument（task_id/event 缺失或非法、progress 空文本）；
// ErrNotFound（任务不存在）；ErrStaleReport（未 ack 先 complete）；
// ErrGateBlocked（complete 判据不满足，missing 携带缺项）；ErrOptimisticLock
// （基线不匹配，409）。
func (s *TaskService) Update(ctx context.Context, req *api.TaskUpdateReq) (*api.TaskUpdateResp, error) {
	if req.TaskId == "" {
		return nil, invalidArg("task_id 必填")
	}
	switch req.Event {
	case "ack", "progress", "complete":
	default:
		return nil, invalidArg("event 须为 ack|progress|complete，实测 %q", req.Event)
	}
	if req.Event == "progress" && req.Progress == "" {
		return nil, invalidArg("event=progress 时 progress 必填")
	}

	var (
		nooped        *repo.Task
		blockedErr    error
		reReviewRoles []string
		reviewReqID   string
		reviewNodeID  string
		respondDone   bool
	)
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		row, err := repo.GetTaskByIDTx(ctx, tx, req.TaskId)
		if err != nil {
			return err
		}
		if row == nil {
			return notFound("任务 %q 不存在", req.TaskId)
		}
		// 乐观锁基线：显式 updated_at 优先（陈旧基线 → 409）；0=以当前库值为基线
		baseline := row.UpdatedAt
		if req.UpdatedAt != 0 {
			baseline = req.UpdatedAt
		}

		// 已终态任务再上报幂等 no-op（spec §2 重复上报行）
		if isTerminalTask(row) {
			nooped = row
			return nil
		}
		// 重复 ack 幂等 no-op
		if req.Event == "ack" && row.LastAck != nil {
			nooped = row
			return nil
		}

		updates := make(map[string]any)
		audits := make([]repo.AuditLog, 0, 3)

		switch req.Event {
		case "ack":
			now := repo.NowMillis()
			if row.DispatchAt == nil {
				updates["dispatch_at"] = now
				audits = append(audits, taskUpdateAudit(row, "dispatchAt", "", formatMillis(now)))
			}
			updates["last_ack"] = now
			audits = append(audits, taskUpdateAudit(row, "lastAck", "", formatMillis(now)))
			if row.Status == repo.TaskStatusTodo {
				updates["status"] = repo.TaskStatusInProgress
				audits = append(audits, taskUpdateAudit(row, "status", row.Status, repo.TaskStatusInProgress))
			}
		case "progress":
			oldProgress := ""
			if row.LastProgress != nil {
				oldProgress = *row.LastProgress
			}
			updates["last_progress"] = req.Progress
			audits = append(audits, taskUpdateAudit(row, "lastProgress", oldProgress, req.Progress))
		case "complete":
			// 时序违规：未 ack（未送达）先 complete，破坏断点恢复口径
			if row.LastAck == nil {
				return errs.Trace(fmt.Errorf("%w: 任务 %q 未 ack，拒绝 complete（先上报 ack）", errs.ErrStaleReport, row.TaskID))
			}
			missing, criteriaErr := s.completeMissing(ctx, row)
			if criteriaErr != nil {
				return criteriaErr
			}
			if len(missing) > 0 {
				// 判据拦截：任务回 in-progress（spec §2「谎报完成」行，事务内持久化）
				if row.Status != repo.TaskStatusInProgress {
					updates["status"] = repo.TaskStatusInProgress
					audits = append(audits, taskUpdateAudit(row, "status", row.Status, repo.TaskStatusInProgress))
				}
				if err := s.applyTaskUpdates(ctx, tx, row, baseline, updates, audits); err != nil {
					return err
				}
				blockedErr = errs.WithMissing(gateBlocked("任务 %q 完成判据不满足，回 in-progress", row.TaskID), missing)
				return nil
			}
			now := repo.NowMillis()
			updates["status"] = repo.TaskStatusCompleted
			updates["done_at"] = now
			audits = append(audits, taskUpdateAudit(row, "status", row.Status, repo.TaskStatusCompleted))
			audits = append(audits, taskUpdateAudit(row, "doneAt", "", formatMillis(now)))
			if row.TaskType == repo.TaskTypeRespond {
				// T8 澄清循环接缝：owner respond 完成判据通过（open issue 清零）即
				// review 重入新轮——同事务（flow_nodes 行锁）内推进，读写原子
				roles, reErr := NewReviewService(s.repo).ReenterReviewRoundTx(ctx, tx, row.RequirementID, row.NodeID)
				if reErr != nil {
					return reErr
				}
				// T11 re_review 通知收件人（事务提交后派发，事务外落库）
				reReviewRoles = roles
				reviewReqID = row.RequirementID
				reviewNodeID = row.NodeID
				respondDone = true
			}
		}

		return s.applyTaskUpdates(ctx, tx, row, baseline, updates, audits)
	})
	if txErr != nil {
		return nil, txErr
	}
	// 通知派发（事务外，spec §1.3——失败不阻断业务）：澄清闭环 → review 重入新轮，
	// 重新表态请求（re_review）逐收件角色派发（§6.8 reviewer followup/复用通道）
	if respondDone {
		s.dispatchReReviewNotifications(ctx, reviewReqID, reviewNodeID, reReviewRoles)
	}
	if nooped != nil {
		return &api.TaskUpdateResp{
			Ok:     true,
			TaskId: nooped.TaskID,
			Status: nooped.Status,
			Noop:   true,
			Task:   taskItem(nooped),
		}, nil
	}
	if blockedErr != nil {
		return nil, blockedErr
	}
	fresh, err := s.repo.GetTaskByTaskID(ctx, req.TaskId)
	if err != nil {
		return nil, errs.Trace(err)
	}
	if fresh == nil {
		return nil, notFound("任务 %q 不存在", req.TaskId)
	}
	return &api.TaskUpdateResp{
		Ok:     true,
		TaskId: fresh.TaskID,
		Status: fresh.Status,
		Task:   taskItem(fresh),
	}, nil
}

// completeMissing 装配 complete 判据事实（repo 直查，不依赖 T6/T7 写端点）并校验：
// 期望产物 = 动作 outputProductTypes（capability 动作；node action 无产物期望），
// 空时以节点 expected_outputs 兜底；表态事实 = 节点 review_results 按 role 取最新；
// respond 事实 = 该节点指派给执行角色的 open issue 计数。
//
// 错误集：底层 IO/JSON 解析错误经 Trace 上抛。
func (s *TaskService) completeMissing(ctx context.Context, row *repo.Task) ([]string, error) {
	facts := domain.TaskFacts{Role: row.RoleID}

	node, err := s.repo.GetFlowNodeByRequirementAndNodeID(ctx, row.RequirementID, row.NodeID)
	if err != nil {
		return nil, errs.Trace(err)
	}

	// 期望产物：动作 outputProductTypes（capability 动作），空则节点 expected_outputs 兜底
	switch row.TaskType {
	case repo.TaskTypeProduce, repo.TaskTypeDev, repo.TaskTypeAccept:
		expected, err := s.expectedProductTypes(ctx, row.ActionID, node)
		if err != nil {
			return nil, err
		}
		facts.ExpectedProductTypes = expected
		artifacts, err := s.repo.ListArtifactsByRequirement(ctx, row.RequirementID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		registered := make([]string, 0, len(artifacts))
		for _, artifact := range artifacts {
			if artifact.NodeID == row.NodeID {
				registered = append(registered, artifact.ProductType)
			}
		}
		facts.RegisteredProductTypes = registered
	case repo.TaskTypeReview:
		if node == nil {
			break
		}
		var results []repo.ReviewResult
		if err := utils.UnmarshalJSONText(node.ReviewResults, &results); err != nil {
			return nil, errs.Trace(err)
		}
		states := make(map[string]string, len(results))
		for _, result := range results {
			// 同角色多轮：后写覆盖（round 递增语义），最新表态生效
			states[result.Role] = result.State
		}
		facts.ReviewStateByRole = states
	case repo.TaskTypeRespond:
		issues, err := s.repo.ListIssuesByRequirement(ctx, row.RequirementID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		for _, issue := range issues {
			if issue.NodeID != row.NodeID || issue.State != "open" {
				continue
			}
			if issue.Assignee != nil && *issue.Assignee == row.RoleID {
				facts.OpenIssueCount++
			}
		}
	}
	return domain.CompleteCriteria(row.TaskType, facts), nil
}

// scopedNodeActions 单节点激活清单取用（TaskService 侧，独立函数便于 T6/T7 同
// 包改动合并）：定义侧 JSON 按实例 scope 裁剪条目级 scope（与 my_task 同函数口径）；
// 定义缺失=seed 异常，返回空清单。
func (s *TaskService) scopedNodeActions(ctx context.Context, node *repo.FlowNode, scope string) (string, error) {
	defNodes, err := s.repo.ListFlowDefNodesByTemplateIDs(ctx, []string{node.DefNodeID})
	if err != nil {
		return "", errs.Trace(err)
	}
	if len(defNodes) == 0 {
		return "[]", nil
	}
	return domain.FilterNodeActions(defNodes[0].NodeActions, scope)
}

// expectedProductTypes 动作产出期望集：action_id 首段为能力 id 时解析其动作
// outputProductTypes；空（node action/评审类）回落节点 expected_outputs 类型集。
func (s *TaskService) expectedProductTypes(ctx context.Context, actionID string, node *repo.FlowNode) ([]string, error) {
	capabilityID, actionIDPart, isCapability := domain.SplitCapabilityAction(actionID)
	if isCapability {
		capability, err := s.repo.GetCapabilityByID(ctx, capabilityID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if capability != nil {
			var actions []repo.ActionDef
			if err := utils.UnmarshalJSONText(capability.Actions, &actions); err != nil {
				return nil, errs.Trace(err)
			}
			for _, action := range actions {
				if action.ActionID == actionIDPart && len(action.OutputProductTypes) > 0 {
					return action.OutputProductTypes, nil
				}
			}
		}
	}
	if node == nil {
		return []string{}, nil
	}
	defNodes, err := s.repo.ListFlowDefNodesByTemplateIDs(ctx, []string{node.DefNodeID})
	if err != nil {
		return nil, errs.Trace(err)
	}
	if len(defNodes) == 0 {
		return []string{}, nil
	}
	var expected []repo.ExpectedOutput
	if err := utils.UnmarshalJSONText(defNodes[0].ExpectedOutputs, &expected); err != nil {
		return nil, errs.Trace(err)
	}
	types := make([]string, 0, len(expected))
	for _, item := range expected {
		types = append(types, item.ProductType)
	}
	return types, nil
}

// loadCapabilityInfos 能力装配（TaskService 侧独立函数）：capability_id →
// 投影信息（动作词表；动作解析判据用）。
func loadCapabilityInfos(ctx context.Context, r *repo.Repo) (map[string]string, map[string]domain.CapabilityInfo, error) {
	caps, err := r.ListCapabilities(ctx)
	if err != nil {
		return nil, nil, errs.Trace(err)
	}
	roles := make(map[string]string, len(caps))
	infos := make(map[string]domain.CapabilityInfo, len(caps))
	for _, capability := range caps {
		roles[capability.CapabilityID] = capability.RoleID
		var actions []repo.ActionDef
		if err := utils.UnmarshalJSONText(capability.Actions, &actions); err != nil {
			return nil, nil, errs.Trace(err)
		}
		infos[capability.CapabilityID] = domain.CapabilityInfo{RoleID: capability.RoleID, Actions: actions}
	}
	return roles, infos, nil
}

// isTerminalTask 任务是否终态（completed/rejected/closed）。
func isTerminalTask(row *repo.Task) bool {
	for _, status := range repo.TerminalTaskStatuses {
		if row.Status == status {
			return true
		}
	}
	return false
}

// applyTaskUpdates 事务内乐观锁更新 + 逐变更字段审计（append-only helper 强制
// created_at=updated_at 同值）。
func (s *TaskService) applyTaskUpdates(ctx context.Context, tx *gormio.DB, row *repo.Task, baseline int64, updates map[string]any, audits []repo.AuditLog) error {
	if len(updates) > 0 {
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Task{}, row.ID, baseline, updates); err != nil {
			return err
		}
	}
	for i := range audits {
		if err := s.repo.InsertAuditLogWith(tx, &audits[i]); err != nil {
			return err
		}
	}
	return nil
}

// taskUpdateAudit 任务 UPDATE 审计条目（by_who=执行角色——上报者口径）。
func taskUpdateAudit(row *repo.Task, field string, oldVal string, newVal string) repo.AuditLog {
	return repo.AuditLog{
		Table:    "tasks",
		RowID:    row.TaskID,
		Action:   "UPDATE",
		Field:    strPtr(field),
		OldValue: strPtr(oldVal),
		NewValue: strPtr(newVal),
		ByWho:    row.RoleID,
	}
}

// formatMillis 毫秒时间戳审计文本（audit_log old/new 值 TEXT 列字符串口径）。
func formatMillis(v int64) string {
	return strconv.FormatInt(v, 10)
}

// taskItem repo 行转 proto 契约出参（NULL 时间列 → 0=未落值）。
func taskItem(row *repo.Task) *api.TaskItem {
	item := &api.TaskItem{
		TaskId:        row.TaskID,
		RequirementId: row.RequirementID,
		NodeId:        row.NodeID,
		ActionId:      row.ActionID,
		Branch:        row.Branch,
		RoleId:        row.RoleID,
		TaskType:      row.TaskType,
		Goal:          row.Goal,
		Context:       row.Context,
		Status:        row.Status,
		ReplyCount:    row.ReplyCount,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.DispatchAt != nil {
		item.DispatchAt = *row.DispatchAt
	}
	if row.LastAck != nil {
		item.LastAck = *row.LastAck
	}
	if row.LastProgress != nil {
		item.LastProgress = *row.LastProgress
	}
	if row.ResumeHint != nil {
		item.ResumeHint = *row.ResumeHint
	}
	if row.DoneAt != nil {
		item.DoneAt = *row.DoneAt
	}
	return item
}
