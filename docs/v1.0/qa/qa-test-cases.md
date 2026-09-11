# wbflow v1.0 全量验收 · 测试用例设计

> 2026-09-10 QA 验收（独立第二双眼睛：既有 e2e 三剧本全绿背景下，User 在并行 session 仍报出 3 个 bug，
> 验收置信度不足——本轮以独立驱动全量复验）。依据：`.plan/wbflow-impl/spec.md`（路由表/注记 A-H）、
> `docs/api/v1/api.proto`（契约真相源）、`docs/v1.0/arch_design.md`、live seed（4 流程 40 定义节点）。
>
> **测试资产**：驱动脚本 `qa/qa_drive.py` + 环境脚本 `qa/qa_env_up.sh` / `qa/qa_env_down.sh`
> （scratch 库 + 真实服务进程一键重建；资产正本固化在仓库根 `qa/`，见其 README）。执行结果见 `qa-execution-report.md`，缺陷登记见 `qa-defects.md`。

## 0. 基本信息

| 项 | 值 |
|---|---|
| 被测对象 | wbflow rd 服务（Go + Gin + MySQL 8.0）· main @ e2b7ada 后工作区 |
| 测试阶段 | 验收（功能全量 + 分支 + 边界 + 幂等 + 对账） |
| 环境 | scratch 库 `wbflow_qa_<ts>`（迁移+seed 后独立演练，不触 wbflow 库）+ 真实服务进程 :18081 |
| 方法 | HTTP 黑盒驱动（演练者扮演全部 agent/User 角色）+ scratch 库只读 SQL 对账（审计/产物矩阵） |
| 用例数 | A 接口专项 30 + B 流程走查 5 剧本 + C 分支 12 + D 对账 2 |

## 1. 用例总览与覆盖矩阵

| 组 | 覆盖 | 用例 | AC 引用 | 重点 |
|---|---|---|---|---|
| A 接口专项 | 28 个 wb_* 端点 + /healthz ×（正常/拦截/幂等/负路径） | A-01~A-30 | AC-V-01（全接口面）；AC-V-09（A-29 TODO 投影） | 每接口至少 1 正 1 负 |
| B 流程走查 | 4 流程定义 40 定义节点全走（business×full 13 / business×rd 9 / tech×full 11 / task×rd 7 / plan 9） | B-1~B-5 | AC-V-02（seed 前置）；AC-V-03（B-1）；AC-V-04（B-2）；AC-V-05（B-4）；AC-V-06（B-1 对齐段）；AC-M-02 后端表态序列（B-1，UI 三态由 fe vitest 覆盖） | 全节点 + 逐节点产物矩阵 |
| C 分支剧本 | 拒绝打回/回退重 review/reproject/澄清/retest/上升/仲裁/reopen 自动上升/join 双序/版本条件解绑/幂等族/通知族 | C-1~C-12 | AC-V-07（C-1/2/5）；AC-V-08（C-6/7）；AC-V-09（C-11）；AC-V-10（C-11 慢路径 skip 标注，真库 IT 补位 rd/service/scanner_it_test.go）；AC-V-11（C-10）；AC-V-12（C-12）；AC-M-01 后端半（C-3 scope 落值/删行不占位，UI 渲染矩阵由 fe vitest 覆盖）；AC-M-03 后端半（C-6，UI 冲突卡由 fe vitest 覆盖）；AC-M-04（C-4/B-1） | spec §4.1 AC-V-01~12 逐条（见 AC 引用列） |
| D 对账 | expected_outputs×artifacts 逐节点矩阵；audit_log 计数对账 | D-1~D-2 | AC-V-02（D-1 产物矩阵）；AC-V-12（D-2 审计） | 库态直查 |
| E 修复回归 | 2026-09-10 批注/修复轮 BUG-001/004/006/008/011/013（E-1a~h）+ dependencyDocs 防御路径（E-2） | E-1~E-2 | 注记 -J1~-J6 回归（注记为设计层条目，不入 §4 AC 编号） | 修复回归 |

## 2. A 接口专项（每接口正常/拦截/幂等）

| 用例 | 接口 | 场景 | 预期 |
|---|---|---|---|
| A-01 | GET /healthz | 探活 | 200，db ok |
| A-02 | 认证中间件 | 无 token / 错 token 打 API | 401 fail-closed；/healthz 豁免 |
| A-03 | wb_create_version / wb_list_versions | 建 v-qa-a；列表含之 | 200；版本名回显 |
| A-04 | wb_update_version | 改名 + updated_at 乐观锁 | 200；旧 updated_at 再改 → 冲突 |
| A-05 | wb_advance_version | 有未完成需求的版本发布 | 200 + `unfinished` 软提示非空 |
| A-06 | wb_deprecate_version | ①其下需求全未开始 ②任一已进入编排 | ①200 + 需求解绑回待分配池 + state=已废弃 ②409 blocked |
| A-07 | wb_create_requirement | category×flow 绑定、scope 落值、实例化矩阵（state 空、序号连续） | 200；flows/flow_nodes 行正确 |
| A-08 | wb_list_requirements | version/priority/status 筛选 + 分页 total | 200；过滤正确 |
| A-09 | wb_get_requirement | 聚合快照（flows/flowNodes/artifacts/issues/sessions/reviewResults） | 字段齐、节点按 nodeOrder |
| A-10 | wb_update_requirement | 改 title/priority | 200；history 出 diff 行 |
| A-11 | wb_list_requirement_history | 注记-D3 只读 | 倒序；field/oldValue/newValue/byWho |
| A-12 | wb_delete_requirement | 删除（confirm）后 GET | 200；删后 404 |
| A-13 | wb_my_task | entry 迎宾（align.opening）/exec 激活动作/评审位摘要/relatedNodes 三向 | 相位×动作×角色一致 |
| A-14 | wb_add_artifact | document/link 正常；product_type 越枚举；kind 越词表 | 200/200；400 拦截 |
| A-15 | wb_get_artifact_content | 真文件回读；href 越出 docs root；文件缺失 | content 非空 / 400 blocked / 404 |
| A-16 | wb_assign_task | 新建 reused=false；同幂等键重放 reused=true 无新行；不可派发 action（align.opening，注记-E） | 200/200 幂等 / 拦截 |
| A-17 | wb_task_update | ack→in-progress、progress 留痕、complete 判据未过 blocked+missing、非法 event | 状态机正确 |
| A-18 | wb_comment | anchor 四级（document/line/range/block）；escalate=true 直升 | issue 行 state=open |
| A-19 | wb_reply | roundCount++；to-fix 态回复 → fixed（注记-D1 路径②） | replies 追加 |
| A-20 | wb_resolve | path=agree open→resolved；path=fix open→to-fix / to-fix→fixed + re_review 通知提评者 | 状态机 + 通知 |
| A-21 | wb_confirm_review | fixed/pending-confirm → resolved | resolve_reviewer_confirmed=1 |
| A-22 | wb_reopen | pending-confirm/fixed → open/to-fix，loop_count++；达阈值自动 escalate | loop_count 递增 |
| A-23 | wb_escalate | 置位 + session 建绑（快照 sessions 出行，注记-D2） | session_id 回显 |
| A-24 | wb_adjudicate | uphold 保持 / override → overridden 终态 | replies 裁决条目回填 |
| A-25 | wb_review | pending→approved/clarify；同轮同角色覆写；非 review 相位 blocked | collect 判定正确 |
| A-26 | wb_decision | approved/rejected；verdict 越集 400；非 gate 相位 blocked | 结论协议落 last_decision |
| A-27 | wb_advance | rotate/cross/wait/noop 四 outcome；exec 判据未过 blocked+missing；entry/respond 相位 blocked | 推进语义正确 |
| A-28 | wb_reproject | 正常重投影/noop 幂等/裁上游 blocked/scope 越界 400 | 注记口径一致 |
| A-29 | wb_list_inbox | notifications 七类投影 + todos 注记-D5 口径 | kind/target_id 正确 |
| A-30 | 依赖树 | 无依赖单需求 | 根节点 found=true，children 空 |

## 3. B 流程走查（真实 flow 全节点）

| 用例 | 剧本 | 节点链 | 内嵌分支 |
|---|---|---|---|
| B-1 | business × full | bus-00→12 全 13 节点 | 澄清循环（bus-03 fe unagree→pm 澄清→fe 复核）；join rd 先完成序（bus-08/09）；QA 验收（bus-10）；用户验收（bus-11） |
| B-2 | business × rd | bus-00/01/02/03/04/05/08/11/12 共 9 节点 | FE/QA 节点不渲染不占位；bus-03 评审位裁剪（rd 保留 fe 裁） |
| B-3 | tech × full | tec-b0→b11 全 11 节点 | join fe 先完成序（tec-b8a/b8b wait→补推）；机械收口（tec-b5 user_gate=0） |
| B-4 | task × rd | tsk-c0/01/02/03/04/06/07 共 7 节点 | align.opening executor 迎宾；无 QA 节点；报障直连开发 |
| B-5 | plan | pln-d0→d8 全 9 节点 | pln-d7 create_reqs（decide-cap）；调研/架构规划产物 |

每剧本断言：节点相位序列、flows.current_node_id 迁移、last_decision 结论协议、
user_gate 分流（agreed 停等 vs 机械 approved）、评审轮 round 语义、收件箱通知族。

## 4. C 分支剧本

| 用例 | 分支 | 步骤与断言 |
|---|---|---|
| C-1 | 拒绝打回 | bus-05 拍板 rejected → 回退 bus-04 exec 重入（reopen_source=downstream-reject、rolled_back_count+1、pending_scope 落值）；bus-05 清零（state 空/review_results=[]/last_decision NULL）；下游节点联动重置；current=bus-04 |
| C-2 | 回退重新 review | C-1 后 rd 重做 rd_design → advance → bus-05 重评（新轮）→ agreed → approved → 迁移恢复 |
| C-3 | reproject 重定范围 | 打回持有 pending_scope 时 reproject full→rd：下游 fe 节点删行不占位、order 重排、flows.scope 落值、标记清；重复调用 noop；裁上游 blocked；scope 越界 400 |
| C-4 | 澄清循环 | unagree→respond 相位 + clarify_task 通知；owner resolve(agree)+respond complete → 重入新轮 + re_review 通知 unagree 位；复核 agree→gate_pending |
| C-5 | QA retest | 缺陷 issue（reviewer_role 落值）→ resolve(fix) → fixed + re_review 通知提评者（target_id=issueId，retest 子型）→ 提评者 confirm → resolved；fixed→fixed 幂等不重发通知 |
| C-6 | 冲突上升 | 手动 escalate → sessions 行 + escalation 通知 → adjudicate uphold（保持/复位）/ override（overridden 终态）双路径 |
| C-7 | reopen 自动上升 | loop_count 1→4 正常 reopen；第 5 次（阈值）自动 escalate=1 + escalation 通知 |
| C-8 | 并行 join 双序 | rd 先 complete（wait 待对侧）→ fe 完成 → 整组推进；fe 先序在 B-3；双侧完成重复 advance 幂等 |
| C-9 | 版本条件解绑联动 | 未开始需求绑定版本 → deprecate 成功解绑回池（version_id NULL + 逐行审计）；需求进入编排后 → deprecate 409 |
| C-10 | 幂等族 | 重复派发/重复表态（覆写取最后）/重复 advance noop/重复 complete/重复 resolve(fixed→fixed) |
| C-11 | 通知族 | gate_pending/clarify_task/re_review（澄清子型 target_id 空 + retest 子型 target_id=issueId）/escalation 投影正确（stall_followup/need_intervene 慢路径标注 skip——扫描器 5/10min 阈值不.waitFor） |
| C-12 | 审计对账 | requirements/flows/flow_nodes/tasks/artifacts INSERT 与步骤计数对账；拍板/回退/推进 UPDATE by_who 归属（user/plugin/system） |

## 5. D 对账（scratch 库只读 SQL）

| 用例 | 内容 |
|---|---|
| D-1 | 产物矩阵：走过节点 `flow_def_nodes.expected_outputs` vs `artifacts` 表逐节点对照（productType/label/kind 全等、无缺登、无多登） |
| D-2 | audit_log：append-only（created_at=updated_at）；关键写路径留痕抽查 |
| E-1 | 修复回归组（2026-09-10 批注/修复轮新增）：BUG-001 status 联动（首次进编排→进行中 E-1a / 落位尾节点→已完成 E-1b）、BUG-004 局部更新 200+显式非法仍拦 400（E-1c/d）、BUG-006 my-task 全名 actionId（E-1e）、BUG-008 wb_escalate 通知行 anchor_kind=escalation DB 实证（E-1f）、BUG-011 评审节点 my-task 带被评审产物置前（E-1g）、BUG-013 评审上下文无锚 issue 评论 400（E-1h） |
| E-2 | dependencyDocs（BUG-011③）：评审表态任务派发注入上游 produce 产物——注：现有 seed 评审表态经 wb_review 不经任务总线，注入为任务侧防御路径，驱动不断言（CR 处置记录） |

## 6. 覆盖自检 Checklist

- [x] 28 wb_* 端点全覆盖（正常+拦截+幂等至少各 1）
- [x] 4 流程定义 40 定义节点全走
- [x] spec §4.1 AC-V-01~12 + §4.2 AC-M-01~04 全数映射（§1「AC 引用」列；AC-V-10=C-11 skip 标注+scanner_it 真库补位；AC-M-01/02/03 的 UI 半边由 fe vitest 覆盖）；机器断言在 qa_drive.py 运行尾（spec 有 AC 而无覆盖声明即 FAIL）
- [x] 注记 A-H 逐条有用例（A-06/C-9=-A；A-27/B*= -B/-C；A-09/C-5=-D1/-D2；A-11=-D3；A-08=-D4；A-29=-D5；A-16=-E/-F；D-2=-G；C-11 通知 outbox=-H）
- [x] 负路径：401/404/400/409 blocked/missing 全覆盖
