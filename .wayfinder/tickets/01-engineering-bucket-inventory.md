# Engineering 桶盘点（18 个技能）

> type: research · blocked by: — · resolved 2026-08-27

## Question

本地 `~/.dsh/skills/mp-*` 中 engineering 桶 18 个技能逐个盘点：功能、触发方式、输入输出、依赖，并按 Q4=A 标准给出**初步**复用判定（当前推演期 / 插件实现期激活 / 不适用）。

范围：ask-matt, code-review, codebase-design, diagnosing-bugs, domain-modeling, grill-with-docs, implement, improve-codebase-architecture, prototype, research, resolving-merge-conflicts, setup-matt-pocock-skills, tdd, to-spec, to-tickets, triage, wayfinder, wizard

## Findings

见 `.wayfinder/research/engineering-inventory.md`。

## Answer

18/18 逐篇全文阅读完成。初步判定：**当前推演期 9**（ask-matt/domain-modeling/grill-with-docs/prototype/research/to-spec/to-tickets/triage/wayfinder）；**插件实现期 9**（code-review/codebase-design/diagnosing-bugs/implement/improve-codebase-architecture/resolving-merge-conflicts/setup-matt-pocock-skills/tdd/wizard）；无「不适用」。关键对照：.workbench 领域模型（req-*/requirements/inbox/comments）与 to-spec/to-tickets/triage 高度同构——flow.json parallelGroup join+部分回滚 = to-tickets blocking edges/expand-contract 的本地实现，inbox.json = triage 流入通道；增量在 User Stories 视角、Out of Scope 显式化、五状态角色（needs-info 往返）、verify/redundancy 纪律；建议落地增强既有 .workbench，不引 `.plan/` 新机制。最终采纳判定移交票 06（HITL）。