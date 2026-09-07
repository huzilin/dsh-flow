import type { ApiClient } from "./client";
import { GetArtifactContentResp } from "./proto/api";

/**
 * 产物正文（T6）：wb_get_artifact_content——阅读器渲染与锚点定位元数据。
 * kind=document：content=docs root 下相对路径文件文本（缺失 404 NOT_FOUND）；
 * kind=link：content 恒空、href 即外链。
 */
export function getArtifactContent(
  client: ApiClient,
  requirementId: string,
  artifactId: string,
): Promise<GetArtifactContentResp> {
  return client.request({
    method: "GET",
    path: `/api/requirements/${encodeURIComponent(requirementId)}/artifacts/${encodeURIComponent(artifactId)}/content`,
    decode: (raw) => GetArtifactContentResp.fromJSON(raw),
  });
}
