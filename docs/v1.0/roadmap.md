# 版本规划 · 团队项目工作台平台 roadmap

> 版本路线图：目标 / 范围 / 需求清单 / 拆分。承接 `arch_design.md` §6 的需求拆解（AUR）与 `arch_design.md` §11 的落地状态。
> 平台的演进按里程碑 M0–M8 展开，详见 `arch_design.md` §11；本文聚焦版本级目标与需求拆解。

---

## 1. 版本目标

| 版本 | 目标 | 状态 |
|---|---|---|
| **v1.0（MVP）** | 可端到端跑通的最小闭环：每个角色有真实子 agent 执行、产出登记、评审闭环、门禁推进、收件箱通知全流程可用 | 已发布 |
| **v1.1** | 增强与稳定性：首屏性能、移动端适配、数据导入导出、并发/回归验证 | 待启动 |
| **v2.0** | 已废弃（历史遗留） | 已废弃 |

## 2. v1.0 MVP 范围（需求清单与拆分）

MVP 核心目标（2026-09-08 按 v3.1 编排模型修订）：新增会话以 team preset + 团队制度启动（**22 个 `wb_*` 工具**：+assign_task/task_update，−decide/override 移出 agent 清单；skill 可加载）；一条需求走完业务流程（**需求对齐（to-brd）→ 产品设计 → 多轮评审（per-agent 表态轮转+collect 聚合）→ RD/FE 并行开发（CR 内嵌）→ QA 验收（仅 full 渲染）→ 用户验收**）无阻塞；收件箱通知链路可用（通知规程 §6.8 七类）。**MVP 设计定稿见 docs-tmp/ui-interaction-final.md + capability-model-final.md v3.1**。

### MVP 需求拆解

| # | 需求 | 落地状态 | 预计工作量 |
|---|---|---|---|
| P1 | **team preset 接线**（M6）：从 standard 复制 → 叠加团队制度 → 挂载 skill → 新会话可加载。【v3.1 修订】subagent **不需要预设**（persona 由 wbflow 节点配置传入，程序化直启实测通过）；team preset 只读治理（制度层） | ⏳ PRD 定稿，尚未接线 | 1–2 天 |
| P2 | **sqlite 应用层迁移**（M8）：wb_* 工具改走 sqlite 事务。【v3.1 修订】schema 按 **v0.29**（flow_defs/flow_def_nodes 更名、node_type/user_gate/node_actions/review_results/reopen_source/escalate/loop_count、sessions 新表、flows.dependencies）；wb_my_task 升级（currentPhase+activeActions+review_results 摘要）| schema v0.29 设计定稿；应用层迁移进行中 | 2–3 天 |
| P3 | **Architect / PM / Investigator 角色卡**（M7）：落地 persona。【v3.1 修订】persona 由 wbflow 节点配置传入（非 subagent 预设）；评审维度走对象化评审 skill（最小上下文） | ⏳ 待落地 | 1–2 天 |
| P4 | **Skill 落地**：【v3.1 修订】**19 个待建 skill**（to-xxx 产出家族 6 / implement-techdesign / 对象化评审 skill 11 / qa-acceptance）取代 wb-agent-guide 单编排 skill；team-conventions 保留 | skill 清单定稿（capability-actions-final §3），待沉淀 | 2–3 天 |
| P5 | **真实 agent 执行**（M7）：子 agent 直接使用 wb_* 工具；Leader 派活 prompt 更短 | req-019 演练 4 角色通过 | 1–2 天 |
| P6 | **需求对齐节点实践**（M7）：【v3.1 修订】对齐 exec=**to-brd**（产出 brd 业务需求文档，ProductType=brd；不再落 issues(discussion)）；完成信号=用户确认（唯一信号）| 已定，待演练 | 0.5 天 |

### 模块拆解（AUR）

见 `arch_design.md` §6：AUR-001 数据层 → AUR-002 模块 II → AUR-003 模块 I → AUR-004 收件箱/TODO → AUR-005 制度/执行。

## 3. v1.1 增强范围

| # | 需求 | 说明 |
|---|---|---|
| E1 | **diff 级锚点** | 产物版本化 + 行级 diff 锚点（需 revision 存储） |
| E2 | **多项目切换 UI** | 顶部项目选择器（**v3.1 前置**：project registry + per-project workbench 隔离设计已定稿——Q6-Q8；MVP 无跨项目 agent 操作） |
| E3 | **多项目收件箱/TODO 聚合** | 跨项目聚合，项目筛选 + 任务名筛选（基线已全局聚合，仅缺多项目数据源接线） |
| E4 | **版本间需求批量迁移** | 大版本拆分/合并时批量迁移需求 |
| E5 | **候补队列（backlog 管理）** | 待分配需求优先级排序、自动推荐归入版本 |
| E6 | **Agent 审计日志** | audit_log 完整度验证、决策变更回放 |
| E7 | **性能优化** | 大需求池筛选优化、sqlite WAL 性能验证 |
| E8 | **冲突对峙增强** | 顺序质询轮（双方 inject 对方结陈有限反驳）/ 多 agent 同会话对峙（待 DSH 能力演进）——v3.1 冲突响应机制的增强（2026-09-07） |
| E9 | **依赖调度器** | 版本内需求按依赖自动调度（就绪池高优拉取；并发度=正在工作的需求数）——MVP 人工卡点，数据层（flows.dependencies）已预留（2026-09-07） |
| E10 | **评审轮完整视图 UI** | P2 评审轮视图 / P3 进度面板（per-agent 轮转+issue 子状态可视化）——MVP 以节点详情聚合快照呈现（2026-09-08 暂缓裁决） |

## 4. 实现路径（依赖关系）

```
P1 preset 接线 ──→ P4 skill 落地 ──→ P5 真实 agent 执行
                                              │
P2 sqlite 迁移 ──────────────────────────────→ P3 角色卡落地 → P6 需求对齐
                                                         │
                                                     MVP（v1.0）交付
                                                         │
                                                         ▼
                                               v1.1 增强（E1–E7）
```

**关键依赖**：P1 与 P2 可并行；P4 依赖 P1；P5 依赖 P1+P2+P3；P6 依赖 P3+P2。【v3.1 注】P2 的 schema 基线=v0.29（data-tools-final）；P4 清单=capability-actions-final §3（19 skill）；P5 机制=subagent 程序化直启（实测通过）

## 5. 版本规划产物状态

| 产物 | 文档 | 状态 |
|---|---|---|
| 需求对齐 | `align.md`（本平台立项即整体对齐，结论落 comments/issues） | ✅ |
| 架构规划 | `arch_design.md` | ✅ |
| 里程碑 | `arch_design.md` §11 | ✅ |

> 注：团队项目工作台平台的版本规划以来立项即整体定义方式推进；`align.md`/`research.md` 对本平台多为基座对齐与选型论证（部分结论并入 `arch_design.md` §3/§7）。
