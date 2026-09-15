// service 包本文件：T9 打回链路——门禁 rejected 的回退重入编排。
//
// 职责（spec §4.1 打回链路 AC + capability-model-final §1.4 回退与重入）：
//   - rejected（wb_decision 拍板落定 last_decision 后同事务消费）→ 回退目标节点
//     exec 重入：reopen_source=downstream-reject、rolled_back_count++、state=exec、
//     pending_scope 落待重定标记、自身门禁态清零（重新提评后重新 collect）；
//   - 下游已渲染节点联动重置：reopen_source=upstream-reject、rolled_back_count++
//     （QA 复验=retest 的多配置选择依据）、相位/门禁态清零等待重新激活；
//   - 评审节点（rejected 发出方）：rolled_back_count=round+1（打回计数，回退幂等
//     键 reqId+nodeId+round 的护栏）、相位/评审轮清零等待重新提评播种新轮；
//   - flows.current_node_id 回退到目标节点（current_node_id 回退语义）。
//
// 事务边界：回退为单事务（编排会话统一锁序 flows → flow_nodes，全部相关行
// SELECT … FOR UPDATE 行锁读后改写）——并发回退仅一次生效；审计逐字段留痕
// （编排迁移 by_who=plugin）。消息/通知派发移出事务——T11 收件箱接线时消费。
package service

import (
	"context"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// RollbackForReject rejected 回退重入入口（独立事务；wb_decision rejected 分支
// 已在拍板事务内调用 RollbackForRejectTx——本方法供带外重放与编排方调用，
// 幂等护栏同源：同轮重复触发 no-op 不改库）。
//
// 入参：requirementID/nodeID=被拒评审节点定位。
// 出参：rollback_node_id=回退目标节点 id；noop（重放）返回空串。
//
// 错误集：ErrNotFound（需求/节点未命中）；ErrGateBlocked（首渲染节点被拒——
// 无上游可回退）；ErrOptimisticLock；底层 IO/JSON 错误经 Trace 上抛。
func (s *ReviewService) RollbackForReject(ctx context.Context, requirementID string, nodeID string) (string, error) {
	var targetID string
	err := s.orch.run(ctx, requirementID, func(sess *orchSession) error {
		// 锁序：flows 行锁（run 会话）→ 节点行锁（C1 统一口径；历史实现先锁节点，
		// 与拍板注记-C 相反、靠时序侥幸未爆交叉死锁——已随迁移改序）
		node, err := sess.node(ctx, nodeID)
		if err != nil {
			return err
		}
		targetID, err = s.RollbackForRejectTx(ctx, sess, node, repo.NowMillis())
		return err
	})
	if err != nil {
		return "", err
	}
	return targetID, nil
}

// RollbackForRejectTx rejected 回退重入（须在编排会话事务内调用——flows 行锁已由
// 会话持有；wb_decision rejected 分支同事务消费——拍板与回退原子）。幂等护栏：
// 评审节点 rolled_back_count > 当前评审轮（CurrentRound）= 本轮打回已应用，no-op
// 返回空串。
//
// 出参：回退目标节点 id；no-op 为空串。
//
// 错误集：ErrGateBlocked（首渲染节点被拒/评审轮 JSON 非法）；ErrOptimisticLock；
// 底层 IO 错误经 Trace 上抛。
func (s *ReviewService) RollbackForRejectTx(ctx context.Context, sess *orchSession, reviewNode *repo.FlowNode, now int64) (string, error) {
	var entries []repo.ReviewResult
	if err := utils.UnmarshalJSONText(reviewNode.ReviewResults, &entries); err != nil {
		return "", errs.Trace(err)
	}
	round := domain.CurrentRound(entries)
	if !domain.RollbackNeeded(reviewNode.RolledBackCount, round) {
		return "", nil
	}

	// 锁定清单重读（拍板写 last_decision 已推进 updated_at，入参行基线陈旧——
	// 乐观锁基线以锁定读为准）
	nodes, err := sess.nodes(ctx)
	if err != nil {
		return "", err
	}
	// 评审节点基线取锁内清单行
	var lockedReview *repo.FlowNode
	for _, node := range nodes {
		if node.NodeID == reviewNode.NodeID {
			lockedReview = node
			break
		}
	}
	if lockedReview == nil {
		return "", gateBlocked("节点 %q 不在需求 %q 渲染集内", reviewNode.NodeID, reviewNode.RequirementID)
	}
	target := domain.RollbackTarget(nodes, reviewNode.NodeOrder)
	if target == nil {
		return "", gateBlocked("节点 %q 为首渲染节点，无上游可回退", reviewNode.NodeID)
	}

	// 评审节点：打回计数落值（重新提评播种新轮=该值）+ 相位/评审轮/门禁态清零
	// （已拒结论由审计回放：last_decision 置 NULL 走 Silent 写而不审计，新轮
	// 重新 collect 重新拍板）
	nextCount := domain.NextRollbackCount(round)
	if err := sess.apply(ctx, lockedReview, auditByWhoPlugin,
		repo.RolledBackCount(nextCount),
		repo.NodeState(repo.PhaseEmpty),
		repo.ReviewResults("[]"),
		repo.Silent(repo.LastDecisionNull()),
	); err != nil {
		return "", err
	}

	// 回退目标：exec 重入（revise/diagnosing-bugs 多配置按 rolled_back_count 选择）+
	// pending_scope 待重定标记（当前 scope，重定后经 reproject 重投影清标记——标记
	// 落值本身不审计，清标记时由 reproject 审计）
	if err := sess.apply(ctx, target, auditByWhoPlugin,
		repo.ReopenSource(domain.ReopenSourceDownstreamReject),
		repo.RolledBackCount(target.RolledBackCount+1),
		repo.NodeState(repo.PhaseExec),
		repo.EnteredAt(now),
		repo.ReviewResults("[]"),
		repo.Silent(repo.LastDecisionNull()),
		repo.Silent(repo.PendingScope(sess.flow().Scope)),
	); err != nil {
		return "", err
	}

	// 下游已渲染节点联动重置（排除评审节点自身——已单独处理）：上游打回波及，
	// rolled_back_count++（QA retest 等重入配置选择依据），相位/门禁态清零等重新激活
	for _, node := range nodes {
		if node.NodeOrder <= target.NodeOrder || node.NodeID == reviewNode.NodeID {
			continue
		}
		if err := sess.apply(ctx, node, auditByWhoPlugin,
			repo.ReopenSource(domain.ReopenSourceUpstreamReject),
			repo.RolledBackCount(node.RolledBackCount+1),
			repo.NodeState(repo.PhaseEmpty),
			repo.EnteredAtNull(),
			repo.ReviewResults("[]"),
			repo.Silent(repo.LastDecisionNull()),
		); err != nil {
			return "", err
		}
	}

	// flows.current_node_id 回退到目标节点（乐观锁 + 审计；回退幂等护栏下不会
	// 已驻留目标，moved=false 仅防御）
	if _, err := sess.moveCurrent(ctx, target, auditByWhoPlugin); err != nil {
		return "", err
	}
	return target.NodeID, nil
}
