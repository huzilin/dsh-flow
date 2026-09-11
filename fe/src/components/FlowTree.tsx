import { useMemo } from "react";
import type { FlowItem, FlowNodeItem } from "../api/proto/api";
import {
  buildFlowRows,
  deriveGateView,
  parseReviewResults,
  summarizeReviews,
  type FlowRow,
} from "../domain/flowview";
import { Badge, ScopeBadge } from "./Badges";

/**
 * 流程树（模块 II 主视图）：SVG 横向时序布局（一版交互蓝本 client.js:602-710「图即
 * 控制流」回归——inventory A2-2/3/4/5，B-18/19/20 履约）。节点按 node_order 左→右
 * 横排；RD∥FE 并行开发组上下双泳道（rd 上泳道蓝 / fe 下泳道绿 + 泳道标签），主链↔
 * 组之间折线分叉/汇聚（一版组级连线语义：一分叉多、多汇聚一；组间 1:1），组内只连
 * 同泳道行（跨分支不拉线）。并行组判定沿用 buildFlowRows 相邻 exec-CR 对（fe 快照
 * 无 parallel_group 字段，口径见 flowview 注）；rd-only/fe-only/单流程无并行对时
 * 退化为单泳道（全部节点同一中线）。
 * 保留现行功能：scope 渲染集合（被裁节点后端不返回，天然不渲染不占位）、节点点击
 * 进详情（键盘可达）、门禁结论点色（含 done=绿）、当前节点章、选中描边。
 */
const NW = 212;
const NH = 64;
const GAP = 44;
const PAD_X = 16;
/** 上泳道（rd）y；中线（单泳道）与下泳道（fe）按 LANE_GAP 递推。 */
const TOP_Y = 30;
const LANE_GAP = 40;
const BOT_PAD = 26;
const CENTER_Y = TOP_Y + NH + LANE_GAP;
const BOT_Y = CENTER_Y + NH + LANE_GAP;
const HEIGHT = BOT_Y + NH + BOT_PAD;

/** 放置结果：节点 + 矩形几何 + 泳道归属（null=主链单泳道）。 */
interface Placed {
  node: FlowNodeItem;
  x: number;
  y: number;
  w: number;
  h: number;
  lane: "rd" | "fe" | null;
}

/** 横向布局列：single=主链单节点列；parallel=并行组列（rd 上 / fe 下双泳道）。 */
interface FlowColumn {
  kind: "single" | "parallel";
  x: number;
  width: number;
  items: Placed[];
}

/** 列布局：node_order 序横向累加 x；并行组列宽=最长分支（当前对组恒单节点）。 */
function layoutColumns(rows: FlowRow[]): { cols: FlowColumn[]; width: number } {
  const cols: FlowColumn[] = [];
  let x = PAD_X;
  for (const row of rows) {
    if (row.kind === "single") {
      cols.push({
        kind: "single",
        x,
        width: NW,
        items: [{ node: row.node, x, y: CENTER_Y - NH / 2, w: NW, h: NH, lane: null }],
      });
    } else {
      const items: Placed[] = [
        { node: row.rd, x, y: TOP_Y, w: NW, h: NH, lane: "rd" },
        { node: row.fe, x, y: BOT_Y, w: NW, h: NH, lane: "fe" },
      ];
      cols.push({ kind: "parallel", x, width: NW, items });
    }
    x += NW + GAP;
  }
  return { cols, width: x - GAP + PAD_X };
}

/** 折线路径（一版组级连线同构）：同 y 水平直连；异 y 中点折线（H-V-H）。 */
function elbow(a: { x: number; y: number }, b: { x: number; y: number }): string {
  if (a.y === b.y) return `M${a.x},${a.y} H${b.x}`;
  const jx = Math.round((a.x + b.x) / 2);
  return `M${a.x},${a.y} H${jx} V${b.y} H${b.x}`;
}

const rightOf = (p: Placed) => ({ x: p.x + p.w, y: p.y + p.h / 2 });
const leftOf = (p: Placed) => ({ x: p.x, y: p.y + p.h / 2 });

/**
 * 列间连接（一版 client.js:653-689 同构）：列内同泳道相邻横连；列间按
 * 单→单直连 / 单→组分叉（一分叉多）/ 组→单汇聚（多汇聚一）/ 组→组 1:1；
 * 组内跨泳道（rd/fe）不画线（items 相邻 y 恒异，无连线产生）。
 */
function buildPaths(cols: FlowColumn[]): string[] {
  const paths: string[] = [];
  cols.forEach((col, gi) => {
    const items = col.items;
    for (let k = 0; k < items.length - 1; k++) {
      const a = items[k]!;
      const b = items[k + 1]!;
      if (a.y === b.y) paths.push(`M${a.x + a.w},${a.y + a.h / 2} H${b.x}`);
    }
    if (gi === 0) return;
    const prev = cols[gi - 1]!.items;
    if (prev.length === 1 && items.length === 1) {
      paths.push(elbow(rightOf(prev[0]!), leftOf(items[0]!)));
    } else if (prev.length === 1 && items.length > 1) {
      const from = rightOf(prev[0]!);
      for (const p of items) paths.push(elbow(from, leftOf(p)));
    } else if (prev.length > 1 && items.length === 1) {
      const to = leftOf(items[0]!);
      for (const p of prev) paths.push(elbow(rightOf(p), to));
    } else {
      items.forEach((p, k) => {
        const from = prev[k];
        if (from !== undefined) paths.push(elbow(rightOf(from), leftOf(p)));
      });
    }
  });
  return paths;
}

export function FlowTree({
  flow,
  nodes,
  currentNodeId,
  selectedNodeId,
  issuesByNode,
  onSelectNode,
}: {
  flow: FlowItem | undefined;
  nodes: FlowNodeItem[];
  currentNodeId: string;
  selectedNodeId: string | null;
  /** node_id → 未闭合 issue 数（节点卡 open 数） */
  issuesByNode: Map<string, number>;
  onSelectNode: (nodeId: string) => void;
}) {
  const rows = useMemo(() => buildFlowRows(nodes), [nodes]);
  const { cols, width } = useMemo(() => layoutColumns(rows), [rows]);
  const paths = useMemo(() => buildPaths(cols), [cols]);
  const hasParallel = rows.some((r) => r.kind === "parallel");
  if (rows.length === 0) {
    return <div className="flow-tree-empty muted">流程实例尚未生成（需求创建即实例化）。</div>;
  }
  return (
    <div className="flow-tree-wrap" id="flow-tree">
      <div className="flow-tree-head">
        <b>
          {flow?.flowDefsId ?? ""} 流程
          {flow && flow.scope !== "" && (
            <>
              {" · 当前渲染 scope："}
              <ScopeBadge scopeOverride={flow.scope} />
            </>
          )}
        </b>
        <span className="muted flow-tree-hint">
          {nodes.length} 节点 · 横向时序
          {hasParallel && " · 并行组上下双泳道（RD 蓝 / FE 绿）"} · 点击节点查看详情
        </span>
      </div>
      <div className="flow-tree-scroll">
        <svg
          className="flow-tree"
          role="list"
          aria-label="流程树"
          width={width}
          height={HEIGHT}
          viewBox={`0 0 ${width} ${HEIGHT}`}
        >
          {paths.map((d, i) => (
            <path key={`p-${i}`} d={d} className="flow-link" />
          ))}
          {cols.map((col) =>
            col.kind === "single" ? (
              <FlowNode
                key={col.items[0]!.node.nodeId}
                p={col.items[0]!}
                currentNodeId={currentNodeId}
                selectedNodeId={selectedNodeId}
                issuesByNode={issuesByNode}
                onSelectNode={onSelectNode}
              />
            ) : (
              <g
                key={`pg-${col.items[0]!.node.nodeId}`}
                role="group"
                aria-label={`并行组 ${col.items[0]!.node.nodeKey} ∥ ${col.items[1]!.node.nodeKey}`}
              >
                {col.items.map((p) => (
                  <FlowNode
                    key={p.node.nodeId}
                    p={p}
                    currentNodeId={currentNodeId}
                    selectedNodeId={selectedNodeId}
                    issuesByNode={issuesByNode}
                    onSelectNode={onSelectNode}
                  />
                ))}
              </g>
            ),
          )}
        </svg>
      </div>
    </div>
  );
}

/**
 * 横向节点卡（一版 client.js:690-705 同构 + fe 现行信息密度）：泳道标签（RD ⇄ 后端 /
 * FE ⇄ 前端，一版同文案）+ 状态点色（门禁结论优先）+ 节点序号名 + 摘要行 + 回滚角标
 * ↣n（rolled_back_count>1，一版为 ↩N 阈值 >0，此处按本单口径 >1 才显）。
 */
function FlowNode({
  p,
  currentNodeId,
  selectedNodeId,
  issuesByNode,
  onSelectNode,
}: {
  p: Placed;
  currentNodeId: string;
  selectedNodeId: string | null;
  issuesByNode: Map<string, number>;
  onSelectNode: (nodeId: string) => void;
}) {
  const { node, x, y, w, h, lane } = p;
  const isCurrent = node.nodeId === currentNodeId;
  const isSelected = node.nodeId === selectedNodeId;
  // 点色语义（2026-09-10 User 反馈：已通过与待拍板不可同色）——门禁结论优先，
  // 相位色兜底：绿=已通过 / 琥珀=待拍板 / 红=已打回；无结论时按相位着色。
  const gateView = deriveGateView(node);
  const dotVariant = gateView ? `gate-${gateView}` : `phase-${node.state || "none"}`;
  const rolledBack = Number(node.rolledBackCount);
  const laneLabel = lane === "rd" ? "RD ⇄ 后端" : lane === "fe" ? "FE ⇄ 前端" : null;
  return (
    <g
      role="listitem"
      tabIndex={0}
      aria-label={`节点 ${node.nodeOrder} ${node.nodeKey}`}
      className="flow-node"
      data-node-id={node.nodeId}
      data-current={isCurrent || undefined}
      onClick={() => onSelectNode(node.nodeId)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onSelectNode(node.nodeId);
        }
      }}
    >
      {laneLabel !== null && (
        <text x={x + w / 2} y={y - 8} className={`flow-lane-label lane-${lane}`}>
          {laneLabel}
        </text>
      )}
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={8}
        className={[
          "flow-node-card",
          lane !== null ? `lane-${lane}` : "",
          isCurrent ? "is-current" : "",
          isSelected ? "is-selected" : "",
          node.state === "" ? "is-idle" : "",
        ].join(" ")}
      />
      <circle cx={x + 16} cy={y + 18} r={5} className={`phase-dot ${dotVariant}`} />
      <text x={x + 28} y={y + 22} className="flow-node-title">
        {node.nodeOrder}. {node.nodeKey}
      </text>
      {rolledBack > 1 && (
        <text x={x + w - 8} y={y + 16} textAnchor="end" className="flow-rollback">
          ↣{rolledBack}
        </text>
      )}
      <text x={x + 16} y={y + 40} className="flow-node-sub">
        {nodeSubLine(node, issuesByNode.get(node.nodeId) ?? 0)}
      </text>
      <NodeMetaText node={node} x={x + 16} y={y + 56} />
      {isCurrent && (
        <>
          <rect x={x + w - 64} y={y + h - 24} width={56} height={16} rx={4} className="flow-current-chip" />
          <text x={x + w - 36} y={y + h - 12} className="flow-current-text">
            当前
          </text>
        </>
      )}
    </g>
  );
}

/** 节点摘要行：评审轮计数 / 门禁裁决 / 未闭合 issue 数（回滚角标已上移 ↣n）。 */
function nodeSubLine(node: FlowNodeItem, openIssues: number): string {
  const parts: string[] = [];
  const counts = summarizeReviews(node);
  if (counts.seats > 0) {
    parts.push(
      `表态 ${counts.agree + counts.unagree + counts.reviewing}/${counts.seats} · 同意 ${counts.agree} · 反对 ${counts.unagree} · 待表态 ${counts.reviewing}`,
    );
  } else {
    parts.push("无评审位");
  }
  if (openIssues > 0) parts.push(`open ${openIssues}`);
  return parts.join(" · ");
}

/** 底行徽标组（SVG 内联文本；gate/user_gate/门禁结论/待拍板/相位收至此行）。 */
function NodeMetaText({ node, x, y }: { node: FlowNodeItem; x: number; y: number }) {
  const chips: string[] = [];
  if (node.state !== "") chips.push(node.state === "done" ? "已完成" : node.state);
  if (node.nodeType !== "") chips.push(node.nodeType);
  if (node.gate === "1") chips.push("gate");
  if (node.userGate === "1") chips.push("user_gate");
  const gateView = deriveGateView(node);
  if (gateView === "agreed") chips.push("待拍板");
  if (gateView === "approved") chips.push("已通过");
  if (gateView === "rejected") chips.push("已打回");
  return (
    <text x={x} y={y} className="flow-node-chips">
      {chips.join(" / ")}
    </text>
  );
}

/** 详情头 scope 徽标：快照 flows.scope 实值直接呈现。 */
export function FlowScopeBadge({ flow }: { flow: FlowItem | undefined }) {
  if (!flow || flow.scope === "") return null;
  return <Badge kind={`scope-${flow.scope}`}>{flow.scope}</Badge>;
}

/** 评审轮表态明细（per-agent 药丸数据行）。 */
export function reviewDetailsOf(node: FlowNodeItem) {
  return parseReviewResults(node.reviewResults);
}
