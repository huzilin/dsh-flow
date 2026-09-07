// Package domain 本文件：T12 冲突上升裁决链纯逻辑——仲裁结论词表、上升/裁决
// replies 条目装配与定位、notification 通知文案装配（spec §2 澄清不收敛行 +
// §4.1 冲突上升 AC + ui-interaction-final §1.4 冲突卡 + capability-model-final
// §6 rule6 冲突响应机制）。全部无 IO，可表驱动单测。
//
// 裁决链状态机（escalate 生命周期）：
//
//	escalate=0 →（手动 +escalate / wb_escalate / 自动：reopen 达阈值）→ escalate=1
//	→ wb_escalate 拉起裁决 session（sessions 建绑，role=user 仲裁语义）
//	→ wb_adjudicate User 出结论回填 issue → escalate=0 复位
//	→ reviewer 重表态（评审轮重入新轮）→ 正常判据链恢复
//
// escalate 复位口径：escalate 语义=「待 User 仲裁」挂起态（spec §0 列注释
// 「1=拉起裁决 session」），非历史徽标——裁决回填即挂起解除，复位 0 使
// 冲突卡聚合（ui-interaction-final §1.4）只呈现进行中冲突、且二次拉锯可
// 重新触发自动上升；裁决结论经 replies 裁决条目 + resolve 族留痕，不复用
// escalate 列承载历史。
package domain

import (
	"fmt"

	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// wb_resolve path=fix|agree 之外的第三路径（T12 仲裁裁定关闭）：仅裁决链
// （wb_adjudicate verdict=override）可落，wb_resolve 入参校验仍拒之门外
// （issues.resolve_path 列注释「fix=修复关闭 / agree=协商一致关闭」的仲裁扩展）。
const IssueResolvePathRuling = "ruling"

// 仲裁结论词表（wb_adjudicate verdict；与既有 approved/rejected 拍板词表区分
// ——拍板是节点级门禁结论，仲裁是 issue 级冲突结论）：
const (
	// AdjudicationVerdictUphold 维持议题：提评者立场成立，issue 回正常澄清/修复链
	// （状态保持不变），escalate 复位。
	AdjudicationVerdictUphold = "uphold"
	// AdjudicationVerdictOverride 仲裁推翻：wb_override 语义收编（spec §1.2 移出
	// 清单「wb_override（escalate 取代）——语义保留在 service 层」），issue 落
	// overridden 终态（resolve_path=ruling + resolve_note 引用结论）。
	AdjudicationVerdictOverride = "override"
)

// ValidAdjudicationVerdict 仲裁结论枚举校验。
func ValidAdjudicationVerdict(verdict string) bool {
	switch verdict {
	case AdjudicationVerdictUphold, AdjudicationVerdictOverride:
		return true
	}
	return false
}

// IssueReply 条目 kind 词表（repo.IssueReply.Kind；空=普通回复）。
const (
	// ReplyKindEscalation 冲突上升条目：wb_escalate 拉起裁决 session 时追加，
	// 携带 sessionId（issue↔session 关联单一载体——sessions 表无 issue 外键列）。
	ReplyKindEscalation = "escalation"
	// ReplyKindAdjudication User 仲裁裁决条目：wb_adjudicate 回填时追加
	// （capability-model-final §6 rule6「结论追加到冲突 issue，数据层留痕、node 可见」）。
	ReplyKindAdjudication = "adjudication"
)

// AppendIssueReply replies 不可变追加装配（裁决链通道与 wb_reply 同构：只增不改，
// 线程内序 r-<n> 续号）。
//
// 返回：追加后的 replies JSON 文本与本条目分配到的 replyID。
//
// 错误集：既有 replies JSON 非法（实例数据损坏）。
func AppendIssueReply(repliesJSON string, entry repo.IssueReply) (string, string, error) {
	var replies []repo.IssueReply
	if err := utils.UnmarshalJSONText(repliesJSON, &replies); err != nil {
		return "", "", err
	}
	entry.ReplyID = fmt.Sprintf("r-%d", len(replies)+1)
	replies = append(replies, entry)
	text, err := utils.MarshalJSONText(replies)
	if err != nil {
		return "", "", err
	}
	return text, entry.ReplyID, nil
}

// LatestReplyOfKind 取 replies 中指定 kind 的最新条目（未命中返回 ok=false）：
// wb_escalate 幂等判定与 wb_adjudicate session 定位数据源（不可变追加语义下
// 末条即最新）。
func LatestReplyOfKind(repliesJSON string, kind string) (repo.IssueReply, bool, error) {
	var replies []repo.IssueReply
	if err := utils.UnmarshalJSONText(repliesJSON, &replies); err != nil {
		return repo.IssueReply{}, false, err
	}
	for i := len(replies) - 1; i >= 0; i-- {
		if replies[i].Kind == kind {
			return replies[i], true, nil
		}
	}
	return repo.IssueReply{}, false, nil
}

// EscalationTriggerAuto / EscalationTriggerManual 通知文案触发来源词表
// （issues notification 行 text 装配口径，T11 收件箱读侧直接呈现）。
const (
	EscalationTriggerAuto   = "auto"
	EscalationTriggerManual = "manual"
)

// BuildEscalationNoticeText 通知位文案装配（issue_type=notification 行 text）：
// 收件箱七类通知的冲突上升条目（spec §1.2 通知枚举 v3 消费侧为 T11，本工单
// 落通知位行）。triggerText 描述触发来源（如「reopen 达阈值 5，自动上升」/
// 「operator=arch 手动上升」）。
func BuildEscalationNoticeText(sourceIssueID string, trigger string) string {
	return fmt.Sprintf("[冲突上升待仲裁] issue %s 待 User 裁决（%s），裁决 session 已就绪", sourceIssueID, trigger)
}

// AutoEscalateTriggerText 自动上升触发文案（reopen 达阈值注入当前阈值）。
func AutoEscalateTriggerText(threshold int64) string {
	return fmt.Sprintf("reopen 达阈值 %d，自动上升", threshold)
}

// ManualEscalateTriggerText 手动上升触发文案。
func ManualEscalateTriggerText(operator string) string {
	return fmt.Sprintf("operator=%s 手动上升", operator)
}

// EscalationReplyText 上升条目 text 装配（wb_escalate reason 空=默认文案）。
func EscalationReplyText(reason string) string {
	if reason != "" {
		return reason
	}
	return "冲突不收敛，上升 User 仲裁"
}
