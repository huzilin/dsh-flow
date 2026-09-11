# 需求→设计文档链 · 对齐协议（正本）

> **版本**：v1.0（2026-09-11 User 拍板，形态 B：协议正本一份 + to-* skill 指针，semantica 留痕）。
> **范围**：wayfinder → to-brd → to-prd → to-rd-techdesign / to-fe-techdesign 四段流程的**段间协议**。各段产出的章节骨架/质量门禁以各自 skill 为权威（v0.29 口径），本协议只管**段与段怎么衔接**——不复制、不覆盖 skill 正文。
> **依据**：`alignment-protocol-proposal.md`（五机制设计 + 业内对照：GitHub Spec Kit /clarify、Amazon Kiro EARS、RTM、Anthropic 对抗评审、MetaGPT 结构化交接，41 条一手来源见调研报告）。
> **消费方式**：to-brd / to-prd / to-rd-techdesign / to-fe-techdesign 四个 skill 头部持有指针；执行者落笔前读本协议对应机制节。

---

## 机制 1：追问关（治漏问+漏写）

每段 skill 产出初稿后、**定稿前**，强制过 `grilling` 追问门禁：

1. 按覆盖面分类扫描歧义：功能范围 / 数据模型 / 交互流 / 非功能 / 外部依赖 / 边界 / 约束 / 术语 / 完成信号 / 占位符（spec-kit `/clarify` 十类）。
2. 按影响 × 不确定性排序；**一轮一问组、每题附推荐答案**；轮次上限（参数，起步值，跑一轮后校准）：

| 段 | 轮次上限 |
|---|---|
| to-brd | ≤3 轮 |
| to-prd | ≤2 轮 |
| to-rd-techdesign / to-fe-techdesign | ≤2 轮 |

3. 每个答案**当场回写进文档对应章节**（不许只在对话里）；文档尾部追加 `## Clarifications` 台账：Q→A→回写章节 逐条。
4. 用户答「这个歧义是有意的」→ 记入 `## Assumptions`（写明不解决的代价），不硬编答案。
5. 完成判据沿用 loop-me：**实现者 agent 不再需要问任何问题**。

## 机制 2：全链编号 + 轻量 RTM（治跨文档漂移）

- 编号规范（条目正文可改、位置可移，**id 不变**）：

```
BR-xxx（业务目标，BRD） → FR-xxx（功能需求，PRD） → TD-xxx（设计条目，技术设计）
                                                            ↘ TC-xxx（测试用例，qa-test-cases「组-序号」）
验收标准沿用现行 AC-V-NN（版本级）/ AC-M-NN（模块级）（spec §4 编号规范注）
```

- 每段文档头部带**「上游依据」声明**：消费了哪些上游 ID（如 to-prd 声明消费 BR-01、BR-03）。
- 每段产出后生成/更新**轻量 RTM**（表格即可：需求 Key → 下游覆盖物 ID → 有无覆盖）；**零覆盖需求标 CRITICAL 不放行**。
- 术语单一真相源 = `CONTEXT.md` 术语表；每段产出后核对同一概念跨文档叫法一致（双源词汇版本一致纪律）。
- **方案敲定三动作（常态化，2026-09-12 起）**——每轮拍板定稿回写 `decision-timeline.md` 的同时，必走：
  1. **词汇表联动**：涉术语变更 → 更新 `CONTEXT.md` 词条（现行定义 + `_Avoid_` 退役词），并在 `.plan/timeline/terminology-evolution.md` 追加演进行（旧名→新名、拍板依据）；
  2. **退役词清查**：被退役的词按 `.plan/timeline/retired-terms-audit.md` 四档流程过一遍（A 误用改 / B 历史语境留 / C 零引用归档 / D 代码返工）；
  3. **审计归位**：沿革与变更叙述进 `.plan/timeline/` 与 `.archive/`，**现行文档只留现行定义，不留变更日志**。
  （历史依据：词汇表梳理做过但未常态化导致断链——09-12 User 拍板固化为本协议强制步。）

## 机制 3：三席评审（治漏评——Three Amigos 的 agent 版）

文档定稿前，派**无本段上下文**的评审 subagent（只给产物 + 评审检查单 + 上游文档；**报 gaps 不报风格**；输出按 CRITICAL/HIGH/MEDIUM/LOW 分级挂文档位置；**CRITICAL 未清零不放行下一段**）：

| 席位 | 盯什么 | BRD | PRD | 技术设计 |
|---|---|---|---|---|
| 业务席 | 目标是否达成上游承诺、范围蔓延 | ✅ | ✅ | ✅ |
| 技术席 | 可行性、与 ADR/semantica 既有决策冲突、接缝完整性 | — | ✅ | ✅ |
| QA 席 | 可测性、状态机条目是否配状态表、验收判据可判定 | — | ✅ | ✅ |

（起步值：BRD 只业务席，PRD/技术设计全三席；每席一个 subagent，跑一轮后按成本校准。）

## 机制 4：拍板一律留痕（治无痕）

- 结构拍板（裁歧义、选方案）按 **ADR 五段式**追加编号记录：Title / Status / Context / Decision / Consequences；被推翻的旧决策标 Superseded **不删**。
- 技术设计强制「Alternatives considered」节。
- 与 semantica 决策账本**双写**：ADR 面向文档读者，semantica 面向 agent 溯源——引用不复制。

## 机制 5：交接物结构化（段间协议）

skill 之间只交接**固定 schema 的文档与编号集**，不传聊天记录（MetaGPT 核心教训）：

| 交接 | 交什么 |
|---|---|
| wayfinder → to-brd | 一页纸问题定义 + 证据清单 |
| to-brd → to-prd | BRD（带 BR-xxx 编号目标集 + Clarifications/Assumptions 台账） |
| to-prd → to-rd/to-fe-techdesign | PRD（带 FR-xxx 编号需求集 + 上游依据声明 + RTM） |
| 技术设计 → 测试设计 | TD-xxx 编号条目 + 状态表（接 qa-test-cases 与 qa_branch_audit） |

## 分层句式约定

- BRD 层：叙事 + 目标编号（不强制句式——目标/愿景强行句式化会丢上下文）。
- PRD 层：验收标准二选一强制 GWT 或 EARS 句式。
- 状态机类条目：强制配 ISTQB 状态表（空格=非法迁移显式化；范例见 `.plan/proposals/state-tables-draft.md`）。

## 维护纪律

- 本协议是**正本**：改协议即全局生效，四个 skill 只持指针——不许把机制内容复制进 skill。
- 参数（轮次上限、席位范围）跑一轮后校准一次，校准记录进 Clarifications 式台账或 ADR。
