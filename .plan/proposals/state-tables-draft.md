# 三张状态机 ISTQB 状态表——草拟稿 v2（按 User 2026-09-11 批注重修）

> **v1 错误声明**：v1 把当前实现（rd/domain/issues.go 迁移表 + spec 注记）当权威逐格照抄，导致 ①实现自带的 `overridden` 状态混进了设计表（与 9-07 issue 状态机重设计结论冲突）②实现里的非预期通路（设计未描述的迁移）被画成了合法格。**v2 权威 = 2026-09-07 issue 状态机重设计（模型 §1.5）+ 两层属性状态机（模型 §1.1/§1.2）+ 本次 override 裁决**；实现与设计的差异全部抽离到「偏差清单」，作为返工项登记，不再混入设计表。
>
> **表的读法（回答「事件是动作还是 node 阶段」）**：ISTQB 状态表**行=状态（阶段），列=事件（触发迁移的东西）**。台 1 的列=issue 上的动作；台 2 的列=编排级事件（既不是动作名也不是阶段名——阶段在行上）；`—`=设计未描述=不存在通路（实现也不该有）。pending-confirm 这类**是阶段（行），不是事件（列）**。
> **每格事件必须有触发者（2026-09-11 User 普适约束）**：除 user 外皆为 agent，agent 只在被调用时响应、不存在自主行动——落格前先答「**谁被调用之后触发它**」，答不出执行者即 `—`。

### 词汇单点定义（本表唯一词表；每次编辑后必跑尾部自检）

- **状态**：issue 五态 `open / pending-confirm / to-fix / fixed / resolved`（台 1）；节点相位 `"" / entry / exec / review / respond / done`（台 2）；评审位表态三态 `reviewing / agree / unagree`（台 3）。
- **事件（台 1 八个）**：`clarified / settle / qualify / reopen / declare-fixed / confirm / revise / override`。
- **退役词黑名单**（正文出现即错；唯一合法位置=「新旧名对照」「偏差清单」「API 名列」的映射语境）：`resolve(note)`、`resolve(fix)`、`resolve(agree)`、`clarifying`、`confirming`、`unagreed`、`agreed`（作存储态）、`overridden`（作状态）、`pass`/`block`（作表态值）。
- **自检命令**（编辑后必跑，命中行必须全部落在映射语境）：
  `grep -nE "resolve\((note|fix|agree\)|clarifying|confirming|unagreed|overridden" .plan/proposals/state-tables-draft.md`

---

## 台 1：issue 状态机（五态，设计权威版）

**状态（5 个）**：`open 待澄清 / pending-confirm 已澄清待复核 / to-fix 待修复 / fixed 已修复待复核 / resolved 终态`。

**`overridden` 作为独立状态废除**——按 2026-09-11 裁决：override 是**动作**，结果 = 置 `resolved`，不新增终态。（现实现 DDL 枚举/issues.go/-L1 注记里的 overridden 态=偏差，见偏差清单 D1。）

建行（wb_comment）→ `open`（建行非迁移）。`loop_count` 单计数器：每次回到前一状态 +1；澄清阶段读=reopen 次数、修复阶段读=revise 次数。

### 两阶段结构（骨架=9-07 定稿；**词汇已对齐 v5 与新事件名提案**，新旧名对照见下）

阶段一 · 澄清（评审轮 unagree → owner 澄清，在评审节点 respond 相位逐条处理）：
```
open（待澄清）
   │ ① clarified —— owner 澄清完成申报
   ▼
pending-confirm（已澄清，待复核）
   │ 评审者复核位，三向出口 = 三个事件：
   ├─ ② settle —— 协商一致/接受澄清（无需改产物）→ resolved
   ├─ ③ qualify —— 定性真问题（触发打回）→ to-fix
   └─ ④ reopen —— 未解释清楚（澄清计数 +1）→ open
```

阶段二 · 修复（打回 → 产出节点 re-entry，owner exec 重入相位）：
```
to-fix（待修复，按打回清单逐条）
   │ ⑤ declare-fixed —— owner 修复完成申报（此前 exec 重入修复=revise / diagnosing-bugs 动作）
   ▼
fixed（已修复，随重新提测上报）
   │ 评审者复核位，两向出口 = 两个事件：
   ├─ ⑥ confirm —— 修复成功（确认关闭）→ resolved
   └─ ⑦ revise —— 修复失败（revise 计数 +1）→ to-fix
```

> **出口标识纪律（本次修正）**：图与表的出口一律以**事件名**标识（settle/qualify/reopen/confirm/revise），到达状态只作箭头后缀说明——**不同阶段的同名到达态（如两个 resolved）靠事件名区分**，禁止拿状态名当出口标签。`confirm` 只在修复侧（fixed 行），澄清侧复核位三事件={settle, qualify, reopen}。

**新旧名对照（9-07 定稿原图措辞 → 本表事件名）**：resolve(note)=`clarified`；resolve(agree)=`settle`；resolve(fix)=`qualify`；wb_confirm_review/「复核确认」=`confirm`；reopen(澄清退回)=`reopen`；「修复完成标记/上报」=`declare-fixed`；reopen(修复退回)=`revise`；9-07 原文中的「clarifying 阶段」一词随 v5 退役（澄清=owner 处理 issue 的过程，非存储态）。

### 事件命名提案（待定稿——合并成单台后 resolve(fix)/(agree) 拆开重命名）

| 提案事件名 | 语义 | 现 API 名 | 出口 |
|---|---|---|---|
| `clarified` | owner 澄清完成申报 | wb_reply（open 行为） | open→pending-confirm |
| `settle` | 协商一致关闭（确认内嵌于协商） | wb_resolve path=agree | →resolved |
| `qualify` | 复核定性真问题（触发打回） | wb_resolve path=fix（定性半） | →to-fix |
| `reopen` | 复核退回·未解释清楚 | wb_reopen（pc 行为） | →open（loop++） |
| `declare-fixed` | owner 修复完成申报 | wb_reply（to-fix 行为） | to-fix→fixed |
| `confirm` | 复核确认关闭 | wb_confirm_review | →resolved |
| `revise` | 修复侧复核退回 | wb_reopen（fixed 行为） | →to-fix（loop++） |
| `override` | 强权仲裁（User 专属，冲突上升后） | wb_override_role / adjudicate | →resolved（仅 pending-confirm） |

> **「confirm 是节点位不是事件」落法**：复核位（澄清侧 pending-confirm 上的复核、修复侧 fixed 上的复核）拆成三个事件——澄清侧={settle, qualify, reopen}，修复侧={confirm, revise}。表中不设单列「confirm」。

### 状态表（设计权威版）

| 当前＼事件 | clarified | settle | qualify | reopen | declare-fixed | confirm | revise | override |
|---|---|---|---|---|---|---|---|---|
| **open** | pending-confirm | resolved（①提评者撤回） | —（D3） | — | — | — | — | — |
| **pending-confirm** | —（③已裁：无触发者） | resolved | to-fix（打回） | open（loop++） | — | — | — | **resolved** |
| **to-fix** | — | — | — | — | fixed | — | — | — |
| **fixed** | — | —（D2） | — | — | —（④已裁：无触发者） | resolved | to-fix（loop++） | resolved（②反复修复不成功上升） |
| **resolved** | — | — | — | — | — | — | — | — |

**override 裁决注记（2026-09-11，两轮批注）**：override 开**两格**——**pending-confirm→resolved**（澄清期冲突上升：issue 先回 pending-confirm 等仲裁，User 拍板 → resolved；不是发回 open 后并行仲裁，故 open 不开格）；**fixed→resolved**（修复期**反复修复不成功→上升**：判定主体=产出节点 re-exec 侧按返工次数（revise 计数）判断，判定时点 issue=fixed，User 仲裁 → resolved，不回 pending-confirm）。to-fix 不开（修复中间态不仲裁）；open 不开。

**已裁落格（2026-09-11 第二轮批注）**：
- ① **open×settle=resolved，限定场景=提评者撤回**——评审者重新审视后认为疑问已解/误报，直接 settle 关闭，不经 owner 澄清申报；操作者=提评者（根评论 author）。⚠️ 实现注意：现行 resolve 操作者=assignee（issues.go ValidateIssueActor），此格需实现层允许提评者 settle 自己的 open issue——随 D1-D6 返工一并开口。
- ② **fixed×override=resolved**：反复修复不成功→上升（判定主体=re-exec 侧返工次数），见 override 注记。
- ⑤ **上升判定分阶段定案**：澄清期=issue 内部 loop≥5 自动（-C7 现行维持）；修复期=**新立**「反复修复不成功上升」（node 层 re-exec 按返工次数）——落权威文档时同步模型 §1.5 与通知族。
- 命名：`clarified` 已获认可；其余事件名（settle/qualify/reopen/declare-fixed/confirm/revise）未获异议按提案默认执行，终稿前仍可改（改契约随 D5）。

**已裁落格（2026-09-11 第三轮批注）**：
- ③ pending-confirm×clarified = **—（非法）**；④ fixed×declare-fixed = **—（非法）**。
- **裁决依据（普适约束，入「表的读法」）**：除 user 外皆为 agent，**agent 只在被调用时响应，不存在自主行动**。issue 停在待复核态时，owner agent 已无未完成的调用——没有任何执行者能发出第二次 clarified / declare-fixed，「owner 想起来补充」这个场景在系统里不存在。「补充信息」的合法路径 = 复核者 **reopen**（pending-confirm→open / fixed→to-fix）：owner 被重新调用后重走申报，循环已覆盖，无需自循环格。
- **本约束是三张表的通用判定法**：任何事件的格子都必须答得出「**谁被调用之后触发它**」——答不出执行者即 `—`。此前 v1 表里多条实现通路（含 reply 自循环类）正是缺了这个判定才混进来的。

---

## 台 2：node.state 相位状态机（纯编排层，v2 清理版）

**状态（6 个）**：`""（未进入编排，不设占位）/ entry / exec / review / respond / done（业务终态，-L2）`。

**v2 清理**：v1 把评审轮机制（collect approved、gate_pending、提评动作）混进了本表事件列——评审轮在台 3（reviewState 层），门禁结果在本表只以「门禁 approved/rejected 事件」出现。**正交关系（模型 §1.2）：exec 期 reviewState 恒 '-'；review 期各评审位各自轮转（台 3），与本表相位独立。**

| 当前＼事件 | 激活（实例化/前节点门禁通过） | entry 相位动作完成 | exec 完成上报判据过 | 门禁 approved | 门禁 rejected（打回） | 澄清闭环重入 | 重新提测 | -L2 扫尾 | 复位（RollbackForRejectTx） |
|---|---|---|---|---|---|---|---|---|---|
| **""** | entry | — | — | — | — | — | — | — | — |
| **entry** | — | exec | — | — | — | — | — | done | — |
| **exec** | — | — | review（内审节点，轮 0）或**下一节点 entry**（无内审） | — | exec（回退重入，rolled_back_count++） | — | review（轮+1） | done | — |
| **review** | — | — | — | **下一节点 entry**（gate_pending 子事件：user_gate=1 等 User 拍板） | **回退目标节点 exec**（targetBranch 数据路由） | respond | — | done | — |
| **respond** | — | — | — | — | — | review（新轮，re_review 通知） | — | done | — |
| **done** | —（重复 advance 幂等 noop） | — | — | — | — | — | — | — | 重走（复位相位） |

来源：相位词表=模型 §1.1（4 相位）+ v1.0 增补 done（-L2）；门禁输出 approved/rejected=模型 §1.2a（last_decision 承载）；打回路由=模型 §1.4（targetBranch/缺陷 assignee 映射，非节点写死）；done 禁挂载/幂等/复位=注记-L2。

**待确认格（承 v1，仍开放）**：
1. entry→exec 迁移触发者：entry 相位动作（align.opening 类，executor=program）完成即迁，还是等用户确认？AC-V-06「entry hold 会话→用户确认」暗示 hold——**hold 是 entry 内子态还是 exec 前置事件**（模型既有结论：hold 是 action 内的强制等待指令，编排层无 hold 概念——本格按此口径应为「entry 动作自带等待，等待结束后上报完成→迁 exec」，请确认）。
2. respond→exec 直达：现口径**无直达**（澄清中发现真问题走 qualify→打回→exec 重入），表内保留非法。请确认。
3. 并行组 join：wait 不是独立 state（-I 裁决死分支）；对侧完成→整组推进=「门禁 approved」的并行组变体，表内不立行。

---

## 台 3：评审轮（reviewState 层——per-agent 表态）

> **✅ v4/v5 已定稿（2026-09-11 User 确认）**：reviewState = **per-agent 三态** reviewing/agree/unagree（模型 §1.2，2026-09-08 重构）；v4 的节点级词汇（unagreed/clarifying/confirming/agreed）退役——agreed=collect 全 agree 瞬时判定输出进门禁（last_decision 承载 approved/rejected），澄清=owner 澄清 issue 的过程、复表态=reviewer 回 reviewing，均不占存储态。spec §0 review_results 枚举与 rd 实现均为 v5。本表按 v5 定稿。

**per-agent 三态**：`reviewing（评审+表态中）/ agree（已表态·一致）/ unagree（已表态·不一致）`。节点级聚合=collect 内置逻辑（不暴露、无 state）。

| 当前＼事件 | 进 review 相位/新轮（round 更新） | agent 自判上报 agree（无 open 工单） | agent 自判上报 unagree（有 open 工单） | 覆写（轮内改主意） | collect 判定（全员已表态时，瞬时） |
|---|---|---|---|---|---|
| **reviewing** | reviewing（round=N） | agree | unagree（→owner 澄清，clarify_task 通知） | — | — |
| **agree** | —（新轮回 reviewing） | agree（幂等） | unagree | —（collect 是判定非态） | 全 agree → 门禁 approved（user_gate=1→gate_pending 等 User；=0→机械）；注记-L1 增补：**agree 前置=本位评论全闭环**（v1.0 口径，未闭环报 agree 被拒） |
| **unagree** | —（新轮回 reviewing） | agree（澄清完成后重新表态） | unagree（幂等） | 同左 | 有 unagree → 澄清循环持续；真问题（qualify）→ 门禁 rejected 打回 |

来源：模型 §1.2 v5（per-agent 三态/plugin 不判 open 工单）+ §1.2a collect 三段 + §1.3 裁决规则；-L1 闭环表态前置（v1.0 注记，对 §1.3「agent 自判」的实现级收紧）；覆写取最后=C-10。

**节点内 CR 异化特例**（模型 §1.3）：永不产生 rejected——unagree 循环（澄清→重表态）至全 agree。

**待拍板格**：
1. **上升（escalate）判定主体**——已分阶段裁（2026-09-11 第二轮）：澄清期=issue 内部 loop≥5 自动（-C7 维持）；修复期=node 层 re-exec 按返工次数判定（新立）。剩余小项：修复期阈值（revise≥N，N 取值建议与澄清期 5 对齐）+ 上升走 escalation 通知族复用。
2. v4/v5 定稿版本确认（见上 ⚠️）。
3. 新轮时旧 verdict 重建为 reviewing 的实现语义（qa_drive C-4 已断言重入新轮，存储语义需对齐）。

---

## 设计 vs 实现偏差清单（本次填表暴露，全部为返工候选，未改码）

| # | 偏差 | 设计口径 | 实现现状 | 返工面 |
|---|---|---|---|---|
| D1 | `overridden` 独立状态 | 无此态；override 动作→resolved | DDL issues.state 枚举含 overridden；issues.go 终态；注记-L1「批量置 overridden」；C-6/C-7/qa_drive 断言 overridden | 迁移枚举+domain 状态机+escalation/override 两路径+spec 注记-L1/-L4（BUG-008 修复注记）+qa_drive C-6/C-7/E 组 |
| D2 | settle（resolve agree）自 to-fix/fixed → resolved | 修复子机出口仅 confirm→resolved / revise→to-fix | issues.go 两格允许 | issues.go 删两键+对应用例 |
| D3 | qualify（resolve fix）自 open → to-fix | 定性只发生在复核位（pending-confirm 三出口②） | issues.go 允许 open 直达 to-fix | issues.go 删一键；「owner 澄清中自认真问题」场景若要保留→先回澄清流程定案 |
| D4 | confirm 语义 | confirm=复核节点位，拆事件（settle/qualify/reopen + confirm/revise） | wb_confirm_review 单端点单事件（pc/fixed→resolved） | 契约拆分或更名（与事件命名提案一并定） |
| D5 | 事件命名 | resolve(path) 合并机内重名 | wb_resolve path=fix/agree 双路径单端点 | 按命名提案改契约（可后置，先改表后改码） |
| D6 | reply 幂等自循环（pc/fixed 追加回复） | 按「agent 只在被调用时响应」判定：owner 自主追加=无触发者，非法；复核中的评审者调用在身，「追问式 reply」是否允许=待裁 | issues.go 对双方开放幂等追加 | 待裁：收紧为仅复核中评审者，或一并禁止（追问走 reopen 已覆盖） |

## 过目后动作（更新）

1. 台 1 待拍板 4 格 + 台 3 待拍板 3 格逐格定；
2. 事件命名提案定稿（决定 D5 是否随本轮改契约）；
3. 偏差清单 D1-D6 逐条定处置（改码返工 / 暂挂记 defect）；
4. 定稿表落权威文档（模型附录或 qa-test-cases 附录）+ C 组按表补断言。
