# wbflow v1.0 重新实现 · 技术规格（Tech Spec）

> **状态**：2026-09-08 定稿稿（待 User 确认接缝与两项默认假设：MySQL 8.0 / 幂等键）。设计权威=五件套（见 Further Notes）；本 spec = 实现层规格，冲突时以设计五件套为准。
> **技术栈（User 指定）**：数据库 **MySQL** · 后端 **Go + Gin** · 前端 **Web**（本期实现在 web；保留嵌入 DSH 能力——API-first 双嵌入）。
> **文档约束（2026-09-08 User 交接「tech-spec 文档要求汇总」，全量采纳为基线）**：数据模型九条（完整 DDL/双时间戳/自增整型主键/无物理外键/updated_at 乐观锁/两类版本区分/**JSON 载荷存 TEXT 不用 JSON 类型**/默认 NOT NULL/语义命名）+ 接口契约五条（GoDoc/proto 真相源/XxxReq-XxxResp/string 承载结构化载荷）+ gofmt 呈现。偏离需先声明与既有裁决的冲突。
> **最高接缝**：**HTTP API**——wb_* 语义一份、HTTP 一张皮。Web 前端、未来 DSH 嵌入（conversation.view 槽）、agent 侧（DSH subagent 的 wb_* 工具=HTTP client）统一消费同一 API 层。测试直接打 HTTP。

---

## 0. 数据模型（MySQL 完整 DDL）

> **约定（普适约束，全部表适用）**：主键 `BIGINT UNSIGNED AUTO_INCREMENT` 纯自增整型（不引入带前缀字符串主键）；**不建物理外键**（`xx_id` 列 + 索引，完整性由 service 层保证）；**乐观锁 = `updated_at`**（UPDATE 带读取时值，0 行命中 → 版本冲突；不设独立 version 列）；**JSON 载荷统一 TEXT 列**（不用 MySQL JSON 类型，序列化由 service 层处理，列名用业务语义不带 json 后缀）；默认 NOT NULL、字符串缺省空串，仅数值/时间/整型引用列保留 NULL（逐列注释语义）；每表 `created_at` + `updated_at`（BIGINT unix 毫秒，应用层维护，append-only 表两者恒同值）。
> **两类「版本」区分**：行级乐观锁（`updated_at`）≠ 业务正文版本号（多版快照机制的组成，如正文版本号/当前版本游标/批注所挂版本）——本期未引入多版快照，未来引入时不得并入乐观锁语义。

```sql
-- 1. 能力注册表
CREATE TABLE capabilities (
  id           BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  capability_id  VARCHAR(64)  NOT NULL COMMENT '能力唯一标识，如 backend-dev / arch.prd-review',
  name           VARCHAR(128) NOT NULL COMMENT '能力显示名',
  role_id        VARCHAR(32)  NOT NULL COMMENT '归属角色，逻辑引用 roles.role_id',
  inputs         TEXT         NOT NULL COMMENT '输入可见性契约 JSON 文本：[{"productType":"…","useThreads":true}]，service 层序列化',
  actions        TEXT         NOT NULL COMMENT '动作清单 JSON 文本：[{"actionId":"…","name":"…","on":"exec","skill":"…","inline":"…","outputProductTypes":[…]}]',
  products       TEXT         NOT NULL COMMENT '产物清单 JSON 文本：[{"docPath":"…","templatePath":"…","demoPath":"…"}]',
  created_at     BIGINT       NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at     BIGINT       NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_capabilities_cid (capability_id),
  KEY idx_capabilities_role (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='能力注册表';

-- 2. 角色注册表
CREATE TABLE roles (
  id           BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  role_id      VARCHAR(32)  NOT NULL COMMENT '角色标识：rd/fe/qa/arch/pm/cr/inv/leader/user/executor',
  name         VARCHAR(64)  NOT NULL COMMENT '角色显示名',
  note         VARCHAR(512) NOT NULL DEFAULT '' COMMENT '角色工作内容注释（非 persona）',
  created_at   BIGINT       NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at   BIGINT       NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_roles_rid (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色注册表';

-- 3. 流程定义（原 flow_templates）
CREATE TABLE flow_defs (
  id            BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  flow_defs_id  VARCHAR(32) NOT NULL COMMENT '流程类型：business/tech/task/plan',
  scope_options TEXT        NOT NULL COMMENT '可选 scope 集合 JSON 文本：business/tech=["full","rd","fe"]，task=["rd","fe"]，plan=[]',
  created_at    BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at    BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_flow_defs_fid (flow_defs_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流程定义表';

-- 4. 定义节点（原 flow_template_nodes）
CREATE TABLE flow_def_nodes (
  id               BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  flow_defs_id     VARCHAR(32) NOT NULL COMMENT '所属流程，逻辑引用 flow_defs.flow_defs_id',
  template_node_id VARCHAR(32) NOT NULL COMMENT '定义节点稳定 id，如 bus-03',
  node_order       INT         NOT NULL COMMENT '编排序号，1 起',
  node_key         VARCHAR(64) NOT NULL COMMENT '节点名',
  node_type        VARCHAR(16) NOT NULL COMMENT '节点类型：produce/review/align/gate',
  gate             TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '是否门禁：1=有准出判定',
  user_gate        TINYINT(1)  NOT NULL DEFAULT 0 COMMENT 'User 终审拍板位：1=agreed 后唤醒 User',
  scope_expr       VARCHAR(64) NOT NULL DEFAULT '' COMMENT '渲染集合表达式：""=any / rd|full / fe|full / full / rd|fe',
  reviewers        TEXT        NOT NULL COMMENT '评审位声明 JSON 文本：[{"role":"…","capabilityId":"…","scopeExpr":"…"}]',
  node_actions     TEXT        NOT NULL COMMENT '动作激活清单 JSON 文本：[{"kind":"capability","capabilityId":"…","actions":[…]}] 与 node action 条目',
  expected_outputs TEXT        NOT NULL COMMENT '期望产物 JSON 文本',
  created_at       BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at       BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_fdnode_tnid (template_node_id),
  UNIQUE KEY uk_fdnode_order (flow_defs_id, node_order),
  KEY idx_fdnode_tpl (flow_defs_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定义节点表';

-- 5. 流程实例
CREATE TABLE flows (
  id              BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  requirement_id  VARCHAR(32) NOT NULL COMMENT '需求 id，逻辑引用 requirements.requirement_id',
  flow_defs_id    VARCHAR(32) NOT NULL COMMENT '流程类型，逻辑引用 flow_defs.flow_defs_id',
  dependencies    TEXT        NOT NULL COMMENT '前置依赖 JSON 文本：["req-…"]（同版本内依赖边）',
  current_node_id VARCHAR(32) NULL COMMENT '当前节点 id；NULL=未进入编排（语义见列注释约定）',
  created_at      BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at      BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_flows_req (requirement_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流程实例表';

-- 6. 实例节点
CREATE TABLE flow_nodes (
  id                 BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  requirement_id     VARCHAR(32) NOT NULL COMMENT '需求 id，逻辑引用 flows.requirement_id',
  node_id            VARCHAR(32) NOT NULL COMMENT '实例节点 id，如 n-05',
  def_node_id        VARCHAR(32) NOT NULL COMMENT '定义节点引用，逻辑引用 flow_def_nodes.template_node_id',
  node_order         INT         NOT NULL COMMENT '编排序号（渲染序号，scope 过滤后重排）',
  node_key           VARCHAR(64) NOT NULL COMMENT '节点名',
  state              VARCHAR(16) NOT NULL DEFAULT '' COMMENT '可配相位：""=未进入编排 / entry / exec / review / respond',
  review_results     TEXT        NOT NULL DEFAULT ('[]') COMMENT '**评审轮结果（单字段承载全部 agent）** JSON 文本：[{"role":"…","capabilityId":"…","state":"reviewing|agree|unagree","round":N,"at":毫秒}]——collect 反复更新与读取',
  reopen_source      VARCHAR(24) NOT NULL DEFAULT '' COMMENT '重开来源：""/upstream-reject/downstream-reject/rework-return',
  rolled_back_count  INT         NOT NULL DEFAULT 0 COMMENT '回退计数：0=首次，≥1=重入（entry/exec 多配置选择）',
  reviewers          TEXT        NOT NULL COMMENT '实例评审位声明 JSON 文本（按 scope_expr 过滤后的静态声明）',
  feature_branch     VARCHAR(128) NULL COMMENT '评审敲定后的分支名；NULL=尚未敲定',
  entered_at         BIGINT      NULL COMMENT '进入执行时间；NULL=未进入',
  pending_scope      TEXT        NULL COMMENT '打回待重定范围；NULL=无',
  created_at         BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at         BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_flow_nodes_nid (node_id),
  KEY idx_flow_nodes_req (requirement_id),
  KEY idx_flow_nodes_def (def_node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='实例节点表';
-- 注：门禁裁决 approved/rejected 由 last_decision JSON 承载（见 flow_nodes 扩展或独立列，见 §0.3 补充列）

-- 7. 任务（唯一待办总线）
CREATE TABLE tasks (
  id             BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  task_id        VARCHAR(40) NOT NULL COMMENT '任务业务 id，如 t-<ts>-<seq>',
  requirement_id VARCHAR(32) NOT NULL COMMENT '需求 id',
  node_id        VARCHAR(32) NOT NULL COMMENT '节点 id',
  action_id      VARCHAR(64) NOT NULL COMMENT '能力×动作定位，如 backend-dev.dev',
  branch         VARCHAR(8)  NOT NULL DEFAULT '' COMMENT '泳道：rd/fe/空',
  role_id        VARCHAR(32) NOT NULL COMMENT '执行角色',
  task_type      VARCHAR(16) NOT NULL COMMENT '任务族：produce/dev/review/respond/accept/create_reqs',
  goal           TEXT        NOT NULL COMMENT '任务目标',
  context        TEXT        NOT NULL COMMENT '确定性上下文 JSON 文本',
  status         VARCHAR(16) NOT NULL DEFAULT 'todo' COMMENT 'todo/in-progress/awaiting-reply/completed（仅 plugin 判据后置）/rejected/closed',
  root_issue_id  VARCHAR(32) NULL COMMENT '锚定根 issue；NULL=非 issue 驱动',
  reply_count    INT    NOT NULL DEFAULT 0 COMMENT '回复数',
  last_reply_at  BIGINT NULL COMMENT '最近回复时间；NULL=无回复',
  dispatch_at    BIGINT NULL COMMENT '派发时间；NULL=未派发',
  last_ack       BIGINT NULL COMMENT '最近确认时刻；NULL=未送达（断点恢复关键）',
  last_progress  TEXT   NULL COMMENT '执行进度；NULL=未开始',
  resume_hint    TEXT   NULL COMMENT '断点续接提示；NULL=无',
  created_at     BIGINT NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at     BIGINT NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  done_at        BIGINT NULL COMMENT '完成时间；NULL=未完成（completed 判据后置）',
  closed_at      BIGINT NULL COMMENT '关闭时间（终态）；NULL=未关闭',
  PRIMARY KEY (id),
  UNIQUE KEY uk_tasks_tid (task_id),
  KEY idx_tasks_req (requirement_id),
  KEY idx_tasks_node (node_id),
  KEY idx_tasks_role (role_id),
  KEY idx_tasks_status (status),
  KEY idx_tasks_action (action_id),
  KEY idx_tasks_dispatch (dispatch_at),
  KEY idx_tasks_last_reply (last_reply_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务待办总线';

-- 8. 产物
CREATE TABLE artifacts (
  id             BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  artifact_id    VARCHAR(32)  NOT NULL COMMENT '产物业务 id，如 art-<ts>',
  requirement_id VARCHAR(32)  NOT NULL COMMENT '需求 id',
  node_id        VARCHAR(32)  NOT NULL COMMENT '节点 id',
  product_type   VARCHAR(32)  NOT NULL COMMENT '产物受控类型：brd/product_doc/rd_design/fe_design/architecture/research/plan/alignment/dev_notes/code_mr/test_result/defect/discussion（无 case_design）',
  label          VARCHAR(128) NOT NULL COMMENT '产物显示名，如《RD 技术设计》',
  href           VARCHAR(512) NOT NULL COMMENT '相对项目路径（doc）或外链（link）',
  kind           VARCHAR(16)  NOT NULL DEFAULT 'document' COMMENT 'document | link',
  note           VARCHAR(512) NOT NULL DEFAULT '' COMMENT '备注，空串=无',
  added_by       VARCHAR(32)  NOT NULL COMMENT '登记人角色',
  created_at     BIGINT       NOT NULL COMMENT '登记时间，unix 毫秒',
  updated_at     BIGINT       NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_artifacts_aid (artifact_id),
  KEY idx_artifacts_req (requirement_id, node_id),
  KEY idx_artifacts_ptype (product_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='产物表';

-- 9. 评论/问题（issue 状态机 v2）
CREATE TABLE issues (
  id              BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  issue_id        VARCHAR(32) NOT NULL COMMENT '评论业务 id，如 cm-<ts>',
  requirement_id  VARCHAR(32) NOT NULL COMMENT '需求 id',
  node_id         VARCHAR(32) NOT NULL COMMENT '节点 id',
  anchor_kind     VARCHAR(16) NOT NULL DEFAULT 'document' COMMENT '锚点：document/line/range/block',
  anchor_artifact_id VARCHAR(32) NULL COMMENT '锚定产物 id；NULL=document 级',
  anchor_line     INT NULL COMMENT '单行号（kind=line）',
  anchor_start_line INT NULL COMMENT '范围起行（kind=range）',
  anchor_end_line INT NULL COMMENT '范围终行（kind=range）',
  anchor_block_start INT NULL COMMENT '代码块起行（kind=block）',
  anchor_block_end INT NULL COMMENT '代码块终行（kind=block）',
  anchor_lang     VARCHAR(24) NULL COMMENT '代码块语言（kind=block）',
  author          VARCHAR(32) NOT NULL COMMENT '发表者',
  role_id         VARCHAR(32) NOT NULL COMMENT '发表者角色',
  reviewer_role   VARCHAR(32) NULL COMMENT '评审归属角色；NULL=未推导/非评审',
  issue_type      VARCHAR(16) NOT NULL DEFAULT 'issue' COMMENT 'issue/notification/discussion',
  assignee        VARCHAR(32) NULL COMMENT '解决责任人；NULL=未指派',
  text            TEXT        NOT NULL COMMENT '根评论内容',
  state           VARCHAR(20) NOT NULL DEFAULT 'open' COMMENT 'issue 状态机 v2：open 待澄清 / pending-confirm 已澄清待复核 / to-fix 待修复 / fixed 已修复待复核 / resolved / overridden',
  escalate        TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '冲突上升标记：1=拉起裁决 session',
  loop_count      INT         NOT NULL DEFAULT 0 COMMENT '回退计数：澄清期读=reopen 次数，修复期读=revise 次数',
  replies         TEXT        NOT NULL COMMENT '回复与裁决回填 JSON 文本（不可变追加）',
  round_count     INT         NOT NULL DEFAULT 0 COMMENT '对话轮数',
  last_reply_at   BIGINT NULL COMMENT '最近回复时间；NULL=无回复',
  resolve_by      VARCHAR(32) NULL COMMENT 'resolve 责任人；NULL=未解决',
  resolve_at      BIGINT NULL COMMENT 'resolve 时间；NULL=未解决',
  resolve_note    TEXT NULL COMMENT 'resolve 说明；NULL=未解决',
  resolve_path    VARCHAR(8) NULL COMMENT 'fix=修复关闭 / agree=协商一致关闭；NULL=未解决',
  resolve_reviewer_confirmed TINYINT(1) NOT NULL DEFAULT 0 COMMENT '评审者复核确认：0/1',
  resolved_by     VARCHAR(32) NULL COMMENT '最终确认人；NULL=未闭环',
  resolved_at     BIGINT NULL COMMENT '最终确认时间；NULL=未闭环',
  created_at      BIGINT      NOT NULL COMMENT '根评论创建时间，unix 毫秒',
  updated_at      BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_issues_iid (issue_id),
  KEY idx_issues_node (requirement_id, node_id),
  KEY idx_issues_assignee (assignee),
  KEY idx_issues_state (state),
  KEY idx_issues_escalate (escalate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论/问题表（issue 状态机：open→pending-confirm→resolved/to-fix/open（澄清，reopen 计数）；to-fix→fixed→复核→resolved/to-fix（修复，revise 计数））';

-- 10. 会话（flow↔session 绑定 + 按需会话通道）
CREATE TABLE sessions (
  id             BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  session_id     VARCHAR(64) NOT NULL COMMENT '会话业务 id',
  requirement_id VARCHAR(32) NOT NULL DEFAULT '' COMMENT '关联需求；空串=无',
  node_id        VARCHAR(32) NOT NULL DEFAULT '' COMMENT '关联节点；空串=无',
  role_id        VARCHAR(32) NOT NULL COMMENT '绑定角色（persona 按节点配置传入）',
  dsh_ref        TEXT        NOT NULL COMMENT 'DSH 会话句柄',
  status         VARCHAR(12) NOT NULL DEFAULT 'active' COMMENT 'active/ended/disposed',
  created_at     BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at     BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_sessions_sid (session_id),
  KEY idx_sessions_req (requirement_id),
  KEY idx_sessions_node (node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话绑定表';

-- 11. 审计（append-only：created_at=updated_at 恒同值）
CREATE TABLE audit_log (
  id         BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  table_name VARCHAR(32) NOT NULL COMMENT '被变更表名',
  row_id     VARCHAR(64) NOT NULL COMMENT '业务主键值',
  action     VARCHAR(8)  NOT NULL COMMENT 'INSERT/UPDATE/DELETE',
  field      VARCHAR(64) NULL COMMENT '变更字段；INSERT/DELETE 为 NULL',
  old_value  TEXT NULL COMMENT '旧值；INSERT/不适字段为 NULL',
  new_value  TEXT NULL COMMENT '新值；DELETE/不适字段为 NULL',
  by_who     VARCHAR(32) NOT NULL COMMENT '操作者：role_id/user/system/plugin',
  created_at BIGINT      NOT NULL COMMENT '审计时间，unix 毫秒',
  updated_at BIGINT      NOT NULL COMMENT '恒=created_at（append-only 统一占位）',
  PRIMARY KEY (id),
  KEY idx_audit_row (table_name, row_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志（append-only）';

-- 12. 元信息
CREATE TABLE meta (
  id         BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  meta_key   VARCHAR(64) NOT NULL COMMENT '元信息键，如 schema_version',
  meta_value TEXT        NOT NULL COMMENT '元信息值',
  created_at BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_meta_key (meta_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='元信息表';
```

### 0.2 语义约定（DDL 配套）

- **updated_at 乐观锁**：UPDATE 语句带读取时 updated_at 值（`WHERE … AND updated_at = ?`），0 行命中 → 版本冲突（ErrOptimisticLock），由上层重试或报错。
- **两类「版本」**：行级乐观锁（updated_at）≠ 业务正文版本号（多版快照机制的组成）——本期未引入多版快照；未来引入时以独立字段承载，禁止并入乐观锁语义。
- **JSON 载荷口径**：TEXT 列存 JSON 文本，service 层 `encoding/json` 序列化/反序列化；结构定义以 Go struct 为单一口径（与 proto 结构化字段 string 承载同构）。
- **NULL 保留清单**（语义已在列注释）：`flows.current_node_id`（未进入编排）、`flow_nodes.feature_branch/entered_at/pending_scope`、`tasks.root_issue_id/last_reply_at/dispatch_at/last_ack/last_progress/resume_hint/done_at/closed_at`、`issues` 锚点与 resolve 族、`sessions` 无。其余一律 NOT NULL + 空串/默认值。
- **review_results 口径**（collect 内置聚合的数据源）：agent 上报 agree/unagree 时 plugin 写入对应条目；collect 读取做推进判定（全部 state≠reviewing 且全 agree → 门禁 approved；有 unagree → owner 澄清循环）。

### 0.3 需求/版本两表（MySQL 化新增，取代 .workbench JSON 载体）

```sql
-- 13. 需求（原 requirements.json 收编）
CREATE TABLE requirements (
  id              BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  requirement_id  VARCHAR(32)  NOT NULL COMMENT '需求业务 id，如 req-001',
  title           VARCHAR(256) NOT NULL COMMENT '标题',
  description     MEDIUMTEXT   NOT NULL COMMENT 'Markdown 描述',
  category        VARCHAR(32)  NOT NULL COMMENT '类别标签：需求任务/规划审批任务/版本规划任务（决定流程类型绑定）',
  flow_defs_id    VARCHAR(32)  NOT NULL COMMENT '流程类型，逻辑引用 flow_defs（由 category 绑定派生）',
  version_id      VARCHAR(32)  NULL COMMENT '归属版本；NULL=待分配池',
  priority        VARCHAR(8)   NOT NULL DEFAULT 'P2' COMMENT 'P0-P3',
  status          VARCHAR(16)  NOT NULL DEFAULT '待启动' COMMENT '待启动/进行中/已完成/已取消',
  owner           VARCHAR(32)  NOT NULL DEFAULT '' COMMENT '负责人角色',
  created_by      VARCHAR(32)  NOT NULL COMMENT '创建人',
  created_at      BIGINT       NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at      BIGINT       NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_requirements_rid (requirement_id),
  KEY idx_requirements_ver (version_id),
  KEY idx_requirements_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='需求表';

-- 14. 版本（原 versions.json 收编）
CREATE TABLE versions (
  id          BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  version_id  VARCHAR(32) NOT NULL COMMENT '版本业务 id，如 v-1',
  name        VARCHAR(64) NOT NULL COMMENT '版本名（自由文本）',
  state       VARCHAR(16) NOT NULL DEFAULT '待启动' COMMENT '待启动/进行中/已发布/已废弃',
  is_backlog  TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '待分配池标记',
  created_at  BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at  BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_versions_vid (version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='版本表';
```

> 表数合计 **14 表**（12 原计划 + requirements/versions MySQL 收编）；sessions 编号顺延为 #10，requirements=#13、versions=#14。`.workbench/` JSON 文件载体废弃，磁盘仅保留产物文档本身（docs/ 仓库）。

---

## 1. 内部接口（HTTP API 契约）

### 1.1 接缝原则与契约口径

- **wb_* 语义一份、HTTP 一张皮**：Web 前端、未来 DSH 嵌入（conversation.view 槽）、agent 侧（DSH subagent 的 wb_* 工具=HTTP client 薄壳）统一消费同一 API 层。
- **契约真相源 = protobuf**：`docs/api/v1/api.proto` 一份协议文件定义全部请求/响应消息；HTTP JSON 序列化走 **protojson**；**枚举态字段用 string 而非 enum**（如 state="open"）。Go 侧 DTO（XxxReq/XxxResp）由 protoc 生成，**文档镜像禁止手写漂移**。
- **命名口径**：请求消息 `XxxReq`、响应 `XxxResp`（对应 HTTP 契约字段 xxx_req / xxx_resp）；与 proto 消息一一对应。
- **单一口径**：短字段直接 typed（string/int64/bool）；结构化载荷一律 **string 承载 JSON 文本**（与 TEXT 列同构，不重复造 schema——单一真相源）。
- **统一响应**：成功 `{ok:true, …数据}`；失败 `{ok:false, error, code, blocked?, missing?}`；门禁拦截带 `blocked:true`（沿既有 lossless JSON 约定）。
- **认证**：本期 Web 本地部署简单 token；DSH 嵌入期替换为 DSH session 适配器（API 路由不变）。

### 1.2 路由表（Gin）

| 方法 | 路由 | wb_* 语义 | Req/Resp（proto 消息） | 说明 |
|---|---|---|---|---|
| GET | /api/requirements | wb_list_requirements | ListRequirementsReq/Resp | version/priority/status/scope 筛选 + 分页 |
| GET | /api/versions | wb_list_versions | ListVersionsReq/Resp | 含待分配池 |
| GET | /api/requirements/:reqId | wb_get_requirement | GetRequirementReq/Resp | 聚合快照（需求+流程树+产物+评论+review_results） |
| GET | /api/requirements/:reqId/artifacts/:artifactId/content | wb_get_artifact_content | GetArtifactContentReq/Resp | 正文（阅读器渲染与锚点定位） |
| POST | /api/requirements/:reqId/my-task | wb_my_task | MyTaskReq/Resp | {role, nodeId?}；返回 currentPhase/activeActions/review_results 摘要/relatedNodes（three-way 校验） |
| POST | /api/requirements | wb_create_requirement | CreateRequirementReq/Resp | category→流程类型绑定 |
| PUT | /api/requirements/:reqId | wb_update_requirement | UpdateRequirementReq/Resp | 自动 diff 变更历史 |
| DELETE | /api/requirements/:reqId | wb_delete_requirement ⚠️ | DeleteRequirementReq/Resp | 需 User 明确指令 |
| POST | /api/versions | wb_create_version | CreateVersionReq/Resp | |
| PUT | /api/versions/:versionId | wb_update_version | UpdateVersionReq/Resp | |
| POST | /api/versions/:versionId/advance | wb_advance_version | AdvanceVersionReq/Resp | 发布软约束：unfinished 仅提示 |
| POST | /api/versions/:versionId/deprecate | wb_deprecate_version ⚠️ | DeprecateVersionReq/Resp | 有引用 blocked |
| POST | /api/requirements/:reqId/nodes/:nodeId/artifacts | wb_add_artifact | AddArtifactReq/Resp | +product_type 枚举校验 |
| POST | /api/requirements/:reqId/comments | wb_comment | CommentReq/Resp | anchor 四级；+escalate? |
| POST | /api/comments/:commentId/replies | wb_reply | ReplyReq/Resp | roundCount++ |
| POST | /api/comments/:commentId/resolve | wb_resolve | ResolveReq/Resp | path=fix\|agree；+escalate? |
| POST | /api/comments/:commentId/confirm | wb_confirm_review | ConfirmReviewReq/Resp | 提评者复核 |
| POST | /api/comments/:commentId/reopen | wb_reopen | ReopenReq/Resp | loop_count++ |
| POST | /api/requirements/:reqId/nodes/:nodeId/review | wb_review | ReviewReq/Resp | 写 review_results（agent 自判 agree/unagree） |
| POST | /api/tasks | wb_assign_task | AssignTaskReq/Resp | 幂等键 reqId+nodeId+action_id+assignee |
| PATCH | /api/tasks/:taskId | wb_task_update | TaskUpdateReq/Resp | event: ack\|progress\|complete（complete 触发判据校验） |
| GET | /api/inbox | wb_list_inbox | ListInboxReq/Resp | 通知枚举 v3 七类 |
| GET | /api/requirements/:reqId/dependencies/tree | （依赖树） | DependencyTreeReq/Resp | 动态加载，分析师视图 |
| GET | /api/projects · POST /api/projects | （registry） | ProjectReq/Resp | dsh 嵌入期启用 |

- **移出 agent/HTTP 清单**：wb_decide（门禁判定 = 内部 collect 判定 + User 拍板专用路由 `POST /api/requirements/:reqId/nodes/:nodeId/decision`，DecisionReq/Resp）、wb_override（escalate 取代）——语义保留在 service 层。
- **实时通道**：收件箱 badge/节点状态变更 = 前端轮询（MVP）；SSE/WebSocket 预留（/api/events，v1.1）。

### 1.3 内部分层（Go）

```
handler/   Gin 路由薄壳：参数绑定 + 日志（入口 log req / defer log resp）+ 转发 service
service/   业务编排：判据校验、collect 聚合、门禁公式、幂等、审计调用（每个 wb_* 语义一个 Service）
domain/    状态机与纯逻辑：review_results 聚合判定、issue 状态机 v2、门禁公式、结论协议装配——无 IO，可单测
repo/      MySQL 访问（GORM）： PreparedStmt + SkipDefaultTransaction（写路径显式事务）
```

- 事务边界：写路径（上报/表态/裁决）单事务 + 行锁（SELECT … FOR UPDATE 于 flow_nodes 聚合判定，防并发双 collect）；消息/通知派发移出事务由 dispatcher 缓冲。
- 错误：内部 errors.Trace 层层包裹 + 哨兵错误（ErrGateBlocked/ErrIdempotentConflict/ErrStaleReport/ErrOptimisticLock…）；HTTP 出参错误码集中映射，接口永不 panic；哨兵错误与各 Service 方法 GoDoc「错误集」交叉引用。
- **Service 签名规范**：全显式类型 + GoDoc 注释覆盖「做什么 / 入参·出参语义 / 错误集（引用哨兵错误）」；DTO 由 protoc 生成的 Go 类型承载（proto 为真相源，禁止手写漂移）。
- **代码格式**：gofmt 风格——struct 逐字段多行、tag 对齐；禁止紧凑单行写法。

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
- [ ] MySQL v0.29 全量表 + seed（四流程 40 定义节点 + 21 能力）初始化通过（P2）
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
