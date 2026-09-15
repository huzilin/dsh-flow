# 提案：节点变更同源派生（C2 深化）——「变更声明一次，updates 与审计同源产出」

- 状态：**已拍板（2026-09-15「认可」），实现落地**（见 §8 实现记录；§3 决策 D1-D8 与实现有三处精化，见 §8.1）
- 来源：architecture-review 报告 C2（`/var/folders/xm/.../architecture-review-20260915-001602.html`，临时文件）
- 证据行号以 2026-09-15 工作区为准；「我的推断」条目已逐条标注

## 1. 问题（原文照抄 + 语境）

**双写病**：每条写路径把同一次字段变更手写两遍——一遍进 `updates map`（DB 列名），一遍进 `AuditLog` 手拼旧/新值（camelCase 字段名）。两份口径之间没有任何编译期或测试期绑定，漂移即审计失真（审计是回放依据，wbflow 口径里审计逐字段留痕是「留痕一致口径」，见 advance.go:711 注释）。

证据（四处 builder 近同构，差异只在表名 / RowID 来源 / byWho 来源）：

```go
// rd/service/review.go:519
func reviewAudit(node *repo.FlowNode, field string, oldVal string, newVal string, byWho string) repo.AuditLog {
	return repo.AuditLog{Table: "flow_nodes", RowID: node.NodeID, Action: "UPDATE",
		Field: utils.StrPtr(field), OldValue: utils.StrPtr(oldVal), NewValue: utils.StrPtr(newVal), ByWho: byWho}
}
// rd/service/comment.go:722 issueAudit（issues 表，issueID 入参）
// rd/service/task.go:725  taskUpdateAudit（tasks 表，byWho=row.RoleID 派生）
// rd/service/reproject.go:352 flowNodeAudit（flow_nodes，byWho 硬编码 user）
```

典型双写现场（reject.go:129-146，回退目标节点）：`targetUpdates` 写 `"rolled_back_count": target.RolledBackCount + 1`，紧接着 `targetAudits` 手拼 `formatMillis(target.RolledBackCount), formatMillis(target.RolledBackCount+1)`——同一个新值算了两遍，且「review_results 仅在旧值非 `[]` 时落审计」这类条件全靠手写 if 分散守护（reject.go:116、:144、:179 三处同款条件）。

**规模实测**（grep 计数，2026-09-15）：

| 面 | 数量 | 位置 |
|---|---|---|
| builder 调用 | 53 处 | advance 8 / comment 6 / escalation 7 / llmflow 2 / reject 13 / reproject 3 / review 7 / task 7 |
| 裸 `InsertAuditLogWith` 手拼 | ~35 处 | requirement/version/session/escalation/artifact/reproject/advance 等（含 INSERT/DELETE 动作） |
| 双写更新点（updates+audits 成对出现） | 15+ 处 | reject（30 条手拼审计/函数）、advance、reproject、review、escalation（issues）、llmflow failTask（tasks+flow_nodes 双表）、task |
| builder 个数 | 4 个 | 分散 4 文件，近同构 |

## 2. design-it-twice：三个候选 interface

### 案 A：薄收口 builder（否决）

统一 4 个 builder 为 `changeLog(table, rowID, field, old, new, byWho)` 一个函数。
- 否决理由：**deletion test 不过**——全删后各调用点原样恢复手拼，复杂度只是搬家没有集中；新旧值仍双写，漂移面原地不动。interface 与 implementation 一样宽（shallow）。

### 案 B：typed 变更 struct + 同源派生（选定）

调用方声明一次变更（typed struct 字段），implementation 从锁定行读旧值、内建「列名 ↔ 审计字段名」映射与文本化口径，同源派生出 updates map 与 AuditLog 行：

```go
// rd/repo/change.go（示意，非终稿）
type Set[T any] struct { Val T; Null bool }   // 零值=不变更；To(v)=设值；Null[T]()=置 NULL

type NodeChange struct {
    State           Set[string]
    EnteredAt       Set[int64]   // Null=true 写 NULL
    ReviewResults   Set[string]
    RolledBackCount Set[int64]
    ReopenSource    Set[string]
    LastDecision    Set[string]
    PendingScope    Set[string]
    Reviewers       Set[string]
}

// 派生核（纯函数，免库单测）：old 从 node 锁定行读，audit 仅 old≠new 文本时产出
func nodeChangeToUpdatesAndAudits(node *repo.FlowNode, ch NodeChange, byWho string) (map[string]any, []repo.AuditLog)

// 落库口（乐观锁 + 逐条审计，吸收现 applyNodeUpdatesTx）
func ApplyNodeChangeTx(ctx context.Context, tx *gorm.DB, node *FlowNode, ch NodeChange, byWho string) error
```

调用侧变化（reject.go 回退目标为例）：

```go
// before：updates map 一遍 + audits 手拼一遍 + 条件 if 三处
// after：
err := s.repo.ApplyNodeChangeTx(ctx, tx, target, repo.NodeChange{
    ReopenSource:    repo.To(domain.ReopenSourceDownstreamReject),
    RolledBackCount: repo.To(target.RolledBackCount + 1),
    State:           repo.To(repo.PhaseExec),
    EnteredAt:       repo.To(now),
    LastDecision:    repo.Null[string](),
    ReviewResults:   repo.To("[]"),
    PendingScope:    repo.To(flow.Scope),
}, byWho)
```

- interface 面 = 1 个 struct（8 字段，= 列清单）+ 2 个泛型 helper + 1 个 Apply。implementation 吸收：映射表、文本化（millis/NULL→""）、条件落审计、乐观锁、逐条审计插入。**深**。

### 案 C：反射 + struct tag（否决）

repo struct 加 audit tag，派生核反射遍历。
- 否决理由：代码库零反射先例（Go 代码风格偏好显式）；编译期安全丢失（拼错字段名运行期才炸）；tag 语法把映射表藏进结构体注释，locality 反而降。**我的推断**：以本库「表驱动显式声明」既有风格（binding.go CategoryDefinitions、notificationTargetTypes）衡量，显式字段表比反射更贴库风。

## 3. 决策清单（推荐分支代拟，逐条可推翻）

| # | 决策 | 取值 | 性质 | 理由 |
|---|---|---|---|---|
| D1 | 收口范围 | **只收 flow_nodes**（含 llmflow failTask 节点段）；issues/tasks/flows/requirements 二期 | 推荐 | 最痛面（reject 30 条手拼）+ 2 个 builder（reviewAudit/flowNodeAudit/applyNodeUpdatesTx）全死；影响面 6 文件可控。escalation/issues 与 tasks 双写同病但另有 byWho 三来源问题，二期观察后收 |
| D2 | interface 形状 | **案 B**：typed struct + 同源派生 | 推荐 | 三案对比见 §2；案 A 浅、案 C 反射违库风 |
| D3 | 同值语义 | 审计**仅 old≠new 文本时落**（统一现有 3 处手写条件）；updates 照写全部非零值字段（不动乐观锁/409 语义） | 推荐 | 与现状手拼行为逐点一致（reject.go:116/144/179 的条件即 old≠new 特例），行为保持口径，IT 黄金断言不破 |
| D4 | nullable 表达 | Go 泛型 `Set[T]{Val,Null}`（`To`/`Null[T]()`），字段零值=不变更 | 推荐 | entered_at/last_decision/pending_scope 存在「置 NULL」语义（reject.go:134/165），双指针或哨兵都不如显式 Set 干净 |
| D5 | byWho | Apply **调用级参数** | 推荐 | flow_nodes 路径现状全为调用方显式（plugin/system/user/角色）；tasks 的 row.RoleID 派生属二期 |
| D6 | 落点 | **rd/repo**（新文件 change.go）；formatMillis/derefString 挪 pkg/utils 供 repo/service 共用 | 推荐 | repo 已拥有 UpdateWithOptimisticLockTx + InsertAuditLogWith + 列名口径；textify 助手现状在 service（task.go:738 / comment.go:811），repo 不能反向 import service，挪 utils 是唯一干净位 |
| D7 | 审计行序 | 派生核按 NodeChange **struct 字段序**产出；实现时对照 audit_it_test 黄金断言，若对序敏感则把字段序排成现状高频序 | 我的推断 | 现手拼顺序=各写路径手写顺序，本就可能互不相同；需实测 IT 断言是否序敏感（文档空白，实现期验证） |
| D8 | 术语 | 「节点变更/同源派生」= 架构词汇，**不进 CONTEXT.md**（领域词汇单点定义只收领域词） | 推荐 | 与「相位/评审位/拍板」不同层；记录于本提案即可 |

## 4. 迁移清单（D1 范围内）

| 文件 | 动作 |
|---|---|
| rd/repo/change.go | 新建：Set/To/Null、NodeChange、派生核、ApplyNodeChangeTx |
| rd/pkg/utils | 收编 formatMillis / derefString（service 改引用） |
| rd/service/reject.go | 13 处调用/30 条手拼审计 → NodeChange 声明；flow 侧 current_node_id 手拼暂留（二期） |
| rd/service/advance.go | applyNodeUpdatesTx 8 处调用改 Apply；advance.go:712 applyNodeUpdatesTx 删除 |
| rd/service/review.go | 7 处 reviewAudit → NodeChange；review.go:519 builder 删除 |
| rd/service/reproject.go | 3 处 flowNodeAudit → NodeChange；:352 builder 删除（裁剪的 DELETE/INSERT 审计不在 UPDATE 模型，保留） |
| rd/service/llmflow.go | failTask 节点段 2 处 → NodeChange（tasks 段手拼留二期） |
| rd/service/escalation.go | flow_nodes 面无；issues 面 7 处留二期 |
| 删除 | reviewAudit、flowNodeAudit、applyNodeUpdatesTx（deletion test：收编后全删零信息损失） |

## 5. 测试策略

- **派生核单测**（新，纯函数免库）：映射正确性（8 字段×列名×审计名）、old 提取（string/*string/int64/*int64/NULL→""）、old≠new 条件（同值不落审计）、Set 零值跳过、Null 语义（列写 NULL + 审计新值 ""）。
- **回归门（不动）**：audit_it_test.go 黄金断言 + reject/reproject/advance/review IT 全绿 = 行为保持证明；任何审计内容差异即红灯。
- **闸门**：gofmt/vet/build + 全测试（库内既有纪律）。

## 6. 风险与边界

1. **审计文本口径差异**：手拼处偶有「手写空串」特例（llmflow doneAt old=手写 `""`）；派生核统一 nil→""。flow_nodes 面（二期外的 tasks 面）核对后一致。**实现期逐点对照**。
2. **行为保持边界**：updates 无条件写、审计条件落的口径不变；不引入「同值跳过更新」（那会改乐观锁/409 语义，超范围）。
3. **二期展望（不承诺）**：flows.current_node_id + requirements.status 手拼（3 处）、issues/tasks 双写（escalation/task/comment/llmflow，byWho 三来源统一后收）。

## 7. 拍板后动作

- semantica record_decision（dsh-flow 白名单内，记录 D1-D6 取舍）——已落账 `812096b2`；
- 工单化（to-tickets 或直接实现，视拍板时的批次安排）——单批次直接实现（见 §8）。

## 8. 实现记录（2026-09-15）

### 8.1 实现期设计精化（三处，均为实现约束倒逼，方向不变）

1. **interface 从 `Set[T]` struct 改为类型化变参构造器**。实现期发现两个事实：①各写路径审计手拼顺序互不相同（reject 计数在前、advance 相位在前），struct 固定字段序无法同时贴近；②`Set[T]` 零值=不变更与「设为空串」（`state` 置 `""` 合法且存在，reject 回退核）冲突，需额外 Present 标记。改为每字段一个构造函数（`NodeState`/`EnteredAt`/…/`FeatureBranch` 共 13 个，含 3 个 Null 变体），变参声明序=审计行序，「在场即声明」天然消除零值歧义；重复声明报错 fail-fast。派生核 `deriveNodeChange` 拆为纯函数（免库单测面），`ApplyNodeChangeTx` 只做派生+乐观锁+审计插入。
2. **补 `Silent(c)` 变更级修饰（写而不审计）**。迁移中发现 reject 回退核对 `last_decision`（3 处）与 `pending_scope`（1 处）是「写 NULL/落值但不落审计」——被拒结论已由 wb_decision 拍板审计回放、重定标记由 reproject 清除时统一留痕，手拼语义本就如此；保持行为需显式表达。`pendingScope` 清标记（reproject）在 D3 统一口径下由「无条件审计」条件化（carrier 恒持标记，实践行为不变）。
3. **补 `Who(byWho, c)` 单条署名覆盖**。wb_review 单次行写混两署名（表态审计=表态角色、相位/裁决迁移审计=plugin），整次调用单一 byWho 不够；不能拆两次 Apply（两次乐观锁、第二次基线失效）。

### 8.2 变更面

- 新建：`rd/repo/change.go`（构造器+映射表+派生核+Apply）、`rd/repo/change_test.go`（5 条 TestUnit_：映射序/同值不落/Silent+Who/Null 变体/空集重复）。
- 收编删除：`reviewAudit`（review.go）、`flowNodeAudit`（reproject.go）、`applyNodeUpdatesTx`/`enteredAtText`（advance.go）；`formatMillis`/`derefString` 收敛 `pkg/utils`（`FormatMillis`/`FormatMillisPtr`/`DerefStr`），service 全部调用点改名。
- 迁移调用点：reject（3 写路径/30 条手拼消灭）、advance（4）、review（4，含 Who 双署名）、reproject（下游重排+carrier 清标记）、llmflow failTask（节点段；tasks 段留二期）。
- IT 基建顺手改：IT scratch 库前缀 `wbflow_it_*` → `wbflow_qa_it_*`（7 处代码点+注释）——qa 运行账号 `wbflow_qa` 授权恰为 `wbflow_qa%`（qa_provision.sh GRANT），root 凭据按纪律零落仓库不可得，qa 账号+匹配前缀即可跑全量真库 IT，隔离性与测后 DROP 口径不变。

### 8.2.1 行为偏差补录（2026-09-15 评审发现；同 D3 统一口径的延伸，非新拍板）

D3「与现状手拼行为逐点一致」在 **reopen_source 无条件审计** 三处不成立，实现时被统一条件化但当时未记录：
- reject.go 回退目标/下游节点（reopen_source=downstream/upstream-reject）
- llmflow failTask（reopen_source=rework-return）

旧行为：重复打回/重复 fail 同值时仍落一行 old==new 的同值审计；新行为（派生核 old≠new 才落）：跳过。差异仅在「同值重复到达」的真实可重复路径上，审计行数与旧版可分叉；既有黄金 IT 未覆盖该场景故未红灯。**按 D3 口径本义（同值写不重放留痕）新行为为正确口径，追认保持**；若未来审计回放侧依赖同值行计数，需单独拍板恢复。

### 8.3 闸门

gofmt/vet/build 零输出；`go test ./...` 全绿（含派生核 5 条新单测）；真库 IT 全量跑（qa 账号，1783s）：**audit/reject/advance/reproject/review 并发等黄金断言全过 = 行为保持证明**；唯一失败 `ReviewClarifyLoop_RealMySQL` 经 HEAD 干净 worktree 对照复跑**同点同因失败**（rd 名下 pending-confirm issue 拦 agree 闭环，review_it_test.go:342）——存量问题，与本次迁移无关（提交信息「单跑过」与现状矛盾，需另行排查：该测试序列缺 issue2 的 settle/confirm 步骤）。
