import type { ApiClient } from "./client";
import { DependencyTreeResp, MyTaskReq, MyTaskResp } from "./proto/api";

/**
 * 流程实例端点（T4）：wb_my_task 三向校验（role × node × 相位）。
 * 详情页以 Web 用户身份（role=user）消费 node 摘要 / current_phase / activeActions /
 * review_summary / relatedTasks——不属于该相位该角色时动作集为空集（不报错）。
 */
export function myTask(
  client: ApiClient,
  requirementId: string,
  role: string,
  nodeId?: string,
): Promise<MyTaskResp> {
  const req: MyTaskReq = { requirementId, role, nodeId: nodeId ?? "" };
  return client.request({
    method: "POST",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/my-task`,
    body: MyTaskReq.toJSON(req),
    decode: (raw) => MyTaskResp.fromJSON(raw),
  });
}

/** 依赖树只读端点（分析师视图；原型 P9 版本依赖树按需拉取逐需求装配）。 */
export function dependencyTree(
  client: ApiClient,
  requirementId: string,
): Promise<DependencyTreeResp> {
  return client.request({
    method: "GET",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/dependencies/tree`,
    decode: (raw) => DependencyTreeResp.fromJSON(raw),
  });
}
