# .archive · 历史归档（2026-09-08）

> 本目录 = 重新实现前的历史归档（git mv 保留完整历史，`git log --follow .archive/<path>` 可溯）。**勿据以实现**——现行设计权威 = `docs/v1.0/arch_design.md`（单一权威，2026-09-12 MIG-07 合并）；实现规格 = `.plan/wbflow-impl/spec.md`。

| 归档内容 | 来源 | 说明 |
|---|---|---|
| `workbench-plugin/` | 根目录 | 旧实现（pkg-16 时代：client.js 829 行 / host.js 534 行 / README）——按 v3.1 编排模型重新实现（已落地） |
| `docs-tmp-history/`（10 份 + 生成器 + xlsx + business-flow.json） | docs-tmp/ | 历史推演层文档（均已标「结论时效」横幅）+ 旧流程生成器 + 旧流程图数据 |
| `tmp/` | 根目录 | 旧章归档（arch_design 拆分前 9 章）+ 架构可视化图——gitignore 外部分已加历史指针 |
| `docs-tmp-final/`（7 md + diagrams） | docs-tmp/（2026-09-12 MIG-07） | 五件套推演底稿（capability-model / capability-actions / business-flow-derivation / flow-derivation-tech-task-plan / data-tools / ui-interaction-final / how-spec-v2），**已并入 docs/v1.0/arch_design.md**，各件顶部有归档横幅 |
