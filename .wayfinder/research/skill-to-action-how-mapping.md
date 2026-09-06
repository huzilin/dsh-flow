# 技能 → 能力×动作 how 参考映射（初稿）

> status: research findings · 角度 B（新判定维度）· created 2026-08-27
> 用途：供 `.wayfinder/tickets/05-how-v2-decision.md`（how v2 逐条定夺）与 `06-skill-adopt-decisions.md`（采纳判定）引用；初稿性质，挂载点疑难点见 §2.3 待用户批注裁决。
> 一手源（全部逐篇阅读，非凭名字推断）：
> - 三份技能盘点：`research/engineering-inventory.md`（18）、`research/productivity-inventory.md`（7）、`research/misc-inprogress-inventory.md`（12）＝37 个技能，含功能/机制/触发方式/旧维度初步判定
> - 能力与动作权威源：`docs-tmp/capability-review-v2.md`（§0.1 ProductType 权威表、§0.2 动作拆分 JSON、§1 评审型 8 能力、§2 产出型 11 能力；backend-dev 完整范例 dev/self_test/submit/respond_to_cr）
> - how 写作规范对照：`research/how-writing-comparison.md`（11 纪律 × 五段式，v2 草案 16 条）

## 判定维度说明（角度 B 与旧维度 Q4=A 的差异）

旧维度（Q4=A）：按「当前阶段能否改进流程/能力/产物/动作**推演工作**」分档——当前推演期采纳 / 插件实现期激活 / 不适用；编码向技能一律标「插件实现期激活」不深入。

本映射（角度 B）：不问「流程性帮助」，问「**每个技能应挂到哪些能力×动作的 how 上作参考**」——19 能力×动作的 how 编写时，吸收该技能的哪个纪律/机制作为参考素材。挂载点分三层：

1. **进动作 how**：挂到具体能力×动作（action_id 表述，见下）；
2. **进全局 how 编写规范 v2**：挂到「how 编写规范 v2」（全局写作纪律，非某个动作——如 writing-for-agents、交接/响应类纪律）；
3. **不进动作 how**：仅作流程层（推演会话/台账/方法论）或工具层（插件实现期/环境基建）参考，或无关。

挂载点命名（沿用能力与动作权威源 §0.2 模式）：
- 产出型动作：`<cap>.<verb>`，如 `backend-dev.dev` / `backend-dev.self_test` / `backend-dev.submit` / `backend-dev.respond_to_cr`；backend-dev 四动作 action_id 已定稿（§0.2），其余产出型能力动作的 action_id 未逐一枚举（§2.2 只给 how 骨架），本映射按其语义动作表述并在疑难点标注。
- 评审型能力动作：通用评审协议（§1 引言）`<cap>.review` / `<cap>.comment` / `<cap>.approve|block` / `<cap>.confirm|reopen`（各审各的 4 步，差异在 reference/focus）。
- `decide-cap`：excel 展开动作为 `decide-cap.create_reqs`（decide 门禁本身由程序契约承担，不展开为动作行）。

---

## 1. 主表：技能 → 能力×动作 how 参考映射

### 1.1 engineering 桶（18）

#### ask-matt（user-invoked）

- **挂载点**：**不进入动作 how，不挂 v2**（用户 2026-08-27 批注裁决：裁掉；疑难点 7）
- **说明**：技能路由器的主流程/on-ramp/vocabulary/standalone 分层与「按情境路由」决策树——**无增量**：同 trigger 多动作选择场景已被 execution-loop §11 决策树（程序判定）+ v2 草案第 9 条（状态化）覆盖；when_to_use 样例以本项目自产决策树为参照即可，接不进当前流程（孤立论证）。本体在插件实现期可作工作台技能路由/帮助入口参考（工具层），仅此。
- **旧维度**：当前推演期 → 角度 B 已裁（流程层/工具层，不挂任何 how）。

#### code-review（model-invoked）

- **挂载点**：`code-review.review`（评审型通用协议 review 动作）——**单动作、how 内串行两子步**（用户 2026-08-27 裁决，疑难点 5 选 B；理由：评审位一次表态语义→不拆双 gate；两轴产出接续（先核对实现照没照设计写，再看设计忠不忠实需求）→自然串行；并行只是执行细节）
- **吸收内容**：双轴评审纪律——Standards 轴（仓库文档化编码标准 + 内置 smell 基线）与 Spec 轴（是否忠实实现发起 issue/spec）；对位本项目 `code-review` 能力：Standards 轴 ≈ worktree 读代码 → `dev.md`「实现-设计对照表」逐条核对 + 编码规约基线（产出「对照偏差清单」）；Spec 轴 ≈ 设计契约对原需求/AC（产出「回归风险说明」）；两轴聚合后表态 approve|block → 走 confirm|reopen 复核（两级评审走通用评审协议，不合并排序）。
- **how 骨架（含 v2 完成标准句）**：子步 1 Standards 轴 → 子步 2 Spec 轴（基于子步 1 结果）→ 聚合表态；完成标准 =「两轴清单齐 + 逐条判定成立（无空话）」。执行量大时可两轴各派独立 subagent 并行后聚合（执行层并行、结构层仍单动作）。
- **旧维度**：插件实现期激活 → **角度 B 重新挂进动作 how**（本任务的指定例证之一）。

#### tdd（model-invoked）

- **挂载点**：`backend-dev.dev` / `frontend-dev.dev`（**合并后单动作内含自测子步**——用户 2026-08-27 裁决：dev + self_test 可以合并，如上设计）
- **吸收内容**：合并后的 dev 动作 how 分三子步——子步① 实现（垂直切片一次一测试一实现、测试经公共接口验证行为、不 mock 内部/不自证预期值/不横向铺测试）；子步② 自测（按自测方案+case_design 口径，先红后绿、回归防回退）；子步③ 记录（实现-设计对照表）；完成标准=自测全绿+对照表逐行成立。**trigger 链简化**：node_enter → dev（含自测）→ dev_complete → submit（唯一动作）——v2 草案第 9 条同 trigger 多动作冲突随合并消解；self_test 独立 trigger 消失，自测完成成为 dev 的完成标准而非独立信号。
- **旧维度**：插件实现期激活 → **角度 B 重新挂进动作 how**（结构重构：合并 dev/self_test；本任务的指定例证之一）。

#### diagnosing-bugs（model-invoked）

- **挂载点**：`backend-dev.respond_to_qa` / `frontend-dev.respond_to_qa`（**新动作，QA 验收缺陷返工响应**——用户 2026-08-27 批注：现有流程只有 respond_to_cr，遗漏了 QA 打回缺陷的对等响应动作；execution-loop 决策树②「respond_to_cr（或 QA 返工闭环）」的 QA 返工从未显式动作化）；次挂 `acceptance-test`（缺陷详情判定步：复现最小化后再下「预期/实际」结论）
- **吸收内容**：①「Phase 1 先建 red-capable/deterministic/fast/agent-runnable 的 tight feedback loop，**没建出就禁止假设**」+「Phase 4 一次一变量探针」→ respond_to_qa 的 how：读《defect》(docs/{{req-version}}/{{req-name}}/qa/defect.md) 归属我的条目 → 按「复现」重建并最小化（逐元素 load-bearing）→ 一次一变量定位 → 修在正确接缝（无接缝=架构发现，记入 dev_notes 对照表）→ 补回归 → resolve → QA 复核 confirm/reopen；②复现最小化纪律的 QA 侧应用 → acceptance-test：断言失败先最小化复现再写 defect 详情（预期/实际来自最小复现，flaky 不归域）
- **新动作设计（与 respond_to_cr 对称，trigger 契约同）**：`action_id: backend-dev.respond_to_qa`，`trigger: reviewer_feedback(open)`，`when_to_use: QA 验收打回的缺陷条目中 assignee=rd（我）且 state=open 时`，`reference: [{query:{reviewerRole:"QA", assignee:"rd", state:"open"}}]`，`focus: [defect.缺陷详情（复现/预期/实际）, code_mr.对应行]`，`output: 修订代码 / resolve`——**待主推演（capability-review-v2）补建**（评估稿标注流程缺口）
- **旧维度**：插件实现期激活 → 角度 B 重新挂进动作 how（缺陷响应动作，本任务的指定例证之一）；并暴露流程缺口（缺 respond_to_qa 动作，移交 docs-tmp 主推演）。

#### codebase-design（model-invoked）

- **挂载点**：`backend-design`（产出《RD 技术设计》的「接口设计（契约）」章节）；`frontend-design`（前端接口设计）弱引用——**正式采用**（用户 2026-08-27 裁决：可以用，用法如下设计）
- **吸收内容（用法设计，用户委托设计）**——接口面原则 4 条（可判定、进模板 §3.5 说明块 + 质量门禁 major 级核对）：
  1. **小接口大实现（deep module）**：接口暴露面窄、行为藏在内部；评审问"删掉这个接口，系统损失可数吗"（deletion test）
  2. **依赖传入不创建（dependency-in）**：接口所需依赖由调用方传入，接口内不自行 new
  3. **返回值无副作用**：调用只返回结果，不改系统状态
  4. **seam 放行为边界**：接口放在可观测行为的分界处（one adapter = hypothetical seam / two = real）
  术语层：deep module / 接口面 / seam 三词条进受控术语（中文为主、英文括注，按 ProductType 同款治理禁止自造同义词；**不是** ProductType 枚举成员——它们是设计原则词汇）；backend-design 质量门禁加核对行。
- **旧维度**：插件实现期激活 → 角度 B **正式挂载**进 `backend-design`（疑难点 9 已裁决：进；模板 §3.5 + 门禁表待落地）；与 setup-ts-deep-modules 同源词汇，后者本体留工具层。

#### domain-modeling（model-invoked）

- **挂载点**：`arch-cap`（架构设计/架构规划产出《架构规划》的动作）、`product-design`（产品设计产出 product_doc）、`backend-design`（RD 技术设计产出 rd_design）——**设计类环节是新领域词汇的产生地**（用户 2026-08-27 批注）；同时作全局层「ProductType 受控枚举治理」的方法论来源（§0.1 命名受控、禁止自造 = 域模型的受控实现）
- **吸收内容**（用户批注原文）：「我理解这个是领域模型，那就应该是架构设计、产品设计、rd 技术设计环节，产生了新的领域词汇，需要在文档的编写中需要这个能力」——「挑战模糊术语、消歧重载词、用具体边界场景压力测试领域关系」四条纪律：设计文档（架构/product_doc/rd_design）编写时做术语消歧自检（新词汇落地成受控术语，不重新发明）；受控枚举（ProductType/trigger）涨词时按「场景压力测试」验证。
- **旧维度**：当前推演期 → 角度 B 落点 = 设计类产出动作（arch-cap/product-design/backend-design）+ 全局受控枚举治理（见疑难点 2 已按用户批注裁决）；决策即时记录走既有 semantica record_decision / OpenViking 记忆（流程层，不落动作 how）。

#### grill-with-docs（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：纠缠式澄清访谈协议（一次一问、每问带推荐答案、事实自查、决策归用户、达共识前不动手）服务于推演会话与用户的澄清循环，属**流程层参考**（与 grilling 同族）；19 能力动作的 how 不涉及与用户访谈。查证源在本项目是 docs-tmp/arch_design 推演稿而非代码库。

#### implement（user-invoked）

- **挂载点**：`backend-dev.dev` / `frontend-dev.dev`（弱）
- **吸收内容**：其「tdd → typecheck → 全量测试 → code-review → commit」动作序列本身即一个 tickets→实现→验收的 how 五段式模板，作开发动作 how 的序列骨架样例（与 §0.2 backend-dev 四动作序列相互印证）；本体（按 spec/tickets 落地）在插件实现期激活。
- **旧维度**：插件实现期激活 → 角度 B **弱挂载**（序列模板样例，非纪律本体）。

#### improve-codebase-architecture（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：编码向架构健康度扫描 + HTML 报告形态，属**工具层参考**（插件实现期对工作台插件代码做深化机会调查）；HTML 报告可视化形态可参考 docs-tmp/diagrams 推演产物可视化，非动作 how 素材。

#### prototype（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：一次性原型方法（LOGIC 分支推状态机、暴露完整状态、答案入库后删除）属**流程层方法论**——推演 flow 编排语义（并行组 join、部分回滚、门禁判定）时用迷你脚本模拟验证（与 gen_flow_rows.py 实践同路线）；「答案入库」纪律对应推演稿记录。「暴露状态」即推演应显式呈现中间态，可作推演稿呈现纪律的参考条目（v2 弱引用，不挂具体动作）。

#### research（model-invoked）

- **挂载点**：`research-cap`（产出《调研文档》的动作，ProductType `research`，版本级）
- **吸收内容**：整体吸收——「对一手源（官方文档、源码、spec、first-party API）调研、每个主张追到所属源头、写成单篇带引用的 Markdown、落仓库既有惯例处（本项目 = `research/`）」即调研文档产出动作的完整纪律；本映射文件本身即该技能实践。
- **旧维度**：当前推演期（已启用）→ 角度 B 明确挂进 `research-cap` 动作 how。

#### resolving-merge-conflicts（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：git 冲突解决纪律（按意图溯源、绝不 `--abort`、跑自动化检查）属**工具层参考**：插件实现期多分支并行（flow.json branch/parallelGroup 已有概念）真实冲突时启用；「绝不发明新行为」可作实现期质量基线。推演期无改进点。

#### setup-matt-pocock-skills（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：前置配置脚手架（issue tracker/标签/domain docs 三假设）——本项目 .workbench 已内建等价物，**不应照跑其 GitHub 默认流程**；属**工具层参考**：若采纳依赖这些配置的工程技能，用其「探索→呈现→逐节确认→写配置文档」模式为 .workbench 与既有文档惯例写适配说明（等价自定义 issue tracker 自由描述通道）。

#### to-spec（user-invoked）

- **挂载点**：**Out of Scope 纪律** → `arch-cap`（架构设计，写进《架构规划》）、`product-design`（产品设计，写进 product_doc）、`backend-design`（RD 技术设计，写进 rd_design）——三个设计类阶段都要引用并输出到对应文档（用户 2026-08-27 批注）；User Stories **不引入**（用户批注：本项目谁产出/谁响应已固定于流程模型，执行指示已由 action.how 承载；actor/want/benefit 三要素已被角色位+需求条目+需求对齐 value 记录消化，无独立价值）
- **吸收内容**：显式 **Out of Scope** 节（贴合用户「按既定 scope 裁剪规则推演」偏好：被想过的、但决定不做的事，写进架构规划/product_doc/rd_design 对应章节）；「seams 先于 spec 定、尽量高且少」对应「先定产物/流程节点边界再写需求」的推演次序（流程序）。
- **旧维度**：当前推演期（借鉴）→ 角度 B 裁决（2026-08-27）：Out of Scope 挂三设计阶段写文档；User Stories 不引入（需求对齐 value 记录已覆盖）。

#### to-tickets（user-invoked）

- **挂载点**：`decide-cap.create_reqs`（需求拆解动作：按架构拆解创建需求）；另作 flow 编排层思想来源
- **吸收内容**：「垂直切片（切穿所有层、完成可独立演示/验证、单工单适配一个 fresh context window）」作需求/节点拆分粒度判据（推演节点粒度自问：该节点产物能独立验收吗）；「blocking 先于执行声明」作节点依赖显式化纪律；「work the frontier」作门禁放行序设计（flow.json 的 blocking edges/并行组 join/部分回滚即其本地实现）。
- **旧维度**：当前推演期（借鉴，flow.json 高度同构）→ 角度 B 挂进 `decide-cap.create_reqs`。

#### triage（user-invoked）

- **挂载点**：`decide-cap.create_reqs`（查重纪律）；另作 inbox/req 状态机流程层增强
- **吸收内容**：「redundancy 检查（按领域概念搜是否已实现/已推演，而非字面搜）」强化 create_reqs 已有查重逻辑（「相似则修改原需求」）；「verify claim（先复现/确证再推进）」作需求受理的验证纪律；五状态角色（needs-triage/needs-info/ready-for-agent/ready-for-human/wontfix）补充 req 状态机「澄清中」语义；`.out-of-scope/` 被拒知识库 = 用户 scope 裁剪规则的持久落点（流程层）。触发面须裁剪：本项目需求多为自建，triage 只处理外来流入。
- **旧维度**：当前推演期（借鉴，inbox.json 同构）→ 角度 B 查重纪律挂进 create_reqs，状态机部分留流程层。

#### wayfinder（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：雾状大想法地图方法（Destination/Notes/Decisions/Not yet specified/Out of scope 五节、refer by name、progress 从 tickets 推导、map 是 index 不是 store）属**流程层**——本项目 `.wayfinder/map.md` 即其实例（本项目已事实上实践）；其纪律是推演会话/台账规约，不是 19 能力动作的执行内容。

#### wizard（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：交互 bash 运维向导（只能人做的步骤：配 infra/凭据/CI secrets/一次性迁移）属**工具层参考**：插件实现期接入第三方服务、配置凭据时按此生成交互脚本；推演期几乎无改进点。注：任务描述将其记为 user-invoked，但盘点一手源（engineering-inventory.md）记录为 model-invoked（撞到只有人能过的墙时自动拉起）——不影响「不进动作 how」结论。

### 1.2 productivity 桶（7）

#### grill-me（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：一问一答单题访谈协议，与 grilling 同作者系同构、仅节奏差异（grill-me 等反馈、grilling 一轮问整个 frontier）。属**流程层**：可并入 grilling 使用说明作「轻量单题节奏」档，不必双技能常驻；与 19 能力动作 how 无挂载关系。

#### grilling（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：frontier 轮次制决策收敛协议属**流程层**（推演会话 consult，map.md 已约定每次会话 consult grilling/domain-modeling）——「事实自查、决策归用户、frontier 清空即结束、无静默假设」是推演会话与用户交互的决策协议，不是 19 能力子代理执行动作的 how 素材；与 grill-with-docs/grill-me/to-questionnaire/wait-what 同属「澄清/收敛协议族」。

#### handoff（user-invoked）

- **挂载点**：**how 编写规范 v2**（登记/交接环节；进阶动作挂载见 claude-handoff）
- **吸收内容**：三条纪律进 v2——①交接文档只引用产物路径/URL、不重复 specs/plans/ADRs/issues/commits 已捕获内容（防上下文膨胀，呼应 v2 草案第 13 条单一真相源、第 15 条 no-op 审计）；②suggested skills 节（续接会话先加载哪些技能）；③脱敏。落点按项目约定改 `.wayfinder/`（map/tickets 已是台账）。
- **旧维度**：当前推演期采纳（裁剪）→ 角度 B 挂进 v2 规范，不挂具体能力动作（交接是会话层，不是 19 能力动作）。

#### teach（user-invoked）

- **挂载点**：**不进入动作 how（无关）**
- **说明**：教学 workspace 产出（lesson/reference/learning-record）与流程/能力/动作推演无交集，推演期无固定教学环节；不进插件实现期清单（盘点判定：不适用）。

#### to-questionnaire（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：异步问卷协议属**流程层**（与 grilling 配套：grilling 当面盘、to-questionnaire 异步盘）——用于向领域用户收集 scope 裁剪规则、向评审位收集评审口径、把判定标准分歧转成问卷；「grill the send, not the subject」与本项目「事实自查、决策问人」哲学同构。产出小文档落 docs-tmp/ 或 .wayfinder/，非 19 能力动作素材。

#### wait-what（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：失准重述协议属**流程层**（与 AGENTS.md 沟通准则合并为「失准即重述」约定即可，不必常驻）。裁剪三点：STE100 → 中文受控表述（短句、单义、少嵌套）；ubiquitous language → 项目既有受控词表（ProductType 枚举、统一 trigger 枚举、评审协议、模板章节名规约本身就是域的 ubiquitous language）；CONTEXT.md 落点 → 项目域模型文档。其「重述只用受控词」与 v2 草案第 11 条 leading words 登记互证，可作 v2 引用而非挂载。

#### writing-for-agents（model-invoked）

- **挂载点**：**how 编写规范 v2**（全局写作纪律本体，非某个动作）——本映射的核心行
- **吸收内容**：整体吸收 how-writing-comparison.md 的逐条结论：11 纪律 × 五段式判定（分层为主：wb_my_task/动作拆分 = disclosed reference 既有实现；强互补 2：completion criteria × quality_gates、单一真相源）→ 产出 v2 条目清单草案 **16 条**（when_to_use 措辞纪律 #2、动作级披露 #4、字段分工 co-location #5、how 长度纪律 #6、完成标准句 #7、拆分标准 #8、leading words 登记 #11、禁令审计 #12、单一真相源 #13、路径真相源 A/B 裁决 #14、no-op 审计 #15、query 属性过滤 #16 等）供工单 05 逐条拍板。三处待 HITL 裁决（同 trigger 多动作、防 rush 执行模型、模板路径 A/B）。
- **旧维度**：当前推演期（靶心）→ 角度 B 挂载点为 v2 规范本身，所有动作 how 均受其约束（单一真相源：`capabilities.actions[].how`）。

### 1.3 misc / in-progress 桶（12）

#### git-guardrails-claude-code（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：Claude Code PreToolUse hook 拦截危险 git 命令，属**工具层参考**（低优先）：DSH 环境非 Claude Code，沙箱/审批机制已承担同类拦截，功能重叠；「明确拦截清单 + 可验证（喂 JSON 测 exit code）」写法可作实现期质量基线。插件实现期若用 Claude Code 写代码可直接照抄。

#### migrate-to-shoehorn（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：TS 测试 `as` 断言迁移工具，属**工具层参考**：插件实现期写插件测试时「部分数据 mock 保持类型安全」（fromPartial/fromAny/fromExact）是高频需求，届时按此迁移模式执行；「测试走公共接口」与 tdd 纪律在插件期协同。当前不深入。

#### scaffold-exercises（model-invoked）

- **挂载点**：**不进入动作 how，不进 v2**（用户 2026-08-27 批注裁决：裁掉；疑难点 8）
- **说明**：练习目录结构脚手架强绑定特定仓库工具链（pnpm ai-hero-cli internal lint），**无增量**：「骨架化 + 机器校验闭环」在本项目已有等价物——产物模板（docs/templates 18 个）+ 各模板附录质量门禁（severity gates 自检）+ 章节骨架标注（required/purpose 表），模板+门禁即机器校验闭环，无需外部机制。该技能整体不适用。

#### setup-pre-commit（model-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：Husky pre-commit + lint-staged 工程设置，属**工具层参考**：插件实现期启动代码仓库时即为基线工程设置；commit 门禁与 DSH approve/沙箱不重叠，属仓库自身体系。

#### claude-handoff（user-invoked）

- **挂载点**：**how 编写规范 v2**（登记/交接环节）；与 handoff 合并处理
- **吸收内容**：三条机制直接进 v2（盘点原文明确）：①摘要只引用产物路径、不重复内容（防上下文膨胀，呼应 v2 第 13/15 条）；②必含 suggested skills；③脱敏。命令本体（`claude --bg`）绑定 Claude Code，替换为 DSH 对应物（subagent/goal 接力）即可复用全部写法。
- **旧维度**：当前推演期采纳 → 角度 B 挂进 v2 规范（交接环节），不挂具体能力动作。

#### implement-spec（user-invoked）

- **挂载点**：**不进入动作 how**（主体）；frontier 编排思想归流程层
- **说明**：task-graph/frontier 多 implementer 并发编排——主体属**工具层**（插件实现期 spec→工单→并发的开发主流程）；推演期借鉴的是 frontier（「当前就绪可抓取的节点集合」）概念，用于描述推演任务推进顺序与批量作业分派，写进 how 规范的推进策略（流程层）。

#### loop-me（user-invoked）

- **挂载点**：**how 编写规范 v2**（响应/输出环节）
- **吸收内容**（盘点明确「可直接吸收进 how 规范 v2」）：①**brief 原则**——给用户呈现「决策就绪摘要」而非原始输出（推演稿呈现纪律）；②**push right**——检查点后移、一次问清（推演批注/评审的节奏设计）；③**trigger**（event 优先于 schedule）对应流程触发设计（与统一 trigger 枚举互证）；④「做到无疑问才 done」对应推演产物完成标准。词汇层：trigger/checkpoint/push right/brief 可直接成为推演动作标准术语。
- **旧维度**：当前推演期采纳（高相关）→ 角度 B 挂进 v2。

#### retro（user-invoked）

- **挂载点**：**how 编写规范 v2**（动作推演审查维度表）
- **吸收内容**：七个环境改进候选类别（Navigation/Automated checks/Coding standards/Global AGENTS.md/Tool economy/No-ops/Information access）直接用作**动作推演/流程审查的维度表**——每项对应一个可推演的动作改进点（导航指针、自动检查、编码标准强制方、no-op 句删除）；「编码标准应由 review agent 强制而非实现 agent」「AGENTS.md 只放导航指针」与项目 AGENTS.md 治理方向一致，写 how 规范时引用。
- **旧维度**：当前推演期采纳 → 角度 B 挂进 v2（审查维度表），非具体动作。

#### setup-ts-deep-modules（user-invoked）

- **挂载点**：**不进入动作 how**
- **说明**：dependency-cruiser deep module 边界配置，属**工具层参考**：插件实现期 TS 包边界基线（「证明规则咬人」的完成判据写法值得写进实现规范）；deep-module 词汇（公开面小而窄、实现藏深、接缝）推演期有隐喻价值，与 codebase-design 一起作产物/能力边界划分的推演语言（见疑难点 9）。

#### writing-beats（user-invoked）

- **挂载点**：`align.discuss`（需求对齐沟通环节的访谈过程，biz/tech/task 需求对齐节点语义）——collect/process 素材；不进入调研文档
- **说明**（用户 2026-08-27 批注：research-cap 调研产出**不合适**——writing 系是**访谈模式**，调研是读竞品/一手源，两者性质不同；应挂**需求对齐环节**「align.discuss（沟通访谈）+ alignment-cap.produce（产出对齐文档）」；两环节同质，差异仅「后接评审与否 + 输入可否看需求池」）：exploit 概念级旅程写作——需求对齐的用户访谈（目标/范围/价值沟通）概念多、依赖深时用 beats 作概念级控制（「每 beat 需要什么 + ground 什么」= 访谈结论术语控制自检）；grounding 纪律（概念落地才可用）用于访谈产出不得引入未对齐术语。挂载层级（align.discuss vs alignment-cap.produce）见疑难点 1（按用户批注裁决：三件套挂需求对齐环节）。

#### writing-fragments（user-invoked）

- **挂载点**：`align.discuss`（需求对齐**访谈素材收集步**：访谈/沟通碎片沉淀）；`alignment-cap.produce`（弱，产出对齐文档前的素材整理）；**不挂 research-cap**（用户 2026-08-27 批注）
- **吸收内容**（用户批注：需求对齐环节 align.discuss、alignment-cap.produce 都适用；同质，差异=后接评审 + 输入可否看需求池）：explore 阶段碎片沉淀（fragments 单文件、静默追加、每次写前重读磁盘、leading word 是最值钱片段）——需求对齐访谈中用户表达的目标/范围/价值片段、评审批注草稿等散点发现用此模式沉淀进 align.discuss 的素材收集步；**leading word**（命名整篇的隐喻词）是「受控术语」的写作侧机制，与 writing-for-agents 的 leading words（v2 第 11 条）同源互证。对齐产出（alignment）是版本级术语交汇处，访谈碎片中的新词按此收敛成受控词。
- **旧维度**：当前推演期采纳 → 角度 B 挂载 = 需求对齐环节（align.discuss + alignment-cap.produce），非调研（按用户批注从 research-cap 移出）。

#### writing-shape（user-invoked）

- **挂载点**：`alignment-cap.produce`（产出《需求对齐文档》align.md 的**成型协议**，主落点）；`align.discuss`（弱，访谈结论落 discussion 的成型）；**不挂 research-cap**（用户 2026-08-27 批注）；同时是推演稿（会话层产物）的成型协议
- **吸收内容**：①「开头承诺别漂移」= 对齐文档论点一致性检查（版本主题/目标范围全文守约）；②格式论证（prose vs list、table vs 重复结构每种格式选择要有意识、可辩护）= 对齐文档形态取舍纪律（align.md 用 arch-version-scope-align-template：版本主题与目标→需求对齐清单→范围边界→工作量评估→排期→风险→变更管理）；③gap-naming（pile 缺素材就明说「这里需要例子而 pile 没有，给我一个，否则砍掉这节」）= 对齐/推演中「证据缺失就明说并给选项」的处理纪律。capability-review-v2 式推演稿建议用 shape 作主档成型。
- **旧维度**：当前推演期采纳 → 角度 B 挂载 = alignment-cap.produce 成型协议 + 推演稿协议（按用户批注从 research-cap 移出；需求对齐讨论产出规划文档的形态即访谈→成文，正合 shape）。

---

## 2. 汇总段

### 2.1 三类统计（37 技能）

**① 进动作 how（13）**：`code-review`（code-review.review）、`tdd`（backend-dev.dev / backend-dev.self_test）、`diagnosing-bugs`（**backend-dev.respond_to_qa / frontend-dev.respond_to_qa 新动作** + acceptance-test 缺陷判定步；用户 2026-08-27 批注，原 dev 缺陷分支撤销）、`codebase-design`（backend-design 接口设计，弱）、`domain-modeling`（arch-cap/product-design/backend-design 设计类术语治理，兼全局受控枚举治理；用户 2026-08-27 批注修正，原 alignment-cap 撤销）、`research`（research-cap）、`to-spec`（Out of Scope 进三设计阶段文档 + 用户故事进背景章节；用户 2026-08-27 批注）、`to-tickets`（decide-cap.create_reqs）、`triage`（decide-cap.create_reqs 查重）、`implement`（backend-dev.dev 序列模板，弱）、`writing-shape`（**alignment-cap.produce 成型协议**，用户批注修正，原 research-cap 撤销）、`writing-fragments`（**align.discuss 访谈素材收集**，用户批注修正，原 research-cap 撤销）、`writing-beats`（align.discuss 概念级访谈控制，随批注并入）

**② 进全局 how 编写规范 v2（5）**：`writing-for-agents`（核心，16 条草案来源）、`handoff`（交接纪律）、`claude-handoff`（交接纪律，与 handoff 合并）、`loop-me`（brief/push right/trigger 完成标准）、`retro`（动作推演审查维度表）；另 `domain-modeling` 第二落点（受控枚举治理）与本类重叠。~~ask-matt / scaffold-exercises~~（2026-08-27 用户裁决裁掉，见疑难点 7/8）。

**③ 不进动作 how（19）**：
- 流程层（推演会话/台账/方法论，8）：`grill-with-docs`、`grill-me`、`grilling`、`to-questionnaire`、`wait-what`、`wayfinder`、`prototype`、`implement-spec`（frontier 思想）
- 工具层（插件实现期/环境基建，9）：`wizard`、`git-guardrails-claude-code`、`migrate-to-shoehorn`、`setup-pre-commit`、`setup-matt-pocock-skills`、`setup-ts-deep-modules`、`improve-codebase-architecture`、`resolving-merge-conflicts`、`ask-matt`（2026-08-27 裁决裁掉弱挂，仅留工具层）
- 无关（2）：`teach`、`scaffold-exercises`（2026-08-27 裁决裁掉）

（13 + 5 + 19 = 37，核对无遗漏。）

### 2.2 与旧维度（当前推演期/插件实现期）的差异说明

旧维度下标注「插件实现期激活」的编码向技能，按角度 B 有动作 how 挂载点的共 **5 个被重新挂进**：

| 技能 | 旧判定 | 角度 B 挂载点 |
|---|---|---|
| tdd | 插件实现期激活 | `backend-dev.dev` / `backend-dev.self_test`（垂直切片、红绿循环、公共接口） |
| code-review | 插件实现期激活 | `code-review.review`（双轴评审纪律） |
| diagnosing-bugs | 插件实现期激活 | **`backend-dev.respond_to_qa` / `frontend-dev.respond_to_qa`（新动作，QA 缺陷返工响应——流程缺口补建）** + acceptance-test 缺陷判定步（反馈环先行、最小化复现、无环禁假设） |
| codebase-design | 插件实现期激活 | `backend-design`（接口设计章节 deep module 原则，弱） |
| implement | 插件实现期激活 | `backend-dev.dev`（动作序列五段式模板样例，弱） |

其余插件实现期技能（wizard、resolving-merge-conflicts、setup-matt-pocock-skills、git-guardrails、migrate-to-shoehorn、setup-pre-commit、setup-ts-deep-modules、improve-codebase-architecture、implement-spec 主体）仍**不进动作 how**（工具层，插件期激活时使用）；`grill-me` 旧判插件期、角度 B 归流程层（与 grilling 合并，仍未进动作 how）。

旧维度「当前推演期采纳」的 20 个技能（engineering 9 + productivity 5 + misc 6），角度 B 下呈现三种去向：挂进具体动作（research/to-spec/to-tickets/triage/domain-modeling/writing-shape/fragments/beats 等）、挂进 v2 规范（writing-for-agents/handoff/claude-handoff/loop-me/retro，ask-matt 弱挂）、留在流程层（grilling/grill-with-docs/to-questionnaire/wait-what/wayfinder/prototype）。要点：**「当前推演期采纳」并不自动等于「进动作 how」**——对话澄清协议族（grilling 等 5 个）是推演会话与用户的交互协议，与 19 能力子代理执行动作的 how 无挂载关系，本映射明确将其留在流程层。

### 2.3 疑难点：挂载不明确、需用户批注裁决

> 已按用户批注裁决并回写主表：**疑难点 1**（writing 三件套）与 **疑难点 2**（domain-modeling）已于 2026-08-27 由用户批注定稿，见下「已裁决」节；以下为剩余待裁决项。

**已裁决（2026-08-27 用户批注）**：

1. **writing 三件套挂载层级（writing-beats/fragments/shape）** ✅已裁决：**不挂 research-cap**（调研产出不合适——writing 系是访谈模式，调研是读竞品/一手源，性质不同）；**改挂需求对齐环节**——`align.discuss`（biz/tech/task 需求对齐节点：User+PM 沟通目标/范围/价值，访谈素材收集 + 概念级控制）+ `alignment-cap.produce`（plan 立项需求对齐产出 align.md：成型协议）。两环节**同质**（访谈→成文），差异仅「后接评审与否（produce 接 D2 立项评审）+ 输入可否看需求池（produce 读版本需求池 + 路线，discuss 输入=用户沟通）」。beats/fragments → discuss 侧，shape → produce 侧（弱挂 discuss）。
2. **domain-modeling 落点** ✅已裁决（用户批注原文：「我理解这个是领域模型，那就应该是架构设计、产品设计、rd 技术设计环节，产生了新的领域词汇，需要在文档的编写中需要这个能力」）：**挂设计类环节**——`arch-cap`（架构规划）、`product-design`（product_doc）、`backend-design`（rd_design）；新领域词汇在这些环节产生，文档编写时做术语消歧自检；兼全局受控枚举治理（ProductType/trigger 涨词按场景压力测试验证）。原建议挂 `alignment-cap` 撤销。

**剩余待裁决**：

（无——全部 9 项已裁决，见下）

**已裁决（追加 2026-08-27）**：

4. **tdd 动作合并（dev + self_test）** ✅已裁决（用户：「可以合并」）+ 合并设计（委托实施）：**`backend-dev.dev` 单动作内含三子步（实现/自测/记录）**——子步① 实现（垂直切片一次一测试一实现、测试经公共接口、三条反模式）；子步② 自测（按自测方案+case_design 口径，先红后绿、回归防回退）；子步③ 记实现-设计对照表；完成标准=自测全绿+对照表逐行成立。**trigger 链简化**：node_enter → dev（含自测）→ dev_complete → submit（唯一）——v2 草案第 9 条同 trigger 多动作冲突随合并消解；self_test 独立 trigger 消失。frontend-dev 同构合并。
9. **deep-module 词汇** ✅已裁决（用户：「可以用，怎么用需要你来设计」）+ 用法设计：**接口面原则 4 条正式进 backend-design §3.5**——①小接口大实现（deletion test：删掉损失可数）；②依赖传入不创建；③返回值无副作用；④seam 放行为边界（one/two adapter）。三词条（deep module/接口面/seam）进受控术语（中文为主、英文括注，禁止自造同义，非 ProductType 枚举成员）；backend-design 质量门禁加核对行（major）。同源 setup-ts-deep-modules 本体留工具层。

3. **to-spec 落点（User Stories / Out of Scope）** ✅已裁决（2026-08-27 两次批注最终稿）：**User Stories 必要，作为文档背景章节的动机补充**——4 个模板已加「动机场景（用户故事）」节（prd §4.4、arch §2.3、rd §1.1、fe §1.1），格式 `As an <角色>, I want <功能/行为>, so that <收益>`，3–5 条轻量形态（非 to-spec 极长清单），骨架标注同步（引用规约章节名优先）；**Out of Scope 三设计阶段引用并输出写进文档**——`arch-cap`（架构规划）/ `product-design`（product_doc）/ `backend-design`（rd_design）；模板改动已落盘 docs/templates/（prd/arch/backend/fe 4 个 + 骨架标注），gen_flow_rows.py 章节引用待核对。
5. **code-review 双轴结构** ✅已裁决（2026-08-27 用户选 B）：**单动作 `code-review.review`、how 内串行两子步**——子步 1 Standards 轴（worktree 读代码 → 实现-设计对照表逐条核对 + 编码规约基线 → 对照偏差清单）→ 子步 2 Spec 轴（基于子步 1 结果：设计契约对原需求/AC → 回归风险说明）→ 两轴聚合完成标准（清单齐 + 逐条判定成立）→ 表态 approve|block → confirm|reopen 复核。不拆双 gate（评审一次表态语义）；并行仅作执行细节（两轴各派 subagent）。
7. **ask-matt** ✅已裁决（2026-08-27 用户裁决：裁掉）：无增量——同 trigger 多动作选择已被 execution-loop §11 决策树（程序判定）+ v2 草案第 9 条（状态化）覆盖；when_to_use 样例以本项目自产决策树为参照，接不进当前流程（孤立论证）。仅留工具层（插件期技能路由/帮助入口）。
8. **scaffold-exercises** ✅已裁决（2026-08-27 用户裁决：裁掉）：无增量——「骨架化+机器校验闭环」在本项目已有等价物（产物模板 + 各模板质量门禁 severity gates + 章节骨架标注 required/purpose 表），模板+门禁即机器校验闭环。整体不适用。

---

## 附：本映射与既有产物的关系

- 本文件是 `research` 技能实践产物（research 工单路径），供工单 05（how v2 逐条定夺）与工单 06（采纳判定）引用；挂载点裁决结果应回写本文件的疑难点小节形成定稿。
- 与三份盘点不矛盾：盘点答「什么时候用该技能」（旧维度），本映射答「how 编写时吸收该技能什么」（角度 B）；对 `research` 主挂载（research-cap）等「当前推演期采纳」技能，两维度答案一致，仅角度不同。
- 与 how-writing-comparison.md 的关系：writing-for-agents 行整体引用其 16 条草案（§2「v2 条目清单草案」），其余技能行只引用与之相关的条款编号（如单一真相源 #13、no-op #15、leading words #11、禁令审计 #12）。