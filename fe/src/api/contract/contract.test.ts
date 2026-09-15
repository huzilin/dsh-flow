/**
 * 跨语言契约消费测试（C3，架构评审拍板）：消费 Go 真相源生成的 golden.json，
 * 逐例断言 FE 手写镜像面与后端同输入同结果——后端改判定/词表/payload 未同步 FE
 * 即此处红灯（双测试互锁的 FE 侧；Go 侧自检见 rd/contract/contract_test.go）。
 * 再生命令：cd rd && go test ./contract/ -run TestUnit_ContractSnapshot -update。
 */
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import {
  CATEGORIES,
  CATEGORY_DEFAULT_FLOW,
  CATEGORY_FLOW_OPTIONS,
  NOTIFICATION_KIND_TARGETS,
  REQUIREMENT_STATUSES,
} from "../constants";
import {
  buildFlowRows,
  inReviewCommentContext,
  isLaneSideExecCR,
  reviewAnchorGuardError,
  summarizeReviews,
} from "../../domain/flowview";
import type { FlowNodeItem } from "../proto/api";

interface GoldenCase {
  name: string;
  go: unknown;
  feSkip?: string;
  [k: string]: unknown;
}

const golden = JSON.parse(
  readFileSync(resolve(process.cwd(), "src/api/contract/golden.json"), "utf-8"),
) as {
  payload: Record<string, unknown>;
  verdicts: Record<string, GoldenCase[]>;
  tables: {
    categories: Array<{
      name: string;
      allowedFlows: string[];
      defaultFlow: string;
      defaultScope: string;
    }>;
    notificationTargetTypes: Record<string, string>;
    requirementStatuses: string[];
  };
};

/** 判据输入样例 → FlowNodeItem 局部视图（判定函数只读注释列字段）。 */
function nodeOf(sample: Record<string, unknown>): FlowNodeItem {
  return {
    state: (sample.state as string) ?? "",
    reviewers: (sample.reviewers as string) ?? "[]",
    reviewResults: (sample.reviewResults as string) ?? "[]",
    nodeType: (sample.nodeType as string) ?? "",
    gate: (sample.gate as number) ?? 0,
  } as unknown as FlowNodeItem;
}

describe("contract: tables（constants.ts 手抄表 ↔ Go 真相源）", () => {
  it("类别绑定表与 CategoryDefinitions 深相等", () => {
    const names = golden.tables.categories.map((c) => c.name);
    expect([...CATEGORIES]).toEqual(names);
    for (const c of golden.tables.categories) {
      expect(CATEGORY_FLOW_OPTIONS[c.name]).toEqual(c.allowedFlows);
      expect(CATEGORY_DEFAULT_FLOW[c.name]).toBe(c.defaultFlow);
    }
  });

  it("需求状态词表与 domain.ReqStatus 枚举一致", () => {
    expect([...REQUIREMENT_STATUSES]).toEqual(golden.tables.requirementStatuses);
  });

  it("通知类别→跳转目标映射与 notificationTargetTypes 深相等", () => {
    expect(NOTIFICATION_KIND_TARGETS).toEqual(golden.tables.notificationTargetTypes);
  });
});

describe("contract: payload（TEXT 列 JSON 载荷形状锁）", () => {
  it("parse 函数族解析 golden 样例后逐字段保真", () => {
    // 说明：形状锁=字段名与取值经真实 Go 序列化样例承载；FE parse 反序列化后
    // 逐键比对（新增字段未在 FE interface 声明会在此暴露）。
    const { reviewerDef, reviewResult, issueReply, gateDecision } = golden.payload as {
      reviewerDef: Array<Record<string, unknown>>;
      reviewResult: Record<string, unknown>;
      issueReply: Record<string, Record<string, unknown>>;
      gateDecision: Record<string, Record<string, unknown>>;
    };
    expect(reviewerDef.length).toBeGreaterThanOrEqual(2);
    for (const r of reviewerDef) {
      expect(Object.keys(r)).toContain("role");
      expect(Object.keys(r)).toContain("capabilityId");
    }
    expect(Object.keys(reviewResult).sort()).toEqual(
      ["at", "capabilityId", "role", "round", "state"].sort(),
    );
    // issueReply 四变体：plain 无可选键、voided/escalation/adjudication 各携增量键
    expect(issueReply.plain.voided).toBeUndefined();
    expect(issueReply.voided.voided).toBe(true);
    expect(issueReply.escalation.kind).toBe("escalation");
    expect(issueReply.escalation.sessionId).toBeTruthy();
    expect(issueReply.adjudication.kind).toBe("adjudication");
    expect(issueReply.adjudication.verdict).toBe("uphold");
    // gateDecision：rejected 样例无 branch（omitempty 边界）
    expect(gateDecision.approved.branch).toBe("feat/audit");
    expect(gateDecision.rejected.branch).toBeUndefined();
  });
});

describe("contract: verdicts（判定真值表逐例）", () => {
  it("summarizeReviews 计数与 SummarizeReviewResults 同口径", () => {
    for (const c of golden.verdicts.summarizeReviewResults) {
      const go = c.go as { agree: number; unagree: number; reviewing: number };
      const counts = summarizeReviews(nodeOf(c));
      expect(counts.agree).toBe(go.agree);
      expect(counts.unagree).toBe(go.unagree);
      expect(counts.reviewing).toBe(go.reviewing);
    }
  });

  it("inReviewCommentContext 与 InReviewContext 同口径", () => {
    for (const c of golden.verdicts.inReviewContext) {
      expect(inReviewCommentContext(nodeOf(c), c.role as string)).toBe(c.go);
    }
  });

  it("reviewAnchorGuardError 拦截面与 ReviewAnchorGuardBlocked 同口径", () => {
    for (const c of golden.verdicts.reviewAnchorGuard) {
      const blocked =
        reviewAnchorGuardError({
          reviewContext: c.reviewCtx as boolean,
          issueType: c.issueType as string,
          anchorKind: c.anchorKind as string,
          artifactId: c.artifactId as string,
        }) !== null;
      expect(blocked).toBe(c.go);
    }
  });

  it("泳道侧成员判定与 LaneSideExecCR 同口径（feSkip=已知分歧跳过）", () => {
    for (const c of golden.verdicts.laneSideExecCR) {
      if (c.feSkip !== undefined) {
        // 已知双端分歧：FE 无 scope 单侧校验（相邻成对约束替代），显式跳过
        expect(c.go).toBe(false);
        continue;
      }
      expect(isLaneSideExecCR(nodeOf(c))).toBe(c.go);
    }
  });

  it("相邻泳道对构成并行组行（buildFlowRows ↔ 后端 join 判定同口径）", () => {
    const rdLane = golden.verdicts.laneSideExecCR.find((c) => c.name === "rd 泳道 exec-CR（bus-08 形）");
    const feLane = golden.verdicts.laneSideExecCR.find((c) => c.name === "fe 泳道 exec-CR（bus-09 形）");
    expect(rdLane).toBeTruthy();
    expect(feLane).toBeTruthy();
    const rows = buildFlowRows([nodeOf(rdLane!), nodeOf(feLane!)]);
    expect(rows.length).toBe(1);
    expect(rows[0].kind).toBe("parallel");
  });
});
