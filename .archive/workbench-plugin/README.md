# 团队项目工作台 插件源码（原型阶段）

> 这是 `wbflow-3` 动态 Cordis 插件的**落盘源码副本**，供版本管理与重启恢复。
> 这里的 `return { ... }` 是 Plugin 函数体——重新用 `cordis_define` 加载时，`code.host`/`code.client` 就是这段函数体。

## ⚠️ cordis_define 铁律（每次提交前必查）

**一次 `cordis_define` 的 `code` 对象必须同时完整包含 `host` 和 `client` 两个 key。**

本会话已多次因「先只传 host 漏 client」或「先只传 client 漏 host」导致：
- 产生一堆 `hasHostHalf:false` / `hasClientHalf:false` 的半成品 Package（pkg-10/11/18/19/20/21/23/26/28/29）；
- 反复 update、反复审批、浪费大量轮次。

**提交前自检清单（逐条过，缺一不提交）：**
1. `code` 里有 `host` 键？有 `client` 键？
2. 两个键都是**完整的最新版**函数体（不是旧版、不是半截）？
3. `host` 用多行清晰写法，**严禁压成超长单行**（曾因压缩单行导致括号错配 `SyntaxError: missing ) after argument list`）；
4. `host` 返回值所有字段必须是合法 JSON（`undefined` → `null`，不能有类实例/函数/Date）；
5. `client` 的 React 组件**所有 hooks（useState/useEffect）集中在函数最顶部**，在任何条件 `return` 之前（否则 React #310）；
6. `client` 用 `React.createElement(...)`，无 JSX/TS/import；
7. 纯 JS 函数体以 `return { ... }` 结尾（`}` 后可有 `;`，**绝无多余的 `)` 或 `});`**）。

违反任何一条，先修，不 `cordis_run`。

## 文件

- `host.js` —— Host 半（Node 进程），读写 `<workspace>/.workbench/` + RPC。
- `client.js` —— Client 半（浏览器），`conversation.view` 「项目」tab 流程树 UI。

## 如何重新加载（进程重启后恢复）

1. `code.host` = `host.js` 全部内容；`code.client` = `client.js` 全部内容（**两个都要**）。
2. `cordis_define` 时 `plugin.kind:'existing', pluginId:'wbflow-3'`（或新建用 idPrefix `wb`）。
3. `cordis_run`（current 存在则 `update`）。

## 已知关键点（避免重踩）

- **写文件必须传 sandboxPolicy**：插件宿主上下文拿不到 session cwd，`sandboxPolicy.resolve()` 会落到 DSH 安装目录（deepseek-harness）作为 fallback root，导致写 workspace 被拒 `file access denied under workspace-write`。修复：`writePolicy()` 显式返回 `{ mode:'workspace-write', workspaceRoot: PROJECT }` 传给 `fs.writeText`。
- **领域数据**在 `<workspace>/.workbench/`（原型假数据）。

## 当前进度（里程碑）

- **M2 流程树全部完成**：门禁（通过/打回+前进回退+全评论 resolved 拦截）/ 评审状态行+角色对话区+reply/resolve/override / 产物文件行+阅读器 / 行级锚点+飞书式阅读器+折叠 / 代码块整块评论 / 范围评论+轻量 md。锚点四级：document / line / range / block。
- **M3 版本看板+需求池全部完成（pkg-7）**：外层看板（`<select>` 版本筛选 + 状态泳道 + 看板/列表切换）、需求优先级 P0-P3（属性+筛选+三处展示）、列表视图分组（公共表头+虚线分割+空组隐藏）、点需求进 M2 流程树。
- **M4 需求池增删改完成（pkg-15 含 v1.1 信息面板）**：
  - 顶栏「＋ 新建需求」+ 新建/编辑弹窗（标题*/描述 textarea/类别/版本/优先级/状态/负责人+datalist）
  - 卡片与列表行 hover「✎ 编辑 / 🗑 删除」（不拦截进详情）；删除二次确认
  - 需求描述支持 Markdown（详情页 md 渲染：标题/粗体/行内代码/代码块/列表/链接）
  - **编辑审计**：Host 计算 7 字段 diff 写 `history[]`，详情页「变更历史」面板倒序展示 `字段：旧值 → 新值`
  - **需求信息面板（v1.1）**：详情页显示 负责人/版本/优先级/状态/类别/创建/更新时间
  - Host 接口：`wb.createRequirement` / `wb.updateRequirement`（审计）/ `wb.deleteRequirement`
- **M5 版本管理完成（pkg-15 为最终版，已落盘）**：
  - 版本 CRUD：新建/改名（**无删除**，废弃是唯一移除手段）
  - 版本状态机：`待启动→进行中→已发布` 单向推进；任意阶段可直接废弃（有需求引用禁止废弃，需先移出；废弃不可恢复）
  - **发布确认**：推进为已发布时若版本下有待启动/进行中需求 → 弹确认框
  - 进度汇总：各版本四状态计数 + 进度条；待分配池锁定
  - Host 接口：`wb.createVersion` / `wb.updateVersion` / `wb.advanceVersion` / `wb.deprecateVersion`（无 deleteVersion）
- **agent 工具注册完成（pkg-24）**：16 个 `wb.*` 领域操作注册为模型工具（`wb_list_requirements` / `wb_list_versions` / `wb_get_requirement` / `wb_get_artifact_content` / `wb_create|update|delete_requirement` / `wb_create|update|advance|deprecate_version` / `wb_comment` / `wb_reply` / `wb_resolve` / `wb_override` / `wb_decide`），`Tool.listTools` 对 agent 可见；RPC 与工具共用同一套 `op*` 领域函数（同一审计/门禁/状态机）；破坏性工具描述标注「需 User 明确指令」。
- **wb_my_task 落地（pkg-41，m8 §11）**：+`wb_my_task`（22 个 `wb_*` 工具）——确定性任务上下文（roleObjective/behaviors/onFeedback + reviewers/artifactsToReview/dependencyDocs/outputs）+ **`sessionsByNode` 全会话供给（v0.30 泛化）：每个 node 的全部会话（issue/discussion/notification 全含，含锚定在产物文档上的会话，anchorArtifact 解析；条条带 nodeId/nodeKey 归属，组内时间升序、replies 完整）**，subagent 被派活后先调本工具拿全部上下文；行为协议表（m8 §2.4.4）内联 ROLE_PROTOCOLS、产出映射 NODE_OUTPUTS、align 判定 = key 含「需求对齐」（sqlite 迁移后切 mode 字段）；**User 拍板：会话维持 issues discussion 唯一载体，不另产物化到 artifacts；全会话按 node 分组供给 subagent**。
- 描述折叠（pkg-16~21）：≤4 行完整显示；>4 行折叠（150px）+ 展开/收起；变更历史自然语句（pkg-23）：`谁将「字段」从「X」变更为「Y」`，描述显示「谁填写/清空/修改了需求说明」。
- 技术设计/PRD：`docs/design/m3-board-tech.md`（回写）、`m4-req-crud.md`（v1.4）、`m4-req-crud-tech.md`、`m5-version.md`（v1.0）、`m5-version-tech.md`
- 后续：M6 team 预设制度；M7 真实子 agent 编排

## 数据层（领域数据，非源码）

见 `<workspace>/.workbench/`。详见 `docs/design/m2-flow-tree-tech.md`。
