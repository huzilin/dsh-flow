import { useMemo, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "../api/client";
import { WEB_USER_ROLE, sessionFindRoleForScope } from "../api/constants";
import { deleteRequirement, updateRequirement } from "../api/requirements";
import type { FlowNodeItem, RequirementItem } from "../api/proto/api";
import { useMyTaskQuery, useRequirementHistoryQuery, useRequirementQuery, useVersionsQuery } from "../hooks/queries";
import { AlertBanner } from "../components/AlertBanner";
import { PriorityBadge, StatusBadge } from "../components/Badges";
import { FlowScopeBadge, FlowTree } from "../components/FlowTree";
import { MarkdownView } from "../components/MarkdownView";
import { NodeDetail } from "../components/NodeDetail";
import { RequirementModal } from "../components/CreateRequirementDialog";
import { Modal } from "../components/Modal";
import { formatTime } from "./RequirementPoolPage";
import { describeHistory } from "../domain/historyview";

/**
 * 需求详情 = 模块 II 主视图（T15）：需求头 + 流程树（SVG 纵向全节点渲染，scope 实值徽标）+
 * 节点详情五区分栏 + 产物阅读器（NodeDetail 内弹出）。
 * 数据源全部走聚合快照 GET /api/requirements/:reqId + my_task，不自拼接口。
 * 选中节点缺省=当前节点（flows.current_node_id；空=未进入编排不预选）。
 * 收件箱跳转定位（T16）：?node=（选中节点）&focus=issue|task&issue=/task=（意见区/任务区锚点高亮）。
 * 原型补齐（ui-interaction-final §6）：detailtop「会话找 X」（N5，scope 动态推导，DSH 嵌入期开放）+
 * 详情页「编辑」（N4：RequirementModal 编辑复用）+ 需求描述卡（BUG-012/OPT-003 归位：
 * 头部元信息与流程树之间的 Markdown 描述卡，变更历史随卡折叠）。
 * OPT-001 两栏：左列流程导航（FlowTree 复用）+ 右列 NodeDetail（§6 m2 原型形态）。
 */
export function RequirementDetailPage() {
  const { reqId = "" } = useParams();
  const [searchParams] = useSearchParams();
  const focusNodeId = searchParams.get("node") ?? "";
  const focusIssueId = searchParams.get("issue") ?? "";
  const focusTaskId = searchParams.get("task") ?? "";
  const detail = useRequirementQuery(reqId);
  const versions = useVersionsQuery();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);
  const queryClient = useQueryClient();

  // 编辑提交（乐观锁 409 → 提示冲突 + 快照/列表失效刷新基线）
  const updateMutation = useMutation({
    mutationFn: (input: { req: Parameters<typeof updateRequirement>[2]; reqId: string }) =>
      updateRequirement(api, input.reqId, input.req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["requirement", reqId] });
      void queryClient.invalidateQueries({ queryKey: ["requirements"] });
      void queryClient.invalidateQueries({ queryKey: ["requirementHistory", reqId] });
      setEditOpen(false);
      setEditError(null);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.isOptimisticLock) {
        void queryClient.invalidateQueries({ queryKey: ["requirement", reqId] });
        setEditOpen(false);
        setEditError(null);
        window.alert("基线冲突：需求已被他人修改，已刷新最新数据，请重试");
      } else {
        setEditError(err instanceof Error ? err.message : String(err));
      }
    },
  });

  const flow = detail.data?.flows[0];
  const nodes = useMemo(() => detail.data?.flowNodes ?? [], [detail.data]);
  const issues = useMemo(() => detail.data?.issues ?? [], [detail.data]);
  const artifacts = useMemo(() => detail.data?.artifacts ?? [], [detail.data]);
  const sessions = useMemo(() => detail.data?.sessions ?? [], [detail.data]);

  // 收件箱 issue 锚点未带 node 参数时，从快照反查该意见挂载节点
  const focusIssueNodeId = focusIssueId === "" ? "" : (issues.find((i) => i.issueId === focusIssueId)?.nodeId ?? "");
  const paramNodeId = focusNodeId || focusIssueNodeId;

  const currentNodeId = flow?.currentNodeId ?? "";
  const effectiveNodeId = selectedNodeId ?? (paramNodeId || currentNodeId);
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
        <span className="row-actions">
          <button type="button" onClick={() => setEditOpen(true)}>
            编辑
          </button>
          {/* 会话找 X（§6 N5：X 随节点绑定角色动态——full=qa / fe-only=fe / rd-only=rd；
              按需会话通道 DSH 嵌入期开放，本期 disabled 占位） */}
          <button
            type="button"
            disabled
            title="按需会话通道（DSH 嵌入期开放）；X 随 scope 动态"
          >
            💬 会话找 {sessionFindRoleForScope(flow?.scope ?? "")}
          </button>
          <DeleteRequirementButton reqId={item.requirementId} open={deleteOpen} setOpen={setDeleteOpen} />
        </span>
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

      {!detail.isLoading && <AlertBanner error={detail.error} onRetry={() => void detail.refetch()} />}

      {/* BUG-012/OPT-003 归位：Markdown 描述卡（有描述默认展开）+ 元信息 + 变更历史，
          位于头部元信息与流程树之间，不再沉底 */}
      <RequirementBriefCard item={item} currentNode={nodes.find((n) => n.nodeId === currentNodeId)} />

      {/* OPT-001 两栏：左列流程导航常驻（SVG 树缩放复用，颜色样式语义不动）+
          右列节点详情；窄屏退化上下堆叠（css 断点） */}
      <div className="flow-detail-columns">
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
          sessions={sessions}
          requirementId={reqId}
          focusIssueId={focusIssueId}
          focusTaskId={focusTaskId}
        />
      </div>

      <RequirementModal
        open={editOpen}
        mode="edit"
        initial={item}
        initialScope={flow?.scope ?? ""}
        versions={versions.data?.items ?? []}
        submitting={updateMutation.isPending}
        submitError={editError}
        onClose={() => {
          setEditOpen(false);
          setEditError(null);
        }}
        onUpdate={(req) => {
          updateMutation.mutate({
            reqId: item.requirementId,
            req: {
              requirementId: item.requirementId,
              title: req.title,
              description: req.description,
              category: req.category,
              priority: req.priority,
              status: req.status,
              owner: req.owner,
              versionId: req.versionId,
              updatedAt: req.updatedAt,
            },
          });
        }}
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

/**
 * 需求描述卡（BUG-012/OPT-003 归位版，原沉底补遗卡上移）：Markdown 描述（有描述默认
 * 展开，轻量渲染不引重库——components/MarkdownView）+ 元信息行 + 变更历史（描述卡下方
 * 折叠区，时间倒序自然语句，不再单独沉底）。
 */
function RequirementBriefCard({
  item,
  currentNode,
}: {
  item: RequirementItem;
  currentNode?: FlowNodeItem;
}) {
  const [histOpen, setHistOpen] = useState(false);
  // BUG-012：有描述默认展开；空描述收起呈占位
  const [descOpen, setDescOpen] = useState(item.description !== "");
  const history = useRequirementHistoryQuery(item.requirementId, histOpen);

  return (
    <section className="brief-card" aria-label="需求描述卡">
      <button
        type="button"
        className="button-link brief-toggle"
        aria-expanded={descOpen}
        onClick={() => setDescOpen((v) => !v)}
      >
        需求说明{descOpen ? " ▴" : " ▾"}
      </button>
      {descOpen &&
        (item.description === "" ? (
          <div className="muted">（无描述）</div>
        ) : (
          <MarkdownView source={item.description} />
        ))}
      <div className="detail-grid footnote-meta">
        <div>负责人：{item.owner === "" ? "—" : item.owner}</div>
        <div>当前节点：{currentNode ? currentNode.nodeKey : "未进入编排"}</div>
        <div>
          进入时间：
          {currentNode && currentNode.enteredAt !== "0" ? formatTime(currentNode.enteredAt) : "—"}
        </div>
        <div>回滚次数：{currentNode ? currentNode.rolledBackCount : 0}</div>
      </div>
      <div className="footnote-history">
        <button type="button" className="button-link" onClick={() => setHistOpen((v) => !v)}>
          变更历史（histPanel · 时间倒序）{histOpen ? " ▴" : " ▾"}
        </button>
        {histOpen && (
          <ul className="history-list">
            {history.isLoading && <li className="muted">加载中…</li>}
            {history.error !== null && (
              <li>
                <AlertBanner error={history.error} onRetry={() => void history.refetch()} />
              </li>
            )}
            {!history.isLoading && (history.data?.items.length ?? 0) === 0 && (
              <li className="muted">暂无变更记录。</li>
            )}
            {(history.data?.items ?? []).map((h, idx) => (
              <li key={`${h.createdAt}-${idx}`} className="history-row">
                <span className="muted">{formatTime(h.createdAt)}</span>
                <span>{describeHistory(h.action, h.field, h.oldValue, h.newValue)}</span>
                <span className="muted">by {h.byWho}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}

// 审计行 → 自然语句装配移至 domain/historyview.ts（纯函数，可直测；N6 口径）
