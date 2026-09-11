# Agent / Agent-Workflow 系统测试用例构建方法调研:流程覆盖、设计断联、从图生成用例

> 调研日期:2026-09-11。
> 调研锚定的真实痛点:Go 后端工作流引擎(40 个流程定义节点、评审 collect 门禁、打回回退链、升级裁决链、会话状态机)与网文创作 agent(大纲→章节生成状态推进)中,大量**设计断联**在开发后期才暴露(例:exec→review 提评迁移没有 HTTP 端点承载、并行分支没有 join 重算语义、动作 ID 带点号导致下游解析失败),测试靠人肉排场景总有遗漏。
> 调研范围限定:**流程/状态机的结构完整性与覆盖完备性**。LLM 输出质量评测只作为外围补充。
> 证据纪律:所有关键结论给一手来源 URL;查证不到的明确标注;推断性内容标注「推断」。

---

## ① 问题定义:把「设计断联」翻译成业内术语

用户所说的「设计断联」,在业内测试理论与形式化方法中对应一组公认术语:

| 用户描述的典型形态 | 业内术语 | 所属层 |
|---|---|---|
| 某状态迁移没有任何 API/处理器承载(如 exec→review 提评无端点) | 迁移缺失(missing transition / 未实现的迁移);从模型检查视角,规格与实现不符(conformance failure,规格-实现不一致) | 设计期验证 + 集成防线 |
| 动作 ID 无法被下游解析(带点号) | 接口契约违约(contract violation);消费者-提供者不匹配(consumer-provider mismatch) | 集成防线(契约测试) |
| 并行分支缺少汇合(join)语义 | 并发交错缺陷:死锁(deadlock)、活性缺失(liveness violation)、丢失线性化(linearizability 缺失) | 设计期验证(模型检查) |
| 下游需要的字段上游没产出 | 契约违约(contract violation)/API drift | 集成防线(契约测试 + schema diff) |
| 不可达状态 / 死迁移 | 不可达状态(unreachable state)、死迁移(dead transition);状态机测试中的「有效迁移覆盖不达标」 | 设计期验证 + 用例派生(覆盖率判据天然暴露) |

对应的四类测试活动(也是本报告的方案地图四层):

1. **设计期验证(模型检查/形式化)**:在写实现之前,对流程/状态机模型做穷举检查,发现死锁、不可达、交错缺陷。业内统称 formal methods / model checking。
2. **用例派生(模型驱动测试,MBT)**:从图/状态机模型自动生成测试路径,以迁移覆盖、状态覆盖、需求覆盖等判据保证「流程覆盖尽可能全」。
3. **运行期评测(LLM agent 评测平台)**:对 agent 轨迹(trace)做记录、回放、判分,覆盖人肉场景排不完的动态行为。
4. **集成防线(契约测试)**:在接口边界自动拦截「下游要的上游没给」「动作 ID 下游解析不了」类断联。

这四层在业内是互补关系,不是备选关系。用户的两类断联痛点分别归属:**结构断联(迁移没人承载、不可达、缺 join)主要靠第 1、2 层;数据断联(字段缺失、ID 解析失败)主要靠第 4 层;动态行为(时序、重试、边界)靠第 3 层兜底。**

---

## ② 方案地图(四层全景)

```
第 1 层 设计期验证            第 2 层 用例派生(MBT)           第 3 层 运行期评测              第 4 层 集成防线
─────────────────────      ─────────────────────          ─────────────────────         ─────────────────────
TLA+ / PlusCal              GraphWalker(图模型)            OpenAI Agents SDK tracing      Pact(消费者驱动契约)
Microsoft P(状态机+交错)    XState graph / Stately Studio   Anthropic 沙箱+guardrails 指南  OpenAPI schema + oasdiff(API drift)
Alloy(结构模型+反例)        Hypothesis RuleBasedStateMachine LangSmith(数据集+LLM 判官+轨迹)
(发现:死锁/不可达/          quickcheck-state-machine        DeepEval(轨迹指标)
 交错缺陷/规格歧义,          fast-check model-based          Langfuse / Braintrust / Phoenix
 全部在写代码前)             (属性化状态机测试,              τ-bench / SWE-bench(环境仿真+规则判分)
                             发现:实现与模型漂移)            VCR 式录制回放 / LangGraph checkpoint
```

---

## ③ 逐方案详解(每案四件事:是什么 / 能发现哪类断联 / 如何派生用例 / 覆盖判据)

### 3.1 第 2 层核心:模型驱动测试(MBT)——与用户「从图推演用例」最直接对照

#### 3.1.1 GraphWalker(开源 MBT 工具,活跃维护中)

- **是什么**:开源 MBT 框架,提供 CLI、REST/WebSocket 服务、图形编辑器 Studio 和 Java Maven 依赖。模型是「顶点(vertex)+边(edge)」构成的图:边代表动作/迁移(API 调用、点击、超时),顶点代表断言(校验系统确实到达了预期状态)。每个模型配「生成器+停止条件」决定路径怎么走、走到哪停。当前版本 4.3.x,正在做 Rust 重写([官网](https://graphwalker.github.io/),[GitHub](https://github.com/GraphWalker/graphwalker-project))。
- **能发现哪类断联**:测试期。如果模型里定义的某个顶点/迁移在真实系统上无法到达或无法执行(对应「迁移没人承载」),覆盖率停不下来,用例执行失败——**覆盖率判据天然把结构断联变成用例失败**。注意:GraphWalker 管的是「图→路径→执行」,模型本身是否完整(漏画的迁移)它不管,发现不了。
- **如何从模型派生用例**:生成器决定遍历算法,停止条件决定覆盖目标,二者可自由组合。官方 wiki 原文确认的生成器:`random`(随机游走)、`weighted_random`(按边权重概率)、`quick_random`(「选未访问边→Dijkstra 最短路→走过去→标记已访问」循环,适合大模型)、`predefined_path`(执行预定义边序列)、`a_star`(最短路到达指定顶点/边)。停止条件:`edge_coverage(n%)`、`vertex_coverage(n%)`、`requirement_coverage(n%)`(按需求标签覆盖)、`dependency_edge_coverage` 等,多个条件可用 `OR/AND` 组合,生成器可菊花链拼接([Generators and stop conditions wiki](https://github.com/GraphWalker/graphwalker-project/wiki))。
- **覆盖判据**:边覆盖(=迁移覆盖)、顶点覆盖(=状态覆盖)、需求覆盖(模型元素挂需求标签,可度量「哪些需求被哪些路径测到」)。这套「生成器×停止条件」组合就是业内「从图直接推演测试路径」最成熟的开源实现。

#### 3.1.2 XState 生态:xstate/graph 与 Stately Studio(状态机原生 MBT)

- **是什么**:XState 是状态图(statechart,UML 状态机分层扩展)的 JS/TS 实现。旧包 `@xstate/test` **已被官方弃用**(stately.ai/docs/testing 原文:"The standalone `@xstate/test` package is deprecated in favor of the integrated testing utilities"),测试工具并入 `xstate` 包的 `xstate/graph` 模块;Stately Studio(可视化状态机编辑器)可在 Pro/Team/Enterprise 付费计划下直接从画布上的状态机生成测试路径([Testing 文档](https://stately.ai/docs/testing))。
- **能发现哪类断联**:建模期/测试期。`getShortestPaths` 返回「每个**可达**状态各一条最短路径」(Dijkstra 算法,官方文档原文:one efficient path per reachable state)——**把模型中定义的状态集合与「可达状态集合」一对比,不可达状态/死迁移直接现形**(后半句为推断,由文档行为推出);模型有迁移、实现没挂处理器,则路径执行时在该步失败。并行分支的层级状态可建模,但 join 语义正确性仍需人工断言。
- **如何从模型派生用例**:`createTestModel` 后,`model.getShortestPaths()`(每个可达状态一条最短路径,最小充分集)、`model.getSimplePaths()`(所有无环简单路径,DFS,官方称可给「complete transition coverage」)、`model.getPaths(pathGenerator)`(自定义生成器),再逐条 `path.test()` 执行并对每步状态断言([xstate/graph 文档](https://stately.ai/docs/xstate-graph))。
- **覆盖判据**:状态覆盖(最短路径集)、迁移覆盖(简单路径集)。未见官方数值化「覆盖率百分比」API(文档未提供,推断需自行统计)。

#### 3.1.3 Hypothesis `RuleBasedStateMachine`(属性化状态机测试,Python)

- **是什么**:Python 属性测试库 Hypothesis 的有状态测试机制:不写死用例,而是定义规则(`@rule`)、前置条件(`@precondition`)、不变量(`@invariant`)、初始化(`@initialize`)和数据束(`Bundle`),由 Hypothesis 自动生成「规则序列」并搜索导致失败的序列,失败时输出**最小可复现的近似 Python 回放程序**([官方文档](https://hypothesis.readthedocs.io/en/latest/stateful.html))。
- **能发现哪类断联**:测试期。经典用法是「真系统 vs 简单模型」双跑对账(model-based 属性):文档示例是数据库实现与内存 dict 模型,用 `values_agree` 规则断言两者一致——**上游该产出的状态/数据没产出,对账规则立刻失败**。它还能发现「某前置条件永远不满足」的规则(即永远执行不到的分支)——Hypothesis 会在规则全被前置条件过滤时报告,这可暴露死分支(此行为为推断,文档确认前置条件不满足的规则会被过滤执行)。
- **如何从模型派生用例**:不派生固定路径,而是「生成动作序列 + 属性断言 + 反例收缩(shrink)」,按 `max_examples`、`stateful_step_count` 控制搜索量。
- **覆盖判据**:没有显式覆盖率指标;判据是「属性在任意长度动作序列下都成立」,属于属性测试范式而非覆盖范式(与 MBT 互补:MBT 管「路径都走到了」,属性测试管「走到的地方状态都对」)。

#### 3.1.4 quickcheck-state-machine(Haskell)与 fast-check model-based testing(TypeScript)

- **quickcheck-state-machine**:Haskell 库,以「模型+前置/后置条件+迁移函数」规格化正确性,顺序模式逐命令「查前置→执行→查后置→推进模型」,失败自动收缩最小反例;**并行模式**把命令序列拆为顺序前缀+并发后缀,记录调用/响应历史后搜索「线性化(linearization)」,**找不到合法线性化即判竞态**——这正是「并行分支缺 join 语义」的形式化测试版。原仓库(advancedtelematic)2021 年归档,现由作者 stevana 轻维护,自称「功能完备稳定」,用户含 IOHK、Wire([GitHub](https://github.com/stevana/quickcheck-state-machine))。
- **fast-check**:TS/JS 属性测试库的模型测试:每个命令实现 `check(model)`(当前状态是否允许)、`run(model, real)`(双跑,真系统与模型分叉即抛错)、`toString()`;`commands` arbitrary 生成可收缩的命令序列,`modelRun`/`asyncModelRun`/`scheduledModelRun` 三种执行器,`scheduledModelRun` 可检出竞态;回放需 `replayPath`。官方告诫模型别写成「系统的复印件」([官方文档](https://fast-check.dev/docs/advanced/model-based-testing/))。
- **能发现哪类断联**:测试期——实现与模型状态漂移、并发交错缺陷。
- **派生方式/判据**:生成命令序列(随机+收缩);判据是「真系统与模型处处一致」「并发历史可线性化」,无路径覆盖率指标。

#### 3.1.5 Conformiq(商业 MBT,活跃)

- **是什么**:商业 MBT 厂商,现产品 ConformIQ Creator(「AI 驱动、脚本化免写、完整覆盖」的测试创建)+ TestRunner(执行),并推出 beta 期「自主测试平台」ConformIQ Infinity(GenAI 理解需求 + Symbolic AI 从「Living Blueprint」确定性生成测试)。官方称从「可视化模型或 Gherkin」自动圈选范围并生成优化用例与执行脚本;**页面未确认支持直接导入 UML/状态图**([官网](https://www.conformiq.com/))。qSampler 在现官网未提及,状态未知。
- **断联/派生/判据**:商业资料称「requirements to execution 自动化」「complete coverage」,但具体覆盖判据与断联检出能力官网未公开细节——**仅二手/厂商营销口径,技术细节未查证到**。

### 3.2 第 1 层核心:设计期验证(模型检查/形式化方法)——「设计断联」在开发前的拦截

#### 3.2.1 TLA+ / PlusCal(AWS 实证)

- **是什么**:Lamport 的形式规约语言,系统建模为状态机+时序逻辑,用 TLC 模型检查器穷举状态空间验证安全/活性性质。
- **能发现哪类断联**:**设计期**。AWS 论文《How Amazon Web Services Uses Formal Methods》(CACM 2015)一手证据(从 PDF 原文提取):
  - "So far we have used TLA+ on 10 large complex real-world systems. In every case TLA+ has added significant value, either finding subtle bugs that we are sure we would not have found by other means"(10 个大型真实系统,每例都有显著收益,发现的隐蔽 bug 用其他手段肯定找不到)。
  - 逐系统战果表(原文):S3 容错底层网络算法「Found 2 bugs」;S3 后台数据重分布「Found 1 bug, and found a bug in the first proposed fix」;DynamoDB 复制与组成员系统「Found 3 bugs, **some requiring traces of 35 steps**」(需要 35 步轨迹才能触发的 bug——人肉排场景根本排不到);EBS 卷管理「Found 3 bugs」等。
  - 结论原文:"They have helped us prevent subtle, serious bugs from reaching production, bugs that we would not have found via any other technique."
  - 论文还记录:非正式走查虽能发现部分 bug,但 TLA+ 揭示了「特定故障与恢复步骤与其他步骤交错」才会触发的数据丢失路径,并明确指出 Alloy 表达力不足(难表达嵌套动态序列)后转投 TLA+([论文 PDF](https://lamport.azurewebsites.net/tla/formal-methods-amazon.pdf))。
- **如何派生用例 / 覆盖判据**:TLA+ 不派生测试用例,而是**穷举可达状态空间**(在有限实例化边界内)检查不变量与活性。模型检查的可达性分析是它的定义性能力:证明「目标状态从初始状态不可达」即直接判死状态/死迁移,这是模型检查的标准能力(定义性,无需单独引证)。TLC 发现的反例轨迹(counterexample trace)本身就是「最难人肉想到的测试用例」——这是形式化方法与测试派生的连接点。
- **对用户痛点的意义**:「exec→review 没有端点承载」这类断联,在 TLA+/PlusCal 层面表现为「模型中存在迁移、但把实现映射进模型(或用 PObserve 类运行时校验)时该迁移无执行者」;更直接的对口工具是 P 语言(见下)。

#### 3.2.2 Microsoft P 语言(状态机并发协议建模,与用户场景最贴合)

- **是什么**:微软出品的「基于状态机的编程语言,用于形式化建模和规约复杂分布式系统」,刻意设计为「编程语言而非数学记号」;P-Checker 系统性穷举「消息与故障的交错(interleavings)」找深层 bug,PEx 做穷举检查,PVerifier 做机器可查证明([官网](https://p-org.github.io/P/))。
- **能发现哪类断联**:**设计期**。官网原文两句直接命中用户痛点:(1) 模型检查「finds corner-case bugs in system design that are **missed by stress and integration testing**」(压力测试和集成测试漏掉的设计期角落 bug);(2) 提供「Reproducible error traces」。违反安全/活性规格(死锁、卡死、状态推进断裂)都会以错误轨迹形式暴露。
- **如何派生用例 / 覆盖判据**:P 内建测试用例构造,test-case 构造可从模型生成测试;**PObserve** 机制把「实现日志 vs P 规约」在测试与生产环境比对,校验实现与已验证设计一致——这正是「规格里有的迁移,实现里必须有承载」的运行时形式。工业案例:AWS 自 S3/EBS 至 DynamoDB/MemoryDB/Aurora/EC2/IoT 多团队使用(2025 年 CACM 文章 + re:Invent 2023 演讲,官网列名);微软与 UC Berkeley 为合作方。项目 2013 年起产学研运行至今。
- **对用户痛点的意义**:用户的 Go 工作流引擎本质是「通信状态机集合」(流程引擎、评审门禁、升级裁决链、会话状态机互发消息),与 P 的目标形态几乎一一对应;「并行分支缺 join」属于交错缺陷,P 的穷举交错检查是业内对口方案。

#### 3.2.3 Alloy(结构模型+反例生成)

- **是什么**:开源「软件建模语言与分析器」,模型查找引擎 Kodkod,Alloy 6 加入可变状态、时序逻辑与新求解器;最新版 Alloy 6.2.0(2025-01-09)([官网](https://alloytools.org/))。
- **能发现哪类断联**:设计期。分析器对模型求反例(counterexample):模型过约束/欠约束、结构不变量被违反、目标状态不可达等都会以可视化反例图呈现。AWS 论文亦记载用 Alloy 在 Chord 成员协议中找到严重 bug 的先例(同上 PDF)。官网本身只笼统提「finding holes in security mechanisms」,细节以论文/教材为准。
- **派生/判据**:无测试用例派生;判据是「有界范围内所有实例是否满足断言」,反例即用例。

### 3.3 第 2 层方法学基线:ISTQB 状态迁移测试(覆盖判据的行业标准定义)

- **是什么**:ISTQB CTFL v4.0.1 教材(2024-09-15 版,78 页)第 4.2.4 节对状态迁移测试给出标准定义(以下均为 PDF 原文,从 [ISTQB 官方下载](https://istqb.org/?sdm_process_download=1&download_id=3345) 提取):
  - 迁移标注语法 `event [guard condition] / action`;
  - **状态表(state table)**:行=状态,列=事件,「**与状态图相反,状态表显式展示无效迁移(空格表示)**」——这是把「漏设计」显式化的经典手法;
  - 三级覆盖判据:
    1. **All states coverage(全状态覆盖)**:覆盖项=状态;
    2. **Valid transitions coverage(有效迁移覆盖,又称 0-switch coverage)**:覆盖项=单个有效迁移;教材原文:"Valid transitions coverage is the most widely used coverage criterion. Achieving full valid transitions coverage guarantees full all states coverage."
    3. **All transitions coverage(全迁移覆盖)**:有效迁移全覆盖 + **逐条尝试执行无效迁移**(每条用例只打一个无效迁移以避免缺陷屏蔽 defect masking);教材原文:"should be a minimum requirement for mission and safety-critical software"。
- **能发现哪类断联**:测试期(设计期残漏)。「无效迁移被错误接受」= 状态机防护缺失;「有效迁移覆盖凑不满」= 存在无法执行的迁移(断联);状态表填空过程本身就是设计评审。
- **如何派生用例**:画状态表 → 逐状态/逐迁移导出事件序列,一条用例串联多个迁移。
- **N-switch / Chow 开关覆盖**:把「迁移对/迁移 n 元组」作为覆盖项(1-switch = 相邻迁移对),源出 Chow, T. S., "Testing Software Design Modeled by Finite-State Machines", IEEE Transactions on Software Engineering, SE-4(3), 1978, DOI: 10.1109/TSE.1978.231865(经典文献,原文在 IEEE Xplore 付费墙内;此 DOI 为标准文献信息,本次未直接阅读原文全文)。ISTQB CTFL v4 教材正文只定义 0-switch;N-switch 见 ISTQB 体系进阶文献(未在本次一手取证范围内,标注为「仅二手/文献常识」)。

### 3.4 第 4 层:契约测试与一致性检查——「下游要的上游没给」的自动化拦截

#### 3.4.1 Pact(消费者驱动契约测试)

- **是什么**:"code-first tool for testing HTTP and message integrations using contract tests";消费者测试运行时生成契约(contract by example,只记录实际用到的交互对),提供者端回放这些交互验证自己真的返回了消费者期望的内容([官方文档](https://docs.pact.io/))。
- **能发现哪类断联**:集成期(上线前)。官方文档原文明确:契约是「static artefact that describes all possible states」(指 OpenAPI)的反面——Pact 用**执行过的用例**强制约束;提供者少给字段、改字段语义,验证直接失败。文档还指出:仅对 OpenAPI 做 schema 校验只能证明 provider 与文档同步,**不能**保证消费者调用正确——正好补 schema 校验的盲区。
- **如何派生用例 / 覆盖判据**:用例=消费者的真实交互意图(每条消费测试即一条契约交互);判据=所有已发布契约全部验证通过(配合 Pact Broker 的 can-i-deploy 门禁;Broker 细节本次未逐页取证,属 Pact 生态公开机制)。
- **对用户痛点的意义**:「动作 ID 带点号下游解析失败」「下游要的字段上游没产出」都是典型消费者-提供者违约,Pact 模式(或 Go 生态内等价的 schema 契约测试)在 CI 里就能拦。

#### 3.4.2 OpenAPI schema diff / API drift 检测(oasdiff)

- **是什么**:Go 写的 Apache-2.0 开源 CLI,「compare and detect breaking changes in OpenAPI specs」,支持 3.0/3.1/3.2;`breaking` 子命令只报破坏性变更,`changelog`/`diff`/`summary` 分级输出;有 GitHub Action、pre-commit、PR 审查托管;号称覆盖「OpenAPI 文档的每一种编辑」的检查映射,可定制规则([GitHub](https://github.com/oasdiff/oasdiff))。
- **能发现哪类断联**:集成期。删字段、加必填、可空性变化等「上游改了下游炸」的 drift,CI 阶段拦截。
- **派生/判据**:不派生用例;判据=「本次 diff 中破坏性变更清单为空(或已评审)」。

### 3.5 第 3 层:LLM agent 评测平台与轨迹测试(外围但必要)

#### 3.5.1 OpenAI Agents SDK tracing

- **是什么**:SDK 内建 tracing,「collecting a comprehensive record of events during an agent run」,默认开启;Trace=一个端到端工作流(workflow_name、trace_id、group_id、metadata),Span=带父子嵌套的计时操作(AgentSpanData、GenerationSpanData 等);默认包住整个 `Runner.run*()`,并给每次 runner 调用、模型轮次、agent 运行、LLM 生成、函数工具调用、guardrail、handoff 建 span([官方文档](https://openai.github.io/openai-agents-python/tracing/))。
- **断联发现**:调试/监控期。handoff、guardrail、工具调用全部落 span,流程断在哪一跳直接看轨迹。可 `add_trace_processor` 双写第三方(Langfuse、Arize Phoenix、MLflow、Datadog、Braintrust、PostHog 等官方列名支持)。
- **派生/判据**:无自动派生;判据=人工/下游工具对轨迹的检查。

#### 3.5.2 Anthropic 官方指南(Building Effective Agents)

- **是什么**:Anthropic 官方工程指南([anthropic.com/research/building-effective-agents](https://www.anthropic.com/research/building-effective-agents))。
- **与测试相关的要点**(原文摘取):agent 开发成败在于「measuring performance and iterating on implementations」;自主 agent 需要「extensive testing in **sandboxed environments**, along with the appropriate guardrails」;工具要跑大量示例输入、观察模型犯的错、迭代工具设计;要设停止条件(如最大迭代数);并行化 workflow 可用来做多维自动评分;evaluator-optimizer 模式把评估嵌进回路;「Code solutions are verifiable through automated tests」——**能用确定性测试验证的,优先用确定性测试**;agent 应每步从环境取 ground truth 评估进度。
- **判据**:无具体覆盖率框架,给的是原则(沙箱+护栏+可验证反馈+透明轨迹)。

#### 3.5.3 LangSmith(LangChain 评测平台)

- **是什么**:数据集(dataset,示例=输入+可选参考输出+元数据,splits 分段,自动版本化)、LLM-as-judge 评估器(参考/无参考两模式、few-shot 变体)、代码评估器(确定性检查,如生成的代码能否编译)、实验(experiments,版本并排对比)、注解队列(人工审全轨迹)([评估概念文档](https://docs.langchain.com/langsmith/evaluation-concepts))。
- **轨迹评测**:run 捕获「All the child runs (tool calls, LLM calls, and so on)」,评估器可评中间步骤;官方建议为 agent 策划「correct tool selection and proper argument formatting or trajectory」的示例集。多轮对话按线程(thread)评「coherence across turns」等(页面描述的是**真实对话的线程级评估**,未覆盖模拟用户压测;LangSmith 旧版有 simulation 功能,本次未在新文档取证到,不下结论)。
- **判据**:数据集通过率 + 判官分数 + 人工标注;无结构覆盖率概念。

#### 3.5.4 DeepEval(轨迹指标)

- **是什么**:开源 LLM 评测框架。README 确认的 agentic 指标:Task Completion(目标是否达成)、Goal Accuracy、Step Efficiency(是否走了多余步骤)、Plan Adherence / Plan Quality、Tool Correctness(「right tools were called with the right arguments」)、Argument Correctness(校验工具调用参数)、MCP Task Completion / MCP Use;官方称支持「Complete agent trajectories across every decision and action」与单步(LLM 调用、工具使用、检索、子 agent 交接)评测;`deepeval test run` 可进 pytest/CI;集成 LangGraph、OpenAI Agents、CrewAI 等([GitHub README](https://github.com/confident-ai/deepeval))。
- **对用户痛点**:Plan Adherence + Tool Correctness 恰好对应「状态推进是否按设计走」「动作 ID/参数是否被下游接受」的运行期检测。

#### 3.5.5 Langfuse / Braintrust / Arize Phoenix(观测+评测,一笔带过)

- Langfuse:开源可自托管;tracing(LLM 与非 LLM 调用、多轮 session、agent 图可视化、OpenTelemetry)、datasets、experiments、LLM-as-a-judge(可对轨迹任意片段 step-wise 打分)、annotation([docs](https://langfuse.com/docs))。Braintrust、Arize Phoenix 本次未单独逐页取证,但被 OpenAI Agents SDK 官方列为 tracing 集成对象(见 3.5.1),定性为同类平台(仅二手)。

#### 3.5.6 确定性回放/录制打桩(VCR 式)与 LangGraph checkpoint

- **VCR.py**:HTTP 交互录制到 cassette(YAML),后续测试拦截同请求回放原响应,「completely deterministic tests」,API 变更则删带重录;pytest 经 pytest-recording 集成([文档](https://vcrpy.readthedocs.io/en/latest/))。LLM 测试中常用于把「真 LLM 响应」固化为可回放桩,让流程层测试确定性化。
- **LangGraph checkpoint/persistence**:checkpointer 把图状态按 thread 持久化为快照,官方列出的用途含「**time travel**、fault tolerance」「resume after an interruption」,可「persist and inspect thread state」([文档](https://docs.langchain.com/oss/python/langgraph/persistence))。对流程测试的意义:从任意 checkpoint 分叉重放,构造「打回回退链/升级裁决」这类难到达的状态而无需跑完前置流程(该句为用途推断,机制为一手确认)。

### 3.6 Agent 系统失败分类学:MAST(流程断联在分类学里的位置)

- **是什么**:《Why Do Multi-Agent LLM Systems Fail?》(arXiv:2503.13657,2025-03,UC Berkeley 等)提出首个多 agent 系统失败分类法 MAST:标注 150 条轨迹,标注者间一致性 kappa=0.88,三类 **14 个失败模式**;配套 MAST-Data(1600+ 条跨 7 个主流 MAS 框架的标注轨迹)与 LLM-as-a-Judge 标注流水线([arXiv 摘要页](https://arxiv.org/abs/2503.13657))。
- **三类 14 模式**(HTML 全文原文提取):
  - **FC1 规格与系统设计失败**(与用户「设计断联」直接同类):FM-1.1 Disobey task specification、FM-1.2 Disobey role specification、FM-1.3 Step repetition、FM-1.4 Loss of conversation history、FM-1.5 Unaware of termination conditions;
  - **FC2 agent 间错位**(协调失败所在类):FM-2.1 Conversation reset、FM-2.2 Fail to ask for clarification、FM-2.3 Task derailment、FM-2.4 Information withholding、FM-2.5 Ignored other agent's input、FM-2.6 Reasoning-action mismatch;
  - **FC3 任务验证与终止**:FM-3.1 Premature termination、FM-3.2 No or incomplete verification、FM-3.3 Incorrect verification。
  - (注:三类样本占比数字在 v1 全文正文的图表中,本次文本提取未捕获到具体百分比,不作引用。)
- **工业应对**:论文用干预实验证明仅调 prompt/对话管理的干预「fail to fully address MAS failures, highlighting the need for **structural MAS redesigns**」——即结构性缺陷要靠结构性(流程/架构)手段解决,与用户「在设计期拦截断联」方向一致。
- **与 MBT 的关系**:MAST 是「运行期失败模式词典」,可反向用作测试断言清单(如:终止条件是否被 agent 识别 = FM-1.5;信息是否丢失 = FM-1.4/2.4),把分类学变成验收 checklist(此用法为推断)。

### 3.7 仿真测试:τ-bench / SWE-bench(环境仿真+规则判分)

#### τ-bench

- **是什么**:Sierra Research 基准,「emulating dynamic conversations between a user (simulated by language models)」与带领域 API 工具+策略规则的 agent([arXiv:2406.12045](https://arxiv.org/abs/2406.12045))。
- **判分机制**(摘要原文):评分为「compares the **database state** at the end of a conversation with the annotated goal state」——不看 LLM 自述,看工具调用把环境数据库推进到了什么状态;同时考「domain-specific policy guidelines」遵守;**pass^k** 指标衡量同一任务多次试验的可靠性(retail 域 gpt-4o pass^8 < 25%)。
- **对用户痛点的意义**:τ-bench 证明「**终态数据库 diff** + 策略规则校验」是流程正确性的可判分判据——用户的 40 节点流程引擎完全可用同思路:每个场景用例断言「流程终态 + 各节点状态 + 门禁记录」而非只断言输出文本。airline 域名在摘要页未提及(在 GitHub 仓库中,未单独取证)。

#### SWE-bench

- **是什么**:来自 12 个 Python 仓库 2294 个真实 GitHub issue 的工程任务基准,报告「% Resolved」;SWE-bench Verified 为与 OpenAI 合作的 500 条人工过滤子集([官网](https://www.swebench.com/))。判分细节(FAIL_TO_PASS/PASS_TO_PASS 回归测试集)在论文/仓库中,官网页未展开——判据定性:**以既有测试套的通过/回归为裁判**,规则判分、环境仿真思路与 τ-bench 同族。

---

## ④ 「架构图/流程图/状态图 → 自动生成测试用例」的业内先例(对照用户 archify 思路)

用户备选思路:用 archify 画架构图 → 从图推演 QA 用例。业内等价物按「图的形态化程度」分三档:

**第一档:图本身就是可执行模型(最成熟,工业级日常使用)**
- **状态机即模型**:XState/Stately(状态图→最短路径/简单路径→可执行测试,Stately Studio 可视化画布直接出测试路径)、GraphWalker(顶点-边图→生成器+停止条件→测试路径+需求覆盖)。
- **并发协议即模型**:Microsoft P(状态机集合+消息交错→穷举检查+错误轨迹;PObserve 反向校验实现日志符合规格)。
- **属性即模型**:Hypothesis / fast-check / quickcheck-state-machine(模型+规则→自动生成并收缩动作序列)。
- 这一档的关键:**图必须是机器可执行的规约(状态机 DSL),不是给人看的图**。GraphWalker 的 JSON/yEd 模型、XState 的 machine 定义、P 的 P# 源码都是这类。

**第二档:商业工具从「需求/可视化模型」自动生成用例(成熟但闭源)**
- ConformIQ Creator/Infinity:从「可视化模型或 Gherkin」自动生成优化用例与执行脚本,厂商称 92% 自动化(银行案例)。细节闭源,判据未公开。

**第三档:从架构图(给人看的)直接推演用例(研究活跃,无 mainstream 工业工具)**
- 以 UML 状态图/顺序图为输入的 MBT 是学术经典方向(Chow 1978 开端的开关覆盖、Lee & Yannakakis 1996 综述等),但「从非形式化架构图(如 draw.io/HTML 图)自动推演完备测试集」在工业界没有查证到 mainstream 落地工具——**未查证到开箱即用的「任意架构图→完备用例」工具;所有可用的业界实现都要求先把图升级为形式化模型**。
- 对用户 archify 思路的判断(推断,附依据):archify 产出是给人看的 HTML/SVG 架构图,**缺机器可执行语义(事件、guard、动作、状态不变量)**,直接从它自动推演用例没有业内等价先例;业内等价路径是「archify 图 → 人工/半自动转成状态机 DSL(XState / GraphWalker JSON / P)→ 工具自动推演」。图的**价值不在直接喂给工具,而在建模时把「状态表填空」问题提前暴露**(ISTQB:状态表显式暴露无效迁移;画图过程即设计评审)。换言之:画图有用,但必须落成形式化模型才有自动推演;这正是本次调研最强的一条工程结论。

---

## ⑤ 局限与适用边界

1. **MBT 不发现「模型本身的缺失」**。GraphWalker/XState 只能保证「画进模型的都被测到」;漏画的迁移、漏画的状态,工具永远不知道。漏画靠:(a) TLA+/P/Alloy 对需求性质的断言(性质写不出来=规格有洞);(b) ISTQB 状态表填空把「无效迁移」显式化;(c) MAST 分类学做失败模式 checklist。
2. **模型检查的实例化边界**。TLA+/P 的穷举是在有界实例(有限状态值、有限消息数)内的穷举;「验证过」≠「任意规模下成立」,需要用对称性/归纳或抽象补(TLA+ 体系内为 TLC 约束与性质参数化;这是形式化方法公认边界)。
3. **建模成本**。AWS 论文明确 TLA+ 学习曲线("engineers ... able to learn TLA+ from scratch and get useful results in 2 to 3 weeks")且写规约本身是设计工作;P 语言定位就是降低此门槛(编程语言形态)。对用户的 40 节点引擎,P 的形态最对口;对网文 agent 的状态推进,XState 级别的轻量状态机+属性测试性价比更高(推断)。
4. **LLM 轨迹评测的不确定性**。LLM-as-judge 有评分方差,τ-bench 的 pass^k 就是为「同一任务通过率不稳」而设;轨迹评测只能当回归网,不能当结构完整性证明——流程结构断联仍须靠第 1/2/4 层的确定性手段。
5. **契约测试只覆盖「实际走过的交互」**。Pact 是 contract by example,没用到的接口行为不进契约;必须配 MBT 的路径覆盖,把「该走的路径都走一遍」才能生成够宽的契约。
6. **查证不到/仅二手的项**:Conformiq 的具体覆盖判据(厂商营销口径);qSampler 现状(官网未提);N-switch 的 ISTQB 进阶出处(未取证);Chow 1978 原文全文(付费墙,仅 DOI 与经典内容转述);MAST 三类占比具体数字(正文图表未提取到);Braintrust/Phoenix 详细能力(未逐页取证);LangSmith 旧版 simulation 功能现状(新文档未查到);「任意架构图→用例」的直接工业工具(未查证到,见 ④)。

---

## ⑥ 参考来源

**MBT / 用例派生**
- GraphWalker 官网:https://graphwalker.github.io/
- GraphWalker 生成器与停止条件 wiki:https://github.com/GraphWalker/graphwalker-project/wiki
- Stately/XState 测试文档(@xstate/test 弃用声明、Studio 生成测试路径):https://stately.ai/docs/testing
- xstate/graph 文档(getShortestPaths / getSimplePaths / 覆盖语义):https://stately.ai/docs/xstate-graph
- Hypothesis stateful 测试:https://hypothesis.readthedocs.io/en/latest/stateful.html
- quickcheck-state-machine:https://github.com/stevana/quickcheck-state-machine
- fast-check model-based testing:https://fast-check.dev/docs/advanced/model-based-testing/
- Conformiq 官网:https://www.conformiq.com/

**状态迁移测试方法学**
- ISTQB CTFL Syllabus v4.0.1 官方下载(状态迁移测试原文):https://istqb.org/?sdm_process_download=1&download_id=3345
- Chow, T. S., "Testing Software Design Modeled by Finite-State Machines", IEEE TSE 1978, DOI: 10.1109/TSE.1978.231865(原文付费墙,未直接全文阅读)

**设计期验证(形式化方法)**
- How Amazon Web Services Uses Formal Methods(CACM 2015,PDF 一手提取):https://lamport.azurewebsites.net/tla/formal-methods-amazon.pdf
- Microsoft P 语言官网:https://p-org.github.io/P/
- Alloy 官网:https://alloytools.org/

**契约测试 / API drift**
- Pact 官方文档:https://docs.pact.io/
- oasdiff:https://github.com/oasdiff/oasdiff

**LLM agent 评测与轨迹**
- Anthropic Building Effective Agents:https://www.anthropic.com/research/building-effective-agents
- OpenAI Agents SDK Tracing:https://openai.github.io/openai-agents-python/tracing/
- LangSmith 评估概念:https://docs.langchain.com/langsmith/evaluation-concepts
- DeepEval:https://github.com/confident-ai/deepeval
- Langfuse 文档:https://langfuse.com/docs
- LangGraph persistence/checkpoint:https://docs.langchain.com/oss/python/langgraph/persistence
- VCR.py:https://vcrpy.readthedocs.io/en/latest/

**失败分类学与仿真基准**
- MAST《Why Do Multi-Agent LLM Systems Fail?》:https://arxiv.org/abs/2503.13657
- τ-bench:https://arxiv.org/abs/2406.12045
- SWE-bench:https://www.swebench.com/

---

## ⑦ 本地衔接与落地建议（huzilin 本地分析，非调研引用，性质均已标注）

> 本节把 ①-⑥ 的调研结论映射到本地现状（wbflow/dsh-flow、novel、既有测试 skill），供拍板。

### 7.1 本地现状与缺口定位

**已有长板（本地事实）**：执行资产层已是业内第 3 层水准——wbflow T17 e2e harness（scratch 库每轮重建、软失败收全、审计逐条对账、三条剧本真库两轮 PASS）就是 τ-bench 式「终态库 diff 判分」的本地实现；`to-qa-testcases` 的资产三原则（一键重建/一键复跑/结果可对比）与之同构。执行期不缺，**缺的是上游**。

**缺口 1（设计期，我的推断，依据=T17 五项疑点的成因）**：无模型健全性检查。`test-design-gate` 七源盘点的结构性盲区：它校验「用例对齐了文档」，但「设计断联」恰恰是文档（oracle）本身的洞——spec 里没写的迁移（exec→review 无端点）、没定义的语义（并行 join），七源照样盘点通过。对应调研第 1 层（P/TLA+/状态表填空）本地为空白。

**缺口 2（派生期，我的推断）**：「全路径走通」「分支剧本」目前是口号不是判据。`to-qa-testcases` B 组写「全节点/全路径」但没定义什么叫全；C 组异常分支靠人肉枚举。对应调研第 2 层的 ISTQB 三级判据（全状态 < 有效迁移 0-switch < 全迁移含无效迁移逐条打）本地未落地。

**缺口 3（集成期，本地事实+推断）**：动作 ID 点号解析失败、快照缺字段这类数据断联靠联调/E2E 才暴露。api.proto 已是契约真相源（本地事实），但差两步：CI 级 breaking diff（oasdiff/buf breaking 等价物）、以及「模型定义迁移集 vs 实际可执行迁移集」差集检查（ISTQB 状态迁移覆盖的量化形式）。

### 7.2 archify 思路终判（与调研互验后）

**结论：方向对，载体要换。** 业内第一档先例（GraphWalker/XState/P）证明「图→用例」成熟，但共同前提是**图必须是机器可执行的状态机规约**；archify 交付的 HTML/SVG 是给人看的，缺事件/guard/动作语义，直接从 HTML 推演用例无业内先例（调研第 ④ 节一手结论）。

可行修正（我的推断）：
- **wbflow 不需要 archify 当模型源**——`flow_def_nodes` seed（40 节点，含 scope_expr/gate/user_gate/node_actions/expected_outputs）已经是可执行真相源，比图更强（图反而丢了字段级信息）。
- archify 的正确角色剩两个：①设计评审可视化介质——按 ISTQB 结论，「画 lifecycle 图+填状态表」的过程本身就是设计评审（把无效迁移显式化），新增/改流程时画一张是值得的；②novel 项目若无 seed 级模型，可「先 archify 图 → 半自动转状态机 DSL」起步。

### 7.3 落地建议（三步，按性价比排序，均待拍板）

**第一步（纯本地脚本，零新依赖，直接命中断联发现）**：写「迁移覆盖差集」检查脚本——从 flow_defs seed 枚举模型定义迁移集（节点×node_actions），对照 e2e 已走迁移集输出差集；同时校验每个非终态节点的每条动作都能被下游解析（align.opening 类缺陷此步即拦，动作→端点映射表逐行核对 exec→review 类缺口）；加 ISTQB 无效迁移抽检（每状态挑代表事件打非法迁移，验证 409 拦截）。产出物 = 断联/漏测清单，喂给 `to-qa-testcases` 当 B/C 组设计依据。

**第二步（设计期，纳入推演流程）**：docs-tmp 推演表中每个状态机（session status、issue 状态机、评审轮转、打回重入）填 ISTQB 状态表（行=状态，列=事件，空格=显式无效迁移）；并行分支处强制显式写 join 语义；新增流程先画 lifecycle 图+填表再落 spec。可用 archify 辅助评审。

**第三步（可选，成本高，按需）**：wbflow 裁决链/回退链的交错类缺陷（并发 collect、并发回退、escalate 竞速）用 P 语言建模检查，或 Go 侧轻量等价物（rapid T+ 模型对账测试）；novel 状态推进用轻量状态机 + 属性测试（Hypothesis 范式）。契约层：CI 加 proto breaking 检查 + expected_outputs 字段溯源表（每个下游读方 ← 上游产出节点），把 T17 疑点③类前移到 CI。

### 7.4 与既有 skill 的衔接（待拍板，未擅自修改）

- `test-design-gate` 七源清单可增第 8 源：「模型覆盖判据」（迁移差集报告+状态表），堵「oracle 本身有洞」的盲区；
- `to-qa-testcases` B 组「全路径走通」改写为可判定判据：「有效迁移覆盖 100% + 关键 1-switch 迁移对覆盖」。

两处改动都是 skill 修订决策，拍板后另行执行并录 semantica。
