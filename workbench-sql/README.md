# 团队工作台 · sqlite 建库与种子数据（workbench-sql/）

> 依据：`docs/design/m8-flow-simplify.md` §3.4（能力模型）、§3.6.2（核心表结构）；`docs/design/m6-flow-templates.md`（tech/task/plan 节点定义）；`docs/design/m8-doc-templates.md`（模板 structure/quality_gates，已内联进 capabilities）。
>
> **模型 v0.15（User 拍板修正）**：roles → capabilities 为 **1 对多**（`capabilities.role_id` 唯一归属，**无 role_capabilities 关联表**）；评审能力按角色拆分（同一评审节点上各角色看各自维度——arch 看技术路线、fe 看接口契约、qa 看可测性、pm 看工期/价值、rd 看设计完整性、user 拍板）。
>
> 目标库：`.workbench/workbench.db`（由 `init.sh` 创建）。

---

## 1. 一键初始化

```bash
bash workbench-sql/init.sh
```

脚本行为（`set -euo pipefail`，可重复执行）：

1. `rm -f` 删除旧库 → 全新重建（幂等，重复执行 = 重置）
2. `sqlite3 workbench.db < schema.sql` 建 11 张表 + 索引
3. `sqlite3 workbench.db < seed.sql` 插入全部种子数据
4. 自检输出：表清单 / 种子行数 / capabilities 按 role_id 分布 / flow_template_nodes 分布 / JSON json_valid

可覆盖目标路径：`WORKBENCH_DB=/path/to/db bash workbench-sql/init.sh`

种子数据仅覆盖**定义层**（roles/capabilities/flows_templates/flow_template_nodes/meta）；实例层/tasks/artifacts/issues 为空，由运行时写入或 JSON→sqlite 迁移脚本导入。

---

## 2. 表结构说明（11 张表）

### 2.1 定义层（4 张）+ 实例层（5 张）

| 表 | 业务主键（TEXT UNIQUE） | 说明 |
|---|---|---|
| `roles` | `role_id` | 角色注册表：note 工作内容注释（一句话，非子 agent persona）；9 个固定角色 |
| `capabilities` | `capability_id` | 能力注册表：**role_id（归属角色，1 对多）** + how（能力本体）/ **products（产出物 JSON 数组，一能力可多产物；评审/决策类 `'[]'`）**。（v0.23：reviewer_focus/inputs/quality_gates 已删；v0.28：doc_path/template_path/demo_path 三列合并为 products；v0.29：products 项删 template_path——产出标准以产出 skill 为权威） |
| `flow_templates` | `flow_templates_id` | 4 种流程模板：scope_options（JSON，可选实施范围；节点拓扑见 flow_template_nodes） |
| `flow_template_nodes` | UNIQUE(flow_templates_id, node_order) | 模板节点定义：**flow_templates_id（逻辑引用 flow_templates.flow_templates_id）** / role_id / capability_id（执行者能力，可空=align 节点）/ **mode（produce｜align）** / gate / branch / parallel_group / **scope（裁剪条件 ''｜be｜fe）** / reviewers（[{role,capability_id,scope?}]）/ expected_outputs |
| `flows` | `requirement_id` | 流程实例：flow_templates_id（引用模板）/ current_node_id 指针 + 时间 |
| `flow_nodes` | UNIQUE(requirement_id, node_id) | 节点实例：state / reviewers（JSON 静态评审声明，v0.18）/ rolled_back_count / last_decision |
| `tasks` | `task_id` | 任务（唯一待办总线，v0.18 取代 inbox）：goal / context(JSON) / status / root_issue_id（锚定线程）/ **dispatch_at/last_ack/last_progress/resume_hint（派发与断点交接，v0.19）** |
| `artifacts` | `artifact_id` | 产物登记：label / href / kind(document|link) |
| `issues` | `issue_id` | 评论/问题/线程（单表 v0.20）：anchor 锚点族 / assignee / state / resolve 双路径字段 + **replies JSON（不可变回复数组，原 issue_replies 并入）** |

> **v0.17 合并（User 拍板）**：产物模板已并入 `capabilities` 表（doc_path / structure / quality_gates / version 内联，12 个产出能力各带完整模板），`document_templates` 独立表删除（15 张表）。前后端开发文档模板独立（backend-dev→rd/dev.md、frontend-dev→fe/dev.md）。
>
> **v0.18 简化（User 拍板）**：`inbox` 并入 `tasks`（task 成唯一待办总线，取代收件箱待办载体）；`flow_node_reviewers` 并入 `flow_nodes.reviewers`（JSON，静态评审声明，动态「谁过了没」由 tasks+issues 推导）；`snapshots` 删除（保留 `audit_log` 作 agent 每步改库的留痕）——**15 张 → 13 张**。
>
> **v0.19 再精简（User 拍板）**：`conversations` 删除——IM 是视图不是数据，通知（含待回复/节点动态）由查询 `tasks`/`issues` 实时推导，不预存通知表；`tasks` 加派发/断点交接字段（`dispatch_at`/`last_ack`/`last_progress`/`resume_hint`）承载通知可靠性与断点续接——**13 张 → 12 张**。
>
> **v0.20 线程合并（User 拍板）**：`issue_replies` 并入 `issues.replies`（JSON，不可变追加）——当前无按回复行查询需求，所有消费场景都是「读整个线程」，JSON 一次读更快；`audit_log` 对 replies 整段 diff 粒度可接受（m7：评论不可编辑/删除）——**12 张 → 11 张**。
>
> **v0.21 NULL 策略（User 拍板）**：非必要不使用 NULL（仅「空串有歧义/无值≠空串」时保留，见 §3）；`branch/parallel_group/note` 收紧为 `NOT NULL DEFAULT ''`；seed.sql 同步把显式 NULL 改为 `''`。**表数不变（11 张）**。
>
> **v0.22 全表时间戳（User 拍板）**：每张表增加 `created_at` / `updated_at`（unix ms，`DEFAULT`=当前时间毫秒，seed 自动填充）。旧等价字段改名对齐——`artifacts.added_at → created_at`、`issues.at → created_at`、`audit_log.at → created_at`（audit_log 也加 updated_at，恒=created_at，审计行不可变由应用层保证）；`flow_nodes.entered_at` 保留（业务语义=节点进入时间，≠行创建）。**表数不变（11 张）**。
>
> **v0.23 能力收缩 + 模板外置（User 拍板）**：`capabilities` 删除 `reviewer_focus` / `inputs` / `quality_gates`；新增 `template_path` / `demo_path`（指向 `docs/templates/xxx-template.md` / `xxx-demo.md`）。`how` 强化为能力本体（与 prd 对齐/接口满足前端/架构一致/可测性）；评审维度归入评审能力自身 how，依赖文档由 `task.context` 承载，质量门禁由模板文件承载。**表数不变（11 张）**。
>
> **v0.28 节点-能力绑定模型（User 拍板，m8 §10）**：`capabilities`.doc_path/template_path/demo_path 三列 → **单列 `products`**（JSON：`[{doc_path, template_path, demo_path}]`，一能力可多产物；评审/决策类 `'[]'`）；**QA 能力合并**（acceptance-test + defect-management → 一个「验收与缺陷管理」挂 2 产物 qa/test.md + qa/defect.md，defect-management 删除，能力 20→19）；`flow_template_nodes` 加 **`mode`（produce｜align）** 且 `capability_id` 可空（tech/task 需求对齐 = align 无产物节点，结论落 issues discussion，wb_my_task 组装链把全流程讨论打包给 subagent）；**reviewers 统一为 `[{role, capability_id, scope?}]`**（focus 移除——维度已在评审能力自身 how；leader 移出 reviewers——决策者≠评审表态者）。**表数不变（11 张）**。
>
> **v0.29 产出标准权威迁 skill（User 拍板 D4，semantica 47c53ed9）**：`products` 项删 `template_path`，现值 `[{doc_path, demo_path}]`——**DB 不存标准指针**：产出/评审端经产出 skill 寻址（`docs/templates/skills/`，frontmatter 声明动作绑定），评审端按 skill 自检节核对（评审能力 how 已加对应步骤）；有 skill 的 4 组模板文件删除（brd/prd/backend/frontend），无 skill 的 9 组模板保留标「skill 待建」；demo 保留并与 skill 骨架保持一致。`schema_version` 28→29。
>
> **实现状态（v0.30 泛化，User 拍板「维持 issues discussion 唯一载体 + 全会话按 node 供给」）**：wb_my_task 已先在 **JSON 插件**落地（`workbench-plugin/host.js` pkg-41，22 个 `wb_*` 工具）——**sessionsByNode 泛化**：不搞「特定会话当产物」，而是把**每个 node 里的所有会话（issue/discussion/notification 全含，包括锚定在产物文档上的会话）**都作为上下文产物，按 node 分组打包进返回（条条带 nodeId/nodeKey 明确归属；`anchor.artifactId` → artifacts 解析 `anchorArtifact={label,href}`；组内时间升序、replies 完整）；对齐节点判定 = key 含「需求对齐」（过渡）；sqlite 组装链（m8 §3.4.4/§11.2）为迁移目标，sessionsByNode 对应 SQL：`issues WHERE requirement_id=? ORDER BY node_id, created_at`（按 node_id 分组，`issue_type` 全含，`anchor_artifact_id` LEFT JOIN artifacts），mode 判定切换为 `flow_template_nodes.mode`。**会话不另产物化到 artifacts**（不新增文档、不登记 artifact 行），文档锚定会话仍是 issues 行、返回时解析产物引用；对话本体唯一载体 = issues。

### 2.2 支撑表（2 张）

| 表 | 业务主键 | 说明 |
|---|---|---|
| `audit_log` | — | 变更审计：table_name / row_id / action / field / old_value / new_value / by_who |
| `meta` | `meta_key` | 元信息：schema_version 等 |

> `audit_log` 无独立业务主键（m8 §3.6.2 原样），仅 `id INTEGER PRIMARY KEY AUTOINCREMENT`。`snapshots` 已删（v0.18，User 拍板：agent 交互项目管理，无需防篡改快照）。`issue_replies` 已并入 `issues.replies`（v0.20）。

---

## 3. 类型约定（m8 §3.6.2 v0.12 规范 + v0.15 时间类型修正）

> **v0.15 修正（User 要求）**：时间字段由 TEXT(ISO8601) 调整为 **INTEGER（unix 毫秒时间戳，epoch ms）**——与 Node/JS 生态（`Date.now()`）一致，可排序、可计算差值，避免字符串解析开销。

| 语义 | 类型 | 示例 |
|---|---|---|
| 内部行 id | `INTEGER PRIMARY KEY AUTOINCREMENT` | 全部 11 张表 |
| 业务主键 | `TEXT NOT NULL UNIQUE` | capability_id / role_id / requirement_id / task_id / artifact_id / issue_id / flow_templates_id |
| 时间 | `INTEGER`（unix ms，epoch 毫秒） | **created_at / updated_at（全部 11 表，v0.22）** + 业务时间：entered_at / done_at / dispatch_at / last_ack / last_reply_at / resolve_at / resolved_at |
| 状态/枚举 | `TEXT`（应用层白名单） | flow_nodes.state（未开始/进行中/待评审/已通过）、issues.state（open/pending-confirm/resolved/overridden） |
| 布尔/标志 | `INTEGER (0/1)` | gate / read / resolve_reviewer_confirmed |
| 计数/序号 | `INTEGER` | node_order / rolled_back_count / round_count / version |
| JSON 结构 | `TEXT`（sqlite 无 jsonb，存 JSON 文本；JSON1 扩展可查询） | capabilities.how、flow_template_nodes.reviewers 等 |

> 注意：`issues.last_decision` 为 JSON TEXT，其内部的 `at` 字段（如 `{verdict,reason,at,branch?}`）建议同样使用 unix ms，由应用层约定。

**NULL 策略（v0.21，User 拍板）**：**非必要不使用 NULL**——仅当「空字符串有歧义」或「字段语义上'无值'与'空串'必须区分」时才允许 NULL；其余一律 `NOT NULL` + 合理 DEFAULT（`''`/`0`/`'[]'`/`'{}'`）。

允许 NULL 的字段（三类语义，均已在 schema.sql 逐列注释说明）：

| 类别 | 语义 | 字段示例 |
|---|---|---|
| 模板缺省 | 该能力无产出模板（评审/决策类），空串无意义 | capabilities.products = `'[]'`（v0.28 起 JSON 数组表达，不再用 NULL 列；v0.29 起产物项内 demo_path 为 JSON null=示例未固化） |
| 无产出能力 | 需求对齐等无产物节点不绑产出能力，NULL 表示「无产出能力」且与 mode='align' 配套 | flow_template_nodes.capability_id |
| 未发生 | 事件尚未发生（时间/状态），空串对 INTEGER 无意义、0 不是合法时间戳 | flows.current_node_id、flow_nodes.feature_branch/entered_at/last_decision/pending_scope、tasks.root_issue_id/last_reply_at/last_ack/last_progress/resume_hint/dispatch_at/done_at/closed_at、issues.resolve_* 族、issues.anchor_* 族（kind 不适用）、audit_log.field/old_value/new_value（action 不适用） |
| 引用缺省 | 逻辑引用目标尚未确定，空串会与真实业务主键冲突 | issues.anchor_artifact_id / assignee / reviewer_role |

**收紧为 NOT NULL + DEFAULT**（v0.21）：`flow_template_nodes.branch/parallel_group`、`flow_nodes.branch/parallel_group`、`tasks.branch` 均 `NOT NULL DEFAULT ''`（`''`=非泳道/非并行，无歧义）；`artifacts.note` → `NOT NULL DEFAULT ''`。

**无外键约束**（v0.10 决策）：跨表关联全部为**逻辑引用**（`<父表>_id` 列）+ 普通索引（16 个 `idx_*`，见 schema.sql 末尾）。应用层（wb_* 工具）负责校验引用存在性。

---

## 4. 角色-能力模型（v0.15：1 对多）

**roles → capabilities 1 对多**：`capabilities.role_id` 唯一归属一个角色；一个角色拥有多个能力。**无关联表**。

- 「角色有哪些能力」→ `SELECT capability_id FROM capabilities WHERE role_id='rd'`
- 「能力属于谁」→ `SELECT role_id FROM capabilities WHERE capability_id='rd-tech-review'`
- 评审节点（gate=1）的 `reviewers[]` 各角色：用**各自拥有**的评审能力（角色→能力 1 对多解析），实现 m6 审批表「各审各的」——例如 RD 技术评审：arch→arch-tech-review（技术路线）、fe→fe-tech-review（接口契约）、qa→qa-tech-review（可测性）、pm→pm-review（工期/依赖）、rd→rd-tech-review（设计完整性/自测）、user→final-approve（拍板）

### 4.1 capabilities 19 项（按 role_id 分布）

| role_id | 能力 | 类型 |
|---|---|---|
| arch | product-design / arch-cap / alignment-cap / **arch-tech-review** | 3 产出 + 1 评审 |
| rd | backend-design / backend-dev / **rd-tech-review** | 2 产出 + 1 评审 |
| fe | frontend-design / frontend-dev / **fe-tech-review** | 2 产出 + 1 评审 |
| qa | case-design-cap / **acceptance-test（验收与缺陷管理，2 产物）** / **qa-tech-review** | 2 产出 + 1 评审 |
| cr | code-review | 评审 |
| pm | pm-review | 评审 |
| inv | research-cap | 产出 |
| leader | decide-cap | 决策 |
| user | final-approve | 拍板 |
| executor | plan-cap | 产出（任务流程动态执行者） |

> **v0.15 关键修正**：原 m8 §3.4.2 的 `role_capabilities` 多对多关联表被 User 否决——能力**不共享**（同一评审节点上各角色看的角度不同），故 `tech-review` 拆分为 4 个角色专属评审能力（arch/rd/fe/qa），`pm-review`（价值/排期）、`code-review`（代码）、`decide-cap`（门禁）、`final-approve`（拍板）各归其角色。`arch-tech-review` 覆盖调研评审（research-review）场景，避免孤儿引用。
> **v0.28 QA 合并（§10.1-3）**：acceptance-test 与 defect-management 合并为一个能力「验收与缺陷管理」，`products` 挂 2 产物（qa/test.md + qa/defect.md，成绩验收节点同时产出两文档）；defect-management 删除——能力 20 → **19 项**。
> **executor 说明**：任务流程（task）的执行者由 User/Leader 动态指定（m8 §3.3 / m6 §三），非固定 9 角色之一；模板层用 `executor` 占位（与 m8-doc-templates 的 plan 模板 role_id 一致），开发能力在实例化时按实际执行者角色解析（rd→backend-dev / fe→frontend-dev），模板默认 backend-dev。

### 4.2 capabilities.inputs 已删除（v0.23）——依赖由 task.context 承载

> **v0.23 决策（User 拍板）**：`capabilities.inputs`（依赖文档清单）已删除。理由：任务派发时主 agent 已在 **task.context.dependencyDocs** 给出相关文档（该类型任务有 prd 就给 prd），DB 里再存一份 inputs 冗余且易漂移。能力不再声明"读哪些文档"，聚焦于"拿到上下文后如何做出合格产出"（how）。

**消费语义（仍生效）**：task.context.dependencyDocs 遵循"条件读取"——有则读、缺失跳过、不阻塞（引 m8 §3.4.4 实例化时按实际存在过滤）。

---

## 5. 种子数据清单

| 表 | 行数 | 内容 |
|---|---|---|
| `roles` | 9 | rd / fe / cr / arch / pm / qa / inv / leader / user（note 一句话工作内容注释） |
| `capabilities` | 19 | 见 §4.1（how 均 4 步：读输入→产出/评审→自检→登记；role_id 唯一归属；v0.28 QA 合并后 19 项） |
| `flow_templates` | 4 | business / tech / task / plan（scope_options） |
| `flow_template_nodes` | 33 | business 13（含待启动）+ tech 9 + task 6 + plan 5 |
| `capabilities` | 11 个产出能力 | 产出能力 products 有 doc_path + demo_path（v0.29 起无 template_path——产出标准按能力→skill 寻址，**已建 skill 4 类**：to-brd / to-prd / to-rd-techdesign / to-fe-techdesign；无 skill 的产物暂由 docs/templates/*-template.md 承载「skill 待建」，索引见 docs/templates/README.md）；backend-dev / frontend-dev / plan-cap 3 项示例未固化（products 项内 demo 为 JSON null）；评审/决策 8 项 products='[]'；acceptance-test 挂 2 产物 |
| `meta` | 1 | schema_version=29 |

### 5.1 flow_template_nodes 分布（按 m6 / m8 节点定义）

| 模板（flow_templates_id） | 节点数 | 节点 key（node_order 顺序） |
|---|---|---|
| business（1） | 13 | 待启动→产品设计→需求评审→RD 技术设计→RD 技术评审→FE 技术设计→FE 设计评审→用例设计→用例设计评审→[dev-1: RD 开发∥FE 开发]→QA 验收→用户验收 |
| tech（2） | 9 | 待启动→需求对齐→RD 技术设计→RD 技术评审（按需）→FE 技术设计（按需）→用例设计→[dev-1: 并行开发]→QA 验收→用户验收 |
| task（3） | 6 | 待启动→需求对齐→方案评审→开发→QA 验收→用户验收 |
| plan（4） | 5 | 待启动→立项（需求对齐）→调研→架构规划→创建需求·启动版本 |

> 节点 `capability_id` = 执行者（role_id）自身拥有的能力；评审节点 reviewers 各自的能力由「角色→能力 1 对多」解析（§4）。

---

## 6. 用 MCP toolbox 校验表结构

建库后可用 Google MCP Toolbox for Databases 校验：

- `mcp__toolbox__list_tables`：列出全部 11 张表，核对列类型 / 主键 / 约束（INTEGER PK AUTOINCREMENT、业务主键 TEXT UNIQUE、无 FOREIGN KEY）
- `mcp__toolbox__execute_sql`：执行校验查询，例如：

```sql
-- 种子行数
SELECT 'roles' t, count(*) FROM roles UNION ALL
SELECT 'capabilities', count(*) FROM capabilities UNION ALL
SELECT 'flow_templates', count(*) FROM flow_templates UNION ALL
SELECT 'flow_template_nodes', count(*) FROM flow_template_nodes UNION ALL
SELECT 'meta', count(*) FROM meta;

-- business 节点数（应 13，含待启动；flow_templates_id='business'）
SELECT count(*) FROM flow_template_nodes WHERE flow_templates_id='business';

-- 1 对多一致性：节点执行者必须拥有节点能力（应全 OK；
--   例外：task 流程 role_id='executor' 为动态占位（§7.3/7.4），开发能力实例化时映射到实际角色，
--   该行 role_id='executor' 与 backend-dev.role_id='rd' 不相等属预期，其余行应全 OK）
SELECT t.flow_templates_id, n.key, n.role_id, n.capability_id, c.role_id AS cap_owner
FROM flow_template_nodes n
JOIN flow_templates t ON t.flow_templates_id = n.flow_templates_id
JOIN capabilities c ON c.capability_id = n.capability_id
WHERE n.role_id != c.role_id AND n.role_id != 'executor';

-- 产出能力 products 完整性：products 非空的产出能力，其第 1 项产物应有 doc_path（示例是否固化另查 demo_path）
SELECT count(*) FROM capabilities WHERE json_array_length(products) > 0 AND json_extract(products, '$[0].doc_path') IS NULL;

-- align 节点检查：mode='align' 应无能力、无产物（应 0 行）
SELECT count(*) FROM flow_template_nodes WHERE mode='align' AND (capability_id IS NOT NULL OR expected_outputs <> '[]');

-- reviewers 结构：每条评审位必须带 capability_id 且能力存在（应全 OK）
SELECT count(*) FROM flow_template_nodes n, json_each(n.reviewers) e
LEFT JOIN capabilities c ON c.capability_id = json_extract(e.value, '$.capability_id')
WHERE json_extract(e.value, '$.capability_id') IS NULL OR c.capability_id IS NULL;

-- JSON 合法性（应全为 0 非法）
SELECT sum(json_valid(how)) + sum(json_valid(products)) FROM capabilities;
```

> 若 toolbox 不可用，等价替代：`sqlite3 .workbench/workbench.db ".schema"`（表结构）+ 上述 SQL（数据）。

---

## 7. 已知设计注记

1. **表数演进**：m8 §3.6.2 原 SQL 为 17 张（含 `role_capabilities` 关联表）→ v0.15 删关联表为 16 张 → v0.17 删 document_templates 为 **15 张** → v0.18 删 inbox/flow_node_reviewers/snapshots、增 conversations 为 13 张 → v0.19 删 conversations 为 12 张 → **v0.20 删 issue_replies 为 11 张**（4 定义层 + 5 实例层 + 2 支撑层）。
2. **m8 §4.2 标题"技术需求（7 节点 + 待启动）"与其表格（8 个业务节点）不一致**：m6 §二 明确 8 个业务节点（需求对齐/RD 设计/RD 评审/FE 设计/用例/并行开发/QA 验收/用户验收），按 m6 落 **9 行**（8 业务 + 待启动）。
3. **task 流程 role_id='executor'**：执行者由 User/Leader 动态指定（m8 §3.3 / m6 §三），非固定 9 角色之一；模板层用 `executor` 占位（与 m8-doc-templates plan 模板 role_id 一致），开发能力实例化时按执行者角色解析。
4. **executor 的 plan-cap 与 roles 表**：`capabilities.role_id='executor'` 在 roles 表 9 行中无对应行（executor 是动态占位，非固定角色）；`flow_template_nodes` 中 role_id='executor' 同理。应用层（wb_my_task）实例化时映射到实际角色。
5. **acceptance 并入 test-result（User 拍板，保证能力→模板 1:1）**：原 12 个模板中 `acceptance`（《验收记录》）已并入 `test-result`，合并为《测试用例与验收记录》（`qa/test.md`，9 节：测试范围/AC 核对/用例明细/联调结果/验收结论/目标达成核对/缺陷遗留/交付物/用户意见）。`final-approve.inputs` 去掉 `qa/accept.md`。`accept.md` 路径废弃，m8-doc-templates.md 的 acceptance 章节仍作历史参考。
6. **capabilities.output_template 字段已废弃（v0.17 合并）**：产物模板不再独立成表，模板字段（doc_path/structure/quality_gates/version）直接内联进 capabilities；评审/决策类能力这些字段为 `NULL`。
7. **关联命名一致性（User 拍板）**：全库子表关联列名 = 父表业务主键名（`<父表>_id` 同名关联）——
   - `capabilities.owner_role` → **`capabilities.role_id`**（引用 `roles.role_id`）
   - `flow_templates.flow_type` → **`flow_templates_id`**（业务主键更名，值仍为 business/tech/task/plan）
   - `flow_template_nodes.flow_type`/`template_id` → **`flow_templates_id`**（引用 `flow_templates.flow_templates_id`，TEXT 业务主键，非 INTEGER id）
   - `flow_template_nodes.owner_role` / `flow_nodes.owner_role` → **`role_id`**（capabilities 内联模板不再有独立 owner_role）
   - `tasks/artifacts/issues.req_id` → **`requirement_id`**（引用 `flows.requirement_id`）
   关联语义：**子表关联列与父表业务主键同名同值（TEXT）**；行 id（INTEGER）仅作内部主键，不做跨表关联。
8. **表间关联矩阵（v0.16，id 同名关联约定）**：

   | 父表（业务主键） | 子表关联列 | 基数 | 说明 |
   |---|---|---|---|
   | roles（role_id） | capabilities.role_id | 1 : n | 一个角色拥有多个能力 |
   | roles（role_id） | flow_template_nodes.role_id | 1 : n | 节点执行者 |
   | roles（role_id） | flow_nodes.role_id | 1 : n | 实例节点执行者 |
   | roles（role_id） | tasks.role_id | 1 : n | 任务执行者 |
   | roles（role_id） | issues.role_id | 1 : n | 评论作者 / 回复者（回复在 issues.replies JSON 内） |
   | capabilities（capability_id） | flow_template_nodes.capability_id | 1 : n | 节点用什么能力 |
   | flow_templates（flow_templates_id） | flow_template_nodes.flow_templates_id | 1 : n | 模板→节点定义 |
   | flow_templates（flow_templates_id） | flows.flow_templates_id | 1 : n | 模板→流程实例 |
   | flows（requirement_id） | flow_nodes.requirement_id | 1 : n | 流程→节点实例 |
   | flows（requirement_id） | tasks.requirement_id | 1 : n | 流程→任务 |
   | flows（requirement_id） | artifacts.requirement_id | 1 : n | 流程→产物 |
   | flows（requirement_id） | issues.requirement_id | 1 : n | 流程→评论 |
   | flow_nodes（requirement_id+node_id） | tasks.node_id / artifacts.node_id / issues.node_id | 1 : n | 节点→任务/产物/评论 |
   | flow_nodes（node_id） | flows.current_node_id | 1 : 1 | 当前指针（可空） |
   | issues（issue_id） | issues.anchor_artifact_id（→artifacts.artifact_id） | 1 : n | 评论锚点指向产物 |
   | tasks（task_id） | tasks.root_issue_id（→issues.issue_id） | 1 : n | 待办→锚定线程（可空） |

9. **CAPABILITIES 与 DOCUMENT_TEMPLATES 合并（User 拍板，v0.17）**：产物模板独立表删除，模板字段（doc_path/structure/quality_gates/version）内联进 capabilities（15 张表，v0.20 已精简至 11 张）。**冲突处理**：
   - dev-notes 不再被前后端共享——拆为 backend-dev（`rd/dev.md`）与 frontend-dev（`fe/dev.md`）各自模板（前后端开发文档应不同，可独立演进）
   - 节点 `expected_outputs` 的 `template_id` 改为 `capability_id`（产出物 = 引用能力）；QA 验收节点可引用 acceptance-test + defect-management 两个能力
   - `wb_my_task` 组装链相应改为：节点 capability_id → capabilities（how/structure/quality_gates 直接内联，不再经 document_templates 表）
10. **artifacts 与 flow_nodes 不合并（User 拍板，v0.20）**：产物**独立表保留**——1 节点可有多产物（QA 验收=test.md+defect.md、并行开发=rd/fe 各一份）；评论锚点 `anchor_artifact_id` 需精确指向单条产物；产物登记是低频 INSERT（独立行审计精确），而 flow_nodes 是高频状态推进（合并会放大单行写冲突）。「产物挂在节点下」由 **JOIN 聚合视图**实现，不合并表。
11. **v0.28 编排核对修复（m8 §10 落库）**：research-cap 模板回填（`research-open-source-tech-template/demo.md`，此前已建档但 seed 未引用）；tech①/task① 需求对齐修正为 `mode='align'` + `capability_id=NULL`（原误用 rd-tech-review / plan-cap）；QA 验收节点 expected_outputs 两条引用都指向合并后的 acceptance-test；节点 reviewers 全量统一 `[{role,capability_id,scope?}]` 对象（原混有裸字符串与 {role,focus} 对象）。