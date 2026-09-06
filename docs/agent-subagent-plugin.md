# agent · subagent · 插件：三者关系与多主 agent 并行时的交互

> 讨论稿（v0.4）——为「后续讨论 agent、subagent、插件之间的关系，以及多个主 agent 并行时插件和 agent 如何交互」准备的事实底座。
> v0.4：新增 §8「与你自己的工作台系统对照」（Leader/角色子agent/wbflow ↔ DSH 实体模型）；v0.3 新增 §4.7 组级隔离路径对比；v0.2 新增 §4.6 本机 profile 插件盘点与 §7 证据索引；v0.1 为初稿。
> 全部结论以 DeepSeek Harness（DSH）源码与文档为准：
> `docs/glossary.md`、`docs/architecture.md`、`docs/subsystems/{core,subagent,extensions,tools,scope,session,web-server}.md`、`docs/cordis-primer.md`、`packages/extensions/README.md`、`packages/subagent/README.md`。
> 本文是「事实 + 争议点」，末尾的开放问题是留给讨论的，不是结论。
> **开聊入口**：配套 30 分钟议程见 `docs/讨论议程-agent-subagent-plugin.md`（含一分钟快答表）。

---

## 0. 一句话模型

DSH 里**没有除插件之外的"核心"**：模型适配器、工具注册表、会话日志、乃至 agent 回合循环本身都是插件（Cordis plugin）。

三者关系不是并列，而是三个层次：

- **插件** = 能力与策略的*注入者*，挂在 **context** 上（服务 `ctx.<key>`、工具、事件、UI slot、HTTP 路由、可逆 effect）；
- **agent（主 agent）** = 一段*回合循环 + 持久会话状态*的执行者，每个 agent 拥有自己的 scoped context（`agent.ctx`）与自己的 Session 日志；
- **subagent** = agent 之间*委托关系*的实现——它不是一个第三种实体，而是"一个 agent 通过工具（`ctx.subagents`）创建出**另一个 agent**"，child 同样是完整 agent（独立 session、独立 scoped ctx、独立回合循环）。

一句话：**插件决定 agent「能干什么、被谁拦、看到什么」；主 agent 决定「谁在干」；subagent 决定「干不完时派谁去」**。

---

## 1. 三者各自的本质

### 1.1 agent（主 agent）

- DSH 没有正式"主 agent"术语；运行时里**顶层 agent**（无 `parentSession`、由 `ctx.agents.roots()` 列出）即"主 agent"。
- 一个 agent ≡ 一个 `SessionId` ≡ 一条 append-only `Session` event log ≡ 一个 scoped world（`agent.ctx`）≡ 一个回合循环（turn → step → tool 执行）。
- 生命周期：`create / resume` → `idle / running` → `dispose`（停止循环、注销 agent、移除 session、最终解构 scoped world）。
- **一个进程可同时承载多个 agent**：`ctx.agents.list()` 返回全部 live agent。当前这台机器的 DSH 进程（`dsh-process.json`）同时挂着 8 个 workdir 的会话——这就是"多个主 agent 并行"的日常形态。

### 1.2 subagent

- 是一个**可选能力 seam**（`ctx.subagents`），不属于 agent loop 本体；与 bash 不同，它允许**多个 provider 并存**：`spawn` / `fork`（进程内）、`acp`、`codex`、`claude-code`、`dsh-sdk`（进程外）。
- 两种形态：
  - **one-shot**：一次前台委托，`SubagentRun` 返回 `{ output, structured?, stopReason }`，用完 `dispose`；
  - **continuable**：持久 child Session + 至多一个进程内 Activation，child 的 inbox 是唯一 FIFO 队列；父用 `followup`（发消息）/ `interrupt`（停回合）/ `listChildren/listDescendants`（枚举）协调，child 用 `reportFrom` 主动回报。
- 关键设计：**child 拿到的是一个全新扁平 scope——不继承父的 scoped 注册**（"Each child gets a new flat scope rather than inheriting parent registrations"）；继承关系只是数据（`parentSession`、`delegationDepth`），不影响可见性。要让 child 拥有父的某些工具/插件能力，需显式用 `agentPresets.composeFrom(childCtx, parentCtx)` 或 preset 组合传递。

### 1.3 插件（plugin / extension / cordis）

- Cordis 是 DSH vendored 的插件框架：plugin 实现 Service，靠 `inject` 声明依赖，注册进 context；"every part of the product is a plugin"。
- 挂接点（capability seams，约 40 个 `ctx.*` 服务）：
  - **工具**：`ctx.tools` 注册模型可调用工具，走 `tools/pre-execute → guards → execute → post-execute → finalizeContent → tools/result` 流水线回到模型上下文；
  - **事件**：`emit`（观察）/ `waterfall`（包装、可短路）/ `parallel` / `serial` 四种分发；agent 生命周期（`agent/created`、`agent/disposed`）、`session/event`、`llm/stream`、`system-prompt/assemble` 都是插件可拦的点；
  - **UI**：`ctx.clientModules` + `ui-layout` slot（如 `shell.overlay`），浏览器端插件把 UI 挂进页面；
  - **HTTP**：`ctx.webServer` 注册命名路由（`/api` 桥、插件 bundle、HMR 事件流）；
  - **动态插件**：`tool-cordis` 让 agent 自己 `define / run / stop / undefine`——host 半跑在 `node:vm` 沙箱，client 半跑在浏览器页面上。
- 一切注册都是**可逆 effect**：`ctx.effect()` 挂 disposer，插件卸载时按注册逆序清理；HMR/重载可预测地拆装。

---

## 2. 三者关系图（分层视角）

```text
┌─ 进程 / Host（一个 dsh 进程）───────────────────────────────┐
│  全局 context：全局注册的工具 / 服务 / 事件 / 路由           │  ← 所有 agent 可见
│  ├─ agent A（主）: agent.ctx ── scoped 注册（仅 A 可见）      │
│  │     └─ subagent A1（child）: 全新扁平 scope               │
│  │           └─ subagent A1a ...（可嵌套，delegationDepth）   │
│  ├─ agent B（主）: agent.ctx ── scoped 注册（仅 B 可见）      │
│  └─ agent C（主, 由 ACP/API 接入的外部会话）                  │
└────────────────────────────────────────────────────────────┘
```

- **插件**活在 context 上：全局 vs scoped 二选一（见 §4.2），没有中间档；
- **agent** 是 scoped context 的"key"（scope key = live Agent 对象），同时也是回合循环的运行者；
- **subagent** 是 agent 借助 `ctx.subagents` 派生的新 agent：新 session、新扁平 scope、可嵌套，lineage 只是数据。

---

## 3. 插件与（主）agent 的交互通道

| 通道 | 机制 | agent 视角 |
|---|---|---|
| 工具 | `ctx.tools` 注册 → 模型在回合中调用 → 执行流水线 → `tools/result` 冻结结果回上下文 | 回合内可调用的能力清单 |
| 服务注入 | `inject` / `ctx.<key>` 声明依赖 | 插件之间、loop 与插件之间解耦 |
| 事件 | `emit/waterfall/parallel/serial`；agent 生命周期、工具、LLM 流、system-prompt 装配 | 观察或拦截另一个组件的活动 |
| UI slot | `ui-layout` 的 slot 席位（注册即 shadowing，最新 run 获胜） | 页面上的面板/浮层 |
| HTTP / API | `ctx.webServer` 路由 + `api-gateway` | 外部访问 agent 能力的网关 |
| MCP / 外部协议 | MCP server 作工具来源注入；ACP 作外部 agent 接入协议 | 能力外包 / 会话外包 |

一次典型交互的完整链路（工具例子）：模型消息带 tool-call → `tool/call` 事件 → UI 待执行卡片 → `tools/pre-execute`（权限/沙箱/审批）→ guards → `tools/execute` → 插件 execute body → `tools/post-execute`（替换/拦截/附加 context）→ `finalizeContent` → `tools/result` → `tool/result`（模型面唯一结果）→ UI 完成卡片。

---

## 4. 多个主 agent 并行：插件与 agent 的隔离与共享规则

### 4.1 进程模型

单 host 进程 + 多 session：每个会话 = 一个独立 agent = 独立 session 日志 + 独立 scoped ctx。web server 只做浏览器的 HTTP 载体（`ctx.webServer`），本身不懂 agent；会话之间没有任何共享状态，除非走全局注册或共享存储。

### 4.2 作用域三原则（这是全部隔离的根）

1. **两层、扁平**：工具的贡献要么 global（每个 agent 可见），要么 scoped（恰好属于一个 scope key）。**scoped 注册不下沉到子 agent**（子 agent 是全新扁平 scope）；"子树行为"用 lineage 数据表达，不用 scope 结构。
2. **scoped dispatch**：关于某 agent 活动的事件带该 agent 的 carrier 分发（scope key = live Agent 对象本身）；而 registry-subject 事件（如 `tools/change`）**故意不过滤**——全局工具集变了，所有 agent 的下一轮装配都该知道。
3. **shadowing + restriction**：scoped 注册可遮蔽同名全局项（最具体者胜，per-agent persona / per-agent 工具变体）；`tools.restrict` 可按 scope 过滤全局工具集（交集合成，过滤掉的工具「在提示词里消失 AND 拒绝执行」）。

### 4.3 动态插件（tool-cordis 定义的）的会话归属——"按 session 记，跨 session 不见，运行态共享"

- `define` 记在**发起 session**名下；别的 session 看它「像不存在而不是被禁止」（不泄露）。
- 但**运行中的 plugin fiber 活在进程内存**，可能影响同进程其他 session（工具注册改变即影响下一轮装配）；`inventory` / `invoke` / `resolveRequestRun` 是**页级全局**事实，不属某个 session。
- registry 是进程内存、唯一真相源、**不落盘**：DSH 重启或 `cordis_undefine` 即消失。

### 4.4 并行冲突点清单（讨论重点）

| 情形 | 现状 |
|---|---|
| 同一插件同时挂在 agent A、B 的 scoped ctx | 互不可见、互不干扰（shadowing 各自生效）|
| 全局注册的能力被所有会话共享 | A 的插件状态（工具开关、服务状态）会作用到 B 的下一轮装配——**这是并行时唯一真正"跨 agent"的通道** |
| 文件系统 / 端口 / 全局存储 | 由 `fs`、sandbox、`webServer` 策略决定，不是 agent 级隔离的承诺 |
| 事件广播（如 `cordis/request-run` → apiproxy → 浏览器）| 页级共享，设计如此 |
| session 查询 / 反馈 / 设置 | 明确的 per-session 隔离（session-query、feedback、settings 各有隔离语义）|
| 同一个 session 的并发 writer | session 日志机制显式容忍并发 session 写（各有自己的落盘边界）|

### 4.5 实操速查

- **只想给 A、不给 B**：在 A 的 scoped ctx 注册（`agent.ctx`）。
- **想给 A 和 B 都要**：全局注册，或走 agent preset 组合。
- **想给子 agent 继承一部分父能力**：`agentPresets.composeFrom(childCtx, parentCtx)` / preset join——不是默认行为，必须显式做。
- **想给"一组会话"**：现状没有这种维度（只有全局 / 单 agent / 按 session 归属的动态插件），这是潜在的讨论点。

### 4.6 实例盘点：你 web profile 上的插件落在模型的哪里

> 以本机 `~/.dsh/profiles/web`（dsh-process.json 中正在运行的 profile）为准。端/挂接点一列已按各插件 README/package.json 逐一核验：`dsh-agent-instructions` 与 `dsh-mcp-client` 为前轮直接核验，其余 18 个由分类子 agent 全查（v0.4）。session 作用域行为仅引 README 明说的，其余标"未提及"。

| 插件 | 端 | 挂接点（机制）| 并行/会话行为线索（README 原话）|
|---|---|---|---|
| `dsh-agent-instructions`（dsh-base 内）| host | prompt 注入（AGENTS.md 以 system-reminder 帧进每会话首轮）| 全局注入；每个 agent 独立收到 |
| `dsh-mcp-client`（`mcp-semantica` 实例）| host | 工具注册（`mcp__semantica__*` 挂 `ctx.tools`）| 全局注册，所有会话可见；可 `tools.restrict` 屏蔽 |
| `dsh-approve`（自研）| client-only | UI slot：`conversation.composer` 链按优先级接管审批面板（遮蔽核心面板）| 每浏览器页面各一实例；审批数据走宿主 approval carrier |
| `dsh-vision-router` | 双层 | 工具注册（14 个 `vision_*` 工具）+ 事件监听 + UI（工具链编排："vision_ground → vision_crop → vision_describe → vision_pixel_diff → fix → screenshot"）| "All fourteen deep tools are registered from plugin startup, so text and image turns can call them immediately" |
| `dsh-restart` | 双层 | 工具（`restart_harness`：agent 可安排进程重启）+ UI/命令（`/restart`）+ web 路由 | 未提及 |
| `@openviking/dsh-memory-plugin` | 双层 | 工具注册（7 个 `viking_*`）+ 事件监听（`agent/pre-step` 检索并追加 recall 消息）+ prompt 注入 + 服务（openvikingMemory）| "Each DSH session maps to dsh-\<session-id\>"；`OPENVIKING_RECALL_PEER_SCOPE`：`all`=跨工作区 recall / `actor`=隔离 |
| `dsh-prompt-enhancer` | 双层 | UI slot（✨/🎤 按钮）| 未提及 |
| `dsh-better-sidebar` | 双层 | UI slot（侧边栏框架）+ 服务（`ctx.betterSidebar`）| "状态按会话持久化 localStorage" |
| `dsh-rewind-plugin` | 双层 | 工具注册（把模型上下文裁回更早的用户消息）+ UI（逐消息 ↶ 按钮）| "In-place, same session/window — the model-visible surface is cut back to the target" |
| `deepseek-idesign` | 双层 | UI（iPolloWork Design 视图）| "Projects stay under `design/<sessionId>/`" |
| `dsh-client-auto-continue` | 双层 | 事件监听（session event firehose）+ UI | "exactly one engine, regardless of how many tabs are open"；per-session pauses；续跑文本经 `agent.followup` 发送（对主 agent 本身调用延续协议）|
| `meow-smooth` | 双层 | 事件监听（宿主审计）+ 反向代理压缩 `/api/*` JSON + 提醒 | "多会话同时跑也不错过"（per-session 提醒）|
| `dshmarket` | 双层 | web 路由（安装/卸载端点）| 未提及 |
| `@michengai/dsh-skills-manager` | 双层 | web 路由（skill 文件操作 API）| 未提及 |
| `@kenz1117/dsh-ui-usage-billing` | 双层 | 服务（`ctx.billingMetrics`）+ web 路由（`/api/billing/*`）| 未提及 |
| `dsh-net-proxy` | 双层 | 全局修改（fetch 拦截，`/_dsh/net-proxy`）| 未提及 |
| `dsh-opencode-palette` | 双层 | 全局修改（主题皮肤切换）| 未提及 |
| `@tt-a1i/archify-dsh` | host-only | Skill 提供（filesystem Skill provider）| 未提及 |
| `dsh-plugin-zen-useragent` | host-only | 全局修改（启动时 requestHeaders 补丁）| 未提及 |
| `dsh-anthropic-fonts` | client-only | 全局修改（CSS 变量覆盖）| 未提及 |

读法：这张表把 §4 的规则落到了你实际在跑的插件上——**工具类插件默认全局可见（所有并行主 agent 都拿到）**，要"只给某个会话"就得走 scoped 注册或 restriction；**UI 类插件天然页级**，不随会话隔离（少数如 better-sidebar 会把状态按会话存 localStorage）；**事件类插件（auto-continue）是"进程一个引擎、会话各自暂停"的典型**，且它经 `agent.followup` 给主 agent 发续跑文本——说明 followup 延续协议不只服务 subagent，插件也可用于主 agent 本身（进程内唯一引擎、按会话调度）；**memory-plugin 是最接近"会话级"的**——session 映射 + recall 作用域（`all`/`actor`）可配；**自研 dsh-approve 是"宿主审批数据 + 客户端 slot 面板"双层插件**，会话归属体现在审批请求数据而非 UI 实体上。

### 4.7 讨论素材：如果真要"组级隔离"（插件只给一组会话），有几条路

先摆事实：**进程 = 一个 profile 组合**。`dsh web`/`dsh desktop` 各自 boot 一个 profile（bundle 栈 + `cordis.patch.yml` patch 层），一次进程只装载一个组合；你现在 8 个 workdir 会话共用 web profile，全部共享同一组合。`$DSH_HOME` 可覆盖（`packages/util/home-paths`：显式配置 > `$DSH_HOME` > `~/.dsh`）。

| 路径 | 做法 | 优点 | 代价/限制 |
|---|---|---|---|
| **现实路径 0：多进程 / 多 profile**（今天就能用）| 不同项目组跑不同 dsh 进程：组 A 用 web profile，组 B 用 `DSH_HOME=/path/groupB-home` + 自己的 profile | 隔离最彻底（组合、会话、设置全分开）；零核心改动 | 多进程常驻、多端口、会话不互通；天然符合"项目组"边界而非"会话组"边界 |
| **设计素材 A：preset 约定** | 每会话创建时按「组」选不同 agent preset（composeFrom 注入组插件集）| 不动核心，纯组合层 | 只作用于创立窗口；子 agent 不继承仍要显式 compose；session 事后改组麻烦 |
| **设计素材 B：scope key 泛化** | scope key 从「live Agent 对象」扩展出「组对象」：注册到组键，组内所有 agent 可见 | 语义最正，agent-scope 体系自然延伸 | 动 `scope` 包与所有注册表（工具/事件/提示段），影响面最大 |
| **设计素材 C：restriction 配置化** | 不做新维度；把「哪些 session 禁哪些全局工具」写成 per-session 限制规则（`tools.restrict` 的表达力）| 核心不动，纯策略；可仅对个别会话关掉全局噪音工具 | 只能删不能加；对非工具能力（UI、服务、事件）无效 |

> 表内为**讨论素材**，不是实施建议。我的倾向：先确认问题规模（开放问题 6）——若只是"个别全局工具吵到别的会话"，方案 C 就够；若真是"插件 A 只服务项目组 X"，现实路径 0（多进程）今天零成本可达，设计素材留作长期议题。

---

## 5. 外部 / 并行 agent 接入：同进程、同 scope 规则、插件随组合走

"多个主 agent 并行"除 GUI 多会话外，还有三条接入通道，全部落在**同一个 host 进程、同一个 context、同一套 scope 规则**上：

| 通道 | 本质 | 与插件的关系 |
|---|---|---|
| Web GUI 多会话 | 每个浏览器会话一个独立 agent（本机现状：8 个 workdir 会话同挂一个进程）| 插件按 §4 的 global / scoped 规则可见 |
| **ACP**（`dsh-acp`，自动化专用 JSON-RPC/stdio 服务器）| 程序化客户端（含 `subagent-acp` 子 agent provider）`session/new` 创建全新 harness agent，多 session 并发；每个 session 独立 prompt slot、workspace、cancellation、disposer；连接断开只清自己拥有的一支，其他前端共享 context 的 continuable forest 不受影响 | agent 是普通 harness agent → **插件随进程组合（composition）提供**；ACP 不额外声明 MCP/tool 能力，工具可见性完全由该 agent 的 scoped ctx 决定 |
| API 网关（Typert / api-gateway，总线 `Connection` + `/api` 路由）| 浏览器/外部程序按 `sessionId` 调 Remote 方法；`agentFor()`：复用 live Agent、自动 resume 冷会话、去重并发 resume、**拒绝 subagent 路由的身份**（子 agent 不对外暴露）| 插件可注册 `@Remote` 方法，成为"外部可调用的 agent 能力"；`@RemoteScope` 直接以 Host Context 为作用域 |

要点：

1. **没有"多进程集群"**：所有并行 agent 共享一个 host 进程和一个 Cordis context。隔离靠 scope（对象身份），不靠进程边界。
2. **外部接入不绕过插件**：ACP 创建的 agent 走 `ctx.agents` 常规创建，全局注册的工具/服务天然可见，scoped 之物按需经 preset/`agentPresets.composeFrom` 显式组合。
3. **per-connection 修剪**：ACP 断开时只 dispose 自己连接下的 owned Agents 及其 continuable 后代——这是"并行 agent 各自生命周期独立"在代码里的直接证据（后续接插件时可放心，不全局清场）。
4. 其他并行形态：**后台 jobs**（`ctx.jobs`，agent 侧异步工作）、**workflow 工具**（`agent()` 调用经 `ctx.subagents` fan-out）、**Ralph**（fresh-agent 轮次）、**headless profile**（`dsh-headless`：无服务器的 one-shot runner 组合）、以及实验性的 **agent team**（roster + task board + mailbox，基于 continuable subagent 的协调层）。它们都是"同一进程内多个 agent 实例"的编排语法，插件交互仍走 §4 的规则。

---

## 6. 留给讨论的开放问题

1. **"主 agent"的口径**：是 UI 会话里的那个角色，还是代码里 `roots()` 意义上的顶层 agent？headless 会话、Ralph 轮、ACP 接入的 agent，算不算"主"？建议约定一个统一口径，讨论才不跑偏。
2. **你说的"多个主 agent 并行"指哪种**：(a) 单进程多会话（现状，8 个 workdir 会话同挂一个进程）；(b) 多进程/多机（cluster，暂无文档支持）；(c) 单会话内多个独立推进线（continuable subagent 并行、workflow fan-out、agent team）。三种的插件交互规则完全不同，先对齐场景。
3. **插件作用域缺"一组"档位**：现状只有 global / 单 agent 两档 + 动态插件按 session 归属。若要"插件 A 只作用于这 3 个会话"，需要新的 scope 维度或约定（比如统一 preset 组合），值得讨论值不值得做。
4. **child 不继承**是刻意的（安全/隔离），但意味着"插件能力传递"要靠 preset。你的流程推演场景里，子 agent 需要哪些父插件能力？这决定 composeFrom 的用法建议。
5. **人类身份 vs agent 身份**：interrupt 授权区分 `user`（父 session 地址）与 `ancestor`（live agent）；并行场景下"哪个 agent 可以打断/指挥哪个 agent"的授权树，是不是你关心的边界？
6. **你的 20 个插件里，哪些你希望"只对特定会话生效"？**（比如 semantica MCP 工具只想留给流程推演会话、vision-router 只想留给有图的需求）——这条直接决定 §4.5 缺的"一组"作用域档位值不值得做。
7. **工作台模式下，角色隔离放哪层？**（见 §8）你们现在是"工具全局可见 + wb_my_task 数据层路由"。要不要部分工具按角色做 scope 隔离，还是保持数据层分工？——这是把 §4 讨论落到你们系统时的关键分叉。

---

## 8. 与你自己的工作台系统（团队项目工作台）的对照

> 这一节把讨论从"DSH 原生机制"接到你已经在跑的系统上。事实来自 `~/workdir/test-flow/workbench-plugin/host.js`（22 个 `wb_*` 工具注册，含 `wb_my_task` 角色路由）与 OpenViking 记忆（四层架构决策）。

### 8.1 你的四层架构 ↔ DSH 概念映射

| 你的层 | 你的实现 | DSH 里的对应物 |
|---|---|---|
| 制度层 | team preset（只读：流程模板/门禁/角色卡）| Cordis 插件组合 + preset（制度=组合的一部分）|
| 交互层 | wbflow 插件（Host + Client）| DSH 插件：`host.js` 里 `harness.registerTool(ctx, def)` + `ctx.effect` 包裹；`client.js` 里 `slots.register` 挂 UI |
| 数据层 | `.workbench/*.json` → sqlite 11 表 | 领域数据（挂在 Session 之外的工具内校验路由）|
| 执行层 | Leader（主 agent）+ 角色子 agent + `wb_*` 工具 | 主 agent 用 `ctx.subagents` 派子 agent；工具全部经 `ctx.tools` 全局注册 |

### 8.2 三个关键对应（讨论的分叉点都在这）

1. **角色是数据，不是实体**。你们系统里 Leader/PM/RD/FE/QA/Architect 是 `wb_my_task(reqId, role)` 的**参数**——同一套 `wb_*` 工具，按 role 返回不同的 roleObjective/behaviors/可见产物。DSH 端实体只有两类：主 agent 与 subagent；**"谁是什么角色"由数据路由赋予，不由架构实体决定**。这正好呼应本文开头"agent 决定谁在干"——在你们系统里"角色"是数据的切片。
2. **隔离分层是互补的**。你们今天的角色隔离=**数据层路由**（wb_my_task 按角色过滤可见内容），而所有 `wb_*` 工具对每个 agent 全局可见（scope 层无过滤）。DSH 的 scope 管"工具*存在与否*"，你们的数据层管"工具*返回什么*"——两层正交互补。这正是 §4.4「全局注册是并行时唯一真正的跨 agent 通道」的具体化解法：**并行需求再多，wb_* 工具只有一份，隔离发生在 reqId×role 的数据切面上**。
3. **你们的执行闭环，DSH 已原生具备**。docs-tmp/execution-loop.md 里"agent 发起任务→subagent 执行并返回→agent 再调度"对应 DSH 的 continuable subagent：`followup`（消息入 FIFO inbox）= 再调度、`interrupt` = 停回合、`reportFrom` = 子 agent 主动回报、`subagent-settled` = 落幕通知。你们自建轮子之前，这些已在使用中的协议里。

### 8.3 落到"多个主 agent 并行"的猜想（待你确认）

按你们系统的形态，你说的"多个主 agent 并行"最可能是以下之一：

- **(a) 多需求并发**：多个需求同时推进，每个需求由 Leader（主 agent）+ 一组角色子 agent 执行——DSH 单进程多 session 天然支持，`wb_*` 全局共享、数据按 reqId 隔离；
- **(b) 单需求内并行**：一个 Leader 同时派多个角色子 agent（如 RD/FE 并行开发）——continuable subagent 并行即可，各子 agent 是独立 session/独立扁平 scope（不继承父的 scoped 注册，但 `wb_*` 是全局注册所以天然可见）；
- **(c) 多工作台实例**：不同项目跑不同 wbflow 实例——注意：wbflow 是**一个进程内一个实例**的插件；多项目共存于同一 profile 时共享同一套 `wb_*` 工具（数据层按项目目录隔离）；要彻底隔离走 §4.7 现实路径 0（多进程/多 profile）。

无论哪种，**结论一致：插件层不需要为并行做任何事——工具全局一份，并行隔离全部发生在数据层**。这大概也是当初 `program_routes_agent_professional` 决策（程序做路由、agent 只留专业能力卡）能成立的原因。

---

## 9. 证据索引（主要论断 → 出处）

| 论断 | 出处 |
|---|---|
| 产品每一部分都是插件（模型适配器/工具注册表/会话日志/agent loop）| `docs/architecture.md`（"Every part of the product is a plugin…"）|
| 顶层 agent（无 parentSession）≈ 主 agent；`roots()` 列出全部 | `docs/subsystems/core.md`（`ctx.agents.roots()`）|
| 一个 agent ≡ SessionId + session log + scoped ctx + 回合循环 | `docs/subsystems/core.md`、`docs/glossary.md`（turn/step）|
| subagent 是可选 seam，多 provider 并存（spawn/fork/acp/codex/claude-code/dsh-sdk）| `packages/subagent/README.md` 表格 |
| child 全新扁平 scope，不继承父 scoped 注册；`composeFrom` 是显式传递手段 | `docs/subsystems/subagent.md`（"Each child gets a new flat scope…"）、`core.md`（`agentPresets.composeFrom`）|
| one-shot vs continuable；Activation/inbox FIFO/`followup`/`interrupt`/`reportFrom` | `docs/subsystems/subagent.md` |
| 作用域两层扁平；scope key = live Agent；scoped dispatch；shadowing；restriction | `docs/glossary.md`、`docs/subsystems/scope.md` |
| 动态插件（tool-cordis）按 session 归属、跨 session "读作不存在"、运行态进程内存 | `packages/extensions/tool-cordis/README.md`、`cordis-host-runner/README.md`（逐字核验）|
| 工具执行流水线 pre-execute→guards→execute→post-execute→`tools/result` | `docs/tool-execution-pipeline.md` |
| cordis 事件四种分发模式、可逆 effect | `docs/cordis-primer.md` |
| ACP：自动化专用 JSON-RPC/stdio；一连接多 session；每 session 独立 slot/workspace/disposer；断开只清本连接拥有的 agent | `packages/acp/acp/README.md` |
| api-gateway `agentFor()`：复用 live、resume 冷会话、拒绝 subagent 路由身份 | `docs/api-gateway.md` |
| MCP client 把外部 server 工具注册到 `ctx.tools`（`mcp__<server>__<tool>`）| `packages/mcp/mcp-client/README.md` |
| AGENTS.md 由 dsh-base 内的 `dsh-agent-instructions` 注入 | `packages/bundle/base/cordis.patch.yml` |
| dsh-approve：client 半边接管 `conversation.composer` slot（优先级 0.5，遮蔽核心面板）| `~/workdir/dsh-plugin/packages/dsh-approve/src/client/index.ts` |
| web profile 插件清单 | `~/.dsh/profiles/web/package.json`（bundle 列表）+ `cordis.patch.yml`（mcp-semantica 等）|
| 进程 = 一个 profile 组合（bundle 栈 + patch 层，一次 boot 装载一个组合）| `apps/cli/src/profile-boot.ts`（"resolve the profile, stack its patch layers"）+ `docs/architecture.md`（profile 定义）|
| `$DSH_HOME` 覆盖生效（显式配置 > `$DSH_HOME` > `~/.dsh`）| `packages/util/home-paths/src/index.ts`（`resolvedHome`）|
| 本机另有 desktop profile（第二个独立组合）| `~/.dsh/profiles/desktop/` |
| §4.6 的 18 个插件端/挂接点分类 | 各插件 README/package.json，位于 `~/.dsh/profiles/web/node_modules/`（分类子 agent 全查，v0.4）|
| wbflow 宿主注册 22 个 `wb_*` 工具（`harness.registerTool` + `ctx.effect`），`wb_my_task(reqId, role)` 做角色路由 | `~/workdir/test-flow/workbench-plugin/host.js`（工具表 + `opMyTask`）|
| wbflow 客户端挂 UI（`slots.register`）| `~/workdir/test-flow/workbench-plugin/client.js` |
| 工作台四层架构（制度 preset / wbflow 插件 / 数据层 / Leader+角色子 agent）与相关决策 | OpenViking 记忆「团队项目工作台」（adopt_four_layer_architecture、program_routes_agent_professional 等）|