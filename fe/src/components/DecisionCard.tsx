import type { ReactNode } from "react";
import type { FlowNodeItem, IssueItem } from "../api/proto/api";
import { countOpenIssues, deriveGateView, parseLastDecision, summarizeReviews } from "../domain/flowview";
import { formatTime } from "../pages/RequirementPoolPage";

/**
 * 门禁/拍板卡（user_gate=1 节点；五区分栏之 ⑤）。
 * 本工单只做「进度只读态」：
 * - 评审轮进行中：表态 N/M + agree/unagree/reviewing 计数 + 工单统计（open/待复核/已解决）；
 * - agreed（评审全 agree）：待拍板提示；
 * - 已裁决：last_decision verdict 徽标（approved/rejected）+ 拍板理由/分支/时间。
 * 三态切换（agreed 迎宾呈现：汇总+通过/打回按钮组+跳转）属 T16——
 * 接缝：`actions` 插槽 props，T16 注入按钮组即可，数据派生复用本组件。
 */
export function DecisionCard({
  node,
  issues,
  actions,
}: {
  node: FlowNodeItem;
  issues: IssueItem[];
  /** T16 接缝：agreed 态按钮组插槽（本工单恒不传） */
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
            <span className="decision-title">评审轮进度</span>
            <span>
              表态 {counts.agree + counts.unagree + counts.reviewing}/{counts.seats || "-"} · ✅同意{" "}
              {counts.agree} · ⛔反对 {counts.unagree} · ○待表态 {counts.reviewing}
            </span>
          </div>
          <div className="decision-issues muted">
            工单统计：待处理 {open} · 已处理待复核 {pendingConfirm} · 已解决 {resolved}
          </div>
          {gateView === "agreed" && !decision && (
            <div role="status" className="info-banner">
              评审轮已全体同意，等待 User 拍板（拍板操作于 T16 开放）。
            </div>
          )}
          {decision && (
            <div className={`decision-result decision-${decision.verdict}`}>
              <span className={`badge badge-verdict-${decision.verdict}`}>
                {decision.verdict === "approved" ? "已拍板通过" : "已拍板打回"}
              </span>
              {decision.reason && <span>理由：{decision.reason}</span>}
              {decision.branch && <span>分支：{decision.branch}</span>}
              <span className="muted">{formatTime(String(decision.at))}</span>
            </div>
          )}
          {gateView === "" && !decision && (
            <p className="muted">评审轮进行中——表态未齐或存在反对（澄清循环），拍板置灰。</p>
          )}
          {actions}
        </div>
      )}
    </section>
  );
}
