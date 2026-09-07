# wbflow · 团队项目工作台（重新实现）

> v1.0 MVP 设计定稿（2026-09-08）：权威五件套在 `docs-tmp/`，技术规格 `.plan/wbflow-impl/spec.md`，历史实现与文档在 `.archive/`。

## 目录结构

```
rd/    后端（Go + Gin + MySQL 8.0）—— 后端 root
fe/    前端（Web，本期实现；预留 DSH 嵌入）
docs/  需求与设计基线（docs/v1.0 三模块 PRD/RD/FE + roadmap + arch_design）
docs-tmp/ 设计定稿五件套（权威）
```

- 技术规格：`.plan/wbflow-impl/spec.md`（数据模型 14 表 DDL / HTTP API 22 端点 / 异常处理 / 性能考量 / 验收标准）
- 交互原型：`docs-tmp/diagrams/ui-prototype-v3.html`
