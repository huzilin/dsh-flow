import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { NodeDetail } from "./NodeDetail";
import type { FlowNodeItem, SessionItem } from "../api/proto/api";

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
