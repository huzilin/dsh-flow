import { useEffect, useMemo, useState, type ReactNode } from "react";
import type {
  ArtifactItem,
  FlowNodeItem,
  IssueItem,
  MyTaskResp,
  SessionItem,
} from "../api/proto/api";
import { NODE_TYPE_LABELS, SESSION_STATUS_LABELS, WEB_USER_ROLE } from "../api/constants";
import {
  deriveGateView,
  inReviewCommentContext,
  parseReviewers,
  parseReviewResults,
  reviewArtifactGroups,
  summarizeReviews,
} from "../domain/flowview";
import { ArtifactReader, CommentThread } from "./ArtifactReader";
import { ConflictCard } from "./ConflictCard";
import { DecisionActions, DecisionCard } from "./DecisionCard";
import { PhaseBadge, ReviewStatePill } from "./Badges";
import { formatTime } from "../pages/RequirementPoolPage";

/**
 * 节点详情五区分栏（ui-interaction-final §1.2 口径落位）：
 * ① 状态卡（双行：执行行=相位徽标+当前挂载动作名；审批行=表态计数）
 * ② 评审状态行（per-agent 表态药丸 + open 数；轮内完整时间线视图属暂缓项 P2）
 * ③ 产物区（文件行+评论数+外链；点击开阅读器）
 * ④ 意见区（评论列表 + escalate 标记 + 在途任务投影；冲突卡聚合属 T16）
 * ⑤ 门禁/拍板卡（本工单只读态；T16 接缝见 DecisionCard）
 * 数据源：聚合快照（flows/flow_nodes/artifacts/issues）+ my_task（activeActions/relatedTasks），
 * 不自拼接口。
 */
export function NodeDetail({
  nodeId,
  node,
  myTask,
  artifacts,
  issues,
  sessions = [],
  requirementId,
  focusIssueId = "",
  focusTaskId = "",
}: {
  nodeId: string | null;
  node: FlowNodeItem | undefined;
  myTask: MyTaskResp | undefined;
  /** 快照 artifacts 全集（面板内按节点过滤） */
  artifacts: ArtifactItem[];
  issues: IssueItem[];
  /** 快照 sessions 全集（⑥ 会话 tab 按节点过滤；§6 N5） */
  sessions?: SessionItem[];
  requirementId: string;
  /** 收件箱跳转定位（T16）：focus=issue 意见区锚点高亮 */
  focusIssueId?: string;
  /** 收件箱跳转定位（T16）：focus=task 任务区锚点高亮 */
  focusTaskId?: string;
}) {
  const [readerArtifactId, setReaderArtifactId] = useState<string | null>(null);

  // 跳转锚点：滚动到首个高亮目标（意见区评论卡 / 在途任务行）
  useEffect(() => {
    if (focusIssueId === "" && focusTaskId === "") return;
    const el = document.querySelector('[data-focus-flash="1"]');
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
  }, [focusIssueId, focusTaskId, nodeId]);

  if (!nodeId || !node) {
    return (
      <section className="node-detail placeholder-note">
        点击流程树节点查看详情（需求尚未进入编排时节点相位为「未进入编排」，详情同样可查看）。
      </section>
    );
  }

  const counts = summarizeReviews(node);
  const gateView = deriveGateView(node);
  const nodeIssues = issues.filter((i) => i.nodeId === nodeId);
  // 冲突卡聚合（T16）：escalate=1 的 issue 提到意见区顶部，其余按时间线呈现
  const escalatedIssues = nodeIssues.filter((i) => i.escalate === "1");
  const normalIssues = nodeIssues.filter((i) => i.escalate !== "1");
  // BUG-011：评审节点产物分两段——被评审产物（上游，my_task relatedArtifacts 置前段）+
  // 本节点产物；非评审节点维持本节点全集（视图模型见 flowview.reviewArtifactGroups）
  const reviewGroups = reviewArtifactGroups(node, myTask, artifacts);
  // BUG-013：评审上下文（相位 review/respond 或发表者在评审位）——阅读器评论锚产物强制
  const reviewCtx = inReviewCommentContext(node, WEB_USER_ROLE);
  // BUG-013：锚点摘要呈现产物名（artifactId→label 映射）
  const artifactLabels = Object.fromEntries(artifacts.map((a) => [a.artifactId, a.label]));
  // 快照 sessions 按当前节点过滤（requirementId 恒匹配本需求快照）
  const nodeSessions = sessions.filter((s) => s.nodeId === nodeId);
  // BUG-011：被评审产物不在本节点全集内，阅读器查快照全量
  const readerArtifact = artifacts.find((a) => a.artifactId === readerArtifactId);

  return (
    <section className="node-detail" data-node-id={nodeId} aria-label="节点详情五区分栏">
      <div className="node-detail-head">
        <h3>
          {node.nodeOrder}. {node.nodeKey}
          <span className="muted req-id">
            {" "}
            {node.nodeId} · {node.defNodeId}
          </span>
        </h3>
        <span className="node-detail-badges">
          {node.nodeType !== "" && (
            <span className="badge badge-category">
              {NODE_TYPE_LABELS[node.nodeType] ?? node.nodeType}
            </span>
          )}
          {node.gate === "1" && <span className="badge badge-gate">gate 门禁</span>}
          {node.userGate === "1" && <span className="badge badge-usergate">user_gate 拍板位</span>}
          {node.reopenSource !== "" && (
            <span className="badge badge-reopen">重开：{node.reopenSource}</span>
          )}
          {node.featureBranch !== "" && (
            <span className="badge badge-branch">分支 {node.featureBranch}</span>
          )}
          {/* 会话通道占位（ui-interaction-final §1.5：按需会话通道，本期不接 DSH） */}
          <button
            type="button"
            disabled
            title="DSH 嵌入期开放"
            className="session-btn"
            aria-label="会话通道（DSH 嵌入期开放）"
          >
            💬 会话找负责人
          </button>
        </span>
      </div>

      <div className="panels">
        <PanelCard no="①" title="状态卡" className="panel-status">
          <div className="status-line">
            <PhaseBadge phase={node.state} />
            <ExecActionText myTask={myTask} />
          </div>
          <div className="status-line muted">
            审批行：表态 {counts.agree + counts.unagree + counts.reviewing}/{counts.seats || "-"} ·
            ✅ {counts.agree} · ⛔ {counts.unagree} · ○ {counts.reviewing}
            {counts.maxRound > 0 && ` · 第 ${counts.maxRound} 轮`}
            {gateView === "agreed" && " · 待拍板"}
            {/* 卡死高亮已上评审药丸本体（ReviewPills is-stalled，一版 A3-2 形态） */}
          </div>
          {node.enteredAt !== "0" && (
            <div className="muted small">进入执行：{formatTime(node.enteredAt)}</div>
          )}
        </PanelCard>

        <PanelCard no="②" title="评审状态" className="panel-review">
          <ReviewPills
            node={node}
            issues={nodeIssues}
            requirementId={requirementId}
            artifactLabels={artifactLabels}
          />
        </PanelCard>

        <PanelCard no="③" title="产物区" className="panel-artifacts">
          <ArtifactGroups
            node={node}
            groups={reviewGroups}
            myTaskLoaded={myTask !== undefined}
            issues={nodeIssues}
            onOpen={setReaderArtifactId}
          />
        </PanelCard>

        <PanelCard no="④" title="意见区" className="panel-issues">
          {escalatedIssues.length > 0 && (
            <div className="conflict-stack" data-testid="conflict-stack">
              {escalatedIssues.map((i) => (
                <ConflictCard
                  key={i.issueId}
                  requirementId={requirementId}
                  issue={i}
                  node={node}
                />
              ))}
            </div>
          )}
          {normalIssues.length === 0 ? (
            escalatedIssues.length > 0 ? (
              <div className="muted">其余评论见冲突卡与产物阅读器。</div>
            ) : (
              <div className="muted">
                本节点暂无评论。打开文档产物可选取行/区间发起锚点评论。
              </div>
            )
          ) : (
            <CommentThread
              issues={normalIssues}
              requirementId={requirementId}
              highlightIssueId={focusIssueId}
              artifactLabels={artifactLabels}
            />
          )}
          <InFlightTasks myTask={myTask} focusTaskId={focusTaskId} />
        </PanelCard>

        <DecisionCard
          node={node}
          issues={nodeIssues}
          artifacts={reviewGroups.own}
          actions={
            gateView === "agreed" ? (
              <DecisionActions requirementId={requirementId} node={node} />
            ) : undefined
          }
        />

        <PanelCard no="⑥" title={`会话（${nodeSessions.length}）`} className="panel-sessions">
          <SessionTab sessions={nodeSessions} nodeId={nodeId} />
        </PanelCard>
      </div>

      {readerArtifact && (
        <ArtifactReader
          requirementId={requirementId}
          nodeId={nodeId}
          artifact={readerArtifact}
          issues={nodeIssues}
          onClose={() => setReaderArtifactId(null)}
          reviewContext={reviewCtx}
          artifactLabels={artifactLabels}
        />
      )}
    </section>
  );
}

function PanelCard({
  no,
  title,
  className = "",
  children,
}: {
  no: string;
  title: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={`panel-card ${className}`}>
      <h4>
        {no} {title}
      </h4>
      {children}
    </section>
  );
}

/**
 * ③ 产物区分组呈现（BUG-011）：评审型节点分「被评审产物（上游）」与「本节点产物」
 * 两段（上游=my_task relatedArtifacts 置前段，点开即读，零跳转）；非评审节点维持
 * 本节点全集单段（口径不变）。
 */
function ArtifactGroups({
  node,
  groups,
  myTaskLoaded,
  issues,
  onOpen,
}: {
  node: FlowNodeItem;
  groups: ReturnType<typeof reviewArtifactGroups>;
  /** my_task 是否已加载（未加载时上游段呈加载态而非「无产物」误报） */
  myTaskLoaded: boolean;
  issues: IssueItem[];
  onOpen: (artifactId: string) => void;
}) {
  if (node.nodeType !== "review") {
    if (groups.own.length === 0) {
      return <div className="muted">本节点暂无登记产物。</div>;
    }
    return <ArtifactRowList items={groups.own} issues={issues} onOpen={onOpen} />;
  }
  return (
    <>
      <div className="artifact-group-label">被评审产物（上游）</div>
      {groups.reviewed.length === 0 ? (
        <div className="muted small">
          {myTaskLoaded ? "上游节点暂无被评审产物。" : "被评审产物加载中…"}
        </div>
      ) : (
        <ArtifactRowList items={groups.reviewed} issues={issues} onOpen={onOpen} />
      )}
      <div className="artifact-group-label">本节点产物</div>
      {groups.own.length === 0 ? (
        <div className="muted small">本节点暂无登记产物。</div>
      ) : (
        <ArtifactRowList items={groups.own} issues={issues} onOpen={onOpen} />
      )}
    </>
  );
}

/** 产物行列表（③ 区行渲染：文件行 + 锚点评论数 + 外链/阅读器入口）。 */
function ArtifactRowList({
  items,
  issues,
  onOpen,
}: {
  items: ArtifactItem[];
  issues: IssueItem[];
  onOpen: (artifactId: string) => void;
}) {
  return (
    <ul className="artifact-list">
      {items.map((a) => {
        const anchorCount = issues.filter((i) => i.anchorArtifactId === a.artifactId).length;
        return (
          <li key={a.artifactId} className="artifact-row">
            {a.kind === "link" ? (
              <a href={a.href} target="_blank" rel="noreferrer" className="artifact-link">
                {a.label}
              </a>
            ) : (
              <button type="button" className="artifact-open" onClick={() => onOpen(a.artifactId)}>
                {a.label}
              </button>
            )}
            <span className="badge badge-artifact-type">{a.productType}</span>
            {a.kind === "link" ? (
              <span className="badge badge-kind-link">外链</span>
            ) : (
              <button
                type="button"
                className="artifact-open-sub"
                onClick={() => onOpen(a.artifactId)}
              >
                💬{anchorCount} 阅读
              </button>
            )}
            {a.note !== "" && <span className="muted small">{a.note}</span>}
          </li>
        );
      })}
    </ul>
  );
}

/** 执行行：当前挂载动作名（my_task activeActions 投影；无激活动作=提示）。 */
function ExecActionText({ myTask }: { myTask: MyTaskResp | undefined }) {
  const actions = myTask?.activeActions ?? [];
  if (!myTask) return <span className="muted small">（任务上下文加载中…）</span>;
  if (actions.length === 0) {
    return <span className="muted small">当前相位无 user 角色激活动作</span>;
  }
  return (
    <span className="exec-actions">
      {actions.map((a) => (
        <span key={`${a.kind}:${a.actionId}`} className="exec-action">
          {a.kind === "node" ? a.actionId : `${a.capabilityId}.${a.actionId}`}
          {a.skill !== "" && ` · skill ${a.skill}`}
        </span>
      ))}
    </span>
  );
}

/**
 * per-agent 表态药丸（评审位 × 最新表态；同轮同角色覆写取最后口径）。
 * OPT-002：药丸可点，展开该角色在本节点的意见区（issues+replies 过滤该角色——
 * 提评人归属角色或发表者角色命中即呈现；讨论发送走既有评论通道「回复」，此处只读）。
 */
function ReviewPills({
  node,
  issues,
  requirementId,
  artifactLabels,
}: {
  node: FlowNodeItem;
  issues: IssueItem[];
  requirementId: string;
  artifactLabels?: Record<string, string>;
}) {
  const seats = useMemo(() => parseReviewers(node.reviewers), [node.reviewers]);
  const [openSeatKey, setOpenSeatKey] = useState<string | null>(null);
  if (seats.length === 0) {
    return <div className="muted">无评审位（审批行恒 '-'）。</div>;
  }
  const openSeat = seats.find((s) => `${s.role}:${s.capabilityId}` === openSeatKey);
  const openSeatIssues =
    openSeat === undefined
      ? []
      : issues.filter((i) => i.reviewerRole === openSeat.role || i.roleId === openSeat.role);
  return (
    <div>
      <div className="review-pills">
        {seats.map((s) => {
          const key = `${s.role}:${s.capabilityId}`;
          const expanded = key === openSeatKey;
          // 药丸本体信息密度（一版 client.js:724 形态回归——inventory A3-2/B-27）：
          // N 未解决 Number badge + 轮次≥10 卡死红边（不再只放药丸行下方文字）。
          const seatIssues = issues.filter((i) => i.reviewerRole === s.role || i.roleId === s.role);
          const openN = seatIssues.filter((i) => i.state !== "resolved").length;
          const seatRound = parseReviewResults(node.reviewResults)
            .filter((r) => r.role === s.role && r.capabilityId === s.capabilityId)
            .reduce((max, r) => Math.max(max, r.round), 0);
          const stalled = seatRound >= 10 && openN > 0;
          return (
            <button
              key={key}
              type="button"
              className={`review-seat${expanded ? " is-open" : ""}${stalled ? " is-stalled" : ""}`}
              title={`${s.role} · ${s.capabilityId}${s.scopeExpr ? ` · ${s.scopeExpr}` : ""}（点击展开该角色意见）`}
              aria-label={`${s.role}（${s.capabilityId}）表态药丸${openN > 0 ? `，${openN} 未解决` : ""}${stalled ? "，疑似卡死" : ""}`}
              aria-expanded={expanded}
              onClick={() => setOpenSeatKey(expanded ? null : key)}
            >
              <ReviewStatePill state={latestStateOf(node, s.role, s.capabilityId)} />
              {openN > 0 && (
                <span className="badge pill-open-count" title="该评审位未闭合评论数">
                  {openN} 未解决
                </span>
              )}
              {stalled && (
                <span className="badge pill-round" title="评审轮次≥10 未收敛，疑似卡死">
                  轮 {seatRound}
                </span>
              )}
            </button>
          );
        })}
      </div>
      <div className="muted small">药丸主态=该评审位最新表态 · N 未解决/卡死高亮在药丸本体 · 点击药丸看该角色意见</div>
      {openSeat !== undefined && (
        <div className="review-seat-thread" data-seat={openSeat.role}>
          {openSeatIssues.length === 0 ? (
            <div className="muted small">
              {openSeat.role} 暂无意见。发表意见请打开被评审产物，在阅读器内锚定产物评论
              （评审上下文必须挂产物）。
            </div>
          ) : (
            <CommentThread
              issues={openSeatIssues}
              requirementId={requirementId}
              artifactLabels={artifactLabels}
            />
          )}
          <div className="muted small">
            参与讨论：在意见区/产物阅读器对应评论使用「回复」（讨论发送走既有评论通道）。
          </div>
        </div>
      )}
    </div>
  );
}

/** 评审位最新表态（round 大者胜；同轮 at 后者胜——与 wb_review 覆写语义一致）。 */
function latestStateOf(node: FlowNodeItem, role: string, capabilityId: string): string {
  let state = "reviewing";
  let bestRound = -1;
  let bestAt = -1;
  for (const r of parseReviewResults(node.reviewResults)) {
    if (r.role === role && r.capabilityId === capabilityId) {
      if (r.round > bestRound || (r.round === bestRound && r.at >= bestAt)) {
        state = r.state;
        bestRound = r.round;
        bestAt = r.at;
      }
    }
  }
  return state;
}

/** 在途任务摘要（my_task relatedTasks；任务区并入意见区下方，避免第六栏）。 */
function InFlightTasks({ myTask, focusTaskId }: { myTask: MyTaskResp | undefined; focusTaskId: string }) {
  const tasks = myTask?.relatedTasks ?? [];
  if (tasks.length === 0) return null;
  return (
    <div className="inflight-tasks">
      <div className="muted small">在途任务（任务总线投影）：</div>
      <ul>
        {tasks.map((t) => (
          <li
            key={t.taskId}
            className={`muted small${focusTaskId === t.taskId ? " focus-flash" : ""}`}
            data-focus-flash={focusTaskId === t.taskId ? "1" : undefined}
          >
            {t.taskId} · {t.actionId} · {t.status}
          </li>
        ))}
      </ul>
    </div>
  );
}

/**
 * ⑥ 会话 tab（ui-interaction-final §6 N5 / 原型 P1 卡⑥）：历史/进行中 subagent
 * 对话列表（sessions 按 node 关联）。条目=绑定角色+状态+时间；点击打开会话视图
 * 属 plugin 泛用接口（DSH 嵌入期，本期条目只读呈现，不跳转）。
 */
function SessionTab({ sessions, nodeId }: { sessions: SessionItem[]; nodeId: string }) {
  if (sessions.length === 0) {
    return (
      <div className="muted small">
        暂无会话绑定（按需会话通道与裁决 session 在 DSH 嵌入期开放，节点 {nodeId}）。
      </div>
    );
  }
  return (
    <ul className="session-list">
      {sessions.map((s) => (
        <li key={s.sessionId} className="session-row">
          <span
            className={`badge session-status-${s.status}`}
            title={`sessions.status=${s.status}`}
          >
            {SESSION_STATUS_LABELS[s.status] ?? s.status}
          </span>
          <span className="session-role">{s.roleId}</span>
          <span className="muted small session-id">{s.sessionId}</span>
          <span className="muted small">{formatTime(s.updatedAt)}</span>
        </li>
      ))}
      <li className="muted small">点击条目打开会话视图：plugin 泛用接口（DSH 嵌入期开放）。</li>
    </ul>
  );
}
