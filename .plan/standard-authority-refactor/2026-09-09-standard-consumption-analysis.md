# 三分消费分析：标准权威唯一处 vs 评审消费 vs 人工写作

> 2026-09-09 · handoff `2026-09-09-skill-standard-authority-template-refactor` §建议 指定的前置输入：在动模板/数据模型之前，把「谁在什么环节消费产物标准」拆成三条线逐一核证。**推演层，未拍板**；§7 给 D1–D7 编号选项与推荐，供裁决。

## 1 问题

上一轮把 4 类产物的标准（章节骨架/required/purpose/质量门禁/准出/接口协议）写进了产出 skill 并声明「skill 为权威」，但模板未动、数据模型未动。这形成一个**过渡态双源**：产出端读 skill、评审端仍读模板。本分析回答 handoff §A/§C 的三个悬置问题：

1. 模板废弃后，评审方（CR/QA/维度评审）与非 skill 人工写作谁接管参照？
2. 章节名权威唯一处落在 skill 后，全链 to-brd→to-prd→to-rd→to-fe + 评审方如何共享同一词表？
3. 产物是否结构化？（上轮已裁：仍是独立 md 文档——本文只复核，不翻案）

## 2 证据基线（实测）

**三条消费线现状**：

| 消费线 | 现在从哪读标准 | 证据 |
|---|---|---|
| 产出端（agent+skill） | skill 内联标准（4 类已迁） | 四个 SKILL.md「标准权威说明」+ 必填节内联 + 准出 blocker |
| 评审端 | `template_path` 指向的模板文件「附：质量门禁」表 | arch_design.md L1488-1496（quality_gates 双重消费②：评审者按 gates 核对）、L1499（评审端按 severity 分级核对）、seed.json capabilities[].products.template_path |
| 非 skill 人工写作 | `docs/templates/*-template.md`（骨架+门禁成对 + demo） | arch_design.md L1483（`*-template.md` 与 `*-demo.md` 成对）；运行模式 L1486/L1515④「改 docs/templates/ 即全局调模板」 |

**数据模型**：`capabilities.products` 为 JSON TEXT 列 `[{doc_path, template_path, demo_path}]`（arch_design.md L406）；`wb_my_task` outputs[] 经 capability_id 展开 template_path 注入产出者（L1366-1375、L1484）。seed.json 实测 19 个 capability、产出型 11 项共 11 条 template_path。

## 3 三个硬发现（决定走向）

**F1 评审端标准源未切换，双源漂移已实际发生，不是理论风险。**
本轮写进 to-rd-techdesign 的两个准出 blocker——「接口与契约真相源（api.proto）逐条一致」「序列化口径一致（protojson/int64→string/ErrorResp 同形/string 枚举）」——在 `backend-design-doc-template.md` 的质量门禁表（L222-230）中**一条都不存在**。评审者按现有机制（读模板 gates）核对，会系统性漏掉新准出。只要双源并存一天，评审端与产出端口径就持续分叉。

**F2 skill 面只覆盖 4/13 类产物，「立即废模板」会让 9 类产物失去 structure+gates。**
seed.json 注册的产物 doc_path 共 13 个（prd、rd/design、fe/design、qa/cases、qa/test、qa/defect、research、arch_design、align、rd/dev、fe/dev、plan，加 to-brd 的 brd/brd.md）。已有 skill 的仅 4 类（brd/prd/rd design/fe design）。其余 9 类（qa×3、research、arch、align、dev-notes×2、plan）的标准只存在于模板——模板一废即断粮。附带发现两个既有缺口：① seed 的 case-design-cap 引用 `docs/templates/qa-testcase-review-template.md`，该文件在 docs/templates/ **不存在**（引用断链）；② to-brd 产出的 brd/brd.md 未注册进任何 capability 的 products（skill 链产物与数据模型产物册脱节）。

**F3 `template_path` 改指 skill 不需要 DDL 迁移。**
products 是 JSON TEXT 列，字段结构不变、只改 seed.json 值（+ arch_design §10.3 散文与示例同步）。handoff §B 说「动数据模型」实际强度比预想低：schema 层零改动，风险集中在文档散文与 my_task 注入逻辑的行为确认（outputs[] 注入的是路径指针，不关心指向的是模板还是 skill——agent 读文件这一步不变）。

## 4 三条消费线逐线分析

### A 产出端（agent 经 skill 写作）——已收敛，剩注入路径确认

skill 内联标准已落地。剩余问题只有：my_task outputs[] 的 template_path 指向 skill 文件后，agent「读模板对照章节名」的现有动作原样变成「读 skill」——四 skill 已把必填节内联，不再依赖模板对照。**此线无未决问题**，等 D4 拍板后改 seed 值即闭环。

### B 评审端——真正的未决线

评审标准的现载体是模板 gates。候选：

- **B-a（推荐）评审端改读 skill 自检节**：skill 的「自检」节已是完整的 severity 分级清单（blocker 准出+major），就是 gates 的新载体。评审者经同一注入机制拿到 skill 路径，按自检节核对。单一真相源成立，F1 消除。代价：评审能力（rd-tech-review 等 8 项）的 how/dependencyDocs 需在实现层确认能注入 skill path——机制与现状同构（现在注入 template_path，届时注入同一指针），实现层改动小。
- B-b 双源同步维护：模板 gates 与 skill 准出逐条人工同步。否决理由：正是 F1 的漂移根源，且违反单一真相源纪律。
- B-c gates 抽独立文件（gates.md）双端共享：多一个载体、多一层指针，skill 自检节与 gates.md 仍是两份。仅当「评审端不便读全量 skill」时才有意义，当前不成立。

### C 非 skill 人工写作——低频线，不值得为它保留标准副本

人工写作（不走 skill 的人直接写文档）现在拿模板当骨架。要点：**skill 本身就是 markdown，人可以直接读**；且 demo 文件承担「成品长什么样」的示例价值，与标准权威无关。候选：

- **C-a（推荐）人也读 skill（纪律）+ demo（示例）**：模板文件可删（见 D3），人工写作失去的只是「骨架速查」，而四 skill 的「对齐必填节」段就是速查本身。
- C-b 为人工写作保留瘦壳模板（只留章节骨架+指针）：保住速查，但章节名一旦在瘦壳里出现就是第二份词表副本，词表漂移风险回来了（见 §5）。
- C-c 不管人工写作：范围外裁剪，与「模板是全团队参照」的既有运行模式冲突，不推荐。

## 5 章节名词表共享（悬置问题 1）

现状已经收敛得比预想好：四 skill 两两通过 Focus 表 +「产出后到下游的交付口径」段**逐节互咬**（brd §2.2→prd §5 AC→rd §3.5→fe §5），链条上每一段引用都是「上游节名→本产物节名」的显式映射，且各自声明「不自造小标题」。即**词表权威 = 各 skill 内联必填节，链条一致性 = Focus 表互咬**，没有第三处需要同步。

真正的残余风险不在全链内部，而在**链外产物**（qa/research/arch/align 等 9 类无 skill）：它们的章节名权威仍在模板。这与 F2 同根——D2 拍板前，这些模板必须原样保留，词表在它们的 skill（待建）建好后自然迁移。不建议为此建独立的「词表注册表文件」：又多一处真相源，违背单一真相源纪律（D6）。

## 6 产物结构化（悬置问题 2）

上轮已裁：产物仍是独立 md 文档，skill 定规则；结构化对象方案未裁。复核结论：**维持**。结构化（把产物拆成结构化对象存储）动的是产物形态与阅读器/评论机制，与「标准权威在哪」正交，不应捆绑在本 effort——强行捆绑会同时放大两件事的风险。维持独立记录，不设拍板项。

## 7 推荐组合与分期路线（供拍板）

**推荐组合**：D1-a + D2-a + D3-a + D4-a + D5-a + D6-a（各选项定义见下表后段落；汇总：评审读 skill 自检节；无 skill 的 9 类模板原样保留标「skill 待建」；有 skill 的 4 组模板文件删除、docs/templates/README.md 一页索引替代；template_path 语义扩为「标准来源指针」改值免 DDL；demo 保留；词表维持 skill 内联+Focus 互咬）。

| # | 决策 | 选项 | 推荐 |
|---|---|---|---|
| D1 | 评审端标准源 | a) 改读 skill 自检节 b) 双源同步维护 c) 抽独立 gates 文件 | **a** |
| D2 | 无 skill 的 9 类产物 | a) 模板原样保留、标注「skill 待建」 b) 本期全部 skill 化 c) 直接废 | **a**（与 F2 对应：废即断粮；b 扩 effort 爆炸半径） |
| D3 | 有 skill 的 4 组模板文件（brd/prd/backend/frontend-template.md） | a) 删文件，docs/templates/README.md 一页索引（产物×权威 skill×demo 指针） b) 瘦壳指针文件 c) 原样保留 | **a**（b 的瘦壳仍含章节名副本，词表漂移回流；c 保双源） |
| D4 | 数据模型 | a) template_path 语义扩为「标准来源指针」，值改指 skill，无 DDL b) 改名 skill_path（需 seed+迁移脚本） c) 删字段 | **a**（F3：值层面改动，schema/ER 图不动；语义变化写进 arch_design §10.3） |
| D5 | demo 文件 | a) 保留（示例价值独立于标准权威） b) 随模板删 | **a** |
| D6 | 章节名词表唯一处 | a) 维持各 skill 内联必填节 + Focus 表互咬 b) 独立词表注册文件 | **a**（§5：现状已互咬，加注册文件=第三处真相源） |
| D7 | 产物结构化 | 上轮已裁独立 md，本文维持 | 维持（§6，不重开） |

**拍板结果（2026-09-09，User 裁决，semantica `47c53ed9…`）**：D1=a ✅（附澄清：**自检直接在 skill 内完成，flow 无专门自检节点**——不新增 flow 节点，评审端核对清单 = skill 自检节）；D2=a ✅；D3=a ✅；**D4 改判「删 template_path 字段」**（偏离推荐 a：标准指针不入 DB，products 收敛为 `[{doc_path, demo_path}]`，产出/评审端经能力→skill 约定寻址，skill frontmatter「用于 X 动作」即绑定声明）；D5=保留 + **demo 与 skill 保持一致** ✅；D6=a ✅；D7=维持 ✅。

**D5 实测差异清单（demo vs skill，effort 内校准）**：
- `prd-demo.md`：缺 **§4.4 动机场景**（to-prd 必填节 + to-rd/to-fe Focus 表按 §4 引用用户场景；demo §4 仅 4.1/4.2/4.3）。
- `backend-design-doc-demo.md` L118：§3.5 标题为「接口设计」，缺 **「（契约）」后缀**（to-rd skill 自检与 to-fe Focus 表均按「§3.5 接口设计（契约）」逐节引用）；内容层另需校准增量清单/真相源指针写法。
- `frontend-design-doc-demo.md`：§1 缺 **§1.1 动机场景**子节（to-fe Process 3 明确「§1 背景与目标 + §1.1 动机场景」）；边界态处理说明为内容层校准项（结构上可落在 §2.3/§5 内文）。

**分期**（按拍板结果调整）：

- **P1**（D1/D3/D4/D5 有 skill 子集，低风险）：seed.json **12 条** template_path 删除（acceptance-test 挂双产物，前文按文件 uniq 误计 11 条，以此为准；products 收敛为 `[{doc_path, demo_path}]`）+ arch_design §10.3 及 L406/428/860/894/1268/1366-1375/1483-1520 散文与示例同步（含「改 docs/templates/ 即全局调模板」运行模式改写为能力→skill 寻址）+ worktree schema.sql/ER.md 注释同步 + 删 4 个模板文件 + docs/templates/README.md 索引 + demo 三处结构差异校准 + wb_my_task outputs[] 不再注入 template_path 的行为确认（在 worktree 基线上做，勿混本分支——handoff 既有建议）。

**✅ P1 实施记录（2026-09-09，User 改判基线：弃 worktree，直接 main 实施）**：
- 数据层：seed.json / seed.sql 双源删 12 条 template_path + 6 条产出 how 切 skill 口径 + 5 个评审能力 how 加 D1 核对步骤 + `schema_version` 28→29；内存库灌 seed.sql 与 seed.json 逐能力对账 **19 能力 0 mismatch**。
- 代码层：`rd/repo/payload.go` ProductDef 删 TemplatePath 死字段（实现线 v1.0 的 outputs 走 outputProductTypes 机制，从未消费 template_path——零行为变更）；`go build` + `go test ./...` 8 包全绿。
- 文档层：arch_design.md 7 处（含 §10.3 重写为「产物标准机制」）+ schema.sql/ER.md/README.md/demo-data.md v0.29 注记与活口径同步。
- 模板层：删 4 组模板文件（brd/prd/backend/frontend）；7 个现存无 skill 模板头部标「skill 待建」（D2）；新建 `docs/templates/README.md` 总索引（13 类产物×标准×demo + 已知缺口）。
- skill 层：四 skill 模板引用 26 处清零（权威说明改「模板已删、以 skill 为权威」、参考材料改指 demo、自检引用从模板 gates 切 skill 自检节）；demo 三处结构校准落盘（prd §4.4 / backend §3.5（契约）+ 真相源口径 / frontend §1.1）。
- 遗留（P2）：qa-testcase 模板/示例文件缺失、brd 未入 products 册、brd demo 待建、9 类无 skill 产物逐类建 skill。
- **P2**（D2 子集，随使用频率逐类决策）：9 类无 skill 产物各自决定「建 skill 或模板长期保留」，每类一次小拍板；顺带修 F2 的两个既有缺口（qa-testcase 模板断链、brd 未入 products 册）。
- **P3**（可选）：demo 处置复核、docs/templates/ 目录整体归位。

## 8 与既有账目的关系

- 承接：handoff `2026-09-09-skill-standard-authority-template-refactor`（§A 冲突点、§B 数据模型、§C 连带决策 = 本文 §4/§5/§6 逐条对上）；semantica `d4bc9703…`（标准进 skill 权威）。
- **D1–D7 已拍板落账**：semantica `47c53ed9-3b51-4ef8-a8f2-37e97858246c`（CAUSED ← d4bc9703-bd78-4ac6-840a-a506fdec1d40，2026-09-09）。本文 §7 拍板结果与分期为正本，handoff 已同步。
- **P1 已实施（2026-09-09，main 直接落地——User 改判弃 worktree 基线）**：实施记录见 §7 P1 实施记录块；semantica `2e9b5d3c-f8d2-4984-9977-81bc88c03793`（CAUSED ← 47c53ed9）。下一步 = P2（9 类无 skill 产物逐类决策 + qa-testcase 断链/brd 入册两个既有缺口）。
