import { afterEach, describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { INBOX_POLL_INTERVAL_MS, useInboxUnreadBadge } from "./queries";
import { jsonResponse } from "../test-utils";

/** 未读 badge 轮询（工单 T16 AC：badge 轮询刷新，MVP 轮询口径 30s 可配）。 */
describe("useInboxUnreadBadge", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("默认轮询口径 30s（VITE_INBOX_POLL_MS 未配置）", () => {
    expect(INBOX_POLL_INTERVAL_MS).toBe(30_000);
  });

  it("unread 只取 total：解码未读数；refetchInterval 到期自动重查", async () => {
    vi.useFakeTimers();
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      void input;
      return Promise.resolve(jsonResponse(200, { ok: true, notifications: [], total: "5", todos: [] }));
    });
    vi.stubGlobal("fetch", fetchMock);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useInboxUnreadBadge(1_000), { wrapper });

    // fake timers 下 waitFor 挂在 setTimeout 上，直接推进时钟并断言
    await vi.advanceTimersByTimeAsync(50);
    expect(result.current.data).toBe(5);
    expect(result.current.isSuccess).toBe(true);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/inbox?read_filter=unread&page_size=1");

    // refetchInterval=1s 到期自动重查
    await vi.advanceTimersByTimeAsync(1_100);
    expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2);
    queryClient.clear();
  });
});
