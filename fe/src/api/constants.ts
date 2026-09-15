/**
 * 模块 I 展示常量（与 rd/domain/binding.go 表驱动同源；后端校验仍是真相源，
 * 此处仅做 UI 选项/徽标呈现，禁在此二次发明校验规则）。
 */

/** 需求类别（requirements.category 全集，展示序同 domain.CategoryDefinitions）。 */
export const CATEGORIES = ["需求任务", "规划审批任务", "版本规划任务"] as const;

/** 类别→flow_defs_id 允许集（创建对话框显式选择；需求任务默认 business）。 */
export const CATEGORY_FLOW_OPTIONS: Record<string, string[]> = {
  需求任务: ["business", "tech", "task"],
  规划审批任务: ["plan"],
  版本规划任务: ["plan"],
};

export const CATEGORY_DEFAULT_FLOW: Record<string, string> = {
  需求任务: "business",
  规划审批任务: "plan",
  版本规划任务: "plan",
};

/** 流程类型显示名。 */
export const FLOW_LABELS: Record<string, string> = {
  business: "业务流程",
  tech: "技术流程",
  task: "任务流程",
  plan: "规划流程",
};

/** 优先级（P0-P3；创建空值默认 P2）。 */
export const PRIORITIES = ["P0", "P1", "P2", "P3"] as const;

/** 需求状态（requirements.status）。 */
export const REQUIREMENT_STATUSES = ["待启动", "进行中", "已完成", "已取消"] as const;

/**
 * Web 端操作身份（T15 口径）：浏览器侧用户以 user 角色操作（拍板/评论/需求编辑）。
 * DSH 嵌入期按会话身份注入 author/roleId，此处只放单一常量，禁散落魔法串。
 */
export const WEB_USER_ROLE = "user";

/** 节点相位（flow_nodes.state 4 相位 + ""=未进入编排；CONTEXT.md「相位」）。 */
export const PHASE_LABELS: Record<string, string> = {
  "": "未进入编排",
  entry: "entry · 进入",
  exec: "exec · 执行",
  review: "review · 评审",
  respond: "respond · 响应",
  done: "done · 已完成",
};

/** 节点类型（flow_def_nodes.node_type）。 */
export const NODE_TYPE_LABELS: Record<string, string> = {
  produce: "产出",
  review: "评审",
  align: "对齐",
  gate: "门禁",
};

/**
 * issue 状态机 v2 状态展示（与 rd/domain issue 状态机词汇对齐；CONTEXT.md：
 * pending-confirm=已处理待复核、resolved=已解决终态；09-12 裁决 overridden 态
 * 废除，override 动作置 resolved）。
 */
export const ISSUE_STATE_LABELS: Record<string, string> = {
  open: "待处理",
  "pending-confirm": "已处理待复核",
  "to-fix": "待修复",
  fixed: "已修复待复核",
  resolved: "已解决",
};

/** 评审表态（review_results.state）。 */
export const REVIEW_STATE_LABELS: Record<string, string> = {
  reviewing: "待表态",
  agree: "同意",
  unagree: "反对",
};

/** resolve 路径展示（fix=修复关闭 / agree=协商一致关闭 / override=强权仲裁关闭）。 */
export const RESOLVE_PATH_LABELS: Record<string, string> = {
  fix: "修复关闭",
  agree: "协商一致关闭",
  override: "强权仲裁关闭",
};

/** 版本状态（versions.state，单向 + 终态）。 */
export const VERSION_STATES = ["待启动", "进行中", "已发布", "已废弃"] as const;

/**
 * 角色全集（roles 表 10 项，与 rd/migration/sql/0002_seed.sql 同源；仅作收件箱
 * 角色/TODO 过滤选项，后端校验仍是真相源）。
 */
export const ROLES = [
  "rd",
  "fe",
  "qa",
  "arch",
  "pm",
  "cr",
  "inv",
  "leader",
  "user",
  "executor",
] as const;

/**
 * 通知七类（issues.anchor_kind 复用列承载；rd/service/notifier.go 单一来源）。
 * 通知规程 v3：gate_pending/re_review/need_intervene 收件=user 为主，其余按触发点。
 */
export const NOTIFICATION_KINDS = [
  "gate_pending",
  "clarify_task",
  "re_review",
  "stall_followup",
  "need_intervene",
  "escalation",
  "session_message",
] as const;

/** 通知类别 → 展示名（ui-interaction-final §3.1 v3 词汇）。 */
export const NOTIFICATION_KIND_LABELS: Record<string, string> = {
  gate_pending: "待你拍板",
  clarify_task: "澄清任务",
  re_review: "重新表态请求",
  stall_followup: "悬置催办",
  need_intervene: "需你介入",
  escalation: "冲突上升裁决",
  session_message: "会话消息",
};

/** 通知类别 → 图标（双通道：符号承载色系语义，文本恒可读）。 */
export const NOTIFICATION_KIND_ICONS: Record<string, string> = {
  gate_pending: "🔔",
  clarify_task: "❓",
  re_review: "🔁",
  stall_followup: "⏰",
  need_intervene: "🛎",
  escalation: "⚔",
  session_message: "💬",
};

/** 通知类别 → 跳转目标类型（rd/service/notifier.go notificationTargetTypes 同源）。 */
export const NOTIFICATION_KIND_TARGETS: Record<string, string> = {
  gate_pending: "node",
  clarify_task: "node",
  re_review: "node",
  stall_followup: "task",
  need_intervene: "task",
  escalation: "issue",
  session_message: "session",
};

/** TODO 投影在途状态（tasks.status 在途三值；终态 completed/rejected/closed 不投影）。 */
export const TODO_STATUS_LABELS: Record<string, string> = {
  todo: "待认领",
  "in-progress": "执行中",
  "awaiting-reply": "待回复",
};

/** 仲裁结论（AdjudicateReq.verdict：uphold | override）。 */
export const ADJUDICATE_VERDICT_LABELS: Record<string, string> = {
  uphold: "维持议题",
  override: "仲裁推翻",
};

/**
 * flow_defs_id → scope 徽标示意（需求卡/列表行）。
 * 需求 scope 实值由流程实例化按 scope_options 决定（T4 消费，T14 前无实例数据）；
 * 此处按 seed 的 flow_defs.scope_options 呈现：business/tech 集合 [full,rd,fe] 默认 full、
 * task 集合 [rd,fe]、plan 无 scope。
 */
export const SCOPE_BADGE_BY_FLOW: Record<string, string> = {
  business: "full",
  tech: "full",
  task: "rd|fe",
  plan: "",
};

/**
 * flow_defs_id → scope_options（RequirementModal scope 下拉选项集；
 * 与 rd/migration 0002_seed.sql 的 flow_defs.scope_options 同源：
 * business/tech=["full","rd","fe"]、task=["rd","fe"]、plan=[]）。
 * plan 无 scope：下拉禁用置 —（§6 N4 定稿口径）。
 */
export const SCOPE_OPTIONS_BY_FLOW: Record<string, string[]> = {
  business: ["full", "rd", "fe"],
  tech: ["full", "rd", "fe"],
  task: ["rd", "fe"],
  plan: [],
};

/**
 * TODO 五分类（ui-interaction-final §6 N3 定稿：TODO=收件箱通知的操作投影，
 * 非 tasks 投影；spec §1.2 注记-D5）。键=展示名；投影映射见 InboxPage
 * todoProjection（悬置催办/需你介入/会话消息为提醒类，不产生 TODO）。
 */
export const TODO_CATEGORIES = [
  "待拍板",
  "待澄清",
  "待重表态",
  "待复验",
  "冲突待裁决",
] as const;

export type TodoCategory = (typeof TODO_CATEGORIES)[number];

/**
 * 会话绑定状态展示（sessions.status：active/ended/disposed；§6 N5 会话 tab
 * 条目状态标签口径）。
 */
export const SESSION_STATUS_LABELS: Record<string, string> = {
  active: "进行中",
  ended: "已结束",
  disposed: "已释放",
};

/**
 * 会话找 X 按钮的绑定角色推导（§6 N5：X 随节点绑定角色动态——
 * scope=full 找 qa / scope=fe 找 fe / scope=rd 找 rd；flows.scope 空串=未定
 * （task 渲染期再选），兜底 qa。DSH 嵌入期开放，本期按钮 disabled 占位）。
 */
export function sessionFindRoleForScope(flowScope: string): string {
  if (flowScope === "rd") return "rd";
  if (flowScope === "fe") return "fe";
  return "qa";
}
