---
name: to-fe-techdesign
description: 把产品需求（product_doc）+ RD 契约（rd_design）综合成前端技术设计文档（fe_design）——不访谈、只综合；消费 rd 契约，逐字段对齐、不自造。标准以本 skill 为权威。用于 fe 产出型动作 frontend-design.produce（exec）。
disable-model-invocation: false
---

# to-fe-techdesign

产出《前端技术设计文档 fe_design》（`docs/{{req-version}}/{{req-name}}/fe/design.md`）。**不访谈用户、不重新收集需求**——把已对齐的材料（product_doc、rd_design 接口契约）转成可评审、可交给 FE 开发 / QA 联调的交互与实现设计。fe_design 是**消费契约**：页面/组件、交互、状态管理、接口消费口径要被下游直接消费。

> **标准权威说明**：本 skill 承载章节骨架 / required / purpose / 质量门禁 / 准出 / 接口消费标准的**权威版本**。原 `frontend-design-doc-template.md` 已于 v0.29 删除（D3 拍板，semantica 47c53ed9）；章节骨架以本 skill 必填节为唯一权威，成品示例见 `docs/templates/frontend-design-doc-demo.md`。
> **契约 owner**：rd 是唯一契约真相源，**FE 只消费、不定义**。fe_design 的接口消费 = 对 rd_design 契约的逐字段对齐索引，不自造方法/字段。

## 依赖（从 my_task 的 inputs 取）

| 输入 | 何时有 / 用于什么 |
|---|---|
| `product_doc` | 页面/功能清单、AC、用户与场景（范围来源） |
| `rd_design`（接口契约，前置已定） | §接口对接（消费 rd 契约）——方法/字段/错误处理必须与 rd 一致 |
| `architecture`（use_threads=false 按需取） | 交互方案对照版本架构方向 |
| `fe_design`（prev 覆盖） | revise（exec 重入）时在上一版基础上修订 |

> ✅D5：必须写明本 skill 的上下文依赖来源。实际流程可能没有 product_doc（task 流走 rd_design），以 my_task 的 inputs 引用为准逐个校验，缺哪个读哪个，缺料不能臆造。

## Focus（必读/盯住的上游章节——本 skill 读 to-prd 的 product_doc + to-rd-techdesign 的 rd_design 产 fe_design）

产出 fe_design 前**逐节读清并盯住**以下上游内容，缺任一即显式声明（gap-naming），不许跳过：

| 上游来源 | 必读章节/字段 | 产出到 fe_design 的哪节 |
|---|---|---|
| rd_design §3.5 接口设计（契约，to-rd-techdesign 产出） | 方法/路径/入参/出参/错误码——**逐条消费，无自造契约** | §5 接口设计（前后端契约） |
| rd_design §3.6 数据与存储设计 | 数据字段/类型 | §4 数据与状态管理（数据流映射字段） |
| product_doc §4 用户与场景（§4.2 主路径） | 主路径 / 交互 | §2 页面与交互设计（§2.2 交互流程 / §2.3 状态梳理） |
| product_doc §5 功能需求（AC） | 功能点 / AC | §2 页面清单、§10.2 自测用例与验收标准（逐 AC） |
| product_doc §3 范围与非范围 | 做哪些页面 | §2.1 页面清单与路由 |
| architecture（可选） | 交互方案对照 | §6 技术选型与工程化（对照） |

> 产出后到下游的交付口径：下游开发（frontend-dev）与 QA 验收读本 fe_design 的 §2 页面/交互、§4 状态、§5 接口消费口径。接口消费节必须与 rd_design 契约一致、可被逐节引用，不自造小标题，否则下游取料断链。

## 参考材料

- **成品示例**：`docs/templates/frontend-design-doc-demo.md`（填充示例，结构与本 skill 必填节一致）
- **上游契约**：rd_design 的 §3.5 接口设计（契约）与 §3.6 数据与存储设计——FE 消费的唯一契约来源
- 已对齐的评审/CR 意见（wb_comment 按属性过滤，revise 时必取）

## 接口消费口径（FE 只消费，逐字段对齐）

fe_design 的接口消费 = 对 rd 契约的**消费声明**，不是第二份契约：

- **逐条消费 rd 契约**：方法/路径/字段/错误处理与 rd_design 一致，**无自造契约**。
- **序列化口径随 rd**：protojson（零值字段可见）；`int64` 字段（时间戳/计数）前端收到的是**字符串**、发送可接受字符串或数字；枚举态字段是 `string`；失败统一走 `{ok:false, error, code, blocked?, missing?}`（ErrorResp 同形）——前端类型定义/边界态据此对齐，不自造解析口径。
- **发现 rd 契约缺**：写「证据缺失 / 需 rd 补」或回 rd 补充，**不得**在 fe_design 里自造方法/字段顶替。

## Process

1. **锁定 reader 基线（grounding）**：写 §1 前确认读者（FE 开发 / QA / CR / Architect）进门已有 product_doc（页面与 AC）与 rd_design（接口契约）。文中**新引入的交互/前端术语**（新组件、状态模型、路由名）**先定义再使用**；页面名、组件名、状态字段首现处给界定。术语复用 product_doc / CONTEXT 受控词，不自造同义。

2. **对齐必填节（标准以本 skill 为权威）**：fe_design 必填节 = 页面/组件清单与交互方案 / 动机场景 / 状态管理与数据流 / 接口对接(消费 rd 契约) / 关键路径与边界处理 / 前端自测方案。章节名以本节清单为准，逐节落文。

3. **按骨架逐节撰写**，严格对照章节名与 purpose，不自行重排、不自造小标题：
   - §1 背景与目标 + §1.1 动机场景（用户故事）。
   - §2 页面与交互设计：页面清单与路由 / 交互流程 / 状态梳理。
   - §3 组件设计。
   - §4 数据与状态管理：数据流 / 状态管理方案。
   - §5 接口设计（前后端契约）——**消费 rd 契约**：逐字段对齐 rd_design，序列化口径随 rd（见「接口消费口径」），**无自造契约**。
   - §9 任务拆分与排期 + §10 风险与验收（含自测用例与验收标准）。
   - 边界态（加载/空/错误/超时）处理说明必写。

4. **形态辩护（format arguments）**：prose / list / table 有意识选择——
   - 页面清单、路由、状态字段 → 表。
   - 交互流程 → 流程描述或图（Mermaid/sequence），不是纯字段罗列。
   - 组件设计取舍、数据流说明 → 散文。
   - 写不下去改形态，不靠堆散文掩饰。

5. **证据缺失显式化（gap-naming）**：某节需证据/材料而手上没有时——
   - 不得编造交互细节、接口字段、性能目标；
   - 写「**证据缺失：待补 <XXX>**」+ 去向（问 PM/rd / 待调研），并入 §10 风险或批注；
   - 影响必填节成立性时降级声明（见 §自检）。

6. **落盘 + 自检**：写到 `docs/{{req-version}}/{{req-name}}/fe/design.md`。按本 skill「自检」节逐条自检，通过后 `wb_add_artifact` 登记 + `wb_comment` 请评。

## 自检（完成前逐条过，任一 blocker 不过不许提评）

> **准出（契约消费可被下游无脑消费）**：下列 blocker 均为准出项——不满足即打回，不得带缺提评。

按本 skill 下列自检项逐条核对，至少覆盖：
- [ ] （blocker）逐条对照 rd_design 检查接口消费（方法/路径/字段/错误处理）一致，**逐字段对齐、无自造契约**
- [ ] （blocker）前端类型/边界态与序列化口径一致（int64 收到为字符串、枚举态为 string、ErrorResp 同形），无自造解析口径
- [ ] （blocker）逐条对照 product_doc 页面/功能清单检查覆盖，无遗漏页面与关键流程
- [ ] （major）关键路径操作步骤可复现（点到哪、出现什么），QA 可据此设计交互用例
- [ ] （major）加载/空/错误/超时等边界态有处理说明
- [ ] （major）**grounding**：文中所有新交互/前端术语已定义或指向定义，无裸用
- [ ] （major）**gap-naming**：缺料处已写「证据缺失：待补 X」或降级声明，未编造交互/字段
- [ ] 已按章节骨架核对必填节全部落文（章节名稳定，供 frontend-dev / QA 逐节引用）

## 边界 / Out of scope

- 不重新访谈、不向用户收集需求。
- 不写代码（那是 frontend-dev / 开发动作的活）。
- 不造后端契约：接口消费只对齐 rd_design；发现 rd 契约缺，写「证据缺失/需 rd 补」，不自造方法或字段。
