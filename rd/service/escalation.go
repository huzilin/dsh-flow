// service 包本文件：T12 冲突上升裁决链双端点——wb_escalate（拉起裁决 session）
// 与 wb_adjudicate（User 仲裁结论回填 + reviewer 重表态恢复评审轮），以及
// wb_override 语义收编方法 OverrideIssueTx。
//
// 定稿出处：
//   - spec §1.2 移出清单：wb_override（escalate 取代）——语义保留在 service 层；
//   - spec §2 澄清不收敛行：loop_count 达阈值（reopen≥5）自动标 escalate → User 仲裁；
//   - spec §4.1 冲突上升 AC：escalate → 裁决 session → 结论回填 issue →
//     reviewer 重表态 → 恢复；
//   - docs-tmp/capability-model-final.md §6 rule6：escalate 标记 → plugin 拉起
//     按需会话通道裁决 session → 用户裁决 → 结论追加到冲突 issue（数据层留痕、
//     node 可见）→ reviewer 重新表态落库 → 正常判据链恢复；
//   - docs-tmp/ui-interaction-final.md §1.4：上升裁决 session → 用户结论 →
//     回填冲突 issue → reviewer 重新表态（回 reviewing）→ 正常判据链恢复。
//
// 事务边界（spec §1.3/§2）：escalate/adjudicate 写路径单事务，issue 行与 session 行
// 均 SELECT … FOR UPDATE（并发双 escalate/双 adjudicate 串行化，仅一次生效）；
// 乐观锁更新 + 同事务逐字段审计；通知位行（issues notification）同事务落库
// ——与 T7「审计失败回滚业务事务」一致。
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

// transitionBlocked 非法状态迁移错误（哨兵 errs.ErrInvalidTransition 为 cause，
// HTTP 409 集中映射口径与 T7 issue 状态机一致）。
func transitionBlocked(format string, args ...any) error {
	return errs.Trace(fmt.Errorf("%w: %s", errs.ErrInvalidTransition, fmt.Sprintf(format, args...)))
}

// EscalationService 冲突上升裁决链 service（wb_escalate / wb_adjudicate /
// OverrideIssueTx 仲裁推翻语义收编）。
type EscalationService struct {
	repo *repo.Repo
}

// NewEscalationService 构造冲突上升裁决链 service。
func NewEscalationService(r *repo.Repo) *EscalationService {
	return &EscalationService{repo: r}
}

// Escalate wb_escalate：手动上升裁决（冲突任一方拉起；自动上升只置标 + 通知，
// 裁决 session 统一经本端点拉起）。
//
// 单事务内：issue 行锁读取 → 终态拦截（resolved/overridden 不可上升）→
// 进行中裁决 session 幂等判定（replies 上升条目 → session 行锁读，active 即
// noop 返回既有 session）→ session 建绑（ad-<ts> 取号、requirement/node 关联
// 冲突 issue 挂载节点、role=user 仲裁语义、dsh_ref 占位待 DSH 接入）→
// escalate 置位（幂等）→ replies 追加上升条目（携 sessionId）→ 通知位行
// （issue_type=notification，assignee=user）落库。
//
// 入参：req.CommentId 必填（commentId 路径）；req.Operator 必填；
// reason 可选（上升条目 text）。
// 出参：escalated=本次新置位；noop=幂等命中既有进行中 session；session_id；
// issue=上升后评论行。
//
// 错误集：ErrInvalidArgument（必填缺失）；ErrNotFound（评论未命中）；
// ErrInvalidTransition（终态不可上升，409）；ErrOptimisticLock；ErrIdempotentConflict
// （session 取号唯一键冲突重试耗尽）；底层 IO/JSON 错误经 Trace 上抛。
func (s *EscalationService) Escalate(ctx context.Context, req *api.EscalateReq) (*api.EscalateResp, error) {
	if req.CommentId == "" {
		return nil, invalidArg("comment_id 必填")
	}
	if req.Operator == "" {
		return nil, invalidArg("operator 必填")
	}

	resp := &api.EscalateResp{}
	if _, err := withIDAllocRetry(func() (struct{}, bool, error) {
		retryable, err := s.escalateOnce(ctx, req, resp)
		if err != nil {
			return struct{}{}, retryable, err
		}
		return struct{}{}, false, nil
	}); err != nil {
		return nil, err
	}
	return resp, nil
}

// escalateOnce 单次上升尝试（事务：issue 行锁 → session 幂等/建绑 → 置位 +
// 上升条目 + 通知位 + 乐观锁更新 + 审计）；retryable=true 表示取号唯一键冲突/
// 锁竞争可整体重试。
func (s *EscalationService) escalateOnce(ctx context.Context, req *api.EscalateReq, resp *api.EscalateResp) (bool, error) {
	row := &repo.Issue{}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		locked, err := s.repo.GetIssueByIssueIDLockTx(ctx, tx, req.CommentId)
		if err != nil {
			return err
		}
		if locked == nil {
			return notFound("评论 %q 不存在", req.CommentId)
		}
		if locked.State == repo.IssueStateResolved || locked.State == repo.IssueStateOverridden {
			return transitionBlocked("评论 %q 已终态（%s），不可上升裁决", locked.IssueID, locked.State)
		}
		*row = *locked

		// 幂等：进行中裁决 session 存在即 no-op（不重复建绑、不重复置位/留痕）
		session, active, err := s.activeAdjudicationTx(ctx, tx, locked)
		if err != nil {
			return err
		}
		resp.Ok = true
		resp.CommentId = locked.IssueID
		if active {
			resp.Noop = true
			resp.Escalated = false
			resp.SessionId = session.SessionID
			return nil
		}

		sessionID, err := s.repo.AllocAdjudicationSessionIDTx(ctx, tx)
		if err != nil {
			return err
		}
		sessionRow := repo.NewAdjudicationSession(sessionID, locked.RequirementID, locked.NodeID)
		if err := s.repo.CreateSessionTx(ctx, tx, sessionRow); err != nil {
			return err
		}
		if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:  "sessions",
			RowID:  sessionID,
			Action: "INSERT",
			ByWho:  req.Operator,
		}); err != nil {
			return err
		}

		now := repo.NowMillis()
		updates := map[string]any{}
		audits := make([]repo.AuditLog, 0, 4)
		resp.Escalated = locked.Escalate == 0
		if resp.Escalated {
			updates["escalate"] = domain.MarkEscalate()
			row.Escalate = 1
			audits = append(audits, issueAudit(locked.IssueID, "escalate", "0", "1", req.Operator))
		}
		repliesJSON, replyID, err := domain.AppendIssueReply(locked.Replies, repo.IssueReply{
			By:        req.Operator,
			RoleID:    req.Operator,
			Text:      domain.EscalationReplyText(req.Reason),
			At:        now,
			Kind:      domain.ReplyKindEscalation,
			SessionID: sessionID,
		})
		if err != nil {
			return errs.Trace(err)
		}
		updates["replies"] = repliesJSON
		row.Replies = repliesJSON
		audits = append(audits, issueAudit(locked.IssueID, "replies", "", replyID, req.Operator))
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Issue{}, locked.ID, locked.UpdatedAt, updates); err != nil {
			return err
		}
		for _, audit := range audits {
			if err := s.repo.InsertAuditLogWith(tx, &audit); err != nil {
				return err
			}
		}
		// 通知位（issues notification 行，assignee=user）：收件箱读侧 T11 消费
		if _, err := writeEscalationNoticeTx(ctx, tx, s.repo, locked,
			domain.ManualEscalateRouseText(req.Operator), req.Operator); err != nil {
			return err
		}
		resp.Noop = false
		resp.SessionId = sessionID
		return nil
	})
	if txErr != nil {
		if repo.IsDuplicateEntry(txErr) || repo.IsLockRace(txErr) {
			return true, errs.Trace(txErr)
		}
		return false, txErr
	}
	resp.Issue = issueItem(row)
	return false, nil
}

// Adjudicate wb_adjudicate：User 仲裁结论回填（裁决链第 3 拍，capability-model-final
// §6 rule6）。仅存在进行中裁决 session 的上升 issue 可裁决，否则 blocked：
//
//   - verdict=uphold（维持议题）：issue 状态保持（回正常澄清/修复链），escalate 复位；
//   - verdict=override（仲裁推翻）：wb_override 语义收编（OverrideIssueTx）——issue 落
//     overridden 终态（resolve_path=ruling + resolve_note 引用结论 + 确认内嵌）；
//
// 两路径共同：replies 追加裁决条目（verdict/conclusion/sessionId 数据层留痕）、
// escalate 复位 0（口径见 domain/escalation.go 包注释——仲裁完成即挂起解除，
// 二次拉锯可重新触发上升）、裁决 session 终局（status=ended）、reviewer 重表态
// （评审轮重入新轮：review/respond 相位节点 reenter，agree 携结论、unagree 复位
// reviewing；其余相位 no-op 不阻断回填）。
//
// 入参：req.CommentId 必填（commentId 路径）；req.Verdict ∈ {uphold, override}；
// req.Conclusion 必填（结论文本）；req.DecidedBy 必填（恒 user，审计 by_who）。
// 出参：verdict/session_id 回显 + reentered（轮重入是否拉起）/ round（重入后轮）+
// issue 回填后评论行。
//
// 错误集：ErrInvalidArgument（必填缺失/verdict 越集）；ErrNotFound（评论未命中）；
// ErrInvalidTransition（issue 已终态仍回填，409）；ErrGateBlocked（未上升裁决/
// 未拉起裁决 session/裁决已终局=重复裁决，409 blocked）；ErrOptimisticLock；
// 底层 IO/JSON 错误经 Trace 上抛。
func (s *EscalationService) Adjudicate(ctx context.Context, req *api.AdjudicateReq) (*api.AdjudicateResp, error) {
	if req.CommentId == "" {
		return nil, invalidArg("comment_id 必填")
	}
	if !domain.ValidAdjudicationVerdict(req.Verdict) {
		return nil, invalidArg("verdict %q 非法（uphold|override）", req.Verdict)
	}
	if req.Conclusion == "" {
		return nil, invalidArg("conclusion 必填（裁决结论文本）")
	}
	if req.DecidedBy == "" {
		return nil, invalidArg("decided_by 必填")
	}

	resp := &api.AdjudicateResp{}
	txErr := s.repo.Gorm().WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		locked, err := s.repo.GetIssueByIssueIDLockTx(ctx, tx, req.CommentId)
		if err != nil {
			return err
		}
		if locked == nil {
			return notFound("评论 %q 不存在", req.CommentId)
		}
		if locked.State == repo.IssueStateResolved || locked.State == repo.IssueStateOverridden {
			return transitionBlocked("评论 %q 已终态（%s），拒绝裁决回填", locked.IssueID, locked.State)
		}
		if locked.Escalate == 0 {
			return gateBlocked("评论 %q 未上升裁决（escalate=0），拒绝回填", locked.IssueID)
		}
		// session 定位（replies 上升条目）+ 行锁终局判定（并发双 adjudicate
		// 串行化：首例 ended 即终局，次例 blocked）
		session, active, err := s.activeAdjudicationTx(ctx, tx, locked)
		if err != nil {
			return err
		}
		if session == nil {
			return gateBlocked("评论 %q 未拉起裁决 session，拒绝回填", locked.IssueID)
		}
		if !active {
			return gateBlocked("评论 %q 裁决 session %q 已终局，拒绝重复裁决", locked.IssueID, session.SessionID)
		}

		now := repo.NowMillis()
		ruling := Ruling{
			Verdict:    req.Verdict,
			Conclusion: req.Conclusion,
			DecidedBy:  req.DecidedBy,
			SessionID:  session.SessionID,
			At:         now,
		}
		row := *locked
		if req.Verdict == domain.AdjudicationVerdictOverride {
			// wb_override 语义收编：仲裁推翻落 overridden 终态（含 escalate 复位
			// 与裁决条目回填，本方法内单次乐观锁更新）
			if err := s.OverrideIssueTx(ctx, tx, &row, ruling); err != nil {
				return err
			}
		} else {
			repliesJSON, upholdErr := s.applyUpholdTx(ctx, tx, locked, ruling)
			if upholdErr != nil {
				return upholdErr
			}
			row.Escalate = 0
			row.Replies = repliesJSON
		}

		// 裁决 session 终局（active→ended；行锁读已取 updated_at 基线）
		if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Session{}, session.ID, session.UpdatedAt, map[string]any{
			"status": repo.SessionStatusEnded,
		}); err != nil {
			return err
		}
		if err := s.repo.InsertAuditLogWith(tx, &repo.AuditLog{
			Table:    "sessions",
			RowID:    session.SessionID,
			Action:   "UPDATE",
			Field:    utils.StrPtr("status"),
			OldValue: utils.StrPtr(repo.SessionStatusActive),
			NewValue: utils.StrPtr(repo.SessionStatusEnded),
			ByWho:    req.DecidedBy,
		}); err != nil {
			return err
		}

		// reviewer 重表态：评审轮重入新轮（review/respond 相位；其余相位 no-op）
		reentered, round, err := s.reenterIssueNodeRoundTx(ctx, tx, locked)
		if err != nil {
			return err
		}

		resp.Ok = true
		resp.CommentId = locked.IssueID
		resp.Verdict = req.Verdict
		resp.SessionId = session.SessionID
		resp.Reentered = reentered
		resp.Round = round
		resp.Issue = issueItem(&row)
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return resp, nil
}

// Ruling 仲裁结论（裁决回填字段集；OverrideIssueTx/applyUpholdTx 共用入参）。
type Ruling struct {
	// Verdict 仲裁结论：uphold | override
	Verdict string
	// Conclusion 裁决结论文本（裁决条目 text 与 override 路径 resolve_note 同源）
	Conclusion string
	// DecidedBy 仲裁人（审计 by_who；恒 user）
	DecidedBy string
	// SessionID 关联裁决 session 业务 id
	SessionID string
	// At 裁决时间，unix 毫秒
	At int64
}

// OverrideIssueTx 仲裁推翻（wb_override 语义收编，spec §1.2 移出清单口径）。
//
// 与 wb_decide/wb_override 的关系（GoDoc 契约）：wb_decide=节点级门禁终审，已收编为
// User 拍板专用路由 wb_decision（T8 ReviewService.Decide，spec §1.2「内部 collect
// 判定 + User 拍板专用路由」）；wb_override=issue 级强权推翻关闭，被冲突上升取代
// （spec §1.2「wb_override（escalate 取代）」），**不暴露 agent/HTTP 端点**——语义
// 保留为本方法，供 wb_adjudicate（User 仲裁链）内部复用：verdict=override 时由
// 裁决链调用落 issue 终态。
//
// 单事务内完成（调用方持有事务与 issue 行锁）：replies 追加裁决条目（verdict/
// conclusion/sessionId）→ escalate 复位 0 → overridden 终态 + resolve 族
// （resolve_by=仲裁人、resolve_path=ruling、resolve_note=结论）+ 确认内嵌
// （resolve_reviewer_confirmed=1 + resolved_by/at）→ 乐观锁更新 + 逐字段审计。
// 入参 issue 为行锁读取副本，方法内原位更新（出参口径）。
//
// 错误集：ErrOptimisticLock（基线冲突）；replies JSON 非法（数据损坏）；
// 底层 IO 错误经 Trace 上抛。
func (s *EscalationService) OverrideIssueTx(ctx context.Context, tx *gormio.DB, issue *repo.Issue, ruling Ruling) error {
	entry := adjudicationReply(ruling)
	repliesJSON, replyID, err := domain.AppendIssueReply(issue.Replies, entry)
	if err != nil {
		return errs.Trace(err)
	}
	updates := map[string]any{
		"replies":                    repliesJSON,
		"escalate":                   int64(0),
		"state":                      repo.IssueStateOverridden,
		"resolve_by":                 ruling.DecidedBy,
		"resolve_at":                 ruling.At,
		"resolve_path":               domain.IssueResolvePathRuling,
		"resolve_note":               ruling.Conclusion,
		"resolve_reviewer_confirmed": int64(1),
		"resolved_by":                ruling.DecidedBy,
		"resolved_at":                ruling.At,
	}
	audits := []repo.AuditLog{
		issueAudit(issue.IssueID, "replies", "", replyID, ruling.DecidedBy),
		issueAudit(issue.IssueID, "escalate", "1", "0", ruling.DecidedBy),
		issueAudit(issue.IssueID, "state", issue.State, repo.IssueStateOverridden, ruling.DecidedBy),
	}
	if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Issue{}, issue.ID, issue.UpdatedAt, updates); err != nil {
		return err
	}
	for _, audit := range audits {
		if err := s.repo.InsertAuditLogWith(tx, &audit); err != nil {
			return err
		}
	}
	// 出参口径同步（updated_at 保持读取基线，与 T7 transitionIssue 一致）
	issue.Replies = repliesJSON
	issue.Escalate = 0
	issue.State = repo.IssueStateOverridden
	issue.ResolveBy = &ruling.DecidedBy
	issue.ResolveAt = &ruling.At
	issue.ResolvePath = utils.StrPtr(domain.IssueResolvePathRuling)
	issue.ResolveNote = &ruling.Conclusion
	issue.ResolveReviewerConfirmed = 1
	issue.ResolvedBy = &ruling.DecidedBy
	issue.ResolvedAt = &ruling.At
	return nil
}

// applyUpholdTx 维持议题路径落库（escalate 复位 + 裁决条目回填单次乐观锁更新；
// issue 状态保持——回正常澄清/修复链，resolve 族保持 NULL=未解决语义）。
// 返回追加后的 replies JSON 文本（出参口径）。
//
// 错误集：ErrOptimisticLock；replies JSON 非法（数据损坏）；底层 IO 错误经 Trace 上抛。
func (s *EscalationService) applyUpholdTx(ctx context.Context, tx *gormio.DB, locked *repo.Issue, ruling Ruling) (string, error) {
	repliesJSON, replyID, err := domain.AppendIssueReply(locked.Replies, adjudicationReply(ruling))
	if err != nil {
		return "", errs.Trace(err)
	}
	if err := repo.UpdateWithOptimisticLockTx(ctx, tx, &repo.Issue{}, locked.ID, locked.UpdatedAt, map[string]any{
		"escalate": int64(0),
		"replies":  repliesJSON,
	}); err != nil {
		return "", err
	}
	audits := []repo.AuditLog{
		issueAudit(locked.IssueID, "escalate", "1", "0", ruling.DecidedBy),
		issueAudit(locked.IssueID, "replies", "", replyID, ruling.DecidedBy),
	}
	for _, audit := range audits {
		if err := s.repo.InsertAuditLogWith(tx, &audit); err != nil {
			return "", err
		}
	}
	return repliesJSON, nil
}

// adjudicationReply 裁决条目装配（kind=adjudication，verdict/sessionId 承载）。
func adjudicationReply(ruling Ruling) repo.IssueReply {
	return repo.IssueReply{
		By:        ruling.DecidedBy,
		RoleID:    ruling.DecidedBy,
		Text:      ruling.Conclusion,
		At:        ruling.At,
		Kind:      domain.ReplyKindAdjudication,
		Verdict:   ruling.Verdict,
		SessionID: ruling.SessionID,
	}
}

// writeEscalationNoticeTx 通知位行落库（issues issue_type=notification，同事务；
// AC「阈值自动 escalate + 通知位」口径——收件箱读侧 T11 消费）：
// assignee=user（仲裁责任人路由）、author/role_id=by_who（自动上升=plugin、
// 手动=operator）、escalate=0（通知行本身非冲突卡聚合对象）、state=open。
// 返回通知行 issue_id（cm-<ts> 取号；唯一键冲突由调用方整体重试兜底）。
//
// 错误集：ErrIdempotentConflict 语义冲突由调用方重试；底层 IO 错误经 Trace 上抛。
func writeEscalationNoticeTx(ctx context.Context, tx *gormio.DB, r *repo.Repo, source *repo.Issue, rouseText string, byWho string) (string, error) {
	issueID, err := r.AllocIssueIDTx(ctx, tx)
	if err != nil {
		return "", err
	}
	notice := &repo.Issue{
		IssueID:       issueID,
		RequirementID: source.RequirementID,
		NodeID:        source.NodeID,
		AnchorKind:    domain.IssueAnchorDocument,
		Author:        byWho,
		RoleID:        byWho,
		IssueType:     domain.IssueTypeNotification,
		Assignee:      utils.StrPtr(repo.SessionRoleAdjudicator),
		Text:          domain.BuildEscalationNoticeText(source.IssueID, rouseText),
		State:         repo.IssueStateOpen,
		Replies:       "[]",
	}
	if err := tx.WithContext(ctx).Create(notice).Error; err != nil {
		return "", errs.Trace(err)
	}
	if err := r.InsertAuditLogWith(tx, &repo.AuditLog{
		Table:  "issues",
		RowID:  issueID,
		Action: "INSERT",
		ByWho:  byWho,
	}); err != nil {
		return "", err
	}
	return issueID, nil
}

// activeAdjudicationTx 定位 issue 当前裁决 session 并行锁判定活性：
// replies 最新上升条目携 sessionId → sessions 行锁读取。返回 (session, active)：
// session=nil=未拉起；active=false=已终局（ended/disposed）。并发双裁决由行锁
// 串行化——首例将 status 落 ended，次例读即 active=false。
//
// 错误集：底层 IO/JSON 错误经 Trace 上抛。
func (s *EscalationService) activeAdjudicationTx(ctx context.Context, tx *gormio.DB, locked *repo.Issue) (*repo.Session, bool, error) {
	entry, ok, err := domain.LatestReplyOfKind(locked.Replies, domain.ReplyKindEscalation)
	if err != nil {
		return nil, false, errs.Trace(err)
	}
	if !ok || entry.SessionID == "" {
		return nil, false, nil
	}
	session, err := s.repo.GetSessionBySessionIDLockTx(ctx, tx, entry.SessionID)
	if err != nil {
		return nil, false, err
	}
	if session == nil {
		return nil, false, nil
	}
	return session, session.Status == repo.SessionStatusActive, nil
}

// reenterIssueNodeRoundTx reviewer 重表态（裁决链第 4 拍）：issue 挂载节点处于
// review/respond 相位时评审轮重入新轮（ReviewService.reenterReviewRoundLocked
// ——agree 携结论、unagree 复位 reviewing，节点回 review），返回 (是否拉起, 轮)；
// 节点缺失/其余相位（exec/entry/未进入编排）no-op 不阻断回填（issue 可挂载于
// 非评审相位节点，如 CR 内嵌开发节点），返回当前轮回显。
//
// 错误集：review_results/reviewers JSON 非法（实例数据损坏）；ErrOptimisticLock；
// 底层 IO 错误经 Trace 上抛。
func (s *EscalationService) reenterIssueNodeRoundTx(ctx context.Context, tx *gormio.DB, locked *repo.Issue) (bool, int64, error) {
	node, err := repo.GetFlowNodeByRequirementAndNodeIDLockTx(ctx, tx, locked.RequirementID, locked.NodeID)
	if err != nil {
		return false, 0, err
	}
	if node == nil {
		return false, 0, nil
	}
	var entries []repo.ReviewResult
	if err := utils.UnmarshalJSONText(node.ReviewResults, &entries); err != nil {
		return false, 0, errs.Trace(err)
	}
	currentRound := domain.CurrentRound(entries)
	if node.State != repo.PhaseReview && node.State != repo.PhaseRespond {
		return false, currentRound, nil
	}
	if _, err := NewReviewService(s.repo).reenterReviewRoundLocked(ctx, tx, node); err != nil {
		return false, 0, err
	}
	return true, currentRound + 1, nil
}
