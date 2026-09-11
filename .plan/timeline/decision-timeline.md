# dsh-flow 历史决策时间线正本

> 生成：2026-09-11。综合自五份取证文件（同目录）：dsh-sessions-decisions.md、zcode-sessions-decisions.md、testflow-sessions-decisions.md、testflow-interaction-inventory.md、repo-doc-generations.md。原话全文以取证文件为准，本表压缩但保留「是否同一问题」的可判定性。

## 判权威规则（必读）

1. 按主题分组、组内按时间排序，**每组最后一条 = 现行权威**；被推翻条目在状态列标 `superseded-by <条目号>`（允许跨组指向）。
2. 方向相反的拍板对全部保留（如版本废弃联动取消、task scope 三值化又回滚、CR 内嵌↔独立），并明确哪条被哪条推翻；不删历史。
3. 本台账是「这个问题是否已有结论」的**第一查询点**：回答任何设计问题前先查对应主题组末条；同题后拍覆盖先拍。
4. **新拍板定稿后必须回写本表**：新条目置于对应组末，并给被推翻条目补 superseded-by；涉术语变更同步更新 `CONTEXT.md` 词条与 `terminology-evolution.md` 演进行、退役词过 retired-terms-audit 四档清查、审计沿革进本目录不进现行文档（方案敲定三动作，见 README 与 alignment-protocol 机制 2）。
5. 来源简写：`dsh·<短名>`＝dsh 会话库（test-flow 期与 dsh-flow 期目录为同批会话镜像，session-42289ff0 记作 4228）；`zc·<会话标题>`＝zcode 库（Wayfinder＝sess_53f2ab22，09-05~09-08，前半 cwd=test-flow）；`文档·`＝仓库文档/commit 侧证据（非用户原话时注明）；`一版·`＝tmp_1/test-flow 一版代码。

---

## 1. issue 状态机（三代：08-16/08-25 祖先 → 09-07 重设计 → 09-10 实现漂移 → 09-11 批注定稿）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| ISS-01 | 08-16 00:42 | dsh·4228 | 「QA 的缺陷管理，方向是反的：qa 建立一条缺陷，需要对应负责人去解决，qa 复核不通过，qa reopen」 | superseded-by ISS-08 |
| ISS-02 | 08-16 00:47~00:57 | dsh·4228 | 细则：非验收场景不指派（谁的文档谁处理）；不支持平级编辑删除；待复核超时提醒+轮询；qa 可打回不认可的修复，rd 一般不能打回 qa 确定的缺陷 | superseded-by ISS-08 |
| ISS-03 | 08-25 20:22~20:33 | dsh·efa3 | open issue 准出：open 全被处理才可过；澄清全在评审阶段完成、全部确定为 issue 后才可退回；响应每次回复一条；user 拍板/回复节点由 user 自行确认完毕防误判推进 | superseded-by ISS-08 |
| ISS-04 | 09-03 22:15 | dsh·6f4e | 分阶段判断：非「open 都 resolved」，每阶段有自己的标准；文档产出阶段 open 变「已处理」而非 resolved | superseded-by ISS-08 |
| ISS-05 | 09-07 23:23 | zc·Wayfinder | 重设计：open→pending-confirm 三出口（澄清解释→resolved／确认问题→待修复／回 open）；修复阶段待修复⇄已修复（重新 review，成功 resolved、失败回待修复）；澄清/修复分别用计数器记 reopen/revise 次数 | superseded-by ISS-08 |
| ISS-06 | 09-08 00:44 | zc·Wayfinder | issue 状态机复述校验通过：「看起来没问题，继续原型」 | superseded-by ISS-08 |
| ISS-07 | 09-10（修复轮） | 文档·repo-doc-generations §2 | 实现漂移：issues.go v2 另加 overridden 态与多条非设计通路（修复轮混入，非用户拍板） | superseded-by ISS-08 |
| ISS-08 | 09-11 22:50~23:15 | zc·ask-matt | 批注定稿：overridden 不存在（该动作置 resolved）；pending-confirm 是上层 node 阶段非动作，导向三动作 resolved/to-fix/open；confirm 是可拆三事件的节点位非事件；未描述的节点→事件通路不存在；open→resolved 直达限场景可增；override 开 pending-confirm+fixed 两格（见 OVR-06）；pending-confirm×clarified、fixed×declare-fixed 自循环=非法（第三轮批注，依据「agent 只被调用时响应」普适约束）；事件新名 settled 系（clarified/settle/qualify/declare-fixed 等）经三轮定稿，图文词汇已统一并内嵌退役词 grep 自检 | 现行权威 |

## 2. override / escalate 与上升判定

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| OVR-01 | 08-14 19:24 | dsh·4228 | user 参与不止强制介入一种方式：可参与讨论、必要时阶段内直接确认，减少返工 | 现行权威 |
| OVR-02 | 09-05 13:56~14:06 | test-flow·zc-L | 冲突上升（澄清无法达成一致）定为冲突响应机制；用户的结论追加在该 issue 中 | 现行权威 |
| OVR-03 | 09-06 20:07 | zc·Wayfinder | 缺漏#4：冲突无法 agree 引入 user 仲裁（后期质量高再考虑评委会投票）；送审需附正反方意见 | 现行权威 |
| OVR-04 | 09-07 10:40~10:51 | zc·Wayfinder | 上升通知规程：门禁汇总结果+待评审内容由责任人报给用户；上升附正反双方结论汇总；冲突双方拉会话对峙记现状+改进项 | 现行权威 |
| OVR-05 | 09-11 22:50~23:15 | zc·ask-matt | 上升判定归 reexec（返工 x 次仍不解决→上升），届时对应 issue 状态应为 fixed；overridden 动作置 resolved；override 只开 pending-confirm | superseded-by OVR-06 |
| OVR-06 | 09-11 23:4x~9-12（第三轮批注，zc·ask-matt） | zc·ask-matt | ①fixed 行开 override 格→resolved（「反复修复不成功→上升」正式采纳：判定主体=re-exec 侧按 revise 计数，判定时点 issue=fixed，User 仲裁→resolved 不回 pending-confirm）；②澄清期上升维持 issue 内 loop≥5 自动（-C7）——上升判定分阶段定案；③open×settle 开格限场景=提评者撤回（操作者=提评者，实现层需开口）；④pending-confirm×clarified、fixed×declare-fixed 自循环=**非法**（普适约束：除 user 外皆 agent 只在被调用时响应，无触发者即非法；补充信息合法路径=复核者 reopen）；⑤reply 幂等自循环（D6）按同一判定法重审，复核中评审者追问式 reply 是否允许待裁 | 现行权威 |

## 3. reviewState 词汇演进（v3 药丸 → v4 → v5 per-agent，完整原话链见 testflow-sessions-decisions.md 专节）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| REV-01 | 08-14 19:05 | dsh·4228 | 各角色评审意见留存、评审状态可监测（最早雏形） | superseded-by REV-07 |
| REV-02 | 09-05 20:21 | test-flow·zc-L | v3 提案：可配动作 state=entry/exec/review/replied；审批状态和结果=-/collect/reviewed/agree/approve/reject | superseded-by REV-07 |
| REV-03 | 09-05 20:34 | test-flow·zc-L | v3 拍板：rework 语义过窄（多为澄清）；reviewState 增「-」未评审；审批类节点 exec 不配置，打回统一回 review；capability-model-final.md 重写为两层属性状态机 | superseded-by REV-07 |
| REV-04 | 09-07 13:43~13:45 | zc·Wayfinder | 药丸 v3 批评：pass/block 若等 reject 就展示 reject；要看 open 问题的澄清状态而非 role 状态栏；state 内小状态机需实现 | superseded-by REV-07 |
| REV-05 | 09-07 23:33~23:48 | zc·Wayfinder | v4：未表态定名 reviewing；approved/rejected 是门禁状态；barrier 不存储；随后用户亲口否定 v4 链「'-'→reviewing→clarifying⇄confirming→agreed 这个是错的」 | superseded-by REV-07 |
| REV-06 | 09-08 00:11~00:26 | zc·Wayfinder | v5 定稿：review state 是每 agent 轮转（1/10）各自上报 agree/unagree；approved/rejected 由 collect 汇总给定；单字段 JSON 存全部 review agent 结果；Review 复用=整体复审不合理 | superseded-by REV-07 |
| REV-07 | 09-11 23:08 | zc·ask-matt | 终稿确认：「每个评审位独立 reviewing/agree/unagree，clarifying/confirming/agreed 不再作为存储态——这个我认可」 | 现行权威 |
| REV-08 | 09-12 | zc·ask-matt | `agreed` 词属裁定：**collect 层内部专用词**（全评审位结论达成一致时 collect 的判定输出，触发进入待确认/拍板等待），**不应扩大**到 collect 之外；不作 reviewState 存储态。API 工具词与标准领域词的 gap（wb_reply/wb_resolve ↔ clarified/settle/qualify 等）修复定序：**先纠正现有文档（保留事件命名提案式映射）→ 代码修复 → 归档旧映射、替换为新代码映射**；先完成可执行 D 类，其余记 todo | 现行权威（REV-07 细化） |

## 4. node 编排模型（trigger 词表 → 两层属性状态机）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| ORCH-01 | 09-03 22:08 | dsh·6f4e | trigger 枚举封闭：「接受」 | superseded-by ORCH-05 |
| ORCH-02 | 09-04 18:56~20:01 | dsh·6f4e | trigger 语义不明确→node 标准 hook（before entry/entry/after entry/before transform/exec/reject）；hook 挂 node 配 json map；action 可与 capability 解耦待推演 | superseded-by ORCH-05 |
| ORCH-03 | 09-05 01:46~13:17 | test-flow·zc-L | 编排骨架 entry→exec→review（响应→复核循环至准出或打回）；hook 本质还是 trigger，后定义为扩展：[before\|after]\<trigger-name\> 才是 hook；回退判定=程序+通用规则+数据（reject 结论协议） | superseded-by ORCH-05 |
| ORCH-04 | 09-05 18:21 | test-flow·zc-L | 去掉 trigger 语义直接统一 state（获认可）；re-entry 并入 entry、re-exec 并入 exec（节点计数位区分首次/再次） | 现行权威（方向） |
| ORCH-05 | 09-05 20:21~20:44 | test-flow·zc-L | 两层属性状态机定稿：可配置动作 state=entry/exec/review/respond；审批状态和结果=-/collect/reviewed/agree/approve/reject（不可配置）；after-exec hook 方案标注废弃或留后期扩展（通知/同步用） | 现行权威 |

## 5. 「一个 state 一个动作」与 respond 挂载

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| ACT-01 | 08-24 06:34~06:53 | dsh·5806 | 开发节点统一含多动作（开发/自测/响应 CR）；能力做动作拆分、按 mytask 确定动作；动作以 JSON 存 capabilities 不建表 | superseded-by ACT-04（动作拆分思想保留，多动作挂载被约束取代） |
| ACT-02 | 09-05 20:40 | test-flow·zc-L | 「dev：respond(clarify+fix_on_review) 是错误的，respond 只有 fix_on_review」 | superseded-by ACT-04 |
| ACT-03 | 09-07 01:51 | zc·Wayfinder | R2：维持 role.capability.review 而非全局公共 skill（解耦/维护/ctx 三理由）；仅迎宾等极简动作考虑公共 skill | 现行权威 |
| ACT-04 | 09-07 03:14~03:36 | zc·Wayfinder | 「目前阶段是一个 state 只挂一个 skill」；「之前已经严正声明过，一个 state 只能挂一个动作，怎么立即就重犯了？」（普适约束，见铁律#1） | 现行权威 |
| ACT-05 | —（commit 侧） | 文档·repo-doc-generations §2.3 | 647a493 R1 双动作 undermined；41f7c06 CR 异化单挂；4b602aa 曾回退内嵌 CR 又再定；最终口径见 capability-actions-final 轨道规则 | 现行权威（文档口径） |

## 6. CR 内嵌 vs 独立评审节点（含评审节点去留；两次方向相反拍板，最终=双轨制）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| CR-01 | 08-15 10:54~10:58 | dsh·4228 | rd-mr/fe-mr 评审节点裁掉、评审挪进开发；同时设计与评审为独立节点（用例/技术设计同此推广） | superseded-by CR-05 |
| CR-02 | 08-24 06:34 | dsh·5806 | RD 开发/自测/CR 三合一（CR 内嵌开发最早裁决） | superseded-by CR-05 |
| CR-03 | 09-05 19:34 | test-flow·zc-L | cr 挂 after-exec（trick：不拆新节点、不增节点内阶段）；「hook 和编排不要混为一谈」 | superseded-by CR-05 |
| CR-04 | 09-05 19:50 | test-flow·zc-L | 区分节点内评审（评本节点产物）与跨节点评审（评上一节点产物）；节点内评审永远不会也不能输出 reject；reject 语义=回退上个节点 | 现行权威（拓扑原则） |
| CR-05 | 09-05 20:05~20:44 | test-flow·zc-L | 「cr 可以考虑挂在 collect，而不通过 trick」；exec 拆 exec+review；文档侧定稿：『取消评审节点』提案撤回，**双轨制**=节点内 CR 异化（挂 collect/respond）+跨节点评审节点（repo-doc-generations §2.5） | 现行权威（最终版） |
| CR-06 | 09-07 04:23 | zc·Wayfinder | 「编写文档节点没有 review 这应该是共识的结论」；技术设计 cr 不参与复审 | 现行权威 |
| CR-07 | 09-07 11:17 | zc·Wayfinder | D2 立项评审：暂不评审，用户的结论就是最终结论 | 现行权威 |

## 7. scope 体系（含 task 三值化回滚）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| SCP-01 | 08-18 13:20~13:24 | dsh·cc0e | scope 投影：业务实例不含 fe 则 fe 节点与 reviewer 均不含 fe（确认）；纠正 m8 文档缺 scope 字段 | 现行权威（投影语义） |
| SCP-02 | 09-07 11:17~11:49 | zc·Wayfinder | 「按需」=根据 scope 判断、四流程都要 check；business scope 非全量则不渲染 QA 验收节点；scope=full\|fe\|rd 三者之一；tech 待定（full\|rd?） | superseded-by SCP-05 |
| SCP-03 | 09-07 11:55→12:06 | zc·Wayfinder | task 三流程动态渲染联调节点、scope 都增加 full → **同日回滚**：「回滚我刚才对 task 流程的改造……保持指定单个开发者，没有联调，去掉测试」 | superseded-by SCP-05（回滚生效） |
| SCP-04 | 09-07 12:10 | zc·Wayfinder | B2 RD 技术设计 scope: rd 错，应当 rd\|full，举一反三全流程 sweep | 现行权威（枚举值；机制见 SCP-05） |
| SCP-05 | —（commit 侧） | 文档·repo-doc-generations §2.4 | task 三值化(e825d76)→回滚(5c25144)→表达式制(3500f77)；最终=渲染集合表达式 | 现行权威（最终形态） |

## 8. QA 验收语义（用例节点与 retest）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| QA-01 | 09-04 02:05 | dsh·6f4e | 用例不再有单独编写阶段，去掉该节点及审批节点 | superseded-by QA-02 |
| QA-02 | 09-04 15:35~15:51 | dsh·6f4e | 终版：保留用例设计但不设单独节点，与验收合并「设计完就直接测试，合并在验收文档中」；qa-testcase-review-template 废弃 | 现行权威 |
| QA-03 | 09-05 01:22 | test-flow·zc-L | 「考虑取消验收节点 fe 和 rd 的响应能力」（可举一反三到产出+审批节点） | superseded-by QA-04 |
| QA-04 | 09-07 04:23 | zc·Wayfinder | QA 验收 exec 重入=针对重新提交的代码**复验重测**（非补充缺陷），确认是否修复并更新之前的缺陷文档；同日 10:40 要求定英文名（见 NAM-06） | 现行权威 |
| QA-05 | 09-05 03:46 | test-flow·zc-L | 轮次制（表态→澄清→响应→复核）统一适用 QA 验收：「可」 | 现行权威 |

## 9. collect / agree 收口判定与门禁拦截

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| COL-01 | 08-24 21:51 | dsh·5806 | 并行评审 join：可同名绝非同一节点（负责人不同）；等全表态 barrier，冲突由 user 拍板避免返工 | superseded-by COL-05（barrier 后被裁） |
| COL-02 | 09-05 17:52 | test-flow·zc-L | complete 存在（与 reject/approve 不同）；plugin 基于 agree 中是否有 open 判 reject/approve；多审批者收集齐再终判；审批 exec 配 skills map | superseded-by COL-05 |
| COL-03 | 09-05 20:21 | test-flow·zc-L | collect 收集全部 reviewer 表态，有 open 且未 agree 的 issue 则推进 replied；agree/collect/approve 不可配置 | superseded-by COL-05 |
| COL-04 | 09-07 23:23~23:33 | zc·Wayfinder | collect 判三子状态（未表态/unagree/agree）；pass/reject 由是否仍有 open 工单解决；全表态即推进；barrier 是瞬间状态不存储 | superseded-by COL-05 |
| COL-05 | 09-08 00:11 | zc·Wayfinder | collect 收集全部信息后给定 approved/rejected：approved 进下节点，rejected 回上节点，仅此而已 | 现行权威 |
| COL-06 | 09-10 03:35~03:41 | zc·QA 全分支会话 | 拦截最小原则：应有产物未提交不允许过 exec，「不应该局限在 approve 阶段拦截」「要在最初的地方拦截，每个环节只关心自己的事情」（approve 不应判断投票/澄清遗漏，违反最小原则；见铁律#5） | 现行权威 |

## 10. 通知可靠性（outbox）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| NTF-01 | 08-14 18:54 | dsh·4228 | 通知=IM 收件箱，支持项目筛选、任务名筛选 | 现行权威 |
| NTF-02 | 08-16 00:49~00:57 | dsh·4228 | 待复核超时提醒+轮询：user 超时延迟 1h、agent 5min、轮询 5min | 现行权威 |
| NTF-03 | 09-07 10:40~10:51 | zc·Wayfinder | 门禁通知是通用能力：汇总结果+待评审内容给用户、由责任人报送；上升附正反结论 | 现行权威 |
| NTF-04 | 09-10 00:40 | zc·CR 循环会话 | outbox 方案拍板：「投递可选消息事务表，写表+推送，定时任务扫消息表中失败的」 | 现行权威（方案） |
| NTF-05 | 09-10 01:27 | zc·CR 循环会话 | 修订 NTF-04 配套重试策略（推翻 assistant 侧 retry5→dead 终态）：「没有 dead，必须全部通知到，无限重试，都是内部系统，给我一个人用的」（见铁律#6） | 现行权威 |

## 11. 产物入库约束（-K）

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| ART-01 | 09-10 02:02 | zc·req-003 会话 | 技术评审产物应是可点击进入的文档视图（可查看/发表 comment），不能是外链 | 现行权威 |
| ART-02 | 09-10 03:52 | zc·分支策略会话 | 提交设计文档直接存系统内，点开是飞书式交互（行/块/文档评论）——「重大的实现遗漏」；技术评审必须看到上一节点产物；所有评审挂在文档上（铁律#4 首见） | 现行权威 |
| ART-03 | 09-10 20:51 | zc·QA 漏测复盘会话 | 「挂载的产物文档，是不能是外链的，必须是系统内部的入库的产物」req-004 复犯追责 | 现行权威（重申） |
| ART-04 | 09-11 01:14 | zc·QA 漏测复盘会话 | 老代码 ~/tmp_1/test-flow/workbench-plugin/ 有全套交互（行级多选/块级/文档级评论，接近飞书）；约束固化进 spec 注记 -K（文档·zcode-sessions 疑义7） | 现行权威 |

## 12. 评论/锚点交互层级

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| CMT-01 | 08-14 20:44~21:01 | dsh·4228 | 否决章节级评论；文档级在正文最底部可回复；已解决问题可折叠 | superseded-by CMT-05（折叠与文档级仍有效，层级被收窄） |
| CMT-02 | 08-14 21:24~21:32 | dsh·4228 | 行级锚点权威规则：首点高亮即评论该行；再点其他行=框选范围；重复点首行=取消；跨行评论只在末行下展示一次（一版逐条实现） | 现行权威（交互规则） |
| CMT-03 | 08-16 00:26~00:30 | dsh·4228 | 解决评论完整设计：多评论者、复核、本地 cr comment 由开发者修复 | superseded-by CMT-05 |
| CMT-04 | 一版·client.js/host.js | 一版代码 | 四级锚点（block/line/range/document）+评论卡状态/轮次/超时四维体系 | superseded-by CMT-05 |
| CMT-05 | 09-11 01:32 | zc·QA 漏测复盘会话 | OPT-005：「按飞书的来……只支持划词+文档级别评论」（四级收窄两级）；OPT-004 所有节点评论全闭环才能通过（collect 职责，「被反复提了十几遍」）；BUG-016 通过到下节点自动进 done，该阶段禁止挂载 | 现行权威 |
| CMT-06 | —（注记侧） | 文档·spec 注记 -L3 / inventory B-36 | line/block 锚后端 400 废弃，range/document 两级定稿（单行=range(n,n)） | 现行权威（固化） |

## 13. 版本/需求状态机

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| VER-01 | 08-14 18:38 | dsh·4228 | 版本维度（v1/v1.1/待分配）+需求池四态（待启动/进行中/已完成/已取消） | 现行权威 |
| VER-02 | 08-15 02:36~02:41 | dsh·4228 | 版本四态（待启动/进行中/已发布/已废弃）；单向推进不允许回退；「废弃后，版本下的需求都应当推进为已取消」 | superseded-by VER-03 |
| VER-03 | 08-15 02:57 | dsh·4228 | 当场纠正：「我们不支持联动取消，之前已经确认了」；有需求不可废弃需先移出版本；提示不写「v1 不可恢复」 | 现行权威 |
| VER-04 | 08-15 02:59 | dsh·4228 | 功能只有废弃，不应当支持删除 | 现行权威 |
| VER-05 | 09-09 01:22 | zc·继续会话 | 调整（裁决 A）：已开始的需求暂时禁止废弃（未来再说）；未开始直接下掉标记；未推进的可删，删除后需求全部取消关联、flow 回待分派池 | 现行权威 |

## 14. 数据模型与工程约束

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| DM-01 | 08-15 09:25 / 08-16 15:39 | dsh·4228 | 存储=SQLite（驳回 JSON/diff 论），需明确模型和语义 | superseded-by DM-08 |
| DM-02 | 08-16 15:49 | dsh·4228 | 不使用外键（后续调整结构过于复杂）；工作模板固化保稳定 | 现行权威（原则，沿用至 MySQL 规） |
| DM-03 | 08-16 16:28~18:41 | dsh·4228/a108ae54 | 字段类型合理化（id 用 int 非 text）；关联只认 id、命名一致（role_id）、梳理表间 1:n | 现行权威 |
| DM-04 | 08-16 19:04 | dsh·a108ae54 | CAPABILITIES 与 DOCUMENT_TEMPLATES 合并（v0.17 起点） | 现行权威（方向；表命名后演 see DM-07） |
| DM-05 | 08-17 12:24~13:22 | dsh·4228 | 去快照表；flow_node_reviewers 并入 flow_nodes；非必要不允许 null；每表 created_at/updated_at | 现行权威（原则） |
| DM-06 | 08-17 18:24 | dsh·4228 | 裁 reviewer_focus/inputs（reviewer 能力写在角色能力上；task 上下文已含相关文档） | 现行权威 |
| DM-07 | 09-05 14:50~16:32 | test-flow·zc-L | 新增 project 维度（不同 project 绑定不同工作区）；flow_template/flow_template_nodes 更名 flow_defs/flow_def_nodes | 现行权威 |
| DM-08 | 09-08 02:04 | zc·Wayfinder | 数据库 MySQL；技术栈 Go；HTTP Gin；前端本期实现在 web（推翻 DM-01 SQLite，随 MVP 重写换栈） | 现行权威 |
| DM-09 | 09-12 | zc·ask-matt | **测试端口范围定死 6000-6099**：全部测试用端口（QA 服务/testpage/后续新增）一律落此区间，不得外溢；现有 18081/18090 已迁改为 6081/6090 | 现行权威 |
| DM-09 | 09-08 02:14 | zc·Wayfinder | 技术规格硬约束：可执行完整 DDL、每表 created_at+updated_at 毫秒精度、主键 BIGINT UNSIGNED AUTO_INCREMENT、不建物理外键、乐观锁=updated_at、不用 MySQL JSON 类型、proto 为真相源 | 现行权威 |

## 15. ProductType 与产物路径

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| PT-01 | 08-24 07:07~07:19 | dsh·5806 | ProductType 标准枚举：能力跨 4 流程复用，how 按 template_node_id 找产物会破坏复用（「很严重的问题」）；禁「node1」位置引用 | 现行权威 |
| PT-02 | 08-26 18:38 | dsh·9528 | how 中路径模板化：docs/{{req-version}}/{{req-name}}/rd/design.md，技术可实施（插件 js，python 仅示例） | 现行权威（形态） |
| PT-03 | 09-03 22:45 | dsh·6f4e | 路径真相源：本期先简单来，不写 skill 暂不支持 template（收窄 PT-02 本期范围） | 现行权威（本期范围） |
| PT-04 | 09-05 03:22 | test-flow·zc-L | brd 路径：docs/{{req-version}}/brd/brd.md | 现行权威 |

## 16. skill 固化/废弃与 how 演进

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| SK-01 | 09-03 00:03 | dsh·6f4e | skill 固化原则：能抽象成固定 skill 的固化提高稳定性；不必要/无法生成则放弃，须经推演确定 | 现行权威 |
| SK-02 | 09-03 13:52 | dsh·6f4e | 逐个裁决（一）：grilling/wait-what=底层不直接用；teach/to-questionnaire 废弃；handoff=程序框架在 mytask 用；五段式骨架可抽象为 how 标准但名词更抽象 | superseded-by SK-04 |
| SK-03 | 09-03 21:14~21:24 | dsh·6f4e | 逐个裁决（二）：triage 归 QA 缺陷管理；retro=cr 用；research=调研员；prototype=产品对齐（有前端）；setup-pre-commit 仅初始化类；git-guardrails/scaffold-exercises 等不启用 | 现行权威（该批归属） |
| SK-04 | 09-05 13:34 | test-flow·zc-L | **how 骨架废除**（重大简化）：「每个 action 直接挂一个 skill，skill 内部可以按需调用其他 skill……极大降低框架复杂度」 | 现行权威 |
| SK-05 | 09-03 22:57 | dsh·6f4e | 能力模型层次：skill≠能力，Capability→actions→how 配置 skills；统一用 trigger | superseded-by SK-04（trigger 部分另见 ORCH-04/05） |
| SK-06 | 09-09 19:57 | zc·mattpocock 会话 | 开发期高强度使用 mattpocock/skills，嵌入现有流程并补充 | 现行权威 |
| SK-07 | 09-10 18:41~18:57 | zc·QA 漏测复盘会话 | QA 教训全局化（换项目不得再犯）：skill 保证测例/测试时必加载；历史教训写语义库引用、不写入 skill 本体；结合 flow 的 qa 验收环节做通用能力（完整测例项目+产出可被 diagnosing-bugs 消费） | 现行权威 |

## 17. 开发链路、编排调度与会话挂载

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| DEV-01 | 08-19 12:10 | dsh·4228 | 先不做 session 绑定（任务都由主 agent 派发） | superseded-by DEV-05 |
| DEV-02 | 08-27 07:24 | dsh·6f4e | 需要 plugin 创建 session、绑定 session 和 flow 的能力（翻转 DEV-01） | superseded-by DEV-05 |
| DEV-03 | 09-03 20:44~20:53 | dsh·6f4e | 编排并发采用 B 并扩展（可设并发度）；开发链路定版：spec→[to-tickets] 拆票+/goal→[implement-spec] 编排调度→tdd/implement 工人执行 | superseded-by NAM-01（命名部分；链路结构现行） |
| DEV-04 | 09-04 20:25~21:46 | dsh·6f4e | 每条 flow 绑定一个会话；subagent 直呼方案实测通过（team preset 直启，先测后拍）；session 要有名字非 id；开场白给 user 看 | superseded-by DEV-05 |
| DEV-05 | 09-05 19:42；09-06 20:29~21:41 | test-flow·zc-L；zc·Wayfinder | 全 agent 通过→按需唤醒 user 终审（未配拍板直接通过）；user_gate 本期人工卡点、长期看需求依赖；并发度=正在工作的需求数（非进行中数）；同版本依赖自动分层树，flow 实例含依赖字段 | 现行权威 |
| DEV-06 | 09-03 19:11~19:19 | dsh·6f4e | subagent 沟通形态非弹窗、类 wayfinder 直呼；会话挂 flow 节点，插件可点节点查看 session 历史 | 现行权威 |

## 18. 测试制度

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| TST-01 | 09-10 01:56 | zc·req-003 会话 | 测试数据必须按流程逐步推进建成，禁止直接建终态节点（「没走正常推进流程，中间很可能跳过了一些问题」）；先记录缺陷 | 现行权威 |
| TST-02 | 09-10 02:05~03:22 | zc·QA 全分支会话 | QA 制度：扮演 qa 编写全接口测例，建真实 flow 全节点全分支（拒绝/回退/重新 review）走一遍并校验各阶段产物；起 subagent diagnosing-bugs 响应 bug；测试脚本固化存 qa 目录复用 | 现行权威 |
| TST-03 | 09-10 04:17~04:23 | zc·QA 全分支会话 | 缺陷台账 docs/v1.0/qa/qa-defects.md，user 发现与 agent 发现分记；修复流程四步（更新技术/架构文档→diagnosing-bugs 修复→cr→补测例回归，有缺陷回步 2） | 现行权威 |
| TST-04 | 09-11 01:32 | zc·QA 漏测复盘会话 | 严重性认定：状态推进无拦截（exec 跳过 collect/replied/approve）、单测空实现不算覆盖，均为严重问题 | 现行权威 |

## 19. UI 原型裁决

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| UI-01 | 09-07 12:44~13:29 | zc·Wayfinder | 原型启动：P1 上下展示（参考一版前端）；右侧四分栏吸收；产物分栏必须有（阅读器入口）；scope 动态渲染、被裁节点不占位；「本期没说的就先不做，不要太自由发挥」 | 现行权威 |
| UI-02 | 09-08 00:53~01:21 | zc·Wayfinder | 面板裁决：p2/p3 暂缓；p5/p4 加跳转按钮；p1/p6 可、p6 加调整按钮；tag=flow 类别标签（如【需求任务】） | 现行权威 |
| UI-03 | 09-10 01:55~03:57 | zc·CR 循环/QA 会话 | 默认工作入口=看板；「并发怎么没了？」（并行展示缺失）；节点颜色是重要语义，完成与进行中不可同色 | 现行权威 |

## 20. 事件/动作命名演进

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| NAM-01 | 09-03 21:39~21:59 | dsh·6f4e | to-spec→to-prd（implement-spec 相应改名 implement-techdesign）；ticket 译「工单」统一 | 现行权威 |
| NAM-02 | 09-05 02:32~02:39 | test-flow·zc-L | 表态词批评 closing/transform/task_stale；表态通过定名 **approve** | 现行权威 |
| NAM-03 | 09-05 03:22~03:46 | test-flow·zc-L | to-brd 定名+路径 brd/brd.md；reviewed 定名；用户验收 reject 打回路由（user→qa→rd/fe）为后端泛用接口不写死 node | 现行权威 |
| NAM-04 | 09-05 14:18~18:21 | test-flow·zc-L | to-brd 为公共 skill 不挂 pm；re-entry 并入 entry、re-exec 并入 exec（术语废弃，仅存计数位） | 现行权威 |
| NAM-05 | 09-05 20:34~20:44 | test-flow·zc-L | **rework 废弃**→respond（澄清容器，dev 上只挂 fix_on_review）；可配 state 定名 entry/exec/review/respond | 现行权威 |
| NAM-06 | 09-07 10:40 | zc·Wayfinder | 「acceptance-test.复验重测 给个英文名吧，后面 skills 也会参考生成对应英文的 skill」（英文名定名 retest 之说未见于取证原话，见疑义#6） | 现行权威 |
| NAM-07 | 09-11 22:50 | zc·ask-matt | **overridden 废除**（该动作置 resolved）；「非预期的你改动了已有结论」追责 | 现行权威 |
| NAM-08 | 09-09 22:39 | dsh·5087 | 实现 to-brd，且 to-prd 参考 to-brd 章节，四文档互相依赖、Focus 对应章节 | 现行权威 |
| NAM-09 | 09-12 | zc·ask-matt | **API 词 ↔ 领域词 gap 修复立项（todo）**：现行 API 工具词（wb_reply/wb_resolve path/wb_confirm_review/wb_reopen 等）与标准领域事件词（clarified/settle/qualify/reopen/declare-fixed/confirm/revise/override）存在 gap——修复顺序：先纠正现有文档并保留映射表（state-tables-draft 事件命名提案式）→ 代码契约修复（改端点/参数名，过 buf breaking 闸门重生成）→ 归档旧映射、替换为新代码映射。涉 D2（agreed 词面）与 D5 类同批处理 | todo（待代码修复工单） |

## 21. test-flow→dsh-flow 迁移与 MVP 定稿

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| MIG-01 | 09-03 23:00 | dsh·6f4e | 目标重定义：将 mvp 版本的设计确定下来；不支持能力推演合并进主文档 | superseded-by MIG-07（仅「不合并」禁令部分，目标定义部分为历史完成时态） |
| MIG-02 | 09-06 15:34~16:05 | zc·迁移会话；dsh·92c1 | 工作目录迁至 ~/workdir/dsh-flow（代码已 copy，只调挂载/配置） | 现行权威 |
| MIG-03 | 09-07 02:24 | zc·Wayfinder | 统一修正记忆锚点，test-flow 迁移到 dsh-flow | 现行权威 |
| MIG-04 | 09-08 01:54~01:59 | zc·Wayfinder | MVP 设计定稿「完结吧」；历史实现与文档归档 .archive，准备重新实现（此前 UI/流程结论以 .archive 为历史底稿） | 现行权威 |
| MIG-05 | 09-09 01:22 | zc·继续会话 | 一期先做 web 页面，未来考虑嵌入 dsh（调整目标） | 现行权威 |
| MIG-06 | 09-10 01:28 | zc·Skill 权威化会话 | 开工令：「可以正式开工了，主线的 plan 已经完成，基于现在 main 直接开工」 | 现行权威 |
| MIG-07 | 09-12 | zc·ask-matt | **arch_design 合并为单一权威**：「合并吧，保留一份权威的 arch_design」——docs-tmp 五件套现行设计（v5 口径）并入 arch_design.md，docs-tmp 原件转推演底稿归档、动线改指 arch_design；含三板引用核对。推翻 MIG-01 的「不支持合并」（后拍覆盖先拍）。执行排序：D1 返工落地后开专项（合并+词汇刷新一并做，消除 §7-§9 pkg-16 残留） | 现行权威 |
| MIG-08 | 09-12 | zc·ask-matt | **架构定位重申**：「dsh 对那边是工具，对你就是全部，项目的目的就是嵌入 dsh，我们就是用 dsh 作为基础」——wbflow = DSH 之上的编排/工作台，终态 DSH 插件（一期独立 Web=过渡壳）；LLM 通路=**经 DSH 网关调底层大模型**（baseURL 默认本地 DSH，模型路由/账号/鉴权归 DSH），novel 调研只 copy 传输机制实证件、不照搬其「外部端点直连」定位；交互场景走 DSH 会话通道不变 | 现行权威（重申 09-10 -L4「经 DSH 调用」本义） |

## 22. 文档纪律、退役机制与响应协议

| 编号 | 日期时间 | 来源 | 拍板内容 | 状态 |
|---|---|---|---|---|
| DOC-01 | 08-21 14:05~14:28 | dsh·5806 | 目录重组定版：docs/\<版本号\>/\<req-id-slug\>/（prd/rd/fe/qa）；历史设计归 v1.0；V1.2 裁掉 | 现行权威 |
| DOC-02 | 08-24 21:58 | dsh·5806 | docs-tmp 是最近设计，比 seed.json 更权威 | superseded-by DOC-04 |
| DOC-03 | 09-06 22:58~09-07 02:04 | zc·Wayfinder | 自查清单每次遗漏复盘更新；域一致性必须：任何结论都要校验对现有代码/文档的影响，作为响应机制检查阶段（铁律#8 配套） | 现行权威 |
| DOC-04 | 09-07 03:17 | zc·Wayfinder | 检查历史工单与文档结论，已过时的标注废弃（铁律#7 首见） | 现行权威 |
| DOC-05 | 09-07 01:46~03:58 | zc·Wayfinder | 罗盘制度（响应请求先分析+列 todo+checklist 防遗漏）；推演/响应两协议升全局：合并 ~/.dsh/AGENTS.md 并软链 ~/.zcode/AGENTS.md，平级合并不 append、保精简 | 现行权威 |
| DOC-06 | 09-11 21:18~23:28 | zc·ask-matt | 流程断链复盘（wayfinder→to-spec→implement 跳步）；废弃归档+老旧信息标注过时；全面复盘历史文档/代码/会话捋时间线定先后（本任务起点） | 现行权威 |

---

## 附一：一版交互参照与履约（指针，不复制正文）

- 参照拍板：09-07 12:59 zc·Wayfinder「先看一下当前项目的前端代码作为参考……有成型的交互了」（当时 cwd=test-flow，即一版 workbench-plugin/client.js）；09-11 01:14 重申老代码路径（见 ART-04）。
- 履约对照：52 项=已履约 33 / 部分履约 11 / 未履约 8；TOP 缺口（横向 SVG 泳道流程图、已解决评论折叠、阅读器 md 渲染、筛选药丸、药丸信息密度等 10 项）与「fe 超出一版勿回退」清单，见 testflow-interaction-inventory.md §B/§C。
- 一版 override 强权已被 fe 冲突裁决卡（wb_adjudicate）取代，属刻意重构（inventory B-29）。

## 附二：铁律级普适约束专节

| # | 铁律 | 首次出现 | 重申 |
|---|---|---|---|
| 1 | 一个 state 只挂一个动作 | 09-07 03:14 zc·Wayfinder（「目前阶段是一个 state 只挂一个 skill」） | 09-07 03:36 严正声明追重犯；文档·repo-doc §2.3（647a493 R1 undermined、41f7c06 单挂） |
| 2 | 文档不自评 | 09-06 03:04 zc·Wayfinder（「明确不自审，其他文档的评审同样不自评」） | 09-06 20:23 节点5/10 违例举一反三；09-07 04:23「编写文档节点没有 review 共识」 |
| 3 | 除 user 外皆 agent，只在被调用时响应 | 09-11 23:22 zc·ask-matt（「不要拟人化……不会自己在未调用的情况自主响应」） | 暂无（唯一来源） |
| 4 | 产物必须系统内入库、禁止外链+飞书式评论 | 09-10 03:52 zc·分支策略会话 | 09-10 20:51 req-004 复犯；09-11 01:14；固化 spec 注记 -K |
| 5 | 在最初的地方拦截（最小原则） | 09-10 03:35 zc·QA 全分支会话 | 09-10 03:41（「每个环节只关心自己的事情……违反了最小原则」） |
| 6 | outbox 无 dead、无限重试 | 09-10 00:40 zc·CR 循环会话（事务表方案） | 09-10 01:27 推翻 dead 终态 |
| 7 | 术语退役词 grep 清零 | 09-07 03:17 zc·Wayfinder（过时结论标注废弃） | 09-08 00:26（「逐文档校验，不要 grep」）；文档·repo-doc §1（09-11 清 18 处 v4 残留，71476da） |
| 8 | 约束级变更全文件 sweep | 09-07 23:33 zc·Wayfinder（「如果我们达成一致就都需要修改」） | 09-08 00:26（「整个项目，逐文档校验……不要遗漏」）；09-07 02:04 域一致性强制度 |

## 附三：疑义（本表综合者判断，需用户裁决的残留项）

1. **来源去重判定**：dsh 侧 test-flow 期目录（migrate-backup）与 dsh-flow 目录为同批会话镜像（session-42289ff0 等同 ID 双现），本表统一记 `dsh·<短名>` 未双计；如按会话数口径统计会得不同条数。
2. **dsh 侧间接证据不计拍板**：reviewState/override 词汇在 dsh 侧仅存于主 agent 写的 subagent 派单（08-19/09-08 快照），按取证规则不作为拍板，本表相关权威链全部取自 zcode 侧用户原话。
3. **时间戳与单字拍板**：zc 09-10 01:28「开工令」在钟序上早于同日 00:01/00:21 条目（跨会话并存，不构成冲突）；「B」「D4 去掉这个字段」（09-09 23:47）等选项式拍板语义在 assistant 侧，无法从用户原话还原，未入表。
4. **SQLite→MySQL 定性**：按「同题后拍覆盖」判 MySQL 现行（DM-08），但 08-15 SQLite 属一版实现语境——是「被推翻」还是「随 MVP 重写整体换栈」用户未明言，需裁决口径。
5. **issue 状态机残留未决格**：09-11 23:15 用户自标「fixed×declare-fixed 这个怎么解释（待补充）」，即现行权威内部存在用户明示未决格。
6. **retest 名存疑**：用户原话只到「给个英文名吧」；「retest」这一具体名未见于两份会话取证（仅任务提示与仓库 skill 侧间接），需回查后续 skill 文档正名。
7. **CR 双轨制定稿先后**：会话侧最后相关拍板为 09-05 20:21（cr 改挂 collect）；文档侧「双轨制定稿」与 commit 4b602aa「回退内嵌 CR 又再定」无法从用户原话对齐先后，本表按 repo-doc §2.5 判定现行，建议用户复核。
8. **QA 验收响应能力方向相反对**：09-05 01:22「取消验收节点 fe/rd 响应能力」vs 09-07 04:23「复验重测保留」——本表按后拍 QA-04 现行（与 zcode 取证疑义4① 一致）。
9. **伪 user 消息不计**：sess_0c24fe6d / sess_872d714b 各自 09-09 23:29 首条消息为 handoff 粘贴/汇报注入（agent 口吻），未作拍板来源。
