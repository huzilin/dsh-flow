import { useNavigate } from "react-router-dom";
import { REQUIREMENT_STATUSES } from "../api/constants";
import { useRequirementsQuery, useVersionsQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { PriorityBadge, ScopeBadge } from "../components/Badges";
import {
  FilterBar,
  useFilterState,
  VERSION_FILTER_ALL,
  VERSION_FILTER_BACKLOG,
} from "../components/FilterBar";

/**
 * 看板：状态四泳道（待启动/进行中/已完成/已取消——PRD §5.2 版本看板），
 * 数据源同需求池接口；后端强制分页，取单页上限 100 呈现。
 */
export function BoardPage() {
  const navigate = useNavigate();
  const filters = useFilterState();
  const versions = useVersionsQuery();
  const list = useRequirementsQuery({
    versionId:
      filters.versionSel === VERSION_FILTER_ALL || filters.versionSel === VERSION_FILTER_BACKLOG
        ? undefined
        : filters.versionSel,
    backlog: filters.versionSel === VERSION_FILTER_BACKLOG,
    priority: filters.priority,
    page: 1,
    pageSize: 100,
  });

  const items = list.data?.items ?? [];
  const total = list.data ? Number(list.data.total) : 0;
  const truncated = total > items.length;

  return (
    <section>
      <div className="page-head">
        <h2>看板</h2>
        {truncated && (
          <span className="muted">
            需求池共 {total} 条，泳道仅呈现前 100 条（后端强制分页上限）
          </span>
        )}
      </div>
      <AlertBanner error={list.error} onRetry={() => void list.refetch()} />
      <FilterBar
        versions={versions.data?.items ?? []}
        versionSel={filters.versionSel}
        onVersionSel={filters.setVersionSel}
        priority={filters.priority}
        onPriority={filters.setPriority}
        status=""
        onStatus={() => undefined}
        showStatus={false}
      />
      <div className="board">
        {REQUIREMENT_STATUSES.map((status) => {
          const lane = items.filter((it) => it.status === status);
          return (
            <div key={status} className="lane">
              <div className="lane-head">
                <span className={`lane-dot lane-dot-${status}`}>{status}</span>
                <span className="muted">{lane.length}</span>
              </div>
              {lane.map((it) => (
                <div
                  key={it.requirementId}
                  className="card"
                  onClick={() => navigate(`/requirements/${it.requirementId}`)}
                >
                  <div className="card-title">{it.title}</div>
                  <div className="card-meta">
                    <span className="badge badge-category">{it.category}</span>
                    <PriorityBadge priority={it.priority} />
                    {it.flowDefsId !== "" && <ScopeBadge flowDefsId={it.flowDefsId} />}
                  </div>
                  <div className="card-foot muted">
                    <span>{it.versionId === "" ? "待分配" : it.versionId}</span>
                    <span>{it.owner === "" ? "未指派" : it.owner}</span>
                  </div>
                </div>
              ))}
              {lane.length === 0 && <div className="lane-empty muted">空</div>}
            </div>
          );
        })}
      </div>
    </section>
  );
}
