import { useState, type ReactNode } from "react";
import type { ArtifactItem, FlowNodeItem, IssueItem } from "../api/proto/api";
import {
  countOpenIssues,
  deriveGateView,
  parseLastDecision,
  rejectedReentryHint,
  seatStatesOf,
  summarizeReviews,
} from "../domain/flowview";
import { REVIEW_STATE_LABELS, WEB_USER_ROLE } from "../api/constants";
import { useDecisionMutation } from "../hooks/queries";
import { ApiError } from "../api/client";
import { formatTime } from "../pages/RequirementPoolPage";
import { AlertBanner } from "./AlertBanner";

/**
 * 门禁/拍板卡（user_gate=1 节点；五区分栏之 ⑤）三态（ui-interaction-final §1.3）：
 * - 进度态（评审轮进行中）：只读——表态 N/M（agree/unagree/reviewing 计数）+ 工单统计；
 * - agreed 迎宾呈现态：门禁汇总（各评审位结论 + resolved 摘要 + 产物链接 + 建议）
 *   + 「通过/打回」按钮组（`actions` 插槽，NodeDetail 注入 DecisionActions）+ 跳转流程树；
 * - 已裁决态：last_decision verdict 徽标（approved/rejected）+ 结论摘要；
 *   rejected 附 T9 重入语义提示（重入 UI 状态从快照相位读取）。
 * 双通道口径：收件箱 gate_pending=系统投递、迎宾态=owner 呈现（§1.3 联动行）。
 */
export function DecisionCard({
  node,
  issues,
  artifacts = [],
  actions,
}: {
  node: FlowNodeItem;
  issues: IssueItem[];
  /** 本节点产物（agreed 迎宾汇总呈现产物链接；产物阅读入口在 ③ 产物区） */
  artifacts?: ArtifactItem[];
  /** agreed 态按钮组插槽（NodeDetail 注入 DecisionActions） */
  actions?: ReactNode;
}) {
  const counts = summarizeReviews(node);
  const gateView = deriveGateView(node);
  const decision = parseLastDecision(node.lastDecision);
  const open = countOpenIssues(issues);
  const pendingConfirm = issues.filter((i) => i.state === "pending-confirm").length;
  const resolved = issues.filter((i) => i.state === "resolved" || i.state === "overridden").length;

  return (
    <section className="panel-card decision-card" data-gate-view={gateView}>
      <h4>⑤ 门禁 / 拍板卡</h4>
      {!node.userGate || node.userGate !== "1" ? (
        <p className="muted">非用户拍板节点（user_gate=0）：门禁由评审轮聚合机械判定。</p>
      ) : (
        <div className="decision-body">
          <div className="decision-progress">
            <span className="decision-title">
              {gateView === "agreed" ? "门禁汇总" : decision ? "拍板结论" : "评审轮进度"}
            </span>
            <span>
              表态 {counts.agree + counts.unagree + counts.reviewing}/{counts.seats || "-"} · ✅同意{" "}
              {counts.agree} · ⛔反对 {counts.unagree} · ○待表态 {counts.reviewing}
            </span>
          </div>
          <div className="decision-issues muted">
            工单统计：待处理 {open} · 已处理待复核 {pendingConfirm} · 已解决 {resolved}
          </div>

          {gateView === "agreed" && !decision && (
            <GateSummary node={node} issues={issues} artifacts={artifacts} />
          )}
          {gateView === "agreed" && !decision && actions}
          {gateView === "agreed" && !decision && (
            <div className="muted small">
              <a href="#flow-tree">↗ 跳转流程树核对相关节点</a>
            </div>
          )}
          {gateView === "" && !decision && (
            <p className="muted">评审轮进行中——表态未齐或存在反对（澄清循环），拍板置灰。</p>
          )}
          {decision && <DecisionResult node={node} decision={decision} />}
        </div>
      )}
    </section>
  );
}

/** agreed 迎宾汇总（§6.8 owner 报送内容投影）：各评审位结论 + resolved 摘要 + 产物链接 + 建议。 */
function GateSummary({
  node,
  issues,
  artifacts,
}: {
  node: FlowNodeItem;
  issues: IssueItem[];
  artifacts: ArtifactItem[];
}) {
  const seats = seatStatesOf(node);
  const settled = issues.filter((i) => i.state === "resolved" || i.state === "overridden");
  return (
    <div className="decision-summary" role="status">
      <h5>各评审位结论</h5>
      <ul className="decision-seats">
        {seats.map((s) => (
          <li key={`${s.role}:${s.capabilityId}`}>
            {s.role} · {s.capabilityId}：
            {REVIEW_STATE_LABELS[s.state] ?? s.state}
          </li>
        ))}
        {seats.length === 0 && <li className="muted">无评审位声明。</li>}
      </ul>
      <h5>工单处理摘要</h5>
      {settled.length === 0 ? (
        <p className="muted small">无已解决工单。</p>
      ) : (
        <ul className="decision-resolved">
          {settled.slice(0, 3).map((i) => (
            <li key={i.issueId} className="muted small">
              {i.issueId} · {i.state}
              {i.resolveNote !== "" && ` · ${i.resolveNote}`}
            </li>
          ))}
          {settled.length > 3 && (
            <li className="muted small">…其余 {settled.length - 3} 条见意见区</li>
          )}
        </ul>
      )}
      <h5>本节点产物</h5>
      {artifacts.length === 0 ? (
        <p className="muted small">无登记产物。</p>
      ) : (
        <ul className="decision-artifacts">
          {artifacts.map((a) => (
            <li key={a.artifactId}>
              {a.label} <span className="badge badge-artifact-type">{a.productType}</span>
            </li>
          ))}
        </ul>
      )}
      <p className="decision-advice">
        建议：评审位全体同意、无反对与待表态——建议通过。保留意见见各评审位结论与工单摘要。
      </p>
    </div>
  );
}

/** 已裁决态：verdict 徽标 + 结论摘要；rejected 附重入提示（快照相位读取）。 */
function DecisionResult({
  node,
  decision,
}: {
  node: FlowNodeItem;
  decision: NonNullable<ReturnType<typeof parseLastDecision>>;
}) {
  return (
    <div className={`decision-result decision-${decision.verdict}`} data-verdict={decision.verdict}>
      <span className={`badge badge-verdict-${decision.verdict}`}>
        {decision.verdict === "approved" ? "已拍板通过" : "已拍板打回"}
      </span>
      {decision.reason && <span>理由：{decision.reason}</span>}
      {decision.branch && <span>分支：{decision.branch}</span>}
      <span className="muted">{formatTime(String(decision.at))}</span>
      {decision.verdict === "rejected" && (
        <div role="status" className="info-banner">
          {rejectedReentryHint(node)}（打回走 T9 重入语义：revise/revive 由 exec 相位承接）
        </div>
      )}
    </div>
  );
}

/**
 * 拍板操作组（agreed 态 actions 插槽内容；Web 身份=WEB_USER_ROLE）：
 * - 通过：reason 可选；打回：reason 必填（reject 结论协议，后端 400 拦截）；
 * - branch：评审敲定分支名，可选；
 * - 409 改判冲突（IDEMPOTENT_CONFLICT）：提示 + 失效刷新（onSettled 已失效聚合快照，
 *   刷新后拍板卡自动切换到已裁决态）；同 verdict 重复拍板后端幂等 no-op。
 */
export function DecisionActions({
  requirementId,
  node,
}: {
  requirementId: string;
  node: FlowNodeItem;
}) {
  const decision = useDecisionMutation(requirementId);
  const [reason, setReason] = useState("");
  const [branch, setBranch] = useState("");

  const conflict =
    decision.error instanceof ApiError && decision.error.status === 409
      ? (decision.error as ApiError)
      : null;

  const submit = (verdict: "approved" | "rejected") =>
    decision.mutate(
      { nodeId: node.nodeId, verdict, reason, branch: branch || undefined },
      {
        onSuccess: () => {
          setReason("");
          setBranch("");
        },
      },
    );

  return (
    <div className="decision-form" data-testid="decision-actions">
      <div className="decision-form-buttons">
        <button
          type="button"
          className="primary"
          disabled={decision.isPending}
          onClick={() => submit("approved")}
        >
          ✅ 通过
        </button>
        <button
          type="button"
          className="danger"
          disabled={decision.isPending || reason.trim() === ""}
          title="打回必填理由（reject 结论协议）"
          onClick={() => submit("rejected")}
        >
          ⛔ 打回
        </button>
      </div>
      <textarea
        aria-label="拍板理由"
        placeholder="拍板理由（打回必填，通过可空）"
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        rows={2}
      />
      <input
        aria-label="评审敲定分支名"
        placeholder="评审敲定分支名（可选）"
        value={branch}
        onChange={(e) => setBranch(e.target.value)}
      />
      <span className="muted small">操作身份：{WEB_USER_ROLE}（user_gate 终审位）</span>
      {conflict && (
        <div role="alert" className="field-hint error">
          拍板冲突（409 {conflict.code}）：{conflict.message}——已刷新最新拍板状态，请基于当前结论操作。
        </div>
      )}
      {!conflict && decision.error && <AlertBanner error={decision.error} />}
      {decision.isSuccess && decision.data.noop && (
        <div role="status" className="field-hint">
          重复拍板幂等 no-op：该节点已按「{decision.data.verdict === "approved" ? "通过" : "打回"}」落定。
        </div>
      )}
    </div>
  );
}
