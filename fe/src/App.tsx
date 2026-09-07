import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { NavLink } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RequirementPoolPage } from "./pages/RequirementPoolPage";
import { BoardPage } from "./pages/BoardPage";
import { VersionsPage } from "./pages/VersionsPage";
import { RequirementDetailPage } from "./pages/RequirementDetailPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 5_000 },
  },
});

/** 模块 I/II 骨架（T14/T15）：需求池 / 看板 / 版本管理 / 需求详情（流程树+五区分栏+阅读器）；收件箱与拍板卡三态为 T16 范围，不注册。 */
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
          </nav>
        </header>
        <main className="content">
          <Routes>
            <Route path="/" element={<Navigate to="/pool" replace />} />
            <Route path="/pool" element={<RequirementPoolPage />} />
            <Route path="/board" element={<BoardPage />} />
            <Route path="/versions" element={<VersionsPage />} />
            <Route path="/requirements/:reqId" element={<RequirementDetailPage />} />
            <Route path="*" element={<Navigate to="/pool" replace />} />
          </Routes>
        </main>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
