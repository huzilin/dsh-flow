# 领域模型与存储

> 数据层唯一真相源：实体字段、状态机、存储形态。
> 交互层的状态机行为见各自模块文档；本文件给出精确转移定义，模块文档只引用。

---

## 1. 存储形态与演进

| 阶段 | 载体 | 状态 |
|---|---|---|
| 起步（M1~） | `<workspace>/.workbench/*.json` 文件存储，透明可 diff | ✅ 插件当前运行载体 |
| 目标（已建库） | `<workspace>/.workbench/workbench.db`（sqlite，11 表，schema_version=28） | ✅ 定义层已 seed；实例层迁移进行中 |

- **多项目**：项目 = DSH workspace 目录；所有实体以 `requirement_id` 归属流程实例，`versions.json`/`requirements.json` 为项目级全局文件（每 workspace 一套）。
- **JSON 载体文件映射**：`requirements.json`、`versions.json`、`inbox.json`（项目级）；`<reqId>/flow.json`、`artifacts.json`、`comments.json`（需求级）。
- **迁移方向**：JSON → sqlite（WAL，单写多读）；审计靠 `audit_log` 表，不依赖 git diff；JSON 保留为导出格式。
- **无外键约束**：跨表关联 = 子表关联列与父表业务主键**同名同值**（逻辑引用），应用层（wb_* 工具）校验；表演进走 `meta.schema_version` + 迁移脚本。
- **类型约定**：内部行 id 用 `INTEGER AUTOINCREMENT`；业务主键 TEXT UNIQUE（可读、跨系统引用）；时间 INTEGER（unix 毫秒，与 `Date.now()` 一致）；状态/枚举 TEXT（应用层白名单）；布尔 INTEGER(0/1)；计数 INTEGER；JSON 结构 TEXT。
- **NULL 策略**：非必要不使用 NULL；仅「空串有歧义」或「无值 ≠ 空串」（未发生时间/未指派引用）时允许。

## 2. 实体定义

### 2.1 Version（版本）

| 字段 | 说明 |
|---|---|
| `id` | 稳定短码 `v-<n>`；`v-backlog` 固定为「待分配」保留项（不可删/不可改/无状态操作） |
| `name` | 展示名，自由文本（`v1.0`、`2.0 迭代`…） |
| `state` | `待启动 / 进行中 / 已发布 / 已废弃` |
| `isBacklog` | 是否待分配池（仅 `v-backlog` 为 true） |

**状态机**（详见 `module-board/rd/design.md` §3.6）：`待启动 → 进行中 → 已发布` 单向推进；任意阶段可直接「废弃」（终态，v1 不可恢复）；废弃联动其下需求 `status → 已取消`。

### 2.2 Requirement（需求）

| 字段 | 说明 |
|---|---|
| `id` | `req-<序号>`（递增补零，如 `req-007`） |
| `title` | 标题（必填） |
| `description` | 需求描述（Markdown，可含代码块/链接） |
| `category` | `需求任务 / 任务 / 版本规划任务`（决定流程模板） |
| `reqType` | `业务需求 / 技术需求`（仅需求任务；`category=需求任务` 时区分） |
| `scope` | `仅前端 / 仅后端 / 前后端都含`（裁剪 FE/RD 侧节点，见 `06`） |
| `versionId` | 归属版本 `id` 或 `v-backlog` |
| `priority` | `P0 / P1 / P2 / P3` |
| `status` | `待启动 / 进行中 / 已完成 / 已取消` |
| `owner` | 负责人（角色）；owner 规则见 `06` |
| `history[]` | 编辑审计（JSON 载体）；`{at, by, changes:[{field, from, to}]}`，仅字段级变化时追加 |
| `createdAt / updatedAt` | 创建/更新时间 |

### 2.3 Flow（流程实例）与 FlowNode（节点实例）

```
Flow:   requirement_id, flow_templates_id(business|tech|task|plan),
        current_node_id（当前指针；并行组内指向进行中分支）, created_at, updated_at

FlowNode: node_id(n-00…), node_order, key, role_id（执行者角色）,
          gate(0/1), branch(''|rd|fe), parallel_group(''|dev-1), feature_branch,
          state, entered_at, rolled_back_count, last_decision,
          pending_scope, reviewers[]（静态评审声明 JSON）
```

- **节点独立**：一条流程标签一个节点，可独立回退重入；通过 → 前进下一节点；打回 → 回退上一节点。
- **reviewers[]**：`[{role, capability_id, scope?}]`（定稿：leader 不在 reviewers 中，决策者 ≠ 评审表态者）；动态「谁过了没」由评审表态 + 评论闭环推导。
- **并行组**：同 `parallel_group` 的节点并行；join = 组内全部 `state=已通过` 才放行组后节点（详见 `06`）。

### 2.4 Issue（评论 / 问题 / 线程）

```
id, requirement_id, node_id（挂载节点）, anchor{kind, artifactId, line?, lineText?,
    startLine?, endLine?, blockStart?, blockEnd?, lang?},
author, role_id, reviewer_role?, issue_type(issue|notification|discussion),
assignee（解决责任人）, text, replies[]（不可变回复数组 JSON）, state,
round_count, last_reply_at, resolve 族（by/at/note/path/reviewer_confirmed）
```

- **统一模型**：任何评论 = 一个待解决问题（不区分「评审 Issue」与「普通评论」）。
- **锚点四级**：`document`（文档级）/ `line`（单行）/ `range`（范围）/ `block`（代码块）；`heading` 章节级已废弃，diff 级后置。
- 评论**不可编辑/删除**（保留审计）；误发用 reply 标注「作废」。

### 2.5 Artifact（产物）

```
id(art-<ts>), requirement_id, node_id, label, href, kind(document|link),
note, added_by（登记人）, created_at, updated_at
```

- 产物即「文件行」：文件名 + href + note；文档正文由作者用文件工具写入项目目录，登记只写元数据。
- 一节点可多产物（QA 验收 = test.md + defect.md；并行开发 = rd/fe 各一份）。

### 2.6 Task（任务，唯一待办总线）

```
task_id(t-<ts>-<seq>), requirement_id, node_id, branch, role_id（执行者）,
task_type(produce|review|accept|decide), goal, context（确定性上下文 JSON）,
status, root_issue_id（锚定线程）, reply_count, last_reply_at,
dispatch_at, last_ack, last_progress, resume_hint, done_at, closed_at
```

- **TASK 是唯一待办总线**：取代收件箱待办载体；通知是视图不是数据——任何通知/待回复由查询 `tasks`（awaiting-reply/未 ack/进度陈旧）+ `issues`（pending-confirm）实时推导，不预存通知表。
- **派发/断点交接**：`dispatch_at`（证明已派出）/ `last_ack`（心跳，NULL=未送达需重发）/ `last_progress`（执行到哪一步）/ `resume_hint`（从哪续做）——重启后扫描「已派未确认」「进度陈旧」即可断点续接。

### 2.7 支撑表

| 表 | 说明 |
|---|---|
| `audit_log` | 变更审计：`{table_name, row_id, action, field, old_value, new_value, by_who, at}`，agent 每步改库留痕 |
| `meta` | 元信息：`schema_version` 等 |

## 3. 状态机（精确转移）

### 3.1 节点状态（FlowNode.state）

| 当前 | 事件 | 下一态 | 副作用 |
|---|---|---|---|
| 未开始 | 进入 | 进行中 | 记 `entered_at` |
| 进行中 | 产出完成（gate 节点） | 待评审 | 评审者初始化待表态 |
| 待评审 | 全部问题闭环 + User 通过 | 已通过 | `current_node_id` 前移下一节点 |
| 待评审 | 打回（reason 必填） | 进行中 | `rolled_back_count++`；上一节点回退为进行中；可产生 `pending_scope` |
| 已通过 | 被下一节点打回 | 进行中 | `rolled_back_count++` |

> 打回语义：回退到**上一个节点**重新产出（非当前节点内打转）。部分回滚（并行组）：`targetBranch` 指定分支，仅该分支回滚，另一分支保持已通过。

### 3.2 评论状态（Issue.state）

```
open ──提评后讨论──▶ open（thread 回复、round_count++）
  ├─ A 修复关闭：assignee resolve(note=修复说明) ──▶ pending-confirm（待提评者复核）
  │     ├─ 提评者 confirm ──▶ resolved（reviewer_confirmed=true，门禁放行）
  │     └─ 提评者 reopen ──▶ open（round_count++，通知 assignee）
  ├─ B 协商一致关闭：提评者 resolve(note=一致结论, path=agree) ──▶ resolved
  ├─ round_count > 5 ──▶ 通知 Leader/User 介入（拉锯升级）
  └─ 强权 override（User 明确指令）──▶ overridden（绕过门禁）
```

- **谁 resolve**：A 路径 = assignee（解决责任人）；B 路径 = 提评者；resolve 必须带 note。
- **谁 confirm/reopen**：仅提评者（不能自确认）。
- **assignee 规则**：普通评审意见 → 锚定产物作者（谁的产物谁改）；QA 验收缺陷 → 显式指派 RD|FE（配合 `targetBranch` 部分回滚）。
- **门禁判定**：`wb_decide pass` 前置 = 无 open + 无 pending-confirm + 全部 resolved 且 `reviewer_confirmed=true`（或 overridden）。
- **待复核超时提醒**：pending-confirm 挂起超时 → 提醒提评者复核（提评者 = User 阈值 1 小时；agent 5 分钟；轮询每 5 分钟）。

### 3.3 任务状态（Task.status）

```
todo → in-progress（执行者开始）→ done（产物登记/评审表态）→ closed（终态）
     ↘ awaiting-reply（等回复/表态）→ done | rejected
rejected → in-progress（返工重做）
```

### 3.4 需求状态与版本状态

- 需求 `status`：`待启动 / 进行中 / 已完成 / 已取消`；废弃版本联动 → 已取消；收尾（最后节点通过）→ `已完成`。
- 版本状态机见 `module-board/rd/design.md` §3.6。

## 4. 审计与一致性

- **审计即证据**：需求字段变更写 `history[]`（谁将什么从什么变更为什么）；流程节点状态由 `decide` 驱动，不手改数据；sqlite 侧由 `audit_log` 全表留痕。
- **审查一致性**：节点执行者必须拥有节点绑定的能力（`role_id` 匹配 `capabilities.role_id`）；唯一例外 task 流程 `executor` 动态占位。
- **写策略**：所有 `wb.*` 写操作返回最新快照（纯 JSON、lossless）；单写入者（Leader 主导），无并发锁；sqlite WAL 单写多读。