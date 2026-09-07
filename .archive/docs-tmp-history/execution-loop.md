> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：currentTrigger 决策树（trigger 词汇）、subagent 返回契约（完成/新反馈/阻塞/异常四类）——被 v3 相位判据表（§0.6）+ 完成上报 event:'complete' + reviewState 轮转取代。

> **追加废弃指针（2026-09-08）**：本文件之外，最新模型已迭代为 v3.1+（两层属性状态机 node.state 4 相位 + **per-agent 评审轮** reviewing/agree/unagree + collect 内置聚合 + review_results 单字段 + 评审能力对象化 {role}.{object}-review + 技术设计无内嵌 CR + task 无 QA 节点 + scope 渲染集合表达式 rd|full/fe|full/full/rd|fe + QA 验收仅 full 渲染）——任何与本文件一致的旧记忆引用，一律以 `docs-tmp/capability-model-final.md` v3.1 现行版为准。
**仍有效**：schedule 包+端点兜底载体、最小返回集、three-way 身份校验、wb_assign_task/wb_task_update 心跳（均已并入 data-tools-final）。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。


# 执行闭环链路：触发 → 派发 → 执行 → 返回 → 再调度

> **状态**：设计文档（临时稿）。把「agent 发起任务 → subagent 执行并返回 → agent 再调度 → subagent 再执行」的完整闭环固化，供后续推演使用。
> **触发源头**：① User 主动发起；② 兜底定时任务（扫描发现节点因异常中断 → 重新调起 agent）。
> **关联**：`artifact-visibility-rework.md`（产物/可见性 v2）、`capability-review-v2.md`（能力×动作 + trigger 契约）、`my-task-response-model.md`（wb_my_task 返回）、`arch_design.md` §9.5（编排协议）、TASK 派发/断点字段。

---

## 1. 闭环全景（一图）

```
┌──────────────┐  ┌────────────────┐
│ 触发源        │  │ 兜底定时任务     │
│ ① User 发起   │  │ ② 扫描异常中断   │
│ ② 定时扫描    │  │   节点→重新调起  │
└──────┬───────┘  └───────┬────────┘
       │ 触发             │ 触发（resume）
       ▼                  ▼
┌─────────────────────────────────────────────┐
│ AGENT（Leader / 主 agent）                   │
│ 1. wb_my_task → 得 currentNode/relatedNodes/│
│    currentTrigger/activeActions             │
│ 2. 判定 when（程序已算好 trigger）            │
│ 3. 选动作 + 组确定性上下文（action_id +      │
│    query/reference + goal）                 │
│ 4. 写 TASK(dispatch_at/goal/context)        │
│ 5. 派发 subagent（prompt 自包含）            │
└──────┬──────────────────────────────────────┘
       │ 派发 subagent（send_message / 起子任务）
       ▼
┌─────────────────────────────────────────────┐
│ SUBAGENT（角色子 agent，如 RD）              │
│ 1. 读 prompt（action_id + query + 产物位置） │
│ 2. 调 wb_my_task 拿确定性上下文              │
│ 3. 执行动作：read(query过滤评论)→改代码/产   │
│    文档→登记 wb_add_artifact→wb_comment     │
│ 4. 更新 TASK(last_ack/last_progress)        │
│ 5. 返回结果（artifactId/commentId/结论/异常）│
└──────┬──────────────────────────────────────┘
       │ 返回（结果 / 阻塞 / 异常中断）
       ▼
┌─────────────────────────────────────────────┐
│ AGENT（再调度）                              │
│ - 结果 OK → 检查下一节点 / 下一个 action     │
│ - 有反馈（reviewer 新评论）→ 重算 trigger    │
│   → 派发下一个 action                       │
│ - 异常中断 → 标记 TASK（进度陈旧）→ 兜底扫描 │
│   或立即重派 resume                         │
└─────────────────────────────────────────────┘
```

## 2. 触发源（谁启动链路）

### 2.1 User 发起
- User 在收件箱/TODO/看板操作（如确认通过、回复评论）或直接指示 → 触发 agent 处理当前节点。

### 2.2 兜底定时任务（异常恢复）
- **触发条件**：扫描到「已派未确认」（`dispatch_at` 有值、`last_ack` NULL）或「进度陈旧」（`in-progress` 但 `last_progress` 久未更新）的 TASK；或节点状态异常（如子 agent 中断导致节点卡在进行中）。
- **动作**：重新调起 agent → agent（Leader）按 TASK 的 `resume_hint`/`last_progress` 续做，不重来。

## 3. when 的判定契约（trigger 程序化）

> `when_to_use` 从自然语言升级为**程序可判定的契约**（capability-review-v2 §0.2 讨论结论）：

| trigger 值 | 程序判定（wb_my_task 计算） |
|---|---|
| `node_enter` | 当前节点 state ∈ {未开始, 进行中}，且无 open 评论 |
| `dev_complete` | 本节点产物已登记（dev.md/MR 存在） |
| `reviewer_feedback(open)` | 存在 query 匹配的 open 评论（如 assignee=rd & reviewerRole=CR & state=open） |
| `node_review_pending` | 节点是 gate 且处于待评审（产物已提交，评审者待表态） |
| `issue_replied` | 评审阶段收到被评审者的回复（澄清或修复，含定位信息）——评审者复核 |
| `reject_rolled_back` | 节点被 reject 打回（rolledBackCount>0） |
| `decide` | 全部评审者通过且无 open 评论（门禁决策） |

wb_my_task 返回时程序直接算好：

```jsonc
{
  "currentTrigger": "reviewer_feedback(open)",
  "activeActions": ["backend-dev.respond_to_cr"]
}
```

- agent 不再"猜" when，程序给定；
- 传导 subagent 时，`currentTrigger + action_id + query` 原样进 prompt，subagent 零判断。

## 4. 派发（agent → subagent）

**agent 组确定性上下文**（写入 prompt + TASK.context）：

```
派发 prompt（自包含，subagent 不需要再推理）：
  角色: RD（backend-dev）
  节点: RD 开发（req-019, n-09）
  动作: backend-dev.respond_to_cr        ← action_id（程序已激活）
  trigger: reviewer_feedback(open)
  目标评论: query={reviewerRole:"Code Reviewer", assignee:"rd", state:"open"}
           → 从 mytask 返回评论产物里过滤，读 text+anchor+thread
  参考产物: dev_notes（实现-设计对照表）、code_mr（branch）
  期望返回: 修订说明 / resolve 的 commentId / 阻塞问题
```

**TASK 落账**（唯一待办总线）：
```
task_id, requirement_id, node_id, role_id, task_type(produce|review|accept|decide),
goal, context(确定性上下文 JSON：action_id/query/reference),
status(todo→in-progress), dispatch_at, root_issue_id
```

**铁律**：Leader 不替角色表态（review 由对应角色 subagent 落账或 by=角色）；子 agent 无 wb_* 工具时按 host.js 格式直写 `.workbench/` JSON，Leader 校验后正式落账。

## 5. 执行（subagent 内部，确定性）

```
1. 读 prompt 的 action_id + query + reference
2. 调 wb_my_task(reqId, role, nodeId)
   → currentNode/relatedNodes（可见产物）+ currentTrigger + activeActions
   （how/reference 中的 {{var}} 模板路径已由 host 端 renderTpl 替换为真实路径，直接可用）
3. 按 query 过滤评论产物（assignee/reviewerRole/state）
   读 text + anchor（锚点定位改动处）+ thread（上下文）
4. 执行动作（改代码/产文档/建测试资产…——按 how 中渲染后的真实路径读写文件）
5. 登记产物 wb_add_artifact(带 productType) / wb_comment(带 reviewerRole)
6. 更新 TASK last_ack / last_progress
7. 返回：{ ok, artifactId?, commentId?, summary, blocked? }
```

## 6. 返回与再调度（闭环关键）

| subagent 返回 | agent 再调度动作 |
|---|---|
| ✅ 完成（产物已登记/评论已发） | 检查门禁条件 → 若满足 wb_review/decide → 推进下一节点 / 派发下一 action |
| 💬 触发新反馈 | reviewer 新 open 评论 → 重算 currentTrigger → 派发 `respond_to_cr` 类动作 |
| ⚠️ 阻塞（分歧/缺信息） | agent 介入：澄清 / 转 User / 派发协作动作 |
| 💥 异常中断 | 标记 TASK 进度陈旧 → 兜底扫描重新调起（go §2.2）或立即重派 resume |

**闭环循环**：agent 每收到一次返回，都重新 `wb_my_task` 算当前 trigger → 若有激活动作 → 再派发 → 直到节点达成门禁通过 / 无激活动作。

## 7. 与既有设计的衔接

| 既有机制 | 在本环路的角色 |
|---|---|
| TASK（唯一待办总线） | 派发记录 + 断点续接（dispatch_at/last_ack/last_progress/resume_hint） |
| wb_my_task | 每次环节的上下文供给：可见产物 + currentTrigger + activeActions |
| capabilities.actions | 动作定义：when(trigger)/how/query/reference/tools —— 在 wb_my_task 展开 |
| 评论产物 query | 传导「看哪些对话」（respond_to_cr 的定位） |
| 收件箱/TODO | User 侧入口（触发源 ①）+ 兜底扫描入口 |
| 协作铁律 ⑦⑧ | feature 分支 / worktree（subagent 读代码的隔离基础） |

## 8. 待确认 / 待办

1. **cd duty 定时任务的载体**：DSH 侧用什么机制跑「兜底扫描」（host 定时任务 / workflow / 外部 cron）——需技术验证；
2. **wb_my_task 的 currentTrigger 计算**：需要把 trigger 判定逻辑落到 opMyTask（读评论状态/节点状态）；
3. **subagent 返回契约**：结构化返回格式（ok/artifactId/commentId/summary/blocked）需在 host/编排层定义；
4. **agent 再调度的重入点**：每轮返回后统一走「wb_my_task → trigger → act」的循环（可统一为 Leader 的主循环）。

---

## 9. subagent 如何获取 how（获取链路明确）

> 确认项 1：action 判断在主 agent，但**执行主体是 subagent**——subagent 拿到的是 prompt 中的 `action_id`，**how 本身它要自己从 wb_my_task 返回的 actions[] 里取**（而不是靠 prompt 里的自然语言描述）。

```
主 agent：
  wb_my_task(reqId, role, nodeId)
    → 返回 currentTrigger + activeActions = [ { action_id, name } ]
    → 选中要执行的 action_id（如 backend-dev.respond_to_cr）
  ↓
  写 TASK(goal/context) + 派发 prompt：
    [要求 subagent：调 wb_my_task 拿到该能力 actions[]，按 action_id=backend-dev.respond_to_cr
     取对应 action 的 how/reference/query/focus/tools 执行
     —— how/reference 中的模板路径（docs/{{req-version}}/{{req-name}}/...）由 host 端
     renderTpl(ctx) 渲染为真实路径后随 wb_my_task 返回，subagent 直接使用]
  ↓
subagent：
  ① 读 prompt：action_id = backend-dev.respond_to_cr
  ② 调 wb_my_task(reqId, role=rd, nodeId)   ← subagent 以自身角色身份调，返回该能力全 actions[]（路径已渲染）
  ③ 按 action_id 匹配 actions[].action_id → 取到 { how, reference, query, focus, tools }
  ④ 按 how + reference/query 执行（读 query 过滤评论 → 改代码 → 登记 → 返回）
```

**要点**：
- **how 的唯一真相源 = `capabilities.actions[]`**，经 wb_my_task 展开给 subagent（与主 agent 看到的一致，不用重复传）；
- prompt 只给 `action_id`（+ 必要的 goal/产物位置），subagent 自己取 how——**prompt 不承载 how 全文**（避免两份真相源漂移）；
- subagent 取 how 的机制 = **wb_my_task 返回 actions[] + 按 action_id 匹配**——等价于「程序路由，agent 执行」。

## 10. agent 领域模型 + 事件总线（触发入口全量）

> 确认项 2：定义 agent 领域模型 + 梳理事件总线（触发 agent 的全部事件入口）。

### 10.1 Agent 领域模型

```
┌────────────────────────────────────────────────────────┐
│ AGENT（执行实体）                                        │
│  → Leader（主 agent）：调度者 —— 判 trigger / 选 action / │
│    派发 / 收结果 / 再调度                                 │
│  → RoleSubAgent（角色子 agent ×N）：执行者 —— 按 action  │
│    产出/评审/验收（RD/FE/QA/CR/Architect/PM/Investigator）│
└────────────────────────────────────────────────────────┘
         │ 通过 wb_my_task 感知状态 / 通过 TASK 记录派发
         ▼
┌────────────────────────────────────────────────────────┐
│ 领域模型（数据层事实，驱动事件判定）                        │
│  requirements / versions / flows / flow_nodes            │
│  artifacts（产物，productType）                          │
│  issues（评论=问题，query 可过滤）                        │
│  tasks（唯一待办总线，派发/断点字段）                      │
│  inbox（User 收件箱入口）                                 │
└────────────────────────────────────────────────────────┘
```

### 10.2 事件总线（触发 agent 的全部事件入口）

```
事件源 → 事件 → 触发 agent 处理
─────────────────────────────────────────────────
① User 操作        收件箱条目/看板/评论动作/回复
                    → 事件：user_acted（decide/comment/reply/resolve/…）
                    → agent：处理对应节点/收件箱项
② 定时兜底扫描      扫描 TASK「已派未确认/进度陈旧」
                    → 事件：task_stale / task_unacked
                    → agent：resume（断点续做）
③ 节点状态变更      flow.node 进入待评审/通过/打回
                    → 事件：node_review_pending / node_passed / node_rejected
                    → agent：触发评审调度 / 打回返工调度
④ 评论/反馈        新 open comment（reviewer 提意见 / assignee 回复）
                    → 事件：feedback_open（含 reviewerRole/assignee）
                    → agent：响应（respond_to_cr / resolve 闭环）
⑤ 产物登记         新 artifact（dev.md/MR/设计文档）
                    → 事件：artifact_registered（productType）
                    → agent：check 门禁 / 推进
⑥ 评审表态         某人 wb_review pass/block
                    → 事件：review_cast
                    → agent：check 全评审者通过 → decide
⑦ 门禁推进         wb_decide pass/reject
                    → 事件：decide_made
                    → agent：推进下一节点 / 回滚 + 返工派发
```

> **统一入口**：所有事件收敛为「agent 主循环被唤醒」——事件以**增量事实**（哪节点/哪评论/哪 TASK 变了）进入，agent 统一 `wb_my_task` 算 trigger → 若 activeActions 非空 → 执行/派发。事件本身不决定动作，只提供「状态已变」的信号。

## 11. when 判断决策树

> 确认项 3：when 的判定——把「节点状态 + 评论状态 + 产物状态 → trigger → action」固化为决策树。

```
① 节点是否被打回过（rolledBackCount>0 或 state 被 reject 重置）？
   └─ yes → trigger=reject_rolled_back → action: <产出能力>.rework（重做/修订）
② 是否有「指派给我」的 open 反馈（query: assignee=我 & state=open）？
   └─ yes → trigger=reviewer_feedback(open)
        → 评审者反馈 → action: <产出能力>.respond_to_cr（或 QA 返工闭环）
③ 节点是 gate 且处于 待评审？
   └─ yes → trigger=node_review_pending → action: 评审能力 action（.review/.comment/.approve）
④ 本节点是否已登记产物但未提评？
   └─ yes → trigger=dev_complete → action: <产出能力>.submit
⑤ 节点 state ∈ {未开始, 进行中} 且无上述反馈？
   └─ yes → trigger=node_enter → action: <产出能力>.dev / .produce（主产出动作）
⑥ 全评审通过 / 产物就绪 → 再无激活动作？
   └─ yes → trigger=decide → action: decide-cap（门禁通过→推进/收尾）
⑦ 最后一个生产节点（用户验收/创建需求·启动版本）通过 → 进入「完成」节点？
   └─ yes → 无动作（DONE 占位，流程终点）→ Leader 确认收尾 / User 显式完成
⑧ 兜底扫描发现 TASK 未确认/陈旧？
   └─ yes → trigger=task_stale → action: resume（断点续做）
```

```
决策树（图）：
node.rolledBackCount>0 ──► reject_rolled_back → rework
        │no
有 assignee=我 的 open 评论 ──► reviewer_feedback(open) → respond_to_cr
        │no
gate & 待评审 ──► node_review_pending → review 动作
        │no
产物已登记未提评 ──► dev_complete → submit
        │no
state∈{未开始,进行中} ──► node_enter → produce 动作
        │no
全评审通过 ──► decide → 门禁推进/收尾
        │no
最后一个生产节点通过 ──► 「完成」节点（DONE 占位，流程终点）
        │no（无平凡路径）
TASK stale/unacked ──► resume（兜底）
```

> 判定顺序自上而下（第一命中）；全部由**程序可读数据**（flow_nodes.state / issues.query / artifacts.productType / tasks.dispatch_at·last_ack）构成，**不依赖自然语言**。

## 12. 能力缺口分析

> 确认项 4：对照「闭环所需」与「现有 22 个 wb_* 工具」，识别缺口。

**现有 22 工具**：listRequirements/listVersions/getRequirement/getArtifactContent/myTask / create|update|deleteRequirement / create|update|advance|deprecateVersion / comment/reply/resolve/override/decide / addArtifact/review/confirmReview/reopen/listInbox。

| # | 闭环所需能力 | 现状 | 缺口 |
|---|---|---|---|
| 1 | **派发**（agent→subagent 写 TASK） | ❌ 无工具操作 tasks | **新增 `wb_assign_task`**（写 tasks：goal/context/action_id/status=todo） |
| 2 | **TASK 推进/断点**（ack/progress/完成） | ❌ 无 | **新增 `wb_task_update`**（last_ack/last_progress/status→done） |
| 3 | **my_task 返回 actions+trigger** | ⚠️ opMyTask 有（旧结构） | **升级 opMyTask**：返回 currentTrigger + activeActions + actions[](含 how/query) |
| 4 | **事件入口统一**（user/定时/反馈…唤醒 agent） | ⚠️ 收件箱有 inbox；但无通用事件 | **事件源收敛**：定时扫描工具（host 侧）或统一事件监听 |
| 5 | **定时兜底扫描** | ❌ 无 | **新增兜底扫描**（host 定时 or agent 被调起时自检 TASK） |
| 6 | **subagent 执行** | ✅ 有（wb_my_task + 产物/评论工具 + fs/代码工具） | 无缺口（靠通用工具，非 wb_* 专属） |
| 7 | **行动作工具链**（produce/review 等） | ✅ 有 | 无工具缺口（走既有 wb_* + 代码工具） |

**结论**：
- **必补工具 2 个**：`wb_assign_task`（派发）、`wb_task_update`（ack/progress/完成）——承载 §4 派发与 §5 执行的心跳；
- **升级 1 个**：`wb_my_task` 增加 `currentTrigger + activeActions + actions[](how/query)` 返回；
- **事件源**：User 操作已由 UI→wb_* 覆盖；兜底扫描需要宿主侧定时机制（或 agent 被唤醒时先自检 TASK）——落地时需确定载体；
- 其余动作（产出/评审/评论/登记）依赖既有 22 工具 + subagent 的通用能力（fs/git/代码工具），**无需新增 wb_* 工具**。