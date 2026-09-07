> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：能力×动作映射表（旧能力名/trigger 词汇）、case_design 引用。

> **追加废弃指针（2026-09-08）**：本文件之外，最新模型已迭代为 v3.1+（两层属性状态机 node.state 4 相位 + **per-agent 评审轮** reviewing/agree/unagree + collect 内置聚合 + review_results 单字段 + 评审能力对象化 {role}.{object}-review + 技术设计无内嵌 CR + task 无 QA 节点 + scope 渲染集合表达式 rd|full/fe|full/full/rd|fe + QA 验收仅 full 渲染）——任何与本文件一致的旧记忆引用，一律以 `docs-tmp/capability-model-final.md` v3.1 现行版为准。
**仍有效**：交互模式结论（对齐开场白、subagent 直接对话）。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。


# 技能使用实现清单（mp-* 37 技能最终处置）

> **状态**：实现期清单（2026-09-02 定稿）。37 个 mp-* 技能全部裁定完毕，按四层归类；含交互模式、qa 职责重定义、开发链路编排——实现期直接照此落位。
> **账本**：`.wayfinder/research/skill-verdict-register.md`（逐条裁定 + 日期 + 理由）；本稿是给实现期的汇总执行件。

---

## 1. 四层归类总览

### ① 直接使用（流程内/研发链路，15 个）

| 技能 | 使用位置 | 引用方式 |
|---|---|---|
| grill-me | 需求对齐阶段（用户可选交互） | user 决定调用：单题盘问 |
| wayfinder | 需求对齐阶段（用户可选交互） | user 决定调用：地图推演 |
| domain-modeling | 需求对齐阶段（用户可选交互） | user 决定调用：术语治理 |
| improve-codebase-architecture | 需求对齐阶段（架构师可选） | 对齐时依次提示本技能、wayfinder 给用户选 |
| prototype | 需求对齐阶段（有前端时） | 一次性原型对齐需求（LOGIC/UI 分支，用完删除或吸收） |
| to-prd（原 to-spec） | **产品阶段产出 skill** | 产出 product_doc 完整版（补足产品内容）——**to-xxx 家族形态模板基础**（to-techdesign 等按 Process+Template 形态复制）；原生 spec 概念废弃 |
| to-techdesign | 技术设计阶段产出 skill | to-rd-techdesign / to-fe-techdesign（对应 rd_design / fe_design），与 to-prd 对齐 |
| to-tickets | 开发链路前置 | **techdesign 产出（rd/fe_design）** → 垂直切片工单（Blocked by+验收清单），结合 /goal 跨会话追踪 |
| implement-techdesign（原 implement-spec） | 开发编排（RD/FE 调用） | 基于 **techdesign 文档** 编排任务图（模块→tdd/implement→依赖序）→ 按并发度派孙 subagent（MVP 串行，并发度可配） |
| tdd | 开发执行（核心逻辑） | 开发动作 how 执行步 |
| implement | 开发执行（UI/CRUD/胶水） | 开发动作 how 执行步 |
| code-review | 评审 | 单动作串行两子步（Standards→Spec） |
| retro | CR 复盘 | 评审收尾改进建议来源 |
| diagnosing-bugs | 缺陷响应（respond_to_qa） | QA 打回缺陷→最小化复现→一次一变量定位→补回归 |
| triage | qa 缺陷管理 | 缺陷状态机（待确认/补信息/归域/人工/忽略）+ 查重 + 复现确认 |
| research | 调研员（research-cap） | 一手源调研、逐主张溯源、带引用落盘 |
| resolving-merge-conflicts | rd/fe preset 下技能 | 冲突解决场景由 rd/fe subagent 调用 |
| migrate-to-shoehorn | 前端提交前校验 | 测试 as 断言迁移（钩子内或提交流程） |

### ② 底层依赖（不直接使用，4 个）

| 技能 | 定位 |
|---|---|
| grilling | grill-me/wayfinder 的底层原语（frontier 轮次制等） |
| wait-wait | 会话侧通用沟通协议（失准重述） |
| codebase-design | 词汇底层（tdd/implement/code-review 间接引用；接口面原则已进 backend-design §3.5） |
| loop-me | 词汇底层（trigger/checkpoint/push right/brief 被框架间接使用） |

### ③ 初始化脚手架（与 flow 无关，2 个）

| 技能 | 定位 |
|---|---|
| setup-ts-deep-modules | 新项目初始化（deep module 边界基线） |
| setup-pre-commit | 新仓库初始化钩子（lint-staged/typecheck/test 由钩子自动触发，**开发环节不调用**） |

### ④ 废弃/忽略/不启用（16 个）

- **废弃**：teach、to-questionnaire、grill-with-docs、writing-beats、writing-fragments、writing-shape、scaffold-exercises
- **不启用**：git-guardrails-claude-code（DSH 沙箱/审批已覆盖）、handoff、claude-handoff（思想由 my_task 最小返回集承载）
- **暂忽略**：wizard（无发布阶段）、setup-matt-pocock-skills（与自建 tracker 不匹配）
- **仅入口**：ask-matt（帮助入口，不挂 any how）

### 辅助定位

- **writing-for-agents**：推演期工具——写更好用的 how 时使用（v2 纪律来源），不常驻注册
- **to-prd** = to-spec 旧名（忽略）

---

## 2. 交互模式（需求对齐阶段）

```
subagent 进入需求对齐 → 主动开场（直接与 user 对话，经 subagent 会话通道，D6 节点挂载可见）：
  ① 说明：哪个需求（reqId/名称）+ 什么阶段 + 要和用户确认什么
  ② 提示：可选交互类 skills（grill-me / wayfinder / domain-modeling / 一句话直接阐述）
  ③ 用户选择后按对应协议推进
```

---

## 3. qa 职责重定义（主推演 merge 时落 flow-alignment / capability-review-v2）

1. **用例设计归开发侧自产**：rd 产后端口径、fe 产前端口径（case_design 不再由 qa 产出；tdd 保持开发侧，无需拆编写给 qa）
2. **qa 评审节点保留**：专为 qa 完善领域模型（评审维度/缺陷判定准则 → qa 专用 skill），减少 qa 上下文
3. **qa 聚焦收拢**：联调约束 + 验收 + 缺陷判定（respond_to_qa 维持；triage 归 qa 缺陷管理）

---

## 4. 开发链路编排（代码类执行定稿，2026-09-02 二版）

```
需求对齐（交互：用户可选 grill-me/wayfinder/domain-modeling/一句话）
  → to-prd（产品阶段：产出完整 product_doc，补足产品内容；to-xxx 家族形态模板）
  → to-techdesign（技术设计：to-rd-techdesign / to-fe-techdesign 产出 rd_design / fe_design，与 to-prd 对齐）
  → [to-tickets] 基于 rd/fe_design 拆工单 + /goal 挂载目标（active→…→complete，跨会话追踪）
  → [implement-techdesign] 基于 rd/fe_design 编排（任务图：模块→tdd/implement→依赖序；并发度=N 并行 / =1 串行）
  → tdd / implement 工人执行模块（孙 subagent，并发度受控）
  → 组装 → 自测 → 登记 →（提交由 setup-pre-commit 钩子自动触发 lint/typecheck/test）
```

- **文档链**：to-prd（产品）→ to-techdesign（技术）→ 开发编排输入 = techdesign 产出；**原生 spec 概念废弃**
- 执行方 = RD/FE（开发侧 subagent，可再派孙）；how 执行要素 = 编排序列形态 `execution: [to-tickets, implement-techdesign]`（受控枚举+可扩展）
- 并发度：MVP 默认串行；DSH 锚点 maxConcurrentJobsPerOwner

---

## 5. 关联产物

- **v2 骨架**（工单 05 第 1 条定稿）：四要素（目标/依赖[动态看 mytask]/执行[skill/编排]/自检）+ 工具知识独立沉淀 skill
- **产出 skill 家族**：to-spec 为模板基础，to-prd→product-design、to-techdesign→backend-design 等（D1=B 模板外引用、D2=每文档独立）
- **skill 落位**：直接使用技能挂 preset（full/skills，按 preset 聚合规则）；to-xxx 家族实现期逐个创建
- 实现期需定：execution 编排序列的受控枚举清单、并发度配置项、qa 领域模型 skill 内容