# wbflow v1.0 重新实现 · 技术规格（Tech Spec）

> **状态**：2026-09-08 定稿稿（待 User 确认接缝与两项默认假设：MySQL 8.0 / 幂等键）。设计权威=五件套（见 Further Notes）；本 spec = 实现层规格，冲突时以设计五件套为准。
> **技术栈（User 指定）**：数据库 **MySQL** · 后端 **Go + Gin** · 前端 **Web**（本期实现在 web；保留嵌入 DSH 的能力——API-first 双嵌入）。
> **最高接缝**：**HTTP API**——wb_* 语义一份、HTTP 一张皮。Web 前端、未来 DSH 嵌入（conversation.view 槽）、agent 侧（DSH subagent 的 wb_* 工具=HTTP client）统一消费同一 API 层。测试直接打 HTTP。

---

## 0. 数据模型（MySQL 8.0 DDL）

> 由 sqlite v0.29 设计（data-tools-final §1）翻译为 MySQL 8.0。约定：id BIGINT UNSIGNED AUTO_INCREMENT；业务主键 TEXT→VARCHAR(64) UNIQUE；枚举/状态 VARCHAR；布尔 TINYINT(1)；JSON 用 MySQL 原生 JSON 类型；时间 BIGINT（unix ms，与既有 epoch ms 约定一致）；字符集 utf8mb4；引擎 InnoDB。无外键约束（跨表逻辑引用 + 索引，沿用既有约定）。

### 0.1 建表 DDL

```sql
-- 1. 能力注册表
CREATE TABLE capabilities (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  capability_id  VARCHAR(64)  NOT NULL UNIQUE,   -- 'backend-dev' | 'arch.prd-review' | …
  name           VARCHAR(128) NOT NULL,
  role_id        VARCHAR(32)  NOT NULL,
  inputs         JSON         NOT NULL,           -- [{productType, use_threads}]
  actions        JSON         NOT NULL,           -- [{action_id, name, on, skill?, inline?, outputProductTypes}]
  products       JSON         NOT NULL,           -- [{doc_path, template_path, demo_path}]
  created_at     BIGINT       NOT NULL,
  updated_at     BIGINT       NOT NULL,
  INDEX idx_capabilities_role (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. 角色注册表
CREATE TABLE roles (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  role_id  VARCHAR(32)  NOT NULL UNIQUE,          -- rd/fe/qa/arch/pm/cr/inv/leader/user/executor
  name     VARCHAR(64)  NOT NULL,
  note     VARCHAR(512) NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3. 流程定义（原 flow_templates）
CREATE TABLE flow_defs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  flow_defs_id  VARCHAR(32) NOT NULL UNIQUE,       -- business/tech/task/plan
  scope_options JSON        NOT NULL,              -- business/tech: ["full","rd","fe"]; task: ["rd","fe"]; plan: []
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4. 定义节点（原 flow_template_nodes）
CREATE TABLE flow_def_nodes (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  flow_defs_id     VARCHAR(32) NOT NULL,
  template_node_id VARCHAR(32) NOT NULL UNIQUE,    -- 'bus-03' 业务可读稳定 id
  node_order       INT         NOT NULL,
  node_key         VARCHAR(64) NOT NULL,
  node_type        VARCHAR(16) NOT NULL,            -- produce | review | align | gate
  gate             TINYINT(1)  NOT NULL DEFAULT 0,
  user_gate        TINYINT(1)  NOT NULL DEFAULT 0,  -- User 终审拍板位
  scope_expr       VARCHAR(64) NOT NULL DEFAULT '', -- 渲染集合表达式：''|'rd|full'|'fe|full'|'full'|'rd|fe'
  reviewers        JSON        NOT NULL DEFAULT ('[]'),  -- [{role, capability_id, scope?}]
  node_actions     JSON        NOT NULL DEFAULT ('[]'),  -- [{kind:capability|node, …}]
  expected_outputs JSON        NOT NULL DEFAULT ('[]'),
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  UNIQUE KEY uq_fdnode (flow_defs_id, node_order),
  INDEX idx_flow_def_nodes_tpl (flow_defs_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5. 流程实例
CREATE TABLE flows (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  requirement_id  VARCHAR(32) NOT NULL UNIQUE,     -- req-001
  flow_defs_id    VARCHAR(32) NOT NULL,
  dependencies    JSON        NOT NULL,            -- [requirement_id…] 同版本内前置依赖
  current_node_id VARCHAR(32) NULL,                -- 逻辑引用 flow_nodes.node_id
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 6. 实例节点
CREATE TABLE flow_nodes (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  requirement_id  VARCHAR(32) NOT NULL,
  node_id         VARCHAR(32) NOT NULL UNIQUE,     -- n-05
  def_node_id     VARCHAR(32) NOT NULL,            -- → flow_def_nodes.template_node_id
  node_order      INT         NOT NULL,
  node_key        VARCHAR(64) NOT NULL,
  state           VARCHAR(16) NOT NULL DEFAULT '', -- '' | entry | exec | review | respond（可配相位 4 值；''=未进入编排）
  review_results  JSON        NOT NULL DEFAULT ('[]'),  -- **单字段**：[{role, capability_id, state:'reviewing'|'agree'|'unagree', round, at}]（per-agent 表态，collect 反复更新/读取）
  reopen_source   VARCHAR(24) NOT NULL DEFAULT '', -- ''|upstream-reject|downstream-reject|rework-return
  rolled_back_count INT       NOT NULL DEFAULT 0,  -- 0=首次，≥1=重入
  reviewers       JSON        NOT NULL DEFAULT ('[]'),  -- 实例化时按 scope 过滤后的静态声明
  feature_branch  VARCHAR(128) NULL,               -- 评审敲定后写入
  entered_at      BIGINT NULL,
  pending_scope   TEXT NULL,
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  INDEX idx_flow_nodes_req (requirement_id),
  INDEX idx_flow_nodes_def (def_node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- 注：last_decision JSON 保留（承载门禁裁决 approved/rejected + reject 结论协议 branches/confirmed_issues/context）

-- 7. 任务（唯一待办总线）
CREATE TABLE tasks (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  task_id        VARCHAR(40) NOT NULL UNIQUE,      -- t-<ts>-<seq>
  requirement_id VARCHAR(32) NOT NULL,
  node_id        VARCHAR(32) NOT NULL,
  action_id      VARCHAR(64) NOT NULL,             -- 派发定位到能力×动作
  branch         VARCHAR(8)  NOT NULL DEFAULT '',
  role_id        VARCHAR(32) NOT NULL,
  task_type      VARCHAR(16) NOT NULL,             -- produce|dev|review|respond|accept|create_reqs
  goal           TEXT        NOT NULL,
  context        JSON        NOT NULL,
  status         VARCHAR(16) NOT NULL DEFAULT 'todo',
                 -- todo | in-progress | awaiting-reply | completed | rejected | closed
                 -- completed 仅 plugin 判据校验通过后置（agent 不可直置）
  root_issue_id  VARCHAR(32) NULL,
  reply_count    INT    NOT NULL DEFAULT 0,
  last_reply_at  BIGINT NULL,
  dispatch_at    BIGINT NULL,
  last_ack       BIGINT NULL,
  last_progress  TEXT NULL,
  resume_hint    TEXT NULL,
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  done_at BIGINT NULL, closed_at BIGINT NULL,
  INDEX idx_tasks_req (requirement_id), INDEX idx_tasks_node (node_id),
  INDEX idx_tasks_role (role_id), INDEX idx_tasks_status (status),
  INDEX idx_tasks_action (action_id), INDEX idx_tasks_dispatch (dispatch_at),
  INDEX idx_tasks_last_reply (last_reply_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 8. 产物
CREATE TABLE artifacts (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  artifact_id    VARCHAR(32) NOT NULL UNIQUE,      -- art-<ts>
  requirement_id VARCHAR(32) NOT NULL,
  node_id        VARCHAR(32) NOT NULL,
  product_type   VARCHAR(32) NOT NULL,             -- 受控枚举（brd/product_doc/…/discussion；无 case_design）
  label          VARCHAR(128) NOT NULL,
  href           VARCHAR(512) NOT NULL,
  kind           VARCHAR(16)  NOT NULL DEFAULT 'document',  -- document | link
  note           VARCHAR(512) NOT NULL DEFAULT '',
  added_by       VARCHAR(32)  NOT NULL,
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  INDEX idx_artifacts_req (requirement_id, node_id),
  INDEX idx_artifacts_ptype (product_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 9. 评论/问题（issue 状态机 v2）
CREATE TABLE issues (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  issue_id       VARCHAR(32) NOT NULL UNIQUE,      -- cm-<ts>
  requirement_id VARCHAR(32) NOT NULL,
  node_id        VARCHAR(32) NOT NULL,
  anchor_kind    VARCHAR(16) NOT NULL DEFAULT 'document',  -- document|line|range|block
  anchor_artifact_id VARCHAR(32) NULL,
  anchor_line INT NULL, anchor_start_line INT NULL, anchor_end_line INT NULL,
  anchor_block_start INT NULL, anchor_block_end INT NULL, anchor_lang VARCHAR(24) NULL,
  author      VARCHAR(32) NOT NULL,
  role_id     VARCHAR(32) NOT NULL,
  reviewer_role VARCHAR(32) NULL,
  issue_type  VARCHAR(16) NOT NULL DEFAULT 'issue',  -- issue|notification|discussion
  assignee    VARCHAR(32) NULL,
  text        TEXT NOT NULL,
  state       VARCHAR(20) NOT NULL DEFAULT 'open',
              -- open | pending-confirm | **to-fix（待修复）** | **fixed（已修复待复核）** | resolved | overridden
  escalate    TINYINT(1)  NOT NULL DEFAULT 0,     -- 冲突响应标记
  loop_count  INT         NOT NULL DEFAULT 0,      -- 回退计数（澄清期读=reopen 次数，修复期读=revise 次数）
  replies     JSON        NOT NULL DEFAULT '[]',   -- [{reply_id, by, role_id, text, at}] 不可变追加（裁决结论回填此处）
  round_count INT NOT NULL DEFAULT 0,
  last_reply_at BIGINT NULL,
  resolve_by VARCHAR(32) NULL, resolve_at BIGINT NULL,
  resolve_note TEXT NULL, resolve_path VARCHAR(8) NULL,   -- fix | agree
  resolve_reviewer_confirmed TINYINT(1) NOT NULL DEFAULT 0,
  resolved_by VARCHAR(32) NULL, resolved_at BIGINT NULL,
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  INDEX idx_issues_node (requirement_id, node_id),
  INDEX idx_issues_assignee (assignee),
  INDEX idx_issues_state (state),
  INDEX idx_issues_escalate (escalate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 10. 会话（新表：flow↔session 绑定 + 按需会话通道）
CREATE TABLE sessions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_id     VARCHAR(64) NOT NULL UNIQUE,
  requirement_id VARCHAR(32) NOT NULL DEFAULT '',
  node_id        VARCHAR(32) NOT NULL DEFAULT '',
  role_id        VARCHAR(32) NOT NULL,
  dsh_ref        TEXT        NOT NULL,            -- DSH 会话句柄
  status         VARCHAR(12) NOT NULL DEFAULT 'active',  -- active|ended|disposed
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  INDEX idx_sessions_req (requirement_id), INDEX idx_sessions_node (node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 11. 审计
CREATE TABLE audit_log (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  table_name VARCHAR(32)  NOT NULL,
  row_id     VARCHAR(64)  NOT NULL,
  action     VARCHAR(8)   NOT NULL,              -- INSERT|UPDATE|DELETE
  field      VARCHAR(64)  NULL,
  old_value  TEXT NULL, new_value TEXT NULL,
  by_who     VARCHAR(32)  NOT NULL,             -- role_id | 'user' | 'system' | 'plugin'
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL,
  INDEX idx_audit_row (table_name, row_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 12. 元信息
CREATE TABLE meta (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  meta_key   VARCHAR(64) NOT NULL UNIQUE,         -- schema_version = v0.29-mysql-1
  meta_value TEXT        NOT NULL,
  created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 0.2 非库层（沿用设计）

- **project registry**：`~/.dsh/wbflow/projects.json`（plugin 全局配置，非 MySQL）——per-project = 每工作区一套库（本期单库单项目；registry 为 dsh 嵌入期预留）。
- **需求/版本本体**：MySQL 化后收编入库？——**否**：`requirements` / `versions` 本期入 MySQL（新建 `requirements`、`versions` 两表，替代 .workbench JSON：字段对齐基线 + 新增 dependencies 由 flows 承载、priority；历史 JSON 迁移脚本交付）。**修正：原 JSON 载体废弃，MySQL 全面承载。**
- **静态流程定义种子**：flow_defs/flow_def_nodes/roles/capabilities 由 seed SQL 初始化（四流程 41 定义节点 + 21 活跃能力，来源=五件套）。

---

## 1. 内部接口（HTTP API 契约）

### 1.1 接缝原则

- **wb_* 语义一份、HTTP 一张皮**：22 个 wb_* 工具语义 → Gin 路由；Web 前端、DSH 嵌入期（conversation.view 复用同一 API）、agent 侧 wb_* 工具（HTTP client 薄壳）统一消费。
- **统一响应**：成功 `{ok:true, …数据}`；失败 `{ok:false, error, code, blocked?, missing?}`；门禁拦截带 `blocked:true`（沿既有 lossless JSON 约定）。
- **认证**：本期 Web 本地部署，简单 token（登录后签发）；DSH 嵌入期替换为 DSH session 适配器（接口不变）。

### 1.2 路由表（Gin）

| 方法 | 路由 | wb_* 语义 | 说明 |
|---|---|---|---|
| GET | /api/requirements | wb_list_requirements | 支持 version/priority/status/scope 筛选 + 分页 |
| GET | /api/versions | wb_list_versions | 含待分配池 |
| GET | /api/requirements/:reqId | wb_get_requirement | 聚合快照（需求+流程树+产物+评论+review_results） |
| GET | /api/requirements/:reqId/artifacts/:artifactId/content | wb_get_artifact_content | 正文（阅读器渲染与锚点定位） |
| POST | /api/requirements/:reqId/my-task | wb_my_task | 入参 {role, nodeId?}；返回 currentPhase/activeActions/review_results 摘要/relatedNodes（three-way 校验） |
| POST | /api/requirements | wb_create_requirement | 含 category→流程类型绑定（需求任务→business…） |
| PUT | /api/requirements/:reqId | wb_update_requirement | 自动 diff 变更历史 |
| DELETE | /api/requirements/:reqId | wb_delete_requirement ⚠️ | 需 User 明确指令 |
| POST | /api/versions | wb_create_version | |
| PUT | /api/versions/:versionId | wb_update_version | |
| POST | /api/versions/:versionId/advance | wb_advance_version | 发布软约束：unfinished 仅提示 |
| POST | /api/versions/:versionId/deprecate | wb_deprecate_version ⚠️ | 有引用 blocked |
| POST | /api/requirements/:reqId/nodes/:nodeId/artifacts | wb_add_artifact | +product_type 枚举校验 |
| POST | /api/requirements/:reqId/comments | wb_comment | anchor 四级；+escalate? |
| POST | /api/comments/:commentId/replies | wb_reply | roundCount++ |
| POST | /api/comments/:commentId/resolve | wb_resolve | path=fix\|agree；+escalate? |
| POST | /api/comments/:commentId/confirm | wb_confirm_review | 提评者复核 |
| POST | /api/comments/:commentId/reopen | wb_reopen | loop_count++ |
| POST | /api/requirements/:reqId/nodes/:nodeId/review | wb_review | 写 review_results（agent 自判 agree/unagree） |
| POST | /api/tasks | wb_assign_task | 幂等键 reqId+nodeId+action_id+assignee |
| PATCH | /api/tasks/:taskId | wb_task_update | event: ack\|progress\|complete（complete 触发判据校验） |
| GET | /api/inbox | wb_list_inbox | 通知枚举 v3 七类 |
| GET | /api/requirements/:reqId/dependencies/tree | （依赖树） | 动态加载，分析师视图 |
| GET | /api/projects · POST /api/projects | （registry） | dsh 嵌入期启用 |

- **移出 agent/HTTP 清单**：wb_decide（门禁判定 = 内部 collect 判定 + User 拍板走专用路由 `POST /api/requirements/:reqId/nodes/:nodeId/decision`，UI 拍板按钮专用）、wb_override（escalate 取代）——语义保留在 service 层。
- **实时通道**：收件箱 badge/节点状态变更 = 前端轮询（MVP）；SSE/WebSocket 预留接口（/api/events，v1.1）。

### 1.3 内部分层（Go）

```
handler/   Gin 路由薄壳：参数绑定 + 日志（入口 log req / defer log resp）+ 转发 service
service/   业务编排：判据校验、collect 聚合、门禁公式、幂等、审计调用（每个 wb_* 语义一个 Service）
domain/    状态机与纯逻辑：review_results 聚合判定、issue 状态机 v2、门禁公式、结论协议装配——无 IO，可单测
repo/      MySQL 访问（GORM）： PreparedStmt + SkipDefaultTransaction（写路径显式事务）
```

- 事务边界：写路径（上报/表态/裁决）单事务 + 行锁（SELECT … FOR UPDATE 于 flow_nodes 聚合判定，防并发双 collect）；消息/通知派发移出事务由 dispatcher 缓冲。
- 错误：内部 errors.Trace 层层包裹 + 哨兵错误（ErrGateBlocked/ErrIdempotentConflict/ErrStaleReport…）；HTTP 出参错误码集中映射，接口永不 panic。

---

## 2. 异常处理

| 场景 | 处理 |
|---|---|
| agent 谎报完成 | complete 上报后 plugin 校验相位判据；不满足 `{ok:false, blocked:true, missing:[…]}`；任务回 in-progress |
| 重复上报/重复派发 | 幂等：complete 判据幂等；派发按 `reqId+nodeId+action_id+assignee` 幂等键；同轮同角色 review 覆写取最后 |
| agent 无响应 | 扫描器：绑定 subagent `agent/status`=running → 等待；已结束 5min 未表态 → followup 催办（遇卡点上报 user）；10min → 通知 user 介入（参数化配置） |
| 启动失败 | agent/request-error 监听 + 重试策略（沿用直启实测结论） |
| 门禁拦截 | 判据不满足的操作返回 `{blocked:true, reason}`，审计留痕 |
| 并发 collect 竞争 | flow_nodes 行锁 + 单事务（聚合判定读改写原子）；消息/通知派发移出事务 |
| 澄清不收敛 | loop_count 达阈值（如 reopen≥5）→ 自动标 escalate → User 仲裁（沿用 wb_reopen roundCount>5 通知语义） |
| MySQL 不可用 | 服务降级只读（GET 走缓存快照）+ 健康检查 /healthz；恢复后扫描器续跑 |
| 审计失败 | 审计写失败回滚业务事务（审计与业务同库同事务，保证留痕一致） |

---

## 3. 性能考量

| 维度 | 策略 |
|---|---|
| 索引 | v0.29 索引清单全量（见 §0.1 各表 INDEX）+ issues.state 组合索引；review_results 聚合判定走 flow_nodes 单行 JSON 读取（免跨表 join） |
| 热点查询 | wb_get_requirement 聚合快照 = 单需求 4 表 JOIN，结果可短 TTL 缓存（写路径失效）；大需求池列表强制分页（LIMIT/OFFSET + 游标预留） |
| JSON 列 | review_results/node_actions 单行读写为主；MySQL JSON 函数仅用于聚合展示，高频判定在应用层解析 |
| 连接池 | GORM 连接池（MaxOpenConns 按部署配置）+ PrepareStmt |
| 事务 | 短事务优先；写路径行锁范围最小化（flow_nodes 单行）；通知派发移出事务 |
| 审计 | 同库同事务写入（一致性优先）；量大后按 table_name+row_id 分区（预留） |
| 前端 | 流程树 SVG 一次渲染全节点（≤41 定义节点上限，无性能问题）；阅读器大文档分段渲染 |

---

## 4. 验收标准

### 4.1 版本级（roadmap P1–P6 全链路）

- [ ] 新会话以 team preset + 团队制度启动，wb_* 22 工具（HTTP 版）对 agent 可用（P1）
- [ ] MySQL v0.29 全量表 + seed（四流程 41 定义节点 + 21 能力）初始化通过（P2）
- [ ] 一条 full 需求端到端：对齐（to-brd）→ 产品设计 → 三轮评审（per-agent 表态/澄清/复核/agreed→approved）→ RD∥FE 并行开发（CR 内嵌）→ QA 验收（仅 full 渲染）→ 用户验收 → 完成，全流程无阻塞（P5）
- [ ] 一条 rd-only 需求：流程树动态重排为 9 节点（FE 侧与 QA 不渲染、不占位），走通至用户验收（P5）
- [ ] 一条 task 需求（7 节点无 QA）：走通至用户验收，报障直连开发执行者（P5）
- [ ] 需求对齐：entry hold 会话 → 用户确认 → to-brd 落盘 → 迁移（P6）
- [ ] 打回链路：评审 unagree→to-fix → rejected → exec 重入（revise）→ 重新提测 → QA retest 复验 → resolved（P5）
- [ ] 冲突上升：escalate → 裁决 session → 结论回填 issue → reviewer 重表态 → 恢复（P5）
- [ ] 收件箱七类通知触发与跳转正确；TODO 投影一致（P5）
- [ ] 扫描器：5min 催办 / 10min 介入按时触发（注入时钟模拟）（P2）
- [ ] 幂等：重复 complete/重复派发/同轮覆写无害（P2）
- [ ] 审计：全部写路径落 audit_log（P2）

### 4.2 模块级 AC

- 模块 I/II/III 基线 PRD 的全部 AC 沿用（既有实现已验证的交互回归）+ 本文增补点新增 AC：
- [ ] 流程树 scope 重排：切换/实例化三种 scope，节点集与序号符合渲染矩阵，被裁节点不占位
- [ ] 拍板卡三态切换：进度只读 → agreed 迎宾呈现（汇总+按钮组+跳转）→ approved/rejected
- [ ] 冲突卡：escalate 聚合、正反对照、裁决回填 issue、跳转节点详情
- [ ] per-agent 表态：review_results 单字段读写一致；reviewing→agree/unagree 迁移判据正确

---

## Testing Decisions

- **接缝测试**：httptest 直打 Gin 路由（最高接缝），覆盖全部 22 端点正常/拦截/幂等分支。
- **领域层单测**：domain/ 纯逻辑（collect 聚合判定、issue 状态机 v2、门禁公式、结论协议装配）表驱动全分支——无 IO 全覆盖。
- **repo 层**：sqlmock / testcontainers-mysql（DDL 真实执行 + 关键查询计划）。
- **既有约定沿用**：`TestUnit_*` 前缀、gomonkey 打桩下游、gotest.tools+testify 混用。
- **端到端**：roadmap P5 演练脚本化（seed 注入 → HTTP 驱动全流程）。

## Out of Scope（本期不做）

- DSH 嵌入适配器（conversation.view 槽 + DSH session 认证）——接口已预留，下期
- P2/P3 评审轮完整视图 UI、冲突对峙增强、依赖调度器、override/decide agent 侧工具（service 层保留语义）
- SSE/WebSocket 实时推送（MVP 前端轮询）
- diff 级锚点、多项目切换 UI 完整版、版本间批量迁移、候补自动推荐（roadmap v1.1 其余项）

## Further Notes

- **MySQL 8.0 假设**：JSON 类型/CTE 依赖 8.0；部署版本若低于 8.0 需提前确认。
- **幂等键**：complete 幂等（判据幂等）；派发 `reqId+nodeId+action_id+assignee`；回退 `reqId+nodeId+round`。
- **设计权威冲突时**：以设计五件套（capability-model-final v3.1 / data-tools-final / ui-interaction-final 等）为准，本 spec 为实现翻译层；翻译发现设计缺口 → 回设计层裁决，不自行发挥。
- **DSH 嵌入预留**：API-first 已保证；嵌入期仅新增认证适配器 + conversation.view 挂载，无 API 变更。
