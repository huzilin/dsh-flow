import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import type { InboxItem, ListInboxResp } from "../api/proto/api";
import { NOTIFICATION_KINDS, NOTIFICATION_KIND_TARGETS } from "../api/constants";
import { InboxPage, inboxItemLink } from "./InboxPage";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { jsonResponse } from "../test-utils";

/**
 * 收件箱页测试（工单 T16 验收 + spec -D5 TODO 五分类投影）：
 * 七类渲染/未读标记/角色·类别·已读筛选/分页/跳转定位/TODO 五分类投影（含 retest 子型
 * 归类与跳转锚点）/待拍板提醒条/全局冲突入口。
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

function inboxResp(notifications: InboxItem[], total = String(notifications.length)): ListInboxResp {
  // TODO 区已改为通知同源投影（spec -D5），todos 通道恒空（后端字段保留）
  return { ok: true, notifications, total, todos: [] };
}

function routeFetch(responders: Array<(url: URL) => Response | undefined>) {
  return vi.fn((input: RequestInfo | URL): Promise<Response> => {
    const url = new URL(String(input), "http://localhost");
    for (const r of responders) {
      const res = r(url);
      if (res) return Promise.resolve(res);
    }
    return Promise.resolve(jsonResponse(200, inboxResp([])));
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
        return jsonResponse(200, inboxResp(escalated, "1"));
      }
      const readFilter = url.searchParams.get("read_filter");
      const list = all.filter((n) => (readFilter === "unread" ? n.state === "open" : true));
      // total=40（>pageSize）保证翻页按钮可用（后端分页总数口径）
      return jsonResponse(200, inboxResp(list, "40"));
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

  it("TODO 五分类投影：dock 条计数=五分类之和，组按 -D5 映射归类，提醒类不产生 TODO", async () => {
    // mock 覆盖 -D5 全分支：gate_pending/clarify_task/re_review×2（澄清子型 targetId 空 +
    // retest 子型 targetId 非空）/escalation，另含 stall_followup（提醒类，不产生 TODO）
    const list = [
      mkInbox("gate_pending"),
      mkInbox("clarify_task"),
      mkInbox("re_review", { targetId: "" }),
      mkInbox("re_review", { issueId: "cm-retest", targetId: "cm-issue-9" }),
      mkInbox("escalation"),
      mkInbox("stall_followup"),
    ];
    vi.stubGlobal(
      "fetch",
      routeFetch([
        (url: URL) => {
          if (url.pathname !== "/api/inbox") return undefined;
          if (url.searchParams.get("kind") === "escalation") {
            return jsonResponse(200, inboxResp([mkInbox("escalation")], "1"));
          }
          return jsonResponse(200, inboxResp(list));
        },
      ]),
    );
    renderPage();
    // 等列表响应落地（TODO 投影区首帧先以空数据渲染）
    await screen.findAllByText(/通知文本/);
    const todo = screen.getByLabelText("TODO 投影");

    // dock 条：未完成=五分类条数之和（stall_followup 不计）
    const dock = within(todo).getByLabelText("TODO 五分类统计");
    expect(dock).toHaveTextContent("未完成 5");
    expect(dock).toHaveTextContent("待拍板 1");
    expect(dock).toHaveTextContent("待澄清 1");
    expect(dock).toHaveTextContent("待重表态 1");
    expect(dock).toHaveTextContent("待复验 1");
    expect(dock).toHaveTextContent("冲突待裁决 1");

    // 分组小节：五组各 1 条；retest 子型（targetId 非空）归「待复验」而非「待重表态」
    const gate = within(todo).getByText("待拍板（1）").closest("div");
    expect(gate?.textContent).toContain("通知文本 gate_pending");
    const reState = within(todo).getByText("待重表态（1）").closest("div");
    expect(reState?.textContent).toContain("通知文本 re_review");
    // 澄清子型仅 node 定位，不带 issue 锚点
    expect(reState?.querySelector('a[href*="focus=issue"]')).toBeNull();
    const retest = within(todo).getByText("待复验（1）").closest("div");
    // retest 子型（targetId 非空）跳意见区锚点
    expect(retest?.querySelector('a[href*="focus=issue&issue=cm-issue-9"]')).not.toBeNull();
    const conflict = within(todo).getByText("冲突待裁决（1）").closest("div");
    expect(conflict?.textContent).toContain("通知文本 escalation");
    // 提醒类不产生 TODO：stall_followup 仅出现在通知列表，不在 TODO 投影区
    expect(within(todo).queryByText("通知文本 stall_followup")).toBeNull();
    expect(screen.getAllByText("通知文本 stall_followup")).toHaveLength(1);
  });

  it("retest 子型跳转：re_review 且 target_id 非空 → 意见区锚点 focus=issue&issue=", async () => {
    // 纯函数分支：澄清子型（targetId 空）仅 node 定位，retest 子型追加 issue 锚点
    expect(inboxItemLink(mkInbox("re_review"))).toBe("/requirements/req-001?node=n-05");
    expect(inboxItemLink(mkInbox("re_review", { targetId: "cm-issue-9" }))).toBe(
      "/requirements/req-001?node=n-05&focus=issue&issue=cm-issue-9",
    );
    // requirementId 空=不可跳（基线口径不变）
    expect(
      inboxItemLink(mkInbox("gate_pending", { requirementId: "", nodeId: "" })),
    ).toBeNull();
  });

  it("待拍板提醒条：存在未读 gate_pending 时呈现并可锚点滚动，已读/无未读不呈现", async () => {
    const user = userEvent.setup();
    Element.prototype.scrollIntoView = vi.fn();
    vi.stubGlobal("fetch", routeFetch(defaultResponders()));
    const first = renderPage();
    const toast = await screen.findByLabelText("待拍板提醒");
    expect(toast.textContent).toContain("📬 有 1 条『待你拍板』待处理");
    await user.click(within(toast).getByRole("button", { name: "查看待拍板 →" }));
    expect(Element.prototype.scrollIntoView).toHaveBeenCalled();
    first.unmount();

    // 全部已读：不呈现提醒条
    vi.stubGlobal(
      "fetch",
      routeFetch([
        (url: URL) => {
          if (url.pathname !== "/api/inbox") return undefined;
          if (url.searchParams.get("kind") === "escalation") {
            return jsonResponse(200, inboxResp([], "0"));
          }
          return jsonResponse(
            200,
            inboxResp([mkInbox("gate_pending", { state: "resolved" })]),
          );
        },
      ]),
    );
    renderPage();
    // 通知列表 + TODO 待拍板分组各渲染一次（同源投影）
    await screen.findAllByText("通知文本 gate_pending");
    expect(screen.queryByLabelText("待拍板提醒")).toBeNull();
  });

  it("全局冲突入口：escalation 未读聚合区 + 跳转 issue 锚点", async () => {
    vi.stubGlobal("fetch", routeFetch(defaultResponders()));
    renderPage();
    const entry = await screen.findByLabelText("全局冲突入口");
    expect(entry).toHaveTextContent(/冲突待裁决/);
    expect(entry.querySelector('a[href*="focus=issue"]')).not.toBeNull();
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
});
