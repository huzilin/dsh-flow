# Tickets: wbflow v1.0 重新实现（MySQL + Go/Gin + Web）

把 `.plan/wbflow-impl/spec.md` 拆成 17 张 tracer-bullet 工单：每张切穿 schema→repo→domain→service→handler→(FE)→测试的窄纵切，完成即独立可验证。已拍板：FE 技术栈 React + Vite + TS；T2 seed 以 docs-tmp 定稿（capability-model-final v3.1 / capability-actions-final / flow 推演稿）为口径重制，旧 `workbench-sql/` sqlite seed 仅作参考搬运。

**横切普适约束**（spec 文档约束全量采纳，不单独立票，已织入各工单验收项）：数据模型九条（完整 DDL/双时间戳/自增整型主键/无物理外键/updated_at 乐观锁/两类版本区分/JSON 载荷存 TEXT/默认 NOT NULL/语义命名）+ 接口契约五条（GoDoc/proto 真相源/XxxReq-XxxResp/string 承载结构化载荷）+ gofmt 呈现。spec 两项默认假设（MySQL 8.0 / 幂等键）在 T1/T2 落地时验证。

Work the **frontier**: any ticket whose blockers are all done. 关键路径 1→2→3→4→{5,6,7}→8→{9,10,11,12}→13/17；FE 14 可与 4 并行开工，15、16 随后端会师。

> **实证记录（2026-09-09 收尾）**：17 工单全部并入 `wbflow-impl` @ `6886434`（含 CR R1-R5 复审循环清零）。全闸门实证：`gofmt/build/vet/test` 全绿；集中 IT 复跑（env 门控真库 MySQL 8.0.46）**0 失败**；e2e 三剧本 PASS（full 19.3s / rd-only 107s / task 44.6s）；FE 浏览器冒烟通过（新建需求含类别→流程类型联动、需求池 scope 徽标、详情流程树 13 节点+评审聚合、收件箱七类筛选、看板四泳道）。
>
> **拍板落地增补（2026-09-09，`6bc9454`+`d1c3731`）**：User 拍板 A-E 后（spec §1.2 注记为正本）——A deprecate 条件解绑、B wb_advance 编排推进（27 端点，e2e SQL 模拟全部换真调用）、C 并行 join 三层（对侧检查/flows 行锁/scanner 兜底巡检）、D scope 暂不支持（零改动）、E semantica 补录 11 条+拍板 1 条。增补后复验：本地闸门全绿、真库 IT 0 失败（service 包 583.7s，**复跑需 `-timeout 900s`**）、e2e 三剧本 PASS（18.5s/15.4s/12.8s）、抽轴复核 16 项全 pass（3 条观察级备忘见复核报告）。下方 checkbox 为工单完成时的静态勾选基线，实证结论以本注记为准；逐项勾选待 AC 终核。
> **原型补齐 + 联调实证（2026-09-09 第 2 轮，`914a7dd`..`9a85f57`）**：ui-interaction-final §6 N1-N7 定稿 + 原型 P8-P11 落地——后端配套（spec §1.2 注记-D1..D6：retest 通知双触发+target_id 锚点 / 快照 sessions 投影 / history 端点 27→28 / dependencies 透传，`914a7dd`；CR-R6 发现 9 条 → R7 复审清零，`f7f89c2`+`1caa340`）+ FE 六模块（RequirementModal 编辑复用 / 详情⑥⑦卡+会话找X / 收件箱 TODO 五分类投影 / 看板双视图+✎编辑接线 / 版本四色聚合+依赖树+调度卡 / 评审疑似卡死高亮）。FE CR-R8（硬违规 1+应修 2：CSS merge 残缺括号、依赖树悬挂根渲染缺陷等）→ R9 复审清零（`9a85f57`）。全量闸门：rd gofmt/build/vet/test 绿 + ProtoD 真库 IT PASS + fe 17 文件 117 测试 / lint / build 绿。**浏览器联调（真实库 wbflow + rd:18080 + fe:5173，主线程实测）**：新建需求（N4 scope 三值派生）→ 看板四泳道 → 详情流程树 13 节点 → HTTP 驱动编排（对齐拍板 / 产品设计 / wb_advance 内聚 noop 幂等 / 评审三位 agree→agreed 停等）→ UI 拍板卡迎宾态+通过迁移 → 收件箱 badge+提醒条+gate_pending → retest 链路（resolve fix→fixed → re_review targetId=issueId → 「待复验」分组+focus=issue 锚点跳转）→ 版本页（四色聚合 / 依赖树空态 / 调度卡就绪池推导 / 编辑挂版本）→ 废弃 blocked（「1 条已开始推演的需求（编排已进入），废弃被拦截」——注记-A 真库生效）。观察项 1 条：Dispatch 吞错口径下通知落库偶发静默失败（req-002 gate_pending 一次未落；探针直连+req-003 复现均正常——spec §1.3「失败不阻断」既有拍板口径，建议未来补重试/审计留痕）。

## 1. 服务壳：配置 / MySQL 接入 / 健康检查

**What to build:** rd 服务可启动、可探活——yaml 配置加载、zap 日志、GORM MySQL 连接（PrepareStmt + 连接池 + SkipDefaultTransaction）、/healthz 健康检查、统一错误映射骨架（哨兵错误 → HTTP 出参集中映射，接口永不 panic）、优雅退出。这是后续所有工单的地基（prefactoring）。

**Blocked by:** None — can start immediately.

- [x] MySQL 8.0 假设实测确认（连接 + 版本断言），低于 8.0 提前暴露
- [x] /healthz 返回服务与数据库连通状态
- [x] 哨兵错误骨架（ErrOptimisticLock/ErrGateBlocked/ErrIdempotentConflict/ErrStaleReport）定义并集中映射
- [x] httptest 冒烟：路由注册 + 日志入口/出口埋点生效

## 2. Schema 14 表落库 + seed + repo 基线

**What to build:** spec §0 全量 14 表 DDL 迁移在真实 MySQL 上执行通过；GORM model + updated_at 乐观锁 helper（0 行命中→ErrOptimisticLock）+ TEXT 列 JSON 序列化 util；seed 数据以 docs-tmp 定稿为口径重制：4 条 flow_defs + 40 定义节点 + 21 能力 + roles 全集。旧 `workbench-sql/`（sqlite v0.15，11 表旧表名）不迁移，仅参考。

**Blocked by:** 1

- [x] 14 表 DDL testcontainers 真实执行通过，逐表列注释与 spec §0 一致（testcontainers 不可用，降级 192.168.3.100 目标库真实执行 + 幂等重跑验证）
- [x] seed 装载后：40 定义节点 scope_expr/reviewers/node_actions 与 docs-tmp 定稿逐项对得上
- [x] 乐观锁 helper：并发更新一胜一败（败者收 ErrOptimisticLock）
- [x] append-only 表（audit_log）created_at=updated_at 恒同值由 helper 保证

## 3. proto 契约真相源 + 需求/版本管理纵切

**What to build:** 第一条 API 纵切：`docs/api/v1/api.proto` 工具链落地（protoc 生成 XxxReq/XxxResp Go DTO + protojson 序列化 + 统一响应 ok/error/blocked/missing）+ 简单 token 认证中间件；其上实现 wb_list/get/create/update/delete_requirement 与 wb_create/update_version、wb_advance_version（unfinished 仅提示软约束）、wb_deprecate_version（有引用 blocked）；需求创建按 category 派生流程类型绑定。删除类端点保持「需 User 明确指令」口径。

**Blocked by:** 2

- [x] DTO 全部由 protoc 生成，文档镜像零手写（改 proto 即改契约）
- [x] 枚举态字段 string 承载；结构化载荷 string 包 JSON 文本（与 TEXT 列同构）
- [x] httptest 覆盖 10 端点正常/拦截/幂等分支；需求详情聚合快照基础版可返回
- [x] handler 薄壳 + service GoDoc（做什么/出入参/错误集）+ gofmt 逐字段多行

## 4. 流程实例化 + scope 渲染 + my_task

**What to build:** 需求创建即建流程实例（flows + flow_nodes 按定义节点 scope_expr 过滤重排：full / rd / fe 三种 scope，被裁节点不渲染不占位、node_order 重排）；wb_my_task 三向校验（role × node × 相位）返回 currentPhase / activeActions / review_results 摘要 / relatedNodes；依赖树只读端点（分析师视图，动态加载）。

**Blocked by:** 3

- [x] 三种 scope 实例化矩阵断言：节点集与序号符合渲染矩阵，被裁节点不占位（spec §4.2 流程树 AC）
- [x] business/tech/plan 各流程实例化正确（plan 无 scope、task 无 QA）
- [x] my_task 对不属于当前节点的角色返回空动作集（三向校验拦截）
- [x] httptest 直打路由验证实例化 + my_task 全分支

## 5. 任务总线：派发与上报

**What to build:** 唯一待办总线走通：wb_assign_task 幂等派发（键 = reqId+nodeId+action_id+assignee，同键重派无害）+ wb_task_update（event: ack/progress/complete；complete 触发相位判据校验——不满足返回 {ok:false, blocked:true, missing:[…]} 且任务回 in-progress；last_ack 支撑断点恢复）。

**Blocked by:** 4

- [x] 重复 complete / 重复派发幂等无害（spec §4.1 P2 AC）（真实库 IT 集中验证由编排者执行）
- [x] 谎报完成被判据拦截并回 in-progress，missing 列出缺项
- [x] task 族（produce/dev/review/respond/accept/create_reqs）判据校验表驱动单测
- [x] 写路径单事务 + 行锁；GoDoc 错误集引用哨兵错误

## 6. 产物登记与正文阅读

**What to build:** wb_add_artifact（product_type 受控枚举校验，无 case_design）+ wb_get_artifact_content（document 相对项目路径 / link 外链两类，返回正文与锚点定位所需元数据）。产物是评审与评论的锚定对象，先于评论链路就位。

**Blocked by:** 4

- [x] product_type 枚举外值被拦（blocked + reason）
- [x] 阅读器按 artifact 取正文：文档存在 / 缺失 / link 三分支
- [x] 登记人（added_by）与审计字段正确落库
- [x] httptest 全分支覆盖（IT 已编码，真实库集中验证由编排者执行）

## 7. 评论 / issue 状态机 v2 全链路

**What to build:** 评论闭环五端点：wb_comment（四级锚点 document/line/range/block + escalate?）→ wb_reply（roundCount++）→ wb_resolve（path=fix|agree 双路径）→ wb_confirm_review（提评者复核）→ wb_reopen（loop_count++）。domain 层状态机纯逻辑（open→pending-confirm→resolved / to-fix→fixed→复核→resolved|to-fix）表驱动全分支单测。

**Blocked by:** 4

- [x] 状态机全迁移分支单测：合法迁移 + 非法迁移拦截（无 IO 全覆盖）
- [x] loop_count 双读法正确（澄清期=reopen 次数，修复期=revise 次数）
- [x] reviewer_role 推导与 assignee 指派落库正确
- [x] escalate 标记可写（裁决链在 12 消费）
- [x] httptest 五端点正常/拦截分支（IT 已编码，真实库集中验证由编排者执行）

## 8. 评审表态 + collect 聚合 + 门禁推进 + 拍板

**What to build:** 编排核心会师点：wb_review 写 review_results（agent 自判 agree/unagree，同轮同角色覆写取最后）；domain collect 聚合判定（全部 state≠reviewing 且全 agree → 门禁 approved；有 unagree → owner 澄清循环）；flow_nodes 行锁 + 单事务防并发双 collect；User 终审拍板专用 decision 路由（user_gate 位）；节点推进门禁公式。wb_decide/wb_override 语义保留在 service 层不暴露 agent 端点。

**Blocked by:** 4, 5, 7

- [x] per-agent 表态 AC：review_results 单字段读写一致，reviewing→agree/unagree 判据正确（spec §4.2）
- [x] 并发 collect 竞争测试：行锁下仅一次推进生效
- [x] 门禁 approved 后节点推进、下一节点相位正确激活
- [x] unagree → owner 澄清循环进入 respond 相位（消费 7 的 issue 链）
- [x] 拍板路由仅 user_gate=1 节点可拍；agreed 后唤醒 User 语义正确

## 9. 打回链路

**What to build:** 评审 unagree → to-fix → rejected → exec 重入（revise 语义：rolled_back_count++、reopen_source 记来源、pending_scope 支持重定范围）→ 重新提测 → QA retest 复验 → resolved。打回幂等键 reqId+nodeId+round。

**Blocked by:** 8

- [x] §4.1 打回链路 AC 全链路 httptest 走通
- [x] entry/exec 多配置按 rolled_back_count 选择重入行为
- [x] QA retest 仅 full scope 节点触发
- [x] 回退重复上报幂等无害

## 10. 扫描器（催办 / 介入）+ 会话绑定

**What to build:** 注入时钟的扫描器：绑定 subagent running → 等待；已结束 5min 未表态 → followup 催办（卡点上报 user）；10min → 通知 user 介入（阈值参数化配置）。sessions 表接线：flow↔session 绑定、persona 按节点配置传入、status 生命周期（active/ended/disposed）。本期不接真实 DSH（嵌入适配器 out of scope），以 sessions 句柄占位。

**Blocked by:** 5, 8

- [x] 注入时钟模拟：5min/10min 按时触发（spec §4.1 P2 AC）
- [x] 阈值参数化：配置改小即可加速演练
- [x] 会话绑定读写 + status 迁移正确
- [x] 扫描器恢复续跑（MySQL 不可用恢复后不丢扫）

## 11. 收件箱 + TODO 投影（通知七类）

**What to build:** wb_list_inbox 落地，通知规程 v3 七类在既有写路径（任务派发/上报/评审表态/门禁/issue 迁移）触发点接线；TODO 投影与任务总线一致（同一数据源，不双写）。

**Blocked by:** 5, 7, 8

- [x] 七类通知各自触发条件有 httptest 断言，跳转定位字段正确（spec §4.1 AC）（stall_followup/need_intervene 触发动作在 T10 扫描器已接线，session_message 为 T15+ 预留枚举位）
- [x] TODO 投影一致性：任务状态迁移后投影同步（completed/rejected 不残留）
- [x] 收件箱按角色/已读过滤分页可用

## 12. 冲突上升裁决链

**What to build:** escalate → 拉起裁决 session → 结论回填 issue → reviewer 重表态 → 恢复评审轮；澄清不收敛（loop_count 达阈值，如 reopen≥5）自动标 escalate 转 User 仲裁。

**Blocked by:** 7, 8

- [x] §4.1 冲突上升 AC：escalate→裁决→回填→重表态→恢复全链路 httptest 走通
- [x] 阈值自动 escalate 触发且通知 user
- [x] 裁决 session 落 sessions 表并与节点关联

## 13. 审计全路径收口

**What to build:** 全部写路径同事务落 audit_log（append-only，created_at=updated_at 恒同值；审计失败回滚业务事务，保留痕一致）；对 3–12 全部写路径做覆盖扫描，漏点补齐。

**Blocked by:** 11, 12

- [x] §4.1 审计 AC：全部写路径落 audit_log（覆盖扫描脚本化断言）
- [x] 审计写失败 → 业务事务回滚（同库同事务验证）
- [x] by_who 口径（role_id/user/system/plugin）全路径正确

## 14. FE 骨架 + 模块 I（需求池 / 版本 / 看板）

**What to build:** React + Vite + TS 前端骨架（路由、API client 消费同一 HTTP API、token 接入）+ 模块 I：需求池列表（version/priority/status/scope 筛选 + 分页）、需求卡（scope 徽标）、版本管理（含待分配池）、模块看板。

**Blocked by:** 3

- [x] 需求创建/列表筛选/详情跳转对真实后端可用（API 契约测试通过，浏览器联调待编排者验收）
- [x] 版本创建/推进/废弃软约束提示正确呈现
- [x] FE 与 proto 契约类型对齐（生成或镜像一份类型源，禁手写漂移）

## 15. FE 模块 II：流程树 + 节点详情五区分栏 + 阅读器锚点评论

**What to build:** 流程树 SVG 一次渲染全节点（scope 重排后节点集与序号）+ 节点详情五区分栏 + 产物阅读器（分段渲染大文档）+ 锚点评论（四级锚点定位）+ 拍板卡进度只读态。会话通道按钮占位（本期不接 DSH）。

**Blocked by:** 4, 6, 7, 8

- [x] 三种 scope 切换后流程树节点集/序号与后端实例一致，被裁节点不占位（后端矩阵断言 + FE 双快照锁定，浏览器联调待编排者验收）
- [x] 阅读器内 line/range/block 锚点选取并发起评论，定位正确
- [x] 详情五区分栏数据源全部来自聚合快照端点

## 16. FE 模块 III/IV：收件箱 / TODO + 拍板卡三态 + 冲突卡

**What to build:** 收件箱/TODO 视图（七类通知 + 跳转）+ 拍板卡三态切换（进度只读 → agreed 迎宾呈现：汇总+按钮组+跳转 → approved/rejected）+ 冲突卡（escalate 聚合、正反对照、裁决回填跳转）。

**Blocked by:** 11, 12

- [x] 拍板卡三态切换 AC（spec §4.2）
- [x] 冲突卡 AC：escalate 聚合、正反对照、裁决回填 issue、跳转节点详情
- [x] 收件箱 badge 轮询刷新（MVP 轮询口径）、TODO 投影一致

## 17. E2E 演练脚本（full / rd-only / task 三条）

**What to build:** seed 注入 → HTTP 驱动的全流程演练脚本化，覆盖 §4.1 P5 三条验收：full 端到端（对齐 to-brd → 产品设计 → 三轮评审 → RD∥FE 并行开发（CR 内嵌）→ QA 验收 → 用户验收）；rd-only 重排 9 节点走通；task 7 节点无 QA 走通 + 报障直连开发执行者。打回/escalate 链路复用 9/12 的 httptest 断言不重复建。

**Blocked by:** 9, 10, 11, 13

- [x] full 流程脚本无阻塞跑完（§4.1 P5 主 AC）（2026-09-09 真库 PASS 19.3s）
- [x] rd-only：FE 侧与 QA 不渲染不占位，走通至用户验收（真库 PASS 107s）
- [x] task：走通至用户验收，报障直连开发执行者（真库 PASS 44.6s）
- [x] 脚本可重复执行（重置 seed → 重跑）（scratch 库每次新建天然可重复，两轮实证）
