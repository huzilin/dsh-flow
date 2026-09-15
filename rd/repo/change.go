// repo 包本文件：flow_nodes 变更同源派生（架构评审 C2 深化，提案
// .plan/proposals/node-change-audit-plan.md）——写路径声明一次变更（类型化构造器，
// 声明序=审计行序），updates map 与 AuditLog 行由 implementation 同源派生：
// 审计旧值取自锁定行基线、列名↔审计字段名映射与审计文本口径内建，updates 与
// 审计两份口径永不漂移。
//
// 口径（与既有手拼写路径行为逐点一致）：
//   - updates：全部变更无条件写入（乐观锁基线=锁定行读取值 updated_at）；
//   - 审计：默认仅 旧值文本≠新值文本 时落行——同值写不重放留痕（替代原各写路径
//     手写的 if old != new 条件）；
//   - Silent：写而不审计（回退核清零门禁态等「结论已由前序动作审计回放」口径，
//     如 rejected 回退将 last_decision 置 NULL 不再重放被拒结论）；
//   - Who：单条审计署名覆盖（wb_review 单次行写混两署名：表态审计=表态角色、
//     相位/裁决迁移审计=plugin——默认署名不足用）。
package repo

import (
	"context"

	"gorm.io/gorm"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
)

// nodeField 变更字段定位（构造器内建映射，调用方不触列名与审计字段名）。
type nodeField int

const (
	fieldState nodeField = iota
	fieldEnteredAt
	fieldReviewResults
	fieldRolledBackCount
	fieldReopenSource
	fieldLastDecision
	fieldPendingScope
	fieldReviewers
	fieldNodeOrder
	fieldFeatureBranch
)

// nodeFieldMeta 列名↔审计字段名同源映射（新增变更字段只改本表与对应构造器）。
var nodeFieldMeta = map[nodeField]struct {
	column     string
	auditField string
}{
	fieldState:           {"state", "state"},
	fieldEnteredAt:       {"entered_at", "enteredAt"},
	fieldReviewResults:   {"review_results", "reviewResults"},
	fieldRolledBackCount: {"rolled_back_count", "rolledBackCount"},
	fieldReopenSource:    {"reopen_source", "reopenSource"},
	fieldLastDecision:    {"last_decision", "lastDecision"},
	fieldPendingScope:    {"pending_scope", "pendingScope"},
	fieldReviewers:       {"reviewers", "reviewers"},
	fieldNodeOrder:       {"node_order", "nodeOrder"},
	fieldFeatureBranch:   {"feature_branch", "featureBranch"},
}

// NodeChange flow_nodes 单字段变更声明（构造器产出；Silent/Who 修饰后仍为本类型）。
type NodeChange struct {
	field   nodeField
	value   any    // updates 写值（string/int64；NULL 变体=nil）
	newText string // 审计新值文本（NULL 变体=空串）
	silent  bool   // 写而不审计
	byWho   string // 非空=该条审计署名覆盖
}

// 变更构造器（声明序即审计行序；值口径与列写值同源——Null 变体列写 NULL、审计新值为空串）。

// NodeState 相位变更（entry/exec/review/respond/""=未进入编排）。
func NodeState(v string) NodeChange { return textChange(fieldState, v) }

// EnteredAt 进入时刻（unix 毫秒）。
func EnteredAt(v int64) NodeChange { return countChange(fieldEnteredAt, v) }

// EnteredAtNull 清进入时刻（列写 NULL）。
func EnteredAtNull() NodeChange { return nullChange(fieldEnteredAt) }

// ReviewResults 评审轮结果 JSON 文本。
func ReviewResults(v string) NodeChange { return textChange(fieldReviewResults, v) }

// RolledBackCount 回退计数。
func RolledBackCount(v int64) NodeChange { return countChange(fieldRolledBackCount, v) }

// ReopenSource 重开来源。
func ReopenSource(v string) NodeChange { return textChange(fieldReopenSource, v) }

// LastDecision 门禁裁决 JSON 文本。
func LastDecision(v string) NodeChange { return textChange(fieldLastDecision, v) }

// LastDecisionNull 清门禁裁决（列写 NULL）。
func LastDecisionNull() NodeChange { return nullChange(fieldLastDecision) }

// PendingScope 打回待重定范围标记。
func PendingScope(v string) NodeChange { return textChange(fieldPendingScope, v) }

// PendingScopeNull 清待重定标记（列写 NULL）。
func PendingScopeNull() NodeChange { return nullChange(fieldPendingScope) }

// Reviewers 评审位声明 JSON 文本。
func Reviewers(v string) NodeChange { return textChange(fieldReviewers, v) }

// NodeOrder 渲染序号（重投影重排）。
func NodeOrder(v int64) NodeChange { return countChange(fieldNodeOrder, v) }

// FeatureBranch 评审敲定分支名。
func FeatureBranch(v string) NodeChange { return textChange(fieldFeatureBranch, v) }

// Silent 该变更写而不审计（不重放留痕口径：被清/被落的结论已由前序动作审计）。
func Silent(c NodeChange) NodeChange { c.silent = true; return c }

// Who 单条审计署名覆盖（默认署名=ApplyNodeChangeTx 的 byWho）。
func Who(byWho string, c NodeChange) NodeChange { c.byWho = byWho; return c }

// textChange 文本列变更构造核。
func textChange(f nodeField, v string) NodeChange {
	return NodeChange{field: f, value: v, newText: v}
}

// countChange 整型列变更构造核（审计文本=十进制）。
func countChange(f nodeField, v int64) NodeChange {
	return NodeChange{field: f, value: v, newText: utils.FormatMillis(v)}
}

// nullChange NULL 变更构造核（列写 NULL，审计新值=空串）。
func nullChange(f nodeField) NodeChange {
	return NodeChange{field: f, value: nil, newText: ""}
}

// ApplyNodeChangeTx flow_nodes 变更落库（乐观锁 + 同源派生审计；调用方持有行锁，
// 基线=锁定行读取值 updated_at——与原 applyNodeUpdatesTx 基线口径一致）。审计旧值
// 从锁定行提取，与 updates 同源；审计失败回滚业务事务（留痕一致口径）。
//
// 错误集：ErrOptimisticLock；空变更集/重复字段（装配缺陷，宁可失败暴露不静默）；
// 底层 IO 错误经 Trace 上抛。
func ApplyNodeChangeTx(ctx context.Context, r *Repo, tx *gorm.DB, node *FlowNode, byWho string, changes ...NodeChange) error {
	updates, audits, err := deriveNodeChange(node, byWho, changes...)
	if err != nil {
		return err
	}
	if err := UpdateWithOptimisticLockTx(ctx, tx, &FlowNode{}, node.ID, node.UpdatedAt, updates); err != nil {
		return err
	}
	for i := range audits {
		if err := r.InsertAuditLogWith(tx, &audits[i]); err != nil {
			return err
		}
	}
	return nil
}

// deriveNodeChange 派生核（纯函数，免库单测面）：变更声明 → updates map + 审计行。
// 审计默认仅 旧值文本≠新值文本 时产出（silent 恒不产出；byWho 覆盖在单条上生效）。
func deriveNodeChange(node *FlowNode, byWho string, changes ...NodeChange) (map[string]any, []AuditLog, error) {
	if len(changes) == 0 {
		return nil, nil, errs.Trace(perr.Errorf("节点 %q 变更集为空（装配缺陷）", node.NodeID))
	}
	updates := make(map[string]any, len(changes)+1)
	seen := make(map[nodeField]bool, len(changes))
	audits := make([]AuditLog, 0, len(changes))
	for _, c := range changes {
		if seen[c.field] {
			return nil, nil, errs.Trace(perr.Errorf("节点 %q 字段 %v 重复声明（装配缺陷）", node.NodeID, c.field))
		}
		seen[c.field] = true
		meta := nodeFieldMeta[c.field]
		updates[meta.column] = c.value
		if c.silent {
			continue
		}
		oldText := nodeOldText(node, c.field)
		if oldText == c.newText {
			continue
		}
		who := byWho
		if c.byWho != "" {
			who = c.byWho
		}
		audits = append(audits, AuditLog{
			Table:    "flow_nodes",
			RowID:    node.NodeID,
			Action:   "UPDATE",
			Field:    utils.StrPtr(meta.auditField),
			OldValue: utils.StrPtr(oldText),
			NewValue: utils.StrPtr(c.newText),
			ByWho:    who,
		})
	}
	return updates, audits, nil
}

// nodeOldText 锁定行基线的审计旧值文本（口径：NULL/nil→空串，整型→十进制文本；
// 与原手拼口径逐点一致）。
func nodeOldText(node *FlowNode, f nodeField) string {
	switch f {
	case fieldState:
		return node.State
	case fieldEnteredAt:
		return utils.FormatMillisPtr(node.EnteredAt)
	case fieldReviewResults:
		return node.ReviewResults
	case fieldRolledBackCount:
		return utils.FormatMillis(node.RolledBackCount)
	case fieldReopenSource:
		return node.ReopenSource
	case fieldLastDecision:
		return utils.DerefStr(node.LastDecision)
	case fieldPendingScope:
		return utils.DerefStr(node.PendingScope)
	case fieldReviewers:
		return node.Reviewers
	case fieldNodeOrder:
		return utils.FormatMillis(node.NodeOrder)
	case fieldFeatureBranch:
		return utils.DerefStr(node.FeatureBranch)
	}
	return ""
}
