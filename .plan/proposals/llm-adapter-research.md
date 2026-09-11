# novel LLM 通路取证调研：直接 copy 清单 + 适配点映射（供 wbflow rd spec 注记 -L4）

> **⚠️ 定位修正（2026-09-12 User 澄清）**：novel 侧 DSH 只是模型端点工具；**wbflow 相反——DSH 是宿主与地基，项目目的就是嵌入 DSH**。本调研只 copy novel 的**传输机制实证件**（围栏剥离/退避/DisableKeepAlives 等），定位随 DSH：adapter = 「DSH 模型通路的客户端」，baseURL 默认=本地 DSH 网关（http://127.0.0.1:3080），模型路由/账号/鉴权归 DSH 管，不照搬 novel「外部端点直连」的用法。

调研对象：`/Users/huzilin/workdir/novel`（be/llm 包 + service 调用链 + config）。
日期：2026-09-11。性质：全部为「文档已写明/代码已写明」，逐条标 file:line；推断处显式标注「推断」。

---

## A. adapter 核心机制拆解

### A1. 传输内核 ChatClient（`novel/be/llm/chat.go`，全文件 121 行）

- 端点拼法：`POST {baseURL}/chat/completions`，baseURL 构造时 `strings.TrimSuffix("/")`（chat.go:58, 84）。注释明确「OpenAI-compatible Chat Completions 调用（DSH 对外形态）」（chat.go:16-17）。**不是** `/v1/chat/completions`——`/v1` 属于 baseURL 的一部分（见 E 节实例）。
- 鉴权：`Authorization: Bearer <apiKey>`，apiKey 空串则不设该头（chat.go:89-91）。
- 请求结构：`chatRequest{Model, Messages[]}`，Messages 恒两条：system + user，单轮、无多轮历史、无流式（chat.go:19-27, 77-80）。
- 响应结构：`chatResponse{Choices[].Message.Content, Usage{prompt_tokens, completion_tokens, cost_yuan, cost}}`；cost_yuan 为 0 时回落 cost 字段，两者都无则 0「不编造」（chat.go:29-40, 112-115）。
- 关键工程决策——`DisableKeepAlives: true`（chat.go:62-67）：LLM 网关断连后 Go 连接池持续复用半开连接，表现为每次请求挂起直到超时（真机 E2E 实测；curl 每次新连接故无此症）。调用频度低，禁复用无成本。**copy 时必须保留**。
- 超时：整个 http.Client 挂 `Timeout`（config 传入，默认 60s，本地实际 120s）。
- 错误口径：连接失败/非 2xx/解析失败/空 Choices 全部归 `domainerr.ErrLLM`（502 语义）；非 2xx 时 `io.LimitReader(resp.Body, 512)` 截断错误体（chat.go:96-111）。
- `Model()` 自报模型名（chat.go:72，埋点分组用）。

### A2. 业务 adapter DSHClient（`novel/be/llm/dsh.go`，335 行）

- `DSHClient{chat *ChatClient}`，全部通路方法最终都走 `chat.Complete(ctx, system, user)`（dsh.go:14-29）。
- `Complete`（dsh.go:27）：裸补全直通。BUG-2 教训：此前 DSHClient 未实现 Complete，装配处类型断言失败返 nil，char_check 自检真机静默空转——**接口实现完整性要有编译期断言**（fake.go:76-80 有 `var _ Completer = (*DSHClient)(nil)`）。
- `GenerateDraft`（dsh.go:31-37）：system 写死网文域角色 + 服从约束；user 用 `fmt.Sprintf` 拼「大纲快照/persona/目标字数/其他上下文」，要求「仅正文，无说明」。
- `RunReview`（dsh.go:39-73）：W3 prompt 定形——system 内嵌完整 JSON 形状契约 `{"pass":bool,"classification":"...","findings":[{"desc","scope","lineFrom","lineTo"}]}` + 硬约束（findings 必须数组、每条 desc 一句中文人话 ≤80 字、desc 内禁 JSON/围栏/键值对、行级定位 scope=line）。真机缺陷实证：旧 prompt 让 findings 出字符串，模型把整段 findings JSON 塞进单条 desc 原样落库。解析：`StripFences` + Unmarshal，失败报「审校产出非预期 JSON」+ truncate 200 字。
- `StripFences`（dsh.go:108-124）：剥 markdown 围栏与首尾杂 text，取**首个 `{..}` 平衡块**（按大括号配对深度，不依赖 ``` 字面量——真机 GLM 实测会包 ```json）。
- `StripFencesAny`（dsh.go:303-334）：同上但支持 `{` 或 `[` 顶层，带字符串内转义感知（inStr 状态机）；推演产物可为裸数组。
- `parseReviewFindings`（dsh.go:88-104）：宽口径——数组形状优先，字符串形状回退单条纯文本意见不中断审校，null/空/缺失归空数组。
- `ReviseDraft`（dsh.go:190-299）：双段输出解析——正文纯文本 + 末尾 ```json 围栏 fixes 处置映射；`parseReviseOutcome`（211）定位末尾块，块缺失/解析失败降级 `Fixes=nil`（服务层全部落 pending_recheck 兜底），**解析失败不中断管线**；`parseReviseFixes`（263）逐字段宽松（commentId 数字或字符串、fixed 缺省 true——「AI 声明不可信」）。
- `EvaluateReject`（dsh.go:163-182）、`Derive`（126-151）：同款「system 定形状 + StripFences + 宽解析」。
- `truncate`（dsh.go:153-158）：错误信息截断工具。

### A3. 接口与类型层（`novel/be/llm/llm.go`）

- `Client` 接口（llm.go:97-108）：GenerateDraft / RunReview / Derive / EvaluateReject / ReviseDraft，每方法返回 `(产物, Usage, error)`。
- `Usage{PromptTokens, CompletionTokens, CostYuan}` + `IsZero()`（llm.go:84-93）：零值不回写。
- `Completer` 最小接口（fake.go:72-74）：`Complete(ctx, system, user)`，与完整 Client 解耦，供起章名/自检等旁路。
- 各通路 Request/Outcome 结构体（llm.go:10-79）：入参全是 string 字段（调用方自行序列化 JSON 素材），**llm 包不 import 业务包**，形状以注释对齐（如 reviewFindingJSON 与 service.ReviewFinding 契约一致，dsh.go:75-76）。

### A4. 埋点出口（`novel/be/llm/telemetry.go`，可选件）

- `CallContext`（telemetry.go:69-77）+ `WithCallContext`（82）：业务分组维度（书/章/草稿/轮次）经 ctx 携带，调用点发请求前挂上。
- `InstrumentedClient`（132）：装饰器包装任意 Client，逐方法计时/取用量/判结果回调 Sink；sink nil 直通零开销；emit 有 panic 隔离（154-160）。
- `resultOf`（181-193）：超时特征识别（`Client.Timeout exceeded` / `context deadline exceeded`）归 timeout，其余 failure。
- `Sink` 接口（52）+ `CallRecord`（34）：落 ai_call_log 表的载体。
- `LabeledCompleter`（100）：共用 Complete 通路的旁路用途标注（char_check/name），业务代码零改动。

### A5. 测试替身（`novel/be/llm/fake.go`）

- `FakeClient`（fake.go:10-68）：各通路固定产物/固定错误/固定 Usage 注入；Complete 空串回落 DraftText。colleague 规范「假模型返回确定产物」的标准实现。

### A6. 装配（`novel/be/main.go`）

- main.go:114-131：`cfg.LLM.DSHBaseURL != ""` 时用 `NewInstrumentedClient(NewDSHClient(...))`；否则全通路 FakeClient + WARN 日志（演示态）。**「baseURL 空 = 替身演示态」是明确的降级开关**。
- 起章名旁路单独包 `NewInstrumentedCompleter(NewChatClient(...))`（main.go:119-120）。

---

## B. produce/review 链路调用图

### B1. produce（创作）链

```
HTTP POST（作者点生成）
  → Writer.DraftChapter（writer.go:68）
      校验章归属（outline.ChapterByLocator）→ 读当前版本号（drafts.CurrentVersion）
      → payload{workId, chapterLocator, baseVersion, context, reason} JSON 序列化
      → taskRepo.EnqueueWork(kind=draft, workID) 归书入队 → 返回 202 task_id（异步）
  → task.Worker 轮询 Claim（worker.go:85-105，500ms 轮询 + CAS 抢单）
  → Writer.DraftTaskHandler（writer.go:93）
      1. 解 payload → 章定位
      2. tryTransit(AIClaim) 章节状态机推进（writer.go:104）
      3. 版本冲突判定：p.BaseVersion != cur → ErrVersionConflict 终态（writer.go:109-112）
      4. refs 存在性预检（LLM 前 fail fast，不烧模型成本，writer.go:113-125）
      5. llm.WithCallContext 挂业务分组（writer.go:128-130）
      6. llm.GenerateDraft(ctx, DraftRequest{ContextJSON})（writer.go:131）
      7. recordUsage 用量累加进 task.usage（writer.go:135，usage.go:16-25 best-effort 旁路）
      8. 同事务原子段：InsertNewVersionInTx（插新版+推游标）+ refs 逐条 Record
         （writer.go:149-165；BUG-6 教训：插版与登记同一事务，失败整体回滚不留脏版本）
      9. enterAIReviewAndFlow 自动流转审校（writer.go:172）
      10. 返回 draftResult JSON → worker MarkDone
```

prompt 素材来源：`context` JSON 由**调用方（handler/前端）组装**传入（大纲快照/persona/节拍窗），refs 清单是 context.refs 可选键（writer.go:245-262）。novel 没有从上游表读素材拼 prompt——上游素材全在 payload.context 里。

### B2. review（审校）链

```
HTTP POST review
  → Reviewer.Review（reviewer.go:129）
      draft 存在校验 → 逐层轮次预检（任一层达 5 轮上限即 ErrRoundLimit 不入队）
      → EnqueueWork(kind=review) → 202 task_id
  → Reviewer.ReviewTaskHandler（reviewer.go:156）
      for layer in [L1 结构, L2 场景, L3 钩子, L4 文风]（reviewer.go:175）:
        1. CountByLayer → round = n+1，超 5 轮 ErrRoundLimit 终止（reviewer.go:176-183）
        2. layerMetrics 确定性本地量化诊断按层挂载（reviewer.go:282-327）：
           L1 节拍窗/情感弧/alerts；L2 场景三件套+未收伏笔清单；L3 钩子标量；L4 风格标量
        3. llm.RunReview(ctx, ReviewRequest{Body, Layer, RoundNo, MetricsJSON})
           （reviewer.go:195-197）
        4. recordUsage 逐层逐次累加（reviewer.go:201）
        5. parseFindings 宽解析（reviewer.go:527-536）；分类缺省/未知兜底归 taste（208-210）
        6. fileOpinions 意见落 review_opinion（state=pending），幂等键
           review_op:{draft}:{layer}:{round}:{index} 防重放双写（reviewer.go:363-391）
           desc 落库前 sanitizeFindingDesc 净化（415-431：JSON 塞 desc 时剥围栏提取人话，
           提不出降级占位文案+原文留档）
        7. 分类分流：mechanical → system 代用户自动采纳（意见→批注→agreed）；
           taste → pendingAuthor=true 送用户终审（224-233）
        8. review_pass 留痕一行 + publishProgress 分块渐进可见
           （reviewer.go:243-250, 492-523：AppendResult + SetProgress 25/50/75/100%）
      finishAIReview 推进章节边 4（reviewer.go:258；中途失败章停在 ai_reviewing）
```

### B3. 失败与重试语义（`novel/be/task/worker.go`）

- 依赖类 kind（`SetDepKind`，LLM 通路属此类）失败走退避重试：`1s×3 → 5s×2 → 30s` 共 6 次，超限 MarkFailed「重试超限」（worker.go:24, 141-153）。
- 确定性终态错误（ErrRoundLimit）不重试直接 failed（worker.go:130-134）；非依赖 kind 失败即错不重试（135-139）。
- handler panic 兜底 MarkFailed（107-116）；进程重启 `RecoverStale` 收尸 running 态（69-71）。
- failed 态任务可人工 `Retry`（repo.go:96-99，CAS failed→queued）。

---

## C. 直接 copy 清单

落点建议：rd 内建 `rd/llm/` 新包（对齐 novel `be/llm` 的包边界），配置件进 `rd/config`，异步骨架按需参考 `rd/service` 现有结构。

| # | 源（novel） | 目标落点（wbflow rd） | copy 方式 | 备注 |
|---|---|---|---|---|
| C1 | `be/llm/chat.go` 全文件（ChatClient/NewChatClient/Complete/chatRequest/chatResponse） | `rd/llm/chat.go` | **原样 copy**，仅换错误构造（domainerr→rd 错误风格）与 import 路径 | DisableKeepAlives、512 字节错误体截断、cost 回落逻辑全保留 |
| C2 | `be/llm/dsh.go` 的 `StripFences`（108-124）、`StripFencesAny`（303-334）、`truncate`（153-158） | `rd/llm/fence.go` | **原样 copy** | 围栏剥离是通用件，无业务假设 |
| C3 | `be/llm/dsh.go` 的 `parseReviewFindings`（88-104）、`reviewFindingJSON`（77-83） | `rd/llm/parse.go` | **原样 copy**（键名按 rd 意见契约改名） | 宽口径解析模式（数组/字符串/null 三收）直接复用 |
| C4 | `be/llm/dsh.go` 的 `parseReviseOutcome`/`parseReviseFixes`/`bodyOrRaw`/`looksLikeFixesJSON`/`stripLeadingFence`（211-299） | `rd/llm/parse.go` | **原样 copy**（若 review 通路需「修订+处置映射」双段输出） | 「解析失败降级不中断」口径随附 |
| C5 | `be/llm/llm.go` 的 `Usage` 结构 + `IsZero`（84-93）、`Completer` 接口（fake.go:72-74） | `rd/llm/llm.go` | **原样 copy** | Client 大接口按 rd 通路裁剪重建，不整抄 |
| C6 | `be/llm/fake.go` 全文件（FakeClient） | `rd/llm/fake.go` | **原样 copy**（方法集对齐 C5 裁剪后的接口） | rd 无 LLM 配置时演示态直接可用（对齐 main.go:121-130 模式） |
| C7 | `be/llm/telemetry.go` 的 `CallContext`/`WithCallContext`/`resultOf`/`InstrumentedClient` 骨架（69-193, 232-300） | `rd/llm/telemetry.go` | **选择性 copy**（rd 有 ai 调用留痕需求才要） | sink 落点改 rd audit/notification 体系 |
| C8 | `be/config/config.go` 的 `LLMConfig`（32-43） | `rd/config/config.go` 增 `LLMConfig` 段 | **结构 copy**，yaml 键名可沿用 | timeout_sec 默认 60、extra_headers、mock 开关三项照搬 |
| C9 | `be/service/usage.go` `recordUsage`（16-25） | `rd/service`（若任务行存 usage） | **模式 copy** | best-effort 旁路、零值不写两条纪律 |
| C10 | `be/task/worker.go` 退避重试骨架（backoff 序列/SetDepKind/RecoverStale/panic 兜底，20-168） | rd 执行链参考（rd 现为同步 service + scanner 兜底，无 worker） | **模式参考**，非直接 copy（rd 任务模型不同） | 若 -L4 落异步执行链，此文件是完整蓝本 |
| C11 | `be/main.go:114-131` 装配模式（baseURL 空→FakeClient 降级） | `rd/cmd/rd/main.go` | **模式 copy** | 演示态/真跑态一开关切换 |
| C12 | `be/llm/dsh.go` 各通路方法（GenerateDraft/RunReview 等 31-205） | `rd/llm/` | **不 copy**，仅借 prompt 定形纪律 | prompt 是网文域写死文本；rd 的 prompt 要从 capability how 来（见 D6） |

---

## D. 适配点清单（novel 业务假设 vs wbflow 差异）

| # | novel 假设 | wbflow 现状/要求 | 适配动作 |
|---|---|---|---|
| D1 | 产物入库：`drafts.InsertNewVersionInTx` 版本快照表 + 推游标 + 同事务 refs 登记对；失败整体回滚（writer.go:146-168） | rd 走 `wb_add_artifact`（`ArtifactService.Add`，rd/service/artifact.go:62；-K kind=document 受控枚举校验） | produce 链 LLM 产出后调 `ArtifactService.Add`（或同事务 repo 层写入）落 artifact；版本语义由 rd artifact 幂等/版本机制承载，不复刻 draft 版本表 |
| D2 | 上游输入：prompt 素材（大纲快照/persona）由**调用方组装进 payload.context**，service 层不读上游表（writer.go:22, 131） | -L4 明确「读输入=上游入库产物」；rd 已有依赖 doc 解析 `TaskService.upstreamProduceDocs`（rd/service/task.go:155）与 `GetContent` 读产物正文（rd/service/artifact.go:208） | produce/review 链开头**新增「素材组装」段**：按能力声明读上游 artifacts 正文拼 prompt user 段；novel 的 `DraftRequest.ContextJSON` 填法从「外部传入」改「服务端组装」 |
| D3 | 配置：单文件 yaml + env 覆盖 + sync.Once（novel/be/config/config.go:76-125，env 优先防密钥落盘） | rd/config 是主配置+local 覆盖两文件合并 + `${VAR}` 展开（rd/config/config.go:1-10）；敏感值走 `${WBFLOW_*}` 占位 | LLM 三要素进 `rd/configs/config.yaml` 增 `llm:` 段，key 用 `${WBFLOW_LLM_API_KEY}` 占位（对齐 rd 凭据红线）；env 展开机制 rd 已有，无需 env 覆盖逻辑 |
| D4 | 失败语义：LLM 失败→worker 退避重试 6 次→failed 终态→人工 Retry（worker.go:24, 141-153; repo.go:96-99） | -L4 要求「LLM 失败→任务回退+need_intervene 人工接管」；rd 已有回退机制（RollbackForRejectTx，rd/service/advance.go:374）与 need_intervene 通知枚举（NotificationKindNeedIntervene，rd/service/notifier.go:50；scanner.go:186-192 已有产生路径） | LLM 错误（ErrLLM 等价物）映射到：任务态回退（复用 rd 回退核）+ 写 need_intervene notification；是否保留退避重试需拍板（novel 重试在回退之前，rd 可先重试 N 次再回退+通知——此为 gap，无现成代码） |
| D5 | 错误载体：`domainerr.ErrLLM` 哨兵 + Code（novel/errors） | rd 错误风格为 service 层构造（pingcap/errors + handler 层状态映射） | copy chat.go 时把 `domainerr.New(domainerr.ErrLLM.Code, ...)` 换成 rd 错误构造；建议 rd 也立 `ErrLLM` 等价哨兵以便 D4 的失败分流判定 |
| D6 | prompt：网文域 system 文本**硬编码在 adapter 方法内**（dsh.go:32, 43-48 等） | -L4 要求「seed capability how=prompt 素材」——prompt 是能力数据，不是代码 | rd 不照抄 DSHClient 方法集；应建「prompt 模板驱动」通用层：adapter 只提供 `Complete(system, user)` + `结构化 JSON 解析件`（C2/C3），system/user 由 capability how + 上游素材在 service 层组装。novel 的各 system prompt 可作为**写法范本**（形状契约内嵌、禁围栏、宽解析三件套纪律） |
| D7 | 审校定位：意见挂 `lineFrom/lineTo` 段落序号 + scope=line/article（dsh.go:44-47；reviewer.go:96-102） | -L3 划词/document 锚（意见挂锚）；rd review 域已有锚体系（AnchorText 等概念在 novel reviewer.go:382 对应 rd 划词） | findings 的 scope/lineFrom/lineTo 需映射到 rd 锚模型（划词偏移或 document 级）；novel「审校无划词恒空串锚、列随表预留」（reviewer.go:382）与 rd -L3 划词要求不同，锚定位是新增 gap |
| D8 | 执行形态：enqueue + 后台 worker 轮询（异步、202+轮询进度、分块渐进 result，worker.go:85-105; reviewer.go:492-523） | rd 现有 service 为同步请求-响应 + scanner 定时兜底（rd/service/scanner.go）；-L4 两条自动化执行链未定同步/异步 | 需拍板：先单节点真跑通（-L4 落地节奏）可先同步（HTTP 请求内完成 LLM 调用+入库），长任务再引 novel 式 worker。**此为架构决策点，非纯 copy** |
| D9 | 埋点：ai_call_log 表 + TelemetrySink + dashboard 成本卡（telemetry.go 全文；usage.go） | rd 留痕体系是 audit_log + notification（rd/service/notifier.go、review.go 的 reviewAudit） | 一期可裁剪 ai_call_log（C7 选 copy）；保留 `resultOf` 超时判定即可支撑 D4 失败分流 |
| D10 | 量化诊断：审校前跑确定性本地 quant 内核按层挂 MetricsJSON（reviewer.go:282-327） | rd 无对应量化内核 | review 通路的「确定性输入随 LLM 请求携带」模式可借（判据素材进 prompt），量化内核本身属 novel 域，不 copy |

---

## E. 配置三要素样例（novel 实际形态，key 脱敏）

来源：`novel/be/config.local.yaml`（8-24 行）+ `novel/be/config/config.go`（31-43）+ `novel/be/llm/chat.go:44`。

```yaml
llm:
  # opencode-go 网关（OpenAI 兼容）。baseURL 含 /v1；代码拼 {baseURL}/chat/completions
  dsh_base_url: "https://opencode.ai/zen/go/v1"
  # key 不落文件：yaml 置空，走 env 注入 LLM_DSH_API_KEY（提取自 ~/.dsh/.credentials.yaml）
  dsh_api_key: ""            # env 注入，形如 sk-****（脱敏）
  model: "deepseek-v4-flash"
  timeout_sec: 120
  extra_headers:
    x-opencode-session: dsh-opencode-go   # 网关路由/缓存必需会话头（非密钥）
  # derive_mock: true         # 替身开关（仅推演域）
```

- 本地 DSH 形态佐证：`baseURL string // 如 http://127.0.0.1:3080（DSH）`（chat.go:44）——即本地 DSH 网关裸 host:port（无 /v1 时拼出 `http://127.0.0.1:3080/chat/completions`）。**端点形态 = OpenAI 兼容 `{baseURL}/chat/completions`，`/v1` 归 baseURL 携带**，无 /v1/models 探活代码（佐证仅此，未实测）。
- 三要素加载优先级：env（LLM_DSH_BASE_URL / LLM_DSH_API_KEY / LLM_MODEL，非空才覆盖）> yaml > 零值（config.go:108-125）。
- rd 侧建议键位（对齐 rd 凭据红线）：`llm.base_url` / `llm.api_key: ${WBFLOW_LLM_API_KEY}` / `llm.model` / `llm.timeout_sec` / `llm.extra_headers`。

---

## F. 与 spec -L4 的对接映射

spec 正本：`/Users/huzilin/workdir/dsh-flow/.plan/wbflow-impl/spec.md` 413 行（原文照抄见该行）。逐句映射：

| -L4 原文（逐句） | novel 对应能力 | gap（rd 需自建） |
|---|---|---|
| 「无论壳形态，exec/review 环节的真执行必须经 DSH 调用底层大模型」 | ChatClient OpenAI 兼容 POST {baseURL}/chat/completions + Bearer（chat.go:76-121） | 无——C1 原样 copy 即满足 |
| 「参照 novel 库 be/llm/dsh.go 模式：rd 服务内建 llm adapter」 | be/llm 包边界：传输内核（chat.go）/业务 adapter（dsh.go）/接口（llm.go）/替身（fake.go）/埋点（telemetry.go）五件分层 | rd 无 llm 包——建 `rd/llm/`，按 C1-C6 copy |
| 「baseURL/apiKey/model 配置化」 | config.LLMConfig + env 覆盖防密钥落盘（config.go:31-43, 108-125）；演示态降级开关（main.go:115-130） | 键位迁入 rd/config 双文件 + `${WBFLOW_LLM_API_KEY}` 机制（D3），env 覆盖逻辑不需要重写 |
| 「prompt 模板驱动结构化 JSON 输出+围栏剥离」 | system 内嵌 JSON 形状契约（dsh.go:43-48）+ StripFences/StripFencesAny（dsh.go:108, 303）+ 宽口径解析（dsh.go:88-104, 211-299）+ 降级不中断纪律 | 通用件 C2-C4 可 copy；「模板从哪来」是 gap——novel 硬编码，rd 要从 capability how 读（D6） |
| 「seed capability how=prompt 素材（读输入=上游入库产物）」 | novel 无对应（素材由调用方传 payload.context，writer.go:22）；只有「LLM 前素材预检 fail fast」纪律（writer.go:113-125） | **最大 gap**：素材组装段需全新——按能力声明读上游 artifacts（GetContent/upstreamProduceDocs）拼 prompt；novel 只提供 DraftRequest 字段形状参考 |
| 「produce 通路：节点激活→LLM 产出文档→服务端入库（wb_add_artifact，-K kind=document）→判据内聚推进」 | writer.go 全链（B1）：入队→预检→调用→计量→同事务入库→状态推进→自动流转下一步 | 入库点换 ArtifactService.Add（D1）；「判据内聚推进」对应 novel enterAIReviewAndFlow 式自动流转，rd 用 advance 域推进——推进语义 rd 已有，串接是 gap |
| 「review 通路：LLM 按评审能力 how 读被评审产物→意见挂锚（-L3 划词/document）+表态」 | reviewer.go 全链（B2）：分层循环→量化挂载→RunReview→意见落库（幂等键）→分类分流（mechanical 自动采纳/taste 送终审） | 意见挂锚换 rd 锚模型（D7）；「LLM 读产物」= 上游产物正文进 prompt user 段（D2 同源 gap）；分类分流（机械/品位）可整体借桜 |
| 「gate 拍板保留 User 专属」 | novel 对应「品位类送作者终审、AI 不代拍板」（reviewer.go:39-40, 232-233；铁律「AI 审校≠批注」reviewer.go:53-57） | rd gate/decision 域已有 User 拍板，无 gap；novel 纪律佐证「LLM 表态≠拍板」 |
| 「LLM 失败→任务回退+need_intervene 人工接管」 | novel：退避重试 6 次→failed→人工 Retry（worker.go:24, 141-153）；ErrLLM 哨兵（chat.go:98） | 失败→**回退**（rd RollbackForRejectTx）+**need_intervene 通知**（rd NotificationKindNeedIntervene 已预留）的串接代码两边都没有，需新写（D4）；「先重试再回退」策略需拍板 |
| 「落地节奏：先单节点真跑通（bus-01 to-brd）再全流程」 | novel 对应经验：main.go baseURL 空→Fake 演示态（121-130）+ worker 先行单 kind 验证 | rd 建议同款：先 to-brd 单节点接真 adapter 真跑，再铺 produce/review 全链 |
| 「自包含 Web 含 LLM 通路（无 LLM 的编排引擎不算可用 MVP）」 | novel 证明全通路替身兜底可行（FakeClient 全通路 + WARN） | rd copy C6 后即得同款演示态能力 |

---

## 附：novel 侧关键文件索引（取证路径）

- `/Users/huzilin/workdir/novel/be/llm/chat.go` — 传输内核（端点/鉴权/请求响应/超时/错误）
- `/Users/huzilin/workdir/novel/be/llm/dsh.go` — 业务 adapter（prompt/围栏剥离/宽解析）
- `/Users/huzilin/workdir/novel/be/llm/llm.go` — Client 接口/Usage/各通路类型
- `/Users/huzilin/workdir/novel/be/llm/fake.go` — 测试替身 + Completer 接口 + 编译期断言
- `/Users/huzilin/workdir/novel/be/llm/telemetry.go` — 埋点出口（可选件）
- `/Users/huzilin/workdir/novel/be/service/writer.go` — produce 链服务
- `/Users/huzilin/workdir/novel/be/service/reviewer.go` — review 链服务
- `/Users/huzilin/workdir/novel/be/service/usage.go` — 用量回写
- `/Users/huzilin/workdir/novel/be/task/worker.go` — 异步执行/退避重试骨架
- `/Users/huzilin/workdir/novel/be/config/config.go` + `be/config.local.yaml` — 配置形态与三要素实例
- `/Users/huzilin/workdir/novel/be/main.go`（102-160）— 装配
- wbflow rd 侧对照：`/Users/huzilin/workdir/dsh-flow/rd/service/artifact.go`（wb_add_artifact）、`rd/service/task.go`（upstreamProduceDocs）、`rd/service/notifier.go`（need_intervene）、`rd/config/config.go`、`rd/service/advance.go`（回退核）
