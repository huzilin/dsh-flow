import { Badge } from "./Badges";
import { SCOPE_BADGE_BY_FLOW } from "../api/constants";

/**
 * 需求行级 scope 徽标（原型 P8 v3 增补：实值三色 full 紫 / rd 蓝 / fe 绿；
 * plan 类无 scope 显「—」——ui-prototype-v3.html P8 hint、ui-interaction-final §6 N4）。
 * 口径：列表出参 RequirementItem 无 scope 实值字段（快照 flows.scope 仅详情侧有），
 * 行级沿用 SCOPE_BADGE_BY_FLOW 按 flow_defs_id 派生（constants.ts 与 seed scope_options 同源）；
 * 三色呈现与详情快照 ScopeBadge(scopeOverride) 同一色系 class（badge-scope-*）。
 */
export function ReqScopeBadge({ flowDefsId }: { flowDefsId: string }) {
  const scope = SCOPE_BADGE_BY_FLOW[flowDefsId] ?? "";
  if (scope === "") return <span className="muted">—</span>;
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

/**
 * 依赖角标（原型 P8：「依赖 ×N」点开/hover 显示前置需求 id 清单——flows.dependencies）。
 * 数据源 RequirementItem.dependencies（列表出参直带，spec §1.2 注记-D4，无需额外请求）；
 * N=0 不渲染（原型无依赖行无角标）。
 */
export function DepBadge({ dependencies }: { dependencies: string[] }) {
  if (dependencies.length === 0) return null;
  return (
    <span
      className="badge badge-dep"
      title={`前置依赖：${dependencies.join("、")}`}
    >
      依赖 ×{dependencies.length}
    </span>
  );
}
