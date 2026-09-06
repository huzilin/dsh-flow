-- =====================================================================
-- 团队工作台 · sqlite 建库脚本（schema.sql）
-- 依据：docs/design/m8-flow-simplify.md §3.6（核心表结构）
--       §3.4 / §10（capabilities 能力模型；v0.24 节点-能力绑定模型重设计）
-- 约定：
--   * 所有表 id INTEGER PRIMARY KEY AUTOINCREMENT（纯内部行 id）
--   * 业务主键（capability_id/role_id/requirement_id/task_id/...）TEXT NOT NULL UNIQUE
--   * 无外键约束，跨表引用一律逻辑引用（<父表>_id 列）+ 普通索引
--   * roles → capabilities 1 对多（capabilities.role_id 唯一归属，无关联表）
--   * 时间 INTEGER(unix ms, epoch 毫秒)；枚举/状态 TEXT；布尔 INTEGER(0/1)；计数 INTEGER；JSON 结构 TEXT
--   * v0.18：inbox 并入 tasks；flow_node_reviewers 并入 flow_nodes.reviewers(JSON)；snapshots 删除
--   * v0.19：conversations 删除（IM 是视图非数据，通知由查询 tasks/issues 实时推导）；
--     tasks 加派发/断点交接字段（dispatch_at/last_ack/last_progress/resume_hint）
--   * v0.20：issue_replies 并入 issues.replies（JSON，不可变追加；无按回复行查询需求）；
--     12 表 → 11 表
--   * NULL 策略（v0.21）：非必要不使用 NULL——仅当「空字符串有歧义」或「字段语义上
--     '无值' 与 '空串' 必须区分」时才允许 NULL；其余一律 NOT NULL + 合理 DEFAULT。
--     NULL 保留清单见各表注释（模板缺省/未发生时间/动作不适用/锚点类型不适用）。
--   * 全表时间戳（v0.22）：每张表都有 created_at / updated_at（unix ms）。
--     已有等价字段改名对齐：artifacts.added_at → created_at；issues.at → created_at；
--     audit_log.at → created_at（audit_log 也加 updated_at，User 拍板"严格全表都有"）。
--     flow_nodes.entered_at 保留（业务语义=节点进入时间，≠行创建）。
--     DEFAULT = 当前时间毫秒（CAST(strftime('%s','now') AS INTEGER)*1000），
--     seed 列名式 INSERT 未指定即自动填充；运行时由应用层显式覆盖。
--   * 每张表建表语句后附【JSON 示例数据】（/* */ 块注释，1 条或多条）：
--     字段名与该表建表语句一一对应，便于理解每个字段的含义与取值。
--   * v0.28（§10 节点-能力绑定模型，User 拍板落库）：
--     - capabilities：doc_path/template_path/demo_path 三列 → 单列 products（JSON 数组，
--       [{doc_path, template_path, demo_path}]，一能力可多产物；评审/决策类 '[]'）——A 点
--     - flow_template_nodes：capability_id 改可空（NULL=无产出能力，mode='align' 节点）；
--       新增 mode 列（'produce'|'align'）——B 点；「需求对齐」类无产物节点 = 无产出能力，
--       子 agent 自主决策（需求不清晰 → 与 User IM 澄清；清晰 → 认知确认 → 再开发），
--       对齐结论落 issues(discussion)，wb_my_task 组装链把全流程讨论一并打包给 subagent
--     - flow_template_nodes.reviewers 统一为 [{role, capability_id, scope?}]（评审者集合，
--       各审各的；capability_id=该评审者所用评审能力；scope 按需评审位）——C 点；
--       focus 移除（评审维度已在评审能力自身 how 内，v0.23 拍板）
-- 执行：sqlite3 workbench.db < schema.sql
-- =====================================================================

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = OFF;

-- ---------------------------------------------------------------------
-- 1. 能力注册表（全局一份；how 承载能力本体——"我是谁、输入后如何做出合格产出"）
--    roles → capabilities 1 对多：role_id 唯一归属（一个角色拥有多个能力；
--    每个能力只属于一个角色，能力不共享——同一评审节点上各角色用各自的能力）
-- 产物（v0.23 起模板文件外置 + v0.28 合并为 products 数组）：
--   products（JSON 数组）= 产出物清单，一能力可多产物（如 qa-acceptance 合并后
--   挂 qa/test.md + qa/defect.md 两项）；每项：doc_path（产出文档存放路径，
--   含 <版本号>/<需求名> 占位）、template_path（章节骨架+质量门槛模板文件，
--   docs/templates/xxx-template.md）、demo_path（填充示例，docs/templates/xxx-demo.md）。
--   模板文件外置（v0.23）：章节骨架/质量门禁由文件承载，不内联 DB。
-- v0.23 收缩：reviewer_focus / inputs / quality_gates 三个字段已删除——
--   评审维度并入"评审能力"自身的 how；依赖文档由派发时的 task.context 承载；
--   质量门槛由模板文件承载。能力表只描述归属 + 行为 + 产出模板。
-- v0.28（§10 A 点，User 拍板）：doc_path/template_path/demo_path 三列 → 单列 products。
--   NULL 策略变化：products 用 '[]' 表示无产出（评审/决策类能力），不再用 NULL 列——
--   模板未固化的产物项其 template_path/demo_path 为 JSON null（如 backend-dev）。
-- ---------------------------------------------------------------------
CREATE TABLE capabilities (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  capability_id    TEXT NOT NULL UNIQUE, -- 'backend-design' | 'backend-dev' | …（唯一标识）
  name             TEXT NOT NULL,        -- 能力显示名（如'后端技术设计'）
  role_id          TEXT NOT NULL,        -- 归属角色（逻辑引用 roles.role_id，1 对多）
  how              TEXT NOT NULL DEFAULT '[]',
                 -- JSON: [行为步骤]（能力本体）——描述作为本角色拿到上下文后怎么做：
                 --   如何与 prd 对齐 / 如何设计接口满足前端页面 / 如何与架构文档保持大方向一致 /
                 --   如何保证可测性 / 自检要点（评审类能力在此写明评审维度与表态流程）
  products         TEXT NOT NULL DEFAULT '[]',
                 -- JSON: [{doc_path, template_path, demo_path}]（产出物清单，v0.28 合并三列）
                 --   一能力可多产物（QA 验收能力 = 2 项：qa/test.md + qa/defect.md）；
                 --   评审/决策类能力 = '[]'（只输出意见/决策，不产出文档、无模板）；
                 --   产物项内 template_path/demo_path=null 表示该产物模板未固化
  created_at       INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms
  updated_at       INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)   -- unix ms
);

-- ---------------------------------------------------------------------
-- 2. 角色注册表（全局唯一；note = 角色工作内容注释，非子 agent persona）
-- 全字段 NOT NULL：角色定义无缺省可用
-- ---------------------------------------------------------------------
CREATE TABLE roles (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  role_id      TEXT NOT NULL UNIQUE,   -- 'rd' | 'fe' | 'cr' | 'arch' | 'pm' | 'qa' | 'inv' | 'leader' | 'user' | 'executor'
  name         TEXT NOT NULL,          -- 角色显示名（如'后端开发'）
  note         TEXT NOT NULL,          -- 角色工作内容注释（人读；非子 agent persona，完整 persona 见 docs/design/m7-agent-protocol.md 角色卡 + team preset，不落库）
  created_at   INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms
  updated_at   INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)   -- unix ms
);

-- ---------------------------------------------------------------------
-- 3. 流程模板（静态定义，4 种）
-- 全字段 NOT NULL：模板定义无缺省可用
-- ---------------------------------------------------------------------
CREATE TABLE flow_templates (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  flow_templates_id TEXT NOT NULL UNIQUE, -- 'business' | 'tech' | 'task' | 'plan'
  scope_options TEXT NOT NULL DEFAULT '[]',  -- JSON: ['full','fe-only','be-only']（可选范围；节点拓扑见 flow_template_nodes）
  created_at    INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms
  updated_at    INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)   -- unix ms
);

-- ---------------------------------------------------------------------
-- 4. 模板节点定义（模板的节点骨架：用什么能力/期望产物/评审人/并行编排）
--    关联：flow_templates_id 逻辑引用 flow_templates.flow_templates_id
-- NULL 策略：branch/parallel_group 非并行节点用 ''（空串=不在泳道/不在并行组，无歧义）
-- v0.28（§10 B/C 点，User 拍板）：
--   * capability_id 改可空——NULL = 无产出能力节点（mode='align' 需求对齐节点不绑产出能力，
--     subagent 自主决策：需求不清晰 → 与 User IM 澄清；清晰 → 产出认知 → User 确认 → 再开发；
--     对齐结论落 issues(issue_type='discussion')，wb_my_task 组装链把全流程讨论一并打包给 subagent）
--   * mode 列：'produce'=有产物常规节点（默认）｜'align'=无产物自主对齐节点
--   * reviewers 统一为 JSON: [{role, capability_id, scope?}]——评审者集合，各审各的：
--     role 决定谁审（wb_review 表态者），capability_id 决定按哪个评审能力审（how 内含该角色
--     评审维度）；条目 scope 可选：缺省随节点侧，全量节点侧特定评审位需显式声明。
--     focus 移除（v0.23 已并入评审能力自身 how，reviewers 不重复存）。
-- ---------------------------------------------------------------------
CREATE TABLE flow_template_nodes (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  flow_templates_id TEXT NOT NULL,      -- 逻辑引用 flow_templates.flow_templates_id
  node_order       INTEGER NOT NULL,    -- 节点序号（编排顺序，1 起）
  key              TEXT NOT NULL,       -- 节点名（产品设计/需求评审/…）
  role_id          TEXT NOT NULL,       -- 执行者角色（逻辑引用 roles.role_id）
  capability_id    TEXT,                -- 用什么能力执行（关联 capabilities.capability_id；NULL=无产出能力，mode='align' 节点）
  mode             TEXT NOT NULL DEFAULT 'produce',  -- 'produce'=有产物常规节点｜'align'=无产物自主对齐节点
  gate             INTEGER NOT NULL DEFAULT 0,   -- 0/1 是否门禁
  branch           TEXT NOT NULL DEFAULT '',     -- 'rd'|'fe'（并行组泳道；''=非并行节点）
  parallel_group   TEXT NOT NULL DEFAULT '',     -- 'dev-1'（并行编排标记；''=非并行节点）
  scope            TEXT NOT NULL DEFAULT '',     -- 裁剪条件：''=全量｜'be'=仅后端侧｜'fe'=仅前端侧（实例化时按 requirement.scope 过滤）
  reviewers        TEXT NOT NULL DEFAULT '[]',   -- JSON: [{role,capability_id,scope?}]（静态评审声明；条目 scope 可选：缺省随节点侧，全量节点侧特定评审位需显式声明）
  expected_outputs TEXT NOT NULL DEFAULT '[]',   -- JSON: [{capability_id,label,kind}]（产出物=引用 capabilities.capability_id）
  created_at       INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms
  updated_at       INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms
  UNIQUE (flow_templates_id, node_order)
);

-- ---------------------------------------------------------------------
-- 5. 流程实例（每个需求/任务一份）
-- NULL 策略：current_node_id 无当前节点时 NULL（空串会与真实 node_id 冲突）
-- ---------------------------------------------------------------------
CREATE TABLE flows (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  requirement_id  TEXT NOT NULL UNIQUE,  -- 业务主键（req-001）
  flow_templates_id TEXT NOT NULL,       -- 逻辑引用 flow_templates.flow_templates_id
  current_node_id TEXT,                  -- 当前指针（n-00…，逻辑引用 flow_nodes.node_id；NULL=未启动/无当前）
  created_at      INTEGER NOT NULL,         -- unix ms (epoch 毫秒)
  updated_at      INTEGER NOT NULL          -- unix ms (epoch 毫秒)
);

-- ---------------------------------------------------------------------
-- 6. 节点实例（独立 id，运行时状态；reviewers JSON 取代独立 flow_node_reviewers 表 v0.18）
-- NULL 策略：branch/parallel_group 非并行=''（同模板节点）；feature_branch 评审敲定前 NULL；
--   entered_at 节点未进入 NULL；last_decision 尚未决策 NULL；pending_scope 无待定范围 NULL
-- created_at = 节点实例化（行创建）时间；entered_at = 节点进入执行（业务状态）时间，保留区分
-- ---------------------------------------------------------------------
CREATE TABLE flow_nodes (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  requirement_id  TEXT NOT NULL,        -- 逻辑引用 flows.requirement_id
  node_id         TEXT NOT NULL,        -- n-00…（实例化时生成，独立可读 id）
  node_order      INTEGER NOT NULL,     -- 节点序号（同模板编排顺序）
  key             TEXT NOT NULL,        -- 节点名（从模板复制）
  role_id         TEXT NOT NULL,        -- 执行者角色（逻辑引用 roles.role_id）
  gate            INTEGER NOT NULL DEFAULT 0,   -- 0/1 是否门禁
  branch          TEXT NOT NULL DEFAULT '',     -- 'rd'|'fe'（并行组泳道；''=非并行节点）
  parallel_group  TEXT NOT NULL DEFAULT '',     -- 'dev-1'（并行编排标记；''=非并行节点）
  reviewers       TEXT NOT NULL DEFAULT '[]',   -- JSON: [{role, capability_id, scope?}] 静态评审声明（v0.28 与模板层同构；v0.18 取代独立表）
                              -- 动态「谁过了没」由 tasks(review)+issues(意见线程) 推导
  feature_branch  TEXT,                 -- 评审节点敲定后写入的 git 分支名（NULL=尚未敲定）
  state           TEXT NOT NULL DEFAULT '未开始',  -- 未开始/进行中/待评审/已通过（应用层白名单）
  entered_at      INTEGER,                 -- unix ms；节点进入执行时间（NULL=未进入）
  rolled_back_count INTEGER NOT NULL DEFAULT 0,   -- 回滚次数（替代 JSON 数组，计数）
  last_decision   TEXT,                 -- JSON: {verdict,reason,at,branch?}（NULL=尚未决策）
  pending_scope   TEXT,                 -- 打回时待重定范围（NULL=无待定范围）
  created_at      INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms（实例化）
  updated_at      INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)   -- unix ms
);

-- ---------------------------------------------------------------------
-- 7. 任务（唯一待办总线；v0.18 取代 inbox——派发记录 + 确定性上下文 + 状态机）
--    待回复场景：root_issue_id 锚定根 issue，其下 threads 由 issues.replies（v0.20）承载
-- NULL 策略（v0.19 断点交接）：
--   branch 非泳道=''；root_issue_id 非 task 驱动无线程=NULL（空串与真实 issue_id 冲突）；
--   last_reply_at 无回复 NULL；dispatch_at 已派发必有值；last_ack NULL=未送达（断点恢复关键）；
--   last_progress 未开始 NULL；resume_hint 无续接提示 NULL；done_at/closed_at 未完成 NULL
--   created_at = 任务创建；dispatch_at = 派发（=创建通常同时，但保留区分：预创建未派发可能）
-- ---------------------------------------------------------------------
CREATE TABLE tasks (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id    TEXT NOT NULL UNIQUE,      -- t-<ts>-<seq>（业务可读）
  requirement_id TEXT NOT NULL,          -- 逻辑引用 flows.requirement_id
  node_id    TEXT NOT NULL,             -- 逻辑引用 flow_nodes.node_id
  branch     TEXT NOT NULL DEFAULT '',  -- 泳道描述（'rd'|'fe'；v0.18 不再承担隔离，只记录物理位置；''=无）
  role_id    TEXT NOT NULL,             -- 负责角色（执行者）
  task_type  TEXT NOT NULL,             -- produce | review | accept | decide
  goal       TEXT NOT NULL,             -- 任务目标（确定性）
  context    TEXT NOT NULL,             -- JSON: 确定性上下文（nodeKey/branch/artifacts/dependencyDocs）
  status     TEXT NOT NULL DEFAULT 'todo',
             -- todo | in-progress | awaiting-reply | done | rejected | closed
             --   todo=待开始；awaiting-reply=等回复/表态（待回复 tab）
  root_issue_id TEXT,                   -- 锚定根 issue（线程入口；NULL=非 task 驱动的纯执行任务）
  reply_count  INTEGER NOT NULL DEFAULT 0,  -- 该待办线程当前回复数
  last_reply_at INTEGER,                -- 线程最新回复时间（列表排序；NULL=尚无回复）
  -- 派发/断点交接（v0.19：承载通知可靠性与断点续接，取代 conversations 通知表）
  dispatch_at  INTEGER,                 -- 主 agent 派发时间（unix ms；证明已派出；NULL=尚未派发）
  last_ack     INTEGER,                 -- subagent 最近确认收到时刻（心跳；NULL=未送达，需重发）
  last_progress TEXT,                   -- 执行到哪一步（如"已产出 design.md，待登记"；断点续接锚点；NULL=未开始）
  resume_hint  TEXT,                    -- 断点续接提示（如何恢复：指向节点/产物/下一步；NULL=无提示）
  created_at INTEGER NOT NULL,             -- unix ms (epoch 毫秒)
  updated_at INTEGER NOT NULL,             -- unix ms (epoch 毫秒)
  done_at    INTEGER,                      -- unix ms (epoch 毫秒)；NULL=未完成
  closed_at  INTEGER                       -- unix ms (epoch 毫秒，final）；NULL=未关闭
);

-- ---------------------------------------------------------------------
-- 8. 产物
-- NULL 策略：note 可选备注，无备注用 ''（空串无歧义）
-- added_by = 登记人；created_at = 登记（创建）时间（原 added_at，v0.22 改名对齐）
-- ---------------------------------------------------------------------
CREATE TABLE artifacts (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  artifact_id TEXT NOT NULL UNIQUE,     -- art-<ts>（业务可读）
  requirement_id TEXT NOT NULL,          -- 逻辑引用 flows.requirement_id
  node_id     TEXT NOT NULL,            -- 逻辑引用 flow_nodes.node_id
  label       TEXT NOT NULL,            -- 产物显示名（如《RD 技术设计》）
  href        TEXT NOT NULL,            -- 文档路径 / 链接 URL
  kind        TEXT NOT NULL DEFAULT 'document',  -- document | link
  note        TEXT NOT NULL DEFAULT '', -- 可选备注（''=无备注）
  added_by    TEXT NOT NULL,            -- 登记人（role_id）
  created_at  INTEGER NOT NULL,            -- unix ms；登记（创建）时间（原 added_at）
  updated_at  INTEGER NOT NULL             -- unix ms
);

-- ---------------------------------------------------------------------
-- 9. 评论/问题/线程（单表 v0.20：根评论 + replies JSON 内嵌回复，issue_replies 已并入）
--    含锚点/指派/双路径状态；回复为不可变 JSON 数组（m7：只 append，不编辑/删除）
-- NULL 策略：
--   anchor_artifact_id 锚点未指向具体产物=NULL（document 级锚点，空串与真实 artifact_id 冲突）
--   anchor_line/start_line/end_line/block_start/block_end/lang：仅对应 anchor_kind 适用时填写，
--     其余 NULL（如 line 锚点无 block_start；document 锚点全 NULL）——用 NULL 表示「该锚点类型不适用」，
--     空串对 INTEGER 列无意义
--   reviewer_role L3 推导前 NULL；assignee 未指派 NULL（空串与角色 id 冲突）
--   last_reply_at 无回复 NULL；resolve_* 族未解决前全 NULL（resolve_at 未解决 NULL，空时间戳无意义）
--   created_at = 根评论创建时间（原 at，v0.22 改名对齐；replies JSON 内 at 仍为回复时间）
-- ---------------------------------------------------------------------
CREATE TABLE issues (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id     TEXT NOT NULL UNIQUE,     -- cm-<ts>（业务可读）
  requirement_id TEXT NOT NULL,           -- 逻辑引用 flows.requirement_id
  node_id      TEXT NOT NULL,            -- 逻辑引用 flow_nodes.node_id
  anchor_kind  TEXT NOT NULL DEFAULT 'document',  -- document|line|range|block
  anchor_artifact_id TEXT,               -- 锚定产物（逻辑引用 artifacts.artifact_id；NULL=未锚定具体产物）
  anchor_line       INTEGER,             -- 单行号（仅 kind=line）
  anchor_start_line INTEGER,             -- 范围起行（仅 kind=range）
  anchor_end_line   INTEGER,             -- 范围终行（仅 kind=range）
  anchor_block_start INTEGER,            -- 代码块起行（仅 kind=block）
  anchor_block_end  INTEGER,             -- 代码块终行（仅 kind=block）
  anchor_lang  TEXT,                     -- 代码块语言（仅 kind=block；如 javascript/mermaid）
  author       TEXT NOT NULL,            -- 作者（role_id 或 'user'）
  role_id      TEXT NOT NULL,            -- 作者角色（逻辑引用 roles.role_id）
  reviewer_role TEXT,                    -- 评审归属（L3 domain 推导后；NULL=尚未推导/非评审评论）
  issue_type   TEXT NOT NULL DEFAULT 'issue',   -- issue|notification|discussion
  assignee     TEXT,                     -- 解决责任人（L1/L2/L3 归属；NULL=尚未指派）
  text         TEXT NOT NULL,            -- 根评论内容
  replies      TEXT NOT NULL DEFAULT '[]',
               -- JSON: [{reply_id, by, role_id, text, at}]（v0.20 并入 issue_replies；
               --   不可变追加；消费场景均为「读整个线程」，无按回复行查询需求）
  state        TEXT NOT NULL DEFAULT 'open',    -- open|pending-confirm|resolved|overridden
  created_at   INTEGER NOT NULL,            -- unix ms；根评论创建时间（原 at）
  round_count  INTEGER NOT NULL DEFAULT 0,  -- 回复轮数（wb_reply 递增）
  last_reply_at INTEGER,                 -- 最近回复时间（NULL=尚无回复）
  resolve_by   TEXT,                     -- 解决责任人（NULL=未解决）
  resolve_at   INTEGER,                  -- 解决时间（NULL=未解决）
  resolve_note TEXT,                     -- 解决说明（NULL=未解决/无说明）
  resolve_path TEXT,                     -- fix|agree（NULL=未解决）
  resolve_reviewer_confirmed INTEGER NOT NULL DEFAULT 0,  -- 0/1 评审者是否确认 closed
  resolved_by  TEXT,                     -- 最终确认人（NULL=未闭环）
  resolved_at  INTEGER,                  -- 最终确认时间（NULL=未闭环）
  updated_at   INTEGER NOT NULL             -- unix ms
);

-- ---------------------------------------------------------------------
-- 10. 审计（v0.18：唯一历史留痕，作 agent 每步改库的留痕；snapshots 已删）
--     主键 INTEGER，引用业务 id 用 TEXT 列
-- NULL 策略：field/old_value/new_value 视 action 而定——
--   INSERT：field/new_value 有值，old_value NULL（无旧值）；DELETE：old_value 有值，new_value NULL；
--   UPDATE：field 有值，old_value 可 NULL（若被更新字段原值为 NULL）。用 NULL 表示「该动作不适用/
--   该字段原值空缺」，空串会与"更新为空串"混淆，必须用 NULL。
-- v0.22：created_at = 审计事件时间（原 at）；updated_at 按 User 拍板严格添加
--   （审计行实际不可变，updated_at 恒等于 created_at，由应用层保证不更新审计行）
-- ---------------------------------------------------------------------
CREATE TABLE audit_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  table_name TEXT NOT NULL,             -- 被变更的表名
  row_id     TEXT NOT NULL,             -- 业务主键值（req-001 / n-09 / cm-xxx / t-xxx）
  action     TEXT NOT NULL,             -- INSERT | UPDATE | DELETE
  field      TEXT,                      -- 变更字段（UPDATE 时；INSERT/DELETE NULL）
  old_value  TEXT,                      -- 旧值（NULL=无旧值，如 INSERT）
  new_value  TEXT,                      -- 新值（NULL=无新值，如 DELETE）
  by_who     TEXT NOT NULL,             -- 操作者（role_id / 'user' / 'system'）
  created_at INTEGER NOT NULL,             -- unix ms；审计事件时间（原 at）
  updated_at INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)   -- unix ms（恒=created_at）
);

-- ---------------------------------------------------------------------
-- 11. 元信息（schema 版本/迁移）
-- 全字段 NOT NULL
-- ---------------------------------------------------------------------
CREATE TABLE meta (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  meta_key       TEXT NOT NULL UNIQUE,   -- 元信息键（如 schema_version）
  meta_value     TEXT NOT NULL,          -- 元信息值
  created_at     INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),  -- unix ms
  updated_at     INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)   -- unix ms
);

-- =====================================================================
-- 逻辑引用索引（加速 JOIN/查询；m8 §3.6.2「关联列建普通索引」约定）
-- =====================================================================
CREATE INDEX idx_capabilities_role ON capabilities(role_id);
CREATE INDEX idx_flow_template_nodes_tpl ON flow_template_nodes(flow_templates_id);
CREATE INDEX idx_flow_nodes_req   ON flow_nodes(requirement_id);
CREATE INDEX idx_tasks_req        ON tasks(requirement_id);
CREATE INDEX idx_tasks_node       ON tasks(node_id);
CREATE INDEX idx_tasks_role       ON tasks(role_id);
CREATE INDEX idx_tasks_status     ON tasks(status);            -- 主 agent 看卡点（待回复）
CREATE INDEX idx_tasks_last_reply ON tasks(last_reply_at);     -- 列表排序
CREATE INDEX idx_artifacts_req    ON artifacts(requirement_id, node_id);
CREATE INDEX idx_issues_node      ON issues(requirement_id, node_id);
CREATE INDEX idx_issues_assignee  ON issues(assignee);
CREATE INDEX idx_tasks_dispatch   ON tasks(dispatch_at);   -- 断点恢复：找未送达（last_ack NULL）或中断态
CREATE INDEX idx_audit_row        ON audit_log(table_name, row_id);