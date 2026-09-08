import { describe, expect, it } from "vitest";
import type { FlowNodeItem, IssueItem } from "../api/proto/api";
import {
  anchorSummary,
  chunkRanges,
  countOpenIssues,
  deriveGateView,
  inSelection,
  issueReviewerOf,
  parseReplies,
  rejectedReentryHint,
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
    expect(s).toEqual({ agree: 1, unagree: 1, reviewing: 1, seats: 2 });
  });
  it("无评审位/损坏 JSON 降级为零值不抛", () => {
    expect(summarizeReviews(node({ reviewers: "not-json", reviewResults: "" }))).toEqual({
      agree: 0,
      unagree: 0,
      reviewing: 0,
      seats: 0,
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
    expect(inSelection(3, { kind: "range", start: 2, end: 5 })).toBe(true);
    expect(inSelection(6, { kind: "range", start: 2, end: 5 })).toBe(false);
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
    expect(anchorSummary(issue("document"))).toBe("document");
  });
  it("issueReviewerOf：reviewerRole 优先缺省 author", () => {
    expect(issueReviewerOf({ reviewerRole: "arch", author: "pm" } as IssueItem)).toBe("arch");
    expect(issueReviewerOf({ reviewerRole: "", author: "pm" } as IssueItem)).toBe("pm");
  });
  it("countOpenIssues：resolved/overridden 不计", () => {
    const mk = (state: string) => ({ state }) as IssueItem;
    expect(countOpenIssues([mk("open"), mk("resolved"), mk("overridden"), mk("to-fix")])).toBe(2);
  });
  it("selectionLabel 三级文案", () => {
    expect(selectionLabel(null)).toBe("全文评论（document）");
    expect(selectionLabel({ kind: "line", start: 42, end: 42 })).toBe("评论第 42 行");
    expect(selectionLabel({ kind: "range", start: 7, end: 9 })).toBe("评论第 7-9 行区间");
    expect(selectionLabel({ kind: "block", start: 2, end: 8, lang: "sql" })).toBe(
      "评论代码块 2-8 行（sql）",
    );
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
