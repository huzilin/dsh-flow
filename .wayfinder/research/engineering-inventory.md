# Engineering 桶盘点（18 个技能）

> 来源：`~/.dsh/skills/mp-<name>/SKILL.md` 逐篇全文阅读（18/18），非按名猜测。
> 判定标准（Q4=A）：按当前阶段「能否改进流程/能力/产物/动作推演工作」为第一标准；编码向技能（tdd、diagnosing-bugs、code-review 等）判定为「插件实现期激活」即可，不深入。
> 三档：**当前推演期**（对正在做的 flow/capability/action/how 推演有用，说明怎么用）/ **插件实现期激活**（编码向但项目开发阶段会用，说明何时激活）/ **不适用**。

---

## ask-matt

- **功能**：技能路由器——回答「该用哪个技能/流程」。给出整个包的总流程图景：主流程 idea→ship（grill-with-docs →（可选 prototype/handoff 绕行）→ to-spec → to-tickets → implement，内部驱动 tdd，收尾 code-review）；on-ramps（triage、diagnosing-bugs、wayfinder 汇入主流程）；codebase health（improve-codebase-architecture）；vocabulary 底层（domain-modeling、codebase-design）；standalone（grill-me/grilling、resolving-merge-conflicts、prototype、research、to-questionnaire、wizard、wait-what、teach、writing-for-agents）；precondition（setup-matt-pocock-skills）。路由判据：working directory→grill-with-docs（无目录→grill-me）；多会话构建→to-spec+to-tickets；超大会话迷雾→wayfinder；并含 context hygiene（smart zone/compact/phase boundaries）建议。
- **触发方式**：user-invoked（frontmatter `disable-model-invocation: true`）。
- **输入输出**：输入为用户情境（是否 working dir、单建/多会话、修 bug、维护、词汇问题）+「该用什么」的提问；输出为推荐技能/流程路径与阶段边界建议，无本地文件产物。
- **依赖**：其余 mp-* 技能的存在与语义（元技能）；概念依赖 PHASE-BOUNDARIES.md 的阶段树；无外部工具。
- **初步复用判定**：**当前推演期**。它是「技能/能力 → 流程路径」路由的已实现样例，本身即是 flow/capability 推演的直接素材：其「主流程/on-ramp/vocabulary/standalone」分层与「按情境路由（会话规模、有没有 working dir、是不是多会话）」的决策树，可映射为本项目动作层的路由规则设计；其 context hygiene / phase boundaries / smart zone 思想对 docs-tmp 推演的执行循环（上下文分段与交接）有启发。插件实现期可作为工作台的技能路由/帮助入口参考。

## code-review

- **功能**：对 HEAD 与固定点（commit/branch/tag/merge-base）之间 diff 做双轴评审——Standards（是否符合仓库文档化编码标准 + 内置 Fowler code smell 基线 12 条，仓库标准可覆盖基线）与 Spec（是否忠实实现发起 issue/spec）；两轴以并行 subagent 独立评审再聚合，分轴汇报、不合并排序；末尾一行总结各轴 findings 数与最重问题。spec 定位序：commit message issue 引用→用户参数→docs/specs/.scratch 下文件→问用户。
- **触发方式**：model-invoked（无 `disable-model-invocation`；用户说 「review since X」/PR/branch 时拉起）。
- **输入输出**：输入固定点 + diff + spec 来源 + standards 来源文件；输出两轴并排报告（`## Standards`、`## Spec`，逐 hunk 引用）+ 每轴一行总结。
- **依赖**：git（three-dot diff）、`docs/agents/issue-tracker.md`（issue 获取流程，缺失时提示跑 setup-matt-pocock-skills）、并行 subagent 能力、内嵌 smell 基线。
- **初步复用判定**：**插件实现期激活**。编码向技能：项目进入真实开发、diff 出现（工作台自身代码或多分支实现）时，按双轴流程评审，尤其 flow.json 的 Code Reviewer 节点可直接套用。推演期弱借用：双轴分离、不合并排序的评审纪律可映射为「推演产物评审 = 内容轴（是否对题）+ 规范轴（是否符合既有架构/裁剪规则）」，属隐喻借用，不深入。

## codebase-design

- **功能**：深模块（deep module）设计词汇与原则的共享语言层：module/interface/depth/seam/adapter/leverage/locality 精确术语（拒绝 component/service/API/boundary）；deep = 小接口大实现、shallow = 大接口小实现；原则含 deletion test、interface is the test surface、one adapter = hypothetical seam / two = real、依赖传入不创建/返回值不副作用/小表面积三项可测试性设计；附 DEEPENING.md、DESIGN-IT-TWICE.md（并行设计多次再比较）。
- **触发方式**：model-invoked（vocabulary 层；tdd、improve-codebase-architecture 等拉入，用户设计模块接口时直接可用）。
- **输入输出**：输入为模块/接口设计讨论或重构场景；输出为术语、深度判定与 seam 放置决策，无文件产物。
- **依赖**：无外部工具；依赖仓库上下文（在哪里放 seam）。
- **初步复用判定**：**插件实现期激活**。编码向模块设计词汇：实现工作台插件、设计其内部模块接口时按此语言落地。推演期仅隐喻借用：「能力封装深度」（行为多、接口小）可作为能力粒度推演的启发，非主用途。

## diagnosing-bugs

- **功能**：硬 bug / 性能回归的诊断纪律，六阶段：Phase 1 建 tight feedback loop（failing test→curl→CLI→headless browser→replay trace→throwaway harness→property/fuzz→bisection→differential→HITL script 十种方式；必须 red-capable（驱本 bug 断言用户症状）/deterministic/fast/agent-runnable，没建出就禁止假设，这是本技能核心）→ Phase 2 reproduce + minimise（每元素 load-bearing）→ Phase 3 出 3–5 个可证伪假设（不先验排名给用户过目）→ Phase 4 一次一变量探针（`[DEBUG-xxxx]` 唯一前缀标记；perf 先测后修）→ Phase 5 修 + regression test（只写在正确 seam，无 seam 本身就是架构发现）→ Phase 6 cleanup（含提交信息写对的那个假设）。全程先 redact 机密。
- **触发方式**：model-invoked（用户报 broken/throwing/failing/slow 时拉起）。
- **输入输出**：输入 bug 报告/现象/复现线索；输出根因、修复、回归测试，无 seam 时输出架构改进发现。
- **依赖**：可运行代码库与命令环境（测试/curl/浏览器/profiler）、CONTEXT.md/ADR 读取、可选 HITL 脚本模板。
- **初步复用判定**：**插件实现期激活**。编码向：实现期排查工作台插件/脚本 bug 时按「反馈环先行、禁止无环假设」执行。推演期注意：本项目 Python 推演脚本（gen_flow_rows.py 等）出 bug 时此流程同样适用，属开发阶段动作。

## domain-modeling

- **功能**：主动构建/打磨项目领域模型（不是读词汇，是改模型）：挑战与 CONTEXT.md 冲突的用词、把模糊/一词多义的说法凝练为规范术语、用具体边界场景压力测试领域关系、把用户说法与代码交叉验证；术语与决策融化即写（CONTEXT.md 术语表 + docs/adr/，懒创建；CONTEXT.md 不含实现细节，纯词汇表）；ADR 只在「难逆转 + 无上下文令人意外 + 真权衡」三条件全满足时提议。
- **触发方式**：model-invoked（grill-with-docs、triage、improve-codebase-architecture 内部调用；用户讨论术语/记决策时直接拉起）。
- **输入输出**：输入会话中的术语/决策讨论；输出 CONTEXT.md 更新、ADR 文件（懒创建）。
- **依赖**：仓库文件结构（CONTEXT.md/CONTEXT-MAP.md、docs/adr/）与格式模板（CONTEXT-FORMAT.md、ADR-FORMAT.md）。
- **初步复用判定**：**当前推演期**。本项目本身在做领域语言工作：flow/capability/action/产物是一套术语体系，「挑战模糊术语、消歧重载词（如 node key / capability 命名）、场景压力测试、决策即时记录」四条纪律可直接用于推演稿与受控枚举（ProductType）打磨。落点需对接既有惯例而非照搬路径：术语/方法论落 OpenViking 记忆、决策走 semantica record_decision（map 已约定），CONTEXT.md/ADR 的物理位置按本项目既有文档权威（docs-tmp、arch_design.md）替换。

## grill-with-docs

- **功能**：对计划/设计做纠缠式访谈直到共识：逐分支走设计树、一次只问一个问题（同时多问是迷惑）、每问带推荐答案；事实查代码库/项目文件，决策归用户并等答复；全程内嵌 domain-modeling（挑战术语、更新 CONTEXT.md/ADR）；达共识前不实施。
- **触发方式**：user-invoked（`disable-model-invocation: true`；working directory 场景，无目录用 grill-me）。
- **输入输出**：输入需打磨的计划/设计/想法；输出澄清后的共识 + CONTEXT.md/ADR 增量。
- **依赖**：代码库/项目文件（查事实用）、domain-modeling 技能。
- **初步复用判定**：**当前推演期**。本项目推演期大量「澄清需求/方案/偏好」场景，与用户「先 review 部分节点批注、再举一反三」「scope 裁剪而非全盘采纳」的工作偏好正好配合：把「事实查 docs-tmp/arch_design、决策逐问用户（一次一问+推荐答案）」制度化，再叠加既有评审批注循环即可；决策即时记录对接 semantica。注意本项目问题源常是推演稿而非代码，查证源相应替换。

## implement

- **功能**：按 spec/tickets 落地实现：能用 tdd 就用（预定的 seams）、常跑 typecheck 与单文件测试、结束跑全量测试、用 code-review 收尾、提交当前分支。极简指令型技能（全篇 15 行）。
- **触发方式**：user-invoked（`disable-model-invocation: true`）。
- **输入输出**：输入 spec 或 tickets（ask-matt 主流程中由 to-tickets 产出）；输出实现 + 测试通过 + 评审通过 + 提交。
- **依赖**：tdd、code-review 技能；测试/类型检查环境。
- **初步复用判定**：**插件实现期激活**。编码向动作：实现期按 ticket 开发时执行其「tdd→typecheck→全量测试→code-review→commit」动作序列。推演期仅作为 action-how 样例：其动作序列本身是一个 tickets→实现→验收的 how 五段式模板，可供动作推演参照，弱借用。

## improve-codebase-architecture

- **功能**：扫描代码库找深化机会（shallow→deep），产出自包含 HTML 报告（`$TMPDIR/architecture-review-<timestamp>.html`，Tailwind+Mermaid via CDN）：每位候选一张卡（files/problem/solution/benefit 以 locality+leverage 表述/before-after 图/推荐强度 Strong·Worth exploring·Speculative）+ Top recommendation；用户选一个后进入 grilling loop 深化设计（用 codebase-design 词汇、domain-modeling 边做边记，拒绝的 load-bearing 理由提议记 ADR）。探索用只读 subagent；不主动提接口，等用户挑。
- **触发方式**：user-invoked（`disable-model-invocation: true`）。
- **输入输出**：输入代码库；输出临时目录 HTML 架构报告 + 选定候选的深化设计讨论与文档增量。
- **依赖**：codebase-design、domain-modeling、HTML 报告模板与 CDN、TMPDIR/xdg-open、subagent。
- **初步复用判定**：**插件实现期激活**。编码向架构健康度扫描：实现期对工作台插件代码做深化机会调查时按此流程。推演期弱借用：HTML 可视化报告形态可参考本项目推演产物可视化（docs-tmp/diagrams），非主用途。

## prototype

- **功能**：一次性原型回答一个设计问题，问题决定形态：LOGIC 分支（「状态/逻辑模型感觉对吗」）→ 迷你交互终端 app 推状态机过难推的用例；UI 分支（「长什么样」）→ 多套激进变体单入口可切换。规则：从头就是 throwaway 并明示、靠近使用处、一条命令可跑、默认无持久化、跳过打磨、每次动作后暴露完整状态、用完删除或吸收；答案捕获到 commit/ADR/spec/NOTES.md 后删除。
- **触发方式**：model-invoked（设计问题纸上难决时拉起；ask-matt 主流程中 via handoff 绕行）。
- **输入输出**：输入设计问题（用户 prompt/周边代码/追问）；输出可运行一次性原型 + 记录的答案（问题+结论）。
- **依赖**：项目任务 runner 与语言/框架；LOGIC.md、UI.md 分支模板。
- **初步复用判定**：**当前推演期**（方法借用为主）。核心不是代码而是「用可运行的粗糙小物验证感觉」：本项目推演 flow 编排语义（并行组 join、部分回滚、门禁判定这类状态机问题）时，可照其 LOGIC 思路用迷你脚本模拟验证（与既有 gen_flow_rows.py 等脚本推演实践同路线）；「答案入库（NOTES.md/ADR）」纪律照搬为 docs-tmp 记录；「暴露状态」即推演应显式呈现中间态。编码原型实例本体在插件实现期激活。

## research

- **功能**：把阅读 legwork 委托给后台 subagent：对一手源（官方文档、源码、spec、first-party API）调研，每个主张追到所属源头，写成单篇带引用的 Markdown，落仓库既有惯例处（无惯例 → `.plan/research/<slug>.md`）。产出是拿来喂给 grill-with-docs/to-spec 的素材，不替代思考。
- **触发方式**：model-invoked（用户要求调研/查文档事实/委托读东西时拉起）。
- **输入输出**：输入研究问题；输出带引用的 Markdown 调研文件（路径回报）。
- **依赖**：subagent、一手源可及性、仓库既有笔记惯例。
- **初步复用判定**：**当前推演期**（已启用）。本任务即其实践：wayfinder map 的 research 票（tickets/01）按此技能执行，findings 落 `.wayfinder/research/`（map.md 约定 "research findings 落 research/"）。后续调研类票（02/03 桶盘点、04 how 对照等）一律照此执行。

## resolving-merge-conflicts

- **功能**：解决进行中的 merge/rebase 冲突：看当前状态 → 为每个冲突找一手源（commit message/PR/原 issue→原意图）→ 逐 hunk 按意图解决（两意尽量兼得；不可兼得取与 merge 目标一致者并记 trade-off；绝不发明新行为；永远 resolve、绝不 `--abort`）→ 跑仓库自动化检查（typecheck→tests→format）→ 完成并提交（rebase 则续到全部 rebase 完）。
- **触发方式**：model-invoked（处于冲突中时拉起）。
- **输入输出**：输入冲突中的 git 状态与相关提交历史；输出解决完的 merge/rebase 提交。
- **依赖**：git、原始 issue/PR 记录（意图溯源）、仓库检查命令。
- **初步复用判定**：**插件实现期激活**。编码向 git 操作：实现期多分支并行（flow.json 已有 branch/parallelGroup 概念）真实冲突时用其「按意图溯源解决、不 `--abort`」纪律。推演期无改进点。

## setup-matt-pocock-skills

- **功能**：工程技能族的前置仓库配置脚手架（一次性跑）：issue tracker 选型（GitHub 默认/ GitLab / local markdown `.scratch/<feature>/` / other 自由描述）、triage 标签词汇（五角色标签串，未装 triage 则跳过）、domain docs 布局（默认 single-context，monorepo 信号才谈 multi-context）；prompt-driven：探索（git remote、AGENTS.md/CLAUDE.md、CONTEXT.md、docs/adr/、docs/agents/、`.scratch/`）→ 逐节带推荐答案确认 → 写 `docs/agents/issue-tracker.md`、`domain.md`、（`triage-labels.md`）+ 编辑 CLAUDE.md 或 AGENTS.md 的 `## Agent skills` 块（两者不并存，不新建缺失者）。
- **触发方式**：user-invoked（`disable-model-invocation: true`；工程桶用前必跑）。
- **输入输出**：输入仓库现状；输出配置文档（docs/agents/*.md）+ AGENTS.md/CLAUDE.md 技能块。
- **依赖**：仓库探测对象（git/config/各文档）、triage 技能安装情况；种子模板文件。
- **初步复用判定**：**插件实现期激活**。它配置的是「issue tracker + 标签 + domain docs」三个假设，本项目已内建等价物（.workbench 即需求/issue 系统、文档惯例既有），不应照跑其 GitHub 默认流程；激活条件与用法：采纳任一依赖这些配置的工程技能（to-spec/to-tickets/triage/code-review）时，用其「探索→呈现→逐节确认→写配置文档」的模式，为 .workbench 与既有文档惯例写一份适配说明（等价于自定义 issue tracker 的自由描述通道），而非重造配置。

## tdd

- **功能**：red→green 循环的纪律参考：好测试经公共接口验证行为（代码随便改、测试不动）；seams 是测试落点，凡测试前先写下预定 seams 并与用户确认（seam 形态存疑时拉 codebase-design 词汇）；三条反模式——implementation-coupled（mock 内部/测私有/旁路验证）、tautological（自证预期值）、horizontal slicing（先写完所有测试再实现，应改垂直切片一次一测试一实现）；循环规则——先红后绿、一次一切片、重构不属于循环（归 code-review 阶段）。
- **触发方式**：model-invoked（实现/修 bug 时；implement 内部驱动）。
- **输入输出**：输入待建功能/待修 bug + 预定 seams；输出垂直切片的测试→实现序列。
- **依赖**：测试框架、codebase-design 词汇（seam 讨论时）。
- **初步复用判定**：**插件实现期激活**。编码向测试纪律：实现期开发工作台插件时执行；其「预定 seams 并确认」与「垂直切片」可视为 to-tickets 的切片原则在测试层的对应。推演期不深入。

## to-spec

- **功能**：把当前会话上下文+代码库理解合成一份 spec（PRD），不访谈用户（只综合已讨论内容）：先用 CONTEXT.md 词汇/尊重 ADR；先定测试 seams（越高越好，理想一个，与用户确认）；再按模板写并落 `.plan/<slug>/spec.md`。模板：Problem Statement / Solution / 极长编号 User Stories（As an actor, I want…） / Implementation Decisions（模块、接口、架构、schema、API 契约；不含文件路径与代码，prototype 决策富片段可内联）/ Testing Decisions（好测试定义、测哪些模块、先例）/ Out of Scope / Further Notes。
- **触发方式**：user-invoked（`disable-model-invocation: true`）。
- **输入输出**：输入会话上下文+代码库理解；输出 `.plan/<slug>/spec.md`（存版本库，项目共享规划记忆）。
- **依赖**：CONTEXT.md/ADR（如有）、`.plan/` 目录约定。
- **初步复用判定**：**当前推演期**（借鉴，详见对照节）。本项目 requirements.json 的 req 条目已内建等价骨架（description 背景/目标/验收点、reqType/scope/priority/owner/status），to-spec 补充价值在三点：User Stories 用户视角长列表（现有验收点偏验收人视角）、Implementation/Testing Decisions 分节（对应 RD/FE 技术设计节点内容应放哪）、Out of Scope 显式节（贴合用户「scope 裁剪」偏好）。落点应是增强既有 req 条目或推演稿，不引入 `.plan/` 新目录。

## to-tickets

- **功能**：把 plan/spec/会话拆成 tracer-bullet 垂直切片 tickets 集，每票声明 blocking edges：vertical slice 切穿所有层、完成后可独立演示/验证、单票适配一个 fresh context window、prefactor 先行；wide refactor（一处机械改动炸全库）是例外，拆 expand→migrate 批次（按 blast radius）→contract，必要时共享集成分支；quiz 用户三点（粒度/blocking 是否正确/合并拆分）后落 `.plan/<slug>/tickets.md`（依赖序、每票 Blocked by + 验收勾选清单）；work the frontier：blockers 全完成的票优先，一票一 fresh session：实现→静态检查+测试→评审→勾 AC→提交→清上下文。
- **触发方式**：user-invoked（`disable-model-invocation: true`）。
- **输入输出**：输入 spec/plan/会话（可传引用路径）；输出 tickets.md（不修改源文档）。
- **依赖**：CONTEXT.md/ADR（若有）、`.plan/` 约定、code-review/测试环境（工作票执行期）。
- **初步复用判定**：**当前推演期**（借鉴，详见对照节）。其切片与编排思想与本项目 flow.json 高度同构：垂直切片=「完成可独立验证」对应流程节点设计判据（节点应产出可独立演示的产物）；blocking edges=流程依赖与并行组 join 判定；wide refactor expand-contract=多分支并行与部分回滚的编排逻辑；frontier=门禁放行序。落点：拆票/节点粒度判据与 blocking 声明规则反哺 flow/capability 推演。

## triage

- **功能**：把「不是自己创建的」流入 issue（bug 报告、外来 feature 请求；可选外部 PR，PR=带代码的 issue）走小状态机：2 category（bug/enhancement）+ 5 state（needs-triage / needs-info / ready-for-agent / ready-for-human / wontfix）；对具体 issue：gather context → 两查（redundancy：按领域概念搜是否已实现；prior rejection：查 `.out-of-scope/` 知识库）→ recommend（给 maintainer 分类+状态建议，等方向）→ verify claim（复现 bug/确证 PR diff，出 confirmed/failed/insufficient）→ 必要时 grill（拉 grilling + domain-modeling）→ 应用结果（ready-for-agent 写 agent brief；needs-info 用模板；wontfix 按「已实现指位置 / 被拒 bug 解释 / 被拒 enhancement 写 .out-of-scope/」分流）；一切 AI 评论带免责声明；支持快速状态覆盖与上次会话续接。
- **触发方式**：user-invoked（`disable-model-invocation: true`）。
- **输入输出**：输入流入 issue/PR 与 maintainer 自然语言指令；输出状态转移 + agent brief/评论/关闭 + `.out-of-scope/` 知识库增量。
- **依赖**：issue tracker 配置与标签映射（setup-matt-pocock-skills 提供）、AGENT-BRIEF.md、`.out-of-scope/`、grilling/domain-modeling。
- **初步复用判定**：**当前推演期**（借鉴，详见对照节）。其流入处理流程与本项目 inbox.json→requirements.json 链路同构：inbox 消息（review/gate/mention）即流入通道，req 的 status/priority/owner 即简化状态机。可借鉴点：五状态角色与 needs-info 往返（对应 inbox 待复核/mention 闭环）、verify claim（复现/确证再进下一步）、redundancy 检查（按领域概念查 docs-tmp 是否已推演过，避免重复推演）、`.out-of-scope/` 知识库（= 用户「按既定 scope 裁剪规则推演而非简单标注」的落点）。须裁剪：本项目需求多为自己创建，triage 明确只处理外来流入，需放宽/定制触发面。

## wayfinder

- **功能**：为超出一个会话的雾状大想法（greenfield/大 feature）绘共享 map 并逐票解谜直到路清：先 grill 命名 destination（决定范围）→ breadth-first 面 frontier 画 fog → 建 map（Destination/Notes/Decisions so far/Not yet specified/Out of scope 五节）+ 现在能指定的 tickets → 二遍接线 blocking → 验证；工作模式：claim 票（先 claim 再干活）→ resolve（四类票：Research(AFK)/Prototype(HITL)/Grilling(HITL)/Task(HITL/AFK)，一次会话只 resolve 一票）→ record answer + Decisions so far 追加 gist+链接 → 毕业 fog/新开票 → 越界即 rule out of scope（closed≠resolved，不满足阻塞边）→ 前提被毁标 undermined。铁律：refer by name（不裸 id）；progress 从 tickets 推导、不写死计数；map 是 index 不是 store（事实只在票里）；产出决策而非交付物，路清即 hand off（进 to-spec，不直连 implement）。
- **触发方式**：user-invoked（`disable-model-invocation: true`）。
- **输入输出**：输入雾状大想法或既有 map；输出 map + decision tickets（+ 被引资产如 research md/原型）。
- **依赖**：adapter（默认 TRACKER-MARKDOWN.md → `.plan/`）、research/prototype/grilling/domain-modeling 技能、subagent 能力。
- **初步复用判定**：**当前推演期**（已事实上实践）。本项目 `.wayfinder/map.md` 就是本技能的直接实例：地图五节、tickets/01–06、blocked by、research 票、一次会话一票、「scope 裁剪而非全盘采纳」即其 Out of scope 语义。建议自查与模板差距：refer by name（票引用是否避免裸 id）、progress 是否从 tickets 推导、undermined 标记与「closed≠resolved」是否落实——差距项即推演改进点。

## wizard

- **功能**：生成交互 bash wizard 走人类只能自己做的步骤（配 infra、凭据/CI secrets、陌生第三方 dashboard、一次性迁移/切换）：复用 template.sh 库（stage 进度条、确认门、跨平台开 URL 含 WSL、隐藏秘钥输入、幂等 `.env` upsert、`gh secret`/`gh variable`、结束汇总；STAGES 标记以下统一，绝不手改库）；agent 只做 scope 流程+写 stages：读仓库（.env/README/CI 里每个 secrets.* 即要产出的值）→ 逐 stage 写人类路径（如 Dashboard→Developers→API keys→Reveal→copy）→ 静态 trace 验证（`bash -n`、shellcheck、不端到端跑——它开浏览器等人输入）→ 交出跑法；默认 ephemeral（一次性），可重复设置路径才入库。
- **触发方式**：model-invoked（撞到只有人能过的墙时自动拉起；agent 自己能做就不该用它）。
- **输入输出**：输入需人类手动的过程描述+仓库现状；输出交互 bash 脚本（+ 提示如何运行；可重复路径时提交并 link README）。
- **依赖**：template.sh、bash/shellcheck、目标服务与 CI 配置上下文。
- **初步复用判定**：**插件实现期激活**。编码/运维向：实现期接入第三方服务、配置凭据或做一次性数据迁移时按此生成交互脚本（本项目若开发工作台插件涉及 CI secrets/外部依赖即激活）。推演期几乎无改进点。

---

### 与 .workbench 领域模型的对照（to-spec/to-tickets/triage）

背景：本项目 `.workbench/` 本身即需求/issue 系统——`requirements.json`（req 条目：id/title/description（背景/目标/验收点）/category/reqType/scope/versionId/priority/status/owner/history 审计）、`req-XXX/flow.json`（流程状态机：nodes 带 order/ownerRole/gate/reviewers/branch/parallelGroup/state/rolledBackAt）、`artifacts.json`（产物挂节点）、`comments.json`（评审评论，含 reviewerRole/state/thread）、`inbox.json`（收件箱消息 review/gate/mention）、`versions.json` + workbench.db(SQLite)。三技能与之的同构与可借鉴点：

**to-spec ↔ 需求条目**
- 同构：spec 的 Problem/Solution 对应 req description 的主体；Testing Decisions 对应验收点；Implementation Decisions 对应 flow 中 RD 技术设计/FE 技术设计节点应沉淀的内容。
- 可借鉴：① User Stories「极长编号列表、As actor I want… so that…」——现有 req 以背景/目标/验收点（验收人视角）书写，缺用户故事视角，可补进推演稿或 req 模板；② 显式 **Out of Scope** 节——用户偏好「按既定 scope 裁剪规则推演而非简单标注」，to-spec 的 Out of Scope 是成型表述，可直接并入需求模板；③ 「seams 先于 spec 定、尽量高且少」思想对应本项目「先定产物/流程节点边界再写需求」的推演次序。
- 差异：to-spec 落 `.plan/<slug>/spec.md`；本项目不应新开目录，落点应为既有的 requirements.json 条目增强与 docs-tmp 推演稿（版本化/审计走现有 history 机制）。

**to-tickets ↔ flow.json 编排**
- 同构：ticket=「窄但完整切穿各层、完成可独立演示验证、适配单一 fresh context」的垂直切片 ↔ flow 节点=「一个角色产出一种产物、经 gate 验证」；Blocked by 声明 ↔ flow 依赖与并行组（req-019 的 parallelGroup dev-1：两分支全过才放行 QA 验收，正是 blocking edges 的实例）；wide refactor 的 expand–contract（迁批次按 blast radius）↔ 本项目「并行组 + 部分回滚（reject 指定 targetBranch 仅回滚该分支）」编排的语义来源；「work the frontier（blockers 全完成的先拿）」 ↔ 门禁放行序/当前节点推进。
- 可借鉴：① 「切片完成即可独立演示/验证」作为节点拆分判据（推演节点粒度时问：该节点产物能独立验收吗）；② blocking 先于执行声明的纪律（推演时应显式声明节点间依赖，flow.json 已有 order，但可加依赖理由记录）；③ quiz 用户的三个问题（粒度/依赖正确性/合并拆分）可直接用于需求拆解评审。
- 差异：to-tickets 面向代码实现（一票一 fresh session 开发）；本项目 flow.json 已是「编排机器」，借鉴的是其拆分与依赖思想，而非照搬 tickets.md 文件格式。

**triage ↔ inbox.json + req 状态机**
- 同构：triage 处理「流入 issue」 ↔ inbox.json 的 inbox 消息（in-*：review/gate/mention）即流入通道；五状态角色 ↔ req 的 status（待启动/进行中/已完成）+ priority + owner（简化版状态机）；verified agent brief ↔ inbox 中「开发说明已登记」+ Code Reviewer 评论。
- 可借鉴：① 五状态角色（needs-triage/needs-info/ready-for-agent/ready-for-human/wontfix）比现有 status 更细的「等待信息往返」语义——needs-info→needs-triage 轮回对应 inbox 待复核/mention 闭环，可补「澄清中」状态；② verify claim 纪律（先复现/确证再推进，compare req-016「先压测复现再修」的推演模式）；③ redundancy 检查（按领域概念而非字面搜是否已实现/已推演——对应推演期查 docs-tmp 避免重复推演某能力）；④ `.out-of-scope/` 被拒知识库 ↔ 用户 scope 裁剪惯例的持久落点（「按既定 scope 裁剪规则推演」的规则库化）；⑤ AI 评论免责声明 ↔ 本项目评论模型（comments.json 的 author/reviewerRole）可考虑加「AI 生成」标记。
- 差异：triage 明确只说「你**没**创建的 issue」；本项目需求多为自建/半自建，触发面需定制（主用于外来请求/外部 PR 通道，对应 inbox 流）。

**总体**：三技能与本项目 .workbench 不是「要不要引入」关系，而是「方法论同构、可双向印证」：.workbench 是这套思想的本地实现（flow 编排已覆盖 blocking/frontier/回滚），三技能的增量在 User Stories 视角、Out of Scope 显式化、五状态角色与验证/查重纪律、以及「切片可独立验证」的粒度判据——均以增强既有 .workbench 模型与推演稿的方式落地，不引入 `.plan/` 等新机制。