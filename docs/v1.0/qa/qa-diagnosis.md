# wbflow v1.0 · 缺陷诊断报告（qa-diagnosis）

> 2026-09-10 诊断轮（接力 qa 全量验收）。对象：[qa-defects.md](qa-defects.md) BUG-003~008 + OBS-001（BUG-001/002 并行 session 已登记，不在本轮）。
> 方法：mp-diagnosing-bugs 诊断循环——每缺陷先建可复现反馈环（[qa_diag_min.py](../../../qa/qa_diag_min.py)，scratch 库 + rd :18081 实测），再代码级根因定位（文件×行号），修复方案给备选×影响面。
> 约束遵守：本轮零产品代码改动（rd/ 一行未动）；实测全部在 scratch 库 `wbflow_qa_*`。
> 实测环境：source /tmp/wbflow-qa-env.env && python3 qa/qa_diag_min.py（复现环，~8s，每缺陷独立节、软失败继续）。

## 0. 诊断结论速览

| 缺陷 | 根因一句话 | 修复层级 | 修复方向（推荐） | 置信度 |
|---|---|---|---|---|
| BUG-003 | exec 推进判据与 complete 内聚触发只认 produce/dev/accept 三族，create_reqs 完成不被承认；rule4 空转又被 capability 挂载阻断 | 服务 | advance.go:433 + task.go:374 触发族纳入 create_reqs（2 行） | 0.98 |
| BUG-004 | Update 入口对 title/priority/status/category 无条件必填校验，diff 层本支持字段级变更但被前置校验挡死；proto3 string 无 presence，契约未声明全量语义 | 契约（+服务可选） | 契约显式声明 PUT=全量语义（零代码）；部分更新留后续（proto optional 改造） | 0.97 |
| BUG-005 | wb_assign_task/wb_task_update 全链只查定义侧激活清单，从不读 flow_nodes.state 实例相位 | 服务 | assign/complete 增「节点已激活（state 非空）」校验；entry 相位放行 | 0.95 |
| BUG-006 | my-task 投影回显能力动作裸 id（action.ActionID），assign 解析只认 `<cap>.<action>` 全名（无点走 node action 路径 400） | 契约（服务单点） | my-task kind=capability 条目回显全名（mytask.go:81 单点） | 0.95 |
| BUG-007 | wait 分支前置条件（fe 节点激活时 rd 未 approved）与唯一激活链矛盾——fe 侧 exec-CR 节点只能在 rd 侧 last_decision=approved 瞬间激活 | 设计裁决 | 文档化死分支（同注记 -E/-F/-G「裁决正本」处理），不动代码 | 0.90 |
| BUG-008 | writeEscalationNoticeTx 通知行 anchor_kind 硬编码 document，手动 wb_escalate 只有这一条通知行，收件箱按 anchor_kind 投影 escalation 断链 | 服务 | 该行落 anchor_kind=escalation + anchor_artifact_id=源 issueId（单点） | 0.90 |
| OBS-001 | wb_decision 校验集=user_gate=1+评审全 agree，无 exec 判据校验（评审位空的 gate 节点 collect 恒 approved） | 设计裁决 | 两可论据见 §7，供 User 拍板 | 现象 0.99 / 设计待拍板 |

---

## 1. BUG-003（S1/P0）plan 流程 pln-d7 死锁

### 1.1 根因链（代码级，台账假设已验证成立，补全第 4 环）

1. **seed**：`rd/migration/sql/0002_seed.sql:196` pln-d7（produce 型）`node_actions='[{"kind":"capability","capabilityId":"decide-cap","actions":["create_reqs"]}]'`、`expected_outputs='[]'`；`0002_seed.sql:134-137` decide-cap 动作 `create_reqs` 的 `outputProductTypes=[]`。全 seed 仅 pln-d7 挂载 create_reqs。
2. **任务可完成**：`rd/domain/task.go:324-325` `CompleteCriteria` 对 `create_reqs` 恒放行（v1 最小判据，注释明言「由 T8/plan 流程接入后收紧」）——`wb_task_update complete` 200，tasks.status=completed。
3. **exec 推进判据不承认**：`rd/service/advance.go:425-464` `execAdvanceReady` 只统计三族（`:433` `case repo.TaskTypeProduce, repo.TaskTypeDev, repo.TaskTypeAccept`）——create_reqs 完成 → `completed=0, inflight=0`，`completed>0` 判据（`:442`）不成立。
4. **rule4 空转兜底被阻断**：`advance.go:445-456` 空转条件 `!mounted && expectedEmpty`；`execActionsMounted`（`:567-578`）见到 kind=capability 条目即 mounted=true → 空转不放行 → 返回 missing「exec_task:节点无已完成 exec 任务」→ 409 GATE_BLOCKED。
5. **complete 内聚触发同样不认**：`rd/service/task.go:373-378` complete 后同事务内聚推进的触发族 switch 同样只含三族——完成瞬间不会自动推进（wb_advance 手动兜底是唯一通道，而它被第 3/4 环挡死）。死锁闭合。

### 1.2 验证证据（qa_diag_min.py BUG-003 节，scratch 库 req 实测）

```
[complete create_reqs] [200, "completed", ""]
[pln-d7 node row] "n-224 | exec"
[task row] "t-1788980754385-043 | decide-cap.create_reqs | create_reqs | completed"
[advance pln-d7] [409, {"error": "...节点 \"n-224\" exec 相位 complete 判据未通过，拒绝推进",
  "code": "GATE_BLOCKED", "missing": ["exec_task:节点无已完成 exec 任务（exec 判据未通过）"]}]
[current] "n-224"   ← 恒驻 pln-d7，pln-d8 永不可达
```
与 qa_drive.py B-5g、台账登记的 req-011 报文逐字一致（稳定复现，非环境噪声）。

### 1.3 修复方案（备选×影响面）

**方案一（推荐）：触发族纳入 create_reqs（2 行改动）**
- `advance.go:433` switch 增 `repo.TaskTypeCreateReqs`；`task.go:374` 同增。
- 影响面：①pln-d7 create_reqs complete → 内聚推进同事务直达 pln-d8（B-5g 转 PASS），wb_advance 手动调用变幂等 noop；②completeMissing（`task.go:461`）与 CompleteCriteria（`domain/task.go:324`）**不必同改**——create_reqs 恒放行判据是 spec 内注释过的 v1 口径，动它反而扩面；③全 seed 仅 pln-d7 挂 create_reqs → 行为变化半径 = plan 尾段，其余 3 流程零影响；④在途守卫（`inflight==0`）与节点相位守卫（`task.go:401` PhaseExec）天然继承，无新风险面。
- 回归判据：qa_drive.py 187 PASS（B-5g 转绿）。

**方案二（不推荐）：rule4 空转放宽为 `expectedEmpty` 单条件**
- `advance.go:453` 去掉 `!mounted` 前置。
- 影响面：全局语义变更——所有「挂载了 capability 动作但 expected_outputs 为空」的 exec 节点在无任务完成时也放行，pln-d7 无需派发任务即可推进，**架空 spec §2「谎报完成」防线**，且与 collect 空转语义（advance.go:9-10 注明同源）口径分叉。波及面不可控，弃。

**修复层级**：服务（rd/service 两文件各 1 行）。

### 1.4 置信度：0.98（根因链逐环代码+seed 双实证，报文与验收轮逐字一致）

---

## 2. BUG-004（S3/P2）wb_update_requirement 强制全量字段

### 2.1 根因链（代码级）

1. `rd/service/requirement.go:586-601` Update 入口四连必填校验**无条件执行**：`:590` title、`:593` ValidPriority（空串非法）、`:596` ValidRequirementStatus（空串非法）、`:599` category 必填——缺任一字段直接 400，局部更新语义不存在。
2. 讽刺点：diff 层本就按字段算变更——`requirement.go:843-874` `diffRequirement` 逐字段比对库行，只落 changed 列 + 逐字段审计（`:674-686`），部分更新的机械已就绪，仅被前置校验挡死。
3. 契约层：`rd/api/api.pb.go:2610-2626` `UpdateRequirementReq` 全部 plain proto3 string（无 `optional`），线上无法区分「未传」与「传空」——部分更新语义**在当前契约下不可表达**。
4. 陷阱面：`owner`（空=未指派）与 `version_id`（空串=移回待分配池，`requirement.go:844-846/861`）以空串为合法值——「空=未变更」约定会封死清空/解绑操作。

### 2.2 验证证据（qa_diag_min.py BUG-004 节实测）

```
PUT /api/requirements/:id  body 仅 {requirementId,title,priority:"P1",updatedAt}
→ 400 {"error":"wbflow: 入参非法: 非法状态 \"\"（允许：待启动/进行中/已完成/已取消）"}
同一请求补齐全量字段回显 → 200 ok
```

### 2.3 修复方案（备选×影响面）

**方案一（推荐，v1.0）：契约显式声明 PUT=全量语义**
- proto 注释 + arch_design/api 文档注明「wb_update_requirement 为全量更新，调用方须回显当前快照（GET 同源）」；零代码改动。
- 影响面：调用方（FE/qa_drive 已按全量适配）无感；diff 变更历史语义不受影响（diff 对库行计算，全量回显不产生伪变更）；与现状行为完全一致，只是把隐式契约显式化。

**方案二（后续增强）：部分更新语义**
- proto 字段改 `optional`（re-gen api.pb.go）+ Update 按 presence 校验/diff；注意 owner/version_id 的「显式清空」需 presence 区分（见 2.1 第 4 点，空=未变更约定不可行）。
- 影响面：契约破坏性变更（所有调用方适配）+ service Update 重写 + 测试资产回改；v1.0 收益（调用方便利）小于成本，建议挂 TODO 随 plugin 期契约整理一并做。

**修复层级**：契约（方案一）/ 契约+服务（方案二）。

### 2.4 置信度：0.97

---

## 3. BUG-005（S3/P2）任务总线不校验节点相位

### 3.1 根因链（代码级）

1. `rd/service/task.go:52-129` Assign：取节点行（`:86-92`）后直接解析动作，`ResolveTaskAction`（`rd/domain/task.go:107-112`）的「激活」校验（`:154-172`）查的是**定义侧 node_actions/reviewers 声明**，与实例节点 `flow_nodes.state` 相位零关联——注释「未激活 400」的「激活」实为定义声明，非相位。
2. `task.go:253-507` Update（ack/progress/complete）：`completeMissing`（`:451-507`）只装配产物/表态/issue 事实，同样不读节点 state——未激活节点上任务可走完全生命周期。
3. 孤儿完成的放大器：`advance.go:426-444` `execAdvanceReady` 按 `requirement_id+node_id` 统计该节点**全部历史任务**（无时间/相位过滤）——节点日后正式激活，exec 判据直接把孤儿 completed 计入（completed>0 即放行），真实执行可被跳过。
4. 实例相位唯一写点在推进核（advance.go:370 激活、reject.go 回退清零），任务总线无一处消费。

### 3.2 验证证据（qa_diag_min.py BUG-005 节实测）

```
bus-02 state before = ""（未激活，bus-01 未拍板）
assign product-design.produce @bus-02 → 200 ok（product-design 归属角色 pm）
ack → progress → add_artifact(product_doc) → complete → 200 "completed"
bus-02 state after = ""（相位未迁移，任务成孤儿 completed）
```
台账侧证（tec-b8b）：激活前 complete 后 review 表态报 `节点 "n-92" 相位 "" 不接受评审表态`——同一缺口在 review 端点有校验（`review.go` 相位族校验），任务总线没有，口径不一。

### 3.3 修复方案（备选×影响面）

**方案一（推荐）：assign + complete 双点「节点已激活」校验**
- Assign：`node.State == ""` → 400「节点未激活，不可派发」；Update complete：同校验（防回退清相位后**在途**任务完成成孤儿）。
- 影响面：①entry 相位**放行**——bus-01 在 entry 相位派发 to-brd 是剧本合法用法（qa_drive.py:460 B-1、e2e 同构），校验只能是「state 非空」而非 `action.On==state`，这点与台账修复方向表述需对齐；②现有 e2e/qa 剧本全部先激活后派发，零回归预期；③回退场景（reject.go 清 state=""）在途任务 complete 将被拦——语义正确（回退重入须重新提评），但需在契约 missing/blocked 文案中说明。
- 注意：review 端点已有相位校验，本修复使任务总线对齐同源口径。

**方案二（最小）：仅 Assign 单点校验**
- 挡住新建孤儿；已存在孤儿任务（回退残留）complete 仍可落——防线有洞，但改动面最小。
- 推荐：v1.0 直接做方案一（两处同构 3 行），不留洞。

**修复层级**：服务（rd/service/task.go）。

### 3.4 置信度：0.95

---

## 4. BUG-006（S3/P2）my-task 与 assign 动作标识口径不一

### 4.1 根因链（代码级）

1. `rd/domain/mytask.go:58-108` `ActiveActionsFor`：kind=capability 条目回显 `ActiveAction.ActionID = action.ActionID`（`:81`，能力动作词表内的**裸名**，如 `create_reqs`/`produce`）。
2. `rd/service/task.go:103` Assign 消费 `ResolveTaskAction` → `SplitCapabilityAction`（`rd/domain/task.go:121-127`）按**最后一个点**切分；无点输入走 node action 路径（`:111 → :228-261`），在 kind=node 条目中找裸名 → 命不中 → 400「node action 未在该节点声明」。裸名 `produce` 恰好也在 `actionTaskTypes`（`:28-48`）有映射，但解析路径根本走不到归族——双口径并存且无文档。
3. node action 类（如 `align.opening`）本身带点、天然同口径——断链只在 capability 动作。

### 4.2 验证证据（qa_diag_min.py BUG-003 节顺带实测，同节点）

```
my-task leader@pln-d7 → activeActions[].actionId = ["create_reqs"]        （裸名）
assign actionId="create_reqs"            → 400 "node action \"create_reqs\" 未在该节点声明"
assign actionId="decide-cap.create_reqs" → 200 ok
```

### 4.3 修复方案（备选×影响面）

**方案一（推荐）：my-task 回显全名（单点改动）**
- `mytask.go:81` kind=capability 分支改 `ActionID: entry.CapabilityID + "." + action.ActionID`；node action 条目不动。
- 影响面：①调用方（FE/驱动脚本）可直接回填 assign——qa_drive.py B-5d 两种写法都接受（`"create_reqs" in acts or "decide-cap.create_reqs" in acts`），e2e 唯一 actionId 断言是 `align.opening`（`rd/e2e/task_test.go:39`，node action，不受影响）；②proto 契约注释同步「actionId=派发可直接回填的定位标识」。

**方案二：assign 兼容裸名**
- ResolveTaskAction 增 fallback：无点输入在节点激活清单内唯一命中 capability 动作时按该解析。
- 影响面：歧义风险——同节点挂多个能力含同名动作（如 produce）时需唯一性校验否则报错口径更绕；且「双口径并存」仍需文档，实质是把复杂度搬进解析层。不推荐。

**修复层级**：契约（服务单点实现，proto 注释同步）。

### 4.4 置信度：0.95

---

## 5. BUG-007（S3/P3）并行 join wait 分支 API 不可达（死代码）

### 5.1 根因链（代码级穷举）

wait 分支 = `rd/service/advance.go:339-345`（case：本侧 fe 侧 exec-CR 节点，prev 为 rd 侧 exec-CR，`!peerApproved(prev)` → wait）。不可达论证——**fe 节点存在 exec 相位的全部路径，rd 侧 last_decision 必为 approved**：

1. **唯一激活点**：flow_nodes.state 从 "" 到激活相位的写点全库仅 `advance.go:364-370`（`activateAndMoveLockedTx`），其唯一调用方是 `advanceAfterGateLockedTx` 的 `activate` 闭包（`:328-330`）。fe 侧节点的激活只发生在 case1（`:333-338`）：rd 侧 gate 落定推进时对侧未 approved → `activate(next)`。而 rd 侧 gate 落定 = last_decision 已写 approved（`review.go:341` wb_decision 路径 / `advance.go:203-207` 机械收口路径，均先落 approved 再调推进核）——**fe 激活瞬间 rd 必 approved**。
2. **打回路径不破坏不变量**：`reject.go:72-205` + `domain/reentry.go:81-92`——回退目标=被拒评审节点之前最大 order 节点。①bus-10 QA 拒 → target=bus-09（fe，`:132` exec 重入 + last_decision 清空），下游联动重置跳过 order≤target（`:154`）→ bus-08 保持 approved → fe 重完成时 `peerApproved(prev)` 仍 true；②bus-09 自身 CR 被拒 → target=bus-08、bus-09 清相位，bus-08 重完成后重激活 bus-09 仍走 case1（rd approved 瞬间）。
3. **scanner 同核**：`SweepParallelJoins`（`advance.go:474-512`）复用 `advanceNodeLocked`——兜底巡检同样不可能产出 wait 语义的实际推进（其报告只计 cross）。

结论：`AdvanceOutcomeWait` 是注记-C①「本侧完成待对侧」对称表述的防御性实现，在「rd 先行激活 fe」的激活链设计下恒不可达——设计级死代码，非缺陷行为偏差。

### 5.2 验证证据

代码穷举（上述 3 点，全库 state 写点 grep 清零：advance.go:239/370、review.go:159/438、reject.go:108/132/164，后四者均为已激活节点相位翻转/回退，不产生「fe 激活而 rd 未 approved」态）；qa_drive.py C-8b SKIP 附因（`:763-764`）。

### 5.3 修复方案（备选×影响面）

**方案一（推荐）：文档化死分支**
- spec 注记追加裁决条目（同注记 -E/-F/-G「裁决正本」处理格式）：wait 分支为防御性保留/已知死分支；或删分支改注释。零行为风险。

**方案二：放宽激活链（bus-08/09 并行双激活）**
- 使「fe 激活而 rd 未 approved」成为合法态，wait 语义可达。
- 影响面：join 判定、flows 锁序、scanner 巡检、回退联动全部重议——注记-C 三层设计推翻重来，v1.0 不建议。

**修复层级**：设计裁决（文档）。

### 5.4 置信度：0.90（代码路径穷举闭环；未覆盖「直接改库」类带外态）

---

## 6. BUG-008（S3/P2）wb_escalate 手动端点通知 anchor_kind 投影断链

### 6.1 根因链（代码级）

1. `rd/service/escalation.go:441-471` `writeEscalationNoticeTx`：通知行 `AnchorKind` **硬编码** `domain.IssueAnchorDocument`（`:450`），且不写 `anchor_artifact_id`（跳转锚点缺失）。
2. 三条上升路径的通知形态不一：
   - **手动 wb_escalate**（`escalation.go:172`）：只有 writeEscalationNoticeTx 一行 → anchor_kind=**document**（收件=user）；
   - **自动阈值**（`comment.go:460-468`）：notice 行（document）**加上** `escalateSet=true`（`:463`）触发的 T11 行（`:533-539` `escalationSpec` → `EnqueueNotificationTx` → `Notify`，`notifier.go:177` `AnchorKind: spec.Kind`=**escalation**，收件=评审角色）→ 投影正确是靠第二行；
   - **wb_comment escalate=true**（`comment.go:222-228`）：仅 T11 escalation 行。
3. 投影断链点：`rd/service/inbox.go:99-113` `inboxItem` `Kind = row.AnchorKind`（`:102`）、`TargetType = notificationTargetTypes[row.AnchorKind]`（`:105`）；`notifier.go:60-72` 七类映射表**无 document** → 手动行 Kind=document、TargetType=""，注记-D5「冲突待裁决←escalation」投影（kind 筛选枚举也只认七类）永远看不到手动仲裁请求。

### 6.2 验证证据（qa_diag_min.py BUG-008 节实测，scratch 库 issues 表）

```
手动 wb_escalate（comment escalate=false 后 /comments/:id/escalate）:
  document  | notification | user | qa | [冲突上升待仲裁] issue cm-…8175 待 User 裁决（operator=qa 手动上升），裁决 session 已就绪
wb_comment escalate=true（同需求对照）:
  escalation | notification | user | qa | 冲突上升裁决：评论 cm-…8252（节点 n-…）待裁决
```
与台账登记的验收轮证据（手动 ×2 document / comment 路径 escalation / 自动阈值 escalation）同构。

### 6.3 修复方案（备选×影响面）

**方案一（推荐）：notice 行对齐 T11 投影契约（单点）**
- `escalation.go:450` 改 `AnchorKind: "escalation"`（NotificationKindEscalation），并补 `AnchorArtifactID: utils.StrPtr(source.IssueID)`（对齐 escalationSpec 的 TargetID 口径，FE 跳转可达）。
- 影响面：①该行 issue_type=notification、非评论锚点，anchor_kind 语义在通知行上=通知类别（T11 路径即此口径），不破坏 issues 锚点校验域（`domain/issues.go` 校验只覆盖评论建行入参）；②anchor_kind 消费面经 grep 仅 inbox 投影一处，波及面收敛；③手动/自动/评论三路径在「用户收件箱呈 escalation」维度对齐。

**方案二：手动路径补 escalationSpec enqueue（与自动阈值路径同形）**
- 让手动路径也发 T11 escalation 行（收件=评审角色||user）。
- 影响面：reviewer 可见性与自动路径对齐（这是现状另一处真实缺口：手动上升 reviewer 无通知）；但**单独使用不修 D5**——user 收件的 notice 行仍是 document。可与方案一叠加（三路径双行完全同形）。

**修复层级**：服务（rd/service/escalation.go；方案二叠加 comment.go 复用）。

### 6.4 置信度：0.90（断链机制代码+实测双证；「escalation 作为通知行 anchor_kind 合法值」需 FE/契约侧最终确认）

---

## 7. OBS-001（待拍板）gate 拍板不校验 exec 判据

### 7.1 现象与代码定位

`rd/service/review.go:267-389` Decide 校验集 = `:306` user_gate=1 + `:310-325` 重复拍板幂等 + `:327-334` 评审全 agree（CollectVerdict）；**无 exec 任务/产物判据校验**。而评审位为空的 gate 节点 `CollectVerdict` 恒 approved（`rd/domain/collect.go:193-201` `len(reviewers)==0 → CollectApproved`）。

实测（qa_diag_min.py OBS-001 节）：新建 business 需求（bus-01 有期望产物 brd + to-brd 挂载），**0 任务 0 产物**直接 `wb_decision approved` → 200，current 迁移 bus-02。与验收轮观察（台账 OBS-001）一致。

### 7.2 两可论据（供 User 拍板，不预定结论）

**甲方（现状可辩——拍板=用户意志，判据归任务层）**：
- 防线分层论：exec 完成判据已在 `wb_task_update complete` 层拦截谎报（`domain/task.go:299-328` CompleteCriteria 表驱动）；拍板表达的是 User 意志而非系统判据复核——User 有权「产物不要了，跳过」；
- 结构性障碍：纯 gate 节点（bus-00 待启动、pln-d0、pln-d8 完成）**天然无 exec 任务/产物**——若 wb_decision 前置 exec 判据，这些节点永远无法拍板，流程启停全卡死；做判据就必须按节点形态豁免，规则立刻复杂化；
- 拍板卡交互契约（ui-interaction-final §1.3）只绑「评审全 agree（置灰）」，未绑产物——现实现与交互定稿一致。

**乙方（应校验——谎报/误操作防线缺口）**：
- bus-01 是「对齐+产物+拍板」复合节点（expected brd + to-brd 挂载 exec），User 期望「看过 brd 再拍板」；判据未齐仍可 approved = 流程完整性缺口，迁移后 bus-02 在无 brd 输入下推进，依赖断链；
- FE 拍板卡不感知产物缺失（§1.3 口径），误操作无任何软提示——OBS 即验收轮真实触发的误用面；
- 与 BUG-003 对照呈不对称：exec 侧判据很严（谎报完成拦截），gate 侧判据为零——同一条「谎报完成」防线（spec §2）在 gate 处缺位。

**折中形态（两案之间的第三选项，列出备参）**：仅对「挂载 exec 动作且 expected_outputs 非空」的 user_gate 节点前置判据（复用 execAdvanceReady 口径），纯 gate 节点豁免；或维持现状、FE 拍板出参/卡片补「产物缺失」软提示。

### 7.3 定级影响

若拍板「应校验」→ 升 S2~S3 缺陷，修复层级=服务（review.go Decide 前置判据，需节点形态豁免规则）；若拍板「现状正确」→ 登记为已知设计口径（契约/文档注明「gate 拍板不校验 exec 判据」），FE 提示为增强项。

---

## 8. 复测口径汇总

- BUG-003：`python3 qa_drive.py` B-5g 转 PASS（current=pln-d8），全量 187 PASS 无回归；`qa_diag_min.py` BUG-003 节 advance 应 200/noop。
- BUG-004：A-10 按选定方案补断言（方案一=文档化无代码断言；方案二=局部 PUT 200）。
- BUG-005：B-3 补正向断言（未激活节点 assign 400）+ bus-01 entry 派发 to-brd 回归放行。
- BUG-006：A-13c 断言 activeActions[].actionId=全名（或 assign 裸名 400 文档化）。
- BUG-008：C-6c 补手动路径断言（收件箱 kind=escalation）。
- OBS-001：按拍板结论定复测口径。
- 诊断资产：[qa_diag_min.py](../../../qa/qa_diag_min.py)（每缺陷独立可复现，scratch 库只读对账，不写生产联调库）。
