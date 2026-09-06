---
type: grilling
blocked_by: []
claimed_by: huzilin+agent session
claimed_at: 2026-09-02
assets: [../../docs-tmp/plugin-agent-subagent-chain.md]
---

# 执行闭环待确认项

## Question

docs-tmp/execution-loop.md §9-§12 的待确认项逐个敲定：

- cd duty 定时兜底任务的载体（宿主侧定时机制选型）；
- `wb_my_task` 的 currentTrigger 计算规则（据 flow_nodes.state/issues.query/artifacts.productType/tasks 自上而下第一命中决策树）；
- subagent 返回契约（完成 / 新反馈 / 阻塞 / 异常中断四类返回的再调度动作）；
- agent 再调度重入点；以及必补工具（wb_assign_task、wb_task_update）与升级（wb_my_task 返回 currentTrigger+activeActions+actions[]）的确认。

依据：execution-loop.md（触发 → 派发 → 执行 → 返回 → 再调度）、capability-review-v2.md §0.3（trigger 全局统一程序契约）。

## 输入素材（2026-08-27）

**`docs-tmp/plugin-agent-subagent-chain.md`（链路推演定稿）**：plugin → 主 agent → subagent 全链路 7 环节可行性验证——结论能跑通。关键证据：`agent.followup()`（idle 开 turn / running 消费于 step 边界）即「plugin 唤醒主 agent」的机制，且用户环境已有生产先例（dsh-client-auto-continue：HTTP 端点接收通知 → followup 唤醒）。

**D1–D5 已逐项定稿（2026-08-27 用户裁定）**：
- **前置能力**：plugin 创建/恢复 session（ctx.agents.create/resume，api-proxy 先例）+ flow↔session 绑定注册表
- **D1**：schedule 包（agent 内周期自检）+ plugin 任务结束关闭 schedule + schedule→plugin 兜底咬合；宿主异常兜底=端点
- **D2**：直连 followup，消息以 user 角色进 session 历史（可见可审计，`[workbench-notify]` 前缀）
- **D3**：wb_my_task 返回全部上下文 + **仅当前动作完整 how（最小返回集，不返回其他动作）**——用户 2026-08-27 最终裁定（纠正"返回全部 actions[]"旧说法；wb_my_task 信息敲定即含执行 action 的 how 文本）
- **D4**：three-way 身份校验（capabilities.role_id ↔ TASK.assignee_role ↔ my_task.role）；team preset 角色补齐**延后到实现阶段**（现状已有 subagent_rd 需更新、其余角色未定义）
- **D5**：常驻 Leader 会话（team preset 载体 + flow→session 注册表 + resume 兜底）

另：execution-loop §12 缺口已确认——wb_assign_task、wb_task_update 必补；wb_my_task 升级（currentTrigger+activeActions+actions[]）。

## Answer（2026-09-02 关闭，用户逐点确认）

D1–D5 与前置能力已在输入素材定稿（见上）；本工单三个待确认点终稿如下：

1. **currentTrigger 计算规则** ✅：execution-loop §11 决策树为权威——自上而下第一命中，全部程序可读字段（`flow_nodes.rolled_back_count>0 → reject_rolled_back`；`issues.query 命中 assignee=我&state=open → reviewer_feedback(open)`；`gate 节点待评审 → node_review_pending`；`产物已登记未提评 → dev_complete`；`state∈{未开始,进行中} → node_enter`；`全评审通过 → decide`；`TASK stale/unacked → task_stale`）。**when_to_use 已废弃（how-spec-v2 第 2 条）**——决策树只读状态字段不读动作文本。
   **trigger 枚举 = 决策树分支集合的受控枚举**（吸收 how-spec-v2 第 3 条语义）：动作声明 trigger 从封闭枚举取，决策树产出同枚举值，天然一一对应；新增 trigger = 同时加决策树分支 + 枚举项（一处受控增改），无需额外"对应校验"机制。
2. **subagent 返回契约 —— 取消（数据层即返回）** ✅：subagent **不向 Leader 返回结构化契约**——执行成果全部落在数据层（产物登记 artifacts / 评论处理 issues / 表态 reviewers），程序读数据层即可推进状态机；再调度由程序算好、通知 Leader 执行。**唯一显式"返回" = 心跳**：`wb_task_update(last_ack/last_progress)`（防中断判定用，非汇报）；阻塞/异常写进数据层（评论/issue 表达）+ TASK 心跳停滞 → 程序兜底扫描（schedule 周期 / task_stale）接管。execution-loop §6 的"四类返回再调度表"改写为"**数据层变更 → 程序推进兜底**"模型。
3. **必补/升级工具** ✅（收敛）：**wb_assign_task**（Leader 派发时写 TASK：goal/context/action_id/assignee_role——派发事实记录 + D4 身份校验源）；**wb_task_update**（**仅心跳 ack/progress，取消 done**——完成由程序从数据层推导，不靠显式 done 汇报）；**wb_my_task 升级**（返回 currentTrigger + activeActions + action[最小集]，见 how-spec-v2 / my-task-response-model）。cd duty 载体 = D1 定稿（schedule 包 + 端点兜底）。

**联动**：execution-loop.md 标记 D1–D5/数据层即返回/工具缺口已定稿，待 MVP 定稿收尾时统一整理；team preset 角色补齐随实现阶段。

## 结论时效（2026-09-05 v3 定稿后标注）

- **已取代**：currentTrigger 计算规则（trigger 决策树词汇）——被 v3 相位判据表（capability-model-final §0.6/§1.2 reviewState 轮转）取代；「常驻 Leader」——被 subagent 程序化直启弱化（Leader 可选，wbflow 程序承担启动）。
- **仍有效**：schedule 包+端点兜底、最小返回集 relatedNodes、three-way 身份校验、wb_assign_task/wb_task_update（均并入 data-tools-final；complete 完成上报为其精化）。
