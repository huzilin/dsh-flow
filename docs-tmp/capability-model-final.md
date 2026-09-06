# MVP 节点编排模型 v3（两层属性状态机 · 定稿）

> **状态**：2026-09-05 用户逐轮推演定稿。本文件=节点编排模型权威层。配套：capability-actions-final.md（动作→skill 挂载表）、business-flow-derivation.md / flow-derivation-tech-task-plan.md（流程实例化，待按 v3 词汇同步）、data-tools-final.md（schema DDL 与工具契约，待产出）。
> **v3 相对历版的收敛**（全部经用户场景验证）：trigger 词表退役 → 编排相位=唯一词汇；单列状态白名单废除 → **两层属性状态机**（node.state 可配动作相位 + node.reviewState 内置审批子状态）；re-entry/re-exec 并入 entry/exec（计数位选择）；评审双轨制（节点内 CR 异化 + 跨节点评审节点）；**hook 层全部退役**（预留通知/同步扩展位）；how 四要素退出框架（skill 挂载制）。

---

## 1. 两层属性状态机（核心）

### 1.1 node.state —— 可配动作相位（4 值）

动作可挂载的编排阶段（BPMN user-task 性质）。**节点未进入编排 = 无 state（不设占位）**。

| state | 语义 | 典型挂载 |
|---|---|---|
| **entry** | 进入节点：首次=开场迎宾；重入=重迎宾（多配置按 rolled_back_count 选） | 开场白 action（how 内置强制等待指令）——交互型专属 |
| **exec** | 执行：**首次=主产出**（produce/dev/accept→skill）；**重入=diagnosing-bugs**（响应打回：复现→定位→修复→回归，rolled_back_count≥1 触发） | skill/inline |
| **review** | 评审相位：评审位按审批 skills map 各自评审表态；**无内嵌审批的节点跳过**；**审批类节点以 review 为产出动作位（exec 不配置）** | review→skill（各评审位） |
| **respond** | owner 响应期：澄清（clarify）与修复（fix_on_review）；**开发节点特化=只挂 fix_on_review（不做澄清，CR 异化裁决）** | clarify / fix_on_review（按节点激活清单） |

### 1.2 node.reviewState —— 内置审批子状态（7 值，plugin 轮转，不可配）

| reviewState | 语义 | 迁移 |
|---|---|---|
| **'-'** | 未评审（exec 期/无评审节点恒为 '-'） | exec 完成上报+判据校验 → collecting |
| **collecting** | 收集表态（评审位各自落库，plugin 收集 barrier——多审批者全落库） | barrier 达成 → 有 open 则 clarifying，无 open 则 agreed |
| **clarifying** | owner 响应中（respond 相位激活，澄清任务派发） | 全部 resolve → confirming |
| **confirming** | 复核中（confirm/reopen；reopen → clarifying 循环） | 全部确定 → agreed |
| **agreed** | 收口：全部 open 项确定 | plugin 裁决（见 1.3） |
| **approved** | 裁决结果：通过 | 节点迁移下一节点 |
| **rejected** | 裁决结果：打回 | plugin 回退目标节点（见 1.4） |

**正交关系**：state 与 reviewState 两列独立——exec 期 reviewState 恒 '-'；review 相位内 reviewState 循环 collecting→clarifying→confirming→agreed。UI 分开显示「当前执行动作」与「当前审批状态」（GitHub state+reviewDecision 同构）。

### 1.3 裁决规则（agree 收口，plugin 判定）

```
agree（全部 open 项确定）→ plugin 收集多审批者结论（barrier）：
   ├─ 评审者含 reject → rejected（裁决事件，人感知的结论与表态）
   │     实际反应不在本节点——plugin 回退到上一个节点（被重做节点）→ 其 exec 重入（diagnosing-bugs）
   │     发起 reject 的评审节点：维持 agreed（本轮结论=reject，等重提）→ 重提后重新 review
   └─ 无 reject →
        ├─ 节点配置 user 拍板位 → 唤醒 User 终审（会话通道呈现）
        │     User 通过 → approved；User 不通过 → 问题记 issue → clarifying 循环 → 循环至全 agree → 重新裁决
        └─ 未配置 → 直接 approved
```

**节点内评审（CR 异化）永不输出 rejected**——不通过 → 回 respond 直接修复（不做澄清），复核循环至通过。

### 1.4 回退与重入

- **reject（裁决事件）→ plugin 回退被重做节点**（数据路由：targetBranch / 缺陷 assignee 映射，非节点写死逻辑）→ 目标节点 exec 重入（diagnosing-bugs）→ 完成上报 → 重新提评 → 评审相位重开（collecting）。
- **重入识别**：rolled_back_count（0=首次，≥1=重入）——entry/exec 的多配置按计数选择。
- 并行组 2对1：只回滚被重做分支（targetBranch），其余分支维持 approved，join 重算。

---

## 2. 节点类型实例化（统一词汇，四类）

| 节点类型 | 可配相位（node.state） | reviewState | 例 |
|---|---|---|---|
| **产出类（有内审）** | exec（首/重入）→ review → respond → review… | '-'→collecting→clarifying→confirming→agreed→approved/rejected | RD 开发（CR 内嵌） |
| **产出类（无内审）** | exec（首/重入） | 恒 '-'（评审在下游审批类节点） | 产品设计 |
| **审批类** | **review（产出动作位，exec 不配置）** | collecting→clarifying→respond→…→agreed→approved/rejected | 产品设计评审 / RD 技术评审 / FE 设计评审 |
| **交互类** | entry → exec(to-brd) | 恒 '-' | 需求对齐 |
| **门禁类** | entry（迎宾呈现+会话通道） | 恒 '-' | 用户验收 |

交互/门禁类无 collect/review（空转规则：无挂载相位直过）。

---

## 3. 评审双轨制

| | 节点内评审（CR 异化） | 跨节点评审 |
|---|---|---|
| 评审对象 | **本节点产物**（代码/MR） | **上一个节点的产物** |
| 适用 | **仅开发节点**（异化，不推广） | 独立评审节点承载（产品设计评审等） |
| 相位 | dev 节点 review 相位（collecting→respond→confirming 循环） | 评审节点同一套相位 |
| 不通过 | **回 respond 直接修复（不做澄清）——永不输出 rejected** | agree 判定输出 rejected → 回退上一节点 |
| User 终审 | 无（机械收口） | 按节点 user 拍板配置位 |

---

## 4. 轮转图

### 4.1 dev 节点（内嵌 CR，全生命周期）

```
exec（dev 首次 → implement-techdesign）→ 完成上报+校验
   → review（reviewState: collecting → CR 表态 barrier）
      → 有 open → respond（fix_on_review 修复，不做澄清）→ review（confirming）→ reopen 循环
      → agreed → approve（终态，join）→ QA 验收
QA 打回 → plugin 回退 → exec 重入（diagnosing-bugs 响应缺陷）→ 重新提评 → review（collecting 重开）→ … → approve → QA 复验
```

### 4.2 跨节点评审节点（如 RD 技术评审）

```
[上游 exec 完成提评迁移激活]
   → review（arch/rd/fe/qa/pm 审批 skills map；collecting → barrier）
      → 有 open → respond（owner clarify + fix_on_review，跨节点路由）→ review（confirming）→ 循环
      → agreed → User 终审（配置位）→ approve（上游节点完成，流转）| rejected → 上游 exec 重入 → 重提 → review 重开
```

---

## 5. 动作 schema 与挂载（skill 挂载制）

```
action = { action_id, name, on(相位), skill?, inline?, outputProductTypes }
capability = { capability_id, role_id, inputs[], actions[] }
```

- **skill 挂载制**：复杂动作挂 skill（内部按需编排其他 skill、自带目标/自检/门禁）；短动作一句话内联；**how 四要素退出框架**（how-spec-v2 降级为 skill 写作参考）。
- **依赖注入**：my_task 上下文（{action_id, skill, nodeId, reqId, inputs 引用}）；inputs[]=跨流程可见性上限，按流程可用产物动态注入（skill 内写明来源，实现阶段校验）。
- **节点动作激活清单**：能力动作=通用词表；**节点决定激活子集**（flow_template_nodes.node_actions=[{capability_id, actions:[…]}]，与 reviewers[] 同模式）。按域分立（2026-09-05 用户纠错）：**开发节点激活 [dev, diagnosing-bugs(exec 重入——响应打回), fix_on_review(respond——只响应 CR 不做澄清)]，无 clarify**；**文档产出节点激活 [produce, revise(exec 重入——响应打回修订), clarify(respond——只澄清不修复)]，无 diagnosing-bugs**；backend-dev/frontend-dev.clarify 仅 QA 验收节点激活（缺陷确认/归属争议）。
- **issue 即产物**：评审类动作产出=issue（plugin 内一等数据，展示于文档/节点 comment），outputProductTypes 空=正确语义。
- **完成上报**：长任务/交互等待动作完成时 `event:'complete'`（上报即数据写入，plugin 校验判据，谎报拒绝，遗漏→扫描器兜底）；短动作（表态/澄清/复核）数据写入即完成，不上报。

### 5.1 dev 节点激活清单（定稿）

| 相位 | 激活动作 | 执行者 |
|---|---|---|
| exec（首次） | dev → implement-techdesign | rd |
| exec（重入） | diagnosing-bugs（响应打回） | rd |
| respond | fix_on_review（只响应 CR，不做澄清） | rd |
| review | code-review（评审位） | cr |

QA 验收节点激活：[accept, backend-dev/frontend-dev.clarify（缺陷确认/归属争议）, 补充缺陷(inline, exec 重入)]。

---

## 6. 推进契约（agent ↔ plugin）

1. **响应者规则**：有挂载动作的相位必有响应者（谁等结果谁负责推进）；executor≠响应者（开场白 executor=program，响应者=拉入的 subagent）。
2. **完成信号双通道**：短动作（表态/澄清/复核）=数据写入即完成（plugin 判据推导）；长任务/交互等待=响应者调标准完成接口（上报即数据写入，plugin 校验判据防谎报，遗漏→扫描器兜底）。与工单 04「数据层即返回」精化衔接。
3. **reject/escalation 结论协议**：branches（单节点留空/并发明确）+ confirmed_issues（从 open issues）+ branch 上下文注入 + escalate 标记；producer 双通道（响应者提交/User 拍板 plugin 自动装配）。
4. **空转规则**：相位无挂载 → plugin 直接轮转（视为通过）。
5. **三重匹配派发**：判据成立+节点声明相位挂载+存在匹配数据；缺一环空转降级。
6. **冲突响应机制**：clarify/confirm 循环不收敛 → escalate 标记 → plugin 拉起按需会话通道裁决 session（响应者=reviewer，开场白=分歧点+双方立场）→ 用户裁决 → **结论追加到冲突 issue**（数据层留痕、node 可见）→ reviewer 重新表态落库 → 正常判据链恢复。
7. **按需会话通道**：plugin 泛用接口（拉起角色会话+开场白模板随绑定配置），前端按节点绑定表达；实例：对齐=PM、拍板呈现=owner、冲突上升=reviewer、用户验收报障=qa——不占相位词汇。

---

## 7. 数据层与边界

- **轮转内置逻辑（plugin，不可配）**：collect barrier 收集、clarifying/confirming 派发、agree 裁决、approve/reject 输出——审批子状态全部 plugin 驱动；可配的只有动作相位上的 action。
- **外部兜底扫描器**：宿主侧扫描（工单 07 实装）——发现中断/悬置 → 重推相位/唤醒/周期提醒（对齐节点确认悬置）。
- **project 注册层**（Q6-Q8 已裁）：per-project workbench 隔离（每工作区一套 .workbench/ + sqlite，11 表零改动）；project registry=plugin 全局配置（~/.dsh/wbflow/projects.json）；MVP 无 agent 跨项目操作（wb_* 按当前工作区隐式定位）；新增 plugin 接口：project 注册/列出/切换/停用⚠️。
- **需求/版本本体**：JSON 源承载（.workbench/requirements.json + versions.json），sqlite 投影——现状不动。

---

## 8. 作废留痕（undermined，勿再引用）

| 作废项 | 被推翻原因 | 取代 |
|---|---|---|
| v1 hook 表（7+1：entry/exec/feedback/replied/reject/exit/transform+task_stale，action 挂 hook） | 轮次制推演；trigger 与编排混同 | 两层属性状态机（§1） |
| trigger 8 词表（含 reviewed/replied/approve/reject 为编排词） | 编排相位=唯一词汇（用户裁决统一 state） | 相位 4 值+reviewState 7 值（approve/reject=裁决事件名） |
| collect/reviewed/replied/complete/ready 作为节点相位 | 相位=有等待的编排步骤；收集/复核降为 reviewState；complete/ready=瞬时信号非相位 | §1.2 reviewState + 完成上报信号 |
| 评审节点无 exec / 评审节点恢复 exec（两次摇摆） | exec/review 拆分定稿：审批类节点 exec 不配置、review=产出动作位 | §2 审批类实例化 |
| re-entry / re-exec 相位 | 并入 entry/exec（计数位选择配置）——plugin 回退动作保留 | §1.4 |
| Q-F「fix_on_review 统一并入 diagnosing-bugs」 | 用户二次纠正：两动作两挂载（fix_on_review@respond 只响应 CR；diagnosing-bugs@exec 重入响应打回） | §5.1 dev 激活清单 |
| qa 意见直通打回 | qa 意见=可测性/安全性设计维度，同走轮次澄清 | §1.3 轮次循环 |
| retest 独立动作 | 复验=accept 重跑组成 | §4.1 |
| respond_to_cr / respond_to_qa 独立动作 | 统一 clarify（来源无关）+ fix_on_review 分立 | §5.1 |
| after-exec hook（提评+评审派发） | 两层属性模型吸收其职责；hook 层全部退役 | §5 完成上报 + §1.2 轮转 |
| 取消独立评审节点（提案） | 评审双轨制确立：跨节点评审需独立节点承载 | §3 |
| pending / done / gate 状态占位 | 编排进度展示无占位；approve=通过终态 | §1.1 |
| pm-review@产品设计评审（自审位，gen_flow_rows:88） | 文档评审产出者不自审（节点=送评委）；自审在产出阶段 | §2 审批类 + business 配置 |
| decide-cap.decide（join 裁决动作） | join 裁决=plugin 内置轮转吸收；decide-cap 收缩为 create_reqs 单动作 | §1.3 + capability-actions-final |
| final-approve 能力 | User 终审职责由 flow_def_nodes.user_gate 配置位承载 | §1.3 |

---

## 9. 待同步（执行清单）

- [ ] capability-actions-final.md：submit 行（并入 exec 闭环内联）、review 行（on=review 相位）、respond 行名对齐
- [ ] business-flow-derivation.md / flow-derivation-tech-task-plan.md：按 v3 相位词汇与双轨制重排
- [ ] data-tools-final.md（工单 07）：DDL（flow_defs/flow_def_nodes 更名、def_node_id、reopen_source、escalate、node_actions 激活清单、user 拍板位、product_type、sessions 新表、project registry）+ 24 工具契约（my_task 上下文/wb_assign_task/wb_task_update event:'complete'/resolve-reopen escalate 参数）+ seed 重写
- [ ] semantica 溯源（v3 模型定稿）
