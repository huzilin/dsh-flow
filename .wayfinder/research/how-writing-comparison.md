# how 编写规范对照分析：writing-for-agents × 既有五段式

> status: research findings · 对应票 `.wayfinder/tickets/04-how-writing-comparison.md` · created 2026-08-27
> 用途：本文件是「how 编写规范 v2 逐条定夺」（票 05，HITL）的输入。每条对照结论给出「值得叠加进 v2」的建议与理由；正文用中文，术语保留英文。

## 一手源（本文件所有引用以此为准）

| 源 | 位置 |
|---|---|
| writing-for-agents 写作纪律 | `~/.dsh/skills/mp-writing-for-agents/SKILL.md`（全文 81 行；同目录无 `SKILL-MECHANICS.md`） |
| 既有 how 写法（能力推演方法论） | `docs-tmp/capability-review-v2.md`（19 能力，动作拆分 + 五段式 how） |
| 评审/自检/quality_gates 用法 | `docs-tmp/capability-artifact-analysis.md`（§3/§4：评审判断依据 = 可见产物 + 模板 quality_gates；产出链路 = 按模板章节骨架撰写 + quality_gates 自检 → 写盘） |
| `wb_my_task` 返回模型与 respond_to_cr 语义 | `docs-tmp/my-task-response-model.md` |
| 对话/评论类产物属性模型、不硬切分支 | `docs-tmp/artifact-visibility-rework.md`（§3，经 my-task-response-model.md L188 引用） |

writing-for-agents 重点纪律（SKILL.md 章节）：`Context pointers`（指针措辞决定触发）、`The two loads`（context load / cognitive load）、`Information hierarchy`（in-file step / in-file reference / disclosed reference + progressive disclosure + co-location + sprawl）、`Steps and completion criteria`（clarity / demand）、`When to split`（by sequence / by invocation）、`Leading words`、`Negation`、`Pruning`（单一真相源 / 环境即真相源 / relevance / no-op / sediment）。

既有五段式（任务定义为准，与能力推演方法论一致）：① 读哪些产物——`ProductType` 标准枚举（14 类，capability-review-v2.md §0.1），不依赖流程内节点位置/名称；② 按哪个模板——模板路径 + 章节名（不用章节编号）；③ 怎么自检——`quality_gates`；④ 登记/提评/响应——`wb_*` 工具；⑤ 输出什么——`productType`。补充规则：`trigger` 是程序契约枚举、`when_to_use` 是场景说明；动作完整字段 `action_id/name/trigger/when_to_use/how/reference/focus/outputProductTypes/tools`；对话/评论类产物不标准化，用 `query` 属性过滤（reviewerRole/assignee/state/anchor）；跨流程复用的能力 `reference` 用 productType/query。

---

## 对照样本（how 实例）

### 样本 A：`backend-dev` 四动作（产出型 · 完整 JSON，capability-review-v2.md §0.2）

```jsonc
{
  "capability_id": "backend-dev", "role_id": "rd",
  "actions": [
    { "action_id": "backend-dev.dev", "trigger": "node_enter",
      "when_to_use": "节点进入时，或设计/反馈要求重做代码时",
      "how": "读《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md) + 《architecture》(docs/{{req-version}}/arch_design.md) + 《product_doc》(docs/{{req-version}}/{{req-name}}/prd/prd.md) → 按设计契约实现（不造接口），对齐架构；产出代码（待自测）",
      "reference": [{ "productType": "rd_design" }, { "productType": "product_doc" }],
      "focus": ["接口设计（契约）", "数据与存储设计"], "output": "代码（MR 前）",
      "tools": ["wb_my_task", "fs", "git"] },
    { "action_id": "backend-dev.self_test", "trigger": "dev_complete",
      "when_to_use": "dev 动作产出代码后，提交前",
      "how": "按《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md) 自测方案 + 《case_design》(docs/{{req-version}}/{{req-name}}/qa/cases.md) 后端口径自测 → 记实现-设计对照表",
      "reference": [{ "productType": "rd_design" }, { "productType": "case_design" }],
      "focus": ["自测方案", "后端自测/联调用例（口径）"], "output": "自测记录", "tools": ["wb_my_task"] },
    { "action_id": "backend-dev.submit", "trigger": "dev_complete",
      "when_to_use": "自测通过后，产生产物并请 CR",
      "how": "wb_add_artifact(《dev_notes》(docs/{{req-version}}/{{req-name}}/{{side}}/dev.md) + MR link, nodeId=当前节点) → wb_comment(to=cr) 请评审",
      "reference": [], "focus": [], "output": "dev.md + MR link",
      "tools": ["wb_add_artifact", "wb_comment"] },
    { "action_id": "backend-dev.respond_to_cr", "trigger": "reviewer_feedback(open)",
      "when_to_use": "当前节点（RD 开发）的评论产物中，存在 reviewerRole=Code Reviewer 且 assignee=rd 且 state=open 的评论时",
      "how": "① 从 mytask 返回的评论产物里，过滤出 assignee=rd 且 state=open 的评论（多半在 document.comments[] 或 reviewerProducts[]，挂在《dev_notes》(...) 上）；
            ② 逐条读：text（CR 要改什么）+ anchor（锚在代码/文档哪一行、哪一块）→ 精确定位改动点；
            ③ 读 thread（往返澄清），理解上下文 → 改代码，或 wb_reply 澄清（若分歧）；
            ④ 修订后 wb_resolve(note=修复说明) → CR 复核 confirm/reopen",
      "reference": [{ "query": { "reviewerRole": "Code Reviewer", "assignee": "rd", "state": "open" } }],
      "focus": ["dev_notes.实现-设计对照表", "code_mr.对应行"],
      "output": "修订代码 / resolve", "tools": ["wb_my_task", "wb_reply", "wb_resolve"] }
  ]
}
```

### 样本 B：评审型能力通用协议（评审型 8 个能力，capability-review-v2.md §1 引言）

> 「评审型能力也做动作拆分，但同一评审协议（各审各的 4 步）——差异只在各动作的 reference/focus」：
> `review`（读可见产物 → 按 focus 核对）→ `<role>.comment`（wb_comment 锚到对应产物）→ `<role>.approve|block`（wb_review pass/block）→ `<role>.confirm|reopen`（作者修订后确认/reopen）
> 评审判断依据的通用骨架：「判断依据 = 可见产物 + 模板 quality_gates」（capability-artifact-analysis.md L171）。

### 样本 C：`respond_to_cr` 的 query 属性过滤（capability-review-v2.md §0.1.1 + my-task-response-model.md L195-231）

- `reference: [{ "query": { "reviewerRole": "Code Reviewer", "assignee": "rd", "state": "open" } }]`
- 语义：评论产物（`document.comments[]` / `reviewerProducts[]`）每条带 `reviewerRole/assignee/state/anchor/thread/roundCount/text`，action 用可过滤字段描述「要看哪些对话」；host 注入产物内容，subagent 过滤后逐条处理。
- 引用规约：「文档/代码类产物 → productType（标准枚举）；对话/评论类产物 → query（属性过滤）。两者都不依赖流程内节点位置/名称」。

### 样本 D：inputs 声明 + trigger 枚举 + 产出链路（capability-review-v2.md §0 / §0.3、capability-artifact-analysis.md L247）

- `inputs: [{ productType: "rd_design", use_threads: true }]`——「用标准产物类型 ProductType，不写 node_key/order，保证能力跨流程复用」。
- trigger 统一枚举（7 个）：`node_enter / dev_complete / reviewer_feedback(open) / node_review_pending / issue_replied / reject_rolled_back / decide`——「动作只声明『由哪个 trigger 激活』；不允许各能力自造」。
- 产出链路：「产出文档/代码 → 按模板章节骨架撰写 + quality_gates 自检 → 写盘（doc_path）」；模板引用一律用章节名。

---

## 逐条对照

对每条纪律给出：**判定**（互补/冲突/分层——分层指「各管各的层面」）、**建议**（是否值得叠加进 v2）、**示例**（基于真实 how）。

### 1. Context pointer（指针措辞决定触发）


**判定：分层 + 部分互补。**

writing 的 context pointer 是「上下文里一行指针，措辞决定 agent 何时去取材料、取到什么」。五段式把「何时取动作 how」工程化成了两层结构：`trigger`（机器可判定的 branch，程序契约）和 `when_to_use`（自然语言场景说明，给人/subagent 读）。二者合起来正是动作级的 context pointer：`trigger` 是结构化 branch 边，`when_to_use` 是这条指针的措辞。

writing 对指针的措辞纪律三条：「front-load the leading word」「one trigger per branch / synonyms 合并」「cut identity the body already carries」。逐条对账：

- **one trigger per branch**：既有模型已强制（trigger 枚举封闭，「不允许各能力自造」，capability-review-v2.md §0.3①）——已满足，无需改。
- **措辞决定触发**：`when_to_use` 应尽量用 agent 届时真能判定的语言。`backend-dev.respond_to_cr` 已示范（`when_to_use` = 评论字段条件，与 `reference.query` 字段一致）；但 `backend-dev.self_test` 的 `when_to_use` = 「dev 动作产出代码后，提交前」是**时序/位置语言**，不是可判定字段——subagent 如何知道「dev 动作已产出代码」？要靠动作间顺序推断。
- **cut identity**：`when_to_use` 与 `trigger` 同义重复的尾巴（如「节点进入时」复述 node_enter）可裁。

**建议（值得叠加）**：v2 加一条「when_to_use 措辞纪律」——`when_to_use` = trigger 的自然语言展开 + 可判定属性条件（评论 `reviewerRole/assignee/state/anchor`、产物状态、quality_gates 结果），禁止用时序/位置依赖表述；每个动作一个 `when_to_use`，同义场景合并且删除与 trigger 重复的部分。理由：这是 pointer 纪律里五段式唯一还没有显式条文的部位，且与现有 `query` 属性语言同构、零新增机制。示例：`backend-dev.self_test.when_to_use` 建议改为「`wb_my_task` 返回的 actions 中 `backend-dev.dev` 已完成（产出代码）且尚无 open CR 时」——具体措辞由 HITL 定，方向是「可判定」。

### 2. 两 load（context load / cognitive load）


**判定：分层（既有模型是 disclosed reference 的工程实现）+ 互补。**

- **context load**：writing 说「只经指针触及的材料免 context load，代价是指针自身那一行」。现状完全符合：`capabilities.actions[].how` 不常驻任何上下文，subagent 调用 `wb_my_task` 才取到目标动作的 how（my-task-response-model.md：「动作从 capabilities.actions 读取，不建独立表」；「Leader 调度 → RD 调 wb_my_task → 返回 actions 含目标 action_id 的 when_to_use/how/reference/focus/tools」）。prompt 只传 `action_id`——即**动作级指针**。这是 writing「disclosed reference」条目的既有实现，互为印证。
- writing 的反向提醒：「always-loaded pointer 的每个词每个 turn 都花钱，剪枝更狠」。动作里常驻指针行 = `action_id/name/trigger/when_to_use`（Leader 调度、wb_my_task 返回、决策树都会带）。这几行的剪枝标准应高于 how 正文——现状 `name`（「开发实现」「响应CR」）已很紧凑，`when_to_use` 见第 1 条。
- **cognitive load**：writing 说它是「人的索引成本，不该最小化，该花在人的判断处」。既有模型里这一成本已压到最低（触发由 trigger 程序判定，无需人记「哪个动作在哪」）；剩下该花 cognitive load 的地方是**修订 how 的人**：改一个动作的 how 要知道「改哪里 = `capabilities.actions[].how`」（单一真相源，见第 11 条），以及「模板路径由 host 渲染，不需人在 how 里维护」（第 11 条环境即真相源）。

**建议（值得叠加）**：v2 加「load 显式记账」条目——动作 how 只经 `wb_my_task` 披露（保留），`action_id/name/trigger/when_to_use` 为常驻指针行、按指针剪枝标准维护；并写明「how 的安全修改面 = capabilities.actions[].how 一处，经过 wb_my_task 展开，prompt 不携带 how 全文」为显式规则（现在写在各推演稿里，未收敛成规范条文）。理由：现状已符合 writing 的机制，叠加成本近零，收益是防止未来实现方把 how 沉到常驻上下文（如直接注入所有能力 how）。

### 3. 信息层级（in-file step / in-file reference / disclosed reference）


**判定：分层（同一决策，两个模型的划分维度不同，不冲突）。**

writing 的阶梯讲「一份文档内，材料放哪一层」；五段式把「动作 how 这份执行文档的材料放哪」结构性分到了三处：

- **in-file step** = how 的正文动作序列（`backend-dev.submit.how` 的两步：`wb_add_artifact(...)` → `wb_comment(to=cr)`；`respond_to_cr.how` 的四步①→④）。
- **in-file reference** = 字段级参考：`reference`（取哪些产物，含 query 过滤规则）、`focus`（盯哪些章节/对照点）、`output`/`tools`（终点与工具）。这些是「按需查阅」的平铺对等集——符合 writing「legitimately flat peer-set」不算臭味。
- **disclosed reference** = 两个方向：a) 产物本体披露到 host 端（`wb_my_task` 按 productType/query 注入；`renderTpl` 渲染真实路径后才发 subagent）；b) 具体执行子文档（模板文件如 prd-template.md、dev-notes-template.md）经「模板路径 + 章节名」指针披露（§0.1 / §2.2）。

互补点：writing 的「branching 是 disclosure 的最佳测试——每分支都要的内联，只有部分分支才用的披露」在五段式有现成对应：`reference` 字段只披露「该动作要看的产物」，`inputs` 只声明该能力在哪些定义节点看什么。`backend-dev` 的 `inputs`（product_doc/rd_design/case_design/architecture）与单个动作的 `reference`（如 self_test 只取 rd_design+case_design）就是两级 disclosure。

**建议（值得叠加，轻）**：v2 加「披露边界」条目——how 正文只写 in-file step；对「取哪些产物」一律经 `reference/inputs` 披露，how 正文不写完整产物清单的结构性重复（见第 5 条 co-location、第 11 条 pruning）；模板细节一律「模板路径+章节名」指针披露，不把模板正文抄进 how。

### 4. Progressive disclosure（下移阶梯保护可读性）


**判定：互补（且已被“动作拆分”实现，值得点名为显式规则）。**

writing 核心句：「push behind a pointer what only some branches reach」。五段式里最成功的 disclosure 实例是**动作拆分**：`backend-dev` 一个节点拆出 4 个动作，每个动作只披露自己分支需要的 how/reference——`submit` 不需要读产物（reference: []），`respond_to_cr` 不需要设计文档，只披露评论 query。若四动作挤成一个 how，产物清单、工具、步骤全数内联，可读性崩坏。这与 writing 的 progressive disclosure 完全同向。

反例警示（可作 v2 负面示例）：`git`/`fs` 工具只在 `dev` 动作出现、`wb_reply/wb_resolve` 只在 `respond_to_cr` 出现——工具清单已经是按动作披露的；若某天合并动作，这里就是第一个受污染的字段。

**建议（值得叠加）**：v2 加「动作级 disclosure 规则」——每个动作的 how/reference/focus/tools/outputProductTypes 只含本动作分支所需的材料；「该动作分支用不到」是裁剪动作字段的合法理由（现有 `submit` 的 `reference: [], focus: []` 已是示范，写进规范背书这一做法）。理由：机制已存在，缺的是把它从「推演稿的默认做法」升格为「规范条文」。

### 5. Co-location（同概念内容放一处）


**判定：分层为主，存在一处可修的 scattering。**

writing 测试：「同概念的定义、规则、caveats 放同一标题下；scattered material 读起来不像给 agent 写的文档。scattering ≠ duplication：duplication 是同一意义两处重复，scattering 是一意义被拆散多处」。

对账 `backend-dev.dev`：同一份「《rd_design》及其用途」出现在**三个字段**——how 首句「读《rd_design》…按设计契约实现」、`reference`（productType: rd_design）、`focus`（「接口设计（契约）」「数据与存储设计」）。三者职责不同（how=执行顺序里先读谁、reference=取产物依据、focus=盯哪章），所以不算 duplication；但「读哪些产物」又确实在三处表达，属 scattering。

修法（writing 给的路）：给字段分工立规，让「概念的各面」各归其位、彼此邻接可查——`reference` 管「host 取什么」，`focus` 管「subagent 盯什么」，how 正文只写「动作序列」，不做静态清单重述。分工写在 schema 注释（即 co-location 定位到字段级而非段落级）。

**建议（值得叠加）**：v2 加「字段分工注释（schema 级 co-location）」——`reference`（取）与 `focus`（盯）与 how 正文（做）三者职责声明；how 正文不重述 `reference` 的静态产物清单（只保留带顺序/目的的引用，见第 11 条 two-sources 裁决），`focus` 与模板章节名同词同序。示例：`dev.how` 若保留「读《rd_design》」只写引用、把「接口设计（契约）/数据与存储设计」从 how 挪走（由 focus 承担）即消除三处散布——具体取舍见第 11 条，HITL 定。

### 6. Sprawl（文档过长、注意力稀释）


**判定：分层（五段式骨架是 anti-sprawl 结构）+ 一条前沿纪律缺失。**

五段式骨架本身就是 sprawl 防线：固定 5 段 + 动作拆分，天然限制单动作 how 的长度（样本 A 中最长的 `respond_to_cr.how` 4 步、约 200 字内）。writing 的 sprawl 警告在当前最大的威胁点是：

- `respond_to_cr.how` 已 4 步、每步带括号细节（挂在哪个数组、锚在哪），再叠加「评论过滤字段规则详解」「thread 语义」就会突破注意带宽；
- `when_to_use` 若追求「自然语言完全解释」会膨胀（第 1 条）。

**建议（值得叠加）**：v2 加「how 长度纪律」——单动作 how 超过约 5 步或单段超 3 句时，把规则性内容（过滤字段、thread 语义、锚点类型）下沉到 reference 或模板，正文只留步骤骨架；同时声明「五段式是段落标签不是强制五段」——某段无内容可省略（`submit` 的读产物段为空已示范）。理由：这是 writing 唯一直接量化可转成规范的纪律，成本低。

### 7. Steps and completion criteria（clarity / demand）


**判定：分层 + 强互补（五段式第③段 quality_gates 就是完成标准的既有工程化，只缺「句子化」）。**

- **clarity（可判定 done/not-done）**：quality_gates 是完成标准的检查表形态（「判断依据 = 可见产物 + 模板 quality_gates」；「产出文档/代码 → 按模板章节骨架撰写 + quality_gates 自检 → 写盘」）。但 quality_gates 住在模板文件里（外部），how 正文自己通常**没有收尾的完成标准句**——看样本 A：`self_test.how` 以「记实现-设计对照表」结束，没有「表内每条设计要点都有实现位置与一致/偏差结论」这类可判定句；`dev.how` 以「产出代码（待自测）」结束，产出物边界（什么算完成）由 `output` 字段暗示而非 how 声明。writing 说 vague bound 招「premature completion」，最弱的一环就是这些缺完成标准句的动作。
- **demand（要求多少）**：writing 说「every rule applied」式穷尽表述驱动 legwork。五段式里 demand 目前由模板 quality_gates 承载，how 里没有「全部/每个」级穷尽词（如「对所有 assignee=rd 的 open CR」）。
- **post-completion steps 的 pull 与 anti-rush**：`respond_to_cr.how` 第④步「wb_resolve(note) → CR 复核 confirm/reopen」把后续步骤（评审复核）当作可见的 pull——符合 writing「visible steps ahead supply the pull」；而 `dev`→`self_test`→`submit` 的动作拆分把后文藏进另一个动作（见第 8 条）。

**建议（值得叠加，重点条目）**：v2 加「完成标准句」——每个动作 how 末句以「可判定 + 穷尽」的标准收尾，作为该动作的 completion criterion；模板 quality_gates 是它的检查表，两者互相引用（自检 = 过一遍 gates，达标 = 完成标准达成）；「自检通过」类模糊 bound 一律替换为可判定的具体条件。示例（基于样本 A 改造，供 HITL 定稿）：
- `dev.how` 末句 →「产出代码（待自测）」改为「产出代码，对照《rd_design》契约全部接口与字段行为有实现位置」；
- `self_test.how` 末句 →「记实现-设计对照表」改为「实现-设计对照表逐条填写：每条设计要点有实现位置 + 一致/偏差结论，未实现项入『偏差与风险说明』」；
- `respond_to_cr.how` 末句 →「所有 assignee=rd 且 state=open 的 CR 均已 resolve（附修复说明）或已 wb_reply 澄清」。

### 8. When to split（by sequence / by invocation）


**判定：分层 + 两处冲突/缺口（同 trigger 多动作的选择、拆分防 rush 的效果依赖执行模型）。**

- **by sequence**：`backend-dev` 四动作拆分 = by sequence 的实例。writing 说拆分的理由之一是「把 post-completion steps 藏出视线防 rush」——但它的前提是「hiding 只跨真实 context boundary 有效；inline 调用后文仍在上下文里、清不掉」。**既有模型的执行形态决定此条是否生效**：`wb_my_task` 按节点展开「当前节点可执行的动作列表」（my-task-response-model.md L191-231「actions = 返回该 capability 的 actions」）——若是同一次返回推给同一 subagent，则 `dev/self_test/submit` 的 how 同时可见，隐藏失效，防 rush 只能靠第 7 条的完成标准 clarity；若每次调度是新 subagent/新上下文（Leader 对 RD「用 backend-dev.respond_to_cr 响应 CR」→ 新调度），则隐藏真实生效。**这条判定需要执行模型定案（HITL/后续票），v2 应显式写明两态下完成标准的写法差异**。
- **by invocation**（skill 特有）→ 对应动作的 `trigger + when_to_use`（何时用哪个动作），见第 1 条。
- **冲突点：同 trigger 多动作**。`self_test` 与 `submit` 共用 `trigger: dev_complete`——程序无法区分，选择依赖 `when_to_use` 的时序语言（「自测通过后」）。这正违背「trigger 是程序契约」的初衷，也与第 1 条「when_to_use 措辞可判定」冲突。出路（两种，HITL 选）：a) 给自测引入显式状态（如 self_test 动作完成 = 对照表写盘 + quality_gates 全过，submit 的 when_to_use 引用该状态）；b) 保留时序语言、接受「行动选错由完成标准兜底」。

**建议（值得叠加）**：v2 加「拆分标准」——一个动作的 how 讲不清「做什么/何时结束/何时触发」→ 拆；拆出的每个动作自带 trigger + when_to_use + own how + outputProductTypes（own how 保留，已符合「动作内嵌 own how」）；「同 trigger 多动作」必须能用可判定条件区分（现状 self_test/submit 例外处理见上）。

### 9. Leading words（预训练概念锚定）


**判定：互补（五段式已有隐式 leading words，缺显式登记与同义收敛）。**

writing：「compact concept 已活在模型 pretraining 里，作 token 重复即可锚定行为；造词要付定义 token，优先用已有词」；「hunt for restatements 用 leading word 退休」。

对账五段式现有措辞，已存在一批事实上的 leading words：

| 词 | 出现处（真实） | 锚定的行为区域 |
|---|---|---|
| 契约（接口设计（契约）/接口契约可对接/按设计契约实现） | backend-dev.dev、fe-tech-review §1.4、rd-tech-review §1.3 | 「接口是义务：照设计实现、不自造」 |
| 口径（自测/联调/验收口径） | self_test、qa-tech-review、rd-tech-review | 「以用例为准的判定基准」 |
| 各审各的 | 评审型 §1 多处 | 「评审职责边界」 |
| 章节名（引用一律用章节名） | 引用规范 | 「模板定位方式」 |
| 对照表（实现-设计对照表） | self_test、code-review | 「落地核验方式」 |

**建议（值得叠加）**：v2 加「leading words 登记 + 同义收敛」——把上表（或 HITL 增补版）登记为动作级标准词，每个词一句话释义；how 内禁止同义换词（如「接口约定」「测试基准」等同义漂移）；释义放哪=co-location 的 v2 附录（一处定义，reference 引用）。举例收缩：若 v2 定义「契约」=「《rd_design》『接口设计（契约）』『数据与存储设计』的字段与行为义务」，则 `dev.how` 可从「读《rd_design》(docs/…) + 《architecture》(docs/…) + 《product_doc》(docs/…) → 按设计契约实现（不造接口），对齐架构」收缩为「读《rd_design》《architecture》《product_doc》→ 按契约实现，对齐架构；产出代码（待自测）」——括号展开由释义承担（收缩幅度与路径去留绑定第 11 条裁决，示例仅演示方向）。

### 10. Negation（禁止性表述）


**判定：部分冲突（现有禁令大多「配了正向但没被规范」）+ 部分分层（受控枚举/环境安全两类硬护栏该保留禁令）。**

writing：「禁语把被禁行为拖进上下文、更可用；prompt positive，只有硬护栏保留禁令并配正向目标」。真实禁令盘点（capability-review-v2.md / artifact-visibility-rework.md / my-task-response-model.md）：

| 禁令原文 | 类型 | 正向目标是否已配 |
|---|---|---|
| 「不造接口」（backend-dev.dev.how） | 职责护栏 | 有（前句「按设计契约实现」） |
| 「FE 不得自造契约」（§1.4） | 职责护栏 | 有（「按『接口设计（前后端契约）』对照」） |
| 「不硬切分支」（rework §3 / my-task L188） | 环境安全护栏 | 有（「经 git worktree（session 级隔离）读取」） |
| 「禁止自造 productType / trigger」（§0.1、§0.3①） | 受控枚举护栏 | 有（「只能绑定上表已有 productType」） |
| 「不评排期」「不评技术路线」（pm-review/arch-tech-review） | 职责边界 | 部分（「PM 只评价值/优先级/范围」为正述，但「不评排期」是纯禁令句） |

结论：栏杆处（枚举、git 安全）保留禁令+正向同位句是正确做法（writing 认可 hard guardrail）；真正的改进面是「职责边界」类——「不评排期」可正述为「评审范围止于价值/优先级/范围，排期不在评审输入内」，且现在「各审各的」已是正向边界词（还是第 9 条的候选 leading word）。

**建议（值得叠加）**：v2 加「禁令审计」——每条「不 X」改或配「做 Y」；纯禁令句仅允许出现在（a）环境安全（git checkout/写盘路径）、（b）受控枚举（productType/trigger）两类硬护栏，且必须紧随正向同位句；职责边界类一律正述为「只审 X / 评审范围止于 Y」。自检段（quality_gates）用正向检查条目，不写「不要遗漏」类。示例改造：pm-review 判断依据中「不评排期」→「评审产出只覆盖价值/优先级/范围；排期不在评审范围」（正向边界句）。

### 11. Pruning（单一真相源 / 环境即真相源 / relevance / no-op）


**判定：强互补，且有**一个确定冲突**与**两个待裁决点**。**

**(a) 单一真相源（确定一致，需写成显式规则）**：既有模型已声明「how 唯一真相源 = `capabilities.actions[].how`，经 `wb_my_task` 展开，prompt 只传 action_id 不传 how 全文」——写作纪律的「keep each meaning in a single source of truth」已被实现为架构。但发现一处**现存重复违反**：`docs-tmp/capability-artifact-analysis.md` §4 的 A1 行动表（「按契约实现、不造接口、对齐架构」）与 `backend-dev.dev.how` 是同一 meaning 的第二份副本（推演稿之间的 how 重述）。→ v2 声明「推演稿/评审表不再复制 how 全文，一律以 action_id 引用」并清理该处（或标注为推演期速览、定稿后删）。

**(b) 环境即真相源（缓存判定）**：`{{var}}` 模板路径与 `renderTpl`（宿主渲染）把「真实路径」归环境/宿主管（ctx 来自 requirements.version/slug、role_id）——how 里写「《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md)」是把环境 lookup 缓存给 subagent。writing 说 cache「只在查询昂贵时值得赚 load」——文档定位对 subagent 确实昂贵（跨仓库找设计文档成本高），所以这是**合理的 cache**。但存在两处重复：路径既在 §0.1 权威表又在每个 how 内联。**待裁决点 1（HITL）**：A（保留双写，靠 renderTpl 渲染一致性兜底——现状）vs B（单真相源：路径只在 §0.1 表/渲染层，how 只写《标准名》）。倾向 B（writing 的单一真相源直接适用），但 B 要求 host 注入的产物对象带 href 供 subagent 定位，依赖 execution-loop 的注入行为定案。

**(c) relevance（段可空可省）**：`backend-dev.submit` 的 `reference: [], focus: []` 是「某段无内容就空着」的示范（已符合 relevance）。缺的是显式规则：「五段式各段无内容可省，不为凑段硬写」（并入第 6 条长度纪律）。另外 `dev` 动作的 `inputs` 含 `product_doc`（功能/AC）但 how 只提「读」未提用途——见 (d)。

**(d) no-op（默认已服从则删）**：how 里每个《产物》必须有动作正文层面的用途（顺序/目的/对照点），否则是 no-op 句。现行 how 大体合格（每份产物都进了动作语义：rd_design→按契约实现、case_design→口径自测、product_doc→功能边界），但「读 X」与「reference 注入 X」的张力需裁决：**待裁决点 2（HITL）**——若 wb_my_task 注入产物内容后，how 的「读《X》」只是信标（目的句已在步骤里），可保留；若注入不保证（readMode 差异），「读《X》」是指令而非 no-op，必须保留。这与 (b) 同源，合并裁决。

**建议（值得叠加，强）**：v2 加 pruning 组四条——①单一真相源声明（how=capabilities.actions[].how，推演稿不复制）；②环境即真相源（路径渲染归 host；how 内联路径 A/B 裁决后定稿）；③relevance（段可空可省、字段无默认值可省）；④no-op（how 中每个产物必须有步骤层用途，否则删）。理由：唯一一条既有模型「部分违约」的纪律（跨文档重复），也是收尾收益最大的一条。

---

## v2 条目清单草案

> 供票 05 HITL 逐条拍板：每条一行（保留/叠加/修改 + 一句话），标注与 writing 纪律的对应。术语保留英文。

1. **保留五段式骨架**：how 保持「读产物 → 按模板 → 自检 → 登记/提评/响应 → 输出」结构；各段无内容可省（relevance，对应第 6/11 条）。
2. **叠加 when_to_use 措辞纪律**：`when_to_use` = trigger 展开 + 可判定字段条件，禁时序/位置依赖表述（第 1 条）。
3. **保留 trigger 枚举封闭**：trigger 即程序化 branch（one trigger per branch），不许自造（第 1 条）。
4. **叠加动作级披露规则**：动作的 how/reference/focus/tools/outputProductTypes 只含本动作分支材料（progressive disclosure，第 4 条）。
5. **叠加字段分工注释（schema 级 co-location）**：`reference`=host 取产物、`focus`=subagent 盯章、how=动作序列，how 不重述静态清单（第 5 条）。
6. **叠加 how 长度纪律**：单动作 how 超约 5 步/单段超 3 句 → 规则性内容下沉 reference/模板（sprawl，第 6 条）。
7. **叠加完成标准句**：每动作 how 末句 = 可判定 + 穷尽的 completion criterion；quality_gates = 该标准的检查表，自检达标即完成（clarity/demand，第 7 条）。
8. **叠加拆分标准**：how 讲不清「做什么/何时结束/何时触发」→ 拆；拆出动作自带 trigger + when_to_use + own how + outputProductTypes（when to split，第 8 条）。
9. **修改同 trigger 多动作选择**：`dev_complete` 同时挂 `self_test`/`submit` 的选择条件需显式状态化（或明示接受时序语言 + 完成标准兜底）——HITL 裁决两个方案（第 8 条）。
10. **叠加执行模型与完成标准的关系声明**：每次调度是同 subagent 串行还是新 subagent 新上下文，决定「隐藏后文防 rush」是否有效；两态下完成标准的写法差异写明（第 8 条）。
11. **叠加 leading words 登记**：契约/口径/各审各的/章节名/对照表等登记为动作级标准词并给一句话释义；how 内禁止同义换词（第 9 条）。
12. **叠加禁令审计**：每条「不 X」配「做 Y」；纯禁令仅限环境安全/受控枚举两类硬护栏（第 10 条）。
13. **叠加单一真相源声明**：how 唯一真相源 = `capabilities.actions[].how`，经 `wb_my_task` 展开，prompt 只传 action_id；推演稿/表格不复制 how 全文，`capability-artifact-analysis.md` A1 行动表等现存重复待清理（第 11 条 (a)）。
14. **修改路径真相源（A/B 裁决）**：`{{var}}` 模板路径只由渲染层/§0.1 表生成 vs how 内联保留双写——选 B（单真相源）则 how 只写《标准名》，依赖 host 注入产物 href（第 11 条 (b)）。
15. **叠加 no-op 审计**：how 中每个《产物》须有步骤层用途（顺序/目的/对照点），否则从 how 删（reference 仍可声明）（第 11 条 (d)）。
16. **保留 query 属性过滤规约**：对话/评论类产物用 `reference.query`（reviewerRole/assignee/state/anchor），即 pointer 措辞的工程化基础，与第 2 条互相支撑（第 1 条 + 既有规约）。