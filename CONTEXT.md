# 团队项目工作台（Team Project Workbench）

DSH 之上「项目管理 + OA 流程 + 评审协作」三合一系统；本 glossary 收录流程/能力/技能/产物推演中确立的领域语言。权威设计见 `docs/v1.0/arch_design.md` 与 `docs-tmp/` 推演稿；本文件只做术语索引。

## Language

**工单（ticket）**:
地图上待决策/待研究的问题单元（原译"票"，统一为"工单"）。技能名 to-tickets 保留英文（它是技能标识，不翻译）。
_Avoid_: 票、卡片、事项

**能力（Capability）**:
某角色在某节点的专业行为单元（role→capability 1 对多），由一个或多个动作组成。
_Avoid_: 技能、任务、插件（这是别的概念）

**动作（Action）**:
能力内的可独立调度单元（如 backend-dev.dev / arch.prd-review.review），以 `on` 挂相位（entry/exec/review/respond）、可挂 skill 或一句话内联；节点以激活清单（node_actions）声明启用哪些动作。**Capability → inputs + actions**（capability 是 role 归属与数据可见性的结构壳，行为在 action/skill）。
_Avoid_: 步骤、子任务；respond_to_cr / respond_to_qa（退役动作名）

**相位（phase，on）**:
node.state 的可配动作相位，4 值：entry（进入/重入迎宾）/ exec（执行：首次主产出、重入返工）/ review（评审表态）/ respond（owner 响应：澄清或修复）。动作以 `on` 字段挂相位；旧 trigger 枚举（node_enter/dev_complete/…）与 trigger 词表已废弃，被相位+评审轮取代。
_Avoid_: trigger、时机、when、hook（编排层废弃，仅通知类扩展位）

**how（已降级为 skill 写作参考）**:
动作不携带 how 四要素（框架不强制）——复杂动作直接挂一个 skill（内部按需编排其他 skill，自带目标/自检/门禁），短动作一句话内联；依赖注入由 my_task 上下文承担。how-spec-v2 降级为 skill 作者的写作参考，运行时不再引用。
_Avoid_: how 四要素作为框架强制；behaviors（旧名）

**技能（skill）**:
DSH 技能（含 mp-* 与自建 to-xxx 产出 skill 家族）；≠ 能力（Capability）——能力是领域行为单元，skill 是 how 的 exec 环节引用的执行/产出技能。
_Avoid_: 能力（领域词）、插件

**工单（ticket）**:
地图上待决策/待研究的问题单元（原译"票"，统一为"工单"）。技能名 to-tickets 保留英文（它是技能标识，不翻译）。**≠ 评论/问题（Issue）**——Issue 是需求评审里的待解决问题（open/pending-confirm/resolved），工单是地图上的决策单元。
_Avoid_: 票、卡片、事项

**已处理（pending-confirm）**:
评论被提交者修订/回复完成、待评审者复核的状态（== IssueState 的 pending-confirm）。**≠ resolved**——resolved 是评审者确认后达成。
_Avoid_: processed（自定义新词，不引入）

**已解决（resolved）**:
评审者确认接受处理后的评论终态（resolve→confirm 双路径的确认侧）。
_Avoid_: 关闭（这是另一种语义）

**编排序列（execution）**:
how 的 exec 环节的形态——子流程受控枚举（如 [to-tickets, implement-techdesign]），可扩展不可随意；exec 可以是单个 skill，也可以是编排序列。
_Avoid_: 流程串、步骤链

**产出 skill 家族（to-xxx）**:
按 to-spec 形态（Process+Template）自建的产出技能：to-brd（业务需求文档）、to-prd（产品文档）、to-techdesign（技术设计，to-rd-techdesign / to-fe-techdesign）等；模板仍是 docs/templates 单一真相源。
_Avoid_: spec、to-spec 命名（旧）

**BRD（业务需求文档）**:
需求对齐节点 exec 动作（to-brd）的产出——用户与 PM 对话结论的固化物（目标/范围/价值），product_doc 的第一上游依赖；ProductType=brd。
_Avoid_: spec（旧概念，勿复用）；alignment（那是 plan 流程版本级立项对齐文档，另一个词）

**implement-techdesign**:
开发侧编排协议（原 implement-spec 改名）：基于 techdesign 文档编排任务图（模块→tdd/implement→依赖序），按并发度派 subagent 执行。
_Avoid_: implement-spec（旧名，原生 spec 概念废弃）

**技能分层（direct / dependency / scaffold / discarded）**:
直接使用（流程内调用）· 底层依赖（被间接引用，不直接调用）· 初始化脚手架（与 flow 无关）· 废弃/忽略。
_Avoid_: 当前推演期/插件实现期（旧判定词，已废弃）

**qa 职责（联调/验收/缺陷）**:
qa 聚焦：联调 + 验收 + 缺陷判定（验收节点仅 full 渲染；exec=accept / 重入=retest 复验重测；缺陷响应链=rd/fe clarify 确认归属 + 修复 diagnosing-bugs）；用例设计归 qa 的 accept 内含（不再独立节点）；qa 评审以对象化评审 skill 减少上下文。
_Avoid_: qa 产用例（旧职责）；respond_to_qa（退役动作名）

**subagent 会话（挂载到 flow 节点）**:
每个 subagent 有独立稳定 sessionId，持久化跨重启；UI 经插件页面从 flow 节点查看历史/进行中的 subagent 对话（不经 Leader 中转）。
_Avoid_: 弹窗问答（ask_user_question 是另一通道）