# dsh-flow 现状 Gap 分析与补齐方案——审批稿

> 日期：2026-09-11。对象：`~/workdir/dsh-flow`（main @ 1f10478，wbflow v1.0 已并入）。
> 对照基准：`.plan/research/agent-flow-test-coverage.md`（测试体系目标）、`.plan/research/requirement-alignment-protocols.md` + `.plan/proposals/alignment-protocol-proposal.md`（对齐协议五机制）。
> 性质标注：【实测】=本日项目内 grep/文件抽样证实；【推断】=我的分析判断；【业内结论】=调研报告有一手来源。

---

## 0. 一页结论

dsh-flow 的**执行层已经是目标体系的水准**（qa 资产 196 PASS 基线、环境/文档双巡检、用例编号体系、术语表、semantica 留痕全部在跑），gap 集中在**上游两处**：①需求条目没有稳定编号，条目级追溯断裂（最大缺口）；②设计推演层无模型完备性检查（状态表/迁移差集），「设计断联」的根因口子还开着。补齐按三期走：一期纯本地脚本+文档规范（1-2 个 session 可完成），二三期挂在你尚未拍板的对齐协议方案上。

## 1. 已达标项（无需补，避免重复建设）【实测】

| 项 | 证据 |
|---|---|
| 测试资产三原则 | `qa/`：qa_env_up/down、qa_drive.py（28 端点×4 流程 40 节点全分支，基线 196 PASS/0 FAIL/1 SKIP）、qa-results.json、退出码可接 CI |
| 教训反哺闭环 | qa_live_audit.py（D-3 用户环境巡检，BUG-015 衍生）、qa_doc_audit.py（D-4 文档口径巡检，BUG-002 衍生）——test-design-gate 的反哺条款已实证 |
| 用例编号体系 | qa-test-cases.md：A/B/C/D 四组编号（A-01~30/B-1~5/C-1~12/D-1~2）+ 组粒度覆盖矩阵 |
| 术语单一真相源 | CONTEXT.md 术语表，_Avoid_ 退役词标注在跑 |
| 决策留痕 | commit message 引 semantica decision id（如 c25c0552） |
| 结构化交接 | .plan/handoffs/（CR 闭环、原型集成等四份）、.wayfinder/（map+tickets+research 分层） |
| 流程末端验证 | rd/e2e 三剧本 + 43 个测试文件 + 真库 IT |

## 2. Gap 清单（按严重度排序）

### G1 条目级追溯断裂【实测+推断】——最核心

- AC 无稳定编号：spec §4.1/§4.2 是 checkbox 列表，无 AC-xxx 编号（全库 grep `BR-/FR-/TD-/TC-` 零命中）。
- 用例对 AC 的引用是段落级（qa-test-cases C 组写「spec §4.1 AC 逐条」）——spec 一改段，引用即漂移。
- PRD §11「追溯链接」是文档级（prd→design.md→arch_design.md），非条目级 RTM。
- **后果【推断】**：没有稳定的「需求条目集合」，就无法计算「哪些需求/迁移没有测试覆盖」——零覆盖检测（Spec Kit /analyze 的 CRITICAL 判定）不可能，测试完备性永远靠人断言「测全了」。

### G2 模型完备性无检查【实测】——设计断联根因口子未关

- docs-tmp 推演稿无 ISTQB 状态表（grep「状态表/状态机表」零命中），状态机描述散在 capability-model-final（state 提及 11 处，非表格）。
- 无「模型定义迁移集（seed 40 节点×node_actions）vs 已测迁移集」差集脚本。
- **后果【推断】**：spec 漏写的迁移（T17 抓过的「提评无端点」类）→ AC 就没有 → C 组就不会测 → 「oracle 本身有洞」盲区原样存在。上一轮测试调研的缺口 1/2 在此项目未补。

### G3 追问关未制度化【实测】

- grilling 在 wayfinder map.md 里是「有疑问时 consult」风格，非每段落笔后的强制门禁。
- prd 模板只有「未决问题（Open Questions）」checkbox 节（L270），无 Clarifications 问答台账（问了什么→答了什么→回写到哪一节）。
- **后果**：漏问的细节静默固化进文档；答过的没有留痕，下轮还会重问。

### G4 文档级对抗评审缺失【实测+推断】

- 现有 CR 闭环（.plan/handoffs/2026-09-08-cr-loop 等）是**代码 CR**；PRD/techdesign 定稿前的 fresh-context 三席评审（业务/技术/QA，报缺口不报风格）无痕迹——文档是生成者自审自过。

### G5 上游变更无下游感知【实测】

- prd 改动后 design.md / qa-test-cases 无联动核对机制；qa_doc_audit 只扫「禁用短语」口径，不查结构性一致（AC 改了用例没跟，它发现不了）。

### G6 契约层 CI 缺位【实测】

- buf generate 工具链在（make proto），但 buf breaking / oasdiff 式 schema 破坏性变更拦截无；「动作 ID 下游解析失败」类断联仍靠 E2E 晚期暴露。

## 3. 补齐方案（三期，均待拍板）

### 一期：纯本地脚本+文档规范，零新依赖，1-2 个 session

1. **AC 稳定编号 + 轻量 RTM（治 G1）**：spec §4.1/4.2 逐条加 `AC-<域>-<序号>`（如 AC-FLOW-01）；qa-test-cases 覆盖矩阵加「AC 引用」列，形成 AC↔用例双向映射；qa_drive.py 的 ck(cid) 已有编号，只需加一层映射断言「每条 AC 至少被一个 cid 覆盖」。
2. **迁移差集脚本（治 G2）**：从 `rd/migration/sql/0002_seed.sql`（40 定义节点×node_actions）枚举模型迁移集，对照 qa_drive.py 已走分支输出差集清单；顺带逐条校验动作 ID 可解析（align.opening 类缺陷此步即拦）。差集=补测清单，喂给 C 组。
3. **推演稿补状态表（治 G2 设计侧）**：对 session 状态机、issue 状态机、评审轮转三张填 ISTQB 状态表（行=状态，列=事件，空格=非法迁移显式化）——填表过程本身就是设计评审【业内结论：ISTQB 教材】。
4. **buf breaking 拦截（治 G6）**：pre-push/CI 加 `buf breaking` 对照 main 基线。

### 二期：对齐协议落地（依赖你上轮未拍板的方案）

5. 对齐协议正本 + to-brd/to-prd/to-rd-techdesign 加指针（=上轮拍板 1 选 B + 拍板 4）；prd 模板「未决问题」节升级为 Clarifications 台账（问→答→回写章节）+ 落笔后 grilling 门禁（治 G3，=上轮拍板 2）。
6. PRD/techdesign 定稿前 fresh-context 三席评审（治 G4，=上轮拍板 2 分歧点）。

### 三期：自动化（可选，协议跑顺后）

7. 上游变更 Refine 提示 hooks（治 G5，=上轮拍板 3）；qa_doc_audit 扩展为 AC↔用例结构一致巡检。

## 4. 待拍板项

1. **一期四项是否授权执行**（纯本项目内改动：spec/qa-test-cases/qa_drive.py/推演稿/CI 配置）。推荐：授权 1、2、4；第 3 项状态表建议放推演稿惯例内由你过目。
2. **二三期与对齐协议拍板合并**：上轮 alignment-protocol-proposal.md 的拍板 1-4 仍悬置，本 Gap 的 G3/G4/G5 就是它的实证输入——两份可一次拍。
3. **AC 编号粒度**：版本级 AC（§4.1）与模块级 AC（§4.2）是否统一编入 RTM，还是只做模块级。推荐：都编，版本级条数少成本低。

## 5. 性质汇总

- 【实测】第 1 节全部、G1-G6 的证据行；
- 【业内结论】ISTQB 状态表填表即评审、零覆盖判 CRITICAL、三席评审协议；
- 【推断】后果链（G1/G2 的「后果」段）、三期划分与工作量估计、全部推荐意见。
