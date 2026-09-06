# how 编写规范 v2 逐条定夺

> type: grilling · blocked by: how 编写规范对照分析（writing-for-agents × 既有五段式）· claimed_by: huzilin+agent session · claimed_at: 2026-08-27

## Question

基于对照分析结论（.wayfinder/research/how-writing-comparison.md），**逐条定夺** how 编写规范 v2：保留五段式哪些环节、叠加哪些 writing-for-agents 纪律、冲突点怎么裁；产出 v2 草案供 docs-tmp 推演使用（用户 review 部分节点批注 → 举一反三全量 how）。

## 定夺进度（diary，逐条记录）

- **第 4 条（动作级披露）** ✅ 定稿（2026-08-27）：wb_my_task 的信息之前已敲定，包含要执行 action 的 how 文本——披露由**程序返回层**行为实现（返回仅含当前动作），how 文本不自做披露控制。
  - **补充（behaviors→how 统一）** ✅：my-task-response-model.md 顶层 `behaviors` 字段并入 `action.how`（行为列表即 how 前身，统一术语）；§actions 已重构为**最小返回集**（单个 `action` 对象 + `currentTrigger`/`activeActions` 审计结，不返回其他动作）。已落盘 docs-tmp/my-task-response-model.md。
- **第 10 条（执行模型）** ✅ 定稿（2026-08-27）：**只返回具体 action**（最小返回集）——纠正此前「返回全部 actions + 各 how」的说法；wb_my_task 每次返回 = 当前上下文 + currentTrigger + activeActions + **仅当前动作完整 how**（其他动作连骨架也不返回）；防 rush/信息最小化由程序返回层完成 ⇢ 已同步修正链路稿 D3 与主推演票 04、my-task-response-model.md §action。
- **第 13 条（单一真相源）** ✅ 定稿（2026-08-27）：是——how 唯一真相源 = `capabilities.actions[].how`；程序按决策树从该源取 how（链路环节⑤），真相源结构性唯一是程序侧刚性要求；推演稿/表格不复制 how 全文（A1 表现存重复待清理）。
- **第 1 条（五段式骨架）** ✅ 定稿（2026-09-02）：**五段式废弃，改为四要素骨架 + 工具知识独立沉淀**：
  - **目标**：要产出什么 + 达成标准（完成程度，可量化，如"xx 覆盖率达到 xx%"）
  - **依赖**：上下文已在 wb_my_task 返回中——这里写**具体看 mytask 中的什么**（动态，依流程类型/阶段）：如 RD 技术设计阶段，业务需求 flow → 看产品文档；task 类 → 看需求对齐；需求对齐后也可执行 to-spec 类固化成文档
  - **执行**：MVP 不扩展多形态——**先基于 skill 执行**（产出型走产出 skill，见 produce-form-skill-derivation.md）
  - **自检**：提交前自查——达成目标 + 按执行要求 + 质量达标
  - **工具沉淀**：登记/评论/响应的接口使用方式**固化为独立 skill**（wb 工具用法固化，不写进每个 how）
  - 原五段式中的「读产物」→ 依赖（动态）；「按模板」→ 执行（skill 化）；「登记/提评/响应」→ 工具知识沉淀 skill；「输出」→ 与执行合并废弃。
- **第 2 条（when_to_use）** ✅ 定稿（2026-09-02）：**字段废弃**——when→action 决策树（trigger 程序契约）唯一承载 when 语义；when_to_use 文本不参与程序决策、subagent 也不读（程序已选动作）；人读价值被 trigger 枚举定义表覆盖（重复副本，违反单一真相源）→ 从 capabilities.actions[] 中**移除 when_to_use 字段**（capability-review-v2 §0.2 / my-task-response-model 同步）。
- **第 3 条（trigger 枚举封闭）** ✅ 定稿（2026-09-02）：接受——trigger 即程序化 branch（one trigger per branch），不许自造。
- **第 7 条（完成标准句）** ✅ 定稿（2026-09-02 修正版）：how 内保留**自检/完成标准句**（达成目标 + 按执行要求 + 质量达标，给 subagent 自己判断做完）；**不与程序 done 信号配对**（how 只写给 subagent，程序状态推进是另一层，不交叉——先前"完成标准 ↔ done 信号"配对说法作废）。**状态推进判定按阶段各有规则**（程序状态机侧）：
  - 节点内操作（如响应 CR）：**全部 open 已 resolved**
  - 文档产出后进入评审：**open → 已处理（processed：修订/回复完成，提交复核）而非 resolved**——resolved 是评审者确认后才达成（resolve→confirm 双路径，flow-alignment §8）
  - 评审表态：所有 reviewer 完成表态；产物：所有应提交产物已提交
- **第 5 条（字段分工注释）** ✅ 定稿（2026-09-02）：**忽略**——reference/focus 不做方案级字段分工（不增加复杂度）；"取哪些产物/盯哪些章节"的逻辑**在 skill 内部实现**（执行 skill 自带），how 字段层不承担。
- **第 6 条（how 长度纪律）** ✅ 定稿（2026-09-02）：**采纳**——how 保持短；**首选手段=执行挂 skill**（执行要素挂到 skill 上，降低 how 复杂度），而非下沉 reference/模板——与第 5 条同向：**how 薄、skill 厚**。
- **第 8 条（拆分标准）** ✅ 定稿（2026-09-02）：**已拆**——从一个 capability 的大 how 拆成多个 action 的 how（trigger/action 层级已存在）；**动作必须是可确定的**（每个 action 边界确定、触发确定）。
- **第 11 条（leading words 登记）** ✅ 定稿（2026-09-02）：**采纳**，参考 domain-modeling 做法（术语/常用语义词表 + 一句话释义 + 禁同义换词）。
- **第 12 条（禁令审计）** ✅ 定稿（2026-09-02）：**采纳**——可基于 writing-for-agents 演进（该纪律本源自 writing-for-agents）；满足要求直接使用。
- **第 14 条（路径真相源 A/B）** ✅ 定稿（2026-09-02）：**选 A（现状，how 内联双写）**——本期先简单来：不写 skill 时**暂不支持"只写《标准名》由 host 查表渲染"的 template 机制**；未来不够用再扩展（B 延后）。
- **第 15 条（no-op 审计）** ✅ 定稿（2026-09-02）：**在 skill 中实现，how 中不考虑**——how 不承担 no-op 审计；"如何读上一步产物"参考 implement-spec 类实现（skill 内部处理）。
- **第 16 条（query 属性过滤）** ✅ 定稿（2026-09-02）：**在 skill 中实现**——`reference.query` 不放在 action 字段层；程序返回层预过滤不采用（编排过死、不利后续扩展）；"按属性找评论/对话类产物"的知识进执行 skill 内部（各 skill 自含查询条件）。⇢ **票 05 全部 16 条已定稿**。

## Answer（2026-09-02 关闭）

**how 编写规范 v2 定稿**，落盘 `docs-tmp/how-spec-v2.md`：
- 骨架：**四要素**（目标/依赖[动态看 mytask]/执行[基于 skill+编排]/自检）+ 工具知识独立 skill；原五段式废弃（读产物→依赖、按模板→执行 skill 化、登记→工具 skill、输出→与执行合并）
- 动作字段收敛：`{action_id, name, trigger, how, tools, outputProductTypes}`——**when_to_use/reference/focus/query 全部移除**（语义分别由 trigger 决策树、skill 内部实现承载）
- 关键原则：**how 只给 subagent 看，程序状态推进独立**（节点内=全 resolved；文档进评审=open→已处理 processed 而非 resolved；评审=全员表态；产物=应提交齐）
- 16 条裁定记录见上；第 9 条消解（dev 合并）、第 14 条选 A（内联双写，本期不支持查表渲染）、第 5/15/16 条忽略/进 skill
- 派生实现任务：my-task-response-model / capability-review-v2 §0.2 同步字段、受控词表登记、to-xxx 产出 skill 创建
- 后续：wayside 子地图（技能评估）全部 6 张工单闭合；skills-evaluation.md 评估稿已落；产出形态推演稿已落