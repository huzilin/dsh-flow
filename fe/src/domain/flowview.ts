/**
 * 流程树/节点详情视图模型（纯函数，禁 IO）。
 * 解析口径与后端同源：reviewers=[{"role","capabilityId","scopeExpr"?}]、
 * review_results=[{"role","capabilityId","state","round","at"}]、
 * replies=[{"replyId","by","roleId","text","voided"?,"at"}]、
 * last_decision={"verdict","reason","at","branch"?}——rd/repo/payload.go 单一口径镜像，
 * 字段漂移由后端 Go struct 契约兜底（FE 侧仅展示）。
 */
import type { ArtifactItem, FlowNodeItem, IssueItem, MyTaskResp } from "../api/proto/api";

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
  /** 当前最大评审轮号（原型 P1 卡②：轮次≥10 未收敛 → 「疑似卡死」黄色高亮） */
  maxRound: number;
}

/** 评审轮聚合计数（与 rd/domain SummarizeReviewResults 同口径：全轮次累计）。 */
export function summarizeReviews(node: FlowNodeItem): ReviewCounts {
  const results = parseReviewResults(node.reviewResults);
  const counts: ReviewCounts = {
    agree: 0,
    unagree: 0,
    reviewing: 0,
    seats: parseReviewers(node.reviewers).length,
    maxRound: 0,
  };
  for (const r of results) {
    if (r.state === "agree") counts.agree++;
    else if (r.state === "unagree") counts.unagree++;
    else if (r.state === "reviewing") counts.reviewing++;
    if (r.round > counts.maxRound) counts.maxRound = r.round;
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

/**
 * 节点未闭合 issue 数（评审药丸 open 数；unresolved=state 非 resolved）。
 * 09-12 裁决 overridden 态废除，override 动作置 resolved。
 */
export function countOpenIssues(issues: IssueItem[]): number {
  return issues.filter((i) => i.state !== "resolved").length;
}

/**
 * BUG-011 评审产物分组（评审节点产物区两段呈现的视图模型）：
 * - reviewed=被评审产物（上游）：消费 wb_my_task relatedArtifacts——rd 侧已把直接上游
 *   produce 节点已登记产物置前（rd/service/flow.go loadRelatedArtifacts 单一口径），
 *   此处按快照反查归属，取「非本节点」段，顺序保持 rd 装配序（评审者先读被评审文档）；
 * - own=本节点产物（快照本节点全集，口径不变）。
 * 评审型节点（node_type=review）才分组；my_task 未加载（in-flight）时 reviewed 空集
 * 由 UI 呈现加载态。relatedArtifacts 中快照查不到/实为本节点的条目跳过（防御）。
 */
export interface ReviewArtifactGroups {
  reviewed: ArtifactItem[];
  own: ArtifactItem[];
}

export function reviewArtifactGroups(
  node: FlowNodeItem,
  myTask: MyTaskResp | undefined,
  artifacts: ArtifactItem[],
): ReviewArtifactGroups {
  const own = artifacts.filter((a) => a.nodeId === node.nodeId);
  if (node.nodeType !== "review" || myTask === undefined) {
    return { reviewed: [], own };
  }
  const byId = new Map(artifacts.map((a) => [a.artifactId, a]));
  const ownIds = new Set(own.map((a) => a.artifactId));
  const reviewed: ArtifactItem[] = [];
  for (const rel of myTask.relatedArtifacts) {
    if (ownIds.has(rel.artifactId)) continue;
    const snap = byId.get(rel.artifactId);
    if (!snap || snap.nodeId === node.nodeId) continue;
    reviewed.push(snap);
  }
  return { reviewed, own };
}

/**
 * 评审上下文判定（BUG-013 锚产物强制前置条件；rd/service/comment.go inReviewContext
 * 同口径镜像）：节点相位 review/respond（评审表态与 owner 响应均为评审对话期），
 * 或发表角色出现在节点评审位声明（评审者身份——未激活节点上的评论同受约束）。
 */
export function inReviewCommentContext(node: FlowNodeItem, roleId: string): boolean {
  if (node.state === "review" || node.state === "respond") return true;
  return parseReviewers(node.reviewers).some((r) => r.role === roleId);
}

/**
 * BUG-013 发表前预校验（rd 新评论硬拦的 FE 前置同口径）：评审上下文的 issue 类
 * document 级锚必须携带 artifactId，否则返回引导文案（line/range/block 天然挂产物、
 * notification/discussion 类不受限）。返回 null=放行。
 */
export function reviewAnchorGuardError(input: {
  reviewContext: boolean;
  issueType: string;
  anchorKind: string;
  artifactId: string;
}): string | null {
  if (!input.reviewContext) return null;
  if (input.issueType !== "issue") return null;
  if (input.anchorKind !== "document") return null;
  if (input.artifactId === "") {
    return "评审上下文的 issue 评论必须锚定产物：请先选定产物（默认为被评审产物）再发表";
  }
  return null;
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

/** 行号是否落在选区内（高亮渲染；注记-L3 划词模型，start≤end 恒成立）。 */
export function inSelection(
  no: number,
  sel: { start: number; end: number } | null,
): boolean {
  if (!sel) return false;
  return no >= sel.start && no <= sel.end;
}

/** 锚点摘要文本（与后端 my_task relatedIssues.anchor 同口径："art-1 line 42" 家族）。
 *  BUG-013：传入 artifactLabelOf 时以产物显示名替代裸 id（评论展示挂产物名）；document
 *  级已挂产物显示「《名》全文」，未挂产物维持 "document"。 */
export function anchorSummary(
  issue: IssueItem,
  artifactLabelOf?: (artifactId: string) => string,
): string {
  const art = (id: string): string => {
    if (id === "") return "";
    const label = artifactLabelOf?.(id) ?? "";
    return label === "" ? id : label;
  };
  switch (issue.anchorKind) {
    case "line":
      return `${art(issue.anchorArtifactId)} line ${issue.anchorLine}`;
    case "range":
      return `${art(issue.anchorArtifactId)} line ${issue.anchorStartLine}-${issue.anchorEndLine}`;
    case "block":
      return `${art(issue.anchorArtifactId)} block ${issue.anchorBlockStart}-${issue.anchorBlockEnd}(${issue.anchorLang})`;
    default:
      return issue.anchorArtifactId !== "" ? `${art(issue.anchorArtifactId)}（全文）` : "document";
  }
}

/** 阅读器选区 → 锚点描述文案（底部输入框状态文案；注记-L3 划词模型）。 */
export function selectionLabel(
  sel: { start: number; end: number } | null,
): string {
  if (!sel) return "全文评论（document）";
  if (sel.start === sel.end) return `评论第 ${sel.start} 行`;
  return `评论第 ${sel.start}-${sel.end} 行区间`;
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

/** 节点内反对中的评审位清单（该位最新表态=unagree 才计入；后轮 agree 覆盖前轮 unagree）。 */
export function unagreeSeatsOf(node: FlowNodeItem): UnagreeSeat[] {
  const latest = new Map<string, UnagreeSeat & { state: string }>();
  for (const r of parseReviewResults(node.reviewResults)) {
    const key = `${r.role}:${r.capabilityId}`;
    const prev = latest.get(key);
    if (!prev || r.round > prev.round || (r.round === prev.round && r.at >= prev.at)) {
      latest.set(key, {
        role: r.role,
        capabilityId: r.capabilityId,
        round: r.round,
        at: r.at,
        state: r.state,
      });
    }
  }
  return [...latest.values()]
    .filter((s) => s.state === "unagree")
    .map(({ role, capabilityId, round, at }) => ({ role, capabilityId, round, at }));
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

/** 并行组行结构：泳道并排渲染的视图行（FlowTree 布局源）。
 *  single=常规单节点行；parallel=RD∥FE 开发组一行两列。 */
export type FlowRow =
  | { kind: "single"; node: FlowNodeItem }
  | { kind: "parallel"; rd: FlowNodeItem; fe: FlowNodeItem };

/** 泳道侧 exec-CR 成员判定（与后端 rd/service advance.isLaneSideExecCR 同口径镜像：
 * produce + gate=1 + 有评审位）。scope 单侧校验以相邻成对约束替代——seed 定稿矩阵
 * （business 40 定义节点）中相邻 exec-CR 对仅 RD∥FE 开发组，单测锁定；未来 seed
 * 若引入相邻 exec-CR 串行对，需后端快照出 parallel_group 标识替代本判定。 */
function isLaneSideExecCR(node: FlowNodeItem): boolean {
  if (node.nodeType !== "produce" || Number(node.gate) !== 1) return false;
  return parseReviewers(node.reviewers).length > 0;
}

/** 流程树行构建：node_order 序列扫描，相邻两个泳道侧 exec-CR 节点判定为并行组
 * （与后端 wb_advance join 判定同口径——相邻 rd 侧 × fe 侧；nodeOrder 小者为 rd 泳道
 * 居左）。非相邻不成组、三连不成组（宁缺勿滥，组语义由 seed 保证）。 */
export function buildFlowRows(nodes: FlowNodeItem[]): FlowRow[] {
  const sorted = [...nodes].sort((a, b) => Number(a.nodeOrder) - Number(b.nodeOrder));
  const rows: FlowRow[] = [];
  let i = 0;
  while (i < sorted.length) {
    const next = sorted[i + 1];
    if (
      next !== undefined &&
      isLaneSideExecCR(sorted[i]) &&
      isLaneSideExecCR(next)
    ) {
      rows.push({ kind: "parallel", rd: sorted[i], fe: next });
      i += 2;
    } else {
      rows.push({ kind: "single", node: sorted[i] });
      i += 1;
    }
  }
  return rows;
}
