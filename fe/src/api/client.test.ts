import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient, ApiError } from "./client";
import { buildListRequirementsQuery, listRequirements } from "./requirements";
import { advanceVersion, deprecateVersion, updateVersion } from "./versions";

/**
 * fetch client 分支覆盖：
 * - Bearer token 注入（env token / 空 token 不带头）
 * - protojson 编解码：int64 字段一律按 string 承接（后端 protojson EmitUnpopulated 出参口径）
 * - 统一失败出参：401 UNAUTHORIZED / 409 OPTIMISTIC_LOCK / 409 GATE_BLOCKED blocked:true
 * - query 组装：backlog 与 version_id 互斥、空值跳过
 */
function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("ApiClient", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("注入 Authorization: Bearer <token>", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { ok: true, items: [] }));
    const client = new ApiClient({
      getToken: () => "tok-123",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    await client.request({
      method: "GET",
      path: "/api/versions",
      decode: (raw) => raw as { ok: boolean },
    });
    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer tok-123");
  });

  it("token 为空不携带 Authorization 头（后端 fail closed 401 兜底）", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { ok: true, items: [] }));
    const client = new ApiClient({
      getToken: () => "",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    await client.request({
      method: "GET",
      path: "/api/versions",
      decode: (raw) => raw as { ok: boolean },
    });
    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>;
    expect(headers.Authorization).toBeUndefined();
  });

  it("protojson 解码：int64 字段按 string 承接（total/createdAt/updatedAt）", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        total: "1788813264607",
        page: "1",
        pageSize: "20",
        items: [
          {
            requirementId: "req-001",
            title: "示例需求",
            category: "需求任务",
            flowDefsId: "business",
            versionId: "",
            priority: "P2",
            status: "待启动",
            owner: "",
            createdBy: "user",
            createdAt: "1788813264607",
            updatedAt: "1788813264999",
          },
        ],
      }),
    );
    const client = new ApiClient({
      getToken: () => "t",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    const resp = await listRequirements(client, { page: 1, pageSize: 20 });
    // 契约：int64 → string；versionId 空串=待分配池
    expect(resp.total).toBe("1788813264607");
    expect(typeof resp.total).toBe("string");
    expect(resp.items[0]?.createdAt).toBe("1788813264607");
    expect(resp.items[0]?.updatedAt).toBe("1788813264999");
    expect(resp.items[0]?.versionId).toBe("");
  });

  it("protojson 编码：updated_at 基线以字符串回传后端（protojson 兼容 string|number）", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        version: {
          versionId: "v-1",
          name: "v2",
          state: "待启动",
          isBacklog: false,
          createdAt: "1788813264607",
          updatedAt: "1788813264999",
        },
      }),
    );
    const client = new ApiClient({
      getToken: () => "t",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    await updateVersion(client, "v-1", {
      versionId: "v-1",
      name: "v2",
      updatedAt: "1788813264607",
    });
    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(body.updatedAt).toBe("1788813264607");
    expect(body.name).toBe("v2");
  });

  it("401 → ApiError(code=UNAUTHORIZED)", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(401, { ok: false, error: "无效或缺失 Bearer token", code: "UNAUTHORIZED" }),
    );
    const client = new ApiClient({
      getToken: () => "bad",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    const err = await client
      .request({ method: "GET", path: "/api/versions", decode: (r) => r })
      .catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(401);
    expect((err as ApiError).code).toBe("UNAUTHORIZED");
    expect((err as ApiError).isUnauthorized).toBe(true);
  });

  it("409 OPTIMISTIC_LOCK → isOptimisticLock", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(409, {
        ok: false,
        error: "wbflow: 乐观锁冲突（updated_at 不匹配）",
        code: "OPTIMISTIC_LOCK",
      }),
    );
    const client = new ApiClient({
      getToken: () => "t",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    const err = await updateVersion(client, "v-1", {
      versionId: "v-1",
      name: "x",
      updatedAt: "1",
    }).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).isOptimisticLock).toBe(true);
    expect((err as ApiError).status).toBe(409);
  });

  it("409 GATE_BLOCKED（deprecate 有引用）→ blocked:true + error 原样", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(409, {
        ok: false,
        error: '版本 "v-1" 存在 2 条未完成需求引用，废弃被拦截（先取消/完成或移出版本）',
        code: "GATE_BLOCKED",
        blocked: true,
      }),
    );
    const client = new ApiClient({
      getToken: () => "t",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    const err = await deprecateVersion(client, "v-1", {
      versionId: "v-1",
      updatedAt: "1",
    }).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).isBlocked).toBe(true);
    expect((err as ApiError).code).toBe("GATE_BLOCKED");
    expect((err as ApiError).message).toContain("存在 2 条未完成需求引用");
  });

  it("advance 成功：unfinished 清单（int64 无涉）解码透传", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        version: {
          versionId: "v-1",
          name: "v2",
          state: "已发布",
          isBacklog: false,
          createdAt: "1",
          updatedAt: "2",
        },
        unfinished: [{ requirementId: "req-001", title: "未完成需求", status: "进行中" }],
      }),
    );
    const client = new ApiClient({
      getToken: () => "t",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    const resp = await advanceVersion(client, "v-1", { versionId: "v-1", updatedAt: "1" });
    expect(resp.version?.state).toBe("已发布");
    expect(resp.unfinished).toHaveLength(1);
    expect(resp.unfinished[0]?.status).toBe("进行中");
  });
});

describe("buildListRequirementsQuery", () => {
  it("backlog 与 version_id 互斥（backlog 优先）；空值键跳过", () => {
    expect(buildListRequirementsQuery({ backlog: true, versionId: "v-1", page: 2 })).toBe(
      "?backlog=true&page=2",
    );
    const q1 = buildListRequirementsQuery({ backlog: true });
    expect(q1).toBe("?backlog=true");
    const q2 = buildListRequirementsQuery({ versionId: "v-1", priority: "P0", status: "进行中" });
    expect(q2).toBe("?version_id=v-1&priority=P0&status=%E8%BF%9B%E8%A1%8C%E4%B8%AD");
    const q3 = buildListRequirementsQuery({});
    expect(q3).toBe("");
  });
});
