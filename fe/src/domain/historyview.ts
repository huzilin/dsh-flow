/**
 * 需求变更历史自然语句装配（§6 N6：histPanel 时间倒序自然语句；
 * 描述不做逐字 diff——呈现 填写/清空/修改了需求说明）。
 */
export function describeHistory(
  action: string,
  field: string,
  oldValue: string,
  newValue: string,
): string {
  if (action === "INSERT") return "创建了需求";
  if (action === "DELETE") return "删除了需求";
  if (field === "description") {
    if (oldValue === "") return "填写了需求说明";
    if (newValue === "") return "清空了需求说明";
    return "修改了需求说明";
  }
  if (field === "status") return `将「状态」从「${oldValue}」变更为「${newValue}」`;
  if (field === "priority") return `将「优先级」从「${oldValue}」变更为「${newValue}」`;
  if (field === "title") return `将「标题」从「${oldValue}」变更为「${newValue}」`;
  if (field === "owner") return `将「负责人」从「${oldValue || "—"}」变更为「${newValue || "—"}」`;
  if (field === "version_id") return `将「版本」从「${oldValue || "待分配"}」变更为「${newValue || "待分配"}」`;
  if (field === "flow_defs_id") return `将「流程类型」从「${oldValue}」变更为「${newValue}」`;
  if (field === "category") return `将「类别」从「${oldValue}」变更为「${newValue}」`;
  return `将「${field}」从「${oldValue}」变更为「${newValue}」`;
}
