// Package repo GORM MySQL 访问层基线：spec §0 全量 14 表 model、
// updated_at 乐观锁 helper、append-only 审计写入 helper、应用层时间戳 helper。
//
// 约定（rd/README.md + spec §0）：
//   - 主键 id BIGINT UNSIGNED 自增；int 统一 int64；
//   - 不建物理外键，xx_id 列 + 索引，完整性由 service 层保证；
//   - 乐观锁 = updated_at（UPDATE 带读取时值，0 行命中 → ErrOptimisticLock），无独立 version 列；
//   - JSON 载荷存 TEXT 列，序列化经 pkg/utils（结构口径见 payload.go Go struct）；
//   - NULL 保留清单列用指针类型，其余一律值类型；
//   - 各表业务查询后续工单按需添加，本包只交付基线。
package repo

// Capability 能力注册表（表 #1 capabilities）。
type Capability struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// CapabilityID 能力唯一标识，如 backend-dev / arch.prd-review
	CapabilityID string `gorm:"column:capability_id;uniqueIndex:uk_capabilities_cid;size:64;notNull" json:"capabilityId"`
	// Name 能力显示名
	Name string `gorm:"column:name;size:128;notNull" json:"name"`
	// RoleID 归属角色，逻辑引用 roles.role_id
	RoleID string `gorm:"column:role_id;index:idx_capabilities_role;size:32;notNull" json:"roleId"`
	// Inputs 输入可见性契约 JSON 文本
	Inputs string `gorm:"column:inputs;type:TEXT;notNull" json:"inputs"`
	// Actions 动作清单 JSON 文本
	Actions string `gorm:"column:actions;type:TEXT;notNull" json:"actions"`
	// Products 产物清单 JSON 文本
	Products string `gorm:"column:products;type:TEXT;notNull" json:"products"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Capability) TableName() string { return "capabilities" }

// Role 角色注册表（表 #2 roles）。
type Role struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// RoleID 角色标识：rd/fe/qa/arch/pm/cr/inv/leader/user/executor
	RoleID string `gorm:"column:role_id;uniqueIndex:uk_roles_rid;size:32;notNull" json:"roleId"`
	// Name 角色显示名
	Name string `gorm:"column:name;size:64;notNull" json:"name"`
	// Note 角色工作内容注释（非 persona）
	Note string `gorm:"column:note;size:512;notNull;default:''" json:"note"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Role) TableName() string { return "roles" }

// FlowDef 流程定义表（表 #3 flow_defs，原 flow_templates）。
type FlowDef struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// FlowDefsID 流程类型：business/tech/task/plan
	FlowDefsID string `gorm:"column:flow_defs_id;uniqueIndex:uk_flow_defs_fid;size:32;notNull" json:"flowDefsId"`
	// ScopeOptions 可选 scope 集合 JSON 文本
	ScopeOptions string `gorm:"column:scope_options;type:TEXT;notNull" json:"scopeOptions"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (FlowDef) TableName() string { return "flow_defs" }

// FlowDefNode 定义节点表（表 #4 flow_def_nodes，原 flow_template_nodes）。
type FlowDefNode struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// FlowDefsID 所属流程，逻辑引用 flow_defs.flow_defs_id
	FlowDefsID string `gorm:"column:flow_defs_id;uniqueIndex:uk_fdnode_order;index:idx_fdnode_tpl;size:32;notNull" json:"flowDefsId"`
	// TemplateNodeID 定义节点稳定 id，如 bus-03
	TemplateNodeID string `gorm:"column:template_node_id;uniqueIndex:uk_fdnode_tnid;size:32;notNull" json:"templateNodeId"`
	// NodeOrder 编排序号，1 起
	NodeOrder int64 `gorm:"column:node_order;uniqueIndex:uk_fdnode_order;notNull" json:"nodeOrder"`
	// NodeKey 节点名
	NodeKey string `gorm:"column:node_key;size:64;notNull" json:"nodeKey"`
	// NodeType 节点类型：produce/review/align/gate
	NodeType string `gorm:"column:node_type;size:16;notNull" json:"nodeType"`
	// Gate 是否门禁：1=有准出判定
	Gate int64 `gorm:"column:gate;notNull;default:0" json:"gate"`
	// UserGate User 终审拍板位：1=agreed 后唤醒 User
	UserGate int64 `gorm:"column:user_gate;notNull;default:0" json:"userGate"`
	// ScopeExpr 渲染集合表达式：""=any / rd|full / fe|full / full / rd|fe
	ScopeExpr string `gorm:"column:scope_expr;size:64;notNull;default:''" json:"scopeExpr"`
	// Reviewers 评审位声明 JSON 文本
	Reviewers string `gorm:"column:reviewers;type:TEXT;notNull" json:"reviewers"`
	// NodeActions 动作激活清单 JSON 文本
	NodeActions string `gorm:"column:node_actions;type:TEXT;notNull" json:"nodeActions"`
	// ExpectedOutputs 期望产物 JSON 文本
	ExpectedOutputs string `gorm:"column:expected_outputs;type:TEXT;notNull" json:"expectedOutputs"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (FlowDefNode) TableName() string { return "flow_def_nodes" }

// Flow 流程实例表（表 #5 flows）。
type Flow struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// RequirementID 需求 id，逻辑引用 requirements.requirement_id
	RequirementID string `gorm:"column:requirement_id;uniqueIndex:uk_flows_req;size:32;notNull" json:"requirementId"`
	// FlowDefsID 流程类型，逻辑引用 flow_defs.flow_defs_id
	FlowDefsID string `gorm:"column:flow_defs_id;size:32;notNull" json:"flowDefsId"`
	// Dependencies 前置依赖 JSON 文本：["req-…"]（同版本内依赖边）
	Dependencies string `gorm:"column:dependencies;type:TEXT;notNull" json:"dependencies"`
	// Scope 需求所选渲染 scope：full/rd/fe；空=无 scope（plan 流程）。
	// 实例节点集 = flow_def_nodes.scope_expr ∩ 本列（迁移 v3 落库，证据见 0003 头注）
	Scope string `gorm:"column:scope;size:8;notNull;default:''" json:"scope"`
	// CurrentNodeID 当前节点 id；NULL=未进入编排
	CurrentNodeID *string `gorm:"column:current_node_id;size:32" json:"currentNodeId"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Flow) TableName() string { return "flows" }

// FlowNode 实例节点表（表 #6 flow_nodes）。
type FlowNode struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// RequirementID 需求 id，逻辑引用 flows.requirement_id
	RequirementID string `gorm:"column:requirement_id;index:idx_flow_nodes_req;size:32;notNull" json:"requirementId"`
	// NodeID 实例节点 id，如 n-05
	NodeID string `gorm:"column:node_id;uniqueIndex:uk_flow_nodes_nid;size:32;notNull" json:"nodeId"`
	// DefNodeID 定义节点引用，逻辑引用 flow_def_nodes.template_node_id
	DefNodeID string `gorm:"column:def_node_id;index:idx_flow_nodes_def;size:32;notNull" json:"defNodeId"`
	// NodeOrder 编排序号（渲染序号，scope 过滤后重排）
	NodeOrder int64 `gorm:"column:node_order;notNull" json:"nodeOrder"`
	// NodeKey 节点名
	NodeKey string `gorm:"column:node_key;size:64;notNull" json:"nodeKey"`
	// State 可配相位：""=未进入编排 / entry / exec / review / respond
	State string `gorm:"column:state;size:16;notNull;default:''" json:"state"`
	// ReviewResults 评审轮结果 JSON 文本（单字段承载全部 agent，collect 反复更新与读取）
	ReviewResults string `gorm:"column:review_results;type:TEXT;notNull;default:'[]'" json:"reviewResults"`
	// ReopenSource 重开来源：""/upstream-reject/downstream-reject/rework-return
	ReopenSource string `gorm:"column:reopen_source;size:24;notNull;default:''" json:"reopenSource"`
	// RolledBackCount 回退计数：0=首次，≥1=重入
	RolledBackCount int64 `gorm:"column:rolled_back_count;notNull;default:0" json:"rolledBackCount"`
	// Reviewers 实例评审位声明 JSON 文本（按 scope_expr 过滤后的静态声明）
	Reviewers string `gorm:"column:reviewers;type:TEXT;notNull" json:"reviewers"`
	// FeatureBranch 评审敲定后的分支名；NULL=尚未敲定
	FeatureBranch *string `gorm:"column:feature_branch;size:128" json:"featureBranch"`
	// EnteredAt 进入执行时间；NULL=未进入
	EnteredAt *int64 `gorm:"column:entered_at" json:"enteredAt"`
	// PendingScope 打回待重定范围；NULL=无
	PendingScope *string `gorm:"column:pending_scope;type:TEXT" json:"pendingScope"`
	// LastDecision 门禁裁决结果 JSON 文本 {verdict,reason,at,branch?}；NULL=未判定
	// （approved/rejected=collect 聚合输出；迁移 v3 补列，arch_design v0.29 同源）
	LastDecision *string `gorm:"column:last_decision;type:TEXT" json:"lastDecision"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (FlowNode) TableName() string { return "flow_nodes" }

// Task 任务待办总线（表 #7 tasks）。
type Task struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// TaskID 任务业务 id，如 t-<ts>-<seq>
	TaskID string `gorm:"column:task_id;uniqueIndex:uk_tasks_tid;size:40;notNull" json:"taskId"`
	// RequirementID 需求 id
	RequirementID string `gorm:"column:requirement_id;index:idx_tasks_req;size:32;notNull" json:"requirementId"`
	// NodeID 节点 id
	NodeID string `gorm:"column:node_id;index:idx_tasks_node;size:32;notNull" json:"nodeId"`
	// ActionID 能力×动作定位，如 backend-dev.dev
	ActionID string `gorm:"column:action_id;index:idx_tasks_action;size:64;notNull" json:"actionId"`
	// Branch 泳道：rd/fe/空
	Branch string `gorm:"column:branch;size:8;notNull;default:''" json:"branch"`
	// RoleID 执行角色
	RoleID string `gorm:"column:role_id;index:idx_tasks_role;size:32;notNull" json:"roleId"`
	// TaskType 任务族：produce/dev/review/respond/accept/create_reqs
	TaskType string `gorm:"column:task_type;size:16;notNull" json:"taskType"`
	// Goal 任务目标
	Goal string `gorm:"column:goal;type:TEXT;notNull" json:"goal"`
	// Context 确定性上下文 JSON 文本
	Context string `gorm:"column:context;type:TEXT;notNull" json:"context"`
	// Status todo/in-progress/awaiting-reply/completed/rejected/closed
	Status string `gorm:"column:status;index:idx_tasks_status;size:16;notNull;default:'todo'" json:"status"`
	// RootIssueID 锚定根 issue；NULL=非 issue 驱动
	RootIssueID *string `gorm:"column:root_issue_id;size:32" json:"rootIssueId"`
	// ReplyCount 回复数
	ReplyCount int64 `gorm:"column:reply_count;notNull;default:0" json:"replyCount"`
	// LastReplyAt 最近回复时间；NULL=无回复
	LastReplyAt *int64 `gorm:"column:last_reply_at;index:idx_tasks_last_reply" json:"lastReplyAt"`
	// DispatchAt 派发时间；NULL=未派发
	DispatchAt *int64 `gorm:"column:dispatch_at;index:idx_tasks_dispatch" json:"dispatchAt"`
	// LastAck 最近确认时刻；NULL=未送达（断点恢复关键）
	LastAck *int64 `gorm:"column:last_ack" json:"lastAck"`
	// LastProgress 执行进度；NULL=未开始
	LastProgress *string `gorm:"column:last_progress;type:TEXT" json:"lastProgress"`
	// ResumeHint 断点续接提示；NULL=无
	ResumeHint *string `gorm:"column:resume_hint;type:TEXT" json:"resumeHint"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
	// DoneAt 完成时间；NULL=未完成（completed 判据后置）
	DoneAt *int64 `gorm:"column:done_at" json:"doneAt"`
	// ClosedAt 关闭时间（终态）；NULL=未关闭
	ClosedAt *int64 `gorm:"column:closed_at" json:"closedAt"`
}

// TableName 表名。
func (Task) TableName() string { return "tasks" }

// Artifact 产物表（表 #8 artifacts）。
type Artifact struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// ArtifactID 产物业务 id，如 art-<ts>
	ArtifactID string `gorm:"column:artifact_id;uniqueIndex:uk_artifacts_aid;size:32;notNull" json:"artifactId"`
	// RequirementID 需求 id
	RequirementID string `gorm:"column:requirement_id;index:idx_artifacts_req,priority:1;size:32;notNull" json:"requirementId"`
	// NodeID 节点 id
	NodeID string `gorm:"column:node_id;index:idx_artifacts_req,priority:2;size:32;notNull" json:"nodeId"`
	// ProductType 产物受控类型（无 case_design）
	ProductType string `gorm:"column:product_type;index:idx_artifacts_ptype;size:32;notNull" json:"productType"`
	// Label 产物显示名
	Label string `gorm:"column:label;size:128;notNull" json:"label"`
	// Href 相对项目路径（doc）或外链（link）
	Href string `gorm:"column:href;size:512;notNull" json:"href"`
	// Kind document | link
	Kind string `gorm:"column:kind;size:16;notNull;default:'document'" json:"kind"`
	// Note 备注，空串=无
	Note string `gorm:"column:note;size:512;notNull;default:''" json:"note"`
	// AddedBy 登记人角色
	AddedBy string `gorm:"column:added_by;size:32;notNull" json:"addedBy"`
	// CreatedAt 登记时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Artifact) TableName() string { return "artifacts" }

// Issue 评论/问题表（表 #9 issues，issue 状态机 v2）。
type Issue struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// IssueID 评论业务 id，如 cm-<ts>
	IssueID string `gorm:"column:issue_id;uniqueIndex:uk_issues_iid;size:32;notNull" json:"issueId"`
	// RequirementID 需求 id
	RequirementID string `gorm:"column:requirement_id;index:idx_issues_node,priority:1;size:32;notNull" json:"requirementId"`
	// NodeID 节点 id
	NodeID string `gorm:"column:node_id;index:idx_issues_node,priority:2;size:32;notNull" json:"nodeId"`
	// AnchorKind 锚点：document/line/range/block
	AnchorKind string `gorm:"column:anchor_kind;size:16;notNull;default:'document'" json:"anchorKind"`
	// AnchorArtifactID 锚定产物 id；NULL=document 级
	AnchorArtifactID *string `gorm:"column:anchor_artifact_id;size:32" json:"anchorArtifactId"`
	// AnchorLine 单行号（kind=line）
	AnchorLine *int64 `gorm:"column:anchor_line" json:"anchorLine"`
	// AnchorStartLine 范围起行（kind=range）
	AnchorStartLine *int64 `gorm:"column:anchor_start_line" json:"anchorStartLine"`
	// AnchorEndLine 范围终行（kind=range）
	AnchorEndLine *int64 `gorm:"column:anchor_end_line" json:"anchorEndLine"`
	// AnchorBlockStart 代码块起行（kind=block）
	AnchorBlockStart *int64 `gorm:"column:anchor_block_start" json:"anchorBlockStart"`
	// AnchorBlockEnd 代码块终行（kind=block）
	AnchorBlockEnd *int64 `gorm:"column:anchor_block_end" json:"anchorBlockEnd"`
	// AnchorLang 代码块语言（kind=block）
	AnchorLang *string `gorm:"column:anchor_lang;size:24" json:"anchorLang"`
	// Author 发表者
	Author string `gorm:"column:author;size:32;notNull" json:"author"`
	// RoleID 发表者角色
	RoleID string `gorm:"column:role_id;size:32;notNull" json:"roleId"`
	// ReviewerRole 评审归属角色；NULL=未推导/非评审
	ReviewerRole *string `gorm:"column:reviewer_role;size:32" json:"reviewerRole"`
	// IssueType issue/notification/discussion
	IssueType string `gorm:"column:issue_type;size:16;notNull;default:'issue'" json:"issueType"`
	// Assignee 解决责任人；NULL=未指派
	Assignee *string `gorm:"column:assignee;size:32;index:idx_issues_assignee" json:"assignee"`
	// Text 根评论内容
	Text string `gorm:"column:text;type:TEXT;notNull" json:"text"`
	// State issue 状态机 v2：open/pending-confirm/to-fix/fixed/resolved/overridden
	State string `gorm:"column:state;index:idx_issues_state;size:20;notNull;default:'open'" json:"state"`
	// Escalate 冲突上升标记：1=拉起裁决 session
	Escalate int64 `gorm:"column:escalate;index:idx_issues_escalate;notNull;default:0" json:"escalate"`
	// LoopCount 回退计数：澄清期读=reopen 次数，修复期读=revise 次数
	LoopCount int64 `gorm:"column:loop_count;notNull;default:0" json:"loopCount"`
	// Replies 回复与裁决回填 JSON 文本（不可变追加）
	Replies string `gorm:"column:replies;type:TEXT;notNull" json:"replies"`
	// RoundCount 对话轮数
	RoundCount int64 `gorm:"column:round_count;notNull;default:0" json:"roundCount"`
	// LastReplyAt 最近回复时间；NULL=无回复
	LastReplyAt *int64 `gorm:"column:last_reply_at" json:"lastReplyAt"`
	// ResolveBy resolve 责任人；NULL=未解决
	ResolveBy *string `gorm:"column:resolve_by;size:32" json:"resolveBy"`
	// ResolveAt resolve 时间；NULL=未解决
	ResolveAt *int64 `gorm:"column:resolve_at" json:"resolveAt"`
	// ResolveNote resolve 说明；NULL=未解决
	ResolveNote *string `gorm:"column:resolve_note;type:TEXT" json:"resolveNote"`
	// ResolvePath fix=修复关闭 / agree=协商一致关闭；NULL=未解决
	ResolvePath *string `gorm:"column:resolve_path;size:8" json:"resolvePath"`
	// ResolveReviewerConfirmed 评审者复核确认：0/1
	ResolveReviewerConfirmed int64 `gorm:"column:resolve_reviewer_confirmed;notNull;default:0" json:"resolveReviewerConfirmed"`
	// ResolvedBy 最终确认人；NULL=未闭环
	ResolvedBy *string `gorm:"column:resolved_by;size:32" json:"resolvedBy"`
	// ResolvedAt 最终确认时间；NULL=未闭环
	ResolvedAt *int64 `gorm:"column:resolved_at" json:"resolvedAt"`
	// CreatedAt 根评论创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Issue) TableName() string { return "issues" }

// Session 会话绑定表（表 #10 sessions）。
type Session struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// SessionID 会话业务 id
	SessionID string `gorm:"column:session_id;uniqueIndex:uk_sessions_sid;size:64;notNull" json:"sessionId"`
	// RequirementID 关联需求；空串=无
	RequirementID string `gorm:"column:requirement_id;index:idx_sessions_req;size:32;notNull;default:''" json:"requirementId"`
	// NodeID 关联节点；空串=无
	NodeID string `gorm:"column:node_id;index:idx_sessions_node;size:32;notNull;default:''" json:"nodeId"`
	// RoleID 绑定角色（persona 按节点配置传入）
	RoleID string `gorm:"column:role_id;size:32;notNull" json:"roleId"`
	// DshRef DSH 会话句柄
	DshRef string `gorm:"column:dsh_ref;type:TEXT;notNull" json:"dshRef"`
	// Status active/ended/disposed
	Status string `gorm:"column:status;size:12;notNull;default:'active'" json:"status"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Session) TableName() string { return "sessions" }

// AuditLog 审计日志（表 #11 audit_log，append-only：created_at=updated_at 恒同值）。
type AuditLog struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// Table 被变更表名（列 table_name——字段名避开 GORM Tabler 方法名 TableName）
	Table string `gorm:"column:table_name;index:idx_audit_row,priority:1;size:32;notNull" json:"table"`
	// RowID 业务主键值
	RowID string `gorm:"column:row_id;index:idx_audit_row,priority:2;size:64;notNull" json:"rowId"`
	// Action INSERT/UPDATE/DELETE
	Action string `gorm:"column:action;size:8;notNull" json:"action"`
	// Field 变更字段；INSERT/DELETE 为 NULL
	Field *string `gorm:"column:field;size:64" json:"field"`
	// OldValue 旧值；INSERT/不适字段为 NULL
	OldValue *string `gorm:"column:old_value;type:TEXT" json:"oldValue"`
	// NewValue 新值；DELETE/不适字段为 NULL
	NewValue *string `gorm:"column:new_value;type:TEXT" json:"newValue"`
	// ByWho 操作者：role_id/user/system/plugin
	ByWho string `gorm:"column:by_who;size:32;notNull" json:"byWho"`
	// CreatedAt 审计时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 恒=created_at（append-only 统一占位；由 InsertAuditLogWith 强制覆盖，不加 autoUpdateTime）
	UpdatedAt int64 `gorm:"column:updated_at;notNull" json:"updatedAt"`
}

// TableName 表名。
func (AuditLog) TableName() string { return "audit_log" }

// Meta 元信息表（表 #12 meta，迁移账本 schema_version 载体）。
type Meta struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// MetaKey 元信息键，如 schema_version
	MetaKey string `gorm:"column:meta_key;uniqueIndex:uk_meta_key;size:64;notNull" json:"metaKey"`
	// MetaValue 元信息值
	MetaValue string `gorm:"column:meta_value;type:TEXT;notNull" json:"metaValue"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Meta) TableName() string { return "meta" }

// Requirement 需求表（表 #13 requirements，原 requirements.json 收编）。
type Requirement struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// RequirementID 需求业务 id，如 req-001
	RequirementID string `gorm:"column:requirement_id;uniqueIndex:uk_requirements_rid;size:32;notNull" json:"requirementId"`
	// Title 标题
	Title string `gorm:"column:title;size:256;notNull" json:"title"`
	// Description Markdown 描述
	Description string `gorm:"column:description;type:MEDIUMTEXT;notNull" json:"description"`
	// Category 类别标签：需求任务/规划审批任务/版本规划任务（决定流程类型绑定）
	Category string `gorm:"column:category;size:32;notNull" json:"category"`
	// FlowDefsID 流程类型，逻辑引用 flow_defs（由 category 绑定派生）
	FlowDefsID string `gorm:"column:flow_defs_id;size:32;notNull" json:"flowDefsId"`
	// VersionID 归属版本；NULL=待分配池
	VersionID *string `gorm:"column:version_id;size:32;index:idx_requirements_ver" json:"versionId"`
	// Priority P0-P3
	Priority string `gorm:"column:priority;size:8;notNull;default:'P2'" json:"priority"`
	// Status 待启动/进行中/已完成/已取消
	Status string `gorm:"column:status;index:idx_requirements_status;size:16;notNull;default:'待启动'" json:"status"`
	// Owner 负责人角色
	Owner string `gorm:"column:owner;size:32;notNull;default:''" json:"owner"`
	// CreatedBy 创建人
	CreatedBy string `gorm:"column:created_by;size:32;notNull" json:"createdBy"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Requirement) TableName() string { return "requirements" }

// Version 版本表（表 #14 versions，原 versions.json 收编）。
type Version struct {
	// ID 纯自增主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	// VersionID 版本业务 id，如 v-1
	VersionID string `gorm:"column:version_id;uniqueIndex:uk_versions_vid;size:32;notNull" json:"versionId"`
	// Name 版本名（自由文本）
	Name string `gorm:"column:name;size:64;notNull" json:"name"`
	// State 待启动/进行中/已发布/已废弃
	State string `gorm:"column:state;size:16;notNull;default:'待启动'" json:"state"`
	// IsBacklog 待分配池标记
	IsBacklog int64 `gorm:"column:is_backlog;notNull;default:0" json:"isBacklog"`
	// CreatedAt 创建时间，unix 毫秒
	CreatedAt int64 `gorm:"column:created_at;notNull;autoCreateTime:milli" json:"createdAt"`
	// UpdatedAt 更新时间，unix 毫秒（乐观锁）
	UpdatedAt int64 `gorm:"column:updated_at;notNull;autoUpdateTime:milli" json:"updatedAt"`
}

// TableName 表名。
func (Version) TableName() string { return "versions" }
