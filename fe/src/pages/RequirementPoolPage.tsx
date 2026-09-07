import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import type { CreateRequirementReq, RequirementItem } from "../api/proto/api";
import { createRequirement } from "../api/requirements";
import { api } from "../api/client";
import { FLOW_LABELS } from "../api/constants";
import { useRequirementsQuery, useVersionsQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { PriorityBadge, ScopeBadge, StatusBadge } from "../components/Badges";
import { CreateRequirementDialog } from "../components/CreateRequirementDialog";
import {
  FilterBar,
  useFilterState,
  VERSION_FILTER_ALL,
  VERSION_FILTER_BACKLOG,
} from "../components/FilterBar";
import { Pagination } from "../components/Pagination";

/** 需求池：version/priority/status 筛选 + 强制分页列表 + 新建需求（T14 范围；详情页 T15 接管）。 */
export function RequirementPoolPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const filters = useFilterState();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createOpen, setCreateOpen] = useState(false);

  const versions = useVersionsQuery();
  const list = useRequirementsQuery({
    versionId:
      filters.versionSel === VERSION_FILTER_ALL || filters.versionSel === VERSION_FILTER_BACKLOG
        ? undefined
        : filters.versionSel,
    backlog: filters.versionSel === VERSION_FILTER_BACKLOG,
    priority: filters.priority,
    status: filters.status,
    page,
    pageSize,
  });

  const [submitError, setSubmitError] = useState<string | null>(null);
  const createMutation = useMutation({
    mutationFn: (req: CreateRequirementReq) => createRequirement(api, req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["requirements"] });
      setCreateOpen(false);
      setSubmitError(null);
    },
    onError: (err) => {
      setSubmitError(err instanceof Error ? err.message : String(err));
    },
  });

  const versionName = (id: string) => {
    if (id === "") return "待分配";
    const v = versions.data?.items.find((it) => it.versionId === id);
    return v ? `${v.name}` : id;
  };

  const items = list.data?.items ?? [];

  return (
    <section>
      <div className="page-head">
        <h2>需求池</h2>
        <button type="button" className="primary" onClick={() => setCreateOpen(true)}>
          ＋新建需求
        </button>
      </div>
      <AlertBanner error={list.error} onRetry={() => void list.refetch()} />
      <FilterBar
        versions={versions.data?.items ?? []}
        versionSel={filters.versionSel}
        onVersionSel={(v) => {
          filters.setVersionSel(v);
          setPage(1);
        }}
        priority={filters.priority}
        onPriority={(p) => {
          filters.setPriority(p);
          setPage(1);
        }}
        status={filters.status}
        onStatus={(s) => {
          filters.setStatus(s);
          setPage(1);
        }}
        showStatus
      />
      <table className="req-table">
        <thead>
          <tr>
            <th>标题</th>
            <th>类别</th>
            <th>流程 / scope</th>
            <th>优先级</th>
            <th>状态</th>
            <th>版本</th>
            <th>负责人</th>
            <th>更新时间</th>
          </tr>
        </thead>
        <tbody>
          {items.map((it) => (
            <RequirementRow
              key={it.requirementId}
              item={it}
              versionName={versionName(it.versionId)}
              onOpen={() => navigate(`/requirements/${it.requirementId}`)}
            />
          ))}
          {items.length === 0 && !list.isLoading && (
            <tr>
              <td colSpan={8} className="muted">
                无符合条件的需求
              </td>
            </tr>
          )}
        </tbody>
      </table>
      {list.data && (
        <Pagination
          page={Number(list.data.page)}
          pageSize={Number(list.data.pageSize)}
          total={Number(list.data.total)}
          onPageChange={setPage}
          onPageSizeChange={(n) => {
            setPageSize(n);
            setPage(1);
          }}
        />
      )}
      <CreateRequirementDialog
        open={createOpen}
        versions={versions.data?.items ?? []}
        onClose={() => {
          setCreateOpen(false);
          setSubmitError(null);
        }}
        onSubmit={(req) => createMutation.mutate(req)}
        submitting={createMutation.isPending}
        submitError={submitError}
      />
    </section>
  );
}

function RequirementRow({
  item,
  versionName,
  onOpen,
}: {
  item: RequirementItem;
  versionName: string;
  onOpen: () => void;
}) {
  return (
    <tr className="req-row" onClick={onOpen}>
      <td>
        {item.title}
        <span className="muted req-id"> {item.requirementId}</span>
      </td>
      <td>
        <span className="badge badge-category">{item.category}</span>
      </td>
      <td>
        {item.flowDefsId === "" ? (
          <span className="muted">—</span>
        ) : (
          <>
            <span className="badge badge-flow">
              {item.flowDefsId}·{FLOW_LABELS[item.flowDefsId]}
            </span>{" "}
            <ScopeBadge flowDefsId={item.flowDefsId} />
          </>
        )}
      </td>
      <td>
        <PriorityBadge priority={item.priority} />
      </td>
      <td>
        <StatusBadge status={item.status} />
      </td>
      <td>{versionName}</td>
      <td>{item.owner === "" ? <span className="muted">未指派</span> : item.owner}</td>
      <td className="muted">{formatTime(item.updatedAt)}</td>
    </tr>
  );
}

export function formatTime(unixMs: string): string {
  const n = Number(unixMs);
  if (!Number.isFinite(n) || n === 0) return "—";
  return new Date(n).toLocaleString("zh-CN", { hour12: false });
}
