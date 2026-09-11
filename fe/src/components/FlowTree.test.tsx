import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import type { FlowItem, FlowNodeItem } from "../api/proto/api";
import { FlowTree } from "./FlowTree";

/**
 * 流程树渲染测试（工单 T15 验收 1/7）：
 * - scope 重排快照用例：full 全渲染 vs rd 裁剪重排（对齐 T4 渲染矩阵——被裁节点不渲染不占位）
 * - current_node_id 高亮、评审轮摘要计数、gate/user_gate 标识、节点点击进详情
 * 渲染矩阵真相源在后端实例化（T4 httptest 已断言），FE 快照锁「集-序」防前端重排漂移。
 */

/** business 流程 scope=full 全 13 节点（seed 0002：bus-00…bus-12）。 */
const FULL_DEFS: Array<[string, string, string, string, string]> = [
  // [defNodeId, nodeKey, nodeType, scopeExpr, gate/userGate]
  ["bus-00", "待启动", "gate", "", "0/1"],
  ["bus-01", "需求对齐", "align", "", "0/1"],
  ["bus-02", "产品设计", "produce", "", "0/0"],
  ["bus-03", "产品设计评审", "review", "", "1/1"],
  ["bus-04", "RD 技术设计", "produce", "rd|full", "0/0"],
  ["bus-05", "RD 技术评审", "review", "rd|full", "1/1"],
  ["bus-06", "FE 技术设计", "produce", "fe|full", "0/0"],
  ["bus-07", "FE 设计评审", "review", "fe|full", "1/0"],
  ["bus-08", "RD 开发", "produce", "rd|full", "1/0"],
  ["bus-09", "FE 开发", "produce", "fe|full", "1/0"],
  ["bus-10", "QA 验收（含联调）", "produce", "full", "1/0"],
  ["bus-11", "用户验收", "gate", "", "1/1"],
  ["bus-12", "完成", "gate", "", "0/0"],
];

/** rd scope 渲染矩阵（9 节点：0→1→2→3→4→5→8→11→12；fe/QA 侧不渲染不占位，序号重排）。 */
const RD_MATRIX = ["bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-08", "bus-11", "bus-12"];

function makeNode(defId: string, order: number, reviewResults = "[]"): FlowNodeItem {
  const def = FULL_DEFS.find(([d]) => d === defId)!;
  const [gate, userGate] = def[4].split("/");
  return {
    nodeId: `n-${String(order).padStart(2, "0")}`,
    defNodeId: defId,
    nodeOrder: String(order),
    nodeKey: def[1],
    state: order <= 4 ? "review" : "",
    reviewResults,
    reopenSource: "",
    rolledBackCount: "0",
    featureBranch: "",
    enteredAt: order <= 4 ? "1730000000000" : "0",
    nodeType: def[2],
    gate,
    userGate,
    reviewers: def[2] === "review" ? '[{"role":"arch","capabilityId":"arch.prd-review"},{"role":"rd","capabilityId":"rd.prd-review"}]' : "[]",
    lastDecision: "",
  };
}

/** scope=full：13 节点全渲染（node_order 1..13 原序）。 */
function fullNodes(): FlowNodeItem[] {
  return FULL_DEFS.map(([defId], i) => makeNode(defId, i + 1));
}

/** scope=rd：被裁节点不返回不占位，node_order 已由后端重排 1..9。 */
function rdNodes(): FlowNodeItem[] {
  return RD_MATRIX.map((defId, i) => makeNode(defId, i + 1));
}

const flow = (scope: string, currentNodeId: string): FlowItem => ({
  flowDefsId: "business",
  dependencies: "[]",
  currentNodeId,
  createdAt: "1",
  updatedAt: "1",
  scope,
});

function renderTree(nodes: FlowNodeItem[], scope: string, currentNodeId = "") {
  const onSelect = vi.fn();
  render(
    <FlowTree
      flow={flow(scope, currentNodeId)}
      nodes={nodes}
      currentNodeId={currentNodeId}
      selectedNodeId={currentNodeId === "" ? null : currentNodeId}
      issuesByNode={new Map()}
      onSelectNode={onSelect}
    />,
  );
  return { onSelect };
}

function renderNodes(nodes: FlowNodeItem[], scope: string) {
  return render(
    <FlowTree
      flow={flow(scope, "")}
      nodes={nodes}
      currentNodeId=""
      selectedNodeId={null}
      issuesByNode={new Map()}
      onSelectNode={vi.fn()}
    />,
  );
}

describe("FlowTree（scope 重排快照）", () => {
  it("scope=full：13 节点全渲染快照（集-序锁定）", () => {
    const { container } = renderNodes(fullNodes(), "full");
    expect(screen.getAllByRole("listitem")).toHaveLength(13);
    const labels = screen
      .getAllByRole("listitem")
      .map((g) => g.getAttribute("aria-label"));
    expect(labels).toEqual([
      "节点 1 待启动",
      "节点 2 需求对齐",
      "节点 3 产品设计",
      "节点 4 产品设计评审",
      "节点 5 RD 技术设计",
      "节点 6 RD 技术评审",
      "节点 7 FE 技术设计",
      "节点 8 FE 设计评审",
      "节点 9 RD 开发",
      "节点 10 FE 开发",
      "节点 11 QA 验收（含联调）",
      "节点 12 用户验收",
      "节点 13 完成",
    ]);
    expect(container.querySelector("svg")).toMatchSnapshot("flowtree-full-13");
  });

  it("scope=rd：9 节点单链快照（fe/QA 侧被裁不占位，序号重排）", () => {
    const { container } = renderNodes(rdNodes(), "rd");
    const items = screen.getAllByRole("listitem");
    expect(items).toHaveLength(9);
    expect(items.map((g) => g.getAttribute("aria-label"))).toEqual([
      "节点 1 待启动",
      "节点 2 需求对齐",
      "节点 3 产品设计",
      "节点 4 产品设计评审",
      "节点 5 RD 技术设计",
      "节点 6 RD 技术评审",
      "节点 7 RD 开发",
      "节点 8 用户验收",
      "节点 9 完成",
    ]);
    expect(container.querySelector("svg")).toMatchSnapshot("flowtree-rd-9");
  });

  it("scope 徽标呈现 flows.scope 实值", () => {
    renderNodes(rdNodes(), "rd");
    expect(screen.getByText("rd")).toBeInTheDocument();
  });

  it("current_node_id 高亮（data-current + 当前章）", () => {
    const nodes = fullNodes();
    const current = nodes[3]!; // bus-03 产品设计评审
    render(
      <FlowTree
        flow={flow("full", current.nodeId)}
        nodes={nodes}
        currentNodeId={current.nodeId}
        selectedNodeId={current.nodeId}
        issuesByNode={new Map()}
        onSelectNode={vi.fn()}
      />,
    );
    const g = screen.getByRole("listitem", { name: "节点 4 产品设计评审" });
    expect(g.getAttribute("data-current")).toBe("true");
    expect(g.querySelector(".flow-current-text")?.textContent).toBe("当前");
    expect(screen.getByRole("listitem", { name: "节点 5 RD 技术设计" }).getAttribute("data-current")).toBeNull();
  });

  it("评审轮摘要计数渲染（表态 N/M · agree/unagree/reviewing）", () => {
    const nodes = [
      makeNode(
        "bus-03",
        1,
        '[{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":1,"at":1},' +
          '{"role":"rd","capabilityId":"rd.prd-review","state":"unagree","round":1,"at":2}]',
      ),
    ];
    renderNodes(nodes, "full");
    expect(screen.getByText(/表态 2\/2 · 同意 1 · 反对 1 · 待表态 0/)).toBeInTheDocument();
  });

  it("无评审位节点显示 无评审位；gate/user_gate 标识在角标行", () => {
    renderNodes([makeNode("bus-02", 1), makeNode("bus-00", 2)], "full");
    expect(screen.getAllByText(/无评审位/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("1. 产品设计").textContent).toContain("产品设计");
    const gateNode = screen.getByRole("listitem", { name: "节点 2 待启动" });
    expect(gateNode.textContent).toContain("gate");
    expect(gateNode.textContent).toContain("user_gate");
  });

  it("节点点击进详情（onSelectNode 回传 nodeId）", async () => {
    const user = (await import("@testing-library/user-event")).default.setup();
    const { onSelect } = renderTree(fullNodes(), "full");
    await user.click(screen.getByRole("listitem", { name: "节点 6 RD 技术评审" }));
    expect(onSelect).toHaveBeenCalledWith("n-06");
  });

  it("实例节点空集：渲染空态提示", () => {
    renderTree([], "full");
    expect(screen.getByText(/流程实例尚未生成/)).toBeInTheDocument();
  });
});

/** 一版交互蓝本回归（inventory A2-2/3/4/5，B-18/19/20）：横向时序 + 双泳道 + 组级折线。
 *  泳道侧 exec-CR 判定需 produce+gate+评审位，bus-08/09（RD/FE 开发）补 qa 评审位成对。 */
describe("FlowTree 横向泳道布局（一版蓝本回归）", () => {
  function fullNodesWithDevPair(): FlowNodeItem[] {
    const nodes = fullNodes();
    for (const n of nodes) {
      if (n.defNodeId === "bus-08" || n.defNodeId === "bus-09") {
        n.reviewers = '[{"role":"qa","capabilityId":"qa.exec-cr"}]';
      }
    }
    return nodes;
  }

  it("并行组上下双泳道：RD 上蓝（topY）/ FE 下绿（botY）+ 泳道标签", () => {
    const { container } = renderNodes(fullNodesWithDevPair(), "full");
    expect(screen.getByText("RD ⇄ 后端")).toBeInTheDocument();
    expect(screen.getByText("FE ⇄ 前端")).toBeInTheDocument();
    const rd = container.querySelector('[data-node-id="n-09"] .flow-node-card')!;
    const fe = container.querySelector('[data-node-id="n-10"] .flow-node-card')!;
    expect(rd.getAttribute("class")).toContain("lane-rd");
    expect(fe.getAttribute("class")).toContain("lane-fe");
    expect(Number(rd.getAttribute("y"))).toBeLessThan(Number(fe.getAttribute("y")));
    // 主链节点不挂泳道色
    const single = container.querySelector('[data-node-id="n-01"] .flow-node-card')!;
    expect(single.getAttribute("class")).not.toContain("lane-");
  });

  it("组级分叉/汇聚折线：进组一分叉多（H-V-H 折点）、出组多汇聚一；单链无折点", () => {
    const full = renderNodes(fullNodesWithDevPair(), "full");
    const fullPaths = Array.from(full.container.querySelectorAll("path.flow-link")).map((p) =>
      p.getAttribute("d"),
    );
    // n-08 → 组（分叉 2 条折线）+ 组 → n-11（汇聚 2 条折线）= 4 条带 V 折点
    const elbows = fullPaths.filter((d) => d?.includes("V"));
    expect(elbows).toHaveLength(4);
    expect(fullPaths.some((d) => !d?.includes("V"))).toBe(true);
    // rd scope：并行对不成组（bus-09 被裁），全单链水平直连无折点
    const rd = renderNodes(rdNodes(), "rd");
    const rdPaths = Array.from(rd.container.querySelectorAll("path.flow-link")).map((p) =>
      p.getAttribute("d"),
    );
    expect(rdPaths.length).toBeGreaterThan(0);
    expect(rdPaths.every((d) => !d?.includes("V"))).toBe(true);
    // 单泳道退化：rd-only 全部节点同一中线 y
    const ys = new Set(
      Array.from(rd.container.querySelectorAll(".flow-node-card")).map((r) => r.getAttribute("y")),
    );
    expect(ys.size).toBe(1);
  });

  it("回滚角标：rolled_back_count>1 显 ↣n；=1 不显角标", () => {
    const nodes = fullNodes();
    nodes[8]!.rolledBackCount = "2"; // bus-08 RD 开发
    renderNodes(nodes, "full");
    expect(screen.getByText("↣2")).toBeInTheDocument();
  });

  it("并行组容器为 role=group，节点仍为唯一 listitem（13 节点恒可数）", () => {
    renderNodes(fullNodesWithDevPair(), "full");
    expect(screen.getAllByRole("listitem")).toHaveLength(13);
    expect(screen.getByRole("group", { name: /并行组 RD 开发 ∥ FE 开发/ })).toBeInTheDocument();
  });
});

/** BUG-010 点色语义回归：门禁结论优先（绿=已通过/琥珀=待拍板/红=已打回），相位兜底（review 无结论=青，不得占琥珀）。 */
describe("FlowTree 点色语义（BUG-010）", () => {
  const dotClass = (node: FlowNodeItem) => {
    const { container } = renderNodes([node], "full");
    // SVG 元素 className 是 SVGAnimatedString，取 class 属性字符串
    return container.querySelector(".phase-dot")!.getAttribute("class") ?? "";
  };

  it("门禁已通过 → gate-approved（绿）", () => {
    const n = makeNode("bus-03", 4);
    n.lastDecision = '{"verdict":"approved","at":1}';
    expect(dotClass(n)).toContain("gate-approved");
  });

  it("全员 agree 待拍板 → gate-agreed（琥珀）", () => {
    const n = makeNode("bus-03", 4, '[{"role":"arch","state":"agree","round":0},{"role":"rd","state":"agree","round":0}]');
    expect(dotClass(n)).toContain("gate-agreed");
  });

  it("门禁已打回 → gate-rejected（红）", () => {
    const n = makeNode("bus-03", 4);
    n.lastDecision = '{"verdict":"rejected","at":1}';
    expect(dotClass(n)).toContain("gate-rejected");
  });

  it("review 相位无结论 → phase-review（青），不得占琥珀", () => {
    const n = makeNode("bus-03", 4, '[{"role":"arch","state":"agree","round":0},{"role":"rd","state":"reviewing","round":0}]');
    const cls = dotClass(n);
    expect(cls).toContain("phase-review");
    expect(cls).not.toContain("gate-agreed");
  });
});
