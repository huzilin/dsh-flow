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

/** 评论/issue 端点（T7，NAM-09 契约改名）封装。操作者约束（后端裁决，UI 按同口径
 *  收敛可见性）：settle/qualify 须=assignee；confirm/reopen/revise 须=提评者
 *  （reviewerRole 优先，缺省 author）。 */

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

/** wb_clarified：owner 澄清完成申报（open→pending-confirm，roundCount++）；非法状态迁移 409 INVALID_TRANSITION。 */
export function postClarified(
  client: ApiClient,
  commentId: string,
  req: Omit<ReplyReq, "commentId">,
): Promise<ReplyResp> {
  const body: ReplyReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/clarified`,
    body: ReplyReq.toJSON(body),
    decode: (raw) => ReplyResp.fromJSON(raw),
  });
}

/** wb_declare_fixed：owner 修复完成申报（to-fix→fixed，roundCount++）。 */
export function postDeclareFixed(
  client: ApiClient,
  commentId: string,
  req: Omit<ReplyReq, "commentId">,
): Promise<ReplyResp> {
  const body: ReplyReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/declare-fixed`,
    body: ReplyReq.toJSON(body),
    decode: (raw) => ReplyResp.fromJSON(raw),
  });
}

/** wb_settle：协商一致关闭（确认内嵌直落 resolved，operator=assignee）。 */
export function postSettle(
  client: ApiClient,
  commentId: string,
  req: Omit<ResolveReq, "commentId" | "path">,
): Promise<ResolveResp> {
  const body: ResolveReq = { ...req, commentId, path: "agree" };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/settle`,
    body: ResolveReq.toJSON(body),
    decode: (raw) => ResolveResp.fromJSON(raw),
  });
}

/** wb_qualify：复核定性真问题（→to-fix 触发打回；to-fix/fixed 重申报幂等），operator=assignee。 */
export function postQualify(
  client: ApiClient,
  commentId: string,
  req: Omit<ResolveReq, "commentId" | "path">,
): Promise<ResolveResp> {
  const body: ResolveReq = { ...req, commentId, path: "fix" };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/qualify`,
    body: ResolveReq.toJSON(body),
    decode: (raw) => ResolveResp.fromJSON(raw),
  });
}

/** wb_confirm：提评者复核确认（pending-confirm/fixed→resolved）。 */
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

/** wb_reopen：复核退回·未解释清楚（pending-confirm→open），loopCount++。 */
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

/** wb_revise：修复侧复核退回（fixed→to-fix），loopCount++。 */
export function postRevise(
  client: ApiClient,
  commentId: string,
  req: Omit<ReopenReq, "commentId">,
): Promise<ReopenResp> {
  const body: ReopenReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/revise`,
    body: ReopenReq.toJSON(body),
    decode: (raw) => ReopenResp.fromJSON(raw),
  });
}

/** 锚点装配（纯函数，独立导出便于测试）：阅读器选区 → CommentAnchor。
 *  行号 1 起；document 不带行号族字段；BUG-013 后评审上下文的 document 锚须携带
 *  artifactId（rd 域校验放行 document+artifact_id 组合），由调用方决定是否携带。 */
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
