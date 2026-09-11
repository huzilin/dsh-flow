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
 *   （NAM-09 拆分路由：澄清侧 clarified / 修复侧 declare-fixed，由 issue.state 选择）
 * - settle / qualify：须=assignee；qualify 定性打回，settle 协商一致关闭。
 * - confirm / reopen / revise：须=提评者（reviewerRole 优先，缺省 author）。
 */
export function CommentCard({
  requirementId,
  issue,
  showAnchor = true,
  highlight = false,
  artifactLabels,
}: {
  requirementId: string;
  issue: IssueItem;
  /** 阅读器锚定场景已带产物语境，可隐藏锚点摘要行 */
  showAnchor?: boolean;
  /** 收件箱跳转锚点高亮（T16：focus=issue 定位） */
  highlight?: boolean;
  /** BUG-013：锚定产物名映射（artifactId→label），锚点摘要以产物名替代裸 id */
  artifactLabels?: Record<string, string>;
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
  // 已解决评论默认折叠（一版 client.js:245-252 回归——inventory A4-11/B-43）：
  // resolved 一行摘要（角色+状态徽标+截断文本）可展开回全文；open/pending-confirm/
  // to-fix/fixed 恒展开（overridden 态已废除，override 动作置 resolved）。
  const collapsible = issue.state === "resolved";
  const [expanded, setExpanded] = useState(!collapsible || highlight);
  // settle/qualify 入口状态域：open/pending-confirm 定性 to-fix；to-fix/fixed 重申报修复关闭；
  // settle 直落 resolved（后端裁决，UI 按 domain 口径收敛；09-12 裁决 overridden 态废除）
  const canResolve = isAssignee && issue.state !== "resolved";
  const canConfirm = isReviewer && (issue.state === "pending-confirm" || issue.state === "fixed");
  const canReopen = isReviewer && (issue.state === "pending-confirm" || issue.state === "fixed");
  const busy = reply.isPending || resolve.isPending || confirm.isPending || reopen.isPending;
  const actionError = reply.error ?? resolve.error ?? confirm.error ?? reopen.error;

  // 折叠态：一行摘要（状态徽标+角色+截断文本+展开箭头），点击展开全文 thread
  if (!expanded) {
    return (
      <article
        className="comment-card is-collapsed"
        data-issue-id={issue.issueId}
        data-state={issue.state}
        data-focus-flash={highlight ? "1" : undefined}
      >
        <button
          type="button"
          className="comment-summary"
          aria-expanded={false}
          aria-label={`展开已解决评论：${issue.text}`}
          onClick={() => setExpanded(true)}
        >
          <IssueStateBadge state={issue.state} />
          <span className="comment-meta">
            {issue.author}（{issue.roleId}）
          </span>
          <span className="comment-summary-text">{issue.text}</span>
          <span className="muted" aria-hidden>
            ▸
          </span>
        </button>
      </article>
    );
  }

  return (
    <article
      className={`comment-card${highlight ? " focus-flash" : ""}`}
      data-issue-id={issue.issueId}
      data-state={issue.state}
      data-focus-flash={highlight ? "1" : undefined}
    >
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
        {collapsible && (
          <button
            type="button"
            className="comment-collapse"
            aria-expanded
            aria-label="收起评论"
            onClick={() => setExpanded(false)}
          >
            收起 ▴
          </button>
        )}
      </header>
      {showAnchor && (
        <div className="comment-anchor muted">
          锚点：
          {anchorSummary(
            issue,
            artifactLabels ? (id) => artifactLabels[id] : undefined,
          )}
        </div>
      )}
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
              title="qualify：定性 to-fix 或重申报修复关闭"
              onClick={() => resolve.mutate({ issue, path: "fix", note })}
            >
              定性打回（qualify）
            </button>
            <button
              type="button"
              disabled={busy}
              title="settle：协商一致直接落 resolved"
              onClick={() => resolve.mutate({ issue, path: "agree", note })}
            >
              协商一致关闭（settle）
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
