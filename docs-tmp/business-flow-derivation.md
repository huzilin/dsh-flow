# 业务需求流程（business）· 节点状态挂载配置 v3.1（逐 state 单列）

> **状态**：2026-09-05 v3.1（格式修正：每节点所有 state 单列，逐 state 明确挂载 action——含跳过相位，配置无歧义）。模型权威=`capability-model-final.md` v3；动作→skill=`capability-actions-final.md`。
> **拓扑**：13 节点。**需求 scope 三值：full（rd+fe）/ rd / fe**（2026-09-07 术语统一：be→rd，与角色名一致）。
>
> **scope × 节点渲染矩阵**：

| 节点 | scope 标注 | full | rd | fe |
|---|---|---|---|---|
| 0 待启动 / 1 需求对齐 / 2 产品设计 / 3 产品设计评审 | ''（中性） | ✓ | ✓ | ✓ |
| 4 RD 技术设计 / 5 RD 技术评审 / 8 RD 开发 | rd | ✓ | ✓ | — |
| 6 FE 技术设计 / 7 FE 设计评审 / 9 FE 开发 | fe | ✓ | — | ✓ |
| 10 QA 验收 | full | ✓ | — | — |
| 11 用户验收 / 12 完成 | ''（中性） | ✓ | ✓ | ✓ |
> **通用约定**（全节点适用，详见各节不重复）：两层属性（node.state 可配相位 + node.reviewState 审批子状态 plugin 轮转不可配）；rolled_back_count 计数位（0=首次，≥1=重入，entry/exec 多配置按计数选）；回退路由=rejected → plugin 回退被重做节点（targetBranch/assignee 映射）→ 目标节点 exec 重入；幂等声明（event:'complete' 幂等、回退按 req+node+round 幂等）；自审禁令（reviewers[] 不得含本节点产物 owner）；escalate 全局规则（轮内不收敛→会话通道 User 仲裁，送审含正反方意见）；gate=1 ⟺ 节点有准出判定；brd 修正路径（评审异议→User→leader 同步 brd）。

**节点表**：

```
0 待启动 → 1 需求对齐 → 2 产品设计 → 3 产品设计评审 → 4 RD技术设计 → 5 RD技术评审
→ 6 FE技术设计 → 7 FE设计评审 → 8 RD开发 ∥ 9 FE开发 → 10 QA验收 → 11 用户验收 → 12 完成
```

---

## 节点 0：待启动

```yaml
node_type: gate | gate: 0 | user_gate: 1 | scope: '' | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| —— | 无可配相位（门禁直判） | —— | —— | User 确认投入资源 → approved → 节点 1 |

**reviewState**: 恒 '-'。

---

## 节点 1：需求对齐

```yaml
node_type: align | gate: 0 | user_gate: 1 | scope: '' | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | align.opening（node action） | program 渲染开场白（how 内置等待指令） | 响应者=pm subagent | 用户完成并告知 → PM 上报 complete |
| exec | to-brd（node action） | to-brd 公共 skill | pm（直启） | brd 落盘+判据校验 |
| review | ——跳过（对齐结论无评审轮） | —— | —— | —— |
| respond | ——跳过 | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：exec 校验通过 → approved（用户确认唯一信号）→ 节点 2。
**产物**：brd（`docs/{{req-version}}/{{req-name}}/brd/brd.md`）。已知取舍：brd 无评审回路（质量由对齐期 User 确认背书；修正=评审异议上升 User → leader 同步 brd）。

---

## 节点 2：产品设计

```yaml
node_type: produce | gate: 0 | user_gate: 0 | scope: '' | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过（非交互型） | —— | —— | —— |
| exec 首次 | product-design.produce | to-prd | pm | 完成上报+判据（product_doc 落库）→ 提评迁移节点 3 |
| exec 重入 | product-design.revise | revise（响应节点 3 rejected——基于评审结论逐条修订） | pm | 完成上报+校验（按节点 3 rejected 结论返工）→ 重新提评 |
| review | ——跳过（评审在节点 3，跨节点；**prd 不内嵌 CR**——CR 职责域=技术产物，✅N4 2026-09-05） | —— | —— | —— |
| respond | ——跳过（响应在节点 3，跨节点路由） | —— | —— | —— |

**reviewState**: 恒 '-'。
**产物**：product_doc。

---

## 节点 3：产品设计评审

```yaml
node_type: review | gate: 1 | user_gate: 1 | scope: '' 
reviewers: [ {arch: arch.prd-review}, {rd|full: rd.prd-review}, {full: fe.prd-review} ]
# pm 不设评审位——产出者不自审（2026-09-05 裁决，旧模型 pm-review@节点3 废除）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过（非交互型） | —— | —— | —— |
| exec | ——不配置（**审批类节点 review=产出动作位**） | —— | —— | —— |
| review | arch.prd-review.review / rd.prd-review.review / fe.prd-review.review（评上一节点 product_doc） | 各自对象化评审 skill（prd-review-{arch,rd,fe}-skill：仅含 prd 评审维度——最小 ctx） | arch / rd / fe | 全表态（实时判据：verdict 全非空，不存储）→ 有 open → clarifying；无 → agreed |
| respond | product-design.clarify（单动作——**一个 state 一个动作**；修复=节点 2 exec 重入 revise） | inline | pm | 全 resolve → confirming；reopen（需修改）→ rejected → 节点 2 exec 重入 |

**reviewState 轮转**：reviewing → clarifying → confirming → [reopen → clarifying 循环] → agreed → 裁决。
**裁决**：无 reject → user_gate 唤醒 User 终审 → 通过 → approved（产品设计完成，迁移节点 4）；含 reject → 门禁 rejected → plugin 回退节点 2 exec 重入；本节点维持 agreed（等重提）。冲突上升 → escalate → 会话通道 User 仲裁（送审含正反方意见）；brd 层面异议 → User → leader 同步 brd。

---

## 节点 4：RD 技术设计

```yaml
node_type: produce | gate: 0 | user_gate: 0 | scope: rd|full | reviewers: []
# 编写文档节点没有 review 相位（2026-09-05 用户澄清共识；此前内嵌 CR 补裁作废——undermined 入模型 §8）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | backend-design.produce | to-rd-techdesign（接口面原则进门禁） | rd | 完成上报+判据（rd_design 落库）→ 提评迁移节点 5 |
| exec 重入 | backend-design.revise | revise（响应节点 5 rejected——基于评审结论逐条修订） | rd | 完成上报+校验（按节点 5 rejected 返工）→ 重新提评 |
| review / respond | ——跳过（评审在节点 5，跨节点） | —— | —— | —— |

**reviewState**: 恒 '-'。
**产物**：rd_design。

---

## 节点 5：RD 技术评审

```yaml
node_type: review | gate: 1 | user_gate: 1 | scope: rd|full
reviewers: [ {arch(rd|full): arch.rd-design-review}, {full: fe.rd-design-review}, {qa: qa.rd-design-review}, {pm: pm.rd-design-review} ]
# fe 位按需：需求含 fe（full）时才参与（rd-only 无接口消费方）——位级 scope（2026-09-07 用户裁决）
# rd 不设评审位——产出者不自审（2026-09-05 举一反三，旧模型「自评+评审」废除）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（审批类 exec 不配置） | —— | —— | —— |
| review | arch.rd-design-review.review / fe.rd-design-review.review / qa.rd-design-review.review / pm.rd-design-review.review（评上一节点 rd_design） | 各自对象化评审 skill（rd-design-review-{arch,fe,qa,pm}-skill） | arch / fe / qa / pm | 全表态（实时判据） → 有 open → clarifying；无 → agreed |
| respond | backend-design.clarify（单动作；修复=节点 4 exec 重入 revise） | inline | rd | 全 resolve → confirming；reopen（需修改）→ rejected → 节点 4 exec 重入 |

**reviewState 轮转**：reviewing → clarifying → confirming → [循环] → agreed → 裁决。
**裁决**：无 reject → User 终审 → 门禁 approved（迁移节点 6）；含 reject → 门禁 rejected → 回退节点 4 exec 重入；本节点维持 agreed。

---

## 节点 6：FE 技术设计

```yaml
node_type: produce | gate: 0 | user_gate: 0 | scope: fe|full | reviewers: []
# 编写文档节点没有 review 相位（同节点 4）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | frontend-design.produce | to-fe-techdesign（页面对照 rd 接口契约，不自造契约） | fe | 完成上报+判据 → 提评迁移节点 7 |
| exec 重入 | frontend-design.revise | revise（响应节点 7 rejected——基于评审结论逐条修订） | fe | 完成上报+校验（按节点 7 rejected 返工）→ 重新提评 |
| review / respond | ——跳过（评审在节点 7，跨节点） | —— | —— | —— |

**reviewState**: 恒 '-'。
**产物**：fe_design。

---

## 节点 7：FE 设计评审

```yaml
node_type: review | gate: 1 | user_gate: 0 | scope: fe|full
reviewers: [ {qa: qa.fe-design-review} ]   # 单评审位——机械判定
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（审批类 exec 不配置） | —— | —— | —— |
| review | qa.fe-design-review.review（评上一节点 fe_design，对照 rd 接口契约） | fe-design-review-qa-skill | qa | 实时判据（单评审位立即成立）→ 有 open → clarifying；无 → agreed |
| respond | frontend-design.clarify（单动作；修复=节点 6 exec 重入 revise） | inline | fe | 全 resolve → confirming；reopen（需修改）→ rejected → 节点 6 exec 重入 |

**reviewState 轮转**：reviewing → clarifying → confirming → [循环] → agreed → 门禁机械 approved；含 reject → 门禁 rejected（放行 8∥9 并行组）；含 reject → 门禁 rejected → 回退节点 6 exec 重入；本节点维持 agreed。

---

## 节点 8：RD 开发（内嵌 CR——节点内评审异化）

```yaml
node_type: produce | gate: 1 | user_gate: 0 | scope: rd|full | branch: rd | parallel_group: dev
reviewers: [ {cr: code-review} ]
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过（非交互型） | —— | —— | —— |
| exec 首次 | backend-dev.dev | implement-techdesign（内部编排 to-techdesign 校验 + tdd/implement 工人） | rd | 完成上报+判据（dev_notes/MR 落库+自测全绿+提评登记）→ plugin 轮转 → review |
| exec 重入 | backend-dev.diagnosing-bugs | diagnosing-bugs | rd | 完成上报+校验（按节点 10 rejected 返工，重新提评登记）→ plugin 轮转 → review 重开 |
| review | code-review.review（评本节点产物——节点内评审异化） | code-review-skill（Standards→Spec 两子步，worktree 读代码） | cr | 实时判据（单评审位立即成立）→ 有 open → clarifying；无 → agreed |
| respond | backend-dev.fix_on_review（**只响应 CR，不做澄清**——CR 异化裁决） | inline | rd | 全 resolve → confirming |

**reviewState 轮转**：reviewing → clarifying（修复响应）→ confirming（复核）→ [reopen → clarifying 循环] → agreed → 裁决。
**裁决（机械）**：**无 rejected 出口**（节点内评审异化）——agreed → approved（并行组 join）→ 迁移节点 10。
**回退进入**：节点 10 rejected(targetBranch=rd) → plugin 回退本节点 → exec 重入（diagnosing-bugs 响应缺陷）。
**产物**：dev_notes, code_mr。

---

## 节点 9：FE 开发（内嵌 CR）——与节点 8 完全同构（fe 侧）

```yaml
node_type: produce | gate: 1 | user_gate: 0 | scope: fe|full | branch: fe | parallel_group: dev
reviewers: [ {cr: code-review} ]
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| exec 首次 | frontend-dev.dev | implement-techdesign | fe | 同节点 8 |
| exec 重入 | frontend-dev.diagnosing-bugs | diagnosing-bugs | fe | 同节点 8 |
| review | code-review.review | code-review-skill | cr | 同节点 8 |
| respond | frontend-dev.fix_on_review（只响应 CR） | inline | fe | 同节点 8 |

**reviewState 轮转 / 裁决 / 回退进入**：同节点 8（targetBranch=fe）。**产物**：dev_notes, code_mr。

---

## 节点 10：QA 验收（含联调）

```yaml
node_type: produce | gate: 1 | user_gate: 0 | scope: full | reviewers: []
# **仅 scope=full 渲染**（2026-09-07 用户裁决：QA 验收=联调职责，单侧需求无联调 → 节点不渲染）
# qa 不自审——验收结论=exec 产出本身（无 review 相位、无评审位）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | acceptance-test.accept | qa-acceptance（用例设计+执行+AC 核对；缺陷登记 defect，triage 纪律） | qa | 完成上报+判据（test_result/defect 落库）→ respond |
| exec 重入 | acceptance-test.retest（复验重测） | inline：「针对重新提测的代码重新执行测试→核对是否修复→更新 defect 文档（已解决标记/新增缺陷登记并分派）」 | qa | 完成上报+校验 → respond 重开 |
| review | ——跳过（**无评审位——qa 不自审**） | —— | —— | —— |
| respond | backend-dev.clarify / frontend-dev.clarify（rd/fe 各自缺陷确认/归属争议——跨节点路由，按 assignee 分派；修复在节点 8/9 exec 重入） | inline | rd / fe | 全 resolve → confirming（qa 复核缺陷响应）→ agreed |

**reviewState 轮转**：'-' → clarifying（缺陷确认/归属争议——前端 bug 根因后端的分歧在此解决并可改派）→ confirming（qa 复核）→ agreed → 裁决。
**裁决（机械）**：无 reject 争议残留 → approved（迁移节点 11）；真缺陷残留 → rejected → targetBranch=qa 分派（rd/fe/both）→ plugin 回退节点 8/9 exec 重入。
**产物**：test_result, defect。

---

## 节点 11：用户验收

```yaml
node_type: gate | gate: 1 | user_gate: 1 | scope: '' | reviewers: []
# 会话通道绑定按 scope 动态（2026-09-07 用户裁决）：full=qa / fe-only=fe / rd-only=rd
# 单侧需求节点 10 不渲染——用户验收直接对接开发节点（报障不经 QA 中转）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | 会话通道迎宾呈现（绑定角色按 scope 动态：full=qa / fe-only=fe / rd-only=rd） | 开场白模板（绑定配置） | 响应者=绑定角色 | User 终拍板 |
| exec / review / respond | ——跳过（门禁类） | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：User 通过 → approved → 节点 12；rejected（报障经绑定角色澄清确认）→ plugin 回退路由按 scope：full → 节点 10 exec 重入（retest）；单侧 → 开发节点（8/9）exec 重入（diagnosing-bugs）。死循环兜底：反复分歧走线下。

---

## 节点 12：完成

```yaml
node_type: gate | gate: 0 | user_gate: 0 | reviewers: [] | node_actions: []
```

| state | 挂载动作 | 离开判据 |
|---|---|---|
| —— | 无（DONE 占位，程序确认收尾） | 流程终态 |

---

## 附：respond 跨节点路由汇总

| 节点 | respond 激活 | 响应者（从哪来） |
|---|---|---|
| 3 产品设计评审 | product-design.clarify | pm（节点 2 owner） |
| 5 RD 技术评审 | backend-design.clarify | rd（节点 4 owner） |
| 7 FE 设计评审 | frontend-design.clarify | fe（节点 6 owner） |
| 8 RD 开发 | backend-dev.fix_on_review | rd（本节点，CR 异化——只修复不澄清） |
| 9 FE 开发 | frontend-dev.fix_on_review | fe |
| 10 QA 验收 | backend-dev/frontend-dev.clarify | rd/fe（跨节点，缺陷确认/归属争议） |
| 11 用户验收 | ——（会话通道绑 qa） | qa（报障澄清，结论回填 issue） |
