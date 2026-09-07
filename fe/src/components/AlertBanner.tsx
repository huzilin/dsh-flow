import type { ReactNode } from "react";
import { ApiError } from "../api/client";

/**
 * 错误横幅：按统一失败出参分支呈现。
 * 401 提示 token 口径；409 OPTIMISTIC_LOCK 提示基线冲突；blocked 原样呈现后端 error 文本。
 */
export function AlertBanner({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  if (!error) return null;
  let text: string;
  if (error instanceof ApiError) {
    if (error.isUnauthorized) {
      text = `认证失败（401）：${error.message}——检查 VITE_WBFLOW_AUTH_TOKEN 与 rd 启动的 WBFLOW_AUTH_TOKEN 是否一致`;
    } else if (error.isOptimisticLock) {
      text = `基线冲突（409 OPTIMISTIC_LOCK）：${error.message}——已刷新最新数据，请基于新基线重试`;
    } else if (error.isBlocked) {
      text = `操作被拦截（${error.code}）：${error.message}`;
    } else {
      text = `请求失败（${error.status} ${error.code}）：${error.message}`;
    }
  } else {
    text = error instanceof Error ? error.message : String(error);
  }
  return (
    <div role="alert" className="alert-banner">
      <span>{text}</span>
      {onRetry && (
        <button type="button" onClick={onRetry}>
          重试
        </button>
      )}
    </div>
  );
}

/** 信息提示条（发布软约束 unfinished 清单等非错误提示）。 */
export function InfoBanner({ children }: { children: ReactNode }) {
  return (
    <div role="status" className="info-banner">
      {children}
    </div>
  );
}
