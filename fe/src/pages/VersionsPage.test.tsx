import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type {
  DependencyNodeItem,
  ListRequirementsResp,
  ListVersionsResp,
  RequirementItem,
  VersionItem,
} from "../api/proto/api";
import { computeReadyPool, toVersionReqStats, VersionsPage } from "./VersionsPage";
import { jsonResponse } from "../test-utils";

/**
 * 版本管理页测试（原型 P9 增补）：版本行需求聚合（需求数+四色进度条+计数）、
 * 依赖树按需拉取（展开/收起、前置为父森林、无依赖边空态）、并发调度预留区、
 * 待分配池锁定无操作。聚合/就绪池纯函数（toVersionReqStats/computeReadyPool）直测。
 */

function mkReq(over: Partial<RequirementItem> = {}): RequirementItem {
  return {
    requirementId: "req-001",
    title: "需求标题",
    description: "",
    category: "需求任务",
    flowDefsId: "business",
    versionId: "ver-1",
    priority: "P1",
    status: "待启动",
    owner: "rd",
    createdBy: "pm",
    createdAt: "1730000000000",
    updatedAt: "1730000000000",
    dependencies: [],
    ...over,
  };
}

const V1: VersionItem = {
  versionId: "ver-1",
  name: "v1.0",
  state: "进行中",
  isBacklog: false,
  createdAt: "1730000000000",
  updatedAt: "1730000000000",
};
const VBL: VersionItem = {
  versionId: "ver-bl",
  name: "待分配池",
  state: "待启动",
  isBacklog: true,
  createdAt: "1730000000000",
  updatedAt: "1730000000000",
};

/** 版本内需求集：req-a 前置 req-b（进行中，故未就绪）；req-c 无前置（就绪）。 */
const V1_REQS: RequirementItem[] = [
  mkReq({ requirementId: "req-a", title: "明细导出", dependencies: ["req-b"] }),
  mkReq({ requirementId: "req-b", title: "对账工单流", status: "进行中" }),
  mkReq({ requirementId: "req-c", title: "前端优化" }),
];

/** tree 出参装配：directMap[reqId]=其直接前置 id 集（children 递归口径同后端）。 */
function treeNode(
  requirementId: string,
  directMap: Record<string, string[]>,
  all: RequirementItem[],
): DependencyNodeItem {
  const src = all.find((r) => r.requirementId === requirementId);
  return {
    requirementId,
    title: src?.title ?? "",
    status: src?.status ?? "",
    versionId: src?.versionId ?? "",
    found: true,
    children: (directMap[requirementId] ?? []).map((id) => treeNode(id, directMap, all)),
  };
}

function routeFetch(v1Reqs: RequirementItem[], directMap: Record<string, string[]> = {}) {
  return vi.fn((input: RequestInfo | URL): Promise<Response> => {
    const url = new URL(String(input), "http://localhost");
    if (url.pathname === "/api/versions") {
      const resp: ListVersionsResp = { ok: true, items: [V1, VBL] };
      return Promise.resolve(jsonResponse(200, resp));
    }
    if (url.pathname === "/api/requirements") {
      const isBacklog = url.searchParams.get("backlog") === "true";
      const items = isBacklog ? [mkReq({ requirementId: "req-d", status: "已取消", versionId: "" })] : v1Reqs;
      const resp: ListRequirementsResp = {
        ok: true,
        total: String(items.length),
        page: "1",
        pageSize: "100",
        items,
      };
      return Promise.resolve(jsonResponse(200, resp));
    }
    const treeMatch = url.pathname.match(/^\/api\/requirements\/(.+)\/dependencies\/tree$/);
    if (treeMatch) {
      const all = [...v1Reqs];
      return Promise.resolve(
        jsonResponse(200, {
          ok: true,
          root: treeNode(decodeURIComponent(treeMatch[1]), directMap, all),
        }),
      );
    }
    return Promise.resolve(jsonResponse(404, { ok: false, error: "not found", code: "NOT_FOUND" }));
  });
}

describe("VersionsPage（原型 P9）", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("版本行需求聚合：需求数+四色进度条（aria 双通道）+计数；待分配池锁定无依赖树入口", async () => {
    vi.stubGlobal("fetch", routeFetch(V1_REQS));
    render(
      <QueryClientProvider
        client={
          new QueryClient({
            defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
          })
        }
      >
        <MemoryRouter>
          <VersionsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    // v1.0：需求 3（待启动 2 · 进行中 1）
    expect(await screen.findByText("需求 3")).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "待启动 2 · 进行中 1 · 已完成 0 · 已取消 0" }),
    ).toBeInTheDocument();
    expect(screen.getByText("待启动 2 · 进行中 1 · 已完成 0 · 已取消 0")).toBeInTheDocument();
    // 待分配池行：需求 1（已取消）+ 锁定无操作
    expect(screen.getByText("待启动 0 · 进行中 0 · 已完成 0 · 已取消 1")).toBeInTheDocument();
    expect(screen.getByText("待分配池锁定，无操作")).toBeInTheDocument();
    // 锁定行不渲染依赖树按钮（仅非 backlog 行有）
    expect(screen.getAllByRole("button", { name: "依赖树" })).toHaveLength(1);
  });

  it("并发调度预留区：并发度=进行中合计；就绪池=待启动且前置全部已完成；卡点说明固定", async () => {
    vi.stubGlobal("fetch", routeFetch(V1_REQS));
    render(
      <QueryClientProvider
        client={
          new QueryClient({
            defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
          })
        }
      >
        <MemoryRouter>
          <VersionsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByText(/正在工作的需求数 · 当前 1（req-b）/)).toBeInTheDocument();
    // req-a 前置 req-b 进行中 → 未就绪；req-c 无前置 → 就绪
    expect(screen.getByText("依赖已满足且待启动 1 条：req-c")).toBeInTheDocument();
    expect(screen.getByText(/MVP 人工卡点（节点 user_gate/)).toBeInTheDocument();
  });

  it("依赖树按需拉取：展开并发拉取后按「前置为父」组装森林，再点收起", async () => {
    const user = userEvent.setup();
    const fetchMock = routeFetch(V1_REQS, { "req-a": ["req-b"] });
    vi.stubGlobal("fetch", fetchMock);
    render(
      <QueryClientProvider
        client={
          new QueryClient({
            defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
          })
        }
      >
        <MemoryRouter>
          <VersionsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await screen.findByText("需求 3");
    await user.click(screen.getByRole("button", { name: "依赖树" }));
    // req-b 为根（无前置），req-a 为其子（前置 req-b）——原型 P9 deproot 方向
    expect(await screen.findByText("前置 req-b")).toBeInTheDocument();
    // 两个根（req-b/req-c 均无前置）
    expect(screen.getAllByText("无前置")).toHaveLength(2);
    // 树接口按版本内需求逐个拉取（req-a/req-b/req-c 共 3 次）
    await waitFor(() => {
      const treeCalls = fetchMock.mock.calls.filter(([input]) =>
        String(input).includes("/dependencies/tree"),
      );
      expect(treeCalls).toHaveLength(3);
    });
    // ↗ 流程树链接（点击跳详情的入口）
    expect(screen.getAllByRole("button", { name: "↗ 流程树" }).length).toBeGreaterThanOrEqual(3);
    // 再点收起
    await user.click(screen.getByRole("button", { name: "收起依赖树" }));
    expect(screen.queryByText("前置 req-b")).not.toBeInTheDocument();
  });

  it("无依赖边空态：版本内需求均无前置时提示「该版本无依赖边」", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("fetch", routeFetch(V1_REQS));
    render(
      <QueryClientProvider
        client={
          new QueryClient({
            defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
          })
        }
      >
        <MemoryRouter>
          <VersionsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await screen.findByText("需求 3");
    await user.click(screen.getByRole("button", { name: "依赖树" }));
    expect(await screen.findByText("该版本无依赖边")).toBeInTheDocument();
  });
});

describe("版本聚合纯函数", () => {
  it("toVersionReqStats：四状态计数 + total 超单页截断标记", () => {
    const stats = toVersionReqStats({
      ok: true,
      total: "102",
      page: "1",
      pageSize: "100",
      items: [
        mkReq({ requirementId: "r1", status: "待启动" }),
        mkReq({ requirementId: "r2", status: "进行中" }),
        mkReq({ requirementId: "r3", status: "进行中" }),
        mkReq({ requirementId: "r4", status: "已完成" }),
        mkReq({ requirementId: "r5", status: "已取消" }),
      ],
    });
    expect(stats.total).toBe(102);
    expect(stats.truncated).toBe(true);
    expect(stats.counts).toEqual({ 待启动: 1, 进行中: 2, 已完成: 1, 已取消: 1 });
  });

  it("computeReadyPool：前置全部已完成才就绪；依赖 id 状态未知不就绪（fail closed）", () => {
    const items = [
      mkReq({ requirementId: "r1", status: "待启动", dependencies: ["r2", "r3"] }),
      mkReq({ requirementId: "r2", status: "已完成" }),
      mkReq({ requirementId: "r3", status: "进行中" }),
      mkReq({ requirementId: "r4", status: "待启动", dependencies: ["r-ghost"] }),
      mkReq({ requirementId: "r5", status: "待启动" }),
    ];
    const statusById = new Map(items.map((it) => [it.requirementId, it.status]));
    expect(computeReadyPool(items, statusById)).toEqual(["r5"]);
  });
});
