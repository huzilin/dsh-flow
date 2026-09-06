---
type: grilling
blocked_by: []
claimed_by: agent session (2026-09-05, closed 2026-09-07)
claimed_at: 2026-09-05
---

# MVP 数据层 + 工具清单定稿

## Question

**MVP 数据层 schema 调整终稿 + wb_* 工具清单终稿**（供实现直接照做）：

1. **sqlite schema 调整清单**（对照现 11 表）：
   - `flow_nodes` 新增 `template_node_id`（定义节点关联，产物可见性 v2）
   - `flow_template_nodes` / `capabilities`：新增 `inputs`（ProductType+use_threads，默认 true）/ `actions`（字段收敛后形态 {action_id,name,trigger,how,tools,outputProductTypes}）
   - **case_design 相关**：`artifacts.productType` 移除 case_design（用例设计并入验收文档）；`flow_template_nodes` 移除用例设计/用例设计评审节点行
   - **sessionsByNode 移除**：wb_my_task 不再返回全需求会话（最小返回集 relatedNodes 取代）——涉及查询投影调整
   - **tasks 表字段**：TASK（goal/context/action_id/assignee_role/status/last_ack/last_progress——**无 done**，完成由数据层推导）；心跳字段支撑 task_stale 兜底
   - issues 状态机：open/pending-confirm/resolved/overridden（既有）——**"已处理=pending-confirm"语义确认**
2. **wb_* 工具清单终稿**：
   - 现有 22 个逐一核对（有无随字段收敛/节点删除而调整：如 wb_list_artifacts 的 productType 枚举变化）
   - **新增 wb_assign_task**（派发：goal/context/action_id/assignee_role/status=todo）
   - **新增 wb_task_update**（**仅心跳 ack/progress，无 done**）
   - **升级 wb_my_task**（返回 currentTrigger + activeActions + action[最小集：上下文+how 四要素]；身份 three-way 校验）
3. **seed.json 演进路径**：现 capabilities 仅 how 字符串数组 → 目标形态（actions[]/inputs）的迁移步骤。

依据：how-spec-v2.md（字段收敛）、plugin-agent-subagent-chain.md（D1-D5/返回语义）、工单 02（use_threads/sessionsByNode）、工单 03（能力终稿——本工单依赖其产出）、workbench-sql/schema.sql、execution-loop.md §12。

## Answer（2026-09-07 关闭）

产出 = `docs-tmp/data-tools-final.md`（提交 d6cefcf，含后续修正 bc7220f）：

1. **schema v0.29**（对照 11 表逐表 delta）：flow_templates→flow_defs / flow_template_nodes→flow_def_nodes 更名（引用列 flow_defs_id、实例列 def_node_id）；capabilities 移 how 加 inputs/actions；flow_def_nodes 加 template_node_id/node_type(四值)/user_gate/node_actions（激活清单+node action 两类条目）；flow_nodes 加 review_state(7 值)/reopen_source（state 白名单收窄 4 相位）；tasks 加 action_id（completed 仅 plugin 判据校验后置）；artifacts 加 product_type（含 brd，−case_design）；issues 加 escalate；新表 sessions；flows 加 dependencies（依赖树 plugin 内置）。边界：project registry=plugin 全局配置（非 sqlite）；需求/版本 JSON 源不动；幂等声明三条。
2. **wb_* 工具 22→23**：+wb_assign_task / wb_task_update（event:'complete' 相位收口信号+判据校验防谎报）；wb_decide 移出 agent 清单（裁决=plugin 内置轮转+User 拍板 UI→service）；wb_my_task 升级（currentPhase+activeActions+relatedNodes 最小集+three-way）；wb_review 契约=写 reviewers[].verdict（barrier 判据=全评审位 verdict.round==rolled_back_count）；resolve/reopen+escalate。
3. **seed 演进路径**：直接重写（不增量迁移）+ gen_flow_rows.py 按 v3.1 重写。
4. **扫描器参数表**：agent/status 判 running/ended；5min followup 催办；10min user 介入（参数化）。

**验收修正记录**（review 轮）：技术设计节点补内嵌 CR（77f9d96）；respond 收敛 clarify 单挂+一 state 一动作全量 sweep（41f7c06/647a493）；协议复检修正（bc7220f）。**遗留**：无（N4 已落：prd 不内嵌 CR）。**前沿**：工单 08 交互层定稿。
