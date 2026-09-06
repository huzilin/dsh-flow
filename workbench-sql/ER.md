# 团队工作台 · 数据模型 ER 图（erDiagram）

> 依据：`.workbench/workbench.db` 实际 schema（11 张表）+ README 注记 8 关联矩阵。
> 关联约定（v0.16）：子表关联列与父表业务主键**同名同值（TEXT）**；`id`（INTEGER）仅作内部主键，不做跨表关联；无外键约束（逻辑引用）。
> 基数标记：`||--o{` = 1 对多；`||--||` = 1 对 1（可空用 o|）。

```mermaid
erDiagram
    %% ================= 角色 / 能力 / 模板（定义层） =================
    ROLES {
        INTEGER id PK
        TEXT role_id UK "业务主键 rd/fe/cr/arch/pm/qa/inv/leader/user"
        TEXT name
        TEXT note "角色工作内容注释（非子 agent persona；完整 persona 见 m7 角色卡 + team preset）"
        INTEGER created_at "unix ms"
        INTEGER updated_at "unix ms"
    }
    CAPABILITIES {
        INTEGER id PK
        TEXT capability_id UK "backend-design / rd-tech-review ..."
        TEXT name
        TEXT role_id FK "引用 ROLES.role_id（1对多，角色拥有能力）"
        TEXT how "JSON 行为步骤（能力本体，v0.23）"
        TEXT products "JSON [{doc_path,template_path,demo_path}] 产出物清单（v0.28 合并三列；一能力可多产物；评审/决策类 '[]'；产物项内 template/demo 为 JSON null=未固化）"
        INTEGER created_at "unix ms"
        INTEGER updated_at "unix ms"
    }
    FLOW_TEMPLATES {
        INTEGER id PK
        TEXT flow_templates_id UK "business / tech / task / plan"
        TEXT scope_options "JSON scope 选项（节点拓扑见 FLOW_TEMPLATE_NODES）"
        INTEGER created_at "unix ms"
        INTEGER updated_at "unix ms"
    }
    FLOW_TEMPLATE_NODES {
        INTEGER id PK
        TEXT flow_templates_id FK "引用 FLOW_TEMPLATES.flow_templates_id（1对多）"
        INTEGER node_order "编排顺序"
        TEXT key "节点名 产品设计/需求评审/..."
        TEXT role_id FK "执行者（引用 ROLES.role_id）"
        TEXT capability_id FK "引用 CAPABILITIES.capability_id（可空=NULL=align 无产物节点）"
        TEXT mode "produce=有产物常规节点｜align=无产物自主对齐（v0.28）"
        INTEGER gate "0/1 是否门禁"
        TEXT branch "rd/fe 泳道（''=非并行）"
        TEXT parallel_group "dev-1 并行标记（''=非并行）"
        TEXT scope "裁剪条件 ''=全量｜be=仅后端｜fe=仅前端"
        TEXT reviewers "JSON [{role,capability_id,scope?}] 评审者集合（v0.28 移除 focus，各审各的用各自评审能力）"
        TEXT expected_outputs "JSON 期望产物（capability_id 引用）"
        INTEGER created_at "unix ms"
        INTEGER updated_at "unix ms"
    }

    %% ================= 实例层 =================
    FLOWS {
        INTEGER id PK
        TEXT requirement_id UK "req-001"
        TEXT flow_templates_id FK "引用 FLOW_TEMPLATES.flow_templates_id"
        TEXT current_node_id FK "当前指针（引用 FLOW_NODES.node_id，可空）"
        INTEGER created_at "unix ms"
        INTEGER updated_at "unix ms"
    }
    FLOW_NODES {
        INTEGER id PK
        TEXT requirement_id FK "引用 FLOWS.requirement_id（1对多）"
        TEXT node_id UK "n-00..."
        INTEGER node_order
        TEXT key
        TEXT role_id FK "执行者（引用 ROLES.role_id）"
        INTEGER gate
        TEXT branch "''=非并行"
        TEXT parallel_group "''=非并行"
        TEXT reviewers "JSON [{role,capability_id,scope?}] 静态评审声明（v0.28；v0.18 取代 FLOW_NODE_REVIEWERS）"
        TEXT feature_branch "NULL=未敲定"
        TEXT state "未开始/进行中/待评审/已通过"
        INTEGER entered_at "节点进入时间（NULL=未进入）"
        INTEGER rolled_back_count
        TEXT last_decision "JSON {verdict,reason,at,branch?}"
        TEXT pending_scope
        INTEGER created_at "实例化时间"
        INTEGER updated_at "unix ms"
    }
    TASKS {
        INTEGER id PK
        TEXT task_id UK "t-<ts>-<seq>"
        TEXT requirement_id FK "引用 FLOWS.requirement_id"
        TEXT node_id FK "引用 FLOW_NODES.node_id"
        TEXT branch "泳道 rd/fe（记录位置，不承担隔离；''=无）"
        TEXT role_id FK "执行者（引用 ROLES.role_id）"
        TEXT task_type "produce/review/accept/decide"
        TEXT goal
        TEXT context "JSON 确定性上下文"
        TEXT status "todo/in-progress/awaiting-reply/done/rejected/closed"
        TEXT root_issue_id FK "锚定根 issue（线程入口，可空）"
        INTEGER reply_count "线程回复数"
        INTEGER last_reply_at "线程最新回复时间"
        INTEGER dispatch_at "派发时间（证明已派出，v0.19）"
        INTEGER last_ack "subagent 最近确认时刻（NULL=未送达，v0.19）"
        TEXT last_progress "执行到哪一步（断点续接锚点，v0.19）"
        TEXT resume_hint "断点续接提示（指向节点/产物/下一步，v0.19）"
        INTEGER created_at
        INTEGER updated_at
        INTEGER done_at
        INTEGER closed_at
    }
    ARTIFACTS {
        INTEGER id PK
        TEXT artifact_id UK "art-<ts>"
        TEXT requirement_id FK "引用 FLOWS.requirement_id"
        TEXT node_id FK "引用 FLOW_NODES.node_id"
        TEXT label
        TEXT href
        TEXT kind "document/link"
        TEXT note "''=无备注"
        TEXT added_by "登记人"
        INTEGER created_at "登记（创建）时间（原 added_at）"
        INTEGER updated_at "unix ms"
    }
    ISSUES {
        INTEGER id PK
        TEXT issue_id UK "cm-<ts>"
        TEXT requirement_id FK "引用 FLOWS.requirement_id"
        TEXT node_id FK "引用 FLOW_NODES.node_id"
        TEXT anchor_kind "document/line/range/block"
        TEXT anchor_artifact_id FK "锚点（引用 ARTIFACTS.artifact_id，可空）"
        INTEGER anchor_line
        INTEGER anchor_start_line
        INTEGER anchor_end_line
        INTEGER anchor_block_start
        INTEGER anchor_block_end
        TEXT anchor_lang
        TEXT author
        TEXT role_id FK "作者角色（引用 ROLES.role_id）"
        TEXT reviewer_role
        TEXT issue_type "issue/notification/discussion"
        TEXT assignee
        TEXT text "根评论内容"
        TEXT replies "JSON [{reply_id,by,role_id,text,at}] 不可变回复（v0.20 并入 ISSUE_REPLIES）"
        TEXT state "open/pending-confirm/resolved/overridden"
        INTEGER created_at "根评论创建时间（原 at）"
        INTEGER round_count
        INTEGER last_reply_at
        TEXT resolve_by
        INTEGER resolve_at
        TEXT resolve_note
        TEXT resolve_path "fix/agree"
        INTEGER resolve_reviewer_confirmed
        TEXT resolved_by
        INTEGER resolved_at
        INTEGER updated_at "unix ms"
    }

    %% ================= 支撑表 =================
    AUDIT_LOG {
        INTEGER id PK
        TEXT table_name
        TEXT row_id "业务主键值"
        TEXT action "INSERT/UPDATE/DELETE"
        TEXT field "UPDATE 时；INSERT/DELETE NULL"
        TEXT old_value
        TEXT new_value
        TEXT by_who
        INTEGER created_at "审计事件时间（原 at）"
        INTEGER updated_at "恒=created_at（审计行不可变）"
    }
    META {
        INTEGER id PK
        TEXT meta_key UK
        TEXT meta_value
        INTEGER created_at "unix ms"
        INTEGER updated_at "unix ms"
    }

    %% ================= 关联（子表列 = 父表业务主键，同名） =================
    ROLES ||--o{ CAPABILITIES : "role_id（角色拥有能力 1:n）"
    ROLES ||--o{ FLOW_TEMPLATE_NODES : "role_id（节点执行者 1:n）"
    ROLES ||--o{ FLOW_NODES : "role_id（实例执行者 1:n）"
    ROLES ||--o{ TASKS : "role_id（任务执行者 1:n）"
    ROLES ||--o{ ISSUES : "role_id（评论作者/回复者 1:n，回复在 replies JSON）"

    CAPABILITIES ||--o{ FLOW_TEMPLATE_NODES : "capability_id（节点用什么能力 1:n）"

    FLOW_TEMPLATES ||--o{ FLOW_TEMPLATE_NODES : "flow_templates_id（模板→节点 1:n）"
    FLOW_TEMPLATES ||--o{ FLOWS : "flow_templates_id（模板→实例 1:n）"

    FLOWS ||--o{ FLOW_NODES : "requirement_id（流程→节点 1:n）"
    FLOWS ||--o{ TASKS : "requirement_id（流程→任务 1:n）"
    FLOWS ||--o{ ARTIFACTS : "requirement_id（流程→产物 1:n）"
    FLOWS ||--o{ ISSUES : "requirement_id（流程→评论 1:n）"
    FLOWS o|--o| FLOW_NODES : "current_node_id（当前指针 1:1，可空）"

    FLOW_NODES ||--o{ TASKS : "node_id（节点→任务 1:n）"
    FLOW_NODES ||--o{ ARTIFACTS : "node_id（节点→产物 1:n）"
    FLOW_NODES ||--o{ ISSUES : "node_id（节点→评论 1:n）"

    TASKS o|--o{ ISSUES : "root_issue_id（待办→锚定线程，可空）"
    ARTIFACTS o|--o{ ISSUES : "anchor_artifact_id（评论锚点→产物，可空）"
```

---

## 图例说明

| 符号 | 含义 |
|---|---|
| `||--o{` | 父表 1 → 子表 n |
| `||--o|` | 父表 1 → 子表 1（子侧可空） |
| `o|--o|` | 双向可空 1:1（如 current_node_id 指针） |
| PK | 内部主键（INTEGER AUTOINCREMENT） |
| UK | 业务主键（TEXT UNIQUE，可读/跨系统引用） |
| FK | 逻辑引用（无外键约束；关联列与父表业务主键同名） |

## 三类分组

1. **定义层**（模板/注册表）：ROLES、CAPABILITIES、FLOW_TEMPLATES、FLOW_TEMPLATE_NODES
2. **实例层**（运行时状态）：FLOWS、FLOW_NODES、TASKS、ARTIFACTS、ISSUES
3. **支撑层**（审计/元信息）：AUDIT_LOG、META

> v0.17：产物模板已内联进 CAPABILITIES（doc_path/structure/quality_gates/version），DOCUMENT_TEMPLATES 独立实体删除。
> v0.18（User 拍板）：INBOX 并入 TASKS（唯一待办总线）；FLOW_NODE_REVIEWERS 并入 FLOW_NODES.reviewers（JSON）；SNAPSHOTS 删除。
> v0.19（User 拍板）：CONVERSATIONS 删除（IM 是视图非数据，通知由查询 TASKS/ISSUES 实时推导）；TASKS 加派发/断点交接字段（dispatch_at/last_ack/last_progress/resume_hint）——13 → 12 实体。
> v0.20（User 拍板）：ISSUE_REPLIES 并入 ISSUES.replies（JSON，不可变追加；无按回复行查询需求）——12 → 11 实体。
> v0.21（User 拍板）：NULL 策略收紧——非必要不 NULL；branch/parallel_group/note 为 NOT NULL DEFAULT ''。
> v0.22（User 拍板）：每实体增加 created_at/updated_at（unix ms）；旧等价字段改名——ARTIFACTS.added_at→created_at、ISSUES.at→created_at、AUDIT_LOG.at→created_at（audit_log 也加 updated_at 恒=created_at）；FLOW_NODES.entered_at 保留（业务时间≠行创建）。
> v0.23（User 拍板）：CAPABILITIES 收缩——删 structure/quality_gates/version/reviewer_focus/inputs；加 template_path/demo_path（模板外置 docs/templates/）；how 强化为能力本体。
> v0.28（User 拍板，m8 §10 节点-能力绑定模型）：CAPABILITIES.doc_path/template_path/demo_path → 单列 products（JSON 数组，一能力可多产物；评审/决策类 '[]'）；QA 能力合并（acceptance-test 挂 2 产物，defect-management 删除）；FLOW_TEMPLATE_NODES 加 mode（produce|align）、capability_id 可空（align 无产物节点）；reviewers 统一 [{role,capability_id,scope?}]（focus 移除）；角色缩写（architect→arch、investigator→inv）。
