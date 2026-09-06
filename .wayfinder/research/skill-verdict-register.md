# mp-* 技能裁定清单（逐一定夺账本）

> **用途**：37 个 mp-* 技能逐一裁定「是否使用、如何使用」的权威登记；裁定完成后据此生成实现清单（直接使用/被依赖分层）并简化盘点文档。
> **流程**：用户按 A-F 组逐一裁定（2026-09-02 起）；已裁条目标注日期与结论；「使用」的后续标注引用方式与场景（接 current 推演稿）。

---

## A. 对话/决策协议类（7）

**用户裁定（2026-09-02）**——交互模式：这一类主要作为**需求对齐阶段的输入**：
> subagent 开始该阶段时直接与 user 交互——先大体说明「自己是哪个需求、什么阶段、要和用户确认什么」，让用户了解上下文；并提示用户可用的交互类 skills；用户自行决策：一句话直接阐述上下文，或调用 grill-me / wayfinder / domain-modeling。

| # | 技能 | 裁定 | 备注 |
|---|---|---|---|
| 1 | grilling | ✅ 保留（底层依赖，不直接使用） | grill-me/wayfinder 的底层原语 |
| 2 | grill-me | ✅ 保留（用户可选调用） | 需求对齐阶段用户自行决定是否调用；一问一答单题节奏 |
| 3 | grill-with-docs | ❌ 废弃（2026-09-02） | 未入保留清单；交互已由「subagent 直接对话 + grill-me/wayfinder/domain-modeling」覆盖 |
| 4 | to-questionnaire | ❌ 废弃（2026-08-27） | 与流程无关 |
| 5 | wait-what | ✅ 保留（底层依赖，不直接使用） | 常识性沟通协议（失准重述） |
| 6 | domain-modeling | ✅ 保留（用户可选调用） | 术语治理；需求对齐阶段用户可调 |
| 7 | wayfinder | ✅ 保留（用户可选调用） | 地图方法；需求对齐阶段用户可调 |

## B. 文档/写作类（5）——类别调整（2026-09-02 用户裁定）

> **类别调整（2026-09-02 二版）**：本类不再用"spec"概念——**to-spec 属产品阶段，直接更名 to-prd**；技术设计层为 **to-techdesign**（to-rd-techdesign / to-fe-techdesign），与 to-prd 对齐；原生 spec 概念废弃不用。

| # | 技能 | 裁定 | 备注 |
|---|---|---|---|
| 8 | writing-for-agents | ✅ 使用（**推演期工具**，写 how 时用，不常驻） | 基于它写更好用的 how（v2 纪律来源）；类别从"文档写作"调整为"推演期工具" |
| 9 | writing-beats | ❌ 废弃（2026-09-02） | 用不太到（概念旅程写作，流程无此需求） |
| 10 | writing-fragments | ❌ 废弃（2026-09-02） | 用不太到（访谈碎片沉淀，已被需求对齐直接对话覆盖） |
| 11 | writing-shape | ❌ 废弃（2026-09-02） | 用不太到（文档成型协议，产出由 skill 模板驱动） |
| 12 | to-spec → **to-prd** | ✅ 使用（产品阶段产出 skill + to-xxx 家族基础） | 2026-09-02 二版更名：**to-spec 改为 to-prd**（产品阶段，产出 product_doc 完整版，补足产品内容）；作为**自建 to-xxx 家族的形态模板基础**（to-techdesign 等按 Process+Template 形态复制）——与产出形态推演 D2 合流 |

> **to-xxx 家族（2026-09-02 二版定名）**：不以"spec"命名——**to-prd**（产品阶段，product_doc）、**to-techdesign**（技术设计，to-rd-techdesign / to-fe-techdesign，对应 rd_design / fe_design），与 to-prd 对齐；to-tickets / implement-spec 等开发链路的输入**对齐到 techdesign 产出文档**（非原生 spec）；模板引用 docs/templates（D1=B），每文档类独立（D2）。

## C. 编码实现类（14）

| # | 技能 | 裁定 | 备注 |
|---|---|---|---|
| 13 | tdd | ✅ 使用（开发侧技能） | 2026-09-02：开发阶段执行 skill（红绿循环/测试驱动实现），与 implement 结合分析（用户参考：tdd=测试定义行为、不适用 UI/胶水类；需求前置=技术设计文档）；**不拆测试编写给 qa**（qa 用例设计已转开发侧自产，见 qa 职责重定义） |
| 14 | code-review | ✅ 已裁（单动作串行两子步） | 2026-08-27 见映射稿 |
| 15 | diagnosing-bugs | ✅ 维持（respond_to_qa 新动作 + acceptance-test 判定步） | 2026-09-02 用户确认维持响应 qa 场景 |
| 16 | codebase-design | ✅ 底层模块（不直接调用） | 2026-09-02：作为底层词汇模块被 tdd/implement/code-review 等间接调用（其接口面原则进 backend-design §3.5 已载） |
| 17 | implement | ✅ 使用（开发型普通实现） | 2026-09-02：tdd vs implement 按模块特性选择（核心逻辑→tdd，UI/CRUD/胶水/API 集成→implement）；与 tdd 结合分析见用户参考 |
| 18 | implement-spec → **implement-techdesign** | ✅ 使用（**开发侧编排协议**，RD/FE 调用） | 2026-09-02 更名+修正：**改名 implement-techdesign**（不使用原生 spec 作为输入，对齐 to-techdesign 产出文档 rd_design/fe_design）；由 rd/fe（subagent）调用（subagent 可再派 subagent——spawn depthLimit=true）；**并发度控制**——编排与调度解耦（任务图：模块→tdd/implement→依赖序；并发度=N 并行 / =1 串行）；MVP 默认串行；DSH 锚点=maxConcurrentJobsPerOwner（默认 10）。落点：backend-dev.dev 执行步引用（开发动作内部协议） |
| 19 | resolving-merge-conflicts | ✅ 使用（rd/fe preset 下的技能） | 2026-09-02：挂在 rd/fe preset 下的技能（冲突解决场景由 rd/fe subagent 调用） |
| 20 | improve-codebase-architecture | ✅ 使用（架构师需求对齐阶段可选） | 2026-09-02：架构师需求对齐阶段可被用户选择；需求对齐交互时依次提示 improve-codebase-architecture、wayfinder |
| 21 | setup-ts-deep-modules | ✅ 使用（新项目初始化脚手架） | 2026-09-02：新项目初始化时调用（deep module 边界基线）；与 flow 无关 |
| 22 | setup-pre-commit | ✅ 使用（**初始化脚手架**，开发环节不使用） | 2026-09-02 修正：仅是初始化钩子（一次性装 Husky+lint-staged+Prettier 写 .husky/pre-commit），**不动态更新钩子**；提交时 lint/typecheck/test 由搭好的钩子自动触发，无需 agent 调 skill。归入初始化类（与 setup-ts-deep-modules 同组），**从开发链路 execution 编排移除** |
| 23 | migrate-to-shoehorn | ✅ 使用（前端提交前校验） | 2026-09-02：前端 pre-commit 前调用（测试 as 断言迁移校验） |
| 24 | git-guardrails-claude-code | ❌ 不启用 | 2026-09-02：DSH 沙箱/审批已承担拦截 |
| 25 | scaffold-exercises | ❌ 已裁：废弃 | 2026-08-27 无增量；2026-09-02 复确认：不使用 |
| 26 | wizard | ⏸ 暂忽略 | 2026-09-02：目前无发布阶段，暂时忽略 |

## D. 任务编排/状态类（6）

| # | 技能 | 裁定 | 备注 |
|---|---|---|---|
| 27 | to-tickets | ✅ 使用（开发链路前置编排，结合 /goal） | 2026-09-02：开发者（rd/fe）在 **implement-techdesign 之前**使用——把 **techdesign 产出（rd_design/fe_design）** 拆成垂直切片工单（Blocked by + 验收清单）；**结合 dsh /goal**（事件溯源目标服务：active/paused/blocked/complete + 自动续跑推进 rounds）——goal 承载"整个开发目标"跨会话追踪。**how 执行要素升级为编排序列**：如 `execution: [to-tickets, implement-techdesign]`（先拆工单→再编排调度） |
| 28 | triage | ✅ 使用（归 qa 测试验收/缺陷管理阶段） | 2026-09-02：机制与 qa 缺陷管理高度咬合——状态机（needs-triage/info/agent/human/wontfix）= 缺陷处理流程（待确认/补信息/归域派发/人工/忽略）；verify claim=最小化复现确认（对应 acceptance-test 判定步）；redundancy=缺陷查重。作为 qa 领域模型一部分（与 respond_to_qa / defect 管理协同） |
| 29 | handoff | ❌ 不使用 | 2026-09-02 修正：原裁"被 my_task 程序化吸收"→ 用户明确不使用（其思想由 my_task 最小返回集承载，不落 skill） |
| 30 | claude-handoff | ❌ 不使用 | 2026-09-02：与 handoff 同族，不落 skill |
| 31 | loop-me | ✅ 底层依赖（不直接使用） | 2026-09-02：trigger/checkpoint/push right/brief 词汇作为底层依赖被其他技能/框架间接使用 |
| 32 | retro | ✅ 使用（CR 复盘环节） | 2026-09-02：编码复盘——CR（代码评审）使用（七类环境改进候选作为评审收尾的改进建议来源） |

## E. 已裁废弃/不适用（2）

| # | 技能 | 裁定 | 备注 |
|---|---|---|---|
| 33 | teach | ❌ 废弃 | 2026-08-27 |
| 34 | ask-matt | ✅ 仅工具层（帮助入口） | 2026-08-27，不挂 any how |

## F. 未分类（3）

| # | 技能 | 裁定 | 备注 |
|---|---|---|---|
| 35 | setup-matt-pocock-skills | ⏸ 忽略 | 2026-09-02：与我们自建 .wayfinder/.plan tracker 体系不匹配，忽略 |
| 36 | prototype | ✅ 使用（产品阶段，需求对齐时） | 2026-09-02：产品阶段需求对齐时——如果有前端，可使用此能力对齐需求（一次性原型：LOGIC/UI 分支，用完删除或吸收答案） |
| 37 | research | ✅ 使用（调研员用） | 2026-09-02：research-cap 由调研员（investigator）使用（一手源调研、逐主张溯源、带引用落盘） |

---

## 交互模式（A 组定稿，2026-09-02）

subagent 进入需求对齐阶段时**主动开场**（直接与 user 对话，经 subagent 会话通道）：
1. 说明：这是哪个需求（reqId/名称）的什么阶段（节点）+ 将要和用户确认什么 → 用户立即获得上下文；
2. 提示：用户可选择的交互类 skills（grill-me 单题盘问 / wayfinder 地图推演 / domain-modeling 术语治理，或直接一句话阐述上下文）；
3. 用户选择后按对应协议推进；对话发生在本节点挂载的 subagent 会话（D6：插件页面节点处可见）。