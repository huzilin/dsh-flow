// service 包本文件：通知收敛写入口（T11）——通知规程 v3 七类（capability-model-final
// §6.8 用户门禁与上升通知规程 + ui-interaction-final §3.1 收件箱条目类型 v3）。
//
// 口径（spec §1.3/§2）：
//   - 通知载体 = issues 表 issue_type=notification 行（anchor_kind=类别、
//     anchor_artifact_id=跳转目标 id、assignee=收件角色、state=open 未读/resolved 已读，
//     列复用口径见 repo/notification.go 头注）；
//   - 通知派发一律事务外（业务事务提交后调用 Dispatch）：写入失败仅记错误日志、
//     不回滚业务——「消息/通知派发移出事务」+「失败不阻断业务路径」；
//   - 投递者 = plugin（系统层触达保障；会话迎宾=agent 层，双通道分工）。
//
// 七类与触发点（接线现状）：
//
//	类别             触发点                                         收件角色  跳转
//	gate_pending     review collect approved+user_gate（T11 已接）   owner    节点
//	clarify_task     review collect=clarify（T11 已接）              owner    节点
//	re_review        respond complete 澄清闭环重入轮（T11 已接）      reviewer 节点
//	stall_followup   扫描器 5min 未表态（T10 触发点，枚举位预留）      reviewer 任务
//	need_intervene   扫描器 10min 未推进（T10 触发点，枚举位预留）     user     任务
//	escalation       escalate 置位（comment/resolve/reopen 自动阈值，T11 已接） reviewer 裁决 issue
//	session_message  按需会话通道（T12+，枚举位预留）                 绑定角色  session
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// 通知类别七类枚举（ui-interaction-final §3.1 v3 定稿；值落 issues.anchor_kind
// 复用列，≤16 字符列宽约束）。
const (
	// NotificationKindGatePending 待你拍板（门禁 agreed + user_gate）
	NotificationKindGatePending = "gate_pending"
	// NotificationKindClarifyTask 澄清任务（collect 判 unagree → owner 澄清循环）
	NotificationKindClarifyTask = "clarify_task"
	// NotificationKindReReview 重新表态请求（澄清闭环后 review 重入新轮）
	NotificationKindReReview = "re_review"
	// NotificationKindStallFollowup 悬置催办（扫描器：已结束 5min 未表态）
	NotificationKindStallFollowup = "stall_followup"
	// NotificationKindNeedIntervene 需你介入（扫描器：10min 仍未推进，报 User）
	NotificationKindNeedIntervene = "need_intervene"
	// NotificationKindEscalation 冲突上升裁决（escalate 置位：显式/自动阈值）
	NotificationKindEscalation = "escalation"
	// NotificationKindSessionMessage 会话消息（按需通道/对齐未读，预留枚举位）
	NotificationKindSessionMessage = "session_message"
)

// notificationTargetTypes 类别 → 跳转目标类型确定性映射（InboxItem.target_type
// 出参口径；类别即定位语义，无需独立存储列）。
var notificationTargetTypes = map[string]string{
	NotificationKindGatePending:    "node",
	NotificationKindClarifyTask:    "node",
	NotificationKindReReview:       "node",
	NotificationKindStallFollowup:  "task",
	NotificationKindNeedIntervene:  "task",
	NotificationKindEscalation:     "issue",
	NotificationKindSessionMessage: "session",
}

// validNotificationKind 类别枚举校验。
func validNotificationKind(kind string) bool {
	_, ok := notificationTargetTypes[kind]
	return ok
}

// notificationStore 通知落库通道接口（consumer 侧 seam：Notifier 依赖倒置，
// 故障注入单测替换实现；*repo.Repo 天然满足）。
type notificationStore interface {
	// AllocIssueID 通知行业务 id 取号（cm-<ts>，与评论共用号段）
	AllocIssueID(ctx context.Context) (string, error)
	// InsertNotification 通知行写入（单事务 INSERT + 审计，见 repo 同名方法）
	InsertNotification(ctx context.Context, row *repo.Issue) error
}

// Notifier 通知收敛写入口：七类通知统一经 Notify 落库（issues 载体），
// 业务写路径经 Dispatch 事务外派发（失败不阻断）。
type Notifier struct {
	store notificationStore
}

// NewNotifier 构造通知写入口（store=*repo.Repo）。
func NewNotifier(r *repo.Repo) *Notifier {
	return &Notifier{store: r}
}

// NotificationSpec 通知派发规格（service 层触发点装配的入参载体）。
type NotificationSpec struct {
	// Kind 通知类别（七类枚举，必填）
	Kind string
	// RequirementID 需求 id（跳转一级定位，必填）
	RequirementID string
	// NodeID 节点 id（跳转二级定位；会话类可空）
	NodeID string
	// TargetType 跳转目标类型；空=按 Kind 确定性映射（notificationTargetTypes）
	TargetType string
	// TargetID 跳转目标业务 id（target_type=node 时空串——定位取 NodeID）
	TargetID string
	// RecipientRole 收件角色（必填；空收件人无投递意义，Notify 拦截）
	RecipientRole string
	// Sender 派发者（触发动作执行者/系统标识；空=plugin）
	Sender string
	// SenderRole 派发者角色；空=plugin（系统投递口径）
	SenderRole string
	// Text 通知文案
	Text string
}

// Notify 落库一条通知（issues 行 issue_type=notification，state=未读）。派发失败
// 返回错误——由 Dispatch 兜底吞错记日志，业务调用方不感知。
//
// 入参：spec.Kind/RequirementID/RecipientRole 必填；Kind 越集 / RecipientRole 空
// 返回错误（触发点装配缺陷，不入用户错误路径）。
// 错误集：落库 IO 错误经 Trace 上抛；入参缺陷以普通错误返回。
func (n *Notifier) Notify(ctx context.Context, spec NotificationSpec) (*repo.Issue, error) {
	if !validNotificationKind(spec.Kind) {
		return nil, fmt.Errorf("通知类别 %q 越集（七类枚举）", spec.Kind)
	}
	if spec.RequirementID == "" {
		return nil, fmt.Errorf("通知 requirement_id 必填")
	}
	if spec.RecipientRole == "" {
		return nil, fmt.Errorf("通知收件角色必填（kind=%s）", spec.Kind)
	}
	targetType := spec.TargetType
	if targetType == "" {
		targetType = notificationTargetTypes[spec.Kind]
	}
	// target_type=node 跳转定位取 node_id 列，target_id 列位空（NULL）
	targetID := spec.TargetID
	if targetType == "node" {
		targetID = ""
	}
	sender := spec.Sender
	if sender == "" {
		sender = "plugin"
	}
	senderRole := spec.SenderRole
	if senderRole == "" {
		senderRole = "plugin"
	}
	issueID, err := n.store.AllocIssueID(ctx)
	if err != nil {
		return nil, errs.Trace(err)
	}
	row := &repo.Issue{
		IssueID:          issueID,
		RequirementID:    spec.RequirementID,
		NodeID:           spec.NodeID,
		AnchorKind:       spec.Kind,
		AnchorArtifactID: strPtrIfSet(targetID),
		Author:           sender,
		RoleID:           senderRole,
		IssueType:        domain.IssueTypeNotification,
		Assignee:         strPtrIfSet(spec.RecipientRole),
		Text:             spec.Text,
		State:            repo.NotificationStateUnread,
		Replies:          "[]",
	}
	if err := n.store.InsertNotification(ctx, row); err != nil {
		return nil, errs.Trace(err)
	}
	return row, nil
}

// Dispatch 事务外通知派发（业务写路径统一出口）：Notify 失败仅记错误日志、
// 不上抛——spec §1.3「消息/通知派发移出事务」+ 工单 AC「通知写入失败不阻断
// 业务路径」。ctx 用背景上下文（业务请求 ctx 在 handler 返回后可能取消）。
func Dispatch(n *Notifier, spec NotificationSpec) {
	if n == nil {
		return
	}
	if _, err := n.Notify(context.Background(), spec); err != nil {
		// 记错误日志不阻断业务（标准库 log——service 层无 zap 依赖，测试期同样安全）
		log.Printf("[notifier] 通知派发失败（不阻断业务）: kind=%s req=%s node=%s recipient=%s err=%v",
			spec.Kind, spec.RequirementID, spec.NodeID, spec.RecipientRole, err)
	}
}

// DeriveOwnerRole 被评审方（owner）角色推导：节点 exec 能力归属角色优先
// （与 issue assignee L2 推导同函数口径 domain.NodeExecRole），回落
// requirements.owner；两者皆空返回空串（调用方跳过派发）。
func DeriveOwnerRole(ctx context.Context, r *repo.Repo, requirementID string, nodeID string) (string, error) {
	node, err := r.GetFlowNodeByRequirementAndNodeID(ctx, requirementID, nodeID)
	if err != nil {
		return "", errs.Trace(err)
	}
	if node == nil {
		return "", notFound("节点 %q 不存在于需求 %q", nodeID, requirementID)
	}
	flow, err := r.GetFlowByRequirementID(ctx, requirementID)
	if err != nil {
		return "", errs.Trace(err)
	}
	scope := ""
	if flow != nil {
		scope = flow.Scope
	}
	defNodes, err := r.ListFlowDefNodesByTemplateIDs(ctx, []string{node.DefNodeID})
	if err != nil {
		return "", errs.Trace(err)
	}
	if len(defNodes) > 0 {
		filtered, filterErr := domain.FilterNodeActions(defNodes[0].NodeActions, scope)
		if filterErr != nil {
			return "", errs.Trace(filterErr)
		}
		caps, err := r.ListCapabilities(ctx)
		if err != nil {
			return "", errs.Trace(err)
		}
		infos := make(map[string]domain.CapabilityInfo, len(caps))
		for _, capability := range caps {
			var actions []repo.ActionDef
			if err := utils.UnmarshalJSONText(capability.Actions, &actions); err != nil {
				return "", errs.Trace(err)
			}
			infos[capability.CapabilityID] = domain.CapabilityInfo{RoleID: capability.RoleID, Actions: actions}
		}
		if filtered != "" {
			execRole, execErr := domain.NodeExecRole(filtered, infos)
			if execErr != nil {
				return "", errs.Trace(execErr)
			}
			if execRole != "" {
				return execRole, nil
			}
		}
	}
	requirement, err := r.GetRequirementByID(ctx, requirementID)
	if err != nil {
		return "", errs.Trace(err)
	}
	if requirement == nil {
		return "", notFound("需求 %q 不存在", requirementID)
	}
	return requirement.Owner, nil
}
