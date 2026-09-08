import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import type { InboxItem } from "../api/proto/api";
import {
  NOTIFICATION_KIND_ICONS,
  NOTIFICATION_KIND_LABELS,
  NOTIFICATION_KINDS,
  ROLES,
} from "../api/constants";
import { INBOX_POLL_INTERVAL_MS, useInboxQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { Pagination } from "../components/Pagination";
import { formatTime } from "./RequirementPoolPage";

/**
 * 收件箱页（模块 III/IV，T16 + spec -D5）：七类通知列表 + TODO 五分类投影 + 全局冲突入口。
 * 数据源=GET /api/inbox 双通道（通知=issues 行 issue_type=notification 投影、
 * TODO=通知的操作投影——基线规则「同源同筛选」+ N3 定稿，非 tasks 投影，FE 只呈现）；
 * MVP 轮询口径 30s（VITE_INBOX_POLL_MS 可配），TanStack Query refetchInterval 失效。
 * 跳转定位：node→详情对应节点高亮 / task→详情任务区 / issue→详情意见区锚点
 * （retest 子型同走 issue 锚点）。
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
      // retest 子型（re_review 且 target_id 非空，spec -D5/N1：修复提交 to-fix→fixed
      // 待复核走重新表态通道）跳详情意见区锚点复验；澄清子型 target_id 空，仅按 node 定位
      if (item.kind === "re_review" && item.targetId !== "") {
        params.set("focus", "issue");
        params.set("issue", item.targetId);
      }
      break;
    default:
      // session_message：DSH 嵌入期开放，MVP 不跳转
      return null;
  }
  const qs = params.toString();
  return `/requirements/${encodeURIComponent(item.requirementId)}${qs === "" ? "" : `?${qs}`}`;
}

/**
 * TODO 五分类投影规则（spec -D5 / ui-interaction-final §6 N3 定稿）：
 * TODO = 收件箱通知的操作投影，同请求 notifications 同源推导、无新请求；
 * 待拍板←gate_pending；待澄清←clarify_task；待重表态←re_review 且 target_id 空（澄清子型）；
 * 待复验←re_review 且 target_id 非空（retest 子型）；冲突待裁决←escalation；
 * stall_followup/need_intervene/session_message 为提醒类，不产生 TODO。
 * 口径注记：投影基于当前筛选+分页返回的通知列表（后端无跨页聚合端点，计数随筛选联动）。
 */
const TODO_GROUPS: Array<{ key: string; label: string; match: (n: InboxItem) => boolean }> = [
  { key: "gate", label: "待拍板", match: (n) => n.kind === "gate_pending" },
  { key: "clarify", label: "待澄清", match: (n) => n.kind === "clarify_task" },
  { key: "re-state", label: "待重表态", match: (n) => n.kind === "re_review" && n.targetId === "" },
  { key: "retest", label: "待复验", match: (n) => n.kind === "re_review" && n.targetId !== "" },
  { key: "conflict", label: "冲突待裁决", match: (n) => n.kind === "escalation" },
];

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
  // 待拍板提醒条数据源：通知列表内未读 gate_pending（同源投影，原型 P10 lead 行「不抢占焦点」）
  const gateUnread = useMemo(
    () => notifications.filter((n) => n.kind === "gate_pending" && n.state === "open"),
    [notifications],
  );
  const conflictItems = useMemo(() => conflicts.data?.notifications ?? [], [conflicts.data]);
  const total = Number(inbox.data?.total ?? 0);

  return (
    <section aria-label="收件箱">
      <div className="page-head">
        <h2>收件箱 / TODO</h2>
        <span className="muted">轮询 {Math.round(INBOX_POLL_INTERVAL_MS / 1000)}s · 未读优先</span>
      </div>

      <GatePendingToast items={gateUnread} />

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
              {n.kind === "re_review" && n.targetId !== "" && (
                // retest 子型徽标（spec -D5：修复提交待复核，区分澄清子型）
                <span className="badge badge-artifact-type" title="修复提交待复核（retest）">
                  待复验
                </span>
              )}
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

      <TodoSection notifications={notifications} />
    </section>
  );
}

/**
 * 待拍板提醒条（原型 P10 lead 行：badge/toast 提示，不弹窗抢占焦点）：
 * 存在未读 gate_pending 时呈现，锚点滚动至 TODO 待拍板分组（todo-gate）。
 */
function GatePendingToast({ items }: { items: InboxItem[] }) {
  if (items.length === 0) return null;
  // 复用 conflict-entry 盒形，色系换琥珀（gate 语义，同 badge-todo-in-progress 色板）
  return (
    <div
      className="conflict-entry"
      style={{ borderColor: "#fedf89", background: "#fffaeb" }}
      aria-label="待拍板提醒"
    >
      📬 有 {items.length} 条『待你拍板』待处理{" "}
      <button
        className="button-link"
        onClick={() =>
          document.getElementById("todo-gate")?.scrollIntoView({ behavior: "smooth" })
        }
      >
        查看待拍板 →
      </button>
    </div>
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

/**
 * TODO 投影区（模块 IV）：通知的操作投影五分类（spec -D5，非 tasks 投影）。
 * 顶部 dock 条常驻（原型 P11：未完成数=五分类条数之和），组空不渲染小节。
 */
function TodoSection({ notifications }: { notifications: InboxItem[] }) {
  const groups = TODO_GROUPS.map((g) => ({ ...g, items: notifications.filter(g.match) }));
  const undone = groups.reduce((sum, g) => sum + g.items.length, 0);
  return (
    <section className="todo-section" aria-label="TODO 投影">
      <div className="todo-dock muted" aria-label="TODO 五分类统计">
        📋 未完成 {undone}
        {groups.map((g) => (
          <span key={g.key}>
            {" · "}
            {g.label} {g.items.length}
          </span>
        ))}
      </div>
      {undone === 0 ? (
        <p className="muted">暂无待办。</p>
      ) : (
        groups
          .filter((g) => g.items.length > 0)
          .map((g) => (
            <div key={g.key} id={`todo-${g.key}`} className="todo-group">
              <h3>
                {g.label}（{g.items.length}）
              </h3>
              <ul>
                {g.items.map((n) => {
                  const href = inboxItemLink(n);
                  return (
                    <li key={n.issueId}>
                      <span className="inbox-kind">{g.label}</span>
                      <span className="inbox-text">{n.text}</span>
                      <span className="muted inbox-meta">
                        {n.requirementId || "—"}
                        {n.nodeId !== "" && ` · ${n.nodeId}`} · {formatTime(n.createdAt)}
                      </span>
                      {href && (
                        <Link className="button-link" to={href}>
                          节点详情 →
                        </Link>
                      )}
                    </li>
                  );
                })}
              </ul>
            </div>
          ))
      )}
    </section>
  );
}
