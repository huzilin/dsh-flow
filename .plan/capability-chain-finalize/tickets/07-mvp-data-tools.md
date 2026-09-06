---
type: grilling
blocked_by: []
claimed_by: agent session (2026-09-05)
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
