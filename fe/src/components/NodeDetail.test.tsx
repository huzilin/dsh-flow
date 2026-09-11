import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { NodeDetail } from "./NodeDetail";
import { renderWithClient } from "../test-utils";
import type { ArtifactItem, FlowNodeItem, IssueItem, MyTaskResp, SessionItem } from "../api/proto/api";

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

// ---------------------------------------------------------------------------
// 药丸本体高亮（一版 A3-2/B-27 回归：N 未解决 Number badge + 轮次≥10 卡死红边）
// ---------------------------------------------------------------------------

function makeIssue(partial: Partial<IssueItem>): IssueItem {
  return {
    issueId: "cm-1",
    nodeId: "n-04",
    anchorKind: "document",
    author: "arch",
    roleId: "arch",
    text: "口径待澄清",
    state: "open",
    escalate: "0",
    loopCount: "0",
    roundCount: "1",
    createdAt: "1730000000000",
    issueType: "issue",
    reviewerRole: "arch",
    assignee: "rd",
    anchorArtifactId: "",
    anchorLine: "0",
    anchorStartLine: "0",
    anchorEndLine: "0",
    anchorBlockStart: "0",
    anchorBlockEnd: "0",
    anchorLang: "",
    replies: "[]",
    lastReplyAt: "0",
    resolvePath: "",
    resolveBy: "",
    resolveAt: "0",
    resolveNote: "",
    resolveReviewerConfirmed: "0",
    resolvedBy: "",
    resolvedAt: "0",
    updatedAt: "1730000000000",
    ...partial,
  };
}

function renderPillCase(reviewResults: string, issues: IssueItem[]) {
  const node = makeNode();
  node.reviewers = '[{"role":"arch","capabilityId":"arch.prd-review"}]';
  node.reviewResults = reviewResults;
  // issues 非空集会渲染 CommentCard（mutations）——走 renderWithClient 装配 QueryClient
  return renderWithClient(
    <NodeDetail
      nodeId="n-04"
      node={node}
      myTask={makeMyTask()}
      artifacts={[makeArtifact({ artifactId: "art-up", nodeId: "n-02" })]}
      issues={issues}
      requirementId="req-001"
    />,
  );
}

describe("NodeDetail ② 药丸本体高亮（一版 A3-2 回归）", () => {
  it("N 未解决 Number badge 上药丸；轮次≥10 且未闭合 → is-stalled 红边 + 轮 N 徽标", () => {
    const { container } = renderPillCase(
      '[{"role":"arch","capabilityId":"arch.prd-review","state":"unagree","round":12,"at":1}]',
      [
        makeIssue({ issueId: "cm-1", state: "open" }),
        makeIssue({ issueId: "cm-2", state: "to-fix", text: "待修复项" }),
      ],
    );
    const pill = screen.getByRole("button", {
      name: "arch（arch.prd-review）表态药丸，2 未解决，疑似卡死",
    });
    expect(pill.className).toContain("is-stalled");
    expect(pill.textContent).toContain("2 未解决");
    expect(pill.textContent).toContain("轮 12");
    expect(pill.getAttribute("aria-expanded")).toBe("false");
    // ① 状态卡不再重复放卡死徽标（高亮已移药丸本体）
    expect(container.querySelector(".badge-stall")).toBeNull();
  });

  it("无未闭合评论：无 badge 无红边（aria-label 不带后缀）", () => {
    renderPillCase(
      '[{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":12,"at":1}]',
      [],
    );
    const pill = screen.getByRole("button", { name: "arch（arch.prd-review）表态药丸" });
    expect(pill.className).not.toContain("is-stalled");
    expect(pill.textContent).not.toContain("未解决");
  });
});
