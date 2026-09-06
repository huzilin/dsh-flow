# 里程碑与落地状态

> 实现沿 M0–M8 演进；每个里程碑独立评审、可回滚（「先设计再逐段拆解」）。
> 状态标记：✅ 完成（含实现回写）｜🟡 部分/进行中 ｜⬜ 未开始

## 1. 里程碑定义与状态

| 里程碑 | 内容 | 状态 |
|---|---|---|
| M0 | 设计对齐：本文档体系 + 定稿决策达成一致 | ✅ |
| M1 | 数据层：`.workbench/` 文件 Schema + 多项目分桶 + CRUD | ✅（并入 M2 落地） |
| M2 | **模块 II** 需求 OA 流程树（纵向切片）：流程树 + 评审闭环 + 产物阅读器 + 锚点评论 + 门禁 | ✅ |
| M3 | **模块 I（上）** 版本看板 + 需求池 + 优先级 + 分组列表 | ✅（pkg-4~7） |
| M4 | **模块 I（中）** 需求 CRUD + 描述 + 编辑审计 | ✅（pkg-8/15/23） |
| M5 | **模块 I（下）** 版本管理（CRUD + 状态机 + 进度汇总） | ✅（pkg-15） |
| M6 | 制度层 team preset + 工具层（wb_* 工具全量注册）+ 通知落库 | 🟡（工具层 ✅ pkg-24/41；preset 待接线；sqlite 落库进行中） |
| M7 | 执行层：真实子 agent 多角色评审编排（并行 join / 部分回滚 / CR 读代码） | 🟡（req-019 演练 ✅；Architect/PM/Investigator 角色卡待落地） |
| M8 | 流程简化程序化：wb_my_task + 默认路由 + 能力模型 + 模板固化 + sqlite | 🟡（wb_my_task ✅ pkg-41；sqlite 已建库 seed；应用层迁移进行中） |

## 2. 当前进度明细（截至定稿）

### 交互层（wbflow 插件，Host + Client）

- **M2 流程树全部完成**：门禁（通过/打回 + 前进回退 + 全评论 resolved 拦截）/ 评审状态行 + 角色对话区 + reply/resolve/override / 产物文件行 + 阅读器 / 行级锚点 + 防漂移 / 范围评论 / 代码块整块评论。锚点四级：document / line / range / block。
- **M3 看板完成（pkg-7）**：版本 `<select>` 筛选 + 状态泳道 + 看板/列表切换 + 优先级 P0–P3 + 列表分组。
- **M4 CRUD 完成（pkg-8/15/23）**：新建/编辑/删除 + 描述 Markdown 渲染 + 折叠 + 变更历史自然语句 + 需求信息面板。
- **M5 版本管理完成（pkg-15）**：版本 CRUD + 状态机 + 发布确认 + 进度汇总 + 待分配池锁定。

### 工具层（pkg-24 → pkg-41）

- 16 个 `wb.*` 领域操作注册为模型工具（`wb_list_*` / `wb_get_*` / `wb_*_requirement` / `wb_*_version` / 评论四件套 / `wb_decide`），RPC 与工具共用同一套 `op*` 领域函数；破坏性工具标注「需 User 明确指令」。
- 缺口补齐（pkg-36）：`wb_add_artifact` / `wb_review` / `wb_comment` 扩展 author/role / inbox 落库 → 19 工具。
- **`wb_my_task`（pkg-41）**：确定性任务上下文 + `sessionsByNode` 全会话供给 → 22 工具；行为协议表内联、align 判定 = key 含「需求对齐」（sqlite 迁移后切 `mode` 字段）。

### 数据层

- JSON 载体（插件运行中）：`requirements.json` / `versions.json` / `<reqId>/{flow,artifacts,comments}.json` / `inbox.json`。
- **sqlite（`workbench.db`）**：11 表已建（roles / capabilities / flow_templates / flow_template_nodes / flows / flow_nodes / tasks / artifacts / issues / audit_log / meta），`schema_version=28`；定义层 seed 完成（9 角色 / 19 能力 / 4 模板 / 33 模板节点）；实例层由运行时写入或迁移脚本导入。

### 制度层 / 执行层

- team preset：PRD 定稿（`m6-team-preset.md` 原稿），**尚未接线**（从 standard 复制 + 叠加制度 + 挂 skill：`wb-agent-guide` / `team-conventions`）。
- M7 编排：req-019 演练验证 RD/FE/Code Reviewer/QA 四角色子 agent + Leader 编排全通过（join / 部分回滚正确）。
- 待落地：Architect / PM / Investigator 角色卡 + 验证；沉淀为 preset skill；子 agent 直接使用 wb_* 工具（替代文件直写）。

## 3. 验证记录

| 日期 | 场景 | 参与角色 | 结果 |
|---|---|---|---|
| 2026-08-15 | req-019 业务需求开发→评审→验收全流程（并行 join / 部分回滚） | RD / FE / Code Reviewer / QA（子 agent）+ Leader | ✅ 全通过 |

## 4. 遗留事项（后置）

1. `workbench-plugin/` 落盘源码与最新 pkg 对齐（重启可从源码恢复）。
2. sqlite 应用层落地：JSON 载体切换（wb_* 工具改走 sqlite 事务）；`wb_my_task` 组装链改 `flow_template_nodes.mode` / `capabilities.products` 展开。
3. team preset 接线 + skill 挂载。
4. M7 剩余角色卡（Architect / PM / Investigator）落地。
5. 需求对齐节点结论落 `issues(discussion)` 的实践沉淀。
6. 锚点 diff 级（产物版本化，M8 后置）。
7. 多项目切换 UI。