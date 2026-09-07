> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：文中 trigger/hook 词汇与「Leader 启动器」编排描述（v3 下 wbflow 程序直启 + Leader 可选）。

> **追加废弃指针（2026-09-08）**：本文件之外，最新模型已迭代为 v3.1+（两层属性状态机 node.state 4 相位 + **per-agent 评审轮** reviewing/agree/unagree + collect 内置聚合 + review_results 单字段 + 评审能力对象化 {role}.{object}-review + 技术设计无内嵌 CR + task 无 QA 节点 + scope 渲染集合表达式 rd|full/fe|full/full/rd|fe + QA 验收仅 full 渲染）——任何与本文件一致的旧记忆引用，一律以 `docs-tmp/capability-model-final.md` v3.1 现行版为准。
**仍有效**：D1-D6 机制结论——subagent 程序化直启实测、persona 由节点配置传入、followup 唤醒、D6 节点会话入口（并入 sessions 表与按需会话通道）。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。


# plugin → 主 agent → subagent 执行链路推演（可行性验证 + 设计定稿）

> **状态**：链路推演定稿（2026-08-27）。主推演「执行闭环确认」工单的输入。
> **问题**：程序（wbflow 插件）推进状态机 → 通知主 agent → 主 agent 启动 subagent → subagent 调 wb_my_task 获上下文执行——这条链路在 DSH 环境能否跑通？
> **结论先行**：**能跑通**。DSH 已提供全部必要机制，且本项目环境已有生产先例（dsh-client-auto-continue 正以「宿主插件 → agent.followup 唤醒」跑在用户 profile 中）。D1–D5 已逐项定稿（§3），另确认前置能力：plugin 创建/恢复 session + 绑定 flow。

---

## 1. 链路拆解（7 环节）

```
① plugin 推进状态机（定时/事件）
   → 算出：下节点 owner 是谁、该哪个动作
② program → 主 agent 通知（唤醒）
③ 主 agent（Leader）启动 subagent
④ subagent → wb_my_task({reqId, role, nodeId})（who 由 agent 告知）
⑤ 程序：结合 who + when（节点状态决策树）→ currentTrigger + 选中动作 → 返回 上下文 + how
⑥ subagent 零判断执行 how
⑦ 返回 → 程序/主 agent 再调度（闭环回去 ①）
```

---

## 2. 逐环节证据（DSH 机制）

### ① plugin 推进状态机（定时任务）
- **机制**：`@deepseek-ai/dsh-schedule`（packages/schedule，`runtime.ts`：disposable live timer projection for one exact root agent；`every`/`scheduledAt` 两种条目，Node timers）——插件内可直接挂周期任务/一次性任务
- **证据强度**：✅ 有现成包；caller 是 agent 上下文。宿主侧定时（无 agent 时）需确认（见 §3 D1）

### ② program → 主 agent 通知（链路的关键一环）
- **机制**：`agent.followup(message: UserMessage): void`（packages/core/agent/src/runtime-types.ts）——会话运行时接口：**idle 的 driver 开新 turn；running 的 driver 在 step 边界消费**。即插件可向**活着的 agent 会话**投递消息并唤醒该 turn
- **生产先例**：`dsh-client-auto-continue`（用户 profile 已装）——host 侧 `ctx.inject(['settings','agents','webServer'])`，经 `/api/auto-continue-action` POST 接收通知 → `runner.handleNoticeAction` → `agent.followup` 唤醒/`agent.cancel` 取消——**与本链路 ② 完全同构**（程序经 HTTP/定时 → followup 唤醒）
- **证据强度**：✅✅ 已有生产实现跑在用户环境（restart 日志可见 "auto-continue: agent for session ... is live, following up"）

### ③ 主 agent 启动 subagent
- **机制**：既有 subagent 工具（tool-subagent，preset standard 已装：provider=spawn/fork，backgroundMode=continuable）；execution-loop §4 的派发协议（prompt 自包含：action_id + query + 参考产物）已有设计
- **证据强度**：✅ 本会话即在使用（本工单的 research 子代理们）

### ④ subagent 调 wb_my_task（who 上报）
- **机制**：wb_my_task 为 wbflow 插件注册的领域工具；subagent 经 tools 拿工具，`{reqId, role, nodeId}` 入参（role=who 由 subagent 自报）
- **现状缺口**：opMyTask 旧结构（无 currentTrigger/activeActions/actions[]）→ 需升级（execution-loop §12 缺口 #3，已列必办）
- **证据强度**：⚠️ 工具存在、机制通，返回结构待升级

### ⑤ 程序算 how（返回正确动作）
- **机制**：无需新 DSH 机制——wbflow host 侧执行决策树（execution-loop §11：rolledBack>0 → reject_rolled_back；assignee=我 open 评论 → reviewer_feedback；gate 待评审 → node_review_pending；产物已登记未提评 → dev_complete；state∈{未开始,进行中} → node_enter；全过 → decide），从 `capabilities.actions[]` 按 action_id 取 how
- **决策树判据**：全部程序可读字段（flow_nodes.state / issues.query / artifacts.productType / tasks）——与 when_to_use 文本无关（★ 支持"when_to_use=人读注释"的定位）
- **证据强度**：✅ 纯程序逻辑，无平台依赖

### ⑥ subagent 零判断执行
- **机制**：既有 wb_* 工具（产物登记/评论）+ 通用工具（fs/git/代码）；execution-loop §5 已设计
- **证据强度**：✅

### ⑦ 返回与再调度
- **机制**：subagent 返回 → 主 agent 重算/wbflow 事件进入 → 回到 ①——execution-loop §6 已设计（完成/新反馈/阻塞/异常中断四类）
- **缺口**：wb_assign_task / wb_task_update 必补（execution-loop §12 缺口 #1/#2）；subagent 返回契约结构化（§8 待办 #3）
- **证据强度**：⚠️ 设计已有，工具待补

---

## 3. 设计点定稿（D1–D5 + 前置能力，2026-08-27 用户逐项裁定）

### 前置能力：plugin 创建/恢复 session + 绑定 flow（用户指出，链路前提）

- **机制证据**：`ctx.agents.create({ sessionId, agentOptions, meta: { cwd, agentPreset }, setup })` 与 `ctx.agents.resume({ resumeSessionId, agentOptions, setup })`（api-proxy.ts L1590-1640 会话创建/恢复、L2300-2340 fork 先例）——plugin host 侧可**程序创建/恢复 session**，preset 以存储为准
- **绑定 flow**：plugin 数据层维护 `flowId ↔ sessionId` 映射；followup 目标解析 = `flow → sessions 映射 → ctx.agents.get(sessionId)`
- **定稿**：wbflow host 增「session 工厂 + flow 绑定注册表」能力；session 创建时写 `meta.agentPreset`（常驻 Leader = team preset）
- **注**：D4 的 preset 角色补齐（subagent_rd 更新 + 其余角色新建）**暂不动**，待推演完全定稿后实现阶段处理（用户 2026-08-27 裁定）

### 子会话的 UI 入口：挂载到 flow 节点（用户 2026-09-02 裁定，D6）

- **背景**：subagent 会话有独立稳定 sessionId、持久化跨重启保留（`continuation.ts`：stable child id + cold resume + sessionPersistence）；但 DSH Web UI 的子会话入口**只在 parent 会话头部的 lineage 树**（ui-subagent：SubagentHeaderLineage + openChild），不进顶层会话列表；一轮对话结束后用户看不到历史子会话入口。
- **裁定（不改 DSH UI）**：**把 subagent 会话挂载到 flow 节点上**——插件页面支持点击节点 → 选择查看 session 信息 → 查看该节点**历史或进行中**的 subagent 对话。UI 呈现由插件层（wbflow 自定义页面）承担，DSH 原 UI 不动。
- **落点**：①插件 UI：流程树/节点详情页增加「会话」入口（列出该节点关联 session：历史/进行中，点开查看对话或续聊）；②数据层：flow 节点 ↔ session 映射表随 TASK/派发记录写（节点启动 subagent 时登记 sessionId → nodeId）；③「父 live 门槛」语义保留（父不 live 时查看历史只读/不可输入），但查看历史不依赖 UI catalog。
- **收益**：流程视角而不是会话视角——用户关心"这个节点发生过什么对话"，而非"哪个会话树下的子代理"；与工作台"节点→产物→评论"的既有查看模式一致。

### D1 定时任务载体 → **schedule 包（agent 内周期自检）+ plugin 关闭 + 兜底咬合**

- **查实**：`@deepseek-ai/dsh-schedule` 是 **agent-scoped**（inject agens/sessions/tools/sessionPersistence；绑定一个 root agent；deliveryMode=session-local）——**不是宿主独立 cron**；agent 调 schedule 工具注册 one-shot/every 条目（Node timer），到期 `requestDrive()` 恢复该 agent turn
- **定稿**：① 常规周期=主 agent 常驻会话内 schedule（every 周期自检：扫 TASK 陈旧/节点异常）；② **plugin 判定任务结束时关闭 schedule**（清定时器防空转）；③ **schedule 调度 plugin 时做兜底**（到期驱动的 turn 内主 agent 自检，即"被唤醒先扫 TASK"的懒扫描，与 schedule 咬合）；④ 宿主侧异常兜底（无活会话）= auto-continue 式 HTTP 端点（外部通道，非常驻）

### D2 通知形态 → **直连 followup，消息进 session 历史（可见可审计）**

- **查实**：`followup(message)` 投递 `createUserMessage`（role: 'user'，llm/src/message.ts）——**进入该 session 对话历史**，UI 可见
- **定稿**：直连——wbflow host 经 flow→session 映射 → `ctx.agents.get(sessionId)` → `followup([workbench-notify] …)`；消息以 user 角色入历史（可审计、可回溯；措辞规范化前缀 `[workbench-notify]`）；idle 开 turn、running 在 step 边界消费

### D3 wb_my_task 返回粒度 → **全部上下文 + 仅当前动作完整 how（最小返回集）**

- **定稿**（用户 2026-08-27 最终裁定：只返回具体 action，减少不必要信息；纠正此前"返回全部 actions[]"的说法——wb_my_task 信息之前已敲定，本就含要执行 action 的 how 文本）：返回 = 当前上下文（currentNode/relatedNodes/threads）+ `currentTrigger` + `activeActions` + **仅当前激活动作完整 how/query/reference/focus**——**不返回其他动作**（连骨架/action_id 列表也隐去，返回最小集）；披露与防 rush 均为程序返回层行为（⇢ how v2 第 4/10 条据此定稿）

### D4 who 信任 → **three-way 校验；preset 角色补齐延后到实现阶段**

- **查实**：team preset 已有 `tool-subagent-rd`（toolName=subagent_rd，persona=RD 职责；含实现-设计对照表等与本推演一致的纪律，但引用旧文档名 m7/m6/「需求评审」、无 actions 定位、无 respond_to_qa）——需更新；其余角色（Arch/PM/FE/QA/Investigator/CR）未单独定义
- **定稿**：身份校验三处一致——`capabilities.role_id`（动作权威角色）↔ `TASK.assignee_role`（派发事实）↔ `wb_my_task` 入参 role（自报），不一致拒绝返回 how；**preset 文件暂不动**（实现阶段按定稿补齐/更新）

### D5 主 agent 常驻性 → **常驻 Leader**

- **定稿**：常驻 Leader 会话（team preset 载体），host 持 flow→session 注册表；生命周期=User 显式启动或插件首次派发自动创建；跨进程重启由 resume（auto-continue 恢复机制）+ 宿主端点兜底；防膨胀=Leader 仅调度人格（wb_my_task/派发/收返回），执行细节在 subagent 会话

---

## 4. 结论

- **链路技术上跑得通**：7 环节中 4 个有现成机制 + 1 个有生产先例（② 唤醒，auto-continue 同构）、2 个是纯程序逻辑；剩余缺口均为**工具/返回结构升级**（wb_my_task 升级、wb_assign_task/wb_task_update 补建）与**设计决策**（D1-D5），无平台级障碍。
- **关键洞察**：② 唤醒环节不是新发明——auto-continue 插件已经在用户环境证明「宿主→agent.followup→turn 恢复」；wbflow 只需同构实现（或直接借鉴其端点模式）。
- **本稿是主推演「执行闭环确认」工单的输入**；与此前技能评估的裁决（respond_to_qa 补建、dev 合并）互相咬合——即 how v2 的第 10/13 条依赖本稿 D3/D4/D5 的定稿；第 7 条（完成标准）按 2026-09-02 修正：**how 只写给 subagent（自检口径），程序状态推进按阶段各有规则、不配对的 done 信号**（详见工单 05 定夺日记）。