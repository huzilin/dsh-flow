import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { FlowNodeItem, IssueItem } from "../api/proto/api";
import { ConflictCard } from "./ConflictCard";
import { jsonResponse } from "../test-utils";

/**
 * 冲突卡测试（工单 T16 AC，ui-interaction-final §1.4）：
 * escalate 聚合卡头/正反对照（review_results 反对位 × issue replies 响应方）/
 * 裁决回填条目呈现 / wb_escalate 拉起（幂等 noop）/ wb_adjudicate uphold|override
 * 表单分支与 409 GATE_BLOCKED 兜底。
 */

function mkNode(): FlowNodeItem {
  return {
    nodeId: "n-03",
    defNodeId: "bus-02",
    nodeOrder: "3",
    nodeKey: "产品设计",
    state: "respond",
    reviewResults:
      '[{"role":"qa","capabilityId":"qa.review","state":"unagree","round":2,"at":1730000005000},{"role":"rd","capabilityId":"rd.review","state":"agree","round":2,"at":1730000006000}]',
    reopenSource: "",
    rolledBackCount: "1",
    featureBranch: "",
    enteredAt: "1730000000000",
    nodeType: "produce",
    gate: "0",
    userGate: "0",
    reviewers: '[{"role":"qa","capabilityId":"qa.review"},{"role":"rd","capabilityId":"rd.review"}]',
    lastDecision: "",
  };
}

function mkIssue(over: Partial<IssueItem> = {}): IssueItem {
  return {
    issueId: "cm-9",
    nodeId: "n-03",
    anchorKind: "document",
    author: "qa",
    roleId: "qa",
    text: "边界口径不清：范围 A 是否包含降级路径",
    state: "open",
    escalate: "1",
    loopCount: "3",
    roundCount: "4",
    createdAt: "1730000000000",
    issueType: "issue",
    reviewerRole: "qa",
    assignee: "rd",
    anchorArtifactId: "",
    anchorLine: "0",
    anchorStartLine: "0",
    anchorEndLine: "0",
    anchorBlockStart: "0",
    anchorBlockEnd: "0",
    anchorLang: "",
    replies: JSON.stringify([
      {
        replyId: "r-1",
        by: "rd",
        roleId: "rd",
        text: "已按降级路径不包含处理，见修订",
        at: 1730000001000,
      },
      {
        replyId: "r-2",
        by: "qa",
        roleId: "qa",
        text: "反对：修订未覆盖边界 2",
        voided: true,
        at: 1730000002000,
      },
      {
        replyId: "r-3",
        by: "user",
        roleId: "user",
        text: "维持议题：按含降级路径的口径执行",
        kind: "adjudication",
        verdict: "uphold",
        sessionId: "ses-7",
        at: 1730000003000,
      },
    ]),
    lastReplyAt: "1730000003000",
    resolvePath: "",
    resolveBy: "",
    resolveAt: "0",
    resolveNote: "",
    resolveReviewerConfirmed: "0",
    resolvedBy: "",
    resolvedAt: "0",
    updatedAt: "1730000000000",
    ...over,
  };
}

function renderCard(issue: IssueItem, node?: FlowNodeItem) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ConflictCard requirementId="req-001" issue={issue} node={node} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ConflictCard", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("卡头与正反对照：分歧点 + 反对评审位（review_results）× 响应方回复（replies）两栏并排", () => {
    renderCard(mkIssue(), mkNode());
    // 卡头：冲突卡徽标 + 裁决进行中 + 节点详情跳转（issue 锚点）
    expect(screen.getByText("⚔ 冲突卡")).toBeInTheDocument();
    expect(screen.getByText("裁决进行中")).toBeInTheDocument();
    const nodeLink = screen.getByText("↗ 节点详情").closest("a");
    expect(nodeLink).toHaveAttribute(
      "href",
      "/requirements/req-001?node=n-03&focus=issue&issue=cm-9",
    );
    // 分歧点（issue.text 同时出现在正方栏，改按容器断言）
    expect(document.querySelector(".conflict-point")?.textContent).toContain(
      "边界口径不清：范围 A 是否包含降级路径",
    );
    // 正方：qa 反对位（round 2 最新）
    const pro = screen.getByTestId("conflict-pro");
    expect(pro.textContent).toContain("qa · qa.review（轮 2）");
    expect(pro.textContent).not.toContain("rd · rd.review");
    // 反方：普通回复呈现、作废回复不进对照
    const con = screen.getByTestId("conflict-con");
    expect(con.textContent).toContain("已按降级路径不包含处理");
    expect(con.textContent).not.toContain("反对：修订未覆盖边界 2");
  });

  it("裁决回填条目：replies kind=adjudication 呈现结论 + session 关联", () => {
    renderCard(mkIssue(), mkNode());
    const box = screen.getByTestId("conflict-adjudications");
    expect(box.textContent).toContain("维持议题");
    expect(box.textContent).toContain("按含降级路径的口径执行");
    expect(box.textContent).toContain("session ses-7");
  });

  it("escalate=1：呈 uphold/override 仲裁表单，结论必填；提交装配 wb_adjudicate", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn((input: RequestInfo | URL, _init?: RequestInit) => {
      expect(String(input)).toBe("/api/comments/cm-9/adjudicate");
      return Promise.resolve(
        jsonResponse(200, {
          ok: true,
          commentId: "cm-9",
          verdict: "override",
          sessionId: "ses-7",
          reentered: true,
          round: "3",
          issue: mkIssue({ state: "resolved", escalate: "0" }),
        }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    renderCard(mkIssue(), mkNode());
    // 表单存在；结论空 → 提交置灰
    const submit = screen.getByRole("button", { name: /提交仲裁/ });
    expect(submit).toBeDisabled();
    await user.selectOptions(screen.getByLabelText("仲裁结论"), "override");
    await user.type(screen.getByLabelText("裁决结论"), "按仲裁口径推翻议题");
    await user.click(submit);
    await waitFor(() => {
      expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({
        commentId: "cm-9",
        verdict: "override",
        conclusion: "按仲裁口径推翻议题",
        decidedBy: "user",
      });
    });
    // 成功提示：仲裁回填 + reviewer 重表态轮已拉起 + 跳转
    expect(await screen.findByText(/仲裁已回填：仲裁推翻/)).toBeInTheDocument();
    expect(screen.getByText(/reviewer 重表态轮已拉起（第 3 轮）/)).toBeInTheDocument();
  });

  it("escalate=1 但后端拒绝（escalate 已复位 409 GATE_BLOCKED）：AlertBanner 兜底", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn((_input: RequestInfo | URL) =>
      jsonResponse(409, {
        ok: false,
        error: "评论未上升裁决（escalate=0），拒绝回填",
        code: "GATE_BLOCKED",
        blocked: true,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    renderCard(mkIssue(), mkNode());
    await user.type(screen.getByLabelText("裁决结论"), "x");
    await user.click(screen.getByRole("button", { name: /提交仲裁/ }));
    expect(await screen.findByText(/GATE_BLOCKED/)).toBeInTheDocument();
  });

  it("escalate=0：呈「上升裁决」入口（wb_escalate 拉起 + 幂等 noop 提示）", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      expect(String(input)).toBe("/api/comments/cm-9/escalate");
      expect(JSON.parse(String(init?.body))).toEqual({
        commentId: "cm-9",
        operator: "user",
        reason: "reopen 拉锯不收敛",
      });
      return Promise.resolve(
        jsonResponse(200, {
          ok: true,
          commentId: "cm-9",
          escalated: false,
          noop: true,
          sessionId: "ses-7",
        }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    renderCard(mkIssue({ escalate: "0" }), mkNode());
    expect(screen.queryByLabelText("仲裁结论")).toBeNull();
    await user.type(screen.getByLabelText("上升理由"), "reopen 拉锯不收敛");
    await user.click(screen.getByRole("button", { name: /上升裁决/ }));
    await waitFor(() =>
      expect(screen.getByText(/幂等 no-op/)).toBeInTheDocument(),
    );
  });
});
