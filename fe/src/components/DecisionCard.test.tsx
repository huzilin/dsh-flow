import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ArtifactItem, FlowNodeItem, IssueItem } from "../api/proto/api";
import { DecisionActions, DecisionCard } from "./DecisionCard";
import { jsonResponse } from "../test-utils";

/**
 * 拍板卡三态测试（工单 T16 AC，spec §4.2 / ui-interaction-final §1.3）：
 * 进度只读 / agreed 迎宾呈现（汇总+按钮组+跳转）/ 已裁决（verdict 徽标+结论摘要+
 * rejected 重入相位提示）；DecisionActions 提交装配与 409 改判冲突分支。
 */

function mkGateNode(over: Partial<FlowNodeItem> = {}): FlowNodeItem {
  return {
    nodeId: "n-05",
    defNodeId: "bus-03",
    nodeOrder: "4",
    nodeKey: "产品设计评审",
    state: "review",
    reviewResults: "[]",
    reopenSource: "",
    rolledBackCount: "0",
    featureBranch: "",
    enteredAt: "1730000000000",
    nodeType: "review",
    gate: "1",
    userGate: "1",
    reviewers:
      '[{"role":"arch","capabilityId":"arch.prd-review"},{"role":"rd","capabilityId":"rd.prd-review"}]',
    lastDecision: "",
    ...over,
  };
}

const RESULTS_AGREED =
  '[{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":1,"at":1730000000000},{"role":"rd","capabilityId":"rd.prd-review","state":"agree","round":1,"at":1730000001000}]';
const RESULTS_PROGRESS =
  '[{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":1,"at":1730000000000},{"role":"rd","capabilityId":"rd.prd-review","state":"reviewing","round":1,"at":1730000001000}]';

const ARTIFACT: ArtifactItem = {
  artifactId: "art-1",
  nodeId: "n-05",
  productType: "prd",
  label: "《产品需求文档》",
  href: "docs/prd.md",
  kind: "document",
  note: "",
  addedBy: "pm",
  createdAt: "1730000000000",
};

const ISSUE: IssueItem = {
  issueId: "cm-1",
  nodeId: "n-05",
  anchorKind: "document",
  author: "qa",
  roleId: "qa",
  text: "边界口径",
  state: "resolved",
  escalate: "0",
  loopCount: "0",
  roundCount: "1",
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
  replies: "[]",
  lastReplyAt: "0",
  resolvePath: "agree",
  resolveBy: "rd",
  resolveAt: "1730000002000",
  resolveNote: "按原口径执行",
  resolveReviewerConfirmed: "1",
  resolvedBy: "qa",
  resolvedAt: "1730000003000",
  updatedAt: "1730000000000",
};

function renderCard(node: FlowNodeItem, options: { withActions?: boolean } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DecisionCard
          node={node}
          issues={[ISSUE]}
          artifacts={[ARTIFACT]}
          actions={
            options.withActions ? <DecisionActions requirementId="req-001" node={node} /> : undefined
          }
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("DecisionCard 三态", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("态一 进度只读：表态计数呈现，无拍板按钮", () => {
    const { container } = renderCard(mkGateNode({ reviewResults: RESULTS_PROGRESS }));
    expect(screen.getByText(/表态 2\/2 · ✅同意 1 · ⛔反对 0 · ○待表态 1/)).toBeInTheDocument();
    expect(screen.getByText(/评审轮进行中/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /通过/ })).toBeNull();
    expect(container.querySelector('[data-gate-view=""]')).not.toBeNull();
  });

  it("态二 agreed 迎宾呈现：评审位结论+工单摘要+产物+建议+通过/打回按钮组+流程树跳转", () => {
    const { container } = renderCard(
      mkGateNode({ reviewResults: RESULTS_AGREED }),
      { withActions: true },
    );
    expect(container.querySelector('[data-gate-view="agreed"]')).not.toBeNull();
    expect(screen.getByText("门禁汇总")).toBeInTheDocument();
    // 各评审位结论
    expect(screen.getByText(/arch · arch.prd-review：同意/)).toBeInTheDocument();
    expect(screen.getByText(/rd · rd.prd-review：同意/)).toBeInTheDocument();
    // resolved 摘要（resolve note 留痕）
    expect(screen.getByText(/cm-1 · resolved · 按原口径执行/)).toBeInTheDocument();
    // 产物链接与建议
    expect(screen.getByText(/《产品需求文档》/)).toBeInTheDocument();
    expect(screen.getByText(/建议：评审位全体同意/)).toBeInTheDocument();
    // 按钮组 + 跳转
    expect(screen.getByRole("button", { name: "✅ 通过" })).toBeEnabled();
    expect(screen.getByRole("button", { name: /⛔ 打回/ })).toBeDisabled(); // 打回必填理由，未填置灰
    expect(screen.getByText("↗ 跳转流程树核对相关节点").closest("a")).toHaveAttribute(
      "href",
      "#flow-tree",
    );
  });

  it("态二快照：agreed 迎宾结构锁定", () => {
    const { container } = renderCard(mkGateNode({ reviewResults: RESULTS_AGREED }), {
      withActions: true,
    });
    expect(container.querySelector(".decision-card")).toMatchSnapshot();
  });

  it("态三 已裁决 approved：verdict 徽标 + 结论摘要（branch/reason/时间）", () => {
    const node = mkGateNode({
      state: "respond",
      lastDecision:
        '{"verdict":"approved","reason":"评审齐备","at":1730000010000,"branch":"feat/x"}',
    });
    const { container } = renderCard(node);
    expect(container.querySelector('[data-gate-view="approved"]')).not.toBeNull();
    expect(screen.getByText("已拍板通过")).toBeInTheDocument();
    expect(screen.getByText(/理由：评审齐备/)).toBeInTheDocument();
    expect(screen.getByText(/分支：feat\/x/)).toBeInTheDocument();
    expect(screen.queryByText(/已打回/)).toBeNull();
  });

  it("态三 已裁决 rejected：重入提示按快照相位切换（review=等待重入 / exec=返工执行中）", () => {
    const lastDecision = '{"verdict":"rejected","reason":"范围未定","at":1730000010000}';
    const { rerender } = renderCard(mkGateNode({ state: "review", lastDecision }));
    expect(screen.getByText("已拍板打回")).toBeInTheDocument();
    expect(screen.getByText(/已打回，等待 exec 重入/)).toBeInTheDocument();

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    rerender(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DecisionCard node={mkGateNode({ state: "exec", lastDecision })} issues={[ISSUE]} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByText(/已打回重入：exec 返工执行中/)).toBeInTheDocument();
  });
});

describe("DecisionActions（wb_decision 提交）", () => {
  it("提交装配：verdict/reason/branch；同 verdict 幂等 noop 提示", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/decision")) {
        expect(JSON.parse(String(init?.body))).toEqual({
          requirementId: "req-001",
          nodeId: "n-05",
          verdict: "approved",
          reason: "同意",
          branch: "feat/x",
        });
        return Promise.resolve(
          jsonResponse(200, {
            ok: true,
            nodeId: "n-05",
            verdict: "approved",
            noop: true,
            lastDecision: '{"verdict":"approved","reason":"同意","at":1}',
            nextNodeId: "",
          }),
        );
      }
      return Promise.resolve(jsonResponse(200, { ok: true }));
    });
    vi.stubGlobal("fetch", fetchMock);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DecisionActions requirementId="req-001" node={mkGateNode({ reviewResults: RESULTS_AGREED })} />
      </QueryClientProvider>,
    );
    await user.type(screen.getByLabelText("拍板理由"), "同意");
    await user.type(screen.getByLabelText("评审敲定分支名"), "feat/x");
    await user.click(screen.getByRole("button", { name: "✅ 通过" }));
    await waitFor(() =>
      expect(screen.getByText(/重复拍板幂等 no-op/)).toBeInTheDocument(),
    );
  });

  it("409 改判冲突：提示冲突并说明已刷新（IDEMPOTENT_CONFLICT）", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      if (String(input).endsWith("/decision")) {
        return Promise.resolve(
          jsonResponse(409, {
            ok: false,
            error: '节点 "n-05" 已拍板 "rejected"，拒绝改判 "approved"',
            code: "IDEMPOTENT_CONFLICT",
          }),
        );
      }
      return Promise.resolve(jsonResponse(200, { ok: true }));
    });
    vi.stubGlobal("fetch", fetchMock);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DecisionActions requirementId="req-001" node={mkGateNode({ reviewResults: RESULTS_AGREED })} />
      </QueryClientProvider>,
    );
    await user.click(screen.getByRole("button", { name: "✅ 通过" }));
    await waitFor(() =>
      expect(screen.getByText(/拍板冲突（409 IDEMPOTENT_CONFLICT）/)).toBeInTheDocument(),
    );
    expect(
      screen.getByText(/已刷新最新拍板状态，请基于当前结论操作/),
    ).toBeInTheDocument();
  });

  it("rejected 缺理由：按钮置灰（reject 结论协议前端收敛）", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      void init;
      return Promise.resolve(jsonResponse(200, { ok: true }));
    });
    vi.stubGlobal("fetch", fetchMock);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <DecisionActions requirementId="req-001" node={mkGateNode({ reviewResults: RESULTS_AGREED })} />
      </QueryClientProvider>,
    );
    const rejectBtn = screen.getByRole("button", { name: /⛔ 打回/ });
    expect(rejectBtn).toBeDisabled();
    await user.type(screen.getByLabelText("拍板理由"), "打回理由");
    expect(rejectBtn).toBeEnabled();
    await user.click(rejectBtn);
    await waitFor(() => {
      const body = JSON.parse(String(fetchMock.mock.calls[0][1]?.body));
      expect(body.verdict).toBe("rejected");
      expect(body.reason).toBe("打回理由");
    });
  });
});
