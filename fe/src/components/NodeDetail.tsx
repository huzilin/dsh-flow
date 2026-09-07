import { useMemo, useState, type ReactNode } from "react";
import type {
  ArtifactItem,
  FlowNodeItem,
  IssueItem,
  MyTaskResp,
} from "../api/proto/api";
import { NODE_TYPE_LABELS } from "../api/constants";
import {
  countOpenIssues,
  deriveGateView,
  parseReviewers,
  parseReviewResults,
  summarizeReviews,
} from "../domain/flowview";
import { ArtifactReader, CommentThread } from "./ArtifactReader";
import { DecisionCard } from "./DecisionCard";
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
  requirementId,
}: {
  nodeId: string | null;
  node: FlowNodeItem | undefined;
  myTask: MyTaskResp | undefined;
  /** 快照 artifacts 全集（面板内按节点过滤） */
  artifacts: ArtifactItem[];
  issues: IssueItem[];
  requirementId: string;
}) {
  const [readerArtifactId, setReaderArtifactId] = useState<string | null>(null);

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
  const nodeArtifacts = artifacts.filter((a) => a.nodeId === nodeId);
  const readerArtifact = nodeArtifacts.find((a) => a.artifactId === readerArtifactId);

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
            {gateView === "agreed" && " · 待拍板"}
          </div>
          {node.enteredAt !== "0" && (
            <div className="muted small">进入执行：{formatTime(node.enteredAt)}</div>
          )}
        </PanelCard>

        <PanelCard no="②" title="评审状态" className="panel-review">
          <ReviewPills node={node} openCount={countOpenIssues(nodeIssues)} />
        </PanelCard>

        <PanelCard no="③" title="产物区" className="panel-artifacts">
          {nodeArtifacts.length === 0 ? (
            <div className="muted">本节点暂无登记产物。</div>
          ) : (
            <ul className="artifact-list">
              {nodeArtifacts.map((a) => {
                const anchorCount = nodeIssues.filter(
                  (i) => i.anchorArtifactId === a.artifactId,
                ).length;
                return (
                  <li key={a.artifactId} className="artifact-row">
                    {a.kind === "link" ? (
                      <a href={a.href} target="_blank" rel="noreferrer" className="artifact-link">
                        {a.label}
                      </a>
                    ) : (
                      <button
                        type="button"
                        className="artifact-open"
                        onClick={() => setReaderArtifactId(a.artifactId)}
                      >
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
                        onClick={() => setReaderArtifactId(a.artifactId)}
                      >
                        💬{anchorCount} 阅读
                      </button>
                    )}
                    {a.note !== "" && <span className="muted small">{a.note}</span>}
                  </li>
                );
              })}
            </ul>
          )}
        </PanelCard>

        <PanelCard no="④" title="意见区" className="panel-issues">
          {nodeIssues.length === 0 ? (
            <div className="muted">
              本节点暂无评论。打开文档产物可选取行/区间发起锚点评论。
            </div>
          ) : (
            <CommentThread issues={nodeIssues} requirementId={requirementId} />
          )}
          <InFlightTasks myTask={myTask} />
        </PanelCard>

        <DecisionCard node={node} issues={nodeIssues} />
      </div>

      {readerArtifact && (
        <ArtifactReader
          requirementId={requirementId}
          nodeId={nodeId}
          artifact={readerArtifact}
          issues={nodeIssues}
          onClose={() => setReaderArtifactId(null)}
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

/** per-agent 表态药丸（评审位 × 最新表态；同轮同角色覆写取最后口径）。 */
function ReviewPills({ node, openCount }: { node: FlowNodeItem; openCount: number }) {
  const seats = useMemo(() => parseReviewers(node.reviewers), [node.reviewers]);
  if (seats.length === 0) {
    return <div className="muted">无评审位（审批行恒 '-'）。</div>;
  }
  return (
    <div>
      <div className="review-pills">
        {seats.map((s) => (
          <span
            key={`${s.role}:${s.capabilityId}`}
            className="review-seat"
            title={`${s.role} · ${s.capabilityId}${s.scopeExpr ? ` · ${s.scopeExpr}` : ""}`}
          >
            <ReviewStatePill state={latestStateOf(node, s.role, s.capabilityId)} />
          </span>
        ))}
      </div>
      <div className="muted small">未闭合评论 {openCount} 条 · 药丸主态=该评审位最新表态</div>
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
function InFlightTasks({ myTask }: { myTask: MyTaskResp | undefined }) {
  const tasks = myTask?.relatedTasks ?? [];
  if (tasks.length === 0) return null;
  return (
    <div className="inflight-tasks">
      <div className="muted small">在途任务（任务总线投影）：</div>
      <ul>
        {tasks.map((t) => (
          <li key={t.taskId} className="muted small">
            {t.taskId} · {t.actionId} · {t.status}
          </li>
        ))}
      </ul>
    </div>
  );
}
