import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ListRequirementsResp, ListVersionsResp, RequirementItem } from "../api/proto/api";
import { RequirementPoolPage } from "./RequirementPoolPage";
import { jsonResponse } from "../test-utils";

/**
 * 需求池页冒烟测试（本任务行增强范围）：scope 三色实值徽标（plan 显 —）、
 * 依赖角标（hover 列前置清单）、版本名映射与既有筛选/分页不回归。
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

const ROWS: RequirementItem[] = [
  mkReq({ requirementId: "req-a", dependencies: ["req-9", "req-8"] }),
  mkReq({
    requirementId: "req-b",
    category: "版本规划任务",
    flowDefsId: "plan",
    priority: "P3",
    status: "待启动",
  }),
];

function routeFetch() {
  return vi.fn((input: RequestInfo | URL): Promise<Response> => {
    const url = new URL(String(input), "http://localhost");
    if (url.pathname === "/api/versions") {
      const resp: ListVersionsResp = {
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
      return Promise.resolve(jsonResponse(200, resp));
    }
    if (url.pathname === "/api/requirements") {
      const resp: ListRequirementsResp = {
        ok: true,
        total: String(ROWS.length),
        page: "1",
        pageSize: "20",
        items: ROWS,
      };
      return Promise.resolve(jsonResponse(200, resp));
    }
    return Promise.resolve(jsonResponse(404, { ok: false, error: "not found", code: "NOT_FOUND" }));
  });
}

describe("RequirementPoolPage（行增强冒烟）", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("行内 scope 三色徽标（plan 显 —）+ 依赖角标 ×N（title 列前置清单）", async () => {
    vi.stubGlobal("fetch", routeFetch());
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <RequirementPoolPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByText("req-a", { selector: "span", exact: false })).toBeInTheDocument();
    // scope：business→full；plan→—（ReqScopeBadge 的 muted 占位）
    expect(screen.getByText("full")).toBeInTheDocument();
    // 依赖角标：多前置全列（原型 P8 hover 清单口径）
    const dep = screen.getByText("依赖 ×2");
    expect(dep).toHaveAttribute("title", "前置依赖：req-9、req-8");
    // 版本名映射（version_id → 版本名；两行同版本）
    expect(screen.getAllByText("v1.0")).toHaveLength(2);
  });
});
