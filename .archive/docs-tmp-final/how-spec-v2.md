# how 编写规范 v2（已降级为 skill 写作参考）

> **已归档（2026-09-12 MIG-07）**：本文件为推演底稿，现行设计权威=docs/v1.0/arch_design.md

> **状态更新（2026-09-05，用户裁决）**：**降级为 skill 写作参考**——动作不再携带 how 四要素（框架不强制），动作 = skill 挂载 / 一句话内联（见 capability-model-final.md §0）。本文件的骨架与门禁要求转为 **skill 作者的推荐结构**（skill 内部自带目标/自检/门禁），运行时框架不再引用。
> **状态**：定稿（2026-09-02）。由 wayfinder 工单 05（how 编写规范 v2 逐条定夺）产出；16 条草案全部裁定完毕。
> **一句话**：how 是**给 subagent 看的执行文档**（不是给程序看的）——四要素骨架 + 程序/编排/skill 分层清楚；原则 = **how 薄、skill 厚、程序确定性**。
> 关联：能力模型（capability.actions[]）、执行链路（plugin-agent-subagent-chain.md）、产出 skill 家族（produce-form-skill-derivation.md / to-xxx）。

---

## 1. 骨架：四要素（替代原五段式）

```
how = {
  目标：要产出什么 + 达成标准（完成程度，可量化，如"xx 覆盖率达到 xx%"）
  依赖：写"具体看 mytask 返回中的什么"——动态按流程类型/阶段
       （业务 flow → 看产品文档；task 类 → 看需求对齐；需求对齐后可执行 to-prd 固化成文档）
  执行：怎么做——MVP 基于 skill 执行（执行要素可挂编排序列，如 [to-tickets, implement-techdesign]）
  自检：提交前——达成目标 + 按执行要求 + 质量达标
}
工具交互（登记/评论/响应）＝ 独立沉淀 skill（wb 接口用法固化，不写进每个 how）
```

- **原五段式去向**：读产物→依赖（动态）；按模板→执行（skill 化）；登记/提评/响应→工具知识独立 skill；输出→与执行合并废弃
- **how 只给 subagent 看**：程序状态推进是另一层，不与 how 配对（见 §4）

## 2. 动作模型字段（终态）

```
capabilities.actions[] = [
  {
    action_id,          // 唯一标识（capability.action，如 backend-dev.dev）
    name,               // 中文名
    trigger,            // ★ 程序契约枚举（node_enter / dev_complete / reviewer_feedback(open)
                        //   / reject_rolled_back / decide / node_review_pending / issue_replied）
    how,                // ★ 四要素文本（目标/依赖/执行/自检）
    tools,              // 可用工具清单（wb_* + 通用）
    outputProductTypes  // 输出产物类型（与 ProductType 闭环）
  }
]
```

**已移除字段**：
- ❌ `when_to_use`——废弃（when→action 由 trigger 决策树唯一承载；人读价值被 trigger 定义表覆盖；重复副本）
- ❌ `reference` / `focus`——不留在字段层（"看哪些产物/盯哪些章节"进 skill 内部实现）
- ❌ `reference.query`——进 skill 内部（按属性找评论的知识归各执行 skill）

## 3. 逐条定稿记录（16 条）

| # | 条款 | 定稿 |
|---|---|---|
| 1 | 五段式骨架 | ✅ **废弃，改四要素骨架 + 工具知识独立 skill**；执行要素可挂编排序列（受控枚举） |
| 2 | when_to_use 措辞纪律 | ✅ **字段废弃**——when 语义由 trigger 决策树唯一承载 |
| 3 | trigger 枚举封闭 | ✅ 接受——one trigger per branch，不许自造 |
| 4 | 动作级披露 | ✅ 程序返回层行为（wb_my_task 最小返回集，只含当前动作） |
| 5 | 字段分工（reference/focus） | ✅ **忽略**——进 skill 内部实现，不增加字段复杂度 |
| 6 | how 长度纪律 | ✅ 采纳——**优先执行挂 skill**（how 薄 skill 厚）；超长即以 skill 承接 |
| 7 | 完成标准句 | ✅ 采纳（修正版：how 内自检口径；**不与程序 done 信号配对**；程序推进按阶段规则，见 §4） |
| 8 | 拆分标准 | ✅ **已拆**（trigger/action 层级存在）；动作必须可确定 |
| 9 | 同 trigger 多动作 | ✅ 消解（dev+self_test 已合并，dev_complete 只剩 submit） |
| 10 | 执行模型 | ✅ 最小返回集（程序按 who+when 返回当前动作 how；防 rush 由程序层完成） |
| 11 | leading words 登记 | ✅ 采纳——参考 domain-modeling（受控词表 + 一句话释义 + 禁同义换词） |
| 12 | 禁令审计 | ✅ 采纳——基于 writing-for-agents 演进（满足要求直接使用） |
| 13 | 单一真相源 | ✅ how 唯一真相源 = `capabilities.actions[].how`；程序经决策树从此取；推演稿/表格不复制 how 全文（A1 表待清理） |
| 14 | 路径真相源 | ✅ **选 A（现状，how 内联双写）**——本期不写 skill 时暂不支持"只写《标准名》由 host 查表渲染"；未来扩展 |
| 15 | no-op 审计 | ✅ 在 skill 中实现，how 中不考虑（如何读上一步产物参考 implement-spec 类实现） |
| 16 | query 属性过滤 | ✅ 在 skill 中实现——程序返回层预过滤不采用（编排过死、不利扩展） |

## 4. 程序状态推进（独立于 how，按阶段各有规则）

程序（wbflow 插件）推进节点状态机的判定——**不是 from how 的"done"**，按阶段：

| 场景 | 推进条件 |
|---|---|
| 节点内操作（如响应 CR） | 全部 open 问题 **resolved**（节点内闭环） |
| 文档产出后进入评审 | open 意见 **已处理（processed：修订/回复完成，提交复核）而非 resolved**——resolved 是评审者确认后达成（resolve→confirm 双路径） |
| 评审表态 | 所有 reviewer 完成表态 |
| 产物 | 所有应提交产物已提交 |

**关键区分**：`已处理（processed）`= 提交者处理完（修订/回复，提交复核）；`resolved` = 评审者确认后达成。文档类产出的完成标准 = "open 意见已处理并提交复核"，不是"全 resolved"。

## 5. 与执行链路/产出 skill 的关系

- **执行链路**（plugin→主 agent→subagent）：wb_my_task 返回最小集（currentTrigger + activeActions + 当前动作完整 how）；how 四要素在返回中呈现
- **开发链路编排**（代码类）：`to-prd → to-techdesign → [to-tickets + /goal] → [implement-techdesign] → tdd/implement`（执行要素=编排序列形态）
- **产出 skill 家族**：to-prd / to-techdesign / to-rd-techdesign 等（Process+Template 形态；skill 内部含参考/focus 逻辑、query 知识、no-op 审计）
- **qa 领域模型**：qa 评审维度/缺陷判定准则沉淀为 qa 专用 skill（参与评审带最小上下文）

## 6. 待实现期任务（派生）

- my-task-response-model.md 的 action 示例按新字段（去 when_to_use/reference/focus，how 四要素）重写
- capability-review-v2 §0.2 样例同步（字段收敛）
- 受控词表（leading words）首版登记（契约/各审各的/章节名/对照表等）
- to-xxx 产出 skill 逐个创建（Process 内实现参考/query/no-op 逻辑）
- 禁令审计基于 writing-for-agents 校准现有 how