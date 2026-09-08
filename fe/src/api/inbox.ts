import type { ApiClient } from "./client";
import {
  AdjudicateReq,
  AdjudicateResp,
  EscalateReq,
  EscalateResp,
  ListInboxResp,
} from "./proto/api";

/** 收件箱/TODO（T11 GET /api/inbox）+ 冲突上升裁决链（T12 wb_escalate / wb_adjudicate）端点封装。 */

export interface ListInboxParams {
  /** 收件角色过滤（通知按 issues.assignee、TODO 按 tasks.role_id）；空=不过滤 */
  role?: string;
  /** 已读过滤：""=全部 / unread=未读 / read=已读（越集 400） */
  readFilter?: string;
  /** 通知类别（七类枚举单值）；空=全部 */
  kind?: string;
  /** 按需求过滤；空=跨需求全量 */
  requirementId?: string;
  /** 页码 1 起 */
  page?: number;
  /** 页大小；0=默认 20 */
  pageSize?: number;
}

/** wb_list_inbox：通知列表（未读优先）+ TODO 投影同响应双通道。 */
export function listInbox(
  client: ApiClient,
  params: ListInboxParams = {},
): Promise<ListInboxResp> {
  return client.request({
    method: "GET",
    path: "/api/inbox",
    query: {
      role: params.role,
      read_filter: params.readFilter,
      kind: params.kind,
      requirement_id: params.requirementId,
      page: params.page,
      page_size: params.pageSize,
    },
    decode: (raw) => ListInboxResp.fromJSON(raw),
  });
}

/** wb_escalate：手动上升裁决（冲突任一方；幂等=已有进行中裁决 session 时 noop=true）。 */
export function postEscalate(
  client: ApiClient,
  commentId: string,
  req: Omit<EscalateReq, "commentId">,
): Promise<EscalateResp> {
  const body: EscalateReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/escalate`,
    body: EscalateReq.toJSON(body),
    decode: (raw) => EscalateResp.fromJSON(raw),
  });
}

/** wb_adjudicate：User 仲裁结论回填（uphold=维持议题 / override=仲裁推翻；
 *  escalate=0 时 409 GATE_BLOCKED；结论必填）。 */
export function postAdjudicate(
  client: ApiClient,
  commentId: string,
  req: Omit<AdjudicateReq, "commentId">,
): Promise<AdjudicateResp> {
  const body: AdjudicateReq = { ...req, commentId };
  return client.request({
    method: "POST",
    path: `/api/comments/${encodeURIComponent(commentId)}/adjudicate`,
    body: AdjudicateReq.toJSON(body),
    decode: (raw) => AdjudicateResp.fromJSON(raw),
  });
}
