# MVP 数据层 + 工具清单定稿（data-tools-final）

> **状态**：2026-09-05 工单 07 产出定稿。模型权威=`capability-model-final.md` v3（两层属性状态机）；节点配置=business-flow-derivation.md v3；动作挂载=capability-actions-final.md。
> **本文回答工单 07 三问**：①sqlite schema 调整清单（v0.29 增量，对照现 11 表）；②wb_* 工具清单终稿（22→**23**）；③seed.json 演进路径（直接重写）。
> **全量结论输入**：工单 03 v3 模型（相位 4 值/reviewState 7 值/双轨评审/推进契约）+ 缺漏审查 8 项裁决（#2 decide 退役、#5 扫描器参数、#6 幂等、#8 gate 语义）。

---

## 1. schema 调整清单（schema.sql v0.28 → v0.29）

### 1.1 表级更名

| 现名 | 新名 | 说明 |
|---|---|---|
| `flow_templates` | **`flow_defs`** | 流程定义（business/tech/task/plan）；逻辑引用列 `flow_templates_id` → `flow_defs_id` |
| `flow_template_nodes` | **`flow_def_nodes`** | 定义节点骨架；与产物模板（docs/templates）不再同名 |

> 改名时机=实现前（工单 07），实现期再改需全量迁移。

### 1.2 逐表 delta

**capabilities（能力注册）**：
- 移除：`how`（内容转 skill——skill 挂载制）
- 新增：`inputs` TEXT NOT NULL DEFAULT '[]'（JSON: [{productType, use_threads}]，use_threads 默认 true）
- 新增：`actions` TEXT NOT NULL DEFAULT '[]'（JSON: [{action_id, name, on, skill?, inline?, outputProductTypes}]）
- `products` 保留（模板外置真相源）
- 能力集合：**17**（19 − case-design-cap − final-approve；decide-cap 保留收缩为 create_reqs 单动作）

**roles**：不变。

**flow_defs（原 flow_templates）**：`scope_options` 保留；其余不变。

**flow_def_nodes（原 flow_template_nodes）**：
- 新增：`template_node_id` TEXT NOT NULL UNIQUE（业务可读稳定 id，如 `bus-03`）
- `mode` 列 → **`node_type`** TEXT NOT NULL（`produce | review | align | gate`——四类节点生命周期）
- `gate` 保留（0/1 准出判定，与 node_type 正交：开发节点 produce+gate=1 内嵌 CR）
- 新增：**`user_gate`** INTEGER NOT NULL DEFAULT 0（User 终审拍板位——agree 收口后按需唤醒）
- 新增：**`node_actions`** TEXT NOT NULL DEFAULT '[]'（JSON 两类条目：
  `{kind:'capability', capability_id, actions:[action_id…]}` = 节点动作激活清单；
  `{kind:'node', action_id, on, executor:'program'|'subagent', executor_role?, skill?|inline?}` = node action（开场白/to-brd））
- `scope` 枚举统一（2026-09-07）：''=中性（任何 scope 渲染）｜'rd'/'fe'=仅含该侧需求渲染（术语统一 be→rd，与角色名一致）｜'full'=仅 full 需求渲染（QA 验收=联调职责）；实例化按 requirement.scope 过滤。scope_options：business/tech={full, rd, fe}、task={rd, fe}、plan=无 scope
- `reviewers` 保留 [{role, capability_id, scope?}]（审批 skills map 派生源）；**运行时实例（flow_nodes.reviewers）条目扩展表态字段**：`verdict: {round, result: 'pass'|'block', at, note?}`——collect barrier 判据=全部条目 verdict.round==rolled_back_count 且非空；历史轮表态由 audit_log 回放（wb_review 全落审计），不加新表；表态与意见分离（意见=issues 线程，表态=结构化 verdict）
- 删除：用例设计/用例设计评审两行（qa 职责重定义）；修正旧残留（plan 拆分拓扑 gate 值、tech/task 需求对齐 gate 统一为 1——有 User 判定即 gate=1）
- `expected_outputs` 保留

**flows（流程实例）**：
- 新增：**`dependencies`** TEXT NOT NULL DEFAULT '[]'（JSON: [requirement_id…]——同版本内依赖的前置需求 id 列表；plan D7 create_reqs 从 architecture「需求拆解」节写入）
- 其余不变（requirement_id 引用；project 维度=工作区隔离，不加列）。
- **依赖树（plugin 层内置，2026-09-05）**：plugin 依据 flows.dependencies 构建版本内需求依赖树——需求实例按依赖自动注册在各自父节点下；**分层由依赖图派生（拓扑分层），不对用户显示层级**（内部维护）；树视图支持**分析师动态加载**（按需拉取，非常驻）。调度器（长期）：就绪池=树中依赖全部 approved 的节点，按高优 priority 拉取（适用 business/tech/task，plan 除外）。

**flow_nodes（实例节点）**：
- 新增：**`def_node_id`** TEXT NOT NULL（引用 flow_def_nodes.template_node_id——工单 02 裁决的「定义节点关联」，随更名定名）
- 新增：**`review_state`** TEXT NOT NULL DEFAULT '-'（'-' / collecting / clarifying / confirming / agreed / approved / rejected——审批子状态，plugin 轮转）
- 新增：**`reopen_source`** TEXT NOT NULL DEFAULT ''（'' / upstream-reject / downstream-reject / rework-return——plugin 回退时写入，重入配置与上下文注入依据）
- `state` 白名单收窄：''（未进入编排，无占位）/ entry / exec / review / respond（可配动作相位 4 值）
- `rolled_back_count` 保留（0=首次，≥1=重入——entry/exec 多配置选择）
- `last_decision` 保留，JSON 扩展承载 reject 结论协议（branches/confirmed_issues/context）
- `feature_branch` / `entered_at` / `pending_scope` 保留

**tasks（任务总线）**：
- 新增：**`action_id`** TEXT NOT NULL（派发定位到能力×动作）
- `task_type` 扩展：produce / dev / review / respond / accept / create_reqs（按相位动作族）
- `status` 语义修正：todo / in-progress / awaiting-reply / **completed（仅 plugin 判据校验通过后置，agent 不可直置）** / rejected / closed
- 派发/断点字段保留（dispatch_at/last_ack/last_progress/resume_hint）
- 幂等：task 派发按 `reqId+nodeId+action_id+assignee` 幂等

**artifacts（产物）**：
- 新增：**`product_type`** TEXT NOT NULL（受控枚举校验：brd / product_doc / rd_design / fe_design / architecture / research / plan / alignment / dev_notes / code_mr / test_result / defect / discussion；**case_design 移除**）
- 其余不变；wb_add_artifact 携带 product_type 与枚举校验

**issues（评论/问题）**：
- 新增：**`escalate`** INTEGER NOT NULL DEFAULT 0（冲突响应标记——wb_resolve/wb_reopen 可选参数写入；escalate=1 → plugin 拉起会话通道裁决 session）
- `state` 枚举不变（open/pending-confirm/resolved/overridden；已处理=pending-confirm）
- 裁决结论回填：追加到 `replies`（JSON 不可变追加）

**sessions（新表 #12）**：

```sql
CREATE TABLE sessions (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id     TEXT NOT NULL UNIQUE,   -- 会话业务 id
  requirement_id TEXT NOT NULL DEFAULT '',
  node_id        TEXT NOT NULL DEFAULT '',
  role_id        TEXT NOT NULL,          -- 绑定角色（subagent persona 按节点配置传入）
  dsh_ref        TEXT NOT NULL,          -- DSH 会话句柄（ctx.agents/subagents 引用）
  status         TEXT NOT NULL DEFAULT 'active',  -- active | ended | disposed
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
```
用途：flow↔session 绑定注册表（D6 节点会话入口 + 按需会话通道共用）。**绑定角色支持按 scope 动态**（plugin 框架：full=qa / fe-only=fe / rd-only=rd——用户验收报障对接方随需求侧变化，2026-09-07）。

**audit_log / meta**：结构不变；`meta.schema_version` → v0.29。

**索引新增**：idx_flow_def_nodes_tpl(flow_defs_id)、idx_flow_nodes_def(def_node_id)、idx_tasks_action(action_id)、idx_artifacts_ptype(product_type)。

### 1.3 非数据层（边界外但有契约）

- **project registry**：plugin 全局配置 `~/.dsh/wbflow/projects.json`（[{project_id, name, workspace, status}]）——per-project workbench 隔离（11 表零改动）；plugin 接口：注册/列出/切换/停用⚠️。
- **需求/版本本体**：JSON 源承载（.workbench/requirements.json + versions.json），sqlite 投影——现状不动；两层同步责任在 plugin 写路径。
- **需求依赖与并发调度（长期预留，2026-09-05）**：依赖字段落 **flow 实例**（flows.dependencies，见 1.2）+ requirement JSON 同步 + `priority`；依赖树由 plugin 内部维护（自动分层派生、不对用户显示层级、树视图分析师动态加载）；并发度=正在工作的需求数（卡 user_gate 的需求让出槽位）；调度器（就绪池高优拉取，适用 business/tech/task、plan 除外）为长期独立 effort，MVP 人工卡点（节点 0 user_gate=1）。
- **幂等声明（全局）**：`event:'complete'` 幂等（判据为数据状态，重复上报无害）；回退动作按 `requirement_id+node_id+round` 幂等；task 派发按 `reqId+nodeId+action_id+assignee` 幂等。

### 1.4 扫描器参数表（宿主侧兜底，工单 07 实装）

| 场景 | 判定依据（dsh 能力） | 动作 |
|---|---|---|
| 绑定 subagent 执行中 | `agent/status`=running（Agent 事件，ch02） | 等待 |
| 已结束 + **5min** 未表态 | status=idle/ended + reviewState 未推进 | followup 催办：「继续当前工作，遇卡点直接上报 user」；需上报 → plugin 通知 user |
| **10min** 仍未推进 | 扫描复核 | 通知 user 介入 |

参数化配置（后续可用性提高可下调 user 介入率）；启动失败检测沿用实测结论（agent/request-error 监听 + 重试）。

---

## 2. wb_* 工具契约（22 → 23）

**变更总览**：+wb_assign_task、+wb_task_update；−wb_decide（裁决=plugin 内置轮转 + User 拍板 UI→service，agent 无调用场景；pass/reject/targetBranch 语义保留在 service 层）；升级 wb_my_task；参数扩展 wb_add_artifact / wb_resolve / wb_reopen。

### 2.1 清单（23）

| 组 | 工具 | 变更 |
|---|---|---|
| 读（5） | wb_list_requirements / wb_list_versions / wb_get_requirement / wb_get_artifact_content / **wb_my_task** | my_task 升级（见 2.2） |
| 需求（3） | wb_create_requirement / wb_update_requirement / wb_delete_requirement⚠️ | 不变（create_reqs 由 leader 直执行逐条调用） |
| 版本（4） | wb_create_version / wb_update_version / wb_advance_version / wb_deprecate_version⚠️ | 不变 |
| 评论/评审（4） | wb_comment / wb_reply / **wb_resolve(+escalate?)** / wb_override⚠️ | escalate 参数新增；**wb_decide 移出 agent 清单**（service 层保留语义） |
| 评审表态（3） | **wb_review / wb_confirm_review / wb_reopen(+escalate?)** | wb_review 契约见 2.2；reopen 增 escalate |
| 产物（1） | **wb_add_artifact(+product_type)** | 枚举校验 |
| 任务（2，新增） | **wb_assign_task / wb_task_update** | 见 2.2 |
| 收件箱（1） | wb_list_inbox | tasks 投影，不变 |

### 2.2 关键工具契约

**wb_my_task（升级——任务确定性来源）**
- 入参：`{reqId, role, nodeId?}`（nodeId 缺省=当前节点）
- 返回：
```jsonc
{
  requirement: {…摘要}, node: {node_id, node_type, state, review_state},
  currentPhase,                     // 当前相位（=state）
  activeActions: [                  // 按节点激活清单 + 当前相位过滤
    { action_id, name, on, skill|inline, 上下文: {产物引用, 意见清单, 打回清单} }
  ],
  relatedNodes: […],                // 最小返回集（sessionsByNode 已移除）
  reviewers: […],                   // 审批 skills map 派生
  上下文注入: {def_node_id, reqId, inputs 可用产物引用}
}
```
- 身份 three-way 校验（reqId+nodeId+role 与派发记录一致，工单 04）；roleObjective/behaviors 由 persona+skill 承载（how 已退役）。

**wb_assign_task（新增——派发）**
- 入参：`{reqId, nodeId, action_id, assignee_role, goal?, context?}`
- 语义：plugin 创建 task（tasks 表）+ 唤醒/启动 assignee subagent；幂等键 `reqId+nodeId+action_id+assignee`。

**wb_task_update（新增——心跳+完成上报）**
- 入参：`{task_id, event: 'ack'|'progress'|'complete', progress?, 结论引用?}`
- 语义：ack/progress=过程信号（断点续接）；**complete=相位收口信号**（长任务/交互等待动作专属，短动作数据写入即完成不上报）→ plugin 校验该相位判据 → 通过置 completed 并轮转；不通过 `{ok:false, blocked:true, missing:[…]}`。
- 幂等：重复 complete 无害（判据幂等）。

**wb_review（表态落库——collect barrier 数据源）**
- 入参：`{reqId, nodeId, verdict: 'pass'|'block'}`
- 语义：写当前轮 `flow_nodes.reviewers[].verdict`（`{round, result, at, note?}`）。
- 幂等：同轮同角色覆写取最后。
- barrier 判据：全部评审位 `verdict.round == rolled_back_count` 且非空 → reviewState collecting 迁移（有 open 意见 → clarifying；无 → agreed）。
- **表态与意见分离**：意见=issues 线程（wb_comment）；表态=reviewers[].verdict（结构化判定）；历史轮表态由 audit_log 回放（不加新表）。

**wb_resolve / wb_reopen（参数扩展）**
- +`escalate?: 0|1`——分歧双方任一方标记 → plugin 拉起会话通道裁决 session（响应者=reviewer，开场白=分歧点+双方立场）→ 用户裁决 → 结论追加到冲突 issue → reviewer 重新表态落库。

**审计**：全部工具走 audit_log（既有约定不变）。

---

## 3. seed.json 演进路径（直接重写，不做增量迁移）

理由：v3 为结构性变更（表更名+列语义替换+拓扑修正），demo 数据量小，增量迁移脚本成本高于重写。

1. **capabilities**：17 能力按 `{inputs, actions}` 重写（挂载映射见 capability-actions-final §1/§2；decide-cap=create_reqs 单动作；final-approve 移除）。
2. **flow_defs / flow_def_nodes**：4 流程按 business v3 配置 + flow-derivation-tech-task-plan（tech 11 / task 8 / plan 9）逐节点全配（node_type/gate/user_gate/reviewers/node_actions/def_node_id）；修正旧残留：用例两节点删除、plan 拆分拓扑 gate 修正、tech/task 对齐 gate=1 统一。
3. **flow_nodes / tasks / artifacts / issues**：按新列语义重生成 demo 数据。
4. **gen_flow_rows.py**：按 v3 配置重写（保持脚本化恢复产物原则；旧版归档）。
5. **seed.sql / workbench.db**：v0.29 建库 + 重新导入。

---

## 4. 工单 07 关口核对

- [x] schema 调整清单（对照 11 表逐表 delta + sessions 新表 + project registry 边界）
- [x] wb_* 工具清单终稿（23 个，含 my_task 升级契约与幂等声明）
- [x] seed.json 演进路径（直接重写 + 生成脚本改造）
- [x] 扫描器参数表（#5：5min 催办 / 10min user 介入）
- [ ] 关闭动作：地图更新 + semantica 溯源 → 解锁工单 08（交互层）
