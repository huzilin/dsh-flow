// repo 包本文件：TEXT 列 JSON 载荷的 Go struct 单一口径（spec §0.2——结构定义以 Go struct 为准）。
// JSON 键为 camelCase（与 proto 契约 string 承载同构）；seed 与运行时序列化共用本文件，
// 序列化工具收敛于 pkg/utils（MarshalJSONText/UnmarshalJSONText）。
package repo

// 渲染 scope 取值（需求 scope 三值，data-tools-final §1.2）。
const (
	ScopeFull = "full"
	ScopeRD   = "rd"
	ScopeFE   = "fe"
)

// 可配动作相位（node.state 值 + 空=未进入编排；done=离场终态，注记-L2）。
const (
	PhaseEmpty   = ""
	PhaseEntry   = "entry"
	PhaseExec    = "exec"
	PhaseReview  = "review"
	PhaseRespond = "respond"
	// PhaseDone 业务终态（BUG-016 拍板，注记-L2）：节点被推进离场时置入——
	// 语义=「已通过/已完成」（老模型 state='已通过' 的 v1.0 对应物）；done 节点
	// 禁止挂载产物/任务，打回重入时由回退核复位重进。
	PhaseDone = "done"
)

// 节点类型（flow_def_nodes.node_type 4 类）。
const (
	NodeTypeProduce = "produce"
	NodeTypeReview  = "review"
	NodeTypeAlign   = "align"
	NodeTypeGate    = "gate"
)

// per-agent 评审表态三态（flow_nodes.review_results 条目 state）。
const (
	ReviewStateReviewing = "reviewing"
	ReviewStateAgree     = "agree"
	ReviewStateUnagree   = "unagree"
)

// node_actions 条目 kind 两类（data-tools-final §1.2）。
const (
	NodeActionKindCapability = "capability"
	NodeActionKindNode       = "node"
)

// node action 执行者（kind=node 条目 executor）。
const (
	ExecutorProgram  = "program"
	ExecutorSubagent = "subagent"
)

// CapabilityInput capabilities.inputs 条目：输入可见性契约。
// 只收录 useThreads=true 条目（capability-actions-final §2 口径）。
type CapabilityInput struct {
	// ProductType 输入产物类型
	ProductType string `json:"productType"`
	// UseThreads 是否携带讨论线程
	UseThreads bool `json:"useThreads"`
}

// ActionDef capabilities.actions 条目：能力内可独立调度单元。
// on=挂载相位（entry/exec/review/respond）；exec 首次/重入双配置由 rolled_back_count 选择。
type ActionDef struct {
	// ActionID 动作标识（能力内唯一，如 produce/revise/dev）
	ActionID string `json:"actionId"`
	// Name 动作显示名
	Name string `json:"name"`
	// On 挂载相位
	On string `json:"on"`
	// Skill 挂载的 DSH skill（复杂动作）；空=无
	Skill string `json:"skill,omitempty"`
	// Inline 一句话内联定义（短动作）；空=无
	Inline string `json:"inline,omitempty"`
	// OutputProductTypes 产出产物类型（受控枚举）；评审类动作为空=正确语义（issue 即产物）
	OutputProductTypes []string `json:"outputProductTypes"`
}

// ProductDef capabilities.products 条目：产物登记清单（路径占位符沿用 docs 约定）。
// v0.29 起无 template_path：产出标准以产出 skill 为权威，DB 不存标准指针。
type ProductDef struct {
	// DocPath 产物文档相对路径
	DocPath string `json:"docPath"`
	// DemoPath 示例路径；未固化为空
	DemoPath string `json:"demoPath,omitempty"`
}

// ReviewerDef flow_def_nodes.reviewers 条目：评审位静态声明（审批 skills map 派生源）。
type ReviewerDef struct {
	// Role 评审角色（wb_review 表态者）
	Role string `json:"role"`
	// CapabilityID 评审能力（按哪个评审能力审）
	CapabilityID string `json:"capabilityId"`
	// ScopeExpr 位级渲染表达式（按需评审位）；空=随节点恒在
	ScopeExpr string `json:"scopeExpr,omitempty"`
}

// NodeActionDef flow_def_nodes.node_actions 条目（两类，data-tools-final §1.2）。
// kind=capability：节点动作激活清单（能力动作=通用词表，节点决定激活子集）；
// kind=node：node action（开场白/to-brd 等不属角色能力的公共动作）。
type NodeActionDef struct {
	// Kind 条目类型：capability | node
	Kind string `json:"kind"`
	// —— kind=capability 字段 ——
	// CapabilityID 激活的能力
	CapabilityID string `json:"capabilityId,omitempty"`
	// Actions 激活的动作 id 子集
	Actions []string `json:"actions,omitempty"`
	// —— 条目级 scope（task 流程 by-scope 激活）；空=随节点 ——
	ScopeExpr string `json:"scopeExpr,omitempty"`
	// —— kind=node 字段 ——
	// ActionID node action 标识（如 align.opening / to-brd）
	ActionID string `json:"actionId,omitempty"`
	// On 挂载相位
	On string `json:"on,omitempty"`
	// Executor 执行者：program | subagent
	Executor string `json:"executor,omitempty"`
	// ExecutorRole 响应者角色；空=按节点会话绑定动态（如用户验收）
	ExecutorRole string `json:"executorRole,omitempty"`
	// Skill 挂载 skill；空=无
	Skill string `json:"skill,omitempty"`
	// Inline 内联定义；空=无
	Inline string `json:"inline,omitempty"`
}

// ExpectedOutput flow_def_nodes.expected_outputs 条目：期望产物。
type ExpectedOutput struct {
	// ProductType 产物受控类型
	ProductType string `json:"productType"`
	// Label 显示名
	Label string `json:"label"`
	// Kind document | link
	Kind string `json:"kind"`
}

// ReviewResult flow_nodes.review_results 条目：per-agent 评审表态（collect 数据源）。
type ReviewResult struct {
	// Role 表态角色
	Role string `json:"role"`
	// CapabilityID 评审能力
	CapabilityID string `json:"capabilityId"`
	// State reviewing | agree | unagree
	State string `json:"state"`
	// Round 评审轮（与 rolled_back_count 对齐）
	Round int64 `json:"round"`
	// At 表态时间，unix 毫秒
	At int64 `json:"at"`
}

// 任务族（tasks.task_type 六值，spec §0 表 #7 列注释；
// 完成判据按族表驱动，见 domain.CompleteCriteria）。
const (
	TaskTypeProduce    = "produce"
	TaskTypeDev        = "dev"
	TaskTypeReview     = "review"
	TaskTypeRespond    = "respond"
	TaskTypeAccept     = "accept"
	TaskTypeCreateReqs = "create_reqs"
)

// TaskContext tasks.context TEXT 列结构（确定性上下文装配单一口径；
// wb_assign_task 装配、执行方反序列化消费）。产物引用/意见清单等由 T6/T7
// 在既有字段集上扩展或经 Supplement 注入，不破坏本结构键位。
type TaskContext struct {
	// Requirement 需求摘要（同源 wb_get_requirement 快照行）
	Requirement TaskContextRequirement `json:"requirement"`
	// Node 目标节点摘要（同源 wb_my_task node）
	Node TaskContextNode `json:"node"`
	// Action 动作定位摘要（capabilities.actions / node_actions 条目同源）
	Action TaskContextAction `json:"action"`
	// Scope 需求渲染 scope：full/rd/fe；空=无（plan）
	Scope string `json:"scope"`
	// Branch 泳道：rd/fe/空
	Branch string `json:"branch"`
	// Supplement 调用方补充上下文 JSON 文本（AssignTaskReq.context 原样透传；
	// 空=无补充）。双字节原文存储，不做键级合并（装配键位不被覆盖）
	Supplement string `json:"supplement,omitempty"`
	// DependencyDocs 评审任务的依赖文档（tasks.context.dependencyDocs 落实现，
	// BUG-011③：被评审产物=直接上游连续 produce 段已登记产物，派发即随行——
	// agent 零跳转；仅 task_type=review 注入，空=无）
	DependencyDocs []TaskContextDependencyDoc `json:"dependencyDocs,omitempty"`
}

// TaskContextDependencyDoc 依赖文档条目（被评审产物摘要，my-task relatedArtifacts
// 同口径子集 + 锚定所需 nodeId）。
type TaskContextDependencyDoc struct {
	// ArtifactID 产物业务 id（评论锚 artifactId 取此值）
	ArtifactID string `json:"artifactId"`
	// NodeID 产物所在节点（上游 produce 实例节点）
	NodeID string `json:"nodeId"`
	// NodeKey 节点名
	NodeKey string `json:"nodeKey"`
	// ProductType 产物受控类型
	ProductType string `json:"productType"`
	// Label 产物显示名
	Label string `json:"label"`
	// Kind document | link
	Kind string `json:"kind"`
	// Href 相对项目路径或外链
	Href string `json:"href"`
}

// TaskContextRequirement 需求摘要。
type TaskContextRequirement struct {
	// RequirementID 需求业务 id
	RequirementID string `json:"requirementId"`
	// Title 标题
	Title string `json:"title"`
	// Status 需求状态
	Status string `json:"status"`
}

// TaskContextNode 节点摘要。
type TaskContextNode struct {
	// NodeID 实例节点 id
	NodeID string `json:"nodeId"`
	// DefNodeID 定义节点引用
	DefNodeID string `json:"defNodeId"`
	// NodeKey 节点名
	NodeKey string `json:"nodeKey"`
	// State 可配相位
	State string `json:"state"`
}

// TaskContextAction 动作摘要。
type TaskContextAction struct {
	// ActionID 动作定位（入参 action_id 回显）
	ActionID string `json:"actionId"`
	// CapabilityID 所属能力；kind=node 动作为空
	CapabilityID string `json:"capabilityId,omitempty"`
	// Name 动作显示名（node action 无独立名，取 action_id）
	Name string `json:"name"`
	// On 挂载相位；未知为空
	On string `json:"on,omitempty"`
	// Skill 挂载 skill；空=无
	Skill string `json:"skill,omitempty"`
	// Inline 一句话内联定义；空=无
	Inline string `json:"inline,omitempty"`
}

// IssueReply issues.replies 条目：不可变追加（回复 / reopen 理由 / 上升与裁决回填
// 同通道；spec §0.2 JSON 载荷 Go struct 单一口径，arch_design §Issue 示例同构）。
// T12 扩展：kind 区分普通回复与上升/裁决条目（空=普通回复，既有条目不受影响）；
// escalation/adjudication 条目经 verdict/sessionId 承载裁决 session 关联
// （sessions 表无 issue 外键列，replies 条目即 issue↔session 关联单一载体）。
type IssueReply struct {
	// ReplyID 回复业务 id（线程内序 r-<n>）
	ReplyID string `json:"replyId"`
	// By 回复者
	By string `json:"by"`
	// RoleID 回复者角色
	RoleID string `json:"roleId"`
	// Text 回复内容（误发 reply 标注作废 / 上升理由 / 裁决结论同走本字段）
	Text string `json:"text"`
	// Voided 作废标注；true=误发作废
	Voided bool `json:"voided,omitempty"`
	// At 回复时间，unix 毫秒
	At int64 `json:"at"`
	// Kind 条目类型（T12）：空=普通回复 / escalation=冲突上升条目 /
	// adjudication=User 仲裁裁决条目
	Kind string `json:"kind,omitempty"`
	// Verdict 仲裁结论（仅 kind=adjudication）：uphold | override
	Verdict string `json:"verdict,omitempty"`
	// SessionID 关联裁决 session 业务 id（kind=escalation/adjudication 携带；
	// sessions 表按 node 关联，issue 级关联经本字段定位）
	SessionID string `json:"sessionId,omitempty"`
}
