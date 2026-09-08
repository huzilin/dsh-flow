import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient, ApiError } from "./client";
import { listInbox, postAdjudicate, postEscalate } from "./inbox";
import { jsonResponse } from "../test-utils";

/**
 * 模块 III/IV API 封装分支覆盖（工单 T16 验收）：
 * wb_list_inbox（query 绑定/双通道解码/越集 400）· wb_escalate / wb_adjudicate
 * （T12 路径与请求体装配、409 GATE_BLOCKED 分支）。
 */

describe("wb_list_inbox（T11）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let client: ApiClient;
  beforeEach(() => {
    fetchMock = vi.fn();
    client = new ApiClient({ getToken: () => "t", fetchImpl: fetchMock as unknown as typeof fetch });
  });

  it("query 绑定：role/read_filter/kind/requirement_id/page/page_size；零值跳过", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        notifications: [
          {
            issueId: "cm-1",
            kind: "gate_pending",
            requirementId: "req-001",
            nodeId: "n-05",
            targetType: "node",
            targetId: "",
            text: "节点【产品设计评审】已 agreed，待你拍板",
            sender: "plugin",
            senderRole: "plugin",
            recipientRole: "user",
            state: "open",
            createdAt: "1730000000000",
          },
        ],
        total: "1",
        todos: [
          {
            taskId: "t-1",
            requirementId: "req-001",
            nodeId: "n-05",
            actionId: "bus-03.clarify",
            taskType: "respond",
            status: "in-progress",
            goal: "澄清工单",
            createdAt: "1730000000000",
          },
        ],
      }),
    );
    const resp = await listInbox(client, {
      role: "user",
      readFilter: "unread",
      kind: "gate_pending",
      requirementId: "req-001",
      page: 2,
      pageSize: 50,
    });
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/inbox?role=user&read_filter=unread&kind=gate_pending&requirement_id=req-001&page=2&page_size=50",
    );
    expect(resp.ok).toBe(true);
    expect(resp.total).toBe("1");
    expect(resp.notifications[0].kind).toBe("gate_pending");
    expect(resp.notifications[0].targetType).toBe("node");
    expect(resp.todos[0].status).toBe("in-progress");
  });

  it("空参数：路径不带 query", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { ok: true, notifications: [], total: "0", todos: [] }));
    await listInbox(client);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/inbox");
  });

  it("read_filter 越集：400 INVALID_ARGUMENT → ApiError", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(400, { ok: false, error: "read_filter 须为空|unread|read", code: "INVALID_ARGUMENT" }),
    );
    const err = await listInbox(client, { readFilter: "bogus" }).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(400);
    expect((err as ApiError).code).toBe("INVALID_ARGUMENT");
  });
});

describe("冲突上升裁决链（T12）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let client: ApiClient;
  beforeEach(() => {
    fetchMock = vi.fn();
    client = new ApiClient({ getToken: () => "t", fetchImpl: fetchMock as unknown as typeof fetch });
  });

  it("wb_escalate：路径 + operator/reason 装配；noop 回显既有 session", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        commentId: "cm-9",
        escalated: false,
        noop: true,
        sessionId: "ses-7",
      }),
    );
    const resp = await postEscalate(client, "cm-9", { operator: "user", reason: "反复拉锯不收敛" });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/comments/cm-9/escalate");
    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(body.operator).toBe("user");
    expect(body.reason).toBe("反复拉锯不收敛");
    expect(resp.noop).toBe(true);
    expect(resp.sessionId).toBe("ses-7");
  });

  it("wb_adjudicate：verdict/conclusion/decided_by 装配；reentered/round 解码", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        commentId: "cm-9",
        verdict: "override",
        sessionId: "ses-7",
        reentered: true,
        round: "3",
        issue: { issueId: "cm-9", state: "overridden", escalate: "0" },
      }),
    );
    const resp = await postAdjudicate(client, "cm-9", {
      verdict: "override",
      conclusion: "按仲裁结论推翻议题",
      decidedBy: "user",
    });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/comments/cm-9/adjudicate");
    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(body.verdict).toBe("override");
    expect(body.conclusion).toBe("按仲裁结论推翻议题");
    expect(body.decidedBy).toBe("user");
    expect(resp.reentered).toBe(true);
    expect(resp.round).toBe("3");
    expect(resp.issue?.state).toBe("overridden");
  });

  it("wb_adjudicate：escalate=0 拒绝回填 → 409 GATE_BLOCKED", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(409, { ok: false, error: "评论未上升裁决（escalate=0）", code: "GATE_BLOCKED", blocked: true }),
    );
    const err = await postAdjudicate(client, "cm-9", {
      verdict: "uphold",
      conclusion: "x",
      decidedBy: "user",
    }).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(409);
    expect((err as ApiError).code).toBe("GATE_BLOCKED");
    expect((err as ApiError).isBlocked).toBe(true);
  });
});
