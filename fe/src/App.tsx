import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { NavLink } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RequirementPoolPage } from "./pages/RequirementPoolPage";
import { BoardPage } from "./pages/BoardPage";
import { VersionsPage } from "./pages/VersionsPage";
import { RequirementDetailPage } from "./pages/RequirementDetailPage";
import { InboxPage } from "./pages/InboxPage";
import { INBOX_POLL_INTERVAL_MS, useInboxUnreadBadge } from "./hooks/queries";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 5_000 },
  },
});

/** 模块 I–IV 骨架（T14/T15/T16）：需求池 / 看板 / 版本管理 / 需求详情 / 收件箱。 */
export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <header className="topbar">
          <span className="brand">wbflow 工作台</span>
          <nav>
            <NavLink to="/pool">需求池</NavLink>
            <NavLink to="/board">看板</NavLink>
            <NavLink to="/versions">版本管理</NavLink>
            <NavLink to="/inbox" aria-label="收件箱">
              收件箱
              <InboxBadge />
            </NavLink>
          </nav>
        </header>
        <main className="content">
          <Routes>
            <Route path="/" element={<Navigate to="/board" replace />} />
            <Route path="/pool" element={<RequirementPoolPage />} />
            <Route path="/board" element={<BoardPage />} />
            <Route path="/versions" element={<VersionsPage />} />
            <Route path="/inbox" element={<InboxPage />} />
            <Route path="/requirements/:reqId" element={<RequirementDetailPage />} />
            <Route path="*" element={<Navigate to="/pool" replace />} />
          </Routes>
        </main>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

/** 顶栏收件箱未读 badge（MVP 轮询口径 30s；未读 0 时隐藏）。 */
function InboxBadge() {
  const unread = useInboxUnreadBadge(INBOX_POLL_INTERVAL_MS);
  const count = unread.data ?? 0;
  if (count <= 0) return <span className="inbox-badge-slot" data-testid="inbox-badge" data-count="0" />;
  return (
    <span className="badge inbox-badge" data-testid="inbox-badge" data-count={count}>
      {count > 99 ? "99+" : count}
    </span>
  );
}
