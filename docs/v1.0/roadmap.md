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

MVP 核心目标：新增会话以 team preset 启动（系统提示含团队制度、22 个 `wb_*` 工具可见、skill 可加载）；一条需求走完业务流程（需求设计 → 评审 → RD/FE 并行开发 → CR → QA 验收 → 用户验收）无阻塞；收件箱通知链路可用。

### MVP 需求拆解

| # | 需求 | 落地状态 | 预计工作量 |
|---|---|---|---|
| P1 | **team preset 接线**（M6）：从 standard 复制 → 叠加团队制度 → 挂载 skill（wb-agent-guide / team-conventions）→ 新会话可加载 | ⏳ PRD 定稿，尚未接线 | 1–2 天 |
| P2 | **sqlite 应用层迁移**（M8）：wb_* 工具改走 sqlite 事务；wb_my_task 组装链切 flow_template_nodes.mode / capabilities.products 展开；JSON 载体废弃 | schema 已建库 seed；应用层迁移进行中 | 2–3 天 |
| P3 | **Architect / PM / Investigator 角色卡**（M7）：落地 persona（专业卡 + 3 行协作协议） | ⏳ 待落地 | 1–2 天 |
| P4 | **Skill 落地**（M6）：wb-agent-guide 编排流程 + team-conventions 制度速查 | skill 草案已有，待沉淀 | 0.5–1 天 |
| P5 | **真实 agent 执行**（M7）：子 agent 直接使用 wb_* 工具；Leader 派活 prompt 更短 | req-019 演练 4 角色通过 | 1–2 天 |
| P6 | **需求对齐节点实践**（M7）：tech①/task① align 结论落 issues(discussion) | 已定，待演练 | 0.5 天 |

### 模块拆解（AUR）

见 `arch_design.md` §6：AUR-001 数据层 → AUR-002 模块 II → AUR-003 模块 I → AUR-004 收件箱/TODO → AUR-005 制度/执行。

## 3. v1.1 增强范围

| # | 需求 | 说明 |
|---|---|---|
| E1 | **diff 级锚点** | 产物版本化 + 行级 diff 锚点（需 revision 存储） |
| E2 | **多项目切换 UI** | 顶部项目选择器（数据层已按 projectId 分桶，仅缺界面） |
| E3 | **多项目收件箱/TODO 聚合** | 跨项目聚合，项目筛选 + 任务名筛选 |
| E4 | **版本间需求批量迁移** | 大版本拆分/合并时批量迁移需求 |
| E5 | **候补队列（backlog 管理）** | 待分配需求优先级排序、自动推荐归入版本 |
| E6 | **Agent 审计日志** | audit_log 完整度验证、决策变更回放 |
| E7 | **性能优化** | 大需求池筛选优化、sqlite WAL 性能验证 |

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

**关键依赖**：P1 与 P2 可并行；P4 依赖 P1；P5 依赖 P1+P2+P3；P6 依赖 P3+P2。

## 5. 版本规划产物状态

| 产物 | 文档 | 状态 |
|---|---|---|
| 需求对齐 | `align.md`（本平台立项即整体对齐，结论落 comments/issues） | ✅ |
| 架构规划 | `arch_design.md` | ✅ |
| 里程碑 | `arch_design.md` §11 | ✅ |

> 注：团队项目工作台平台的版本规划以来立项即整体定义方式推进；`align.md`/`research.md` 对本平台多为基座对齐与选型论证（部分结论并入 `arch_design.md` §3/§7）。
