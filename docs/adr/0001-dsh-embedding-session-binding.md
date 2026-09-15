# ADR-0001：DSH 嵌入适配——project 工作区、flow 挂载 session 与 subagent 两层执行

- 状态：**Accepted**（2026-09-13 全部决策点拍板定稿；提案全文见 `.plan/proposals/dsh-embedding-session-plan.md`；semantica `6d28e845`）
- 日期：2026-09-13
- 关联：T10（会话绑定占位，已完）、T11（裁决 session，已完）、LLM 通路（已完）、tickets.md #18、DM-07（project 维度）、「MVP 架构模式选择」模式 B

## 背景

wbflow v1.0 本期 17 张工单已收官，会话绑定按 T10 边界「本期不接真实 DSH，以 sessions 句柄占位」落地（`rd/service/session.go`，placeholder JSON + RoleNoteResolver 占位）。现在补嵌入适配器。

架构约束（User 拍板 2026-09-13）：现行方案=web 页面 + Go 底层，**与 DSH 只经 MCP 交互**——v1「wbflow cordis 插件进程内直启」否决（要求 DSH 进程常驻 wbflow 代码）。

事实输入：
- `ctx.subagents` 是 DSH 进程内 Cordis 服务（dsh-knowledge ch11）；绑定 session 内的 agent 是其合法调用方。
- rd 现状无 MCP server，wb_* 仅为 HTTP 端点概念名。
- novel Hy4 实测：BE 最小 MCP server 经 ACP `session/new` 的 `mcpServers`（http 型）挂载；typert `session/prompt` 驱动、`session/list` 轮询观测。

## 决策

1. **D1（已拍板）M2 两层式 + 程序创建主会话**：项目制，整个 flow 挂载于绑定 session；绑定 session 由 **rd 程序创建**（`session/new`，cwd=project 工作区，挂 rd MCP；「MVP 架构模式选择」模式 B：flow↔session 一对一、所有 subagent 拉起经该会话；空白会话须发首条消息激活 non-blank）；节点动作由主会话内 spawn 的 subagent 承接（child=sub session，节点启动指令+角色说明经 wb MCP 任务材料下发，child sessionId 经新增 MCP 工具 `wb_child_report` 回告 rd）。rd 创建主会话即知 sessionId，直接落 bind 行无需回告。User 理由：不机械模仿 novel（单会话审校链不需要 subagent），符合历史推演与模式 B 先例。v1 插件方案、v2 M1 一层式、用户手动开会话方案均否决。
2. **D2 dsh_ref 真实形态 = `{"kind":"dsh","sessionId","cwd"}`**，placeholder 保留为未接线态，读侧按 kind 宽容解析；主会话行 node_id=''，sub session 行 requirement+node+role。
3. **D3（已拍板）**：bind 成功后落行（active + 真实句柄），失败不落行；会话创建重试上限默认 5（retry_max 参数化），耗尽告警通知 user。通知 outbox 无限补投（NTF-04/05）不受影响。
4. **D5（拍板补充）工作区 = 内部 project**：会话按项目挂载 cwd（补历史缺口：上轮 DSH 测试未挂载工作目录）；/api/projects registry 启用（spec §1.2 预留位、DM-07 落地）；存储形态推荐 rd projects 表（待确认），需求归属列暂不加（沿用零改动口径）。
5. **D4 范围 = T18 八条 AC**：工作区挂载 / flow 挂载登记 / sub session 回告落行 / MCP 全链 / 会话找 X / 卡死观测 / 失败链 / 裁决真实化。

## 遗留（实现期细项，非决策）

①projects 表存储形态按推荐默认采纳（rd MySQL 新表，id/name/workspace/status）——如有异议随实现推翻；②`wb_child_report` 工具的入参 schema 随 T18 实现定；③主会话创建的触发时机（需求启动编排时按需创建）随实现细化。

## 后果

- 正：对齐「项目制」历史推演、模式 B 先例与 User 架构口径；spawn 发生在 DSH 侧 agent 内（进程内能力的合法用法），rd 零插件；flow→session→sub session 三级绑定可观测（sessions 表全留痕）；主会话 rd 自建自知，无回告时序问题。
- 负/风险：主会话常驻由 rd 生命周期管理（需求完结/废弃时会话 End/Dispose 策略随实现定）；child 回告前 rd 不可见 child（停滞兜底由 scanner 观测补）；rd 需新建 MCP server。
- 中立：sessions 表新增主会话行语义（node_id=''）；projects registry 落库。

## 确认后动作（已完成）

CONTEXT.md 词条定稿（绑定 session / sub session / project 工作区，2026-09-13）；T18 进 frontier。
