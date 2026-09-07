// service 包本文件：评论/issue 状态机 v2 全链路五端点（spec §1.2 路由表 comments 五行）——
// wb_comment / wb_reply / wb_resolve / wb_confirm_review / wb_reopen。
// DTO 以 protoc 生成的 api 包类型承载（proto 为契约真相源，spec §1.3 禁手写漂移）。
//
// 写路径统一口径：domain 状态机判定先行（非法迁移 409 INVALID_TRANSITION）→
// 单事务（行锁 SELECT … FOR UPDATE 防并发丢行）→ 乐观锁更新（updated_at 基线）→
// 同事务审计（audit_log 逐字段留痕，审计失败回滚业务事务）。
package service

import (
	"context"
	"fmt"

	gormio "gorm.io/gorm"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/domain"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/pkg/utils"
	"github.com/huzilin/wbflow/rd/repo"
)

// IssueService 评论/issue 五端点 service。
type IssueService struct {
	repo *repo.Repo
	// escalateLoopThreshold 澄清不收敛自动上升阈值（T12：config escalate.loop_threshold
	// 注入，spec §2 reopen≥5 口径参数化；≤0=关闭自动上升，手动上升不受影响）。
	escalateLoopThreshold int64
	// notifier 通知收敛写入口（T11：事务外派发，失败不阻断业务）
	notifier *Notifier
}

// NewIssueService 构造评论/issue service。escalateLoopThreshold 取配置
// escalate.loop_threshold（缺省 5=domain.DefaultEscalateLoopThreshold）。
func NewIssueService(r *repo.Repo, escalateLoopThreshold int64) *IssueService {
	return &IssueService{repo: r, escalateLoopThreshold: escalateLoopThreshold, notifier: NewNotifier(r)}
}

// Comment wb_comment：四级锚点评论建行（issues.state=open）。
//
// 入参：req.RequirementId 必填（reqId 路径）；req.NodeId 必填且须为该需求实例节点；
// req.Anchor 未填充=kind=document 级；anchor.kind 与行号族字段一致性由
// domain.ValidateIssueAnchor 校验（line/range/block 必锚产物且产物须存在）；
// author/role_id/text 必填；issue_type 三值（空=issue）；escalate 可选置位。
// 推导：reviewer_role=发表者 ∈ 节点评审位（domain.DeriveReviewerRole）；
// assignee=显式指派 > L1 锚点产物登记人 > L2 节点 exec 能力归属（domain.DeriveAssignee）。
// 出参：建行后评论行（issue_id=cm-<ts> 事务内取号）。
//
// 错误集：ErrInvalidArgument（必填缺失/锚点不一致/issue_type 越集）；
// ErrNotFound（需求/流程实例/节点/锚定产物未命中）；ErrIdempotentConflict
// （取号唯一键冲突重试耗尽）；ErrOptimisticLock/底层 IO 错误经 Trace 上抛。
func (s *IssueService) Comment(ctx context.Context, req *api.CommentReq) (*api.CommentResp, error) {
	if req.RequirementId == "" {
		return nil, invalidArg("requirement_id 必填")
	}
	if req.NodeId == "" {
		return nil, invalidArg("node_id 必填")
	}
	if req.Author == "" {
		return nil, invalidArg("author 必填")
	}
	if req.RoleId == "" {
		return nil, invalidArg("role_id 必填")
	}
	if req.Text == "" {
		return nil, invalidArg("text 必填")
	}
	if !domain.ValidIssueType(req.IssueType) {
		return nil, invalidArg("issue_type %q 非法（issue/notification/discussion）", req.IssueType)
	}

	anchorKind := domain.IssueAnchorDocument
	anchorArtifactID := ""
	var anchorLine, anchorStartLine, anchorEndLine, anchorBlockStart, anchorBlockEnd int64
	anchorLang := ""
	if req.Anchor != nil {
		anchorKind = req.Anchor.Kind
		anchorArtifactID = req.Anchor.ArtifactId
		anchorLine = req.Anchor.Line
		anchorStartLine = req.Anchor.StartLine
		anchorEndLine = req.Anchor.EndLine
		anchorBlockStart = req.Anchor.BlockStart
		anchorBlockEnd = req.Anchor.BlockEnd
		anchorLang = req.Anchor.Lang
	}
	if err := domain.ValidateIssueAnchor(anchorKind, anchorArtifactID, anchorLine, anchorStartLine, anchorEndLine, anchorBlockStart, anchorBlockEnd, anchorLang); err != nil {
		return nil, invalidArg("%v", err)
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

	// L1 锚点产物：必锚产物须存在（取登记人做归属推导）；document 级无产物。
	artifactAddedBy := ""
	if anchorArtifactID != "" {
		artifact, err := s.findArtifact(ctx, req.RequirementId, anchorArtifactID)
		if err != nil {
			return nil, errs.Trace(err)
		}
		if artifact == nil {
			return nil, notFound("锚定产物 %q 不存在于需求 %q", anchorArtifactID, req.RequirementId)
		}
		artifactAddedBy = artifact.AddedBy
	}

	// reviewer_role 推导 + assignee 推导（L1 → L2）
	reviewerRole, err := domain.DeriveReviewerRole(req.RoleId, node.Reviewers)
	if err != nil {
		return nil, errs.Trace(err)
	}
	execRole, err := s.nodeExecRole(ctx, node, flow.Scope)
	if err != nil {
		return nil, errs.Trace(err)
	}
	assignee := domain.DeriveAssignee(req.Assignee, artifactAddedBy, execRole)

	issueType := req.IssueType
	if issueType == "" {
		issueType = domain.IssueTypeIssue
	}
	escalate := int64(0)
	if req.Escalate {
		escalate = domain.MarkEscalate()
	}

	row := &repo.Issue{
		RequirementID:    req.RequirementId,
		NodeID:           req.NodeId,
		AnchorKind:       anchorKind,
		Author:           req.Author,
		RoleID:           req.RoleId,
		IssueType:        issueType,
		Text:             req.Text,
		State:            repo.IssueStateOpen,
		Escalate:         escalate,
		Replies:          "[]",
		AnchorArtifactID: strPtrIfSet(anchorArtifactID),
		AnchorLine:       int64PtrIfSet(anchorLine),
		AnchorStartLine:  int64PtrIfSet(anchorStartLine),
		AnchorEndLine:    int64PtrIfSet(anchorEndLine),
		AnchorBlockStart: int64PtrIfSet(anchorBlockStart),
		AnchorBlockEnd:   int64PtrIfSet(anchorBlockEnd),
		AnchorLang:       strPtrIfSet(anchorLang),
		ReviewerRole:     strPtrIfSet(reviewerRole),
		Assignee:         strPtrIfSet(assignee),
	}
	for attempt := 0; attempt < idAllocRetryLimit; attempt++ {
		retryable, err := s.commentOnce(ctx, row)
		if err != nil {
			if retryable {
				continue
			}
			return nil, err
		}
		// 通知派发（事务外，spec §1.3——失败不阻断业务）：评论携 escalate 置位
		// → 冲突上升裁决通知（§6.8 reviewer 主持；收件=评审归属角色，回落 User）
		if escalate == 1 {
			s.dispatchEscalationNotification(ctx, row)
		}
		return &api.CommentResp{Ok: true, Issue: issueItem(row)}, nil
	}
	return nil, errs.Trace(errs.ErrIdempotentConflict)
}

// dispatchEscalationNotification 冲突上升裁决通知派发（escalate 置位后调用，
// 事务外落库）：收件角色=issue 评审归属角色（reviewer 主持，§6.8），非评审上下文
// （reviewer_role 空）回落 User 仲裁（spec §2 澄清不收敛口径）；跳转=冲突 issue。
func (s *IssueService) dispatchEscalationNotification(ctx context.Context, row *repo.Issue) {
	recipient := derefString(row.ReviewerRole)
	if recipient == "" {
		recipient = auditByWhoUser
	}
	Dispatch(s.notifier, NotificationSpec{
		Kind:          NotificationKindEscalation,
		RequirementID: row.RequirementID,
		NodeID:        row.NodeID,
		TargetID:      row.IssueID,
		RecipientRole: recipient,
		Sender:        row.Author,
		SenderRole:    row.RoleID,
		Text:          fmt.Sprintf("冲突上升裁决：评论 %s（节点 %s）待裁决", row.IssueID, row.NodeID),
	})
}

// commentOnce 单次建行尝试（事务：cm-<ts> 取号 → 插入 → INSERT 审计）；
// 返回 retryable=true 表示唯一键冲突/锁竞争可整体重试。
func (s *IssueService) commentOnce(ctx context.Context, row *repo.Issue) (bool, error) {
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		issueID, err := s.repo.AllocIssueIDTx(ctx, tx)
		if err != nil {
			return err
		}
		row.IssueID = issueID
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "issues",
			RowID:  issueID,
			Action: "INSERT",
			ByWho:  row.RoleID,
		})
	})
	if txErr != nil {
		if repo.IsDuplicateEntry(txErr) || repo.IsLockRace(txErr) {
			return true, errs.Trace(txErr)
		}
		return false, errs.Trace(txErr)
	}
	return false, nil
}

// Reply wb_reply：replies 不可变追加（只增不改）+ roundCount++ + last_reply_at。
// 状态迁移由 domain 状态机判定：open→pending-confirm（澄清回复完成待复核）/
// to-fix→fixed（修复完成申报）；pending-confirm/fixed 追加回复状态停留不回退
// （多轮说明幂等）；resolved/overridden 拒绝（终态不可追加）。
//
// 入参：req.CommentId 必填（commentId 路径）；by/role_id/text 必填；
// voided=误发 reply 标注作废（ui-interaction-final §3，回复照常留痕）。
// 出参：追加后对话轮数 + 评论行全量（replies JSON）。
//
// 错误集：ErrInvalidArgument（必填缺失）；ErrNotFound（评论未命中）；
// ErrInvalidTransition（状态机拒绝，409）；ErrOptimisticLock（并发更新冲突）；
// 底层 IO 错误经 Trace 上抛。
func (s *IssueService) Reply(ctx context.Context, req *api.ReplyReq) (*api.ReplyResp, error) {
	if req.CommentId == "" {
		return nil, invalidArg("comment_id 必填")
	}
	if req.By == "" {
		return nil, invalidArg("by 必填")
	}
	if req.RoleId == "" {
		return nil, invalidArg("role_id 必填")
	}
	if req.Text == "" {
		return nil, invalidArg("text 必填")
	}
	row, err := s.transitionIssue(ctx, transitionInput{
		issueID:   req.CommentId,
		action:    domain.IssueActionReply,
		actorRole: req.RoleId,
		reply: repo.IssueReply{
			By:     req.By,
			RoleID: req.RoleId,
			Text:   req.Text,
			Voided: req.Voided,
		},
	})
	if err != nil {
		return nil, err
	}
	return &api.ReplyResp{Ok: true, RoundCount: row.RoundCount, Issue: issueItem(row)}, nil
}

// Resolve wb_resolve：解决双路径申报（操作者须=assignee）。
// path=fix（修复关闭路径）：open/pending-confirm→to-fix（定性需修复进入修复期）/
// to-fix/fixed→fixed（修复关闭申报待复核，重申报状态幂等）；resolve 族列落值。
// path=agree（协商一致关闭）：非终态→resolved，确认内嵌于协商一致
// （resolve 族 + resolve_reviewer_confirmed=1 + resolved_by/at 同步落值）。
//
// 入参：req.CommentId 必填；req.Path 必填（fix|agree）；req.Operator 必填（须=assignee）；
// note 说明（空=不落说明列）；escalate 可选置位。
// 出参：resolve 后评论行。
//
// 错误集：ErrInvalidArgument（必填缺失/path 越集）；ErrNotFound（评论未命中）；
// ErrInvalidTransition（状态机拒绝/操作者非 assignee/未指派，409）；
// ErrOptimisticLock；底层 IO 错误经 Trace 上抛。
func (s *IssueService) Resolve(ctx context.Context, req *api.ResolveReq) (*api.ResolveResp, error) {
	if req.CommentId == "" {
		return nil, invalidArg("comment_id 必填")
	}
	if req.Path != domain.IssueResolvePathFix && req.Path != domain.IssueResolvePathAgree {
		return nil, invalidArg("path %q 非法（fix|agree）", req.Path)
	}
	if req.Operator == "" {
		return nil, invalidArg("operator 必填")
	}
	row, err := s.transitionIssue(ctx, transitionInput{
		issueID:   req.CommentId,
		action:    domain.IssueActionResolve,
		resolve:   req.Path,
		operator:  req.Operator,
		actorRole: req.Operator,
		note:      req.Note,
		escalate:  req.Escalate,
	})
	if err != nil {
		return nil, err
	}
	return &api.ResolveResp{Ok: true, Issue: issueItem(row)}, nil
}

// Confirm wb_confirm_review：提评者复核确认（pending-confirm/fixed→resolved；
// resolve_reviewer_confirmed=1 + resolved_by/at）。
// 操作者须=提评者（reviewer_role 优先，缺省 author——domain.ValidateIssueActor）。
//
// 入参：req.CommentId 必填；req.Operator 必填。
// 出参：确认后评论行。
//
// 错误集：ErrInvalidArgument（必填缺失）；ErrNotFound（评论未命中）；
// ErrInvalidTransition（状态机拒绝/操作者非提评者，409）；ErrOptimisticLock；
// 底层 IO 错误经 Trace 上抛。
func (s *IssueService) Confirm(ctx context.Context, req *api.ConfirmReviewReq) (*api.ConfirmReviewResp, error) {
	if req.CommentId == "" {
		return nil, invalidArg("comment_id 必填")
	}
	if req.Operator == "" {
		return nil, invalidArg("operator 必填")
	}
	row, err := s.transitionIssue(ctx, transitionInput{
		issueID:   req.CommentId,
		action:    domain.IssueActionConfirm,
		operator:  req.Operator,
		actorRole: req.Operator,
	})
	if err != nil {
		return nil, err
	}
	return &api.ConfirmReviewResp{Ok: true, Issue: issueItem(row)}, nil
}

// Reopen wb_reopen：澄清 reopen（pending-confirm→open）/ 修复 revise（fixed→to-fix），
// 均 loop_count++（双读法：澄清期读 reopen 次数、修复期读 revise 次数）。
// loop_count 达阈值（config escalate.loop_threshold 注入，spec §2 reopen≥5 口径）
// 自动标 escalate 并写通知位行（issues notification，T12 裁决链消费）。reopen
// 理由 text 非空时追加 replies（不可变追加，裁决回填同通道）。操作者须=提评者。
//
// 入参：req.CommentId 必填；req.Operator 必填；text 可选理由。
// 出参：追加后 loop_count + 评论行。
//
// 错误集：ErrInvalidArgument（必填缺失）；ErrNotFound（评论未命中）；
// ErrInvalidTransition（状态机拒绝/操作者非提评者，409）；ErrOptimisticLock；
// 底层 IO 错误经 Trace 上抛。
func (s *IssueService) Reopen(ctx context.Context, req *api.ReopenReq) (*api.ReopenResp, error) {
	if req.CommentId == "" {
		return nil, invalidArg("comment_id 必填")
	}
	if req.Operator == "" {
		return nil, invalidArg("operator 必填")
	}
	row, err := s.transitionIssue(ctx, transitionInput{
		issueID:   req.CommentId,
		action:    domain.IssueActionReopen,
		operator:  req.Operator,
		actorRole: req.Operator,
		reply: repo.IssueReply{
			By:     req.Operator,
			RoleID: req.Operator,
			Text:   req.Text,
		},
	})
	if err != nil {
		return nil, err
	}
	return &api.ReopenResp{Ok: true, LoopCount: row.LoopCount, Issue: issueItem(row)}, nil
}

// transitionInput 状态迁移统一入参（五端点收敛点：行锁读 → 操作者约束 →
// 状态机判定 → 列更新 + 同事务审计）。
type transitionInput struct {
	// issueID 评论业务 id
	issueID string
	// action 动作（domain.IssueAction*）
	action string
	// resolve resolve 路径（仅 resolve 动作）
	resolve string
	// operator 操作者（审计 by_who）
	operator string
	// actorRole 参与角色约束校验身份（=operator）
	actorRole string
	// note resolve 说明（resolve 动作）
	note string
	// escalate 可选置位（comment/resolve 动作）
	escalate bool
	// reply 追加回复条目（reply 恒填；reopen 携带理由时填 by/roleId/text）
	reply repo.IssueReply
}

// transitionIssue 五端点写路径收敛：单事务内行锁读取（并发 reply 不丢行）→
// 操作者角色约束（domain.ValidateIssueActor）→ 状态机判定（domain.IssueTransition，
// 非法迁移 409）→ 乐观锁更新 + 同事务审计（逐字段留痕）。
//
// 错误集：ErrNotFound（评论未命中）；ErrInvalidTransition；ErrOptimisticLock；
// replies JSON 非法（数据损坏）；底层 IO 错误经 Trace 上抛。
func (s *IssueService) transitionIssue(ctx context.Context, in transitionInput) (*repo.Issue, error) {
	row := &repo.Issue{}
	escalateSet := false
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		locked, err := s.repo.GetIssueByIssueIDLockTx(ctx, tx, in.issueID)
		if err != nil {
			return err
		}
		if locked == nil {
			return notFound("评论 %q 不存在", in.issueID)
		}
		*row = *locked

		view := domain.IssueView{
			State:        row.State,
			Author:       row.Author,
			ReviewerRole: derefString(row.ReviewerRole),
			Assignee:     derefString(row.Assignee),
		}
		if err := domain.ValidateIssueActor(in.action, in.actorRole, view); err != nil {
			return err
		}
		effect, err := domain.IssueTransition(in.action, in.resolve, row.State)
		if err != nil {
			return err
		}

		now := repo.NowMillis()
		updates := map[string]any{"state": effect.NextState}
		audits := make([]repo.AuditLog, 0, 4)
		if row.State != effect.NextState {
			audits = append(audits, issueAudit(row.IssueID, "state", row.State, effect.NextState, in.operator))
		}

		if effect.RoundCountDelta > 0 {
			updates["round_count"] = row.RoundCount + effect.RoundCountDelta
			row.RoundCount = row.RoundCount + effect.RoundCountDelta
		}
		if effect.LoopCountDelta > 0 {
			updates["loop_count"] = row.LoopCount + effect.LoopCountDelta
			row.LoopCount = row.LoopCount + effect.LoopCountDelta
			audits = append(audits, issueAudit(row.IssueID, "loop_count",
				fmt.Sprintf("%d", row.LoopCount-effect.LoopCountDelta), fmt.Sprintf("%d", row.LoopCount), in.operator))
			// 澄清不收敛自动上升（spec §2：loop_count 达阈值如 reopen≥5，阈值经
			// config escalate.loop_threshold 注入）+ 通知位行（T12，同事务）
			if in.action == domain.IssueActionReopen && domain.AutoEscalateNeeded(row.LoopCount, s.escalateLoopThreshold) && row.Escalate == 0 {
				updates["escalate"] = domain.MarkEscalate()
				row.Escalate = 1
				escalateSet = true
				audits = append(audits, issueAudit(row.IssueID, "escalate", "0", "1", in.operator))
				if _, err := writeEscalationNoticeTx(ctx, tx, s.repo, row,
					domain.AutoEscalateTriggerText(s.escalateLoopThreshold), auditByWhoPlugin); err != nil {
					return err
				}
			}
		}
		if effect.AppendReply && in.reply.By != "" {
			var replies []repo.IssueReply
			if err := utils.UnmarshalJSONText(row.Replies, &replies); err != nil {
				return errs.Trace(err)
			}
			entry := in.reply
			entry.ReplyID = fmt.Sprintf("r-%d", len(replies)+1)
			entry.At = now
			replies = append(replies, entry)
			text, err := utils.MarshalJSONText(replies)
			if err != nil {
				return errs.Trace(err)
			}
			updates["replies"] = text
			row.Replies = text
			audits = append(audits, issueAudit(row.IssueID, "replies", "", entry.ReplyID, in.operator))
		}
		if effect.TouchLastReplyAt {
			updates["last_reply_at"] = now
			row.LastReplyAt = &now
		}
		if effect.SetResolveFamily {
			updates["resolve_by"] = in.operator
			updates["resolve_at"] = now
			updates["resolve_path"] = in.resolve
			row.ResolveBy = &in.operator
			row.ResolveAt = &now
			row.ResolvePath = &in.resolve
			if in.note != "" {
				updates["resolve_note"] = in.note
				row.ResolveNote = &in.note
			}
		}
		if effect.ConfirmClose {
			updates["resolve_reviewer_confirmed"] = int64(1)
			updates["resolved_by"] = in.operator
			updates["resolved_at"] = now
			row.ResolveReviewerConfirmed = 1
			row.ResolvedBy = &in.operator
			row.ResolvedAt = &now
		}
		if in.escalate {
			if row.Escalate == 0 {
				escalateSet = true
			}
			updates["escalate"] = domain.MarkEscalate()
			row.Escalate = 1
		}
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Issue{}, row.ID, locked.UpdatedAt, updates); err != nil {
			return err
		}
		for _, audit := range audits {
			if err := s.repo.InsertAuditLogWith(tx, &audit); err != nil {
				return err
			}
		}
		// 同步派生列（出参口径）
		row.State = effect.NextState
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	// 通知派发（事务外，spec §1.3——失败不阻断业务）：本次迁移新置 escalate
	// → 冲突上升裁决通知（§6.8；重复置位/已上升幂等不重发）
	if escalateSet {
		s.dispatchEscalationNotification(ctx, row)
	}
	return row, nil
}

// issueAudit issue UPDATE 审计行装配（field 级留痕，old/new 文本化）。
func issueAudit(issueID string, field string, oldVal string, newVal string, byWho string) repo.AuditLog {
	return repo.AuditLog{
		Table:    "issues",
		RowID:    issueID,
		Action:   "UPDATE",
		Field:    strPtr(field),
		OldValue: strPtr(oldVal),
		NewValue: strPtr(newVal),
		ByWho:    byWho,
	}
}

// findArtifact 需求下按业务 id 找产物（T6 登记端点未接线前登记量小，列表过滤）。
func (s *IssueService) findArtifact(ctx context.Context, requirementID string, artifactID string) (*repo.Artifact, error) {
	rows, err := s.repo.ListArtifactsByRequirement(ctx, requirementID)
	if err != nil {
		return nil, errs.Trace(err)
	}
	for _, row := range rows {
		if row.ArtifactID == artifactID {
			return row, nil
		}
	}
	return nil, nil
}

// nodeExecRole 节点产出角色（L2 归属推导输入）：定义侧激活清单按实例 scope 裁剪后
// 取 exec 相位能力归属角色（定义缺失=seed 异常，返回空串不阻断建评）。
func (s *IssueService) nodeExecRole(ctx context.Context, node *repo.FlowNode, scope string) (string, error) {
	defNodes, err := s.repo.ListFlowDefNodesByTemplateIDs(ctx, []string{node.DefNodeID})
	if err != nil {
		return "", errs.Trace(err)
	}
	if len(defNodes) == 0 {
		return "", nil
	}
	filtered, err := domain.FilterNodeActions(defNodes[0].NodeActions, scope)
	if err != nil {
		return "", errs.Trace(err)
	}
	caps, err := s.repo.ListCapabilities(ctx)
	if err != nil {
		return "", errs.Trace(err)
	}
	infos := make(map[string]domain.CapabilityInfo, len(caps))
	for _, cap := range caps {
		var actions []repo.ActionDef
		if err := utils.UnmarshalJSONText(cap.Actions, &actions); err != nil {
			return "", errs.Trace(err)
		}
		infos[cap.CapabilityID] = domain.CapabilityInfo{RoleID: cap.RoleID, Actions: actions}
	}
	return domain.NodeExecRole(filtered, infos)
}

// issueItem repo 行 → proto 出参（指针列零值收敛：NULL → 空串/0）。
func issueItem(row *repo.Issue) *api.IssueItem {
	return &api.IssueItem{
		IssueId:                  row.IssueID,
		NodeId:                   row.NodeID,
		AnchorKind:               row.AnchorKind,
		Author:                   row.Author,
		RoleId:                   row.RoleID,
		Text:                     row.Text,
		State:                    row.State,
		Escalate:                 row.Escalate,
		LoopCount:                row.LoopCount,
		RoundCount:               row.RoundCount,
		CreatedAt:                row.CreatedAt,
		IssueType:                row.IssueType,
		ReviewerRole:             derefString(row.ReviewerRole),
		Assignee:                 derefString(row.Assignee),
		AnchorArtifactId:         derefString(row.AnchorArtifactID),
		AnchorLine:               derefInt64(row.AnchorLine),
		AnchorStartLine:          derefInt64(row.AnchorStartLine),
		AnchorEndLine:            derefInt64(row.AnchorEndLine),
		AnchorBlockStart:         derefInt64(row.AnchorBlockStart),
		AnchorBlockEnd:           derefInt64(row.AnchorBlockEnd),
		AnchorLang:               derefString(row.AnchorLang),
		Replies:                  row.Replies,
		LastReplyAt:              derefInt64(row.LastReplyAt),
		ResolvePath:              derefString(row.ResolvePath),
		ResolveBy:                derefString(row.ResolveBy),
		ResolveAt:                derefInt64(row.ResolveAt),
		ResolveNote:              derefString(row.ResolveNote),
		ResolveReviewerConfirmed: row.ResolveReviewerConfirmed,
		ResolvedBy:               derefString(row.ResolvedBy),
		ResolvedAt:               derefInt64(row.ResolvedAt),
		UpdatedAt:                row.UpdatedAt,
	}
}

// derefString 指针字符串零值收敛（nil → 空串，NULL 列出参口径）。
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// derefInt64 指针整数零值收敛（nil → 0，NULL 列出参口径）。
func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// strPtrIfSet 空串映射 NULL（NULL 保留清单列建行口径），非空取指针。
func strPtrIfSet(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// int64PtrIfSet 零值映射 NULL（锚点行号族 0=未设），非零取指针。
func int64PtrIfSet(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}
