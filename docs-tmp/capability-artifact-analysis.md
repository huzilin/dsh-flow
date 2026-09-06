# 能力 × 产物 逐能力分析（临时稿）

> **状态**：临时分析文档（未并入架构总集）。确认后合并至 `docs/v1.0/arch_design.md`（建议作为 §8.7「能力 × 产物 · 逐能力分析」或独立附录）。
> **结论先行**：产物注册当前处于「元数据结构化、正文半结构化」状态——`artifacts`/`issues` 的字段是结构化的，但产物内容自身（文档正文/代码/对话）未结构化；本分析给出「内容侧结构化」的最小增强方案（不改表结构，只增约定）。
> **引用规范**：本分析与后续产物一律引用模板**章节名**（以附录「章节骨架标注」表为准）而非章节编号，避免模板调整引起连锁改动。
> **重要发现**：qa 系列模板存在「正文 `##` 标题 vs 章节骨架标注」不同步（见 §6-1）——引用时以**骨架标注**为准。

---

## 1. 当前产物注册机制（实况）

产物注册统一入口为 `wb_add_artifact`（文档/链接）与 `wb_comment`（对话），落库 `artifacts` / `issues` 两表。

### 1.1 文档类产物（`wb_add_artifact kind=document`）

```jsonc
// artifacts 行（实例：req-019 实况数据）
{
  "id": "art-1786787966530",
  "nodeId": "n-09",              // 挂载到产出节点 ← 结构化
  "label": "《RD 开发说明》",     // 显示名 ← 结构化
  "href": "docs/v1.1/req-019-join-rollback/rd/dev.md",  // 相对项目路径 ← 结构化（真实路径 = doc_path 模板渲染）
  "kind": "document",            // document|link ← 结构化
  "note": "RD 自测完成待 CR",    // 自由备注 ← 半结构化
  "addedBy": "RD",               // 登记人 ← 结构化
  "product_type": "dev_notes",   // ★ 标准产物类型（ProductType 枚举，跨流程稳定）← 结构化
  "addedAt": "2026-08-15T09:59:26.529Z"
}
```

- **结构化**：`nodeId`（产物归属节点）、`kind`（doc/link）、`addedBy`（责任角色）——这三字段是可见性/路由的核心依据；**`product_type`（标准产物类型）**是能力跨流程复用的稳定索引（rework §2.6，能力 inputs/action 按类型取产物）。
- **半结构化**：`href` 指向的**文档正文**是自由 Markdown（按模板章节骨架产出，但骨架不落库）；`note` 自由文本。
- **路径模板化（J7 定稿）**：`href` 由**路径模板**（`docs/{{req-version}}/{{req-name}}/rd/dev.md`，双花括号 JS/Python 通用）按当前需求 ctx 渲染；how/ref 中产物引用为「《标准名》(模板路径)」——代码加载后直接替换 `{{var}}` 即得真实路径（host 端 `renderTpl`，见 capability-review-v2 §0.1）。

### 1.2 节点内对话（`wb_comment`，type=issue|discussion|notification）

```jsonc
// issues 行（实例：req-019 实况数据）
{
  "id": "cm-1786787966531",
  "requirementId": "req-019",
  "nodeId": "n-09",                       // 对话挂载节点 ← 结构化
  "anchor": { "kind": "document", "artifactId": "art-1786787966530" },  // 锚点 ← 结构化（可空=document 级）
  "author": "RD", "role": "rd",           // 发言者/角色 ← 结构化
  "reviewerRole": "Code Reviewer",        // 评审归属 ← 结构化
  "type": "issue",                        // issue|discussion|notification ← 结构化
  "assignee": "",                         // 解决责任人（L1/L2/L3）← 结构化
  "text": "RD 开发完成：…请评审。",        // 内容 ← 文本
  "state": "resolved", "thread": [],      // 状态/子对话 ← 结构化
  "resolvedBy": "Coordinator", "resolvedAt": "…"
}
```

- **结构化**：nodeId / anchor / author/role / reviewerRole / type / assignee / state / thread 状态。
- **对话正文体**：`text` + `thread[]` 为自由文本（不可编辑，审计留痕），语义靠人/agent 理解。

### 1.3 代码类产物（`kind=link` 或文档内描述）

- **现状**：`wb_add_artifact kind=link` 登记 **MR 外链**（`href` 为 GitHub MR URL），`label` 如「代码 MR（RD）」；代码正文不在 `.workbench/` 内（在 git 仓库/远端）。
- **对话中的代码引用**：评审意见用 `anchor.kind=block` 锚到开发说明文档的代码块，或 text 里写 `path:line`。
- **判断**：代码产物当前**未结构化**——只有 MR 链接，无「实现了哪些设计要点、哪些降级」的结构化清单。这正是开发说明书 `dev.md` 中「实现-设计对照表」要承载的（见 §4.4）。

### 1.4 结构化程度判定

| 产物类型 | 元数据（artifacts/issues 字段） | 内容侧（正文/代码/对话） | 判定 |
|---|---|---|---|
| 文档 | ✅ 结构化（nodeId/kind/href/addedBy） | ⚠️ 模板章节骨架，但结构不落库 | 半结构化 |
| 对话 | ✅ 结构化（nodeId/anchor/type/state/thread） | ⚠️ text 自由文本 | 半结构化 |
| 代码 | ✅ 结构化（kind=link/MR href） | ❌ 只有链接，无实现对照清单 | 非结构化 |

### 1.5 结构化增强方案（不改表结构，只加约定）

**A. 文档类——正文结构落库（可选，低成本）**
- 产物登记时在 `artifacts.note` 或新建 `metadata` JSON 字段记录「已按模板章节骨架产出」：`{"template_id":"dev-notes","sections":["文档信息","实现-设计对照表",…]}`——使可见性/门禁可程序化校验「章节齐全度」。
- **最小方案（推荐）：不落库，登记时程序预检**。产出者 `wb_add_artifact` 登记（或提评 `wb_comment`）时，Host 读产物正文 → 解析标题 → 与模板 `structure` 必填节比对「章节齐全度」→ 一次性返回 `{ok, missingSections[], gatesUnmet[]}` 作为即时报错/警告，**把「章节齐全」作为 blocker 门槛**（缺必填节即拦截提评）。*注：预检依赖正文标题与模板骨架章节名一致——这也要求 §6-1 的模板正文/骨架标注先同步。*

**B. 代码类——实现-设计对照表结构化（重点）**
- `dev.md` 的「实现-设计对照表」**已是结构化的表格**（设计要点/来源文档/实现位置/一致与偏差/说明），只是以 Markdown 表格存在而非 JSON。
- 增强：`wb_add_artifact` 对 `kind=link`（MR）支持额外字段 `devNotesRef` 指向对应 `dev.md`，使 CR 能从 MR 直接跳到对照表。
- **自审降级声明**：dev.md 中「偏差与风险说明」节须显式声明「实现降级」项（未实现的 AC、简化实现、遗留 TODO）——这是 CR/QA 判断「是否完整实现」的结构化入口。

**C. 对话类——保持文本为主，补结构化标签**
- `text` 保持自由文本（人机可读）；如需程序化判断可在 issues 加 `aspects[]`（如 `["接口契约","边界处理"]`）标注评论主题，供 wb_my_task 的 sessionsByNode 按主题聚合（可选，v1.1）。

---

## 2. 节点上下文与可见性（推演视角，非独立能力）

> **可见性不是一个能力，而是推演的结果**：我们扮演对应角色，按照该角色能力注册在流程的哪个节点、处于哪个流程类型（business/tech/task/plan）与阶段，推断「此刻能看到什么」；再基于可见内容推演「审批（评审）时参照什么、需要什么专业判断」。

### 2.1 推演框架（两步）

```
Step 1 · 可见性推导（我能看到什么）
  输入：流程定义表（flow_template_nodes → business/tech/task/plan）
       + 该角色能力注册的节点（gate 节点 reviewers 含我的 capability_id）
       + scope（如 business + full）
  推导：本节点之前已登记/应存在的产物（前置文档/代码）→ 即我可见的内容

Step 2 · 评审依据推演（我看到后，用什么来判定）
  输入：Step 1 的可见产物 + 我的评审能力 how（评审维度）
       + 对应模板 quality_gates + 我的角色专业职责
  推导：我判「过/不过」需要核对什么（一致性/可测性/规范符合性等）
```

### 2.2 推演示例：QA 评审 RD 技术设计（业务需求，full scope）

**Step 1 · 可见性**——查定义表：
- 流程：`business`（业务需求，full）；节点：`RD 技术评审`（gate=true）；
- `flow_template_nodes[RD 技术评审].reviewers` 含 `{role: qa, capability_id: qa-tech-review}` → **QA 的评审能力注册在此节点**；
- 该节点之前（order < N）的产物：`产品设计`（《product_doc》）+ `RD 技术设计`（《rd_design》）——**QA 可见：产品文档 + 技术文档**。

**Step 2 · 评审依据**——QA 以 `qa-tech-review` review 时需要判断：
- **一致性**：RD 设计是否覆盖产品文档要求的页面、能力（对照 prd「页面/功能清单」「功能需求」与 rd「总体方案设计」）；
- **可测性**：设计是否可测——接口/数据是否便于构造用例、有无联调口径（对照 rd「接口设计（契约）」「数据与存储设计」，为后续 case-design 提供输入）；
- **规范符合性**：接口是否符合技术规范——如幂等性是否被考虑、错误码/边界是否定义（对照 rd「接口设计（契约）」「风险与未决问题」）。

> 判断口径与模板挂钩：QA 评审 RD 设计时按 `backend-design-template` 的**章节骨架标注**核对（接口/数据/自测方案等必填节），按**质量门禁** severity 分级（blocker 不满足则不可过）。

### 2.3 可见性字段（wb_my_task 返回，作为推演的载体）

| 字段 | 含义 | 服务推演 |
|---|---|---|
| `node / nodeKey / mode / gate / state` | 当前节点，produce\|align，是否门禁 | 确定「我处于哪个阶段的哪个节点」 |
| `reviewers[]` | 本节点评审归属（各审各的） | 确定「我的评审能力注册在哪、谁同台」 |
| `artifactsToReview` | 前置节点产物（文档/代码） | = Step 1 可见性的结构化输出 |
| `dependencyDocs` | 前提文档（prd/rd/fe/qa；QA 验收额外含接口协议+代码） | = Step 1 可见性的文档维度 |
| `outputs[]` | 本节点应产出（doc_path/template_path） | 产出型角色「该产什么」 |
| `sessionsByNode` | 全需求每个 node 的全部会话（含锚定文档会话） | 补充「上下文」——讨论/评审历史 |

> `sessionsByNode` 全量返回是刻意为之——subagent 被派活后一次调用拿全部上下文，无需自行检索（m8 §11 AC1–AC5）。

### 2.4 可见性与「结构化产物」的衔接

- 第 1 节的结构化增强（章节齐全度、devMd 对照表、aspects 标签）**服务于推演的可靠性**：可程序校验 `artifactsToReview` 的章节覆盖率，把「缺章节」作为 blocked 依据（§1.5-A 登记预检）。
- 各角色在各类节点上的可见性与评审依据，见 §3 逐能力推演表。

---

## 2.5 how 推演方法论（新模型，节点内多动作）

> 能力 `how` 不再散成多条"读X/做Y"的无序句子，而是**一个节点内的动作序列**（如 RD 开发节点=开发+自测+登记+响应 CR，非拆成多个节点）。推演路径：**动作推演（依据 template_node_id 看哪些节点产物）→ 结合模板推算重点文档章节 + 文档挂载节点 → 落地为具体 how 动作序列**。

**推演三步**：

```
① 动作推演：该角色在本节点做什么
   - 依据 flow_nodes.template_node_id（实例节点关联定义节点）+ capabilities.inputs（依赖声明）
   - 确定要看哪些定义节点的产物（如 RD 开发看 node2 产品设计/node4 RD 技术设计/node8 用例设计/架构(版本级)）
② 结合模板推算重点文档章节 + 文档挂载节点
   - 每份产物挂载在相应 template_node_id 节点（product-design→node2、backend-design→node4…）
   - 用模板章节名定位要重点读的部分（如 rd「接口设计（契约）」）
③ 落地为 how 动作序列：节点内按序/按触发执行的动作（产出→登记→提评→响应反馈）
```

**完整范例：`backend-dev`（RD 开发节点，node_order=10）——节点内多动作**：

| 动作 | 参考产物（template_node_id 定位） | 重点章节（模板） | 实施 |
|---|---|---|---|
| A1 开发 | RD技术设计(node4)+架构(版本级)+产品设计(node2) | rd「接口设计(契约)」「数据与存储设计」；arch「整体架构」；prd「功能需求」 | 按契约实现、不造接口、对齐架构 |
| A2 自测 | RD技术设计(node4)+用例设计(node8) | rd「自测方案」；case「后端自测/联调用例(口径)」 | 自测覆盖 case 后端口径 |
| A3 登记提评 | 本节点产物《dev_notes》+ MR | dev-notes「实现-设计对照表」 | wb_add_artifact → wb_comment(to=cr) |
| A4 响应CR | 节点会话 open comment（CR 意见） | dev-notes「实现-设计对照表」 | 改代码/wb_reply 澄清→修订 resolve→复核 |

> A1–A4 都在 `RD 开发` **一个节点**内；CR 评论作为该节点 reviewerProducts/文档评论进入会话。frontend-dev 同构（参考《fe_design》+rd 契约+《product_doc》+《case_design》）。逐能力详细 inputs/判断依据见 `capability-review-v2.md`。

---

## 3. 逐能力 · 判断依据（扮演角色 → 用可见产物判断什么）

> 评审类能力（`-tech-review`/`pm-review`/`code-review`/`final-approve`/`decide-cap`）无产出，产出「评审意见」；判断依据 = 可见产物 + 模板 quality_gates。
> **逐能力的「inputs 声明 + 可见产物 + 判断依据」完整推演见 `capability-review-v2.md` §1**（本文 §3.1 为概览表）。

### 3.1 评审类能力（判断依据）

| capability_id | 角色 | 可见产物（artifactsToReview + dependencyDocs） | 判断依据（怎么用产物） |
|---|---|---|---|
| `pm-review` 价值/范围评审 | pm | 《product_doc》+《rd_design》 | 按《product_doc》「目标与成功指标」「范围与非范围」评价值/优先级/范围；对产品方案澄清质疑；**不评排期**（J23 批注已去）（各审各的） |
| `arch-tech-review` 技术方案评审（架构视角） | arch | 被评审设计文档（rd/fe/case design）+ `arch_design.md` | 对照 `arch_design.md`「整体架构」「架构设计原则与决策」评技术路线/架构一致/重构影响；research 场景评「结论与建议」依据充分性 |
| `rd-tech-review`（后端视角） | rd | rd-design + case-design + prd | 按 rd「总体方案设计」「接口设计（契约）」「数据与存储设计」评设计完整性/自测方案/后端口径 |
| `fe-tech-review`（前端视角） | fe | fe-design + rd-design | 按 fe「接口设计（前后端契约）」对照 rd「接口设计（契约）」评接口契约可对接；按「页面与交互设计」评前端口径 |
| `qa-tech-review`（测试视角） | qa | case-design + rd/fe design | 按 case「用例总览与覆盖矩阵」「联调场景与数据准备」评可测性/用例覆盖/验收口径（AC 链） |
| `code-review` 代码评审 | cr | MR link + `dev.md`（实现-设计对照表）+ 对应设计文档 | **关键**：MR 链接不可直接读正文 → 以 dev.md「实现-设计对照表」核对设计落地：逐条对照「设计要点/来源文档/实现位置/一致与偏差」；按「偏差与风险说明」评回归风险；真实代码经 fs/代码工具读取（若可用） |
| `final-approve` 最终拍板 | user | 《test_result》（AC 逐条核对+结论）+《defect》 | 按《test_result》「AC 逐条核对结果」「结论与遗留」评目标达成/可交付；按《defect》「遗留/挂起缺陷」评风险 |
| `decide-cap` 门禁决策 | leader | 评审表态汇总 + 未解决评论 | 全评论 resolved + 评审者已通过 → pass；否则 reject（并行组 targetBranch 部分回滚）；记录 last_decision |

> **各审各的铁律**：评审者只对职责内维度判断；职责外异议仅供参考，不构成 pass/block 依据（arch_design §8.5）。

### 3.2 产出型能力 · 评审环节的「被审者视角」（模拟）

以 **backend-design（RD 技术设计）** 为例，扮演 RD 收到评审意见后的判断闭环：
1. `wb_my_task` 拿 `sessionsByNode` → 看到各评审者（arch/fe/qa/pm）的 open 评论；
2. 对每条意见按模板「章节骨架」定位到对应章节（如 fe 意见锚到「接口设计（契约）」→ 修订该节）；
3. 与提评者分歧 → `wb_reply` 澄清（design 意图/权衡）→ 达成一致 resolve 或请 Leader 协调；
4. 全部 resolve → 各评审者 `wb_review pass` → Leader `wb_decide` 推进。

（同模式适用于 fe-design / case-design / dev-notes / acceptance-test——评审者 5 步 + 被评审者 5 步统一协议，arch_design §8.6。）

---

## 4. 逐能力 · 产出（模板 / 代码要求 / 自审降级）

> 产出型 11 项。模板引用一律用**章节名**。代码类（backend-dev/frontend-dev）单独列「代码要求 + 自审」如下。
> **节点的「how 动作序列 + inputs + 产出」完整推演见 `capability-review-v2.md` §2**（本文 §4 为模板/自审要点，backend-dev 完整范例见 §2.5）。

### 4.1 产出型能力总表

| capability_id | 角色 | 产出 doc（doc_path 模板） | 模板文件 | 模板章节名（产出骨架） |
|---|---|---|---|---|
| `product-design` | pm | `docs/{{req-version}}/{{req-name}}/prd/prd.md` | prd-template | 问题陈述 → 目标与成功指标 → 范围与非范围 → 用户与场景 → 功能需求 → 非功能需求 → UI/UX 与交互说明 → 数据埋点与口径 → 依赖、风险与决策 → 上线与运营 · 附：PRD 评审清单 |
| `backend-design` | rd | `docs/{{req-version}}/{{req-name}}/rd/design.md` | backend-design-doc-template | 背景与动机 → 目标与非目标 → 总体方案设计 → 备选方案与取舍 → 非功能设计 → 风险与未决问题 → 测试与验收 → 排期与任务拆分 |
| `frontend-design` | fe | `docs/{{req-version}}/{{req-name}}/fe/design.md` | frontend-design-doc-template | 背景与目标 → 页面与交互设计 → 组件设计 → 数据与状态管理 → 接口设计（前后端契约） → 技术选型与工程化 → 性能与兼容性 → 埋点与监控 → 任务拆分与排期 → 风险与验收 |
| `case-design-cap` | qa | `docs/{{req-version}}/{{req-name}}/qa/cases.md` | qa-testcase-review-template | 骨架标注章节：用例总览与覆盖矩阵 → 后端自测/联调用例（口径） → 前端自测/联调用例（口径） → 联调场景与数据准备 → 回归范围 → 验收用例（映射 AC） ⚠️正文标题仍是「评审流程」结构，待同步 |
| `acceptance-test` | qa | `docs/{{req-version}}/{{req-name}}/qa/test.md` + `docs/{{req-version}}/{{req-name}}/qa/defect.md` | qa-acceptance-template + qa-defect-management-template | test 骨架：测试范围与执行环境 → AC 逐条核对结果 → 用例执行明细 → 测试资产（数据构造+测试代码，必填） → 联调结果 → 结论与遗留；defect 骨架：缺陷清单总览 → 缺陷详情（复现/预期/实际） → 归属判定（domain） → 返工与复验记录 → 遗留/挂起缺陷 ⚠️正文标题（如「测试概述与结论」）与骨架标注存在差异，待同步 |
| `research-cap` | inv | `docs/{{req-version}}/research.md` | research-open-source-tech-template | 调研背景与目标 → 调研方法与信息源 → 候选方案盘点 → 开源项目评估 → 候选方案对比 → POC 原型验证 → 结论与建议 → 风险与合规清单 |
| `arch-cap` | arch | `docs/{{req-version}}/arch_design.md` | arch-version-design-template | 执行摘要 → 版本背景与范围 → 架构设计原则与决策 → 整体架构 → 组件与模块设计 → 需求拆解 → 数据架构 → 关键技术方案 → 非功能架构 → 架构演进路线 → 风险与开放问题 |
| `alignment-cap` | arch | `docs/{{req-version}}/align.md` | arch-version-scope-align-template | 版本主题与目标 → 需求对齐清单 → 范围边界 → 工作量与人力评估 → 版本排期 → 风险与待决问题 → 变更管理 |
| `backend-dev` / `frontend-dev` | rd/fe | `docs/{{req-version}}/{{req-name}}/{{side}}/dev.md` | dev-notes-template | 文档信息 → 开发范围与变更清单 → 实现-设计对照表 → 关键实现说明 → 自测记录 → 偏差与风险说明 → 质量门禁 |
| `plan-cap` | executor | `docs/{{req-version}}/{{req-name}}/plan.md` | plan-template | 文档信息 → 问题定位与根因 → 方案设计 → 影响面与回归范围 → 验证方案 → 风险与回滚 → 分支约定 |

### 4.2 代码类产出要求与自审（backend-dev / frontend-dev）

**代码应符合的前置文档要求**（依赖链）：
- 设计契约：《rd_design》（接口设计（契约）/数据与存储设计）——代码**不得自造契约**；
- 前端对齐：《fe_design》（接口设计（前后端契约））——接口消费与 rd 契约一致；
- 自测口径：《case_design》（各端自测/联调用例）——自测记录须覆盖 cases 口径；
- 大方向：《architecture》（架构设计原则与决策/关键技术方案）——技术路线一致。

**自审（实现完整性 & 降级识别，dev.md「质量门禁」节 + 「实现-设计对照表」）**：

| 自审项 | 判定 | severity |
|---|---|---|
| 覆盖：对照表覆盖《rd_design》/《fe_design》**全部设计要点**，无遗漏行 | 逐条打勾，缺一行即降级声明 | blocker |
| 一致：接口实现/消费与 rd「接口设计（契约）」一致（方法/字段/错误码，抽样 ≥3） | 抽样比对 + 一致性自评 | blocker |
| 偏差：每项偏差有说明；**行为/接口变更或范围遗漏 → 停止并报告 Leader**，不自行改设计 | 偏差列逐条说明 | blocker |
| 边界：边界/异常处理（加载/空/错误/超时/无权限）逐项有对应实现说明 | 对照 prd 异常场景 | major |
| 测序：自测记录与《case_design》自测口径一致，通过项有依据 | 场景→依据→结果→证据四列 | major |

**降级识别（显式声明，CR/QA 判断入口）**：
- 「偏差与风险说明」节必须显式列出**实现降级项**：未实现的 AC、简化实现（如 mock 替代）、遗留 TODO、性能未达标项；
- 降级分三档：**合理简化**（标注说明，评审接受）/ **行为/接口变更**（停止报告 Leader，不自行改）/ **范围遗漏**（停止并报告 Leader）；
- 降级不隐藏：CR 按「实现-设计对照表」偏差列逐条核对，QA 按 dev「自测记录」复跑验证。

### 4.3 产出后登记闭环（所有产出型能力通用）

```
产出文档/代码 → 按模板章节骨架撰写 + quality_gates 自检 → 写盘（doc_path）
→ wb_add_artifact 登记（kind=document|link，nodeId=产出节点）→ 提评 wb_comment(to=评审归属)
```

---

## 5. 与架构总集的衔接（确认后合并）

- 本文 §1（注册机制 + 结构化增强）→ 并入 `arch_design.md` §7（数据层）下，作为「7.5.x 产物注册与结构化」；
- 本文 §2（可见性）→ 并入 §9.2（wb_my_task）扩展「可见性契约」；
- 本文 §3/§4（逐能力判断依据 + 产出要求）→ 并入 §8.4（能力模型）扩展，或作为 §8.7「能力 × 产物 · 逐能力分析」独立小节；
- **模板章节名引用规范** → 并入 §10.3（产物模板机制）：「引用模板一律用章节名，不用章节编号」。

## 6. 本次分析发现（待办清单）

> 以下为分析过程中暴露的问题，供确认后转入实现/文档修正。

| # | 发现 | 影响 | 建议处置 |
|---|---|---|---|
| 1 | **qa 系列模板正文与骨架标注不一致**：`qa-testcase-review-template` 正文 `##` 标题是「评审流程」结构（基本信息/评审目标与范围/评审参与人/用例评审检查清单/评审记录/评审结论），但**章节骨架标注**是「用例文档」结构（用例总览与覆盖矩阵/…）；`qa-acceptance-template` 正文「测试概述与结论」vs 骨架「测试范围与执行环境」，同型问题 | 产物作者按正文写会缺骨架要求的核心节；评审者按骨架核对会对不上 | ⚠️ **以「章节骨架标注」为准**（它是设计定稿意图）；修正模板正文 `##` 标题与骨架标注对齐（改 `docs/templates/*` 文件即可，全局生效） |
| 2 | **backend-dev/frontend-dev/plan-cap 的 demo 缺失**：模板已存在（dev-notes-template / plan-template），但 `dev-notes-demo.md` / `plan-demo.md` 未建，seed.json 中 demo_path=null | 角色产出无示例参照，首次产出质量不稳 | 补建 2 个 demo 文件（按 template 章节骨架填示例） |
| 3 | **代码产物仅 kind=link（MR），无结构化实现清单**：CR 只能靠 dev.md 对照表间接核对 | 代码评审依赖 dev.md 质量，无程序化校验入口 | 保持 dev.md「实现-设计对照表」为代码侧唯一结构化入口；可选增强：`wb_add_artifact(kind=link)` 支持 `devNotesRef` 字段把 MR 与 dev.md 绑定 |
| 4 | **产物正文「章节齐全度」无程序化校验**：`wb_add_artifact` 只登记元数据，不校验正文是否按模板骨架产出 | 缺章节的产物也能登记、提到门禁 | 可选增强：登记时校验正文标题是否覆盖模板 `structure` 必填节（参照 §1.5-A） |