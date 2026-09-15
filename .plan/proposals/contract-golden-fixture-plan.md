# 提案：跨语言契约快照（C3 深化）——Go 真相源导出 golden fixture，双端消费同一份

- 状态：**已拍板（2026-09-15「认可」），实现落地**（见 §8 实现记录）
- 前置：无（独立于 C1/C2，可与已落地两步并行评审）

## 1. 问题（原文照抄 + 语境）

**双语言手写镜像，注释兜底**。FE 生产代码 16 处「同口径/镜像」注释中，真正的跨语言镜像面两类：

一、**flowview.ts 判定/payload 镜像 ×7**（fe/src/domain/flowview.ts）：
- :6 「解析口径与后端同源……rd/repo/payload.go 单一口径镜像」——reviewers/review_results/replies/last_decision 四个 TEXT 列 JSON 形状双语言手写（FE 侧 parseReviewers/parseReviewResults/parseReplies/parseLastDecision）；
- :85 「summarizeReviews 与 rd/domain SummarizeReviewResults 同口径」；
- :165 「inReviewCommentContext……rd/service/comment.go inReviewContext 同口径镜像」；
- :174 「reviewAnchorGuardError……rd 新评论硬拦的 FE 前置同口径」；
- :251 「anchorSummary 与后端 my_task relatedIssues.anchor 同口径」；
- :359/:369 「isLaneSideExecCR/buildFlowRows 与后端 wb_advance join 判定同口径镜像」（:361 自注「未来 seed 若引入相邻 exec-CR 串行对，需后端快照出 parallel_group 标识替代本判定」）。

二、**constants.ts 两张表手抄**（fe/src/api/constants.ts）：
- :7-20 类别绑定表手抄 rd/domain/binding.go:58 CategoryDefinitions（类别→允许流程集+默认流程）；
- :144 NOTIFICATION_KIND_TARGETS 手抄 rd/service/notifier.go:60 notificationTargetTypes（七类通知→跳转目标类型）。

**漂移只能靠注释与 review 纪律兜底**——改词表/改判定时两端各自手改，漏一边=运行期口径分叉（如后端新增通知类别、FE 映射表缺项→收件箱跳转目标错）。

## 2. design-it-twice：三个候选机制

### 案 A：TS 类型 codegen（否决，暂）

从 Go struct 生成 TS 类型/常量。本轮否决：引第三方生成器、类型对齐工程量大；且「类型一致」不解决「判定函数一致」（summarizeReviews 漂移是逻辑不是类型）。留作二期选项。

### 案 B：golden fixture 双端消费（选定）

Go 真相源导出一份 JSON 快照（payload 样例 + 判定函数真值表 + 常量表），**提交进 fe/**；Go 侧 golden 自检测试（默认比对已提交 fixture，`-update` 重写）+ FE vitest 消费同一文件逐例断言。漂移在任一侧测试红灯，不等运行期。

### 案 C：JSON Schema（否决）

schema 只锁形状不锁判定与取值集，覆盖面不足。

## 3. 决策清单（推荐分支代拟，逐条可推翻）

| # | 决策 | 取值 | 性质 | 理由 |
|---|---|---|---|---|
| D1 | 收口范围 | flowview.ts 判定/payload ×7 + constants.ts 两张表；CommentCard 操作可见性等「UI 收敛且后端 409 兜底」类**不进**（低风险，二期观察） | 推荐 | 真相源在 Go 且无兜底拦截的才需要快照；有硬拦兜底的镜像漂移代价=可接受 UX |
| D2 | 机制 | 案 B：`rd/contract/` 新包（生成器+golden 自检）→ fixture 落 `fe/src/api/contract/golden.json`（提交进库）→ FE vitest 消费 | 推荐 | 提交进库=双端不要求同机跑；Go 自检测试使「忘跑 FE 测试」也在 Go 侧红灯；protobuf 已是「单文件双端」先例，模式一致 |
| D3 | fixture 内容三节 | ①`payload`：repo/payload.go 全 struct 的序列化样例（含空值/NULL 边界）；②`verdicts`：判定真值表（输入样例→Go 判定输出：summarize/collectVerdict/inReviewContext/anchorGuard/isLaneSideExecCR+buildFlowRows/anchorSummary 拼装）；③`tables`：CategoryDefinitions + notificationTargetTypes（+ REQ 状态词表顺带） | 推荐 | 三节对应三类漂移面：形状/逻辑/词表 |
| D4 | 断言口径 | verdicts 断言「拦截与否+结果值」，**不断言文案措辞**（FE 引导文案与后端 400 message 允许各自措辞） | 推荐 | 文案是呈现层；锁行为不锁措辞，避免 fixture 变成文案审批 |
| D5 | parallel_group API 出参 | 本轮**不加**（flowview.ts:361 自注的将来项）；fixture 锁「当前 seed 矩阵下双端同输入同判」样例 | 推荐 | 改 API 出参是行为变更，等真需求（seed 引入相邻 exec-CR 串行对）再立 |
| D6 | 再生纪律 | 命令：`cd rd && go test ./contract/ -run TestUnit_ContractSnapshot -update`（重生成）+ `cd fe && npx vitest run src/api/contract`（消费验证）；SOP 写进 qa/README「契约变更」节；Go 侧默认比对模式=漂移自检 | 推荐 | 仓库无 CI，双测试互锁（改真相源必撞 Go 比对红灯） |

## 4. 迁移清单

| 位点 | 动作 |
|---|---|
| rd/contract/contract.go | 新建：真相源装配（引用 domain/service 导出的判定函数与常量表——包内私有者经测试面导出或最小重构） |
| rd/contract/contract_test.go | 新建：TestUnit_ContractSnapshot（默认比对 / `-update` 重写 golden） |
| fe/src/api/contract/golden.json | 新建（生成产物，提交进库） |
| fe/src/api/contract/contract.test.ts | 新建：三节逐例断言（constants 表深相等；flowview 判定函数对真值表逐例跑） |
| qa/README.md | 增「契约变更 SOP」节（命令 + 何时跑） |

注：comment.go inReviewContext / advance.go isLaneSideExecCR 为包内私有——生成器在 service 包内无法直接引用时，以最小导出（或经既有导出面构造样例输入）实现，不为此改动判定逻辑本体。

## 5. 测试策略（即本提案主体——fixture 双测试互锁）

- **Go 侧**：`TestUnit_ContractSnapshot` 默认运行=生成器输出与已提交 golden 比对（真相源被改而未再生 → 红灯强制再生提交）；
- **FE 侧**：vitest 消费 golden——constants 表深相等 + flowview 判定函数真值表逐例（后端改判定未同步 FE → FE 红灯）；
- 既有 flowview.test.ts（469 行行为测试）不动，双层互补：行为层锁 FE 自身语义，契约层锁双端一致。

## 6. 风险与边界

1. **golden.json 进库**：新增一个双语言共享文件；更新纪律靠 §5 双测试互锁，无 CI 下已是可行最强约束。
2. **样例单向流**：真值表输入样例由 Go 侧构造（真相源语言），FE 发现样例缺口 → 改 Go 生成器+重生成，无双写面。
3. **二期（不承诺）**：CommentCard 操作可见性矩阵快照；TS 类型 codegen 替代手写 interface；parallel_group 出参化。

## 7. 拍板点

D1-D6 全部按推荐拍板（2026-09-15「认可」）；semantica `2ef99a5f`。

## 8. 实现记录（2026-09-15）

### 8.1 落地面

- 新建 `rd/contract/`（contract.go 真相源装配 Build() + contract_test.go 自检：默认归一化比对 golden，`-update` 重写）；
- 生成 `fe/src/api/contract/golden.json`（提交进库）；新建 `fe/src/api/contract/contract.test.ts`（9 条：tables 深相等 ×3 / payload 形状锁 / verdicts 真值表逐例 ×4 + 相邻成对并行组样例）；
- 最小导出（判定逻辑零改动）：`comment.go inReviewContext→InReviewContext`、`advance.go isLaneSideExecCR→LaneSideExecCR`、锚守卫提纯为 `ReviewAnchorGuardPre/ReviewAnchorGuardBlocked`（Create 惰性求值边界保留——三项前置命中才解析评审位，损坏 reviewers 数据不在无关路径扩大为错误）、`notifier.go NotificationTargetTypes()` 只读副本出口；
- FE `flowview.ts` 导出 `isLaneSideExecCR`（契约断言消费）；
- qa/README 增「契约变更 SOP」节。

### 8.2 实现期发现（两点）

1. **anchorSummary 排除出契约面（D4 的应用）**：实现期实读发现 Go `SummarizeIssueAnchor`（"range 3-5"）与 FE `anchorSummary`（"line 3-5"）措辞本就不同——它是展示层重渲染而非契约镜像，flowview.ts:251 的「同口径」注释口径过宽（已由本提案排除，注释待后续顺手修）；
2. **泳道判定分歧面比预想宽**：FE 谓词完全不含 scope 校验（凡 produce+gate+评审位即 true），不止 scope=full 一例——golden 以 2 个 `feSkip` 样例（lane 视角、full scope）显式记录该分歧边界，FE 测试跳过并断言其存在。fixture 首轮即逮住真实分歧=机制有效性实证。

### 8.3 闸门

`go test ./...` 全绿（含 contract 比对+更新双模式）；FE `tsc -b` 零错、vitest 182/182（含新 9 条）、build 成功。
