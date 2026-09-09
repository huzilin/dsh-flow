import type { ApiClient } from "./client";
import {
  CommentAnchor,
  CommentReq,
  CommentResp,
  ConfirmReviewReq,
  ConfirmReviewResp,
  IssueItem,
  ReopenReq,
  ReopenResp,
  ReplyReq,
  ReplyResp,
  ResolveReq,
  ResolveResp,
} from "./proto/api";

/** 评论/issue 五端点（T7）封装。操作者约束（后端裁决，UI 按同口径收敛可见性）：
 *  resolve 须=assignee；confirm/reopen 须=提评者（reviewerRole 优先，缺省 author）。 */

/** wb_comment：四级锚点评论（anchor 未填充=document 级）。 */
export function postComment(
  client: ApiClient,
  requirementId: string,
  req: Omit<CommentReq, "requirementId">,
): Promise<CommentResp> {
  const body: CommentReq = { ...req, requirementId };
  return client.request({
    method: "POST",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/comments`,
    body: CommentReq.toJSON(body),
    decode: (raw) => CommentResp.fromJSON(raw),
  });
}

/** wb_reply：追加回复（roundCount++）；非法状态迁移 409 INVALID_TRANSITION。 */
export function postReply(
  client: ApiClient,
  commentId: string,
  req: Omit<ReplyReq, "commentId">,
): Promise<ReplyResp> {
  const body: ReplyReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/replies`,
    body: ReplyReq.toJSON(body),
    decode: (raw) => ReplyResp.fromJSON(raw),
  });
}

/** wb_resolve：path=fix（修复关闭申报）| agree（协商一致直落 resolved）。 */
export function postResolve(
  client: ApiClient,
  commentId: string,
  req: Omit<ResolveReq, "commentId">,
): Promise<ResolveResp> {
  const body: ResolveReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/resolve`,
    body: ResolveReq.toJSON(body),
    decode: (raw) => ResolveResp.fromJSON(raw),
  });
}

/** wb_confirm_review：提评者复核（pending-confirm/fixed→resolved）。 */
export function postConfirmReview(
  client: ApiClient,
  commentId: string,
  req: Omit<ConfirmReviewReq, "commentId">,
): Promise<ConfirmReviewResp> {
  const body: ConfirmReviewReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/confirm`,
    body: ConfirmReviewReq.toJSON(body),
    decode: (raw) => ConfirmReviewResp.fromJSON(raw),
  });
}

/** wb_reopen：澄清 reopen（pending-confirm→open）/ 修复 revise（fixed→to-fix），loopCount++。 */
export function postReopen(
  client: ApiClient,
  commentId: string,
  req: Omit<ReopenReq, "commentId">,
): Promise<ReopenResp> {
  const body: ReopenReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/reopen`,
    body: ReopenReq.toJSON(body),
    decode: (raw) => ReopenResp.fromJSON(raw),
  });
}

/** 锚点装配（纯函数，独立导出便于测试）：阅读器选区 → CommentAnchor。
 *  行号 1 起；document 不带行号族字段且 artifactId 空（后端 domain.ValidateIssueAnchor 同口径）。 */
export function assembleCommentAnchor(input: {
  kind: "document" | "line" | "range" | "block";
  artifactId?: string;
  line?: number;
  startLine?: number;
  endLine?: number;
  blockStart?: number;
  blockEnd?: number;
  lang?: string;
}): CommentAnchor {
  const n = (v: number | undefined): string => (typeof v === "number" && v > 0 ? String(v) : "0");
  return {
    kind: input.kind,
    artifactId: input.artifactId ?? "",
    line: input.kind === "line" ? n(input.line) : "0",
    startLine: input.kind === "range" ? n(input.startLine) : "0",
    endLine: input.kind === "range" ? n(input.endLine) : "0",
    blockStart: input.kind === "block" ? n(input.blockStart) : "0",
    blockEnd: input.kind === "block" ? n(input.blockEnd) : "0",
    lang: input.kind === "block" ? (input.lang ?? "") : "",
  };
}

/** 评论按锚点位置排序键（阅读器锚定渲染：行号升序，document 恒末尾）。 */
export function issueAnchorOrder(issue: IssueItem): number {
  const end = Number(issue.anchorEndLine) || Number(issue.anchorLine) || Number(issue.anchorBlockEnd);
  return end > 0 ? end : Number.MAX_SAFE_INTEGER;
}
