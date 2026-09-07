# M7：团队文档目录结构与开发完成检查（v2）

> 状态：**v0.2（重构：版本 → 需求 → 角色 三层组织；明确文档依赖矩阵；review 产物遗漏）**
> 两个核心概念：**版本（v-xx）→ 需求（req-xxx）**。文档按版本隔离，版本内按需求与角色落位；架构文档位于版本层，被该版本全部需求遵循。

## 1. 目录结构（改进示例）

```
docs/
├── design/                            # 制度与流程（不属版本，全局只读制度）
│   ├── m6-team-preset.md
│   ├── m6-flow-templates.md
│   ├── m6-agent-tools.md
│   ├── m6-gap-closure.md
│   ├── m7-agent-protocol.md
│   └── m7-doc-structure.md            # 本文档
├── <版本号>/                          # 如 v1.1/ v1.2/
│   ├── arch_design.md                 # 架构设计（Architect）——版本内全部需求大方向遵循
│   ├── roadmap.md                     # 版本规划：目标/范围/需求清单/拆分（版本规划任务产物）
│   ├── align.md                       # 需求对齐文档（版本规划①）
│   ├── research.md                    # 调研文档（版本规划②，Investigator）
│   └── <需求名>/                      # 需求名 = req-<id>-<短slug>（如 req-019-join-rollback）
│       ├── prd.md                     # 需求说明 + 产品设计（User/Leader 说明 + Architect 产品方案；业务需求①）
│       ├── rd/
│       │   ├── design.md              # RD 技术设计（业务③/技术②）
│       │   └── dev.md                 # RD 开发说明（业务⑨.1/技术⑥，含实现-设计对照表）
│       ├── fe/
│       │   ├── design.md              # FE 技术设计（业务⑤/技术④按需）
│       │   └── dev.md                 # FE 开发说明（业务⑨.2/技术⑥，含实现-设计对照表）
│       ├── qa/
│       │   ├── cases.md               # 用例/联调设计（业务⑦/技术⑤）
│       │   ├── test.md                # 测试用例与结果（业务⑩/技术⑦，含 §测试资产：数据构造+测试代码）
│       │   ├── defect.md              # 缺陷文档（QA 验收发现 bug 时）
│       │   ├── data/                  # 测试资产·数据构造（种子/造数脚本，幂等可重建，如 seed.sql/reset.sh/gen_user.py）
│       │   ├── code/                  # 测试资产·测试代码（回归/压测/校验脚本，含运行命令，如 test_settle.py/perf.jmx）
│       │   └── accept.md              # 验收记录（QA；v0.28 起并入 test.md，废弃留痕）
│       └── plan.md                    # 任务方案文档（category=任务 时，执行者产出）
└── prototype/                         # 历史 mock 产物（演练数据，迁移归并后删除）
```

**命名规则**：
- 需求目录：`req-<id>-<短slug>`（保留 id 关联工作台 .workbench 数据；slug 取需求主题，如 login-500）
- 版本目录：`v<主>.<次>`（与 .workbench/versions.json 的 id 对应，如 v-11 → `v1.1`）
- 文件：固定名（design.md / dev.md / cases.md…），版本演进覆盖同文件（git 留痕）
- 登记：产物写入后 `wb_add_artifact` 登记（href 指向该文件，label 用《文档名》，kind=document）

## 2. 文档依赖矩阵（各角色「读取什么、以什么为准」）

| 角色产出 | 必须一致/参考 | 来源（优先级从高到低） |
|---|---|---|
| **RD 开发**（rd/dev.md） | 本需求 RD 技术设计（rd/design.md） | 本需求 rd/design.md → 本需求 prd.md → 本需求 qa/cases.md（自测口径） |
| | 符合本需求 PRD 设计 | prd.md |
| | 自测满足本需求测例 | qa/cases.md |
| | 大方向技术设计符合当前版本架构 | v-xx/arch_design.md |
| **FE 开发**（fe/dev.md） | 本需求 FE 技术设计（fe/design.md） | 本需求 fe/design.md → rd/design.md（接口契约）→ prd.md（页面设计）→ qa/cases.md（自测） |
| | 接口参考 RD 技术文档 | rd/design.md |
| | 页面设计参考 PRD | prd.md |
| | 自测参考测例文档 | qa/cases.md |
| **RD 技术设计**（rd/design.md） | 本需求 prd + 版本架构 | prd.md + v-xx/arch_design.md |
| **FE 技术设计**（fe/design.md） | 本需求 prd + rd/design.md 接口 | prd.md + rd/design.md |
| **QA 用例**（qa/cases.md） | 本需求 prd + rd/fe design | prd.md + rd/design.md + fe/design.md |
| **QA 验收**（qa/test.md） | 本需求 cases + dev 对照表 | qa/cases.md + rd/dev.md + fe/dev.md |
| **QA 测试资产**（qa/data/ + qa/code/） | 本需求 cases 数据准备口径 + 可复跑脚本 | qa/cases.md（数据准备）→ test.md §测试资产沉淀；回归直接复用，不重复编写 |
| **版本规划**（align/research/arch_design/roadmap） | 版本内需求与上版本路线 | 需求池（wb_list_requirements）+ 上版本规划 |

**开发完成检查（RD/FE 通用，各审各的）**：
1. 按上表读取全部依据文档
2. 逐项核对：**覆盖**（无遗漏）/ **一致**（无偏离）/ **契约**（接口/数据结构）/ **边界**（状态处理与用例口径）
3. 输出「实现-设计对照表」附 dev.md：`| 设计要点 | 来源文档 | 实现位置 | 一致/偏差 | 说明 |`
4. 偏差处理：合理简化标注说明；行为/接口变更或范围遗漏 → **停止并报告 Leader**，不自行改设计
5. 完成判定：对照表全一致（或已说明合理偏差）+ 自测通过 + MR 就绪

## 3. 产物遗漏 review（对照 m6-flow-templates 全部定义）

| 流程定义的产物 | 新目录落位 | 此前遗漏？ |
|---|---|---|
| 产品设计文档（业务①） | `<需求>/prd.md`（并入需求说明，避免文档过碎） | ⚠️ 之前无独立落位（并入 desc） |
| RD 技术设计（业务③/技术②） | `<需求>/rd/design.md` | ✅ 已有（旧 prototype） |
| RD 开发说明 | `<需求>/rd/dev.md` | ✅ |
| FE 技术设计（业务⑤/技术④） | `<需求>/fe/design.md` | ✅ |
| FE 开发说明 | `<需求>/fe/dev.md` | ✅ |
| 用例/联调设计（业务⑦/技术⑤） | `<需求>/qa/cases.md` | ✅ |
| 测试用例与结果 | `<需求>/qa/test.md` | ✅ |
| 缺陷文档 | `<需求>/qa/defect.md` | ✅ |
| 验收记录 | `<需求>/qa/accept.md` | ✅ |
| 方案文档（任务②） | `<需求>/plan.md` | ⚠️ 之前无固定命名 |
| 需求对齐文档（版本规划①） | `v-xx/align.md` | ✅（req-018 示例） |
| 调研文档（版本规划②） | `v-xx/research.md` | ✅ |
| 架构规划（版本规划③） | `v-xx/arch_design.md` | ✅（大方向架构，全版本共享） |
| 需求清单/启动版本（版本规划④） | `v-xx/roadmap.md` | ⚠️ 之前为 req-018 内产物，应上提至版本层 |
| **架构文档（跨需求大方向）** | `v-xx/arch_design.md` | ⚠️ **此前缺失**：设计评审只对照本需求文档，缺版本级架构约束——RD/FE 开发新增「大方向符合 arch_design」检查项 |
| **版本规划路线（上版本输入）** | `v-xx/roadmap.md`（历史版本归档） | ⚠️ 版本规划①的「上版本路线」输入此前无归档来源 |

**结论（遗漏 3 类，已补进规范）**：
1. **版本级架构文档**（arch_design.md）：新增「RD/FE 开发大方向符合当前版本架构」检查项
2. **版本规划路线归档**（roadmap.md + 历史版本目录）：作为版本规划①输入
3. **产品设计/方案文档固定落位**（prd.md / plan.md）

## 4. 迁移计划

- 现有 `docs/prototype/req-0xx-*.md` → `docs/v1.1/<需求名>/{rd,fe,qa}/…`（按 §1 结构归位），同步更新 `.workbench/req-0xx/artifacts.json` 的 href
- 迁移后删除 `docs/prototype/`（历史 mock 清理）
- req-019 演练产物按新结构归位（rd/dev.md、fe/dev.md、qa/test.md）
