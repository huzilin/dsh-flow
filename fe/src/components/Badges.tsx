import type { ReactNode } from "react";
import { SCOPE_BADGE_BY_FLOW } from "../api/constants";

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

/** scope 徽标（full / rd / fe 三色；T14 按流程类型示意，实值 T4 实例化决定）。 */
export function ScopeBadge({ flowDefsId }: { flowDefsId: string }) {
  const scope = SCOPE_BADGE_BY_FLOW[flowDefsId];
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
