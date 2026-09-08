// service 包本文件：T8 编排核心会师点——wb_review（agent 表态 + collect 聚合门禁推进）
// 与 wb_decision（User 终审拍板专用路由，spec §1.2 移出清单收编）。
//
// 事务边界（spec §1.3/§2）：表态写路径与 collect 聚合判定读改写同事务，flow_nodes 行
// SELECT … FOR UPDATE 串行化并发双 collect（仅一次推进生效）；消息/通知派发移出事务
// ——本期留接线注释，T11 收件箱接线时消费。
//
// 定稿出处（docs-tmp/capability-model-final.md）：
//   - §1.2a collect 三段：收票（先齐后判）→ 判定（全 agree=approved / 有 unagree=澄清）
//     → 再收票（澄清后重入新轮）；
//   - §1.3 裁决规则：user_gate=1 唤醒 User 终审（agreed 拍板卡迎宾呈现态，ui-interaction
//     -final §1.3）、user_gate=0 机械 approved 落 last_decision 并推进；
//   - approved/rejected 是 collect 的输出，写入 last_decision（结论协议见 domain.GateDecision）。
package service

import (
	"context"
	"fmt"
	"log"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// 审计 by_who 口径补充（T8）：plugin=collect 内置编排迁移；user=User 终审拍板。
const (
	auditByWhoPlugin = "plugin"
	auditByWhoUser   = "user"
)

// ReviewService 评审表态与门禁推进 service（wb_review / wb_decision / 澄清重入接缝）。
type ReviewService struct {
	repo *repo.Repo
	// notifier 通知收敛写入口（T11：事务外派发，失败不阻断业务）
	notifier *Notifier
}

// NewReviewService 构造评审门禁 service。
func NewReviewService(r *repo.Repo) *ReviewService {
	return &ReviewService{repo: r, notifier: NewNotifier(r)}
}

// Review wb_review：agent 自判 agree/unagree 上报，写 review_results（单字段承载
// 全部 agent 的 JSON 数组条目：role/capabilityId/state/round/at）——同轮同角色覆写
// 取最后（spec §2 幂等行）；同事务 collect 聚合判定（flow_nodes 行锁，先齐后判，
// capability-model-final §1.2a）：
//
//   - pending：未齐（当前轮仍有 reviewing 位），仅持久化表态；
//   - clarify：全表态且存在 unagree → owner 澄清循环（节点 state=respond，
//     owner 经 my_task respond 相位拿 clarify 上下文与未闭合 issue 清单）；
//   - approved：全表态且全 agree → 门禁通过——user_gate=1 停在 agreed（唤醒 User，
//     wb_decision 拍板）；user_gate=0 机械 approved：last_decision 落定 +
//     flows.current_node_id 前移 + 下一节点按激活相位落 state/entered_at（空转规则
//     直过无挂载相位）。
//
// round 由服务端按节点当前评审轮推导（澄清后重入新轮），入参不携带；
// review 相位校验：非 review 相位/非评审位声明 (role, capabilityId) 拦截（blocked）。
// 写路径单事务：行锁读 → 条目覆写 → 判定 → 乐观锁更新 + 逐字段审计（表态审计
// by_who=表态角色；编排迁移审计 by_who=plugin）→ 推进写（flow/下一节点行）。
//
// 入参：requirement_id/node_id/role/capability_id 必填；state ∈ {agree, unagree}
// （reviewing 为系统初始化态，不接受上报）。
// 出参：落库后表态与 collect 判定结果（review_results 全量 JSON / gate_state /
// next_node_id）。
//
// 错误集：ErrInvalidArgument（必填缺失/state 越集）；ErrNotFound（需求/节点未命中）；
// ErrGateBlocked（非 review 相位/评审位未声明）；ErrOptimisticLock（基线冲突，409）；
// 底层 IO/JSON 错误经 Trace 上抛。
func (s *ReviewService) Review(ctx context.Context, req *api.ReviewReq) (*api.ReviewResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.NodeId == "" {
		return nil, invalidArg("node_id 必填")
	}
	if req.Role == "" {
		return nil, invalidArg("role 必填")
	}
	if req.CapabilityId == "" {
		return nil, invalidArg("capability_id 必填")
	}
	if req.State != repo.ReviewStateAgree && req.State != repo.ReviewStateUnagree {
		return nil, invalidArg("state 须为 agree|unagree（reviewing 为系统初始化态），实测 %q", req.State)
	}

	resp := &api.ReviewResp{}
	outcome := domain.CollectPending
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		node, err := repo.GetFlowNodeByRequirementAndNodeIDLockTx(ctx, tx, req.RequirementId, req.NodeId)
		if err != nil {
			return err
		}
		if node == nil {
			return notFound("节点 %q 不存在于需求 %q", req.NodeId, req.RequirementId)
		}
		if node.State != repo.PhaseReview {
			return gateBlocked("节点 %q 相位 %q 不接受评审表态（仅 review 相位）", node.NodeID, node.State)
		}
		if err := matchReviewerPosition(req.Role, req.CapabilityId, node.Reviewers); err != nil {
			return err
		}

		var entries []repo.ReviewResult
		if err := utils.UnmarshalJSONText(node.ReviewResults, &entries); err != nil {
			return errs.Trace(err)
		}
		round := domain.CurrentRound(entries)
		now := repo.NowMillis()
		entries = domain.UpsertReviewResult(entries, repo.ReviewResult{
			Role:         req.Role,
			CapabilityID: req.CapabilityId,
			State:        req.State,
			Round:        round,
			At:           now,
		})
		resultsJSON, err := utils.MarshalJSONText(entries)
		if err != nil {
			return errs.Trace(err)
		}

		updates := map[string]any{"review_results": resultsJSON}
		audits := []repo.AuditLog{reviewAudit(node, "reviewResults", node.ReviewResults, resultsJSON, req.Role)}

		var collectErr error
		outcome, collectErr = domain.CollectVerdict(node.Reviewers, resultsJSON)
		if collectErr != nil {
			return errs.Trace(collectErr)
		}
		resp.Ok = true
		resp.NodeId = node.NodeID
		resp.Role = req.Role
		resp.State = req.State
		resp.Round = round
		resp.ReviewResults = resultsJSON
		resp.CollectOutcome = string(outcome)

		needAdvance := false
		switch outcome {
		case domain.CollectClarify:
			// owner 澄清循环：节点进 respond 相位（通知派发移出事务——T11 接线）
			updates["state"] = repo.PhaseRespond
			audits = append(audits, reviewAudit(node, "state", node.State, repo.PhaseRespond, auditByWhoPlugin))
		case domain.CollectApproved:
			defNode, defErr := repo.GetFlowDefNodeByTemplateIDTx(ctx, tx, node.DefNodeID)
			if defErr != nil {
				return defErr
			}
			if defNode == nil {
				return notFound("定义节点 %q 缺失（seed 异常）", node.DefNodeID)
			}
			if defNode.UserGate == 1 {
				// agreed：唤醒 User 终审（拍板卡迎宾呈现态），不写 last_decision、不推进
				resp.GateState = domain.GateStateAgreed
			} else {
				resp.GateState = domain.GateStateApproved
				if node.LastDecision == nil {
					decisionText, buildErr := domain.BuildGateDecision(domain.GateVerdictApproved, "collect 全 agree 通过", now, "")
					if buildErr != nil {
						return errs.Trace(buildErr)
					}
					updates["last_decision"] = decisionText
					audits = append(audits, reviewAudit(node, "lastDecision", "", decisionText, auditByWhoPlugin))
					needAdvance = true
				}
				// else：已落定（并发双 collect 败者视角）——仅一次推进生效护栏
			}
		case domain.CollectPending:
			// 继续收票
		}
		if err := s.applyNodeUpdates(ctx, tx, node, updates, audits); err != nil {
			return err
		}
		if needAdvance {
			nextNodeID, advanceErr := s.advanceApprovedNode(ctx, tx, node, now)
			if advanceErr != nil {
				return advanceErr
			}
			resp.NextNodeId = nextNodeID
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	// 通知派发（事务外，spec §1.3——失败不阻断业务）：门禁 agreed（user_gate=1）
	// 唤醒 owner 拍板 / collect=clarify 转 owner 澄清循环（§6.8 双通道收件箱侧）
	s.dispatchReviewNotifications(ctx, req.RequirementId, req.NodeId, outcome, resp.GateState)
	return resp, nil
}

// dispatchReviewNotifications 评审写路径通知派发（事务提交后调用，事务外落库）：
//   - gate_state=agreed：待你拍板（gate_pending）——收件 owner（被评审方：节点 exec
//     能力归属角色优先，回落 requirements.owner），跳转=节点详情；
//   - collect=clarify：澄清任务（clarify_task）——收件 owner，跳转=节点详情
//     （respond 相位，澄清 issue 清单经 my_task 呈现）。
//
// owner 推导失败仅记日志跳过（通知为非关键路径）。
func (s *ReviewService) dispatchReviewNotifications(ctx context.Context, requirementID string, nodeID string, outcome domain.CollectOutcome, gateState string) {
	switch {
	case gateState == domain.GateStateAgreed:
	case outcome == domain.CollectClarify:
	default:
		return
	}
	owner, err := DeriveOwnerRole(ctx, s.repo, requirementID, nodeID)
	if err != nil {
		log.Printf("[notifier] owner 角色推导失败，跳过派发: req=%s node=%s err=%v", requirementID, nodeID, err)
		return
	}
	if owner == "" {
		return
	}
	spec := NotificationSpec{
		RequirementID: requirementID,
		NodeID:        nodeID,
		RecipientRole: owner,
	}
	switch {
	case gateState == domain.GateStateAgreed:
		spec.Kind = NotificationKindGatePending
		spec.Text = fmt.Sprintf("节点 %s 评审全数 agree，待你拍板", nodeID)
	default:
		spec.Kind = NotificationKindClarifyTask
		spec.Text = fmt.Sprintf("节点 %s 评审存在 unagree，请澄清回应", nodeID)
	}
	Dispatch(s.notifier, spec)
}

// Decide wb_decision：User 终审拍板专用路由（spec §1.2 移出清单——wb_decide 收编）。
// 仅 user_gate=1 节点可拍（否则 blocked）；拍板开放判据=当前评审轮全表态全 agree
// （collect=approved，拍板卡进度态时置灰拦截）；agreed 后 approved → 节点 gate 落定
// （last_decision 写 verdict JSON）+ branch 非空时 feature_branch 落值（T9 补链）+
// current_node_id 前移 + 下一节点激活；rejected → 同事务回退重入（T9：目标节点
// exec 重入 revise，reopen_source/rolled_back_count/pending_scope 落值，下游节点
// upstream-reject 联动重置，current_node_id 回退——见 RollbackForRejectTx）。
// 重复拍板同 verdict 幂等 no-op（库不变、不写审计、不重复回退）；改判（verdict
// 不一致）ErrIdempotentConflict 拒绝。写路径单事务：行锁读 → 判据校验 →
// last_decision 乐观锁更新 + 审计（by_who=user）→ 推进/回退写。
//
// 入参：requirement_id/node_id/verdict 必填；verdict ∈ {approved, rejected}；
// rejected 时 reason 必填（打回走 reject 结论协议）；branch 评审敲定分支名可选。
// 出参：落库结论（noop=true=幂等命中既有结论）；rejected 附 rollback_node_id。
//
// 错误集：ErrInvalidArgument（必填缺失/verdict 越集/rejected 缺理由）；
// ErrNotFound（需求/节点未命中）；ErrGateBlocked（非 user_gate 位/评审未全 agree/
// rejected 首渲染节点无上游可回退）；ErrIdempotentConflict（已拍板后改判）；
// ErrOptimisticLock（基线冲突，409）；底层 IO/JSON 错误经 Trace 上抛。
func (s *ReviewService) Decide(ctx context.Context, req *api.DecisionReq) (*api.DecisionResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.NodeId == "" {
		return nil, invalidArg("node_id 必填")
	}
	if req.Verdict != domain.GateVerdictApproved && req.Verdict != domain.GateVerdictRejected {
		return nil, invalidArg("verdict 须为 approved|rejected，实测 %q", req.Verdict)
	}
	if req.Verdict == domain.GateVerdictRejected && req.Reason == "" {
		return nil, invalidArg("rejected 打回必填理由（reject 结论协议）")
	}

	resp := &api.DecisionResp{}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		node, err := repo.GetFlowNodeByRequirementAndNodeIDLockTx(ctx, tx, req.RequirementId, req.NodeId)
		if err != nil {
			return err
		}
		if node == nil {
			return notFound("节点 %q 不存在于需求 %q", req.NodeId, req.RequirementId)
		}
		defNode, err := repo.GetFlowDefNodeByTemplateIDTx(ctx, tx, node.DefNodeID)
		if err != nil {
			return err
		}
		if defNode == nil {
			return notFound("定义节点 %q 缺失（seed 异常）", node.DefNodeID)
		}
		if defNode.UserGate != 1 {
			return gateBlocked("节点 %q 非 User 拍板位（user_gate=0），评审门禁由 collect 机械装配", node.NodeID)
		}

		// 重复拍板：同 verdict 幂等 no-op；改判冲突拒绝
		existing, err := domain.ParseGateDecision(derefString(node.LastDecision))
		if err != nil {
			return err
		}
		resp.Ok = true
		resp.NodeId = node.NodeID
		resp.LastDecision = derefString(node.LastDecision)
		if node.LastDecision != nil {
			resp.Verdict = existing.Verdict
			if existing.Verdict == req.Verdict {
				resp.Noop = true
				return nil
			}
			return errs.Trace(fmt.Errorf("%w: 节点 %q 已拍板 %q，拒绝改判 %q", errs.ErrIdempotentConflict, node.NodeID, existing.Verdict, req.Verdict))
		}

		// 拍板开放判据：当前评审轮全表态全 agree（拍板卡进度态=置灰，ui-interaction-final §1.3）
		outcome, err := domain.CollectVerdict(node.Reviewers, node.ReviewResults)
		if err != nil {
			return errs.Trace(err)
		}
		if outcome != domain.CollectApproved {
			return gateBlocked("节点 %q 评审未全数 agree（collect=%s），拍板未开放", node.NodeID, outcome)
		}

		now := repo.NowMillis()
		decisionText, err := domain.BuildGateDecision(req.Verdict, req.Reason, now, req.Branch)
		if err != nil {
			return errs.Trace(err)
		}
		updates := map[string]any{"last_decision": decisionText}
		audits := []repo.AuditLog{reviewAudit(node, "lastDecision", "", decisionText, auditByWhoUser)}
		// 评审敲定分支名落值（feature_branch 列——结论协议 branch 字段的持久化补链，
		// T9）：仅 approved 且携带 branch 时写（rejected 打回不敲定分支）
		if req.Verdict == domain.GateVerdictApproved && req.Branch != "" {
			updates["feature_branch"] = req.Branch
			audits = append(audits, reviewAudit(node, "featureBranch", "", req.Branch, auditByWhoUser))
		}
		if err := s.applyNodeUpdates(ctx, tx, node, updates, audits); err != nil {
			return err
		}

		// rejected：T9 回退重入（同事务）——目标节点 exec 重入 revise
		// （reopen_source=downstream-reject、rolled_back_count++、pending_scope 待重定
		// 标记），下游节点 upstream-reject 联动重置，current_node_id 回退；
		// 首渲染节点被拒（无上游）整体回滚（拍板不生效，blocked）。
		if req.Verdict == domain.GateVerdictApproved {
			nextNodeID, advanceErr := s.advanceApprovedNode(ctx, tx, node, now)
			if advanceErr != nil {
				return advanceErr
			}
			resp.NextNodeId = nextNodeID
		} else {
			flow, flowErr := repo.GetFlowByRequirementIDTx(ctx, tx, node.RequirementID)
			if flowErr != nil {
				return flowErr
			}
			if flow == nil {
				return notFound("需求 %q 无流程实例", node.RequirementID)
			}
			rollbackNodeID, rollbackErr := s.RollbackForRejectTx(ctx, tx, flow, node, now)
			if rollbackErr != nil {
				return rollbackErr
			}
			resp.RollbackNodeId = rollbackNodeID
		}
		resp.Verdict = req.Verdict
		resp.LastDecision = decisionText
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return resp, nil
}

// ReenterReviewRoundTx 澄清循环接缝（T5 respond complete 判据通过后同事务调用）：
// owner 澄清完成（该角色该节点 open issue 清零）→ review 重入新轮（collect 三段③
// 再收票）——节点 state=respond → review，review_results 全量替换为新轮快照
// （agree 携结论进新轮、unagree 复位 reviewing，domain.ReenterRoundEntries）。
// T12 扩展：review 相位同构放行（冲突上升裁决后 wb_adjudicate 直触 reviewer
// 重表态，不经 owner complete 判据——capability-model-final §6 rule6「正常判据链
// 恢复」）；其余相位幂等 no-op（重复 complete/失配无害）。
// 通知派发移出事务——T11 接线。
//
// 出参：被复位为 reviewing 的评审位角色清单（T11 re_review 通知收件人——事务
// 提交后由调用方派发；空切片=未重入/no-op）。
//
// 错误集：ErrOptimisticLock（基线冲突）；底层 IO/JSON 错误经 Trace 上抛。
func (s *ReviewService) ReenterReviewRoundTx(ctx context.Context, tx *gormio.DB, requirementID string, nodeID string) ([]string, error) {
	node, err := repo.GetFlowNodeByRequirementAndNodeIDLockTx(ctx, tx, requirementID, nodeID)
	if err != nil {
		return nil, err
	}
	if node == nil || (node.State != repo.PhaseRespond && node.State != repo.PhaseReview) {
		return nil, nil
	}
	return s.reenterReviewRoundLocked(ctx, tx, node)
}

// reenterReviewRoundLocked 评审轮重入核（调用方持有节点行锁与相位守卫——
// respond/review 双相位，T5 澄清循环与 T12 裁决链共用）：review_results 全量
// 替换为新轮快照（round+1）+ 节点 state=review + 编排审计。
//
// 出参：被复位为 reviewing 的评审位角色清单（T11 re_review 通知收件人；
// 裁决链调用方无需消费可弃）。
//
// 错误集：ErrOptimisticLock（基线冲突）；底层 IO/JSON 错误经 Trace 上抛。
func (s *ReviewService) reenterReviewRoundLocked(ctx context.Context, tx *gormio.DB, node *repo.FlowNode) ([]string, error) {
	var entries []repo.ReviewResult
	if err := utils.UnmarshalJSONText(node.ReviewResults, &entries); err != nil {
		return nil, errs.Trace(err)
	}
	nextRound := domain.CurrentRound(entries) + 1
	reentered, err := domain.ReenterRoundEntries(node.Reviewers, node.ReviewResults, nextRound, repo.NowMillis())
	if err != nil {
		return nil, errs.Trace(err)
	}
	resultsJSON, err := utils.MarshalJSONText(reentered)
	if err != nil {
		return nil, errs.Trace(err)
	}
	updates := map[string]any{
		"state":          repo.PhaseReview,
		"review_results": resultsJSON,
	}
	audits := []repo.AuditLog{
		reviewAudit(node, "state", node.State, repo.PhaseReview, auditByWhoPlugin),
		reviewAudit(node, "reviewResults", node.ReviewResults, resultsJSON, auditByWhoPlugin),
	}
	if err := s.applyNodeUpdates(ctx, tx, node, updates, audits); err != nil {
		return nil, err
	}
	resetRoles := make([]string, 0, len(reentered))
	for _, entry := range reentered {
		if entry.Round == nextRound && entry.State == repo.ReviewStateReviewing {
			resetRoles = append(resetRoles, entry.Role)
		}
	}
	return resetRoles, nil
}

// advanceApprovedNode 门禁 approved 节点推进（当前节点 gate 落定后调用）：
// flows.current_node_id 前移到渲染序下一节点；下一节点按激活相位落 state
// （domain.ActivationPhase——gate 驻留 entry、review 直入评审位、produce/align
// 空转直过无挂载相位）+ entered_at 落值；review 相位激活时初始化评审轮
// （全位 reviewing 轮 0）。尾节点（编排终点）current_node_id 保持驻留。
// 注意：调用前当前节点 last_decision 须已由调用方落定（本函数不再写当前节点行）。
func (s *ReviewService) advanceApprovedNode(ctx context.Context, tx *gormio.DB, node *repo.FlowNode, now int64) (string, error) {
	flow, err := repo.GetFlowByRequirementIDTx(ctx, tx, node.RequirementID)
	if err != nil {
		return "", err
	}
	if flow == nil {
		return "", notFound("需求 %q 无流程实例", node.RequirementID)
	}
	next, err := repo.GetNextFlowNodeTx(ctx, tx, node.RequirementID, node.NodeOrder)
	if err != nil {
		return "", err
	}
	if next == nil {
		// 尾节点：编排终点，current_node_id 驻留（流程完成语义由后续工单收口）
		return "", nil
	}
	defNode, err := repo.GetFlowDefNodeByTemplateIDTx(ctx, tx, next.DefNodeID)
	if err != nil {
		return "", err
	}
	if defNode == nil {
		return "", notFound("定义节点 %q 缺失（seed 异常）", next.DefNodeID)
	}
	phase, err := domain.ActivationPhase(defNode.NodeType, defNode.NodeActions)
	if err != nil {
		return "", errs.Trace(err)
	}

	nextUpdates := map[string]any{
		"state":      phase,
		"entered_at": now,
	}
	nextAudits := []repo.AuditLog{
		reviewAudit(next, "state", next.State, phase, auditByWhoPlugin),
		reviewAudit(next, "enteredAt", "", formatMillis(now), auditByWhoPlugin),
	}
	if phase == repo.PhaseReview {
		seeded, err := domain.InitReviewRound(next.Reviewers, 0, now)
		if err != nil {
			return "", errs.Trace(err)
		}
		seededJSON, err := utils.MarshalJSONText(seeded)
		if err != nil {
			return "", errs.Trace(err)
		}
		nextUpdates["review_results"] = seededJSON
		nextAudits = append(nextAudits, reviewAudit(next, "reviewResults", next.ReviewResults, seededJSON, auditByWhoPlugin))
	}
	if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.FlowNode{}, next.ID, next.UpdatedAt, nextUpdates); err != nil {
		return "", err
	}
	for i := range nextAudits {
		if err := s.repo.InsertAuditLogWith(tx, &nextAudits[i]); err != nil {
			return "", err
		}
	}
	if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Flow{}, flow.ID, flow.UpdatedAt, map[string]any{
		"current_node_id": next.NodeID,
	}); err != nil {
		return "", err
	}
	if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
		Table:    "flows",
		RowID:    flow.RequirementID,
		Action:   "UPDATE",
		Field:    strPtr("currentNodeId"),
		OldValue: strPtr(derefString(flow.CurrentNodeID)),
		NewValue: strPtr(next.NodeID),
		ByWho:    auditByWhoPlugin,
	}); err != nil {
		return "", err
	}
	return next.NodeID, nil
}

// applyNodeUpdates 节点行乐观锁更新 + 逐字段审计（行锁事务内，基线=锁定行读取值；
// 审计失败回滚业务事务——留痕一致口径）。
func (s *ReviewService) applyNodeUpdates(ctx context.Context, tx *gormio.DB, node *repo.FlowNode, updates map[string]any, audits []repo.AuditLog) error {
	if len(updates) > 0 {
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.FlowNode{}, node.ID, node.UpdatedAt, updates); err != nil {
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

// matchReviewerPosition 评审位声明校验：(role, capabilityId) 须在该节点实例评审位
// 声明中（scope 裁剪后），否则门禁拦截（spec §1.2 wb_my_task 三向校验的同源约束）。
func matchReviewerPosition(role string, capabilityID string, reviewersJSON string) error {
	reviewers, err := domain.ParseReviewers(reviewersJSON)
	if err != nil {
		return errs.Trace(err)
	}
	for _, reviewer := range reviewers {
		if reviewer.Role == role && reviewer.CapabilityID == capabilityID {
			return nil
		}
	}
	return gateBlocked("评审位未声明 (role=%s, capabilityId=%s)，拒绝表态", role, capabilityID)
}

// reviewAudit flow_nodes UPDATE 审计条目（by_who：表态=角色，编排迁移=plugin，
// User 拍板=user）。
func reviewAudit(node *repo.FlowNode, field string, oldVal string, newVal string, byWho string) repo.AuditLog {
	return repo.AuditLog{
		Table:    "flow_nodes",
		RowID:    node.NodeID,
		Action:   "UPDATE",
		Field:    strPtr(field),
		OldValue: strPtr(oldVal),
		NewValue: strPtr(newVal),
		ByWho:    byWho,
	}
}
