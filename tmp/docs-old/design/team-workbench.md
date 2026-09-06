# 团队项目工作台（Team Project Workbench）设计文档

> 状态：Draft v0.3（已确认关键决策见 §1.3；协作流程见 §0）
> 定位：一个「项目管理 + OA 流程 + 评审协作」三合一的交互系统，运行在 DeepSeek Harness（DSH）之上。
> 单一事实来源：本文件。后续实现一律以本文件为基准拆分。

---

## 0. 协作流程（固化约定）

每个里程碑遵循**同一套协作节奏**，不因里程碑而变：

1. **先细化文档 → 产出「产品文档」**：针对该里程碑，写出面向页面/交互的产品说明（页面大致长什么样、有哪些区域/控件、交互路径、数据字段），而非直接写代码。
2. **用户确认**：你（User）审产品文档，认可后才进入实现。
3. **再实现**：确认后据此写代码；过程中若遇新决策点，回到 1 补充产品文档再定。

> 注：纯「数据层/后端」里程碑若无可见页面，则产品文档以「数据 Schema + 命令/查询接口说明 + 一张未来 UI 会用到的数据样例」呈现，同样需用户确认后再实现。

筛选交互形态（已定）：收件箱用**「项目 tab + 搜索框」**——顶部若干项目 tab（含「全部」），tab 内一个搜索框按任务名/关键词过滤；TODO 列表同款。

---

## 1. 背景与目标

我们不是在做一个通用的可编辑流程引擎，而是一个**流程写死、但交互体验对齐成熟项目协作软件**的工作台。它与普通项目管理软件的核心区别在于：

- 每个流程阶段**绑定产物**（如【PRD 评审】绑定 PRD 文档）；
- 每个阶段下**挂评审沟通记录**（评审意见、打回理由、改进说明、讨论往返）；
- 阶段负责的是 **Agent 角色**（PM/RD/Architect/FE/QA/…），而非仅人类成员；
- 门禁 = **User 评审决策**，且评审是一个**双向、多轮的对话过程**，而非单次通过/打回。

### 1.1 关键概念边界

| 概念 | 含义 | 归属 |
|---|---|---|
| **流程/角色/门禁制度** | 三类流程（需求/规划审批/版本规划）、8 角色、各阶段与标签规则 | 团队预设（preset，只读制度，供 Agent 了解） |
| **交互工作台** | 看板 / OA 流程树 / 通知 / TODO | 插件（plugin，供 User 与团队交互） |
| **过程数据** | 版本、需求、流程节点状态、制品、沟通记录 | 持久化存储（数据层） |
| **执行引擎** | 谁在推进节点、生成制品、回复评审 | 团队 Agent（Leader + 角色子 agent）+ workflow |

**一句话**：制度进预设（Agent 学习并遵循），界面进插件（User 操作），数据进存储（持久化），执行靠 Agent 编排。

### 1.2 设计原则

1. **流程写死、不造引擎**：三类流程以确定的节点/门禁建模，不做可视化流程编辑。
2. **门禁是人机交互的自然面**：User 的确认 = 评审卡点，评审必有产出物 + 可对话 + 可带理由打回 + 可异议多轮 + 可引入新范围反向确认。
3. **数据与 UI 解耦**：数据层是单一可查询的领域模型；UI 只是它的投影。
4. **渐进落地**：先静态流程骨架 + 交互 demo，再接入真实 Agent 编排。

### 1.3 已确认关键决策（v0.2）

1. **数据存储**：文件存储（`<workspace>/.workbench/*.json`），透明可 diff，规模上来再迁 `storageDomain`。
2. **制度只读**：流程/角色/门禁制度落在 team 预设（只读）；User 明文告知 → 由我协助调整，执行角色不得自行改制度。
3. **通知 = 必须处理的 IM 收件箱**：任何需要 User 回应的沟通都进收件箱（产出修订完成待验收、产出者反驳、产出者要求 User 补充遗漏答复等），双向、可回应、逐条必办；而非仅门禁提醒。
4. **锚点三级全支持**：文档级、章节级、行级/diff 级，粒度越精确越省产出 Agent 重读 token。
5. **项目 = DSH workspace/目录**：每个项目（目录）各有一套独立工作台数据；工作台是单一全局入口，顶部切换项目。通知为 **User 全局收件箱**（跨项目聚合，可按项目筛选）。

---

## 2. 四个核心模块

### 2.1 模块 I：项目工作台（版本看板 + 需求池）

类比飞书项目，但维度**固定写死**（不支持动态添加看板列/自定义流程）。

```
软件版本视图（看板列 = 版本）
├─ v1.0
├─ v1.1
├─ 待分配需求        ← 未进入任何版本的需求
└─ + （仅 Leader 可建版本，列写死不可增删流程）

进入某版本 / 待分配需求 或 点击版本列 → 需求池
需求池状态（看板的泳道/列）：
  待启动 | 进行中 | 已完成 | 已取消
```

- **版本**：`v1.0`、`v1.1`、`待分配需求`（虚拟列）。版本是需求容器，不承载流程。
- **需求池**：需求的集合，需求有状态。点击某需求 → 进入模块 II 的 OA 流程视图。
- 需求在此级别的属性：标题、类别（需求/规划审批/版本规划）、所属版本、当前状态、负责人（Leader）、创建时间。

### 2.2 模块 II：需求 OA 流程视图（流程树）

点击具体需求后进入的详情视图，**流程树用户可见**，每节点展示四要素：

- 做什么（节点职责）
- 负责人（角色，如 PM / RD / FE / QA / Architect / Leader）
- 进入时间
- 回滚时间（何时被退回 / 何时重做）

**节点粒度 = 一条流程标签一个独立节点**（需求任务 19 个，规划审批 4 个，版本规划 11 个）。例如「产品设计」与「产品设计评审」是**两个独立节点**，不合并成大节点内的子态。原因：User 可能在评审时补充需求（增加某功能/改进某页面），导致 RD、FE 也需要重新评审，故「评审」必须可独立被回退、重新进入。

**前进/回退语义**：

- **通过** → 前进到下一个节点；
- **打回** → 回退到**上一个节点**（重新产出），而非在当前节点内打转。

```
节点【产品设计评审】（review 阶段，被评审者=产出者 PM）
  ├─ 产出物：docs/pm/<模块>/product-spec.md（点击 → 弹窗阅读器）
  ├─ 评审状态行（紧凑）：Architect ✅  RD ✅  FE ⚠2  QA ⚠轮14（疑似卡死）
  │     └─ 点某角色 → 展开该角色的「意见 ↔ 被评审者反馈」对话区
  │           ├─ 每个 Issue 一条对话串（评审者提意见 / 产出者反馈 / User 可介入发言或拍板）
  │           └─ User 介入：参与讨论 + 介入确认（关闭 Issue） + 强权〔强行通过/强制修改〕
  └─ 文档级沟通记录（节点内 thread）
```

**评审是两阶段，角色评审在前、User 终审在后，但 User 可随时介入**（核心闭环）：

1. 节点进入 review 阶段，多个**评审角色**（Architect/RD/QA/FE 等，按节点配置）各自评审，各自产出意见（Issue）并给出 `已通过 / 有问题` 状态；
2. 每个 Issue 有 `roundCount`（围绕它的往来轮次），可监测"谁卡住、卡了几轮"；
3. 每个 Issue 下是**意见 ↔ 被评审者反馈的对话串**（评审者提意见、产出者回应、可多轮）；**User 可作为参与方随时发言，也可「介入确认」直接拍板关闭某 Issue**（减少返工、省 token）；
4. **所有评审角色都通过后**，才进入 **User 终审**（通过 → 前进 / 打回 → 回退上一节点）；
5. User **必要的时候仍可强权干预**（区别于参与讨论）：
   - **强行通过**：某角色/某 Issue 长期卡死，User 判定无须再等，override 为通过并推进；
   - **强制修改**：User 对产出者下强制修改令，产出者必须改、**不可异议**。

**产物与锚点评论的交互**（与纯项目管理软件的分水岭）：

- 产物在节点详情里以**文件行**呈现（点文件 → 弹出**阅读器**）；
- 阅读器：左半文档正文 + 右半侧边栏锚点评论；
- **锚点三级**：
  1. **文档级**：节点内 thread（整个产物的讨论串）；
  2. **章节级（heading）**：侧边栏评论锚定某章节；
  3. **行级**：在正文某行 hover 出现「+ 评论」，评论锚定该行（M2 先做文档类；代码产物走 git-review 式行内 comment，后置）。

> 锚点统一表达：`anchor = { kind(document|heading|line|diff), targetId, heading?, line?, diffRef? }`。粒度越精确，产出 Agent 重读范围越小、越省 token。

#### 2.2.1 三类流程的节点定义（沿用 Instructions）

见附录 A（需求任务 19 节点 / 规划审批 4 节点 / 版本规划 11 节点），此处只列门禁点。

| 流程 | 门禁点（需 User 确认） |
|---|---|
| 需求任务 | 产品设计后、技术设计后、最终用户验收（另有前端技术设计/测例评审为非 User 门禁） |
| 规划审批任务 | 待审批（一次性审批，打回则回到进行中） |
| 版本规划任务 | 架构设计对齐后、架构文档后 |

### 2.3 模块 III：门禁通知（= User 的 IM 收件箱）

定位：**它不是一次性门禁提醒，而是「必须处理的 IM 收件箱」**。任何需要 User 回应的沟通都进来，逐条必办、双向可回应。

收件箱条目类型（均需 User 处理）：

| 类型 | 触发 | 来源 |
|---|---|---|
| 待评审门禁 | 节点进入「待评审」 | 流程状态变更 |
| 待界定新范围 | 打回理由中引入新范围，边界待 User 界定 | 产出者反馈 |
| 产出修订完成 | 产出者按 User 打回意见修订完毕，待复验 | 产出者 |
| 产出者反驳 | 产出者就 User 打回意见提出异议 | 产出者 |
| 要求补充答复 | 产出者发现 User 的答复有遗漏，要求 User 补充 | 产出者 |
| 对话待回复 | 任一方向有未回复的沟通 | 双向 |

原则：**不打断、可聚合、可点击进入、逐条可回应、可筛选**。

- 形式：`shell.overlay` badge/toast + 未读数；点击进入收件箱，再跳对应节点评审区；
- 收件箱为 **User 全局**（跨项目聚合），**必须支持筛选**：**项目筛选**（按 workspace）+ **任务名筛选**（按需求/任务，支持关键词搜索），两层可叠加；消息多时不会淹没；
- 收件箱条目按 `projectId` + `requirementId/requirementTitle` 建立可筛索引（见 §3.4）；
- DSH 现成可挂靠：`approval/request`（系统审批）、`session/event`、`goal/changed`、`domain/changed`。业务收件箱建议走**领域数据状态变更 + 客户端订阅**，不污染系统 approval 语义。

### 2.4 模块 IV：TODO 列表

TODO 是**收件箱（模块 III）的可操作投影**，同样为 User 全局、跨项目聚合，支持**项目筛选 + 任务名筛选**（与收件箱一致）：

- 待评审节点（门禁）
- 待界定新范围
- 打回后等待产出者修订（追踪中）
- 我的意见待回复 / 待补充答复（对话追踪）

放置位：`conversation.input.dock`（输入框上方，已有 todo/goal/queue 先例）。以「未完成数 + 分类」聚合，辅助判断还有哪些没做；点击跳对应节点。

---

## 3. 数据模型（领域模型，待细化）

> 以下为逻辑模型，落地存储实现见 §4。
> **多项目**：所有实体都以 `projectId`(= DSH workspace 目录路径) 分桶，项目各自一套独立数据。

### 3.1 顶层

```yaml
Project:        projectId(= 目录路径), name, createdAt
Version:        projectId, id, name(v1.0…), isBacklog(待分配), createdAt
Requirement:    projectId, id, title, category(需求/规划审批/版本规划),
                versionId|null, status(待启动|进行中|已完成|已取消),
                owner(Leader), createdAt, updatedAt
```

### 3.2 OA 流程实例（含节点内多角色评审状态）

```yaml
FlowInstance:   projectId, requirementId, kind(需求|规划审批|版本规划),
                currentNodeId, startedAt

FlowNode:       id, key(PRD评审…), kind, order,
                ownerRole(产出者角色，如 PM/RD/FE/QA/Architect/Leader),
                state(未开始|进行中|待评审|通过|打回),
                enteredAt, rolledBackAt[], lastDecision,
                gate(需要User终审?),
                reviewers[]:             # review 阶段的各评审角色，独立各一条
                  { role(Architect|RD|FE|QA|…),
                    state(待评审|已通过|有问题),
                    issueIds[],
                    roundCount }          # 该角色讨论总轮次，用于识别卡点
```

### 3.3 评审问题、产物、锚点与沟通记录

```yaml
# 评审角色对产物提的意见 = Issue（可独立追踪状态与轮次，含意见↔反馈对话）
Issue:          id, nodeId, reviewerRole, anchor,
                text, state(open|resolved|overridden),
                roundCount, lastReplyAt,
                thread[]:      # 该 Issue 下意见↔反馈对话串
                  { by(reviewerRole|产出者|User), role(reviewer|producer|user),
                    text, at, userDecision? }   # userDecision=介入确认/拍板

# User 介入（参与讨论）+ 强权动作
UserMsg:        issueId, by(User), text, at, userDecision?(确认/裁定)  # 参与讨论
Override:       nodeId, issueId|reviewerRole, by(User), at             # 强行通过（强权）
ForceRevise:    nodeId, order, by(User), at                            # 强制修改令（强权）

Artifact:       id, nodeId, label, href, note(设计理念/摘要)
Anchor:         kind(document|heading|line|diff),   # 三级粒度
                targetId, heading?, line?, diffRef?
Comment:        id, anchor, author(角色|User), role(user|reviewer|producer),
                text, at, kind(评审意见|打回理由|异议|修订说明|边界界定)
Decision:       nodeId, verdict(pass|reject), reason, at, by(User)
```

> 关键：一个节点的「评审通过」= 所有 `reviewers[].state` 均为 `已通过`（或该角色 Issue 全部 `resolved`/`overridden`），之后才进入 User 终审。`roundCount` 高 + 仍在 `open`，即"疑似卡死"，供 User 关注并可 `override`/`forceRevise`。

### 3.4 收件箱（模块 III/IV 共用的数据）

```yaml
InboxItem:      id, projectId, requirementId, requirementTitle(冗余，便于筛选展示),
                kind(待评审门禁|待界定新范围|产出修订完成|
                产出者反驳|要求补充答复|对话待回复),
                target(requirementId|nodeId|commentId), summary,
                state(未处理|已处理), createdAt, doneAt
```

> 为支持「项目筛选 + 任务名筛选」，每条 InboxItem 冗余 `projectId` 与 `requirementId/requirementTitle`，查询时按此过滤/搜索；`requirementTitle` 冗余避免跨文件 join。

### 3.5 状态转移（关键不变量）

- 节点进入「待评审」= review 阶段，各 `reviewers[]` 初始 `待评审`，产生 **门禁类** InboxItem；
- 评审者提 Issue（`roundCount` 递增）；产出者改后 Issue 可 `resolved`；
- **节点通过** = 所有 `reviewers[].state` 为 `已通过`（Issue 均 `resolved`/`overridden`），并进入 User 终审；
- **强行通过（override）**：User 将指定 `issueId`/`reviewerRole` 的未解决项置 `overridden`，等价于该角色通过；
- **强制修改（forceRevise）**：User 对产出者下强制令，产出者必须改、不可异议（区别于普通打回可异议）；
- **通过 → 前进下一节点；打回 → 回退上一节点**（节点独立，一条标签一个节点）；
- 「打回」必须携带 `reason`（非空），当前节点记 `rolledBackAt`，流程回退到上一节点重新产出；
- 产出者异议 = 追加 Comment（不回滚节点状态），产生 `产出者反驳` InboxItem；
- 引入新范围 = 打回理由标记 → `pendingScope` + `待界定新范围` InboxItem；
- 任何方向未回复沟通 → `对话待回复`/`要求补充答复` InboxItem，直到闭环。

---

## 4. DSH 落地映射（技术方案）

> 分三层：**制度层（preset）**、**交互层（plugin）**、**数据层（storage）**。执行引擎由 Agent + workflow 承担。

### 4.1 制度层：团队预设（team preset）

- 从 `standard` `copy()` 得到（**已确认不选 `cordis`**），persona 写入 §2.2.1/附录 A 的完整制度。
- 制度**只读**：供 Leader 与各角色子 agent 学习遵循，不负责渲染 UI；变更仅经「User 明文告知 → 我协助调整」通道，执行角色不得自行改。
- 团队工具集：`subagent`、`subagent_fork`、`workflow`、`todo_write`、`ask_user_question`、`bash`、`fs` 等（沿用 standard）。

### 4.2 交互层：工作台插件（workbench plugin）

| UI 载体 | 承载内容 | Slot |
|---|---|---|
| 版本看板 + 需求池（模块 I） | 全屏视图 tab，内部含**项目选择器**（顶部切换 workspace） | `conversation.view`（新增 tab id=`workbench`，`order: 20`，label 由函数返回） |
| 需求 OA 流程树（模块 II） | 全屏视图（同上 tab 内二级视图，或切换） | 同上 |
| 门禁通知/收件箱（模块 III） | frame 级 badge/toast + 未读数 + 收件箱列表（User 全局，跨项目可筛） | `shell.overlay` |
| TODO 列表（模块 IV） | 输入框上方聚合栏（User 全局，跨项目可筛） | `conversation.input.dock`（todo 先例） |

- 插件用 **Host + Client 两半**：Host 持有领域读写（读写 §3 的模型，按 `projectId` 分桶），通过 `harness.handle` 暴露 `workbench.listVersions/listRequirements/listNodes/decide/comment/inbox/…` JSON 方法；Client 渲染四模块 UI，经 `host.call` 交互。`inbox` 方法带筛选参数（`projectId`、`requirementId`、`q`（任务名关键词））。
- 数据订阅：Host 在数据变更后经 `harness` 通知 / Client 轮询或订阅 `session/event`、`domain/changed` 等既有事件，驱动 UI 刷新。

### 4.3 数据层：持久化存储（已确认）

**采用文件存储**：`<workspace>/.workbench/*.json`，每个项目（workspace）一套独立数据，透明可 diff、便于 git 管理与人工检视。

- 示例结构：`<ws>/.workbench/versions.json`、`requirements.json`、`<requirementId>/flow.json`、`<requirementId>/comments.json`、`inbox.json`。
- 后续规模/并发要求上升，可迁移 `storageDomain`（保留本层抽象，迁移不碰 UI/制度）。

### 4.4 执行层：Agent 编排（后期接线）

- Leader 维护流程实例状态，按节点调度角色子 agent 产出物。
- 角色子 agent 用 `subagents.startContinuable`（保活）+ `followup` 实现「评审↔产出者多轮对话」。
- 固定流水线可用 `workflow` 的 phase 表达；门禁处暂停交 User。

> **MVP 边界**：静态流程 + 交互 demo **不接真实子 agent**，产出者回复由 Host 模拟（已验证可行）；体验确认后再接 `subagents` 真实编排。

---

## 5. 实现拆分（里程碑）

> 每个里程碑可独立评审、可回滚，符合「先设计再逐段拆解」。

- **M0 设计对齐**：本设计文档 + 待确认清单达成一致（当前，v0.2）。
- **M1 数据层**：文件存储 Schema + 多项目分桶 + CRUD（`<ws>/.workbench/`）。
- **M2 模块 II 流程树（纵向切片）**：需求 OA 流程视图 + 产物 + 三级锚点评论（文档/章节/行级先做，diff 级后置）+ 通过/打回/异议/新范围完整评审闭环（UI + Host RPC）——**价值核心，先做深**。
- **M3 模块 I 版本看板 + 需求池 + 项目选择器**：项目切换、版本列、需求池状态、点击进入流程树。
- **M4 模块 III 收件箱**：六类 InboxItem 生成 + `shell.overlay` badge + 收件箱列表 + 点击跳节点。
- **M5 模块 IV TODO**：收件箱的操作投影，`conversation.input.dock` 聚合栏。
- **M6 团队预设接线**：persona 制度录入 + 领域工具接入（Leader 用）。
- **M7 执行层**：真实 continuable 子 agent 多角色评审编排，替换 Host 模拟回复。
- **M8 锚点 diff 级**：产物版本化 + diff 锚点（若 M2 未做，后置到此）。

---

## 6. 待确认问题清单（剩余阻塞点）

已确认（v0.2）：数据存储=文件、制度只读、通知=IM 收件箱、锚点三级、项目=workspace + 全局收件箱。

仍待确认：

1. **版本列与需求的创建权限**：版本/需求由谁建？（倾向：Leader 建，User 也可手动建，但流程不变更。）
2. **流程树是否允许「同一流程多实例并发」**：一个需求同一时刻是否只能走一个流程实例？（倾向：一需求一流程实例。）
3. **门禁通知的触发粒度**：已扩展为「收件箱六类」，需确认是否还有遗漏的必须处理类型。
4. **TODO 与收件箱的关系**：TODO 就是收件箱的操作投影（已倾向），还是两者独立（TODO 含更多非门禁项）？待最终确认。
5. **锚点 diff 级的技术来源**：行级锚点好定（绝对行号 + 内容 hash），但 diff 级锚点需基于「某次改动的前后 diff」——产物文件当前是 markdown 平铺在仓库，diff 级需引入版本化产物（每次修订存 revision + diff）。**已确认：diff 级后置到 M8，M2 先做文档/章节/行三级。**
6. **工作台 tab 可见性**：`conversation.view` 新增 tab 是否对所有会话可见？（倾向：仅团队会话/有 `.workbench` 数据的 workspace，但 DSH 的 view tab 是 session 级，需要确认如何按项目门控。）
7. **收件箱跨项目聚合的「我」身份**：DSH 里 User 身份如何跨会话稳定标识？（影响全局收件箱的归属。）

---

## 附录 A：三类流程节点（沿用 Instructions，待精简为节点表）

### 需求任务（19 节点标签）
待启动 → 产品设计中 → 待产品设计评审 → 待产品设计用户确认 → 技术设计中 → 待技术设计评审 → 待技术设计用户确认 → 前端技术设计中 → 待前端技术设计评审 → 测例编写中 → 待测例评审 → 后端开发中 → 前端开发中 → 待后端代码评审 → 待前端代码评审 → 待联调验收 → 联调中 → 用户验收中 → 已完成

### 规划审批任务（4 节点）
待规划 → 进行中 → 待审批 → 审批完成

### 版本规划任务（11 节点）
待规划 → 调研中 → 待调研评审 → 架构设计对齐中 → 待架构设计对齐评审 → 待架构设计对齐用户确认 → 架构文档编写中 → 待架构文档评审 → 待架构文档用户确认 → 创建需求任务 → 已完成

> 门禁点见 §2.2.1 表。
