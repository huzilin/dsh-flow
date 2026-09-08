import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import type { InboxItem, ListInboxResp, TodoItem } from "../api/proto/api";
import { NOTIFICATION_KINDS, NOTIFICATION_KIND_TARGETS } from "../api/constants";
import { InboxPage, inboxItemLink, todoItemLink } from "./InboxPage";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { jsonResponse } from "../test-utils";

/**
 * 收件箱页测试（工单 T16 验收）：
 * 七类渲染/未读标记/角色·类别·已读筛选/分页/跳转定位/TODO 投影/全局冲突入口。
 * 后端排序（未读优先）即呈现序，FE 不重排。
 */

function mkInbox(kind: string, over: Partial<InboxItem> = {}): InboxItem {
  const targetType = NOTIFICATION_KIND_TARGETS[kind] ?? "node";
  return {
    issueId: `cm-${kind}`,
    kind,
    requirementId: "req-001",
    nodeId: "n-05",
    targetType,
    targetId: targetType === "issue" ? "cm-esc-1" : targetType === "task" ? "t-9" : "",
    text: `通知文本 ${kind}`,
    sender: "plugin",
    senderRole: "plugin",
    recipientRole: "user",
    state: "open",
    createdAt: "1730000000000",
    ...over,
  };
}

const TODO_FIXTURE: TodoItem = {
  taskId: "t-1",
  requirementId: "req-001",
  nodeId: "n-05",
  actionId: "bus-03.clarify",
  taskType: "respond",
  status: "in-progress",
  goal: "澄清工单 CM-9 的边界口径",
  createdAt: "1730000000000",
};

function inboxResp(
  notifications: InboxItem[],
  todos: TodoItem[] = [],
  total = String(notifications.length),
): ListInboxResp {
  return { ok: true, notifications, total, todos };
}

function routeFetch(responders: Array<(url: URL) => Response | undefined>) {
  return vi.fn((input: RequestInfo | URL): Promise<Response> => {
    const url = new URL(String(input), "http://localhost");
    for (const r of responders) {
      const res = r(url);
      if (res) return Promise.resolve(res);
    }
    return Promise.resolve(jsonResponse(200, inboxResp([], [], "0")));
  });
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <InboxPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/** 默认 responder：列表查询给七类全量；escalation 聚合查询给 1 条冲突。 */
function defaultResponders() {
  const all = NOTIFICATION_KINDS.map((k) => mkInbox(k));
  const escalated = [mkInbox("escalation")];
  return [
    (url: URL) => {
      if (url.pathname !== "/api/inbox") return undefined;
      if (url.searchParams.get("kind") === "escalation") {
        return jsonResponse(200, inboxResp(escalated, [], "1"));
      }
      const readFilter = url.searchParams.get("read_filter");
      const list = all.filter((n) => (readFilter === "unread" ? n.state === "open" : true));
      // total=40（>pageSize）保证翻页按钮可用（后端分页总数口径）
      return jsonResponse(200, inboxResp(list, [TODO_FIXTURE], "40"));
    },
  ];
}

describe("InboxPage（模块 III/IV）", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("七类通知全渲染：图标+标签、未读标记、跳转链接按 target_type 装配", async () => {
    vi.stubGlobal("fetch", routeFetch(defaultResponders()));
    renderPage();
    for (const k of NOTIFICATION_KINDS) {
      expect(await screen.findAllByText(`通知文本 ${k}`)).not.toHaveLength(0);
    }
    // 未读标记（state=open）
    expect(await screen.findAllByText("未读").then((els) => els.length)).toBeGreaterThanOrEqual(7);
    // 跳转：node 类别 → ?node=；escalation → focus=issue；task 类别 → focus=task
    const escLink = await screen.findByText("冲突上升裁决").then((el) => el.closest("li"));
    expect(escLink?.querySelector('a[href*="focus=issue"]')).not.toBeNull();
    const stallLink = screen.getAllByText("通知文本 stall_followup")[0].closest("li");
    expect(stallLink?.querySelector('a[href*="focus=task"]')).not.toBeNull();
    const gateLink = screen.getAllByText("通知文本 gate_pending")[0].closest("li");
    expect(gateLink?.querySelector('a[href="/requirements/req-001?node=n-05"]')).not.toBeNull();
    // session_message：通道未开放不跳转
    const sessionRow = screen.getAllByText("通知文本 session_message")[0].closest("li");
    expect(sessionRow?.querySelector("a")).toBeNull();
    expect(sessionRow?.textContent).toContain("DSH 嵌入期开放");
  });

  it("TODO 投影区：在途任务呈现（含状态徽标与节点跳转），completed/rejected 不残留由后端同源保证", async () => {
    vi.stubGlobal("fetch", routeFetch(defaultResponders()));
    renderPage();
    expect(await screen.findByText("澄清工单 CM-9 的边界口径")).toBeInTheDocument();
    expect(screen.getByText("执行中")).toBeInTheDocument();
    expect(screen.getByText("节点详情 →").closest("a")).toHaveAttribute(
      "href",
      "/requirements/req-001?node=n-05",
    );
  });

  it("全局冲突入口：escalation 未读聚合区 + 跳转 issue 锚点", async () => {
    vi.stubGlobal("fetch", routeFetch(defaultResponders()));
    renderPage();
    expect(await screen.findByText(/冲突待裁决/)).toBeInTheDocument();
    const entry = screen.getByText(/冲突待裁决/).closest("div");
    expect(entry?.querySelector('a[href*="focus=issue"]')).not.toBeNull();
  });

  it("筛选与分页：改动触发对应 query 重查（page/kind/read_filter）", async () => {
    const user = userEvent.setup();
    const fetchMock = routeFetch(defaultResponders());
    vi.stubGlobal("fetch", fetchMock);
    renderPage();
    await screen.findAllByText(/通知文本/);

    await user.selectOptions(screen.getByLabelText("通知类别筛选"), "gate_pending");
    await user.selectOptions(screen.getByLabelText("已读状态筛选"), "unread");
    await user.selectOptions(screen.getByLabelText("收件角色筛选"), "user");
    await waitFor(() => {
      const calls = fetchMock.mock.calls.map((c) => new URL(String(c[0]), "http://localhost"));
      expect(
        calls.some(
          (u) =>
            u.searchParams.get("kind") === "gate_pending" &&
            u.searchParams.get("read_filter") === "unread" &&
            u.searchParams.get("role") === "user",
        ),
      ).toBe(true);
    });
    // 等筛选响应渲染完成（total=7 → 下一页可用）再翻页
    await screen.findAllByText(/通知文本/);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "下一页" })).toBeEnabled(),
    );

    await user.click(screen.getByRole("button", { name: "下一页" }));
    await waitFor(() => {
      const calls = fetchMock.mock.calls.map((c) => new URL(String(c[0]), "http://localhost"));
      expect(calls.some((u) => u.searchParams.get("page") === "2")).toBe(true);
    });
  });

  it("inboxItemLink/todoItemLink 纯函数：requirementId 空=不可跳", () => {
    expect(
      inboxItemLink(mkInbox("gate_pending", { requirementId: "", nodeId: "" })),
    ).toBeNull();
    expect(
      todoItemLink({ ...TODO_FIXTURE, nodeId: "" }),
    ).toBe("/requirements/req-001");
    expect(todoItemLink({ ...TODO_FIXTURE, requirementId: "" })).toBeNull();
  });
});
