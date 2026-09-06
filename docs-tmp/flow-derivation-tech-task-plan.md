# tech / task / plan 三流程 · 节点状态挂载配置 v3.1（逐 state 单列）

> **状态**：2026-09-05 v3.1 定稿稿（格式与 business v3.1 同构：每节点所有 state 单列，逐 state 明确挂载 action，含跳过相位；蒸馏视角缺漏自查已过，见 §6）。模型权威=`capability-model-final.md` v3；**通用约定见 `business-flow-derivation.md` §0**（两层属性/激活清单/计数位/回退路由/幂等/自审禁令/escalate 全局规则/gate 语义/依赖调度预留——三流程全部适用，不重复）；动作挂载=`capability-actions-final.md`。

---

## 1. tech 技术需求流程（11 节点）

```
B0 待启动 → B1 需求对齐 → B2 RD技术设计 → B3 RD技术评审(按需) → B4 FE技术设计(按需)
→ B5 FE技术评审(按需) → B8a RD开发 ∥ B8b FE开发 → B9 QA验收 → B10 用户验收 → B11 完成
```

### B0 待启动

```yaml
node_type: gate | gate: 0 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | 离开判据 |
|---|---|---|
| —— | 无可配相位（门禁直判） | User 确认投入资源 → approved → B1 |

**reviewState**: 恒 '-'。

### B1 需求对齐

```yaml
node_type: align | gate: 0 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | align.opening（node action） | program 渲染开场白（等待指令） | 响应者=pm subagent | 用户完成并告知 → PM 上报 complete |
| exec | to-brd（node action，公共 skill） | to-brd | pm（直启） | brd 落盘+判据校验 |
| review / respond | ——跳过（对齐结论无评审轮） | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：exec 校验通过 → approved → B2。**产物**：brd（✅D1：统一产 brd 喂 rd_design；to-brd=公共 skill 复用）。已知取舍：brd 无评审回路（同 business，质量由对齐期 User 确认背书）。

### B2 RD 技术设计

```yaml
node_type: produce | gate: 1 | user_gate: 0 | scope: be | reviewers: [ {cr: code-review} ]
# 内嵌 CR 轮（同 business 节点 4）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | backend-design.produce | to-rd-techdesign（接口面原则进门禁） | rd | 完成上报+判据 → 提评（plugin 轮转）→ review |
| exec 重入 | backend-design.revise | revise（响应 B3 rejected——基于评审结论逐条修订） | rd | 完成上报+校验（按 B3 rejected 返工）→ 重新提评 → review 重开 |
| review | code-review.review（CR 评本节点 rd_design） | code-review-skill | cr | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | backend-design.fix_on_review（**只修复不澄清**——节点内 CR 异化裁决） | inline | rd | 全 resolve → confirming |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决（机械，无 rejected 出口）→ approved → 提评迁移 B3。
**产物**：rd_design。inputs 动态注入（✅D5）：tech=brd / business=product_doc。

### B3 RD 技术评审（按需）

```yaml
node_type: review | gate: 1 | user_gate: 1 | scope: be
reviewers: [ {arch(be): arch.rd-design-review}, {fe: fe.rd-design-review}, {qa: qa.rd-design-review} ]
# 无 rd 位（产出者不自审）、无 pm 位（技术需求无产品视角评审）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（**审批类节点 review=产出动作位，exec 不配置**） | —— | —— | —— |
| review | arch.rd-design-review.review / fe.rd-design-review.review / qa.rd-design-review.review（评上一节点 rd_design） | 各自对象化评审 skill（rd-design-review-{arch,fe,qa}-skill） | arch / fe / qa | barrier → 有 open → clarifying；无 → agreed |
| respond | backend-design.clarify（单动作；修复=B2 exec 重入 revise） | inline | rd | 全 resolve → confirming；reopen → rejected → B2 exec 重入 |

**reviewState 轮转**：collecting → clarifying → confirming → [reopen 循环] → agreed → 裁决。
**裁决**：无 reject → User 终审 → approved（迁移 B4）；含 reject → rejected → 回退 B2 exec 重入；本节点维持 agreed。

### B4 FE 技术设计（按需）

```yaml
node_type: produce | gate: 1 | user_gate: 0 | scope: fe | reviewers: [ {cr: code-review} ]
# 内嵌 CR 轮（同 B2）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | frontend-design.produce | to-fe-techdesign | fe | 完成上报+判据 → 提评（plugin 轮转）→ review |
| exec 重入 | frontend-design.revise | revise（响应 B5 rejected——基于评审结论逐条修订） | fe | 完成上报+校验（按 B5 rejected 返工）→ 重新提评 → review 重开 |
| review | code-review.review（CR 评本节点 fe_design） | code-review-skill | cr | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | frontend-design.fix_on_review（**只修复不澄清**——节点内 CR 异化裁决） | inline | fe | 全 resolve → confirming |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决（机械，无 rejected 出口）→ approved → 提评迁移 B5。
**产物**：fe_design。

### B5 FE 技术评审（按需）

```yaml
node_type: review | gate: 1 | user_gate: 0 | scope: fe
reviewers: [ {qa: qa.fe-design-review} ]   # 单评审位——机械判定
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（审批类 exec 不配置） | —— | —— | —— |
| review | qa.fe-design-review.review（评上一节点 fe_design） | fe-design-review-qa-skill | qa | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | frontend-design.clarify（单动作；修复=B4 exec 重入 revise） | inline | fe | 全 resolve → confirming；reopen → rejected → B4 exec 重入 |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决。
**裁决（机械）**：无 reject → approved（放行 B8a∥B8b）；含 reject → rejected → 回退 B4 exec 重入；本节点维持 agreed。

### B8a / B8b 并行开发（内嵌 CR——节点内评审异化）

```yaml
node_type: produce | gate: 1 | user_gate: 0 | branch: rd|fe | parallel_group: dev | scope: be|fe
reviewers: [ {cr: code-review} ]
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | backend-dev.dev（B8a）/ frontend-dev.dev（B8b） | implement-techdesign | rd / fe | 完成上报+判据（提评登记内联）→ plugin 轮转 → review |
| exec 重入 | backend-dev / frontend-dev.diagnosing-bugs | diagnosing-bugs | rd / fe | 完成上报+校验（按 B9 rejected 返工）→ 重新提评 |
| review | code-review.review（评本节点产物——节点内评审异化） | code-review-skill（Standards→Spec） | cr | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | backend-dev / frontend-dev.fix_on_review（**只响应 CR，不做澄清**——CR 异化裁决） | inline | rd / fe | 全 resolve → confirming |

**reviewState 轮转**：collecting → clarifying（修复响应）→ confirming（复核）→ [reopen 循环] → agreed → 裁决。
**裁决（机械）**：**无 rejected 出口**（节点内评审异化）——agreed → approved（并行组 join 重算）→ B9。
**回退进入**：B9 rejected(targetBranch) → plugin 回退本节点 → exec 重入（diagnosing-bugs）。
**产物**：dev_notes, code_mr。

### B9 QA 验收

```yaml
node_type: produce | gate: 1 | user_gate: 0 | reviewers: []
# qa 不自审——验收结论=exec 产出本身（无 review 相位、无评审位）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | acceptance-test.accept | qa-acceptance（用例设计+执行+AC 核对；缺陷登记 defect） | qa | 完成上报+判据 → respond |
| exec 重入 | acceptance-test.补充缺陷 | inline：「登记 B10 报障澄清的新缺陷并分派」 | qa | 完成上报+校验 → respond 重开 |
| review | ——跳过（**无评审位——qa 不自审**） | —— | —— | —— |
| respond | backend-dev.clarify / frontend-dev.clarify（rd/fe 各自缺陷确认/归属争议——跨节点路由，按 assignee 分派；修复在 B8a/B8b exec 重入） | inline | rd / fe | 全 resolve → confirming（qa 复核缺陷响应）→ agreed |

**reviewState 轮转**：'-' → clarifying → confirming → agreed → 裁决。
**裁决（机械）**：无 reject 争议残留 → approved（迁移 B10）；真缺陷残留 → rejected → targetBranch=qa 分派 → plugin 回退 B8a/B8b exec 重入。
**产物**：test_result, defect。

### B10 用户验收

```yaml
node_type: gate | gate: 1 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | 会话通道迎宾呈现（绑定 qa） | 开场白模板（绑定配置） | 响应者=qa | User 终拍板 |
| exec / review / respond | ——跳过（门禁类） | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：approve → B11；rejected → 回退 B9 exec 重入（补充缺陷）。

### B11 完成：DONE 占位。

---

## 2. task 任务流程（8 节点）

```
C0 待启动 → C1 需求对齐 → C2 技术设计 → C3 技术评审 → C4 开发 → C5 QA验收 → C6 用户验收 → C7 完成
```

### C0 待启动

```yaml
node_type: gate | gate: 0 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | 离开判据 |
|---|---|---|
| —— | 无可配相位 | User 确认 → approved → C1 |

### C1 需求对齐

```yaml
node_type: align | gate: 0 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | align.opening（node action） | program 渲染开场白（等待指令） | 响应者=执行者 subagent | 用户完成并告知 → 执行者上报 complete |
| exec | ——**跳过（✅D2：维持无产物）** | —— | —— | —— |
| review / respond | ——跳过 | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：用户确认对齐完成 → approved → C2。**无产物**——对齐结论=简单描述，C2 基础（my_task 上下文注入）。

### C2 技术设计

```yaml
node_type: produce | gate: 1 | user_gate: 0 | reviewers: [ {cr: code-review} ]
# 内嵌 CR 轮（同 B2）
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | plan-cap.produce | to-plan（基础=C1 对齐简单描述——✅D2：无 prd/brd，my_task 注入） | 执行者（rd|fe by scope） | 完成上报+判据 → 提评迁移 C3 |
| exec 重入 | plan-cap.revise | revise（响应 C3 rejected——基于评审结论逐条修订） | 执行者 | 完成上报+校验（按 C3 rejected 返工）→ 重新提评 |
| review | code-review.review（CR 评本节点 plan——技术设计文档） | code-review-skill | cr | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | plan-cap.fix_on_review（**只修复不澄清**——节点内 CR 异化裁决） | inline | 执行者 | 全 resolve → confirming |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决（机械，无 rejected 出口）→ approved → 提评迁移 C3。**产物**：plan。

### C3 技术评审

```yaml
node_type: review | gate: 1 | user_gate: 1 | reviewers: [ {arch: arch.plan-review} ]   # 单评审位
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（审批类 exec 不配置） | —— | —— | —— |
| review | arch.plan-review.review（评上一节点 plan） | plan-review-arch-skill | arch | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | plan-cap.clarify（单动作；修复=C2 exec 重入 revise） | inline | 执行者 | 全 resolve → confirming；reopen → rejected → C2 exec 重入 |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决。
**裁决**：无 reject → User 终审 → approved（迁移 C4）；含 reject → rejected → 回退 C2 exec 重入；本节点维持 agreed。

### C4 开发（内嵌 CR——节点内评审异化）

```yaml
node_type: produce | gate: 1 | user_gate: 0 | reviewers: [ {cr: code-review} ]   # 单分支无并行组
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | backend-dev.dev 或 frontend-dev.dev（by scope） | implement-techdesign | 执行者 | 完成上报+判据（提评登记内联）→ plugin 轮转 → review |
| exec 重入 | backend-dev / frontend-dev.diagnosing-bugs | diagnosing-bugs | 执行者 | 完成上报+校验（按 C5 rejected 返工）→ 重新提评 |
| review | code-review.review（评本节点产物——节点内评审异化） | code-review-skill | cr | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | backend-dev / frontend-dev.fix_on_review（**只响应 CR，不做澄清**） | inline | 执行者 | 全 resolve → confirming |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决。
**裁决（机械）**：**无 rejected 出口**——agreed → approved → C5。
**回退进入**：C5 rejected → plugin 回退本节点 → exec 重入。
**产物**：dev_notes, code_mr。

### C5 QA 验收

```yaml
node_type: produce | gate: 1 | user_gate: 0 | reviewers: []   # qa 不自审
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | acceptance-test.accept | qa-acceptance | qa | 完成上报+判据 → respond |
| exec 重入 | acceptance-test.补充缺陷 | inline：「登记 C6 报障澄清的新缺陷并分派」 | qa | 完成上报+校验 → respond 重开 |
| review | ——跳过（无评审位） | —— | —— | —— |
| respond | backend-dev.clarify / frontend-dev.clarify（执行者 缺陷确认/归属争议——跨节点路由，按 assignee 分派；修复在 C4 exec 重入） | inline | 执行者 | 全 resolve → confirming（qa 复核）→ agreed |

**reviewState 轮转**：'-' → clarifying → confirming → agreed → 裁决。
**裁决（机械）**：无 reject 争议残留 → approved（迁移 C6）；真缺陷残留 → rejected → 回退 C4 exec 重入。
**产物**：test_result, defect。

### C6 用户验收

```yaml
node_type: gate | gate: 1 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | 会话通道迎宾呈现（绑定 qa） | 开场白模板 | 响应者=qa | User 终拍板 |
| exec / review / respond | ——跳过（门禁类） | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：approve → C7；rejected → 回退 C5 exec 重入（补充缺陷）。

### C7 完成：DONE 占位。

---

## 3. plan 版本规划流程（9 节点）

```
D0 待启动 → D1 立项需求对齐 → D2 立项评审 → D3 调研 → D4 调研评审 → D5 架构规划 → D6 架构规划评审 → D7 创建需求·启动版本 → D8 完成
```

### D0 待启动

```yaml
node_type: gate | gate: 0 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | 离开判据 |
|---|---|---|
| —— | 无可配相位 | User 确认 → approved → D1 |

### D1 立项需求对齐

```yaml
node_type: align | gate: 0 | user_gate: 1 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | align.opening（node action） | program 渲染开场白（等待指令） | 响应者=arch subagent | 用户完成并告知 → arch 上报 complete |
| exec | to-alignment（node action，公共 skill） | to-alignment | arch（直启） | alignment 落盘+判据校验 |
| review / respond | ——跳过（D2 跨节点评审） | —— | —— | —— |

**reviewState**: 恒 '-'。
**裁决**：exec 校验通过 → approved → D2。**产物**：alignment（版本级 `docs/{{req-version}}/align.md`）。✅D3：交互对齐+固化（用户可调多 skills：wayfinder/改进项/遗留 TODO/使用发现，to-alignment 汇总）。

### D2 立项评审

```yaml
node_type: review | gate: 1 | user_gate: 1 | reviewers: [ {pm: pm.alignment-review} ]
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（**审批类节点 review=产出动作位，exec 不配置**） | —— | —— | —— |
| review | pm.alignment-review.review（评上一节点 alignment） | alignment-review-pm-skill | pm | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | alignment-cap.clarify（单动作；修复=D1 exec 重入 to-alignment 重写） | inline | arch | 全 resolve → confirming；reopen → rejected → D1 exec 重入 |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决。
**裁决**：无 reject → User 终审 → approved（迁移 D3）；含 reject → rejected → 回退 D1 exec 重入；本节点维持 agreed。

### D3 调研

```yaml
node_type: produce | gate: 0 | user_gate: 0 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | research-cap.produce | research（已有） | inv | 完成上报+判据 → 提评迁移 D4 |
| exec 重入 | research-cap.revise | revise（响应 D4 rejected——基于评审结论逐条修订） | inv | 完成上报+校验（按 D4 rejected 返工）→ 重新提评 |
| review / respond | ——跳过（评审在 D4，跨节点） | —— | —— | —— |

**reviewState**: 恒 '-'。**产物**：research。

### D4 调研评审

```yaml
node_type: review | gate: 1 | user_gate: 0 | reviewers: [ {arch: arch.research-review} ]   # 单评审位——机械
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（审批类 exec 不配置） | —— | —— | —— |
| review | arch.research-review.review（评上一节点 research） | research-review-arch-skill | arch | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | research-cap.clarify（单动作；修复=D3 exec 重入 revise） | inline | inv | 全 resolve → confirming；reopen → rejected → D3 exec 重入 |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决。
**裁决（机械）**：无 reject → approved（迁移 D5）；含 reject → rejected → 回退 D3 exec 重入；本节点维持 agreed。

### D5 架构规划

```yaml
node_type: produce | gate: 0 | user_gate: 0 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec 首次 | arch-cap.produce | to-arch | arch | 完成上报+判据 → 提评迁移 D6 |
| exec 重入 | arch-cap.revise | revise（响应 D6 rejected——基于评审结论逐条修订） | arch | 完成上报+校验（按 D6 rejected 返工）→ 重新提评 |
| review / respond | ——跳过（评审在 D6，跨节点） | —— | —— | —— |

**reviewState**: 恒 '-'。**产物**：architecture（含需求拆解节——D7 依赖边来源）。

### D6 架构规划评审

```yaml
node_type: review | gate: 1 | user_gate: 1 | reviewers: [ {pm: pm.alignment-review} ]
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry / exec | ——跳过（审批类 exec 不配置） | —— | —— | —— |
| review | pm.architecture-review.review（评上一节点 architecture） | architecture-review-pm-skill | pm | barrier（单位立即）→ 有 open → clarifying；无 → agreed |
| respond | arch-cap.clarify（单动作；修复=D5 exec 重入 revise） | inline | arch | 全 resolve → confirming；reopen → rejected → D5 exec 重入 |

**reviewState 轮转**：collecting → clarifying → confirming → [循环] → agreed → 裁决。
**裁决**：无 reject → User 终审 → approved（迁移 D7）；含 reject → rejected → 回退 D5 exec 重入；本节点维持 agreed。

### D7 创建需求·启动版本

```yaml
node_type: produce | gate: 0 | user_gate: 0 | reviewers: []
```

| state | 挂载动作 | skill / inline | 执行者 | 离开判据 |
|---|---|---|---|---|
| entry | ——跳过 | —— | —— | —— |
| exec | decide-cap.create_reqs | to-tickets | **leader（直执行，不起 subagent——✅#2 连带）** | 完成上报+判据（需求全创建落库+版本推进）→ approve → D8 |
| review / respond | ——跳过（机械执行，D6 已拍板） | —— | —— | —— |

**reviewState**: 恒 '-'。
**产物**：requirements[]（逐需求创建+**写入 dependencies[]（需求拆解节依赖边）**+priority）+ 版本推进。

### D8 完成：DONE 占位。

---

## 4. 跨流程复用验证矩阵（16 能力 × 4 流程）

| 能力 | business | tech | task | plan |
|---|---|---|---|---|
| product-design | 节点 2 | — | — | — |
| backend-design | 节点 4 | B2 | — | — |
| frontend-design | 节点 6 | B4 | — | — |
| research-cap | — | — | — | D3 |
| arch-cap | — | — | — | D5 |
| plan-cap | — | — | C2 | — |
| alignment-cap | — | — | — | D2（评审响应） |
| backend-dev | 节点 8 | B8a | C4(scope=rd) | — |
| frontend-dev | 节点 9 | B8b | C4(scope=fe) | — |
| prd-review-{arch,rd,fe} | 节点 3 | — | — | — |
| rd-design-review-{arch,fe,qa,pm} | 节点 5 | B3（无 pm 位） | — | — |
| fe-design-review-qa | 节点 7 | B5 | — | — |
| plan-review-arch | — | — | C3 | — |
| alignment-review-pm | — | — | — | D2 |
| research-review-arch | — | — | — | D4 |
| architecture-review-pm | — | — | — | D6 |
| code-review | 节点 4/6（评设计文档）+ 8/9（评代码） | B2/B4 + B8a/B8b | C2 + C4 | — |
| acceptance-test | 节点 10 | B9 | C5 | — |
| decide-cap | ——（join=plugin） | —— | — | D7(create_reqs, leader 直执行) |
| ~~final-approve~~ | 退役（user_gate 取代） | | | |

> 机制差异仅三处：**评审位配置位置**（节点内 CR=本节点 reviewers / 跨节点=独立评审节点）+ **reject 权**（跨节点评审输出 rejected 回退上一节点；节点内 CR 永不输出——不通过回 respond 修复）+ **respond 挂载**（节点内评审=fix_on_review 单挂不澄清；跨节点评审=clarify+fix_on_review 双动作 R1）。rest 全同一套相位与 reviewState 词汇。

## 5. D1-D5 裁决记录（2026-09-05，全部定稿）

| # | 决策 | 裁决 |
|---|---|---|
| D1 | tech B1 对齐产物 | 统一产 brd；to-brd=公共 skill（business/tech 复用） |
| D2 | task C1 对齐 | 维持无产物；C2 基础=C1 对齐简单描述（mytask 注入） |
| D3 | plan D1 形态 | 交互对齐+to-alignment 公共 skill 固化（连带 alignment-cap 收缩为 D2 评审响应能力） |
| D4 | D4 调研评审 | 维持 arch 评审、用户不参与（机械）；对照 C3 保留 User=任务级唯一方向点 |
| D5 | inputs 动态注入 | 确认——skill 内写明上下文依赖来源，实现阶段逐个校验 |

## 6. 蒸馏自查记录（缺漏扫描，2026-09-05，v3.1 格式复核后仍成立）

**扫描维度**：全动作覆盖 / 正反成对 / 自审禁令 / gate 语义 / User 终审配置位 / 依赖调度预留 / state 单列完整性。

**结论：无结构性缺漏；一项待拍板遗留。**

1. **全动作覆盖**：16 能力 × 3 流程逐一落位（§4 矩阵）。
2. **正反成对**：三流程全部评审轮 rejected→回退路径落位（B3→B2、B5→B4、C3→C2、D2→D1、D4→D3、D6→D5、B9→B8a/B8b、C5→C4）；escalate 全局规则兜底轮内不收敛。
3. **自审禁令全量核验（零残留）**：B3（无 rd ✓）、C3（arch 评 executor 的 plan ✓）、D2（pm 评 arch 的 alignment ✓）、D4（arch 评 inv 的 research ✓）、B9/C5（无 qa 位 ✓）。
4. **机械/拍板分工跨流程一致**：单专业评审位（B5 qa、D4 arch）→机械；多评审位含方向确认（B3、C3、D2、D6）→User 拍板。
5. **state 单列完整性**：每节点 4 相位逐一列出（含跳过标注）；reviewState 轮转逐节点标注（恒 '-' 或完整链）。
6. **已知取舍落盘**：tech brd 无评审；task C1 无产物（C2 基础=简单描述）；D7 leader 直执行完成判据=数据落库。

**评审能力对象化（2026-09-05 用户裁决：最小上下文准则）**：跨节点评审能力（pm/arch/rd/fe/qa-tech-review）废除——评审能力按「对象×角色」切分（{role}.{object}-review，如 arch.prd-review），inputs 仅含评审对象，skill 仅含该对象维度。**ATAM 权衡点标注**：能力碎片化（5→12） vs 每节点最小 ctx——裁最小上下文（用户准则）。rd-tech-review 退役问题随之消解（对象化后自然不存在）。能力池 17→**24**（12 基础 + 12 对象化评审）。
