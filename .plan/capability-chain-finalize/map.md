# 能力链路推演定稿 · Wayfinder 地图

> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿）**：下方 Decisions-so-far 各条中涉及 trigger 词汇、轮次制循环、评审能力名（arch-tech-review 等）、how 字段的表述已被 **v3 两层属性状态机 + 评审能力对象化** 取代——现行权威 = `dsh-flow/docs-tmp/capability-model-final.md` v3 及四份配套文档（提交 d6cefcf/77f9d96/41f7c06）；工单内时效标注见各 ticket「结论时效」节。

## Destination

**将 MVP 版本（v1.0）的设计确定下来**——docs-tmp 推演（产物可见性 v2 → 能力×动作 → 执行闭环 → wb_my_task 模型 → 四流程对齐）逐个敲定，形成**可供实现阶段直接照做的完整 MVP 设计**（docs-tmp 定稿层）。

> **2026-09-02 用户目标重定义**：**不支持将能力推演合并到主文档（arch_design.md）**——推演产出收敛在 docs-tmp 定稿层即可，arch_design 保持现状基线；术语统一（CONTEXT.md ↔ 既有文档）留到推演全部完成后最后统一执行。

## Notes

- **领域**：团队项目工作台（DSH 之上的「项目管理 + OA 流程 + 评审协作」系统）；当前阶段为流程/能力/产物链路设计推演，不涉及插件代码实现。
- **技能**：每次会话 consult `grilling`、`domain-modeling`；涉及外部事实用 `research`；模板/Excel 用 `xlsx`。
- **文档权威顺序**：docs-tmp（最新设计，正在敲定）→ arch_design.md（已 merge 权威基线）→ seed.json（实现数据，仅参考）。
- **引用规范**：模板一律用**章节名**，不用章节编号（模板骨架标注为准，见「模板骨架标注同步」票）。
- **scope 裁剪**：评审位裁剪按 arch_design §8.3 的 scope 语义推演（business/tech 有 scope，task/plan 无 scope 概念），不是简单标注"按需"。
- **能力×动作**：能力 = 节点内动作集合（action_id + 内嵌 how + trigger + when_to_use）；文档/代码类产物用 productType（受控枚举）定位，对话/评论类产物用 query（属性过滤）定位，均不依赖流程内节点位置。
- **一致性**：seed.json 现行 capabilities 仅 how（字符串数组）无结构化 inputs/actions，flow_template_nodes 无顶层 id——均待回填，为推演结果的目标形态而非现状。

## Decisions so far

- [技能复用评估](tickets/01-skill-reuse-assessment.md)（2026-08-27 关闭）：37 技能裁决完成（评估稿 `docs-tmp/skills-evaluation.md`；判定维度=技能→能力×动作 how 挂载）。回填本图的裁决：①**补建 respond_to_qa 动作**（QA 缺陷返工缺口）；②**dev+self_test 合并**单动作（trigger 链简化）；③4 模板加「动机场景（用户故事）」节+骨架标注同步；④code-review 单动作串行两子步；⑤接口面原则 4 条进 backend-design；⑥to-tickets/triage 挂 decide-cap.create_reqs、writing 三件套挂需求对齐环节、domain-modeling 挂设计类环节。已回填 03 / 05 票。**（另：技能评估子地图全 6 工单已闭合，how 编写规范 v2 定稿见 how-spec-v2.md）**
- [合并回 arch_design 策略](tickets/06-merge-arch-design.md)（2026-09-02 **Ruled out**）：**不支持合并**——推演产出收敛 docs-tmp 定稿层，arch_design 保持基线；合并为未来独立 effort（Out of scope）。
- [产物可见性 v2 待确认点](tickets/02-product-visibility-v2-confirm.md)（2026-09-02 关闭）：①inputs 引用=ProductType 最终形态（node_key/顶层 id 均被取代）；②**use_threads 默认 true**（显式 false 例外；不按动作类型推导——过度设计难测试）；③**sessionsByNode 废弃**（被最小返回集 relatedNodes 取代，防上下文爆炸；深排查可走按需查询）。联动：my-task-response-model / capability-artifact-analysis §3.3 标记同步。
- [执行闭环待确认项](tickets/04-execution-loop-confirm.md)（2026-09-02 关闭）：D1–D5 定稿（schedule 自检/直连 followup/最小返回集/three-way 校验/常驻 Leader）；**subagent 返回契约取消——数据层即返回**（成果落数据层，程序读之推进；唯一显式=心跳 ack/progress，完成由数据层推导，阻塞走数据层+兜底扫描）；**trigger 枚举=决策树分支集合受控枚举**（吸收第 3 条，无需额外校验）；工具缺口=wb_assign_task（派发事实+身份校验源）+ wb_task_update（**仅心跳，取消 done**）+ wb_my_task 升级（最小集）。
- [模板骨架标注同步](tickets/05-template-skeleton-sync.md)（2026-09-02 关闭）：**qa 职责重定义落地**——①qa-acceptance-template 重写（用例设计合并：新增 §2 用例设计 7 节[覆盖矩阵/后端口径/前端口径/联调场景/回归范围/验收用例映射 AC/设计自检 Checklist]+骨架表 13 必填节+门禁更新[qa/cases.md→本文 §2、设计即准出]）；②qa-testcase-review-template+demo 移除（不再保留）；③qa-defect-management 正文=骨架对齐（新增缺陷清单总览、归属判定成节、返工与复验记录改名）。**移交工单 03**：gen_flow_rows/xlsx 与 capability-review-v2/flow-alignment 的 case_design/用例设计节点清理（随能力回填同批）。
- [MVP 能力模型终稿（能力×动作回填 + 流程节点表终稿）](tickets/03-capability-action-backfill.md)（2026-09-05 关闭）：**trigger/hook v2.1 模型**——8 词（entry/re-entry/exec/re-exec/reviewed/replied/approve/reject）+ 轮次制澄清循环（reviewed barrier→批量澄清→批量响应→残留真问题才打回；reject 单义化=打回）+ 推进契约（响应者规则/完成信号双通道/reject-escalation 协议/空转规则）+ 冲突响应机制（escalate→会话通道裁决→回填 issue）；**skill 挂载制**——how 四要素退出框架（how-spec-v2 降级参考），动作={on, skill?|inline?}，issue 即产物；**18 能力终稿**（case-design-cap 移除 19→18；to-brd/to-alignment=公共 skill；alignment-cap 收缩为评审响应能力；待建 skill 10 个）；**三流程举一反三**（tech 11/task 8/plan 9；D1-D5：tech 对齐产 brd、task 对齐无产物、plan D1 交互+固化、调研评审机械、inputs 动态注入）+ **跨流程复用验证通过**（无流程专属能力/动作）。落盘四文档：capability-model-final / business-flow-derivation / capability-actions-final / flow-derivation-tech-task-plan。
- [MVP 数据层+工具清单定稿](tickets/07-mvp-data-tools.md)（2026-09-07 关闭）：**schema v0.29**（flow_defs/flow_def_nodes 更名、node_type 四值、user_gate、node_actions 激活清单、review_state/reopen_source、product_type、escalate、sessions 新表、flows.dependencies 依赖树）+ **wb_* 23 个**（+assign_task/task_update event:'complete'；decide 移出 agent 清单；my_task 升级 currentPhase+activeActions；review 写 reviewers[].verdict）+ seed 直接重写 + 扫描器参数（5min 催办/10min 介入）。产出=data-tools-final.md；期间完成 **v3 编排模型定稿**（两层属性状态机、评审能力对象化 {role}.{object}-review、技术设计节点内嵌 CR、一个 state 一个动作、prd 不内嵌 CR）——权威 capability-model-final v3。
- [MVP 交互层定稿](tickets/08-mvp-interaction-layer.md)（2026-09-08 关闭）：产出 **ui-interaction-final.md**（交互层 v3 增补稿：模块 II 五区分栏/scope 动态渲染重排/拍板卡三态/冲突卡上升/会话通道；模块 I scope 徽标+依赖树视图；模块 III/IV 收件箱条目 v3 七类；UI↔数据对应表；暂缓项 P2/P3 视图）+ **原型七面板** docs-tmp/diagrams/ui-prototype-v3.html（P1/P4/P5/P6 确认+跳转按钮，P2/P3 暂缓开发，P7 阅读器吸收）。
- [MVP 范围确认+定稿收尾](tickets/09-mvp-scope-finalize.md)（2026-09-08 关闭）：**roadmap 对照剪裁**（P1-P6 逐项修订：preset 无预设/P2 schema v0.29/P3 persona 节点配置/P4 19 skill/P6 to-brd；v1.1 E2/E3 部分前置、新增 E8 对峙增强/E9 依赖调度器/E10 评审轮视图）+ 文档同步核对 + 术语统一核查 + **MVP 设计定稿宣布**（权威五件套）。**地图全部工单完成，关闭。**

## Not yet specified

- **插件实现期决策**：wb_assign_task / wb_task_update 实装细节、定时兜底任务载体（cd duty）选型，留到推演定稿后。 <clears-with: 04>（✅ 04 关闭：载体=schedule 包+端点兜底已定，实装细节归实现阶段）
- ~~技能包工作流纪律~~：<clears-with: 01> 已解——writing-for-agents 推演期工具（写 how 时用）；ask-matt/scaffold-exercises/teach 裁掉。
- ~~19 能力定稿后的遗留细节~~：<clears-with: 03> ✅ 已解——inputs 收敛进 v2.1 schema（可见性上限+跨流程动态注入）；feature_branch 以 code-review worktree readMode 注记承载，不单列 productType。
- ~~技能裁决落地~~：<clears-with: 03> ✅ 已解——respond_to_qa 撤销并入 fix_on_review（diagnosing-bugs 承载）、dev+self_test 合并（implement-techdesign 编排）、接口面原则进 to-rd-techdesign 门禁。
- **to-xxx 产出 skill 创建（清单已定：10 待建）、team preset 角色补齐、wb_* 实装**：实现阶段（Out of scope）。

## Out of scope

- **合并回 arch_design.md**：**不支持**（2026-09-02 用户重定义）——推演产出收敛于 docs-tmp 定稿层；arch_design 保持现状基线，合并/同步是未来独立 effort（如有）。工单 06（合并回 arch_design 策略）相应改为「MVP 定稿收尾」或关闭。
- **插件代码实现本身**：工作台 wbflow 插件的编码落地不在本地图内，本地图终点是 MVP 设计定稿（docs-tmp）。
- **编码向技能（tdd / code-review / diagnosing-bugs 等）落地**：与当前设计推演阶段无关，留待插件实现期单独评估。
- **v1.1 增强项**：不在 v1.0 MVP 推演范围内。
- **冲突对峙增强**（2026-09-07 记录）：现状=MVP 用「双方结陈 + reviewer 主持人汇总呈用户」（DSH 会话与 agent 一对一，无多 subagent 同会话机制，见 capability-model-final §6.8）；改进项=顺序质询轮（双方各 inject 对方结陈有限反驳）/ 多 agent 同会话真对峙（待 DSH 能力演进）——插件实现期评估。
- **术语统一执行**：CONTEXT.md 术语与既有文档（docs/README 术语表、arch_design）的统一合并，留到推演全部完成后最后统一执行（2026-09-02）。