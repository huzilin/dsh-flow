import type { ReactNode } from "react";
import {
  ISSUE_STATE_LABELS,
  PHASE_LABELS,
  REVIEW_STATE_LABELS,
  SCOPE_BADGE_BY_FLOW,
} from "../api/constants";

/** 徽标基底：色系 class + 文本。 */
export function Badge({ kind, children }: { kind: string; children: ReactNode }) {
  return <span className={`badge badge-${kind}`}>{children}</span>;
}

/** 优先级徽标（P0 红 / P1 橙 / P2 蓝 / P3 灰——01-module-board PRD §5.2）。 */
export function PriorityBadge({ priority }: { priority: string }) {
  if (priority === "") return <span className="muted">—</span>;
  return <Badge kind={`p-${priority.toLowerCase()}`}>{priority}</Badge>;
}

/** 需求/版本状态徽标（色圆点语义由底色承载，文本双通道可读）。 */
export function StatusBadge({ status }: { status: string }) {
  const kind: Record<string, string> = {
    待启动: "st-pending",
    进行中: "st-running",
    已完成: "st-done",
    已取消: "st-canceled",
    已发布: "st-published",
    已废弃: "st-deprecated",
  };
  return <Badge kind={kind[status] ?? "st-pending"}>{status}</Badge>;
}

/** scope 徽标（full / rd / fe 三色）。flowDefsId=按流程类型示意（T14 需求卡）；
 *  scopeOverride=快照 flows.scope 实值直接呈现（T15 流程树/详情头）。 */
export function ScopeBadge({
  flowDefsId,
  scopeOverride,
}: {
  flowDefsId?: string;
  scopeOverride?: string;
}) {
  const scope = scopeOverride ?? (flowDefsId ? SCOPE_BADGE_BY_FLOW[flowDefsId] : undefined);
  if (!scope) return null;
  return (
    <span>
      {scope.split("|").map((s) => (
        <Badge key={s} kind={`scope-${s}`}>
          {s}
        </Badge>
      ))}
    </span>
  );
}

/** 节点相位徽标（node.state 4 相位 + ""=未进入编排；CONTEXT.md「相位」词汇）。 */
export function PhaseBadge({ phase }: { phase: string }) {
  if (phase === "") return <span className="muted">{PHASE_LABELS[""]}</span>;
  return <Badge kind={`phase phase-${phase}`}>{PHASE_LABELS[phase] ?? phase}</Badge>;
}

/** 评审表态药丸（reviewing/agree/unagree——双通道：符号+文字）。 */
export function ReviewStatePill({ state }: { state: string }) {
  const symbols: Record<string, string> = { reviewing: "○", agree: "✅", unagree: "⛔" };
  const label = REVIEW_STATE_LABELS[state] ?? state;
  return (
    <Badge kind={`rv-${state}`}>
      {symbols[state] ?? "○"} {label}
    </Badge>
  );
}

/** issue 状态机 v2 状态徽标（open/pending-confirm/to-fix/fixed/resolved/overridden）。 */
export function IssueStateBadge({ state }: { state: string }) {
  const kinds: Record<string, string> = {
    open: "is-open",
    "pending-confirm": "is-pending",
    "to-fix": "is-tofix",
    fixed: "is-fixed",
    resolved: "is-resolved",
    overridden: "is-overridden",
  };
  return (
    <Badge kind={`issue ${kinds[state] ?? "is-open"}`}>{ISSUE_STATE_LABELS[state] ?? state}</Badge>
  );
}
