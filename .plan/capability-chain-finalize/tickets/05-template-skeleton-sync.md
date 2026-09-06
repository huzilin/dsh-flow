---
type: task
blocked_by: []
claimed_by: huzilin+agent session
claimed_at: 2026-09-02
---

# 模板骨架标注同步

## Question

修复 qa 系列模板「正文 `##` 标题 vs 章节骨架标注」不同步问题（capability-artifact-analysis.md §6-1 发现）：统一模板正文标题与模板骨架章节名（以骨架标注为准），为产物登记预检（§1.5-A）与 inputs/actions 的章节引用（用章节名）铺路。

完成后记录：同步规则与涉及模板清单。

## 关联进展（2026-08-27，来自技能评估工单 01 resolution）

技能评估已按同向纪律落盘 4 个模板的变更（docs/templates/）：
- **prd-template.md / arch-version-design-template.md / backend-design-doc-template.md / frontend-design-doc-template.md** 各自新增「动机场景（用户故事）」节（prd §4.4 / arch §2.3 / rd §1.1 / fe §1.1），**章节骨架标注表同步补条目**（✅ 必填）——已按"正文=骨架"规则执行，未引入不同步。
- to-spec 的 Out of Scope 纪律进三设计阶段文档（arch/product/rd）。

## Answer（2026-09-02 关闭）

**前置变化**：用户裁定 qa 职责重定义（用例设计不再设独立节点与评审节点——QA 在验收节点内设计完直接测试，合并进验收文档；case_design ProductType 废弃）。本工单据此扩展执行：

### 1. qa-acceptance-template.md 重写（用例设计合并）✅
- 新增 **§2 用例设计（覆盖矩阵与口径）**：吸收原用例评审模板的 6 个骨架必填节（用例总览与覆盖矩阵 / 后端自测联调用例口径 / 前端自测联调用例口径 / 联调场景与数据准备 / 回归范围 / 验收用例映射 AC）+ **§2.7 用例设计自检 Checklist**（原评审 checklist 转为设计准出门槛——设计自查通过才进执行）
- 原"测试概述/范围策略/资产/统计/缺陷/AC核对/联调/遗留/结论"各节保留重组（资产§4、AC核对§7、联调§9）
- **骨架标注表更新**：13 必填节（原 6 + 用例设计 7）；**质量门禁更新**：qa/cases.md 引用改为"本文 §2"（case_design 废弃后用例在验收文档内），新增"§2 Checklist 全过才进执行（设计即准出）"
- 正文与骨架已对齐（全部骨架节在正文有对应章节）

### 2. qa-testcase-review-template / demo 移除 ✅
- 用例设计评审节点已去、内容已并入验收文档 → **原文档不再保留**（qa-testcase-review-template.md + demo 已移出 docs/templates，暂存 /tmp 备份）

### 3. qa-defect-management-template.md 正文=骨架同步 ✅
- 新增**「缺陷清单总览」**节（骨架第一必填节——程序路由入口行）
- **「归属判定（domain=rd|fe 与理由）」**显式成节（服务程序路由）
- 「C. 处理与跟踪」改名**「返工与复验记录」**（对齐骨架名）
- B 节标题对齐「缺陷详情（复现/预期/实际）」——正文与骨架 5 节全部对齐

### 4. 移交工单 03 的同步项（避免重复改动）
- gen_flow_rows.py / gen_flow_xlsx.py：case_design ProductType 行、qa-testcase-review-template 模板路径行、「用例设计/用例设计评审」节点行（business L93-94 / tech L108-109）、case-design-cap 动作 how（L195-208）、dev 自测引用 case_design（L218）——**随工单 03 能力×动作回填一并改**（与 respond_to_qa 补建、dev 合并、字段收敛同批）
- capability-review-v2 §0.1 标准产物表 case_design 行、§0.2 case-design-cap——同上移交
- flow-alignment.md 节点表（business A7/A8、tech B6/B7 删除、QA 验收描述改"用例设计+执行验收一体"）——同上移交

**同步规则记录**：正文标题以骨架标注为准；新增节必须同步骨架表；废弃模板移出 docs/templates；ProductType 废弃后其引用（脚本/能力表/流程）随能力回填工单统一清理。