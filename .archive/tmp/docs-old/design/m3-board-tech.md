# M3 技术设计（实现回写）：版本看板 + 需求池 + 优先级 + 分组列表

> 里程碑：M3
> 状态：**已实现并验证**（pkg-4 ~ pkg-7，currentPackageId=pkg-7）
> 前置：产品文档 `m3-board.md`（v1.0，方案 B）；技术文档 `m2-flow-tree-tech.md`（内层流程树复用）
> 运行平台：DeepSeek Harness（DSH），动态 Cordis 插件 `wbflow-1`（Host + Client 两半）

---

## 1. 目标与非目标

### 1.1 目标
在「项目」tab 落地**外层入口**，形成 `外层（看板/列表）⇄ 内层（流程树）` 两级结构：
- 版本看板：`<select>` 版本筛选 + 状态泳道（待启动/进行中/已完成/已取消）
- 需求池：看板 / 列表双视图切换；列表按**状态分组**、**公共表头一次**、组间虚线分割
- 需求优先级：P0/P1/P2/P3 四档属性，看板卡/列表行/详情页三处展示，与版本筛选**组合生效**
- 点击需求 → 进入内层 M2 流程树（复用），顶部「← 返回看板」

### 1.2 非目标（后置）
- 创建/编辑/删除需求入口（只读 mock）
- 版本管理 UI（建版本、拖拽入版本）
- 项目切换（数据层已按 `projectId` 分桶）
- 真实子 agent 编排（M7）；筛选交互不做服务端分页

---

## 2. 架构分层

沿用四层架构，M3 落地**数据层 + 交互层**（外层部分）。Host 持有全部领域读写，Client 只经 `host.call`。

```
Client（浏览器）                          Host（Node 进程）
────────────────                          ────────────────
conversation.view tab id="workbench"
└─ WorkbenchView（根，两级视图状态）
   ├─ view='board'（外层）
   │   ├─ Topbar（标题/workspace/版本select/优先级select/看板·列表切换）
   │   ├─ kind='board' → 泳道看板（4 列 = 状态）
   │   └─ kind='list'  → 公共表头 + 按状态分组 + 虚线分割
   └─ view='detail'（内层，复用 M2 组件树）
       ├─ DetailTop（←返回看板 + 标题 + 类别 + 优先级 + 状态）
       └─ StepList / NodeDetail / ArtifactViewer（M2）
             │                                  │
             └────────── host.call ◄────────────┘
                          │
                          ▼
   harness.handle RPC:  listRequirements / listVersions / getRequirement /
                        getArtifactContent / comment / reply / resolve /
                        override / decide
   （fs + JSON 读写 .workbench/，writePolicy 固定 workspaceRoot=PROJECT）
```

- **数据订阅**：进入 tab 时 `listRequirements` + `listVersions` 拉列表；进入详情时 `getRequirement` 拉全量快照；每次写操作返回最新快照驱动重渲染。无轮询/订阅推送（数据量小，优化后置）。

---

## 3. 数据存储（文件 Schema）

根：`<workspace>/.workbench/`。M3 新增/变更集中在**需求列表级**字段。

```
<ws>/.workbench/
  versions.json       # M3 新增：版本列表
  requirements.json   # M3 扩展：每条 + versionId + priority
  <reqId>/            # M2 既有（flow/artifacts/comments），M3 不改结构
    flow.json
    artifacts.json
    comments.json
```

### 3.1 versions.json（M3 新增）
```json
[
  { "id": "v-1",       "name": "v1.0" },
  { "id": "v-11",      "name": "v1.1" },
  { "id": "v-backlog", "name": "待分配", "isBacklog": true }
]
```
> `id` 为稳定短码（列表/需求引用），`name` 为展示名；`isBacklog=true` 表示需求池（未排期）。「全部」是前端虚拟选项（`__all__`），不入库。

### 3.2 requirements.json（M3 扩展两字段）
```json
[
  {
    "id": "req-001", "projectId": "/Users/.../test-flow",
    "title": "盘前简报模块", "category": "需求任务",
    "versionId": "v-1", "priority": "P0",
    "status": "进行中", "owner": "Leader",
    "createdAt": "…", "updatedAt": "…"
  }
]
```

| 字段 | 类型 | 取值 | 说明 |
|---|---|---|---|
| `versionId` | string | 版本 `id` 或 `v-backlog` | 关联 `versions.json`；`v-backlog`=未排期 |
| `priority` | string | `P0`/`P1`/`P2`/`P3` | 四档优先级（已确认），缺省渲染 `—` |
| `status` | string | `待启动`/`进行中`/`已完成`/`已取消` | 复用既有字段，决定泳道列与列表分组 |

> 展示版本名：`versionId → versions.json.name` 映射，查不到回退 `versionId || '待分配'`。

---

## 4. 前端视图状态（WorkbenchView）

所有 `useState` 在组件顶部一次性声明（React hooks 顺序铁律），关键状态：

| state | 初值 | 含义 |
|---|---|---|
| `view` | `'board'` | `board`（外层）\| `detail`（内层流程树） |
| `kind` | `'board'` | 外层内：`board`（泳道）\| `list`（分组列表） |
| `verFilter` | `'__all__'` | 版本筛选（`__all__`=全部，或版本 id） |
| `priFilter` | `'__all__'` | 优先级筛选（`__all__`=全部，或 P0~P3） |
| `reqList` | `[]` | `wb.listRequirements` 结果 |
| `verList` | `[]` | `wb.listVersions` 结果 |
| `req` | `null` | 详情快照（`getRequirement`）；`{missing:true}`=无流程 |
| `sel` | `''` | 当前流程节点 id（M2 沿用） |

**筛选组合**：`filteredReqs = reqList.filter(r => (verFilter==='__all__' || r.versionId===verFilter) && (priFilter==='__all__' || r.priority===priFilter))`——两个维度取**交集**，看板与列表共用同一过滤结果。

**视图切换**：`view==='board'` 时渲染外层；点击需求卡/行 → `openDetail(id)` 拉快照并 `view='detail'`；详情顶部「← 返回看板」→ `backToBoard()` 复位。若 `getRequirement` 返回 `null`（该需求无 flow.json），置 `req={missing:true}` 渲染「暂无流程数据」+ 返回按钮（**不卡加载中**，见 §7 错误处理）。

---

## 5. 外层两种视图的渲染

### 5.1 泳道看板（kind='board'，默认）
- 横向 flex 4 列，列=状态：`待启动 | 进行中 | 已完成 | 已取消`，列宽 `flex:1, minWidth:180`，超宽横向滚动。
- 列头：状态色圆点 + 状态名 + 该列条数（`filteredReqs` 内计数）。
- 需求卡：标题 + 类别徽标（蓝）+ **优先级徽标（P0 红/P1 橙/P2 蓝/P3 灰）** + 版本名 + 负责人；卡片 onClick → `openDetail`。
- 空列显示空容器（列头计数 0）。

### 5.2 分组列表（kind='list'）
- **公共表头渲染一次**（flex 行）：标题/类别/优先级/版本/状态/负责人，列宽 `colFlex=[3,1,0.7,1,1,1]`。
- 按状态顺序分组：`groups = statusCols.map(s => ({status:s, items: filteredReqs.filter(r=>r.status===s)})).filter(g=>g.items.length>0)`——**空状态组不渲染**。
- 每组 = 状态标题行（色点 + 状态名 + `n 项`）+ 若干数据行（flex 行，复用表头列宽）；组间用**虚线分割** `borderTop: 1px dashed #3a4356`（语义分隔，非固定字符）。
- 数据行 onClick → `openDetail`；行分隔 `borderBottom: 1px solid #1e2230`。

> 两组视图共用 `filteredReqs` 与 `verNameOf`/`priColor` 映射，保证筛选/展示一致。

---

## 6. Host RPC 接口（harness.handle，纯 JSON）

M3 新增 2 个列表接口，其余沿用 M2（签名不变，`loadRequirement` 有加固）。

| 方法 | 入参 | 出参/副作用 |
|---|---|---|
| `wb.listRequirements` | — | `requirements.json` 数组（含 `versionId`/`priority`），非数组返回 `[]` |
| `wb.listVersions` | — | `versions.json` 数组 |
| `wb.getRequirement` | `{ reqId }` | 聚合快照 `{requirement, flow, artifacts, comments}`；**flow 缺失 → `null`** |
| `wb.getArtifactContent` | `{ reqId, artifactId }` | `{art, content}` 或 `null` |
| `wb.comment` / `wb.reply` / `wb.resolve` / `wb.override` | M2 同 | 写后返回 `{ok, snapshot}` 最新快照 |
| `wb.decide` | M2 同 | 通过（校验全评论 resolved）/打回，返回快照 |

> **lossless-JSON 铁律**：所有出参必须是纯 JSON；`undefined` 字段会被 guard 拦截（§7 的线上故障即由此引发）。
> **写策略**：`fs.writeText(..., writePolicy())`，`writePolicy()` 显式返回 `{mode:'workspace-write', workspaceRoot: PROJECT}`，规避沙箱默认解析到 DSH 安装目录导致 `file access denied`。

### 6.1 agent 接口规划（后置，M4 起随写接口一并注册）

目标：让 **agent（本会话模型）与未来子 agent 可直接操作工作台数据**——如"基于看板推进任务、拆分大版本自动创建需求、澄清需求后发评论"，且**与 UI 走同一套审计/门禁/状态机逻辑**。注册方式：Host 用 `harness` 注册模型工具（`Tool.listTools` 可见），执行路径 = 复用 §6 的 `wb.*` handler。

| agent 工具 | 对应 RPC | 入参 | 出参 | 业务语义 |
|---|---|---|---|---|
| `wb_list_requirements` | `wb.listRequirements` | — | `[{id,title,category,versionId,priority,status,owner,description}]` | 看板/需求池全量；agent 据此判断任务推进或拆分 |
| `wb_list_versions` | `wb.listVersions` | — | `[{id,name,isBacklog}]` | 版本清单（拆分大版本时选目标版本） |
| `wb_get_requirement` | `wb.getRequirement` | `{reqId}` | 聚合快照或 `null` | 读详情/流程树/产物/评论 |

> 工具名采用 `wb_*` 蛇形（避免与内置工具冲突）；`execute` 直接调用 §6 既有 handler 的领域逻辑。
> 场景示例：agent 收到"把 v2.0 拆成 5 条需求"→ `wb_list_versions` 确认 v2.0 存在 → 循环 `wb_create_requirement`（见 m4-req-crud-tech.md §8）→ 每条自动带 `history=[]` 审计起点。

---

## 7. 线上故障与修复记录（经验沉淀）

| 故障 | 根因 | 修复（pkg） |
|---|---|---|
| Client 调 `wb.getRequirement` 报 `result.flow must be lossless JSON` | 只有 req-001 有 `flow.json`；其余需求 `readJson` 返回 `undefined`，快照 `flow: undefined` 触发 guard | pkg-5：`loadRequirement` 在 flow 缺失/`__err` 时整体返回 `null`；Client `openDetail` 对 `null` 渲染「暂无流程数据」+ 返回按钮 |
| 点非 req-001 卡片后一直「加载中」 | 上述 null 快照在前端 `if(!req)` 分支死等 | pkg-5 同步修复（`req.missing` 早退分支） |

**通用规则（M3 起强制执行）**：
1. `cordis_define` **必须同时传 `code.host` 与 `code.client`**（曾多次只传一半导致半包）。
2. Host 返回**绝不出现 `undefined` 字段**——缺值用 `null`/`[]` 归一。
3. 前端 hooks 全部前置到组件顶部，之后才允许条件 return。
4. 循环内 onClick 闭包用块级 `const`（`while` 内 `var` 共享函数作用域会全指向最后一行）。

---

## 8. 已实现确认的决策

1. 方案 B：顶部 `<select>` 版本筛选 + 状态泳道看板 + 看板/列表切换。
2. 版本筛选用 `<select>`（不用 pills）。
3. **优先级四档 P0/P1/P2/P3**（User 选定），属性 + 筛选 + 三处展示（卡/行/详情）。
4. 列表视图：**按状态分组、公共表头一次、组间虚线分割、空组隐藏**（User 指定）。
5. 版本与优先级筛选**组合生效**（交集）。
6. 数据只读 mock（无创建入口）；`versions.json` 用 `v-1`/`v-11`/`v-backlog` 短码。

---

## 9. 实现记录（pkg 演进）

| pkg | 内容 | 结果 |
|---|---|---|
| pkg-4 | 看板+列表+版本筛选+详情复用（首版） | 有语法问题两次修复后通过 |
| pkg-5 | `loadRequirement` null 守卫（lossless） | 修复线上报错 |
| pkg-6 | 优先级 P0-P3 属性+筛选+三处展示 | 通过 |
| pkg-7 | 列表分组（公共表头+虚线分割） | 通过（current） |

插件重启后动态包不持久——`workbench-plugin/` 落盘源码需与 pkg-7 对齐（待同步）。

---

## 10. 后续扩展点（不属本次实现）

1. **需求池增删改**（M4）：新建需求入口、改 versionId/priority/status 的写接口 + 表单。
2. **版本管理**（M5）：版本 CRUD、需求拖拽入版本、版本进度汇总。
3. **项目切换**：多 workspace 时 Topbar 加项目选择（数据层已按 `projectId` 分桶，仅需 UI）。
4. **拖拽换列**：看板卡片拖拽改 `status`（当前仅点击进详情）。
