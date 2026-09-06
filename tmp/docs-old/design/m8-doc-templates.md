# M8：产物模板固化设计（12 个文档模板结构化定义）

> 状态：**v0.1（12 模板章节骨架 + 质量门禁定稿）；v0.23 起模板内容外置 docs/templates/*.md 文件承载（structure/quality_gates 不入库，本文件作为模板结构的人读参考/历史记录）；v0.28 QA 能力合并后《缺陷文档》模板仍存在，作为 acceptance-test 第 2 产物（qa/defect.md）的模板引用。**
> 定位：为 `document_templates` 表（m8-flow-simplify §3.6.6，v0.17 已删，改由 capabilities.products[].template_path 指向外置文件）提供可评审、可执行的结构化定义——每个模板的 `structure`（章节骨架 JSON）与 `quality_gates`（质量门禁 JSON）的**人读版 + 评审契约**。
> 关联：m8-flow-simplify.md（§2.3① wb_my_task 引用模板、§3.6.6 表结构与固化范围）、m7-doc-structure.md（doc_path 目录约定、实现-设计对照表格式）、m6-flow-templates.md（各评审门禁「各审各的」职责）。

---

## 1. 模板固化目的与作用机制

### 1.1 目的

1. **稳定产出质量**：每个角色被派活后，工具返回确定性任务（`wb_my_task`），`outputs[]` 引用模板——角色按章节骨架产出、按质量门禁自检，产出结构与检查点不再依赖 persona 里零散的「产物要点」文字。
2. **程序做路由、agent 做专业**：模板管「产出结构 + 质量门禁」，persona 只管「专业能力」。模板写清每个 section 服务哪个评审者、每条 gate 依赖哪些文档，评审者「审什么」由模板承接，不再靠 agent 人肉推理。
3. **模板演进而非重构**：调整模板 = 改 `document_templates` 一行，全局生效（不碰 persona / 不碰代码）；模板升级走 `version` 字段，历史版本可回溯。
4. **模板间互相呼应**：下游模板的章节/门禁直接引用上游模板的产出（如 rd-design 的接口/数据节是 fe-design 接口契约节的上游），评审依据链闭环（见 §3）。

### 1.2 与 wb_my_task / document_templates 的关系

```
document_templates（sqlite，12 行）
  ├─ structure      → wb_my_task.outputs[] 命中 template_id 后注入产出者：按章节骨架组织正文
  ├─ quality_gates  → 双重消费：
  │                    ① 产出者完成前按 gates 自检（check 是具体动作，ref 指明对照文档）
  │                    ② 评审者（各审各的）评审时按 gates 核对；CR/QA 验收时按结构逐节核对
  └─ owner_role     → 与 flow 节点 ownerRole 匹配，程序校验「谁产出」；doc_path 与 m7 目录约定一致
wb_my_task.dependencyDocs  ← 与 quality_gates[].ref 同源（同一套 m7 路径约定）
```

- **产出端**：`wb_my_task` 返回 `outputs[]`（label/path/template_id）→ 角色按 `structure` 逐节撰写、按 `quality_gates` 自检 → 写盘（doc_path）→ `wb_add_artifact` 登记。
- **评审端**：评审者拿到模板的 `quality_gates`，按 `severity` 分级核对（blocker 不满足则门禁不过、major 必查、minor 建议）。
- **ref 约定**：`ref` 一律写 m7 目录约定的相对路径短名（`prd.md` / `rd/design.md` / `fe/design.md` / `qa/cases.md` / `qa/test.md` / `qa/defect.md` / `qa/accept.md` / `rd/dev.md` / `fe/dev.md` / `plan.md` / `align.md` / `research.md` / `arch_design.md` / `roadmap.md` / 需求池）；`—（自含）` 表示该检查只依据本文件内容，无外部依赖。

### 1.3 12 模板速查

| template_id | 名称 | owner_role | doc_path |
|---|---|---|---|
| prd | 《产品设计文档》 | arch | docs/<版本号>/<需求名>/prd.md |
| rd-design | 《RD 技术设计》 | rd | docs/<版本号>/<需求名>/rd/design.md |
| fe-design | 《FE 技术设计》 | fe | docs/<版本号>/<需求名>/fe/design.md |
| case-design | 《用例/联调设计》 | qa | docs/<版本号>/<需求名>/qa/cases.md |
| dev-notes | 《开发说明》 | rd \| fe | docs/<版本号>/<需求名>/{rd,fe}/dev.md |
| research | 《调研文档》 | inv | docs/<版本号>/research.md |
| defect | 《缺陷文档》 | qa | docs/<版本号>/<需求名>/qa/defect.md |
| test-result | 《测试用例与验收记录》（含 §测试资产：数据构造+测试代码） | qa | docs/<版本号>/<需求名>/qa/test.md |
| acceptance | 《验收记录》 | qa | docs/<版本号>/<需求名>/qa/accept.md（v0.28 起并入 test-result，见 §2.9） |
| plan | 《方案文档》 | executor（执行者） | docs/<版本号>/<需求名>/plan.md |
| architecture | 《架构规划》 | arch | docs/<版本号>/arch_design.md |
| alignment | 《需求对齐文档》 | arch | docs/<版本号>/align.md |

> owner_role 说明：`dev-notes`、`plan` 为多角色模板，表中记逻辑归属（rd\|fe / executor），实际实例化时由 flow 节点 ownerRole 决定写哪个分支路径（dev-notes：`rd/dev.md` 或 `fe/dev.md`）。

---

## 2. 模板定义（12 节）

### 2.1 prd《产品设计文档》

- **template_id**：prd ｜ **name**：《产品设计文档》 ｜ **owner_role**：arch
- **doc_path**：`docs/<版本号>/<需求名>/prd.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 需求背景与问题定义 | ✅ | 服务 **PM**：评估需求价值/优先级（解决什么问题、值不值得做）；服务 **User**：判断需求是否真实 |
| 目标、范围与非目标 | ✅ | 服务 **PM**：评审范围是否合理；服务 **User**：拍板范围；同时是全流程下游（rd/fe/cases）的**范围约束**，不得越界 |
| 目标用户与核心场景 | ✅ | 服务 **PM**：价值判断依据（谁是受益者）；服务 **FE**：页面设计输入 |
| 产品方案（交互流程与信息架构） | ✅ | 服务 **Architect**：评估技术可行性（方案能否落地、与架构是否冲突）；服务 **FE**：技术设计输入 |
| 页面/功能清单与关键路径 | ✅ | 服务 **FE**：页面结构与关键流程依据；服务 **QA**：关键路径可测性起点 |
| 验收标准（AC） | ✅ | 服务 **User**：拍板目标是否达成；服务 **QA**：用例设计（case-design）与验收（test-result 逐条核对）的**上游** |
| 优先级与版本归属建议 | ⭕（可选） | 服务 **PM**：优先级评审参考（PM 可裁决覆盖） |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐条核对目标/范围/非目标表述无歧义，评审者无需再向作者追问范围边界 | 需求描述（wb_list_requirements） | blocker |
| 逐条核对每条 AC 均可用「通过/不通过」判定，无不可验证项（无「体验更好」类空话） | —（自含） | blocker |
| 核对页面/功能清单覆盖需求描述全部要点，逐项打勾无遗漏 | 需求描述 | major |
| 对照 arch_design.md 检查产品方案与版本架构方向不冲突 | arch_design.md | major |
| 检查优先级与版本归属给出建议值（可被 PM 覆盖） | roadmap.md（上版本路线） | minor |

---

### 2.2 rd-design《RD 技术设计》

- **template_id**：rd-design ｜ **name**：《RD 技术设计》 ｜ **owner_role**：rd
- **doc_path**：`docs/<版本号>/<需求名>/rd/design.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 方案概述与目标 | ✅ | 服务 **Architect**：快速判断方案方向；服务 **User**：方案确认入口；对齐 prd 目标 |
| 技术路线与架构一致 | ✅ | 服务 **Architect**：评审技术路线、与版本架构一致性；服务 **CR**：核对设计落地 |
| 接口设计（契约） | ✅ | 服务 **FE**：接口契约可对接评审；服务 **QA**：联调口径；是 **fe-design 接口对接节**的上游 |
| 数据设计（模型/存储/字段） | ✅ | 服务 **FE**：数据结构契约来源；服务 **QA**：构造测试数据；服务 **CR**：实现核对 |
| 模块划分与实现要点 | ✅ | 服务 **RD**：自评设计完整性；服务 **CR**：设计落地核对 |
| 自测方案 | ✅ | 服务 **QA**：联调口径与用例设计的**后端口径**来源；服务 **RD 开发**：rd/dev.md 自测依据 |
| 工期与依赖 | ✅ | 服务 **PM**：评审工期影响与跨需求依赖 |
| 风险与回滚 | ✅ | 服务 **CR**：回归风险；服务 **QA**：回归影响 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐条对照 prd.md 功能点检查接口设计无遗漏，且与 arch_design.md 架构约束一致 | prd.md, arch_design.md | blocker |
| 核对数据模型覆盖 prd 关键实体，字段类型/必填/约束完整 | prd.md | blocker |
| 验证每条接口含方法/路径/入参/出参/错误码，FE 可据此直接对接（不依赖口头补充） | —（自含） | blocker |
| 检查自测方案可执行：QA 在用例设计（qa/cases.md）与验收时可按此设计/核对联调口径 | prd.md（口径来源），qa/cases.md（验收核对） | major |
| 抽查模块划分覆盖 prd 全部功能点，无设计空白 | prd.md | major |
| 核对工期给出明确数值、依赖列出具体项，PM 可据此评估排期 | 需求池（并行需求） | major |
| 检查风险/回滚给出具体触发条件与动作，非空话 | —（自含） | minor |

---

### 2.3 fe-design《FE 技术设计》

- **template_id**：fe-design ｜ **name**：《FE 技术设计》 ｜ **owner_role**：fe
- **doc_path**：`docs/<版本号>/<需求名>/fe/design.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 页面/组件清单与交互方案 | ✅ | 服务 **QA**：评审交互可测性（每个页面/组件可操作）；服务 **FE 开发**：落地依据 |
| 状态管理与数据流 | ✅ | 服务 **QA**：验证状态变化可观察；服务 **CR**：核对实现 |
| 接口对接（消费 rd 契约） | ✅ | 服务 **QA**：联调口径；服务 **CR**：核对接口消费与 rd/design.md 一致；上游 = **rd-design 接口/数据节** |
| 关键路径与边界处理 | ✅ | 服务 **QA**：评审关键路径可验证（登录/提交/异常）；服务 **CR**：回归风险 |
| 前端自测方案 | ✅ | 服务 **QA**：评审前端自测口径；服务 **FE 开发**：fe/dev.md 自测依据 |
| 兼容性与风险 | ⭕（可选，涉及多端/浏览器差异时必填） | 服务 **QA**：回归影响评估 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐条对照 prd.md 页面/功能清单检查覆盖，无遗漏页面与关键流程 | prd.md | blocker |
| 逐条对照 rd/design.md 检查接口消费（方法/字段/错误处理）一致，无自造契约 | rd/design.md | blocker |
| 验证关键路径操作步骤可复现（点到哪、出现什么），QA 可据此设计交互用例 | prd.md（AC） | major |
| 检查加载/空/错误/超时等边界态有处理说明 | prd.md, rd/design.md | major |
| 核对前端自测用例在 qa/cases.md 中有对应口径，验收时可逐条核对 | qa/cases.md | minor |

---

### 2.4 case-design《用例/联调设计》

- **template_id**：case-design ｜ **name**：《用例/联调设计》 ｜ **owner_role**：qa
- **doc_path**：`docs/<版本号>/<需求名>/qa/cases.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 用例总览与覆盖矩阵 | ✅ | 服务 **用例评审（RD/FE/QA）**：评估覆盖度；服务 **QA 验收**：确定 test-result 执行范围 |
| 后端自测/联调用例（口径） | ✅ | 服务 **RD**：评审后端口径；服务 **RD 开发**：rd/dev.md 自测依据 |
| 前端自测/联调用例（口径） | ✅ | 服务 **FE**：评审前端口径；服务 **FE 开发**：fe/dev.md 自测依据 |
| 联调场景与数据准备 | ✅ | 服务 **RD/FE**：评审联调口径；服务 **QA 验收**：数据准备 |
| 回归范围 | ✅ | 服务 **RD/FE**：评审回归影响；服务 **QA 验收**：回归执行范围 |
| 验收用例（映射 AC） | ✅ | 服务 **QA 验收**：test-result 逐条执行的上游；与 prd 的 AC 一一映射 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐条核对用例覆盖 prd.md 全部 AC，每条 AC 至少映射一条用例 | prd.md | blocker |
| 对照 rd/design.md 检查后端用例的接口/数据结构口径一致 | rd/design.md | blocker |
| 对照 fe/design.md 检查前端用例覆盖关键路径（含异常分支） | fe/design.md | major |
| 验证联调场景含前置条件/数据准备/预期结果，可直接执行 | rd/design.md, fe/design.md | major |
| 检查明确 RD/FE 各自自测范围与回归范围，无交叉空白 | prd.md（范围） | major |
| 抽查每条用例预期结果可判定通过/不通过 | —（自含） | minor |

---

### 2.5 dev-notes《开发说明》（含实现-设计对照表）

- **template_id**：dev-notes ｜ **name**：《开发说明》 ｜ **owner_role**：rd \| fe（按分支实例化）
- **doc_path**：`docs/<版本号>/<需求名>/rd/dev.md`（RD）／`docs/<版本号>/<需求名>/fe/dev.md`（FE）

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 开发范围与变更清单 | ✅ | 服务 **CR**：快速定位变更面；服务 **QA**：验收对照 |
| 实现-设计对照表 | ✅ | 服务 **CR**：设计落地核对核心（m7 格式：`\| 设计要点 \| 来源文档 \| 实现位置 \| 一致/偏差 \| 说明 \|`）；服务 **QA**：验收对照 |
| 关键实现说明（含边界处理） | ✅ | 服务 **CR**：代码质量与边界核对 |
| 自测记录 | ✅ | 服务 **QA**：验收前置核对（自测是否覆盖 cases 口径） |
| 偏差与风险说明 | ✅ | 服务 **CR**：回归风险；服务 **QA**：回归影响 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐项核对对照表覆盖 rd/design.md（或 fe/design.md）全部设计要点，无遗漏行 | rd/design.md, fe/design.md | blocker |
| 验证每项偏差有说明；行为/接口变更或范围遗漏已**停止并报告 Leader**，未自行改设计 | prd.md, rd/design.md, fe/design.md | blocker |
| 抽查接口实现/消费与 rd/design.md 契约一致（方法/字段/错误码，抽样 ≥3 条） | rd/design.md | blocker |
| 核对自测记录与 qa/cases.md 自测口径一致，自测通过项有依据 | qa/cases.md | major |
| 检查边界/异常处理在实现中逐项有对应说明 | prd.md | minor |

---

### 2.6 research《调研文档》

- **template_id**：research ｜ **name**：《调研文档》 ｜ **owner_role**：inv
- **doc_path**：`docs/<版本号>/research.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 调研目标与范围 | ✅ | 服务 **Architect**：评审覆盖度（范围是否与立项对齐）；界定候选集合 |
| 候选方案对比 | ✅ | 服务 **Architect**：评审覆盖度（候选是否完整、对比维度是否合理） |
| 评估维度与结论 | ✅ | 服务 **Architect**：评审依据充分性（每项结论附数据/链接/实测） |
| 建议（引入/重写/自研） | ✅ | 服务 **Architect**：评审建议合理性；服务 **架构规划**：architecture 的能力规划输入 |
| 风险与后续动作 | ✅ | 服务 **Architect**：评审建议可行性；服务 **架构规划**：风险承接 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 核对候选覆盖 align.md 约定的调研范围，逐项无遗漏候选 | align.md | blocker |
| 验证每项结论附依据（来源链接/实测数据/对比表），可回溯 | —（自含，外部来源） | blocker |
| 检查建议（引入/重写/自研）有明确理由与成本/风险对比，非拍脑袋 | align.md, roadmap.md（上版本路线） | major |
| 核对建议考虑既有架构约束，可被后续 architecture 规划承接 | 历史版本 arch_design.md（上版本架构，如存在） | major |
| 检查风险与后续动作可执行（有负责人/时点） | —（自含） | minor |

---

### 2.7 defect《缺陷文档》

- **template_id**：defect ｜ **name**：《缺陷文档》 ｜ **owner_role**：qa
- **doc_path**：`docs/<版本号>/<需求名>/qa/defect.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 缺陷清单总览 | ✅ | 服务 **QA 验收**：闭环总览；服务 **Leader**：全局视图；程序路由的入口行 |
| 缺陷详情（复现/预期/实际） | ✅ | 服务 **RD/FE 返工**：按 assignee 可直接执行（复现步骤完整） |
| 归属判定（domain=rd\|fe 与理由） | ✅ | 服务 **程序路由**：按 domain 自动指派 assignee/to；是 QA L3 跨端根因判定输出 |
| 返工与复验记录 | ✅ | 服务 **QA 验收**：复验闭环（resolve 依据） |
| 遗留/挂起缺陷 | ⭕（存在遗留时必填） | 服务 **User**：验收风险告知 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐条检查每条缺陷 domain∈{rd,fe} 且判定理由明确，程序可据此自动路由 assignee | qa/test.md（执行结果） | blocker |
| 验证每条缺陷含复现步骤与预期/实际，RD/FE 可直接返工 | qa/cases.md（用例口径） | blocker |
| 核对缺陷关联失败用例（qa/cases.md 或 AC 编号），可追溯 | qa/test.md, qa/cases.md | major |
| 检查返工后复验记录完整（复验人/结果/时间） | —（自含） | major |
| 检查严重度分级有依据（阻塞/严重/一般/轻微） | —（自含） | minor |

---

### 2.8 test-result《测试用例与结果》

- **template_id**：test-result ｜ **name**：《测试用例与结果》 ｜ **owner_role**：qa
- **doc_path**：`docs/<版本号>/<需求名>/qa/test.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 测试范围与执行环境 | ✅ | 服务 **QA 验收**：口径自述；服务 **User**：可读性 |
| AC 逐条核对结果 | ✅ | 服务 **QA 验收**：核心节——prd AC → 用例 → 结果 三列逐条核对 |
| 用例执行明细 | ✅ | 服务 **QA 验收**：证据（用例/结果/截图） |
| 测试数据构造与测试代码（测试资产） | ✅ | 服务 **QA 验收**：复测低成本（数据一键重建、脚本一键复跑）；服务 **User**：可复现证据；服务 **下轮回归**：直接复用，不重复编写测试数据与脚本 |
| 联调结果 | ✅ | 服务 **QA 验收**：联调通过性结论 |
| 结论与遗留 | ✅ | 服务 **User**：验收依据；未闭环项引用 defect.md |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 逐条核对 prd.md 全部 AC，每项给出「通过/不通过/未覆盖」三态之一，无缺项 | prd.md | blocker |
| 验证每条结论可追溯到用例编号（qa/cases.md）或直接验证证据 | qa/cases.md | blocker |
| 核对未通过项已在 defect.md 建缺陷或说明原因，无悬空失败 | qa/defect.md | major |
| 检查执行覆盖 qa/cases.md 全部验收用例，跳过项有说明 | qa/cases.md | major |
| 检查联调结论与 rd/dev.md、fe/dev.md 自测记录一致 | rd/dev.md, fe/dev.md | minor |
| **验证「测试资产」节含测试数据构造脚本/种子文件与测试代码（含运行命令、依赖、输出），可在验收环境一键复跑，后续回归不重复编写** | qa/cases.md（数据准备口径）、workbench-sql/（重建范式参考） | **major** |

---

### 2.9 acceptance《验收记录》（历史章节，已并入 test-result）

> 说明：v0.28 起 acceptance 并入 test-result（《测试用例与验收记录》，qa/test.md），模板文件为 `qa-acceptance-template.md`（含 §3 测试资产必填节）；本节保留为历史参考，用户验收角度（结论/目标达成/遗留/交付物/意见）已并入 test-result 结论与遗留节。

- **template_id**：acceptance ｜ **name**：《验收记录》 ｜ **owner_role**：qa
- **doc_path**：`docs/<版本号>/<需求名>/qa/accept.md`（已废弃，README §7.5）

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 验收结论摘要 | ✅ | 服务 **User**：拍板依据（通过/不通过） |
| 目标达成核对（对照 prd 目标与 AC） | ✅ | 服务 **User**：确认需求目标达成 |
| 缺陷与遗留清单 | ✅ | 服务 **User**：风险知晓（未闭环项逐条说明） |
| 交付物清单 | ✅ | 服务 **User**：确认可交付 |
| 用户意见记录 | ✅ | 服务 **User**：签收/意见存档 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 核对验收结论与 test-result 结论一致，无未闭环缺陷未说明 | qa/test.md, qa/defect.md | blocker |
| 逐项对照 prd.md 目标/AC 检查达成说明，无跳过 | prd.md | blocker |
| 检查遗留缺陷有风险说明与处理计划（负责人/时点） | qa/defect.md | major |
| 核对交付物清单完整（文档已 wb_add_artifact 登记、代码分支已合并） | artifacts（wb_add_artifact 登记） | major |
| 检查用户验收意见已记录 | —（自含） | minor |

---

### 2.10 plan《方案文档》（任务场景）

- **template_id**：plan ｜ **name**：《方案文档》 ｜ **owner_role**：executor（执行者，User/Leader 指定）
- **doc_path**：`docs/<版本号>/<需求名>/plan.md`

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 问题定位与根因 | ✅ | 服务 **User**：判断方案必要性；服务 **QA**：明确测试对象 |
| 方案设计（怎么修/怎么做） | ✅ | 服务 **User**：拍板方案可接受；服务 **开发**：落地依据 |
| 影响面与回归范围 | ✅ | 服务 **QA**：评审回归影响 |
| 验证方案（如何测试） | ✅ | 服务 **QA**：评审可测性；服务 **QA 验收**：验收依据 |
| 风险与回滚 | ✅ | 服务 **QA/User**：风险知晓 |
| 分支约定 | ✅ | 服务 **开发/CR/QA**：后续同分支 `fix/<需求id>-<主题>` |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 检查方案步骤可落地（实施路径明确到可编码），执行者可据此开发 | 任务说明（缺陷报告/需求描述） | blocker |
| 验证「验证方案」可执行（怎么测、怎么判过），QA 可据此设计验收用例 | qa/cases.md（如已存在），否则 prd.md（AC） | blocker |
| 核对影响面与回归范围明确（涉及模块/文档/数据，逐项列出） | rd/design.md, fe/design.md（如涉及） | major |
| 检查分支名符合 `fix/<需求id>-<主题>` 约定 | —（约定） | major |
| 检查风险与回滚有说明（触发条件 + 动作） | —（自含） | minor |

---

### 2.11 architecture《架构规划》

- **template_id**：architecture ｜ **name**：《架构规划》 ｜ **owner_role**：arch
- **doc_path**：`docs/<版本号>/arch_design.md`（版本级，全版本需求共享）

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 版本目标与范围（短板、本期解决什么） | ✅ | 服务 **User**：确认目标；服务 **Leader**：评审范围 |
| 能力引入/实现规划 | ✅ | 服务 **Leader**：评审资源投入；承接 research 建议（引入/重写决策） |
| 需求拆解清单（归属版本） | ✅ | 服务 **Leader/PM**：范围与排期；服务 **创建需求·启动版本**：拆解输入 |
| 资源与依赖评估 | ✅ | 服务 **Leader**：评审资源；服务 **PM**：评审依赖 |
| 排期建议 | ✅ | 服务 **PM**：评审排期 |
| 架构约束与风险 | ✅ | 服务 **版本内 RD/FE**：rd-design/fe-design 技术路线一致性的对照上游；服务 **Leader**：风险 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 核对版本目标/能力规划与 research.md 建议一致（引入/重写决策有依据，未擅自偏离） | research.md | blocker |
| 逐条核对需求拆解覆盖版本目标与 align.md 范围，无遗漏 | align.md, roadmap.md（上版本路线） | blocker |
| 验证资源/排期评估有依据（人力/依赖/先后顺序），PM 可排期 | 需求池（wb_list_requirements） | major |
| 检查架构约束明确（技术选型/边界/共享设计），可供后续 rd-design/fe-design 遵循 | —（自含，下游消费） | major |
| 检查风险与回退路径有说明 | research.md | minor |

---

### 2.12 alignment《需求对齐文档》

- **template_id**：alignment ｜ **name**：《需求对齐文档》 ｜ **owner_role**：arch
- **doc_path**：`docs/<版本号>/align.md`（版本规划①立项产出；业务/技术/任务流程的需求对齐节点无产物，不套用此模板）

**章节骨架 structure**

| section | required | purpose（服务谁、为什么需要） |
|---|---|---|
| 版本背景与方向 | ✅ | 服务 **User**：确认版本方向 |
| 需求池梳理与价值评估 | ✅ | 服务 **PM**：评审价值与优先级 |
| 范围界定（本期/后续） | ✅ | 服务 **PM**：评审范围；服务 **User**：拍板 |
| 对齐结论与分歧记录 | ✅ | 服务 **立项评审**：结论存档（分歧点可追溯） |
| 后续动作（调研重点/待确认项） | ✅ | 服务 **调研/架构规划**：research、architecture 的输入 |

**质量门禁 quality_gates**

| check（检查动作） | ref（依赖文档） | severity |
|---|---|---|
| 核对价值/优先级评估逐条基于需求池事实（wb_list_requirements），可溯源 | 需求池 | blocker |
| 检查版本方向与范围表述明确，可指导后续调研与规划 | roadmap.md（上版本路线） | blocker |
| 核对分歧点有记录与结论（讨论 thread 或决策说明），无悬而未决 | 评论 thread（wb_comment） | major |
| 检查范围界定考虑资源与上版本路线约束 | roadmap.md | major |
| 检查后续动作明确（调研重点/待确认项/负责人） | research.md（下游） | minor |

---

## 3. 模板间依赖关系小结

### 3.1 设计/内容链（谁的哪一节是谁的上游）

| 上游模板节 | 下游消费模板节 | 依赖性质 |
|---|---|---|
| prd.验收标准（AC） | case-design.验收用例（映射 AC）→ test-result.AC 逐条核对 → acceptance.目标达成核对 | AC 逐级传递：设计 → 用例 → 执行 → 验收，QA 验收逐条闭环 |
| prd.目标/范围/页面清单 | rd-design.方案概述 / fe-design.页面清单 / case-design.覆盖矩阵 | 范围约束 + 覆盖度基准 |
| rd-design.接口设计/数据设计 | fe-design.接口对接（消费 rd 契约）｜ case-design.后端用例口径 | 契约上游：FE 不得自造契约，QA 按 rd 契约写后端口径 |
| rd-design.自测方案 | case-design.后端自测口径 → rd/dev.md.自测记录 | 后端口径传递 |
| fe-design.关键路径/交互方案 | case-design.前端用例口径 → fe/dev.md.自测记录 | 前端口径传递 |
| case-design.验收用例/自测口径 | dev-notes.自测记录 ｜ test-result.用例执行明细 | 开发自测与 QA 执行共用同一口径 |
| case-design.联调场景与数据准备 | test-result.测试数据构造与测试代码（测试资产）→ 后续回归/复测直接复用 | 数据准备口径沉淀为可复跑资产：数据一键重建 + 脚本一键复跑，回归不重复编写 |
| dev-notes.实现-设计对照表 | test-result（联调一致性抽查）｜ CR 评审 | 设计落地证据，供 CR 核对 + QA 对照 |
| test-result.结论 + defect.缺陷详情 | acceptance.结论/遗留清单 | 验收结论链：QA 自审闭环后交 User |
| defect.归属判定（domain） | 程序路由 → RD/FE 返工 → defect.复验记录 | 跨端根因判定驱动自动指派（L3） |
| alignment.范围/后续动作 | research.调研范围 → architecture.能力规划 | 版本规划链：立项 → 调研 → 架构 |
| research.建议（引入/重写/自研） | architecture.能力引入/实现规划 | 架构规划的依据来源 |
| architecture.需求拆解 + 架构约束 | 版本内各需求 prd.范围 ｜ rd-design/fe-design.技术路线一致 | 版本级约束向下传导 |
| plan.验证方案/回归范围 | dev-notes（任务场景自测）→ test-result | 任务场景独立小链，与设计链在开发/验收处汇合 |

### 3.2 评审依据链（模板 = 哪个门禁的依据，谁按它审）

| 评审门禁 | 依据模板 | 评审者（各审各的） |
|---|---|---|
| 需求评审（②） | prd | PM（价值/优先级/范围）、Architect（可行性）、User（拍板） |
| RD 技术评审（④） | rd-design | Architect（路线/一致）、RD（完整性/自测）、FE（接口契约）、QA（可测性）、PM（工期）、User（方案确认） |
| FE 设计评审（⑥） | fe-design | QA（交互可测性/关键路径） |
| 用例评审（⑧） | case-design | RD（后端口径）、FE（前端口径） |
| 开发评审（⑨） | dev-notes（对照 rd/design.md、fe/design.md） | Code Reviewer（质量/落地/回归风险） |
| QA 验收（⑩） | test-result + defect + acceptance（依据 prd AC 与 cases） | QA |
| 用户验收（⑪） | acceptance（引用 test-result/defect） | User |
| 方案评审（任务②） | plan | QA（可测性/回归）、User（拍板） |
| 调研评审（版本②） | research | Architect（覆盖度/依据/建议） |
| 架构规划评审（版本③） | architecture | Leader（范围资源）、PM（排期依赖）、User（目标） |
| 立项评审（版本①） | alignment | PM（价值）、User（方向） |

---

## 4. 质量门禁严重度定义

| severity | 含义 | 处理规则 |
|---|---|---|
| **blocker** | **阻塞门禁**：不满足则该产出不可被评审通过 | 产出者自检必须全部通过才可提评；评审者发现任一 blocker 不满足 → 评论 issue（type=issue）+ `wb_review` block，`wb_decide` 不得 pass，返工修复并 `wb_resolve` 后重新评审。典型场景：接口契约不一致、AC 未覆盖、domain 归属不清导致程序无法路由、验收结论与执行结果矛盾 |
| **major** | **评审必查**：评审者必须逐条核对，发现不满足应在评审中提出 issue 要求修订 | 评审者按 ref 对照逐条核对并表态；不满足项进入评论闭环（roundCount 机制），需在评审内闭环，但不必然阻塞 User 拍板 |
| **minor** | **建议**：评审者可提建议，产出者尽力满足 | 不阻塞评审；发现时以 discussion/notification 提出即可，产出者择机改进 |

> 执行口径：所有 check 的 `ref` 文档必须在产出前可读（上游已产出）或在验收时存在（下游核对）；`—（自含）` 项只依据本文件。模板升级（改 structure/gates）走 `document_templates.version` 递增，旧版本存 git 留痕，不影响已产出文档。

---

## 5. 附录：document_templates 落库对应

- 每节「章节骨架」表格 → `structure` JSON：`[{section, required, purpose}]`（required 用 true/false）。
- 每节「质量门禁」表格 → `quality_gates` JSON：`[{check, ref, severity}]`（ref 为逗号分隔的 m7 短路径；`—（自含）` 落 `""`）。
- 模板行：`{template_id, name, owner_role, doc_path, structure, quality_gates, version=1}`，共 **12 行**，按 §1.3 速查表落库。
- 入库后 `wb_my_task.outputs[].template_id` 引用即生效；评审端按 §4 严重度执行。
