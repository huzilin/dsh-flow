import { useState } from "react";
import { Modal } from "./Modal";

/** 二次确认弹窗（删除/废弃等破坏性操作——PRD §6 安全红线）。 */
export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "确认",
  onConfirm,
  onClose,
  submitting,
  error,
}: {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onClose: () => void;
  submitting?: boolean;
  error?: string | null;
}) {
  return (
    <Modal title={title} open={open} onClose={onClose}>
      <p className="confirm-message">{message}</p>
      {error && <span className="field-hint error">{error}</span>}
      <div className="modal-actions">
        <button type="button" onClick={onClose}>
          取消
        </button>
        <button type="button" className="danger" disabled={submitting} onClick={onConfirm}>
          {confirmLabel}
        </button>
      </div>
    </Modal>
  );
}

/** 版本改名弹窗（乐观锁：提交时回传行上的 updated_at 基线）。 */ export function RenameVersionDialog({
  open,
  initialName,
  onClose,
  onSubmit,
  submitting,
  error,
}: {
  open: boolean;
  initialName: string;
  onClose: () => void;
  onSubmit: (name: string) => void;
  submitting?: boolean;
  error?: string | null;
}) {
  const [name, setName] = useState(initialName);
  const canSave = name.trim() !== "" && name !== initialName;
  return (
    <Modal title="版本改名" open={open} onClose={onClose}>
      <form
        className="form"
        onSubmit={(e) => {
          e.preventDefault();
          if (canSave) onSubmit(name.trim());
        }}
      >
        <label>
          版本名
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={name.trim() === "" ? "invalid" : ""}
          />
        </label>
        {error && <span className="field-hint error">{error}</span>}
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

/** 新建版本弹窗（name 必填；is_backlog=待分配池标记行）。 */
export function CreateVersionDialog({
  open,
  onClose,
  onSubmit,
  submitting,
  error,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (name: string, isBacklog: boolean) => void;
  submitting?: boolean;
  error?: string | null;
}) {
  const [name, setName] = useState("");
  const [isBacklog, setIsBacklog] = useState(false);
  const canSave = name.trim() !== "";
  return (
    <Modal title="新建版本" open={open} onClose={onClose}>
      <form
        className="form"
        onSubmit={(e) => {
          e.preventDefault();
          if (canSave) onSubmit(name.trim(), isBacklog);
        }}
      >
        <label>
          版本名 *
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={name.trim() === "" ? "invalid" : ""}
            placeholder="自由文本，如 v1.2 移动端"
          />
        </label>
        {name.trim() === "" && <span className="field-hint error">版本名不能为空</span>}
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={isBacklog}
            onChange={(e) => setIsBacklog(e.target.checked)}
          />
          待分配池标记（is_backlog，仅显示无操作）
        </label>
        {error && <span className="field-hint error">{error}</span>}
        <div className="modal-actions">
          <button type="button" onClick={onClose}>
            取消
          </button>
          <button type="submit" disabled={!canSave || submitting}>
            创建
          </button>
        </div>
      </form>
    </Modal>
  );
}
