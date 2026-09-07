import { useState } from "react";
import { PRIORITIES, REQUIREMENT_STATUSES } from "../api/constants";
import type { VersionItem } from "../api/proto/api";

/** 版本筛选值元选项（「全部」为前端虚拟选项不入库，同 PRD §5.2 口径）。 */
export const VERSION_FILTER_ALL = "__all__";
export const VERSION_FILTER_BACKLOG = "__backlog__";

/** 版本×优先级×状态筛选条（需求池/看板共用，取交集；看板不传 status）。 */
export function FilterBar({
  versions,
  versionSel,
  onVersionSel,
  priority,
  onPriority,
  status,
  onStatus,
  showStatus,
}: {
  versions: VersionItem[];
  versionSel: string;
  onVersionSel: (v: string) => void;
  priority: string;
  onPriority: (p: string) => void;
  status: string;
  onStatus: (s: string) => void;
  showStatus: boolean;
}) {
  const backlogExists = versions.some((v) => v.isBacklog);
  return (
    <div className="filter-bar">
      <label>
        版本
        <select value={versionSel} onChange={(e) => onVersionSel(e.target.value)}>
          <option value={VERSION_FILTER_ALL}>全部</option>
          <option value={VERSION_FILTER_BACKLOG}>
            待分配池{backlogExists ? "" : "（无标记行）"}
          </option>
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
        <select value={priority} onChange={(e) => onPriority(e.target.value)}>
          <option value="">全部</option>
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
      </label>
      {showStatus && (
        <label>
          状态
          <select value={status} onChange={(e) => onStatus(e.target.value)}>
            <option value="">全部</option>
            {REQUIREMENT_STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
      )}
    </div>
  );
}

/** 受控筛选组 hook（版本选择元值解析归页面）。 */
export function useFilterState() {
  const [versionSel, setVersionSel] = useState(VERSION_FILTER_ALL);
  const [priority, setPriority] = useState("");
  const [status, setStatus] = useState("");
  return { versionSel, setVersionSel, priority, setPriority, status, setStatus };
}
