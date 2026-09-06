# 团队工作台 · 数据 Demo（11 张表 × 1 条示例）

> 目的：用一串**连贯的真实业务数据**（需求 req-019 的「RD 技术设计→评审」片段）展示每张表每个字段的含义。
> 格式：JSONC（`//` 行注释是字段说明；`null` 表示该字段按 NULL 策略保留为空）。
> 时间戳 = unix ms（与 `Date.now()` 一致）。表名按 schema.sql 顺序。
> 表中 `id` 为内部行 id（INTEGER AUTOINCREMENT），业务上一般不直接使用，示例里也给出便于理解。

---

## 1. capabilities —— 能力注册表（1 条：后端技术设计，产出型）

```jsonc
{
  "id": 1,
  "capability_id": "backend-design",        // 能力唯一标识
  "name": "后端技术设计",                    // 能力显示名
  "role_id": "rd",                          // 归属角色（1 对多：rd 拥有多个能力）
  "how": [                                  // JSON: 能力本体=行为步骤（wb_my_task 返回给角色照着做）
    "读输入：prd/prd.md（产品设计）+ 版本 arch_design.md（大方向）",
    "产出《RD 技术设计》：按模板 backend-design-doc 章节骨架逐节编写",
    "自检质量门禁：接口/数据设计对照 prd/prd.md+arch_design.md 一致",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评"
  ],
  "products": [                             // JSON: 产出物清单（v0.28 起三列合并；一能力可多产物）
    {
      "doc_path": "docs/<版本号>/<需求名>/rd/design.md",      // 产出文档存放路径
      "template_path": "docs/templates/backend-design-doc-template.md", // 模板文件（章节骨架+质量门槛）
      "demo_path": "docs/templates/backend-design-doc-demo.md"         // 填充示例
    }
  ],                                        // 评审/决策类能力 products=[]；产物项内 template/demo 为 JSON null=模板未固化
  "created_at": 1786000000000,              // 行创建时间
  "updated_at": 1786000500000               // 行最后修改时间（模板演进时刷新）
}
```

> **v0.23（User 拍板）**：`structure`/`quality_gates`/`version`/`reviewer_focus`/`inputs` 五个字段已删除——章节骨架与质量门槛改由**模板文件**承载（`template_path`/`demo_path`），评审维度写在**评审能力自身**的 how 里，依赖文档由派发时的 **task.context** 承载。能力表只描述：归属 + 行为（how）+ 产出模板。
> **v0.28（User 拍板，m8 §10）**：doc_path/template_path/demo_path 三列合并为单列 `products`（JSON 数组，一能力可多产物——如 QA 验收能力 acceptance-test 挂 qa/test.md + qa/defect.md 两产物）。

---

## 2. roles —— 角色注册表（1 条：rd）

```jsonc
{
  "id": 2,
  "role_id": "rd",        // 角色唯一标识（9 固定 + executor 动态占位）
  "name": "后端开发",      // 角色显示名
  "note": "负责技术设计与后端实现：领域建模、接口契约、数据一致性；对设计完整性与自测负责。",  // 角色工作内容注释（非子 agent persona，完整 persona 见 m7 角色卡）
  "created_at": 1786000000000,   // 行创建时间
  "updated_at": 1786000000000    // 行最后修改时间
}
```

---

## 3. flow_templates —— 流程模板（1 条：business 业务全流程）

```jsonc
{
  "id": 1,
  "flow_templates_id": "business",           // 模板唯一标识
  "scope_options": ["full", "fe-only", "be-only"],  // JSON: 可选实施范围（节点拓扑见 §4 flow_template_nodes）
  "created_at": 1786000000000,   // 行创建时间
  "updated_at": 1786000000000    // 行最后修改时间
}
```

---

## 4. flow_template_nodes —— 模板节点定义（1 条：RD 技术设计）

```jsonc
{
  "id": 5,
  "flow_templates_id": "business",      // 逻辑引用 flow_templates.flow_templates_id
  "node_order": 3,                      // 节点序号（编排顺序，0 起：0=待启动,1=产品设计,2=需求评审,3=RD 技术设计）
  "key": "RD 技术设计",                  // 节点名
  "role_id": "rd",                      // 执行者角色
  "capability_id": "backend-design",    // 用什么能力执行（关联 capabilities.capability_id；NULL=align 无产物节点）
  "mode": "produce",                    // produce=有产物常规节点｜align=无产物自主对齐（v0.28）
  "gate": 0,                            // 0/1 是否门禁（本节点非门禁）
  "branch": "",                         // 泳道（''=非并行节点；rd/fe=并行泳道）
  "parallel_group": "",                 // 并行组标记（''=非并行）
  "scope": "be",                        // 裁剪条件：''=全量｜'be'=仅后端侧｜'fe'=仅前端侧（实例化按 requirement.scope 过滤）
  "reviewers": [                        // JSON: 评审者集合（v0.28：[{role,capability_id,scope?}]，各审各的用各自评审能力，focus 在能力 how 内）
    { "role": "rd", "capability_id": "rd-tech-review", "scope": "be" },
    { "role": "arch", "capability_id": "arch-tech-review", "scope": "be" },
    { "role": "fe", "capability_id": "fe-tech-review", "scope": "fe" },
    { "role": "qa", "capability_id": "qa-tech-review" },
    { "role": "pm", "capability_id": "pm-review" }
  ],
  "expected_outputs": [                 // JSON: 期望产物（引用 capabilities.capability_id）
    { "capability_id": "backend-design", "label": "《RD 技术设计》", "kind": "document" }
  ],
  "created_at": 1786000000000,
  "updated_at": 1786000000000
}
```

---

## 5. flows —— 流程实例（1 条：req-019）

```jsonc
{
  "id": 12,
  "requirement_id": "req-019",              // 业务主键（每个需求/任务一份）
  "flow_templates_id": "business",          // 实例→模板
  "current_node_id": "n-04",                // 当前指针（逻辑引用 flow_nodes.node_id；null=未启动）
  "created_at": 1786000000000,
  "updated_at": 1786001000000               // 每次节点推进/状态变化都刷新
}
```

---

## 6. flow_nodes —— 节点实例（2 条：RD 技术设计已过 + 当前停在评审）

```jsonc
// ① n-03：RD 技术设计（已通过）
{
  "id": 33,
  "requirement_id": "req-019",
  "node_id": "n-03",                    // 实例化时生成，独立可读 id
  "node_order": 3,
  "key": "RD 技术设计",
  "role_id": "rd",
  "gate": 0,
  "branch": "",
  "parallel_group": "",
  "reviewers": "[]",                    // JSON: 静态评审声明（本节点无评审）
  "feature_branch": "feat/req-019-rd-design",  // 评审敲定后写入的分支名（本节点用于演示）
  "state": "已通过",                     // 未开始/进行中/待评审/已通过
  "entered_at": 1786000100000,          // 节点进入执行时间（null=未进入）
  "rolled_back_count": 0,               // 回滚次数（计数）
  "last_decision": null,                // JSON: {verdict,reason,at,branch?}（null=未决策；门禁节点才有）
  "pending_scope": null,                // 打回时待重定范围（null=无）
  "created_at": 1786000000000,          // 实例化时间
  "updated_at": 1786000800000
}

// ② n-04：RD 技术评审（门禁节点，当前待评审）
{
  "id": 34,
  "requirement_id": "req-019",
  "node_id": "n-04",
  "node_order": 4,
  "key": "RD 技术评审",
  "role_id": "rd",                      // 执行者（发起者 rd；评审由多位评审者共同完成）
  "gate": 1,                            // 门禁节点
  "branch": "",
  "parallel_group": "",
  "reviewers": [                        // JSON: 静态评审声明（v0.18 起取代独立表；v0.28 格式 = [{role,capability_id,scope?}]）
    { "role": "arch", "capability_id": "arch-tech-review" },
    { "role": "rd", "capability_id": "rd-tech-review" },
    { "role": "fe", "capability_id": "fe-tech-review" },
    { "role": "qa", "capability_id": "qa-tech-review" },
    { "role": "pm", "capability_id": "pm-review" }
  ],
  "feature_branch": null,               // 尚未敲定
  "state": "待评审",
  "entered_at": 1786000800000,
  "rolled_back_count": 0,
  "last_decision": null,                // 尚未决策
  "pending_scope": null,
  "created_at": 1786000000000,
  "updated_at": 1786000800000
}
```

---

## 7. tasks —— 任务（唯一待办总线，2 条：已完成产出 + 等待评审）

```jsonc
// ① t-…-001：派给 rd 的产出任务（已完成，展示断点字段的终态）
{
  "id": 41,
  "task_id": "t-20260815-001",
  "requirement_id": "req-019",
  "node_id": "n-03",                    // 绑定节点
  "branch": "",                         // 泳道（''=无）
  "role_id": "rd",                      // 负责角色
  "task_type": "produce",               // produce | review | accept | decide
  "goal": "产出《RD 技术设计》（n-03）",
  "context": {                          // JSON: 主 agent 组装好的确定性上下文
    "nodeKey": "RD 技术设计",
    "branch": "",
    "artifacts": [],
    "dependencyDocs": ["prd/prd.md", "arch_design.md"]
  },
  "status": "done",                     // todo | in-progress | awaiting-reply | done | rejected | closed
  "root_issue_id": null,                // 线程锚点（纯执行任务无线程）
  "reply_count": 0,
  "last_reply_at": null,
  "dispatch_at": 1786000010000,         // 派发时间（证明已派出）
  "last_ack": 1786000020000,            // subagent 确认收到时刻（心跳）
  "last_progress": "已登记产物 art-101",  // 断点续接锚点
  "resume_hint": null,                  // 已完成，无需续接
  "created_at": 1786000010000,
  "updated_at": 1786000900000,
  "done_at": 1786000900000,             // 完成时间
  "closed_at": 1786000900000            // 终态时间
}

// ② t-…-002：派给 arch 的评审任务（等待表态，展示待回复态 + 线程锚点）
{
  "id": 42,
  "task_id": "t-20260815-002",
  "requirement_id": "req-019",
  "node_id": "n-04",
  "branch": "",
  "role_id": "arch",
  "task_type": "review",
  "goal": "评审 RD 技术设计（n-04，架构视角）",
  "context": {
    "nodeKey": "RD 技术评审",
    "branch": "",
    "artifacts": [{ "id": "art-101", "label": "《RD 技术设计》", "href": "docs/v2.0/req-019/rd/design.md" }],
    "dependencyDocs": ["prd/prd.md", "rd/design.md", "arch_design.md"]
  },
  "status": "awaiting-reply",           // 待回复 tab
  "root_issue_id": "cm-1002",           // 锚定评审意见线程
  "reply_count": 2,                     // 线程已 2 轮回复
  "last_reply_at": 1786001200000,
  "dispatch_at": 1786000950000,
  "last_ack": 1786000960000,
  "last_progress": "已读 design.md，提出意见 cm-1002",
  "resume_hint": "等待 arch 对 cm-1002 的最终确认",
  "created_at": 1786000950000,
  "updated_at": 1786001200000,
  "done_at": null,
  "closed_at": null
}
```

---

## 8. artifacts —— 产物（1 条：RD 技术设计文档）

```jsonc
{
  "id": 101,
  "artifact_id": "art-101",
  "requirement_id": "req-019",
  "node_id": "n-03",                    // 挂到产出节点（多产物时同节点多条）
  "label": "《RD 技术设计》",            // 显示名
  "href": "docs/v2.0/req-019/rd/design.md",  // 文档路径（kind=link 时是 URL）
  "kind": "document",                   // document | link
  "note": "含接口契约 v2 修订",          // 可选备注（''=无）
  "added_by": "rd",                     // 登记人（角色）
  "created_at": 1786000800000,          // 登记（创建）时间
  "updated_at": 1786000800000
}
```

---

## 9. issues —— 评论/问题/线程（1 条：arch 的评审意见 + 2 轮回复）

```jsonc
{
  "id": 1002,
  "issue_id": "cm-1002",
  "requirement_id": "req-019",
  "node_id": "n-04",                    // 挂在评审节点
  "anchor_kind": "block",               // document|line|range|block（锚到代码块）
  "anchor_artifact_id": "art-101",      // 锚定产物
  "anchor_line": null,                  // 仅 kind=line
  "anchor_start_line": null,            // 仅 kind=range
  "anchor_end_line": null,              // 仅 kind=range
  "anchor_block_start": 42,             // 代码块起行（仅 kind=block）
  "anchor_block_end": 58,               // 代码块终行（仅 kind=block）
  "anchor_lang": "sql",                 // 代码块语言（仅 kind=block；如 javascript/mermaid/sql）
  "author": "arch",                // 作者
  "role_id": "arch",               // 作者角色
  "reviewer_role": null,                // L3 归属（null=非跨端缺陷）
  "issue_type": "issue",                // issue|notification|discussion
  "assignee": "rd",                     // 解决责任人（L1 锚点产物作者）
  "text": "数据模型表 user_order 缺唯一约束，并发下单会重复。",
  "replies": [                          // JSON: 不可变回复数组（v0.20 起内嵌）
    { "reply_id": "r-1", "by": "rd", "role_id": "rd", "text": "已补 UNIQUE(user_id, order_no)，见 design.md §3", "at": 1786001100000 },
    { "reply_id": "r-2", "by": "arch", "role_id": "arch", "text": "确认，可以关闭。", "at": 1786001200000 }
  ],
  "state": "pending-confirm",           // open|pending-confirm|resolved|overridden
  "created_at": 1786001000000,          // 根评论创建时间
  "round_count": 2,                     // 回复轮数
  "last_reply_at": 1786001200000,       // 最近回复时间
  "resolve_by": "rd",                   // 解决责任人（null=未解决）
  "resolve_at": 1786001100000,          // 解决时间（null=未解决）
  "resolve_note": "已补唯一约束",        // 解决说明
  "resolve_path": "fix",                // fix|agree（本例走 fix 修复路径）
  "resolve_reviewer_confirmed": 1,      // 评审者是否确认（pending-confirm 等确认）
  "resolved_by": null,                  // 最终确认人（null=未闭环）
  "resolved_at": null,                  // 最终确认时间（null=未闭环）
  "updated_at": 1786001200000
}
```

---

## 10. audit_log —— 审计（1 条：task 状态从 in-progress → done）

```jsonc
{
  "id": 5201,
  "table_name": "tasks",                // 被变更的表
  "row_id": "t-20260815-001",           // 业务主键值
  "action": "UPDATE",                   // INSERT | UPDATE | DELETE
  "field": "status",                    // 变更字段（INSERT/DELETE 为 null）
  "old_value": "in-progress",           // 旧值（null=无旧值，如 INSERT）
  "new_value": "done",                  // 新值（null=无新值，如 DELETE）
  "by_who": "rd",                       // 操作者（role_id / 'user' / 'system'）
  "created_at": 1786000900000,          // 审计事件时间
  "updated_at": 1786000900000           // 恒=created_at（审计行不可变）
}
```

---

## 11. meta —— 元信息（1 条：schema 版本）

```jsonc
{
  "id": 1,
  "meta_key": "schema_version",         // 元信息键
  "meta_value": "28",                   // 元信息值（迁移版本号，每次结构演进 +1；v0.28：products 单列 + mode/align + reviewers 升级）
  "created_at": 1786000000000,
  "updated_at": 1786000000000
}
```

---

## 串起来看（表间关系）

```
req-019 (flows)
  └─ flow_templates = business (flow_templates.flow_templates_id)
        └─ node_order=3 "RD 技术设计" (flow_template_nodes)
              └─ capability_id=backend-design (capabilities) → role rd (roles)
  └─ n-03 RD 技术设计 (flow_nodes, 已通过)
        └─ tasks: t-…-001 (rd produce, done)
        └─ artifacts: art-101 《RD 技术设计》
  └─ n-04 RD 技术评审 (flow_nodes, 门禁, 待评审)  ← flows.current_node_id
        └─ tasks: t-…-002 (arch review, awaiting-reply)
        └─ issues: cm-1002 (arch 意见 → replies 2 轮 → pending-confirm)
        └─ audit_log: 记录了 task 状态变迁
```