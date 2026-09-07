# M2 技术设计：需求 OA 流程树（需求详情页）

> 里程碑：M2（数据层 M1 并入）
> 状态：Draft v0.1（待 User 评审）
> 前置：产品文档 `m2-flow-tree.md`（已定稿）
> 运行平台：DeepSeek Harness（DSH），作为动态 Cordis 插件（Host + Client 两半）

---

## 1. 目标与非目标

### 1.1 目标
把产品文档 `m2-flow-tree.md` 描述的「需求详情页 = OA 流程树」实现为可运行的插件：
- 流程树（节点独立，通过→前进、打回→回退上一节点）
- 评审状态行（多角色 + 点角色展开「意见↔反馈」对话区 + User 参与讨论/介入确认 + 强权干预）
- 产物（文件行 → 弹窗阅读器 + 文档级/章节级/行级锚点评论）
- 门禁（通过 / 打回带理由 / 新范围界定）

### 1.2 非目标（后置里程碑）
- 版本看板 + 需求池（M3）、收件箱（M4）、TODO（M5）
- 代码产物的 git-review 行内评论（后置，接近 M7）
- diff 级锚点（M8，需产物版本化）
- 真实 continuable 子 agent 编排（M7）；本里程碑产出者回复由 Host 模拟
- 多项目切换 UI（M3 做，但数据层已按 `projectId` 分桶）

---

## 2. 架构分层

沿用主设计文档 §4 的四层，本里程碑落地**数据层 + 交互层**，暂不动制度层/执行层。

```
Client（浏览器）                    Host（Node 进程）
────────────────                   ────────────────
┌─ conversation.view 新增 tab ─┐    ┌─ 数据访问库（fs + JSON）─┐
│  id="workbench"              │    │  .workbench/*.json 读写    │
│  FlowTreeView 组件树          │◄──▶│  （按 projectId 分桶）     │
│   ├─ StepList 左列            │host│                           │
│   ├─ NodeDetail 右列          │call│  harness.handle RPC:       │
│   │   ├─ ReviewerStrip        │    │   getRequirement          │
│   │   ├─ ArtifactFiles        │    │   comment / anchorComment │
│   │   ├─ DocThread            │    │   decide / override /     │
│   │   └─ GateBar              │    │   forceRevise / defineScope│
│   └─ ArtifactViewer 弹窗      │    │   replyIssue              │
└──────────────────────────────┘    └───────────────────────────┘
```

- **Host 持有所有领域读写**（`ctx.get('fs')`），Client 不直接碰文件，全部经 `host.call`。
- **数据订阅**：Host 写后，Client 通过「每次操作返回最新快照」或轻量轮询刷新；M2 用**操作后回读全量需求**的简单策略（数据量小），订阅优化后置。

---

## 3. 数据存储（文件 Schema）

根：`<workspace>/.workbench/`（`projectId` = workspace 绝对路径）。

```
<ws>/.workbench/
  requirements.json
  <reqId>/
    flow.json
    artifacts.json
    comments.json       # 统一：评论=问题（文档级/章节级/行级锚点 + 状态 + 子对话）
```

> 设计：一个需求一个目录，避免单文件过大；跨文件以 `requirementId`/`id` 关联。所有时间用 ISO 字符串；所有 `id` 用稳定随机串（避免依赖时间在并发下碰撞）。

### 3.1 requirements.json
```json
[
  {
    "id": "req-001",
    "projectId": "/Users/.../test-flow",
    "title": "盘前简报模块",
    "category": "需求任务",
    "versionId": "v1.0",
    "status": "进行中",
    "owner": "Leader",
    "createdAt": "2025-01-01T00:00:00.000Z",
    "updatedAt": "2025-01-02T00:00:00.000Z"
  }
]
```

### 3.2 flow.json（流程实例 + 节点状态）
```json
{
  "requirementId": "req-001",
  "kind": "需求任务",
  "currentNodeId": "n-03",
  "nodes": [
    {
      "id": "n-01", "key": "待启动", "order": 1,
      "ownerRole": "Leader", "gate": false,
      "state": "已通过", "enteredAt": "…", "rolledBackAt": []
    },
    {
      "id": "n-02", "key": "产品设计", "order": 2,
      "ownerRole": "PM", "gate": false,
      "state": "进行中", "enteredAt": "…", "rolledBackAt": ["…"]
    },
    {
      "id": "n-03", "key": "产品设计评审", "order": 3,
      "ownerRole": "PM", "gate": true,
      "state": "待评审", "enteredAt": "…", "rolledBackAt": [],
      "reviewers": [
        { "role": "Architect", "state": "已通过", "commentIds": [] },
        { "role": "RD", "state": "已通过", "commentIds": [] },
        { "role": "FE", "state": "有问题", "commentIds": ["cm-004","cm-005"] },
        { "role": "QA", "state": "有问题", "commentIds": ["cm-006"] }
      ]
    }
  ]
}
```

> **节点状态机** `state`：`未开始 → 进行中 → 待评审 → 已通过`；打回时当前节点记 `rolledBackAt` 并置回 `进行中`，同时上一节点从 `已通过` 回到 `进行中`（回退语义）。
> **reviewer.state**：`待评审 → 已通过 | 有问题`。派生值：该角色名下所有评论（`commentIds`）都 `resolved` 才算 `已通过`。
> **reviewers 是可选的**：非 review 门禁节点可无 `reviewers`；但**任何节点的通过条件都包含「该节点全部评论已 resolved」**（见 §4）。

### 3.3 artifacts.json（产物，文件行）
```json
[
  {
    "id": "art-001", "nodeId": "n-03",
    "label": "产品文档 product-spec.md",
    "href": "docs/pm/premarket/product-spec.md",
    "kind": "document",           // document | code（code 后置）
    "note": "以5分钟速览为主线"
  }
]
```

> 产物即文件行：文件名 + href + note。**无 sections 章节树**（早期「章节级/heading」锚点已废弃，改为文档级/行级/范围/块四级；见 §3.4）。正文在阅读器内由 Client 轻量解析 markdown，按行渲染。

### 3.4 comments.json（统一模型：评论=问题，四级锚点 + 状态 + 子对话）
```json
[
  { "id": "cm-001", "requirementId": "req-001", "nodeId": "n-03",
    "anchor": { "kind": "document", "artifactId": "art-001" },
    "author": "User", "role": "user", "text": "请补充降级说明", "at": "…",
    "state": "resolved", "thread": [
      { "by": "PM", "role": "producer", "text": "已补降级横幅方案", "at": "…" }
    ], "resolvedBy": "User", "resolvedAt": "…" },

  { "id": "cm-003",
    "anchor": { "kind": "line", "artifactId": "art-001", "line": 13, "lineText": "宕机不阻塞浏览" },
    "author": "User", "role": "user", "text": "这行降级说明要写清楚", "at": "…",
    "state": "open", "roundCount": 0, "thread": [] },

  { "id": "cm-004", "nodeId": "n-03", "reviewerRole": "FE",
    "anchor": { "kind": "range", "artifactId": "art-001", "startLine": 5, "endLine": 8 },
    "author": "FE", "role": "reviewer", "text": "这段范围需补充分支对比", "at": "…",
    "state": "open", "roundCount": 3, "thread": [] },

  { "id": "cm-005", "nodeId": "n-03", "reviewerRole": "QA",
    "anchor": { "kind": "block", "artifactId": "art-001", "blockStart": 22, "blockEnd": 30, "lang": "mermaid" },
    "author": "QA", "role": "reviewer", "text": "这个流程图缺异常分支", "at": "…",
    "state": "open", "roundCount": 14, "thread": [] }
]
```

> **统一模型**：任何评论 = 一个待解决问题。
> **锚点四级**：`document`（文末）/ `line`（单行）/ `range`（多行范围 startLine~endLine）/ `block`（代码块 blockStart~blockEnd + lang）。**已废弃** `heading`/章节级。
> **每条评论字段**：`state(open|resolved|overridden)`、可选 `reviewerRole`、`thread[]`、`resolvedBy`/`resolvedAt`、`roundCount`。
> **resolve 语义**：提问方/有权者 resolve；`overridden` = User 强权强行通过。

---

## 4. 状态机（精确转移表）

### 4.1 节点（FlowNode.state）
| 当前 | 事件 | 下一态 | 副作用 |
|---|---|---|---|
| 未开始 | 进入 | 进行中 | 记录 enteredAt |
| 进行中 | 产出完成（若 gate） | 待评审 | reviewers 初始化 `待评审` |
| 待评审 | 全部评论 resolved + User 通过 | 已通过 | 前进下一节点 |
| 待评审 | 打回 | 进行中 | rolledBackAt 追加；回退上一节点为进行中 |
| 已通过 | （被下一节点打回） | 进行中 | rolledBackAt 追加 |

### 4.2 Comment.state（统一「评论=问题」）
| 当前 | 事件 | 下一态 | 说明 |
|---|---|---|---|
| open | 被回复到可关闭，提问方/有权者 resolve | resolved | 记 resolvedBy/resolvedAt |
| open | User 强权 override | overridden | 等价通过（极端卡死） |
| resolved/overridden | — | （终态） | — |

### 4.2.1 节点「通过」判定（门禁 = 全部评论 resolved）
- **节点的通过条件 = 该节点名下所有评论均 `resolved` 或 `overridden`**（无论评审角色提的、User 提的、还是文档级/行级/范围/块级锚点评论）。
- UI 始终展示 `已解决/总数`（如 `9/10`、`2/2`）；**未全部解决时「通过」按钮置灰/拦截**，防止处理中还误通过。
- 有 `reviewers` 时，额外派生：每个 `reviewer.state` = 其 `commentIds` 全部 resolved 才算 `已通过`。

### 4.3 Review 通过与打回（User 终审）
- `通过`：**前置校验**该节点全部评论已 resolved；通过后节点 `已通过` → `currentNodeId` 前移下一节点。
- `打回`：必须 `reason` 非空；当前节点 `进行中` + `rolledBackAt`；`currentNodeId` 回退上一节点。可选产生 `pendingScope`。

---

## 5. Host RPC 接口（`harness.handle`）

> 所有方法返回**最新该需求聚合快照**，Client 直接用于重渲染。参数/返回值仅含纯 JSON。

| 方法 | 入参 | 出参/副作用 |
|---|---|---|
| `wb.getRequirement` | `{ projectId, reqId }` | 聚合快照：requirement + flow + artifacts + comments（统一模型） |
| `wb.getArtifactContent` | `{ projectId, artifactId }` | 读取产物文件正文（供阅读器展示）+ 解析出的章节树 |
| `wb.comment` | `{ reqId, nodeId, anchor, text, role? }` | 追加一条评论（open）；Host 模拟产出者回复追加一条 thread |
| `wb.reply` | `{ reqId, commentId, text }` | 在评论下追加一条对话 |
| `wb.resolve` | `{ reqId, commentId }` | 评论 → resolved（记 resolvedBy=当前操作者） |
| `wb.decide` | `{ reqId, nodeId, verdict(pass\|reject), reason, newScope? }` | 通过（校验全评论 resolved）→前进；打回→回退 +（可选）pendingScope |
| `wb.override` | `{ reqId, commentId\|reviewerRole }` | 评论或整角色评论 → overridden |
| `wb.forceRevise` | `{ reqId, nodeId, order }` | 记 ForceRevise 令（产出者不可异议） |
| `wb.defineScope` | `{ reqId, nodeId, boundary }` | 关闭 pendingScope，写入边界 |

> 写冲突处理：M2 单用户操作，读-改-写无并发锁；每次写用 `fs.writeText` 全量覆盖对应文件。后续多人并发再引入乐观锁。

### 5.1 agent 接口规划（后置，随里程碑注册）

目标：让 agent / 子 agent 通过**模型工具**直接驱动流程与评审（推进节点、参与讨论、解决评论），与 UI 走同一套门禁/状态机/审计逻辑。注册方式：Host 用 `harness` 注册（`Tool.listTools` 可见），执行路径复用下表 handler。

| agent 工具 | 对应 RPC | 入参 | 出参 | 业务语义 |
|---|---|---|---|---|
| `wb_get_requirement` | `wb.getRequirement` | `{reqId}` | 聚合快照或 `null` | 读流程树/评论/产物（评审前置） |
| `wb_get_artifact_content` | `wb.getArtifactContent` | `{reqId, artifactId}` | `{art, content}` 或 `null` | 读产物正文（定位锚点评论） |
| `wb_comment` | `wb.comment` | `{reqId, nodeId, anchor, text}` | `{ok, snapshot}` | 追加问题评论（open） |
| `wb_reply` | `wb.reply` | `{reqId, commentId, text}` | `{ok, snapshot}` | 评论下追加对话（roundCount+1） |
| `wb_resolve` | `wb.resolve` | `{reqId, commentId}` | `{ok, snapshot}` | 评论 → resolved |
| `wb_override` | `wb.override` | `{reqId, commentId?/reviewerRole?}` | `{ok, snapshot}` | 评论/整角色 → overridden（强权） |
| `wb_decide` | `wb.decide` | `{reqId, nodeId, verdict, reason?, newScope?}` | `{ok, snapshot}` | **门禁**：通过（校验全评论 resolved）/打回（回退+理由） |

> 场景示例：agent 评审产物 → `wb_get_artifact_content` 读文档 → `wb_comment` 提行级问题 → User 处理后 `wb_decide` 通过/打回。**所有写操作自动遵守 §4 状态机与门禁**（未全 resolved 时 `decide` 返回 `blocked`）。

---

## 6. Client 组件结构（React，纯 createElement）

```
FlowTreeView（workbench tab 的根）
├─ RequirementHeaderTop（顶部条 + ←返回需求池占位）
├─ StepList（左列，receives nodes）
│    └─ StepItem（节点：状态点/名/↩/时间）
├─ NodeDetail（右列，receives currentNode）
│    ├─ NodeMetaCard（职责/负责人/进入/回滚）
│    ├─ ReviewerStrip（评审状态行）
│    │    ├─ ReviewerPill（角色药丸，onClick 展开）
│    │    └─ ReviewerDisc（对话区：该角色全部评论 + 每评论的 thread + resolve + User 发言框）
│    ├─ ArtifactFiles（产物文件行）
│    ├─ CommentSummaryBar（已解决/总数，如 9/10，未全解决时通过置灰的提示来源）
│    └─ GateBar（通过/打回内联表单/新范围界定；通过被门禁计数拦截）
└─ ArtifactViewer（弹窗，点文件行打开）
     ├─ ViewerDoc（左：正文，行 hover 「+评论」）
     └─ ViewerSidebar（右：锚点评论列表，每条含 thread + resolve + 添加评论）
```

- 状态：用 `React.useState` + `host.call` 回读快照驱动；组件间通过 props 传数据（不引入 store 库）。
- 弹窗：`ArtifactViewer` 用 absolute/fixed 定位渲染在 `workbench` tab 内（不依赖 shell.overlay，因为它是 tab 内局部 UI）。

---

## 7. 行级锚点防漂移方案（已确认）

- 行级评论记录 `line`（绝对行号，从 1 起）+ `lineText`（该行**去首尾空白后的原文**，作为定位指纹）。
- **锚点指纹不用密码学哈希**：Host Builtin 已核实无 `crypto`/`Buffer`/`process.getBuiltinModule`，仅有 `btoa`/`atob`/`TextEncoder`。因此直接用「去空白后的行原文」作指纹，字符串相等即命中，无需哈希。
- 定位策略：渲染阅读器时，
  1. 先按 `lineText`（去空白后原文）在当前文档反查内容相同的行；
  2. 命中 → 锚到该新行号；
  3. 未命中 → 按 `line` 就近展示，并标「⚠ 锚点可能失效」；
  4. 用户重新确认时更新 `line`/`lineText`。
- 章节级锚点用 `sectionId` + 标题文本双定位（标题改名则标记失效）。

---

## 8. 与现有 board-2 的关系

- **重建**：现有 `board-2` 是审批板 mock（`.board.json` 平铺），数据结构与 M2 的 `.workbench/` 目录模型不兼容。
- 计划：新建插件（如 `wb` 前缀），实现 M2；`board-2` 与 `.board.json` **废弃**（不迁移，因为它只是 demo seed 数据）。
- 数据 seed：新建 `.workbench/` 示例（演示 1 个需求的完整流程树 + 2 个评审者带对话区 + 1 个产物带三级锚点评论）。

---

## 9. 待确认（技术设计评审点）

1. ~~行内容哈希~~ **已解决**：无 crypto，改用「去空白后的行原文」作指纹（见 §7）。
2. ~~markdown 章节层级~~ **已确认**：层级动态跟随产出物实际结构（`#`/`##`/`###`/`####`…），不固定两级；`sections` 为嵌套树，锚点可挂任意层级。
3. **需求初始从哪来**：M3 才做"创建需求"的 UI；M2 的 `.workbench/` 数据用手工 seed。（已确认后置）
4. **模拟产出者回复**：M2 用关键词规则模拟，文档级 thread 与评论子对话**都模拟**。（已确认）
5. ~~comments 与 issues 边界~~ **已确认：合并为统一「评论=问题」模型**（§3.4）。每条评论 `state(open|resolved|overridden)` + `thread[]` + `resolvedBy`；通过门禁 = 节点全部评论 resolved，UI 展示 `已解决/总数`。
6. ~~resolve 权限~~ **已确认**：提问方 resolve 自己的问题；User 对任何人的问题既可普通 `resolve` 也可 `override`（强行通过）。`override` 用于提问方长期不 resolve 但 User 判定已无必要的极端情况。

---

## 10. 实现顺序（小步多确认）

1. **T1 数据层**：`.workbench/` Schema + Host `fs` 读写 + `wb.getRequirement` + seed 数据（统一评论模型）。
2. **T2 流程树骨架**：`workbench` tab + 左列 StepList + 右列 NodeMeta（静态渲染 seed）。
3. **T3 门禁**：GateBar 通过/打回 + 状态机（前进/回退）+ 全评论 resolved 拦截 + 新范围界定。
4. **T4 评审状态行**：ReviewerStrip + ReviewerPill + 展开对话区 + 评论回复 + resolve + override/forceRevise。
5. **T5 产物 + 阅读器**：文件行 → ArtifactViewer + 文档级/章节级/行级锚点评论（动态层级）+ 防漂移。
6. **T6 沟通记录 + 已解决计数**：CommentSummaryBar（已解决/总数）+ 模拟产出者回复。
7. 每步实现后给你确认，再进下一步。
