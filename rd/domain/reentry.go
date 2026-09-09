// Package domain 本文件：打回链路纯逻辑（T9）——rejected 回退的重入状态机投影：
// reopen_source 词表、回退目标选取、回退幂等护栏、exec 首产/返工（revise）配置选择。
// 无 IO，可单测。
//
// 定稿出处（docs-tmp/capability-model-final.md + docs-tmp/data-tools-final.md）：
//   - §1.4 回退与重入：reject（裁决事件）→ plugin 回退被重做节点 → 目标节点 exec
//     重入（diagnosing-bugs/revise）→ 完成上报 → 重新提评 → 评审相位重开；
//     重入识别=rolled_back_count（0=首次，≥1=重入——entry/exec 多配置按计数选择）；
//   - reopen_source 三值（data-tools-final §1.2 flow_nodes 列注释）：
//     downstream-reject=被下游评审打回（回退目标）/ upstream-reject=上游返工波及
//     （下游已激活节点联动重置）/ rework-return=节点自身返工回收（预留词表，
//     本链路 v1 未触发——节点内 CR 异化永不 rejected）；
//   - §5.1 定稿口径：exec 首次=主产出（produce/dev/accept），重入=返工
//     （revise/diagnosing-bugs/retest，QA 复验=retest）；
//   - collect 读取 round == rolled_back_count（data-tools-final §2 wb_review 行）：
//     回退幂等键 reqId+nodeId+round——回退以评审节点 rolled_back_count（打回次数）
//     为护栏，同轮重复触发 no-op；spec Further Notes「回退 reqId+nodeId+round」。
package domain

import (
	"github.com/huzilin/wbflow/rd/repo"
)

// reopen_source 重开来源三值（flow_nodes.reopen_source 列词表）。
const (
	// ReopenSourceDownstreamReject 被下游评审节点 rejected 打回（回退目标节点落值）。
	ReopenSourceDownstreamReject = "downstream-reject"
	// ReopenSourceUpstreamReject 上游节点返工波及（下游已渲染节点联动重置落值）。
	ReopenSourceUpstreamReject = "upstream-reject"
	// ReopenSourceReworkReturn 节点自身返工回收（预留词表；v1 打回链路未触发）。
	ReopenSourceReworkReturn = "rework-return"
)

// exec 首产/返工动作词表（capability-model-final §5.1 定稿；两集合互斥补集——
// 两集合之外的 exec 动作（如 submit 登记请评）不参与计数选择，恒随相位激活）。
var (
	// execFirstActions 首产动作：rolled_back_count=0 时激活。
	execFirstActions = map[string]bool{
		"produce": true,
		"dev":     true,
		"accept":  true,
	}
	// execReentryActions 返工动作：rolled_back_count≥1 时激活（revise=文档修订 /
	// diagnosing-bugs=缺陷定位修复 / retest=QA 复验重测）。
	execReentryActions = map[string]bool{
		"revise":          true,
		"diagnosing-bugs": true,
		"retest":          true,
	}
)

// FilterExecActionsByReentry exec 相位首产/返工配置选择（T9 重入投影）：
// 按 rolled_back_count（0=首次 / ≥1=重入）从已激活 exec 动作集中二选一——
// 首次保留首产集、剔除返工集；重入反之。非 exec 相位或计数为负原样返回；
// 两集合外的动作（如 submit）不参与选择。
func FilterExecActionsByReentry(actions []ActiveAction, rolledBackCount int64) []ActiveAction {
	if rolledBackCount < 0 {
		rolledBackCount = 0
	}
	reentry := rolledBackCount >= 1
	kept := make([]ActiveAction, 0, len(actions))
	for _, action := range actions {
		if execFirstActions[action.ActionID] {
			if reentry {
				continue
			}
		} else if execReentryActions[action.ActionID] {
			if !reentry {
				continue
			}
		}
		kept = append(kept, action)
	}
	return kept
}

// RollbackTarget 回退目标选取：被 rejected 评审节点渲染序（node_order）的
// 上一渲染节点（跨节点评审=上一节点产物，capability-model-final §4.2；v1 口径
// ——「缺陷 assignee 映射等数据路由」按后续工单精化）。nodes 按 node_order 有序；
// 无上游渲染节点（首节点被拒）返回 nil。
func RollbackTarget(nodes []*repo.FlowNode, reviewNodeOrder int64) *repo.FlowNode {
	var target *repo.FlowNode
	for _, node := range nodes {
		if node.NodeOrder >= reviewNodeOrder {
			break
		}
		if target == nil || node.NodeOrder > target.NodeOrder {
			target = node
		}
	}
	return target
}

// RollbackNeeded 回退幂等护栏（回退幂等键 reqId+评审节点 nodeId+round）：
// round=被拒评审节点当前评审轮（CurrentRound）；评审节点 rolled_back_count
// （=已发出的打回次数）> round 说明本轮打回已应用（或更新轮已处理）——重放 no-op。
// clarify 轮内循环只升 round 不打回（rolled_back_count 不变），护栏不受影响。
func RollbackNeeded(reviewRolledBackCount int64, round int64) bool {
	return reviewRolledBackCount <= round
}

// NextRollbackCount 打回后评审节点 rolled_back_count 落值=当前轮+1：
// 与「collect 读取 round == rolled_back_count」对齐——重新提评播种的新评审轮
// 取该值，同轮重放护栏单调递增。
func NextRollbackCount(round int64) int64 {
	return round + 1
}
