import type { ApiClient } from "./client";
import {
  AdvanceVersionReq,
  AdvanceVersionResp,
  CreateVersionReq,
  CreateVersionResp,
  DeprecateVersionReq,
  DeprecateVersionResp,
  ListVersionsResp,
  UpdateVersionReq,
  UpdateVersionResp,
} from "./proto/api";

export function listVersions(client: ApiClient): Promise<ListVersionsResp> {
  return client.request({
    method: "GET",
    path: "/api/versions",
    decode: (raw) => ListVersionsResp.fromJSON(raw),
  });
}

export function createVersion(
  client: ApiClient,
  req: CreateVersionReq,
): Promise<CreateVersionResp> {
  return client.request({
    method: "POST",
    path: "/api/versions",
    body: CreateVersionReq.toJSON(req),
    decode: (raw) => CreateVersionResp.fromJSON(raw),
  });
}

/** 改名（乐观锁：updated_at 必回传 GET 所得基线；状态迁移走 advance/deprecate）。 */
export function updateVersion(
  client: ApiClient,
  versionId: string,
  req: UpdateVersionReq,
): Promise<UpdateVersionResp> {
  return client.request({
    method: "PUT",
    path: `/api/versions/${encodeURIComponent(versionId)}`,
    body: UpdateVersionReq.toJSON(req),
    decode: (raw) => UpdateVersionResp.fromJSON(raw),
  });
}

/** 发布（软约束：resp.unfinished 仅提示不拦截）。 */
export function advanceVersion(
  client: ApiClient,
  versionId: string,
  req: AdvanceVersionReq,
): Promise<AdvanceVersionResp> {
  return client.request({
    method: "POST",
    path: `/api/versions/${encodeURIComponent(versionId)}/advance`,
    body: AdvanceVersionReq.toJSON(req),
    decode: (raw) => AdvanceVersionResp.fromJSON(raw),
  });
}

/** 废弃=终态（有未完成需求引用时后端 blocked:true）。 */
export function deprecateVersion(
  client: ApiClient,
  versionId: string,
  req: DeprecateVersionReq,
): Promise<DeprecateVersionResp> {
  return client.request({
    method: "POST",
    path: `/api/versions/${encodeURIComponent(versionId)}/deprecate`,
    body: DeprecateVersionReq.toJSON(req),
    decode: (raw) => DeprecateVersionResp.fromJSON(raw),
  });
}
