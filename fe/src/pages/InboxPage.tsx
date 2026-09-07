import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import type { InboxItem, TodoItem } from "../api/proto/api";
import {
  NOTIFICATION_KIND_ICONS,
  NOTIFICATION_KIND_LABELS,
  NOTIFICATION_KINDS,
  ROLES,
  TODO_STATUS_LABELS,
} from "../api/constants";
import { INBOX_POLL_INTERVAL_MS, useInboxQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { Pagination } from "../components/Pagination";
import { formatTime } from "./RequirementPoolPage";

/**
 * 收件箱页（模块 III/IV，T16）：七类通知列表 + TODO 投影 + 全局冲突入口。
 * 数据源=GET /api/inbox 双通道（通知=issues 行 issue_type=notification 投影、
 * TODO=tasks 在途任务同源投影——completed/rejected/closed 不残留，FE 只呈现）；
 * MVP 轮询口径 30s（VITE_INBOX_POLL_MS 可配），TanStack Query refetchInterval 失效。
 * 跳转定位：node→详情对应节点高亮 / task→详情任务区 / issue→详情意见区锚点。
 */

/** 通知条目跳转链接（target_type 由后端按 kind 确定性映射；session=会话通道未开放不跳）。 */
export function inboxItemLink(item: InboxItem): string | null {
  if (item.requirementId === "") return null;
  const params = new URLSearchParams();
  if (item.nodeId !== "") params.set("node", item.nodeId);
  switch (item.targetType) {
    case "task":
      if (item.targetId !== "") {
        params.set("focus", "task");
        params.set("task", item.targetId);
      }
      break;
    case "issue":
      if (item.targetId !== "") {
        params.set("focus", "issue");
        params.set("issue", item.targetId);
      }
      break;
    case "node":
      break;
    default:
      // session_message：DSH 嵌入期开放，MVP 不跳转
      return null;
  }
  const qs = params.toString();
  return `/requirements/${encodeURIComponent(item.requirementId)}${qs === "" ? "" : `?${qs}`}`;
}

/** TODO 条目跳转链接（node_id 空=仅定位需求页）。 */
export function todoItemLink(item: TodoItem): string | null {
  if (item.requirementId === "") return null;
  const params = new URLSearchParams();
  if (item.nodeId !== "") params.set("node", item.nodeId);
  const qs = params.toString();
  return `/requirements/${encodeURIComponent(item.requirementId)}${qs === "" ? "" : `?${qs}`}`;
}

const READ_FILTER_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "", label: "全部" },
  { value: "unread", label: "未读" },
  { value: "read", label: "已读" },
];

export function InboxPage() {
  const [role, setRole] = useState("");
  const [readFilter, setReadFilter] = useState("");
  const [kind, setKind] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const inbox = useInboxQuery({ role, readFilter, kind, page, pageSize }, INBOX_POLL_INTERVAL_MS);
  // 全局冲突入口：escalation 类未读通知聚合（escalate 置位即通知，载体=issues 行同源）
  const conflicts = useInboxQuery(
    { kind: "escalation", readFilter: "unread", pageSize: 50 },
    INBOX_POLL_INTERVAL_MS,
  );

  const notifications = useMemo(() => inbox.data?.notifications ?? [], [inbox.data]);
  const todos = useMemo(() => inbox.data?.todos ?? [], [inbox.data]);
  const conflictItems = useMemo(() => conflicts.data?.notifications ?? [], [conflicts.data]);
  const total = Number(inbox.data?.total ?? 0);

  return (
    <section aria-label="收件箱">
      <div className="page-head">
        <h2>收件箱 / TODO</h2>
        <span className="muted">轮询 {Math.round(INBOX_POLL_INTERVAL_MS / 1000)}s · 未读优先</span>
      </div>

      <ConflictEntry items={conflictItems} />

      <div className="filter-bar">
        <label>
          类别
          <select
            aria-label="通知类别筛选"
            value={kind}
            onChange={(e) => {
              setKind(e.target.value);
              setPage(1);
            }}
          >
            <option value="">全部类别</option>
            {NOTIFICATION_KINDS.map((k) => (
              <option key={k} value={k}>
                {NOTIFICATION_KIND_LABELS[k]}
              </option>
            ))}
          </select>
        </label>
        <label>
          收件角色
          <select
            aria-label="收件角色筛选"
            value={role}
            onChange={(e) => {
              setRole(e.target.value);
              setPage(1);
            }}
          >
            <option value="">全部角色</option>
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </label>
        <label>
          已读状态
          <select
            aria-label="已读状态筛选"
            value={readFilter}
            onChange={(e) => {
              setReadFilter(e.target.value);
              setPage(1);
            }}
          >
            {READ_FILTER_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      {!inbox.isLoading && (
        <AlertBanner error={inbox.error ?? conflicts.error} onRetry={() => void inbox.refetch()} />
      )}

      <ul className="inbox-list" data-testid="inbox-notifications">
        {notifications.length === 0 && !inbox.isLoading && (
          <li className="muted inbox-empty">暂无通知。</li>
        )}
        {notifications.map((n) => {
          const href = inboxItemLink(n);
          return (
            <li key={n.issueId} className={`inbox-row ${n.state === "open" ? "is-unread" : ""}`}>
              <span className="inbox-kind" title={n.kind}>
                {NOTIFICATION_KIND_ICONS[n.kind] ?? "✉"} {NOTIFICATION_KIND_LABELS[n.kind] ?? n.kind}
              </span>
              {n.state === "open" && <span className="badge badge-unread">未读</span>}
              <span className="inbox-text">{n.text}</span>
              <span className="muted inbox-meta">
                {n.requirementId || "—"}
                {n.nodeId !== "" && ` · ${n.nodeId}`} · {n.sender || "plugin"}（{n.senderRole}）→{" "}
                {n.recipientRole} · {formatTime(n.createdAt)}
              </span>
              <span className="inbox-jump">
                {href ? (
                  <Link className="button-link" to={href}>
                    查看 →
                  </Link>
                ) : (
                  <span className="muted">会话通道（DSH 嵌入期开放）</span>
                )}
              </span>
            </li>
          );
        })}
      </ul>

      <Pagination
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={setPage}
        onPageSizeChange={(s) => {
          setPageSize(s);
          setPage(1);
        }}
      />

      <TodoSection items={todos} />
    </section>
  );
}

/** 全局冲突入口：聚合所有 escalation 未读通知（冲突卡聚合呈现于对应节点详情意见区顶部）。 */
function ConflictEntry({ items }: { items: InboxItem[] }) {
  if (items.length === 0) return null;
  return (
    <div className="conflict-entry" aria-label="全局冲突入口">
      <h3>⚔ 冲突待裁决（{items.length}）</h3>
      <ul>
        {items.map((c) => {
          const href = inboxItemLink(c);
          return (
            <li key={c.issueId}>
              {href ? (
                <Link to={href}>
                  {c.requirementId}
                  {c.nodeId !== "" && ` · ${c.nodeId}`} · {c.text}
                </Link>
              ) : (
                <span>
                  {c.requirementId} · {c.text}
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

/** TODO 投影区（模块 IV）：tasks 在途任务同源投影，completed/rejected 不残留（后端同源，FE 只呈现）。 */
function TodoSection({ items }: { items: TodoItem[] }) {
  return (
    <section className="todo-section" aria-label="TODO 投影">
      <h3>TODO（在途任务 {items.length}）</h3>
      {items.length === 0 ? (
        <p className="muted">无在途任务。</p>
      ) : (
        <table className="req-table todo-table">
          <thead>
            <tr>
              <th>任务</th>
              <th>需求 / 节点</th>
              <th>任务族</th>
              <th>状态</th>
              <th>目标</th>
              <th>创建</th>
              <th>跳转</th>
            </tr>
          </thead>
          <tbody>
            {items.map((t) => {
              const href = todoItemLink(t);
              return (
                <tr key={t.taskId}>
                  <td className="muted">{t.taskId}</td>
                  <td>
                    {t.requirementId || "—"}
                    {t.nodeId !== "" && ` / ${t.nodeId}`}
                  </td>
                  <td>
                    <span className="badge badge-artifact-type">{t.taskType}</span>
                  </td>
                  <td>
                    <span className={`badge badge-todo badge-todo-${t.status}`}>
                      {TODO_STATUS_LABELS[t.status] ?? t.status}
                    </span>
                  </td>
                  <td className="todo-goal">{t.goal}</td>
                  <td className="muted">{formatTime(t.createdAt)}</td>
                  <td>
                    {href && (
                      <Link className="button-link" to={href}>
                        节点详情 →
                      </Link>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </section>
  );
}
