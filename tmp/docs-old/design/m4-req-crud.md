# M4 PRD：需求池增删改（CRUD）+ 描述 + 编辑审计

> 里程碑：M4
> 状态：**v1.4（已实现并验证，pkg-23；v1.1 补需求信息展示，v1.2 补描述折叠，v1.3 折叠改行数阈值+高度 150px，v1.4 变更历史改自然语句格式）**
> 依赖：M3 看板/列表/筛选/优先级；可视化参考 `docs/prototype/m4-req-crud-prototype.html`
> 实现载体：`wbflow-1` 插件（pkg-8 起，Host 加写接口 + Client 加表单弹窗）

---

## 1. 目标

把 M3 的**只读**需求池变为**可维护**：在「项目」tab 支持
- **新建需求**（弹窗表单）
- **编辑需求**（弹窗表单，预填当前值，**带审计**：记录每次改了什么）
- **删除需求**（二次确认）
- **需求描述**（Markdown，Leader / 后续 PM 等理解要做什么）
- 改完即刷新看板/列表（无需手动刷新）

**不包含**（后置）：版本管理 UI（M5）、流程树生成（**确认：暂不支持扩展流程，新建需求无流程树、显示「暂无流程数据」**）、拖拽换列、多项目切换。

---

## 2. 数据模型

沿用 `requirements.json`（单文件数组），**新增 `description` 与 `history` 两字段**，其余字段不变，仅新增写操作。

| 字段 | 编辑表单控件 | 说明 |
|---|---|---|
| `title` | 文本输入（必填） | 标题 |
| `description` | **多行 textarea（支持 Markdown，含代码块）** | 需求描述：让 Leader / 后续 PM 等明白要做什么；可写 md（列表/代码块）或贴文档/链接 |
| `category` | 下拉（需求任务/规划审批任务/版本规划任务） | 复用三类 |
| `versionId` | 下拉（v1.0/v1.1/待分配） | 版本 |
| `priority` | 下拉（P0/P1/P2/P3） | 优先级 |
| `status` | 下拉（待启动/进行中/已完成/已取消） | 状态 |
| `owner` | 文本输入 + 角色 datalist 建议 | 负责人（按需求类型：架构需求→Architect；纯后端→RD；纯前端→FE；审批/讨论/简单 task→直接指定 Leader/PM 等） |
| `history` | （只读展示，不在表单编辑） | **编辑审计**：每次保存追加一条 `{ at, by, changes:[{field, from, to}] }` |
| `createdAt` / `updatedAt` | 自动 | 新建=now；编辑刷新 `updatedAt` |

`id` 生成：`req-` + 递增序号（取现有最大序号 +1，如 `req-007`）。

### 2.1 编辑审计（history）

- **每次编辑保存**时计算字段级 diff（title/description/category/versionId/priority/status/owner 七字段），`from !== to` 的字段记入 `changes: [{ field, from, to }]`；
- 追加 `{ at: ISO时间, by: 'User'（当前操作者） }` 到该需求 `history[]`；
- **无任何字段变化时不追加**（避免空审计）；
- 删除需求不写审计（整条移除）；新建需求的 history 为空数组。

### 2.2 需求信息 + 描述展示（详情页）

详情页标题下方依次展示两个卡片：

**「需求信息」卡片**（需求本身的属性，一眼可读）：
- 负责人（owner）· 版本（versionId → 版本名）· 优先级（P0-P3 彩色徽标）· 状态（待启动/进行中/已完成/已取消 彩色徽标）· 类别 · 创建时间 · 更新时间
- 以 key-value 网格展示；缺字段显示 `—`。

**「需求描述」卡片**：
- 用**轻量 Markdown 渲染**展示 `description`（复用阅读器 md 能力：`**粗体**` / `` `行内代码` `` / `#`~`######` 标题 / ` ``` ` 代码块 / 列表 / 链接 / 空行分段）；
- 未填写时显示「暂无描述」占位（灰色）；
- **超长折叠（pkg-21）**：描述 **≤ 4 行完整显示无折叠；> 4 行折叠**（默认高度 **150px**，底部「展开全部 ▾」→ 点击展开完整内容变「收起 ▴」）；切换需求时重置为折叠；段落容器 `whiteSpace:pre-wrap` 保证多行正确换行；
- 看板卡片/列表行**不**展示描述（保持卡片简洁，进详情看）。

### 2.3 变更历史展示（详情页）

- 详情页加 **「变更历史」面板**（可折叠，放在描述下方或右侧列），按时间倒序展示每次，**自然语句格式（pkg-23）**：`谁将「字段」从「旧值」变更为「新值」`（如 `User 将「版本」从「v1.0」变更为「v1.1」`、`User 将「优先级」从「P2」变更为「P1」`）；描述字段不做逐字 diff，显示 `User 填写/清空/修改了需求说明`；
- 无历史时显示「暂无变更记录」。

---

## 3. 页面与交互

### 3.1 入口（顶栏右侧，看板/列表共用）
- 在版本/优先级筛选右侧加 **「+ 新建需求」** 按钮（绿色）。

### 3.2 新建 / 编辑弹窗（模态框，风格同阅读器弹窗）
- 字段见 §2；`title` 必填（空则按钮禁用/红框提示）；
- **描述用多行 textarea**（placeholder：`支持 Markdown，可写背景/目标/验收点，或贴文档链接…`），高度 3~4 行可拖拽；
- 「保存」→ 写 `requirements.json` → 刷新 `reqList` → 关弹窗；
- 新建默认值：`category=需求任务`、`versionId=待分配`、`priority=P2`、`status=待启动`、`owner=Leader`、`description=''`。

### 3.3 编辑入口
- **卡片 hover 出现「✎ 编辑」按钮**（点击编辑弹窗，`stopPropagation` 不拦截点击进详情）；
- 详情页标题区加「编辑」按钮（进详情后也能改）；
- 保存后：看板/列表即时刷新，`updatedAt` 与 `history` 自动更新。

### 3.4 删除入口
- 编辑弹窗底部「删除」（红色，危险区）；
- 点击后弹**二次确认**（「删除后不可恢复，确定删除需求「xxx」？」）——已确认；
- 确认 → 从数组移除 → 刷新；**不删** `reqId/` 目录（保留磁盘数据）。

### 3.5 新建需求的流程树
- **确认：暂不支持扩展流程**——新建需求无 `flow.json`，点进详情显示「暂无流程数据」（与 req-002~006 一致）；流程树仍为写死的 mock 模板。

### 3.6 owner 语义（已确认）
- 保留 owner 字段 + 角色 datalist 建议（User/Leader/Investigator/Architect/RD/PM/FE/QA）；
- 填写规则（提示给用户，不做强校验）：架构需求→Architect；纯后端改动→RD；纯前端改动→FE；审批/讨论/简单 task→直接指定。

---

## 4. 已确认决策（User 回复）

1. 编辑入口：卡片 hover ✎ + 详情页「编辑」按钮（**编辑带审计**，记录改了什么）。
2. 流程树：**不扩展**，新建需求保持「暂无流程数据」。
3. 删除：**二次确认**。
4. owner：保留，角色 datalist 建议；按需求类型指定负责人。
5. 新建保存后：留在看板刷新列表。
6. 描述：支持 Markdown（含代码块渲染）。

---

## 5. Host 写接口（新增）

| 方法 | 入参 | 出参/副作用 |
|---|---|---|
| `wb.createRequirement` | `{ title, description, category, versionId, priority, status, owner }` | 追加需求（history=[]）→ `{ ok, reqList }` |
| `wb.updateRequirement` | `{ reqId, ...可改字段 }` | 计算 diff → 追加 `history` → `{ ok, reqList }`（无变化则仅刷新 updatedAt？**实现：无变化不追加 history，但 updatedAt 仍刷新**） |
| `wb.deleteRequirement` | `{ reqId }` | 移除 → `{ ok, reqList }` |

> 全部走 `writeJson('requirements.json', ...)` + `writePolicy()`（workspaceRoot=PROJECT）；返回值纯 JSON（lossless 铁律）。
> 编辑时由 Host 计算字段级 diff（防 Client 伪造审计）；`by` 固定 `'User'`（当前无登录体系）。

---

## 6. 实现记录（pkg-8，已实现）

- **Host 新增**：`wb.createRequirement`（title 必填；id 自动 `req-`+递增序号补零；默认 category=需求任务/版本=待分配/P2/待启动/Leader；`history=[]`）、`wb.updateRequirement`（7 字段 diff → 有变化才追加 `{at, by:'User', changes:[{field,from,to}]}` → 刷新 updatedAt）、`wb.deleteRequirement`（移除列表项，不删 `reqId/` 目录）。
- **Client 新增**：
  - 顶栏「＋ 新建需求」按钮（看板/列表共用）；
  - 新建/编辑弹窗（标题* / 描述 textarea / 类别 / 版本 / 优先级 / 状态 / 负责人+datalist 角色建议）；
  - 卡片与列表行 **hover 出现 ✎ 编辑 / 🗑 删除**（`stopPropagation` 不拦截进详情）；
  - 删除二次确认弹窗（`删除后不可恢复，确定删除需求「xxx」？`）；
  - 详情页：**需求描述卡片**（md 渲染：标题/粗体/行内代码/代码块/列表/链接，未填显示「暂无描述」）+ **编辑按钮** + **变更历史面板**（倒序，自然语句 `谁将「字段」从「旧」变更为「新」`，版本映射为版本名、描述显示「谁填写/清空/修改了需求说明」）；
  - 编辑保存后：看板列表刷新；若在详情页编辑则重拉快照；删除当前详情需求自动返回看板。
- **描述 md 渲染**：`renderMd()` 复用阅读器轻量解析（`renderInline` 扩展链接 `[t](url)`），独立函数，不改动阅读器。
- 新建需求无流程树 → 详情显示「暂无流程数据」+ 返回按钮（同 req-002~006）。

## 7. 待办（后置）

- **agent 工具注册**（User 确认后做）：把 `createRequirement`/`updateRequirement`/`deleteRequirement`（及既有 `listRequirements`/`decide` 等）注册为 agent 可调用工具，支持「基于看板推进任务 / 拆分大版本自动创建需求」，走同一套审计与状态机逻辑。
- 版本管理 UI（M5）、真实流程树生成（M7）、拖拽换列、多项目切换。
