import { useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import { WEB_USER_ROLE } from "../api/constants";
import { deleteRequirement } from "../api/requirements";
import { useMyTaskQuery, useRequirementQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { PriorityBadge, StatusBadge } from "../components/Badges";
import { FlowScopeBadge, FlowTree } from "../components/FlowTree";
import { NodeDetail } from "../components/NodeDetail";
import { Modal } from "../components/Modal";
import { formatTime } from "./RequirementPoolPage";

/**
 * 需求详情 = 模块 II 主视图（T15）：需求头 + 流程树（SVG 纵向全节点渲染，scope 实值徽标）+
 * 节点详情五区分栏 + 产物阅读器（NodeDetail 内弹出）。
 * 数据源全部走聚合快照 GET /api/requirements/:reqId + my_task，不自拼接口。
 * 选中节点缺省=当前节点（flows.current_node_id；空=未进入编排不预选）。
 */
export function RequirementDetailPage() {
  const { reqId = "" } = useParams();
  const detail = useRequirementQuery(reqId);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const flow = detail.data?.flows[0];
  const nodes = useMemo(() => detail.data?.flowNodes ?? [], [detail.data]);
  const issues = useMemo(() => detail.data?.issues ?? [], [detail.data]);
  const artifacts = useMemo(() => detail.data?.artifacts ?? [], [detail.data]);

  const currentNodeId = flow?.currentNodeId ?? "";
  const effectiveNodeId = selectedNodeId ?? currentNodeId;
  const myTask = useMyTaskQuery(reqId, effectiveNodeId === "" ? null : effectiveNodeId);

  const issuesByNode = useMemo(() => {
    const map = new Map<string, number>();
    for (const i of issues) {
      if (i.state !== "resolved" && i.state !== "overridden") {
        map.set(i.nodeId, (map.get(i.nodeId) ?? 0) + 1);
      }
    }
    return map;
  }, [issues]);

  if (detail.isLoading) {
    return <section className="muted">加载中…</section>;
  }
  const item = detail.data?.requirement;
  if (!item) {
    return (
      <section>
        <Link to="/pool">← 返回需求池</Link>
        <AlertBanner error={detail.error} onRetry={() => void detail.refetch()} />
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
        <DeleteRequirementButton reqId={item.requirementId} open={deleteOpen} setOpen={setDeleteOpen} />
      </div>
      <div className="detail-grid">
        <div>
          类别：<span className="badge badge-category">{item.category}</span>
        </div>
        <div>
          流程：<span className="badge badge-flow">{item.flowDefsId}</span>{" "}
          <FlowScopeBadge flow={flow} />
        </div>
        <div>
          优先级：<PriorityBadge priority={item.priority} />
        </div>
        <div>
          状态：<StatusBadge status={item.status} />
        </div>
        <div>版本：{versionName}</div>
        <div>负责人：{item.owner === "" ? "未指派" : item.owner}</div>
        <div>创建人：{item.createdBy}</div>
        <div>创建：{formatTime(item.createdAt)}</div>
        <div className="muted">操作身份：{WEB_USER_ROLE}（my_task 三向校验）</div>
      </div>
      {item.description !== "" && (
        <details className="req-description">
          <summary className="muted">描述</summary>
          <pre className="description">{item.description}</pre>
        </details>
      )}

      {!detail.isLoading && <AlertBanner error={detail.error} onRetry={() => void detail.refetch()} />}

      <FlowTree
        flow={flow}
        nodes={nodes}
        currentNodeId={currentNodeId}
        selectedNodeId={effectiveNodeId === "" ? null : effectiveNodeId}
        issuesByNode={issuesByNode}
        onSelectNode={setSelectedNodeId}
      />

      <NodeDetail
        nodeId={effectiveNodeId === "" ? null : effectiveNodeId}
        node={nodes.find((n) => n.nodeId === effectiveNodeId)}
        myTask={myTask.data}
        artifacts={artifacts}
        issues={issues}
        requirementId={reqId}
      />
    </section>
  );
}

/** 删除需求（User 明确指令口径：显式 confirm=true + 确认弹窗；T14 README 交接入口落位本页）。 */
function DeleteRequirementButton({
  reqId,
  open,
  setOpen,
}: {
  reqId: string;
  open: boolean;
  setOpen: (v: boolean) => void;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const del = useMutation({
    mutationFn: () => deleteRequirement(api, reqId, true),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["requirements"] });
      setOpen(false);
      navigate("/pool");
    },
  });
  return (
    <>
      <button type="button" className="danger" onClick={() => setOpen(true)}>
        删除需求
      </button>
      <Modal title="删除需求（不可恢复）" open={open} onClose={() => setOpen(false)}>
        <p className="confirm-message">
          确认删除需求 <b>{reqId}</b>？该操作携带 confirm=true 直接生效（User 明确指令口径）。
        </p>
        {del.error && <AlertBanner error={del.error} />}
        <div className="modal-actions">
          <button type="button" onClick={() => setOpen(false)}>
            取消
          </button>
          <button type="button" className="danger" disabled={del.isPending} onClick={() => del.mutate()}>
            确认删除
          </button>
        </div>
      </Modal>
    </>
  );
}
