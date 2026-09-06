# how 编写规范对照分析（writing-for-agents × 既有五段式）

> type: research · blocked by: — · resolved 2026-08-27

## Question

将 writing-for-agents 的写作纪律（context pointer / 两 load / 信息层级 / leading words / negation / pruning / completion criteria）与既有能力推演方法论中 how 的写法（五段式：读产物[productType]→按模板[路径+章节名]→自检[quality_gates]→登记/提评/响应[wb_* 工具]→输出[productType]；trigger 程序契约；对话类产物用 query 属性过滤）逐条对照：每条纪律与五段式是**互补、冲突还是分层**？哪些值得叠加进 how 编写规范 v2？

对照需基于真实 how 实例（docs-tmp/capability-review-v2.md 中的 19 能力 how）而非抽象谈论。

## Findings

见 `.wayfinder/research/how-writing-comparison.md`。

## Answer

11 条纪律逐条对照（全部基于真实文件：mp-writing-for-agents/SKILL.md + capability-review-v2.md 四样本 + my-task-response-model.md 等）。结论：**分层为主**——wb_my_task 按需取 how/动作拆分 = disclosed reference + progressive disclosure 的既有实现，缺的是显式规范条文；**强互补 2**：①completion criteria × quality_gates（how 缺「可判定+穷尽」的完成收尾句，自检步是满分样板）；②pruning 单一真相源（发现 capability-artifact-analysis.md A1 行动表复制 backend-dev.dev.how 全文的现存重复，待清理）；**待 HITL 裁 3**：①self_test/submit 同 trigger=dev_complete 的选择条件需状态化或明示接受时序语言（v2 草案第 9 条）；②防 rush 的「隐藏后文」依赖执行模型（同 subagent 串行 vs 新上下文）需写明两态差异（第 10 条）；③模板路径双写 A/B 单真相源裁决（第 14 条，productivity 盘点同建议）。negation：现禁令大多已配正向目标，改进面=「不评排期」类纯禁令句→正述边界。leading words：契约/口径/各审各的/章节名已是事实词条，缺登记+同义收敛。产出 **v2 条目清单草案 16 条**（research 文件末尾），移交票 05 逐条拍板。