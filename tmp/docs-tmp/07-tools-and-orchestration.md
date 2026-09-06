# 执行层：wb_* 工具 + Agent 编排协议 + 程序路由

> 本文档覆盖**执行层**：22 个 `wb_*` 领域模型工具 + Agent 编排协议 + 程序路由。
> 取代旧稿：`m6-agent-tools.md` / `m6-gap-closure.md` / `m7-agent-protocol.md` / `m8-flow-simplify.md` §2/§4/§11。
> 职责边界：工具清单、确定性任务（`wb_my_task`）、默认路由与问题归属、交互模式、编排协议。
> 流程节点表、角色/能力与协作铁律见 `06-flows-and-roles.md`（本文只引用不重写）。

---

## 1. 定位

执行层 = **Leader（主 agent）+ 角色子 agent**，通过 22 个 `wb_*` 模型工具直接驱动工作台——推进任务、拆分版本、评审闭环、澄清需求；与 UI 走同一套 `op*` 领域函数（同一审计 / 门禁 / 状态机，见 `01-architecture.md` §4）。

一句话：**程序做路由，agent 做专业**——「你在哪、看什么、产什么、通知谁」由程序从数据推导（`wb_my_task` / 默认路由），Agent 只做内容与专业判断（方案设计、内容质量、缺陷根因判定、评审结论），不持有流程路由表。

- 工具经 `Tool.listTools` 对 agent 可见，**参数 schema 即接口文档**，无需查外部文档。
- 技术实现：22 个 `wb.*` handler 提取为具名领域函数 `op*`，经 `harness.defineTool` + `registerTool` 注册为模型工具（Fiber 生命周期，插件 stop/update 自动注销），与 UI `host.call` 共用同一逻辑。
- 制度只读：执行角色不得自行改制度/流程；调整仅经「User 明文告知 → 协助调整 preset 文件」通道。

## 2. 工具清单（22 个 wb_*）

前置说明：

- ⚠️ = 破坏性/强权操作，工具描述标注「需 User 明确指令」，agent 调用前必须获得 User 确认。
- 失败统一 `{ok:false, error, ...}`；门禁拦截带 `blocked:true`；成功返回均为纯 JSON（lossless）。
- 单工具语义见下表；多步业务编排见 §5 交互模式与 §6 编排协议。

### 2.1 读（5）

| 工具 | 入参 | 出参 | 业务语义 |
|---|---|---|---|
| `wb_list_requirements` | — | 需求池数组（含 description） | 全量需求，推进/拆分前置 |
| `wb_list_versions` | — | 版本数组（id/name/state/isBacklog） | 确认目标版本（拆分选版） |
| `wb_get_requirement` | `{reqId}` | 聚合快照或 null | 需求字段+流程树+产物+评论；评审前置 |
| `wb_get_artifact_content` | `{reqId, artifactId}` | `{art, content}` 或 null | 读产物正文，定位评论锚点 |
| `wb_my_task` | `{reqId, role, nodeId?}` | 确定性任务上下文或 `{ok:false}` | **任务确定性来源**（详见 §3） |

### 2.2 需求（3）

| 工具 | 入参 | 出参 | 业务语义 |
|---|---|---|---|
| `wb_create_requirement` | `{title, description?, category?, versionId?, priority?, status?, owner?}` | `{ok, reqList}` | 建需求（默认同表单；`history=[]` 审计起点） |
| `wb_update_requirement` | `{reqId, ...7 字段}` | `{ok, reqList}` | 改需求（Host 算字段 diff 自动写审计） |
| `wb_delete_requirement` ⚠️ | `{reqId}` | `{ok, reqList}` | 删除需求（不删磁盘目录） |

### 2.3 版本（4）

| 工具 | 入参 | 出参 | 业务语义 |
|---|---|---|---|
| `wb_create_version` | `{name}` | `{ok, verList}` | 建版本（待启动，名称自由文本） |
| `wb_update_version` | `{versionId, name}` | `{ok, verList}` | 改版本名（`v-backlog` 待分配池锁定） |
| `wb_advance_version` | `{versionId}` | `{ok, verList, unfinished}` | 单向推进 待启动→进行中→已发布；发布返回未完成需求数 |
| `wb_deprecate_version` ⚠️ | `{versionId}` | `{ok, verList}` | 废弃版本（有需求引用 → blocked，需先移出） |

### 2.4 评论 / 评审（5）

| 工具 | 入参 | 出参 | 业务语义 |
|---|---|---|---|
| `wb_comment` | `{reqId, nodeId, text, anchor?, type?, domain?}` | `{ok, snapshot}` | 追加 open 评论（阻塞门禁）；**默认路由**（reviewerRole/to/assignee 程序推导，见 §4.1） |
| `wb_reply` | `{reqId, commentId, text}` | `{ok, snapshot}` | 评论下对话（round_count+1） |
| `wb_resolve` | `{reqId, commentId, note}` | `{ok, snapshot}` | 评论 → resolved（resolve 必须带 note；谁 resolve 见 `05` §3.2） |
| `wb_override` ⚠️ | `{reqId, commentId?/reviewerRole?}` | `{ok, snapshot}` | 强权：单条或整角色 open 评论 → overridden（绕过门禁） |
| `wb_decide` | `{reqId, nodeId, verdict, reason?, newScope?, targetBranch?}` | `{ok, snapshot}` | 门禁 pass（未闭环 blocked）/ reject（回退，reason 必填，targetBranch=部分回滚）；criterion 见 `05` §3 |

### 2.5 其他（5）

| 工具 | 入参 | 出参 | 业务语义 |
|---|---|---|---|
| `wb_add_artifact` | `{reqId, nodeId, label, href, kind(document\|link), note?}` | `{ok, artifactList}` | **产物登记**：文档正文由作者用文件工具写入项目目录，登记只写元数据 |
| `wb_review` | `{reqId, nodeId, role, verdict(pass\|block), commentIds?}` | `{ok, snapshot}` | **评审表态**：置 `flow.nodes[].reviewers[]` 对应 role 的 state；block 需附理由（评论）；`wb_decide pass` 联动校验全部 reviewers 已通过 |
| `wb_confirm_review` | `{reqId, commentId}` | `{ok, snapshot}` | **提评者复核通过**（pending-confirm → resolved，reviewer_confirmed=true，门禁放行） |
| `wb_reopen` | `{reqId, commentId}` | `{ok, snapshot}` | **提评者复核不通过**（pending-confirm → open，round_count++，通知 assignee 返工） |
| `wb_list_inbox` | — | 待办/通知列表 | 收件箱查询：通知是视图不是数据，由 `tasks`/`issues` 实时推导（见 `01` 决策 3 / `05` §2.6） |

## 3. 确定性任务（wb_my_task）

`wb_my_task` 是「程序做路由」的核心：程序从 flow.json / artifacts.json / comments.json 推导并返回**确定性任务上下文**，subagent 被派活后先调本工具拿全部上下文，无需自行检索、无需任何推断。

- **入参**：`{reqId, role, nodeId?}`。`nodeId` 缺省 = 当前节点（`flow.currentNodeId`）；nodeId 只决定 task/reviewers/outputs 的定位，`sessionsByNode` 总是全量返回；请求节点不存在 → `{ok:false, error}`。
- **返回结构**：

```
{
  requirement,          // 需求字段（id/title/category/reqType/scope/owner/status）
  node,                 // 当前节点（id/key/ownerRole/gate/state/isCurrent/branch/parallelGroup）
  mode,                 // 'produce' | 'align'（需求对齐节点 = align）
  gate, state,
  task,                 // 节点任务（gate → 评审+表态；否则产出）
  roleObjective,        // 本角色在本节点的行为目标（§2.4 行为协议表抽出）
  behaviors,            // 可做/应做动作（含建议工具）
  onFeedback,           // 被评审者收到反馈后的职责链（P2/P3/P4 节点）
  reviewers,            // 本节点评审归属（role/state，各审各的，评审维度在评审能力 how）
  artifactsToReview,    // 本节点待审/已产产物
  dependencyDocs,       // 前置节点已登记产物 href（QA 验收额外含 prd/cases/rd·fe 设计与代码）
  outputs,              // 本节点应产出（align 节点 = []）
  sessionsByNode: [...], // 全需求每个 node 的全部会话，按 node 分组（见下）
}
```

### 3.1 sessionsByNode：全会话供给

- **覆盖全部节点**：每个 node 一个组（空组 `sessions=[]`，无会话不报错）；组内按时间升序、`replies` 完整。
- **类型全含**：`issue` / `discussion` / `notification` 全部打包（含需求对齐阶段的澄清对话、各节点评审意见、锚定在产物文档上的会话）。
- **双重复位**：每组与每条会话均带 `nodeId` + `nodeKey` 明确归属，嵌套不冲突；文档锚定会话解析出 `anchorArtifact = {label, href}`。
- **对话本体唯一载体 = issues**：会话及其结论**不**另产物化到 artifacts；回复链内嵌在 `replies`，由 `wb_my_task` 打包供给——设计/开发节点 subagent 由此获得全流程完整上下文（含与 User 的对齐结论），评审者从目标节点组的会话看到已提意见历史，避免重复提问。

### 3.2 任务派生规则（程序，从编排信息 → 执行任务）

```
给定 flow.nodes + role：
  对每个「进行中或待评审」的节点 n（含并行组内全部未通过分支）：
    n.ownerRole === role                     → 产出任务(n)
    n.gate 且 n.reviewers 含 role 且未表态    → 评审任务(n, focus)
  返回任务列表：每个任务绑定 nodeId / nodeKey / branch / 产物 / 职责 / 上下文
```

### 3.3 并行编排 vs 执行者确定性任务

```
编排层（Leader / flow.json）：并行组是流程结构
  dev-1 = { n-09(RD 开发, branch=rd), n-10(FE 开发, branch=fe) }   // 供 Leader 调度
执行层（各角色 / wb_my_task）：任务永远确定性
  CR 收到 → 2 个明确任务：评审 n-09（rd，产物 rd/dev.md）+ 评审 n-10（fe，产物 fe/dev.md）
  RD 收到 → 1 个明确任务：产出 n-09（rd，产物 rd/dev.md）
```

要点：

1. 每个 node 有独立 id（n-09 / n-10 各自独立），`parallelGroup` 只是节点上的编排标记——**不是任务的形态**。
2. **`wb_my_task` 不返回「并行组」**，返回该角色要执行的具体任务列表，列表由程序从编排信息推导后**扁平化**给出。
3. **上下文确定性**：Leader 派活时上下文已是「谁完成了开发（产物已登记）、节点 id、等待评审」，执行者调 `wb_my_task` 直接得到明确任务，无需推断。
4. 部分回滚同理：QA 打回 n-10(fe) → 上下文变为「n-10 待返工」→ `wb_my_task` 给 FE「产出 n-10 返工」、给 CR「评审 n-10 返工结果」——仍是确定性任务。

## 4. 默认路由与问题归属

### 4.1 wb_comment 全切默认路由

- **agent 只传** `reqId + nodeId + text + anchor`（+ `type`? + `domain`?）；`reviewerRole` / `to` / `assignee` 一律由程序推导，不再向后兼容旧调用。
- **程序默认推导**：

```
reviewerRole = 当前节点 gate 的评审归属（flow.nodes[].reviewers[]）
to           = reviewerRole（或节点 ownerRole）
assignee     = anchor 产物作者（artifacts.addedBy）；domain=rd|fe 时 → RD|FE
```

- **覆盖参数**（reviewerRole/to/assignee 显式传）仅保留给 Leader 特殊场景；普通角色一律走默认。

### 4.2 问题归属三层

问题归属是通用模型：**谁发现产物中的什么问题 → 归属谁**（缺陷只是 QA 验收阶段、可能跨端根因的特例）。程序按顺序尝试，能定即定：

| 层 | 归属规则 |
|---|---|
| **L1 锚点产物作者（默认）** | 评论锚到哪个产物 → assignee = 该产物作者（rd/design.md→RD、fe/dev.md→FE、qa/cases.md→QA）——谁的产物问题就是谁的 |
| **L2 节点 owner** | 无锚点 → assignee = 当前节点 ownerRole（如设计评审意见 → 设计作者） |
| **L3 跨端根因判定** | L1/L2 无法唯一归属（问题横跨前后端，QA 验收、非纯端需求时）→ QA 做根因分析 → 输出 `domain=rd\|fe` → 程序路由 assignee/to（reviewerRole=开发节点评审归属）；QA 写《缺陷文档》+ `wb_comment(domain)` |

- **domain 是 L3 的根因判定输出，不是「打回谁」的标签**——只有 L1/L2 无法唯一归属时才需要 QA 判定；纯前端/纯后端需求（scope 决定）通常落 L1（产物作者唯一），不走 L3。
- **锚点兜底**：QA 锚到具体产物（如 fe/dev.md 某行）→ 程序按 L1 自动归属，无需 domain。
- 程序侧辅助判定上下文：`wb_my_task` 提供关联契约文档对照（rd/design.md 接口契约 vs fe/design.md 调用契约 vs qa/cases.md 预期）。

## 5. 交互模式（5 种可复用模式）

四个流程虽有 30+ 节点，但角色交互只有 5 种重复模式（这正是程序可统一的）：

| 模式 | 出现的节点 | 角色交互 | 程序统一处理 |
|---|---|---|---|
| **P1 产出→提评** | 产品设计 / RD·FE 设计 / 用例 / 方案 / 调研 / 架构规划 / 需求对齐 | 被评审者产 doc → 登记 → wb_comment 请审 | wb_my_task 给产出清单；wb_comment 自动 to=评审归属 |
| **P2 多评审者表态** | 需求评审 / 技术评审 / 设计评审 / 用例评审 / 立项 / 调研 / 架构规划 | 各评审者读产物 → 各自评论（各审各的）→ wb_review 表态 | wb_my_task 给「评审任务 + 我要审的产物」；reviewerRole 按节点推导 |
| **P3 开发→CR 并行评审** | 业务/技术/任务流程的开发节点 | 开发 + MR → CR 评审（并行组多分支） | wb_my_task 返回多任务（扁平化，§3.3）；domain 自动指派 |
| **P4 验收→缺陷闭环** | 业务/技术/任务流程的 QA 验收 | QA 验收 → 缺陷 domain 指派 → 返工 → resolve / confirm / reopen | wb_comment(domain) 全自动；reopen 拉锯升级（>5 轮 Leader 介入） |
| **P5 拍板推进** | 各 gate 节点末 | 评审通过后 Leader wb_decide 推进 | wb_decide 状态机（join / 部分回滚，见 `05` §3） |

**结论**：四流程 = 5 种交互模式 × 参数化（节点 / 产物 / 评审者来自 flow.json）。`wb_my_task` + `wb_comment` 默认路由即可覆盖全部，无需在 persona 中写任何节点专属逻辑（具体流程节点表见 `06`，不重复）。

## 6. 编排协议

### 6.1 Leader（主 agent）行为协议

Leader 是编排的唯一协调者，7 条行为：

1. **全局视图**：`wb_list_requirements` / `wb_list_versions` / `wb_get_requirement` 掌握需求池与流程状态。
2. **拆解分派**：按流程节点把工作分派给对应角色子 agent，prompt 自包含（需求 id、节点、产物路径、期望输出格式）。
3. **子 agent 并行**：无依赖的角色（如 RD/FE 开发）并行起后台子 agent；有依赖的串行等结果。
4. **正式落账**：子 agent 的产物/评论/评审经 `wb_*` 工具落账；子 agent 无工具时按 host.js 格式直写 `.workbench/`，Leader 负责校验并正式落账。
5. **门禁推进**：resolve 全部 open 评论 → 相关角色 `wb_review pass` → `wb_decide pass`（并行组 join 自动判定）。
6. **返工循环**：评审 block 时用 `send_message` 续话对应子 agent 返工 → 重评审；QA 部分回滚用 `wb_decide reject + targetBranch`。
7. **收尾**：最后节点通过后 `wb_update_requirement(status=已完成)`。

> **铁律**：Leader 不替角色表态（review 由对应角色子 agent、或以 by=角色 落账）；只做协调与流程推进。

### 6.2 角色子 agent 通用约定

- **prompt 自包含**：子 agent 看不到 Leader 对话，prompt 必须含需求 id、节点 id/key、角色职责、输入文档路径、产物路径与格式、期望输出格式、协作铁律。
- **无 wb_* 工具**（宿主注册的工具子会话不可见）：按 host.js 的 opAddArtifact / opComment 格式直接写 `.workbench/` JSON；写入后报告 artifactId / commentId。
- **不越权**：不改 flow.json 的节点状态 / currentNodeId（流程推进归 Leader）；只产文档、登记产物、发评论。
- **feature 分支**：开发类角色按 featureBranch 约定（评审节点敲定分支名）。

### 6.3 角色卡精简表

| 角色 | 扮演者 | 落地状态 |
|---|---|---|
| User | 人类（Human） | 固定：需求提出、最终拍板、制度修改入口 |
| Leader | 主 agent | ✅ 已定（行为协议见 §6.1） |
| RD / FE / Code Reviewer / QA | 子 agent | ✅ **已实证**（persona 已瘦身为「专业能力卡 + 产物规范 + 一句协作协议」三块，行为协议由 `wb_my_task` 返回） |
| Architect / PM / Investigator | 子 agent | ⏳ **待落地**：落地时按「专业卡 + 一句协作协议」生成，不写流程路由文字 |

角色职责视图、专业能力卡、能力与评审分工见 `06`；各角色的 roleObjective / behaviors / onFeedback 由 `wb_my_task` 按「节点类型 × 角色」查表返回，persona 不再持有。

### 6.4 缺陷闭环与部分回滚

- **建缺陷**：QA 验收发现缺陷 → 写《缺陷文档》并登记产物 → 跨端根因不明时按 L3 判定 `domain=rd|fe` → `wb_comment(domain)` 程序自动指派 RD|FE（普通缺陷按 L1/L2 归属，见 §4.2）。
- **返工闭环**：assignee 修复 → `wb_resolve(note=修复说明)`（进入 pending-confirm）→ QA 复核：通过 `wb_confirm_review` / 不通过 `wb_reopen`（round_count++ 重新指派）；QA 重跑自动化回测（`qa/data/` + `qa/code/` 落代码，供复验）。
- **部分回滚**：`wb_decide reject(targetBranch=rd|fe)` 仅将该分支开发节点回滚（state → 进行中，rolled_back_count++），另一分支保持已通过，并行组 join 状态重算。
- 评论状态机与谁 resolve/confirm/reopen 的精确转移见 `05` §3.2，不重复。

### 6.5 断点续接

Task 的 `dispatch_at / last_ack / last_progress / resume_hint` 字段承载派发与断点交接（见 `05` §2.6）：重启后扫描「已派未确认（last_ack=NULL）」「进度陈旧」即可断点续接——重新派发或按 resume_hint 从断点继续，无需人工对账。

## 7. 演进注记

- 执行层形态历经迭代后定稿如下：22 个 `wb_*` 工具、TASK 为唯一待办总线、`sessionsByNode` 全会话供给；本文与 `01`/`05`/`06` 只记录定稿。