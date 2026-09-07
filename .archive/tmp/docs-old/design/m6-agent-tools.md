# M6：Agent 工具链（产品 + 技术合编）

> 状态：**v1.2（wb_my_task 已落地 pkg-41）** — 22 个 `wb_*` 模型工具已注册并对 agent 可见（`Tool.listTools`）；本文档是专有收编文档，替代散落在 m2/m3/m4/m5 技术文档中的各工具规划。

## 1. 定位

工作台四层架构中的**执行层能力**：让 agent（本会话模型与未来子 agent）通过模型工具直接驱动工作台——推进任务、拆分大版本、评审闭环、澄清需求，**与 UI 走同一套审计 / 门禁 / 状态机逻辑**。

| 层 | 谁在用 | 形式 |
|---|---|---|
| 模型工具 | agent | 16 个 `wb_*`（schema 即接口文档） |
| Skill 指南 | agent | 组合流程/业务规则（随 M6 team preset 挂载） |
| 本文档 | 人/存档 | 产品语义 + 技术实现 + 场景编排 |

## 2. 工具清单（22 个）

### 2.1 读（5）
| 工具 | 入参 | 出参 | 语义 |
|---|---|---|---|
| `wb_list_requirements` | — | 需求池数组 | 全量需求（含 description），推进/拆分前置 |
| `wb_list_versions` | — | 版本数组 | id/name/state/isBacklog；拆分选目标版本 |
| `wb_get_requirement` | `{reqId}` | 聚合快照或 null | 需求字段+流程树+产物+评论；评审前置 |
| `wb_get_artifact_content` | `{reqId, artifactId}` | `{art, content}` 或 null | 读产物正文，定位评论锚点 |
| `wb_my_task` | `{reqId, role, nodeId?}` | 确定性任务上下文或 `{ok:false}` | **任务确定性来源（pkg-41）**：返回本角色在本节点的 roleObjective/behaviors/onFeedback + reviewers/artifactsToReview/dependencyDocs/outputs + **`sessionsByNode`——该需求每个 node 的全部会话（issue/discussion/notification 全含，含锚定在产物文档上的会话并解析 anchorArtifact；条条带 nodeId/nodeKey 明确归属，组内时间升序、replies 完整）**——subagent 被派活后先调本工具拿全部上下文，无需自行检索讨论 |

### 2.2 需求（3）
| 工具 | 入参 | 出参 | 语义 |
|---|---|---|---|
| `wb_create_requirement` | `{title, description?, category?, versionId?, priority?, status?, owner?}` | `{ok, reqList}` | 建需求（默认同表单；`history=[]` 审计起点） |
| `wb_update_requirement` | `{reqId, ...7 字段}` | `{ok, reqList}` | 改需求（Host 算 diff 自动写审计） |
| `wb_delete_requirement` | `{reqId}` | `{ok, reqList}` | ⚠️ 删除需求（不删磁盘目录） |

### 2.3 版本（4）
| 工具 | 入参 | 出参 | 语义 |
|---|---|---|---|
| `wb_create_version` | `{name}` | `{ok, verList}` | 建版本（待启动，版本名自由文本） |
| `wb_update_version` | `{versionId, name}` | `{ok, verList}` | 改版本名（待分配池锁定） |
| `wb_advance_version` | `{versionId}` | `{ok, verList, unfinished}` | 推进：待启动→进行中→已发布（单向）；发布返回未完成需求数 |
| `wb_deprecate_version` | `{versionId}` | `{ok, verList}` | ⚠️ 废弃版本（有需求引用 blocked，需先移出） |

### 2.4 评论 / 评审（5）
| 工具 | 入参 | 出参 | 语义 |
|---|---|---|---|
| `wb_comment` | `{reqId, nodeId, anchor?, text}` | `{ok, snapshot}` | 追加 open 评论（阻塞门禁）；anchor 四类：document/line/range/block |
| `wb_reply` | `{reqId, commentId, text}` | `{ok, snapshot}` | 评论下对话（roundCount+1） |
| `wb_resolve` | `{reqId, commentId}` | `{ok, snapshot}` | 评论 → resolved |
| `wb_override` | `{reqId, commentId?/reviewerRole?}` | `{ok, snapshot}` | ⚠️ 强权：单条或整角色 open 评论 → overridden |
| `wb_decide` | `{reqId, nodeId, verdict, reason?, newScope?}` | `{ok, snapshot}` | 门禁：pass（未全 resolved 拦截）/ reject（回退+理由，newScope=重界定范围） |

⚠️ = 破坏性/强权操作，工具描述已标注「需 User 明确指令」，agent 调用前必须获得 User 确认。

## 3. Agent 使用方式

### 3.1 直接调用（接口即文档）
工具经 `Tool.listTools` 对 agent 可见，调用时直接呈现参数 schema 与中文语义，**无需查外部接口文档**。工具返回均为 lossless JSON；失败统一 `{ok:false, error, ...}`，门禁拦截带 `blocked:true`。

### 3.2 组合流程（Skill 指南，随 M6 team preset 挂载）
工具描述只承载单工具语义；**多步业务编排**由 skill `wb-agent-guide` 承载（agent 接到相关任务时主动加载）：

- **任务上下文（pkg-41，所有角色通用）**：被派活先调 `wb_my_task(reqId, role)` 拿确定性上下文——本节点做什么（roleObjective/behaviors/onFeedback）、审什么（reviewers/artifactsToReview）、依据什么（dependencyDocs + **sessionsByNode 全需求每 node 全会话，含需求确认阶段与 User 的对话与文档锚定会话**）；随后按返回的 outputs 产出或按 reviewers 表态。
- **拆分大版本**：`wb_list_versions` 确认目标 → 按 owner 规则逐条 `wb_create_requirement`（带描述 md、owner）→ 每条自动带审计起点
- **推进任务**：`wb_list_requirements` 定位 → `wb_get_requirement` 看节点/评论 → 补齐产物与评论 → `wb_resolve` 全部 → `wb_decide pass`；被打回则 `wb_decide reject` 理由回退
- **评审闭环**：`wb_get_requirement` → `wb_get_artifact_content` 读正文 → `wb_comment` 行级/代码块级问题 → User 处理 → 循环至全 resolved → `wb_decide pass`
- **版本生命周期**：`wb_advance_version` 单向推进；发布前检查 `unfinished`，有未完成需求需 User 确认；`wb_deprecate_version` 前先确认需求归属

## 4. 技术实现（pkg-24）

### 4.1 架构：op* 共享领域函数
22 个 `wb.*` handler 提取为具名领域函数 `op*`（`opListRequirements` … `opDecide`、`opMyTask`），三处共用同一套逻辑：

```
op* 领域函数（读 .workbench/*.json + 审计/门禁/状态机）
 ├─ harness.handle('wb.xxx')  →  Client host.call（UI）
 ├─ harness.defineTool+registerTool →  agent 模型工具（wb_xxx）
 └─ （未来）子 agent 同工具
```

### 4.2 注册方式
- `harness.defineTool({ name, description, parameters, output:{schema,render}, execute })` 构造定义（沙箱校验 output/render/execute）
- `harness.registerTool(ctx, def)` 注册，返回 disposer；用 `ctx.effect(() => harness.registerTool(...))` 包裹 → **Fiber 生命周期**，插件 stop/update 自动注销
- 参数 DSL：`{ field: { type:'string'|'boolean'|'json', required:true, description } }`；输出 `output.schema:{type:'json'}`，`render` 返回 `[{type:'text', text:JSON.stringify(value,null,2)}]`
- execute 返回经沙箱 `cloneJson` 归一（lossless；`undefined` 会被拒，领域函数均返回 JSON-safe 值）

### 4.3 一致性保证
- 审计：`wb_update_requirement` 由 Host 算 7 字段 diff 写 `history[]`（谁将什么从什么变更为什么）——工具与 UI 同一行为
- 门禁：`wb_decide pass` 校验当前节点 open 评论全 resolved，否则 `{blocked:true}`——工具与 UI 同一状态机
- 破坏性保护：delete/deprecate/override 描述标注「需 User 明确指令」；v1 不做权限系统

## 5. Skill 草案（wb-agent-guide，M6 挂载素材）

> 目标：preset 目录 `skills/wb-agent-guide/skill.md`，随 M6 team preset 挂载。内容为 §3.2 的组合流程 + 数据位置（`<workspace>/.workbench/`）+ 工具调用顺序 + 破坏性操作确认要求。挂载方式见 M6 文档（team preset 从 standard 复制后加入 skills 目录）。

## 6. 相关文档

- 原散落规划：`m2-flow-tree-tech.md` §5.1（评论/评审）、`m3-board-tech.md` §6（看板/列表）、`m4-req-crud-tech.md` §8.1（需求 CRUD，已回写 ✅）、`m5-version-tech.md`（版本）
- 数据层：`m2-flow-tree-tech.md`（流程/评论/产物 schema）
- 源码落盘：`workbench-plugin/host.js`（pkg-24）
