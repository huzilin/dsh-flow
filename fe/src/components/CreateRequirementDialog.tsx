import { useState } from "react";
import type { CreateRequirementReq, VersionItem } from "../api/proto/api";
import {
  CATEGORIES,
  CATEGORY_DEFAULT_FLOW,
  CATEGORY_FLOW_OPTIONS,
  FLOW_LABELS,
  PRIORITIES,
} from "../api/constants";
import { Modal } from "./Modal";

/**
 * 新建需求对话框（01-module-board PRD §5.2 需求管理 + ui-interaction §2 类别↔流程类型绑定）：
 * - category 必选；flow_defs_id 在类别允许集内显式选择（需求任务 business/tech/task，规划/版本规划恒 plan）；
 * - 标题/创建人必填，标题空 → 保存禁用 + 红框（PRD 异常场景）；
 * - 版本缺省 = 待分配池；优先级缺省 P2。
 */
export function CreateRequirementDialog({
  open,
  versions,
  onClose,
  onSubmit,
  submitting,
  submitError,
}: {
  open: boolean;
  versions: VersionItem[];
  onClose: () => void;
  onSubmit: (req: CreateRequirementReq) => void;
  submitting?: boolean;
  submitError?: string | null;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState<string>("");
  const [flowDefsId, setFlowDefsId] = useState<string>("");
  const [versionId, setVersionId] = useState("");
  const [priority, setPriority] = useState("");
  const [owner, setOwner] = useState("");
  const [createdBy, setCreatedBy] = useState("");

  const flowOptions = category !== "" ? CATEGORY_FLOW_OPTIONS[category] : [];
  const canSave = title.trim() !== "" && createdBy.trim() !== "" && category !== "";

  const reset = () => {
    setTitle("");
    setDescription("");
    setCategory("");
    setFlowDefsId("");
    setVersionId("");
    setPriority("");
    setOwner("");
    setCreatedBy("");
  };

  const handleCategoryChange = (next: string) => {
    setCategory(next);
    // 类别切换即重置流程选择为该类别默认（后端 DeriveFlowDefsID 同规则）
    setFlowDefsId(next === "" ? "" : CATEGORY_DEFAULT_FLOW[next]);
  };

  const submit = () => {
    if (!canSave) return;
    onSubmit({
      title: title.trim(),
      description,
      category,
      flowDefsId,
      versionId,
      priority,
      owner,
      createdBy: createdBy.trim(),
      // scope 空串=取 flow_defs.scope_options 首值（后端派生；T15 proto 重生成新增字段）
      scope: "",
    });
    reset();
  };

  return (
    <Modal title="新建需求" open={open} onClose={onClose}>
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
          类别 *
          <select value={category} onChange={(e) => handleCategoryChange(e.target.value)}>
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
              onChange={(e) => setFlowDefsId(e.target.value)}
              disabled={flowOptions.length === 1}
            >
              {flowOptions.map((f) => (
                <option key={f} value={f}>
                  {f} · {FLOW_LABELS[f]}
                </option>
              ))}
            </select>
          </label>
        )}

        <label>
          版本
          <select value={versionId} onChange={(e) => setVersionId(e.target.value)}>
            <option value="">待分配池（默认）</option>
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

        <label>
          负责人
          <input
            value={owner}
            onChange={(e) => setOwner(e.target.value)}
            placeholder="可空=未指派"
          />
        </label>

        <label>
          创建人 *
          <input
            value={createdBy}
            onChange={(e) => setCreatedBy(e.target.value)}
            className={createdBy.trim() === "" ? "invalid" : ""}
            placeholder="必填"
          />
        </label>

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
