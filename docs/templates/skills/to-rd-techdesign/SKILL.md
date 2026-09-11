---
name: to-rd-techdesign
description: 把产品需求（product_doc）+ 架构方向综合成 RD 技术设计文档（rd_design）——不访谈、只综合；接口契约以 api.proto 为真相源，本 skill 承载标准权威。产出自测方案与契约。用于 rd 产出型动作 backend-design.produce（exec）。
disable-model-invocation: false
---

# to-rd-techdesign

产出《RD 技术设计文档 rd_design》（`docs/{{req-version}}/{{req-name}}/rd/design.md`）。**不访谈用户、不重新收集需求**——把已对齐的材料（product_doc、architecture 方向）转成可评审、可交给 RD 开发 / FE 对接 / QA 联调的技术设计。rd_design 是**契约层**：接口契约、数据模型、模块划分要被下游直接消费。

> **标准权威说明**：本 skill 承载章节骨架 / required / purpose / 质量门禁 / 准出 / 接口协议标准的**权威版本**。原 `backend-design-doc-template.md` 已于 v0.29 删除（D3 拍板，semantica 47c53ed9）；章节骨架以本 skill 必填节为唯一权威，成品示例见 `docs/templates/backend-design-doc-demo.md`。

## 依赖（从 my_task 的 inputs 取）

| 输入 | 何时有 / 用于什么 |
|---|---|
| `product_doc` | 功能需求 / AC、目标、范围（业务 flow 必给） |
| `architecture`（use_threads=false 按需取） | §3 技术路线与架构一致、§4 备选取舍对照 |
| `brd`（tech flow 场景） | inputs 动态注入按流程（tech 拿 brd / business 拿 product_doc）——读哪个在 my_task 明示 |
| `rd_design`（prev 覆盖） | revise（exec 重入）时在上一版基础上修订 |

> ✅D5：**必须写明本 skill 的上下文依赖来源**（跨流程动态注入）。实际读 product_doc 还是 brd，以 my_task 的 inputs 引用为准，逐个校验，缺哪个读哪个来源，缺料不能臆造。

## Focus（必读/盯住的上游章节——本 skill 读 to-prd 的 product_doc（或 tech 流读 to-brd 的 brd）产 rd_design，是 to-fe-techdesign 的上游）

产出 rd_design 前**逐节读清并盯住**以下上游内容，缺任一即显式声明（gap-naming），不许跳过：

| 上游来源 | 必读章节/字段 | 产出到 rd_design 的哪节 |
|---|---|---|
| product_doc §5 功能需求（含 AC） | 功能点 / AC——**接口与数据设计的依据** | §3.5 接口设计、§3.6 数据与存储设计（逐 AC 覆盖） |
| product_doc §2 目标与成功指标 | 目标 / 护栏 | §1 背景与动机、§2 目标与非目标（对齐 prd 目标） |
| product_doc §3 范围与非范围 | 做哪些 / 明确不做 | §2.2 非目标、§3 总体方案设计（范围落方案） |
| product_doc §4 用户与场景 | 用户角色 / 主路径 | §1.1 动机场景（用户故事） |
| architecture（版本方向） | 架构约束 / 模块边界 | §3.2 整体架构图、§4 备选方案与取舍 |
| brd §6 遗留（tech flow 时） | 待调研约束 | §6 风险与未决问题 |

> 产出后到下游的交付口径：下游 **to-fe-techdesign** 读本 rd_design 的 **§3.5 接口设计（契约）** 与 **§3.6 数据与存储设计**——方法/字段/错误码必须让 FE 可直接消费、不自造。接口/数据节章节名稳定、可被逐节引用，否则下游取料断链。

## 参考材料

- **成品示例**：`docs/templates/backend-design-doc-demo.md`（填充示例，结构与本 skill 必填节一致）
- **契约真相源**：项目 proto 目录（如 `docs/api/v1/api.proto`，按架构约定定位）——接口的完整权威定义
- 已对齐的评审/CR 意见（wb_comment 按属性过滤，revise 时必取）

## 接口契约真相源与序列化约定（普适纪律，rd_design 必须遵守）

rd 侧是**契约唯一真相源 owner**：新增/修改接口消息、字段、错误码都以 rd_design（及其指向的 proto）为准，FE 只消费、不自造。design.md 的接口节 = 对契约真相源的**索引 + 本模块增量清单**，不是平行第二份契约。

- **接口增量清单而非全量内嵌**：§3.5 列出本模块**新增/改动**的接口与消息（方法、路径、字段、错误码），并写清相对契约真相源的增量；未改动的既有契约不重复展开，只给指针。FE 据此对接。
- **序列化约定（与本项目契约真相源一致）**：protojson（`EmitUnpopulated`：零值字段可见、出参字段恒显式）；`int64`（时间戳 unix 毫秒等）JSON 出参为**字符串**、入参可字符串或数字；枚举态字段用 `string` 而非 enum；统一失败响应 `{ok:false, error, code, blocked?, missing?}`（ErrorResp 同形）；结构化载荷一律 string 承载 JSON 文本。design 里的接口描述与这些约定一致，不另造编解码口径。
- **命名约定**：请求消息 `XxxReq` / 响应消息 `XxxResp`；业务 id 语义命名（如 `requirement_id`）。
- **方法/路径/鉴权/错误码**：每接口含方法 + 路径 + 入参 + 出参 + 错误码，FE 可据此直接对接（不依赖口头补充）。

## Process

1. **锁定 reader 基线（grounding）**：写 §1 前确认读者（RD 开发 / FE / QA / Architect / CR）进门已有 product_doc（需求与 AC）与架构方向。文中**新引入的技术术语**（第三方库、协议、领域模型名）**必须先定义再使用**；数据表、接口、模块名首现处给界定，禁止裸用后文才解释。术语复用 product_doc / CONTEXT 受控词，不自造同义。

2. **对齐必填节（标准以本 skill 为权威）**：rd_design 必填节 = 方案概述与目标 / 动机场景 / 技术路线与架构一致 / 接口设计(契约) / 数据设计 / 模块划分与实现要点 / 自测方案 / 工期与依赖 / 风险与回滚。章节名以本节清单为准，逐节落文。

3. **按本 skill 骨架逐节撰写**，严格对照必填节章节名与 purpose，不自行重排、不自造小标题：
   - §1 背景与动机 + §1.1 动机场景（用户故事）：对齐 prd 目标，讲清「设计在解决谁的什么问题」。
   - §2 目标与非目标。
   - §3 总体方案设计（核心）：系统上下文图 / 整体架构图 / 关键流程 / 模块与依赖 / **§3.5 接口设计（契约）** / §3.6 数据与存储设计。
   - §4 备选方案与取舍：给取舍理由（为什么选 A 否 B），可辩护。
   - §5 非功能设计（性能/稳定性/安全/灰度降级/可观测性 按需）。
   - §6 风险与未决问题 + §7 测试与验收 + §8 排期与任务拆分。
   - §9 参考资料。
   - **接口设计（契约）** 落在 §3.5：写**本模块新增/改动的接口增量清单**（方法/路径/入参/出参/错误码，对齐序列化约定），并对既有契约给真相源指针；不内嵌全量、不自造第二份契约。FE 据此直接对接。

4. **接口面四原则（进 §3.5 门禁，codebase-design 词汇）**：
   - **小接口大实现（deep module）**：接口暴露面窄、行为藏内部；删掉这个接口，系统损失可数。
   - **依赖传入不创建（dependency-in）**：接口所需依赖由调用方传入，不在接口内 new。
   - **返回值无副作用**：调用只返回结果，不改系统状态。
   - **seam 放行为边界**：接口放在可观测行为分界处（one adapter=hypothetical seam / two=real）。
   自检对照（severity major）：每个对外接口过这四条，不满足就在 §9.3 关键决策或批注记录理由。

5. **形态辩护（format arguments）**：prose / list / table 有意识选择——
   - 同形重复 3+ 次同字段 → 表（接口清单、数据表字段、模块划分用表）。
   - 系统上下文/架构 → 图（Mermaid 或 diagram），不是纯文字。
   - 取舍理由、演进说明 → 散文。
   - 写不下去改形态，不靠堆散文掩饰。

6. **证据缺失显式化（gap-naming）**：某节需证据/材料而手上没有时——
   - 不得编造接口字段、数据类型、性能数值；
   - 写「**证据缺失：待补 <XXX>**」+ 去向（问上游 / 待调研 / 依赖未定），并入 §6 风险与未决问题；
   - 影响必填节成立性时降级声明（见 §自检）。

7. **落盘 + 自检**：写到 `docs/{{req-version}}/{{req-name}}/rd/design.md`。按本 skill「自检」节逐条自检，通过后 `wb_add_artifact` 登记 + `wb_comment` 请评。

## 自检（完成前逐条过，任一 blocker 不过不许提评）

> **准出（契约可被下游无脑消费）**：下列 blocker 均为准出项——不满足即打回，不得带缺提评。

按本 skill 下列自检项逐条核对，至少覆盖：
- [ ] （blocker）接口**与契约真相源（api.proto / 既有契约）逐条一致**：新增走增量声明，改动显式记录，无与真相源冲突的自造方法/字段
- [ ] （blocker）接口描述含方法/路径/入参/出参/错误码，且序列化口径（protojson / int64→string / ErrorResp 同形 / string 枚举）一致，FE 可据此直接对接（无口头依赖）
- [ ] （blocker）逐条对照 prd 功能点检查接口设计无遗漏，且与架构方向约束一致
- [ ] （blocker）数据模型覆盖 prd 关键实体，字段类型/必填/约束完整
- [ ] （major）接口面四原则逐条核对，deep module / dependency-in / 无副作用 / seam
- [ ] （major）**grounding**：文中所有新技术术语已定义或指向定义，无裸用
- [ ] （major）**gap-naming**：缺料处已写「证据缺失：待补 X」或降级声明，未编造字段/数值
- [ ] 已按章节骨架核对必填节全部落文（章节名稳定，供 to-fe / FE 开发 / QA 逐节引用）

## 边界 / Out of scope

- 不重新访谈、不向用户收集需求。
- 不写代码 / 不做前端交互设计（那是 to-fe-techdesign / 开发动作的活）。
- 不自行改需求：需求来自 product_doc/brd；越界点写 §6 或批注待上游决策。
