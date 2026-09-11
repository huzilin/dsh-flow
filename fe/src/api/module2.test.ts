import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient, ApiError } from "./client";
import { myTask } from "./flows";
import { getArtifactContent } from "./artifacts";
import {
  assembleCommentAnchor,
  issueAnchorOrder,
  postClarified,
  postComment,
  postConfirmReview,
  postDeclareFixed,
  postQualify,
  postReopen,
  postRevise,
  postSettle,
} from "./comments";
import { postDecision, postReview } from "./reviews";
import { jsonResponse } from "../test-utils";

/**
 * 模块 II API 封装分支覆盖（工单 T15 验收 7）：
 * artifact content（document/link）· wb_review / wb_decision · 评论端点（NAM-09 拆分路由）·
 * 锚点装配（document/line/range/block）· 409 INVALID_TRANSITION。
 */

describe("artifact content（T6 端点）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let client: ApiClient;
  beforeEach(() => {
    fetchMock = vi.fn();
    client = new ApiClient({ getToken: () => "t", fetchImpl: fetchMock as unknown as typeof fetch });
  });

  it("document：content/href/nodeId/mime 解码透传", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        artifactId: "art-1",
        nodeId: "n-03",
        productType: "rd_design",
        label: "《RD 技术设计》",
        kind: "document",
        href: "docs/v1/rd-design.md",
        content: "# 设计\n正文第二行",
        mimeType: "text/markdown",
      }),
    );
    const resp = await getArtifactContent(client, "req-001", "art-1");
    expect(fetchMock.mock.calls[0][0]).toBe("/api/requirements/req-001/artifacts/art-1/content");
    expect(resp.kind).toBe("document");
    expect(resp.nodeId).toBe("n-03");
    expect(resp.content).toContain("正文第二行");
  });

  it("link：content 恒空、href 即外链", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        artifactId: "art-2",
        nodeId: "n-05",
        productType: "code_mr",
        label: "《RD 代码 MR》",
        kind: "link",
        href: "https://git.example.com/mr/1",
        content: "",
        mimeType: "",
      }),
    );
    const resp = await getArtifactContent(client, "req-001", "art-2");
    expect(resp.kind).toBe("link");
    expect(resp.content).toBe("");
    expect(resp.href).toBe("https://git.example.com/mr/1");
  });

  it("文档缺失：404 NOT_FOUND → ApiError", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(404, { ok: false, error: "产物文档不存在", code: "NOT_FOUND" }),
    );
    const err = await getArtifactContent(client, "req-001", "art-404").catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(404);
    expect((err as ApiError).code).toBe("NOT_FOUND");
  });
});

describe("评论五端点（T7）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let client: ApiClient;
  beforeEach(() => {
    fetchMock = vi.fn();
    client = new ApiClient({ getToken: () => "t", fetchImpl: fetchMock as unknown as typeof fetch });
  });

  it("wb_comment：锚点 line 装配进请求体（artifact_id/line 必填口径）", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { ok: true, issue: { issueId: "cm-1" } }));
    await postComment(client, "req-001", {
      nodeId: "n-03",
      anchor: assembleCommentAnchor({ kind: "line", artifactId: "art-1", line: 42 }),
      author: "user",
      roleId: "user",
      text: "这里的口径?",
      issueType: "issue",
      assignee: "",
      escalate: false,
    });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/requirements/req-001/comments");
    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    // ts-proto toJSON 省略零值字段（protojson 兼容），断言非零值装配
    expect(body.anchor).toEqual({ kind: "line", artifactId: "art-1", line: "42" });
    expect(body.nodeId).toBe("n-03");
  });

  it("wb_clarified：409 INVALID_TRANSITION（resolved 后再申报）", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(409, { ok: false, error: "issue 已 resolved，禁止回复", code: "INVALID_TRANSITION" }),
    );
    const err = await postClarified(client, "cm-1", { by: "user", roleId: "user", text: "x", voided: false }).catch(
      (e) => e,
    );
    expect(fetchMock.mock.calls[0][0]).toBe("/api/comments/cm-1/clarified");
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).isInvalidTransition).toBe(true);
    expect((err as ApiError).status).toBe(409);
  });

  it("wb_declare_fixed：POST declare-fixed（修复完成申报）", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { ok: true, issue: { state: "fixed" } }));
    await postDeclareFixed(client, "cm-4", { by: "user", roleId: "user", text: "已修复", voided: false });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/comments/cm-4/declare-fixed");
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      commentId: "cm-4",
      by: "user",
      roleId: "user",
      text: "已修复",
    });
  });

  it("wb_settle / wb_qualify：路由承载语义（path 由端点固化：settle=agree / qualify=fix）", async () => {
    fetchMock.mockImplementation(() => jsonResponse(200, { ok: true, issue: {} }));
    await postSettle(client, "cm-2", {
      operator: "user",
      note: "口径一致",
      escalate: false,
    });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/comments/cm-2/settle");
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      commentId: "cm-2",
      operator: "user",
      note: "口径一致",
      path: "agree",
    });
    await postQualify(client, "cm-2", {
      operator: "user",
      note: "",
      escalate: false,
    });
    expect(fetchMock.mock.calls[1][0]).toBe("/api/comments/cm-2/qualify");
    // toJSON 空串字段不落（proto 惯例），qualify body 无 note 键
    expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({
      commentId: "cm-2",
      operator: "user",
      path: "fix",
    });
  });

  it("wb_confirm / wb_reopen / wb_revise：POST confirm / reopen / revise", async () => {
    fetchMock.mockImplementation(() => jsonResponse(200, { ok: true, issue: {} }));
    await postConfirmReview(client, "cm-3", { operator: "user" });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/comments/cm-3/confirm");
    await postReopen(client, "cm-3", { operator: "user", text: "仍未修复" });
    expect(fetchMock.mock.calls[1][0]).toBe("/api/comments/cm-3/reopen");
    expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({
      commentId: "cm-3",
      operator: "user",
      text: "仍未修复",
    });
    await postRevise(client, "cm-3", { operator: "user", text: "" });
    expect(fetchMock.mock.calls[2][0]).toBe("/api/comments/cm-3/revise");
  });
});

describe("assembleCommentAnchor（四级锚点装配）", () => {
  it("document：不带行号族字段且 artifactId 空（toJSON 后仅剩 kind）", () => {
    const a = assembleCommentAnchor({ kind: "document" });
    expect(a.kind).toBe("document");
    expect(a.artifactId).toBe("");
    expect(a.line).toBe("0");
    expect(a.startLine).toBe("0");
  });
  it("range：start≤end 装配", () => {
    const a = assembleCommentAnchor({ kind: "range", artifactId: "art-1", startLine: 7, endLine: 9 });
    expect(a.startLine).toBe("7");
    expect(a.endLine).toBe("9");
    expect(a.line).toBe("0");
  });
  it("block：lang 非空必填口径", () => {
    const a = assembleCommentAnchor({ kind: "block", artifactId: "art-1", blockStart: 2, blockEnd: 8, lang: "sql" });
    expect(a.blockStart).toBe("2");
    expect(a.blockEnd).toBe("8");
    expect(a.lang).toBe("sql");
  });
  it("issueAnchorOrder：document 恒排正文末尾", () => {
    const doc = { anchorKind: "document", anchorLine: "0", anchorEndLine: "0", anchorBlockEnd: "0" };
    const line = { anchorKind: "line", anchorLine: "42", anchorEndLine: "0", anchorBlockEnd: "0" };
    expect(issueAnchorOrder(doc as never)).toBe(Number.MAX_SAFE_INTEGER);
    expect(issueAnchorOrder(line as never)).toBe(42);
  });
});

describe("wb_review / wb_decision（T8 端点封装）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let client: ApiClient;
  beforeEach(() => {
    fetchMock = vi.fn();
    client = new ApiClient({ getToken: () => "t", fetchImpl: fetchMock as unknown as typeof fetch });
  });

  it("wb_review：encode + collect_outcome/gate_state 解码", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        nodeId: "n-03",
        role: "arch",
        state: "agree",
        round: "1",
        reviewResults: "[]",
        collectOutcome: "approved",
        gateState: "agreed",
        nextNodeId: "n-04",
      }),
    );
    const resp = await postReview(client, "req-001", "n-03", {
      role: "arch",
      capabilityId: "arch.prd-review",
      state: "agree",
    });
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/requirements/req-001/nodes/n-03/review",
    );
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      requirementId: "req-001",
      nodeId: "n-03",
      role: "arch",
      capabilityId: "arch.prd-review",
      state: "agree",
    });
    expect(resp.collectOutcome).toBe("approved");
    expect(resp.gateState).toBe("agreed");
    expect(resp.nextNodeId).toBe("n-04");
  });

  it("wb_decision：verdict/reason/branch encode；noop 解码", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        nodeId: "n-00",
        verdict: "approved",
        noop: false,
        lastDecision: '{"verdict":"approved","at":1}',
        nextNodeId: "n-01",
      }),
    );
    const resp = await postDecision(client, "req-001", "n-00", {
      verdict: "approved",
      reason: "",
      branch: "feat/req-001",
    });
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/requirements/req-001/nodes/n-00/decision",
    );
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      requirementId: "req-001",
      nodeId: "n-00",
      verdict: "approved",
      branch: "feat/req-001",
    });
    expect(resp.noop).toBe(false);
    expect(resp.nextNodeId).toBe("n-01");
  });

  it("wb_decision 非可拍态：409 GATE_BLOCKED + blocked:true", async () => {
    fetchMock.mockResolvedValueOnce(
      jsonResponse(409, {
        ok: false,
        error: "评审轮未全体同意，拍板被拦截",
        code: "GATE_BLOCKED",
        blocked: true,
      }),
    );
    const err = await postDecision(client, "req-001", "n-00", { verdict: "approved", reason: "", branch: "" }).catch(
      (e) => e,
    );
    expect((err as ApiError).isBlocked).toBe(true);
  });
});

describe("wb_my_task（role=user 消费）", () => {
  it("node 摘要 + activeActions + relatedTasks 解码", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      jsonResponse(200, {
        ok: true,
        requirementId: "req-001",
        role: "user",
        node: { nodeId: "n-03", defNodeId: "bus-03", nodeType: "review", state: "review", reviewers: "[]" },
        currentPhase: "review",
        activeActions: [
          {
            actionId: "clarify",
            name: "澄清",
            kind: "capability",
            capabilityId: "product-design",
            on: "respond",
          },
        ],
        reviewSummary: { agree: "2", unagree: "0", reviewing: "1", details: [] },
        relatedNodes: [],
        relatedArtifacts: [{ artifactId: "art-1", productType: "brd", label: "BRD", kind: "document" }],
        relatedTasks: [{ taskId: "t-1", actionId: "pm.to-brd", status: "in-progress" }],
        relatedIssues: [],
      }),
    );
    const client = new ApiClient({ getToken: () => "t", fetchImpl: fetchMock as unknown as typeof fetch });
    const resp = await myTask(client, "req-001", "user", "n-03");
    expect(fetchMock.mock.calls[0][0]).toBe("/api/requirements/req-001/my-task");
    expect(resp.currentPhase).toBe("review");
    expect(resp.reviewSummary?.agree).toBe("2");
    expect(resp.relatedTasks[0]?.status).toBe("in-progress");
    expect(resp.relatedArtifacts[0]?.label).toBe("BRD");
  });
});
