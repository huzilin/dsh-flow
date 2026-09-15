# dsh-flow 技术名词/技术方案演进映射表

> 生成：2026-09-12。目的：每个现行术语都能回溯到用户最初的中文方向性描述与中间代际名；任何一次改名都能机械查到「哪些文件哪些位置需要跟着改」。
> 数据源（同目录）：decision-timeline.md（126 条拍板，条目号引用如 ISS-08）、dsh/zcode/testflow-sessions-decisions.md（用户原话）、CONTEXT.md（现行词汇表）、docs-tmp/capability-model-final.md §8（作废留痕）。
> 权威规则：同题后拍覆盖先拍；「中间代际名」列按时间序，括注废弃日期；**无用户原话、仅间接证据的代际名标 ⚠️**。

**文件缩写**：cm=docs-tmp/capability-model-final.md；ca=docs-tmp/capability-actions-final.md；dt=docs-tmp/data-tools-final.md；bf=docs-tmp/business-flow-derivation.md；ctx=CONTEXT.md；ad=docs/v1.0/arch_design.md（注意：含一版残留词，见 B 节）；inv=testflow-interaction-inventory.md；sa=rd/domain/issues.go 等现行代码。

---

## A. 术语演进映射总表

### A1 issue 状态机域

| 现行术语 | 用户最初中文方向性描述（原话摘录+日期来源） | 中间代际名（按时间序，标废弃日期） | 现行定义（一句话+权威§） | 拍板依据 |
|---|---|---|---|---|
| **issue（评论/评审问题）** | 「QA 的缺陷管理，方向是反的：qa 建立一条缺陷，需要对应负责人去解决，qa 复核不通过，qa reopen」（08-16 00:42 dsh·4228） | 缺陷/评论混称（08-16）→ issues discussion 唯一载体（08-19 01:50）→ 与 wayfinder ticket 混译「工单」（09-03 21:59 划界废弃混用） | 需求评审中的待解决问题，≠ 地图工单 ticket；ctx 词条「评论/问题（Issue）」 | ISS-01~08 |
| **open（待澄清）** | 「open 的 issue 全都被处理」准出（08-25 20:22 dsh·efa3）；中文方向「待处理/待澄清」（09-03 22:15「将 open 的变为已处理，而不是 resolved」） | 无独立前代名（open 英文词 08-16 起直用，中文「待处理」为并行口语，09-07 定名 pending-confirm 后弃用） | issue 初态：待澄清；澄清循环回退入口（reopen 计数 loop_count） | ISS-03/04/05/08；cm §1.5② |
| **pending-confirm（已澄清待复核）** | 「open -> pending-confirm 后面会有 3 个状况：1 澄清解释了疑问 resolved 2 确定了是问题标记待修复 3 回到 open」（09-07 23:23 zc·Wayfinder） | 「已处理」（09-03 22:15 口语拍板，09-07 废弃）→「待复核」（08-16 00:49 超时提醒语境口语） | 评论被提交者修订/回复完成、待评审者复核的状态；**是上层 node 阶段非动作**；ctx 词条「已处理（pending-confirm）」 | ISS-04/05/08 |
| **to-fix（待修复）** | 「确定了是问题，可以标记待修复」（09-07 23:23） | 中文「待修复」（09-07 23:23）→ to-fix 英文定名（09-11 ISS-08 三动作定稿） | 确认为真问题、进入修复期的状态；触发打回 | ISS-05/08；cm §1.5③ |
| **fixed（已修复）** | 「待修复->已修复 重新进行 review，修复成功则 resolved，修复失败回到待修复」（09-07 23:23） | 中文「已修复」（09-07 23:23）→ fixed（09-11 ISS-08）；declare-fixed 事件名（09-11 批注，自循环裁非法） | 修复完成待复核；上升判定时点状态（issue 应为 fixed） | ISS-05/08；OVR-05/06 |
| **resolved（已解决）** | 「只有澄清后确认是问题，两方达成一致才是需要被解决的任务。但全部被解决才能通过是没问题的」（08-24 21:45 dsh·5806） | 「open 都 resolved」单判据（08-25~09-03 22:08，被 09-03 22:15 分阶段判断推翻）→ 澄清/修复两阶段共终态（09-07 23:23 起） | 评审者确认接受后的评论终态（resolve→confirm 双路径确认侧）；overridden 动作并入此态 | ISS-03/04/05/08 |
| （废除）**overridden** | 无用户原话——一版实现注入（issues.go v2 修复轮混入）⚠️ | overridden（一版动作/状态，09-10 实现；08-19/09-08 dsh 派单间接快照 ⚠️） | **不存在**：「overridden 不存在，我理解 overridden 动作也应当置为 resolved」（09-11 22:50）；override 词仅存于上升仲裁开格语义 | ISS-07/08；NAM-07 |
| **reopen（重开）/ revise（修复重做）计数** | 「每次回到之前状态都要计数……澄清阶段使用这个计数器表示 reopen 次数，修复阶段表示 revise 的次数」（09-07 23:23） | 双计数器设想（09-07 23:23）→ 单字段 loop_count 阶段语义解释（cm §1.5：澄清读=reopen、修复读=revise） | loop_count 单字段，按所处阶段解释为 reopen/revise 次数 | ISS-05/08；cm §1.5 |

### A2 评审状态域（reviewState 五代）

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **reviewing / agree / unagree（per-agent 三态）** | 「Architect、RD、QA、FE 都参与评审，他们各自的评审意见我也要看到、留存。并且他们的评审状态也要是可监测的」（08-14 19:05 dsh·4228） | v3 审批状态词「-、collect、reviewed、agree、approve、reject」（09-05 20:21，v5 废）→ UI 药丸「pass/block/-/澄清中/复核中」（09-07 13:43 被批评，同日废）→ v4「'-'→reviewing→clarifying⇄confirming→agreed」（09-07 23:33 定名，**23:48 用户亲口否定**）→ v5 per-agent reviewing/agree/unagree（09-08 00:11/00:26） | 每评审位独立轮转：reviewing 中→自判上报 agree/unagree；「每个评审位独立 reviewing/agree/unagree，clarifying/confirming/agreed 不再作为存储态——这个我认可」（09-11 23:08） | REV-01~07；cm §1.2 |
| **collect（聚合收集）** | 「collect 收集所有 reviewers 的表态，收集完成后存在 open 且未 agree 的 issue，则推进到 replied」（09-05 20:21） | 相位 collect（v3 可配期，09-05 20:21）→ collect 内置化「不需要挂载任何 action，它是关于审批的内置的东西」（09-08 00:11）→ barrier 判定裁掉（09-07 23:33「瞬间状态不存储」） | plugin 内置三段：收票→判定（全 agree→approved；有 unagree→澄清循环）→再收票；不暴露、无专门 state | COL-01~05；cm §1.2a |
| **approved / rejected（门禁判定）** | 「至于 pass 或 reject 可以通过是否仍有工单处于 open 来解决」（09-07 23:23）；「approved 和 rejected 我们可以认为是门禁的状态」（09-07 23:33） | pass/block（一版药丸，09-07 13:43 废）→ approve/reject（09-05 表态语义，09-07 23:33 起升为门禁汇总） | collect 输出的节点级汇总结果（last_decision）：approved 进下节点、rejected 回上节点，「仅此而已」 | COL-04/05；REV-05/06；cm §1.2a/§1.3 |
| （代际⚠️）**collecting** | 无用户原话 ⚠️——仅存于 dsh 派单间接快照与 .plan/proposals/state-tables-draft.md 残留 | collecting（v2 期 UI/派单词，09-05 20:21 被 collect 正式词取代） | 退役；现行同义=collect 内置收票 | COL-03；dsh-sessions 疑义1 |
| （退役）**reviewed** | 「reviewed 可以」（09-05 03:46，作 collect 后状态定名） | reviewed（09-05 03:46 定名→09-05 20:21 进入 v3 审批状态词表→v5 退役） | 退役；职责并入 collect 内置判定与 per-agent 轮转 | NAM-03；REV-06 |
| （退役）**clarifying / confirming / agreed（unagreed）** | v4 是 assistant 生成、用户 09-07 23:48 亲口否定「这个是错的」——无确认原话 ⚠️ | clarifying⇄confirming→agreed（09-07 23:33~23:48，v4 链）→ unagreed 变体（09-07 23:48 用户口述形状）→ v5 全废（09-08 00:11） | 退役：「clarifying/confirming/agreed 不再作为存储态」（09-11 23:08） | REV-05/06/07 |
| **agreed（collect 层专用词）** | 09-12 裁决原话：「agreed 是 collect state，对于每一个 reviewer 的结论中，达成一致的专用词汇。当全部进入 agreed 则进入 pending-confirm。所以这个只是 collect 内部的东西。但不应该扩大。」 | v4 曾用作节点级存储态（09-07 链）→ v5 废除存储用法（09-08/09-11）→ 09-12 定格为 collect 层判定输出/门禁展示词 | 全评审位结论一致时 collect 判定输出 → 进入待确认（gate_pending）；仅限 collect 层，不作 reviewState 存储态、不扩大到 collect 之外。API 侧对应词（gate_state/collect.go）按 NAM-09 gap 修复流程待代码修复后统一 | REV-07/08；CONTEXT agreed 词条 |

### A3 表态/裁决动词域

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **agree / unagree（评审位表态）** | 「collect 判三子状态：未表态、unagree、agree」（09-07 23:23） | approve（09-05 02:39 表态通过定名，后废）→ agree（09-05 18:09「agree 我认可」）→ v5 agent 自判上报（09-08 00:11「有工单就直接告诉你未达成」） | agent 自判：有 open 工单报 unagree、没有报 agree；plugin 不判 | COL-04/05；REV-06/07；NAM-02 |
| （退役）**pass / block** | 一版 UI 词，用户批评即废：「block 是不是等于 reject？如果是的那就展示 reject」（09-07 13:43） | pass/block/-/澄清中/复核中（v3 药丸枚举，09-07 13:43 废） | 退役；表态=agree/unagree、门禁=approved/rejected | REV-04 |
| （退役）**closing / transform / task_stale** | 「closing 我认为也不合理……qa 表态通过，如 pass」「transform 的语义我感觉也不准确」「task_stale 这个也很奇怪」（09-05 02:32） | closing/transform/task_stale（一版 hook/表态词，09-05 02:39 废） | 退役；表态定名 approve（后演 agree/unagree） | NAM-02 |
| **reject（打回裁决事件，单义化）** | 「reject 可以保留，比如用户拍板拒绝……都会有一个 reject 动作」（09-05 01:46）；「reject 是一种人感知的结论和表态，实际反应不在节点上，是回退到上一个节点」（09-05 18:58） | reject 多义期（表态/门禁/回退混用，09-05）→ 单义化=回退上节点（09-05 18:58）+节点内评审永不输出 reject（09-05 19:50） | reject=跨节点评审门禁不通过的裁决事件→plugin 回退上一节点；打回/回退口语均归此义 | ORCH-03/05；CR-04/05；NAM-02 |
| **barrier（裁）/ complete（降级）** | 「等全表态吧，否则并行任务会很困惑」（08-24 21:51）；「complete 可以存在，即与 reject 和 approve 不同。这个阶段只是表示我审批完了」（09-05 17:52） | 全表态 barrier（08-24，09-07 23:33「barrier 不必要」废）→ complete 相位（09-05 17:52，§8 降为瞬时信号废相位） | barrier 退役（实时查询即可）；complete=完成上报信号 `event:'complete'`，非相位 | COL-01/02/04；cm §5/§8 |

### A4 编排域（trigger / hook / state / 相位）

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **相位 phase（node.state 4 值）** | 「明确一套节点内编排，如 entry->exec(任何节点都要有一个明确的动作或者产出)->review 后面是一个循环的节点 响应->复核 直到 准出或打回」（09-05 01:46） | trigger 枚举封闭（09-03 22:08）→ node 标准 hook before/after（09-04 18:56）→ hook 本质还是 trigger（09-05 01:46）→「去掉 trigger 的语义直接统一 state」（09-05 18:21）→ v3 state=entry/exec/review/replied（09-05 20:21）→ **定稿 entry/exec/review/respond**（09-05 20:44） | 可配动作相位 4 值 entry/exec/review/respond；ctx 词条「相位（phase，on）」；旧 trigger 枚举（node_enter/dev_complete/…）与 trigger 词表废弃 | ORCH-01~05；ctx |
| （退役）**trigger 词表 8 词**（含 reviewed/replied/approve/reject 编排词） | 「trigger 枚举封闭，接受」（09-03 22:08）→ 后自评「trigger 的语义并不明确」（09-04 18:56） | trigger 8 词表（09-03 封闭→09-05 18:21 废；§8 留痕「trigger 8 词表」） | 退役；approve/reject 保留为裁决事件名，其余并入相位+reviewState | ORCH-01/02/04/05；cm §8 |
| （退役）**after-exec hook** | 「cr 挂在 after exec 上……为了不拆分 cr 到新节点，也不增加节点内的更多阶段，通过 trick 的方式解决」（09-05 19:34） | after-exec trick（09-05 19:34）→「cr 可以考虑挂在 collect，而不通过 trick」（09-05 20:05）→「你怎么又出幻觉了，把废弃的 after-exec 又拿出来了。你需要标注 hook 方案是废弃的」（09-05 20:44） | hook 层全部退役，仅预留通知/同步扩展位 | CR-03/05；ORCH-05；cm §8 |
| （并入）**re-entry / re-exec → entry / exec** | 「对重新被打开的节点来说，是 re-entry 可能更清晰」（09-05 01:22）；「re-exec 则定义为，重做、修改主产出」（09-05 01:58） | re-entry/re-exec 独立相位（09-05 01:22~19:34）→「re-entry 我认为可以去掉，并入 entry……re-exec 也并入 exec，区分首次进入还是再次进入，这个可以在节点上增加计数位」（09-05 18:21） | entry/exec 各支持首次/重入双配置，按 rolled_back_count 选择；plugin 回退动作保留 | ORCH-04；NAM-04；cm §1.4/§8 |
| **respond（owner 响应相位）** | 「rework 不够好，大部分场景其实都不是 rework，而是澄清，所以这个语义过窄」（09-05 20:34）；更早「同节点内的响应是 feedback」（09-05 01:22） | feedback（09-05 01:22，并入 respond 废）→ rework（09-05 20:34 废）→ **respond 定名**（09-05 20:44）；v3 短暂含 replied 相位（09-05 20:21，20:44 废） | respond=owner 响应期：澄清（clarify）与修复（fix_on_review）；开发节点特化只挂 fix_on_review（CR 异化） | NAM-05；ORCH-05；ACT-02 |
| **done（节点完成阶段）** | 「那就是如果通过到下一个节点，就自动进入 done 这个阶段吧。这个阶段禁止挂载」（09-11 01:32） | pending/done/gate 状态占位（一版，§8 裁「编排进度展示无占位」废）→ done 复活为自动阶段（09-11） | 通过后自动进入 done，禁止挂载；pending/gate 占位仍退役 | CMT-05；cm §8（冲突见 C2） |

### A5 动作域（action / clarify / fix / retest）

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **clarify（澄清）** | 「当 rd 收到评审反馈……需要澄清沟通分歧，或者调整补充文档内容」（08-16 15:04）；「评审阶段需要让被评审者有澄清的能力」（08-25 20:33） | rework（09-05 20:34 废）→ respond_to_cr / respond_to_qa 分动作（一版，§8 废「统一 clarify 来源无关」） | respond 相位单动作（跨节点评审路由）；开发节点不挂（cr 不进行澄清，09-07 03:14） | NAM-05；QA/CR；cm §5/§8 |
| **fix_on_review（响应 CR 修复）** | 「这三个（开发/自测/响应 CR）……统一都在 rd 的开发节点上」（08-24 06:34）；「fix_on_review 挂在 replied 上，这个只响应 cr 的」（09-05 19:34） | respond_to_cr（一版动作名，§8 废）→ respond(clarify+fix_on_review) 双挂（09-05 20:40 被纠「respond 只有 fix_on_review」）→ Q-F「并入 diagnosing-bugs」被用户二次纠正推翻（§8） | dev 节点 respond 相位唯一动作，只响应 CR 不做澄清 | ACT-01/02/04；cm §5.1 |
| **diagnosing-bugs（exec 重入响应打回）** | 「diagnosing-bugs 要挂在 reexec，也就是现在的 exec 中。再次进入后触发这个流程」（09-05 19:34） | re-exec 挂载（09-05 01:58~19:34）→ 并入 exec 重入（09-05 18:21）；「diagnosing-bugs 这个是 rd 的能力名」不可跨用（09-06 22:30 纠错） | dev exec 重入动作：响应打回（复现→定位→修复→回归） | ORCH-04；cm §5 |
| **revise（文档节点 exec 重入修订）** | 「修复阶段表示 revise 的次数」（09-07 23:23）；重入修复=「基于问题修复文档」（08-24 21:45） | 无独立前代名（隐含在 re-exec/re-entry 期） | 文档产出节点 exec 重入动作：[produce, revise, clarify] 激活清单 | ISS-05；cm §5 |
| **retest（复验重测）** | 「exec 重入……不应该是补充缺陷，应该是针对重新提上来的代码，重新进行测试，确认是否修复，并更新之前的缺陷文档」（09-07 04:23）；「acceptance-test.复验重测 给个英文名吧」（09-07 10:40） | acceptance-test.补充缺陷（09-07 04:23 前实现，被纠正废）→ 复验重测（中文名）→ retest（英文名**未见于用户原话**，见 C1）→ §8 留痕「retest 独立动作退役，复验=accept 重跑」与 §5.1「exec 重入=retest」矛盾（见 C1） | QA 验收节点 exec 重入=复验重测（ctx 词条） | QA-04；NAM-06；ctx |
| **一个 state 一个动作（skill 单挂）** | 「目前阶段是一个 state 只挂一个 skill」（09-07 03:14）；「之前已经严正声明过，一个 state 只能挂一个动作，怎么立即就重犯了？」（09-07 03:36） | 节点多动作模型（08-24 06:34，思想保留）→ R1 双动作（commit 647a493，undermined）→ 41f7c06 CR 异化单挂 | 铁律#1：每动作单挂 skill/inline；CR 内嵌 respond 只挂 fix_on_review | ACT-03/04/05 |
| **迎宾/开场白 → align.opening（entry）** | 「开场白，我们缺少这个内容」（09-04 18:42）；「迎宾也就是开场白，直接配在 entry 和 re-entry 中」（09-05 13:17）；「迎宾中，明确告知 agent，发送后不要有任何动作……等待本身就是 action」（09-05 13:26） | 开场白/迎宾（口语）→ entry 迎宾位（09-05 13:17）→ align.opening node action（现行 bf §节点0） | 交互节点 entry 相位 align.opening：program 渲染开场白、how 内置强制等待 | bf；DEV/ORCH |

### A6 能力/技能/产物名域

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **capability → actions（结构壳）** | 「skill 不等于能力，Capability→actions→其中 how 配置了 skills」（09-03 22:57） | capability 含 how/behaviors/reviewer_focus/inputs（一版）→ 裁 reviewer_focus（08-17 18:24「reviewer 的能力应该直接写在它的能力里」）→ how 骨架废（09-05 13:34）→ capability=inputs+actions 结构壳 | 能力=role 归属与数据可见性（inputs）+动作清单（actions）；行为在 action/skill | SK-04/05；DM-06；ctx |
| （退役）**how 四要素 / behaviors** | 「how 对应 behaviors，可以将原有文档更新成 how」（08-27 07:54 behaviors 改名 how）；「五段式骨架，可以抽象成 how 的标准，但需要名词更抽象」（09-03 13:52） | behaviors（→08-27 废）→ how/五段式（09-03）→ how 骨架废除（09-05 13:34「我倾向每个 action 直接挂一个 skill……极大降低框架复杂度」） | how 降级为 skill 写作参考（how-spec-v2），运行时不再引用；ctx「Avoid: behaviors（旧名）」 | SK-02/04/05；ctx |
| **brd（业务需求文档）** | 「可以叫 to-brd，来自用户的业务需求文档，统一就叫 brd」「对应路径也变成 docs/{{req-version}}/brd/brd.md」（09-05 03:22） | 需求对齐无产物（08-15 10:02「不需要任何产物」）→ 沟通视为产物（08-19 01:44）→ spec 概念（09-03 20:16 前）→ brd 定名（09-05 03:22） | ProductType=brd；用户与 PM 对话结论固化物；ctx「Avoid: spec（旧概念）」 | NAM-03；PT-04；ctx |
| **product_doc（prd）** | 「不再使用原生的 spec 作为原生输入……to-spec 我们认为是产品阶段的东西，比如我们就直接改为 to-prd」（09-03 21:39） | spec（原生输入概念，09-03 21:39 废）→ to-spec→to-prd 改名（09-03 21:39）→ to-prd 参考 to-brd 章节（09-09 22:39） | ProductType=product_doc；skill=to-prd；四文档互依赖 Focus 对应章节 | NAM-01/08；dt §product_type |
| **rd_design / fe_design（技术设计）** | 「implement-spec 也要调整命名，如 implement-techdesign。to-techdesign 和 to-prd 对齐」（09-03 21:39） | rd/design.md 目录形态（08-21 14:22 DOC-01）→ techdesign（09-03 21:39）→ rd_design/fe_design 受控枚举（dt） | ProductType=rd_design/fe_design；skill=to-techdesign；implement-techdesign 编排协议 | NAM-01；DOC-01；dt |
| **ProductType 受控枚举** | 「能力是可以被复用的，即我们当前有 4 个流程……按 flow_nodes.template_node_id 找产物可能导致复用性出现问题。这是一个很严重的问题……定义一个标准的产物的枚举」（08-24 07:07）；「还在使用 node1 这种表述（不行）」（08-24 07:19） | 位置引用 node1（08-24 07:19 禁）→ how 路径模板化（08-26 18:38）→ 本期收窄（09-03 22:45「本期先简单来」） | 受控枚举 13 值：brd/product_doc/rd_design/fe_design/architecture/research/plan/alignment/dev_notes/code_mr/test_result/defect/discussion（case_design 移除） | PT-01~04；dt §product_type |
| **工单（ticket）** | 「ticket 不用再翻译成票，翻译成工单。这个统一一下」（09-03 21:59） | 票/卡片（→09-03 21:59 废，ctx Avoid 列） | 地图上待决策/待研究的问题单元；≠ Issue；to-tickets 技能名保留英文 | NAM-01；ctx |
| **to-xxx 产出 skill 家族** | 「to-spec 作为后续我们自己实现 to-prd、to-techdesign 之类的基础」（09-03 20:16）→ 后 to-spec 命名本身退役 | to-spec（09-03 20:16 定位、21:39 改名废弃）→ to-prd/to-techdesign（09-03）→ to-brd（09-05 03:22）→ to-alignment（plan D1 node action） | 按 to-spec 形态（Process+Template）自建的产出技能家族；ctx「Avoid: to-spec 命名（旧）」 | NAM-01/03/08；ctx |
| **skill 分层（direct/dependency/scaffold/discarded）** | 「grilling 是底层能力……不直接调用，可废弃」「teach，用不上，废弃」（09-03 13:52） | 「当前推演期/插件实现期」判定词（旧，ctx Avoid 列废弃）→ 四分层 | 直接使用/底层依赖/初始化脚手架/废弃忽略 | SK-02/03；ctx |
| **arch.prd-review.review 命名样式（role.capability.action）** | 「arch-tech-review.review 应当是 rd.prd-review.review 类似这种（role.capability.action），你可以换成更合适的」（09-07 02:08） | arch-tech-review.review（全局平铺名，09-07 02:08 废） | 动作标识=role.capability.action 三段式 | ACT/命名；bf/ca |

### A7 scope / 版本 / 流程结构域

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **scope（full/rd/fe 渲染集合）** | 「当我们的一个业务需求实例，选择了不包括 fe，那 fe 相关的节点，以及节点中 reviewer 都不包含 fe 是么」（08-18 13:20） | scope 投影语义（08-18）→「scope=full\|fe\|rd 三者之一」（09-07 11:49）→ task 三值化+联调（09-07 11:55→12:06 同日回滚「回滚我刚才对 task 流程的改造」）→ 表达式制（commit 3500f77） | 渲染集合表达式；business 三值 full/rd/fe；B2 举一反三 sweep（rd\|full） | SCP-01~05 |
| **版本四态（待启动/进行中/已发布/已废弃）** | 「版本管理需要增加版本，待启动、进行中、已发布、已废弃。也要支持调整和推进」（08-15 02:36） | 废弃联动取消需求（08-15 02:41「废弃后，版本下的需求都应当推进为已取消」→ 02:57 当场纠正「我们不支持联动取消，之前已经确认了」）→ 已开始需求禁止废弃（09-09 01:22 裁决 A） | 四态单向推进；有需求不可废弃需先移出；未开始可删且需求回待分派池 | VER-01~05 |
| **需求池四态（待启动/进行中/已完成/已取消）** | 「软件版本→需求池（待启动/进行中/已完成/已取消）」（08-14 18:38） | 无代际变化（最早基线词汇，沿用至今） | 需求池四态基线，无废弃态（删除=取消关联） | VER-01；VER-05 |
| **评审双轨制（节点内 CR 异化 + 跨节点评审节点）** | 「rd-mr、fe-mr 评审这个节点去掉，评审直接挪到开发里」（08-15 10:54）；「不是 3 个节点，而是统一都在 rd 的开发节点上」（08-24 06:34） | 评审独立节点（08-15 10:58）→ CR 内嵌（08-15 10:54/08-24）→ after-exec trick（09-05 19:34 废）→ collect 挂载（09-05 20:05）→ 双轨制定稿（repo-doc §2.5） | 节点内评审评本节点产物永不输出 rejected；跨节点评审独立节点承载 | CR-01~06 |
| **flow_defs / flow_def_nodes（表名）** | 「使用 flow_defs / flow_def_nodes」（09-05 16:32，此前「flow_template/flow_template_nodes 名字不够直观」） | flow_template / flow_template_nodes（一版表名，09-05 16:32 废） | 流程定义表现行名；+node_actions 激活清单 | DM-07 |
| **（裁）reviewer_focus /（裁）快照表 / flow_node_reviewers 并入** | 「reviewer_focus 也是很怪的东西……reviewer 的能力，应该直接写在它的能力里即可」（08-17 18:24）；「去掉快照」（08-17 12:24）；「flow_node_reviewers 合并进 flow_nodes」（08-17 12:28） | reviewer_focus/inputs（08-17 18:24 裁）→ capability.inputs 复活为跨流程可见性上限（cm §5，语义已不同） | 字段退役；可见性由 capability.inputs + scope 承担 | DM-05/06 |

### A8 工具/交互/基础设施域

| 现行术语 | 用户最初中文方向性描述 | 中间代际名 | 现行定义 | 拍板依据 |
|---|---|---|---|---|
| **escalate（冲突上升）→ user 仲裁** | 「我不应当只有强制介入一种方式，我也可以参与他们的讨论……必要情况如果需要我确认，我可以直接在这个阶段参与确认」（08-14 19:24） | 强权 override / wb_override / overrideRole ⚠️（一版，09-11 裁 overridden 废除；「反复修复不成功→上升」由 escalate 承接）→ escalate 冲突响应机制（09-05 13:56）→ user 仲裁（09-06 20:07）→ fe 冲突裁决卡 wb_adjudicate（一版重构，刻意取代强权，inv B-29） | 澄清循环不收敛→escalate 标记→按需会话通道 User 仲裁→结论追加到冲突 issue | OVR-01~06；inv B-29 |
| **（移出）wb_decide → plugin 内置门禁 + user_gate** | 无直接中文原话；功能需求=「user 拍板/回复节点由 user 自行确认完毕防误判推进」（08-25 20:22） | wb_decide pass/reject（一版 Leader 门禁工具，ad 残留）→ final-approve 能力（§8 废「User 终审职责由 flow_def_nodes.user_gate 配置位承载」）→ decide-cap 收缩 create_reqs（§8） | 门禁判定=collect 内置输出 approved/rejected + flow_def_nodes.user_gate 配置位；wb_decide 工具退役 | COL-05；CR-07；cm §1.3/§8 |
| **评论锚点两级（range/document）** | 「要都支持（文章级/行级/diff 级）」（08-14 18:49）；「之前没说有章节级别的评论」（08-14 20:44 否决章节级） | 章节/行/块/文档多级设想（08-14）→ 四级锚点 block/line/range/document（一版实现，09-11 01:32 收窄）→「那就按飞书的来吧……我们就只支持划词+文档级别评论」（09-11 01:32）→ line/block 后端 400 废弃（单行=range(n,n)） | range/document 两级锚点 | CMT-01~06 |
| **persona → 说明/注释** | 「roles 里的 persona 直接改名为注释吧，或者说明」（08-18 12:32） | persona 字段（→08-18 12:32 废） | roles.persona=专业能力卡+协作协议说明 | DM/角色域 |
| **visibleNodes → 上下节点/inputs** | 「不必要叫 visibleNodes 这个名字，可以就叫其他节点或者上下节点。可见是推演阶段的东西」（08-24 04:22） | visibleNodes（08-24 04:22 废） | 可见性=capability.inputs（跨流程可见性上限）+scope 投影 | PT/DM-06；cm §5 |
| **inbox → task 合并** | 「我们取消 inbox 合并到 task，task 就有了被展示的需求」（08-17 12:05） | inbox（08-17 12:05 废） | 收件箱并入 task 视角；通知走 IM 收件箱通道（NTF-01） | DM/NTF-01 |
| **SQLite → MySQL** | 「3. sqlite」（08-15 09:25）；「我需要你使用 sqlite 作为存储……我需要明确的模型和语义」（08-16 15:39） | SQLite（08-15~09-08 01:59 归档期）→「数据库 MySQL。技术栈：Go。HTTP：Gin」（09-08 02:04） | MySQL+Go+Gin+web 前端；DDL/proto 硬约束（DM-09） | DM-01/08/09 |
| **outbox（通知可靠性）** | 「投递可选消息事务表，写表+推送，定时任务扫消息表中失败的」（09-10 00:40）；「没有 dead，必须全部通知到，无限重试，都是内部系统，给我一个人用的」（09-10 01:27） | retry5→dead 终态（assistant 方案，09-10 01:27 被推翻） | 事务表+无限重试，无 dead 终态（铁律#6） | NTF-04/05 |
| **test_result / accept（QA 验收）** | 「设计完就直接测试，合并在验收文档中」（09-04 15:35）；一版「test-result、acceptance 两个合并」（08-16 17:54） | 独立用例设计节点+用例评审节点（08-15 09:46 流程，09-04 02:05 初裁去、15:35 终裁并入）→ qa-testcase-review-template 废弃（09-04 15:46）→ case_design 枚举移除（dt） | qa 职责=联调+验收+缺陷判定；exec=accept、用例内含；ProductType=test_result/defect | QA-01/02；dt |
| **user_gate（用户拍板卡点位）** | 「节点 0：待启动 user_gate: 1 本期的先人来做卡点。但长期是要看需求依赖的」（09-06 20:29）；「全 agent 通过 → 唤醒 User 终审，这个要按需」（09-05 19:42） | 用户拍板节点（08-15 流程定义口语）→ final-approve 能力（§8 废）→ user_gate 配置位 | flow_def_nodes.user_gate=1 时门禁后唤醒 user 终审，否则机械通过 | DEV-05；cm §1.3 |

---

## B. 「改名必改点」：代际名反查 grep 模式（退役清查用）

> 用法：按行执行 rg（word 边界 `\b`）；标 ✅ 的文件=留痕/归档/Avoid 行，属**合法出现**，不修；标 🔥 的=疑似需改残留。`rework-return`、`wb_adjudicate`、`settled 系`（clarified/settle/qualify/declare-fixed）是**现行词**，勿误杀。

| 代际词 | 建议检查范围（rg 命令骨架） | 已知热点（本次 grep 实测） |
|---|---|---|
| `rework`（废相位/动作，09-05 20:44） | `rg -nw 'rework' --type-add 'all:*.{md,go,ts,tsx,sql,proto,vue}' -t all`（排除 `rework-return` 现行预留值） | ✅ rd/service/reject_it_test.go（2026-09-13 分支夹具改 feat/rd-fix，见 retired-terms-audit §6）；✅ .archive docs-tmp/data-tools-final.md（归档区豁免）；✅ rd/domain/reentry.go+rd/migration/sql/0001_schema.sql+docs/api/v1/api.proto（均为 rework-return）；✅ cm §8/时间线留痕 |
| `re-entry` / `re-exec` | `rg -n 're-entry\|re-entry\b'` 与 `rg -nw 're-exec'` | 🔥 docs/agent-subagent-plugin.md（re-exec）；🔥 .plan/capability-chain-finalize/tickets/03、map.md；✅ cm §8 留痕；✅ timeline/* |
| `after-exec` | `rg -n 'after-exec\|after exec'` | ✅ cm §8/CR-05 留痕；🔥 .plan/capability-chain-finalize/tickets/03 |
| `respond_to_cr` / `respond_to_qa` | `rg -n 'respond_to_(cr\|qa)'` | 🔥 .plan/capability-chain-finalize/tickets/01/03/05/09、map.md；✅ ca/cm §8 留痕、ctx Avoid 行 |
| `closing` / `transform` / `task_stale` | `rg -nw 'closing\|transform\|task_stale'`（transform 需人工辨 CSS transform） | 🔥 ca（closing）、.plan/capability-chain-finalize/tickets/03/04/07；✅ cm §8 留痕 |
| `pass` / `block`（表态语境） | `rg -n 'pass/block\|verdict.{0,30}(pass\|block)\|(pass\|block).{0,10}reject'`（通用词，需上下文） | 🔥 ad（门禁/表态段落）；✅ docs/v1.0/_archive/*；✅ timeline/* |
| `collecting` | `rg -nw 'collecting'` | 🔥 .plan/proposals/state-tables-draft.md |
| `clarifying` / `confirming` / `agreed`（存储态语境） | `rg -nw 'clarifying\|confirming\|agreed\|unagreed'`（agreed 通用性高，人工辨） | 🔥 .plan/proposals/state-tables-draft.md；✅ cm/ctx/timeline（「不作存储态」声明合法） |
| `barrier` | `rg -nw 'barrier'` | 🔥 .plan/capability-chain-finalize/map.md、tickets/03/07 |
| `replied` / `reviewed`（相位/状态语境） | `rg -nw 'replied\|reviewed'`（reviewed 需人工辨过去式用法） | 🔥 docs-tmp/how-spec-v2.md、ca；✅ cm §8 留痕 |
| `complete`（相位语境） | `rg -nw 'complete'`（合法=完成信号 `event:'complete'`/Complete 上报；人工辨） | 🔥 全仓 57 文件命中，优先 .plan/proposals、docs-tmp 流程描述段 |
| `overridden` | `rg -nw 'overridden'`（Go 通用词，人工辨状态语义） | 🔥 ad:349；🔥 docs/v1.0/02-module-flow-tree/{prd/prd.md,rd/design.md}；🔥 docs/v1.0/qa/{qa-test-cases,qa-defects}.md；🔥 rd/domain/issues.go:16 注释「resolved/overridden 拒绝」；✅ _archive/* |
| `wb_decide` | `rg -n 'wb_decide'` | 🔥 ad（349/355/1098/1254/1258/1310/1348/1392）；🔥 docs/v1.0/02-module-flow-tree/prd/prd.md；✅ _archive/* |
| `wb_override` | `rg -n 'wb_override'`（需先核实现行工具表是否保留重开 pending-confirm 的 override 工具，见 C9） | 🔥 ad；docs/v1.0/qa/qa-defects.md；docs-tmp/{data-tools-final,ui-interaction-final}.md |
| `decide-cap` / `final-approve` / `plan-cap` / `alignment-cap` | `rg -n 'decide-cap\|final-approve\|plan-cap\|alignment-cap'` | ✅ cm §8/ca 留痕（退役行合法）；🔥 ad 正文与 docs/v1.0/qa/*；✅ _archive/* |
| `flow_template` | `rg -n 'flow_template'`（注意排除现行 flow_template_nodes 字段引用错漏——现行表名 flow_defs/flow_def_nodes，但 node_actions 字段所在表节点定义可含 template 语境，需逐个辨） | 🔥 ad + docs-tmp/ca；✅ _archive/* |
| `reviewer_focus` | `rg -n 'reviewer_focus'` | 🔥 ad；✅ timeline/* |
| `to-spec` / `implement-spec` | `rg -n 'to-spec\|implement-spec'` | 🔥 docs-tmp/how-spec-v2.md（implement-spec）；✅ ca 留痕、ctx Avoid 行 |
| `qa-testcase-review-template` | `rg -n 'qa-testcase-review-template'` | 🔥 docs/v1.0/qa/* 与 docs-tmp 引用处（废弃声明行合法） |
| 旧 trigger 枚举（`node_enter` / `dev_complete` 等） | `rg -n 'node_enter\|dev_complete\|node_entered'` | 🔥 docs/agent-subagent-plugin.md 及 ad 工具段落（待核） |
| `inbox`（域概念语境） | `rg -nw 'inbox'`（现行合法=IM 收件箱通知通道；废=独立 inbox 域） | 需人工辨，全仓少量 |
| `visibleNodes` | `rg -n 'visibleNodes'` | 🔥 ad/_archive；代码若存同名字段一并清 |
| v4 链整句残留 | `rg -n "'-' ?→ ?reviewing\|clarifying.?⇄.?confirming\|unagreed"` | 🔥 .plan/proposals/state-tables-draft.md |

---

## C. 疑义（无法溯源/多义/缺用户原话）

1. **retest 正名缺原话+文档自相矛盾**：用户原话只到「给个英文名吧」（09-07 10:40，NAM-06），「retest」一词未见于会话取证（decision-timeline 疑义6 同判）；且 cm §8 作废留痕写「retest 独立动作退役，复验=accept 重跑组成」，而 §5.1 与 ctx 写「exec 重入=retest 复验重测」——两处矛盾，最终形态（独立动作 or accept 重跑语义位）需用户裁决。
2. **done 双重身份无权威定义文**：cm §8 曾裁「pending/done/gate 状态占位」退役，但 09-11 01:32 用户令「自动进入 done 这个阶段，该阶段禁止挂载」——done 现行是阶段、展示态还是 last_decision 推导值，无成文定义。
3. **settled 系事件名与代码词汇未对齐**：ISS-08 定稿「事件新名 settled 系（clarified/settle/qualify/declare-fixed 等）」，但现行代码/proto 用的是工具动作名（wb_reply/wb_resolve/wb_confirm_review/wb_reopen，rd/domain/issues.go、docs/api/v1/api.proto），settled 系词未出现——图文定稿词汇与实现词汇的映射表缺失。
4. **fixed×declare-fixed 自循环**：用户自标「fixed×declare-fixed 这个怎么解释（待补充）」（09-11 23:15），现行权威内部存在明示未决格。
5. **「待处理/已处理」中文词无独立首命名拍板**：09-03 22:15 口语「将 open 的变为已处理」与 09-07 23:23 英文定名 pending-confirm 之间，无用户逐词确认英文名的中间记录；ctx 词条名「已处理（pending-confirm）」是事后绑定。
6. **reviewed 定名（09-05 03:46「reviewed 可以」）的词位漂移**：其所指（collect 后状态 or 触发器）在 01:22 段与 20:21 段取证表述不一，后并入 v3/v5，无法精确还原当时所指。
7. **overrideRole / forceRevise / resolveReviewerConfirmed 等词**仅见于主 agent 派单（dsh 侧 08-19/09-08 间接快照），无用户拍板原话，无法入主链（dsh-sessions 疑义1 同源）。
8. **药丸 v3 中文词「澄清中/复核中」与 v4 clarifying/confirming 的对应**是 assistant 翻译，无用户逐词确认原话。
9. **ad（docs/v1.0/arch_design.md）权威级与词汇代际冲突**：ctx 称 ad 为权威设计，但其正文残留一版词汇（wb_decide/wb_override/overridden/decide-cap/final-approve/flow_template/reviewer_focus/pass/reject 门禁语义），与 docs-tmp 现行词汇冲突；「以哪份为准修订」未成文。
10. **打回/回退/rollback 口语链未完全定名**：reject 单义化（回退上节点）后，「rollback」「打回」「回退」在现行代码落为 reopen_source=upstream-reject/downstream-reject/rework-return（rd/migration/sql/0001_schema.sql:93）——这组枚举值的用户拍板原话缺失，属实现侧命名。
11. **wb_override 现行存废**：OVR-06 裁「override 只开 pending-confirm（后改 resolved）」指 issue 动作；但 wb_override 作为插件工具名是否保留（工具语义=提评者撤回/重开？）在 dt 工具契约与 ISS-08 间未对齐，需核 dt 契约原文与用户确认。
12. **collecting 代际词**（v2 期）无用户原话，仅 .plan/proposals/state-tables-draft.md 与派单间接快照可证其存在过；如需精确代际时间需回查 test-flow 一版代码 git 历史。

---

### 统计

- A 节映射行数：**55**（A1 8 / A2 6 / A3 5 / A4 6 / A5 7 / A6 9 / A7 7 / A8 7）
- B 节 grep 模式行数：**23**
- C 节疑义条数：**12**
