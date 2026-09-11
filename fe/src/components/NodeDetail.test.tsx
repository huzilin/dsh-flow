import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { NodeDetail } from "./NodeDetail";
import type { ArtifactItem, FlowNodeItem, MyTaskResp, SessionItem } from "../api/proto/api";

/** ⑥ 会话 tab 测试（§6 N5：sessions 按节点过滤呈现，空集占位）。 */
function makeNode(): FlowNodeItem {
  return {
    nodeId: "n-04",
    defNodeId: "bus-04",
    nodeOrder: "4",
    nodeKey: "产品设计评审",
    state: "review",
    reviewResults: "[]",
    reopenSource: "",
    rolledBackCount: "0",
    featureBranch: "",
    enteredAt: "0",
    nodeType: "review",
    gate: "1",
    userGate: "0",
    reviewers: "[]",
    lastDecision: "",
  };
}

function makeSession(overrides?: Partial<SessionItem>): SessionItem {
  return {
    sessionId: "s-adj-1",
    requirementId: "req-001",
    nodeId: "n-04",
    roleId: "arch",
    status: "active",
    updatedAt: "1730000000000",
    ...overrides,
  };
}

function renderDetail(sessions: SessionItem[]) {
  return render(
    <NodeDetail
      nodeId="n-04"
      node={makeNode()}
      myTask={undefined}
      artifacts={[]}
      issues={[]}
      sessions={sessions}
      requirementId="req-001"
    />,
  );
}

describe("NodeDetail ⑥ 会话 tab（§6 N5）", () => {
  it("空集呈现占位说明（DSH 嵌入期开放）", () => {
    renderDetail([]);
    expect(screen.getByText(/暂无会话绑定/)).toBeInTheDocument();
  });

  it("按当前节点过滤并呈现绑定角色/状态/时间", () => {
    renderDetail([
      makeSession(),
      makeSession({ sessionId: "s-other", nodeId: "n-05", status: "ended" }),
    ]);
    // 命中本节点的会话条目（角色 arch · 进行中），他节点会话不出现
    expect(screen.getByText("arch")).toBeInTheDocument();
    expect(screen.getByText("进行中")).toBeInTheDocument();
    expect(screen.getByText("s-adj-1")).toBeInTheDocument();
    expect(screen.queryByText("s-other")).not.toBeInTheDocument();
    expect(screen.getByTitle("sessions.status=active")).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// BUG-011 评审产物分组 / OPT-002 评审药丸可点（2026-09-10 批注轮）
// ---------------------------------------------------------------------------

function makeArtifact(overrides: Partial<ArtifactItem>): ArtifactItem {
  return {
    artifactId: "art-1",
    nodeId: "n-02",
    productType: "rd_design",
    label: "《RD 技术设计》",
    href: "docs/rd.md",
    kind: "document",
    note: "",
    addedBy: "rd",
    createdAt: "1730000000000",
    ...overrides,
  };
}

function makeMyTask(): MyTaskResp {
  return {
    ok: true,
    requirementId: "req-001",
    role: "user",
    node: undefined,
    currentPhase: "review",
    activeActions: [],
    reviewSummary: undefined,
    relatedNodes: [],
    relatedArtifacts: [
      { artifactId: "art-up", productType: "rd_design", label: "《RD 技术设计》", kind: "document" },
      { artifactId: "art-own", productType: "discussion", label: "《评审纪要》", kind: "document" },
    ],
    relatedTasks: [],
    relatedIssues: [],
  };
}

function renderFull(overrides?: { reviewers?: string }) {
  const node = makeNode();
  node.reviewers = overrides?.reviewers ?? "[]";
  return render(
    <NodeDetail
      nodeId="n-04"
      node={node}
      myTask={makeMyTask()}
      artifacts={[
        makeArtifact({ artifactId: "art-up", nodeId: "n-02" }),
        makeArtifact({ artifactId: "art-own", nodeId: "n-04", label: "《评审纪要》", productType: "discussion" }),
      ]}
      issues={[]}
      requirementId="req-001"
    />,
  );
}

describe("NodeDetail ③ 产物区分组（BUG-011）", () => {
  it("评审节点分「被评审产物（上游）/本节点产物」两段，被评审产物可点开（复用阅读器）", () => {
    renderFull();
    expect(screen.getByText("被评审产物（上游）")).toBeInTheDocument();
    expect(screen.getByText("本节点产物")).toBeInTheDocument();
    // 上游被评审产物行可点（开阅读器按钮出现）
    expect(screen.getByRole("button", { name: "《RD 技术设计》" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "《评审纪要》" })).toBeInTheDocument();
  });
  it("上游段顺序取 rd 置前装配序（上游在前、本节点在后）", () => {
    const { container } = renderFull();
    const labels = [...container.querySelectorAll(".artifact-group-label")].map((el) => el.textContent);
    expect(labels).toEqual(["被评审产物（上游）", "本节点产物"]);
  });
});

describe("NodeDetail ② 评审药丸可点（OPT-002）", () => {
  it("点击药丸展开该角色意见区，再点收起", () => {
    const reviewers = '[{"role":"arch","capabilityId":"arch.prd-review"}]';
    renderFull({ reviewers });
    const pill = screen.getByRole("button", { name: "arch（arch.prd-review）表态药丸" });
    expect(pill).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(pill);
    expect(pill).toHaveAttribute("aria-expanded", "true");
    // 该角色无意见时呈现引导（评审上下文必须挂产物）
    expect(screen.getByText(/暂无意见/)).toBeInTheDocument();
    fireEvent.click(pill);
    expect(pill).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText(/暂无意见/)).not.toBeInTheDocument();
  });
});
