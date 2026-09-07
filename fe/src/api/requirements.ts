import type { ApiClient } from "./client";
import {
  CreateRequirementReq,
  CreateRequirementResp,
  DeleteRequirementResp,
  GetRequirementResp,
  ListRequirementsResp,
  UpdateRequirementReq,
  UpdateRequirementResp,
} from "./proto/api";

/** GET /api/requirements 查询参数（query 名与后端 c.Query 键一致，snake_case）。 */
export interface ListRequirementsParams {
  /** 按版本筛选；空=不过滤 */
  versionId?: string;
  /** true=仅看待分配池（与 versionId 互斥，同给时后端 backlog 优先） */
  backlog?: boolean;
  priority?: string;
  status?: string;
  /** 页码 1 起；0 视为 1 */
  page?: number;
  /** 页大小；0 视为默认 20，后端截断上限 100 */
  pageSize?: number;
}

/** 组装 GET /api/requirements 的 query（独立导出便于单测）。 */
export function buildListRequirementsQuery(params: ListRequirementsParams): string {
  const search = new URLSearchParams();
  if (params.backlog) {
    search.set("backlog", "true");
  } else if (params.versionId) {
    search.set("version_id", params.versionId);
  }
  if (params.priority) search.set("priority", params.priority);
  if (params.status) search.set("status", params.status);
  if (params.page !== undefined) search.set("page", String(params.page));
  if (params.pageSize !== undefined) search.set("page_size", String(params.pageSize));
  const s = search.toString();
  return s === "" ? "" : `?${s}`;
}

export function listRequirements(
  client: ApiClient,
  params: ListRequirementsParams,
): Promise<ListRequirementsResp> {
  return client.request({
    method: "GET",
    path: "/api/requirements",
    query: {
      version_id: params.backlog ? undefined : params.versionId,
      backlog: params.backlog ? "true" : undefined,
      priority: params.priority,
      status: params.status,
      page: params.page,
      page_size: params.pageSize,
    },
    decode: (raw) => ListRequirementsResp.fromJSON(raw),
  });
}

export function getRequirement(
  client: ApiClient,
  requirementId: string,
): Promise<GetRequirementResp> {
  return client.request({
    method: "GET",
    path: `/api/requirements/${encodeURIComponent(requirementId)}`,
    decode: (raw) => GetRequirementResp.fromJSON(raw),
  });
}

// query 组装复用同一函数（request 的 query 走通用 buildQuery，这里保持一致性由测试兜底）
export function createRequirement(
  client: ApiClient,
  req: CreateRequirementReq,
): Promise<CreateRequirementResp> {
  return client.request({
    method: "POST",
    path: "/api/requirements",
    body: CreateRequirementReq.toJSON(req),
    decode: (raw) => CreateRequirementResp.fromJSON(raw),
  });
}

export function updateRequirement(
  client: ApiClient,
  requirementId: string,
  req: UpdateRequirementReq,
): Promise<UpdateRequirementResp> {
  return client.request({
    method: "PUT",
    path: `/api/requirements/${encodeURIComponent(requirementId)}`,
    body: UpdateRequirementReq.toJSON(req),
    decode: (raw) => UpdateRequirementResp.fromJSON(raw),
  });
}

export function deleteRequirement(
  client: ApiClient,
  requirementId: string,
  confirm: boolean,
): Promise<DeleteRequirementResp> {
  return client.request({
    method: "DELETE",
    path: `/api/requirements/${encodeURIComponent(requirementId)}`,
    query: { confirm: confirm ? "true" : undefined },
    decode: (raw) => DeleteRequirementResp.fromJSON(raw),
  });
}
