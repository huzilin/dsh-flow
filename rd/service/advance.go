// service 包本文件：wb_advance 编排推进端点（2026-09-09 拍板-B，spec §1.2 注记-B/C）
// ——三类推进的统一承载与并行 join 三层中的「提交瞬间检查 + flows 行锁」层：
//
//   - ①跨节点推进：exec 相位 complete 判据已通过 → 下一节点相位初始化
//     （domain.ActivationPhase；review 相位 InitReviewRound 轮 0）+ flows.current_node_id
//     前移（与 wb_review 机械 approved 的 advanceApprovedNode 同一套推进核）；
//   - ②节点内 CR 轮转：dev→review（本节点 state=review + InitReviewRound 轮 0，
//     current 不动——节点内评审异化）；
//   - ③QA/门禁机械收口：collect rule4 空转语义（exec 相位无挂载动作视为通过直接
//     轮转；review 相位评审位空转/全 agree 由本端点补落 last_decision 并推进）。
//
// 并行 join（注记-C①②）：RD∥FE 并行开发组（flow_def_nodes 配置判定：相邻 rd 侧 ×
// fe 侧的 produce+gate=1+有评审位节点，业务流程 full 场景 bus-08/bus-09 即该组）
// 推进时检查对侧——对侧已 complete（last_decision approved）→ 整组推进下一阶段；
// 对侧未完成 → 本侧完成待对侧（对称判定）。推进路径全部经 flows 行
// SELECT … FOR UPDATE（注记-C②，RC + 单 master 行锁已足），锁序统一 flows →
// flow_nodes，防双侧同时 complete 的并发双推进。
//
// 触发方（注记-B）：wb_task_update complete 判据通过后同事务内聚调用
// advanceNodeLocked（Web 一期服务端内聚触发）；本端点同时暴露供兜底与手动场景；
// 巡检补推（注记-C③，scanner）复用同一推进核。
package service

import (
	"context"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// wb_advance 出参 outcome 迁移结果摘要词表。
const (
	// AdvanceOutcomeRotate 节点内 CR 轮转（exec→review 同节点）
	AdvanceOutcomeRotate = "rotate"
	// AdvanceOutcomeCross 跨节点推进（含并行组整组推进）
	AdvanceOutcomeCross = "cross"
	// AdvanceOutcomeWait 并行组本侧完成待对侧（对侧 advance/巡检对称补推）
	AdvanceOutcomeWait = "wait"
	// AdvanceOutcomeNoop 幂等命中（推进已完成，返回当前态）
	AdvanceOutcomeNoop = "noop"
)

// AdvanceService 编排推进 service（wb_advance 端点 + complete 内聚触发 + 巡检补推
// 的共享推进核）。
type AdvanceService struct {
	repo *repo.Repo
}

// NewAdvanceService 构造编排推进 service。
func NewAdvanceService(r *repo.Repo) *AdvanceService {
	return &AdvanceService{repo: r}
}

// advanceResult 单次推进结果（resp 装配与调用方判定载体）。
type advanceResult struct {
	// Outcome 迁移结果摘要（rotate/cross/wait/noop）
	Outcome string
	// NodeState 推进后该节点相位
	NodeState string
	// CurrentNodeID 推进后 flows.current_node_id
	CurrentNodeID string
	// NextNodeID 跨节点推进落位目标（未跨节点空串）
	NextNodeID string
}

// Advance wb_advance：编排推进端点（reqId/nodeId 取路径）。
//
// 幂等（注记-B）：重复调用时若推进已完成（跨节点已落位/轮转已生效/组已推进），
// 返回 200 当前态 noop=true 不重复推进；相位不满足推进条件返回 ErrGateBlocked
// （409 blocked）。
//
// 入参：requirement_id/node_id 必填。
// 出参：推进后聚合状态（节点相位 / flows.current_node_id / 迁移结果摘要）。
//
// 错误集：ErrInvalidArgument（必填缺失）；ErrNotFound（需求无流程实例/节点未命中）；
// ErrGateBlocked（相位不满足推进条件/exec 判据未通过/user_gate 待拍板/respond 澄清中/
// entry 待拍板/节点未进入编排）；ErrOptimisticLock；底层 IO/JSON 错误经 Trace 上抛。
func (s *AdvanceService) Advance(ctx context.Context, req *api.AdvanceReq) (*api.AdvanceResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.NodeId == "" {
		return nil, invalidArg("node_id 必填")
	}
	var res *advanceResult
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		flow, err := repo.GetFlowByRequirementIDLockTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}
		if flow == nil {
			return notFound("需求 %q 无流程实例", req.RequirementId)
		}
		nodes, err := repo.ListFlowNodesByRequirementLockTx(ctx, tx, req.RequirementId)
		if err != nil {
			return err
		}
		res, err = s.advanceNodeLocked(ctx, tx, flow, nodes, req.NodeId, auditByWhoPlugin)
		return err
	})
	if txErr != nil {
		return nil, txErr
	}
	return &api.AdvanceResp{
		Ok:            true,
		RequirementId: req.RequirementId,
		NodeId:        req.NodeId,
		NodeState:     res.NodeState,
		CurrentNodeId: res.CurrentNodeID,
		NextNodeId:    res.NextNodeID,
		Outcome:       res.Outcome,
		Noop:          res.Outcome == AdvanceOutcomeNoop,
	}, nil
}

// advanceNodeLocked 推进核分发（调用方事务内已持有 flows 行锁；nodes 为锁定读
// 清单，推进写以清单行为基线）。按节点相位分发三类推进：
//
//   - exec：complete 判据（exec 族任务完成）或 rule4 空转 → 有评审位=②节点内
//     CR 轮转；无评审位=①跨节点推进（仅 current 驻留本节点时；已前移=幂等 noop）；
//   - review：last_decision 已 approved → 组/下一节点补推（幂等）；collect approved
//     且 user_gate=0 → ③机械收口（补落 last_decision + 推进）；其余（pending/
//     clarify）→ 节点内轮转已完成，noop 返回当前态（幂等语义例外）；
//   - respond/entry/空相位 → blocked（分别由 owner complete 链路 / wb_decision /
//     实例化激活驱动）。
//
// 错误集：ErrNotFound（节点未在渲染集）；ErrGateBlocked；ErrOptimisticLock；
// 底层 IO/JSON 错误经 Trace 上抛。
func (s *AdvanceService) advanceNodeLocked(ctx context.Context, tx *gormio.DB, flow *repo.Flow, nodes []*repo.FlowNode, nodeID string, byWho string) (*advanceResult, error) {
	idx := flowNodeIndex(nodes, nodeID)
	if idx < 0 {
		return nil, notFound("节点 %q 不存在于需求 %q", nodeID, flow.RequirementID)
	}
	node := nodes[idx]
	res := &advanceResult{
		NodeState:     node.State,
		CurrentNodeID: derefString(flow.CurrentNodeID),
	}

	switch node.State {
	case repo.PhaseExec:
		defNode, err := repo.GetFlowDefNodeByTemplateIDTx(ctx, tx, node.DefNodeID)
		if err != nil {
			return nil, err
		}
		if defNode == nil {
			return nil, notFound("定义节点 %q 缺失（seed 异常）", node.DefNodeID)
		}
		ready, missing, err := s.execAdvanceReady(ctx, tx, defNode, node)
		if err != nil {
			return nil, err
		}
		if !ready {
			return nil, errs.WithMissing(gateBlocked("节点 %q exec 相位 complete 判据未通过，拒绝推进", node.NodeID), missing)
		}
		reviewers, err := domain.ParseReviewers(node.Reviewers)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if len(reviewers) > 0 {
			// ② 节点内 CR 轮转：exec→review 同节点，current 不动（节点内评审异化，
			// 全位 reviewing 轮 0）；重复调用时相位已非 exec → 走 review 分支幂等
			return s.rotateSelfLocked(ctx, tx, node, res)
		}
		// ① 跨节点推进：仅 current 驻留本节点时有语义（已前移=推进已完成，幂等 noop）
		if flow.CurrentNodeID == nil || *flow.CurrentNodeID != node.NodeID {
			res.Outcome = AdvanceOutcomeNoop
			return res, nil
		}
		return s.advanceAfterGateLockedTx(ctx, tx, flow, nodes, idx, repo.NowMillis(), byWho, res)
	case repo.PhaseReview:
		decision, err := domain.ParseGateDecision(derefString(node.LastDecision))
		if err != nil {
			return nil, err
		}
		if decision.Verdict != domain.GateVerdictApproved {
			// 未落定：collect 判定通过与否分流——approved 且 user_gate=0 机械收口；
			// pending/clarify=轮转已完成等待评审收敛（幂等 noop，重复 advance 不重复推进）
			outcome, verdictErr := domain.CollectVerdict(node.Reviewers, node.ReviewResults)
			if verdictErr != nil {
				return nil, errs.Trace(verdictErr)
			}
			if outcome != domain.CollectApproved {
				res.Outcome = AdvanceOutcomeNoop
				return res, nil
			}
			defNode, defErr := repo.GetFlowDefNodeByTemplateIDTx(ctx, tx, node.DefNodeID)
			if defErr != nil {
				return nil, defErr
			}
			if defNode == nil {
				return nil, notFound("定义节点 %q 缺失（seed 异常）", node.DefNodeID)
			}
			if defNode.UserGate == 1 {
				return nil, gateBlocked("节点 %q 评审已全数 agree，待 User 终审拍板（wb_decision）", node.NodeID)
			}
			// ③ 机械收口：补落 last_decision（结论协议）后走统一推进
			decisionText, buildErr := domain.BuildGateDecision(domain.GateVerdictApproved, "wb_advance 机械收口（collect approved）", repo.NowMillis(), "")
			if buildErr != nil {
				return nil, errs.Trace(buildErr)
			}
			if err := applyNodeUpdatesTx(ctx, s.repo, tx, node, map[string]any{
				"last_decision": decisionText,
			}, []repo.AuditLog{
				reviewAudit(node, "lastDecision", derefString(node.LastDecision), decisionText, byWho),
			}); err != nil {
				return nil, err
			}
		}
		// 已落定 approved：组/下一节点补推（幂等——已推进则 noop）
		return s.advanceAfterGateLockedTx(ctx, tx, flow, nodes, idx, repo.NowMillis(), byWho, res)
	case repo.PhaseRespond:
		return nil, gateBlocked("节点 %q 澄清中（respond 相位由 owner complete 判据触发评审重入）", node.NodeID)
	case repo.PhaseEntry:
		return nil, gateBlocked("节点 %q entry 相位由 wb_decision 拍板推进", node.NodeID)
	case repo.PhaseDone:
		// 注记-L2：done=已被推进离场（业务完成态）——重复 advance 幂等命中 noop
		// （「已完成推进应 200 幂等」口径不变），不得回落为 blocked。
		res.Outcome = AdvanceOutcomeNoop
		return res, nil
	default:
		return nil, gateBlocked("节点 %q 未进入编排（state 空），无可推进相位", node.NodeID)
	}
}

// rotateSelfLocked ② 节点内 CR 轮转写路径：state=review + entered_at 落值 +
// InitReviewRound 轮 0 播种（与 e2e 原 rotateSelf 接缝同口径），current 不动。
func (s *AdvanceService) rotateSelfLocked(ctx context.Context, tx *gormio.DB, node *repo.FlowNode, res *advanceResult) (*advanceResult, error) {
	now := repo.NowMillis()
	seeded, err := domain.InitReviewRound(node.Reviewers, 0, now)
	if err != nil {
		return nil, errs.Trace(err)
	}
	seededJSON, err := utils.MarshalJSONText(seeded)
	if err != nil {
		return nil, errs.Trace(err)
	}
	updates := map[string]any{
		"state":          repo.PhaseReview,
		"entered_at":     now,
		"review_results": seededJSON,
	}
	audits := []repo.AuditLog{
		reviewAudit(node, "state", node.State, repo.PhaseReview, auditByWhoPlugin),
		reviewAudit(node, "enteredAt", enteredAtText(node), formatMillis(now), auditByWhoPlugin),
		reviewAudit(node, "reviewResults", node.ReviewResults, seededJSON, auditByWhoPlugin),
	}
	if err := applyNodeUpdatesTx(ctx, s.repo, tx, node, updates, audits); err != nil {
		return nil, err
	}
	res.Outcome = AdvanceOutcomeRotate
	res.NodeState = repo.PhaseReview
	return res, nil
}

// advanceAfterGateLockedTx 门禁落定后的统一推进核（wb_review 机械 approved /
// wb_decision approved / wb_advance 补推与跨节点共用；调用方持有 flows 行锁）：
// 并行 join 判定（注记-C①）——
//
//   - 本侧 rd 侧 exec-CR 节点（produce+gate=1+有评审位+scope 仅 rd 侧）且下一节点
//     为 fe 侧 exec-CR 节点：对侧已 approved → 整组推进（跳过对侧激活下一阶段）；
//     未 approved → 激活对侧交接（本侧完成待对侧的常规面）；
//   - 本侧 fe 侧 exec-CR 节点且前一节点为 rd 侧 exec-CR 节点：对侧未 approved →
//     wait（不动 current，本侧完成待对侧，对侧 advance/巡检对称判定补推）；
//     已 approved → 推进下一阶段；
//   - 非并行组：常规推进下一节点（域 gate/review/produce 节点语义不变）。
//
// 出参：落位目标节点 id 与 moved（false=尾节点驻留/目标已落位/current 未变——幂等）。
// 对侧 complete 判定 = 对侧 last_decision verdict=approved（exec-CR 节点门禁落定
// 即开发完成语义）。
//
// 错误集：ErrNotFound（定义节点缺失）；ErrOptimisticLock；底层 IO/JSON 错误经 Trace 上抛。
func (s *AdvanceService) advanceAfterGateLockedTx(ctx context.Context, tx *gormio.DB, flow *repo.Flow, nodes []*repo.FlowNode, idx int, now int64, byWho string, res *advanceResult) (*advanceResult, error) {
	node := nodes[idx]
	// 幂等护栏：current 驻留本节点或紧邻组员（交接/补推视角）才有推进语义——
	// current 已越过本组 = 推进已完成，陈旧节点重放不得把 current 拖回。
	// current 为空（未进入编排）放行：首节点拍板/激活即首次推进。
	currentID := derefString(flow.CurrentNodeID)
	if currentID != "" && currentID != node.NodeID {
		adjacent := (idx+1 < len(nodes) && nodes[idx+1].NodeID == currentID) ||
			(idx-1 >= 0 && nodes[idx-1].NodeID == currentID)
		if !adjacent {
			res.Outcome = AdvanceOutcomeNoop
			return res, nil
		}
	}
	var next, prev, next2 *repo.FlowNode
	if idx+1 < len(nodes) {
		next = nodes[idx+1]
	}
	if idx-1 >= 0 {
		prev = nodes[idx-1]
	}
	if idx+2 < len(nodes) {
		next2 = nodes[idx+2]
	}

	// 定义侧批量装配（并行组判定与激活相位读取的静态 seed）
	defIDs := make([]string, 0, 4)
	for _, n := range []*repo.FlowNode{node, prev, next, next2} {
		if n == nil {
			continue
		}
		dup := false
		for _, id := range defIDs {
			if id == n.DefNodeID {
				dup = true
				break
			}
		}
		if !dup {
			defIDs = append(defIDs, n.DefNodeID)
		}
	}
	defRows, err := repo.ListFlowDefNodesByTemplateIDsTx(ctx, tx, defIDs)
	if err != nil {
		return nil, err
	}
	defBy := make(map[string]*repo.FlowDefNode, len(defRows))
	for _, defNode := range defRows {
		defBy[defNode.TemplateNodeID] = defNode
	}
	nodeDef := defBy[node.DefNodeID]
	if nodeDef == nil {
		return nil, notFound("定义节点 %q 缺失（seed 异常）", node.DefNodeID)
	}

	activate := func(target *repo.FlowNode) (*advanceResult, error) {
		return s.activateAndMoveLockedTx(ctx, tx, flow, nodes, defBy, target, now, byWho, res)
	}

	switch {
	case isRdSideExecCR(nodeDef) && next != nil && isFeSideExecCR(defBy[next.DefNodeID]):
		// 并行组本侧=rd：对侧已 complete → 整组推进下一阶段（跳过对侧）
		if peerApproved(next) {
			return activate(next2)
		}
		return activate(next)
	case isFeSideExecCR(nodeDef) && prev != nil && isRdSideExecCR(defBy[prev.DefNodeID]):
		// 并行组本侧=fe：对侧未 complete → 本侧完成待对侧（不动 current）
		if !peerApproved(prev) {
			res.Outcome = AdvanceOutcomeWait
			return res, nil
		}
		return activate(next)
	default:
		return activate(next)
	}
}

// activateAndMoveLockedTx 目标节点激活（空相位→激活相位初始化，review 相位播种
// 评审轮 0；已激活=幂等跳过，不重复初始化）+ flows.current_node_id 前移（已驻留
// 目标=不写）。moved=false=目标已落位或尾节点（幂等）。
func (s *AdvanceService) activateAndMoveLockedTx(ctx context.Context, tx *gormio.DB, flow *repo.Flow, nodes []*repo.FlowNode, defBy map[string]*repo.FlowDefNode, target *repo.FlowNode, now int64, byWho string, res *advanceResult) (*advanceResult, error) {
	if target == nil {
		// 尾节点：编排终点，current_node_id 驻留（幂等 noop）
		res.Outcome = AdvanceOutcomeNoop
		return res, nil
	}
	defNode := defBy[target.DefNodeID]
	if defNode == nil {
		return nil, notFound("定义节点 %q 缺失（seed 异常）", target.DefNodeID)
	}

	// 注记-L2（BUG-016 拍板 2026-09-10）：推进离场终态——目标节点之前的全部活跃相位
	// 节点（entry/exec/review/respond）统一置 done（业务完成态，老模型「已通过」的
	// v1.0 对应物）。按切片序（node_order 有序）收口：常规离场 + 并行组对侧 + 存量
	// 脏状态一次自愈。done 禁止挂载产物/任务（artifact/assign 守卫）；打回重入由
	// 回退核复位（RollbackForRejectTx），reproject 裁剪不涉 done。
	targetIdx := -1
	for i, n := range nodes {
		if n.NodeID == target.NodeID {
			targetIdx = i
			break
		}
	}
	if targetIdx > 0 {
		for _, leave := range nodes[:targetIdx] {
			switch leave.State {
			case repo.PhaseEntry, repo.PhaseExec, repo.PhaseReview, repo.PhaseRespond:
				audits := []repo.AuditLog{reviewAudit(leave, "state", leave.State, repo.PhaseDone, byWho)}
				if err := applyNodeUpdatesTx(ctx, s.repo, tx, leave, map[string]any{"state": repo.PhaseDone}, audits); err != nil {
					return nil, err
				}
			}
		}
	}
	if target.State == "" {
		phase, err := domain.ActivationPhase(defNode.NodeType, defNode.NodeActions)
		if err != nil {
			return nil, errs.Trace(err)
		}
		updates := map[string]any{
			"state":      phase,
			"entered_at": now,
		}
		audits := []repo.AuditLog{
			reviewAudit(target, "state", target.State, phase, byWho),
			reviewAudit(target, "enteredAt", enteredAtText(target), formatMillis(now), byWho),
		}
		if phase == repo.PhaseReview {
			seeded, err := domain.InitReviewRound(target.Reviewers, 0, now)
			if err != nil {
				return nil, errs.Trace(err)
			}
			seededJSON, err := utils.MarshalJSONText(seeded)
			if err != nil {
				return nil, errs.Trace(err)
			}
			updates["review_results"] = seededJSON
			audits = append(audits, reviewAudit(target, "reviewResults", target.ReviewResults, seededJSON, byWho))
		}
		if err := applyNodeUpdatesTx(ctx, s.repo, tx, target, updates, audits); err != nil {
			return nil, err
		}
	}
	// 需求状态联动（2026-09-10 BUG-001 拍板，spec §7.3 收尾联动补实现）：触发点
	// ①流程首次进入编排 / ②落位尾节点，详见 syncRequirementStatusLockedTx。
	if err := s.syncRequirementStatusLockedTx(ctx, tx, flow, nodes, target, byWho); err != nil {
		return nil, err
	}
	if flow.CurrentNodeID != nil && *flow.CurrentNodeID == target.NodeID {
		res.Outcome = AdvanceOutcomeNoop
		res.NextNodeID = target.NodeID
		res.CurrentNodeID = target.NodeID
		return res, nil
	}
	if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Flow{}, flow.ID, flow.UpdatedAt, map[string]any{
		"current_node_id": target.NodeID,
	}); err != nil {
		return nil, err
	}
	if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
		Table:    "flows",
		RowID:    flow.RequirementID,
		Action:   "UPDATE",
		Field:    utils.StrPtr("currentNodeId"),
		OldValue: utils.StrPtr(derefString(flow.CurrentNodeID)),
		NewValue: utils.StrPtr(target.NodeID),
		ByWho:    byWho,
	}); err != nil {
		return nil, err
	}
	res.Outcome = AdvanceOutcomeCross
	res.NextNodeID = target.NodeID
	res.CurrentNodeID = target.NodeID
	return res, nil
}

// syncRequirementStatusLockedTx requirements.status ↔ 编排推进联动写核（2026-09-10
// BUG-001 拍板，spec §7.3「收尾→已完成」联动补实现）。落点在激活落位核
// activateAndMoveLockedTx——wb_advance / wb_decision / complete 内聚推进 /
// scanner 巡检四条路径统一收口于此。触发点二：
//
//   - 流程首次进入编排（flows.current_node_id 由空变非空）→ 待启动→进行中；
//   - 落位尾节点（实例链最后一个节点被激活/驻留）→ 已完成（优先于进行中判定——
//     单节点流程首次推进即完成）。
//
// 写口径：同事务 + 审计（by_who=推进方 plugin/system）；目标态与现值同值不写
// （幂等，不产生重复审计）；已取消=终态不覆盖（取消需求不应再推进，防御性跳过）。
// requirements 乐观锁不介入：不带基线直更 status 列（行锁由 flows 链路串行化），
// 避免与 wb_update_requirement 手动更新互相制造 409 冲突面。
//
// 错误集：ErrNotFound（需求行缺失）；底层 IO 错误经 Trace 上抛。
func (s *AdvanceService) syncRequirementStatusLockedTx(ctx context.Context, tx *gormio.DB, flow *repo.Flow, nodes []*repo.FlowNode, target *repo.FlowNode, byWho string) error {
	if target == nil || len(nodes) == 0 {
		return nil
	}
	firstEntry := flow.CurrentNodeID == nil || *flow.CurrentNodeID == ""
	tail := target.NodeID == nodes[len(nodes)-1].NodeID
	if !firstEntry && !tail {
		return nil
	}
	targetStatus := domain.ReqStatusRunning
	if tail {
		targetStatus = domain.ReqStatusDone
	}
	row, err := repo.GetRequirementByIDTx(ctx, tx, flow.RequirementID)
	if err != nil {
		return errs.Trace(err)
	}
	if row == nil {
		return notFound("需求 %q 不存在", flow.RequirementID)
	}
	switch {
	case row.Status == domain.ReqStatusCanceled:
		return nil // 已取消=终态不覆盖
	case row.Status == domain.ReqStatusDone && targetStatus != domain.ReqStatusDone:
		return nil // 已完成预置不降级（CR 复审边界口径，2026-09-10）
	case row.Status == targetStatus:
		return nil // 同值幂等不写
	}
	if err := tx.WithContext(ctx).Model(&repo.Requirement{}).
		Where("requirement_id = ?", row.RequirementID).
		Update("status", targetStatus).Error; err != nil {
		return errs.Trace(err)
	}
	return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
		Table:    "requirements",
		RowID:    row.RequirementID,
		Action:   "UPDATE",
		Field:    utils.StrPtr("status"),
		OldValue: utils.StrPtr(row.Status),
		NewValue: utils.StrPtr(targetStatus),
		ByWho:    byWho,
	})
}

// execAdvanceReady exec 相位推进判据：exec 族（produce/dev/accept）任务已有完成
// 且无在途；否则 rule4 空转兜底——相位无挂载能力动作且无期望产物 → 视为通过直接
// 轮转（capability-model-final §6 rule4，collect.go 空转语义同源）。
// 出参 ready=false 时携带判据缺项（HTTP 出参 missing）。
func (s *AdvanceService) execAdvanceReady(ctx context.Context, tx *gormio.DB, defNode *repo.FlowDefNode, node *repo.FlowNode) (bool, []string, error) {
	tasks, err := repo.ListTasksByRequirementAndNodeTx(ctx, tx, node.RequirementID, node.NodeID)
	if err != nil {
		return false, nil, err
	}
	completed, inflight := 0, 0
	for _, task := range tasks {
		if !domain.IsExecAdvanceFamily(task.TaskType) {
			continue
		}
		switch {
		case task.Status == repo.TaskStatusCompleted:
			completed++
		case !isTerminalTask(task):
			inflight++
		}
	}
	if completed > 0 && inflight == 0 {
		return true, nil, nil
	}
	mounted, err := execActionsMounted(defNode.NodeActions)
	if err != nil {
		return false, nil, err
	}
	expectedEmpty, err := expectedOutputsEmpty(defNode.ExpectedOutputs)
	if err != nil {
		return false, nil, err
	}
	if !mounted && expectedEmpty {
		// rule4 空转：exec 相位无挂载动作 → 视为通过直接轮转
		return true, nil, nil
	}
	missing := make([]string, 0, 1)
	if inflight > 0 {
		missing = append(missing, "exec_task:节点存在在途 exec 任务（须先完成）")
	} else {
		missing = append(missing, "exec_task:节点无已完成 exec 任务（exec 判据未通过）")
	}
	return false, missing, nil
}

// SweepParallelJoins 并行组兜底巡检（注记-C③，scanner 每轮消费）：逐编排中流程
// （current_node_id 非空）对其 current 节点走同一推进核（同 flows 行锁 + 审计
// by_who=system）补推——「双侧均 complete 但组未推进」的漏推自愈；非卡点流程
// （exec 判据未过/评审未齐等 blocked 与幂等 noop）跳过不动。
//
// 出参：本轮发生推进的需求 id 集（巡检报告载体）。
//
// 错误集：底层 IO/JSON 错误经 Trace 上抛（单流程失败整体上抛，由扫描循环容错）。
func (s *AdvanceService) SweepParallelJoins(ctx context.Context) ([]string, error) {
	flows, err := s.repo.ListFlowsInOrchestration(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}
	pushed := make([]string, 0)
	for _, flow := range flows {
		swept := ""
		txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
			locked, err := repo.GetFlowByRequirementIDLockTx(ctx, tx, flow.RequirementID)
			if err != nil || locked == nil {
				return err
			}
			nodes, err := repo.ListFlowNodesByRequirementLockTx(ctx, tx, locked.RequirementID)
			if err != nil {
				return err
			}
			// 打回重定窗口守卫（2026-09-10）：任一节点持有 pending_scope（rejected 回退
			// 已落、reproject 未做）时巡检跳过——窗口内推进会让尚未重投影的下游节点按
			// 裁剪前评审位激活播种，与随后的重投影裁剪竞速留下幽灵评审位。
			for _, n := range nodes {
				if n.PendingScope != nil {
					return nil
				}
			}
			res, err := s.advanceNodeLocked(ctx, tx, locked, nodes, derefString(locked.CurrentNodeID), auditByWhoSystem)
			if err != nil {
				// 非并行组卡点（exec 判据未过/相位不符/待拍板等）：巡检跳过不视为失败
				if errs.Is(err, errs.ErrGateBlocked) || errs.Is(err, errs.ErrNotFound) {
					return nil
				}
				return err
			}
			if res.Outcome == AdvanceOutcomeCross {
				swept = locked.RequirementID
			}
			return nil
		})
		if txErr != nil {
			return nil, errs.Trace(txErr)
		}
		if swept != "" {
			pushed = append(pushed, swept)
		}
	}
	return pushed, nil
}

// isRdSideExecCR 并行组 rd 侧成员判定（flow_def_nodes 配置口径）：produce 型 +
// gate=1 + 有评审位声明（内嵌 CR）+ scope 表达式命中 rd 不命中 fe（单泳道侧）。
// 业务流程 full 场景 bus-08（RD 开发）即该侧；设计/评审类节点（bus-05）与无 CR
// 的产出节点（bus-04/bus-06）不成组——RD∥FE 并行仅发生在开发阶段。
func isRdSideExecCR(defNode *repo.FlowDefNode) bool {
	return isLaneSideExecCR(defNode, repo.ScopeRD)
}

// isFeSideExecCR 并行组 fe 侧成员判定（对称口径，bus-09 即该侧）。
func isFeSideExecCR(defNode *repo.FlowDefNode) bool {
	return isLaneSideExecCR(defNode, repo.ScopeFE)
}

// isLaneSideExecCR 泳道侧 exec-CR 成员判定核。
func isLaneSideExecCR(defNode *repo.FlowDefNode, lane string) bool {
	if defNode == nil {
		return false
	}
	if defNode.NodeType != repo.NodeTypeProduce || defNode.Gate != 1 {
		return false
	}
	other := repo.ScopeRD
	if lane == repo.ScopeRD {
		other = repo.ScopeFE
	}
	if !domain.ScopeMatches(defNode.ScopeExpr, lane) || domain.ScopeMatches(defNode.ScopeExpr, other) {
		return false
	}
	reviewers, err := domain.ParseReviewers(defNode.Reviewers)
	if err != nil || len(reviewers) == 0 {
		// 解析失败 fail-closed：不成组（并行判定宁缺勿滥，组语义由 seed 保证）
		return false
	}
	return true
}

// peerApproved 对侧 complete 判定：last_decision verdict=approved（exec-CR 节点
// CR 门禁落定即开发完成；打回重置后 last_decision 清空回到未完成）。
func peerApproved(peer *repo.FlowNode) bool {
	if peer == nil {
		return false
	}
	decision, err := domain.ParseGateDecision(derefString(peer.LastDecision))
	if err != nil {
		return false
	}
	return decision.Verdict == domain.GateVerdictApproved
}

// execActionsMounted exec 相位挂载判定：激活清单存在 kind=capability 条目
// （能力动作只挂 exec/review/respond，kind=node 为交互动作不参与本判定）。
//
// 错误集：node_actions JSON 非法（实例数据损坏）。
func execActionsMounted(nodeActionsJSON string) (bool, error) {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(nodeActionsJSON, &entries); err != nil {
		return false, errs.Trace(err)
	}
	for _, entry := range entries {
		if entry.Kind == repo.NodeActionKindCapability {
			return true, nil
		}
	}
	return false, nil
}

// expectedOutputsEmpty 期望产物集空判定（rule4 空转联合条件）。
//
// 错误集：expected_outputs JSON 非法（实例数据损坏）。
func expectedOutputsEmpty(expectedOutputsJSON string) (bool, error) {
	var expected []repo.ExpectedOutput
	if err := utils.UnmarshalJSONText(expectedOutputsJSON, &expected); err != nil {
		return false, errs.Trace(err)
	}
	return len(expected) == 0, nil
}

// enteredAtText 节点 entered_at 审计旧值文本（NULL=空串）。
func enteredAtText(node *repo.FlowNode) string {
	if node.EnteredAt == nil {
		return ""
	}
	return formatMillis(*node.EnteredAt)
}

// flowNodeIndex 锁定清单中按实例节点 id 定位下标；未命中 -1。
func flowNodeIndex(nodes []*repo.FlowNode, nodeID string) int {
	for i, n := range nodes {
		if n.NodeID == nodeID {
			return i
		}
	}
	return -1
}

// applyNodeUpdatesTx 节点行乐观锁更新 + 逐字段审计（推进/表态/拍板/回退写路径
// 共用；基线=锁定行读取值；审计失败回滚业务事务——留痕一致口径）。
func applyNodeUpdatesTx(ctx context.Context, r *repo.Repo, tx *gormio.DB, node *repo.FlowNode, updates map[string]any, audits []repo.AuditLog) error {
	if len(updates) > 0 {
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.FlowNode{}, node.ID, node.UpdatedAt, updates); err != nil {
			return err
		}
	}
	for i := range audits {
		if err := r.InsertAuditLogWith(tx, &audits[i]); err != nil {
			return err
		}
	}
	return nil
}
