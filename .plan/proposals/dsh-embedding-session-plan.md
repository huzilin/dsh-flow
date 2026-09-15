# wbflow × DSH 嵌入适配方案：project 工作区 + flow 挂载 session + subagent 两层执行（T18 提案 v3 · 定稿）

> 状态：**定稿（全部决策点已拍板 2026-09-13）**。D1=M2 两层式（项目制，不机械模仿 novel）；绑定 session=程序创建（模式 B 历史拍板对齐）；D3=bind 成功后落行、重试上限默认 5；D5=工作区=内部 project、会话按项目挂载 cwd（补历史缺口：上轮 DSH 测试未挂载工作目录）。
> 日期：2026-09-13。参考：Hy4 记录、DM-07（project 维度）、data-tools-final L104（project registry）、spec §1.2 /api/projects（预留「dsh 嵌入期启用」）、「MVP 架构模式选择」模式 B、CONTEXT.md 词条（绑定 session / sub session / project 工作区，已同步定稿）。
> 性质分类：【已写明】=既有拍板在案；【空白】=文档无规定；【推断】=推荐（projects 表存储形态一项，默认采纳）。

---

## 1. 历史拍板底座（【已写明】，本方案不改动只接线）

1. **T10 占位边界**（tickets.md #10）：「sessions 表接线：flow↔session 绑定、persona 按节点配置传入、status 生命周期（active/ended/disposed）。**本期不接真实 DSH（嵌入适配器 out of scope），以 sessions 句柄占位**。」——本工单（T18）即补「嵌入适配器」这块。
2. **subagent 程序化直启**（2026-09-08 实测，roadmap P5 机制位）：wbflow 插件调 `ctx.subagents.startContinuable('spawn', {prompt, persona, parent})`；persona（现行为角色说明 note，见 retired-terms-audit §7）由节点配置传入，不需要 subagent 预设；child 挂 parent lineage；开场白 = agent 首条 assistant 输出、由节点启动指令触发；启动失败检测 = `agent/request-error` 监听 + 兜底 + 重试策略（原记录「task_stale 兜底」为退役词，现口径=running 停滞兜底）。
3. **绑定行口径**（session.go `Bind` 现行）：同任务键重复绑定**不去重**（多轮会话各占一行），active 代理判据取最早一行。
4. **裁决 session**（T11 已实现）：escalate → 裁决 session 落 sessions 表并与节点关联——本期只把占位句柄换真实句柄。
5. **会话找 X**（ui-interaction §6 N5，FE disabled 占位已实现）：X 随节点绑定角色动态——scope=full 找 qa / scope=fe 找 fe / scope=rd 找 rd；「按需会话通道 DSH 嵌入期开放，本期 disabled 占位」。
6. **CONTEXT.md 词条**：「subagent 会话（挂载到 flow 节点）：每个 subagent 有独立稳定 sessionId，持久化跨重启；UI 经插件页面从 flow 节点查看历史/进行中的 subagent 对话（**不经 Leader 中转**）。」
7. **通路并存判据**（llmflow.go 现行）：`kind=node 动作须 executor=subagent（program=内置动作非 LLM 通路）`——executor 配置即 llm 通路与 subagent 通路的分流位。

## 2. DSH 通道能力（Hy4 实测事实，novel 侧 2026-09-13）

- typert RPC：`POST http://127.0.0.1:3080/api/<namespace>/<method>`，envelope `{type:'client-request', rpcId, method, payload:{args:{_request|request}}}`（包装名按端点：list 用 `_request`，prompt 用 `request`）。
- `session/list` → 每会话 `running` / `blank` / `asOfSeq`（日志水位）/ projections（title/goal/tokens/modelMs）——**卡死观测与催办的现成通道**（「以为卡死实际在跑」由 running+seq 推进证实）。
- `session/prompt` → `{sessionId, mode:'queue', content:[{type:'text',text}]}`——会话驱动。
- ACP `session/new` 原生支持 `mcpServers`（http/stdio）；会话日志 `~/.dsh/sessions/<cwd>/<sessionId>/session.jsonl.zstd`。
- 标准调用模式（Hy4 零失败）：**单 session 串行 + running 轮询 + 同 session 重试**。
- rd 已有 DSH 网关配置（llm 段 base_url 默认 `127.0.0.1:3080`，key 走 `WBFLOW_LLM_API_KEY`）——typert 客户端复用同源端点配置。

## 3. 决策点（4 项，默认=推荐，待确认）

### D1 拉起通道架构【已拍板 2026-09-13：M2 两层式】

**否决记录**：v1「插件直启」（wbflow cordis 插件在 DSH 进程内调 `ctx.subagents.startContinuable`）被否——要求 DSH 进程常驻 wbflow 代码，与「web 页面 + Go 底层 + 只经 MCP 交互」不符。v2 的「M1 一层式」（会话即执行者）同日被否——User：不机械模仿 novel，novel 是单会话审校链不需要 subagent，wbflow 是**项目制**，要把整个 flow 挂载到当前 session 中。

**定稿：M2 两层式**：
- **绑定 session（主会话）**：flow 挂载其中；cwd=所属 project 的工作区（见 D5）；会话挂载 rd 的 wb MCP server。
- **sub session（child）**：节点动作执行时，由绑定 session 内的 agent spawn subagent 承接——节点启动指令与角色说明（note）经 wb MCP 任务材料下发；child sessionId 经 MCP 工具回告 rd，bind 成功后落行（D3）。
- **依据**：`ctx.subagents.startContinuable/spawn` 是 DSH 进程内能力（dsh-knowledge ch11），绑定 session 内的 agent 是 DSH 侧合法调用方——M2 下 spawn 发生在 DSH 进程内的 agent 侧，rd 不碰进程内 API，符合 MCP-only。
- **绑定 session 来源【已拍板 2026-09-13：程序创建】**：rd 经 `session/new` 自动创建常驻主会话（cwd=project 工作区、挂 rd wb MCP）。历史依据「MVP 架构模式选择」模式 B：程序直起会话优于 Leader spawn；**flow↔session 一对一，所有 subagent 拉起都通过该会话**；会话创建传 workspaceId 挂工作区。实现要点：空白会话在 UI 列表隐藏——创建后必须 `session/prompt` 发首条消息（节点启动指令）激活 non-blank 才可见。
- **bind 落行分工**：主会话 rd 自己创建、创建即知 sessionId——**rd 直接落行，无需回告**；child（sub session）由 agent spawn、rd 不可见——经新增 MCP 工具 `wb_child_report`（childId+taskId）回告，bind 成功后落行（D3）。

### D2 dsh_ref 真实形态【空白→推荐】

`dsh_ref` 从占位 JSON 升级为：`{"kind":"dsh","sessionId":"…","cwd":"…"}`；`{"kind":"placeholder","note":…}` 保留为「未接线」合法态。读侧按 `kind` 分支宽容解析。SessionItem 投影（spec §1.2 注记-D2 的「最近消息摘要随嵌入期补齐」偏差）随本票补齐字段。

### D3 编排闭环与落行时机【已拍板 2026-09-13】

- **落行时机**：bind（会话创建/回告成功、拿到真实 sessionId）**之后**才落 `Bind` 行（status=active，dsh_ref=真实句柄）；bind 失败不落行。
- **重试**：会话创建失败重试**加上限，默认 5**（参数化 retry_max=5，对齐 rd/llm 重试器形态）；耗尽告警通知 user，不落行。
- 口径区分：此为「subagent 会话创建重试」；通知 outbox 的「无限补投无 dead」（NTF-04/05 既有拍板）是另一域，不受影响。

### D5 project 工作区挂载【拍板补充 2026-09-13】

- **拍板**：工作区 = 内部 **project**；会话按项目挂载工作目录（cwd）。历史缺口：上轮 DSH 测试跑通但 DSH 侧未挂载工作目录——本票必须补上。
- **历史依据（已写明）**：DM-07（09-05，现行权威）「新增 project 维度（不同 project 绑定不同工作区）」；data-tools-final「project registry：`[{project_id, name, workspace, status}]`，per-project workbench 隔离，表零改动」；spec §1.2 路由表已预留 `GET/POST /api/projects`（标注「registry，dsh 嵌入期启用」）——T18 即该端点的启用时机。
- **【推断→待确认】**：registry 存储形态——历史是插件时代全局配置文件（`~/.dsh/wbflow/projects.json`）；rd 常驻 web 形态下推荐落 **rd MySQL 新表 projects**（id/name/workspace/status，符合数据模型九条），/api/projects 启用即读写该表；需求归属 project 的列**暂不加**（沿用「零改动」口径：rd 部署默认工作区起步，多 project 归属留到 project 维度全面启用时）。

### D4 本期范围与验收动作【空白→推荐】

- 工单 **T18**（tickets 编号顺延，见 §4）。
- subagent 通路真 DSH 验收至少覆盖：①一条 produce 链（节点执行→直启→产出落盘→提评）；②会话找 X 开放（真实打开绑定角色会话，不经 Leader 中转）；③裁决 session 真实化。
- LLM 通路不撤：两通路按 capability 动作 `executor` 配置并存分流。

## 4. T18 工单（已追加 tickets.md）

见 `tickets.md` §18，AC 六条（真实句柄落表 / 完成上报全链 / 会话找 X / 卡死观测 / 失败链 / 裁决真实化）。

## 5. 影响面（MCP-only + M2 两层式）

- **rd 新增 MCP server（T18 最大新增件）**：wb_* 工具族（wb_my_task / wb_advance / wb_review / wb_comment / 产物提交 / **session 登记·child 回告**）经 MCP 暴露——rd 现状只有 HTTP 端点（router.go），同一 service 层双面入口。
- **rd 新增 DSH 会话客户端**：session/new（挂 mcpServers + cwd=project 工作区）、session/prompt、session/list（观测），与 llm HTTP client 同源配置段。
- **project registry**：projects 表 + /api/projects 启用（spec 预留位兑现，DM-07 落地）。
- **rd**：RoleNoteResolver 接节点配置真实数据源；scanner 观测接线；SessionItem 投影扩展（主会话行+sub session 行双态）。
- **FE**：会话找 X disabled 摘除；SessionItem 摘要字段呈现。
- **不再需要**：wbflow cordis 插件（v1 方案组件，随 D1 否决移除）。
- **约束依赖**：真模型演练依赖 DSH supplier 恢复（llm 通路同款）；绑定 session 需用户在 DSH 侧开启（来源粒度见 D1 遗留①）。
- **验证方式**：真库 + 真 DSH 端到端（T18 AC1-8），失败注入用例（AC7：重试 5 次耗尽告警、不落行）；**cwd 挂载实证**（AC1：DSH 侧会话日志落点=项目工作区归档，对照上轮未挂载缺口）。

## 6. 确认后动作（本提案转正时）

1. ADR-0001 状态 Proposed → Accepted；CONTEXT.md 补「嵌入适配器 / dsh_ref 真实形态 / 会话找 X」词条定稿。
2. T18 进入 frontier（blocked by 10/LLM 通路均已完，可立即开工）。
