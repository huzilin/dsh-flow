---
type: grilling
blocked_by: [03, 07, 08]
claimed_by: agent session (2026-09-08, closed 2026-09-08)
claimed_at: 2026-09-08
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

## Answer（2026-09-08 关闭——MVP 设计定稿）

**1. roadmap 对照剪裁**（逐项，结论已落 roadmap.md 本体）：

| 项 | 剪裁结论 |
|---|---|
| P1 preset 接线 | 保留+修订：subagent 不需要预设（persona 由 wbflow 节点配置传入，直启实测）；preset 只读治理 |
| P2 sqlite 迁移 | 保留+修订：schema 基线=v0.29（data-tools-final） |
| P3 角色卡 | 保留+修订：persona 由节点配置传入；评审维度走对象化评审 skill |
| P4 Skill 落地 | 修订：**19 个待建 skill**（capability-actions-final §3）取代 wb-agent-guide 单编排 skill |
| P5 真实 agent 执行 | 保留（req-019 演练通过 + subagent 程序化直启实测） |
| P6 对齐节点实践 | 修订：exec=to-brd 产出 brd（不再落 issues.dissussion） |
| v1.1 E2/E3 | **部分前置**：project registry + per-project workbench 设计已定稿（Q6-Q8） |
| v1.1 E8/E9/E10 | **新增**：冲突对峙增强 / 依赖调度器 / 评审轮完整视图 UI（均 MVP 暂缓项提升为 v1.1 显式条目） |

**2. 文档同步核对**（增量已完成，本工单核对）：五件套（model v3.1/actions v3.1/business+三流程 v3.1/data-tools v0.29/ui-interaction）全部同步；10 份历史文档时效横幅+追加废弃指针；CONTEXT 4 词条修正；roadmap 本体本轮剪裁落盘。

**3. 术语统一核查**：CONTEXT 修正 4 词条（触发器→相位 on 4 值 / how 降级 / 动作例与挂载 / qa 职责 retest 链）+ BRD/to-xxx 既有 ✓；残留过时词全项目核验零命中（collecting/barrier/be/'-态外旧词）。

**4. MVP 设计定稿宣布**：**wbflow v1.0 MVP 设计定稿**——权威五件套：capability-model-final v3.1（两层属性状态机+评审轮小状态机）/ business + flow-derivation-tech-task-plan v3.1（四流程节点配置）/ capability-actions-final v3.1（21 活跃能力+19 待建 skill）/ data-tools-final（schema v0.29+22 工具）/ ui-interaction-final（交互增补）。进入实现阶段（工单地图关闭；实现期事项见地图 Out of scope 区与 roadmap v1.1）。