> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：case_design 相关推导（qa 职责重定义后废弃）、how 挂载表述（skill 挂载制后 how 退役）。
**仍有效**：to-xxx 产出 skill 形态推导（Process+Template+门禁）。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。


# 产出形态 × skill 固化推演（v2 第 1 条骨架的形态分化依据）

> **状态**：推演临时稿（2026-09-02）。供 how 编写规范 v2 第 1 条（五段式骨架）定稿使用；是工单 05 的输入之一。
> **问题**：产出型 how 的"执行产出"一步，按产物形态是否要分化？哪些形态能固化为固定 skill（参考 mp-to-spec 形态：Process+Template 一体），哪些不能、需保留领域知识？
> **结论先行**：产出形态至少分**代码类 / 文档类 / 验收执行类 / 评审类**四大类；文档类与验收类可固化为产出 skill（按 to-spec 形态：Process+Template），代码类与评审类不固化（依赖领域内容），但**领域知识需有确定载体**（skill 内嵌或 how 内嵌）。

---

## 1. 判定维度（用户确认版）

| 维度 | 问什么 | 用户裁定（2026-09-02） |
|---|---|---|
| ① 形态是否同质 | 跨流程（business/tech/task/plan）实例产出步骤是否一致 | 推演确定 |
| ② 是否需要领域知识 | 步骤能否脱离具体需求内容写成固定流程 | **是**；但如 mp-code-review 这类 skill 已内嵌领域知识——skill 化是领域知识的**载体**而非排除 |
| ③ 是否有模板/协议载体 | 已有模板/能力是否已覆盖 | 模板=最旧设计；**参考 mp-to-spec 形态升级**（Process+Template 一体） |
| ④ 稳定性收益 | 固化后是否降漂移 | 是 skill 化的动机 |
| ⑤ 维护成本 | 谁维护、如何演进 | 随模板体系演进 |

**方向（用户裁定）**：能明确固化为固定 skill 的 → **固化（提高稳定性）**；不必要或无法生成 → 经推演后**放弃**；qa 验收、CR 是否独立形态 → **推演确定**。

---

## 2. mp-to-spec 形态拆解（供对照给读者）

- **Process**：探索仓库 → 定 seams（测试接缝）→ 与用户确认 → 按模板写 → 落 `.plan/<slug>/spec.md`
- **Template**：Problem Statement / Solution / User Stories（长编号 As-an-actor）/ Implementation Decisions / Testing Decisions / Out of Scope / Further Notes
- **本质**：把"已确认的信息转成规范文档"——固化输入→模板→产出的链路，领域知识（seams 思想、用户故事写法）内嵌在 skill 文本里
- **可借鉴点**：①Process 步骤明确（做什么、什么时候问用户、落哪）；②Template 内嵌 skill 而非独立文件（模板=skill 的一部分）；③领域知识写进 skill 正文

---

## 2.5 qa 职责重定义（用户 2026-09-02 裁定；影响 flow-alignment / capability-review-v2，主推演 merge 时落）

**AI agent 业态下的 qa 定位调整**（三项裁定，所有流程 business/tech/task/plan 统一适用）：

1. **用例设计归属改开发侧**：case_design 由开发侧自产——rd 产后端口径、fe 产前端口径；qa 不再产出用例设计（原 `case-design-cap`（qa）职责转移；tdd 为开发侧技能，无需拆测试编写给 qa——qa 的用例工作已转开发侧）
2. **qa 评审节点保留**：保留 qa 参与的评审（如产品设计评审/RD 技术评审等）——**专门针对 qa 完善领域模型（qa 评审维度/缺陷判定准则沉淀为 qa 专用 skill），减少 qa 上下文体量**（只带最小上下文参与评审）
3. **qa 聚焦收拢**：联调约束（前后端 join 处约束）+ 验收 + 缺陷判定（respond_to_qa 场景维持）

**影响**：
- flow-alignment：case-design 节点（A7/B6）owner 从 qa 改开发侧；qa 相关评审位保留但 qa 以领域模型 skill 参与
- capability-review-v2：case-design-cap 能力归属调整；qa 领域模型 = 评审维度表（D3）+ qa-acceptance skill（D4=B）+ 缺陷判定准则
- 与 C 组裁定闭合：tdd/implement 保持开发侧，无拆分；implement-spec 作为 Leader 编排层（见 §3.1 补充）

---

## 3. 形态分类推演（四大类 + qa 验收/CR 专项）

### 3.1 代码类产出（backend-dev.dev / frontend-dev.dev）

- **同质性**：⚠️ 半同质——步骤框架一致（读依赖→实现→自测→登记），但实现内容完全依赖需求
- **领域知识**：**高**（设计契约实现、自测口径），且随需求变化
- **载体**：how 内嵌（现有五段式）；tdd 纪律已评估进 how
- **固化判定**：**不固化为产出 skill**（开发不能表单化）；领域知识=how 内嵌 + 上游文档（rd_design/case_design 即领域知识的载体）
- **但可固化的是"步骤框架"**：读依赖→实现→自测→登记的序列可作为**开发类 how 模板**（v2 骨架的代码类分支），非 skill

### 3.2 文档类产出（product-design / backend-design / frontend-design / case-design-cap / research-cap / arch-cap / alignment-cap / plan-cap）

- **同质性**：✅ 高——"读依赖信息 → 按模板章节骨架撰写 → quality_gates 自检 → 写盘"全流程一致，实例差异只在模板与 inputs
- **领域知识**：低-中——流程可表单化；领域判断（如产品方案可行性）靠评审环节兜底，产出环节只需按模板撰写
- **载体现状**：模板（docs/templates 18 个）+ 章节骨架标注 + quality_gates
- **固化判定**：✅ **固化为"产出 skill"**（按 to-spec 形态升级）：每个文档类能力对应一个 skill（**to-xxx 家族**，如 to-prd→product-design、to-techdesign→backend-design），Process=读 inputs → 按模板骨架撰写 → 自检 → 写盘；Template=现有 prd-template 等内嵌进 skill；领域知识（该文档的撰写要点、引用规约）写进 skill 正文
  - **to-spec 形态是家族模板，但命名不用 spec**（2026-09-02 二版）：**to-prd**（产品阶段，product_doc）、**to-techdesign**（技术设计，to-rd-techdesign / to-fe-techdesign），与 to-prd 对齐；原生 spec 概念废弃（"to-prd 是 to-spec 旧名"理解为方向：产品文档即 PRD，不再称 spec）
  - **收益**：模板+流程+知识一体，替代"模板独立文件 + how 自由散写"——降漂移（对应你说"模板是最旧的设计"）
  - **落点**：模板文件保持单一真相源（docs/templates/*），skill 引用（D1=B）；每文档类独立 skill（D2）

## 3.3.5 开发链路编排定稿（用户 2026-09-02 确认，二版命名）

**代码类执行的完整编排链路**（开发动作 how 的执行要素=子流程编排序列）：

```
需求对齐（交互）→ to-prd（产品阶段：完整 product_doc，补足产品内容）
  → to-techdesign（技术设计：to-rd-techdesign / to-fe-techdesign 产出 rd_design / fe_design）
  → [to-tickets] 基于 rd/fe_design 拆工单 + /goal 挂载目标（active→…→complete 跨会话追踪）
  → [implement-techdesign] 基于 rd/fe_design 编排（任务图：模块→tdd/implement→依赖序；按并发度串行/并行派孙 subagent）
  → tdd / implement 工人执行模块 → 组装 → 自测 → 登记
```

- 执行方：**RD/FE（开发侧 subagent）**（B 层级）；subagent 可再派 subagent（spawn depthLimit=true）
- to-tickets × /goal：**techdesign 产出**拆成垂直切片工单（Blocked by + 验收清单），goal 承载开发目标的事件溯源阶段（active/paused/blocked/complete + 自动续跑推进 rounds）
- 并发度：编排与调度解耦（先任务图后调度）；并发度=N 并行、=1 串行；MVP 默认串行；DSH 锚点 maxConcurrentJobsPerOwner
- how 执行要素形态：`execution: [to-tickets, implement-techdesign]`（受控枚举 + 可扩展，防自由编排漂移）——并入 v2 骨架定稿

---

### 3.3 qa 验收类（acceptance-test）——专项推演

- **形态**：**执行 + 文档混合**——不是纯文档产出：
  - 执行部分：建测试资产 → 按用例执行验收 → 缺陷返工闭环（依赖真实系统/代码）
  - 文档部分：test_result（AC 逐条核对）/ defect（缺陷详情）按模板产出
- **同质性**：⚠️ 半同质——执行步骤依赖被测内容；文档步骤同质
- **领域知识**：中（用例口径、缺陷判定）
- **固化判定**：**分片固化**——文档侧（test_result/defect 撰写）可固化为 skill（同 3.2 文档类）；执行侧（建资产/执行/缺陷闭环）**不固化**（依赖实际代码与用例，属代码类形态）
- **结论**：qa 验收 = **执行形态 + 文档形态的复合**，非独立第三类；其 skill 固化 = 文档侧单独固化

### 3.4 CR / 代码评审类（code-review 能力）

- **形态**：评审型（读被评对象 → 双轴对照 → 表态）
- **同质性**：✅ 步骤框架同质（双轴评审协议已定稿：Standards 轴→Spec 轴→聚合）
- **领域知识**：高（评审判定依赖内容），但**判定准则可内嵌**（mp-code-review 即内嵌了 smell 基线与评审步骤）
- **固化判定**：⚠️ **评审执行不固化为产出 skill**（判定不能表单化），但**评审准则可内嵌为 skill 领域知识**——**借鉴 mp-code-review**：把本项目双轴评审协议 + 编码规约基线 + 对照表核对方法，固化进一个 `code-review` 技能（评审者 subagent 调用），即"领域知识技能化"而非"产出流程技能化"
- **结论**：CR 是**评审类**形态（与产出类并列），其固化形态 = 评审知识 skill（非产出 skill）

---

## 4. 决策定稿（2026-09-02 用户批注）

| # | 决策 | 定稿 |
|---|---|---|
| D1 | 文档类 skill 的模板落点 | **B**：skill 引用外部模板路径（docs/templates/xxx）；模板=章节骨架/门禁单一真相源 |
| D2 | 文档类 skill 的数量 | **不同文档不同 skill**（每文档类能力一个 skill，非通用+参数化）——product-design-skill / backend-design-skill / frontend-design-skill / case-design-skill / research-skill / arch-skill / alignment-skill / plan-skill 等 |
| D3 | 评审知识 skill | **B**：共享评审协议 skill + 各维度 focus 表 |
| D4 | qa 验收文档侧 skill | **B**：单独 qa-acceptance skill（非并入文档类通用） |
| D5 | 模板体系演进 | 保留模板库为单一真相源，skill=Process+知识+引用（待确认，倾向保留） |

> 本次推演稿还延伸出新任务：**mp-* 技能按类别归入不同阶段/流程复用**（用户 2026-09-02），基于 skill-to-action-how-mapping.md 的基础分析重新确认引用方式与场景，与开发流程相关的技能逐一确认——见后续独立稿。

---

## 5. 关联影响

- **v2 第 1 条** ✅ 定稿（2026-09-02）：五段式废弃 → **四要素骨架**（目标[产出+可量化完成标准] / 依赖[具体看 mytask 返回中的什么，动态按流程类型/阶段] / 执行[MVP 基于 skill] / 自检[达成目标+按执行要求+质量达标]）+ **工具知识独立沉淀为 skill**（wb 接口用法固化，不写进每个 how）。产出型/评审型在四要素内部分支（各要素语义按形态标注）。详见工单 05 定夺日记。
- **技能评估稿**（skills-evaluation.md）：此前评估的是**外部技能包**复用；本推演是**项目自建技能**（产出 skill）——二者互补不冲突（外部技能作参考，自建技能进流程）
- **实现期**：自建 skill 落 `~/.dsh/.agent-presets/full/skills/`（按 preset 聚合规则）；需领域知识写作参考 mp-to-spec 的 Process+Template 一体化结构