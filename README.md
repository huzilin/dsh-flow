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

## 质量闸门

- **pre-push hook（2026-09-11 拍板一期④）**：`git config core.hooksPath scripts/githooks` 启用（clone 后一次性）。push 前自动跑 `buf breaking`（proto 契约对「被推前远端 tip」的 wire 兼容拦截）+ `qa/qa_branch_audit.py`（seed 动作可解析性 + 失效豁免；补测差集默认放行，`--strict` 锁定）。确需破坏性契约变更：commit 注记声明后 `--no-verify` 绕过。
- **提交闸门惯例**：Go 侧 `gofmt -l . && go vet ./... && go build ./... && go test ./...`；FE 侧 `npm test`；QA 驱动 `qa/qa_drive.py`（需 qa_env_up.sh 环境）。
