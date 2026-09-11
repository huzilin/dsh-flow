---
name: to-brd
description: 把需求对齐对话收敛成业务需求文档（brd）——不重新访谈、只综合已有讨论；写清背景/目标与AC/范围/对齐结论，标准以本 skill 为权威。用于 pm 在需求对齐节点（business/tech/plan）的 exec 产出动作。需求对齐产物，接 to-prd 等产出 skill 的输入。
disable-model-invocation: false
---

# to-brd

产出《业务需求文档 brd》（`docs/{{req-version}}/{{req-name}}/brd/brd.md`）。brd 是**需求对齐节点**的收敛产物：把 pm 与 User 在这一节点已进行的对齐对话综合成可交付文档。**不重新访谈、不重新收集需求**——对齐对话已发生，这里只做综合与固化。brd 无评审回路，质量由对齐期 User 确认背书。

> **标准权威说明**：本 skill 承载章节骨架 / required / purpose / 质量门禁 / 准出标准的**权威版本**。原 `brd-template.md` 已于 v0.29 删除（D3 拍板，semantica 47c53ed9）；章节骨架以本 skill 必填节为唯一权威，不自造小标题。
> **对齐协议指针（落笔前门禁，2026-09-11 拍板形态 B）**：本 skill 是「需求→设计文档链」一环，定稿前过《对齐协议》`docs/templates/alignment-protocol.md`——追问关（grilling，本段 ≤3 轮 + Clarifications/Assumptions 台账）、上游依据声明（wayfinder 问题定义）与 BR-xxx 编号目标集交出、三席评审（本段：业务席）、拍板 ADR+semantica 双写、结构化交接。协议为正本，本 skill 不复制正文。

## 依赖（从 my_task 的 inputs 取）

| 输入 | 何时有 / 用于什么 |
|---|---|
| 需求对齐对话记录（wb_comment / my_task 附带） | §1 背景诉求、§4 用户场景、§5 对齐结论的来源——本节点已发生的讨论 |
| 对应 req 条目（wb_list_requirements） | §0 文档信息字段 + §1/§2 的 description 背景/目标/验收点核对来源 |
| 架构方向（architecture，use_threads=false 按需取） | §3 范围裁剪对照版本架构方向 |

> 对齐节点是**交互类**（entry → exec），entry 阶段 pm 已与用户对话、用户已确认。brd 的 exec 只做综合，产出后再向 User 确认一次即 approved。跨流程（business/tech/task/plan）都以 my_task 的 inputs 引用为准，逐个校验缺哪个来源，缺料不臆造。

## Focus（必读/盯住的上游章节——本 skill 产 brd，是 to-prd 的上游）

产出 brd 前**逐节读清并盯住**以下上游内容，缺任一即显式声明（gap-naming），不许跳过：

| 上游来源 | 必读章节/字段 | 产出到 brd 的哪节 |
|---|---|---|
| 需求对齐对话（wb_comment，本节点） | 用户诉求、范围确认、分歧与结论 | §1 背景 / §4 用户场景 / §5 对齐结论与分歧记录 |
| req 条目 description（wb_list_requirements） | **背景 / 目标 / 验收点**（description 的三个子块） | §1 背景 / §2 目标与 AC（§2.2 验收标准） |
| req 条目字段 | id / title / versionId / category / priority / status | §0 文档信息 |
| 架构方向（architecture，可选） | 版本方向 / 模块边界（若提供） | §3 范围裁剪对照 |

> 产出后到下游的交付口径：下游 **to-prd** 按本 brd 的「章节骨架标注」逐节取料——尤其 §2 目标与 AC、§3 范围与非范围、§4 用户与场景 是 to-prd 的直接输入。brd 里这些节必须可被 to-prd 逐节引用（章节名稳定、不自造小标题），否则下游取料断链。

## 参考材料

- **成品示例**：暂无 brd demo（待建）；标准以本 skill 为权威
- 需求池该 req 条目（requirements.json 的 背景/目标/验收点）
- 上游（Roadmap / 版本对齐结论 / 用户反馈），若 my_task 提供

## Process

1. **锁定 reader 基线（grounding）**：写 §1 前确认读者（pm 自身/User 复核/下游 to-prd）进门已有：本节点对齐对话 + 需求池 req 条目的 背景/目标/验收点。文中**新引入的领域/业务概念**（新模块名、新业务名词）**先定义再使用**，不自造与既有受控词冲突的同义；能指向 req 条目/CONTEXT 定义的就指向。

2. **对齐必填节（标准以本 skill 为权威）**：brd 必填节 = 背景与诉求 / 目标与成功标准(AC) / 范围与非范围 / 用户与场景 / 对齐结论与分歧记录 / 遗留问题与后续动作。章节名以本节清单为准，逐节落文，不得缺必填节、不自造小标题。

3. **核对需求池 req 条目**：读该 req 的 `description`（含背景/目标/验收点）与 `priority/category/status`，作为 §0/§1/§2 的事实底稿——brd 是 req 条目在需求对齐层的展开，**范围/AC 不得超出 req 条目 + 对齐对话已确认的范围**。

4. **按本 skill 骨架逐节撰写**，严格对照必填节章节名与 purpose：
   - §0 文档信息：req 字段带出（编号/名称/版本/类别/优先级/状态/上游来源）。
   - §1 需求背景与诉求：req 背景 + 对齐确认，讲清现状/痛点/不做的代价。
   - §2 目标与成功标准：目标 + 可验证 AC（沿用 req 验收点口径，对齐补充/修正）。
   - §3 范围与非范围：In / Out of scope 都写。Out of scope 必须含「想过但决定不做」项及原因（用户偏好：不静默裁剪/扩张）。
   - §4 用户与场景：角色 + 主场景 + 关键异常（对齐确认口径，简于 product_doc）。
   - §5 对齐结论与分歧记录：结论存档，分歧「结论+拍板人」，无悬而未决。
   - §6 遗留问题与后续动作：未决/待确认/待调研显式列出，**不静默假设**。

5. **形态辩护（format arguments）**：prose / list / table 有意识选择——
   - AC 清单、分歧记录、变更记录、In/Out 对照 → 表/list；
   - 背景叙事、价值论证 → 散文；
   - 写不下去改形态，不靠堆散文掩饰。

6. **证据缺失显式化（gap-naming）**：某节需证据/事实而手上材料没有时——
   - 不得编造 AC、范围、数据证据；
   - 写「**证据缺失：待补 <XXX>**」进 §6 遗留问题并给去向；
   - 影响必填节成立性（如 AC 无法判定、范围无法界定）时**回对齐对话向用户补齐**，而非带缺提交（brd 是 User 背书产物，缺关键事实不得静默通过）。

7. **落盘 + 自检**：写到 `docs/{{req-version}}/{{req-name}}/brd/brd.md`。按本 skill「自检」节逐条自检，通过后上报 exec 判据校验（brd 落盘 + 判据校验通过 → 向 User 确认）。

## 自检（完成前逐条过，任一 blocker 不过不许提交）

> **准出（brd 可被下游 to-prd 无歧义消费）**：下列 blocker 均为准出项——不满足即打回/回对齐对话补齐，不得带缺提交。

对照本 skill「自检」节逐条核对，至少覆盖：
- [ ] （blocker）逐条对照 req 条目 description（背景/目标/验收点）核对覆盖，无遗漏、无自行新增范围
- [ ] （blocker）每条 AC 均可用「通过/不通过」判定，无不可验证项
- [ ] （blocker）In/Out of scope 均有记录；Out of scope 含「想过但决定不做」项及原因
- [ ] （blocker）**下游可无歧义消费**：to-prd 能据 §2 目标与 AC / §3 范围 / §4 用户与场景 逐节取料展开 product_doc——无「待补才能开工」的悬空描述（悬空的进 §6 显式列出，不进正文当已定）
- [ ] （major）对齐分歧点已记录结论与拍板人，无悬而未决（未决项在 §6 显式列出）
- [ ] （major）**grounding**：新领域词已定义或指向定义，无裸用；不自造与既有受控词冲突的同义
- [ ] （major）**gap-naming**：凡证据缺失已写「待补 X + 去向」，未编造范围/AC
- [ ] 已按章节骨架核对必填节全部落文（章节名稳定，供 to-prd 逐节引用）

## 边界 / Out of scope

- 不重新访谈、不向用户收集新需求（对话已在对齐节点 entry 完成）。
- 不写产品方案（用户故事展开/功能清单/交互/UI）——那是下游 to-prd 产 product_doc 的活。
- 不写技术方案 / 接口 / 数据模型——那是 rd/fe 技术设计的活。
- 不自行定范围：范围来自对齐对话 + req 条目；越界点写 §6 或回对话确认。
