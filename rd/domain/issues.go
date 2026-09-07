// Package domain 本文件：issue 状态机 v2 纯逻辑（spec §0 表 #9 列注释 + §1.3
// 「domain 状态机与纯逻辑」）——合法迁移表、操作者角色约束、锚点四级一致性校验、
// loop_count 双读法副作用、escalate 置位、reviewer_role/assignee 推导、锚点摘要。
// 全部无 IO，可表驱动单测。
//
// 状态机 v2（spec §0 issues.state 与表注释，CONTEXT.md「已处理（pending-confirm）」
// 「已解决（resolved）」措辞）：
//
//	open（待澄清）→ pending-confirm（已澄清待复核）→ resolved | to-fix | open（澄清 reopen，loop_count++）
//	to-fix（待修复）→ fixed（已修复待复核）→ 复核 → resolved | to-fix（修复 revise，loop_count++）
//
// 迁移与端点映射（五端点，spec §1.2 路由表 comments 五行）：
//   - wb_reply：open→pending-confirm（CONTEXT「已处理」=「评论被提交者修订/回复完成、
//     待评审者复核的状态」）；to-fix→fixed（修复完成回复即申报，状态机 to-fix→fixed）；
//     pending-confirm/fixed 追加回复不回退状态（多轮澄清/修复说明，状态幂等）；
//     resolved/overridden 拒绝（评论不可编辑/删除、终态不可追加——ui-interaction-final §3）。
//   - wb_resolve path=fix（修复路径）：open/pending-confirm→to-fix（定性需修复进入修复期，
//     状态机 pending-confirm→to-fix）；to-fix/fixed→fixed（修复关闭申报待复核——
//     arch_design §Issue 示例：resolve_path=fix 且 state 停留待复核、resolved_by=NULL 未闭环）。
//     resolve 族列落值（resolve_by/at/note/path）。
//   - wb_resolve path=agree（协商一致关闭）：open/pending-confirm/to-fix/fixed→resolved；
//     协商一致即双方确认（CONTEXT「已解决：评审者确认接受处理后的评论终态」——agree
//     路径的确认内嵌于协商一致，无独立复核步），resolve 族 + resolved_by/at +
//     resolve_reviewer_confirmed=1 同步落值。
//   - wb_confirm_review（提评者复核确认）：pending-confirm→resolved / fixed→resolved
//     （状态机「复核」侧；resolve→confirm 双路径的确认侧，CONTEXT.md）。
//     resolve_reviewer_confirmed=1 + resolved_by/at。
//   - wb_reopen：pending-confirm→open（澄清 reopen，loop_count++）；fixed→to-fix
//     （修复 revise，loop_count++）——loop_count 双读法（spec 列注释：澄清期读=reopen
//     次数，修复期读=revise 次数）。
//
// 操作者角色约束（本文件 ValidateIssueActor，依据列各分支 GoDoc）：
//   - resolve：操作者=assignee（arch_design §agent 交互模式「缺陷闭环」：
//     wb_comment 自动指派 → **assignee resolve** → QA confirm/reopen）；
//     assignee 未指派=无从校验，拒绝（须先经评论指派）。
//   - confirm/reopen：操作者=提评者，即根评论 author；reviewer_role 已推导时以
//     reviewer_role 为准（arch_design §评审者 5 步：评论 → 表态 → 复核（confirm/reopen）；
//     ui-interaction-final §1.4 复核由评审者执行）。
//   - reply：对话双方（author 侧/assignee 侧）不设限——replies 不可变追加语义
//     （arch_design §Issue「误发用 reply 标注作废」）。
package domain

import (
	"strconv"

	"fmt"

	perr "github.com/pingcap/errors"

	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// issue 状态机 v2 状态（issues.state，spec §0 表 #9 列注释）。
const (
	IssueStateOpen           = "open"
	IssueStatePendingConfirm = "pending-confirm"
	IssueStateToFix          = "to-fix"
	IssueStateFixed          = "fixed"
	IssueStateResolved       = "resolved"
	IssueStateOverridden     = "overridden"
)

// issue 五端点动作（wb_reply / wb_resolve / wb_confirm_review / wb_reopen；
// wb_comment 是建行不是迁移）。
const (
	IssueActionReply   = "reply"
	IssueActionResolve = "resolve"
	IssueActionConfirm = "confirm"
	IssueActionReopen  = "reopen"
)

// resolve 双路径（issues.resolve_path 列注释：fix=修复关闭 / agree=协商一致关闭）。
const (
	IssueResolvePathFix   = "fix"
	IssueResolvePathAgree = "agree"
)

// 锚点四级（issues.anchor_kind 列注释）。
const (
	IssueAnchorDocument = "document"
	IssueAnchorLine     = "line"
	IssueAnchorRange    = "range"
	IssueAnchorBlock    = "block"
)

// issue_type 三值（issues.issue_type 列注释）。
const (
	IssueTypeIssue        = "issue"
	IssueTypeNotification = "notification"
	IssueTypeDiscussion   = "discussion"
)

// DefaultEscalateLoopThreshold 澄清不收敛自动上升阈值（spec §2：loop_count 达阈值
// 如 reopen≥5 自动标 escalate；T12 裁决链消费通知，置位本身在本工单落地）。
const DefaultEscalateLoopThreshold = 5

// IssueTransitionKey 迁移表键：动作 × 路径 × 当前态。
type IssueTransitionKey struct {
	// Action 五端点动作（IssueAction*）
	Action string
	// ResolvePath resolve 路径（仅 Action=resolve 消费；fix/agree）
	ResolvePath string
	// From 当前状态
	From string
}

// IssueEffect 迁移副作用描述（service 层据此组装 UPDATE 列与审计）。
type IssueEffect struct {
	// NextState 迁移后状态
	NextState string
	// RoundCountDelta 对话轮数增量（reply=1，其余 0）
	RoundCountDelta int64
	// LoopCountDelta 回退计数增量（reopen=1：澄清 reopen / 修复 revise 双读法共用列，其余 0）
	LoopCountDelta int64
	// AppendReply 回复追加（reply 恒 true；reopen 携带理由时也追加）
	AppendReply bool
	// TouchLastReplyAt 是否刷新 last_reply_at（AppendReply 且非 reopen 理由回填时）
	TouchLastReplyAt bool
	// SetResolveFamily resolve 族列落值（resolve_by/at/note/path）
	SetResolveFamily bool
	// ConfirmClose 复核确认闭环：resolve_reviewer_confirmed=1 + resolved_by/at
	ConfirmClose bool
}

// issueTransitions 合法迁移表（全量显式枚举，禁止 if-else 隐式分支）。
// 键缺失 = 非法迁移（ErrInvalidTransition）。
var issueTransitions = map[IssueTransitionKey]IssueEffect{
	// wb_reply：澄清回复完成 → 待复核；修复完成回复 → 已修复待复核；
	// 待复核态追加回复不回退状态（多轮澄清/修复说明）。
	{Action: IssueActionReply, From: IssueStateOpen}: {
		NextState: IssueStatePendingConfirm, RoundCountDelta: 1,
		AppendReply: true, TouchLastReplyAt: true,
	},
	{Action: IssueActionReply, From: IssueStateToFix}: {
		NextState: IssueStateFixed, RoundCountDelta: 1,
		AppendReply: true, TouchLastReplyAt: true,
	},
	{Action: IssueActionReply, From: IssueStatePendingConfirm}: {
		NextState: IssueStatePendingConfirm, RoundCountDelta: 1,
		AppendReply: true, TouchLastReplyAt: true,
	},
	{Action: IssueActionReply, From: IssueStateFixed}: {
		NextState: IssueStateFixed, RoundCountDelta: 1,
		AppendReply: true, TouchLastReplyAt: true,
	},
	// wb_resolve path=fix：定性需修复进入修复期 / 修复关闭申报（待复核态重申报幂等）。
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathFix, From: IssueStateOpen}: {
		NextState: IssueStateToFix, SetResolveFamily: true,
	},
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathFix, From: IssueStatePendingConfirm}: {
		NextState: IssueStateToFix, SetResolveFamily: true,
	},
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathFix, From: IssueStateToFix}: {
		NextState: IssueStateFixed, SetResolveFamily: true,
	},
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathFix, From: IssueStateFixed}: {
		NextState: IssueStateFixed, SetResolveFamily: true,
	},
	// wb_resolve path=agree：协商一致关闭（确认内嵌于协商一致）。
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathAgree, From: IssueStateOpen}: {
		NextState: IssueStateResolved, SetResolveFamily: true, ConfirmClose: true,
	},
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathAgree, From: IssueStatePendingConfirm}: {
		NextState: IssueStateResolved, SetResolveFamily: true, ConfirmClose: true,
	},
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathAgree, From: IssueStateToFix}: {
		NextState: IssueStateResolved, SetResolveFamily: true, ConfirmClose: true,
	},
	{Action: IssueActionResolve, ResolvePath: IssueResolvePathAgree, From: IssueStateFixed}: {
		NextState: IssueStateResolved, SetResolveFamily: true, ConfirmClose: true,
	},
	// wb_confirm_review：提评者复核确认（澄清侧 / 修复侧「复核」）。
	{Action: IssueActionConfirm, From: IssueStatePendingConfirm}: {
		NextState: IssueStateResolved, ConfirmClose: true,
	},
	{Action: IssueActionConfirm, From: IssueStateFixed}: {
		NextState: IssueStateResolved, ConfirmClose: true,
	},
	// wb_reopen：澄清 reopen（loop_count++）/ 修复 revise（loop_count++）——双读法共用列。
	{Action: IssueActionReopen, From: IssueStatePendingConfirm}: {
		NextState: IssueStateOpen, LoopCountDelta: 1,
		AppendReply: true,
	},
	{Action: IssueActionReopen, From: IssueStateFixed}: {
		NextState: IssueStateToFix, LoopCountDelta: 1,
		AppendReply: true,
	},
}

// IssueTransition 状态机判定：动作（resolve 附路径）× 当前态查合法迁移表。
// 命中返回副作用描述；未命中返回以 errs.ErrInvalidTransition 为 cause 的错误
// （handler 集中映射 409 INVALID_TRANSITION）。
//
// 错误集：errs.ErrInvalidTransition（非法迁移）；入参非法（resolve 缺 path / path 越集）。
func IssueTransition(action string, resolvePath string, from string) (IssueEffect, error) {
	if action == IssueActionResolve && resolvePath == "" {
		return IssueEffect{}, fmt.Errorf("resolve 缺少 path（fix|agree）")
	}
	effect, ok := issueTransitions[IssueTransitionKey{Action: action, ResolvePath: resolvePath, From: from}]
	if !ok {
		return IssueEffect{}, fmt.Errorf("%w: 状态 %s 不允许动作 %s%s",
			errs.ErrInvalidTransition, from, action, resolvePathSuffix(resolvePath))
	}
	return effect, nil
}

// resolvePathSuffix 错误信息后缀（resolve 附路径，其余动作为空）。
func resolvePathSuffix(path string) string {
	if path == "" {
		return ""
	}
	return "（path=" + path + "）"
}

// IssueView 状态机判定所需的 issue 最小视图（service 层从 repo.Issue 投影；
// domain 不 import repo.Issue 全列，保持判定输入显式）。
type IssueView struct {
	// State 当前状态
	State string
	// Author 根评论作者（提评者）
	Author string
	// ReviewerRole 评审归属角色；空=未推导/非评审
	ReviewerRole string
	// Assignee 解决责任人；空=未指派
	Assignee string
}

// ValidateIssueActor 操作者角色约束判定（谁可 resolve/confirm/reopen——依据见包注释）。
// reply 不设限（对话双方追加语义）。返回 nil=允许。
//
// 错误集：errs.ErrInvalidTransition（操作者越权：resolve 非 assignee /
// assignee 未指派；confirm/reopen 非提评者）。
func ValidateIssueActor(action string, operatorRole string, issue IssueView) error {
	switch action {
	case IssueActionResolve:
		if issue.Assignee == "" {
			return fmt.Errorf("%w: resolve 需先指派 assignee（当前未指派）", errs.ErrInvalidTransition)
		}
		if operatorRole != issue.Assignee {
			return fmt.Errorf("%w: resolve 操作者 %q 非 assignee %q", errs.ErrInvalidTransition, operatorRole, issue.Assignee)
		}
	case IssueActionConfirm, IssueActionReopen:
		allowed := issue.Author
		if issue.ReviewerRole != "" {
			allowed = issue.ReviewerRole
		}
		if operatorRole != allowed {
			return fmt.Errorf("%w: %s 操作者 %q 非提评者 %q", errs.ErrInvalidTransition, action, operatorRole, allowed)
		}
	}
	return nil
}

// ValidateIssueAnchor 锚点四级一致性校验（issues 锚点族列注释——kind 与字段一致）：
//   - document：仅 artifact 级之外的正文级，行号族必须全空（artifact_id 可空——
//     DDL 注释「anchor_artifact_id NULL=document 级」）；
//   - line：anchor_line 必填（1 起），其余行号族必须空，artifact_id 必填
//     （行级锚点防漂移按产物正文定位——ui-interaction-final §3）；
//   - range：start/end 必填且 start≤end，其余空，artifact_id 必填；
//   - block：block_start/block_end 必填且 start≤end，lang 必填，其余空，artifact_id 必填。
//
// 错误集：入参非法（kind 越集 / 字段缺失 / 冗余字段 / 范围倒挂）。
func ValidateIssueAnchor(kind string, artifactID string, line int64, startLine int64, endLine int64, blockStart int64, blockEnd int64, lang string) error {
	switch kind {
	case IssueAnchorDocument:
		if line != 0 || startLine != 0 || endLine != 0 || blockStart != 0 || blockEnd != 0 || lang != "" {
			return fmt.Errorf("锚点 kind=document 不允许携带行号/代码块字段")
		}
	case IssueAnchorLine:
		if artifactID == "" {
			return fmt.Errorf("锚点 kind=line 必填 artifact_id")
		}
		if line < 1 {
			return fmt.Errorf("锚点 kind=line 必填 line（1 起）")
		}
		if startLine != 0 || endLine != 0 || blockStart != 0 || blockEnd != 0 || lang != "" {
			return fmt.Errorf("锚点 kind=line 不允许携带范围/代码块字段")
		}
	case IssueAnchorRange:
		if artifactID == "" {
			return fmt.Errorf("锚点 kind=range 必填 artifact_id")
		}
		if startLine < 1 || endLine < 1 {
			return fmt.Errorf("锚点 kind=range 必填 start_line/end_line（1 起）")
		}
		if startLine > endLine {
			return fmt.Errorf("锚点 kind=range 范围倒挂（%d > %d）", startLine, endLine)
		}
		if line != 0 || blockStart != 0 || blockEnd != 0 || lang != "" {
			return fmt.Errorf("锚点 kind=range 不允许携带单行/代码块字段")
		}
	case IssueAnchorBlock:
		if artifactID == "" {
			return fmt.Errorf("锚点 kind=block 必填 artifact_id")
		}
		if blockStart < 1 || blockEnd < 1 {
			return fmt.Errorf("锚点 kind=block 必填 block_start/block_end（1 起）")
		}
		if blockStart > blockEnd {
			return fmt.Errorf("锚点 kind=block 范围倒挂（%d > %d）", blockStart, blockEnd)
		}
		if lang == "" {
			return fmt.Errorf("锚点 kind=block 必填 lang")
		}
		if line != 0 || startLine != 0 || endLine != 0 {
			return fmt.Errorf("锚点 kind=block 不允许携带单行/范围字段")
		}
	default:
		return fmt.Errorf("未知锚点 kind %q（document/line/range/block）", kind)
	}
	return nil
}

// ValidIssueType issue_type 枚举校验（空=默认 issue，合法）。
func ValidIssueType(issueType string) bool {
	switch issueType {
	case "", IssueTypeIssue, IssueTypeNotification, IssueTypeDiscussion:
		return true
	}
	return false
}

// ValidIssueState issue 状态枚举校验（含空=建行默认 open 前的零值）。
func ValidIssueState(state string) bool {
	switch state {
	case IssueStateOpen, IssueStatePendingConfirm, IssueStateToFix, IssueStateFixed, IssueStateResolved, IssueStateOverridden:
		return true
	}
	return false
}

// MarkEscalate escalate 置位（幂等恒 1；spec §0 escalate 列「1=拉起裁决 session」，
// T12 裁决链消费）。
func MarkEscalate() int64 {
	return 1
}

// AutoEscalateNeeded 澄清不收敛判定：loop_count 达阈值（spec §2「reopen≥5 自动标
// escalate」口径，阈值参数化由调用方注入）。
func AutoEscalateNeeded(loopCount int64, threshold int64) bool {
	return loopCount >= threshold && threshold > 0
}

// DeriveReviewerRole 评审归属角色推导：发表者角色出现在节点评审位（flow_nodes.reviewers
// 静态声明）→ 发表者角色（该 issue 评审归属=提评角色，复核 confirm/reopen 操作者）；
// 否则空串=未推导/非评审（issues.reviewer_role 列注释）。
//
// 错误集：reviewers JSON 非法（实例数据损坏）。
func DeriveReviewerRole(authorRole string, reviewersJSON string) (string, error) {
	var reviewers []repo.ReviewerDef
	if err := utils.UnmarshalJSONText(reviewersJSON, &reviewers); err != nil {
		return "", perr.Trace(err)
	}
	for _, reviewer := range reviewers {
		if reviewer.Role == authorRole {
			return authorRole, nil
		}
	}
	return "", nil
}

// DeriveAssignee 解决责任人推导（arch_design §8 问题归属三层 L1→L2，越前越优先）：
//   - explicit 显式指派（入参 assignee）恒优先；
//   - L1 锚点产物登记人（anchor_artifact_id → artifacts.added_by，命中才返回）；
//   - L2 节点产出角色（节点 exec 相位激活清单的能力归属角色——产出者即责任人）；
//   - 全部落空 → 空串=未指派（resolve 前须补指派）。
func DeriveAssignee(explicit string, anchorArtifactAddedBy string, nodeExecRole string) string {
	if explicit != "" {
		return explicit
	}
	if anchorArtifactAddedBy != "" {
		return anchorArtifactAddedBy
	}
	return nodeExecRole
}

// NodeExecRole 节点产出角色推导：node_actions（条目级 scope 裁剪后）kind=capability
// 条目激活的动作含 exec 相位 → 能力归属角色（首个命中）。无 exec 能力（review/gate
// 等纯评审节点）返回空串。
//
// 错误集：node_actions JSON 非法（实例数据损坏）。
func NodeExecRole(nodeActionsJSON string, capabilityRoles map[string]string) (string, error) {
	var entries []repo.NodeActionDef
	if err := utils.UnmarshalJSONText(nodeActionsJSON, &entries); err != nil {
		return "", perr.Trace(err)
	}
	for _, entry := range entries {
		if entry.Kind != repo.NodeActionKindCapability {
			continue
		}
		role, ok := capabilityRoles[entry.CapabilityID]
		if !ok || role == "" {
			continue
		}
		for _, actionID := range entry.Actions {
			if actionID == "exec" {
				return role, nil
			}
		}
	}
	return "", nil
}

// SummarizeIssueAnchor 锚点摘要（my_task relatedIssues / 冲突卡定位消费）：
// document=「document」；line=「line 42」；range=「range 10-20」；block=「block 42-58(sql)」；
// 锚定产物时前置 artifact id。零值字段按未设处理。
func SummarizeIssueAnchor(kind string, artifactID string, line int64, startLine int64, endLine int64, blockStart int64, blockEnd int64, lang string) string {
	summary := ""
	switch kind {
	case IssueAnchorLine:
		summary = "line " + formatInt(line)
	case IssueAnchorRange:
		summary = "range " + formatInt(startLine) + "-" + formatInt(endLine)
	case IssueAnchorBlock:
		summary = "block " + formatInt(blockStart) + "-" + formatInt(blockEnd) + "(" + lang + ")"
	default:
		summary = IssueAnchorDocument
	}
	if artifactID != "" && kind != IssueAnchorDocument {
		summary = artifactID + " " + summary
	}
	return summary
}

// formatInt int64 锚点行号格式化（摘要拼接用）。
func formatInt(v int64) string {
	return strconv.FormatInt(v, 10)
}
