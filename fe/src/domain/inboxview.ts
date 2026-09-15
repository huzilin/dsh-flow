/**
 * 收件箱视图模型（纯函数，禁 IO）。TODO 五分类投影规则（spec -D5 /
 * ui-interaction-final §6 N3 定稿）——业务投影归位 domain（C4），页面层只消费。
 */
import type { InboxItem } from "../api/proto/api";

/** TODO = 收件箱通知的操作投影，同请求 notifications 同源推导、无新请求；
 *  待拍板←gate_pending；待澄清←clarify_task；待重表态←re_review 且 target_id 空
 *  （澄清子型）；待复验←re_review 且 target_id 非空（retest 子型）；冲突待裁决←escalation；
 *  stall_followup/need_intervene/session_message 为提醒类，不产生 TODO。 */
export interface TodoGroup {
  key: string;
  label: string;
  match: (n: InboxItem) => boolean;
}

export const TODO_GROUPS: TodoGroup[] = [
  { key: "gate", label: "待拍板", match: (n) => n.kind === "gate_pending" },
  { key: "clarify", label: "待澄清", match: (n) => n.kind === "clarify_task" },
  { key: "re-state", label: "待重表态", match: (n) => n.kind === "re_review" && n.targetId === "" },
  { key: "retest", label: "待复验", match: (n) => n.kind === "re_review" && n.targetId !== "" },
  { key: "conflict", label: "冲突待裁决", match: (n) => n.kind === "escalation" },
];

/** 投影输出：分组（含 items 引用，顺序=TODO_GROUPS 声明序）。 */
export interface TodoProjection {
  key: string;
  label: string;
  items: InboxItem[];
}

/** TODO 投影（页面 dock 条计数与分组渲染的单一数据源）。
 *  口径注记：基于当前筛选+分页返回的通知列表（后端无跨页聚合端点，计数随筛选联动）。 */
export function projectTodoGroups(notifications: InboxItem[]): TodoProjection[] {
  return TODO_GROUPS.map((g) => ({
    key: g.key,
    label: g.label,
    items: notifications.filter(g.match),
  }));
}
