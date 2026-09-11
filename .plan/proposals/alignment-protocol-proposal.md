# 对齐协议方案（wayfinder → BRD → PRD → 技术设计）——审批稿

> 日期：2026-09-11。状态：待拍板。
> 依据材料：`.plan/research/requirement-alignment-protocols.md`（业内对齐协议调研，41 条来源）、`.plan/research/agent-flow-test-coverage.md`（测试体系调研，上游衔接）。
> 性质标注约定：【业内结论】=调研报告有一手来源；【本地事实】=本地文件/记忆可查；【推断】=我的分析判断。

---

## 0. 一页结论

你的四段流程缺的不是文档生成能力（to-brd/to-prd/to-rd-techdesign 都在），而是**段与段之间的协议**。业内 2025-2026 已把这件事协议化（GitHub Spec Kit、Amazon Kiro），核心可抄的是五个机制，全部可以移植到本地已有 skill 族上，**不需要引入外部工具**。推荐落地形态：**独立协议文档作为单一真相源 + 现有 skill 加指针**（符合你「单一真相源：蒸馏/skill 引用指针，不复制」的既有纪律）。

## 1. 背景与问题

### 1.1 现状流程【本地事实】

wayfinder（调研定题）→ to-brd（BRD）→ to-prd（PRD）→ to-rd-techdesign（技术设计）。生成能力齐备，但段间靠人肉衔接。

### 1.2 三个缺口【推断，依据=本地 skill 盘点 + 上轮测试调研】

1. **追问未挂接**：本地 `grilling` skill 已有完整追问协议（设计树+前沿问题轮次、每题附推荐答案、事实自查/决策交用户），`loop-me` 已有完成定义（「实现者 agent 不用问一个问题才算完成」），但 to-brd/to-prd 落笔前没有强制过这道关——漏问的细节直接固化进文档。
2. **链间无追溯**：BRD 条目↔PRD 条目↔设计条目↔测试用例之间没有双向对照。上轮测试调研发现的「文档-实现漂移」盲区（test-design-gate B4），其上游版本就是「文档-文档漂移」——同一个病，病根在这一段。
3. **无角色评审**：文档生成者自审自过（自己写自己查，视角单一）；你说的「agent 之间对齐」缺机制。

### 1.3 与测试体系的衔接【推断】

对齐协议的产出（编号化需求、状态表、行为清单）直接成为 test-design-gate 的 oracle 源；全链编号的末端 TC-xxx 接 to-qa-testcases 的用例编号（组-序号）。这样上轮调研的「迁移覆盖差集」脚本才能知道「哪些需求没有测试覆盖」。

## 2. 业内协议对照（详证见调研报告第 3、4 节）

| 失灵模式 | 业内协议 | 一手来源 | 可抄机制 |
|---|---|---|---|
| 漏问 | Spec Kit `/clarify` | github.com/github/spec-kit | 覆盖面分类扫描→Impact×Uncertainty 排序→限问→当场回写 |
| 漏写 | Spec Kit Clarifications 台账 + Kiro Assumptions | 同上 / kiro.dev | 答案回写对应章节+尾部留痕；「有意歧义」记假设不硬编 |
| 跨文档漂移 | RTM 可追溯矩阵 + Spec Kit `/analyze` | Gotel & Finkelstein 1994（DOI 已验）/ spec-kit | 全链编号→零覆盖判 CRITICAL→术语漂移检测 |
| 无角色评审 | Anthropic 对抗评审 + MetaGPT 文档通信 | code.claude.com / arXiv | fresh-context 评审 agent 报缺口不报风格；agent 间只交接结构化文档 |
| 拍板无痕 | ADR 五段式（Nygard） | adr.github.io | Title/Status/Context/Decision/Consequences；被推翻标 Superseded 不删 |

【业内结论，已核】两条关键实证：MetaGPT 论文原话「agents communicate through documents and diagrams rather than dialogue」（agent 间对齐载体应是结构化文档而非自由对话，防级联幻觉）；Anthropic 官方工程指南把「让 agent 采访你（产出自包含 SPEC.md）」和「加对抗评审步骤（fresh-context subagent 只看产物+验收标准）」写成了最佳实践；Agent-as-a-Judge（arXiv:2410.10934）实证带探查能力的评审 agent 可靠性接近人类评审基线。

【业内结论，已核的局限】MetaGPT/ChatDev 实验都在玩具任务上，角色分工是 prompt 组织技巧不是组织治理；EARS 句式只适合系统级行为需求，BRD 层目标/愿景强行句式化会丢上下文（调研报告局限 4）；Spec Kit ≤5 问/轮是防打扰设计，BRD 级议题一问值一小时，上限应做成参数（调研报告局限 2）。

## 3. 方案设计：五机制（协议内容，待拍板后固化）

**设计原则【推断】**：协议本身写成一份独立文档（单一真相源），to-* skill 只加指针引用——与你全局 AGENTS.md 的指针模式同构。

### 机制 1：追问关（治漏问+漏写）

每段 skill（to-brd/to-prd/to-rd-techdesign）产出初稿后、定稿前，强制过 `grilling`：按覆盖面分类扫描歧义（功能范围/数据模型/交互流/非功能/外部依赖/边界/约束/术语/完成信号/占位符），按影响×不确定性排序，**一轮一问组、每题带推荐答案、轮次上限做成参数**；每个答案**当场回写进文档对应章节**，文档尾部追加 `## Clarifications` 台账（Q→A 逐条）；用户答「这个歧义是有意的」时记入 Assumptions 并写明不解决的代价，不硬编答案。完成判据沿用 loop-me：「实现者 agent 不再需要问任何问题」。

### 机制 2：全链编号 + 轻量追溯矩阵（治跨文档漂移）

- 编号规范：BR-xxx（业务目标）→ FR-xxx（功能需求）→ TD-xxx（设计条目）→ TC-xxx（测试用例，接 to-qa-testcases 现有「组-序号」）。
- 每段文档头带「上游依据」声明（消费了哪些上游 ID，CrewAI Task.context 模式）；每段产出后生成/更新轻量 RTM（需求 Key → 下游覆盖物 ID → 有无覆盖），**零覆盖需求直接标 CRITICAL 不放行**（Spec Kit /analyze 协议）。
- 术语漂移检测：词汇表（CONTEXT.md 纪律，domain-modeling skill 已有）作为链上唯一真相源，每段产出后核对同一概念跨文档叫法一致——与你全局「双源词汇版本一致」纪律同构。

### 机制 3：三席评审会（治漏评，Three Amigos 的 agent 版）

文档定稿前派 **无本段上下文** 的评审 subagent（Anthropic 对抗评审协议：只给它产物+评审检查单+上游文档，报 gaps 不报风格偏好，输出按 CRITICAL/HIGH/MEDIUM/LOW 分级挂文档位置，CRITICAL 未清零不放行下一段）：

- **业务席**：目标是否达成 BRD 承诺、范围是否蔓延；
- **技术席**：可行性、与既有架构决策（ADR/semantica）冲突、接缝完整性（衔接上轮调研的「迁移承载检查」）；
- **QA 席**：可测性、状态机条目是否配了 ISTQB 状态表、验收判据是否可判定（衔接 test-design-gate 断言三维）。

### 机制 4：拍板一律留痕（治无痕）

结构拍板（裁歧义、选方案）按 ADR 五段式追加编号记录；被推翻的旧决策标 Superseded 不删；techdesign 强制「Alternatives considered」节（Google design docs 惯例）；与 semantica 决策账本双写（ADR 面向文档读者，semantica 面向 agent 溯源——引用不复制）。

### 机制 5：交接物结构化（段间协议）

skill 之间只交接固定 schema 的文档与编号集，不传聊天记录（MetaGPT 核心教训）。wayfinder 交一页纸问题定义+证据清单给 to-brd；BRD 交带编号目标集给 to-prd；PRD 交带编号需求集给 to-rd-techdesign。

**分层句式约定**（采纳调研局限 4 的修正）：BRD 层用叙事+目标编号（不强制句式）；PRD 层验收标准二选一强制 GWT 或 EARS；状态机类条目强制配状态表（衔接上轮 ISTQB 调研结论）。

## 4. 落地形态选项（拍板项 1）

| 选项 | 内容 | 影响面 | 风险 |
|---|---|---|---|
| **A 内嵌** | 协议内容直接写进 to-brd/to-prd/to-rd-techdesign 三个 skill | 3 个 skill 大改 | 协议副本 ×3，违反单一真相源纪律，改协议要同步三处 |
| **B 协议文档+指针（推荐）** | 先写《对齐协议》正本一份，4 个 skill 各加「落笔前过协议」指针行；新增三席评审为协议内流程（可先由主 agent 手动执行，不新做 skill） | 1 份新文档 + 3-4 处指针 | 无；协议可先手动跑，工具化后置 |
| **C 协议+自动化** | B 的基础上加 hooks：上游文档变更触发下游 Refine 提示、RTM 自动重扫 | 再加 hook 配置 | 成本高；Spec Kit 自己承认启发式扫描有假阳/假阴，建议协议跑顺后再做 |

**推荐 B，理由【推断】**：①符合你既有记忆纪律（指针模式）；②协议先行的价值不依赖工具化——手动跑第一轮就能验证追问关和三席评审是否过重；③Kiro 的 Refine 机制证明自动化可行，但那是商业产品随版本演进的能力，协议层建稳后 hooks 是增量活（调研报告局限 5、6：spec-driven 工具品类在洗牌，应建立在协议层而非具体工具生命周期上）。

## 5. 待拍板项

### 拍板 1：落地形态选 A / B / C
- 语境：见第 4 节。三选项互斥，C 可作为 B 跑顺后的第二期。
- 推荐：B。

### 拍板 2：五机制采纳范围
- 语境：第 3 节五机制是一个包，也可拆。两个内含分歧点：
  - **三席评审的范围**：PRD 与 techdesign 必过、BRD 轻量过（只业务席）？还是三份全过三席？评审成本每份约 3 个 subagent。推荐：PRD+techdesign 全三席，BRD 只业务席。
  - **追问轮次参数**：Spec Kit 默认 ≤5 问/轮。BRD 级议题建议参数化：BRD 段 ≤3 轮、PRD 段 ≤2 轮、techdesign 段 ≤2 轮（数值可改）。推荐：参数化，起步用此默认值。
- 推荐：五机制整体采纳，两个分歧点按推荐值起步，跑一轮后校准。

### 拍板 3：上游变更 Refine 的时机
- 语境：机制 2 的动态面——上游文档改动后下游是否强制再生检查（Kiro Refine 机制）。选项：先人肉纪律（协议写「上游变更后下一段开头必须核对 RTM」）/ 直接做 hooks 自动提示。
- 推荐：先人肉纪律，hooks 留第二期（与拍板 1 选 C 与否联动）。

### 拍板 4：skill 修订授权
- 语境：拍板 1 选 B 后需要动的文件：新增《对齐协议》正本（放哪？建议与你其他方法论文档同层，如 OpenViking preferences 或 workspace 顶层）+ to-brd/to-prd/to-rd-techdesign 各加指针行 + `grilling` 不动（被引用）。to-brd/to-prd/to-rd-techdesign 不在本会话可见的 skill 目录（在 dsh/wbflow 环境），需要你告知正本路径。
- 推荐：授权后我出协议正本草稿再过一轮你审。

## 6. 性质标注汇总

- 【业内结论】第 2 节对照表全部条目、五机制对应的协议原文——调研报告 41 条来源支撑；
- 【本地事实】1.1 流程现状、1.2 中 grilling/loop-me/domain-modeling 能力描述、test-design-gate B4 盲区；
- 【推断】三个缺口的归因、五机制的移植组合方式、第 4 节推荐及理由、所有参数默认值。
