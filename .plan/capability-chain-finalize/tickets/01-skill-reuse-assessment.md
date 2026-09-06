---
type: research
blocked_by: []
claimed_by:
claimed_at:
---

# 技能复用评估

## Question

mattpocock/skills 技能包（本地已装全套 mp-\*，上游为 [mattpocock/skills](https://github.com/mattpocock/skills)）中哪些技能可与本项目（团队项目工作台）当前的流程/能力/产物推演工作复用？各技能能力如何、怎么结合更好？

需要产出：技能 × 本项目工作流映射表（存 `.plan/.../assets/`），按「当前推演期可采纳 / 留待插件实现期 / 不适用」三档分类，并给出采纳建议。

候选关注方向（高契合初判）：writing-for-agents（能力 how 是给 agent 看的文档）、wayfinder（推演→决策票）、grilling/domain-modeling（评审与术语）、to-tickets/to-spec/triage（与 .workbench 领域模型相似）。

## Answer（2026-08-27 完成）

评估在 `技能包评估地图`（.wayfinder/，独立子地图）完成，产出与裁决：

- **评估稿**：`docs-tmp/skills-evaluation.md`（37 技能 = 进动作 how 13 / 进 how 规范 v2 5 / 不进 19；判定维度按用户修正为**角度 B**：技能→能力×动作 how 挂载，非旧三档）
- **映射初稿**：.wayfinder/research/skill-to-action-how-mapping.md（9 项疑难点全部用户批注裁决）
- **三桶盘点**：.wayfinder/research/{engineering,productivity,misc-inprogress}-inventory.md；**how 写作对照**：.wayfinder/research/how-writing-comparison.md（v2 草案 16 条）

**回填主推演的裁决（影响 03/05 票）**：
1. **流程缺口：补建 `respond_to_qa` 动作**（backend-dev / frontend-dev）——QA 缺陷返工从未显式动作化（execution-loop 决策树②只有概念），diagnosing-bugs 纪律（最小化复现→一次一变量→修在正确接缝→补回归）落此 → 03 票必须补建
2. **dev + self_test 合并**单动作（三子步实现/自测/记录），trigger 链简化（dev_complete 只剩 submit 唯一动作），v2 草案第 9 条冲突消解 → 03 票执行
3. **模板变更已落盘**：4 模板加「动机场景（用户故事）」节 + 骨架标注同步（prd/arch/backend/fe）→ 与 05 票（模板骨架同步）同向，05 需覆盖 qa 系列遗留 + 核对 gen_flow_rows.py 引用
4. code-review 单动作串行两子步（Standards→Spec）；codebase-design 接口面原则 4 条进 backend-design（受控术语：deep module/接口面/seam）；to-tickets/triage 挂 decide-cap.create_reqs；writing 三件套挂需求对齐环节（align.discuss + alignment-cap.produce）；domain-modeling 挂设计类环节