# 业务需求流程（business）· 节点配置终稿 v3（两层属性状态机完整带入）

> **状态**：2026-09-05 v3 定稿稿。模型权威见 `capability-model-final.md` v3（两层属性状态机：node.state 可配相位 4 值 + node.reviewState 审批子状态 7 值）；动作→skill 挂载见 `capability-actions-final.md`。
> **本文件 = business 流程 13 节点的完整节点配置**（可直接回填 seed 的粒度）：每个节点给出 node_type / 相位序列 / node_actions 激活清单 / reviewers / user 拍板位 / reviewState 流转 / 裁决与流转 / 产物。
> **拓扑**：13 节点（用例设计/用例设计评审已删）。scope 裁剪（business 有 scope）：RD 侧=be、FE 侧=fe，其余恒在。

---

## 0. 通用配置约定（全部节点适用）

```
两层属性：
  node.state      可配动作相位（entry/exec/review/respond，按 node_type 实例化子集）
  node.reviewState 审批子状态（'-' / collecting / clarifying / confirming / agreed / approved / rejected，plugin 轮转不可配）

节点配置字段（flow_def_nodes 列）：
  template_node_id / node_type(produce|review|align|gate) / gate(0|1)
  reviewers[]: [{role, capability_id, scope?}]        ← 审批 skills map 派生源
  node_actions[]: [{capability_id, actions:[…]}]       ← 节点动作激活清单（配置权在节点）
  user_gate(0|1)                                       ← User 终审拍板位（agree 条件唤醒）
  branch / parallel_group / scope / expected_outputs

计数位：rolled_back_count（0=首次，≥1=重入）——entry/exec 多配置按计数选择
回退路由：agree 判 rejected → plugin 回退被重做节点（targetBranch/assignee 映射）→ 目标节点 exec 重入
完成上报：exec 完成闭环 → event:'complete' → plugin 判据校验 → 轮转（提评迁移/review 相位）
```

**节点表（13）**：

```
0 待启动 → 1 需求对齐 → 2 产品设计 → 3 产品设计评审 → 4 RD技术设计 → 5 RD技术评审
→ 6 FE技术设计 → 7 FE设计评审 → 8 RD开发 ∥ 9 FE开发 → 10 QA验收 → 11 用户验收 → 12 完成
```

---

## 节点 0：待启动

```yaml
node_type: gate
gate: 0
state 相位: （无——门禁直判）
reviewers: []
node_actions: []
user_gate: 1          # User 确认投入资源即拍板
scope: ''
reviewState: 恒 '-'
```

流转：User 确认 → approved → 迁移节点 1。无 agent 动作。

---

## 节点 1：需求对齐

```yaml
node_type: align
gate: 0
state 相位: entry → exec(to-brd) → approve
node_actions:
  - align.opening   # entry：开场白（program 渲染，模板存对话型 skill；how 内置等待指令）
  - to-brd          # exec：node action（executor_role=pm，公共 skill——business/tech 复用）
reviewers: []
user_gate: 1          # 用户完成对齐（告知 PM→PM 上报）→ brd 落盘 → approve 迁移
scope: ''
reviewState: 恒 '-'
expected_outputs: [brd]   # docs/{{req-version}}/{{req-name}}/brd/brd.md
```

说明：自由对话（entry 期，非动作）；对齐完成信号=用户告知响应者（PM）→ PM 上报；program 兜底=周期提醒。**to-brd/to-alignment=公共 skill，不属 pm 能力**。

---

## 节点 2：产品设计

```yaml
node_type: produce
gate: 0
state 相位: exec（首次 / 重入）——无 review（评审在节点 3）
node_actions:
  - capability: product-design
    actions:
      - { produce, on: exec(首次), skill: to-prd }            # 产出 product_doc
      - { diagnosing-bugs, on: exec(重入), skill: diagnosing-bugs }  # 响应节点 3 rejected 打回
reviewers: []
user_gate: 0
scope: ''
reviewState: 恒 '-'（评审轮在节点 3）
expected_outputs: [product_doc]
```

流转：exec 完成（完成上报+判据：product_doc 落库）→ **提评迁移节点 3**。节点 3 rejected → plugin 回退本节点 → exec 重入（diagnosing-bugs 基于评审结论返工）→ 重新提评 → 节点 3 review 重开。

---

## 节点 3：产品设计评审

```yaml
node_type: review
gate: 1
state 相位: review（评审位表态）→ respond（owner 响应）→ review 循环 → agree → 裁决
node_actions:
  - capability: product-design          # owner 响应（跨节点路由：pm 从节点 2 进入 respond）
    actions:
      - { clarify, on: respond }         # 澄清意见
      - { fix_on_review, on: respond }   # 修复（reopen 驱动）
reviewers:
  - { role: arch, capability_id: arch-tech-review }
  - { role: rd,   capability_id: rd-tech-review,  scope: be }
  - { role: fe,   capability_id: fe-tech-review,  scope: fe }
user_gate: 1          # 全 agent 通过 → 唤醒 User 终审（评审者含 reject → 自动 rejected）
scope: ''
reviewState 流转: collecting（收集表态 barrier）→ clarifying（pm respond）→ confirming（复核）→ [reopen→clarifying 循环] → agreed → approved / rejected
```

裁决：approved → 产品设计完成，迁移节点 4；rejected → 回退节点 2 exec 重入。冲突上升（clarify/confirm 不收敛）→ escalate → 会话通道裁决 session → 结论回填 issue。

---

## 节点 4：RD 技术设计

```yaml
node_type: produce
gate: 0
state 相位: exec（首次 / 重入）
node_actions:
  - capability: backend-design
    actions:
      - { produce, on: exec(首次), skill: to-rd-techdesign }   # 接口面原则进门禁
      - { diagnosing-bugs, on: exec(重入) }                     # 响应节点 5 rejected
reviewers: []
user_gate: 0
scope: be
reviewState: 恒 '-'
expected_outputs: [rd_design]
```

流转：同节点 2（提评迁移节点 5）。

---

## 节点 5：RD 技术评审

```yaml
node_type: review
gate: 1
state 相位: review → respond → review 循环 → agree → 裁决
node_actions:
  - capability: backend-design          # owner 响应（rd 从节点 4 进入 respond）
    actions: [ { clarify, on: respond }, { fix_on_review, on: respond } ]
reviewers:
  - { role: arch, capability_id: arch-tech-review, scope: be }
  - { role: rd,   capability_id: rd-tech-review,   scope: be }   # 自评+评审
  - { role: fe,   capability_id: fe-tech-review }                # 评接口契约可对接（inputs=rd_design+product_doc）
  - { role: qa,   capability_id: qa-tech-review }                # 可测性/联调/安全性维度
  - { role: pm,   capability_id: pm-review }                     # 需求覆盖
user_gate: 1
scope: be
reviewState 流转: collecting → clarifying → confirming → [循环] → agreed → approved / rejected
```

裁决：approved → 迁移节点 6；rejected → 回退节点 4 exec 重入（diagnosing-bugs）。

---

## 节点 6：FE 技术设计

```yaml
node_type: produce
gate: 0
state 相位: exec（首次 / 重入）
node_actions:
  - capability: frontend-design
    actions:
      - { produce, on: exec(首次), skill: to-fe-techdesign }   # 页面对照 rd 接口契约，不自造契约
      - { diagnosing-bugs, on: exec(重入) }
reviewers: []
user_gate: 0
scope: fe
reviewState: 恒 '-'
expected_outputs: [fe_design]
```

流转：同节点 2/4（提评迁移节点 7）。

---

## 节点 7：FE 设计评审

```yaml
node_type: review
gate: 1
state 相位: review → respond → review 循环 → agree → 裁决
node_actions:
  - capability: frontend-design         # owner 响应（fe 从节点 6 进入 respond）
    actions: [ { clarify, on: respond }, { fix_on_review, on: respond } ]
reviewers:
  - { role: qa, capability_id: qa-tech-review }   # 单评审位
user_gate: 0          # 无 User 拍板位——机械判定
scope: fe
reviewState 流转: collecting → clarifying → confirming → agreed → approved / rejected
```

裁决（机械）：agreed → approved（放行 8∥9 并行组）；rejected → 回退节点 6 exec 重入。

---

## 节点 8：RD 开发（内嵌 CR——节点内评审异化）

```yaml
node_type: produce
gate: 1
state 相位: exec（首次 / 重入）→ review（CR）→ respond（修复）→ review 循环 → agree → 裁决
node_actions:
  - capability: backend-dev
    actions:
      - { dev,             on: exec(首次), skill: implement-techdesign }  # 内部编排 to-techdesign 校验+tdd/implement
      - { diagnosing-bugs, on: exec(重入), skill: diagnosing-bugs }        # 响应节点 10 rejected（qa 缺陷）
      - { fix_on_review,   on: respond }                                    # 只响应 CR（不做澄清——CR 异化裁决）
  - capability: code-review
    actions:
      - { review,  on: review }   # Standards→Spec 两子步（worktree 读代码）
      - { confirm, on: replied 对应复核期 }
reviewers:
  - { role: cr, capability_id: code-review }
user_gate: 0          # CR 无 reject 输出（节点内评审异化）——全确定即 approved
branch: rd
parallel_group: dev
scope: be
reviewState 流转: collecting（CR 表态 barrier）→ clarifying（fix_on_review 修复响应）→ confirming（复核）→ agreed → approved / rejected
expected_outputs: [dev_notes, code_mr]
```

裁决：agreed → approved（并行组 join）→ QA 验收；**节点内评审无 rejected 出口**。节点 10 rejected(targetBranch=rd) → plugin 回退本节点 → exec 重入（diagnosing-bugs 响应缺陷）→ 重新提评 → review 重开 → 复验。

---

## 节点 9：FE 开发——与节点 8 完全同构（fe 侧）

```yaml
node_type: produce
gate: 1
branch: fe
parallel_group: dev
scope: fe
node_actions: [frontend-dev 同构（dev→implement-techdesign / diagnosing-bugs / fix_on_review）+ code-review（review/confirm）]
reviewers: [ { role: cr, capability_id: code-review } ]
user_gate: 0
expected_outputs: [dev_notes, code_mr]
```

---

## 节点 10：QA 验收（用例设计+执行验收一体）

```yaml
node_type: produce
gate: 1
state 相位: exec（首次 / 重入）→ review（qa 表态）→ respond（rd/fe 缺陷确认）→ review 循环 → agree → 裁决
node_actions:
  - capability: acceptance-test
    actions:
      - { accept, on: exec(首次), skill: qa-acceptance }   # 用例设计+执行+AC 核对；缺陷登记 defect（triage 纪律）
      - { 补充缺陷, on: exec(重入), inline: "登记节点 11 报障澄清的新缺陷并分派" }   # 来源=downstream-reject
  - capability: backend-dev      # rd 缺陷响应（跨节点路由）
    actions: [ { clarify, on: respond } ]   # 缺陷确认/归属争议（澄清阶段；修复在节点 8 exec 重入）
  - capability: frontend-dev     # fe 缺陷响应
    actions: [ { clarify, on: respond } ]
  - capability: qa-tech-review
    actions: [ { review, on: review }, { confirm, on: 复核期 } ]
reviewers:
  - { role: qa, capability_id: qa-tech-review }   # 单评审位
user_gate: 0          # 机械判定
scope: ''
reviewState 流转: collecting（qa 表态+缺陷登记 barrier）→ clarifying（rd/fe 缺陷确认/归属争议——前端 bug 根因后端的分歧在此解决并可改派）→ confirming（qa 复核）→ agreed → approved / rejected
expected_outputs: [test_result, defect]
```

裁决（机械）：agreed → approved（迁移节点 11）；rejected（真缺陷残留）→ **targetBranch=qa 分派**（缺陷 assignee 映射 rd/fe/both）→ plugin 回退节点 8/9 exec 重入（diagnosing-bugs）→ 重新提评 → join 重算 → 本节点 review 重开（复验：accept 重跑，回归范围从 defect 继承）。

---

## 节点 11：用户验收

```yaml
node_type: gate
gate: 1
state 相位: entry（qa 迎宾呈现 + 会话通道）→ approve
node_actions:
  - 会话通道绑定: qa    # 用户报障对话（plugin 泛用接口：拉起角色会话+开场白模板）
reviewers: []
user_gate: 1          # User 终拍板（对照 test_result AC + defect 遗留风险）
scope: ''
reviewState: 恒 '-'
```

裁决：approve → 迁移节点 12；rejected（用户报障经 qa 会话澄清确认）→ plugin 回退节点 10 → exec 重入（补充缺陷）→ qa 分派 → 既有打回循环。死循环兜底：User 与 qa 反复分歧走线下，流程内不建模。

---

## 节点 12：完成

```yaml
node_type: gate
gate: 0
state 相位: （无——DONE 占位）
reviewers: []
node_actions: []
user_gate: 0
```

流转到达即流程终态（程序确认收尾）。

---

## 附：跨节点响应路由汇总（respond 相位的激活与路由）

| 节点 | respond 激活 | 响应者（从哪来） |
|---|---|---|
| 3 产品设计评审 | product-design.clarify/fix_on_review | pm（节点 2 owner） |
| 5 RD 技术评审 | backend-design.clarify/fix_on_review | rd（节点 4 owner） |
| 7 FE 设计评审 | frontend-design.clarify/fix_on_review | fe（节点 6 owner） |
| 8 RD 开发 | backend-dev.fix_on_review | rd（本节点，CR 异化——只修复不澄清） |
| 9 FE 开发 | frontend-dev.fix_on_review | fe |
| 10 QA 验收 | backend-dev/frontend-dev.clarify | rd/fe（跨节点路由，缺陷确认/归属争议） |
| 11 用户验收 | ——（会话通道绑 qa） | qa（报障澄清，结论回填 issue） |
