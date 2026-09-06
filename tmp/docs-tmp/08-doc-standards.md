# 团队项目工作台 · 文档规范（目录结构 / 文档依赖 / 产物模板 / 质量门禁）

> **地位**：本文档是团队的文档规范（制度层）——规定 `docs/` 目录结构、角色间的文档依赖与「开发完成检查」、产物模板（章节骨架 + 质量门禁）与严重度执行口径。
> 取代旧稿 `m7-doc-structure.md` / `m8-doc-templates.md`，目录与依赖、模板清单、门禁口径以本文为准。
> 职责边界：**doc 目录约定、产物模板、质量门禁**。流程节点与各节点产物清单的唯一真相源在 `06-flows-and-roles.md`（节点表），本文档的模板与各节点产物按节点表对应，只引用不重写节点表。
> 模板正文（章节细则 / 门禁 check 明细）由 `docs/templates/*.md` 外置文件承载，本文档不复制模板正文，只描述章节骨架要点与速查关系。

---

## 1. 目录结构

**三层组织：版本 → 需求 → 角色**。文档按版本隔离，版本内按需求与角色落位；架构文档位于版本层，被该版本全部需求遵循。

```
docs/
├── design/                             # 制度层（不属版本，全局只读）：制度/流程/规范文档（m6~m8）
├── <版本号>/                           # 版本目录，如 v1.1 / v1.2
│   ├── arch_design.md                  # 架构规划（版本级，全版本需求大方向遵循）
│   ├── roadmap.md                      # 版本规划：目标/范围/需求清单/拆分
│   ├── align.md                        # 需求对齐文档（版本规划①产物）
│   ├── research.md                     # 调研文档（版本规划②产物）
│   └── <需求名>/                       # 需求目录，如 req-019-join-rollback
│       ├── prd.md                      # 产品设计文档（业务①/任务场景按需）
│       ├── rd/
│       │   ├── design.md               # RD 技术设计（业务③/技术②）
│       │   └── dev.md                  # RD 开发说明（业务⑨.1/技术⑥，含实现-设计对照表）
│       ├── fe/
│       │   ├── design.md               # FE 技术设计（业务⑤/技术④按需）
│       │   └── dev.md                  # FE 开发说明（业务⑨.2/技术⑥，含实现-设计对照表）
│       ├── qa/
│       │   ├── cases.md                # 用例/联调设计（业务⑦/技术⑤）
│       │   ├── test.md                 # 测试用例与验收记录（业务⑩/技术⑦，含 §测试资产）
│       │   ├── defect.md               # 缺陷文档（QA 验收发现 bug 时）
│       │   ├── accept.md               # 验收记录（已并入 test.md，废弃留痕，见 §7）
│       │   ├── data/                   # 测试资产·数据构造（种子/造数脚本，幂等可重建）
│       │   └── code/                   # 测试资产·测试代码（回归/校验脚本，含运行命令，一键复跑）
│       └── plan.md                     # 方案文档（category=任务，执行者产出）
└── prototype/                          # 历史 mock 产物（演练数据，迁移归并后删除）
```

**命名与登记规则**：

| 项 | 规则 |
|---|---|
| 需求目录 | `req-<id>-<短slug>`——id 保留工作台 `.workbench` 数据关联；slug 取需求主题（如 `req-019-join-rollback`） |
| 版本目录 | `v<主>.<次>`——与 `.workbench/versions.json` 的 id 对应（如 v-11 → `v1.1`） |
| 文件命名 | 固定名（`design.md` / `dev.md` / `cases.md` / `test.md` …），版本演进覆盖同文件，git 留痕 |
| 产物登记 | 产物写入后调用 `wb_add_artifact` 登记（href=文件路径，label=《文档名》，kind=document） |

## 2. 文档依赖矩阵与开发完成检查

### 2.1 文档依赖矩阵

| 角色产出 | 必须一致/参考 | 来源（优先级从高到低） |
|---|---|---|
| **RD 开发**（rd/dev.md） | 本需求 RD 技术设计、符合 PRD 设计、自测满足测例、大方向符合版本架构 | 本需求 `rd/design.md` → `prd.md` → `qa/cases.md`（自测口径）→ `arch_design.md`（版本架构） |
| **FE 开发**（fe/dev.md） | 本需求 FE 技术设计、接口参考 RD 契约、页面参考 PRD、自测参考测例 | 本需求 `fe/design.md` → `rd/design.md`（接口契约）→ `prd.md`（页面设计）→ `qa/cases.md`（自测口径） |
| **RD 技术设计**（rd/design.md） | 本需求 prd + 版本架构 | `prd.md` + `arch_design.md` |
| **FE 技术设计**（fe/design.md） | 本需求 prd + rd/design.md 接口 | `prd.md` + `rd/design.md` |
| **QA 用例**（qa/cases.md） | 本需求 prd + rd/fe 设计 | `prd.md` + `rd/design.md` + `fe/design.md` |
| **QA 验收**（qa/test.md + qa/defect.md） | 本需求 cases、dev 对照表、测试资产口径 | `qa/cases.md` + `rd/dev.md` + `fe/dev.md`；测试资产（`qa/data/` + `qa/code/`）按 `qa/cases.md` 数据准备口径沉淀，回归直接复用不重复编写 |
| **版本规划**（align/research/arch_design/roadmap） | 版本内需求与上版本路线 | 需求池（`wb_list_requirements`）+ 上版本规划（历史版本目录） |

> 所有 `ref`/来源均为 §1 目录约定的相对短名（`prd.md` / `rd/design.md` / `qa/cases.md` …），下游引用同一套路径约定，可程序校验。

### 2.2 开发完成检查（RD/FE 通用，各审各的）

1. **读**：按上表读取全部依据文档；
2. **核对**：逐项核对四类一致性——**覆盖**（无遗漏）/ **一致**（无偏离）/ **契约**（接口/数据结构）/ **边界**（状态处理与用例口径）；
3. **输出**：实现-设计对照表附于 dev.md，格式：

   | 设计要点 | 来源文档 | 实现位置 | 一致/偏差 | 说明 |
   |---|---|---|---|---|

4. **偏差处理**：合理简化标注说明；行为/接口变更或范围遗漏 → **停止并报告 Leader**，不自行改设计；
5. **完成判定**：对照表全一致（或已说明合理偏差）+ 自测通过 + MR 就绪。

## 3. 产物模板

### 3.1 机制：模板 = 章节骨架 + 质量门禁，外置文件承载

- 每个产物的模板 = **章节骨架（structure）+ 质量门禁（quality_gates）**，由 `docs/templates/*.md` 外置文件承载——`*-template.md`（章节骨架 + 质量门槛）与 `*-demo.md`（填充示例）**成对**存在。
- **capabilities.products 引用**：`capabilities.products[]` 的 `template_path / demo_path` 指向模板文件；`wb_my_task` 的 `outputs[]` 经 capability_id 展开出 doc_path/template_path，角色按章节骨架产出、按质量门禁自检，写盘后 `wb_add_artifact` 登记。
- **程序做路由、agent 做专业**：结构/门禁由文件承载（程序据此路由与校验），agent 只做内容与专业判断；**调整模板 = 改 docs/templates/ 对应文件，全局生效**（不碰 persona / 不碰代码）。
- 本规范只描述各模板章节骨架要点与速查关系；全文细则以模板文件为准。

### 3.2 模板速查表（产出型 11 能力 → 12 产物行）

| capability_id | template_id | 名称 | owner_role | doc_path | 已固化 |
|---|---|---|---|---|---|
| product-design | prd | 《产品设计文档》 | arch | prd.md | ✅ `prd-template.md` |
| backend-design | rd-design | 《RD 技术设计》 | rd | rd/design.md | ✅ `backend-design-doc-template.md` |
| frontend-design | fe-design | 《FE 技术设计》 | fe | fe/design.md | ✅ `frontend-design-doc-template.md` |
| case-design-cap | case-design | 《用例/联调设计》 | qa | qa/cases.md | ✅ `qa-testcase-review-template.md` |
| acceptance-test | test-result | 《测试用例与验收记录》（acceptance 已并入，含 §测试资产） | qa | qa/test.md | ✅ `qa-acceptance-template.md` |
| acceptance-test | defect | 《缺陷文档》（acceptance-test 第 2 产物） | qa | qa/defect.md | ✅ `qa-defect-management-template.md` |
| research-cap | research | 《调研文档》 | inv | research.md | ✅ `research-open-source-tech-template.md` |
| arch-cap | architecture | 《架构规划》（版本级） | arch | arch_design.md | ✅ `arch-version-design-template.md` |
| alignment-cap | alignment | 《需求对齐文档》（版本级） | arch | align.md | ✅ `arch-version-scope-align-template.md` |
| backend-dev | dev-notes | 《开发说明》（RD 分支） | rd | rd/dev.md | ⏳ 未固化 |
| frontend-dev | dev-notes | 《开发说明》（FE 分支） | fe | fe/dev.md | ⏳ 未固化 |
| plan-cap | plan | 《方案文档》 | executor | plan.md | ⏳ 未固化 |

已固化 **9 组**（template+demo 成对，即 `docs/templates/` 现有 18 个文件）；未固化 **3 项**（backend-dev / frontend-dev / plan-cap，模板文件待建，产出结构暂按 `<6 章节骨架` 约定执行）。

### 3.3 模板章节骨架要点（12 模板）

> 下列「必填小节」为骨架要点；每节的服务对象标注在下（“各审各的”职责见 06 节点表与模板文件内 quality_gates）。

**① prd《产品设计文档》**〔product-design · arch · `prd.md`〕
定位：需求的共同理解载体 + 交付边界约束 + 可验证结果的交付契约。必填小节：需求背景与问题定义（服务 PM/User 价值判断）、目标/范围/非目标（全流程下游的范围约束）、目标用户与核心场景（FE 页面输入）、产品方案（架构师可行性/FE 输入）、页面/功能清单与关键路径（FE 结构、QA 可测起点）、**验收标准 AC**（**QA 用例与验收的上游**）、优先级与版本归属建议（可选，PM 可覆盖）。

**② rd-design《RD 技术设计》**〔backend-design · rd · `rd/design.md`〕
定位：后端方案的评审载体与全链路契约来源。必填小节：方案概述与目标（Architect/User）、技术路线与架构一致（Architect）、**接口设计（契约）**（**fe-design 接口对接节的上游**；FE 可对接、QA 联调口径）、**数据设计**（FE 数据结构契约、QA 造数）、模块划分与实现要点（RD/CR）、自测方案（QA 后端口径、rd/dev.md 自测依据）、工期与依赖（PM 排期）、风险与回滚（CR 回归风险/QA 回归影响）。

**③ fe-design《FE 技术设计》**〔frontend-design · fe · `fe/design.md`〕
定位：前端方案的评审载体。必填小节：页面/组件清单与交互方案（QA 可测性、FE 开发落地）、状态管理与数据流（QA 可观察、CR 核对）、**接口对接（消费 rd 契约）**（上游 = rd-design 接口/数据节；QA 联调口径、CR 核对一致性）、关键路径与边界处理（QA 可验证、CR 回归风险）、前端自测方案（QA 口径、fe/dev.md 自测依据）、兼容性与风险（可选，涉及多端时必填）。

**④ case-design《用例/联调设计》**〔case-design-cap · qa · `qa/cases.md`〕
定位：前后端自测/联调/回归的统一口径与验收用例来源。必填小节：用例总览与覆盖矩阵（用例评审、QA 验收执行范围）、后端自测/联调用例（RD 口径）、前端自测/联调用例（FE 口径）、联调场景与数据准备（RD/FE 评审、QA 数据准备）、回归范围（RD/FE）、**验收用例（映射 prd 的 AC）**（**test-result 逐条执行的上游**）。

**⑤ dev-notes《开发说明》**〔backend-dev/frontend-dev · rd\|fe · `rd/dev.md` / `fe/dev.md`〕
定位：设计落地的证据与返工/验收对照依据（rd、fe 两分支同一结构）。必填小节：开发范围与变更清单（CR 定位、QA 对照）、**实现-设计对照表**（§2.2 格式，CR 设计落地核对核心、QA 验收对照）、关键实现说明（含边界处理，CR）、自测记录（QA 验收前置核对）、偏差与风险说明（CR 回归风险、QA 回归影响）。

**⑥ research《调研文档》**〔research-cap · inv · `research.md`（版本级）〕
定位：立项后的技术选型论证。必填小节：调研目标与范围（Architect 覆盖度）、候选方案对比（Architect）、评估维度与结论（每项结论附依据，Architect）、建议（引入/重写/自研）（Architect、**architecture 能力规划输入**）、风险与后续动作（Architect、架构规划风险承接）。

**⑦ defect《缺陷文档》**〔acceptance-test 第 2 产物 · qa · `qa/defect.md`〕
定位：验收缺陷的闭环载体与自动路由源。必填小节：缺陷清单总览（QA 验收闭环、Leader 全局视图、程序路由入口行）、缺陷详情（复现/预期/实际，RD/FE 返工可直接执行）、**归属判定（domain=rd\|fe 与理由）**（QA L3 跨端根因判定输出，程序按 domain 自动指派）、返工与复验记录（QA 复验闭环）、遗留/挂起缺陷（存在遗留时必填，User 风险告知）。

**⑧ test-result《测试用例与验收记录》**〔acceptance-test · qa · `qa/test.md`〕
定位：测试执行证据 + 验收结论 + **可复跑测试资产**的合一文档（acceptance 已并入）。必填小节：测试范围与执行环境、**AC 逐条核对结果**（prd AC → 用例 → 结果三列逐条核对）、用例执行明细（证据）、**§测试资产（数据构造 + 测试代码）**（数据一键重建、脚本一键复跑，回归直接复用）、联调结果、结论与遗留（User 验收依据，未闭环项引用 defect.md）。

**⑨ plan《方案文档》**〔plan-cap · executor · `plan.md`〕
定位：任务场景（修 bug/非需求改动）的方案拍板载体。必填小节：问题定位与根因（User 必要性、QA 测试对象）、方案设计（怎么修/怎么做，User 拍板、开发落地）、影响面与回归范围（QA 回归影响）、验证方案（怎么测、怎么判过，QA 验收依据）、风险与回滚、分支约定（`fix/<需求id>-<主题>`，开发/CR/QA 同分支）。

**⑩ architecture《架构规划》**〔arch-cap · arch · `arch_design.md`（版本级）〕
定位：版本级目标/能力/拆解与架构约束（全版本需求遵循）。必填小节：版本目标与范围（User 目标、Leader 范围）、**能力引入/实现规划**（承接 research 建议，Leader 资源）、**需求拆解清单**（Leader/PM 范围排期、创建需求·启动版本输入）、资源与依赖评估（Leader/PM）、排期建议（PM）、**架构约束与风险**（**版本内 rd-design/fe-design 技术路线一致性的对照上游**、Leader 风险）。

**⑪ alignment《需求对齐文档》**〔alignment-cap · arch · `align.md`（版本级）〕
定位：版本规划的立项产出，定义版本方向与范围。必填小节：版本背景与方向（User 确认）、需求池梳理与价值评估（PM）、范围界定（本期/后续，PM 评审、User 拍板）、对齐结论与分歧记录（立项评审结论存档）、后续动作（调研重点/待确认项，**research / architecture 的输入**）。

> 业务/技术/任务流程的「需求对齐」节点无文档产物（mode=align，结论落 issues），不套用 alignment 模板——alignment 模板仅用于版本规划①。

## 4. 质量门禁

**严重度定义**：

| severity | 含义 | 处理规则 |
|---|---|---|
| **blocker** | **阻塞门禁**：不满足则该产出不可被评审通过 | 产出者自检必须全部通过才可提评；评审者发现任一 blocker 不满足 → 评论 issue + 评审打回（block），门禁决策不得 pass；返工修复并 resolve 后重新评审。典型：接口契约不一致、AC 未覆盖、domain 归属不清致程序无法路由、验收结论与执行结果矛盾 |
| **major** | **评审必查**：评审者必须逐条核对 | 发现不满足须在评审中提出 issue 要求修订，进入评论闭环（roundCount 机制），需在评审内闭环，但不必然阻塞 User 拍板 |
| **minor** | **建议**：评审者可提建议 | 不阻塞评审；以 discussion 提出，产出者择机改进 |

**执行口径**：

- 每条 check 的 `ref` 文档必须在产出前可读（上游已产出）或在验收时存在（下游核对）；`—（自含）` 项只依据本文件内容。
- 模板升级（改章节/门禁）走 **git 版本化**：直接修改 `docs/templates/` 对应文件，经 git 留痕全局生效，不影响已产出文档。

## 5. 模板间依赖链（总览）

1. **AC 链**：prd.验收标准（AC）→ case-design.验收用例（映射 AC）→ test-result.AC 逐条核对 → 验收结论（QA 自审闭环后交 User 拍板）。
2. **契约链**：rd-design.接口设计/数据设计 → fe-design.接口对接（消费 rd 契约，FE 不得自造契约）→ qa 后端口径（case-design 后端用例/联调用例）。
3. **版本链**：alignment（align.md）→ research（research.md）→ architecture（arch_design.md）→ 需求拆解（roadmap.md / 创建需求·启动版本）→ 版本内各需求 prd / rd-design / fe-design 遵循版本架构约束。
4. **验收链**：test-result.结论 + defect.缺陷详情/归属判定 → 验收结论（defect 的 domain 驱动程序自动指派返工，复验后闭环）。

## 6. 产物形态与展示

- **doc / link 两类产物**：doc = 文档（章节正文），link = 链接（如代码 MR）。
- **doc 展示**：阅读器读正文，支持锚点评论（document / line / range / block 四级锚点，模型见 `05-domain-model.md`，交互见 `03-module-flow-tree.md`）。
- **link 展示**：新窗口跳转打开目标链接。
- 产物行以类型标识区分：📄 = doc，🔗 = link；节点无产物时显示「**该节点无产物**」（如仅评审意见的门禁节点）。

## 7. 演进注记

- acceptance 已并入 test-result（《测试用例与验收记录》，含 §测试资产），`qa/accept.md` 废弃留痕；模板内容自外置 `docs/templates/*.md` 文件承载起由文件管理（原内联子数据表方案废弃），本帖模板清单即当前固化的 9 组 + 待固化 3 项。