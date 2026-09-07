# 模块 III/IV：门禁通知收件箱 与 TODO 列表


> ⚠️ **历史归档（2026-09-08 标注）**：本文件为 arch_design 拆分前旧版快照——现行权威见 `docs/v1.0/`（PRD/RD/FE 分册）与 `docs/v1.0/arch_design.md`；交互层 v3 增补见 `docs-tmp/ui-interaction-final.md`。仅作历史参考，勿据以实现。

> **地位**：本文档是模块 III（门禁通知收件箱）＋模块 IV（TODO 列表）的需求＋设计合并稿。
> 收编原 `team-workbench.md` §2.3/§2.4/§3.4 与 `m8-flow-simplify.md` §3.3 的收件箱/任务总线内容，只保留定稿结论。
> 两模块同源：**TODO 是收件箱的可操作投影**。
> 职责边界：本文只覆盖收件箱条目与 TODO；不含流程树（见 `03-module-flow-tree.md`）与看板（见 `02-module-board.md`）。

---

## 1. 定位

**收件箱不是一次性门禁提醒，而是「必须处理的 IM 收件箱」**：

- 任何需要 User 回应的沟通都进来：待评审门禁、待界定新范围、产出修订完成、产出者反驳、要求补充答复、对话待回复；
- **逐条必办、双向可回应**：收件箱是 User 与执行侧对话的自然面，不是单向提醒；
- **TODO 是收件箱的可操作投影**：同一批待办换一个更聚合、更可执行的呈现，供输入框旁常驻。

> 演进注记：v0.19 起 TASK 为唯一待办总线，通知/待回复由查询 `tasks`/`issues` 实时推导，不预存通知表（决策 3，见 `01-architecture.md`）。

## 2. 收件箱条目类型（模块 III）

六类条目，均需 User 处理：

| 类型 | 触发 | 来源 |
|---|---|---|
| 待评审门禁 | 节点进入「待评审」 | 流程状态变更 |
| 待界定新范围 | 打回理由中引入新范围，边界待 User 界定 | 产出者反馈 |
| 产出修订完成 | 产出者按 User 打回意见修订完毕，待复验 | 产出者 |
| 产出者反驳 | 产出者就 User 打回意见提出异议 | 产出者 |
| 要求补充答复 | 产出者发现 User 的答复有遗漏，要求 User 补充 | 产出者 |
| 对话待回复 | 任一方向有未回复的沟通 | 双向 |

## 3. 交互设计（模块 III · 收件箱）

### 3.1 呈现

- 形式：`shell.overlay` badge/toast + 未读数；点击进入收件箱列表；条目点击跳对应节点评审区（模块 II 流程树上下文，看产物与评论）。
- DSH 挂靠：可复用现有事件形态 `approval/request`、`session/event`、`goal/changed`、`domain/changed`；业务收件箱走**领域数据状态变更 + 客户端订阅**，不污染系统 approval 语义。

### 3.2 全局聚合与筛选

- 收件箱为 **User 全局**（跨项目聚合，决策 6）。
- **两层可叠加筛选**：项目筛选（按 workspace）× 任务名筛选（按需求/任务，关键词搜索）。
- 每条目冗余 `projectId` + `requirementId`/`requirementTitle`，查询时按此过滤/搜索；`requirementTitle` 冗余避免跨文件 join。

### 3.3 交互原则

**不打断、可聚合、可点击进入、逐条可回应、可筛选**：

- 不打断：badge/toast 提示，不弹窗抢占；
- 可聚合：未读数 + 列表，消息多不淹没（靠 3.2 两层筛选）；
- 可点击进入：条目 → 对应节点评审区；
- 逐条可回应：每条在原上下文中可回复 / 表态 / 关闭。

## 4. TODO 列表（模块 IV）

### 4.1 条目（四类）

| 条目 | 说明 |
|---|---|
| 待评审节点（门禁） | 同收件箱「待评审门禁」 |
| 待界定新范围 | 同收件箱「待界定新范围」 |
| 打回后等待产出者修订（追踪中） | 打回后进入等待，追踪产出者修订进展 |
| 我的意见待回复 / 待补充答复（对话追踪） | 追踪 User 发言后待回复、以及需补充的答复 |

### 4.2 放置位

`conversation.input.dock`（输入框上方；已有 todo/goal/queue 先例）。

### 4.3 聚合呈现

以「**未完成数 + 分类**」聚合：一眼看清还有哪些没做、集中在哪类；点击条目跳对应节点。

### 4.4 与收件箱的关系

**TODO = 收件箱的操作投影**：同源（同一批待办数据）、同筛选（项目 × 任务名两层），区别只在呈现粒度——收件箱以条目/对话为准，TODO 以「未完成工作」为准，适合常驻输入框旁。

## 5. 数据与规则（设计）

### 5.1 条目字段

- 真相源：`05-domain-model.md` §2.6 **Task 实体**（字段表不重写，只引用）；收件箱/TODO 条目是 `tasks`/`issues` 的**查询投影**，不建独立通知表（决策 3）。
- 投影行携带：`taskId/issueId`、`projectId`、`requirementId`、`requirementTitle`（冗余，供两层筛选）、`nodeId`、类型（§2 六类 / §4.1 四类的投影规则）、`summary`、状态（未处理/已处理）。
- 视图快照如需落盘：JSON 载体 `inbox.json` 与 sqlite 对应表使用同一 schema（见 `05` §1 载体映射；不另建通知表）。

### 5.2 触发点

| 触发 | 产生条目 |
|---|---|
| `decide` 推进 / 节点进入待评审 | 待评审门禁；打回带新范围 → 待界定新范围 |
| 评论（`wb_comment` 提及/归属） | 产出者反驳 / 要求补充答复 / 对话待回复 |
| 评审待表态（角色评审 / User 终审） | 待评审门禁 |
| 缺陷指派（QA 传 `domain=rd\|fe`，程序自动指派） | 返工修订 → 产出修订完成待复验 |

### 5.3 断点续接与待回复推导

- **派发/断点字段**（Task 实体，定义见 `05` §2.6，不重写）：`dispatch_at`（证明已派出）/ `last_ack`（心跳，NULL=未送达需重发）/ `last_progress`（执行到哪一步）/ `resume_hint`（从哪续做）。
- **重启续接**：扫描「已派未确认」（`dispatch_at` 有值且 `last_ack` NULL → 重新通知）与「进度陈旧」（`in-progress` 但 `last_progress` 久未更新 → 从 `last_progress`/`resume_hint` 续做，不重来）。
- **待回复视图**：`tasks WHERE status='awaiting-reply'` + `issues WHERE state='pending-confirm'` —— 实时查询推导，不预存通知表（决策 3；状态机见 `05` §3.2/§3.3）。

### 5.4 对话本体唯一载体 = issues

对话本体唯一载体是 `issues`（根评论 + `replies` JSON 内嵌回复，见 `05` §2.4）；IM/收件箱只是**带跳转的提示**——点开跳对应节点看产物与完整回复链。执行侧由 `wb_my_task` 以 `sessionsByNode` 按 node 分组打包全会话供给（含锚定在产物文档上的会话，每条带 nodeId/nodeKey 归属；契约见 `07-tools-and-orchestration.md`），收件箱与执行侧读同一份对话真相源。

## 6. 定稿要点

1. 收件箱 = **必须处理的 IM 收件箱**：任何需 User 回应的沟通都进来，逐条必办、双向可回应、可点击进入、可筛选；非一次性门禁提醒。
2. 六类条目：待评审门禁 / 待界定新范围 / 产出修订完成 / 产出者反驳 / 要求补充答复 / 对话待回复。
3. 收件箱为 **User 全局**（跨项目聚合），「项目筛选 × 任务名筛选」两层叠加；条目冗余 `projectId`+`requirementId/requirementTitle` 支撑筛选、免跨文件 join。
4. 形态：`shell.overlay` badge/toast + 未读数 → 收件箱列表 → 跳对应节点评审区；不打断、不弹窗抢占。
5. TODO = 收件箱的**可操作投影**，同源同筛选，四类条目，放 `conversation.input.dock`（todo/goal/queue 先例），以「未完成数 + 分类」聚合。
6. **TASK 是唯一待办总线**（决策 3）：收件箱条目是 `tasks`/`issues` 的查询投影，不建通知表。
7. 契约：投影行字段以 `05` Task 实体为准；视图快照的 JSON 载体 `inbox.json` 与 sqlite 表同 schema。
8. 断点续接靠 `dispatch_at`/`last_ack`/`last_progress`/`resume_hint`；重启扫「已派未确认」「进度陈旧」续做，不重来。
9. 待回复视图 = `tasks(status='awaiting-reply')` + `issues(state='pending-confirm')`，实时推导。
10. 对话本体唯一载体 = `issues`（含 `replies`）；收件箱只是带跳转的提示；执行侧 `wb_my_task`（`sessionsByNode`）供给同一真相源。