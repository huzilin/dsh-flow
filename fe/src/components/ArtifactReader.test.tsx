import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ArtifactItem, IssueItem } from "../api/proto/api";
import { ArtifactReader } from "./ArtifactReader";
import { CommentCard } from "./CommentCard";
import { jsonResponse, renderWithClient } from "../test-utils";

/**
 * 阅读器锚点装配测试（工单 T15 验收 2/7）：
 * - document 默认态 / 划词单击序列（注记-L3：首点=起点、重复点起点=取消、点他行=框选；
 *   单行=range(n,n)；line/block 锚已废弃，后端 400）
 *   四级锚点装配进 POST /api/requirements/:reqId/comments 请求体
 * - 懒加载分段「加载后续 N 行」按钮 + 锚定评论就位渲染 + link 外链卡
 * - CommentCard 操作矩阵：resolve=assignee、confirm/reopen=提评者、回复恒可见
 */

const DOC_CONTENT = [
  "# RD 技术设计",
  "",
  "```sql",
  "select 1 from t;",
  "```",
  "设计正文第六行",
  "设计正文第七行",
  "设计正文第八行",
].join("\n");

function artifact(kind = "document"): ArtifactItem {
  return {
    artifactId: "art-1",
    nodeId: "n-04",
    productType: "rd_design",
    label: "《RD 技术设计》",
    href: "docs/v1/rd-design.md",
    kind,
    note: "",
    addedBy: "rd",
    createdAt: "1730000000000",
  };
}

function issue(partial: Partial<IssueItem>): IssueItem {
  return {
    issueId: "cm-1",
    nodeId: "n-04",
    anchorKind: "line",
    author: "arch",
    roleId: "arch",
    text: "这里的口径?",
    state: "open",
    escalate: "0",
    loopCount: "0",
    roundCount: "1",
    createdAt: "1730000000000",
    issueType: "issue",
    reviewerRole: "arch",
    assignee: "rd",
    anchorArtifactId: "art-1",
    anchorLine: "6",
    anchorStartLine: "0",
    anchorEndLine: "0",
    anchorBlockStart: "0",
    anchorBlockEnd: "0",
    anchorLang: "",
    replies: "[]",
    lastReplyAt: "0",
    resolvePath: "",
    resolveBy: "",
    resolveAt: "0",
    resolveNote: "",
    resolveReviewerConfirmed: "0",
    resolvedBy: "",
    resolvedAt: "0",
    updatedAt: "1730000000000",
    ...partial,
  };
}

describe("ArtifactReader（锚点装配）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  function mockContent(content: string) {
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const path = String(input);
      if (path.endsWith("/content")) {
        return Promise.resolve(
          jsonResponse(200, {
            ok: true,
            artifactId: "art-1",
            nodeId: "n-04",
            productType: "rd_design",
            label: "《RD 技术设计》",
            kind: "document",
            href: "docs/v1/rd-design.md",
            content,
            mimeType: "text/markdown",
          }),
        );
      }
      return Promise.resolve(jsonResponse(200, { ok: true, issue: { issueId: "cm-new" } }));
    });
  }

  async function postCommentBody(): Promise<Record<string, unknown>> {
    const call = fetchMock.mock.calls.find((c) => String(c[0]).endsWith("/comments"));
    return JSON.parse((call?.[1] as { body: string }).body);
  }

  it("默认态：document 级锚点（无行号族字段）", async () => {
    mockContent(DOC_CONTENT);
    const user = userEvent.setup();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    expect(await screen.findByText("设计正文第七行")).toBeInTheDocument();
    expect(screen.getByText(/全文评论（document）/)).toBeInTheDocument();
    await user.type(screen.getByLabelText("评论内容"), "整体口径确认");
    await user.click(screen.getByRole("button", { name: "发表评论" }));
    await waitFor(() => {
      const body = fetchMock.mock.calls.find((c) => String(c[0]).endsWith("/comments"));
      expect(body).toBeDefined();
    });
    const body = await postCommentBody();
    // BUG-013：document 锚携带本文物 artifactId（评审上下文锚产物强制的前置口径）
    expect(body.anchor).toEqual({ kind: "document", artifactId: "art-1" });
    expect(body.nodeId).toBe("n-04");
    expect(body.text).toBe("整体口径确认");
  });

  it("单击起点：单行选中 → range(n,n) 装配（注记-L3 行锚废弃）", async () => {
    mockContent(DOC_CONTENT);
    const user = userEvent.setup();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("设计正文第七行");
    await user.click(screen.getByText("设计正文第七行").closest(".reader-line")!);
    expect(screen.getByText("评论第 7 行")).toBeInTheDocument();
    await user.type(screen.getByLabelText("评论内容"), "第七行问题");
    await user.click(screen.getByRole("button", { name: "发表评论" }));
    await waitFor(async () => expect((await postCommentBody()).anchor).toBeDefined());
    expect(await postCommentBody()).toMatchObject({
      anchor: { kind: "range", artifactId: "art-1", startLine: "7", endLine: "7" },
    });
  });

  it("单击序列框选：起点 → 点他行=区间 → 再点=重新开始（注记-L3 老交互手感）", async () => {
    mockContent(DOC_CONTENT);
    const user = userEvent.setup();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("设计正文第七行");
    // 起点=第八行；点第六行=框选 6-8；再点第七行=重新开始为单点（老交互语义）
    await user.click(screen.getByText("设计正文第八行").closest(".reader-line")!);
    await user.click(screen.getByText("设计正文第六行").closest(".reader-line")!);
    expect(screen.getByText(/评论第 6-8 行区间/)).toBeInTheDocument();
    await user.click(screen.getByText("设计正文第七行").closest(".reader-line")!);
    expect(screen.getByText("评论第 7 行")).toBeInTheDocument();
    await user.type(screen.getByLabelText("评论内容"), "区间问题");
    await user.click(screen.getByRole("button", { name: "发表评论" }));
    await waitFor(async () => expect((await postCommentBody()).anchor).toBeDefined());
    expect(await postCommentBody()).toMatchObject({
      anchor: { kind: "range", artifactId: "art-1", startLine: "7", endLine: "7" },
    });
  });

  it("选块按钮已移除（注记-L3 block 锚废弃）", async () => {
    mockContent(DOC_CONTENT);
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("设计正文第七行");
    expect(screen.queryByTitle(/选代码块/)).toBeNull();
  });

  it("锚定评论就位渲染（line 6 行下方）+ document 级评论在正文末尾", async () => {
    mockContent(DOC_CONTENT);
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[
          issue({ anchorKind: "line", anchorLine: "6" }),
          issue({ issueId: "cm-doc", anchorKind: "document", anchorLine: "0", text: "整体建议" }),
        ]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("设计正文第七行");
    expect(screen.getAllByText("这里的口径?").length).toBe(1);
    expect(screen.getByText("整体建议")).toBeInTheDocument();
  });

  it("懒加载分段：>600 行文档先渲染 600 行，按钮续载", async () => {
    const long = Array.from({ length: 700 }, (_, i) => `第${i + 1}行`).join("\n");
    mockContent(long);
    const user = userEvent.setup();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    expect(await screen.findByText("第1行")).toBeInTheDocument();
    expect(screen.queryByText("第700行")).not.toBeInTheDocument();
    expect(screen.getByText(/已渲染 600\/700 行/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /加载后续 100 行/ }));
    expect(await screen.findByText("第700行")).toBeInTheDocument();
  });

  it("link 产物：外链卡呈现，不发正文请求", async () => {
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact("link")}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    expect(screen.getByRole("link", { name: "docs/v1/rd-design.md" })).toHaveAttribute(
      "href",
      "docs/v1/rd-design.md",
    );
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

describe("CommentCard（操作矩阵可见性，与 T7 操作者约束同口径）", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("提评者=user：fixed 态可见 复核确认/reopen，不可见 resolve", () => {
    renderWithClient(
      <CommentCard
        requirementId="req-001"
        issue={issue({ state: "fixed", reviewerRole: "", author: "user", roleId: "user", assignee: "rd" })}
      />,
    );
    expect(screen.getByRole("button", { name: "复核确认关闭" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "reopen" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /定性打回（qualify）/ })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "回复" })).toBeInTheDocument();
  });

  it("assignee=user：可见 resolve 双路径，不可见 confirm/reopen", () => {
    renderWithClient(
      <CommentCard
        requirementId="req-001"
        issue={issue({ state: "open", reviewerRole: "arch", author: "arch", roleId: "arch", assignee: "user" })}
      />,
    );
    expect(screen.getByRole("button", { name: /定性打回（qualify）/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /协商一致关闭（settle）/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "复核确认关闭" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "reopen" })).not.toBeInTheDocument();
  });

  it("均非操作者：仅回复可见；escalate 标记呈现", () => {
    renderWithClient(
      <CommentCard
        requirementId="req-001"
        issue={issue({ escalate: "1", reviewerRole: "arch", assignee: "rd" })}
      />,
    );
    expect(screen.getByRole("button", { name: "回复" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "复核确认关闭" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /定性打回（qualify）/ })).not.toBeInTheDocument();
    expect(screen.getByText("已上升裁决")).toBeInTheDocument();
    expect(screen.getByText(/锚点：art-1 line 6/)).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// BUG-013 评审上下文锚产物（2026-09-10 批注轮）
// ---------------------------------------------------------------------------

describe("ArtifactReader（BUG-013 评审上下文锚产物）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  function mockContent() {
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      if (String(input).endsWith("/content")) {
        return Promise.resolve(
          jsonResponse(200, {
            ok: true,
            artifactId: "art-1",
            nodeId: "n-04",
            productType: "rd_design",
            label: "《RD 技术设计》",
            kind: "document",
            href: "docs/v1/rd-design.md",
            content: "设计正文",
            mimeType: "text/plain",
          }),
        );
      }
      return Promise.resolve(jsonResponse(200, { ok: true, issue: { issueId: "cm-new" } }));
    });
  }

  it("reviewContext=true：底部常驻引导提示「评论将锚定《…》」", async () => {
    mockContent();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
        reviewContext
        artifactLabels={{ "art-1": "《RD 技术设计》" }}
      />,
    );
    expect(await screen.findByText(/评审上下文：评论将锚定 《RD 技术设计》/)).toBeInTheDocument();
  });

  it("reviewContext=true 发表 document 级评论：请求体 anchor 携带本文物 artifactId", async () => {
    mockContent();
    const user = userEvent.setup();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
        reviewContext
      />,
    );
    await screen.findByText("设计正文");
    await user.type(screen.getByLabelText("评论内容"), "评审意见挂产物");
    await user.click(screen.getByRole("button", { name: "发表评论" }));
    await waitFor(() => {
      const call = fetchMock.mock.calls.find((c) => String(c[0]).endsWith("/comments"));
      expect(call).toBeDefined();
    });
    const call = fetchMock.mock.calls.find((c) => String(c[0]).endsWith("/comments"));
    const body = JSON.parse(String(call?.[1]?.body)) as { anchor: { kind: string; artifactId: string } };
    expect(body.anchor).toEqual({ kind: "document", artifactId: "art-1" });
  });

  it("CommentCard 传 artifactLabels：锚点摘要呈现产物名而非裸 id", () => {
    renderWithClient(
      <CommentCard
        requirementId="req-001"
        issue={issue({ anchorKind: "document", anchorArtifactId: "art-1" })}
        artifactLabels={{ "art-1": "《RD 技术设计》" }}
      />,
    );
    expect(screen.getByText(/锚点：/)).toHaveTextContent("锚点：《RD 技术设计》（全文）");
  });
});

// ---------------------------------------------------------------------------
// 一版交互蓝本回归（inventory A4-11/B-43、A4-2/B-34、A4-16/B-47）
// ---------------------------------------------------------------------------

describe("CommentCard 已解决折叠（一版 A4-11 回归）", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("resolved 默认折叠一行摘要（角色+状态徽标+文本），展开回全文 thread，可再收起", () => {
    renderWithClient(
      <CommentCard
        requirementId="req-001"
        issue={issue({
          state: "resolved",
          text: "口径已对齐，按此执行",
          resolvePath: "agree",
          resolveBy: "rd",
          resolveNote: "双方确认",
          replies: JSON.stringify([
            { replyId: "r-1", by: "rd", roleId: "rd", text: "收到", at: 1730000001000 },
          ]),
        })}
      />,
    );
    // 折叠态：摘要行（徽标+角色+文本+箭头），全文操作/回复/resolve 信息不渲染
    expect(screen.getByText("已解决")).toBeInTheDocument();
    expect(screen.getByText("arch（arch）")).toBeInTheDocument();
    expect(screen.getByText("口径已对齐，按此执行")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "回复" })).not.toBeInTheDocument();
    expect(screen.queryByText(/协商一致关闭/)).not.toBeInTheDocument();
    expect(screen.queryByText("收到")).not.toBeInTheDocument();
    // 展开：全文 thread + resolve 信息 + 操作矩阵
    fireEvent.click(screen.getByRole("button", { name: /展开已解决评论/ }));
    expect(screen.getByRole("button", { name: "回复" })).toBeInTheDocument();
    expect(screen.getByText("收到")).toBeInTheDocument();
    expect(screen.getByText(/协商一致关闭 · rd · 双方确认/)).toBeInTheDocument();
    // 再收起
    fireEvent.click(screen.getByRole("button", { name: "收起评论" }));
    expect(screen.queryByRole("button", { name: "回复" })).not.toBeInTheDocument();
  });

  it("open/to-fix/fixed/pending-confirm 保持展开（to-fix 直接可见操作矩阵）", () => {
    renderWithClient(
      <CommentCard
        requirementId="req-001"
        issue={issue({ state: "to-fix", assignee: "user" })}
      />,
    );
    expect(screen.getByRole("button", { name: "回复" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /协商一致关闭（settle）/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /展开已解决评论/ })).not.toBeInTheDocument();
  });
});

describe("ArtifactReader 轻量 md 渲染（一版 A4-2 回归，行映射 1:1 不漂移）", () => {
  const MD_CONTENT = [
    "# 标题一",
    "",
    "- **加粗项** 与 `wb_code` 项",
    "> 引用一行",
    "| a | b |",
    "|---|---|",
    "| 1 | 2 |",
    "普通段落 **粗体**",
  ].join("\n");

  let fetchMock: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  function mockContent(content: string) {
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      if (String(input).endsWith("/content")) {
        return Promise.resolve(
          jsonResponse(200, {
            ok: true,
            artifactId: "art-1",
            nodeId: "n-04",
            productType: "rd_design",
            label: "《RD 技术设计》",
            kind: "document",
            href: "docs/v1/rd-design.md",
            content,
            mimeType: "text/markdown",
          }),
        );
      }
      return Promise.resolve(jsonResponse(200, { ok: true, issue: { issueId: "cm-new" } }));
    });
  }

  it("标题/列表/粗体/行内代码/表格单元格/blockquote 就地渲染", async () => {
    mockContent(MD_CONTENT);
    const { container } = renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("标题一");
    expect(container.querySelector(".reader-line .md-h1")?.textContent).toBe("标题一");
    expect(screen.getByText("加粗项").tagName).toBe("STRONG");
    expect(screen.getByText("wb_code").tagName).toBe("CODE");
    expect(screen.getByText("引用一行").closest(".line-quote")).not.toBeNull();
    // 管道表格按行切单元格（表头行加粗底、数据行单元格）
    const headRow = container.querySelector('.reader-line[data-line="5"] .line-table-row')!;
    expect(headRow.querySelectorAll(".line-cell")).toHaveLength(2);
    const dataRow = container.querySelector('.reader-line[data-line="7"] .line-table-row')!;
    expect(dataRow.querySelectorAll(".line-cell")).toHaveLength(2);
    expect(screen.getByText("普通段落")).toBeInTheDocument();
  });

  it("锚点不漂移：渲染行数=源行数（data-line 1..N 恒在），划词锚按渲染后行定位", async () => {
    mockContent(MD_CONTENT);
    const { container } = renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("标题一");
    const lineNos = Array.from(container.querySelectorAll(".reader-line")).map((el) =>
      el.getAttribute("data-line"),
    );
    expect(lineNos).toEqual(["1", "2", "3", "4", "5", "6", "7", "8"]);
    // 渲染行点击仍按源行号装配 range 锚（md 渲染不改坐标系）
    const user = userEvent.setup();
    await user.click(container.querySelector('.reader-line[data-line="4"]')!);
    expect(screen.getByText(/评论第 4 行区间|评论第 4 行/)).toBeInTheDocument();
  });
});

describe("ArtifactReader 评论筛选药丸（一版 A4-16 回归）", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  function mockContent(content: string) {
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      if (String(input).endsWith("/content")) {
        return Promise.resolve(
          jsonResponse(200, {
            ok: true,
            artifactId: "art-1",
            nodeId: "n-04",
            productType: "rd_design",
            label: "《RD 技术设计》",
            kind: "document",
            href: "docs/v1/rd-design.md",
            content,
            mimeType: "text/markdown",
          }),
        );
      }
      return Promise.resolve(jsonResponse(200, { ok: true, issue: { issueId: "cm-new" } }));
    });
  }

  it("作者（按 role）× 状态五态药丸：点选过滤、再点取消", async () => {
    mockContent(DOC_CONTENT);
    const user = userEvent.setup();
    renderWithClient(
      <ArtifactReader
        requirementId="req-001"
        nodeId="n-04"
        artifact={artifact()}
        issues={[
          issue({ anchorKind: "line", anchorLine: "6" }),
          issue({
            issueId: "cm-doc",
            anchorKind: "document",
            anchorLine: "0",
            text: "整体建议",
            roleId: "qa",
            reviewerRole: "qa",
            state: "resolved",
          }),
        ]}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("设计正文第七行");
    // 初始全量：行内评论 + document 评论（resolved 折叠摘要也可见）
    expect(screen.getByText("这里的口径?")).toBeInTheDocument();
    expect(screen.getByText("整体建议")).toBeInTheDocument();
    // 状态过滤：待处理（open）→ resolved 的 qa 评论隐藏
    await user.click(screen.getByRole("button", { name: "待处理" }));
    expect(screen.getByText("这里的口径?")).toBeInTheDocument();
    expect(screen.queryByText("整体建议")).not.toBeInTheDocument();
    // 再点取消
    await user.click(screen.getByRole("button", { name: "待处理" }));
    expect(screen.getByText("整体建议")).toBeInTheDocument();
    // 作者过滤：qa → arch 行内评论隐藏
    await user.click(screen.getByRole("button", { name: "qa" }));
    expect(screen.queryByText("这里的口径?")).not.toBeInTheDocument();
    expect(screen.getByText("整体建议")).toBeInTheDocument();
  });
});
