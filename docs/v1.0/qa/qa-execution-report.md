# wbflow v1.0 全量验收 · 执行验收报告

> 2026-09-10。用例设计见 [qa-test-cases.md](qa-test-cases.md)，缺陷台账见 [qa-defects.md](qa-defects.md)（与并行 session 核查轮共用台账，本轮新增 BUG-003~008 + OBS-001）。
> 测试资产（固化于仓库根 `qa/`，见 [qa/README.md](../../qa/README.md)）：`qa_drive.py`（一键复跑）+ `qa_env_up.sh` / `qa_env_down.sh`（scratch 库 + 真实服务进程一键重建）+ `qa-results.json`（末轮逐用例结果）。

## 0. 基本信息

| 项 | 值 |
|---|---|
| 被测对象 | wbflow rd 服务（Go + Gin + MySQL 8.0）· main @ e2b7ada 后工作区 |
| 测试阶段 | 验收（接口全量 + 4 流程 40 节点走查 + 分支 + 幂等 + 对账） |
| 环境 | scratch 库 `wbflow_qa_<ts>`（迁移 v4 + seed 后独立演练）+ 真实服务进程 :18081（5 轮执行，末轮为准） |
| 执行方式 | HTTP 黑盒驱动（演练者扮演全部 agent/User 角色）+ scratch 库只读 SQL 对账 |
| 报告日期 | 2026-09-10 |

## 1. 测试概述与结论

本轮为**独立第二双眼睛**验收：既有 e2e 三剧本（full/rdonly/task）全绿的背景下，以独立驱动（不复用 e2e 断言）覆盖全部 28 个 wb_* 端点 + 4 流程定义 40 定义节点全走 + 打回/回退/reproject/澄清/retest/上升/仲裁/join/版本条件解绑/幂等族/通知族分支 + 产物矩阵与审计对账。

末轮结果：**PASS 195 / FAIL 0 / SKIP 1**（100.4s，含批注轮修复回归 E-1 组 9 断言；前基线 188 PASS）；含 2026-09-10 拍板修复回归——BUG-003/005/009 修复落地，OBS-001 按最小原则拍板关闭，详见 §5）。唯一 FAIL = BUG-003（plan 流程尾段死锁，稳定复现）；SKIP = BUG-007（wait 分支 API 不可达，附因跳过）。过程中另捕获 BUG-004/005/006/008 与观察项 OBS-001；与并行 session 核查轮的 BUG-001（需求状态零联动）/BUG-002（文档同步）在 [qa-defects.md](qa-defects.md) 统一台账管理，不重复计数。

**验收结论**：**不通过（阻塞项 1 个 S1）**——business×full / business×rd / tech×full / task×rd 四条链全绿 + 接口面全绿，但 plan 流程无法走完（BUG-003），spec §4.1「plan 需求走通至用户验收」AC 不达成。

- **遗留问题**：S1×1（BUG-003 plan 死锁）、S3×5（BUG-004~008）、观察×1（OBS-001 待拍板）、并行轮 S2×1（BUG-001 状态联动）+ S3×1（BUG-002 文档）。

## 2. 覆盖与执行结果

| 组 | 覆盖 | 结果 |
|---|---|---|
| A 接口专项 | 28 wb_* 端点 + healthz：正常/拦截（401/404/400/409 blocked+missing）/幂等/乐观锁/路径穿越 | 全 PASS |
| B 流程走查 | business×full 13 节点（含澄清循环+join 序A+QA 验收）、business×rd 9、tech×full 11（join）、task×rd 7（retest）、plan 9 | 4 链全绿；plan 卡 pln-d7（BUG-003） |
| C 分支 | 拒绝打回（回退重入/下游联动/审计）、回退重 review、reproject（含裁剪/越界/noop）、澄清循环、retest（-D1 双路径+幂等）、escalate/adjudicate 双路径、reopen×5 自动上升、join 双序、版本条件解绑双态、幂等族、通知族 | 全 PASS（C-8b SKIP=BUG-007） |
| D 对账 | 产物矩阵（expected_outputs×artifacts 逐节点 productType/label/kind 全等）6 需求全过；audit append-only；列表/库一致 | 全 PASS |

过程有效性说明：4 轮试跑共修正驱动侧问题（docs root 解析、anchor 契约嵌套+artifact_id 必填、confirm/reopen operator 须提评者、update 全量字段适配），**产品侧缺陷均以稳定复现 + DB 证据入账**，非环境噪声。

## 3. 缺陷汇总（详见 qa-defects.md）

| 编号 | 一句话 | 级别 | 状态 |
|---|---|---|---|
| BUG-003 | plan 流程 pln-d7 死锁（create_reqs 不被 exec 判据承认） | S1/P0 | 已修复（复测 PASS） |
| BUG-004 | wb_update_requirement 强制全量字段 | S3/P2 | 新建 |
| BUG-005 | 任务总线不校验节点相位（未激活节点可派发完成） | S3/P2 | 已修复（复测 PASS） |
| BUG-006 | my-task 与 assign 动作标识口径不一 | S3/P2 | 新建 |
| BUG-007 | join wait 分支 API 不可达（死代码） | S3/P3 | 新建 |
| BUG-008 | wb_escalate 通知 kind 投影断链（anchor_kind=document） | S3/P2 | 新建 |
| OBS-001 | gate 拍板不校验 exec 判据 | 观察 | 已拍板关闭（最小原则，非缺陷） |
| BUG-009 | reproject 幽灵评审位（与 scanner 巡检竞速） | S2/P1 | 已修复（复测 PASS） |
| BUG-001 | 需求状态与流程推进零联动（并行核查轮） | S2/P1 | 已确认 |
| BUG-002 | arch_design.md 未同步注记-A（并行核查轮） | S3/P2 | 新建 |

## 4. 各阶段产物符合性结论

- 6 条真实 flow（2×business full、business rd、tech full、task rd、plan）全部节点期望产物（productType/label/kind）与 flow_def_nodes.expected_outputs **逐节点全等**，document 产物内容回读含验证 marker，code_mr 为 link 类外链——产物矩阵维度**符合预期**。
- 通知族七类中 gate_pending / clarify_task / re_review（澄清子型 target_id 空 + retest 子型 target_id=issueId）/ escalation 四类触发与投影实测正确；stall_followup / need_intervene 依赖扫描器 5/10min 慢路径，本轮标注 skip（既有 scanner IT 以注入时钟覆盖）；session_message 为预留枚举位。
- 审计：append-only（created_at=updated_at 全表一致）、写路径留痕抽查通过。

## 5. 回归指引（复测口径）

```bash
cd qa
QA_MYSQL_PASSWORD='<密码>' bash qa_env_up.sh          # scratch 库 + 服务 :18081
source /tmp/wbflow-qa-env.env && python3 qa_drive.py  # 末轮基线 186 PASS / 1 FAIL
bash qa_env_down.sh                                   # 拆环境（DROP scratch 库）
```

- BUG-003/005/009 修复回归判据已达成：**195 PASS / 0 FAIL / 1 SKIP**（B-5g 翻绿；BUG-005a/C-2d 验证；批注轮修复 E-1a~h 九断言新增并通过）。
- BUG-004/005/006/008 修复后分别在 A-10 / B-3 / A-13c / C-6c 补正向断言（当前以「缺陷证据」形态记录）。
- e2e 剧本建议补 plan 流程剧本 + status 联动断言（BUG-001 复测口径）。

## 6. 呈现通道冒烟记录（2026-09-10 深夜，test-design-gate 双通道红线落地）

> 环境：rd :18080 **重启至当前工作区代码**（含 -J1~J6/-K 全部修复；旧进程为修复前 go-build 缓存二进制）
> + FE dev :5174（vite，WBFLOW_API_PORT=18080）。存量数据回填：req-002/003/004 按 -J1①口径 status→进行中（audit by_who=user）。

| # | 测试点 | 对应缺陷 | 结果 | 证据 |
|---|---|---|---|---|
| T0 | 看板四泳道分类（req-002/003/004=进行中，req-001=待启动） | BUG-001 呈现面 | **PASS** | 截图 call_dba5ce…（进行中泳道 3 条） |
| T1 | 详情页两栏布局（左流程导航常驻+右节点详情） | OPT-001 | **PASS** | 截图 call_4e335ce… |
| T1b | 描述卡位于头部与流程树之间、有描述默认展开、Markdown 渲染（标题/粗体/列表/行内代码） | BUG-012 | **PASS** | 截图 call_1af5eb1…（req-004 描述经 wb_update_requirement -J2 局部更新写入） |
| T2 | 评审节点「被评审产物（上游）」置前展示；点开《RD 技术设计》→ 阅读器打开入库真文档+行号列；行点击产生 in-selection 选中态（锚定评论前提） | BUG-011 / **BUG-015** | **PASS** | DOM 快照（阅读器全文+行号）；截图 call_fd98cdf…（关闭阅读器后全貌） |
| T3 | 评审药丸点击展开该角色意见区（arch 药丸选中高亮+意见面板含「评审上下文必须挂产物」引导） | OPT-002 / BUG-013 呈现面 | **PASS** | 截图 call_fd98cdf… |
| T4 | 流程树点色语义（绿=已通过 / 蓝=exec / **琥珀=待拍板**（节点 6 当前）/ 灰=未进入） | BUG-010 | **PASS** | 同上截图左列 |

**观察项**：OPT-005（blockquote 字面显示，轻量渲染器支持清单外）——已登记台账。
**过程有效性**：测试中途发现 rd :18080 跑旧二进制（req-004 泳道错位为旧代码+存量数据所致，非新缺陷）→ 按环境准备流程重启+回填后重测；IAB 渲染进程一度卡死（截图空白/点击超时），重开标签页恢复。
**遗留**：OPT-004（leader 裁定动作，CR 拆单待排）、OPT-005（blockquote）。

## 2026-09-16 架构增量 QA 轮（C1-C5 架构批次验收 + Q 组新增）

**结论：有条件通过 → 修复后通过。** 基线复跑 319 全绿（架构重构行为保持矩阵）→ 新增 Q 组（架构行为增量 20 断言）首跑 6 FAIL → BUG-016（S2/P1，SweepLLMTasks 相位闸漏 entry，align 宿主 to-brd 永不自动派发；IT 造数置 exec 掩盖真实链）修复 + 用例侧 5 处预期错当轮修正 → 末轮 **PASS 339 / FAIL 0 / SKIP 1**（137s）。

- **新基线**：339 PASS / 0 FAIL / 1 SKIP（qa-results.json；SKIP=BUG-007 附因）。
- **缺陷**：BUG-016（rd，已修复复测 PASS，最小复现入口见台账 E 节）；用例侧预期错 5 处（align 相位/拍板步/artifacts 端点/取消步骤/healthz 字段）+ 软失败纪律违例（PHASE-CRASH 断链）当轮重写独立化。
- **显式降险**：0-findings 评审分支黑盒不可达（FakeClient 文本不可配）——rd/llm 单测 + llmflow IT 结构化样例锁定；同值审计语义黑盒不可达（reproject 幂等拦截）——rd/repo change_test 锁定。
- **双通道**：协议通道全量（本报告）；呈现通道=FE 重构面由 vitest 185 条承载（latestSeatMap/inboxview/lib-time 收敛，纯视图函数无 DOM 变更，浏览器走查并入下轮 UI 冒烟）。
- **AC 覆盖**：AC-V-01~12 全数保持 PASS（末轮输出 COVER 段）；新增 Q 组不涉新 AC（架构批次无新需求）。
