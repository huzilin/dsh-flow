> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：判定 1（respond_to_qa 补建——已撤销，缺陷响应由轨道规则承载：跨节点=clarify、节点内=fix_on_review、打回返工=revise/diagnosing-bugs）；判定 2 表述（dev+self_test 合并——现为 implement-techdesign 编排内含自测）；writing 三件套挂需求对齐（已消解——对齐节点无专属能力，node action to-brd）；code-review 挂载表述（评审能力对象化后为 code-review-skill）。

> **追加废弃指针（2026-09-08）**：本文件之外，最新模型已迭代为 v3.1+（两层属性状态机 node.state 4 相位 + **per-agent 评审轮** reviewing/agree/unagree + collect 内置聚合 + review_results 单字段 + 评审能力对象化 {role}.{object}-review + 技术设计无内嵌 CR + task 无 QA 节点 + scope 渲染集合表达式 rd|full/fe|full/full/rd|fe + QA 验收仅 full 渲染）——任何与本文件一致的旧记忆引用，一律以 `docs-tmp/capability-model-final.md` v3.1 现行版为准。
**仍有效**：接口面原则 4 条进技术设计门禁、to-tickets/triage 挂 create_reqs、维度判定方法论。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。


# 技能包评估稿：mattpocock skills × 团队项目工作台

> **状态**：评估稿 v1（2026-08-27 定稿）。本稿为 wayfinder 地图「技能包评估地图」票 06（技能→能力×动作 how 参考映射采纳判定）的 Resolution 产物。
> **一句话**：37 个技能逐一裁决了与 19 能力×动作的 how 挂载关系（角度 B：不问"流程性帮助"，问"每个技能挂到哪些能力×动作的 how 上作参考"）；其中 4 项裁决直接暴露/修正了流程模型（respond_to_qa 补建、dev/self_test 合并、模板变更、受控术语新增）。
> **引用规约**：模板/章节一律用章节名（骨架标注为准），不用章节编号。
> **待回填**：respond_to_qa 动作、dev 合并动作为流程缺口/重构建议，移交 docs-tmp 主推演（capability-review-v2 / execution-loop）落地。

---

## 1. 结论先行

- **挂载结构**：37 技能 = 进动作 how 13 + 进 how 编写规范 v2 5 + 不进动作 how 19（流程层 8 / 工具层 9 / 无关 2）；全部裁决明细见 §3。
- **流程缺口（必须回主推演）**：① 缺 `respond_to_qa` 动作（QA 缺陷返工从未显式动作化）；② `dev` + `self_test` 合并（trigger 链简化）。
- **模板变更（已落盘）**：4 个模板新增「动机场景（用户故事）」节 + 骨架标注同步；Out of Scope 进三设计阶段文档。
- **受控术语新增**：deep module / 接口面 / seam（设计原则词汇，非 ProductType 枚举）。

---

## 2. 判定维度与依据

| 项 | 内容 |
|---|---|
| 判定维度 | **角度 B**（用户 2026-08-27 修正）：每个技能应挂到哪些能力×动作的 how 上作参考——19 能力×动作的 how 编写时吸收该技能的哪个纪律/机制。非旧维度（当前推演期/插件实现期，子代理曾跑偏，已纠） |
| 评估范围 | 全量 37 个（用户 Q6=B：推演的流程包括开发/CR 等，编码向技能也要） |
| how 五段式基线 | 读产物[ProductType] → 按模板[路径+章节名] → 自检[quality_gates] → 登记/提评/响应[wb_* 工具] → 输出[productType]；trigger 程序契约；对话类产物用 query 属性过滤 |
| 一手源 | 三份技能盘点（.wayfinder/research/{engineering,productivity,misc-inprogress}-inventory.md）、capability-review-v2.md（能力×动作权威）、how-writing-comparison.md（11 纪律对照 + v2 草案 16 条）、flow-alignment.md（4 流程节点权威） |

---

## 3. 逐技能裁决（37）

### 3.1 进动作 how（13）

| 技能 | 挂载点（能力×动作） | 吸收内容（一句话） |
|---|---|---|
| `research` | research-cap | 整体吸收调研纪律：一手源、逐主张溯源、带引用落盘 |
| `to-tickets` | decide-cap.create_reqs | 垂直切片粒度判据（该节点产物能独立验收吗）、blocking 先于执行、work the frontier |
| `triage` | decide-cap.create_reqs | redundancy 查重（按领域概念搜，非字面）、verify claim；状态机增强留流程层 |
| `to-spec` | Out of Scope → arch-cap / product-design / backend-design | 显式 Out of Scope 节写进三设计阶段文档 |
| `domain-modeling` | arch-cap / product-design / backend-design + 全局受控枚举治理 | 设计类环节的术语消歧自检（新领域词汇产生地）；ProductType/trigger 涨词按场景压力测试 |
| `code-review` | code-review.review | **单动作串行两子步**：Standards 轴（对照表+规约基线）→ Spec 轴（设计契约对需求/AC）→ 聚合表态 |
| `tdd` | backend-dev.dev / frontend-dev.dev | **合并动作后**三子步（实现/自测/记录）纪律：垂直切片、公共接口、红绿循环 |
| `diagnosing-bugs` | **backend-dev.respond_to_qa / frontend-dev.respond_to_qa（新动作）** + acceptance-test 缺陷判定步 | 最小化复现 → 一次一变量定位 → 修在正确接缝 → 补回归；QA 侧复现最小化后再下结论 |
| `codebase-design` | backend-design §3.5 接口设计 | 接口面原则 4 条（见 §4.3） |
| `implement` | backend-dev.dev（弱） | 动作序列五段式模板样例 |
| `writing-shape` | alignment-cap.produce（成型协议） | 开头承诺守约 / 格式论证 / gap-naming |
| `writing-fragments` | align.discuss（访谈素材收集） | 碎片静默沉淀、leading word、每次写前重读 |
| `writing-beats` | align.discuss（概念级访谈控制） | grounding 纪律、每 beat 需什么+ground 什么 |

### 3.2 进 how 编写规范 v2（5）

| 技能 | 吸收内容 |
|---|---|
| `writing-for-agents` | **核心**：11 纪律 × 五段式对照 → v2 条目清单草案 16 条（how-writing-comparison.md），供票 05 逐条拍板 |
| `handoff` / `claude-handoff`（合并） | 交接只引用产物路径不重复内容、必含 suggested skills、脱敏 |
| `loop-me` | brief（决策就绪摘要）、push right（检查点后移）、trigger（event 优先）、「无疑问才 done」 |
| `retro` | 七类环境改进候选 = 动作推演审查维度表 |

### 3.3 不进动作 how（19）

- **流程层（8）**：grill-with-docs、grill-me、grilling、to-questionnaire、wait-what、wayfinder、prototype、implement-spec（frontier 思想）——推演会话/台账/方法论协议，与 19 能力动作无挂载关系（「当前推演期采纳 ≠ 进动作 how」）
- **工具层（9）**：wizard、git-guardrails-claude-code、migrate-to-shoehorn、setup-pre-commit、setup-matt-pocock-skills、setup-ts-deep-modules、improve-codebase-architecture、resolving-merge-conflicts、ask-matt——插件实现期/环境基建参考
- **无关（2）**：teach、scaffold-exercises

---

## 4. 流程/模板/术语变更清单（移交主推演）

### 4.1 流程缺口：新建 respond_to_qa 动作（用户 2026-08-27 批注）

- **缺口**：现有 actions 只有 `respond_to_cr`（query: reviewerRole=CR）；execution-loop 决策树②「respond_to_cr（**或 QA 返工闭环**）」的 QA 返工从未显式动作化。
- **补建**（与 respond_to_cr 对称）：
```jsonc
{
  "action_id": "backend-dev.respond_to_qa",   // frontend-dev.respond_to_qa 同构
  "name": "响应QA缺陷",
  "trigger": "reviewer_feedback(open)",
  "when_to_use": "QA 验收打回的缺陷条目中 assignee=rd（我）且 state=open 时",
  "how": "读《defect》(docs/{{req-version}}/{{req-name}}/qa/defect.md) 归属到我的条目 → 按「复现」重建并最小化（逐元素 load-bearing）→ 一次一变量定位根因（禁止凭描述猜）→ 修在正确接缝（无接缝=架构发现，记入《dev_notes》对照表）→ 补回归验证 → wb_resolve(note=修复说明) → QA 复核 confirm/reopen",
  "reference": [{ "query": { "reviewerRole": "QA", "assignee": "rd", "state": "open" } }],
  "focus": ["defect.缺陷详情（复现/预期/实际）", "code_mr.对应行"],
  "output": "修订代码 / resolve",
  "tools": ["wb_my_task", "wb_reply", "wb_resolve"]
}
```
- **同步点**：capability-review-v2 §0.2 actions 数组、execution-loop 决策树②、acceptance-test 缺陷返工闭环勾连。

### 4.2 动作重构：dev + self_test 合并（用户 2026-08-27 裁决）

- **合并**：`backend-dev.dev` 单动作内含三子步——① 实现（垂直切片一次一测试一实现、测试经公共接口）；② 自测（按自测方案+case_design 口径，先红后绿、回归防回退）；③ 记录（实现-设计对照表，含偏差与风险说明）；完成标准 = 自测全绿 + 对照表逐行成立。
- **trigger 链**：node_enter → dev（含自测）→ dev_complete → submit（**唯一动作**）——v2 草案第 9 条「self_test/submit 同 trigger」冲突随合并**消解**；self_test 独立 trigger 消失。
- **同步点**：capability-review-v2 §0.2 backend-dev 范例 + §2.1、frontend-dev 同构、execution-loop 决策树⑤。

### 4.3 模板变更（已落盘 docs/templates/）

| 模板 | 新增节 | 骨架标注 |
|---|---|---|
| prd-template.md | §4.4 动机场景（用户故事） | ✅ 补条目（必填） |
| arch-version-design-template.md | §2.3 版本动机（用户故事） | ✅ 补条目（必填） |
| backend-design-doc-template.md | §1.1 动机场景（用户故事） | ✅ 补条目（必填） |
| frontend-design-doc-template.md | §1.1 动机场景（用户故事） | ✅ 补条目（必填） |

- User Stories：3–5 条轻量形态（`As an <角色>, I want <功能/行为>, so that <收益>`），给读者补动机 context；**Out of Scope**：三设计阶段（架构规划/product_doc/rd_design）显式写出"被想过的、决定不做的事"。
- **待核对**：gen_flow_rows.py 对模板章节的引用需同步（脚本一致性）。

### 4.4 受控术语新增（设计原则词汇，非 ProductType 枚举）

- **接口面原则 4 条**（backend-design §3.5 说明块 + 质量门禁 major 核对行）：①小接口大实现（deletion test：删掉损失可数）；②依赖传入不创建；③返回值无副作用；④seam 放行为边界（one/two adapter）。
- 词条：`deep module` / `接口面` / `seam`（中文为主、英文括注；按 ProductType 同款治理禁止自造同义词）。

---

## 5. 与既有文档的关系

- 三层文档权威顺序（flow-alignment.md）：docs-tmp（本评估稿层，正在敲定）→ arch_design.md（待 merge 基线）→ seed.json（实现，仅参考）。
- 本质上是**推演输入**而非推演本身：为 capability-review-v2（能力×动作 how 推演）提供"哪些技能纪律该吸收、挂在哪"的裁决表；为 execution-loop 提供 respond_to_qa 缺口与 trigger 链简化建议。
- 关键决策已 record_decision 到 semantica（技能包评估、维度修正、合并、缺口等）。