/**
 * 流程树/节点详情视图模型（纯函数，禁 IO）。
 * 解析口径与后端同源：reviewers=[{"role","capabilityId","scopeExpr"?}]、
 * review_results=[{"role","capabilityId","state","round","at"}]、
 * replies=[{"replyId","by","roleId","text","voided"?,"at"}]、
 * last_decision={"verdict","reason","at","branch"?}——rd/repo/payload.go 单一口径镜像，
 * 字段漂移由后端 Go struct 契约兜底（FE 侧仅展示）。
 */
import type { FlowNodeItem, IssueItem } from "../api/proto/api";

/** JSON 文本安全解析（TEXT 列同构载荷；损坏/空返回缺省值不抛——展示层降级）。 */
export function parseJSONText<T>(text: string, fallback: T): T {
  if (text === "") return fallback;
  try {
    const v = JSON.parse(text);
    return v ?? fallback;
  } catch {
    return fallback;
  }
}

export interface ReviewerDef {
  role: string;
  capabilityId: string;
  scopeExpr?: string;
}

export interface ReviewResult {
  role: string;
  capabilityId: string;
  state: "reviewing" | "agree" | "unagree" | string;
  round: number;
  at: number;
}

export interface IssueReply {
  replyId: string;
  by: string;
  roleId: string;
  text: string;
  voided?: boolean;
  at: number;
  /** T12 条目类型：空=普通回复 / escalation=冲突上升条目 / adjudication=仲裁裁决条目 */
  kind?: string;
  /** 仲裁结论（仅 kind=adjudication）：uphold | override */
  verdict?: string;
  /** 关联裁决 session 业务 id（kind=escalation/adjudication 携带） */
  sessionId?: string;
}

export interface LastDecision {
  verdict: string;
  reason?: string;
  at: number;
  branch?: string;
}

export function parseReviewers(json: string): ReviewerDef[] {
  return parseJSONText<ReviewerDef[]>(json, []);
}

export function parseReviewResults(json: string): ReviewResult[] {
  return parseJSONText<ReviewResult[]>(json, []);
}

export function parseReplies(json: string): IssueReply[] {
  return parseJSONText<IssueReply[]>(json, []);
}

export function parseLastDecision(json: string): LastDecision | null {
  if (json === "") return null;
  return parseJSONText<LastDecision | null>(json, null);
}

export interface ReviewCounts {
  agree: number;
  unagree: number;
  reviewing: number;
  /** 评审位总数（M；无评审位=0，UI 恒 '-'） */
  seats: number;
}

/** 评审轮聚合计数（与 rd/domain SummarizeReviewResults 同口径：全轮次累计）。 */
export function summarizeReviews(node: FlowNodeItem): ReviewCounts {
  const results = parseReviewResults(node.reviewResults);
  const counts: ReviewCounts = {
    agree: 0,
    unagree: 0,
    reviewing: 0,
    seats: parseReviewers(node.reviewers).length,
  };
  for (const r of results) {
    if (r.state === "agree") counts.agree++;
    else if (r.state === "unagree") counts.unagree++;
    else if (r.state === "reviewing") counts.reviewing++;
  }
  return counts;
}

/** 门禁视图态：approved/rejected=已裁决（last_decision）；agreed=评审全 agree 待拍板；
 *  ""=无门禁语义（非 gate 节点或评审未齐）。agreed 判定：有评审位且无待表态无反对且有同意。 */
export type GateView = "approved" | "rejected" | "agreed" | "";

export function deriveGateView(node: FlowNodeItem): GateView {
  const decision = parseLastDecision(node.lastDecision);
  if (decision) {
    return decision.verdict === "approved" ? "approved" : decision.verdict === "rejected" ? "rejected" : "";
  }
  const counts = summarizeReviews(node);
  if (counts.seats > 0 && counts.reviewing === 0 && counts.unagree === 0 && counts.agree > 0) {
    return "agreed";
  }
  return "";
}

/** 节点未闭合 issue 数（评审药丸 open 数；unresolved=state 非 resolved/overridden）。 */
export function countOpenIssues(issues: IssueItem[]): number {
  return issues.filter((i) => i.state !== "resolved" && i.state !== "overridden").length;
}

export interface ContentLine {
  /** 1 起行号 */
  no: number;
  text: string;
}

/** 正文按行切分（阅读器行号渲染源；\n 分割，尾行保真）。 */
export function splitContentLines(content: string): ContentLine[] {
  if (content === "") return [];
  const parts = content.split("\n");
  if (parts.length > 1 && parts[parts.length - 1] === "") parts.pop();
  return parts.map((text, i) => ({ no: i + 1, text }));
}

/** 围栏代码块扫描（``` 或 ~~~ 起止；返回块区间与 lang，供 block 锚点选块）。 */
export interface FenceBlock {
  startLine: number;
  endLine: number;
  lang: string;
}

export function scanFenceBlocks(lines: ContentLine[]): FenceBlock[] {
  const blocks: FenceBlock[] = [];
  let open: { startLine: number; lang: string; marker: string } | null = null;
  for (const line of lines) {
    const m = /^\s*(```|~~~)\s*(\S*)\s*$/.exec(line.text);
    if (!m) continue;
    const marker = m[1];
    const info = m[2] ?? "";
    if (!open) {
      open = { startLine: line.no, lang: info, marker };
    } else if (marker === open.marker) {
      blocks.push({ startLine: open.startLine, endLine: line.no, lang: open.lang });
      open = null;
    }
  }
  return blocks;
}

/** 阅读器懒加载分段（段大小按行数切；返回 [start,end] 1 起闭区间）。 */
export function chunkRanges(totalLines: number, chunkSize: number): Array<[number, number]> {
  const ranges: Array<[number, number]> = [];
  for (let start = 1; start <= totalLines; start += chunkSize) {
    ranges.push([start, Math.min(start + chunkSize - 1, totalLines)]);
  }
  return ranges;
}

/** 行号是否落在选区内（高亮渲染）。 */
export function inSelection(
  no: number,
  sel: { kind: "line" | "range" | "block"; start: number; end: number } | null,
): boolean {
  if (!sel) return false;
  return no >= sel.start && no <= sel.end;
}

/** 锚点摘要文本（与后端 my_task relatedIssues.anchor 同口径："art-1 line 42" 家族）。 */
export function anchorSummary(issue: IssueItem): string {
  const art = issue.anchorArtifactId;
  switch (issue.anchorKind) {
    case "line":
      return `${art} line ${issue.anchorLine}`;
    case "range":
      return `${art} line ${issue.anchorStartLine}-${issue.anchorEndLine}`;
    case "block":
      return `${art} block ${issue.anchorBlockStart}-${issue.anchorBlockEnd}(${issue.anchorLang})`;
    default:
      return "document";
  }
}

/** 阅读器选区 → 锚点描述文案（底部输入框状态文案）。 */
export function selectionLabel(
  sel: { kind: "document" | "line" | "range" | "block"; start: number; end: number; lang?: string } | null,
): string {
  if (!sel || sel.kind === "document") return "全文评论（document）";
  if (sel.kind === "line") return `评论第 ${sel.start} 行`;
  if (sel.kind === "range") return `评论第 ${sel.start}-${sel.end} 行区间`;
  return `评论代码块 ${sel.start}-${sel.end} 行（${sel.lang || "未标注语言"}）`;
}

/** 提评者口径（confirm/reopen 操作者约束：reviewerRole 优先，缺省 author）。 */
export function issueReviewerOf(issue: IssueItem): string {
  return issue.reviewerRole !== "" ? issue.reviewerRole : issue.author;
}

/** 反对表态的评审位（正反对照「正方」数据源：该位最新表态=unagree）。 */
export interface UnagreeSeat {
  role: string;
  capabilityId: string;
  round: number;
  at: number;
}

/** 节点内反对中的评审位清单（同位多轮取最新一条 unagree）。 */
export function unagreeSeatsOf(node: FlowNodeItem): UnagreeSeat[] {
  const latest = new Map<string, UnagreeSeat>();
  for (const r of parseReviewResults(node.reviewResults)) {
    if (r.state !== "unagree") continue;
    const key = `${r.role}:${r.capabilityId}`;
    const prev = latest.get(key);
    if (!prev || r.round > prev.round || (r.round === prev.round && r.at >= prev.at)) {
      latest.set(key, { role: r.role, capabilityId: r.capabilityId, round: r.round, at: r.at });
    }
  }
  return [...latest.values()];
}

/** 评审位最新表态清单（门禁汇总用；同轮同角色覆写取最后——与 wb_review 覆写语义一致）。 */
export function seatStatesOf(
  node: FlowNodeItem,
): Array<{ role: string; capabilityId: string; state: string }> {
  const latest = new Map<string, { role: string; capabilityId: string; state: string; round: number; at: number }>();
  for (const s of parseReviewers(node.reviewers)) {
    latest.set(`${s.role}:${s.capabilityId}`, {
      role: s.role,
      capabilityId: s.capabilityId,
      state: "reviewing",
      round: -1,
      at: -1,
    });
  }
  for (const r of parseReviewResults(node.reviewResults)) {
    const key = `${r.role}:${r.capabilityId}`;
    const prev = latest.get(key);
    if (!prev || r.round > prev.round || (r.round === prev.round && r.at >= prev.at)) {
      latest.set(key, { role: r.role, capabilityId: r.capabilityId, state: r.state, round: r.round, at: r.at });
    }
  }
  return [...latest.values()].map(({ role, capabilityId, state }) => ({ role, capabilityId, state }));
}

/**
 * rejected 拍板后的重入提示（T9 重入语义呈现——重入 UI 状态从快照相位读取）：
 * 节点仍在 review/respond 等非 exec 相位=「已打回，等待 exec 重入」；
 * 相位已回 exec=「已打回重入：exec 返工执行中」。
 */
export function rejectedReentryHint(node: FlowNodeItem): string {
  return node.state === "exec"
    ? "已打回重入：exec 返工执行中"
    : "已打回，等待 exec 重入";
}
