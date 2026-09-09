# 交互层 v3 增补稿（模块 I / II / III·IV · diff 层）

> **定位**：既有 PRD（`docs/v1.0/01-module-board`、`02-module-flow-tree`、`03-module-inbox-todo`）为**基线，不动**；本文 = v3 编排模型（capability-model-final v3.1）+ scope 体系 + 评审轮 per-agent 语义在 UI 上的**增补层**——只写「改什么、加什么」，未提及处基线照旧。
> **原型**：`docs-tmp/diagrams/ui-prototype-v3.html`（六面板已 review：P1/P4/P5/P6 确认，P5/P4/P6 已补节点跳转；P2/P3 视图暂缓开发——设计保留实现靠后）。**P8–P11（2026-09-09 补齐，待 review）**：模块 I 看板/需求池（P8 四泳道+scope 徽标+列表视图+RequirementModal）、版本管理（P9 状态机+依赖树按需拉取+并发调度预留）、模块 III 收件箱（P10 通知七类）、模块 IV TODO（P11 五分类投影）；跳转闭环已接线（看板↔节点详情、拍板卡↔收件箱双通道、冲突卡/TODO→详情）并修复 P1 scope 切换的 flowhint 缺失。同日对 PRD/fe-design 全量扫缺，P1 补会话 tab（⑥）/详情补遗卡（⑦）/会话找 X/缺陷闭环示例——暴露的设计空洞定稿见 **§6**。
> **模型依据**：两层属性状态机（node.state 4 相位 + per-agent 评审轮 reviewing/agree/unagree + collect 内置聚合）；scope 渲染集合表达式（rd|full / fe|full / full / '' / rd|fe）。

---

## 1. 模块 II · 流程树 + 节点详情（增补）

### 1.1 流程树 → SVG 横向流程图（沿用现有 renderFlow 形态，新增动态渲染）

| 增补点 | 规格 |
|---|---|
| **scope 动态渲染重排** | 节点集 = `flow_def_nodes.scope 表达式 ∩ requirement.scope`；过滤后**重新分组排布 + 重编号**（被裁节点不占位）；并行组仅当两侧节点都可见时保留泳道/分叉汇聚，单侧可见并入主链（不并发不保留并发设计） |
| **节点行双属性** | 节点方块 = 状态点（沿用）+ 节点名（渲染序号）+ `↩N`（沿用）+ **相位角标**：exec 期=当前动作名；review 期=collect 聚合快照「表态 N/M · unagree N」（派生自 review_results，非节点级状态） |
| **渲染示例** | full=0-12 全渲染（8∥9 泳道）；rd=9 节点单主链（0→1→2→3→4→5→8→11→12）；fe 对称；QA 验收仅 full |
| 交互 | 点击节点 → 右列详情（不变）；横向滚动（不变） |

### 1.2 节点详情：右列四区 → **五区分栏**（吸收原型四分栏 + 产物区回归）

| 区 | 内容 | v3 变化 |
|---|---|---|
| ① 状态卡（新增，取代元信息卡的状态部分） | **双行**：执行行（node.state 徽标 + 当前挂载动作名）；审批行（collect 聚合快照：agree N · unagree N · reviewing N；无评审位恒 '-'） | 派生自 review_results，非节点级状态 |
| ② 评审状态行 | per-agent 表态：各评审位 **reviewing / ✅agree / ⛔unagree** + open 数；点药丸展开角色对话区（评论操作矩阵不变） | 药丸主态=verdict（表态看药丸）；轮内完整视图（时间线+问题列表）**暂缓开发** |
| ③ 产物区 | 文件行 + 💬N + link 外链（**全部不变**）——点击 → 既有产物阅读器（四级锚点评论） | 无变化 |
| ④ 意见区 | 评论操作矩阵（状态×操作者，不变）+ **escalate 标记** + 冲突卡（见 1.4） | +escalate |
| ⑤ 门禁 / 拍板卡 | 三态（见 1.3） | 重构 |
| 会话 tab | 节点关联 sessions 列表（D6：历史/进行中 subagent 对话，不经中转） | 新增 |
| 变更历史（基线 histPanel 保留） | 需求字段审计自然语句（与流程评审正交，仍挂详情页） | 无变化 |

**节点激活示例（exec 相位）**：状态卡执行行显示当前动作（如「C4 开发 · implement-techdesign · 编排进度 2/5」——goal/tasks 投影，并发度配置入口挂版本管理）。

### 1.3 门禁 / 拍板卡（三态规格，user_gate 节点）

| 态 | 呈现 | 操作 |
|---|---|---|
| 进度态（reviewing/clarifying/confirming） | 只读：表态 N/M（agree/unagree 计数）+ 工单统计（open/pending-confirm/resolved）| 无按钮（拍板置灰） |
| **agreed → 迎宾呈现态** | 门禁汇总（各评审位结论 + resolved 摘要 + 产物链接 + 建议）——§6.8 owner 报送内容 | 通过 / 打回（必填理由，走 reject 结论协议）/ 会话找 owner / **↗ 节点详情** |
| 收件箱联动 | 「待你拍板」通知（点此跳节点详情） | 双通道：收件箱=系统投递、迎宾=owner 呈现 |

### 1.4 冲突卡与上升（取代 override 强权）

- 澄清/复核循环不收敛（reopen 拉锯）→ 任一方 `escalate` 标记 → 评论区聚合**冲突卡**：分歧点 + **正反双方结论汇总两栏对照** + issue 线程引用 + 「上升裁决」按钮 + **「↗ 节点详情」跳转**。
- 上升裁决 session：reviewer 主持（开场白=分歧点+双方立场）→ 用户结论 → **回填冲突 issue**（评论区可见，数据层留痕）→ reviewer 重新表态（回 reviewing）→ 正常判据链恢复。
- **override 强权移除**（wb_override 移出 agent 清单；冲突走上升仲裁）。
- 死循环兜底：轮内不收敛走 escalate；User 与评审者反复分歧走线下（流程内不建模）。

### 1.5 按需会话通道按钮（节点绑定角色，动态 scope）

- 节点详情/门禁卡常驻「会话找 X」按钮（X=绑定角色）：**full=qa / 单侧（fe-only=fe、rd-only=rd）**——用户验收报障直连开发执行者。
- 点击 → plugin 泛用接口拉起该角色会话（开场白模板随绑定配置）→ 对话 → 报障澄清 → 结论回填 issue。

### 1.6 对齐节点详情（交互型：会话主视图）

- 右列以**会话为主体**：顶部状态条（`entry · 等待你的输入` + 响应者 PM + 可用交互提示 /wayfinder、grill-me）；会话视图（PM 对话轮）。
- 底部单按钮「**完成对齐**」（User 确认 → to-brd 生成落盘 → 迁移产品设计）+「↗ 节点 1 详情」——**按推演无 BRD 预览审批步**（brd 在节点 2 产物区可见）。
- 用户确认 = 唯一完成信号；程序兜底 = 周期提醒（扫描器）。

### 1.7 产物阅读器（既有设计吸收 · 零改动）

基线：`02-module-flow-tree/fe/design.md`（组件 ArtifactReader/AnchorPicker）+ `rd/design.md`（行级锚点防漂移）。v3 全程不变，仅列规格备查：

- 弹窗飞书式 ~72vw×85vh；轻量 Markdown 渲染 + 行号（1-based）；底部固定输入框（文案随选中态变化）。
- 锚点四级展示：line=行下方；range=末行一次 + `↔A~B`；block=代码块下方；document=正文末尾。
- **行级锚点防漂移**：按 `lineText` 反查内容相同行重锚定；未命中就近展示标「⚠」，用户确认时更新。
- 评论操作矩阵（状态×操作者）与「不可编辑/删除、误发 reply 标注作废」规则全部沿用。

**滞后修正注记**：fe/design.md 的「左列流程树+右列四区」布局描述滞后于实现（client.js pkg-28 已改上下分栏+横向 SVG 泳道）——布局以本文 §1.1 与 client.js 现状为准；fe/design 其余交互规格（阅读器/锚点选择/对话区）继续有效。

---

## 2. 模块 I · 看板 / 需求池 / 版本管理（增补）

| 增补点 | 规格 |
|---|---|
| **scope 徽标** | 需求卡/列表行/详情补 scope 徽标（full / rd / fe，三色区分）——需求 scope 三值 |
| **dependencies 角标** | 需求卡/详情补依赖数（`依赖 ×N`）；点开显示前置需求名列表（flows.dependencies） |
| **依赖树视图** | 版本管理弹窗内新增「依赖树」入口：按版本动态加载依赖树（plugin 内置分层派生，不对用户显示层级；**分析师按需拉取，非常驻**） |
| **并发度配置入口** | 版本管理内「并发调度」区（长期预留展示：并发度=正在工作的需求数、就绪池规则说明）——MVP 人工卡点（节点 0 user_gate=1），调度器为长期 effort |
| 版本管理基线交互（**完整吸收，零改动**） | 版本状态机 待启动→进行中→已发布（单向）+ 任意阶段废弃=终态（联动其下需求全部已取消，不写 history）；发布软约束（有未完成需求仅提示不阻止——bugfix 场景）；删除有引用禁止；进度四色条+计数；行操作 推进/废弃/改名 | 
| **类别标签（tag，既有吸收）** | 需求卡/列表/详情的类别徽标（【需求任务】【规划审批任务】【版本规划任务】）——基线已有，零改动沿用；**类别↔流程类型绑定**：需求任务→business/tech/task（scope 三值）、规划审批任务/版本规划任务→plan（无 scope）；scope_options 随类别派生 |
| **owner 提示规则**（新建需求，scope 辅助） | 架构→Architect；纯后端→RD；纯前端→FE——与 scope 三值呼应，新建弹窗按类别/侧提示默认负责人 |
| **tag** | 现有 docs/代码**无 tag 概念**（全库检索零命中）——非遗漏；是否引入版本/需求 tag 为新需求，待 User 裁决（默认不引入） |
| **变更历史面板** | 既有 histPanel（自然语句审计）保留，与五区分栏并存（需求字段审计与流程评审正交） |

---

## 3. 模块 III/IV · 收件箱 / TODO（增补）

### 3.1 收件箱条目类型 v3（对齐 §6.8 通知规程 + 扫描器参数表；基线六类映射如下）

| v3 类型 | 触发 | 报送责任人 | 基线映射 |
|---|---|---|---|
| **待你拍板** | 门禁 agreed + user_gate | owner（迎宾呈现+收件箱双通道） | 「待评审门禁」 |
| **澄清任务** | collect 判 unagree | owner（澄清 issue 清单） | 「待界定新范围」+「产出者反驳」语义合并 |
| **重新表态请求** | 澄清后 plugin 发回 reviewer；**修复提交（to-fix→fixed）待复核同走本通道（retest，2026-09-09 补：基线「产出修订完成」在 v3 映射中原本遗漏，见 §6 N1）** | reviewer/提评者（followup/复用通道） | 新增（原「要求补充答复」近义；基线「产出修订完成」并入此通道） |
| **悬置催办** | 已结束 5min 未表态 | reviewer（followup 文案：遇卡点上报） | 新增（扫描器） |
| **需你介入** | 10min 仍未推进 | plugin → User | 新增（扫描器） |
| **冲突上升裁决** | escalate 标记 | reviewer 主持 | 新增（冲突响应机制） |
| **会话消息** | 按需通道/对齐未读 | 绑定角色 | 「对话待回复」 |

### 3.2 TODO 列表（模块 IV）

- 基线规则不变（= 收件箱操作投影，同源同筛选）；v3 增分类：待拍板 / 待澄清 / 待重表态 / 待复验（retest）/ 冲突待裁决。
- 点击跳转：对应节点详情（新增 scope 过滤后节点定位）。

---

## 4. UI ↔ 数据/工具对应表（每组件的消费契约）

| UI 组件 | 消费数据（字段） | 工具/接口 |
|---|---|---|
| 流程树 | flow_def_nodes（scope 表达式/node_type/gate/user_gate）∩ requirement.scope；flow_nodes（state/review_results 派生聚合/rolled_back_count/last_decision） | wb_get_requirement |
| 节点状态卡 | flow_nodes.state + review_results 派生（agree/unagree/reviewing 计数）+ 当前挂载动作（node_actions ∩ state） | wb_my_task（currentPhase+activeActions） |
| 角色药丸 | flow_nodes.review_results（per-agent reviewing/agree/unagree + open 数派生） | wb_review（agent 上报写此） |
| 意见区/冲突卡 | issues（state 含 to-fix/fixed、escalate、loop_count、replies 回填） | wb_comment/wb_resolve/wb_reopen(+escalate) |
| 拍板卡 | review_results + last_decision + user_gate | User 拍板 → service 层 wb_decide 语义 |
| 会话 tab/按钮 | sessions 表（按 node_id/requirement_id） | plugin 泛用拉起接口（绑定角色动态） |
| 依赖树视图 | flows.dependencies + requirement（priority） | plugin 动态加载接口 |
| 收件箱/TODO | 通知枚举（§3.1）+ tasks 投影 | wb_list_inbox / wb_task_update |

---

## 5. 暂缓项（实现靠后，设计保留）

- **P2 评审轮视图 / P3 进度面板**（per-agent 完整时间线+问题列表）：复杂度高、需求不强烈——数据结构（review_results）不变，UI 后置。
- 冲突对峙增强（顺序质询轮/多 agent 同会话）：待 DSH 能力演进。
- 依赖调度器：长期 effort（MVP 人工卡点）。

---

## 6. 原型补齐定稿注记（2026-09-09 · 对照 PRD/fe-design 全量扫缺时暴露的设计空洞 · 可推翻）

> 背景：P8–P11 补齐后对既有技术文档（三份 PRD + 三份 fe/design.md + 本文）做双向核对——A 类「文档已定但原型缺」直接补原型（P1 会话 tab/详情补遗卡/会话找 X/缺陷闭环示例、P8 RequirementModal）；B 类「文档未定、原型必须给出形态」的空洞在下方定稿。暂缓项（§5）不受影响。

| # | 空洞 | 定稿 |
|---|---|---|
| **N1** | 基线六类「产出修订完成」（待复验）在 §3.1 v3 七类映射中**遗漏**，而 §3.2 TODO 五分类含「待复验（retest）」——通知无来源 | 并入「重新表态请求」：触发扩为「澄清回填**或修复提交（to-fix→fixed）**后，请 reviewer/提评者回到评审循环」；§3.1 表已就地标注 |
| **N2** | 七类通知逐类的「用户动作/跳转目标」未定义（PRD 只说「逐条可回应」） | 待你拍板→迎宾拍板卡；澄清任务→节点详情意见区；重新表态请求（含 retest）→节点详情；悬置催办→reviewer followup（不进 User 列表）；需你介入→节点详情；冲突上升裁决→冲突卡；会话消息→对应会话视图。收件箱按接收人投递，User 列表=报送对象含 User 的子集（原型 P10 为演示全量七类不区分接收人） |
| **N3** | TODO 五分类 ← 七类通知的投影映射未定 | 待拍板←待你拍板；待澄清←澄清任务；待重表态←重新表态请求（澄清子型）；待复验←重新表态请求（retest 子型）；冲突待裁决←冲突上升裁决；悬置催办/需你介入为提醒类，不产生 TODO |
| **N4** | 新建/编辑需求弹窗（RequirementModal）的 scope 字段形态未定 | select 三值（full/rd/fe）；类别=规划审批/版本规划时禁用置 —（scope_options 随类别派生）；标题空→保存禁用+红框（基线 AC） |
| **N5** | 会话 tab（D6）条目字段与打开行为未展开（正文仅一句话） | 条目=绑定角色 + 状态（进行中/已结束）+ 最近消息摘要（已结束=结论回填摘要）+ 时间；点击 → plugin 泛用接口打开会话视图（不经中转）；「会话找 X」按钮常驻详情页 detailtop 与门禁卡（X 随节点绑定角色动态：full=qa / fe-only=fe / rd-only=rd） |
| **N6** | 变更历史面板（histPanel）与描述展示在 v3 上下分栏详情页的挂载形态未定 | 详情页底部折叠卡，与五区分栏并存：描述 Markdown（>4 行折叠 150px + 展开全部 ▾）+ 元信息行（负责人/进入时间/回滚次数/职责）+ 变更历史（时间倒序自然语句）——见原型 P1 卡 ⑦ |
| **N7** | QA 缺陷闭环（02 PRD §5.2 验收节点特有交互）原型无示例 | P1 意见区补缺陷 thread 示例：缺陷文档登记产物 + domain=rd\|fe 自动指派；resolve 带理由=异议表达（不存在「RD/FE 打回 QA」）；QA confirm/reopen 最终裁定；评论 reopen（单条缺陷）≠ wb_decide reject(targetBranch)（验收整体打回、部分回滚） |
