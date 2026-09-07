import type { ApiClient } from "./client";
import { DecisionReq, DecisionResp, ReviewReq, ReviewResp } from "./proto/api";

/** 评审/拍板端点（T8）。UI 本工单只做只读态呈现；封装入库供 T16 拍板卡三态直接消费。 */

/** wb_review：agent 自判表态（同轮同角色覆写取最后）。 */
export function postReview(
  client: ApiClient,
  requirementId: string,
  nodeId: string,
  req: Omit<ReviewReq, "requirementId" | "nodeId">,
): Promise<ReviewResp> {
  const body: ReviewReq = { ...req, requirementId, nodeId };
  return client.request({
    method: "POST",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/nodes/${encodeURIComponent(nodeId)}/review`,
    body: ReviewReq.toJSON(body),
    decode: (raw) => ReviewResp.fromJSON(raw),
  });
}

/** wb_decision：User 终审拍板（仅 user_gate=1 节点、评审轮全 agree 后可拍；重复拍板 noop）。 */
export function postDecision(
  client: ApiClient,
  requirementId: string,
  nodeId: string,
  req: Omit<DecisionReq, "requirementId" | "nodeId">,
): Promise<DecisionResp> {
  const body: DecisionReq = { ...req, requirementId, nodeId };
  return client.request({
    method: "POST",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/nodes/${encodeURIComponent(nodeId)}/decision`,
    body: DecisionReq.toJSON(body),
    decode: (raw) => DecisionResp.fromJSON(raw),
  });
}
