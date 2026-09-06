# 能力 · 逐能力 Review（新产物可见性模型 v2）

> **状态**：临时评审稿。基于 `artifact-visibility-rework.md`（产物可见性 v2）= 统一产物模型（document 含评论 / link / 节点级 comment）+ 可见性由 `capabilities.inputs` 声明驱动。
> **目标**：为 19 个能力逐个确定——①inputs 依赖声明（看哪些定义节点的什么产物）；②可见产物（经 inputs 推导）；③判断依据（评审/产出如何用）；④产出型补产出模板与自审。
> **关键方法**：**能力 = 节点内动作集合**——对能力做**动作拆分**（可独立调度的 action 单元，§0.2），调度时据 `wb_my_task` 精确定位到能力×动作。
> **引用规范**：模板一律用**章节名**，不用章节编号。

> **一致性校验（与 seed.json/arch_design）**：seed.json 的 `capabilities` 现行结构为 `how`（字符串数组），**尚无结构化 `inputs`/`actions` 字段**；`flow_template_nodes` 无顶层 `id`（可用 `(flow_templates_id, node_order)` 作定义节点标识）。本文定义 `inputs`/`actions` 为待回填 seed 的目标形态（rework §5 待办：`capabilities.inputs` 声明 + `flow_nodes.template_node_id`）。

---

## 0. 评审/产出能力总览（19）

| 类型 | 能力 | 角色 |
|---|---|---|
| 评审型（8） | `pm-review` `arch-tech-review` `rd-tech-review` `fe-tech-review` `qa-tech-review` `code-review` `decide-cap` `final-approve` | pm/arch/rd/fe/qa/cr/leader/user |
| 产出型（11） | `product-design` `backend-design` `frontend-design` `case-design-cap` `acceptance-test` `research-cap` `arch-cap` `alignment-cap` `backend-dev` `frontend-dev` `plan-cap` | arch/rd/fe/qa/inv/executor |

> inputs 声明统一格式（**用标准产物类型 ProductType，不写 node_key/order，保证能力跨流程复用**，见 rework §2.6）：
```
inputs: [
  { productType: "product_doc", use_threads: true },   // 找当前流程上下文里的《产品设计文档》产物
  { productType: "rd_design",   use_threads: true }
]
```

### 0.1 标准产物定义（ProductType 权威表，全局唯一引用标识）

> **为什么需要**：能力跨流程复用，inputs/action 引用产物必须用**稳定的标准类型**，不能写流程内节点位置或中文 label（跨流程会变，见 rework §2.6）。下表为**唯一权威定义**——所有 inputs/action 的 `productType` 取值、文档登记 `wb_add_artifact`, 以及新增流程产出节点绑定，都以此为准。

| productType | 标准产物名 | 产出能力(capability_id) | 路径模板 `{{var}}` 渲染后 | 模板文件 | 层级 |
|---|---|---|---|---|---|
| `product_doc` | 《产品设计文档》 | product-design | `docs/{{req-version}}/{{req-name}}/prd/prd.md` | prd-template.md | 需求级 |
| `rd_design` | 《RD 技术设计》 | backend-design | `docs/{{req-version}}/{{req-name}}/rd/design.md` | backend-design-doc-template.md | 需求级 |
| `fe_design` | 《FE 技术设计》 | frontend-design | `docs/{{req-version}}/{{req-name}}/fe/design.md` | frontend-design-doc-template.md | 需求级 |
| `case_design` | 《用例/联调设计》（tech 流程作《用例设计》） | case-design-cap | `docs/{{req-version}}/{{req-name}}/qa/cases.md` | qa-testcase-review-template.md | 需求级 |
| `test_result` | 《测试用例与验收记录》 | acceptance-test | `docs/{{req-version}}/{{req-name}}/qa/test.md` | qa-acceptance-template.md | 需求级 |
| `defect` | 《缺陷文档》 | acceptance-test | `docs/{{req-version}}/{{req-name}}/qa/defect.md` | qa-defect-management-template.md | 需求级 |
| `dev_notes` | 《开发说明》（business 带 RD/FE 前缀） | backend-dev / frontend-dev | `docs/{{req-version}}/{{req-name}}/{{side}}/dev.md` | dev-notes-template.md | 需求级 |
| `plan` | 《方案文档》 | plan-cap | `docs/{{req-version}}/{{req-name}}/plan.md` | plan-template.md | 需求级 |
| `research` | 《调研文档》 | research-cap | `docs/{{req-version}}/research.md` | research-open-source-tech-template.md | 版本级 |
| `architecture` | 《架构规划》 | arch-cap | `docs/{{req-version}}/arch_design.md` | arch-version-design-template.md | 版本级 |
| `alignment` | 《需求对齐文档》 | alignment-cap | `docs/{{req-version}}/align.md` | arch-version-scope-align-template.md | 版本级 |
| `roadmap` | 《版本规划》 | （版本规划④） | `docs/{{req-version}}/roadmap.md` | — | 版本级 |
| `code_mr` | 代码 MR（link 产物） | backend-dev / frontend-dev | MR 外链 URL | — | 需求级 |
| `feature_branch` | 特性分支（tech 设计阶段敲定，与 code_mr 关联） | — | flow_nodes.feature_branch | — | 需求级 |

> **命名受控**：新增流程/节点时，产物只能绑定上表已有 `productType`，**禁止自造**（如「产品文档」≠ `product_doc`）；`wb_add_artifact` 登记携带 `product_type`，与标准表校验。这样能力跨流程复用（inputs/action 按类型取产物）不会因流程/命名差异失效。

> **doc_path 是模板（J7 how 模板化定稿 · 技术可实施）**：表中 `doc_path` 是**路径模板**（`docs/{{req-version}}/{{req-name}}/…`，双花括号 `{{var}}`，JS/Python 通用），不是真实路径。how/ref 中每个产物直接写成「标准名 + 模板路径」：
> ```
> 《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md)
> ```
> 宿主端加载 how 字符串后，用渲染函数替换 `{{var}}` 即得真实相对路径（零依赖实现）：

> ```ts
> // host 侧（JS/TS 插件）—— 一次定义，全 how/ref 复用
> // 变量名可含连字符（req-version 等），故用 [^{}]+ 匹配
> function renderTpl(s: string, ctx: Record<string, string>): string {
>   return s.replace(/\{\{([^{}]+)\}\}/g, (_m, k) => {
>     const key = k.trim();
>     if (!(key in ctx)) throw new Error(`how 模板变量未提供: {{{key}}}`);
>     return ctx[key];
>   });
> }
> const ctx = { 'req-version': 'v1.1', 'req-name': 'req-019-join-rollback', 'side': 'rd' };
> renderTpl('docs/{{req-version}}/{{req-name}}/rd/design.md', ctx)
> // -> 'docs/v1.1/req-019-join-rollback/rd/design.md'
> ```

> **模板变量元数据（ctx 来源）**：
> | 变量 | 含义 | 来源 |
> |---|---|---|
> | `{{req-version}}` | 需求所属版本号（v1.1…） | `requirements.version` |
> | `{{req-name}}` | 需求 slug（req-019-join-rollback…） | `requirements.slug` |
> | `{{side}}` | 开发侧（rd\|fe，仅 dev_notes 路径用） | 能力 `role_id` |
>
> **渲染时机**：定义层（`capabilities.actions[].how` / `inputs.reference`）存模板；`wb_my_task` 返回时按当前需求 ctx 调 `renderTpl` 渲染后再发给 subagent——subagent 拿到即真实路径，无需自行拼接。代码类（code_mr/feature_branch）无文件路径，保留逻辑引用 `「code_mr」`。

> **input 引用示例**（替代 node_key）：`inputs: [{ productType: "rd_design", use_threads: true }]`、`action.reference: [{ "productType": "product_doc" }]`。

### 0.1.1 对话类产物（评论/节点会话）的定位：不标准化，用属性查询

**为什么不在 ProductType 里定义「节点对话/评论」**：对话是**动态、流程相关**的——内容、归属者、锚点、所处节点都随流程/角色变化（与复用性问题同源：节点名跨流程不同，评论更无法归一）。因此**对话不放进静态 productType 枚举**。

**对话类产物的定位方式**：靠**结构化属性查询**（评论产物 Issue 自带可过滤字段），而非固定类型名。mytask 返回的评论产物（`document.comments[]`/`reviewerProducts[]`）每条带：
- `reviewerRole`（谁提的，如 Code Reviewer）
- `assignee`（指派给谁，如 rd）
- `state`（open/pending-confirm/resolved）
- `anchor`（锚在哪：文档行/块/文档级；或节点级）
- `thread`（往返对话）、`roundCount`、`text`

**how/action 用属性查询描述「要看哪些对话」**，例如 respond_to_cr：

```
reference: { query: { reviewerRole: "Code Reviewer", assignee: "rd", state: "open" } }
```

→ RD 从 mytask 评论产物里精确取到属于自己的开放 CR 意见（text + anchor + thread），据此改进代码或澄清。

> **引用规约**：文档/代码类产物 → `productType`（标准枚举）；对话/评论类产物 → `query`（属性过滤）。两者都**不依赖流程内节点位置/名称**，跨流程成立。这是能力复用性问题的完整解法（文档靠类型、对话靠属性）。

### 0.2 能力 → 动作拆分（动作内嵌于 capabilities，不建独立表）

**动机**：能力是「某角色在某节点的一组职责」，但调度时需**精确定位到动作**。例：CR 反馈代码问题 → Leader 调度 RD → RD 经 `wb_my_task` 确认自己在 `backend-dev` 能力下要用**哪个动作**响应（`respond_to_cr`），而非笼统一句"开发"。

**存储方式（用户决策）**：**动作不单独建表**——因为动作不从属于能力复用、也不会被单独取出（总是随能力/节点上下文一起访问），故用 **JSON 数组存在 `capabilities.actions` 字段**内；**每个动作内嵌自己的 `how`**（实施方式），并带 `trigger`/`when_to_use` 场景说明，让 subagent 判断**什么场景用哪个动作**。

**能力 = 节点内动作集合**，结构化存储于 `capabilities.actions`：

```jsonc
// capabilities 行新增 actions（JSON 数组，替代原 how 字符串数组）
{
  "capability_id": "backend-dev",
  "role_id": "rd",
  "actions": [
    {
      "action_id": "backend-dev.dev",
      "name": "开发实现",
      "trigger": "node_enter",                       // ★ 何时触发
      "when_to_use": "节点进入时，或设计/反馈要求重做代码时",  // ★ subagent 判断场景
      "how": "读《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md) + 《architecture》(docs/{{req-version}}/arch_design.md) + 《product_doc》(docs/{{req-version}}/{{req-name}}/prd/prd.md) → 按设计契约实现（不造接口），对齐架构；产出代码（待自测）",
      "reference": [{ "productType": "rd_design" },
                    { "productType": "product_doc" }],
      "focus": ["接口设计（契约）", "数据与存储设计"],
      "output": "代码（MR 前）", "tools": ["wb_my_task", "fs", "git"]
    },
    {
      "action_id": "backend-dev.self_test",
      "name": "开发自测",
      "trigger": "dev_complete",
      "when_to_use": "dev 动作产出代码后，提交前",
      "how": "按《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md) 自测方案 + 《case_design》(docs/{{req-version}}/{{req-name}}/qa/cases.md) 后端口径自测 → 记实现-设计对照表",
      "reference": [{ "productType": "rd_design" },
                    { "productType": "case_design" }],
      "focus": ["自测方案", "后端自测/联调用例（口径）"],
      "output": "自测记录", "tools": ["wb_my_task"]
    },
    {
      "action_id": "backend-dev.submit",
      "name": "登记提评",
      "trigger": "dev_complete",
      "when_to_use": "自测通过后，产生产物并请 CR",
      "how": "wb_add_artifact(《dev_notes》(docs/{{req-version}}/{{req-name}}/{{side}}/dev.md) + MR link, nodeId=当前节点) → wb_comment(to=cr) 请评审",
      "reference": [], "focus": [],
      "output": "dev.md + MR link", "tools": ["wb_add_artifact", "wb_comment"]
    },
    {
      "action_id": "backend-dev.respond_to_cr",
      "name": "响应CR",
      "trigger": "reviewer_feedback(open)",
      "when_to_use": "当前节点（RD 开发）的评论产物中，存在 reviewerRole=Code Reviewer 且 assignee=rd（我）且 state=open 的评论时",
      "how": "① 从 mytask 返回的评论产物里，过滤出 assignee=rd 且 state=open 的评论（多半在 document.comments[] 或 reviewerProducts[]，挂在《dev_notes》(docs/{{req-version}}/{{req-name}}/{{side}}/dev.md)/code 上）；
            ② 逐条读：text（CR 要改什么）+ anchor（锚在代码/文档哪一行、哪一块）→ 精确定位改动点；
            ③ 读 thread（往返澄清），理解上下文 → 改代码，或 wb_reply 澄清（若分歧）；
            ④ 修订后 wb_resolve(note=修复说明) → CR 复核 confirm/reopen",
      "reference": [{ "query": { "reviewerRole": "Code Reviewer", "assignee": "rd", "state": "open" } }],
      "focus": ["dev_notes.实现-设计对照表", "code_mr.对应行"],
      "output": "修订代码 / resolve", "tools": ["wb_my_task", "wb_reply", "wb_resolve"]
    }
  ]
}
```

> **调度定位**：Leader 对 RD「用 backend-dev.respond_to_cr 响应 CR」→ RD 调 `wb_my_task` → 返回该 capability 的 `actions`（含目标 action_id 的 when_to_use/how/reference/focus/do）→ RD 据此精确执行。动作**不单独取**（始终随能力+节点上下文返回），故内嵌 JSON 即可，无需独立表。
> **subagent 理解场景**：每个动作带 `trigger`（程序可判定的触发时机）+ `when_to_use`（自然语言场景说明），subagent 据此从 `actions[]` 中选对当前场景的动作执行。

**how 落位**：原 `capabilities.how`（字符串数组）演进为 `capabilities.actions[]` 中每个动作的 `how` 字段（动作内嵌实施说明）；宏指令级「评审者 5 步 / 被评审者 5 步」仍是动作间的通用协议（评审型见 §1）。

### 0.3 与新模型（execution-loop）的衔接复盘（2026-08-24）

> 结合 `execution-loop.md`（trigger 契约 / how 获取链路 / 决策树 / 能力缺口）复盘，能力模型 4 处对齐：

**① trigger 为全局统一程序契约（禁用自造）**
```
统一 trigger 枚举（excel 链路表实际使用的 7 个；execution-loop §3/§11）：
  node_enter / dev_complete / reviewer_feedback(open) / node_review_pending
  / issue_replied / reject_rolled_back / decide
动作只声明「由哪个 trigger 激活」；不允许各能力自造（如 self_test_passed → 并入 dev_complete）。
（task_stale 为兜底扫描事件，不属于动作级 trigger。）
```

**② 评审型能力也须有 actions[]（subagent 取 how 的统一机制）**
```
execution-loop §9：subagent 经 wb_my_task 返回的 actions[] 按 action_id 取 how。
→ 评审型 6 个（pm/arch/rd/fe/qa-tech-review + code-review）已有 actions[]：
   review（读可见产物→按 focus 核对）+ review_response（处理澄清/修复复核），
   focus 各异。
（excel 链路表已展开为每评审位 review/review_response 两行。）
```

**③ decide-cap / final-approve 与决策树接轨**
```
- decide-cap：trigger=decide（全评审通过）；inputs 用 query（评审表态汇总 + 未解决评论），
  非站位符 productType；对应 decision tree ⑥分支。
- final-approve：trigger=node_review_pending（用户验收 gate）；inputs = test_result + defect + product_doc。
```

**④ 动作补 `outputProductTypes`（与 ProductType 闭环）**
```
动作增加 outputProductTypes: ["dev_notes", "code_mr"]：
  → wb_add_artifact 登记校验 product_type 与动作声明一致（execution-loop §5）。
```

---

## 1. 评审型能力（8）

> 评审型能力也做**动作拆分**，但同一评审协议（各审各的 4 步）——差异只在各动作的 reference/focus（维度）：
> - `review`（评审）：读可见产物 → 按 focus 核对；
> - `<role>.comment`（提意见）：wb_comment 锚到对应产物；
> - `<role>.approve|block`（表态）：wb_review pass/block；
> - `<role>.confirm|reopen`（复核）：作者修订后确认/reopen。
> 各评审能力的具体 focus 见下各节。

### 1.1 `pm-review` 价值/范围评审（pm）

**作用节点**：产品设计评审（business order3）、RD 技术评审（business order5，reviewer 位）、立项评审（plan order2）、架构规划评审（plan order6）

**inputs（依赖声明）**：
```
inputs: [
  { productType: "product_doc", use_threads: true },
  { productType: "rd_design", use_threads: true }
]
```

**可见产物**（经 inputs 推导，出现在 currentNode/relatedNodes）：
- `currentNode`（如产品设计评审节点）：owner=PM 的《产品设计文档》+ 其上评论；relatedNodes 含前置节点产物；
- 具体：产品设计文档（《product_doc》）+ RD 技术设计（《rd_design》，RD 技术评审场景）。

**判断依据**（Step 2）：
- 按 prd「目标与成功指标」「范围与非范围」评**需求价值 / 优先级 / 范围**（值不值得做、做到哪里停）；
- 对产品方案提出**澄清质疑**（向下游追问业务理解，与 User/产出者沟通）；**不评排期**（agent 无排期，J23 批注已去）；
- **各审各的**：PM 只评价值/优先级/范围，不评技术路线。

### 1.2 `arch-tech-review` 技术方案评审（架构视角）（arch）

**作用节点**：产品设计评审（business order3 reviewer）、RD 技术评审（business order5 reviewer，scope=be）、调研评审（plan order4，arch 审 research）、技术评审（task order3，arch 审 plan）

**inputs**：
```
inputs: [
  { productType: "rd_design", use_threads: true },
  { productType: "fe_design", use_threads: true },
  { productType: "research", use_threads: true }   // 版本规划②场景
]
```

**可见产物**：`rd_design`、`fe_design`（评审场景）；`research`（版本规划）。

**判断依据**：
- 对照 `arch_design.md`「整体架构」「架构设计原则与决策」评**技术路线 / 架构一致性 / 重构影响**；
- research 场景评「结论与建议」依据充分性；
- 各审各的：arch 评技术方向，不评实现细节/工期。

### 1.3 `rd-tech-review` 技术方案评审（后端视角）（rd）

**作用节点**：产品设计评审（business order3 reviewer，scope=be）、RD 技术评审（business order5，ownerRole=rd 自评 + reviewers 位）、用例设计评审（business order9 reviewer，scope=be）

**inputs**：
```
inputs: [
  { productType: "product_doc", use_threads: true },
  { productType: "rd_design", use_threads: true },
  { productType: "case_design", use_threads: true }
]
```

**可见产物**：`product_doc` + `rd_design` + `case_design`。

**判断依据**：
- 按 rd「总体方案设计」「接口设计（契约）」「数据与存储设计」评**设计完整性 / 接口数据 / 自测方案**；
- 按 case「后端自测/联调用例（口径）」评**后端口径**；
- 对照 prd 确认设计是否覆盖功能点。

### 1.4 `fe-tech-review` 技术方案评审（前端视角）（fe）

**作用节点**：产品设计评审（business order3 reviewer，scope=fe）、RD 技术评审（business order5 reviewer，scope=fe）、FE 设计评审（business order7 reviewer）、用例设计评审（business order9 reviewer，scope=fe）

**inputs**：
```
inputs: [
  { productType: "fe_design", use_threads: true },
  { productType: "rd_design", use_threads: true },   // 接口契约来源
  { productType: "case_design", use_threads: true }
]
```

**可见产物**：`fe_design` + `rd_design`（接口契约）+ `case_design`。

**判断依据**：
- 按 fe「接口设计（前后端契约）」对照 rd「接口设计（契约）」评**接口契约可对接**（FE 不得自造契约）；
- 按「页面与交互设计」评前端口径；
- 按 case「前端自测/联调用例（口径）」评前端自测口径。

### 1.5 `qa-tech-review` 技术方案评审（测试视角）（qa）

**作用节点**：RD 技术评审（business order5 reviewer）、FE 设计评审（business order7 reviewer）、用例设计评审（business order9 ownerRole=QA 自评）

**inputs**：
```
inputs: [
  { productType: "product_doc", use_threads: true },
  { productType: "rd_design", use_threads: true },
  { productType: "fe_design", use_threads: true },
  { productType: "case_design", use_threads: true }
]
```

**可见产物**：`product_doc` + `rd_design`/`fe_design` + `case_design`（覆盖 AC 链）。

**判断依据**：
- 按 prd「目标与成功指标」+「功能需求」评**可测性 / AC 是否可验证**（一致性：RD 设计是否覆盖产品页面/能力）；
- 按 rd「接口设计（契约）」评**接口可测性、规范符合性**（幂等/错误码/边界）；
- 按 case「用例总览与覆盖矩阵」「联调场景与数据准备」评**用例覆盖 / 联调口径 / 验收口径（AC 链）**。

### 1.6 `code-review` 代码评审（cr）

**作用节点**：RD 开发（business order10 reviewer）、FE 开发（business order11 reviewer）、并行开发（tech order8/9）、开发（task order4）

**inputs**：
```
inputs: [
  { productType: "code_mr", use_threads: true },
  { productType: "code_mr", use_threads: true },
  { productType: "rd_design", use_threads: false },
  { productType: "fe_design", use_threads: false }
]
```

**可见产物**：
- 代码产物 `kind=link`（MR）+ `readMode: worktree` + `featureBranch`（worktree 检出读真实代码）；
- 设计文档（rd/fe design）用于对照设计落地。

**判断依据**：
- worktree 检出 feature 分支读**真实代码**；
- 以 `dev.md`「实现-设计对照表」逐条核对**设计落地**（设计要点/来源文档/实现位置/一致与偏差）；
- 按「偏差与风险说明」评**回归风险**；实测接口/边界处理。

### 1.7 `final-approve` 最终拍板（user）

**作用节点**：用户验收（business order13）、各流程收尾（tech order11 / task order6；用户验收节点为 User 交互占位行，无 capability action）

**inputs**：
```
inputs: [
  { productType: "test_result", use_threads: true },
  { productType: "product_doc", use_threads: false }
]
```

**可见产物**：`test_result` + `defect` + `product_doc`（对照目标）。

**判断依据**：
- 按《test_result》「AC 逐条核对结果」「结论与遗留」评**目标达成 / 可交付**；
- 按《defect》「遗留/挂起缺陷」评**风险**；
- 对照 prd「目标与成功指标」最终拍板通过/打回。

### 1.8 `decide-cap` 门禁决策（leader）

**作用节点**：创建需求·启动版本（plan order7，leader 拆解需求产物）

**动作（excel 实际展开）**：
- `create_reqs`（创建需求·启动版本）：架构规划通过后，按架构拆解创建需求（查重：相似则修改原需求）→ 归入版本 → 版本待启动→进行中 → 《roadmap》产物登记。
- **decide（门禁通过→推进/回滚）由程序契约承担，不展开为 agent 动作行**（用户决策：推进=hook/定时扫描，无「发起评审/门禁决策」显式动作；参见 clarify 讨论同源）。

**inputs**：
```
inputs: [
  { productType: "<当前 gate 节点>", use_threads: true }
]
```

**可见产物**：评审成员表态汇总（reviewers[].state）+ 未解决评论。

**判断依据**：
- 全部评论 resolved 且评审者已通过 → **pass**；
- 否则 **reject**（reason 必填；并行组 targetBranch 部分回滚）；
- 记录 `last_decision`。

> 说明：`final-approve` 在 excel 中为 User 交互占位行（能力/动作列留空，参与表示拍板交互）；products 亦不登记动作。各流程末尾追加 `完成` 节点（DONE 标记，无能力/动作，标志流程终点，由 Leader 确认收尾）。

---

## 2. 产出型能力（11）

> 产出型能力在**一个节点内包含多个动作**（非拆成多个节点）——how 是"节点内动作序列"。产出文档按模板；代码按自审对照。
> how 推演路径：① **动作推演**（节点内该角色做什么，依据 template_node_id 看哪些节点产物）→ ② **结合模板推算重点文档章节 + 文档挂载节点** → ③ 落地为具体动作序列（how）。

### 2.1 完整范例：`backend-dev`（RD 开发）（rd）

**作用节点**：RD 开发（business node_order=10，gate=true，owner=RD，parallel dev-1 branch=rd）

**节点内多动作（统一在 RD 开发节点，非三个节点）→ 结构化为 `capabilities.actions[]` JSON（§0.2）**：

```
capabilities.actions（4 个动作内嵌 how + trigger + when_to_use，见表 §0.2）
  backend-dev.dev / backend-dev.self_test / backend-dev.submit / backend-dev.respond_to_cr
```

- 行动的完整 JSON（含 `when_to_use` 场景说明、内嵌 `how`）见 `§0.2` 示例，此处不重复；
- **inputs（配合 template_node_id 推导可见产物）**：
```
inputs: [
  { productType: "product_doc", use_threads: false },  // 功能/AC
  { productType: "rd_design", use_threads: true },  // 设计契约（含其上评审评论）
  { productType: "case_design", use_threads: false }, // 自测口径
  { productType: "architecture", use_threads: false }
]
```

> **关键**：三个子动作 A1–A4 都在 `RD 开发` 一个节点内完成，共用 `template_node_id` 定位的可见产物；CR 评论作为该节点的 reviewerProducts / 文档评论进入节点会话，RD 据此产出 A4。

### 2.2 其他产出型能力（简版 how 推演）

> 各能力均为"节点内多动作"；此处列核心动作与 inputs（模板/章节见 capability-artifact-analysis §4.1）。

| capability_id | 角色 | 节点内动作序列（how 骨架） | inputs（参考节点） |
|---|---|---|---|
| `product-design` | pm | 读产品需求+架构 → 产出《产品设计文档》→ 登记提评(to=PM) | product_doc；architecture |
| `backend-design` | rd | 读产品+架构 → 产出《RD 技术设计》→ 登记提评(to=评审归属) | product_doc；architecture |
| `frontend-design` | fe | 读产品+rd 设计 → 产出《FE 技术设计》→ 登记提评(to=QA) | product_doc；rd_design |
| `case-design-cap` | qa | 读 rd/fe 设计 → 产出《用例/联调设计》→ 登记提评(to=RD/FE) | rd_design；fe_design |
| `acceptance-test` | qa | 读 product+case+接口+代码 → 建测试资产+执行验收 → test/defect → 缺陷返工闭环 → 登记提评 | product_doc；case_design；rd_design/fe_design；code_mr(branch) |
| `research-cap` | inv | 读对齐+路线 → 调研竞品 → 产出《调研文档》→ 登记提评(to=Architect) | alignment；roadmap |
| `arch-cap` | arch | 读调研+对齐 → 产出《架构规划》→ 登记提评(to=Leader/PM) | research；alignment |
| `alignment-cap` | arch | 读版本需求池+路线 → 产出《需求对齐文档》→ 登记提评(to=PM) | roadmap |
| `frontend-dev` | fe | 开发（fe_design+rd接口+product_doc）→ 自测（fe_design+case_design）→ 登记(MR+dev_notes) → 响应CR | fe_design；rd_design；product_doc；case_design |
| `plan-cap` | executor | 读任务说明+架构 → 产出《方案文档》→ 登记提评(to=QA) | architecture |

> **how 模板化约定（J7 定稿，技术可实施）**：how/ref 里每个产物直接写成「《标准名》(模板路径)」——如 `《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md)`；host 端 `renderTpl(s, ctx)`（§0.1）替换 `{{var}}` 即得真实路径，**how 里禁止写死相对路径**（rd/design.md 之类不再出现），需求/版本变化时无需改 how。
> 动作统一携带 `outputProductTypes`（与 ProductType 闭环，wb_add_artifact 校验同源）。