-- =====================================================================
-- wbflow 14 表全量 DDL（迁移 v1: create_schema）
-- 正本：.plan/wbflow-impl/spec.md §0 + §0.3（逐表照录，编号与列注释保持一致）
-- 约定（spec §0 头注）：BIGINT UNSIGNED 自增主键 / 无物理外键 / updated_at 乐观锁 /
--   JSON 载荷 TEXT 列 / 默认 NOT NULL（仅 NULL 清单列保留 NULL）/ 双时间戳 unix 毫秒应用层维护
-- 禁止自动 DROP：本文件只含 CREATE TABLE。
-- =====================================================================

-- 1. 能力注册表
CREATE TABLE IF NOT EXISTS capabilities (
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
CREATE TABLE IF NOT EXISTS roles (
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
CREATE TABLE IF NOT EXISTS flow_defs (
  id            BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  flow_defs_id  VARCHAR(32) NOT NULL COMMENT '流程类型：business/tech/task/plan',
  scope_options TEXT        NOT NULL COMMENT '可选 scope 集合 JSON 文本：business/tech=["full","rd","fe"]，task=["rd","fe"]，plan=[]',
  created_at    BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at    BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_flow_defs_fid (flow_defs_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流程定义表';

-- 4. 定义节点（原 flow_template_nodes）
CREATE TABLE IF NOT EXISTS flow_def_nodes (
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
CREATE TABLE IF NOT EXISTS flows (
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
CREATE TABLE IF NOT EXISTS flow_nodes (
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
CREATE TABLE IF NOT EXISTS tasks (
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
CREATE TABLE IF NOT EXISTS artifacts (
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
CREATE TABLE IF NOT EXISTS issues (
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
CREATE TABLE IF NOT EXISTS sessions (
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
CREATE TABLE IF NOT EXISTS audit_log (
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
CREATE TABLE IF NOT EXISTS meta (
  id         BIGINT UNSIGNED AUTO_INCREMENT COMMENT '纯自增主键',
  meta_key   VARCHAR(64) NOT NULL COMMENT '元信息键，如 schema_version',
  meta_value TEXT        NOT NULL COMMENT '元信息值',
  created_at BIGINT      NOT NULL COMMENT '创建时间，unix 毫秒',
  updated_at BIGINT      NOT NULL COMMENT '更新时间，unix 毫秒（乐观锁）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_meta_key (meta_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='元信息表';

-- 13. 需求（原 requirements.json 收编）
CREATE TABLE IF NOT EXISTS requirements (
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
CREATE TABLE IF NOT EXISTS versions (
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
