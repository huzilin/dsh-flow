import { useState } from "react";
import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { REQUIREMENT_STATUSES } from "../api/constants";
import type { RequirementItem } from "../api/proto/api";
import { useRequirementsQuery, useVersionsQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { PriorityBadge } from "../components/Badges";
import { DepBadge, ReqScopeBadge } from "../components/ReqMetaBadges";
import {
  FilterBar,
  useFilterState,
  VERSION_FILTER_ALL,
  VERSION_FILTER_BACKLOG,
} from "../components/FilterBar";

type BoardView = "board" | "list";

/** 分组表头行色系（原型 P8 列表视图：进行中琥珀/待启动灰/已完成绿/已取消红）。 */
const GROUP_HEAD_CLASS: Record<string, string> = {
  待启动: "board-group-head-pending",
  进行中: "board-group-head-running",
  已完成: "board-group-head-done",
  已取消: "board-group-head-canceled",
};

/**
 * 卡片 ✎ 编辑占位（编辑弹窗 RequirementModal 本体由详情域改造提供，ui-interaction-final §6 N4；
 * 本页只留卡角入口：stopPropagation 阻断卡片跳详情，点击行为待弹窗接入后接 onEdit）。
 */
function CardEditPlaceholder({ requirementId }: { requirementId: string }) {
  return (
    <span className="card-edit-slot">
      <button
        type="button"
        className="card-edit"
        title="编辑（编辑弹窗接入中）"
        aria-label={`编辑 ${requirementId}`}
        onClick={(e) => {
          // TODO(编辑弹窗)：接入 RequirementModal 后由此回调打开；当前仅占位不动作
          e.stopPropagation();
        }}
      >
        ✎
      </button>
    </span>
  );
}

/**
 * 看板（原型 P8）：「看板|列表」双视图 + 状态四泳道（待启动/进行中/已完成/已取消——PRD §5.2）。
 * 泳道卡片 v3 增补：scope 徽标三色实值、依赖角标（dependencies.length>0 显「依赖 ×N」，
 * hover title 列前置 id 清单）、卡角 ✎ 编辑占位。数据源同需求池接口；后端强制分页，单页上限 100。
 */
export function BoardPage({
  renderCardEdit,
}: {
  /** 编辑入口插槽：默认渲染占位 ✎（另一 subagent 完成编辑弹窗后经路由 props 注入） */
  renderCardEdit?: (item: RequirementItem) => ReactNode;
}) {
  const navigate = useNavigate();
  const [view, setView] = useState<BoardView>("board");
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
  const versionName = (id: string) =>
    versions.data?.items.find((v) => v.versionId === id)?.name ?? id;

  const renderCardEditFn = renderCardEdit ?? ((it: RequirementItem) => <CardEditPlaceholder requirementId={it.requirementId} />);

  return (
    <section>
      <div className="page-head">
        <h2>看板</h2>
        {truncated && (
          <span className="muted">
            需求池共 {total} 条，仅呈现前 100 条（后端强制分页上限）
          </span>
        )}
        {/* 双视图切换（原型 P8 detailtop：sbtn 风格「看板|列表」） */}
        <div className="view-switch" role="group" aria-label="视图切换">
          <button
            type="button"
            className={`sbtn${view === "board" ? " on" : ""}`}
            aria-pressed={view === "board"}
            onClick={() => setView("board")}
          >
            看板
          </button>
          <button
            type="button"
            className={`sbtn${view === "list" ? " on" : ""}`}
            aria-pressed={view === "list"}
            onClick={() => setView("list")}
          >
            列表
          </button>
        </div>
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
      {view === "board" ? (
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
                    <div className="card-title">
                      {it.title}
                      {renderCardEditFn(it)}
                    </div>
                    <div className="card-meta">
                      <span className="badge badge-category">{it.category}</span>
                      <PriorityBadge priority={it.priority} />
                      <ReqScopeBadge flowDefsId={it.flowDefsId} />
                      <DepBadge dependencies={it.dependencies} />
                    </div>
                    <div className="card-foot muted">
                      <span>{it.versionId === "" ? "待分配" : versionName(it.versionId)}</span>
                      <span>{it.owner === "" ? "未指派" : it.owner}</span>
                    </div>
                  </div>
                ))}
                {lane.length === 0 && <div className="lane-empty muted">空</div>}
              </div>
            );
          })}
        </div>
      ) : (
        <table className="req-table board-list-table" aria-label="需求列表（按状态分组）">
          <thead>
            <tr>
              <th>标题</th>
              <th>类别</th>
              <th>scope</th>
              <th>优先级</th>
              <th>版本</th>
              <th>负责人</th>
              <th>依赖</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {/* 按状态分组行（原型 P8：● 状态 · n 项 分组头 colspan 合并；空状态组整组隐藏） */}
            {REQUIREMENT_STATUSES.map((status) => {
              const group = items.filter((it) => it.status === status);
              if (group.length === 0) return null;
              return (
                <GroupRows
                  key={status}
                  status={status}
                  group={group}
                  versionName={versionName}
                  onOpen={(id) => navigate(`/requirements/${id}`)}
                />
              );
            })}
            {items.length === 0 && !list.isLoading && (
              <tr>
                <td colSpan={8} className="muted">
                  无符合条件的需求
                </td>
              </tr>
            )}
          </tbody>
        </table>
      )}
    </section>
  );
}

function GroupRows({
  status,
  group,
  versionName,
  onOpen,
}: {
  status: string;
  group: RequirementItem[];
  versionName: (id: string) => string;
  onOpen: (id: string) => void;
}) {
  return (
    <>
      <tr className={`board-group-head ${GROUP_HEAD_CLASS[status] ?? ""}`}>
        <td colSpan={8}>
          ● {status} · {group.length} 项
        </td>
      </tr>
      {group.map((it) => (
        <tr
          key={it.requirementId}
          className="req-row"
          onClick={() => onOpen(it.requirementId)}
        >
          <td>
            {it.title}
            <span className="muted req-id"> {it.requirementId}</span>
          </td>
          <td>
            <span className="badge badge-category">{it.category}</span>
          </td>
          <td>
            <ReqScopeBadge flowDefsId={it.flowDefsId} />
          </td>
          <td>
            <PriorityBadge priority={it.priority} />
          </td>
          <td>{it.versionId === "" ? "待分配" : versionName(it.versionId)}</td>
          <td>{it.owner === "" ? <span className="muted">未指派</span> : it.owner}</td>
          <td>
            {it.dependencies.length === 0 ? (
              <span className="muted">—</span>
            ) : (
              <DepBadge dependencies={it.dependencies} />
            )}
          </td>
          <td>{status}</td>
        </tr>
      ))}
    </>
  );
}
