import { useMemo } from "react";
import type { FlowItem, FlowNodeItem } from "../api/proto/api";
import { deriveGateView, parseReviewResults, summarizeReviews } from "../domain/flowview";
import { Badge, ScopeBadge } from "./Badges";

/**
 * 流程树（模块 II 主视图）：SVG 一次渲染全实例节点（seed 上限 40；被裁节点已由后端
 * scope 过滤不返回，故天然不渲染不占位）。节点按 node_order 纵向排布单主链——
 * 实例节点无并行组/泳道字段（flow_nodes DDL 无该列），原型横向泳道不适用，纵向渲染；
 * 渲染矩阵一致性由后端 T4 实例化测试兜底（AC：三种 scope 节点集/序号一致）。
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
  const sorted = useMemo(
    () => [...nodes].sort((a, b) => Number(a.nodeOrder) - Number(b.nodeOrder)),
    [nodes],
  );
  if (sorted.length === 0) {
    return <div className="flow-tree-empty muted">流程实例尚未生成（需求创建即实例化）。</div>;
  }
  const height = sorted.length * (CARD_H + GAP) + GAP * 2;
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
          {sorted.length} 节点 · 被裁节点不渲染不占位（node_order 已重排）· 点击节点查看详情
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
        {sorted.map((node, i) => {
          const top = GAP + i * (CARD_H + GAP);
          const isCurrent = node.nodeId === currentNodeId;
          const isSelected = node.nodeId === selectedNodeId;
          const gateView = deriveGateView(node);
          return (
            <g
              key={node.nodeId}
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
              {i > 0 && (
                <path
                  d={`M${ROW_W / 2},${top - GAP} v${GAP}`}
                  className={`flow-link${gateView === "approved" ? " approved" : ""}`}
                />
              )}
              <rect
                x={GAP}
                y={top}
                width={ROW_W - GAP * 2}
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
                cx={GAP + 18}
                cy={top + CARD_H / 2}
                r={5}
                className={`phase-dot phase-${node.state || "none"}`}
              />
              <text x={GAP + 34} y={top + 22} className="flow-node-title">
                {node.nodeOrder}. {node.nodeKey}
              </text>
              <NodeMetaText node={node} x={ROW_W - GAP - 8} y={top + 22} />
              <text x={GAP + 34} y={top + 42} className="flow-node-sub">
                {nodeSubLine(node, issuesByNode.get(node.nodeId) ?? 0)}
              </text>
              {isCurrent && (
                <>
                  <rect
                    x={ROW_W - GAP - 64}
                    y={top + 32}
                    width={56}
                    height={16}
                    rx={4}
                    className="flow-current-chip"
                  />
                  <text x={ROW_W - GAP - 36} y={top + 44} className="flow-current-text">
                    当前
                  </text>
                </>
              )}
            </g>
          );
        })}
      </svg>
    </div>
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
