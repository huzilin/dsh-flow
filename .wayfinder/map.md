# 技能包评估地图（mattpocock skills × 团队项目工作台）

> status: **closed（全部工单完成）** · tracker: 本地 markdown（`.wayfinder/`）· created 2026-08-26 · closed 2026-09-02

## Destination

对 mattpocock/skills 包做全量评估（37 个技能），走到「评估稿 + how 编写规范 v2 定稿」：

1. **`docs-tmp/skills-evaluation.md` 逐技能评估稿**：37 个技能逐一给出复用判定（当前推演期采纳 / 插件实现期激活 / 不适用），含复用方式与结合点。**（✅ 已完成 2026-08-27，判定维度按用户修正为角度 B：技能→能力×动作 how 挂载）**
2. **how 编写规范 v2 草案**：writing-for-agents 写作纪律 × 既有 how 五段式（读产物→按模板→自检→登记/提评/响应→输出）逐条定夺后的定稿。**（✅ 已完成 2026-09-02，`docs-tmp/how-spec-v2.md` 定稿：四要素骨架 + 动作字段收敛）**
3. 关键决策 record_decision 到 semantica。**（✅ 已完成：技能评估、qa 职责重定义、subagent 会话挂载等均已记）**

评估完成 = 地图关闭；之后回到 docs-tmp 主推演（那是`docs-tmp 推演收敛`这个后续 effort 的事，不在这张图上）。**（✅ 2026-09-02 地图关闭）**

## Notes

- 领域：DSH 团队项目工作台（flow/capability/artifact 推演），docs-tmp 推演稿层；被评资产 = 本地 `~/.dsh/skills/mp-*`（37 个，对应 mattpocock/skills 仓库）+ 仓库结构（engineering/productivity/misc/in-progress/deprecated 五桶）
- 判定标准（用户 Q4=A）：按当前阶段「能否改进流程/能力/产物/动作推演工作」为第一标准；编码向技能（tdd/diagnosing-bugs 等）标注「插件实现期激活」不深入
- 评估范围（用户 Q6=B）：全部 37 个都评估——推演链路含开发/CR 等阶段，编码向技能也要
- 触发根因（用户）：技能包直接影响 **action 的 how 如何编写**（writing-for-agents 是靶心）
- 每次会话 consult：grilling、domain-modeling（有疑问时两者都调）
- 用户工作偏好：先 review 部分节点批注、再举一反三；scope 裁剪而非全盘采纳；ProductType 受控枚举、模板引用章节名；决策溯源 record_decision；方法论落 OpenViking 记忆不落项目文档
- tracker 约定：地图 = 本文件；工单 = `tickets/<id>-<slug>.md`（status: open/closed · type · blocked by 标题）；research findings 落 `research/`；关闭工单在 resolution 中记录答案并在下方 Decisions so far 索引

## Decisions so far

- [Engineering 桶盘点（18 个技能）](tickets/01-engineering-bucket-inventory.md)：18/18 全评，当前推演期 9（ask-matt/domain-modeling/grill-with-docs/prototype/research/to-spec/to-tickets/triage/wayfinder），插件实现期 9；.workbench 与 to-spec/to-tickets/triage 高度同构，落地走增强既有 .workbench 不引新机制。
- [Productivity 桶盘点（7 个技能）](tickets/02-productivity-bucket-inventory.md)：7/7 全评，当前推演期 5（grilling/to-questionnaire/handoff/wait-wait/writing-for-agents[靶心]），grill-me 插件期、teach 不适用；how 内嵌路径 vs §0.1 权威表重复是最大裁点（建议 how 只写《标准名》，路径 renderTpl 渲染）。
- [misc / in-progress 桶盘点（12 个技能）](tickets/03-misc-inprogress-inventory.md)：12/12 全评，当前推演期 6（claude-handoff/loop-me/retro/writing-beats/fragments/shape），插件期 5，不适用 1（scaffold-exercises）；写作三件套=人读文稿、writing-for-agents=agent 指令文档，互补非竞争。
- [how 编写规范对照分析（writing-for-agents × 既有五段式）](tickets/04-how-writing-comparison.md)：11 纪律逐条对照，分层为主（wb_my_task/动作拆分=disclosed reference 既有实现）；强互补 2（completion criteria×quality_gates、单一真相源）；待 HITL 裁 3（同 trigger 多动作、防 rush 执行模型、模板路径 A/B）；产出 **v2 条目清单草案 16 条** 供工单 05 拍板。
- [技能 → 能力×动作 how 参考映射（采纳判定）](tickets/06-skill-adopt-decisions.md)：9 项疑难点全部裁决；评估稿落盘 `docs-tmp/skills-evaluation.md`；37 技能 = 进动作 how 13 + 进 v2 5 + 不进 19。关键裁决：writing 三件套挂需求对齐、domain-modeling 挂设计类、Out of Scope 三设计阶段+User Stories 进模板背景节、code-review 单动作串行两子步、**流程缺口补建 respond_to_qa**、**dev+self_test 合并**、接口面原则 4 条进 backend-design、ask-matt/scaffold-exercises/teach 裁掉或无关。

## Not yet specified

- ~~how v2 草案最终形态~~（✅ 2026-09-02 定稿：how-spec-v2.md，16 条全裁）
- 流程缺口移交回填（respond_to_qa 补建 + dev 合并进 capability-review-v2 / execution-loop）——属于`docs-tmp 推演收敛`后续 effort，本图只出裁决
- 模板变更的脚本同步（gen_flow_rows.py 章节引用核对）——随主推演落地
- 实现期派生任务（见 how-spec-v2.md §6）：my-task-response-model / capability-review-v2 §0.2 字段同步、受控词表登记、to-xxx 产出 skill 创建、禁令审计校准——移交主推演/实现阶段

## Out of scope

- docs-tmp 主推演本身的推进（评估稿之外的能力/产物推演正常进行，不阻塞评估）
- 修改/贡献 mattpocock/skills 本体
- 将采纳技能真正接入 DSH 环境（本图只产出判定与规范，不执行接入；接入是插件实现期的事）