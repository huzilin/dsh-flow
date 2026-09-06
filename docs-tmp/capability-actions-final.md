# MVP 能力动作终稿（18 能力 · skill 挂载制 · 可回填 seed.json 形态）

> **状态**：2026-09-05 v2.1 定稿（skill 挂载制）。**动作不携带 how 四要素**（用户裁决：how-spec-v2 降级为 skill 写作参考，框架不强制）——复杂动作直接挂一个 skill（内部按需编排其他 skill、自带目标/自检/门禁）；短协议性动作不挂 skill，一句话内联定义。
> 模型权威见 `capability-model-final.md` v2.1（trigger 8 词 / hook 层 / 轮次制循环 / 推进契约）；逐节点实例化见 `business-flow-derivation.md` v2。business 已定稿；task/plan 标注处随举一反三回填。

---

## 0. 动作 schema（收敛）

```
action = { action_id, name, on, skill?, inline?, outputProductTypes }
capability = { capability_id, role_id, inputs[], actions[] }
```

- **on**：挂载点受控枚举 = 可配相位 4 值（entry / exec / review / respond）；exec 支持首/重入双配置（按 rolled_back_count 选）。
- **inputs[]**：能力级数据可见性契约（ProductType + use_threads 默认 true）——保留，不受本次精简影响。
- **上下文注入**：my_task 返回 {action_id, skill, nodeId, reqId, inputs 引用}——原 how「依赖」要素的职责由 mytask 承担，skill 内部自行取用。
- **完成判定**：skill 内部自检（agent 侧）+ 推进契约数据判据（plugin 侧校验，见模型 §0.7）。

## 1. 动作 → skill 挂载表

| 动作（on） | skill | 状态 | 短动作一句话内联 |
|---|---|---|---|
| produce ×6（exec） | to-prd / to-rd-techdesign / to-fe-techdesign / research / to-arch / to-plan | 5 待建，research 已有 | — |
| 对齐 exec（交互节点 entry-hold 后：business/tech 需求对齐、plan D1） | **to-brd / to-alignment ——公共 skill**（node action 挂载，不属角色能力；business/tech/plan 复用） | 2 待建 | — |
| dev（exec） | **implement-techdesign**（内部编排 to-techdesign 校验 + tdd/implement 工人；并发度可配） | 待建 | — |
| submit（exec 闭环内联） | 不挂——登记=exec 完成判据的一部分（wb_add_artifact 落库），提评=plugin 判据校验通过后轮转 review 相位 | — | 「产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成」 |
| clarify（reviewed） | 不挂 | — | 「读 assignee=本角色 且 open 意见/缺陷（CR 意见、qa 缺陷确认与归属争议同型，来源无关）→ 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve」；**由节点激活清单声明服务哪些节点的 reviewed**（backend-dev.clarify 同时服务开发节点与 QA 验收节点） |
| respond_to_cr（reviewed） | ——**已并入 clarify，退役**（2026-09-05：CR 意见=clarify 的实例，节点激活清单区分服务场景） | — | — |
| respond_to_cr（reviewed） | 不挂 | — | 「读 reviewerRole=Code Reviewer 且 assignee=本角色 且 open 意见 → 逐条定位改动点回应（改代码或澄清）→ resolve」 |
| confirm（replied） | 不挂 | — | 「读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)」 |
| fix_on_review（**respond**） | 不挂 | — | **节点内评审（CR 异化）轨道专用、只修复不澄清**——激活节点：开发节点 + 技术设计节点内嵌 CR 轮（4/6/8/9/B2/B4/C2）。评审轮内的意见修复响应（CR 复核 reopen 驱动）：读 CR 意见 → 逐条修复 → resolve → 重新进入复核；轮内闭环，不打回、不重入。**普适约束：一个 state 只能挂一个动作**（2026-09-05 用户严正声明，取代 R1 双动作——undermined）——跨节点评审 respond=clarify 单挂（修复=rejected → 产出节点 exec 重入 revise/diagnosing-bugs） |
| diagnosing-bugs（**exec 重入**） | **diagnosing-bugs**（已有 mp） | 已有 | **打回返工流程，仅 dev/fe 节点 exec 重入**（代码域）——agree 判 reject 后再次进入触发：基于评审结论（qa 缺陷/cr 真问题）复现→一次一变量定位→修在正确接缝→补回归 → 重新提评（plugin 轮转） |
| revise（**exec 重入**） | 不挂——inline「基于评审结论逐条修订〈产物〉对应章节 → 模板自检 → 重新提评」 | — | **文档产出节点 exec 重入**（文档域）——响应评审打回的修订返工；与 diagnosing-bugs 按域分立（代码=diagnosing-bugs，文档=revise） |
| review（各评审能力·对象化） | **{role}.{object}-review 各自独立 skill**（2026-09-05 两步裁决：①否决全局 review-protocol——解耦/维护/ctx；②评审能力对象化——最小上下文，inputs 仅含评审对象、skill 仅含该对象维度） | 待建×12 | — |
| accept（exec；重入按重开来源分派） | **qa-acceptance**（内含 triage + diagnosing-bugs 判定步） | 待建 | — |
| create_reqs（exec@plan D7） | **to-tickets**（triage 纪律内部调用） | 已有 | — |
| decide（approve） | ——程序契约（wb_decide），无 skill | — | — |

## 2. 逐能力终稿（18）

| 能力 | role | inputs（只列 use_threads=true） | actions（on → skill/inline） | 产物 |
|---|---|---|---|---|
| product-design | pm | brd, architecture(false) | produce(exec→to-prd) / **revise(exec 重入)** / clarify(respond) / submit | product_doc |
| backend-design | rd | product_doc, architecture(false) | produce(exec→to-rd-techdesign) / **revise(exec 重入)** / clarify(respond) / submit | rd_design |
| frontend-design | fe | product_doc, rd_design, architecture(false) | produce(exec→to-fe-techdesign) / **revise(exec 重入)** / clarify(respond) / submit | fe_design |
| research-cap | inv | alignment | produce(exec→research) / **revise(exec 重入)** / clarify(respond) / submit | research |
| arch-cap | arch | research, alignment(false) | produce(exec→to-arch) / **revise(exec 重入)** / clarify(respond) / submit | architecture |
| plan-cap | leader | ——**基础=task C1 对齐简单描述**（✅D2：无 prd/brd，mytask 上下文注入） | produce(exec→to-plan) / **revise(exec 重入)** / clarify(respond) / submit | plan |
| alignment-cap | arch | alignment（D2 评审对象，true） | **评审响应能力（✅D3 连带收缩）**：clarify(respond)——produce 由 node action to-alignment（公共 skill）承接 | alignment |
| backend-dev | rd | rd_design, architecture(false), product_doc(false), test_result | dev(exec→implement-techdesign) / submit(inline) / **fix_on_review(respond，只响应 CR——不做澄清)** / **diagnosing-bugs(exec 重入，响应打回)** / **clarify(仅 QA 验收节点激活：缺陷确认/归属争议)** | dev_notes, code_mr |
| frontend-dev | fe | fe_design, rd_design(false), architecture(false), product_doc(false), test_result | 同上（fe 侧） | dev_notes, code_mr |
| arch.prd-review | arch | product_doc | review(prd-review-arch-skill) / confirm | — |
| rd.prd-review | rd | product_doc | review(prd-review-rd-skill) / confirm | — |
| fe.prd-review | fe | product_doc | review(prd-review-fe-skill) / confirm | — |
| arch.rd-design-review | arch | rd_design, architecture(false), research(false) | review / confirm | — |
| fe.rd-design-review | fe | rd_design, product_doc(false) | review / confirm | — |
| qa.rd-design-review | qa | rd_design, product_doc | review / confirm | — |
| pm.rd-design-review | pm | rd_design, product_doc, brd(false) | review / confirm | — |
| qa.fe-design-review | qa | fe_design, rd_design(false), product_doc | review / confirm | — |
| arch.plan-review | arch | plan | review / confirm | — |
| pm.alignment-review | pm | alignment | review / confirm | — |
| arch.research-review | arch | research, alignment(false) | review / confirm | — |
| pm.architecture-review | pm | architecture | review / confirm | — |
| code-review | cr | 技术设计轮=rd_design/fe_design/plan；开发轮=dev_notes, code_mr（worktree readMode） | review(review→code-review-skill：Standards→Spec 两子步)/ confirm——作用节点：技术设计节点（4/6/B2/B4/C2，评设计文档）+ 开发节点（8/9/B8/C4，评代码） | — |
| acceptance-test | qa | product_doc, rd_design(false), fe_design(false), dev_notes(false), code_mr(false) | accept(exec→qa-acceptance；re-entry 来源分派：rework-return→重跑复验 / downstream-reject→re-exec 补充缺陷) | test_result, defect |
| decide-cap | leader | ——（decide **退役**：join 裁决=plugin 内置轮转吸收；**create_reqs=leader 直执行 ✓**——不起 subagent） | create_reqs(exec@plan D7→to-tickets) | — |
| final-approve | ——**退役**（2026-09-05：User 终审职责由 flow_def_nodes.user_gate 配置位承载） | —— | —— | — |

> **已移除**：case-design-cap（19→18）；retest；respond_to_qa；align.closing；spec 命名；**how 四要素**。
> **v3 词汇注记**：评审动作挂 review 相位（collecting 收集表态）；复核动作挂 respond 响应期的确认环节（confirming）；跨节点按 assignee+reporterRole 路由。**issue 即产物**：评审类动作产出=issue（plugin 内一等数据），outputProductTypes 为空是正确语义。
> **跨流程复用（✅D5）**：inputs[]=可见性上限，实际按流程可用产物注入 my_task（backend-design 在 business 拿 product_doc、tech 拿 brd）——**每个 skill 必须写明上下文依赖来源，skill 实现阶段逐个校验，防止遗漏**。三流程实例化见 flow-derivation-tech-task-plan.md。

## 3. 待建 skill 清单（10）

| # | skill | 参照/内容 |
|---|---|---|
| 1 | to-brd（公共 skill +brd-template） | 对齐节点 node action 挂载（business/tech 复用）；产物 brd（需求级路径） |
| 2 | to-prd | mp-to-spec 形态 + prd-template |
| 3 | to-rd-techdesign / to-fe-techdesign | 同形态 + backend/frontend-design-template（接口面原则进门禁） |
| 4 | to-arch / to-plan | 同形态（to-plan 基础=task C1 简单描述） |
| 5 | to-alignment（公共 skill） | plan D1 node action 挂载；汇总对齐结论+对话中发现的改进项/遗留 TODO/使用发现 |
| 6 | implement-techdesign | mp-implement-spec 改造（内部编排 to-techdesign 校验 + tdd/implement 工人） |
| 7 | 对象化评审 skill ×12 | **{role}.{object}-review 各自独立**（维度自含，最小 ctx）：prd-review-{arch,rd,fe}（需求覆盖/可实现性/交互完整性）、rd-design-review-{arch,fe,qa,pm}（技术路线/契约可对接/可测性安全性/需求覆盖）、fe-design-review-qa、plan-review-arch、alignment-review-pm、research-review-arch、architecture-review-pm |
| 8 | qa-acceptance | qa-acceptance-template + triage + diagnosing-bugs 判定步 |

已有复用：research / tdd / implement / diagnosing-bugs / to-tickets / triage。wb 工具用法不再独立成 skill（短动作内联 + plugin 接口文档）。
**skill 实现校验（✅D5）**：每个 skill 必须写明上下文依赖来源（跨流程动态注入：business/tech/task/plan 各自可用的 inputs），实现阶段逐个校验防止遗漏。

## 4. seed.json 迁移注记（详表→工单 07）

- 现状：capabilities 仅 how 字符串数组、flow_template_nodes 无顶层 id。
- 目标：按本文件 `actions[{action_id, name, on, skill?|inline?}]` + `inputs[]` 回填；on 枚举校验；skill 注册表（10 待建 + 6 复用）；flow_nodes 补 template_node_id。
