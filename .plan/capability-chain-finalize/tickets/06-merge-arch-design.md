---
type: grilling
blocked_by: []
claimed_by:
claimed_at:
---

# 合并回 arch_design 策略

## Ruled out

2026-09-02 用户目标重定义：**本工单划为 Out of scope**——用户明确**不支持将能力推演合并到主文档（arch_design.md）**：

- 推演产出收敛于 **docs-tmp 定稿层**（MVP 设计确定下来，供实现阶段直接照做）；
- arch_design.md 保持现状基线；合并/同步（本章节落位、seed/脚本/Excel 一致性核对）是未来**独立 effort** 的事（如有需要再重新开图）；
- 原 Question（合并顺序/§8.x 落位/收敛标准）全部随此作废。

## Question（原）

docs-tmp 各稿如何按 arch_design 的章节结构合并回 docs/v1.0/arch_design.md：

- 合并顺序与 §8.x 章节落位（§8.7「能力 × 产物 · 逐能力分析」或独立附录）;
- 合并后 seed.json / gen_flow_rows.py / gen_flow_xlsx.py / Excel 链路表的一致性核对与同步修正；
- docs-tmp 与 docs 收敛的标准（何时算"合并完成"）。

依据：docs/v1.0/arch_design.md（权威基线）、docs-tmp/flow-alignment.md（三层文档关系）、capability-artifact-analysis.md（§8.7 或附录提议）。