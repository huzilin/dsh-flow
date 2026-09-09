import { useMemo, type ReactNode } from "react";
import type { FlowItem, FlowNodeItem } from "../api/proto/api";
import { buildFlowRows, deriveGateView, parseReviewResults, summarizeReviews } from "../domain/flowview";
import { Badge, ScopeBadge } from "./Badges";

/**
 * 流程树（模块 II 主视图）：SVG 一次渲染全实例节点（seed 上限 40；被裁节点已由后端
 * scope 过滤不返回，故天然不渲染不占位）。布局按 buildFlowRows 行结构：常规节点纵向
 * 单链；RD∥FE 并行开发组（后端 wb_advance join 同口径：相邻泳道侧 exec-CR）一行
 * 两列泳道并排 + join 连接线（原型 P1 并行泳道形态）。渲染矩阵一致性由后端 T4
 * 实例化测试兜底（AC：三种 scope 节点集/序号一致）。
 */
const ROW_W = 460;
const CARD_H = 54;
const GAP = 12;

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
  /** flows.current_node_id：当前节点高亮 */
  currentNodeId: string;
  selectedNodeId: string | null;
  /** node_id → 未闭合 issue 数（节点卡 open 数） */
  issuesByNode: Map<string, number>;
  onSelectNode: (nodeId: string) => void;
}) {
  const rows = useMemo(() => buildFlowRows(nodes), [nodes]);
  if (rows.length === 0) {
    return <div className="flow-tree-empty muted">流程实例尚未生成（需求创建即实例化）。</div>;
  }
  const laneW = (ROW_W - GAP * 3) / 2;
  const height = rows.length * (CARD_H + GAP) + GAP * 2;
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
          {nodes.length} 节点 · 被裁节点不渲染不占位（node_order 已重排）· 点击节点查看详情
        </span>
      </div>
      <svg
        className="flow-tree"
        role="list"
        aria-label="流程树"
        width={ROW_W}
        height={height}
        viewBox={`0 0 ${ROW_W} ${height}`}
      >
        {rows.map((row, ri) => {
          const top = GAP + ri * (CARD_H + GAP);
          const linkIn =
            ri > 0 ? (
              <path
                d={`M${ROW_W / 2},${top - GAP} v${GAP}`}
                className="flow-link"
              />
            ) : null;
          if (row.kind === "single") {
            return (
              <NodeCard
                key={row.node.nodeId}
                node={row.node}
                x={GAP}
                width={ROW_W - GAP * 2}
                top={top}
                linkIn={linkIn}
                currentNodeId={currentNodeId}
                selectedNodeId={selectedNodeId}
                issuesByNode={issuesByNode}
                onSelectNode={onSelectNode}
              />
            );
          }
          const joinY = top + CARD_H / 2;
          return (
            <g key={`pg-${row.rd.nodeId}-${row.fe.nodeId}`} role="listitem" aria-label={`并行组 ${row.rd.nodeKey} ∥ ${row.fe.nodeKey}`}>
              {linkIn}
              <NodeCard
                node={row.rd}
                x={GAP}
                width={laneW}
                top={top}
                linkIn={null}
                currentNodeId={currentNodeId}
                selectedNodeId={selectedNodeId}
                issuesByNode={issuesByNode}
                onSelectNode={onSelectNode}
              />
              <NodeCard
                node={row.fe}
                x={GAP * 2 + laneW}
                width={laneW}
                top={top}
                linkIn={null}
                currentNodeId={currentNodeId}
                selectedNodeId={selectedNodeId}
                issuesByNode={issuesByNode}
                onSelectNode={onSelectNode}
              />
              {/* join 连接线：两泳道完成在本组汇合（注记-C join 三层的视觉面） */}
              <path
                d={`M${GAP + laneW},${joinY} H${GAP * 2 + laneW}`}
                className="flow-link approved"
              />
            </g>
          );
        })}
      </svg>
    </div>
  );
}

/** 单节点卡（并行泳道卡同构，宽度参数化）。 */
function NodeCard({
  node,
  x,
  width,
  top,
  linkIn,
  currentNodeId,
  selectedNodeId,
  issuesByNode,
  onSelectNode,
}: {
  node: FlowNodeItem;
  x: number;
  width: number;
  top: number;
  linkIn: ReactNode;
  currentNodeId: string;
  selectedNodeId: string | null;
  issuesByNode: Map<string, number>;
  onSelectNode: (nodeId: string) => void;
}) {
  const isCurrent = node.nodeId === currentNodeId;
  const isSelected = node.nodeId === selectedNodeId;
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
      {linkIn}
      <rect
        x={x}
        y={top}
        width={width}
        height={CARD_H}
        rx={8}
        className={[
          "flow-node-card",
          isCurrent ? "is-current" : "",
          isSelected ? "is-selected" : "",
          node.state === "" ? "is-idle" : "",
        ].join(" ")}
      />
      <circle
        cx={x + 18}
        cy={top + CARD_H / 2}
        r={5}
        className={`phase-dot phase-${node.state || "none"}`}
      />
      <text x={x + 34} y={top + 22} className="flow-node-title">
        {node.nodeOrder}. {node.nodeKey}
      </text>
      <NodeMetaText node={node} x={x + width - 8} y={top + 22} />
      <text x={x + 34} y={top + 42} className="flow-node-sub">
        {nodeSubLine(node, issuesByNode.get(node.nodeId) ?? 0)}
      </text>
      {isCurrent && (
        <>
          <rect
            x={x + width - 64}
            y={top + 32}
            width={56}
            height={16}
            rx={4}
            className="flow-current-chip"
          />
          <text x={x + width - 36} y={top + 44} className="flow-current-text">
            当前
          </text>
        </>
      )}
    </g>
  );
}

/** 节点第二行摘要：评审轮计数 / 门禁裁决 / 未闭合 issue 数。 */
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
  if (node.rolledBackCount !== "0") parts.push(`↩${node.rolledBackCount}`);
  return parts.join(" · ");
}

/** 右上角徽标组（SVG 内联文本，避免 foreignObject 兼容负担）。 */
function NodeMetaText({ node, x, y }: { node: FlowNodeItem; x: number; y: number }) {
  const chips: string[] = [];
  if (node.state !== "") chips.push(node.state);
  if (node.nodeType !== "") chips.push(node.nodeType);
  if (node.gate === "1") chips.push("gate");
  if (node.userGate === "1") chips.push("user_gate");
  const gateView = deriveGateView(node);
  if (gateView === "agreed") chips.push("待拍板");
  if (gateView === "approved") chips.push("已通过");
  if (gateView === "rejected") chips.push("已打回");
  return (
    <text x={x} y={y} textAnchor="end" className="flow-node-chips">
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
