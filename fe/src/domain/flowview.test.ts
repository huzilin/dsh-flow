import { describe, expect, it } from "vitest";
import type { ArtifactItem, FlowNodeItem, IssueItem, MyTaskResp } from "../api/proto/api";
import {
  anchorSummary,
  chunkRanges,
  countOpenIssues,
  deriveGateView,
  inReviewCommentContext,
  inSelection,
  issueReviewerOf,
  parseReplies,
  rejectedReentryHint,
  reviewAnchorGuardError,
  reviewArtifactGroups,
  scanFenceBlocks,
  seatStatesOf,
  selectionLabel,
  splitContentLines,
  summarizeReviews,
  unagreeSeatsOf,
} from "./flowview";

/** 视图模型纯函数表驱动测试（与 rd/domain 口径逐项对齐）。 */

function node(overrides: Partial<FlowNodeItem>): FlowNodeItem {
  return {
    nodeId: "n-03",
    defNodeId: "bus-03",
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
    ...overrides,
  };
}

describe("summarizeReviews（与 SummarizeReviewResults 同口径）", () => {
  it("全轮次累计 agree/unagree/reviewing，seats=评审位数", () => {
    const n = node({
      reviewers: '[{"role":"arch","capabilityId":"arch.prd-review"},{"role":"rd","capabilityId":"rd.prd-review"}]',
      reviewResults:
        '[{"role":"arch","capabilityId":"arch.prd-review","state":"agree","round":1,"at":1},' +
        '{"role":"rd","capabilityId":"rd.prd-review","state":"reviewing","round":1,"at":2},' +
        '{"role":"qa","capabilityId":"qa.rd-design-review","state":"unagree","round":0,"at":3}]',
    });
    const s = summarizeReviews(n);
    expect(s).toEqual({ agree: 1, unagree: 1, reviewing: 1, seats: 2, maxRound: 1 });
  });
  it("无评审位/损坏 JSON 降级为零值不抛", () => {
    expect(summarizeReviews(node({ reviewers: "not-json", reviewResults: "" }))).toEqual({
      agree: 0,
      unagree: 0,
      reviewing: 0,
      seats: 0,
      maxRound: 0,
    });
  });
});

describe("deriveGateView（拍板卡只读态判定）", () => {
  it("last_decision approved/rejected 优先", () => {
    expect(deriveGateView(node({ lastDecision: '{"verdict":"approved","at":1}' }))).toBe("approved");
    expect(deriveGateView(node({ lastDecision: '{"verdict":"rejected","at":1}' }))).toBe("rejected");
  });
  it("全 agree 无反对无待表态 → agreed（待拍板）", () => {
    expect(
      deriveGateView(
        node({
          userGate: "1",
          reviewers: '[{"role":"qa","capabilityId":"qa.fe-design-review"}]',
          reviewResults:
            '[{"role":"qa","capabilityId":"qa.fe-design-review","state":"agree","round":1,"at":1}]',
        }),
      ),
    ).toBe("agreed");
  });
  it("有待表态/有反对/无评审位 → 空", () => {
    const reviewers = '[{"role":"qa","capabilityId":"qa.fe-design-review"}]';
    expect(
      deriveGateView(
        node({
          reviewers,
          reviewResults:
            '[{"role":"qa","capabilityId":"qa.fe-design-review","state":"reviewing","round":1,"at":1}]',
        }),
      ),
    ).toBe("");
    expect(
      deriveGateView(
        node({
          reviewers,
          reviewResults:
            '[{"role":"qa","capabilityId":"qa.fe-design-review","state":"unagree","round":1,"at":1}]',
        }),
      ),
    ).toBe("");
    expect(deriveGateView(node({ reviewers: "[]" }))).toBe("");
  });
});

describe("正文/代码块/分段", () => {
  it("splitContentLines：1 起行号 + 尾空行保真", () => {
    expect(splitContentLines("# 标题\n\n尾行")).toEqual([
      { no: 1, text: "# 标题" },
      { no: 2, text: "" },
      { no: 3, text: "尾行" },
    ]);
    expect(splitContentLines("")).toEqual([]);
    expect(splitContentLines("single")).toEqual([{ no: 1, text: "single" }]);
  });
  it("scanFenceBlocks：成对围栏取 lang，不成对忽略", () => {
    const lines = splitContentLines("a\n```sql\nselect 1;\n```\nb\n```\n未闭合");
    const blocks = scanFenceBlocks(lines);
    expect(blocks).toEqual([{ startLine: 2, endLine: 4, lang: "sql" }]);
  });
  it("chunkRanges：200 行分段", () => {
    expect(chunkRanges(5, 200)).toEqual([[1, 5]]);
    expect(chunkRanges(450, 200)).toEqual([
      [1, 200],
      [201, 400],
      [401, 450],
    ]);
  });
  it("inSelection：区间命中", () => {
    expect(inSelection(3, { start: 2, end: 5 })).toBe(true);
    expect(inSelection(6, { start: 2, end: 5 })).toBe(false);
    expect(inSelection(1, null)).toBe(false);
  });
});

describe("锚点/评论展示辅助", () => {
  it("anchorSummary 四级文案与后端同口径", () => {
    const base = {
      anchorArtifactId: "art-1",
      anchorLine: "42",
      anchorStartLine: "7",
      anchorEndLine: "9",
      anchorBlockStart: "2",
      anchorBlockEnd: "8",
      anchorLang: "sql",
    };
    const issue = (kind: string) => ({ anchorKind: kind, ...base }) as unknown as IssueItem;
    expect(anchorSummary(issue("line"))).toBe("art-1 line 42");
    expect(anchorSummary(issue("range"))).toBe("art-1 line 7-9");
    expect(anchorSummary(issue("block"))).toBe("art-1 block 2-8(sql)");
    // BUG-013 后 document 锚可携带 artifactId：已挂产物呈「id（全文）」，未挂产物维持 "document"
    expect(anchorSummary(issue("document"))).toBe("art-1（全文）");
    expect(
      anchorSummary({ anchorKind: "document", anchorArtifactId: "" } as unknown as IssueItem),
    ).toBe("document");
  });
  it("issueReviewerOf：reviewerRole 优先缺省 author", () => {
    expect(issueReviewerOf({ reviewerRole: "arch", author: "pm" } as IssueItem)).toBe("arch");
    expect(issueReviewerOf({ reviewerRole: "", author: "pm" } as IssueItem)).toBe("pm");
  });
  it("countOpenIssues：resolved 不计（09-12 裁决 overridden 态废除）", () => {
    const mk = (state: string) => ({ state }) as IssueItem;
    expect(countOpenIssues([mk("open"), mk("resolved"), mk("to-fix")])).toBe(2);
  });
  it("selectionLabel 划词两级文案（注记-L3）", () => {
    expect(selectionLabel(null)).toBe("全文评论（document）");
    expect(selectionLabel({ start: 42, end: 42 })).toBe("评论第 42 行");
    expect(selectionLabel({ start: 7, end: 9 })).toBe("评论第 7-9 行区间");
  });
  it("parseReplies：voided 标注保留", () => {
    const replies = parseReplies(
      '[{"replyId":"r-1","by":"pm","roleId":"pm","text":"已修订","at":5},' +
        '{"replyId":"r-2","by":"pm","roleId":"pm","text":"误发","voided":true,"at":6}]',
    );
    expect(replies).toHaveLength(2);
    expect(replies[1]?.voided).toBe(true);
  });
});

describe("T16 拍板卡/冲突卡纯函数", () => {
  const node = (over: Partial<FlowNodeItem>): FlowNodeItem =>
    ({
      nodeId: "n-03",
      defNodeId: "bus-02",
      nodeOrder: "3",
      nodeKey: "x",
      state: "review",
      reviewResults: "[]",
      reopenSource: "",
      rolledBackCount: "0",
      featureBranch: "",
      enteredAt: "0",
      nodeType: "produce",
      gate: "0",
      userGate: "0",
      reviewers: "[]",
      lastDecision: "",
      ...over,
    }) as FlowNodeItem;

  it("unagreeSeatsOf：反对位取最新轮；agree/reviewing 不计", () => {
    const n = node({
      reviewResults:
        '[{"role":"qa","capabilityId":"q","state":"unagree","round":1,"at":1},' +
        '{"role":"qa","capabilityId":"q","state":"agree","round":2,"at":2},' +
        '{"role":"rd","capabilityId":"r","state":"unagree","round":2,"at":3},' +
        '{"role":"rd","capabilityId":"r","state":"unagree","round":2,"at":4}]',
    });
    const seats = unagreeSeatsOf(n);
    expect(seats).toHaveLength(1);
    expect(seats[0]).toMatchObject({ role: "rd", capabilityId: "r", round: 2, at: 4 });
  });

  it("seatStatesOf：评审位全覆盖（无结果=reviewing）；同轮覆写取最后", () => {
    const n = node({
      reviewers: '[{"role":"qa","capabilityId":"q"},{"role":"rd","capabilityId":"r"}]',
      reviewResults:
        '[{"role":"qa","capabilityId":"q","state":"agree","round":1,"at":1},' +
        '{"role":"qa","capabilityId":"q","state":"unagree","round":1,"at":2}]',
    });
    const seats = seatStatesOf(n);
    expect(seats).toHaveLength(2);
    expect(seats.find((s) => s.role === "qa")?.state).toBe("unagree");
    expect(seats.find((s) => s.role === "rd")?.state).toBe("reviewing");
  });

  it("rejectedReentryHint：快照相位读取（review=等待重入 / exec=返工执行中）", () => {
    expect(rejectedReentryHint(node({ state: "review" }))).toBe("已打回，等待 exec 重入");
    expect(rejectedReentryHint(node({ state: "" }))).toBe("已打回，等待 exec 重入");
    expect(rejectedReentryHint(node({ state: "exec" }))).toBe("已打回重入：exec 返工执行中");
  });

  it("parseReplies：T12 条目 kind/verdict/sessionId 透传", () => {
    const replies = parseReplies(
      '[{"replyId":"r-1","by":"user","roleId":"user","text":"结论","at":5,' +
        '"kind":"adjudication","verdict":"uphold","sessionId":"ses-7"}]',
    );
    expect(replies[0]?.kind).toBe("adjudication");
    expect(replies[0]?.verdict).toBe("uphold");
    expect(replies[0]?.sessionId).toBe("ses-7");
  });
});

import { buildFlowRows } from "./flowview";

/** exec-CR 节点工厂（produce+gate+1 评审位——并行组判定命中型）。 */
function laneNode(order: number, key: string, opts?: { noGate?: boolean; noSeats?: boolean; type?: string }): FlowNodeItem {
  return {
    nodeId: `n-${order}`,
    defNodeId: `d-${order}`,
    nodeOrder: String(order),
    nodeKey: key,
    state: "",
    reviewResults: "[]",
    reopenSource: "",
    rolledBackCount: "0",
    featureBranch: "",
    enteredAt: "0",
    nodeType: opts?.type ?? "produce",
    gate: opts?.noGate ? "0" : "1",
    userGate: "0",
    reviewers: opts?.noSeats ? "[]" : '[{"role":"rd","capabilityId":"implement-techdesign"}]',
    lastDecision: "",
  } as FlowNodeItem;
}

describe("buildFlowRows 并行泳道行构建（wb_advance join 同口径镜像）", () => {
  it("business full：相邻 RD∥FE 开发（produce+gate+评审位）成 parallel 行，rd 居左", () => {
    const nodes = [
      laneNode(9, "RD 开发"),
      laneNode(10, "FE 开发"),
    ];
    const rows = buildFlowRows(nodes);
    expect(rows).toHaveLength(1);
    expect(rows[0].kind).toBe("parallel");
    if (rows[0].kind === "parallel") {
      expect(rows[0].rd.nodeKey).toBe("RD 开发");
      expect(rows[0].fe.nodeKey).toBe("FE 开发");
    }
  });

  it("rd-only：FE 侧已被 scope 裁剪，单个 exec-CR 不成组（全 single 行）", () => {
    const rows = buildFlowRows([laneNode(9, "RD 开发")]);
    expect(rows).toHaveLength(1);
    expect(rows[0].kind).toBe("single");
  });

  it("非 exec-CR 相邻（缺门禁/缺评审位/非 produce）不成组", () => {
    const rows = buildFlowRows([
      laneNode(1, "串行产出", { noGate: true }),
      laneNode(2, "串行产出2"),
    ]);
    expect(rows.every((r) => r.kind === "single")).toBe(true);

    const rows2 = buildFlowRows([
      laneNode(1, "无评审位产出", { noSeats: true }),
      laneNode(2, "开发"),
    ]);
    expect(rows2.every((r) => r.kind === "single")).toBe(true);

    const rows3 = buildFlowRows([
      laneNode(1, "评审节点", { type: "review" }),
      laneNode(2, "开发"),
    ]);
    expect(rows3.every((r) => r.kind === "single")).toBe(true);
  });

  it("纵向单链混排：组前/组中/组后节点均保持 single 行且顺序不乱", () => {
    const nodes = [
      laneNode(8, "FE 设计评审", { type: "review", noGate: true }),
      laneNode(9, "RD 开发"),
      laneNode(10, "FE 开发"),
      laneNode(11, "QA 验收", { noSeats: true }),
    ];
    const rows = buildFlowRows(nodes);
    expect(rows.map((r) => r.kind)).toEqual(["single", "parallel", "single"]);
  });
});

// ---------------------------------------------------------------------------
// BUG-011 评审产物分组 / BUG-013 评审上下文锚产物（2026-09-10 批注轮）
// ---------------------------------------------------------------------------

function artifact(overrides: Partial<ArtifactItem>): ArtifactItem {
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

function myTask(relatedArtifacts: Array<Pick<ArtifactItem, "artifactId" | "productType" | "label" | "kind">>): MyTaskResp {
  return {
    ok: true,
    requirementId: "req-001",
    role: "user",
    node: undefined,
    currentPhase: "review",
    activeActions: [],
    reviewSummary: undefined,
    relatedNodes: [],
    relatedArtifacts,
    relatedTasks: [],
    relatedIssues: [],
  };
}

describe("reviewArtifactGroups（BUG-011 评审产物两段视图模型）", () => {
  it("评审节点：relatedArtifacts 非本节点段=被评审产物（保持 rd 置前装配序），本节点段=快照全集", () => {
    const n = node({ nodeId: "n-04", nodeType: "review" });
    const snap = [
      artifact({ artifactId: "art-up-1", nodeId: "n-02", label: "《RD 技术设计》" }),
      artifact({ artifactId: "art-up-2", nodeId: "n-03", label: "《架构方案》" }),
      artifact({ artifactId: "art-own", nodeId: "n-04", label: "《评审纪要》" }),
    ];
    const g = reviewArtifactGroups(
      n,
      myTask([
        { artifactId: "art-up-2", productType: "architecture", label: "《架构方案》", kind: "document" },
        { artifactId: "art-up-1", productType: "rd_design", label: "《RD 技术设计》", kind: "document" },
        { artifactId: "art-own", productType: "discussion", label: "《评审纪要》", kind: "document" },
      ]),
      snap,
    );
    expect(g.reviewed.map((a) => a.artifactId)).toEqual(["art-up-2", "art-up-1"]);
    expect(g.own.map((a) => a.artifactId)).toEqual(["art-own"]);
  });
  it("非评审节点不分段（reviewed 恒空，own=本节点全集）", () => {
    const n = node({ nodeId: "n-02", nodeType: "produce", state: "exec" });
    const g = reviewArtifactGroups(n, myTask([]), [artifact({ artifactId: "art-own", nodeId: "n-02" })]);
    expect(g.reviewed).toEqual([]);
    expect(g.own.map((a) => a.artifactId)).toEqual(["art-own"]);
  });
  it("my_task 未加载（undefined）时 reviewed 空集（UI 呈加载态），own 不受影响", () => {
    const n = node({ nodeId: "n-04", nodeType: "review" });
    const g = reviewArtifactGroups(n, undefined, [artifact({ artifactId: "art-own", nodeId: "n-04" })]);
    expect(g.reviewed).toEqual([]);
    expect(g.own).toHaveLength(1);
  });
  it("relatedArtifacts 中快照查不到的条目跳过（防御脏投影）", () => {
    const n = node({ nodeId: "n-04", nodeType: "review" });
    const g = reviewArtifactGroups(
      n,
      myTask([{ artifactId: "art-ghost", productType: "plan", label: "幽灵产物", kind: "document" }]),
      [artifact({ artifactId: "art-own", nodeId: "n-04" })],
    );
    expect(g.reviewed).toEqual([]);
  });
});

describe("inReviewCommentContext（BUG-013；rd inReviewContext 同口径镜像）", () => {
  it("相位 review / respond 恒为评审上下文", () => {
    expect(inReviewCommentContext(node({ state: "review" }), "user")).toBe(true);
    expect(inReviewCommentContext(node({ state: "respond" }), "qa")).toBe(true);
  });
  it("非评审相位但发表角色在评审位声明中（未激活节点同受约束）", () => {
    const n = node({
      state: "exec",
      reviewers: '[{"role":"arch","capabilityId":"arch.prd-review"}]',
    });
    expect(inReviewCommentContext(n, "arch")).toBe(true);
    expect(inReviewCommentContext(n, "user")).toBe(false);
  });
});

describe("reviewAnchorGuardError（BUG-013 发表前预校验；rd 400 同口径前置）", () => {
  const ctx = { reviewContext: true, issueType: "issue", anchorKind: "document" };
  it("评审上下文 issue 类 document 锚缺 artifactId=拦下并给引导文案", () => {
    const err = reviewAnchorGuardError({ ...ctx, artifactId: "" });
    expect(err).toContain("必须锚定产物");
  });
  it("携带 artifactId 放行", () => {
    expect(reviewAnchorGuardError({ ...ctx, artifactId: "art-1" })).toBeNull();
  });
  it("line/range/block 天然挂产物不受限；非 issue 类不受限；非评审上下文不受限", () => {
    expect(reviewAnchorGuardError({ ...ctx, anchorKind: "line", artifactId: "" })).toBeNull();
    expect(reviewAnchorGuardError({ ...ctx, issueType: "discussion", artifactId: "" })).toBeNull();
    expect(
      reviewAnchorGuardError({ reviewContext: false, issueType: "issue", anchorKind: "document", artifactId: "" }),
    ).toBeNull();
  });
});

describe("anchorSummary 产物名呈现（BUG-013 评论展示挂产物名）", () => {
  it("传入 label 映射时以产物名替代裸 id，document 已挂产物呈「名（全文）」", () => {
    const issue = {
      anchorKind: "document",
      anchorArtifactId: "art-1",
      anchorLine: "0",
      anchorStartLine: "0",
      anchorEndLine: "0",
      anchorBlockStart: "0",
      anchorBlockEnd: "0",
      anchorLang: "",
    } as IssueItem;
    expect(anchorSummary(issue, (id) => (id === "art-1" ? "《RD 技术设计》" : ""))).toBe(
      "《RD 技术设计》（全文）",
    );
    // 不传映射时回落裸 id（非 "document"——已挂产物与未挂产物可区分）
    expect(anchorSummary(issue)).toBe("art-1（全文）");
    expect(
      anchorSummary({ ...issue, anchorArtifactId: "" }, (id) => (id === "art-1" ? "《RD 技术设计》" : "")),
    ).toBe("document");
  });
  it("line 锚产物名 + 行号形态保持", () => {
    const issue = {
      anchorKind: "line",
      anchorArtifactId: "art-1",
      anchorLine: "42",
      anchorStartLine: "0",
      anchorEndLine: "0",
      anchorBlockStart: "0",
      anchorBlockEnd: "0",
      anchorLang: "",
    } as IssueItem;
    expect(anchorSummary(issue, () => "《设计》")).toBe("《设计》 line 42");
  });
});
