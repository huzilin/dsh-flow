---
type: grilling
blocked_by: [03, 07]
claimed_by: agent session (2026-09-07, closed 2026-09-08)
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

## Answer（2026-09-08 产出；2026-09-08 用户 review 通过，关闭）

产出 = `docs-tmp/ui-interaction-final.md`（交互层 v3 增补稿，diff 层）+ `docs-tmp/diagrams/ui-prototype-v3.html`（原型六面板，已 review）：

1. **模块 II**：流程树 scope 动态渲染重排（表达式∩需求 scope，重编号不占位，单侧不保留泳道）；节点详情四区→**五区分栏**（状态卡双属性/药丸 per-agent/产物不变/意见+escalate/拍板卡三态）+ 会话 tab（D6）；拍板卡三态规格（进度只读→agreed 迎宾呈现→按钮组+跳转）；冲突卡上升（取代 override）；会话通道按钮（绑定角色动态）；对齐节点会话主视图+完成对齐单按钮。
2. **模块 I**：scope 徽标、dependencies 角标、依赖树视图（动态加载）、并发度配置入口（长期预留展示）。
3. **模块 III/IV**：收件箱条目 v3（七类映射基线六类：待拍板/澄清任务/重表态请求/悬置催办 5min/需你介入 10min/冲突上升/会话消息）；TODO=投影不变。
4. **§4 UI↔数据对应表**：每组件消费字段与工具。
5. **暂缓项**：P2/P3 视图、对峙增强、依赖调度器。
6. **四区域完整吸收记录（2026-09-08）**
   - **源码**：client.js 829 行（详情五块纵排/阅读器/版本管理弹窗/变更历史/流程图 SVG——全部吸收）；host.js 534 行（22 RPC 全名单+描述——v5 契约 delta 已记：my_task 旧版待升级/review pass·block→agree·unagree/decide·override 移出/reopen roundCount>5 通知为扫描器雏形）
   - **tmp/**：arch_design 拆分前 9 章旧文档归档（已加历史归档横幅）+ **架构可视化图**（2048 dark/1440 light·dark PNG + screenshots 内嵌分享页）——五层链路/UI 与工具同一领域逻辑/信任边界与 v3.1 一致；过时点=「wb_*×22/11 表」（已演进 v0.29/22 工具），图待重渲染（实现期）
   - **docs-tmp/**：10 份历史文档（横幅+追加废弃指针已全量覆盖）；diagrams/business-flow.json（流程图数据，节点结构与 v3.1 一致性随 seed 重写同步）
   - **docs/**：三模块 PRD/RD/FE（基线全部通读，吸收对照见 Answer 6）；diagrams（runtime html/json 同 tmp 归档性质）
7. **既有设计吸收对照**
7. **完整阅读记录（2026-09-08 补齐，用户质询版本管理/tag 遗漏后）**：①host.js 22 RPC 全名单核对——my_task 为 pkg-41 旧版（sessionsByNode/how 语义），v5 契约升级已在 data-tools；②版本管理交互全吸收（状态机单向/废弃联动需求已取消/发布软约束 bugfix 场景/删除引用禁止/四色进度条）→ §2 零改动沿用；③**tag 更正（用户澄清）**：tag=类别标签（【需求任务】等）——PRD 新建字段/client 类别徽标/host category 取值（需求任务/规划审批任务/版本规划任务）**基线已有，零改动沿用**；v3 增补=类别↔流程类型绑定（需求任务→business/tech/task，规划/版本规划→plan，scope_options 随类别派生）→ ui-interaction §2；④client.js 详情区现状=meta/产物/药丸/门禁/变更历史五块纵排（变更历史保留与五区分栏并存）；⑤owner 提示规则（架构→Architect/纯后端→RD/纯前端→FE）与 scope 三值呼应。
6. **四区域完整吸收记录（2026-09-08）**
   - **源码**：client.js 829 行（详情五块纵排/阅读器/版本管理弹窗/变更历史/流程图 SVG——全部吸收）；host.js 534 行（22 RPC 全名单+描述——v5 契约 delta 已记：my_task 旧版待升级/review pass·block→agree·unagree/decide·override 移出/reopen roundCount>5 通知为扫描器雏形）
   - **tmp/**：arch_design 拆分前 9 章旧文档归档（已加历史归档横幅）+ **架构可视化图**（2048 dark/1440 light·dark PNG + screenshots 内嵌分享页）——五层链路/UI 与工具同一领域逻辑/信任边界与 v3.1 一致；过时点=「wb_*×22/11 表」（已演进 v0.29/22 工具），图待重渲染（实现期）
   - **docs-tmp/**：10 份历史文档（横幅+追加废弃指针已全量覆盖）；diagrams/business-flow.json（流程图数据，节点结构与 v3.1 一致性随 seed 重写同步）
   - **docs/**：三模块 PRD/RD/FE（基线全部通读，吸收对照见 Answer 6）；diagrams（runtime html/json 同 tmp 归档性质）
7. **既有设计吸收对照**（docs/v1.0/02 rd+fe design ↔ 本文）：ArtifactReader/AnchorPicker（阅读器+四级锚点+防漂移）→ §1.7 零改动沿用；FlowTree 左列 → §1.1 升级为上下分栏 SVG 泳道+动态渲染重排；NodeDetail 四区 → §1.2 五区分栏；ReviewerPillRow → §1.2② per-agent verdict 药丸；RoleThread → §1.2④ 意见区（+escalate）；fe/design 左右布局描述滞后（以 client.js pkg-28 与本文为准）。**交付物归拢**：增补稿=ui-interaction-final.md · 原型=diagrams/ui-prototype-v3.html（P7 阅读器示意已吸收）· 基线=docs/v1.0/01~03 PRD 不动。

**遗留**：无（P2/P3 暂缓为优先级裁决非遗留）。**review 通过后工单 08 关闭**，前沿=工单 09（范围确认+定稿收尾）。