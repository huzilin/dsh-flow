# M4 技术设计（实现回写）：需求池增删改 + 描述 + 编辑审计

> 里程碑：M4
> 状态：**已实现并验证**（pkg-8，currentPackageId=pkg-8）
> 前置：产品文档 `m4-req-crud.md`（v1.0）；可视化参考 `docs/prototype/m4-req-crud-prototype.html`
> 运行平台：DeepSeek Harness（DSH），动态 Cordis 插件 `wbflow-1`（Host + Client 两半）
> 说明：本文为**实现后回写**（M4 开发前未单独产技术设计，流程疏漏，此处补记），与 m3-board-tech.md 同为"实现回写"型文档。

---

## 1. 目标与非目标

### 1.1 目标
把 M3 的只读需求池变为可维护：
- 新建 / 编辑 / 删除需求（弹窗表单 + 二次确认）
- 需求描述（Markdown，让 Leader / 后续 PM 理解要做什么）
- **编辑审计**：每次保存由 Host 计算字段级 diff，写 `history[]`，详情页可查「变更历史」

### 1.2 非目标（后置）
- agent 工具注册（后置项，User 已确认要做）
- 版本管理 UI（M5）、真实流程树生成（M7，新建需求显示「暂无流程数据」）
- 拖拽换列、多项目切换、登录体系（审计 `by` 暂固定 `'User'`）

---

## 2. 架构分层

沿用四层架构，M4 在 **数据层 + 交互层** 扩展写路径。Host 持所有领域读写，Client 只经 `host.call`；**审计计算在 Host 完成**（防 Client 伪造）。

```
Client（浏览器）                          Host（Node 进程）
────────────────                          ────────────────
WorkbenchView（两级视图 + 弹窗）
├─ view='board'（看板/列表 + hover ✎🗑）
│    ├─ Topbar「＋ 新建需求」
│    └─ formModal / delModal（fixed 覆盖层）
└─ view='detail'
     ├─ DetailTop（←返回 + 徽标 + 「编辑」）
     ├─ DescPanel（需求描述 md 渲染）
     ├─ HistPanel（变更历史，倒序）
     └─ M2 流程树（复用）
           │
           └── host.call ──►  harness.handle:
                                createRequirement / updateRequirement（diff+审计）/
                                deleteRequirement / getRequirement / listRequirements
                                （fs + JSON 写 .workbench/requirements.json）
```

- **数据订阅**：增删改后返回最新 `reqList`（Client `setReqList` 直接刷新）；详情页编辑后重拉快照 `getRequirement`。

---

## 3. 数据存储（文件 Schema）

仅扩展 `requirements.json` 单文件数组（不新增文件、不动 `<reqId>/` 目录结构）。

### 3.1 requirements.json（+2 字段）
```json
[
  {
    "id": "req-001", "projectId": "/Users/.../test-flow",
    "title": "盘前简报模块",
    "description": "## 背景\n盘前 5 分钟…",     // M4 新增：Markdown 文本，可含代码块/链接
    "category": "需求任务", "versionId": "v-1", "priority": "P0",
    "status": "进行中", "owner": "Leader",
    "createdAt": "…", "updatedAt": "…",
    "history": [                                 // M4 新增：编辑审计（新建=[]）
      { "at": "…", "by": "User", "changes": [
          { "field": "priority", "from": "P1", "to": "P0" },
          { "field": "versionId", "from": "v-backlog", "to": "v-1" }
      ] }
    ]
  }
]
```

### 3.2 审计（history）模型
| 字段 | 类型 | 说明 |
|---|---|---|
| `at` | ISO 时间串 | 保存时刻 |
| `by` | string | 操作者（当前固定 `'User'`，无登录体系） |
| `changes[]` | 数组 | 字段级 diff：`{field, from, to}` |

- **参与 diff 的 7 字段**：`title / description / category / versionId / priority / status / owner`；
- **diff 规则**：`args[f] !== undefined && String(args[f]) !== String(req[f] || '')` 才算变化；`from`/`to` 存**原始值**（`versionId` 存短码 `v-1`，展示时 Client 映射版本名）；
- **无变化不追加 history**（避免空审计）；`updatedAt` 始终刷新；
- 新建需求 `history=[]`；删除需求整条移除（不写审计、不删 `<reqId>/` 目录）。
- **展示（pkg-23，v1.4）**：倒序渲染为自然语句 `by 将「字段中文名」从「旧值」变更为「新值」`（版本映射版本名；空值显示「（空）」）；描述字段不做逐字 diff，按 from/to 空否显示 `by 填写/清空/修改了需求说明`。

---

## 4. Host RPC 接口（新增 3 个）

| 方法 | 入参 | 出参/副作用 |
|---|---|---|
| `wb.createRequirement` | `{ title, description, category, versionId, priority, status, owner }` | `title` 必填；`id = 'req-' + (maxSeq+1)` 补零（`req-007` 起）；默认 category=需求任务/版本=v-backlog/P2/待启动/Leader；`history=[]` → `{ ok, error, reqList }` |
| `wb.updateRequirement` | `{ reqId, ...7 字段 }` | 7 字段 diff → 有变化追加 `history` → 刷新 `updatedAt` → `{ ok, error, reqList }` |
| `wb.deleteRequirement` | `{ reqId }` | 移除列表项（不删目录）→ `{ ok, error, reqList }` |

> 三个写接口均走 `writeJson('requirements.json', list)` + `writePolicy()`（`{mode:'workspace-write', workspaceRoot: PROJECT}`）；返回值纯 JSON（lossless 铁律，缺值用 `null`/`[]`）。

---

## 5. Client 组件与状态（React，纯 createElement）

### 5.1 新增 state（全部前置到组件顶部，hooks 铁律）
| state | 初值 | 含义 |
|---|---|---|
| `form` | `null` | 表单弹窗：`{mode:'new'\|'edit', reqId?, title, description, category, versionId, priority, status, owner}` |
| `delReq` | `null` | 删除确认：`{reqId, title}` |
| `hoverId` | `null` | 卡片/列表行 hover 的 id（控制 ✎🗑 显示） |

### 5.2 关键函数
- `openNewForm()`：新建默认值表单；`openEditForm(r)`：预填当前值；
- `saveForm()`：title 空拦截 → `create` 或 `update` → `setReqList` + 关弹窗；**编辑态且在详情页**时重拉 `getRequirement` 刷新 `req`；
- `askDelete(r)` / `confirmDelete()`：删除确认 → 成功后刷新列表；**删除的是当前详情需求**则 `backToBoard()`；
- `renderMd(text)`：详情描述轻量 md 渲染（段落/`#`标题/```代码块```/`-`列表/粗体/行内代码/`[链接](url)`），`renderInline` 扩展链接分支（`\[[^\]\n]+\]\([^)\n]+\)`），独立函数不改阅读器。

### 5.3 弹窗层级
- `formModal` zIndex 200（表单）；`delModal` zIndex 210（删除确认盖在表单上）；阅读器 `viewer` zIndex 100（表单可盖阅读器，可接受）；
- 弹窗挂到 `view==='board'` 与 detail 两个 return 的末尾（`formModal, delModal`）。

### 5.4 需求信息面板 + 描述折叠（详情页）
- **`infoPanel`**（v1.1）：详情页标题下方、「需求描述」上方的 key-value 网格卡片，展示需求本身属性：负责人 / 版本（`verNameOf(versionId)`）/ 优先级（`priColor` 彩色徽标）/ 状态（`statusColor` 彩色徽标）/ 类别 / 创建时间 / 更新时间（时间 `String(at).replace('T',' ').slice(0,16)` 截断）；缺字段显示 `—`。
- **`descPanel` 折叠**（v1.2，v1.3 改行数+高度）：`descOpen` state（组件顶部声明）；`descLines = description.split('\n').length`；`needFold = descLines > 4`；折叠态 `maxHeight:150, overflow:hidden`，底部「展开全部 ▾ / 收起 ▴」切换；`openDetail` 时 `setDescOpen(false)` 重置。≤4 行不折叠。段落容器（flushPara）`whiteSpace:'pre-wrap'`（pkg-17）保证多行换行。
- 与 `descPanel` 一起挂在 detail 与 missing 两个 return 的 `detailTop` 之后；`req.missing` 时 `req.requirement` 为 meta（reqList 中该条），信息面板照常渲染（无流程也能看需求属性）。
- 数据来源：`req.requirement`（`getRequirement` 返回的 requirement 或 openDetail 失败时的 reqList meta），无新 RPC。

---

## 6. 已确认决策（User 回复汇总）

1. 编辑入口：卡片/列表行 hover ✎ + 详情页「编辑」按钮；**编辑带审计**（记录改了什么）。
2. 流程树：**不扩展**，新建需求保持「暂无流程数据」（写死模板）。
3. 删除：二次确认；不删磁盘目录。
4. owner：保留 + 角色 datalist（User/Leader/Investigator/Architect/RD/PM/FE/QA）；按需求类型指定负责人（架构→Architect、纯后端→RD、纯前端→FE、审批/讨论→直接指定）。
5. 新建保存后：留在看板刷新列表。
6. 描述：支持 Markdown（含代码块渲染）。
7. agent 工具注册：**后置**（先 UI，工具后续补）。

---

## 7. 实现记录与经验

- pkg-8（单包完成全部 M4）；`wbflow-1` 当前 pkg 演进：pkg-4 看板 → pkg-5 null 守卫 → pkg-6 优先级 → pkg-7 分组列表 → **pkg-8 M4 CRUD**。
- 落盘源码：`workbench-plugin/host.js` / `client.js` 与 pkg-8 对齐（重启可从源码恢复）。
- **流程教训（本文补记原因）**：M4 开发前未单独产出技术设计文档（PRD v0.2 定稿后直接实现）。后续里程碑严格遵守：PRD → 技术设计 → 原型 → 实现 → 回写。

---

## 8. 后置

### 8.1 agent 接口规划（✅ 已实现，pkg-24）

把全部 `wb.*` 写/读接口注册为 agent 可调用模型工具（Host `harness.defineTool` + `harness.registerTool(ctx, def)`，`Tool.listTools` 可见），agent 与 UI 走同一套审计/门禁/状态机。**实现**：16 个 `wb.*` handler 提取为 `op*` 领域函数，RPC 与 16 个 `wb_*` 模型工具共用；`ctx.effect` 包裹注册（Fiber 生命周期，停止/更新自动注销）；破坏性工具（delete/deprecate/override）描述标注「需 User 明确指令」。工具清单：

| agent 工具 | 对应 RPC | 入参 | 出参 | 业务语义 |
|---|---|---|---|---|
| `wb_list_requirements` | `wb.listRequirements` | — | 需求池列表 | 拆分/筛选前置 |
| `wb_list_versions` | `wb.listVersions` | — | 版本列表 | 拆分/筛选前置 |
| `wb_get_requirement` | `wb.getRequirement` | `{reqId}` | 聚合快照或 null | 读详情/流程/评论 |
| `wb_get_artifact_content` | `wb.getArtifactContent` | `{reqId, artifactId}` | `{art, content}` 或 null | 读产物正文 |
| `wb_create_requirement` | `wb.createRequirement` | `{title, description?, category?, versionId?, priority?, status?, owner?}` | `{ok, reqList}` | 建需求（默认值同表单；`history=[]` 审计起点） |
| `wb_update_requirement` | `wb.updateRequirement` | `{reqId, ...7 字段}` | `{ok, reqList}` | 改需求（**Host 算 diff 自动写审计**） |
| `wb_delete_requirement` | `wb.deleteRequirement` | `{reqId}` | `{ok, reqList}` | 删需求（不删磁盘目录）⚠️ 需 User 指令 |
| `wb_create_version` | `wb.createVersion` | `{name}` | `{ok, verList}` | 建版本（待启动） |
| `wb_update_version` | `wb.updateVersion` | `{versionId, name}` | `{ok, verList}` | 改版本名 |
| `wb_advance_version` | `wb.advanceVersion` | `{versionId}` | `{ok, verList, unfinished}` | 推进版本（待启动→进行中→已发布）；发布提示 unfinished |
| `wb_deprecate_version` | `wb.deprecateVersion` | `{versionId}` | `{ok, verList}` | 废弃版本（有需求引用 blocked）⚠️ 需 User 指令 |
| `wb_comment` / `wb_reply` / `wb_resolve` / `wb_override` | 同 M2 §5.1 | — | `{ok, snapshot}` | 评论四件套（override ⚠️ 需 User 指令） |
| `wb_decide` | `wb.decide` | `{reqId, nodeId, verdict, reason?, newScope?}` | `{ok, snapshot}` | 门禁裁决（未全 resolved 时 blocked） |

> 场景示例："把 v2.0 拆成 5 条需求" → `wb_list_versions` 确认 v2.0 → 循环 `wb_create_requirement`（每条带描述 md、按 owner 规则指定负责人）→ 每条自动带审计起点。后续 agent 推进状态/节点用 `wb_update_requirement` / `wb_decide`（m2-flow-tree-tech.md §5.1）。

### 8.2 其他后置

1. 版本管理 UI（M5）、真实流程树生成（M7）、拖拽换列、多项目切换。
