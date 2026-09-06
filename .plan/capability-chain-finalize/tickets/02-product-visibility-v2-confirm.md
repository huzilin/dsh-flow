---
type: grilling
blocked_by: []
claimed_by: huzilin+agent session
claimed_at: 2026-09-02
---

# 产物可见性 v2 待确认点

## Question

docs-tmp/artifact-visibility-rework.md 中遗留的待确认点逐个敲定：

- `capabilities.inputs` 的 node_key 引用：用定义节点标识 `(flow_templates_id, node_order)` 还是新增顶层 id？跨流程复用如何保持成立？
- `use_threads` 是否默认 true？
- `relatedNodes`（其他/上下节点）与 `sessionsByNode`（全需求讨论）的关系与去留。

依据：artifact-visibility-rework.md（§2.4/§2.5 产物模型与推导流程）、my-task-response-model.md（目标态示例）、capability-review-v2.md（一致性校验）。

## Answer（2026-09-02 关闭）

三个遗留点全部敲定：

1. **inputs 引用方式** ✅（既有推演已决，本工单确认）：**ProductType 标准枚举最终形态**——`inputs: [{productType: "rd_design", use_threads: true}]`，不写 node_key/order（能力跨流程复用）；`flow_nodes` 新增 `template_node_id` 关联定义节点；`reference` 同理 productType/query。**问题①（node_key vs 顶层 id）不成立**——两者都被 ProductType 取代。
2. **use_threads 默认值** ✅ 定稿（用户 2026-09-02）：**默认 true**（默认连同产物上的评论增量一起返回；个别动作显式 `false` 省略）。理由：评审/响应场景天然要看评论；按动作类型推导（选项 3）过度设计且行为不一致难测试、新扩展有复杂度。
3. **sessionsByNode 去留** ✅ 定稿（用户 2026-09-02）：**废弃 sessionsByNode**——被最小返回集（工单 04 D3）的 relatedNodes（当前+相关节点，顶层 doc/link/评论平等）取代；不再全量返回全需求每 node 会话（与"只返回具体 action + 相关上下文"方向一致，防上下文爆炸；深排查需求如需要可走按需查询，不随 my_task 返回）。

**联动影响**：my-task-response-model.md 移除 sessionsByNode 表述（现有补充说明 §4 提的旧结构待改）；capability-artifact-analysis §3.3 sessionsByNode 条目标记废弃；inputs/use_threads 语义进 how-spec-v2 字段收敛后的相关定义。