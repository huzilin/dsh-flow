# 提案：编排会话 module（C1 深化）——锁序纪律藏进 implementation，事务开场一处定义

- 状态：**已拍板（2026-09-15「按推荐来」），实现落地**（见 §8 实现记录）
- 前置：C2 已落地（NodeChange/ApplyNodeChangeTx，`812096b2`）——本提案的 Session 写方法直接消费它
- 证据行号以 2026-09-15 工作区为准

## 1. 问题（原文照抄 + 语境）

**锁序纪律不入类型**。repo/flow.go:191 注释：「锁序约定：全部推进路径先锁 flows 行再锁 flow_nodes 行，避免交叉死锁」。该约定只活在注释里，43 处 LockTx 调用点靠人守；reject.go:40-48 实际**先锁节点行再读 flow**——与注释口径相反，靠时序上无交叉未爆。

**编排事务开场手写重复**。「Transaction → 锁 flow → 锁 nodes → notFound 转换」完整开场 6 处：

| 位点 | 形状 |
|---|---|
| advance.go:91（wb_advance） | 锁 flow + 锁 nodes 清单 |
| advance.go:563（SweepParallelJoins 巡检） | 同上 + pending_scope 守卫 |
| reject.go:40（RollbackForReject 带外重放） | **锁序颠倒**（先节点后 flow） |
| review.go:96（wb_review） | 锁 flow + 锁单节点 |
| review.go:301（wb_decision） | 锁 flow + 锁单节点 |
| reproject.go:46（Reproject） | flow **非锁读** + 锁 nodes 清单 |

中段变体 2 处：task.go:455（complete 判据通过后事务内嵌推进——事务中段才锁 flow+nodes）、review.go:211（needAdvance 后补锁读 nodes 清单）。

**提交后 flush 人肉**。review.go:265、task.go:486 在编排事务提交后手动 `s.notifier.FlushPending(ctx)`（全库 6 处，另 4 处在评论/扫描器路径）——漏写=通知延迟到 scanner 补投（30s 窗口）。

## 2. design-it-twice：三个候选形状

### 案 A：包内自由函数（可但散）

`runOrchestrationTx(ctx, r, notifier, reqID, fn func(*session) error)`。能用，但 repo+notifier 依赖每次传参，装配面随位点重复；session 类型也要包级导出给 6 个文件共用。浅。

### 案 B：Orchestrator struct（选定）

```go
// rd/service/orch.go（包内私有，示意非终稿）
type Orchestrator struct { repo *repo.Repo; notifier *Notifier }

// 顶层：事务生命周期 + 锁序 + 未命中转换 + 提交后 flush，一处定义
func (o *Orchestrator) Run(ctx, reqID string, fn func(*orchSession) error) error

// 底层：已开事务中段接管（task complete 内嵌推进形状；锁序同核）
func (o *Orchestrator) Lock(ctx, tx *gorm.DB, reqID string) (*orchSession, error)

type orchSession struct { ... }
//   Flow() *repo.Flow                    // 锁定 flow 基线（reproject 模式=非锁读）
//   Nodes() []*repo.FlowNode             // 锁定 nodes 清单
//   Node(id) (*repo.FlowNode, error)     // 锁定单节点 + notFound 转换
//   Apply(node, byWho, changes...)       // 转 C2 ApplyNodeChangeTx
//   MoveCurrent(target *repo.FlowNode)   // flows.current_node_id 前移 + 审计（收编 advance:433/reject:150 两处手拼）
//   Notify(spec)                         // EnqueueNotificationTx
//   Tx() *gorm.DB                        // 逃生舱：advanceNodeLocked 等收口函数消费
```

- interface 面 = 2 个入口（Run/Lock）+ 7 个 Session 方法；implementation 吸收锁序、notFound 转换、提交后 flush、current_node_id 审计。**深**。

### 案 C：repo 层 FlowStore 接口（否决）

repo 包定义 FlowStore 接口 + 事务变体收编。否决理由：只有一个 adapter（MySQL），接缝假想（一个 adapter=假想 seam）；且把编排语义（锁序、推进守卫）下沉 repo 层错位——repo 不该懂编排。

## 3. 决策清单（推荐分支代拟，逐条可推翻）

| # | 决策 | 取值 | 性质 | 理由 |
|---|---|---|---|---|
| D1 | 收口范围 | 6 处完整开场全收 + 中段变体 2 处（经 `Lock` 接管）；24 处 Transaction 中非编排事务（issues/sessions/requirements/versions）**不收** | 推荐 | 编排锁序纪律只在 flow 域有意义；非编排事务无锁序问题，收了是为收而收 |
| D2 | 锁序统一 | Session 统一「flows → flow_nodes」；**reject.go 颠倒点改序**（先锁 flow 再锁节点） | 推荐，**行为变化点①** | 与 repo/flow.go:191 拍板注释对齐；现状颠倒靠时序侥幸，统一后交叉死锁面反而收窄。锁获取顺序变化=并发行为变化，需 IT 并发用例复验（review 并发 collect IT 已覆盖同锁组） |
| D3 | reproject 模式 | Session 提供 flow 锁定/非锁读两模式，reproject 走非锁读（保持现状：flow 行无并发写面，依赖 nodes 锁） | 推荐，行为保持 | 不改变 reproject 并发语义 |
| D4 | 提交后 flush | `Run` 提交成功后恒 `FlushPending`（Orchestrator 构造注入 notifier）；无通知=一次 pending 轻查询 | 推荐，**行为变化点②**（多一次空查询，无害） | 漏写 flush 从「纪律」变「不可能」；scanner SweepParallelJoins 走 Run 同样 flush（其不产通知=空查询） |
| D5 | 落点 | `rd/service/orch.go` 包内私有（不建新包） | 推荐 | 全部调用方在 service 包内；新包只有 import 纪律收益、零边界收益 |
| D6 | MoveCurrent 收编 | `sess.MoveCurrent` 收编 advance.go:433-448 与 reject.go:150-165 两处 flows 手拼（更新+审计 8 行×2） | 推荐 | C2 二期项顺手清掉；编排推进核完全收口入 Session |
| D7 | requirements.status 联动 | 不动（syncRequirementStatusLockedTx 已是函数收口） | 推荐 | 已收口，无重复面 |

## 4. 迁移清单

| 位点 | 动作 |
|---|---|
| rd/service/orch.go | 新建：Orchestrator/Run/Lock/orchSession |
| advance.go Advance / SweepParallelJoins | Run 改写（开场 12 行→1 行）；current_node_id 手拼→MoveCurrent |
| reject.go RollbackForReject | Run 改写 + 锁序改序（D2）；flows 手拼→MoveCurrent |
| review.go wb_review / wb_decision | Run 改写（开场+flush 收编，:265 FlushPending 删）；needAdvance 补读→sess.Nodes 复用 |
| reproject.go | Run（flow 非锁模式）改写 |
| task.go complete 内聚推进 | `Lock` 中段接管（事务已开）；:486 FlushPending 删 |
| 装配 | main.go 各 service 构造注入 Orchestrator（或包内单例构造函数） |

## 5. 测试策略

- 派生核无新纯函数（锁序/事务不可纯函数化），**以既有 IT 为回归门**：advance/reject/reproject/review/task/audit IT 全绿=行为保持；
- 并发 IT 复验锁序改动：ReviewConcurrentCollect（同锁组双 collect）+ 并发回退用例（reject IT 已有幂等重放）必须全绿；
- 新增单测面有限（orchSession 是薄装配层），不强行造 mock（一个 adapter 不立接缝）。

## 6. 风险与边界

1. **锁序改动（D2）是唯一并发语义变化**：deadlock 面理论收窄，但需真库并发 IT 复验；若发现历史路径依赖旧序（无证据），回退该点单独拍板。
2. **flush 恒发（D4）**：scanner 路径多一次空查询/轮，量级可忽略（扫描周期 15s）。
3. 二期（不承诺）：非编排 24 处 Transaction 的开场样板若也想收，等第二个真实形状出现再说（防过早抽象）。

## 7. 拍板点

D1-D7 全部按推荐拍板（2026-09-15「按推荐来」）；semantica `d6236b5e`。

## 8. 实现记录（2026-09-15）

### 8.1 实现期澄清（三处，提案遗漏/口径精化）

1. **task complete 保留其自身 FlushPending**：提案迁移清单写「:486 FlushPending 删」，实现期澄清——task 事务先锁任务行（锁序不得换 run 的 flow 先锁），走 `lock` 中段接管而非 `run`，事务生命周期归调用方，提交后 flush 随之自留（run 恒 flush 口径只覆盖 run 路径）。
2. **命名与可见性**：类型/构造器/方法全包内私有（`orchestrator`/`newOrchestrator`/`run`/`runFlowRead`/`lock`/`orchSession`），比提案草图的 exported 形态更贴 D5「包内私有」口径；各 service 构造期自建（无状态，main.go 装配零改动）。
3. **锁定读不缓存**：orchSession.nodes/node 每次调用即现锁现读——保住 review needAdvance 推进前重锁清单的「最新基线」语义与 reject 的「拍板写后基线刷新」语义；sweep 对候选集取得后流程消失（notFound）保持容错跳过。

### 8.2 变更面

- 新建：`rd/service/orch.go`（orchestrator run/runFlowRead/lock + orchSession：flow/nodes/node/apply/moveCurrent/notify/txHandle）。
- 迁移 7 个入口：advance（Advance/Sweep）、review（Review/Decide，人肉 flush 删 2 处→run 恒 flush）、reject（RollbackForReject，**锁序改序 D2**）、reproject（runFlowRead D3）、task（complete 内嵌推进 lock 接管）。
- 签名收编：advanceNodeLocked/advanceAfterGateLockedTx/activateAndMoveLockedTx/RollbackForRejectTx 的 `tx+flow` 参数 → `sess`（flow 基线/事务句柄/写入口由会话承载）；advance/reject 的 flows.current_node_id 手拼 ×2 → `sess.moveCurrent`（D6）。
- 范围外留存（D1）：escalation/llmflow/reenter/assign 的单节点行锁——各自事务以自身行锁为并发基线，非编排 flow 基线锁，不属锁序纪律管辖。
- 服务构造器增 orch 字段：Advance/Review/Flow/Task（其余 service 不涉编排事务）。

### 8.3 闸门

gofmt/vet/build 零输出；`go test ./...` 全绿；真库 IT 全量（重点并发：ReviewConcurrentCollect 同锁组双 collect、reject 幂等重放）结果见执行记录——存量失败 `ReviewClarifyLoop`（HEAD 对照已证实与两次重构无关）除外须全绿。
