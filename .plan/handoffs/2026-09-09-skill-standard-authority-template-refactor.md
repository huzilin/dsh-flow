# Handoff: 产出 skill 承载全部标准 → 模板迁移/废弃 + 数据模型重构

> 2026-09-09 · 前会话：经 grilling 访谈收敛方向——**标准全部写进产出 skill（权威），模板瘦身/废弃，数据模型 `products[].template_path` 语义重构**，作为**独立 effort** 后续执行。本会话已把标准写入四个产出 skill（to-brd / to-prd / to-rd-techdesign / to-fe-techdesign），但**尚未**动数据模型/seed/arch_design，也**未**废模板。下一 effort 需接本交接继续。
> **✅ 2026-09-09 追记：D1–D7 已拍板**（三分消费分析 + User 裁决，semantica `47c53ed9…`，正本见 `.plan/standard-authority-refactor/2026-09-09-standard-consumption-analysis.md` §7）——本 handoff §A/§B/§C 的开放问题全部收敛，effort 可按分析文档 P1 清单开工。
> **✅ 2026-09-09 追记 2：P1 已实施完毕（main 直接落地，User 改判弃 worktree 基线）**——seed 双源删 template_path（schema_version 29）、评审 how 加 skill 核对步骤、payload.go 删死字段（go test 8 包绿）、arch_design §10.3 重写、4 模板删除 + 7 模板标「skill 待建」+ README 索引、四 skill 模板引用清零、demo 三处校准。实施明细见分析文档 §7「P1 实施记录」。剩余 = P2（9 类产物逐类建 skill + qa-testcase 断链 + brd 入册）。

## 访谈收敛结论（正本）

- **接口表达层**：rd_design 接口节 = 引用 `api.proto`（契约真相源）+ **本模块增量清单**（新增/改动 message、字段、错误码），不内嵌全量 proto。
- **序列化约定（普适纪律）**：protojson（`EmitUnpopulated`：零值可见/出参恒显式）；`int64`（unix 毫秒等）JSON 出参为字符串、入参可字符串或数字；枚举态字段用 string；统一失败响应 `{ok:false, error, code, blocked?, missing?}`（ErrorResp 同形）；结构化载荷 string 承载 JSON 文本。**已写入 to-rd-techdesign**。
- **契约 owner**：rd 唯一真相源；fe 只消费、逐字段对齐、无自造契约。
- **准出标注**：沿用 severity（blocker/major/minor）分级；「契约可被下游**无脑消费**」提升为 blocker；四文档各加契约一致性准出——
  - to-brd / to-prd 侧无 proto，以「**下游可无歧义消费**」为准出（brd→to-prd 逐节取料无缺；prd→to-rd/to-fe 逐 AC 展开无歧义）。
  - to-rd-techdesign：接口与 api.proto/既有契约逐条一致 + 序列化口径一致 = blocker。
  - to-fe-techdesign：接口消费逐字段对齐 rd 契约 + 前端类型/序列化口径随 rd = blocker。
- **标准权威归属**：**以 skill 为权威**，模板暂留为「产出结构示意」供对照章节名；skill 与模板不一致处以 skill 为准，并在每个 skill 顶部注明「模板待迁移」。

## 本轮已改动（文件清单）

- `docs/templates/skills/to-brd/SKILL.md` — authority 声明 + 必填节内联 + 「下游 to-prd 无歧义消费」准出 blocker
- `docs/templates/skills/to-prd/SKILL.md` — authority 声明 + 必填节内联 + 「下游 to-rd/to-fe 无歧义消费」准出 blocker
- `docs/templates/skills/to-rd-techdesign/SKILL.md` — **契约真相源 + 序列化约定**节 + 必填节内联 + 「接口与 api.proto 逐条一致 + 序列化口径一致」准出 blocker
- `docs/templates/skills/to-fe-techdesign/SKILL.md` — **接口消费口径（FE 只消费逐字段对齐）**节 + 必填节内联 + 「逐字段对齐 + 序列化口径随 rd」准出 blocker

## 下一 effort 范围（尚未做）

### A. 模板迁移/废弃
- 评估 9 个产出类模板（backend/frontend-design/prd/brd/research/arch/qa-acceptance/qa-defect/plan/align-template）+ 对应 demo 的去留。
- 把「章节骨架 required/purpose + 质量门禁 + 准出」从模板整体收敛进各产出 skill（作为权威唯一处），模板瘦身为纯结构示意或删除。
- **冲突点**：模板是产出结构示意，也是**评审方**（CR/QA 按章节结构核对）与**非 skill 人工写作**的参照。废弃后这三类使用者谁接管？——✅ 已裁：评审方读 skill 自检节（D1=a）、人工写作读 skill+demo（D5/C-a 分析结论）、有 skill 的 4 组模板删除换 README 索引（D3=a）、无 skill 的 9 组模板保留标「skill 待建」（D2=a）。

### B. 数据模型重构（跨层，风险最高）——✅ 已拍板：删 template_path 字段（D4）
- `arch_design.md` 数据模型把 `capabilities.products[]` 存 `{doc_path, template_path, demo_path}`（`docs/v1.0/arch_design.md` L406/428/860/1368/1375/1484/1506-1520），喂 `wb_my_task` outputs[]，并有「改 docs/templates/ 即全局调模板」的运行模式。
- **拍板（D4）**：`template_path` **删除**，products 收敛为 `[{doc_path, demo_path}]`——标准指针不入 DB，产出/评审端经能力→skill 约定寻址（skill frontmatter「用于 X 动作」即绑定）。原「改指 skill / 瘦壳」两案被否。
- 实施面：seed.json（11 条 template_path 删）+ worktree schema.sql/ER.md 注释 + arch_design §10.3 散文与示例（含运行模式 L1486/L1515④ 改写）+ wb_my_task outputs[] 不再注入 template_path 的行为确认。
- 涉及 worktree `.worktrees/wbflow-impl`（实现层已按旧模型建 schema/seed）。**在 worktree 基线上改，勿混本分支。**

### C. 需一并定稿的连带决策——✅ 全部已裁（semantica `47c53ed9…`）
- 章节名权威唯一处（D6=a）：维持各 skill 内联必填节 + Focus 表互咬，不建独立词表注册文件。
- 评审端标准源（D1=a）：评审读 skill 自检节；自检在 skill 内完成，flow 无专门自检节点。
- 产物形态（D7）：维持独立 md。
- 其余：D2=a（无 skill 的 9 类模板原样保留标「skill 待建」）、D3=a（有 skill 的 4 组模板文件删除，docs/templates/README.md 一页索引）、D5（demo 保留 + 与 skill 保持一致，三处结构差异清单见分析文档 §7）。

## 关联决策记录
- semantica：`1db5b40d…`（brd 需求对齐层骨架）、`d88d461c…`（实现 to-brd + to-prd 承接）、`cae1bcf6…`（四 skill 各加 Focus 表）——本 handoff 追加 `67e2435b…`（删 7 技能）无关。本次访谈/改动决策见会话内记录。

## 建议
- 新 effort 若把「模板 vs skill」定为独立设计问题（而非本轮仓促废），先产一份「标准权威唯一处 vs 评审消费 vs 人工写作」三分消费分析，再动数据模型。git 基线上改，勿在本分支混改。
- ✅ 该分析已产出：`.plan/standard-authority-refactor/2026-09-09-standard-consumption-analysis.md`（2026-09-09）——三消费线核证 + 硬发现 F1（评审端按模板 gates 审会漏 skill 新准出，双源漂移已实际发生）/F2（skill 仅覆盖 4/13 类产物）/F3（template_path 改值免 DDL），D1-D7 编号选项供拍板。
