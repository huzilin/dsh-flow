# 技能 → 能力×动作 how 参考映射（采纳判定）

> type: grilling · blocked by: Engineering 桶盘点（18 个技能） · Productivity 桶盘点（7 个技能） · misc / in-progress 桶盘点（12 个技能） · 判定维度修正（用户 2026-08-27）· resolved 2026-08-27

## Question

**判定维度（用户修正，角度 B）**：不是「技能对我们当前推演流程有没有流程性帮助」，而是**每个技能应挂到哪些能力×动作的 how 上作参考**——19 能力×动作（capability-review-v2.md：8 评审型 + 11 产出型，动作表见 §0.2）的 how 编写时，哪些技能作为参考素材/纪律被吸收（如 code-review 双轴评审 → CR 动作 how；tdd 红绿循环 → 开发自测动作 how）。开发/CR 等动作就在推演范围内，「插件实现期」仅保留极少数真正与工作台无关的技能。

产出：`docs-tmp/skills-evaluation.md` 评估稿（含「技能 → 能力×动作 how 参考映射表」+ 逐技能采纳判定 + 结论合并去向确认）。先由 research 产出映射初稿（.wayfinder/research/skill-to-action-how-mapping.md），用户逐条批注、举一反三定稿。

## Answer

**已定稿（2026-08-27）**。映射初稿 `.wayfinder/research/skill-to-action-how-mapping.md` 9 项疑难点全部由用户批注裁决，评估稿落盘 **`docs-tmp/skills-evaluation.md`**（37 技能 = 进动作 how 13 + 进 v2 5 + 不进 19；另含流程/模板/术语变更清单）。

**裁决要点**：①判定维度按角度 B（技能→能力×动作 how）重梳理；②writing 三件套挂需求对齐环节（align.discuss + alignment-cap.produce），非 research-cap；③domain-modeling 挂设计类环节（arch/product/rd 设计）；④to-spec：User Stories 进背景章节（4 模板已加节+骨架标注）、Out of Scope 三设计阶段写文档；⑤code-review 单动作串行两子步（Standards→Spec）；⑥**流程缺口：补建 respond_to_qa 动作**（QA 缺陷返工从未显式动作化，diagnosing-bugs 纪律落此）；⑦tdd：dev+self_test **合并**单动作（trigger 链简化，v2#9 冲突消解）；⑧codebase-design 接口面原则 4 条进 backend-design（受控术语）；⑨ask-matt / scaffold-exercises / teach 裁掉或无关。流程缺口（respond_to_qa 补建、dev 合并）移交 docs-tmp 主推演；模板变更已落盘；关键决策 record_decision 到 semantica。