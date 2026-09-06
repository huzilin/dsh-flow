---
type: grilling
blocked_by: [03, 07]
claimed_by:
claimed_at:
---

# MVP 交互层定稿（插件页面/四模块 UI）

## Question

**MVP 交互层设计定稿**——工作台插件（wbflow）四模块的页面结构与交互推演（此前推演集中在机制层，UI/交互层未系统化；MVP 设计要能照做，交互层不能空）：

1. **模块 I 看板/需求池/版本管理**：版本看板结构、需求 CRUD 交互、版本推进操作——既有 prd/rd/fe 设计（docs/v1.0/01-module-board）为基线，对照新裁决（qa 重定义/编排链路）确认是否调整。
2. **模块 II 流程树**：流程树/节点详情/产物阅读器/锚点评论——重点：**节点会话入口（D6：点击节点查看历史/进行中 subagent 对话）**、评审表态交互、门禁判定展示。
3. **模块 III/IV 收件箱/TODO**：badge/收件箱列表/TODO dock——对照新链路（程序通知 followup 进 Leader 会话历史，`[workbench-notify]`）确认收件箱语义。
4. **新增交互**：需求对齐阶段 subagent 直接与 user 对话的 UI 呈现（D6 会话挂载 + 交互 skills 提示）、开发编排进度（goal/工单状态在节点上的呈现）、并发度配置入口。
5. **UI 与数据层的对应**：每页面/组件消费哪些 wb_* 工具/数据（与工单 07 工具清单对齐）。

依据：docs/v1.0/01~03-module（既有 UI 设计基线）、docs-tmp/plugin-agent-subagent-chain.md（D6）、how-spec-v2.md、skill-usage-implementation.md（交互模式）。
