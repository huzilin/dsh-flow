import { useState } from "react";
import type { IssueItem } from "../api/proto/api";
import { RESOLVE_PATH_LABELS, WEB_USER_ROLE } from "../api/constants";
import {
  useConfirmReviewMutation,
  useReopenMutation,
  useReplyMutation,
  useResolveMutation,
} from "../hooks/queries";
import { anchorSummary, issueReviewerOf, parseReplies } from "../domain/flowview";
import { formatTime } from "../pages/RequirementPoolPage";
import { IssueStateBadge } from "./Badges";

/**
 * 评论卡（issue 状态机 v2 展示 + 操作矩阵）。
 * 操作者约束与后端 T7 同口径收敛可见性（后端仍是真相源，409 INVALID_TRANSITION 兜底）：
 * - 回复：任何身份可回（Web 身份=WEB_USER_ROLE）；误发 reply 可标注作废。
 * - resolve：须=assignee；path=fix（修复关闭申报）| agree（协商一致关闭）。
 * - confirm / reopen：须=提评者（reviewerRole 优先，缺省 author）。
 */
export function CommentCard({
  requirementId,
  issue,
  showAnchor = true,
}: {
  requirementId: string;
  issue: IssueItem;
  /** 阅读器锚定场景已带产物语境，可隐藏锚点摘要行 */
  showAnchor?: boolean;
}) {
  const reply = useReplyMutation(requirementId);
  const resolve = useResolveMutation(requirementId);
  const confirm = useConfirmReviewMutation(requirementId);
  const reopen = useReopenMutation(requirementId);
  const [replyOpen, setReplyOpen] = useState(false);
  const [replyText, setReplyText] = useState("");
  const [voided, setVoided] = useState(false);
  const [note, setNote] = useState("");

  const replies = parseReplies(issue.replies);
  const isAssignee = issue.assignee === WEB_USER_ROLE;
  const isReviewer = issueReviewerOf(issue) === WEB_USER_ROLE;
  // resolve 双路径入口状态域：open/pending-confirm 定性 to-fix；to-fix/fixed 修复关闭申报；
  // agree 直落 resolved（后端裁决，UI 按 domain 口径收敛）
  const canResolve = isAssignee && issue.state !== "resolved" && issue.state !== "overridden";
  const canConfirm = isReviewer && (issue.state === "pending-confirm" || issue.state === "fixed");
  const canReopen = isReviewer && (issue.state === "pending-confirm" || issue.state === "fixed");
  const busy = reply.isPending || resolve.isPending || confirm.isPending || reopen.isPending;
  const actionError = reply.error ?? resolve.error ?? confirm.error ?? reopen.error;

  return (
    <article className="comment-card" data-issue-id={issue.issueId} data-state={issue.state}>
      <header className="comment-head">
        <IssueStateBadge state={issue.state} />
        {issue.issueType !== "" && issue.issueType !== "issue" && (
          <span className="badge badge-issue-type">{issue.issueType}</span>
        )}
        {issue.escalate === "1" && <span className="badge badge-escalate">已上升裁决</span>}
        <span className="comment-meta">
          {issue.author}（{issue.roleId}） · {formatTime(issue.createdAt)} · 轮 {issue.roundCount}
          {issue.loopCount !== "0" && ` · 回退 ${issue.loopCount}`}
        </span>
      </header>
      {showAnchor && <div className="comment-anchor muted">锚点：{anchorSummary(issue)}</div>}
      <p className="comment-text">{issue.text}</p>
      {issue.resolvePath !== "" && (
        <div className="comment-resolve muted">
          {RESOLVE_PATH_LABELS[issue.resolvePath] ?? issue.resolvePath} · {issue.resolveBy}
          {issue.resolveNote !== "" && ` · ${issue.resolveNote}`}
          {issue.resolvedBy !== "" && ` · 复核确认：${issue.resolvedBy}`}
        </div>
      )}

      <ul className="comment-replies">
        {replies.map((r) => (
          <li key={r.replyId} className={r.voided ? "reply voided" : "reply"}>
            <span className="muted">
              {r.replyId} · {r.by}（{r.roleId}） · {formatTime(String(r.at))}
              {r.voided && " · 已作废"}
            </span>
            <span>{r.text}</span>
          </li>
        ))}
      </ul>

      <div className="comment-actions">
        {!replyOpen && (
          <button type="button" disabled={busy} onClick={() => setReplyOpen(true)}>
            回复
          </button>
        )}
        {canResolve && (
          <>
            <button
              type="button"
              disabled={busy}
              title="修复路径：定性 to-fix 或申报修复关闭"
              onClick={() => resolve.mutate({ issue, path: "fix", note })}
            >
              修复路径（fix）
            </button>
            <button
              type="button"
              disabled={busy}
              title="协商一致：直接落 resolved"
              onClick={() => resolve.mutate({ issue, path: "agree", note })}
            >
              协商一致关闭（agree）
            </button>
          </>
        )}
        {canConfirm && (
          <button type="button" disabled={busy} onClick={() => confirm.mutate(issue)}>
            复核确认关闭
          </button>
        )}
        {canReopen && (
          <button
            type="button"
            disabled={busy}
            onClick={() => reopen.mutate({ issue, text: note })}
          >
            reopen
          </button>
        )}
      </div>

      {replyOpen && (
        <div className="reply-form">
          <textarea
            aria-label="回复内容"
            placeholder="回复内容（必填）"
            value={replyText}
            onChange={(e) => setReplyText(e.target.value)}
            rows={2}
          />
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={voided}
              onChange={(e) => setVoided(e.target.checked)}
            />
            误发标注作废
          </label>
          <button
            type="button"
            className="primary"
            disabled={busy || replyText.trim() === ""}
            onClick={() =>
              reply.mutate(
                { issue, text: replyText, voided },
                {
                  onSuccess: () => {
                    setReplyOpen(false);
                    setReplyText("");
                    setVoided(false);
                  },
                },
              )
            }
          >
            发送回复
          </button>
        </div>
      )}
      {(canResolve || canReopen) && !replyOpen && (
        <input
          className="comment-note"
          aria-label="resolve 说明 / reopen 理由"
          placeholder="resolve 说明 / reopen 理由（可选）"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      )}
      {actionError && (
        <div role="alert" className="field-hint error">
          {actionError instanceof Error ? actionError.message : String(actionError)}
        </div>
      )}
    </article>
  );
}
