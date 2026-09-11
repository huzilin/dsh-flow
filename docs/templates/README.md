# docs/templates · 产物标准索引（v0.29）

> **v0.29（User 拍板 D1-D6，semantica 47c53ed9）**：产出标准的权威载体 = **产出 skill**（`skills/<skill>/SKILL.md`，章节骨架/required/purpose/质量门禁/准出/接口协议内联）；DB `capabilities.products` 不存标准指针（`[{doc_path, demo_path}]`，template_path 已删）。无 skill 的产物暂由本目录 `*-template.md` 承载（**skill 待建**，文件头部有标注）。`*-demo.md`（填充示例）恒保留，与对应标准骨架保持一致（D5）。
>
> 运行规则：改 skill 文件即全局调标准（不碰 persona/代码/DB）；评审者按评审能力 how 的「按产出 skill 自检节逐条核对」步骤评审（D1）；自检在产出动作内完成，flow 无专门自检节点。

## 产物 × 标准 × 示例 总索引（13 类）

| 产物 | doc_path | 标准（权威） | 填充示例 |
|---|---|---|---|
| brd（业务需求文档） | `docs/<版本>/<需求>/brd/brd.md` | **to-brd** skill | — |
| prd（产品需求文档） | `docs/<版本>/<需求>/prd/prd.md` | **to-prd** skill | prd-demo.md |
| rd_design（RD 技术设计） | `docs/<版本>/<需求>/rd/design.md` | **to-rd-techdesign** skill | backend-design-doc-demo.md |
| fe_design（前端技术设计） | `docs/<版本>/<需求>/fe/design.md` | **to-fe-techdesign** skill | frontend-design-doc-demo.md |
| 用例/联调设计 | `docs/<版本>/<需求>/qa/cases.md` | qa-testcase-review-template.md（skill 待建） | qa-testcase-review-demo.md（待建） |
| 测试结果与验收记录 | `docs/<版本>/<需求>/qa/test.md` | qa-acceptance-template.md（skill 待建） | qa-acceptance-demo.md |
| 缺陷文档 | `docs/<版本>/<需求>/qa/defect.md` | qa-defect-management-template.md（skill 待建） | qa-defect-management-demo.md |
| 调研报告 | `docs/<版本>/research.md` | research-open-source-tech-template.md（skill 待建） | research-open-source-tech-demo.md |
| 架构规划 | `docs/<版本>/arch_design.md` | arch-version-design-template.md（skill 待建） | arch-version-design-demo.md |
| 需求对齐（版本范围） | `docs/<版本>/align.md` | arch-version-scope-align-template.md（skill 待建） | arch-version-scope-align-demo.md |
| RD 开发笔记 | `docs/<版本>/<需求>/rd/dev.md` | dev-notes-template.md（skill 待建） | — |
| FE 开发笔记 | `docs/<版本>/<需求>/fe/dev.md` | dev-notes-template.md（skill 待建） | — |
| 方案设计（任务） | `docs/<版本>/<需求>/plan.md` | plan-template.md（skill 待建） | — |

## 已知缺口（P2 待办）

- seed 引用的 `qa-testcase-review-template.md` / `qa-testcase-review-demo.md` **文件不存在**（case-design-cap 的 products 此前指向它；v0.29 删 template_path 后断链已消，但 cases.md 的标准载体实际缺失，建 skill 或补模板时一并解决）。
- to-brd 产出的 `brd/brd.md` 未注册进任何 capability 的 products（skill 链产物与数据模型产物册脱节，P2 处理）。

## 历史口径（v0.23-v0.28）

v0.23 模板外置（`template_path`/`demo_path` 入 DB）、v0.28 三列合并单列 products 的沿革见 `workbench-sql/schema.sql` 头注与 `workbench-sql/README.md` 版本注记；v0.29 起本目录职责收敛为「无 skill 产物的暂承载 + demo 示例 + 本索引」。
