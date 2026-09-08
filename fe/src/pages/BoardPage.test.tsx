import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ListRequirementsResp, ListVersionsResp, RequirementItem } from "../api/proto/api";
import { BoardPage } from "./BoardPage";
import { jsonResponse } from "../test-utils";

/**
 * 看板页测试（原型 P8）：四泳道卡片（scope 三色/依赖角标/✎ 编辑入口）、
 * 列表视图按状态分组（空组整组隐藏）、行点击跳详情、编辑弹窗打开且阻断冒泡。
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
    status: "进行中",
    owner: "rd",
    createdBy: "pm",
    createdAt: "1730000000000",
    updatedAt: "1730000000000",
    dependencies: [],
    ...over,
  };
}

const CARDS: RequirementItem[] = [
  mkReq({
    requirementId: "req-a",
    title: "交易对账明细导出",
    status: "进行中",
    flowDefsId: "business",
    dependencies: ["req-b"],
  }),
  mkReq({
    requirementId: "req-b",
    title: "v1.1 版本规划",
    status: "待启动",
    category: "版本规划任务",
    flowDefsId: "plan",
    priority: "P3",
  }),
];

function versionsResp(): ListVersionsResp {
  return {
    ok: true,
    items: [
      {
        versionId: "ver-1",
        name: "v1.0",
        state: "进行中",
        isBacklog: false,
        createdAt: "1730000000000",
        updatedAt: "1730000000000",
      },
    ],
  };
}

function reqsResp(items: RequirementItem[]): ListRequirementsResp {
  return { ok: true, total: String(items.length), page: "1", pageSize: "100", items };
}

function routeFetch(items: RequirementItem[]) {
  return vi.fn((input: RequestInfo | URL): Promise<Response> => {
    const url = new URL(String(input), "http://localhost");
    if (url.pathname === "/api/versions") {
      return Promise.resolve(jsonResponse(200, versionsResp()));
    }
    if (url.pathname === "/api/requirements") {
      return Promise.resolve(jsonResponse(200, reqsResp(items)));
    }
    return Promise.resolve(jsonResponse(404, { ok: false, error: "not found", code: "NOT_FOUND" }));
  });
}

function renderBoard() {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({
          defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
        })
      }
    >
      <MemoryRouter initialEntries={["/board"]}>
        <Routes>
          <Route path="/board" element={<BoardPage />} />
          <Route path="/requirements/:reqId" element={<div>需求详情占位</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("BoardPage（原型 P8 双视图）", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("看板视图：四泳道卡片含 scope 三色徽标（plan 显 —）与依赖角标（hover 列前置清单）", async () => {
    vi.stubGlobal("fetch", routeFetch(CARDS));
    renderBoard();
    expect(await screen.findByText("交易对账明细导出")).toBeInTheDocument();
    // 泳道分组头（待启动/进行中/已完成/已取消四泳道恒渲染）
    expect(screen.getByText("待启动")).toBeInTheDocument();
    expect(screen.getByText("已取消")).toBeInTheDocument();
    // scope 三色：business→full 徽标；plan→「—」
    expect(screen.getByText("full")).toBeInTheDocument();
    // 依赖角标：×N + title 列前置 id 清单
    const dep = screen.getByText("依赖 ×1");
    expect(dep).toHaveAttribute("title", "前置依赖：req-b");
    // 编辑占位（✎）在卡片标题行
    expect(screen.getByRole("button", { name: "编辑 req-a" })).toBeInTheDocument();
  });

  it("列表视图：按状态分组头（● 状态 · n 项），空组整组隐藏，行点击跳详情", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("fetch", routeFetch(CARDS));
    renderBoard();
    await screen.findByText("交易对账明细导出");
    await user.click(screen.getByRole("button", { name: "列表" }));
    // 分组头（进行中/待启动各 1 项）
    expect(await screen.findByText("● 进行中 · 1 项")).toBeInTheDocument();
    expect(screen.getByText("● 待启动 · 1 项")).toBeInTheDocument();
    // 空组隐藏：无已完成/已取消需求 → 分组头不渲染
    expect(screen.queryByText(/● 已完成/)).not.toBeInTheDocument();
    expect(screen.queryByText(/● 已取消/)).not.toBeInTheDocument();
    // 行含 reqId；点击行跳详情
    await user.click(screen.getByText("req-a", { selector: "span", exact: false }));
    await waitFor(() => {
      expect(screen.getByText("需求详情占位")).toBeInTheDocument();
    });
  });

  it("编辑入口：点击 ✎ 打开编辑弹窗且不触发卡片跳详情（N4 编辑复用）", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("fetch", routeFetch(CARDS));
    renderBoard();
    await screen.findByText("交易对账明细导出");
    await user.click(screen.getByRole("button", { name: "编辑 req-a" }));
    // 弹窗打开（编辑模式标题+基线回填），且卡片跳详情未触发
    expect(await screen.findByText("编辑需求")).toBeInTheDocument();
    expect(screen.getByLabelText(/^标题/)).toHaveValue("交易对账明细导出");
    expect(screen.queryByText("需求详情占位")).not.toBeInTheDocument();
  });
});
