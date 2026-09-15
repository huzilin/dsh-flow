import { useState } from "react";
import { Link } from "react-router-dom";
import type { FlowNodeItem, IssueItem } from "../api/proto/api";
import { ADJUDICATE_VERDICT_LABELS, WEB_USER_ROLE } from "../api/constants";
import { issueReviewerOf, parseReplies, unagreeSeatsOf } from "../domain/flowview";
import { useAdjudicateMutation, useEscalateMutation } from "../hooks/queries";
import { formatTime } from "../lib/time";
import { AlertBanner } from "./AlertBanner";
import { IssueStateBadge } from "./Badges";

/**
 * 冲突卡（T16 / ui-interaction-final §1.4，取代 override 强权）：
 * - escalate=1 的 issue 聚合呈现于节点详情意见区顶部（NodeDetail 注入）；
 *   全局冲突入口（InboxPage 冲突待裁决区）跳转至本卡锚点。
 * - 正反对照两栏：正方=评审位反对表态（review_results 最新 unagree，数据源一）+
 *   提评者；反方=owner 响应回复（issue replies 非 voided 普通条目，数据源二）。
 * - 裁决动作（user 角色可见，Web 身份=WEB_USER_ROLE）：
 *   escalate=0 时「上升裁决」（wb_escalate 拉起，幂等 noop）；
 *   escalate=1 时 wb_adjudicate uphold|override 表单（结论必填）。
 * - 裁决回填：replies 追加 kind=adjudication 条目（数据层留痕、节点可见），
 *   回填后呈现结论条目 + 跳转（↗ 节点详情 / issue 锚点）；reviewer 重表态由后端联动。
 */
export function ConflictCard({
  requirementId,
  issue,
  node,
}: {
  requirementId: string;
  issue: IssueItem;
  /** 挂载节点（正方数据源 review_results；快照缺节点时正方列降级呈现提评者） */
  node?: FlowNodeItem;
}) {
  const replies = parseReplies(issue.replies);
  const escalated = issue.escalate === "1";
  // 回执条目（kind=adjudication）= 裁决回填留痕；普通条目=双方线程引用
  const adjudications = replies.filter((r) => r.kind === "adjudication");
  const threadReplies = replies.filter((r) => r.kind !== "adjudication" && r.kind !== "escalation" && !r.voided);
  const unagrees = node ? unagreeSeatsOf(node) : [];

  return (
    <article className="conflict-card" data-issue-id={issue.issueId} data-testid="conflict-card">
      <header className="comment-head">
        <span className="badge badge-escalate">⚔ 冲突卡</span>
        <IssueStateBadge state={issue.state} />
        {escalated && <span className="badge badge-unread">裁决进行中</span>}
        <span className="comment-meta">
          {issue.issueId} · 提评 {issueReviewerOf(issue)} · 回退 {issue.loopCount}
        </span>
        <Link
          className="button-link"
          to={`/requirements/${encodeURIComponent(requirementId)}?node=${encodeURIComponent(issue.nodeId)}&focus=issue&issue=${encodeURIComponent(issue.issueId)}`}
        >
          ↗ 节点详情
        </Link>
      </header>

      <div className="conflict-point">
        <b>分歧点：</b>
        {issue.text}
      </div>

      <div className="conflict-versus">
        <div className="conflict-side" data-testid="conflict-pro">
          <h6>正方 · 评审位（反对维持议题）</h6>
          {unagrees.length === 0 ? (
            <p className="muted small">快照内无反对表态记录（或未挂载节点）。</p>
          ) : (
            <ul>
              {unagrees.map((u) => (
                <li key={`${u.role}:${u.capabilityId}`}>
                  {u.role} · {u.capabilityId}（轮 {u.round}）：{issue.text}
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="conflict-side" data-testid="conflict-con">
          <h6>反方 · 响应方（处理/反驳回复）</h6>
          {threadReplies.length === 0 ? (
            <p className="muted small">暂无响应回复。</p>
          ) : (
            <ul>
              {threadReplies.map((r) => (
                <li key={r.replyId}>
                  {r.by}（{r.roleId}）· {formatTime(String(r.at))}：{r.text}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {adjudications.length > 0 && (
        <div className="conflict-adjudication" data-testid="conflict-adjudications">
          <h5>裁决回填（评论区留痕）</h5>
          <ul>
            {adjudications.map((r) => (
              <li key={r.replyId}>
                <span className={`badge badge-verdict-${r.verdict === "override" ? "rejected" : "approved"}`}>
                  {ADJUDICATE_VERDICT_LABELS[r.verdict ?? ""] ?? r.verdict}
                </span>{" "}
                {r.text}{" "}
                <span className="muted small">
                  {r.by}（{r.roleId}）· {formatTime(String(r.at))}
                  {r.sessionId !== "" && ` · session ${r.sessionId}`}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <ConflictActions requirementId={requirementId} issue={issue} escalated={escalated} />
    </article>
  );
}

/** 裁决动作区（user 角色可见）：wb_escalate 拉起 / wb_adjudicate uphold|override 表单。 */
function ConflictActions({
  requirementId,
  issue,
  escalated,
}: {
  requirementId: string;
  issue: IssueItem;
  escalated: boolean;
}) {
  const escalate = useEscalateMutation(requirementId);
  const adjudicate = useAdjudicateMutation(requirementId);
  const [reason, setReason] = useState("");
  const [verdict, setVerdict] = useState<"uphold" | "override">("uphold");
  const [conclusion, setConclusion] = useState("");
  const busy = escalate.isPending || adjudicate.isPending;
  const error = escalate.error ?? adjudicate.error;

  return (
    <div className="conflict-adjudication" data-testid="conflict-actions">
      {escalated ? (
        <form
          className="adjudicate-form"
          onSubmit={(e) => {
            e.preventDefault();
            if (conclusion.trim() === "") return;
            adjudicate.mutate(
              { issue, verdict, conclusion },
              { onSuccess: () => setConclusion("") },
            );
          }}
        >
          <label>
            仲裁结论
            <select
              aria-label="仲裁结论"
              value={verdict}
              onChange={(e) => setVerdict(e.target.value as "uphold" | "override")}
            >
              <option value="uphold">{ADJUDICATE_VERDICT_LABELS.uphold}（回正常澄清/修复链）</option>
              <option value="override">{ADJUDICATE_VERDICT_LABELS.override}（issue 落已裁决终态）</option>
            </select>
          </label>
          <textarea
            aria-label="裁决结论"
            placeholder="裁决结论（必填；回填 issue 评论区留痕）"
            value={conclusion}
            onChange={(e) => setConclusion(e.target.value)}
            rows={2}
          />
          <button type="submit" className="primary" disabled={busy || conclusion.trim() === ""}>
            提交仲裁（{WEB_USER_ROLE}）
          </button>
        </form>
      ) : (
        <div className="adjudicate-form">
          <input
            aria-label="上升理由"
            placeholder="上升理由（可选）"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <button
            type="button"
            disabled={busy}
            title="escalate 置位并拉起裁决 session（幂等：已有进行中 session 时 noop）"
            onClick={() => escalate.mutate({ issue, reason }, { onSuccess: () => setReason("") })}
          >
            ⚔ 上升裁决（wb_escalate）
          </button>
          {escalate.isSuccess && (
            <span role="status" className="field-hint">
              {escalate.data.noop
                ? `已有进行中裁决 session（${escalate.data.sessionId}），幂等 no-op。`
                : `已拉起裁决 session（${escalate.data.sessionId}）。`}
            </span>
          )}
        </div>
      )}

      {adjudicate.isSuccess && (
        <div role="status" className="field-hint">
          仲裁已回填：{ADJUDICATE_VERDICT_LABELS[adjudicate.data.verdict] ?? adjudicate.data.verdict}
          {adjudicate.data.reentered
            ? ` · reviewer 重表态轮已拉起（第 ${adjudicate.data.round} 轮）`
            : " · 挂载节点非评审/响应相位，轮重入 no-op"}
          {" — "}
          <Link
            className="button-link"
            to={`/requirements/${encodeURIComponent(requirementId)}?node=${encodeURIComponent(issue.nodeId)}&focus=issue&issue=${encodeURIComponent(issue.issueId)}`}
          >
            ↗ 查看 issue 所在节点
          </Link>
        </div>
      )}
      {error && <AlertBanner error={error} />}
    </div>
  );
}
