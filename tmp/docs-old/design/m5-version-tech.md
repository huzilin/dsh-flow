# M5 技术设计：版本管理（CRUD + 状态机 + 进度汇总）

> 里程碑：M5
> 状态：Draft v0.1（待实现，pkg-9）
> 前置：产品文档 `m5-version.md`（v1.0，决策已确认）
> 实现载体：`wbflow-1` 插件（pkg-9，Host 加版本写接口 + Client 加版本管理弹窗）

---

## 1. 目标与非目标

### 1.1 目标
- 版本 CRUD：新建 / 改名 / 删除（有需求引用禁止删）
- 版本状态机：`待启动 → 进行中 → 已发布` 单向推进；**任意阶段可直接废弃**；废弃联动其下需求全部转 `已取消`；**v1 废弃不可回滚**
- 版本进度汇总：各版本需求数 + 四状态分布 + 进度条
- 需求归入版本：沿用 M4 编辑表单下拉（不加快捷入口）

### 1.2 非目标（后置）
- 版本恢复/回滚（v1 废弃即终态；恢复按需求审计 `history` 回滚 → 后置）
- 迭代排期/燃尽图、批量迁移拖拽、真实子 agent 拆版（M7）

---

## 2. 数据存储（文件 Schema）

### 2.1 versions.json（每条 + `state`）
```json
[
  { "id": "v-1", "name": "v1.0", "state": "进行中", "isBacklog": false },
  { "id": "v-11", "name": "v1.1", "state": "待启动", "isBacklog": false },
  { "id": "v-backlog", "name": "待分配", "state": "待启动", "isBacklog": true }
]
```

### 2.2 requirements.json（不变，仅 `status` 被废弃联动改写）
废弃联动：`status: 任意 → 已取消`；`updatedAt` 刷新；**不写** `history`（联动非人工编辑，审计后置）。

---

## 3. 版本状态机（精确转移表）

状态常量：`S = ['待启动', '进行中', '已发布', '已废弃']`。

| 动作 | 前置 | 后置 | 副作用 |
|---|---|---|---|
| `advance` | state ∈ {待启动, 进行中} | 前进一格（待启动→进行中，进行中→已发布） | 发布时若无未完成需求可附提示（不阻止） |
| `advance` | state ∈ {已发布, 已废弃} | 拒绝 | 返回 `{ok:false, error}` |
| `advance` | `v-backlog` | 拒绝 | 待分配池无状态操作 |
| `deprecate` | state ≠ 已废弃 | 已废弃 | **其下所有需求 status → 已取消** |
| `deprecate` | state = 已废弃 | 拒绝（幂等） | 返回 `{ok:false, error:'already deprecated'}` |

> `v-backlog` 的 `state` 恒为 `待启动` 且不参与任何转移；UI 锁定不显示操作。

---

## 4. Host RPC 接口（新增 5 个）

| 方法 | 入参 | 出参/副作用 |
|---|---|---|
| `wb.createVersion` | `{ name }` | name 必填（trim 后非空）；`id='v-'+(maxSeq+1)`；state=待启动；isBacklog=false → `{ok, error, verList}` |
| `wb.updateVersion` | `{ versionId, name }` | `v-backlog` 拒绝；name trim 非空；改 name → `{ok, error, verList}` |
| `wb.advanceVersion` | `{ versionId }` | 按 §3 转移；发布时若版本下有未完成需求返回附加字段 `unfinished: n`（提示用，不阻止）→ `{ok, error, verList}` |
| `wb.deprecateVersion` | `{ versionId }` | 二次确认在 Client；Host：state→已废弃 + 其下需求 status→已取消 + updatedAt 刷新 → `{ok, error, verList, reqList, depCount}`（depCount=被取消需求数） |
| `wb.deleteVersion` | `{ versionId }` | 有需求引用 → `{ok:false, blocked:true, refCount:n, error}`；无引用 → 删除 → `{ok, verList}` |

> **写文件**：`writeJson('versions.json', ...)` / `writeJson('requirements.json', ...)` + `writePolicy()`。
> **deprecateVersion 双写**：先写 versions（state=已废弃），再写 requirements（需求取消）。任一失败返回 error（mock 不引入事务；若 versions 已写成功而 requirements 失败，下次 deprecate 幂等返回 error 且 state 已是已废弃——可接受，人工可调）。
> **lossless 铁律**：无 `undefined` 字段；`verList`/`reqList` 恒为数组。

---

## 5. Client 组件与状态

### 5.1 新增 state（顶部）
| state | 初值 | 含义 |
|---|---|---|
| `verMgrOpen` | `false` | 版本管理弹窗开关 |
| `verNameInput` | `''` | 新建版本名输入 |
| `verDelTarget` | `null` | 删除确认 `{id, name, refCount}` |
| `verDepTarget` | `null` | 废弃确认 `{id, name, depCount}` |
| `verMsg` | `''` | 弹窗内错误/提示（toast 或行内） |

### 5.2 版本行渲染
- 徽标：`st-待启动` 灰 / `st-进行中` 橙 / `st-已发布` 绿 / `st-已废弃` 红；
- 进度：统计该版本下需求 `{待启动,进行中,已完成,已取消}` 计数 + 分段进度条（四色段宽按占比）；
- 操作（`v-backlog` 除外）：`▶ 推进`（state∈{待启动,进行中} 显示）/ `✖ 废弃`（state≠已废弃 显示）/ `✎ 改名` / `🗑 删除`；
- 已废弃：行灰显 + 「已废弃 · v1 不可恢复」提示。

### 5.3 交互流
- **新建**：输入 name → `wb.createVersion` → 成功刷新 `verList`（顶栏下拉同步）+ 清空输入；
- **推进**：`wb.advanceVersion` → 若返回 `unfinished>0` 且目标是发布 → 行内提示「还有 n 条未完成需求（bugfix 场景可发布）」；成功刷新；
- **废弃**：弹二次确认「废弃「vX」？其下 n 条需求将全部转为已取消，v1 不支持恢复」→ 确认 → `wb.deprecateVersion` → 刷新 `verList` + `reqList`（看板/列表同步）+ 提示 depCount；
- **改名**：行内 prompt（自由文本）→ `wb.updateVersion` → 刷新；
- **删除**：`wb.deleteVersion` → blocked 时 toast 提示 refCount；成功刷新。

### 5.4 顶栏
- 版本筛选下拉旁加「版本管理」按钮（btnSmall 中性色）；`verMgrOpen` 时渲染弹窗（zIndex 200，与表单弹窗同级）。

---

## 6. agent 接口规划（本里程碑扩展）

在 M4 规划的 `wb_*` 工具集基础上新增：

| agent 工具 | 对应 RPC | 入参 | 业务语义 |
|---|---|---|---|
| `wb_create_version` | `wb.createVersion` | `{name}` | 建版本（待启动） |
| `wb_update_version` | `wb.updateVersion` | `{versionId, name}` | 改版本名 |
| `wb_advance_version` | `wb.advanceVersion` | `{versionId}` | 推进版本（待启动→进行中→已发布）；发布提示 unfinished |
| `wb_deprecate_version` | `wb.deprecateVersion` | `{versionId}` | **废弃版本 + 联动需求全部取消**（破坏性，agent 需 User 确认后调用） |
| `wb_delete_version` | `wb.deleteVersion` | `{versionId}` | 删版本（有引用 blocked） |

> 场景：agent 推进版本生命周期「把 v1.1 推进到已发布」→ `wb_advance_version`；「废弃 v1.0」→ 先确认需求归属 → `wb_deprecate_version`。
> **破坏性工具保护**：`wb_deprecate_version` / `wb_delete_version` 属高风险，注册时可在描述中标注需 User 明确指令；v1 不做权限系统。

---

## 7. 已确认决策（对应 m5-version.md §4）

1. 状态机：单向推进 + 任意阶段直接废弃；废弃联动需求→已取消；废弃二次确认；**v1 不可回滚**（恢复按审计回滚后置）。
2. 发布约束：不阻止（bugfix 场景），提示 unfinished 数。
3. 删除：有需求引用禁止。
4. 归入版本：不加快捷入口（编辑表单下拉即可；推进主要靠 agent）。
5. 进度汇总：计数 + 进度条。
6. 版本名：自由文本。

---

## 8. 实现顺序（pkg-9 单包）

1. **T1 Host**：5 个版本写接口 + 状态机转移 + 引用检查 + 废弃联动双写。
2. **T2 Client**：版本管理弹窗（列表/进度/新建/推进/废弃/改名/删除）+ 顶栏入口 + 二次确认。
3. **T3 联动**：废弃后看板/列表刷新；顶栏版本下拉刷新。
4. 实现后验证：建 v2.0 → 推进 → 发布（有未完成提示）→ 废弃（需求全取消）→ 删除空版本 / 有引用版本 blocked。
5. 回写：m5-version.md 状态 → 已实现；同步 `workbench-plugin/` 源码。
