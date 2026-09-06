# misc / in-progress 桶盘点（12 个技能）

> type: research · blocked by: — · resolved 2026-08-27

## Question

本地 `~/.dsh/skills/mp-*` 中 misc 桶 4 个 + in-progress 桶 8 个（含写作 three-beats 系列）逐个盘点：功能、触发方式、输入输出、依赖，并按 Q4=A 标准给出**初步**复用判定（当前推演期 / 插件实现期激活 / 不适用）。

范围：git-guardrails-claude-code, migrate-to-shoehorn, scaffold-exercises, setup-pre-commit（misc）；claude-handoff, implement-spec, loop-me, retro, setup-ts-deep-modules, writing-beats, writing-fragments, writing-shape（in-progress）

## Findings

见 `.wayfinder/research/misc-inprogress-inventory.md`。

## Answer

12/12 逐篇全文阅读完成。初步判定：**当前推演期 6**（claude-handoff 交接三机制——只引用产物路径/必含 suggested skills/脱敏——直进 how 规范 v2；loop-me trigger/checkpoint/push right 与推演同构；retro 7 类审计维度=动作推演检查清单；writing-beats/fragments/shape 三段式=推演稿 explore→exploit 生产流水线）；**插件实现期 5**（setup-pre-commit/migrate-to-shoehorn/git-guardrails(低优先，与 DSH 沙箱/审批重叠)/setup-ts-deep-modules/implement-spec）；**不适用 1**（scaffold-exercises，强绑 ai-hero-cli，骨架化+lint 校验闭环机制可借鉴）。写作三件套=人读文稿（项目推演稿用）、writing-for-agents=agent 指令文档（how v2/AGENTS.md 用），互补非竞争。最终采纳判定移交票 06。