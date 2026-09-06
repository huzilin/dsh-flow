---
type: grilling
blocked_by: []
claimed_by: agent session (2026-09-05)
claimed_at: 2026-09-05
---

# MVP 能力模型终稿（能力×动作回填 + 流程节点表终稿）

> **2026-09-02 改题**（目标重定义：MVP 设计定稿）：原「能力×动作回填确认」扩展为 **MVP 能力模型终稿**——19 能力×动作完整回填 + **流程节点表终稿**（flow-alignment 更新）+ case-design-cap 处置。

> **2026-09-05 推演进度**：trigger/hook 模型经用户逐轮推演**闭合**（8 trigger：entry/re-entry/exec/re-exec/reviewed/replied/approve/reject；轮次制澄清循环；reject 单义化=打回事件；to-brd/brd 定稿；case-design-cap 移除 19→18；respond_to_qa 撤销并入 fix_on_review；retest 移除；按需会话通道；重开来源）。**已落盘**：capability-model-final.md v2（模型权威层）、business-flow-derivation.md v2（business 13 节点逐节点表）、capability-actions-final.md（18 能力可回填终稿）。**剩余**：tech/task/plan 三流程举一反三 + 跨流程复用验证（工单 03 第 4 项）。

## Question

**MVP 能力模型终稿**（吸收全部已裁决输入，产出可直接回填 seed.json 的终稿）：

1. **19 能力 × actions[] 逐能力定稿**：每动作 `{action_id, name, trigger, how(四要素), tools, outputProductTypes}`（字段按 how-spec-v2 收敛：无 when_to_use/reference/focus/query）；inputs 按 ProductType 最终形态（use_threads 默认 true）。
2. **case-design-cap 处置**（qa 职责重定义）：用例设计不再设独立节点/评审节点，内容并入 qa-acceptance（验收文档 §2）——该能力**移除**还是转移（如并入 acceptance-test 的设计子步）？
3. **流程节点表终稿**（flow-alignment 更新）：business 删 A7/A8、tech 删 B6/B7（用例设计/评审）；QA 验收节点描述改"用例设计+执行验收一体"；business 12→10 节点、tech 13→11；respond_to_qa 补进 RD/FE 开发节点动作链。
4. **跨流程复用验证**：同一能力在 business/tech/task/plan 实例化时 how 仍有效（scope 裁剪对称）。

依据：capability-review-v2.md、flow-alignment.md、how-spec-v2.md、skills-evaluation.md、skill-usage-implementation.md、produce-form-skill-derivation.md、plugin-agent-subagent-chain.md。

## 技能评估回填的已裁决输入（2026-08-27，来自工单 01 resolution）

本工单敲定 actions 时必须吸收以下裁决（详见 docs-tmp/skills-evaluation.md §4）：

1. **补建 `backend-dev.respond_to_qa` / `frontend-dev.respond_to_qa` 动作**（流程缺口）：QA 验收打回的缺陷无对等响应动作（现只有 respond_to_cr，query 限 reviewerRole=Code Reviewer；execution-loop 决策树②「QA 返工闭环」从未显式动作化）。新动作与 respond_to_cr 对称：trigger=reviewer_feedback(open)，how 吸收 diagnosing-bugs 纪律（最小化复现→一次一变量定位→修在正确接缝→补回归→resolve→QA 复核 confirm/reopen）。
2. **合并 `backend-dev.dev` + `.self_test` 为单动作**（三子步：实现[垂直切片/公共接口]/自测[红绿循环]/记录[实现-设计对照表]；完成标准=自测全绿+对照表逐行成立）；trigger 链简化为 node_enter → dev（含自测）→ dev_complete → submit（唯一）；frontend-dev 同构。self_test 独立 trigger 消失。
3. **评审型 code-review 动作**：单动作 .review how 内串行两子步（Standards 轴=对照表+规约基线 → Spec 轴=设计契约对需求/AC → 聚合表态），不拆双 gate。
4. **backend-design 动作**：接口设计章节按接口面原则 4 条编写（小接口大实现/依赖传入不创建/返回值无副作用/seam 行为边界），门禁 major 核对；受控术语 deep module/接口面/seam。
5. **decide-cap.create_reqs**：吸收 to-tickets 垂直切片粒度判据 + triage 查重纪律（redundancy 按领域概念搜、verify claim）。
6. **align/alignment-cap**：需求对齐交互模式——subagent 主动开场（说明需求/阶段/要确认什么，提示可选交互 skills[grill-me/wayfinder/domain-modeling/一句话]）；alignment-cap.produce 对齐后产出（to-xxx 家族）。

## 字段与骨架终稿输入（2026-09-02，how-spec-v2）

- 动作字段收敛：`{action_id, name, trigger, how(四要素), tools, outputProductTypes}`——when_to_use/reference/focus/query 移除（语义由 trigger 决策树与 skill 内部承载）
- how 四要素：目标（产出+可量化标准）/ 依赖（动态看 mytask 返回，按 flow 类型/阶段）/ 执行（基于 skill 或编排序列 [to-tickets, implement-techdesign]）/ 自检（达成目标+按执行要求+质量达标）
- 工具知识沉淀为独立 skill（wb 接口用法）；query/no-op 逻辑在 skill 内部
- 程序状态推进独立于 how（节点内=全 resolved；文档进评审=open→pending-confirm；评审=全员表态；产物=应提交齐）

## Answer（2026-09-05 关闭）

**MVP 能力模型终稿定稿**（四问全解 + 编排底座推演）：

1. **trigger/hook 模型 v2.1**（编排底座，用户逐轮推演定稿）：trigger 8 词（entry / re-entry / exec / re-exec / reviewed / replied / approve / reject）；hook=`[before|after]<trigger>`；**轮次制澄清循环**（reviewed barrier → owner 批量澄清 → 评审者批量响应 → 残留真问题才打回；reject 单义化=打回事件；User 拍板后移至轮次完成）；四类节点生命周期（产出型有唯一 exec / 评审型纯事件响应 / 交互型 / 门禁型）；**推进契约**（响应者规则=谁等结果谁负责；完成信号双通道=短动作数据写入即完成+长任务/交互等待响应者上报；reject/escalation 结论协议；空转规则=无挂载直接轮转）；**冲突响应机制**（escalate 标记 → 按需会话通道裁决 session → 裁决回填冲突 issue → reviewer 重新表态恢复轮次）；外部兜底扫描器（task_stale 重发 trigger，不在词表）。
2. **skill 挂载制**（2026-09-05 裁决）：how 四要素退出框架（how-spec-v2 降级为 skill 写作参考）；动作={action_id, name, on, skill?|inline?}——复杂动作挂 skill（内部按需编排其他 skill、自带目标/自检/门禁），短动作一句话内联；**issue 即产物**（评审类动作 outputProductTypes 空=正确语义）。
3. **18 能力终稿**（19−case-design-cap）：to-brd/to-alignment=**公共 skill**（对齐节点 node action 挂载，不属角色能力）；alignment-cap 收缩为 D2 评审响应能力（clarify/fix_on_review）；挂载映射 produce×6→to-xxx、dev→implement-techdesign、fix_on_review→diagnosing-bugs、review→review-protocol、accept→qa-acceptance、create_reqs→to-tickets；待建 skill 10 个。→ `capability-actions-final.md`
4. **流程节点表终稿**：business 13 节点（`business-flow-derivation.md` v2.1）；tech 11 / task 8 / plan 9（`flow-derivation-tech-task-plan.md`，D1-D5 裁决：tech 对齐产 brd、task 对齐无产物[技术设计基础=简单描述]、plan D1 交互+to-alignment 汇总、调研评审机械门禁、inputs 动态注入确认）。flow-alignment.md 的同步归工单 09 文档定稿。
5. **跨流程复用验证（第 4 项）**：18 能力 × 4 流程矩阵全覆盖，**无流程专属能力/动作**；差异仅节点组合+评审位集合+scope 裁剪；inputs[]=可见性上限、按流程可用产物动态注入 my_task（skill 内写明来源，实现时校验）。

**遗留**：无。**前沿**：工单 07（数据层+工具清单定稿——my_task 上下文契约、wb_assign_task/wb_task_update、schema 调整以其为输入）。