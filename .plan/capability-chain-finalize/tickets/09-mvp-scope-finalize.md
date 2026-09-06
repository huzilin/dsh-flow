---
type: grilling
blocked_by: [03, 07, 08]
claimed_by:
claimed_at:
---

# MVP 范围确认 + 定稿收尾

## Question

**MVP 范围边界确认 + 全图收尾**（本图最后一张工单，完成后地图关闭、进入实现阶段）：

1. **roadmap 对照剪裁**：docs/v1.0/roadmap.md 的 v1.0 MVP 六项 + v1.1 增强七项，对照本轮全部新裁决（qa 重定义/用例设计合并/编排链路/数据层即返回/to-xxx 家族/交互层调整）逐项确认：MVP 边界是否变化（新增/移出/降级）。
2. **文档同步**（推演结论落定后的定稿同步）：
   - my-task-response-model.md：按字段收敛+最小返回集+use_threads/sessionsByNode 重写
   - capability-review-v2.md：§0.1（case_design 删除）/§0.2（respond_to_qa 补建、dev 合并、字段收敛）
   - flow-alignment.md：节点表终稿（工单 03 产出）
   - execution-loop.md：数据层即返回改写（四类返回表→数据层变更模型）、D1-D5 编入
   - gen_flow_rows.py / gen_flow_xlsx.py：case_design/用例设计节点清理 + respond_to_qa/dev 合并行（工单 05 移交清单）
3. **术语统一**（最后统一执行）：CONTEXT.md 术语 ↔ docs/README.md 术语表 ↔ 既有文档对齐（用户 2026-09-02：留到最后）。
4. **推演完成定义确认**：全部 docs-tmp 定稿层文档一致性核对；宣布 MVP 设计定稿、进入实现阶段（插件编码/team preset 补齐/to-xxx 创建/wb_* 实装——实现阶段 Out of scope）。

依据：本图全部已关闭工单（01-05 + 06 ruled out）的 Answer、roadmap.md、CONTEXT.md。
