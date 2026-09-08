import { useState } from "react";
import type { RequirementItem, VersionItem } from "../api/proto/api";
import {
  CATEGORIES,
  CATEGORY_DEFAULT_FLOW,
  CATEGORY_FLOW_OPTIONS,
  FLOW_LABELS,
  PRIORITIES,
  REQUIREMENT_STATUSES,
  SCOPE_OPTIONS_BY_FLOW,
} from "../api/constants";
import { Modal } from "./Modal";

/**
 * 需求新建/编辑弹窗（01-module-board PRD §5.2 + ui-interaction-final §6 N4 定稿：
 * RequirementModal，编辑复用同弹窗——卡片 hover ✎ / 详情页「编辑」）。
 * - category 必选；flow_defs_id 在类别允许集内显式选择（需求任务 business/tech/task，规划/版本规划恒 plan）；
 * - scope 下拉随 flow_defs_id 的 scope_options 派生（N4：scope select 三值；
 *   规划审批/版本规划→plan 禁用置 —）；创建空值=后端取 scope_options 首值；
 * - 标题空 → 保存禁用 + 红框（PRD 异常场景）；版本缺省=待分配池；优先级缺省 P2。
 * - 编辑模式（mode="edit"）：title/description/priority/status/owner/versionId 可改；
 *   category/scope/流程类型只读——PUT 无重投影（流程实例不随类别变更重建），且
 *   spec 拍板「scope 主动切换暂不支持」（变更仅走打回→reproject 链路）。
 */
export function CreateRequirementDialog(props: CreateRequirementDialogProps) {
  if (!props.open) return null;
  // key 绑定编辑基线：open 重新拉起/基线变化时重挂载表单，useState 惰性初始化回填
  const key = `${props.mode ?? "create"}-${props.initial?.requirementId ?? ""}-${props.initial?.updatedAt ?? ""}`;
  return <RequirementModalForm key={key} {...props} />;
}

interface CreateRequirementDialogProps {
  open: boolean;
  versions: VersionItem[];
  onClose: () => void;
  /** mode=create 时必传（CreateRequirementReq 载体） */
  onSubmit?: (req: {
    title: string;
    description: string;
    category: string;
    flowDefsId: string;
    scope: string;
    versionId: string;
    priority: string;
    owner: string;
    createdBy: string;
  }) => void;
  /** mode=edit 时必传（UpdateRequirementReq 载体；updatedAt 由本组件透传 initial 基线） */
  onUpdate?: (req: {
    title: string;
    description: string;
    category: string;
    priority: string;
    status: string;
    owner: string;
    versionId: string;
    updatedAt: string;
  }) => void;
  submitting?: boolean;
  submitError?: string | null;
  mode?: "create" | "edit";
  /** mode=edit 的基线行（快照 requirement） */
  initial?: RequirementItem;
  /** mode=edit 的 scope 实值（flows.scope；RequirementItem 无此字段，父组件透传） */
  initialScope?: string;
}

function RequirementModalForm({
  versions,
  onClose,
  onSubmit,
  onUpdate,
  submitting,
  submitError,
  mode = "create",
  initial,
  initialScope = "",
}: Omit<CreateRequirementDialogProps, "open">) {
  // 编辑=基线回填；新建=空值（外层 key 重挂载保证每次打开均为新鲜初值）
  const [title, setTitle] = useState(() => initial?.title ?? "");
  const [description, setDescription] = useState(() => initial?.description ?? "");
  const [category, setCategory] = useState<string>(() => initial?.category ?? "");
  const [flowDefsId, setFlowDefsId] = useState<string>(() => initial?.flowDefsId ?? "");
  const [scope, setScope] = useState<string>(initialScope);
  const [versionId, setVersionId] = useState(() => initial?.versionId ?? "");
  const [priority, setPriority] = useState(() => initial?.priority ?? "");
  const [status, setStatus] = useState(() => initial?.status ?? "");
  const [owner, setOwner] = useState(() => initial?.owner ?? "");
  const [createdBy, setCreatedBy] = useState(() => initial?.createdBy ?? "");


  const flowOptions = category !== "" ? CATEGORY_FLOW_OPTIONS[category] : [];
  const scopeOptions = flowDefsId !== "" ? (SCOPE_OPTIONS_BY_FLOW[flowDefsId] ?? []) : [];
  const canSave =
    title.trim() !== "" &&
    (mode === "create"
      ? createdBy.trim() !== "" && category !== ""
      : true);

  const handleCategoryChange = (next: string) => {
    setCategory(next);
    // 类别切换即重置流程选择为该类别默认（后端 DeriveFlowDefsID 同规则）
    setFlowDefsId(next === "" ? "" : CATEGORY_DEFAULT_FLOW[next]);
    setScope("");
  };

  const submit = () => {
    if (!canSave) return;
    if (mode === "edit" && initial) {
      onUpdate?.({
        title: title.trim(),
        description,
        category,
        priority: priority === "" ? "P2" : priority,
        status: status === "" ? initial.status : status,
        owner,
        versionId,
        updatedAt: initial.updatedAt, // GET 所得基线必回传（乐观锁）
      });
      return;
    }
    onSubmit?.({
      title: title.trim(),
      description,
      category,
      flowDefsId,
      scope,
      versionId,
      priority,
      owner,
      createdBy: createdBy.trim(),
    });
  };

  return (
    <Modal title={mode === "edit" ? "编辑需求" : "新建需求"} open onClose={onClose}>
      <form
        className="form"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        <label>
          标题 *
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className={title.trim() === "" ? "invalid" : ""}
            placeholder="必填"
          />
        </label>
        {title.trim() === "" && <span className="field-hint error">标题不能为空</span>}

        <label>
          描述（Markdown）
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={4}
            placeholder="可空"
          />
        </label>

        <label>
          类别 {mode === "create" ? "*" : "（只读）"}
          <select
            value={category}
            onChange={(e) => handleCategoryChange(e.target.value)}
            disabled={mode === "edit"}
            title={mode === "edit" ? "类别变更涉及流程重投影，暂不开放" : undefined}
          >
            <option value="">请选择</option>
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>

        {category !== "" && (
          <label>
            流程类型（类别 {category} 的允许集）
            <select
              value={flowDefsId}
              onChange={(e) => {
                setFlowDefsId(e.target.value);
                setScope("");
              }}
              disabled={mode === "edit" || flowOptions.length === 1}
            >
              {flowOptions.map((f) => (
                <option key={f} value={f}>
                  {f} · {FLOW_LABELS[f]}
                </option>
              ))}
            </select>
          </label>
        )}

        {category !== "" && (
          <label>
            scope{scopeOptions.length === 0 && "（—）"}
            {mode === "edit" ? (
              <input value={scope === "" ? "—" : scope} disabled readOnly />
            ) : (
              <select
                value={scope}
                onChange={(e) => setScope(e.target.value)}
                disabled={scopeOptions.length === 0}
                title={
                  scopeOptions.length === 0
                    ? "规划/版本规划流程无 scope（N4：禁用置 —）"
                    : undefined
                }
              >
                <option value="">
                  {scopeOptions.length === 0
                    ? "—"
                    : `默认（${scopeOptions[0]}）`}
                </option>
                {scopeOptions.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            )}
            {mode !== "edit" && scopeOptions.length > 0 && (
              <span className="field-hint">
                scope_options 随所选流程派生；空=取首值
              </span>
            )}
          </label>
        )}

        <label>
          版本
          <select value={versionId} onChange={(e) => setVersionId(e.target.value)}>
            <option value="">待分配池{mode === "create" ? "（默认）" : ""}</option>
            {versions
              .filter((v) => !v.isBacklog)
              .map((v) => (
                <option key={v.versionId} value={v.versionId}>
                  {v.name}（{v.versionId}）
                </option>
              ))}
          </select>
        </label>

        <label>
          优先级
          <select value={priority} onChange={(e) => setPriority(e.target.value)}>
            <option value="">P2（默认）</option>
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </label>

        {mode === "edit" && (
          <label>
            状态
            <select value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="">不修改</option>
              {REQUIREMENT_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </label>
        )}

        <label>
          负责人
          <input
            value={owner}
            onChange={(e) => setOwner(e.target.value)}
            placeholder="可空=未指派"
          />
        </label>

        {mode === "create" && (
          <label>
            创建人 *
            <input
              value={createdBy}
              onChange={(e) => setCreatedBy(e.target.value)}
              className={createdBy.trim() === "" ? "invalid" : ""}
              placeholder="必填"
            />
          </label>
        )}

        {submitError && <span className="field-hint error">{submitError}</span>}

        <div className="modal-actions">
          <button type="button" onClick={onClose}>
            取消
          </button>
          <button type="submit" disabled={!canSave || submitting}>
            保存
          </button>
        </div>
      </form>
    </Modal>
  );
}

/** RequirementModal 别名（§6 N4 定稿词汇；编辑复用同弹窗语义的显式命名）。 */
export { CreateRequirementDialog as RequirementModal };
