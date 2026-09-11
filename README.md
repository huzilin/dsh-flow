# wbflow · 团队项目工作台（重新实现）

> v1.0 MVP 设计定稿（2026-09-08）：**设计权威 = `docs/v1.0/arch_design.md`（单一权威，2026-09-12 MIG-07 合并）**，技术规格 `.plan/wbflow-impl/spec.md`，历史实现与推演底稿在 `.archive/`（五件套底稿在 `.archive/docs-tmp-final/`）。

## 目录结构

```
rd/    后端（Go + Gin + MySQL 8.0）—— 后端 root
fe/    前端（Web，本期实现；预留 DSH 嵌入）
docs/  需求与设计基线（docs/v1.0 三模块 PRD/RD/FE + roadmap + arch_design 权威）
.plan/wbflow-impl/ 技术规格（spec.md 数据模型 DDL / HTTP API / 验收标准）+ 工单
```

- 技术规格：`.plan/wbflow-impl/spec.md`（数据模型 14 表 DDL / HTTP API 22 端点 / 异常处理 / 性能考量 / 验收标准）
- 交互原型：`.archive/docs-tmp-final/diagrams/ui-prototype-v3.html`

## 质量闸门

- **pre-push hook（2026-09-11 拍板一期④）**：`git config core.hooksPath scripts/githooks` 启用（clone 后一次性）。push 前自动跑 `buf breaking`（proto 契约对「被推前远端 tip」的 wire 兼容拦截）+ `qa/qa_branch_audit.py`（seed 动作可解析性 + 失效豁免；补测差集默认放行，`--strict` 锁定）。确需破坏性契约变更：commit 注记声明后 `--no-verify` 绕过。
- **提交闸门惯例**：Go 侧 `gofmt -l . && go vet ./... && go build ./... && go test ./...`；FE 侧 `npm test`；QA 驱动 `qa/qa_drive.py`（需 qa_env_up.sh 环境）。
- **测试端口约定（2026-09-12 拍板 DM-09）**：测试用端口一律落在 **6000-6099** 区间（QA 服务 :6081、testpage :6090），不得外溢。

## 历史决策时间线（先查台账再拍板）

**`.plan/timeline/decision-timeline.md` = 全部历史拍板的时间线正本**（五源合流：dsh 会话/zcode 会话/test-flow 会话/仓库文档代际/test-flow 一版代码）。纪律：讨论任何设计问题前先查台账——同一问题以**最后一次拍板**为现行权威，旧结论逐条标 superseded-by，**禁止把已废弃结论再拿来请用户确认**。台账未覆盖的新问题才进入新一轮拍板，定稿后回写台账。配套：repo-doc-generations.md（文档分层与多代沉积）、testflow-interaction-inventory.md（一版交互设计与 fe 履约对照）、terminology-evolution.md（**术语演进映射**：中文原话→历代名→现行定名）、retired-terms-audit.md（退役词引用清查四档台账）、zcode/dsh/testflow-sessions-decisions.md（三份会话原始提取）。

**方案敲定三动作（常态化）**：每轮拍板定稿，随台账回写必走——①术语变更更新 `CONTEXT.md`（现行定义+_Avoid_）并在 terminology-evolution.md 加行；②退役词过 retired-terms-audit 四档清查；③审计沿革进 timeline/archive，现行文档不留变更日志。详见 `docs/templates/alignment-protocol.md` 机制 2。
