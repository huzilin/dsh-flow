import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { AdvanceVersionResp, VersionItem } from "../api/proto/api";
import { advanceVersion, createVersion, deprecateVersion, updateVersion } from "../api/versions";
import { api, ApiError } from "../api/client";
import { useVersionsQuery } from "../hooks/queries";
import { AlertBanner, InfoBanner } from "../components/AlertBanner";
import { StatusBadge } from "../components/Badges";
import {
  ConfirmDialog,
  CreateVersionDialog,
  RenameVersionDialog,
} from "../components/VersionDialogs";
import { formatTime } from "./RequirementPoolPage";

/**
 * 版本管理：版本列表（含待分配池 is_backlog 行）+ 创建/改名（乐观锁 409 冲突提示并刷新基线）+
 * advance（发布软约束：unfinished 清单提示条呈现）+ deprecate（blocked 错误原样呈现）。
 */
export function VersionsPage() {
  const queryClient = useQueryClient();
  const versions = useVersionsQuery();

  const [createOpen, setCreateOpen] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [renameTarget, setRenameTarget] = useState<VersionItem | null>(null);
  const [renameError, setRenameError] = useState<string | null>(null);
  const [deprecateTarget, setDeprecateTarget] = useState<VersionItem | null>(null);
  const [deprecateError, setDeprecateError] = useState<string | null>(null);
  const [advanceInfo, setAdvanceInfo] = useState<AdvanceVersionResp | null>(null);

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ["versions"] });

  // 乐观锁 409 统一交互：提示冲突 + invalidate 刷新基线（GET 恒回传新 updated_at）
  const onConflict = () => {
    invalidate();
    window.alert("基线冲突：版本已被他人修改，已刷新最新数据，请重试");
  };

  const createMutation = useMutation({
    mutationFn: (args: { name: string; isBacklog: boolean }) =>
      createVersion(api, { name: args.name, isBacklog: args.isBacklog }),
    onSuccess: () => {
      invalidate();
      setCreateOpen(false);
      setCreateError(null);
    },
    onError: (err) => setCreateError(err instanceof Error ? err.message : String(err)),
  });

  const renameMutation = useMutation({
    mutationFn: (args: { target: VersionItem; name: string }) =>
      updateVersion(api, args.target.versionId, {
        versionId: args.target.versionId,
        name: args.name,
        updatedAt: args.target.updatedAt, // GET 所得基线必回传（宽松模式后端也恒回传）
      }),
    onSuccess: () => {
      invalidate();
      setRenameTarget(null);
      setRenameError(null);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.isOptimisticLock) {
        setRenameTarget(null);
        setRenameError(null);
        onConflict();
      } else {
        setRenameError(err instanceof Error ? err.message : String(err));
      }
    },
  });

  const advanceMutation = useMutation({
    mutationFn: (target: VersionItem) =>
      advanceVersion(api, target.versionId, {
        versionId: target.versionId,
        updatedAt: target.updatedAt,
      }),
    onSuccess: (resp) => {
      invalidate();
      setAdvanceInfo(resp);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.isOptimisticLock) {
        onConflict();
      } else {
        window.alert(err instanceof Error ? err.message : String(err));
      }
    },
  });

  const deprecateMutation = useMutation({
    mutationFn: (target: VersionItem) =>
      deprecateVersion(api, target.versionId, {
        versionId: target.versionId,
        updatedAt: target.updatedAt,
      }),
    onSuccess: () => {
      invalidate();
      setDeprecateTarget(null);
      setDeprecateError(null);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.isOptimisticLock) {
        setDeprecateTarget(null);
        setDeprecateError(null);
        onConflict();
      } else {
        // blocked（有未完成需求引用拦截）：后端 error 文本原样呈现在弹窗内
        setDeprecateError(err instanceof Error ? err.message : String(err));
      }
    },
  });

  const items = versions.data?.items ?? [];

  return (
    <section>
      <div className="page-head">
        <h2>版本管理</h2>
        <button type="button" className="primary" onClick={() => setCreateOpen(true)}>
          ＋新建版本
        </button>
      </div>
      <AlertBanner error={versions.error} onRetry={() => void versions.refetch()} />
      {advanceInfo?.version && advanceInfo.unfinished.length > 0 && (
        <InfoBanner>
          <strong>发布软约束提示</strong>：版本 {advanceInfo.version.name}（
          {advanceInfo.version.versionId}） 已发布，内含 {advanceInfo.unfinished.length}{" "}
          条未完成需求（仅提示不拦截）：
          <ul className="unfinished-list">
            {advanceInfo.unfinished.map((ref) => (
              <li key={ref.requirementId}>
                {ref.requirementId} · {ref.title}（{ref.status}）
              </li>
            ))}
          </ul>
        </InfoBanner>
      )}
      <table className="req-table">
        <thead>
          <tr>
            <th>版本</th>
            <th>id</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>基线 updated_at</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {items.map((v) => (
            <tr key={v.versionId}>
              <td>
                {v.name}
                {v.isBacklog && <span className="badge badge-backlog">待分配池</span>}
              </td>
              <td className="muted">{v.versionId}</td>
              <td>
                <StatusBadge status={v.state} />
              </td>
              <td className="muted">{formatTime(v.createdAt)}</td>
              <td className="muted">{v.updatedAt}</td>
              <td className="row-actions">
                {v.isBacklog ? (
                  <span className="muted">待分配池锁定，无操作</span>
                ) : v.state === "已废弃" ? (
                  <span className="muted">终态</span>
                ) : (
                  <>
                    <button
                      type="button"
                      onClick={() => advanceMutation.mutate(v)}
                      disabled={advanceMutation.isPending}
                    >
                      {v.state === "已发布" ? "重新发布" : "推进"}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setRenameTarget(v);
                        setRenameError(null);
                      }}
                    >
                      改名
                    </button>
                    <button
                      type="button"
                      className="danger"
                      onClick={() => {
                        setDeprecateTarget(v);
                        setDeprecateError(null);
                      }}
                    >
                      废弃
                    </button>
                  </>
                )}
              </td>
            </tr>
          ))}
          {items.length === 0 && !versions.isLoading && (
            <tr>
              <td colSpan={6} className="muted">
                暂无版本（待分配池需求以 version_id 为空承载，无需标记行也可用）
              </td>
            </tr>
          )}
        </tbody>
      </table>

      <CreateVersionDialog
        open={createOpen}
        onClose={() => {
          setCreateOpen(false);
          setCreateError(null);
        }}
        onSubmit={(name, isBacklog) => createMutation.mutate({ name, isBacklog })}
        submitting={createMutation.isPending}
        error={createError}
      />

      {renameTarget && (
        <RenameVersionDialog
          open
          initialName={renameTarget.name}
          submitting={renameMutation.isPending}
          error={renameError}
          onClose={() => setRenameTarget(null)}
          onSubmit={(name) => renameMutation.mutate({ target: renameTarget, name })}
        />
      )}
      <ConfirmDialog
        open={deprecateTarget !== null}
        title="废弃版本"
        message={`废弃后为终态且不可恢复（版本 ${deprecateTarget?.name ?? ""}）。存在未完成需求引用时将被拦截。`}
        confirmLabel="废弃"
        submitting={deprecateMutation.isPending}
        error={deprecateError}
        onConfirm={() => deprecateTarget && deprecateMutation.mutate(deprecateTarget)}
        onClose={() => {
          setDeprecateTarget(null);
          setDeprecateError(null);
        }}
      />
    </section>
  );
}
