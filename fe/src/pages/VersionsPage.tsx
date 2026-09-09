import { Fragment, useState } from "react";
import { useMutation, useQueries, useQueryClient } from "@tanstack/react-query";
import type {
  AdvanceVersionResp,
  ListRequirementsResp,
  RequirementItem,
  VersionItem,
} from "../api/proto/api";
import { advanceVersion, createVersion, deprecateVersion, updateVersion } from "../api/versions";
import { listRequirements } from "../api/requirements";
import { api, ApiError } from "../api/client";
import { REQUIREMENT_STATUSES } from "../api/constants";
import { useVersionsQuery } from "../hooks/queries";
import { AlertBanner, InfoBanner } from "../components/AlertBanner";
import { StatusBadge } from "../components/Badges";
import { VersionDependencyForest } from "../components/DependencyForest";
import {
  ConfirmDialog,
  CreateVersionDialog,
  RenameVersionDialog,
} from "../components/VersionDialogs";
import { formatTime } from "./RequirementPoolPage";

/**
 * 版本管理（原型 P9）：版本列表（含待分配池 is_backlog 行，锁定无操作）+
 * 创建/改名（乐观锁 409 冲突提示并刷新基线）+ advance（发布软约束：unfinished 清单提示条呈现）+
 * deprecate（blocked 错误原样呈现）+ v3 增补：需求聚合四色进度条 / 依赖树按需拉取 / 并发调度预留区。
 * 注：spec §1.2 无版本删除端点，不做删除按钮（原型 disabled 删除仅为展示态，不在本期范围）。
 */

/** 版本需求聚合（四状态计数 + 总数；单次拉取口径同看板：page_size=100 后端强制上限）。 */
export interface VersionReqStats {
  total: number;
  /** total 超过单页 100 时为 true（计数仅覆盖前 100 条，行内提示） */
  truncated: boolean;
  counts: Record<(typeof REQUIREMENT_STATUSES)[number], number>;
  items: RequirementItem[];
}

/** 列表出参 → 聚合（纯函数，单测直测）。 */
export function toVersionReqStats(resp: ListRequirementsResp): VersionReqStats {
  const counts = { 待启动: 0, 进行中: 0, 已完成: 0, 已取消: 0 };
  for (const it of resp.items) {
    if ((REQUIREMENT_STATUSES as readonly string[]).includes(it.status)) {
      counts[it.status as keyof typeof counts] += 1;
    }
  }
  const total = Number(resp.total);
  return { total, truncated: total > resp.items.length, counts, items: resp.items };
}

/**
 * 就绪池判定（原型 P9 并发调度区）：待启动且 dependencies 全部已完成（逐行判定）。
 * 跨版本依赖按全局状态表判定；依赖 id 状态未知（超页/悬挂）= 不就绪（fail closed）。
 */
export function computeReadyPool(
  items: RequirementItem[],
  statusById: Map<string, string>,
): string[] {
  return items
    .filter(
      (it) =>
        it.status === "待启动" &&
        it.dependencies.every((d) => statusById.get(d) === "已完成"),
    )
    .map((it) => it.requirementId);
}

/** 需求聚合单元格（四色进度条 + 计数，原型 P9 vbar：灰/琥珀/绿/红）。 */
function VersionAggCell({ stats }: { stats: VersionReqStats | undefined }) {
  if (!stats) {
    return <span className="muted">统计中…</span>;
  }
  const pct = (n: number) => (stats.total > 0 ? (n / stats.total) * 100 : 0);
  const c = stats.counts;
  return (
    <div className="ver-agg">
      <div className="ver-agg-top">
        <span className="muted">需求 {stats.total}</span>
        <div
          className="vbar"
          role="img"
          aria-label={`待启动 ${c.待启动} · 进行中 ${c.进行中} · 已完成 ${c.已完成} · 已取消 ${c.已取消}`}
        >
          {REQUIREMENT_STATUSES.map((s) => (
            <i
              key={s}
              className={`vbar-seg vbar-seg-${s}`}
              style={{ width: `${pct(c[s as keyof typeof c])}%` }}
            />
          ))}
        </div>
      </div>
      <span className="muted ver-agg-line">
        待启动 {c.待启动} · 进行中 {c.进行中} · 已完成 {c.已完成} · 已取消 {c.已取消}
      </span>
      {stats.truncated && (
        <span className="muted ver-agg-line">（仅统计前 100 条：后端强制分页上限）</span>
      )}
    </div>
  );
}

export function VersionsPage() {
  const queryClient = useQueryClient();
  const versions = useVersionsQuery();
  /** 展开依赖树的版本 id（原型 P9：再点收起，单开） */
  const [depOpenId, setDepOpenId] = useState<string | null>(null);

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

  // 逐版本需求聚合（并发 useQueries；键名带 versionId，归入 requirements 族便于联动失效；
  // 待分配池走 backlog=true，其余按 version_id——listRequirements 同参互斥口径）
  const statQueries = useQueries({
    queries: items.map((v) => ({
      queryKey: ["requirements", "versionAgg", v.versionId],
      queryFn: () =>
        listRequirements(
          api,
          v.isBacklog
            ? { backlog: true, page: 1, pageSize: 100 }
            : { versionId: v.versionId, page: 1, pageSize: 100 },
        ),
    })),
  });
  const statsById = new Map<string, VersionReqStats | undefined>();
  items.forEach((v, i) => {
    const resp = statQueries[i]?.data;
    statsById.set(v.versionId, resp ? toVersionReqStats(resp) : undefined);
  });

  // 并发调度预留区推导（原型 P9：并发度=进行中合计；就绪池=待启动且前置全部已完成）
  const allItems = statQueries.flatMap((q) => (q.data ? q.data.items : []));
  const statusById = new Map(allItems.map((it) => [it.requirementId, it.status]));
  const working = allItems.filter((it) => it.status === "进行中");
  const readyPool = computeReadyPool(allItems, statusById);
  const schedLoaded = items.length === 0 || statQueries.every((q) => !q.isPending);

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
            <th>需求聚合</th>
            <th>创建时间</th>
            <th>基线 updated_at</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {items.map((v) => {
            const stats = statsById.get(v.versionId);
            const depOpen = depOpenId === v.versionId;
            return (
              <Fragment key={v.versionId}>
                <tr>
                  <td>
                    {v.name}
                    {v.isBacklog && <span className="badge badge-backlog">待分配池</span>}
                  </td>
                  <td className="muted">{v.versionId}</td>
                  <td>
                    <StatusBadge status={v.state} />
                  </td>
                  <td>
                    <VersionAggCell stats={stats} />
                  </td>
                  <td className="muted">{formatTime(v.createdAt)}</td>
                  <td className="muted">{v.updatedAt}</td>
                  <td className="row-actions">
                    {v.isBacklog ? (
                      <span className="muted">待分配池锁定，无操作</span>
                    ) : (
                      <>
                        {v.state !== "已废弃" && (
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
                        {v.state === "已废弃" && <span className="muted">终态</span>}
                        {/* 依赖树入口（原型 P9：按需拉取，展开/收起；已废弃版本仍可只读查看） */}
                        <button
                          type="button"
                          aria-expanded={depOpen}
                          onClick={() => setDepOpenId(depOpen ? null : v.versionId)}
                        >
                          {depOpen ? "收起依赖树" : "依赖树"}
                        </button>
                      </>
                    )}
                  </td>
                </tr>
                {depOpen && (
                  <tr key={`${v.versionId}-deproot`} className="dep-tree-row">
                    <td colSpan={7}>
                      {stats ? (
                        <VersionDependencyForest reqs={stats.items} />
                      ) : (
                        <div className="muted">需求聚合加载中，依赖树待就绪…</div>
                      )}
                    </td>
                  </tr>
                )}
              </Fragment>
            );
          })}
          {items.length === 0 && !versions.isLoading && (
            <tr>
              <td colSpan={7} className="muted">
                暂无版本（待分配池需求以 version_id 为空承载，无需标记行也可用）
              </td>
            </tr>
          )}
        </tbody>
      </table>

      {/* 并发调度预留区（原型 P9：长期预留展示，MVP 人工卡点，纯展示无操作） */}
      {schedLoaded && (
        <div className="sched-card">
          <h3>并发调度（长期预留展示 · MVP 人工卡点）</h3>
          <div className="sched-row">
            <span className="sched-key">并发度</span>
            <span>
              正在工作的需求数 · 当前 {working.length}
              {working.length > 0 && `（${working.map((it) => it.requirementId).join(" · ")}）`}
            </span>
          </div>
          <div className="sched-row">
            <span className="sched-key">就绪池</span>
            <span>
              {readyPool.length > 0
                ? `依赖已满足且待启动 ${readyPool.length} 条：${readyPool.join(" · ")}`
                : "当前无就绪需求（口径：待启动且前置依赖全部已完成）"}
            </span>
          </div>
          <div className="sched-row">
            <span className="sched-key">卡点</span>
            <span>
              MVP 人工卡点（节点 user_gate 人工放行）；依赖调度器 = 长期 effort（E9，
              flows.dependencies 数据层已预留）
            </span>
          </div>
        </div>
      )}

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
