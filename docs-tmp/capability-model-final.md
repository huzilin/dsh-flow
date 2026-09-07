# MVP 节点编排模型 v3（两层属性状态机 · 定稿）

> **状态**：2026-09-05 用户逐轮推演定稿。本文件=节点编排模型权威层。配套（均已产出并同步）：capability-actions-final.md（动作→skill 挂载表）、business-flow-derivation.md / flow-derivation-tech-task-plan.md（四流程 v3.1 节点配置）、data-tools-final.md（schema v0.29 DDL 与 23 工具契约）。
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

### 1.2 评审状态（reviewState）—— per-agent 表态轮转（2026-09-08 v5 重构）

**reviewState 是每一个评审位（agent）各自的轮转状态，不是节点整体状态**——10 个 agent 即各自独立的一份（1/10）；节点级聚合由 collect 内置逻辑完成（不暴露、无专门 state）。

**per-agent 三态**：
| 态 | 语义 | 出口 |
|---|---|---|
| reviewing | 评审+表态进行中 | agent 自判上报表态值：**知道自己有 open 工单 → 报 unagree；没有 → 报 agree**（plugin 不判 open 工单） |
| agree（已表态） | 达成一致 | 等 collect 聚合 |
| unagree（已表态） | 未达成一致 | 等 collect 聚合 → 触发 owner 澄清 |

**澄清后重新表态（内置机制，不设专门 state）**：owner 澄清后，plugin 将澄清 issue 发给对应 reviewer（followup 该 reviewer 会话 / 复用既有通道）→ 该 reviewer **回 reviewing 重新表态** → collect 重新收集。这个「等待重新表态」的阶段不挂载任何 action、不暴露——它是审批内置逻辑。

### 1.2a collect · 聚合收集（plugin 内置，不暴露）

完整动作三段：①**收票**——是否全表态完成（实时查询各 reviewer verdict）；②**判定**——**全 agree → approved（进入下一节点）；有 unagree → 触发 owner 澄清 → 回①重新收集循环**；③**再收票**——澄清后重新等待 reviewer 表态（重复流程）。确认真问题（to-fix）→ rejected 回上一节点（沿既有裁决打回）。

**approved / rejected 是 collect 的输出结果**（汇总级，写入 last_decision）——review 状态（per-agent）不包含它们。

**正交关系**：state 与 reviewState 两列独立——exec 期 reviewState 恒 '-'（无评审位节点亦然）；review 期间各评审位各自的 reviewState 轮转 reviewing→agree/unagree（per-agent，非节点整体）。UI 分开显示「当前执行动作」与「各评审位表态」（per-agent）。

### 1.3 裁决规则（collect 输出，plugin 内置）

```
collect 收集全部评审位上报（agree / unagree）：
  ├─ 全 agree → approved → 门禁（user_gate=1 唤醒 User 终审：通过 → 迁移下一节点；不通过 → 问题记 issue 回澄清循环）
  │                                  （user_gate=0 → 机械 approved）
  └─ 有 unagree → owner 澄清（plugin 派发澄清任务）→ 澄清后 plugin 将 issue 发回对应 reviewer 重新表态（回 reviewing）→ 重新收集循环
                     └─ 确认真问题（to-fix）→ 打回：rejected → 上一节点 exec 重入
```

- 节点内评审（CR 异化）永不产生 rejected——unagree 循环（澄清→重表态）至全 agree。
- 上升（escalate）：循环不收敛 → 会话通道 User 仲裁 → 结论回填 issue → reviewer 重新表态。

### 1.4 回退与重入

- **reject（裁决事件）→ plugin 回退被重做节点**（数据路由：targetBranch / 缺陷 assignee 映射，非节点写死逻辑）→ 目标节点 exec 重入（diagnosing-bugs）→ 完成上报 → 重新提评 → 评审相位重开（reviewing）。
- **重入识别**：rolled_back_count（0=首次，≥1=重入）——entry/exec 的多配置按计数选择。
- 并行组 2对1：只回滚被重做分支（targetBranch），其余分支维持 approved，join 重算。

---

## 1.5 评审轮内部小状态机（2026-09-07 详细设计）

**① collect · 无状态机（纯等待）**：每个评审位 agent 独立轮转 reviewing → 上报 agree / unagree（agent 自判——有 open 工单报 unagree，没有报 agree；plugin 不判）。推进规则：实时查询 review_results 全部 agent 均已上报（无 reviewing）即进入判定，无子机。

**② 澄清阶段 · issue 状态机**（owner respond 相位，逐 issue）：
```
open（待澄清）→ pending-confirm（已澄清待复核）
pending-confirm 出口三向：
  ① resolved（澄清解释了疑问）
  ② to-fix 待修复（确定是真问题 → 触发打回）
  ③ open（未解释清楚，reopen 计数 +1）
```

**③ 修复阶段 · issue 状态机**（产出节点 exec 重入相位，逐 issue）：
```
to-fix（待修复）→ fixed（已修复）→ 复核 → resolved（修复成功）/ to-fix（失败回退，revise 计数 +1）
```

**计数器**：单字段 loop_count（回退计数），阶段语义解释——澄清阶段读=reopen 次数、修复阶段读=revise 次数。

**UI 视角规则**：reviewing 看表态标签矩阵（各评审位 × reviewing/agree/unagree）；澄清/修复看 **issue 视角**（每条工单的子状态+计数），非角色状态栏。

## 2. 节点类型实例化（统一词汇，四类）

| 节点类型 | 可配相位（node.state） | reviewState | 例 |
|---|---|---|---|
| **产出类（有内审）** | exec（首/重入）→ review → respond → review… | '-'→reviewing→clarifying→confirming→agreed（门禁：approved，last_decision 承载） | RD 开发（CR 内嵌） |
| **产出类（无内审）** | exec（首/重入） | 恒 '-'（评审在下游审批类节点） | 产品设计 |
| **审批类** | **review（产出动作位，exec 不配置）** | reviewing→clarifying→confirming→agreed（门禁：approved / rejected） | 产品设计评审 / RD 技术评审 / FE 设计评审 |
| **交互类** | entry → exec(to-brd) | 恒 '-' | 需求对齐 |
| **门禁类** | entry（迎宾呈现+会话通道） | 恒 '-' | 用户验收 |

交互/门禁类无 collect/review（空转规则：无挂载相位直过）。

---

## 3. 评审双轨制

| | 节点内评审（CR 异化） | 跨节点评审 |
|---|---|---|
| 评审对象 | **本节点产物**（代码/MR） | **上一个节点的产物** |
| 适用 | **仅开发节点**（编写文档节点没有 review——2026-09-05 用户澄清共识；此前技术设计节点内嵌 CR 补裁作废，见 §8） | 独立评审节点承载（产品设计评审等） |
| 相位 | dev 节点 review 相位（各 CR agent reviewing→上报 agree/unagree） | 评审节点同一套 per-agent 轮转 |
| 不通过 | **回 respond 直接修复（不做澄清）——永不输出 rejected** | agree 判定输出 rejected → 回退上一节点 |
| User 终审 | 无（机械收口） | 按节点 user 拍板配置位 |

---

## 4. 轮转图

### 4.1 dev 节点（内嵌 CR，全生命周期）

```
exec（dev 首次 → implement-techdesign）→ 完成上报+校验
   → review（各 CR agent：reviewing → 上报 agree/unagree）
      → collect 判定：全 agree → 门禁 approved（join）→ QA 验收
      → 有 unagree → respond（fix_on_review 修复，不做澄清）→ CR 重新表态（回 reviewing）→ 循环至全 agree
QA 打回 → plugin 回退 → exec 重入（diagnosing-bugs 响应缺陷）→ 重新提评 → review 重开（reviewing）→ … → 门禁 approved → QA 复验
```

### 4.2 跨节点评审节点（如 RD 技术评审）

```
[上游 exec 完成提评迁移激活]
   → review（arch/fe/qa/pm 审批 skills map；各 agent reviewing → 上报 agree/unagree）
      → collect 判定：全 agree → 门禁 approved（user_gate=1 唤醒 User 终审 → 上游节点完成，流转）
      → 有 unagree → respond（owner clarify 单动作，跨节点路由）→ 重新表态（回 reviewing）→ 循环
      → 确认真问题（to-fix）→ 门禁 rejected → 上游 exec 重入 revise → 重新提评 → review 重开
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

QA 验收节点激活：[accept, backend-dev/frontend-dev.clarify（缺陷确认/归属争议）]；exec 重入=retest（复验重测：重新提测代码重测→核对修复→更新 defect：已解决/新增）。

---

## 6. 推进契约（agent ↔ plugin）

1. **响应者规则**：有挂载动作的相位必有响应者（谁等结果谁负责推进）；executor≠响应者（开场白 executor=program，响应者=拉入的 subagent）。
2. **完成信号双通道**：短动作（表态/澄清/复核）=数据写入即完成（plugin 判据推导）；长任务/交互等待=响应者调标准完成接口（上报即数据写入，plugin 校验判据防谎报，遗漏→扫描器兜底）。与工单 04「数据层即返回」精化衔接。
3. **reject/escalation 结论协议**：branches（单节点留空/并发明确）+ confirmed_issues（从 open issues）+ branch 上下文注入 + escalate 标记；producer 双通道（响应者提交/User 拍板 plugin 自动装配）。
4. **空转规则**：相位无挂载 → plugin 直接轮转（视为通过）。
5. **三重匹配派发**：判据成立+节点声明相位挂载+存在匹配数据；缺一环空转降级。
6. **冲突响应机制**：clarify/confirm 循环不收敛 → escalate 标记 → plugin 拉起按需会话通道裁决 session（响应者=reviewer，开场白=分歧点+双方立场）→ 用户裁决 → **结论追加到冲突 issue**（数据层留痕、node 可见）→ reviewer 重新表态落库 → 正常判据链恢复。
7. **按需会话通道**：plugin 泛用接口（拉起角色会话+开场白模板随绑定配置），前端按节点绑定表达；实例：对齐=PM、拍板呈现=owner、冲突上升=reviewer、用户验收报障=qa——不占相位词汇。
8. **用户门禁与上升通知规程**（2026-09-05 定稿：通用能力——门禁汇总结果与待评审内容报送用户，由责任人发出）：

| 阶段 | 报送责任人 | 通知内容 | 渠道 |
|---|---|---|---|
| 用户拍板（user_gate 节点） | **owner（被评审方）** | 门禁汇总：reviewState 结果 + 各评审位 verdict + 已 resolved 意见摘要 + 待拍板产物链接 + 建议 | 会话通道（owner 迎宾呈现）+ 收件箱通知 |
| 冲突上升（escalate） | **reviewer** | 分歧点 + **正反双方结论汇总**（各一段最终立场）+ issue 线程引用 | 裁决 session（reviewer 迎宾等待） |
| 用户验收（节点 11 类） | **qa**（通道绑定） | 验收材料呈现（test_result/defect）+ 报障收集 | 会话通道（qa 开场白） |

   - 双通道分工：收件箱通知=系统层（plugin 投递，触达保障）；会话迎宾=agent 层（责任人呈现材料，承接交互）——通知由责任人负责，plugin 只做系统投递。
   - **冲突对峙的 DSH 能力结论（2026-09-05 查证）**：DSH 会话与 agent 一对一（「Agent 收件箱是唯一队列」，session 私有），**无多 subagent 共享同一会话的原生机制**——「拉冲突双方同会话对峙」不可直接实现。MVP 替代方案（等效材料对峙）：plugin 分别 followup 双方 subagent 提交最终立场结陈 → 主持人（reviewer）在裁决 session 汇总正反结论呈用户裁决；后期增强可选「顺序质询轮」（双方各 inject 对方结陈反驳一轮，有限轮次）。

---

## 7. 数据层与边界

- **轮转内置逻辑（plugin，不可配）**：collect 全表态收集（review_results 更新/读取）、clarifying 派发、全表态判定、approved/rejected 输出——审批子状态全部 plugin 驱动；可配的只有动作相位上的 action。
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
| 产出类设计节点无内审（v3.1：技术设计 review/respond 跳过） | 用户补裁：技术设计文档也要过 CR——节点 4/6/B2/B4/C2 内嵌 CR 轮（评 rd/fe_design、plan） | §3 双轨（适用范围扩大） |
| R1「跨节点评审 respond=clarify+fix_on_review 双动作」 | 「一个 state 只能挂一个动作」普适约束（用户严正声明）取代——跨节点 respond=clarify 单挂，修复=rejected → 产出节点 exec 重入 revise | §1.1 respond 行 |
| 技术设计节点内嵌 CR 轮（77f9d96 补裁：节点 4/6/B2/B4/C2 review=code-review） | 用户澄清「编写文档节点没有 review」是共识——已回退（gate 回 0、reviewers 清空） | §2 产出类（无内审） |
| plan-cap 跨流程实例化（task C2）/ plan-review-arch / alignment-cap（D2 评审响应）/ alignment-review-pm | 2026-09-07 用户裁决：C2 按场景用 backend/frontend-design（产物=对应侧 design）；D2 暂不评审（用户结论=最终结论，改门禁型）——四者退役；新增 arch.fe-design-review（C3 scope=fe 场景）；能力池 24→**21 活跃**（后续 plan-cap/alignment-cap 再退役，见上行与下行合并口径） | capability-actions-final |

---

## 9. 同步核对记录（2026-09-08 全量通读校验）

- [x] capability-actions-final.md：submit 内联 / review 行 on=review 相位 / respond 轨道规则 / 待建 skill 22 —— 已同步
- [x] business-flow-derivation.md / flow-derivation-tech-task-plan.md：v3.1 逐 state 配置 + scope 渲染矩阵 + 对象化评审 + per-agent 评审轮 —— 已同步
- [x] data-tools-final.md：schema v0.29 + review_results 单字段 + 23 工具契约 —— 已产出
- [x] semantica 溯源（v3 模型定稿 36a226f6 / 工单 07 36a226f6 关联）
