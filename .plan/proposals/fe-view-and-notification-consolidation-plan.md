# 提案：C4+C5 合并实现——FE 视图模型收敛 + 通知校验/owner 推导归位

- 状态：**已实现**（2026-09-15「继续」按推荐分支；semantica `d9fac49a`；闸门见 §闸门）
- 前置：C1-C3 已落地

## C4 · FE 视图模型收敛

| # | 决策 | 取值 | 理由 |
|---|---|---|---|
| C4-D1 | latest-wins 覆写规则（round 大者胜/同轮 at 后者胜）三写合一 | flowview.ts 收敛私有 `latestSeatMap`（round/at 比较唯一处）+ 导出 `latestSeatState(node, role, capabilityId)`；unagreeSeatsOf/seatStatesOf 改为消费同一 map；NodeDetail.tsx 删除本地 latestStateOf 改 import | 判定一处修处处生效（locality）；行为零变化（同比较式） |
| C4-D2 | TODO 五分类投影（spec -D5/N3 定稿的业务投影）归位 domain | 新建 `fe/src/domain/inboxview.ts`：TODO_GROUPS + `projectTodoGroups` 纯函数；InboxPage 只消费 | 业务投影规则不住页面层；补 inboxview.test.ts |
| C4-D3 | formatTime 跨页借用归位 | 新建 `fe/src/lib/time.ts`（moved 原实现）；RequirementPoolPage/InboxPage 改 import | 页面不再互为工具库 |

## C5 · 通知校验/owner 推导归位

| # | 决策 | 取值 | 理由 |
|---|---|---|---|
| C5-D1 | Notify 与 outboxRowOf 双写校验合一 | `normalizeNotificationSpec(spec) (NotificationSpec, error)`：越集/必填校验 + sender/senderRole 默认值填充一次完成，两处消费 | 校验单一真相源，加类别改一处 |
| C5-D2 | owner 推导纯判定下沉 domain | `domain.OwnerRoleWithFallback(execRole, requirementOwner)`（exec 能力归属角色优先，回落 requirements.owner）；deriveOwnerRoleTx 保留装配（4 次 repo 读），尾部判定改调 domain | 判定归 domain 纯函数层（免库可测）；CONTEXT.md 无此领域词，不进词汇表（D8 同 C2） |

范围外：CommentCard 操作可见性（C3 已裁）、comment.go inReview 上下文（C3 已导出）。

## 实现记录（2026-09-15）

- C4-D1：flowview.ts 收敛 `latestSeatMap`（比较式唯一处）+ 导出 `latestSeatState`；NodeDetail 删本地第三实现；顺带收紧 seatStatesOf 仅渲染声明评审位（原实现会把声明外结果条目也渲染，reproject 裁剪后不可能出现，防御性收紧，182→185 测试零回归证实无依赖）。
- C4-D2：`domain/inboxview.ts`（TODO_GROUPS + projectTodoGroups）+ inboxview.test.ts 3 条；InboxPage 只消费。
- C4-D3：`lib/time.ts` 承接 formatTime——**实现期发现消费方 8 处**（报告只计 2 处：CommentCard/ConflictCard/DecisionCard/NodeDetail/RequirementDetail/Versions 也在跨页/跨层借），全部改指 lib，页面/组件不再互为工具库。
- C5-D1：`normalizeNotificationSpec`（越集/必填校验 + sender/senderRole 默认值）单点，Notify 与 outboxRowOf 消费；notifier_test 既有断言全过=行为保持。
- C5-D2：`domain.OwnerRoleWithFallback`（issues.go，NodeExecRole 邻位）+ TestUnit 3 断言；deriveOwnerRoleTx 装配保留、尾部判定改调 domain。

## 闸门

go 11 包 ok；gofmt/vet 零输出；FE tsc 零错、vitest 185/185（新增 inboxview 3 条）、build 成功。
