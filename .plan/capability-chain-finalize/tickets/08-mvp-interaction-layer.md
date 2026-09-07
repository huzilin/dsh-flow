---
type: grilling
blocked_by: [03, 07]
claimed_by: agent session (2026-09-07)
claimed_at: 2026-09-07
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

> **2026-09-07 推演进度**：基线核查（三模块 PRD + workbench-plugin/client.js 829 行：上下分栏/SVG 横向泳道/四级锚点/评论操作矩阵均已实现）；交互原型 `docs-tmp/diagrams/ui-prototype-v3.html` 六面板（节点详情上下分栏吸收四区分栏/SVG 并行泳道+scope 动态渲染重排/药丸 verdict/user_gate 拍板卡/冲突卡上升/对齐会话+完成对齐）——用户逐面板 review 中；结论收敛后产出「模块 II v3 交互增补稿」并推进模块 I/III。**reviewState v5（per-agent 三态 reviewing/agree/unagree + collect 内置聚合 + review_results 单字段）已确认达成一致**。
> **2026-09-08 原型 review 结论（用户批注）**：P1 节点详情（上下分栏+四区分栏+scope 动态渲染重排）✅、P6 对齐会话（完成对齐单按钮）✅、P5 冲突卡 ✅+**补跳转当前 flow 节点详情入口**；**P2 评审轮视图/P3 进度面板=暂缓开发**（复杂度高、需求不强烈——设计保留，实现靠后；reviewState per-agent 数据结构不变，仅 UI 视图后置）。原型=docs-tmp/diagrams/ui-prototype-v3.html。

## Answer（2026-09-08 产出，待 review）

产出 = `docs-tmp/ui-interaction-final.md`（交互层 v3 增补稿，diff 层）+ `docs-tmp/diagrams/ui-prototype-v3.html`（原型六面板，已 review）：

1. **模块 II**：流程树 scope 动态渲染重排（表达式∩需求 scope，重编号不占位，单侧不保留泳道）；节点详情四区→**五区分栏**（状态卡双属性/药丸 per-agent/产物不变/意见+escalate/拍板卡三态）+ 会话 tab（D6）；拍板卡三态规格（进度只读→agreed 迎宾呈现→按钮组+跳转）；冲突卡上升（取代 override）；会话通道按钮（绑定角色动态）；对齐节点会话主视图+完成对齐单按钮。
2. **模块 I**：scope 徽标、dependencies 角标、依赖树视图（动态加载）、并发度配置入口（长期预留展示）。
3. **模块 III/IV**：收件箱条目 v3（七类映射基线六类：待拍板/澄清任务/重表态请求/悬置催办 5min/需你介入 10min/冲突上升/会话消息）；TODO=投影不变。
4. **§4 UI↔数据对应表**：每组件消费字段与工具。
5. **暂缓项**：P2/P3 视图、对峙增强、依赖调度器。
6. **既有设计吸收对照**（docs/v1.0/02 rd+fe design ↔ 本文）：ArtifactReader/AnchorPicker（阅读器+四级锚点+防漂移）→ §1.7 零改动沿用；FlowTree 左列 → §1.1 升级为上下分栏 SVG 泳道+动态渲染重排；NodeDetail 四区 → §1.2 五区分栏；ReviewerPillRow → §1.2② per-agent verdict 药丸；RoleThread → §1.2④ 意见区（+escalate）；fe/design 左右布局描述滞后（以 client.js pkg-28 与本文为准）。**交付物归拢**：增补稿=ui-interaction-final.md · 原型=diagrams/ui-prototype-v3.html（P7 阅读器示意已吸收）· 基线=docs/v1.0/01~03 PRD 不动。

**遗留**：无（P2/P3 暂缓为优先级裁决非遗留）。**review 通过后工单 08 关闭**，前沿=工单 09（范围确认+定稿收尾）。