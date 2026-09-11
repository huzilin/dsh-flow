# 业内如何保证「需求 → 业务文档 → 产品规格 → 技术设计」文档链的完备性与对齐

- 调研日期：2026-09-11
- 适用场景：wayfinder（调研定题）→ to-brd（BRD）→ to-prd（PRD）→ to-rd-techdesign（技术设计）这条由 agent skill 组成的文档生产链，痛点是链路靠人肉推进、容易漏项、对齐靠感觉，需要 (a) agent 结构化追问用户的业内做法，(b) agent 之间按通用协议互相对齐的业内做法。
- 来源标注约定：各结论后标注【一手已核】= 本次调研直接抓取了该 URL 的原文并摘录；【DOI/Crossref 已核】= 通过 Crossref API 验证了论文书目与 DOI 存在（正文内容未读）；【未直接验证】= URL 给出但本次环境无法访问（付费墙/反爬/DNS 污染），内容依据多方一致转述，未读原文，引用时请知悉。

---

## 1. 问题定义：对齐链的四种失灵模式

把「需求 → 业务文档 → 产品规格 → 技术设计」链上的问题归类，业内所有方法其实都在修以下四类失灵：

| 失灵模式 | 表现 | 对应的业内解法 |
|---|---|---|
| **漏问**（该问的没问） | 需求方默认了很多前提，文档里没写，实现时才暴露 | 结构化追问：Spec Kit /clarify、Kiro Analyze Requirements、Claude Code "Let Claude interview you"、Amazon PR/FAQ |
| **漏写**（问了没答、答了没记） | 讨论过但没落盘，或文档缺边界/异常/非功能项 | 完备性检查单：Spec Kit /checklist、ISO 29148 特征清单、Wiegers 需求评审、Edge Cases 占位模板 |
| **跨文档漂移**（上下文档不一致） | 上游改了下游没跟上；同一概念两处叫法不同；需求无下游任务覆盖 | 双向追溯矩阵 RTM、需求编号体系（FR-001）、Spec Kit /analyze 的一致性扫描、Kiro Refine 级联再生、术语表纪律 |
| **拍板无痕**（谁定的、为什么定、何时定的查不到） | 决策只存在于聊天记录或当事人脑中 | ADR（架构决策记录）、Spec Kit 的 `## Clarifications` 会话记录、Rust RFC/PEP 的公开裁决记录、设计评审文化 |

此外还有一条元规则贯穿：**对齐不靠"感觉"，靠结构**——固定句式（EARS、Given/When/Then）、固定编号（FR-001/SC-001）、固定门槛（Definition of Ready）、固定产物（模板 + 检查单）。结构化之后，机器和人才都能检查"完不完备、对没对齐"。

---

## 2. 方案地图

按"解决哪一环"把调研到的方案放进四个格子：

| 环节 | 方法/工具 | 类型 | 对多 agent 自动化的适配度 |
|---|---|---|---|
| 追问补全 | GitHub Spec Kit `/clarify` | AI 工作流命令（2025-） | 原生为 agent 设计，可直接抄协议 |
| 追问补全 | Amazon Kiro Analyze Requirements | 商业 AI IDE 功能（2025-） | 内置 agent 评审 + 澄清问题流，可参照其协议 |
| 追问补全 | Claude Code "Let Claude interview you" | 官方最佳实践（2025-） | 原生为 agent 设计（AskUserQuestion 工具驱动） |
| 追问补全 | Amazon Working Backwards（PR/FAQ） | 人肉方法论（2000s-） | 低；可作 BRD 前置文档产物参照 |
| 需求句式/验收 | EARS（Mavin 2009；Kiro 采用） | 语法标准 | 高——句式即模板，机器可生成可校验 |
| 需求句式/验收 | Given/When/Then（BDD/Gherkin） | 语法标准 | 高——已是可执行测试格式 |
| 需求句式/验收 | INVEST / Definition of Ready | 人肉清单 | 中——可作为 agent 产 story 后的自检门禁 |
| 跨文档追溯 | RTM 需求追溯矩阵 + 需求编号 | 经典 RE 方法（1994-） | 高——矩阵天然可由 agent 生成与检查 |
| 跨文档追溯 | Spec Kit `/analyze` 跨产物一致性 | AI 工作流命令（2025-） | 原生为 agent 设计，最接近"自动对齐协议" |
| 跨文档追溯 | Kiro Refine（上游改→下游再生） | 商业功能 | 高——级联更新协议可抄 |
| 跨文档追溯 | ISO/IEC/IEEE 29148 需求质量特征 | 国际标准 | 中——特征清单可转成 agent 检查项 |
| 完备性检查 | Spec Kit `/checklist`（"英语的单元测试"） | AI 工作流命令 | 原生为 agent 设计 |
| 完备性检查 | Wiegers 需求评审/检查单 | 经典 RE 方法 | 中——检查项可转 agent checklist |
| 完备性检查 | Three Amigos / Specification by Example | 人肉协作 | 中——可转"三个不同立场的 agent 评审" |
| 完备性检查 | Event Storming / User Story Mapping | 人肉工作坊 | 低-中——产物（事件清单/故事地图）可结构化 |
| 角色评审 | MetaGPT SOP 流水线 | 多 agent 框架/论文（ICLR 2024） | 原生多 agent，直接对应 |
| 角色评审 | ChatDev 双 agent 对话链 | 多 agent 框架/论文（ACL 2024） | 原生多 agent；注意已转型 |
| 角色评审 | CrewAI 层级流程（manager 评审） | 多 agent 框架 | 原生多 agent |
| 角色评审 | Anthropic evaluator-optimizer | 官方工程指南（2024-） | 原生多 agent 协议 |
| 角色评审 | Agent-as-a-Judge | 学术框架（2024-） | 高——评审 agent 的系统化版本 |
| 拍板留痕 | ADR | 惯例（2011-） | 高——纯文本格式，agent 可写 |
| 拍板留痕 | Rust RFC / Python PEP / IETF | 社区流程 | 高——状态机 + 模板 + 公开裁决可抄 |
| 拍板留痕 | Google design docs | 公司惯例 | 中——章节模板 + 评审门槛可抄 |

---

## 3. 逐方案详解

### 3.1 GitHub Spec Kit（spec-driven development 的开源事实标准）【一手已核】

- 来源：
  - 仓库：https://github.com/github/spec-kit
  - 官方发布公告：https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/
- **是什么**：GitHub 2025 年 9 月发布的开源工具包，给任意 AI 编码 agent（Copilot / Claude Code / Gemini CLI 等）提供一套 spec-driven 流程命令。命令序列：`/speckit.constitution`（项目宪法，一次性）→ `/speckit.specify`（写 spec，"只管 what 和 why，不管技术栈"）→ `/speckit.clarify`（消歧，官方建议放在 plan 之前）→ `/speckit.plan` → `/speckit.tasks` → `/speckit.analyze`（跨产物一致性检查，tasks 之后 implement 之前）→ `/speckit.implement`。2026 年形态还增加了 `/speckit.converge`（把代码库与 spec/plan/tasks 对账，"Repeat until Converged"）和 `/speckit.taskstoissues`（任务转 GitHub issues）。
- **解决哪个环节**：漏问（clarify）、漏写（checklist）、跨文档漂移（analyze）、拍板留痕（constitution + Clarifications 节）——四环全覆盖，是目前与用户四段流程最同构的工具。
- **具体机制**：

  **(a) /clarify 的结构化追问机制**（摘自仓库内命令模板 `templates/commands/clarify.md`【一手已核】）：
  1. 先做**歧义扫描**：按 10 类覆盖面建立内部地图，每类标 Clear / Partial / Missing——①功能范围与行为（目标/成功标准/范围外/persona）②领域与数据模型（实体、唯一性、生命周期、规模）③交互与 UX 流（旅程、错误态/空态/加载态、无障碍/本地化）④非功能质量属性（性能、可扩展、可靠性、可观测、安全隐私、合规）⑤集成与外部依赖（外部 API 及其失败模式、导入导出格式、协议版本）⑥边界与失败处理（负路径、限流、冲突消解）⑦约束与取舍（技术约束、被否方案）⑧术语与一致性（规范术语表、废弃同义词）⑨完成信号（可测验收标准、可量化 DoD）⑩杂项/占位符（TODO、未量化的形容词如"robust"）。
  2. **选问题**：只提"会实质影响架构、数据建模、任务分解、测试设计、UX 行为、运维就绪或合规验证"的问题；按 **Impact × Uncertainty** 启发式排序；**整个会话最多 5 问**，一次只问一个。
  3. **问题格式**：必须出成 2–5 个选项的选择题（带 `**Recommended:** 选项 X - 理由` 的 A–E 表格，用户可回字母/回"yes"接受推荐/自填），或 ≤5 词的简答题（带 `**Suggested:**` 建议答案）。每问附一句"Why it matters"（不回答会砸到什么）。
  4. **逐答回写落盘**（不是最后批量写）：每接受一个答案，先在 spec 的 `## Clarifications` 节记一行 `- Q: ... → A: ...`（带会话日期），再把答案**应用到最合适的正文节**——功能歧义改 Functional Requirements、数据形状改 Data Model、非功能改 Success Criteria（模糊形容词强制转成可量化指标）、边界补 Edge Cases、术语统一（可标注 `(formerly referred to as "X")`）；被推翻的原文**替换**而非追加；每次写入后做校验（无残留占位、无矛盾），并联动更新 checklist 勾选框、报告前后通过数。

  **(b) spec 模板**（`templates/spec-template.md`【一手已核】）：用户故事按优先级 P1/P2/P3 排序且"必须可独立测试（单独交付一个仍是有价值的 MVP）"；验收场景用 Given/When/Then；功能需求带编号 `FR-001: System MUST ...`，不清楚处行内打 `[NEEDS CLARIFICATION: ...]` 标记；成功准则 `SC-001...` 必须"技术中立且可度量"；另有 Edge Cases 与 Assumptions 占位节（带"ACTION REQUIRED"提示）。

  **(c) /analyze 跨产物一致性机制**（`templates/commands/analyze.md`【一手已核】）：严格只读，跨 `spec.md`/`plan.md`/`tasks.md` 建立四类语义模型（需求清单 FR/SC、用户故事与验收清单、任务↔需求覆盖映射——按关键词或显式引用推断、宪法规则集），然后跑六遍检测：A 重复需求、B 歧义（"fast/scalable/secure/robust"等无度量形容词、TODO/TKTK 占位）、C 欠规约（需求缺对象/结果、故事缺验收、任务引用了 spec/plan 里不存在的文件）、D 宪法对齐（与 MUST 级原则冲突**自动判 CRITICAL**，"只准改产物，不准稀释原则"）、E 覆盖缺口（**零任务覆盖的需求**、没有对应需求的任务、未覆盖的可交付成功准则）、F 不一致（**术语漂移**——同一概念跨文件不同叫法、实体漂移、需求冲突如同时出现 Next.js 与 Vue）。产出报告：findings 表（ID/类别/严重度/位置/建议）、覆盖率表（每个需求"有无任务？任务 ID？"）、指标（覆盖率 %、歧义数、重复数）、CRITICAL 问题阻塞 implement；修复必须经用户批准，绝不自动改。

  **(d) /checklist 的"英语单元测试"机制**（`templates/commands/checklist.md`【一手已核】）：原文"If your spec is code written in English, the checklist is its unit test suite"——检查的是**需求文本质量**而非实现。检查维度：需求完备性、清晰度、一致性、验收标准质量、场景覆盖（主路径/备选/异常/恢复/非功能）、边界覆盖、非功能需求、依赖与假设、歧义与冲突。硬规则：**≥80% 检查项必须带追溯引用**（`[Spec §X.Y]` 或 `[Gap]`/`[Ambiguity]`/`[Conflict]`/`[Assumption]` 标记）；文件放 `checklists/<领域>.md`，条目 `- [ ] CHK###` 全局递增编号、只追加不删除；**勾选权属于评审人**，implement 命令只读勾选状态不许改。
- **对多 agent 的适配度**：最高。它本质就是把"追问、写 spec、评审、对齐"全部命令化、模板化、机器可检查化。追问协议（≤5 问、选择题+推荐、逐答回写、覆盖分类扫描）和评审协议（六遍扫描、严重度分级、覆盖率指标）都可以直接移植成用户 to-brd/to-prd skill 内部的 prompt 协议。

### 3.2 Amazon Kiro（requirements.md / design.md / tasks.md 三件套）【一手已核】

- 来源：
  - Spec 总文档：https://kiro.dev/docs/specs/
  - Feature Specs（EARS 说明）：https://kiro.dev/docs/specs/feature-specs/
  - Analyze Requirements：https://kiro.dev/docs/specs/analyze-requirements/
  - Requirements-First 工作流：https://kiro.dev/docs/specs/feature-specs/requirements-first/
- **是什么**：AWS 的 AI IDE（2025-07 发布；2026 年文档显示已演进到带 CLI/Web/Mobile/Crew 的 3.x 形态），核心工作法是每个 feature spec 固定产出三个文件：requirements.md（用户故事 + 验收标准）、design.md（"系统架构与组件设计、时序图与数据流、错误处理与测试策略"）、tasks.md（"离散、可跟踪的任务，含任务依赖；并行执行时按依赖图分波，波间串行、波内并发"）。
- **解决哪个环节**：漏问（Analyze Requirements）、跨文档漂移（Refine 级联再生）、漏写（EARS 句式强制想全条件）。
- **具体机制**：
  1. **EARS 句式**：官方原文"The requirements.md file uses EARS (Easy Approach to Requirements Syntax) notation to provide structured, testable requirements"，文档中展示的句式为 `WHEN [条件/事件]` + `THE SYSTEM SHALL [预期行为]`（例："WHEN a user submits a form with invalid data / THE SYSTEM SHALL display validation errors next to the relevant fields"）。Kiro 列了四个好处：清晰（无歧义）、可测（每条需求可直接翻译成测试用例）、可追溯（单条需求可贯穿实现跟踪）、完备（句式逼你想全所有条件分支）。
     - EARS 标准本身：Mavin、Wilkinson、Harwood、Novak，《Easy Approach to Requirements Syntax (EARS)》，2009 第 17 届 IEEE International Requirements Engineering Conference，DOI 10.1109/RE.2009.9【DOI/Crossref 已核】；后续扩展《Big Ears (The Return of "Easy Approach to Requirements Engineering")》，RE 2010，DOI 10.1109/RE.2010.39【DOI/Crossref 已核】。（注意：原论文还包括 Ubiquitous 句"The system shall..."、状态驱动 WHILE、非期望 IF 等完整家族；Kiro 文档只展示了 WHEN 形态，其余形态依据论文常识转述【未直接验证原文】。）
  2. **Analyze Requirements（agent 当评审人）**：从 requirements 进入 design 之前，可让 Kiro 对**整个需求集合**做跨需求推理（不是逐条孤立检查），官方列出它抓五类问题：逻辑不一致（"两条需求单独看都成立、合在一起不可能"）、歧义（""large files"或"fast response times"这类会导致实现分叉的语言）、冲突约束（"功能与非功能需求不可能同时满足"）、未言明假设（引用了未定义的概念或行为）、缺失边界（"happy path 之外的失败模式、边界条件、并发访问场景"）。发现会以**澄清问题流**的形式逐条进 chat，每问都带"涉及哪些需求条目 + 通俗问题解释 + 可选的修复建议"，你也可以自填答案或**声明歧义是有意为之而弃权**。
  3. **Refine（变更级联）**：需求变更后编辑 requirements.md，再到 design.md 点 Refine，Kiro 会**重新生成下游产物**（tasks 同理）。Requirements-First 与 Design-First 两种工作流可选，Quick Spec 模式可跳过审批门一次生成三件套。
- **对多 agent 的适配度**：高。三件套 = 文档链的固定产物接口；EARS = 机器可校验的需求语法；Analyze Requirements = "评审 agent 对 spec 提质询"的已工业化实现（问题流 + 逐条挂需求 + 可选修复 + 允许弃权）；Refine = 上游变更触发下游再生的对齐协议。

### 3.3 Anthropic 工程实践（evaluator-optimizer / spec interview / 对抗评审）【一手已核】

- 来源：
  - 《Building effective agents》：https://www.anthropic.com/engineering/building-effective-agents
  - Claude Code 最佳实践（原 anthropic.com/engineering/claude-code-best-practices 已 308 重定向至此）：https://code.claude.com/docs/en/best-practices
- **是什么/解决哪个环节**：Anthropic 官方把"agent 互评""agent 追问用户"写成了工程指南，是 2025-2026 年最有代表性的工业实践：
  1. **evaluator-optimizer 模式**（Building effective agents）：一个 LLM 生成、另一个 LLM 评审并反馈、循环往复；适用条件是"有明确评审标准、且迭代精化有可度量价值"。同文还有 orchestrator-workers（中央 LLM 动态拆任务、派工、汇总）。workflows 与 agents 的分界：workflows 是"预定义代码路径编排"，agents 是"模型自己动态决定流程"；建议默认从最简单方案开始。
  2. **"Let Claude interview you"（结构化追问）**：官方建议大特性先让 agent **采访用户**——用最小 prompt（"I want to build [描述]. Interview me in detail using the AskUserQuestion tool. Ask about technical implementation, UI/UX, edge cases, concerns, and tradeoffs. Don't ask obvious questions, dig into the hard parts..."），问完写一份完整 spec 到 `SPEC.md`，然后**开全新会话**只拿 spec 去实现。官方还给了"好 spec 的自包含标准"：点名涉及的文件与接口、写明范围外（out of scope）、以一个端到端验证步骤收尾。
  3. **"Add an adversarial review step"（对抗评审）**：任务当"完成"处理之前，**派一个全新上下文的 subagent 只看 diff 和验收标准来评审**："A reviewer running in a fresh subagent context sees only the diff and the criteria you give it, not the reasoning that produced the change, so it evaluates the result on its own terms."；评审提示词模板点名"检查每条需求是否实现、列出的边界是否有测试、是否有任务范围外的改动，报缺口不报风格偏好"。内置 `/code-review` skill 即此模式。
- **对多 agent 的适配度**：这三个协议（生成-评审循环、采访式追问、fresh-context 对抗评审）都是可直接抄进 agent 协作链的——尤其是"评审 agent 必须无生产过程的上下文污染"这一条，是防止"自己评自己越评越对"的关键设计。

### 3.4 其他活跃 spec-driven 工具动态

- **Tessl**【一手已核】：https://tessl.io/ ——2025 年上半年以"Spec-Driven Development 平台"姿态出现（spec 注册表、spec 作为真相源），但 2026-09 官网已重新定位为"Agent Enablement Platform"（"Skills are the new code"），首页不再提 spec-driven。给用户工作流的启示：**spec-driven 的理念已被吸收进 agent skills 生态，而独立的"spec 平台"品类并不稳固**。
- **Jama Software**：需求管理老牌厂商，其官网已上线"Spec-Driven Development"栏目（见 3.10 Wiegers 文章页导航），传统 RM 工具与 AI spec 流程正在合流【一手已核（导航结构）】。
- OpenAI 方向未找到与"spec 链对齐"直接对应的一手工程指南（其 agent 指南偏构建模式，不偏文档链），如实标注【未查证到】。

### 3.5 RTM 需求追溯矩阵（跨文档漂移的经典解）

- 来源：
  - 追溯性概念的开山论文：Gotel & Finkelstein，《An analysis of the requirements traceability problem》，Proceedings of IEEE International Conference on Requirements Engineering，1994，DOI 10.1109/ICRE.1994.292398【DOI/Crossref 已核】。该文给出的经典定义是"在前后两个方向上描述并跟随一个需求生命周期的能力"（此名句依据业界一致转述【未直接验证原文】）。
  - 厂商侧：Jama Software 的 Requirements Traceability 方案页 https://www.jamasoftware.com/solutions/requirements-traceability/（页面为 JS 渲染，本次未能提取正文【未直接验证】）；IBM DOORS、Visure、Modern Requirements、Perforce Helix RM 均有同类能力（厂商文档页本次多 404/反爬，未逐一验证）。
- **是什么**：一张双向矩阵，把每条需求与它的上游（业务目标/干系人）和下游（设计元素、代码模块、测试用例）逐一编号挂钩。
- **解决哪个环节**：跨文档漂移。**前向追溯**回答"这条需求在设计和测试里落在哪、有没有实现有没有测试"；**后向追溯**回答"这个设计/这段代码/这条测试为哪条需求而存在"（反向暴露镀金与范围蔓延）。
- **具体机制**：典型列 = 需求 ID、需求描述、设计章节、代码模块、测试用例 ID、测试结果、状态。关键前提是全链**编号体系**（BR-001 → FR-001 → TD-x → TC-x），否则无处挂钩。工具上，人肉时代用 Excel，工业级用 DOORS/Jama 等需求管理平台。
- **对多 agent 的适配度**：高。矩阵本来就是结构化数据；Spec Kit /analyze 的"Requirement Key → Has Task? → Task IDs → 覆盖率 %"表就是一个轻量自动 RTM 的活例，agent 完全可以在每次产出下游文档时生成/更新追溯矩阵，并把"零覆盖需求""无主任务"列为 CRITICAL。

### 3.6 ISO/IEC/IEEE 29148（需求质量特征的权威清单）

- 来源：ISO 官方页 https://www.iso.org/standard/72089.html （本次访问 403 反爬【未直接验证】；标准名《Systems and software engineering — Life cycle processes — Requirements engineering》，2018 版）。
- **是什么/解决哪个环节**：漏写与写不清。该标准给"单条需求"和"需求集"各定义了一套质量特征，是几乎所有需求检查单的源头。
- **具体机制**（清单内容依据业界多源一致转述【未直接验证原文】）：单条需求应满足——必要（Necessary）、贴切（Appropriate）、无歧义（Unambiguous）、完整（Complete）、单一（Singular，一条只说一件事）、可行（Feasible）、可验证（Verifiable，存在有限成本的检验手段）、正确（Correct）、合规（Conforming，符合模板与惯例）；需求集整体应满足——完整、一致、可行、可理解。
- **对多 agent 的适配度**：中-高。特征清单可以直接转成 agent 产出 spec 后的逐条自检规则（Spec Kit /checklist 的检查维度就是这套思想的 AI 时代翻版：clarity/consistency/completeness/scenario coverage/edge cases/NFR）。注意标准本体付费。

### 3.7 INVEST / Definition of Ready / Given-When-Then（story 级质量门禁）

- **INVEST**【一手已核】 https://agilealliance.org/glossary/invest/ ：好用户故事的六个判据——Independent（与其他故事无依赖）、Negotiable（不是铁合同）、Valuable、Estimable、Small（能塞进一个迭代）、Testable（原则上可测）；2003 年 Bill Wake 提出，2004 年经 Mike Cohn《User Stories Applied》推广。用途：agent 产完故事后逐字母自检，不过关就重写。
- **Definition of Ready**【一手已核】 https://agilealliance.org/glossary/definition-of-ready/ ：与 DoD 相对的**入口门禁**——故事进入迭代前必须满足团队显式化的就绪判据（"一般基于 INVEST"），价值是"给团队一个正当理由拒收定义不良的特性"；词源可追到 1985 年的 ETVX（Entry-Task-Validation-Exit）框架。多 agent 适配：把 DoR 写成 skill 的硬校验，不达标不进入下一段流程，等于给 BRD→PRD 交接装了闸机。
- **Given/When/Then**【一手已核】 https://cucumber.io/docs/gherkin/reference/ + https://cucumber.io/docs/bdd/ ：Given 描述"系统初始上下文/已知状态"，When 描述"事件或动作"，Then 描述"可观察的预期结果"（用断言比对，而非查内部状态）；Cucumber 把 BDD 实践总结为 Discovery → Formulation → Automation 三步，强调"可执行规格"让共享理解可以被机器确认。多 agent 适配：验收标准全部 GWT 化之后，评审 agent 可机械核对"每条需求至少一个场景、每个场景三段齐全"，Spec Kit spec 模板与 Kiro 验收标准均采用此式或 EARS 式。

### 3.8 协作对齐方法（人肉时代的三件套）

- **Three Amigos**【一手已核】 https://agilealliance.org/glossary/three-amigos/ ：开发前让**业务/开发/测试三种视角**坐在一起审一个工作增量——业务问"What problem are we trying to solve?"，开发问"How might we build a solution?"，测试问"What about this, what could possibly happen?"；目的"定义要做什么，并就'怎样算做对'达成一致"，产出"更清晰的工作描述（常以例子表达）→ 团队共享理解"；同时把讨论规模控制在三人，"a reasonable guard rail"。源自 George Dinwiddie 2009。**agent 化**：三个不同 system prompt 的评审 agent（业务立场/技术立场/破坏性测试立场）分别审同一份 spec，就是把 Three Amigos 变成协议。
- **Event Storming**【一手已核】 https://www.eventstorming.com/ ：Alberto Brandolini 的"协作探索复杂业务域的柔性工作坊"，官网给四种玩法（Improve 梳理业务流 / Envision 验证新商业模式 / Explore 服务与流程设计 / Design 关键软件与 DDD），卖点是"跨学科、跨筒仓的复杂对话"并"围绕下一步行动达成集体理解与共识"。**agent 化**：工作坊本身难自动化，但其**产物**（按时间轴排列的领域事件清单、命令、策略）是结构化的，可作为 BRD 的域模型附件由 agent 生成与维护。
- **User Story Mapping**【一手已核】 https://www.jpattonassociates.com/user-story-mapping/ ：Jeff Patton 方法——横轴铺"用户使用全生命周期/活动"，纵轴在各活动下按优先级排故事；地图让团队"识别扁平 backlog 藏不住的漏洞与遗漏（holes and omissions）"，并按"端到端薄片"切发布（2005 年《HOW YOU SLICE IT》）。**agent 化**：故事地图天然是二维结构化数据，适合 agent 在 PRD 阶段生成、用来做覆盖完整性自查（每个活动列是否为空？每个切片是否端到端？）。
- **Amazon Working Backwards（PR/FAQ）**：写产品前先写**假想的新闻稿 + FAQ**，逼团队先回答"客户为什么在乎"。来源（书作者官方模板与讲解）：https://workingbackwards.com/resources/working-backwards-pr-faq/ 【本次未逐字抓取正文，链接来自官方渠道检索结果，标注【未直接验证】】；官方书摘 https://www.aboutamazon.com/news/workplace/an-insider-look-at-amazons-culture-and-processes 【未直接验证】；AWS 官方博客 https://aws.amazon.com/blogs/smb/working-backwards-to-drive-customer-experience-and-smb-innovation-forward/ 【未直接验证】。**对四段流程的适配**：这就是 to-brd 前置的"定题产物"模板——wayfinder 结束时先产出 PR/FAQ 式一页纸（客户是谁、痛点、为什么现在、上线后世界有何不同），BRD 再扩写，可防止"调研完不知道要建什么"。

### 3.9 设计评审文化（Google design docs / RFC / PEP / IETF / ADR）

- **Google design docs**【一手已核】 https://www.industrialempathy.com/posts/design-docs-at-google/ （作者 Malte Ubl）：
  - 章节惯例：Context and scope（"只写客观背景事实"）、Goals and non-goals（non-goals 是"本可以是目标但刻意排除的"）、The actual design（系统上下文图、API 草图、数据存储方案；"设计文档就是把 trade-off 写下来的地方"）、**Alternatives considered（"最重要的章节之一"——显式写为什么选这个）**、Cross-cutting concerns（安全/隐私/可观测；"Google 项目强制有独立 privacy design doc"）。
  - 评审文化：轻则贴团队邮件列表收评论，重则正式设计评审会（"面对常是资深工程师的听众讲一遍"）；核心价值是"在改动还便宜的时点暴露问题"。生命周期四段：起草速迭 → 评审 → 实现与迭代（"上线前一定要回头更新文档"）→ 维护与学习。
  - 什么时候写：问题或方案复杂到"设计有歧义"才写；"如果文档其实是实现手册（只说怎么做、没有取舍），直接写代码去"。
  - **agent 化**：章节模板直接可抄进 to-rd-techdesign；"Alternatives considered 强制节"专治拍板无痕。
- **Rust RFC**【一手已核】 https://github.com/rust-lang/rfcs （README 原文）：只收"substantial"变更；流程 = idea/pre-RFC（论坛预热，草率提案会被快速拒）→ 按 `0000-template.md` 写正文（**motivation、design impact、drawbacks、alternatives**）→ 开 PR 编号 → 公开讨论（作者用新 commit 修订，禁止 squash，保留讨论痕迹）→ **Final Comment Period**（子团队全员 sign-off 后开至少 5 个工作日的最终异议窗口，处置为 merge/close/postpone；出现重大新论点则取消 FCP 重开讨论）→ 合并成 active 或关闭（close 常带 postponed 标签留档）。决策权在领域子团队："在收益与代价都被充分理解后由子团队做最终决定"。**关键可抄点：公开的、有强制时长与状态机的裁决窗口，和"drawbacks/alternatives 必填"的模板。**
- **Python PEP**【一手已核】 https://peps.python.org/pep-0001/ ：PEP 是"a design document providing information to the Python community, or describing a new feature"；三类（Standards Track / Informational / Process）；状态机 Draft → Active → Accepted → Provisional → Final →（Rejected / Withdrawn / Superseded / Deferred）；裁决权在 Steering Council（可授权 PEP-Delegate）；讨论必须在公开场合（Discourse 等），"实质性讨论不属于私聊"。**关键可抄点：状态机 + 委托裁决人 + 公开主线程（避免对齐发生在看不见的地方）。**
- **IETF RFC**【一手已核】 https://www.ietf.org/standards/rfcs/ ："RFC 是 IETF 的核心产出"；发布后"永不更改"，只被新 RFC obsolete 或 update；IETF 流的 RFC 必须"community rough consensus"；标准成熟度阶梯 Proposed Standard → Internet Standard。**关键可抄点：文档不可变 + 版本替代链（superseded 链）——对文档链历史可追溯的启示。**
- **ADR（架构决策记录）**【一手已核】：
  - 概念页：https://adr.github.io/ ——架构决策是"针对功能或非功能需求的、有理由的设计选择"，ADR"捕捉单个决策及其理由"，项目全部 ADR 构成决策日志。
  - Michael Nygard 2011 原文：https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions ——五段式：Title / Status（proposed→accepted→deprecated/superseded，**被取代的记录保留不删**）/ Context（中立描述 forces at play）/ Decision（"We will ..."主动语态全句）/ Consequences（"正面的、负面的、中性的都要列"）；文件编号单调递增不复用；"整篇一两页"；写"给未来开发者的对话"。动机："项目中最难追踪的是决策背后的动机"；不记就会"盲目接受或盲目推翻（blind acceptance or blind reversal）"。
  - **agent 化**：ADR 是四类失灵里"拍板无痕"的最小可行解——每次结构拍板（如"P3 落 BRD 还是 PRD"）落一条编号记录，下游 agent 才有依据可查。

### 3.10 完备性检查单（ Wiegers 等）

- **Karl Wiegers 需求同行评审**【一手已核】 https://www.jamasoftware.com/blog/peer-reviews-two-eyes-arent-enough-part-1/ （Wiegers 本人 2012 年文章，改写自其书《More about Software Requirements》）：
  - 核心论断："在我看来，软件业今天最强大的质量实践就是对需求文档的 inspection（正式评审）。"同行评审 = "作者以外的人检查工作产品以发现缺陷与改进机会"；inspection 是更正式严格的团队评审，能发现个人看不出的问题；需求期除错省下的是后期缺陷指数级上涨的修复成本。
  - 对人肉评审的批评同样适用于 agent："评审人只扫一眼（甚至不看），漏掉规格里的大缺陷"，而且**无效的评审给人虚假的质量信心**——这条对"agent 互评"尤其重要：评审 agent 必须有明确检查单，否则等于没评。
  - Wiegers 强调要先"教育评审人"（培训怎么看、看什么），对应 agent 版 = 给评审 agent 注入检查单与缺陷类型学。
  - 其经典检查单（需求质量/SRS/同行评审检查单）历史上挂在 processimpact.com，本次访问该站旧路径已 404（站点结构变动），检查单正文以其著作《Software Requirements》(3rd ed., Wiegers & Beatty) 与《Peer Reviews in Software》为准【未查证到可抓取的一手 PDF】。
- **Google/微软公开 design review checklist**：Google 侧以 design docs 惯例文章为主（见 3.9）；微软侧本次**未查证到**公开发布的官方 design review checklist 一手页面——如实标注。可用的替代是 Spec Kit /checklist（3.1(c)），它本质上就是"可机器执行的评审检查单"。

---

## 4. 专节：多 agent 对齐协议对比（MetaGPT / ChatDev / Spec Kit / Kiro / Anthropic / CrewAI）

这是与「agent 之间互相对齐」最直接对应的部分。

### 4.1 MetaGPT【一手已核，摘要 + HTML 全文关键段】

- 来源：论文 https://arxiv.org/abs/2308.00352 （Hong 等，2023 首发，v7 2024-11；公开发表信息为 ICLR 2024——此 venue 依据公开报道，arXiv 页未标注【未直接验证 venue】）；代码 https://github.com/geekan/MetaGPT
- **是什么**：把软件公司的 **SOP（标准作业程序）编码进 prompt 序列**的多 agent 框架："MetaGPT encodes Standardized Operating Procedures (SOPs) into prompt sequences"，用流水线范式给 agent 分配角色（Product Manager / Architect / Project Manager / Engineer / QA Engineer）。
- **对齐机制（原文关键句）**：
  1. **角色产物结构化**："Product Managers analyze competition and user needs to create Product Requirements Documents (PRDs) using a standardized structure"；"the Architect agent generates two outputs: the system interface design and a sequence flow diagram. These ... serve as important deliverables for Engineers."——每个角色有**固定的输出 schema**（PRD、接口设计、时序图、代码、测试用例），下游只消费上游的结构化产物。
  2. **文档优于对话**："Unlike ChatDev, agents in MetaGPT communicate through documents and diagrams (structured outputs) rather than dialogue. These documents contain all necessary information..."——这是本报告最值得抄的单条结论：**把对齐载体从聊天记录换成有固定结构的文档，天然防漂移、防漏**。
  3. **结构化通信接口 + 发布-订阅消息池**：为每个角色建 schema 与格式，订阅机制让 agent 只拿"方向性信息（别人的产出）+ 公共信息（环境）"，避免全文聊天记录灌满上下文。
  4. **中间产物人工式核验**：SOP 让"有领域专长的 agent 验证中间结果以减少错误"——即流水线上每一站都有质量关卡；另有 executable feedback（运行时报错回灌自纠）。
- **动机**：朴素地把 LLM 串起来会"cascading hallucinations（级联幻觉）导致逻辑不一致"——这正是用户痛点"链靠人肉推进容易漏"的学术表述。

### 4.2 ChatDev【一手已核，摘要 + 官方 README】

- 来源：论文 https://arxiv.org/abs/2307.07924 （ACL 2024）；仓库 https://github.com/OpenBMB/ChatDev
- **是什么/现状**：1.0 是"虚拟软件公司"——agent 扮演 CEO/CTO/Programmer 等角色，"参与专门的职能研讨（seminars），自动化软件开发生命周期——designing, coding, testing, documenting"；论文核心是 **chat chain**（把流水线拆成一对对 instructor-assistant 的双 agent 任务对话，"guides agents in what to communicate (via chat chain) and how to communicate (via communicative dehallucination)"——dehallucination 即在对话里强制收敛出明确的、可执行的决定而不是各说各话）与 Human-Agent-Interaction 模式（**人可以顶替 reviewer 角色给 programmer 提意见**）。**2026-01 已发布 2.0（DevAll），转型为通用零代码多 agent 编排平台，1.0 移入 legacy 分支维护**——引用时注意时效。
- **对齐机制**：角色配对 + 阶段化对话链 + 对话内收敛协议；评审角色（reviewer/tester）内建在阶段里（coding 阶段后 programmer 与 reviewer 就代码对话多轮）。
- **与 MetaGPT 的分野**（MetaGPT 原文点评）：ChatDev 靠自由语言对话对齐，MetaGPT 靠结构化文档对齐。学界结论：结构化产物比自由对话更抗幻觉级联。**对用户四段流程的启示：skill 之间的交接物应当是文档文件，而不是把上一段聊天记录整体传给下一段。**

### 4.3 Spec Kit 与 Kiro 的协议本质（对照表）

| 维度 | GitHub Spec Kit | Amazon Kiro |
|---|---|---|
| 链条产物 | constitution / spec / plan / tasks (+checklists) | requirements.md / design.md / tasks.md |
| 追问机制 | /clarify：10 类覆盖扫描，Impact×Uncertainty 选 ≤5 问，选择题+推荐项，逐答回写并记 `## Clarifications` | Analyze Requirements：跨需求深度分析，澄清问题流（涉及需求+解释+可点选修复），允许"歧义是有意的"弃权 |
| 需求语法 | FR-### "System MUST"，行内 [NEEDS CLARIFICATION]，GWT 验收场景 | EARS（WHEN...THE SYSTEM SHALL...） |
| 跨文档对齐 | /analyze 六遍扫描（重复/歧义/欠规约/宪法冲突/**覆盖缺口**/术语漂移），覆盖率 % 指标，CRITICAL 阻塞 | Refine：上游文档改动后一键再生下游产物 |
| 评审机制 | /checklist"英语单元测试"（CHK### 编号、≥80% 带追溯引用、勾选权归评审人） | 内建于 Analyze（评审 agent 提质询） |
| 拍板留痕 | constitution（不可稀释的 MUST 原则）+ Clarifications 会话节 | spec 版本产物本身（Kiro 是商业 IDE，决策留痕弱于 Spec Kit） |
| 与人流程的关系 | 全部命令可手跑，人逐关拍板 | 三件套各有审批门（可跳过：Quick Spec） |

### 4.4 CrewAI / AutoGen / Agent-as-a-Judge

- **CrewAI**【一手已核】 https://docs.crewai.com/concepts/processes ：hierarchical process"模拟公司科层"，必须配 manager（`manager_llm` 或自定义 manager agent），manager"统筹执行，包括规划、委派与校验"，"审查产出、评估任务完成度"；任务不预分配，按能力动态派工。即框架内建了"管理者 agent 评审"闭环。sequential process 里 Task 有 `context` 参数显式声明消费哪些上游产出。
- **AutoGen**【一手已核】 https://arxiv.org/abs/2308.08155 ：微软多 agent 对话框架；conversable agent = "LLM、人类输入、工具"的组合；用自然语言或代码"编程会话模式"。它提供的是对话式协作的底层协议，文档链对齐要自己搭。
- **Agent-as-a-Judge**【一手已核】 https://arxiv.org/abs/2410.10934 （Zhuge 等，2024-10）：提出"用 agentic 系统评审 agentic 系统"，是 LLM-as-a-Judge 的有机扩展——评审 agent 会主动在产物中间过程里找证据、给**过程性反馈**，而非只看最终结果；配套 DevAI 基准（55 个真实 AI 开发任务、**365 条层级化用户需求**——需求本身就是树状 ID 化的）；实测"显著优于 LLM-as-a-Judge，且与人类评审基线一样可靠"。**这条是"评审 agent 可信"的最强实证背书，且它证明评审 agent 必须带检索/探查能力（agentic），单次打分（LLM-as-a-Judge）不够。**
- **2025-2026 "red-team-the-spec" 命名的工业实践**：**未查证到以该名字存在的一手工业实践**。最接近的已核实实践是：Spec Kit /analyze（把宪法冲突自动判 CRITICAL，本质是对 spec 的红队扫描）、Kiro Analyze Requirements、Claude Code "Add an adversarial review step"（官方明用 adversarial 一词）、Anthropic evaluator-optimizer。结论：**该实践已存在且成熟，只是各家没统一命名为"red-team-the-spec"**。

---

## 5. 对用户四段流程（wayfinder → BRD → PRD → techdesign）的适配建议

按"抄哪个协议、装在哪一段"给出建议清单（均标注来源协议）：

1. **每段之间加"追问关"（治漏问）**：to-brd 与 to-prd 产出初稿后，插入 Spec Kit /clarify 式的追问阶段——按 10 类覆盖面扫描（功能范围/数据模型/UX 流/非功能/外部依赖/边界/约束/术语/完成信号/占位符），选 **Impact×Uncertainty** 最高的问题，**一次一问、选择题带推荐项**、整轮 **≤5 问**上限防轰炸；每个答案**当场回写进文档对应章节**并在文档尾部追加 `## Clarifications` 会话记录（Q→A 逐条留痕）。Kiro 的补充规则值得并入：允许用户答"这个歧义是有意的"，此时把"已知歧义+不解决的代价"记入 Assumptions 而不是硬编一个答案。
2. **模板内建机器可扫的完备性钩子（治漏写）**：spec/PRD 模板加三类占位——行内 `[NEEDS CLARIFICATION: ...]` 标记、Edge Cases 与 Assumptions 的"ACTION REQUIRED"占位节、每条需求强制 `FR-###: 系统 MUST ...` 编号句式（BRD 层用目标编号，PRD 层用功能编号，techdesign 层用组件/接口编号）。验收标准二选一强制：GWT 或 EARS。产出后跑 /checklist 式自检（检查需求文本质量，禁止"Verify/Click/render"类实现测试语言混入），条目带 `CHK###` 编号与 ≥80% 追溯引用。
3. **全链 ID + 自动追溯矩阵（治跨文档漂移）**：BR-xxx（业务）→ FR-xxx（产品）→ TD-xxx（设计）→ TC-xxx（测试）全链编号；每段 skill 产出后自动生成/更新轻量 RTM（需求 Key → 下游覆盖物 ID → 有无覆盖），**零覆盖需求与无主任务直接列 CRITICAL**（Spec Kit /analyze 协议）；同时跑**术语漂移检测**（同一概念跨文档不同叫法）——这与你全局 AGENTS.md 的"双源词汇版本一致"纪律同构，可以把词汇表做成链上唯一真相源，每段产出后核对。
4. **上游变更触发下游再生（治漂移的动态面）**：抄 Kiro Refine——文档链上任一上游文件被编辑后，下游 skill 感知并提示"requirements 已变更，是否 Refine 再生 design"，再生完成后自动重跑第 3 条的一致性扫描。
5. **每段产出后加"fresh-context 评审 agent"（治漏评、治自我确认）**：抄 Anthropic adversarial review + evaluator-optimizer——派一个**无本段上下文**的 subagent，只给它（产物 + 评审检查单 + 上游文档），让它报 gaps 不报风格偏好；评审输出按 CRITICAL/HIGH/MEDIUM/LOW 分级并逐条挂文档位置；CRITICAL 未清零不放行下一段。可再进一步做 Three Amigos 的 agent 版：业务立场/技术立场/破坏性测试立场三个评审 agent 并行审同一份产物。
6. **拍板一律落 ADR 或决策节（治拍板无痕）**：结构拍板（裁歧义、选方案）按 Nygard 五段式（Title/Status/Context/Decision/Consequences）追加编号记录，被推翻的旧决策标 Superseded 但**不删**（IETF/RFC 的不可变精神）；Alternatives considered 作为 techdesign 强制节（Google design docs 惯例）。
7. **交接物结构化，不传聊天记录**：MetaGPT 的核心教训——skill 之间只交接固定 schema 的文档与图（wayfinder 交 PR/FAQ 式一页纸给 to-brd；BRD 交带编号目标集给 to-prd；PRD 交带编号需求集给 to-rd-techdesign），每段开头声明消费了哪些上游 ID（CrewAI Task.context 模式）。

---

## 6. 局限与诚实声明

1. **学术与工业的差距**：MetaGPT/ChatDev 的实验都在 toy 任务（小游戏、小工具）上做，"虚拟软件公司"在真实业务复杂度下的表现并无强证据；它们的"角色分工"更多是 prompt 组织技巧，不能直接等同组织级文档治理。
2. **追问上限的取舍**：Spec Kit ≤5 问、Impact×Uncertainty 排序是为防打扰设计的；BRD/PRD 级议题常常一问就值一小时，5 问可能远不够——移植时上限应做成参数。
3. **自动一致性扫描的假阳/假阴**：/analyze 的任务-需求映射靠关键词/引用推断，Spec Kit 自己也承认是启发式；术语漂移检测依赖词汇表质量。覆盖率 100% 不代表语义对齐。
4. **EARS 的适用边界**：适合系统级行为需求；BRD 层的目标/愿景/商业论证本来就不该被句式化，强行套用会丢上下文。分层最重要：BRD 层用叙事+目标编号，PRD 层才上 EARS/GWT。
5. **商业锁定**：Kiro 是 AWS 商业产品（功能随版本演进快）；ISO 29148 标准本体付费；DOORS/Jama 等追溯工具是重型企业软件。开源可抄的是协议不是产品。
6. **Tessl 转型提示**：spec-driven 工具品类本身在洗牌（Tessl 已转向 agent skills 平台），落地方案应建立在协议层而非具体工具生命周期上。
7. **本次未查证到一手来源的项（如实声明）**：
   - ISO/IEC/IEEE 29148 官方页（iso.org 反爬 403，未读到原文；特征清单来自业界多源一致转述）；
   - RTM 的厂商一手定义页（Jama/Perforce/Visure 相关页 404 或 JS 渲染取不到正文；追溯性概念以 Gotel & Finkelstein 1994 论文 DOI 为锚，其著名定义原文未直接核到）；
   - Wiegers 检查单的可下载 PDF（processimpact 旧页 404）；
   - Microsoft 公开的官方 design review checklist；
   - OpenAI 侧与"spec 链对齐"直接对应的一手工程指南；
   - 以"red-team-the-spec"命名的一手工业实践（等价实践存在，名字未见于官方材料）；
   - Wikipedia（本环境 DNS 污染无法访问）。
8. **调研环境限制**：2026-09-11 执行；部分来源（arXiv HTML、raw.githubusercontent、各厂商页）在网络层间歇性超时，已尽多重通道（WebFetch/curl）交叉抓取；所有【一手已核】结论均有原文摘录支撑。

---

## 7. 参考来源（按报告章节顺序）

**Spec-driven 工具**
1. GitHub Spec Kit 仓库 — https://github.com/github/spec-kit 【一手已核】
2. Spec Kit clarify 命令模板 — https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/clarify.md 【一手已核】
3. Spec Kit spec 模板 — https://raw.githubusercontent.com/github/spec-kit/main/templates/spec-template.md 【一手已核】
4. Spec Kit analyze 命令模板 — https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/analyze.md 【一手已核】
5. Spec Kit checklist 命令模板 — https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/checklist.md 【一手已核】
6. GitHub 官方公告 — https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/ 【未直接验证（检索确认存在）】
7. Kiro Specs 总文档 — https://kiro.dev/docs/specs/ 【一手已核】
8. Kiro Feature Specs（EARS） — https://kiro.dev/docs/specs/feature-specs/ 【一手已核】
9. Kiro Analyze Requirements — https://kiro.dev/docs/specs/analyze-requirements/ 【一手已核】
10. Kiro Requirements-First — https://kiro.dev/docs/specs/feature-specs/requirements-first/ 【一手已核】
11. Tessl 官网（已转型 Agent Enablement Platform） — https://tessl.io/ 【一手已核】

**EARS 与需求工程经典**
12. Mavin, Wilkinson, Harwood, Novak. Easy Approach to Requirements Syntax (EARS). 17th IEEE Intl. Requirements Engineering Conference, 2009. DOI: 10.1109/RE.2009.9 【DOI/Crossref 已核】
13. Mavin & Wilkinson. Big Ears (The Return of "Easy Approach to Requirements Engineering"). RE 2010. DOI: 10.1109/RE.2010.39 【DOI/Crossref 已核】
14. Gotel & Finkelstein. An analysis of the requirements traceability problem. IEEE ICRE 1994. DOI: 10.1109/ICRE.1994.292398 【DOI/Crossref 已核】
15. ISO/IEC/IEEE 29148:2018 目录页 — https://www.iso.org/standard/72089.html 【未直接验证（403）】
16. Agile Alliance INVEST 词条 — https://agilealliance.org/glossary/invest/ 【一手已核】
17. Agile Alliance Definition of Ready 词条 — https://agilealliance.org/glossary/definition-of-ready/ 【一手已核】
18. Cucumber BDD（Discovery/Formulation/Automation） — https://cucumber.io/docs/bdd/ 【一手已核】
19. Cucumber Gherkin Reference（Given/When/Then） — https://cucumber.io/docs/gherkin/reference/ 【一手已核】
20. Jama Requirements Traceability 方案页 — https://www.jamasoftware.com/solutions/requirements-traceability/ 【未直接验证（JS 渲染）】

**协作对齐**
21. Agile Alliance Three Amigos 词条 — https://agilealliance.org/glossary/three-amigos/ 【一手已核】
22. EventStorming 官网 — https://www.eventstorming.com/ 【一手已核】
23. Jeff Patton User Story Mapping — https://www.jpattonassociates.com/user-story-mapping/ 【一手已核】
24. Working Backwards PR/FAQ 模板（书作者官方） — https://workingbackwards.com/resources/working-backwards-pr-faq/ 【未直接验证】
25. About Amazon《Working Backwards》书摘 — https://www.aboutamazon.com/news/workplace/an-insider-look-at-amazons-culture-and-processes 【未直接验证】
26. AWS SMB Blog：Working Backwards — https://aws.amazon.com/blogs/smb/working-backwards-to-drive-customer-experience-and-smb-innovation-forward/ 【未直接验证】

**评审文化**
27. Malte Ubl. Design Docs at Google — https://www.industrialempathy.com/posts/design-docs-at-google/ 【一手已核】
28. Rust RFC README — https://github.com/rust-lang/rfcs （原文抓取自 raw README）【一手已核】
29. PEP 1（PEP Purpose and Guidelines） — https://peps.python.org/pep-0001/ 【一手已核】
30. IETF RFCs — https://www.ietf.org/standards/rfcs/ 【一手已核】
31. ADR GitHub 组织页 — https://adr.github.io/ 【一手已核】
32. Michael Nygard. Documenting Architecture Decisions（2011） — https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions 【一手已核】

**多 agent 对齐**
33. MetaGPT 论文 — https://arxiv.org/abs/2308.00352 （摘要【一手已核】；HTML 全文关键段【一手已核】 https://arxiv.org/html/2308.00352 ）；仓库 — https://github.com/geekan/MetaGPT 【一手已核】
34. ChatDev 论文 — https://arxiv.org/abs/2307.07924 【一手已核（摘要）】；仓库（含 2.0 DevAll 转型公告，2026-01） — https://github.com/OpenBMB/ChatDev 【一手已核】
35. CrewAI Processes 文档 — https://docs.crewai.com/concepts/processes 【一手已核】
36. AutoGen 论文 — https://arxiv.org/abs/2308.08155 【一手已核（摘要）】
37. Agent-as-a-Judge 论文 — https://arxiv.org/abs/2410.10934 【一手已核（摘要）】
38. Anthropic. Building effective agents — https://www.anthropic.com/engineering/building-effective-agents 【一手已核】
39. Claude Code 最佳实践（含 Let Claude interview you / adversarial review） — https://code.claude.com/docs/en/best-practices （原 https://www.anthropic.com/engineering/claude-code-best-practices 308 重定向）【一手已核】

**检查单**
40. Karl Wiegers. Peer Reviews: Two Eyes Aren't Enough, Part 1 — https://www.jamasoftware.com/blog/peer-reviews-two-eyes-arent-enough-part-1/ 【一手已核】
41. Wiegers《Software Requirements》(3rd ed., Wiegers & Beatty) /《Peer Reviews in Software》——检查单正文载体；processimpact.com 旧检查单页已 404【未查证到一手 PDF】
