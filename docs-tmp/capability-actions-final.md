# MVP 能力动作终稿（18 能力 · skill 挂载制 · 可回填 seed.json 形态）

> **状态**：2026-09-05 v2.1 定稿（skill 挂载制）。**动作不携带 how 四要素**（用户裁决：how-spec-v2 降级为 skill 写作参考，框架不强制）——复杂动作直接挂一个 skill（内部按需编排其他 skill、自带目标/自检/门禁）；短协议性动作不挂 skill，一句话内联定义。
> 模型权威见 `capability-model-final.md` v2.1（trigger 8 词 / hook 层 / 轮次制循环 / 推进契约）；逐节点实例化见 `business-flow-derivation.md` v2。business 已定稿；task/plan 标注处随举一反三回填。

---

## 0. 动作 schema（收敛）

```
action = { action_id, name, on, skill?, inline?, outputProductTypes }
capability = { capability_id, role_id, inputs[], actions[] }
```

- **on**：挂载点受控枚举 = trigger 8 词 ∪ {after-exec, after-re-exec}；首次/重提可同动作承接（on 数组）。
- **inputs[]**：能力级数据可见性契约（ProductType + use_threads 默认 true）——保留，不受本次精简影响。
- **上下文注入**：my_task 返回 {action_id, skill, nodeId, reqId, inputs 引用}——原 how「依赖」要素的职责由 mytask 承担，skill 内部自行取用。
- **完成判定**：skill 内部自检（agent 侧）+ 推进契约数据判据（plugin 侧校验，见模型 §0.7）。

## 1. 动作 → skill 挂载表

| 动作（on） | skill | 状态 | 短动作一句话内联 |
|---|---|---|---|
| produce ×6（exec） | to-prd / to-rd-techdesign / to-fe-techdesign / research / to-arch / to-plan | 5 待建，research 已有 | — |
| 对齐 exec（交互节点 entry-hold 后：business/tech 需求对齐、plan D1） | **to-brd / to-alignment ——公共 skill**（node action 挂载，不属角色能力；business/tech/plan 复用） | 2 待建 | — |
| dev（exec） | **implement-techdesign**（内部编排 to-techdesign 校验 + tdd/implement 工人；并发度可配） | 待建 | — |
| submit（after-exec ∪ after-re-exec） | 不挂 | — | 「产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成」 |
| clarify（reviewed） | 不挂 | — | 「读 assignee=本角色 且 open 意见/缺陷（CR 意见、qa 缺陷确认与归属争议同型，来源无关）→ 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve」；**由节点激活清单声明服务哪些节点的 reviewed**（backend-dev.clarify 同时服务开发节点与 QA 验收节点） |
| respond_to_cr（reviewed） | ——**已并入 clarify，退役**（2026-09-05：CR 意见=clarify 的实例，节点激活清单区分服务场景） | — | — |
| respond_to_cr（reviewed） | 不挂 | — | 「读 reviewerRole=Code Reviewer 且 assignee=本角色 且 open 意见 → 逐条定位改动点回应（改代码或澄清）→ resolve」 |
| confirm（replied） | 不挂 | — | 「读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)」 |
| fix_on_review（**replied**） | 不挂 | — | **只响应 CR**——评审轮内的意见修复响应（CR 复核 reopen 驱动）：读 CR 意见 → 逐条修订/修复 → resolve → 重新进入复核；轮内闭环，不打回、不重入 |
| diagnosing-bugs（**exec 重入**） | **diagnosing-bugs**（已有 mp） | 已有 | **打回返工流程**——agree 判 reject 后再次进入触发：基于评审结论（qa 缺陷/cr 真问题）复现→一次一变量定位→修在正确接缝→补回归 → after-exec 重新提评 |
| review ×6（after-exec ∪ after-re-exec） | **review-protocol**（共享评审协议：内含 pm/arch/rd/fe/qa/cr 六角色维度表——qa 领域模型并入） | 待建 | — |
| accept（exec；重入按重开来源分派） | **qa-acceptance**（内含 triage + diagnosing-bugs 判定步） | 待建 | — |
| create_reqs（exec@plan D7） | **to-tickets**（triage 纪律内部调用） | 已有 | — |
| decide（approve） | ——程序契约（wb_decide），无 skill | — | — |

## 2. 逐能力终稿（18）

| 能力 | role | inputs（只列 use_threads=true） | actions（on → skill/inline） | 产物 |
|---|---|---|---|---|
| product-design | pm | brd, architecture(false) | produce(exec→to-prd) / submit / clarify / fix_on_review | product_doc |
| backend-design | rd | product_doc, architecture(false) | produce(exec→to-rd-techdesign) / submit / clarify / fix_on_review | rd_design |
| frontend-design | fe | product_doc, rd_design, architecture(false) | produce(exec→to-fe-techdesign) / submit / clarify / fix_on_review | fe_design |
| research-cap | inv | alignment | produce(exec→research) / submit / clarify / fix_on_review | research |
| arch-cap | arch | research, alignment(false) | produce(exec→to-arch) / submit / clarify / fix_on_review | architecture |
| plan-cap | leader | ——**基础=task C1 对齐简单描述**（✅D2：无 prd/brd，mytask 上下文注入） | produce(exec→to-plan) / submit / clarify / fix_on_review | plan |
| alignment-cap | arch | alignment（D2 评审对象，true） | **评审响应能力（✅D3 连带收缩）**：clarify(reviewed) / fix_on_review(re-exec)——produce 由 node action to-alignment（公共 skill）承接 | alignment |
| backend-dev | rd | rd_design, architecture(false), product_doc(false), test_result | dev(exec→implement-techdesign) / submit(inline) / **fix_on_review(respond，只响应 CR——不做澄清)** / **diagnosing-bugs(exec 重入，响应打回)** / **clarify(仅 QA 验收节点激活：缺陷确认/归属争议)** | dev_notes, code_mr |
| frontend-dev | fe | fe_design, rd_design(false), architecture(false), product_doc(false), test_result | 同上（fe 侧） | dev_notes, code_mr |
| pm-review | pm | product_doc, brd(false), rd_design(false) | review(after-exec∪after-re-exec→review-protocol) / confirm(inline) | — |
| arch-tech-review | arch | rd_design, product_doc(false), architecture, research(false) | review / confirm | — |
| rd-tech-review | rd | product_doc, rd_design, architecture(false) | review（自评+评审）/ confirm | — |
| fe-tech-review | fe | product_doc, rd_design(false) | review / confirm | — |
| qa-tech-review | qa | rd_design, fe_design, product_doc | review / confirm | — |
| code-review | cr | rd_design/fe_design(按分支), dev_notes | review（review-protocol 内 Standards→Spec 两子步；代码经 worktree readMode）/ confirm | — |
| acceptance-test | qa | product_doc, rd_design(false), fe_design(false), dev_notes(false), code_mr(false) | accept(exec→qa-acceptance；re-entry 来源分派：rework-return→重跑复验 / downstream-reject→re-exec 补充缺陷) | test_result, defect |
| decide-cap | leader | ——（decide 走 query：表态汇总+未解决评论；create_reqs 走 alignment） | create_reqs(exec@plan D7→to-tickets) / decide(approve——程序契约) | — |
| final-approve | user | test_result, defect, product_doc(false)（UI 呈现） | ——（approve 触发的 UI 拍板；wb_decide 唯一契约） | — |

> **已移除**：case-design-cap（19→18）；retest；respond_to_qa；align.closing；spec 命名；**how 四要素**。
> **review 无 exec 注记**：六个评审能力均为纯事件响应（review@after-exec、confirm@replied），由被评审工作的 exec 完成事件引发，跨节点按 assignee+reporterRole 路由。**issue 即产物**：评审类动作产出=issue（plugin 内一等数据），outputProductTypes 为空是正确语义。
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
| 7 | review-protocol | 六角色维度表（pm/arch/rd/fe/qa/cr）——qa 领域模型并入（可测性/联调/安全性/缺陷判定准则） |
| 8 | qa-acceptance | qa-acceptance-template + triage + diagnosing-bugs 判定步 |

已有复用：research / tdd / implement / diagnosing-bugs / to-tickets / triage。wb 工具用法不再独立成 skill（短动作内联 + plugin 接口文档）。
**skill 实现校验（✅D5）**：每个 skill 必须写明上下文依赖来源（跨流程动态注入：business/tech/task/plan 各自可用的 inputs），实现阶段逐个校验防止遗漏。

## 4. seed.json 迁移注记（详表→工单 07）

- 现状：capabilities 仅 how 字符串数组、flow_template_nodes 无顶层 id。
- 目标：按本文件 `actions[{action_id, name, on, skill?|inline?}]` + `inputs[]` 回填；on 枚举校验；skill 注册表（10 待建 + 6 复用）；flow_nodes 补 template_node_id。
