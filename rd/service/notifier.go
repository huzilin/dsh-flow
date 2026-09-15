// service 包本文件：通知收敛写入口（T11）——通知规程 v3 七类（capability-model-final
// §6.8 用户门禁与上升通知规程 + ui-interaction-final §3.1 收件箱条目类型 v3）。
//
// 口径（spec §1.3/§2）：
//   - 通知载体 = issues 表 issue_type=notification 行（anchor_kind=类别、
//     anchor_artifact_id=跳转目标 id、assignee=收件角色、state=open 未读/resolved 已读，
//     列复用口径见 repo/notification.go 头注）；
//   - 通知可靠投递 outbox 化（spec §1.2 注记-H）：业务事务内 EnqueueNotificationTx
//     写 notify_outbox（通知意图与业务提交原子——「Dispatch 未被调用即丢」窗口
//     消除），事务提交后 FlushPending 推送为 issues 通知行；推送失败由 scanner
//     SweepOutbox 周期补投（无限重试直至送达，semantica `c25c0552`）。「失败不阻断
//     业务路径」口径不变：flush/sweep 失败只影响通知时效，不回滚业务；
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
//	escalation       escalate 置位（comment/settle/qualify/reopen 自动阈值，T11 已接） reviewer 裁决 issue
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
	"gorm.io/gorm"
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

// NotificationTargetTypes 类别 → 跳转目标类型映射的只读出口（收件箱 target_type
// 出参口径）。导出（C3 契约快照消费：rd/contract 词表单一真相源）；返回副本，
// 调用方改动不影响本包判定。
func NotificationTargetTypes() map[string]string {
	out := make(map[string]string, len(notificationTargetTypes))
	for k, v := range notificationTargetTypes {
		out[k] = v
	}
	return out
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

// outboxStore 消息事务表通道接口（outbox seam：flush/sweep 单测可替换实现；
// *repo.Repo 天然满足）。
type outboxStore interface {
	// ListPendingOutbox 待推进行（immediate flush 消费）
	ListPendingOutbox(ctx context.Context, limit int) ([]*repo.Outbox, error)
	// ListRetryableOutbox 补投候选（failed 预算内 / pending·sending 停滞）
	ListRetryableOutbox(ctx context.Context, staleBeforeMs int64, limit int) ([]*repo.Outbox, error)
	// ClaimOutbox 原子认领（status→sending；影响行=0 表示已被并发方认领）
	ClaimOutbox(ctx context.Context, id int64, fromStatuses []string) (bool, error)
	// SettleOutbox 推送落账（sent / failed+retry_count++）
	SettleOutbox(ctx context.Context, id int64, success bool, sendErr string) error
}

// Notifier 通知收敛写入口：七类通知统一经 Notify 落库（issues 载体）；
// 业务写路径事务内 EnqueueNotificationTx 写 outbox、事务提交后 FlushPending 推送
// （失败不阻断，scanner SweepOutbox 补投——spec §1.2 注记-H）。
type Notifier struct {
	store  notificationStore
	outbox outboxStore
}

// NewNotifier 构造通知写入口（store/outbox=*repo.Repo）。
func NewNotifier(r *repo.Repo) *Notifier {
	return &Notifier{store: r, outbox: r}
}

// outboxSweepLimit 单轮 flush/sweep 认领上限（防单轮长事务拖垮扫描周期）。
const outboxSweepLimit = 64

// outboxSendingStaleMs sending 停滞判定窗口（认领后进程崩溃残留的恢复阈值）。
const outboxSendingStaleMs = 5 * 60 * 1000

// outboxPendingStaleMs pending 停滞判定窗口（业务提交后 immediate flush 未触达
// ——进程即崩的残留；scanner 补投兜底）。小于 scanner 默认轮询间隔即可。
const outboxPendingStaleMs = 30 * 1000

// NotificationSpec 通知派发规格（service 层触发点装配的入参载体）。
type NotificationSpec struct {
	// Kind 通知类别（七类枚举，必填）
	Kind string
	// RequirementID 需求 id（跳转一级定位，必填）
	RequirementID string
	// NodeID 节点 id（跳转二级定位；会话类可空）
	NodeID string
	// TargetID 跳转目标业务 id；node 类未传时空串（定位取 NodeID），显式传入
	// 保留为锚点增强（注记-D1：re_review retest 子型传 issueId）
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

// normalizeNotificationSpec 入参校验 + 默认值填充（C5-D1 单一真相源：Notify 与
// outboxRowOf 同口径消费——此前两处手写同款校验）。Kind 越集 / RequirementID /
// RecipientRole 空=装配缺陷返回错误；Sender/SenderRole 空=plugin（系统投递口径）。
func normalizeNotificationSpec(spec NotificationSpec) (NotificationSpec, error) {
	if !validNotificationKind(spec.Kind) {
		return spec, fmt.Errorf("通知类别 %q 越集（七类枚举）", spec.Kind)
	}
	if spec.RequirementID == "" {
		return spec, fmt.Errorf("通知 requirement_id 必填")
	}
	if spec.RecipientRole == "" {
		return spec, fmt.Errorf("通知收件角色必填（kind=%s）", spec.Kind)
	}
	if spec.Sender == "" {
		spec.Sender = "plugin"
	}
	if spec.SenderRole == "" {
		spec.SenderRole = "plugin"
	}
	return spec, nil
}

// Notify 落库一条通知（issues 行 issue_type=notification，state=未读）。派发失败
// 返回错误——由 Dispatch 兜底吞错记日志，业务调用方不感知。
//
// 入参：spec.Kind/RequirementID/RecipientRole 必填；Kind 越集 / RecipientRole 空
// 返回错误（触发点装配缺陷，不入用户错误路径）。
// 错误集：落库 IO 错误经 Trace 上抛；入参缺陷以普通错误返回。
func (n *Notifier) Notify(ctx context.Context, spec NotificationSpec) (*repo.Issue, error) {
	spec, err := normalizeNotificationSpec(spec)
	if err != nil {
		return nil, err
	}
	// target_id 列位承载跳转锚点增强：target_type=node 的 kind 默认列位空
	// （定位取 node_id 列）；调用方显式传入时保留（注记-D1：re_review retest
	// 子型传 issueId 锚点，FE「待复验」投影与意见区跳转消费）
	issueID, allocErr := n.store.AllocIssueID(ctx)
	if allocErr != nil {
		return nil, errs.Trace(allocErr)
	}
	row := &repo.Issue{
		IssueID:          issueID,
		RequirementID:    spec.RequirementID,
		NodeID:           spec.NodeID,
		AnchorKind:       spec.Kind,
		AnchorArtifactID: strPtrIfSet(spec.TargetID),
		Author:           spec.Sender,
		RoleID:           spec.SenderRole,
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

// EnqueueNotificationTx 业务事务内 enqueue 一条通知意图（notify_outbox 行 pending，
// 与业务提交原子——spec §1.2 注记-H）。入参校验与 Notify 同口径（kind 越集 /
// requirement_id·recipient_role 必填拦截——装配缺陷不入用户错误路径，但事务内
// 拦截直接回滚业务：装配缺陷属代码 bug，宁可事务失败暴露也不静默吞掉）。
func EnqueueNotificationTx(ctx context.Context, tx *gorm.DB, spec NotificationSpec) error {
	row, err := outboxRowOf(spec)
	if err != nil {
		return err
	}
	return repo.InsertOutboxTx(ctx, tx, row)
}

// outboxRowOf NotificationSpec → notify_outbox 行装配（校验+默认值与 Notify
// 同源——normalizeNotificationSpec 单点，C5-D1）。
func outboxRowOf(spec NotificationSpec) (*repo.Outbox, error) {
	spec, err := normalizeNotificationSpec(spec)
	if err != nil {
		return nil, err
	}
	return &repo.Outbox{
		Kind:          spec.Kind,
		RequirementID: spec.RequirementID,
		NodeID:        spec.NodeID,
		TargetID:      strPtrIfSet(spec.TargetID),
		RecipientRole: spec.RecipientRole,
		Sender:        spec.Sender,
		SenderRole:    spec.SenderRole,
		Text:          spec.Text,
		Status:        repo.OutboxStatusPending,
	}, nil
}

// outboxSpec notify_outbox 行 → NotificationSpec（推送时还原装配口径）。
func outboxSpec(row *repo.Outbox) NotificationSpec {
	return NotificationSpec{
		Kind:          row.Kind,
		RequirementID: row.RequirementID,
		NodeID:        row.NodeID,
		TargetID:      utils.DerefStr(row.TargetID),
		RecipientRole: row.RecipientRole,
		Sender:        row.Sender,
		SenderRole:    row.SenderRole,
		Text:          row.Text,
	}
}

// FlushPending 业务事务提交后的即时推送：消费 status=pending 行逐条认领推送
// （best-effort——失败置 failed 留给 SweepOutbox 补投，不上抛，业务调用方无感）。
func (n *Notifier) FlushPending(ctx context.Context) {
	rows, err := n.outbox.ListPendingOutbox(ctx, outboxSweepLimit)
	if err != nil {
		log.Printf("[notifier] outbox 即时推送候选查询失败（不阻断业务）: err=%v", errs.Trace(err))
		return
	}
	for _, row := range rows {
		n.pushOne(ctx, row, []string{repo.OutboxStatusPending})
	}
}

// SweepOutbox 周期补投（scanner 消费）：failed（retry 预算内）+ pending/sending
// 停滞（进程崩溃残留）逐条认领推送。返回本轮推送成功条数（扫描报告用）。
func (n *Notifier) SweepOutbox(ctx context.Context, nowMillis int64) int {
	rows, err := n.outbox.ListRetryableOutbox(ctx, nowMillis-outboxPendingStaleMs, outboxSweepLimit)
	if err != nil {
		log.Printf("[notifier] outbox 补投候选查询失败: err=%v", errs.Trace(err))
		return 0
	}
	sent := 0
	for _, row := range rows {
		from := []string{repo.OutboxStatusPending, repo.OutboxStatusFailed}
		if row.Status == repo.OutboxStatusSending {
			// sending 停滞行仅在其确已超窗后恢复认领（与查询条件一致，防御
			// 查询与认领间的时间差）
			if nowMillis-row.UpdatedAt < outboxSendingStaleMs {
				continue
			}
			from = []string{repo.OutboxStatusSending}
		}
		if n.pushOne(ctx, row, from) {
			sent++
		}
	}
	return sent
}

// pushOne 单行推送：原子认领（防双推并发）→ Notify 落 issues 通知行 → 落账。
// 返回是否推送成功。失败置 failed——无限补投（semantica `c25c0552`），行永久
// 处于 SweepOutbox 候选，直至送达。
func (n *Notifier) pushOne(ctx context.Context, row *repo.Outbox, fromStatuses []string) bool {
	claimed, err := n.outbox.ClaimOutbox(ctx, row.ID, fromStatuses)
	if err != nil {
		log.Printf("[notifier] outbox 行 %d 认领失败: err=%v", row.ID, errs.Trace(err))
		return false
	}
	if !claimed {
		return false
	}
	if _, err := n.Notify(ctx, outboxSpec(row)); err != nil {
		if settleErr := n.outbox.SettleOutbox(ctx, row.ID, false, repo.OutboxStringErr("%v", errs.Trace(err))); settleErr != nil {
			log.Printf("[notifier] outbox 行 %d 失败落账异常: err=%v", row.ID, errs.Trace(settleErr))
		}
		return false
	}
	if err := n.outbox.SettleOutbox(ctx, row.ID, true, ""); err != nil {
		log.Printf("[notifier] outbox 行 %d 成功落账异常（通知已投递）: err=%v", row.ID, errs.Trace(err))
		return false
	}
	return true
}

// deriveOwnerRoleTx 被评审方（owner）角色推导：节点 exec 能力归属角色优先
// （与 issue assignee L2 推导同函数口径 domain.NodeExecRole），回落
// requirements.owner；两者皆空返回空串（调用方跳过派发）。
// 事务闭包内可调用（读走 db——tx 或 Gorm() 均可，全程非锁只读）。
// 包内私有（唯一调用方 review.go collect 事务内装配收件人）。
func deriveOwnerRoleTx(ctx context.Context, db *gorm.DB, requirementID string, nodeID string) (string, error) {
	node, err := repo.GetFlowNodeByRequirementAndNodeIDTx(ctx, db, requirementID, nodeID)
	if err != nil {
		return "", errs.Trace(err)
	}
	if node == nil {
		return "", notFound("节点 %q 不存在于需求 %q", nodeID, requirementID)
	}
	flow, err := repo.GetFlowByRequirementIDTx(ctx, db, requirementID)
	if err != nil {
		return "", errs.Trace(err)
	}
	scope := ""
	if flow != nil {
		scope = flow.Scope
	}
	execRole := ""
	defNodes, err := repo.ListFlowDefNodesByTemplateIDsTx(ctx, db, []string{node.DefNodeID})
	if err != nil {
		return "", errs.Trace(err)
	}
	if len(defNodes) > 0 {
		filtered, filterErr := domain.FilterNodeActions(defNodes[0].NodeActions, scope)
		if filterErr != nil {
			return "", errs.Trace(filterErr)
		}
		if filtered != "" {
			caps, capsErr := repo.ListCapabilitiesTx(ctx, db)
			if capsErr != nil {
				return "", errs.Trace(capsErr)
			}
			infos, infosErr := buildCapabilityInfoMap(caps)
			if infosErr != nil {
				return "", infosErr
			}
			var execErr error
			execRole, execErr = domain.NodeExecRole(filtered, infos)
			if execErr != nil {
				return "", errs.Trace(execErr)
			}
		}
	}
	requirement, reqErr := repo.GetRequirementByIDTx(ctx, db, requirementID)
	if reqErr != nil {
		return "", errs.Trace(reqErr)
	}
	if requirement == nil {
		return "", notFound("需求 %q 不存在", requirementID)
	}
	// exec 能力归属角色优先，空回落 requirement.owner（domain 单点判定；
	// 需求行恒校验存在——缺失即数据损坏报 notFound，不再借 exec 角色提前返回）
	return domain.OwnerRoleWithFallback(execRole, requirement.Owner), nil
}
