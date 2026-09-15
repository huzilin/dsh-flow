# wbflow v1.0 重新实现 · 技术规格（Tech Spec）

> **状态**：2026-09-08 定稿稿；2026-09-09 User 拍板修订——①**目标调整：一期 = 自包含 Web 应用**（编排推进能力本期以 wb_advance 补齐，Web 可独立跑通全流程；DSH plugin 嵌入为未来阶段，届时编排触发权移交 plugin）；②A-D 裁决落款（deprecate 条件解绑 / wb_advance 补端点 / join 锁+兜底巡检 / scope 暂不支持主动切换，细则见 §1.2 注记）。两项默认假设（MySQL 8.0 / 幂等键）已验证。设计权威=五件套（见 Further Notes）；本 spec = 实现层规格，冲突时以设计五件套+本拍板为准。
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
  scope           VARCHAR(8)  NOT NULL DEFAULT '' COMMENT '生效 scope（迁移 v3 补列，T4）：business/tech 取 scope_options 首值等；空=未定（task 渲染期再选）',
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
  state           VARCHAR(20) NOT NULL DEFAULT 'open' COMMENT 'issue 状态机 v2：open 待澄清 / pending-confirm 已澄清待复核 / to-fix 待修复 / fixed 已修复待复核 / resolved（09-12 裁决：overridden 态废除，override 动作置 resolved，semantica 920030de）',
  escalate        TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '冲突上升标记：1=拉起裁决 session',
  loop_count      INT         NOT NULL DEFAULT 0 COMMENT '回退计数：澄清期读=reopen 次数，修复期读=revise 次数',
  replies         TEXT        NOT NULL COMMENT '回复与裁决回填 JSON 文本（不可变追加）',
  round_count     INT         NOT NULL DEFAULT 0 COMMENT '对话轮数',
  last_reply_at   BIGINT NULL COMMENT '最近回复时间；NULL=无回复',
  resolve_by      VARCHAR(32) NULL COMMENT 'resolve 责任人；NULL=未解决',
  resolve_at      BIGINT NULL COMMENT 'resolve 时间；NULL=未解决',
  resolve_note    TEXT NULL COMMENT 'resolve 说明；NULL=未解决',
  resolve_path    VARCHAR(8) NULL COMMENT 'fix=修复关闭 / agree=协商一致关闭 / override=强权仲裁关闭（09-12 裁决：override 类动作统一本值）；NULL=未解决',
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
  role_id        VARCHAR(32) NOT NULL COMMENT '绑定角色（角色说明按节点配置传入）',
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
| POST | /api/versions/:versionId/deprecate | wb_deprecate_version ⚠️ | DeprecateVersionReq/Resp | 条件解绑（2026-09-09 拍板，细则见 §1.2 注记-A）：其下全部需求未开始（流程未进入编排）→ 废弃成功并解绑全部需求回待分配池；任一需求已开始 → 409 blocked |
| POST | /api/requirements/:reqId/nodes/:nodeId/artifacts | wb_add_artifact | AddArtifactReq/Resp | +product_type 枚举校验 |
| POST | /api/requirements/:reqId/comments | wb_comment | CommentReq/Resp | anchor 四级；+escalate? |
| POST | /api/comments/:commentId/clarified | wb_clarified | ReplyReq/Resp | owner 澄清完成申报（open→pending-confirm），roundCount++ |
| POST | /api/comments/:commentId/declare-fixed | wb_declare_fixed | ReplyReq/Resp | owner 修复完成申报（to-fix→fixed） |
| POST | /api/comments/:commentId/settle | wb_settle | ResolveReq/Resp | 协商一致关闭（确认内嵌直落 resolved）；+escalate? |
| POST | /api/comments/:commentId/qualify | wb_qualify | ResolveReq/Resp | 复核定性真问题（→to-fix 触发打回；to-fix/fixed 重申报幂等） |
| POST | /api/comments/:commentId/confirm | wb_confirm | ConfirmReviewReq/Resp | 提评者复核确认（→resolved） |
| POST | /api/comments/:commentId/reopen | wb_reopen | ReopenReq/Resp | 复核退回·未解释清楚（pending-confirm→open），loop_count++ |
| POST | /api/comments/:commentId/revise | wb_revise | ReopenReq/Resp | 修复侧复核退回（fixed→to-fix），loop_count++ |

> **NAM-09 契约改名（2026-09-11，decision-timeline REV-08）**：comments 端点路由/工具名对齐标准领域事件词（clarified/settle/qualify/reopen/declare-fixed/confirm/revise，映射见 state-tables-draft「事件命名提案」）。纯词面修复、行为不变：clarified/declare-fixed 共用原 reply 核心、settle/qualify 共用原 resolve 核心（path 参数取消，路由名承载语义，服务端不校验 from-state）、reopen/revise 共用原 reopen 核心。D2「settle 自 to-fix/fixed」与 D3「qualify 自 open」偏差语义保持现状待裁。proto 消息结构与字段不动（不触发 breaking），旧路由不留兼容。
| POST | /api/comments/:commentId/escalate | wb_escalate | EscalateReq/Resp | 手动上升裁决（裁决 session 建绑） |
| POST | /api/comments/:commentId/adjudicate | wb_adjudicate | AdjudicateReq/Resp | User 仲裁回填（uphold/override） |
| POST | /api/requirements/:reqId/nodes/:nodeId/review | wb_review | ReviewReq/Resp | 写 review_results（agent 自判 agree/unagree） |
| POST | /api/requirements/:reqId/nodes/:nodeId/decision | wb_decision | DecisionReq/Resp | User 终审拍板（gate 相位） |
| POST | /api/requirements/:reqId/reproject | wb_reproject | ReprojectScopeReq/Resp | 打回后重定 scope 下游重投影；scope 主动切换无端点（2026-09-09 拍板：暂不支持，变更仅走打回→本链路） |
| POST | /api/tasks | wb_assign_task | AssignTaskReq/Resp | 幂等键 reqId+nodeId+action_id+assignee |
| PATCH | /api/tasks/:taskId | wb_task_update | TaskUpdateReq/Resp | event: ack\|progress\|complete（complete 触发判据校验） |
| POST | /api/requirements/:reqId/nodes/:nodeId/advance | wb_advance | AdvanceReq/Resp | 编排推进（2026-09-09 拍板新增，细则见 §1.2 注记-B/C） |
| GET | /api/inbox | wb_list_inbox | ListInboxReq/Resp | 通知枚举 v3 七类 |
| GET | /api/requirements/:reqId/dependencies/tree | （依赖树） | DependencyTreeReq/Resp | 动态加载，分析师视图 |
| GET | /api/requirements/:reqId/history | wb_list_requirement_history | ListRequirementHistoryReq/Resp | 需求字段变更历史（注记-D3，仅读） |

> **2026-09-09 拍板注记（A–G 细则，实现与测试以本注记为准）**：
>
> **-A 版本废弃 = 条件解绑**：`wb_deprecate_version` 语义由「有引用一律 blocked」改为条件解绑——判定单位为需求级，需求「已开始」=流程实例已进入编排（`flows.current_node_id` 非 NULL；T4 起需求创建即实例化 flows 行，故以编排进入为准而非「存在实例」——字面判定会使未开始解绑特性死置，实现口径见 semantica `23aeeaea`）；其下**任一**需求已开始 → 整个废弃请求 409 blocked（语义：已开始的推演不允许被版本废弃波及，未来再议）；其下**全部**需求未开始 → 废弃成功（versions.state=已废弃，终态），同时事务内将其下全部需求 version_id 置空（解绑，回到待分配池），逐行审计。与 ui-interaction-final §2「联动取消」口径的偏差：不取消需求、不写需求取消 history，改为解绑回池（本注记为裁决正本，UI 文档偏差由实现交付注记）。
>
> **-B 编排推进端点 wb_advance**：目标调整为一期自包含 Web 后，原「编排接缝三类无 HTTP 承载（plugin 职责）」不再成立。新增 `wb_advance`（AdvanceReq{reqId,nodeId}/Resp），承载三类推进：①exec 相位完成（complete 判据已通过）→ 下一节点相位初始化（review 相位则 InitReviewRound）；②节点内 CR 轮转（dev→review，轮 0 重开）；③QA/门禁机械收口（collect rule4 空转语义）。wb_advance 自身幂等：重复调用时若推进已完成则返回当前状态不重复推进。触发方：wb_task_update complete 判据通过后**服务端同步调用**推进逻辑（Web 一期由服务端内聚触发），端点同时暴露供兜底与手动场景；e2e 原接缝 SQL 模拟全部替换为 wb_advance 真调用（SQL 模拟代码删除）。DSH 嵌入期该端点收编为 plugin 内部调用，HTTP 契约不变。
>
> **-C 并行 join 判定**：并行组（flows.branch rd/fe 泳道）推进采用「提交瞬间检查 + 锁 + 兜底巡检」三层——①wb_advance 推进 exec 节点时检查同组对侧节点：对侧已完成（complete 且未推进）→ 本侧推进即整组推进到下一阶段；对侧未完成 → 本侧标记完成待对侧。②并发防护：推进路径对 flows 行取 `SELECT … FOR UPDATE` 锁（单事务内），防双侧同时 complete 的并发双推进（库为 RC + 单 master 无复制延迟，行锁已足）。③scanner（T10 基础设施）新增兜底巡检：每轮扫描「双侧均 complete 但未推进」的并行组并补推进（漏推自愈），巡检推进同样走锁路径并落审计。
>
> **-D 原型补齐实现增补（2026-09-09，依据 ui-interaction-final §6 N1–N7 定稿 + ui-prototype-v3 P8–P11；实现层翻译，设计权威仍在 §6 定稿）**：
>
> **-D1（N1 retest 通知）**：`re_review`（重新表态请求）触发由单一「澄清闭环 respond complete 重入轮」扩为双触发——②修复关闭申报迁入 `fixed` 后（NAM-09 改名前词面 resolve path=fix；现由 wb_qualify 自 to-fix/fixed 或 wb_declare_fixed 承载），事务外派发 re_review 通知收件=提评者（`issues.reviewer_role`，空则不发）。子型区分口径：澄清子型 `target_id` 恒空串（跳节点详情）；retest 子型 `target_id`=缺陷 issueId（收件人复核锚点，FE 呈现「待复验」并跳意见区）。`notificationTargetTypes` 映射不变（kind→node 定位语义不变，target_id 为锚点增强）。触发以「本次事务实际迁入 fixed」为准（prevState≠fixed）——覆盖 wb_qualify（to-fix/fixed 修复关闭申报）与 wb_declare_fixed（修复完成申报）两条提交路径；Fixed→Fixed 幂等重入（重复申报）不重发（CR 复审 B1 守卫）。
>
> **-D2（N5 会话 tab 数据源）**：`GetRequirementResp` 增 `repeated SessionItem sessions`（该需求 sessions 行投影：sessionId/requirementId/nodeId/roleId/status/updatedAt；按 requirement_id 全量，FE 按选中节点过滤）。sessions 写路径本期仅裁决链（wb_escalate 拉起）落行，空集合法（FE 呈现「暂无会话」占位）。与 N5 字面偏差：SessionItem 不含「最近消息摘要」字段（本期 dsh_ref 为占位句柄、无消息内容数据源），条目=绑定角色+状态+时间；摘要随 DSH 嵌入期会话端点补齐（偏差由实现交付注记）。
>
> **-D3（N6 变更历史端点）**：新增 `GET /api/requirements/:reqId/history`（wb_list_requirement_history，`ListRequirementHistoryReq/Resp`），读 `audit_log` 中 `table_name=requirements AND row_id=reqId` 行时间倒序（field/oldValue/newValue/byWho/createdAt 原样透出，自然语句渲染归 FE）。端点数 27→28。仅读，无新写路径。
>
> **-D4（P8 依赖角标数据源）**：`RequirementItem` 增 `repeated string dependencies`（flows.dependencies 透传）；列表与快照同源填充，FE 呈现「依赖 ×N」角标。创建/更新响应中本字段反映库值——创建期恒空集（flows.dependencies 写入路径属依赖调度器 E9 预留，CreateReq.dependencies 入参本期不落库），与快照口径一致。
>
> **-D5（N3 TODO 五分类投影口径）**：TODO = 收件箱通知的操作投影（基线规则+N3 定稿），非 tasks 投影——FE 从 notifications 投影五分类：待拍板←gate_pending；待澄清←clarify_task；待重表态←re_review（target_id 空）；待复验←re_review（target_id 非空，retest 子型）；冲突待裁决←escalation；stall_followup/need_intervene/session_message 为提醒类不产生 TODO。tasks 在途任务投影保留于节点详情④意见区（任务维度呈现），收件箱页 TODO 区按本口径改造。
>
> **-D6（P9 原型超集裁决）**：原型 P9 版本行「删除」按钮无对应路由（路由表无 wb_delete_version），不实现——原型为丢弃式视觉参照，功能边界以本路由表为准；「废弃」按钮已按注记-A 条件解绑实现。P9 依赖树入口（按需拉取）消费既有 GET /api/requirements/:reqId/dependencies/tree 逐需求装配，无新端点。
>
> **-E（node action 派发边界；CR 复审裁决 F，semantica `8e488988`）**：seed 词表中 `executor:"program"` 类 node action（如 `align.opening`，`on:"entry"`）不经 `wb_assign_task` 任务总线派发——node action 由服务端在节点相位内程序化执行（与 wbflow 程序化直启架构一致），任务总线仅承载 `actionTaskTypes` 有映射的 agent/人工动作；未知动作经 `TaskTypeForAction`「不可派发」拦截（domain/task.go），`align.opening` 无映射即天然不可派，实现行为与设计口径一致，本注记为文档化正本，不改路由。
>
> **-F（submit 动作不激活；CR 复审裁决 G，semantica `abf45f97`）**：seed 词表含 `submit`（登记请评，inline：「产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成」）但 `actionTaskTypes` 无映射、不激活——登记场景由 inline 动作经工具调用（`wb_add_artifact`+`wb_comment`）机械完成，不经任务总线；produce/complete 判据已覆盖其完成语义，补映射反而引入与判据重复的空派发，本注记为裁决正本。
>
> **-G（tasks.reply_count 死列；CR 复审裁决 I，semantica `ec4639b8`）**：迁移 v1 遗留列，现无写入路径（仅 model 定义与快照 DTO 读零值），回复计数语义由评论查询聚合承载——保留列 + TODO 注记插件期接线，不做迁移 DROP（v4 迁移成本大于收益），本注记为裁决正本。
>
> **-H（通知可靠投递 outbox 化；2026-09-10 User 方案，semantica `22864e64`）**：原「通知派发一律事务外、Dispatch 吞错记日志」口径（联调实证 req-002 gate_pending 静默丢失一次——无重试、无留痕）升级为消息事务表模式。新增 `notify_outbox`（迁移 v4，状态机 pending→sending→sent/failed；同日 User 追加拍板 semantica `c25c0552`：单人内部系统通知必须全部送达——failed 无限补投无放弃态，retry_count 仅观测）：①**业务事务内 enqueue**——各通知触发点（escalation/retest/re_review 重入轮/collect gate_pending·clarify/scanner 催办介入）在业务事务闭包内 `EnqueueNotificationTx` 写通知意图，与业务提交原子（「Dispatch 未被调用即丢」窗口消除）；enqueue 失败随业务回滚——同一 MySQL 实例下表故障即库故障，业务主写同样不可达，语义可接受；②**事务提交后 FlushPending** 即时推送（原子认领防双推→Notify 写 issues 通知行→sent）；③**scanner SweepOutbox** 周期补投（failed 全量 + pending/sending 停滞的崩溃残留），无限重试直至送达。「通知写入失败不阻断业务路径」口径保留于推送侧——flush/sweep 失败只影响通知时效，不回滚业务。收件箱读侧不变（仍读 issues 通知行投影），FE 零改动。
>
> **-J7（-J5 契约侧补记；2026-09-10 批注轮 fe/契约侧，与 -J5 同拍板出处）**：my-task `relatedArtifacts` 语义升级——review 型节点=直接上游连续 produce 段已登记产物（置前=被评审产物）+ 本节点产物；`RelatedArtifactItem` 无 upstream 标注字段，FE 按快照 nodeId 反查分段（未来多消费方需要显式标注再加字段）。
>
> **-J8（-J6 契约侧补记；同上出处）**：评论锚契约收紧——评审上下文（state∈{review,respond} ∪ 发表者∈评审位）issue 类评论必须携带 `anchor.artifactId`（document 级含之）；`ValidateIssueAnchor` document 分支允许 artifact_id 非空，DDL「artifact_id NULL=document 级」注记自本注记起废止（NULL 仍合法=历史数据，新写入恒带值）。
> **-I（并行 join wait 分支=设计级死代码；2026-09-10 BUG-007 裁决，文档化不动代码）**：advanceAfterGateLockedTx 的 `AdvanceOutcomeWait` 分支（本侧 fe 侧 exec-CR 节点完成而对侧 rd 侧未 approved → 本侧完成待对侧）经 API 不可达——fe 侧 exec-CR 节点唯一激活链是 rd 侧 gate 落定 approved 的瞬间（activateAndMoveLockedTx 唯一激活写点；打回联动与 scanner 巡检均不破坏该不变量），故 `!peerApproved(prev)` 恒 false。**裁决正本：wait 分支为防御性保留的死分支，仅 scanner 兜底巡检语义预留，不删不改**（同注记 -E/-F/-G「裁决正本」处理格式；诊断证据见 qa-diagnosis.md §5）。
>
> **-J（批注轮修复注记；2026-09-10 User 修复指令拍板 + 最小原则先例，逐条行为变更）**：
>
> **-J1（BUG-001 需求状态联动）**：`requirements.status` 与编排推进自动联动（方案 A，spec §7.3「收尾→已完成」本就要求）。触发点定义：①流程**首次进入编排**（flows.current_node_id 由空变非空）→ 待启动→进行中；②推进**落位尾节点**（实例链最后一个节点被激活/驻留）→ 已完成。实现落点=推进核 activateAndMoveLockedTx（wb_advance / wb_decision / complete 内聚 / scanner 巡检四条路径统一收口）；同事务 + 审计（by_who=推进方 plugin/system）；只在值变化时写（幂等）；已取消=终态不覆盖；requirements 乐观锁不介入（不带基线直更，避免与 wb_update_requirement 互相制造 409 冲突面）。
>
> **-J2（BUG-004 局部更新契约）**：`wb_update_requirement` 由「全量必填」改为**空串=不修改**语义——title/description/category/priority/status/owner/versionId 非空才校验+diff+写，全空=no-op；owner/version_id 以空串为合法值（清空/解绑）的旧口径废止（清空指派/解绑版本不经本端点）；proto 契约注释同步（proto 为真相源，api.pb.go 经 buf 重生成，注释级变更不影响序列化）。
>
> **-J3（BUG-006 my-task 动作全名）**：`wb_my_task` activeActions 的 actionId 口径统一——kind=capability 条目回显 `<capabilityId>.<action>` 全名（与 wb_assign_task 入参同口径，调用方可直接回填派发）；kind=node 条目保持原 id（align.opening 本就带点）。
>
> **-J4（BUG-008 通知 kind 投影）**：writeEscalationNoticeTx 通知行（issues notification）anchor_kind 由硬编码 document 改落七类词表 **escalation**，anchor_artifact_id=源 issueId（跳转锚）——手动 wb_escalate 与自动阈值两路径的通知行在收件箱「冲突待裁决←escalation」投影维度对齐 T11/comment 路径（通知行 anchor_kind 语义=通知类别，非评论锚点，issues 锚点校验域不覆盖通知行）。
>
> **-J5（BUG-011 被评审产物口径）**：评审节点的被评审产物 = **直接上游 produce 节点已登记的产物**（提评来源；多上游取全部直接上游 produce——实例链线性，即紧邻前驱起的连续 produce 段）。实现：my_task 装配 relatedArtifacts 时 node_type=review 的节点追加上游产物并置前（本节点产物殿后）；FE 产物区分段归 fe 批次。
>
> **-J6（BUG-013 评论锚产物强制）**：口径「所有的评审都是挂在文档上的」——评审上下文（节点 state ∈ {review, respond}，或 author 角色 ∈ 节点评审位）的 **issue 类**评论 anchor_kind=document 且未携 anchorArtifactId → 400 invalidArg（错误信息引导挂锚）；notification/discussion 类不受限（内部通知不走此通道）；存量数据不回溯（新评论硬拦）。既有 e2e 为旧口径产物由 QA 驱动统一更新，service 层 IT 冲突用例已同步锚产物。
>
> **-K（产物入库约束；2026-09-10 BUG-015 拍板）**：expected_outputs 期望的产物必须是系统内入库文档——kind=document，href 为相对 docs root 路径且文件真实存在。登记口（wb_add_artifact）：期望 document 的产物以 link 登记 → 400 blocked（`rd/service/artifact.go`，期望外 product_type 如 code_mr 补充外链不受限）；判据口（complete / advance exec 同口径，`rd/service/task.go` completeMissing）：仅 kind 合规的登记计入判据，kind 不符 → 显式缺项（token 前缀 `artifact_kind:<type>` + 人话说明，同 `exec_task:` 缺项族口径）。link 仅用于期望外的补充外链（如 code_mr）。用户验收环境数据巡检（D-3，`qa/qa_live_audit.py` 对联调库跑同一期望矩阵）违规即回归不通过。背景：联调库 9 产物曾为假域名外链占位，文档阅读→锚定评论→评审链在用户环境不可达（qa-defects.md BUG-015）。
>
> **-L（2026-09-10 深夜对齐轮四项拍板；User 逐条定案；spec 偏差清零批）**：
>
> **-L1（collect=评论闭环表态；OPT-004 拍板落地；09-12 裁决修订）**：collect 表态的本义对齐老设计（被反复确认十几遍的门禁链）——**agree 的前提=该评审位名下评论全部被响应并澄清**（state ∈ resolved；open/to-fix/fixed/pending-confirm 未闭环）；有未闭环评论须先解决（resolve 族）或保持 unagree（确认是问题 → 打回前一节点）。**approve 前置=节点名下全部 issue 类评论闭环**（要么打回要么全 resolved 才进下一节点）；rejected 不受限（打回即「确认是问题」的合法出口）。强权泄压阀：新增 `wb_override_role`（POST /api/comments/override-role，operator 须=user）——整角色未闭环评论批量置 resolved（resolve_path=override）。**09-12 裁决废除 overridden 态：override 动作=批量 resolve，resolve_path=override；存量若有 overridden 行一次性 UPDATE 为 resolved**（semantica 920030de）。（「强行通过 X 全部剩余」原型对应物。）空集=闭环（vacuous，0 评论可直接表态）。notification/discussion 行不参与闭环。轮次高亮沿用 -C7 reopen≥5 自动上升口径。
>
> **-L2（done 离场终态；BUG-016 拍板落地）**：flow_nodes.state 增加 **done** 业务终态（老模型「已通过」的 v1.0 对应物）——wb_advance 推进激活目标节点时，目标之前的全部活跃相位节点（entry/exec/review/respond，按 node_order 切片序）统一置 done（并行组对侧+存量脏状态一次自愈）；done 节点**禁止挂载**（wb_add_artifact/wb_assign_task → 400/409）；对 done 节点重复 wb_advance 幂等 noop 200；打回重入由 RollbackForRejectTx 复位；裁决轮重入对 done 节点 no-op（不阻断回填）。FE 点色 done=绿、徽标「done · 已完成」。
>
> **-L3（评论锚点简化为划词+文档级；OPT-005 拍板落地）**：锚点词表收紧为 **range（划词）+ document（文档级）** 两级——line/block 废弃（wb_comment 400 拒绝并引导本注记）；存量 line/block 数据只读兼容（展示不拒绝）。FE 阅读器交互对齐老阅读器手感：纯单击序列（首点=起点 / 重复点起点=取消 / 起点后点他行=框选区间 / 已有区间再点=重新开始），单行=range(n,n)，无选区=document。Markdown 渲染器扩容：+管道表格（对齐老 renderMd）+blockquote（User 拍板新增）。
>
> **-L4（大模型通路=本体需求；「DSH 插件 vs 独立 web」只是壳）**：无论壳形态，exec/review 环节的真执行必须**经 DSH 调用底层大模型**——参照 novel 库 `be/llm/dsh.go` 模式：rd 服务内建 llm adapter（baseURL/apiKey/model 配置化，prompt 模板驱动结构化 JSON 输出+围栏剥离），seed capability how=prompt 素材（读输入=上游入库产物）。produce 通路：节点激活→LLM 产出文档→服务端入库（wb_add_artifact，-K kind=document）→判据内聚推进；review 通路：LLM 按评审能力 how 读被评审产物→意见挂锚（-L3 划词/document）+表态；gate 拍板保留 User 专属；LLM 失败→任务回退+need_intervene 人工接管。落地节奏：先单节点真跑通（bus-01 to-brd）再全流程。2026-09-09 拍板①「一期自包含 Web」修订为：自包含 Web **含 LLM 通路**（无 LLM 的编排引擎不算可用 MVP——User 2026-09-10 深夜重定义）。
| GET | /api/projects · POST /api/projects | （registry） | ProjectReq/Resp | dsh 嵌入期启用 |

- **移出 agent/HTTP 清单**：wb_decide（门禁判定 = 内部 collect 判定 + User 拍板专用路由 `POST /api/requirements/:reqId/nodes/:nodeId/decision`，DecisionReq/Resp，**已注册实现**）、wb_override（escalate/adjudicate 取代，`POST /api/comments/:commentId/adjudicate` 承载）——语义保留在 service 层。
- **实时通道**：收件箱 badge/节点状态变更 = 前端轮询（MVP）；SSE/WebSocket 预留（/api/events，v1.1）。

### 1.3 内部分层（Go）

```
handler/   Gin 路由薄壳：参数绑定 + 日志（入口 log req / defer log resp）+ 转发 service
service/   业务编排：判据校验、collect 聚合、门禁公式、幂等、审计调用（每个 wb_* 语义一个 Service）
domain/    状态机与纯逻辑：review_results 聚合判定、issue 状态机 v2、门禁公式、结论协议装配——无 IO，可单测
repo/      MySQL 访问（GORM）： PreparedStmt + SkipDefaultTransaction（写路径显式事务）
```

- 事务边界：写路径（上报/表态/裁决）单事务 + 行锁（SELECT … FOR UPDATE 于 flow_nodes 聚合判定，防并发双 collect）；通知意图经 EnqueueNotificationTx 与业务同事务落 notify_outbox，推送（flush/sweep）在事务外 best-effort（2026-09-10 注记-H 升级口径）。
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
| 启动失败 | agent/request-error 监听 + 重试策略（沿用直启实测结论）——**归属 DSH/plugin 侧（wbflow 插件），非 rd 服务端待办** |
| 门禁拦截 | 判据不满足的操作返回 `{blocked:true, reason}`，审计留痕 |
| 并发 collect 竞争 | flow_nodes 行锁 + 单事务（聚合判定读改写原子）；通知意图同事务入 outbox，推送事务外 best-effort |
| 澄清不收敛 | loop_count 达阈值（如 reopen≥5）→ 自动标 escalate → User 仲裁（沿用 wb_reopen roundCount>5 通知语义） |
| MySQL 不可用 | 服务降级：API 路由不注册（仅 /healthz 上报 db missing）；恢复后重启进程续跑（当前实现为降级启动+healthz 上报，「GET 走缓存快照」为 v1.1 预留口径，本期未实现） |
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
| 前端 | 流程树 SVG 一次渲染全节点（≤40 定义节点上限，无性能问题）；阅读器大文档分段渲染 |

---

## 4. 验收标准

> **AC 编号规范（2026-09-11 拍板）**：版本级条目编 `AC-V-NN`、模块级条目编 `AC-M-NN`——AC id 是稳定追溯键，条目正文可改写、位置可移动，id 不变。AC↔用例双向映射维护在 `docs/v1.0/qa/qa-test-cases.md` §1 覆盖矩阵「AC 引用」列；机器断言（每条 AC 至少一个已执行 cid 覆盖）在 `qa/qa_drive.py` 运行尾执行，spec 新增 AC 未声明覆盖即 FAIL。

### 4.1 版本级（roadmap P1–P6 全链路）

- [ ] AC-V-01：新会话以 team preset + 团队制度启动，wb_* 28 工具（HTTP 版，含 wb_advance/wb_list_requirement_history）对 agent 可用（P1）
- [ ] AC-V-02：MySQL v0.29 全量表 + seed（四流程 40 定义节点 + 21 能力）初始化通过（P2）
- [ ] AC-V-03：一条 full 需求端到端：对齐（to-brd）→ 产品设计 → 三轮评审（per-agent 表态/澄清/复核/agreed→approved）→ RD∥FE 并行开发（CR 内嵌）→ QA 验收（仅 full 渲染）→ 用户验收 → 完成，全流程无阻塞（P5）
- [ ] AC-V-04：一条 rd-only 需求：流程树动态重排为 9 节点（FE 侧与 QA 不渲染、不占位），走通至用户验收（P5）
- [ ] AC-V-05：一条 task 需求（7 节点无 QA）：走通至用户验收，报障直连开发执行者（P5）
- [ ] AC-V-06：需求对齐：entry hold 会话 → 用户确认 → to-brd 落盘 → 迁移（P6）
- [ ] AC-V-07：打回链路：评审 unagree→to-fix → rejected → exec 重入（revise）→ 重新提测 → QA retest 复验 → resolved（P5）
- [ ] AC-V-08：冲突上升：escalate → 裁决 session → 结论回填 issue → reviewer 重表态 → 恢复（P5）
- [ ] AC-V-09：收件箱七类通知触发与跳转正确；TODO 投影一致（P5）
- [ ] AC-V-10：扫描器：5min 催办 / 10min 介入按时触发（注入时钟模拟）（P2）
- [ ] AC-V-11：幂等：重复 complete/重复派发/同轮覆写无害（P2）
- [ ] AC-V-12：审计：全部写路径落 audit_log（P2）

### 4.2 模块级 AC

- 模块 I/II/III 基线 PRD 的全部 AC 沿用（既有实现已验证的交互回归，不另编号）+ 本文增补点新增 AC：
- [ ] AC-M-01：流程树 scope 重排：切换/实例化三种 scope，节点集与序号符合渲染矩阵，被裁节点不占位
- [ ] AC-M-02：拍板卡三态切换：进度只读 → agreed 迎宾呈现（汇总+按钮组+跳转）→ approved/rejected
- [ ] AC-M-03：冲突卡：escalate 聚合、正反对照、裁决回填 issue、跳转节点详情
- [ ] AC-M-04：per-agent 表态：review_results 单字段读写一致；reviewing→agree/unagree 迁移判据正确

---

## Testing Decisions

- **接缝测试**：httptest 直打 Gin 路由（最高接缝），覆盖全部 28 端点正常/拦截/幂等分支。
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
- **设计权威冲突时**：以 `docs/v1.0/arch_design.md`（单一设计权威，2026-09-12 MIG-07 五件套并入）为准，本 spec 为实现翻译层；翻译发现设计缺口 → 回设计层裁决，不自行发挥。
- **DSH 嵌入预留**：API-first 已保证；嵌入期仅新增认证适配器 + conversation.view 挂载，无 API 变更。
