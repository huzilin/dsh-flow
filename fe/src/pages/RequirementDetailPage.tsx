import { Link, useParams } from "react-router-dom";
import { useRequirementQuery, useVersionsQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { PriorityBadge, ScopeBadge, StatusBadge } from "../components/Badges";
import { formatTime } from "./RequirementPoolPage";

/** 需求详情占位页（聚合快照基础版呈现；流程树/节点详情五区分栏由 T15 接管）。 */
export function RequirementDetailPage() {
  const { reqId = "" } = useParams();
  const detail = useRequirementQuery(reqId);
  const versions = useVersionsQuery();

  if (detail.isLoading || versions.isLoading) {
    return <section className="muted">加载中…</section>;
  }
  const errorNode = <AlertBanner error={detail.error} onRetry={() => void detail.refetch()} />;
  const item = detail.data?.requirement;
  if (!item) {
    return (
      <section>
        <Link to="/pool">← 返回需求池</Link>
        {errorNode}
      </section>
    );
  }

  const versionName = (() => {
    const v = detail.data?.version;
    if (v && v.versionId !== "") return `${v.name}（${v.versionId}）`;
    return item.versionId === "" ? "待分配" : item.versionId;
  })();

  return (
    <section>
      <Link to="/pool">← 返回需求池</Link>
      <div className="page-head">
        <h2>
          {item.title} <span className="muted req-id">{item.requirementId}</span>
        </h2>
      </div>
      <div className="detail-grid">
        <div>
          类别：<span className="badge badge-category">{item.category}</span>
        </div>
        <div>
          流程：<span className="badge badge-flow">{item.flowDefsId}</span>{" "}
          {item.flowDefsId !== "" && <ScopeBadge flowDefsId={item.flowDefsId} />}
        </div>
        <div>
          优先级：
          <PriorityBadge priority={item.priority} />
        </div>
        <div>
          状态：
          <StatusBadge status={item.status} />
        </div>
        <div>版本：{versionName}</div>
        <div>负责人：{item.owner === "" ? "未指派" : item.owner}</div>
        <div>创建人：{item.createdBy}</div>
        <div>创建：{formatTime(item.createdAt)}</div>
        <div>基线 updated_at：{item.updatedAt}</div>
      </div>
      <h3>描述</h3>
      <pre className="description">
        {item.description === "" ? "（暂无描述）" : item.description}
      </pre>
      <div className="placeholder-note">
        占位说明：流程树 + 节点详情五区分栏 + 阅读器锚点评论由工单 T15 接管；当前
        flows/artifacts/issues 等集合字段在 T4/T6/T7 实例化前为空集。
      </div>
    </section>
  );
}
