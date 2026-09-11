# wbflow v1.0 · 缺陷（Bug）管理台账

> 2026-09-10 建档（req-003 状态矛盾批注核查轮）。载体模板：`docs/templates/qa-defect-management-template.md`；
> 与 [qa-test-cases.md](qa-test-cases.md) 验收线共用本台账（其头部预告的缺陷登记即本文件）。
> 环境真相源：联调真实库 `wbflow`（MySQL 8.0.46 @192.168.3.100，凭据 `rd/configs/config.local.yaml`）+ rd :18080（main @ e2b7ada 后工作区）。

## 缺陷清单总览

| 缺陷号 | 标题 | 严重程度 | domain（归属） | assignee | 状态 | 关联验收用例 | 发现源（复盘维度） |
|---|---|---|---|---|---|---|---|
| BUG-001 | 需求状态（requirements.status）与流程推进零联动——推演到技术评审仍滞留「待启动」 | 一般（S2）/ P1 | rd | 待指派 | **已修复（方案 A，回归 PASS）** | B-1~B-5（流程走查，status 断言维度缺失） | **User**（看板巡检发现 req-003 状态矛盾） |
| BUG-002 | arch_design.md 废弃版本联动描述未同步注记-A 拍板（L235/L380 与实现口径冲突；修复轮扩展为 3 文档 11 处同源旧口径） | 轻微（S3）/ P2 | docs | 待指派 | **已修复（3 文档 11 处 + D-4 机械巡检，回归 PASS）** | qa_doc_audit.py（D-4 新增） | **User**（User session 核查轮顺带发现） |
| BUG-003 | plan 流程 pln-d7 死锁：create_reqs 完成不被 exec 判据承认，无法到达 pln-d8 | 严重（S1）/ P0 | rd | 待指派 | **已修复（复测 PASS）** | B-5g（qa_drive.py 稳定复现） | **QA 验收轮**（qa_drive.py 黑盒全量） |
| BUG-004 | wb_update_requirement 强制全量字段：缺 category/status/priority 任一即 400 | 轻微（S3）/ P2 | rd | 待指派 | **已修复（空串=不修改，回归 PASS）** | A-10 | **QA 验收轮**（A-10a/b 报文实证） |
| BUG-005 | wb_assign_task/wb_task_update 不校验节点相位：未激活节点可派发并完成任务 | 轻微（S3）/ P2 | rd | 待指派 | **已修复（复测 PASS）** | B-3 / BUG-005a | **QA 验收轮**（B-3 剧本顺序暴露） |
| BUG-006 | wb_my_task 动作标识与 wb_assign_task 入参口径不一致（裸 produce vs 全名） | 轻微（S3）/ P2 | rd | 待指派 | **已修复（回显全名，回归 PASS）** | A-13c | **QA 验收轮**（A-13c 断言对照） |
| BUG-007 | 并行 join「wait 待对侧」分支经 API 不可达（peerApproved 恒 true） | 轻微（S3）/ P3 | rd | 待指派 | **已文档化（spec 注记-I）** | C-8b | **QA 验收轮**（C-8b 分支穷举判定死代码） |
| BUG-008 | wb_escalate 手动端点通知 anchor_kind=document，收件箱 escalation 投影断链 | 轻微（S3）/ P2 | rd | 待指派 | **已修复（落 escalation+源 issueId，回归 PASS）** | C-6c | **QA 验收轮**（scratch 库通知行 DB 实证） |
| OBS-001 | gate 拍板不校验 exec 判据：bus-01 无 brd 产物仍可 approved 迁移（设计两可） | 观察 | rd | User | **已拍板关闭（最小原则，非缺陷）** | B-1e | **QA 验收轮**（run1 环境故障侧露）→ User 拍板关闭 |
| BUG-009 | reproject 裁剪评审位不同步清理 review_results，与 scanner 巡检竞速留下幽灵评审位 | 一般（S2）/ P1 | rd | 待指派 | **已修复（复测 PASS）** | C-2d | **QA 修复回归轮**（scanner 竞速暴露，audit 实证） |
| BUG-010 | 流程树节点点色只编码相位不编码门禁结论——「已通过」与「待拍板」同为琥珀色 | 轻微（S3）/ P2 | fe | 待指派 | **已修复（刷新即生效）** | User 亲测反馈 | **User**（浏览器亲测截图反馈） |
| OPT-004 | 评审药丸「强行通过/强制修改」leader 裁定动作（自 OPT-002 拆单，涉权限与门禁语义） | 建议（S4）/ P3 | rd+fe | 待指派 | 新建 | CR 复审拆单（2026-09-10） | **User session**（CR 拆单） |
| BUG-011 | 技术评审上下文断裂：评审节点拿不到被评审产物（wb_my_task relatedArtifacts 与 FE 产物区均只查本节点） | 一般（S2）/ P1 | rd+fe | 待指派 | **已修复（含 dependencyDocs 注入，回归 PASS）** | User 批注「重大实现遗漏」（2026-09-10） | **User**（批注轮） |
| BUG-012 | 详情页需求说明缺失：描述卡沉底折叠且纯文本无 Markdown 渲染，背离 m4 原型「标题与流程树之间的 Markdown 描述卡」 | 一般（S2）/ P2 | fe | 待指派 | **已修复（mini md + 归位，待冒烟）** | User 批注「漏掉了需求说明」（2026-09-10） | **User**（批注轮） |
| BUG-013 | 评审评论可游离：document 级锚点允许不挂产物，评审意见不强制锚定被评审文档 | 轻微（S3）/ P2 | rd+fe | 待指派 | **已修复（rd 强制锚 + fe 预拦/锚名显示，回归 PASS）** | User 批注「所有的评审都是挂在文档上的」（2026-09-10） | **User**（批注轮） |
| BUG-014 | branch 协作口径过期：铁律⑦「同一 feature 分支」（评审敲定、全程同一分支）与 worktree 并行实践冲突，feature_branch 字段语义未随集成线模式更新 | 轻微（S3）/ P2 | docs | 待指派 | **已修复（集成线口径改写）** | User 批注（2026-09-10） | **User**（批注轮） |
| OPT-001 | flow 页布局重构：SVG 流程树在上+详情在下堆叠，改为 m2 原型「左列流程导航常驻+右列节点详情」两栏（颜色样式维持） | 建议（S4）/ P2 | fe | 待指派 | **已修复（两栏+窄屏退化，待冒烟）** | User 批注「现在的太丑了」（2026-09-10） | **User**（批注轮） |
| OPT-002 | 评审药丸只读不可点：缺 m2 原型「点角色展开对话区+参与讨论/裁定+强行通过/强制修改」交互 | 建议（S4）/ P3 | fe | 待指派 | **已修复（药丸可点展开意见线程，待冒烟）** | 原型对照 diff（2026-09-10） | **User session**（批注轮原型对照 diff） |
| OPT-003 | 变更历史形态与归位：自然语句列表折叠沉底，原型为字段级 diff 样式、随描述卡归位 | 建议（S4）/ P3 | fe | 待指派 | **已修复（随描述卡归位）** | 原型对照 diff（2026-09-10） | **User session**（批注轮原型对照 diff） |
| BUG-015 | 联调库全部产物为假域名外链占位（9/9 kind=link，document 类 0 行），文档阅读→锚定评论→评审交互链在用户验收环境不可达；系统不校验产物期望 kind（seed 全 document）放行外链冒充文档并一路推进 | 严重（S1）/ P0 | rd+data+docs+qa | 待指派 | **已修复（①数据②守卫落地，复测 PASS；③已落 spec 注记-K；FE 呈现通道已验：阅读器+行号+行选中）** | A-15g（守卫断言）/ D-3 巡检（新增） | **User**（批注「挂载的产物文档不能是外链…req-004 还全是外链」2026-09-10） |
| OPT-005 | 轻量 Markdown 渲染器不解析 blockquote（`> 引用`字面显示；支持清单=标题/粗体/列表/行内代码/代码块/链接，属 BUG-012 修复口径内边界） | 建议（S4）/ P3 | fe | 待指派 | 新建 | 浏览器冒烟轮（2026-09-10，req-004 描述卡实测） | **QA**（呈现通道冒烟轮） |
| BUG-016 | 流程节点无业务终态：produce 节点推进离场后 state 恒 exec，点色蓝点被读作「进行中/卡 exec」（老模型有已通过终态，v1.0 相位制缺失） | 一般（S2）/ P1 | rd+fe | 待指派 | **已修复（-L2 done 终态：推进离场收口+禁挂载+幂等 noop；IT/断言齐）** | User 截图质询（req-004 节点 3/5） | **User**（2026-09-10 深夜） |
| BUG-017 | done × 打回/reproject 时序缺口：C-2a 打回重做派发 409 done（复投影/重评路径未先复位）；D-2a override 审计时间戳（已修）；C-4 clarify open 计数（已修） | 一般（S2）/ P1 | rd | 待指派 | **新建（e2e 3 FAIL 载体；-L2 与回退/重投影交互待补复位逻辑）** | qa_drive C-2a/C-2b/D-2a（-L2 引入的回归面） | **QA**（-L 落地回归轮，2026-09-10 深夜） |
| OPT-005↑ | 扩容：Markdown 对齐老代码（+表格）+ blockquote（User 拍板支持）+ 阅读器交互对齐（单击框选/筛选栏/锚点回跳）——升 S3/P2 | 一般（S3）/ P2 | fe | 待指派 | 新建（对齐轮扩容） | 老代码对质 + User 拍板 | **User**（「严重漏实现」定性） |

---

## BUG-001 需求状态与流程推进零联动

### A. 基本信息

| 字段 | 值 |
|---|---|
| 缺陷编号 | BUG-001 |
| 标题 | 需求状态（requirements.status）与流程推进零联动——推演到技术评审仍滞留「待启动」 |
| 发现人 | huzilin（看板巡检发现 req-003 状态矛盾；ZCode 核查轮实证） |
| 发现日期 / 时间 | 2026-09-10 |
| 所属项目 / 版本 | wbflow v1.0（联调真实库 wbflow @192.168.3.100） |
| 所属模块 / 功能 | rd / 需求状态机 × 流程编排联动（requirements.status ↔ wb_advance/评审/门禁） |
| 指派负责人 | 待指派（rd） |
| 相关需求 / 用例 | `docs/v1.0/arch_design.md` §7.3 L380「收尾（最后节点通过）→ 已完成」、§8.3 L142 状态机行；关联数据 req-003 / req-004 |

**严重程度 / 优先级**：一般（S2）/ P1

### B. 缺陷详情

**缺陷描述**：流程编排全链路（wb_advance 推进 / 评审表态 / 门禁拍板 / 任务上报）不回写 `requirements.status`；需求状态自创建后恒为「待启动」，除非人工调用 `wb_update_requirement` 显式改状态。看板四泳道分类失真（已深度推演的需求滞留「待启动」泳道）；版本发布软约束 `unfinishedRefs`（`rd/service/version.go:348`，status NOT IN (已完成,已取消) 即 unfinished）在任何正常推演下恒提示 unfinished。spec 明文要求的「收尾（最后节点通过）→ 已完成」联动未实现；「启动→进行中」无任何触发点定义。

**复现步骤**（2026-09-10 正规编排重建实证，req-004「逐步推进验证需求（正常流程构造）」）：
1. `POST /api/requirements` 建 business×full 需求（status=待启动）；
2. 逐步推进：待启动节点 wb_decision approved → 需求对齐 to-brd 上报+拍板 → 产品设计 product-design.produce（登记 brd/product_doc 产物）+ wb_advance 提评 → 产品设计评审三位逐位 agree（arch/rd/fe）+ 拍板 → RD 技术设计（rd_design 产物）+ wb_advance 提评 → RD 技术评审四位逐位 agree（arch/fe/qa/pm）停等；
3. 每步 `GET /api/requirements/:id` 看 `requirement.status`。

**预期结果**：流程进入编排后状态应随推进联动——至少「收尾（最后节点通过）→ 已完成」（spec §7.3 明文）；若定义启动触发点（如待启动节点通过）则应为「进行中」。

**实际结果**：七步探针 `requirement.status` 每步均为「待启动」，与流程位置（RD 技术评审 gate 停等）矛盾。审计佐证：`audit_log` 中 requirements 表 req-004 仅 1 条 INSERT，零条 status UPDATE。

**环境信息**：

| 项 | 值 |
|---|---|
| 环境 | 联调真实库 wbflow（MySQL 8.0.46 @192.168.3.100）+ rd :18080 |
| 版本 | main @ e2b7ada 后工作区（17 工单 + 注记 A-H 全并入） |
| 数据 / 前置条件 | business×full 13 节点模板；req-003（速推复现样本）与 req-004（正规重建样本）同深度对照 |

**根因定位（代码级，2026-09-10 全库 sweep）**：`requirements.status` 全库仅两个写点——创建默认「待启动」（`rd/service/requirement.go:387`，repo default 同）与手动 Update 端点（`rd/service/requirement.go:597`）；`rd/service/advance.go`（wb_advance 三类推进）、review/task/version/scanner 全链路零 status 引用。即 req-003 的「待启动 vs 技术评审」非速推数据伪影，而是任何正规推进 1:1 复现的系统缺口。

**附注**：
- 「待启动」一词双义（需求状态值 + 流程首节点 key），为该类矛盾的呈现放大器，术语统一时一并处理；
- 并行快照口径差异：`.workbench/requirements.json`（SQLite 侧自管理库）req-003=已完成，与 MySQL 侧无关，勿混淆归因。

### C. 返工与复验记录

| 字段 | 值 |
|---|---|
| 缺陷状态 | 已确认（User 2026-09-10 拍板「先记录缺陷」） |
| 原因分类 | 程序错误（spec §7.3 联动要求漏实现；启动→进行中触发点 spec 亦未定义） |
| 修复版本 / 排期 | 待拍板 |
| 修复方案 | 待拍板二选一：A=实现自动联动（收尾→已完成 为 spec 既有要求必须；启动→进行中 建议挂「待启动节点通过」触发）；B=维持手动 Update + UI/文档明示口径 |
| 复测结果 | 待修复后复测（复测口径=BUG-001 复现步骤全链重跑 + e2e full 剧本补 status 断言） |
| 关闭人 / 日期 | — |

### D. 归属判定（domain 与理由）

| 项 | 值 |
|---|---|
| domain | rd |
| 判定理由 | requirements 表与推进编排均为 rd 服务职责；联动缺失在 rd service 层（advance/review 路径未回写），FE 仅按 API 快照呈现，无前端侧成因。 |

---

## BUG-002 arch_design.md 废弃版本联动描述未同步注记-A

### A. 基本信息

| 字段 | 值 |
|---|---|
| 缺陷编号 | BUG-002 |
| 标题 | arch_design.md 废弃版本联动描述未同步注记-A 拍板（L235/L380 与实现口径冲突） |
| 发现人 | ZCode 核查轮（BUG-001 排查中顺带发现） |
| 发现日期 / 时间 | 2026-09-10 |
| 所属项目 / 版本 | wbflow v1.0 文档基线 |
| 所属模块 / 功能 | docs / arch_design.md 状态机章节 |
| 指派负责人 | 待指派（docs） |
| 相关需求 / 用例 | `docs/v1.0/arch_design.md` L235、L380；裁决正本 `.plan/wbflow-impl/spec.md` §1.2 注记-A（2026-09-09） |

**严重程度 / 优先级**：轻微（S3）/ P2

### B. 缺陷详情

**缺陷描述**：arch_design.md 两处仍写「废弃版本联动其下需求 status → 已取消」（L235「废弃联动其下需求 `status → 已取消`」、L380「废弃版本联动 → 已取消」），而 2026-09-09 注记-A 已裁决为**条件解绑**：任一需求已开始编排 → 409 blocked 拦截废弃；全部未开始 → 废弃成功并解绑回待分配池，**不取消需求**。实现（`rd/service/version.go` DeprecateVersion）按注记-A 落地，文档两行成为过期口径。

**复现步骤**：对照 arch_design.md L235/L380 与 spec.md 注记-A、`rd/service/version.go` DeprecateVersion 实现。

**预期结果**：文档与裁决正本一致（或显式标注「已由注记-A 修订」）。

**实际结果**：文档保留旧口径「联动 → 已取消」，与实现冲突。

### C. 返工与复验记录

| 字段 | 值 |
|---|---|
| 缺陷状态 | **已修复（2026-09-10 复盘轮）** |
| 原因分类 | 需求变更（注记-A 裁决替代原设计口径，文档未跟进） |
| 修复方案 | **同源旧口径全量清零（3 文档 11 处，非原登记的 2 行）**：arch_design.md L235（正文改条件解绑口径）；`01-module-board/rd/design.md` 5 处（版本废弃流/路由表/状态转移表/双写顺序/AC）；`01-module-board/prd/prd.md` 5 处（安全性指标/交互流程/业务规则/AC ×2）。`_archive/` 2 处为历史归档不改。均改为注记-A 口径：全部未开始 → 解绑回池（不取消、不改 status、逐行审计）；任一已进入编排 → 409 blocked |
| 复测结果 | **PASS**——`qa/qa_doc_audit.py`（D-4 文档一致性巡检，本轮新增固化）：禁用短语（旧口径 4 种）扫 docs/v1.0/**（排除 _archive/qa），修复前 11 违规 → 修复后 0 违规，退出码 0；脚本进回归口径 |
| 关闭人 / 日期 | ZCode / 2026-09-10 |

### D. 归属判定（domain 与理由）

| 项 | 值 |
|---|---|
| domain | docs |
| 判定理由 | 纯文档同步问题，实现已按裁决正本落地（真库废弃 blocked 实证过），无代码改动。 |

---

## BUG-003 plan 流程 pln-d7 死锁（2026-09-10 全量验收轮登记）

### A. 基本信息

| 字段 | 值 |
|---|---|
| 缺陷编号 | BUG-003 |
| 标题 | plan 流程 pln-d7 死锁：create_reqs 完成不被 exec 判据承认，无法到达 pln-d8 |
| 发现人 | ZCode QA 全量验收轮（qa_drive.py） |
| 发现日期 / 时间 | 2026-09-10 |
| 所属项目 / 版本 | wbflow v1.0（scratch 库 + rd :18081 实测，main @ e2b7ada 后工作区） |
| 所属模块 / 功能 | rd / wb_advance exec 判据 × plan 流程尾段 |
| 指派负责人 | 待指派（rd） |
| 相关需求 / 用例 | qa-test-cases.md B-5g；`.plan/wbflow-impl/spec.md` 注记-B |

**严重程度 / 优先级**：严重（S1）/ P0（plan 流程 AC 完全阻塞）

### B. 缺陷详情

**缺陷描述**：plan 流程走到 pln-d7（创建需求·启动版本），`decide-cap.create_reqs` 任务已完成（tasks.status=completed），`wb_advance` 恒 409，流程永远无法迁移 pln-d8（完成）。plan 线 9 节点只能走通 8 节点。

**根因（代码级）**：`rd/service/advance.go` `execAdvanceReady` 只统计 `produce/dev/accept` 三族任务；`create_reqs`（TaskTypeCreateReqs）完成不被承认 → completed=0；又因 pln-d7 node_actions 挂载 capability 动作（`execActionsMounted=true`），rule4 空转兜底同样不生效 → 恒 blocked。`task.go` complete 内聚推进亦仅对三族触发。

**复现步骤**：建规划审批任务需求 → 逐节点推进至 pln-d7 → 派发并完成 `decide-cap.create_reqs` → `POST /api/requirements/:reqId/nodes/:nodeId/advance`。

**预期结果**：create_reqs 完成后 advance 推进 pln-d8。

**实际结果**（scratch 库 req-011 / n-110 实测）：
`{"ok":false,"error":"wbflow: 门禁拦截: 节点 \"n-110\" exec 相位 complete 判据未通过，拒绝推进","code":"GATE_BLOCKED","blocked":true,"missing":["exec_task:节点无已完成 exec 任务（exec 判据未通过）"]}`；同节点任务行 `t-1788979733428-030 | decide-cap.create_reqs | create_reqs | completed`。

**修复方案（建议）**：execAdvanceReady/complete 触发族纳入 create_reqs，或 pln-d7 期望产物为空时放行 rule4。

---

## BUG-004 wb_update_requirement 强制全量字段（2026-09-10 全量验收轮登记）

- **严重程度**：轻微（S3）/ P2；**domain**：rd
- **现象**：PUT 只传想改的字段即 400，缺省字段逐一报 `非法状态 ""（允许：待启动/进行中/已完成/已取消）`、`非法优先级 ""（允许 P0-P3）`、`category 必填`——局部更新语义不存在，调用方必须全量回显，与 spec「自动 diff 变更历史」使用方式冲突，契约未注明全量必填。
- **复现**：`PUT /api/requirements/:reqId` body 仅 `{requirementId,title,priority,updatedAt}` → 400 `category 必填`。
- **修复方向**：空字段跳过校验与 diff（显式传入才校验），或契约显式声明 PUT=全量语义。

## BUG-005 任务总线不校验节点相位（2026-09-10 全量验收轮登记）

- **严重程度**：轻微（S3）/ P2；**domain**：rd
- **现象**：对未激活节点（flow_nodes.state=""）可成功 `wb_assign_task`→`ack`→`progress`→`complete`（status=completed），但节点相位不迁移 → 孤儿完成任务；节点正式激活后 exec 判据统计又把历史完成计入（completed>0 即放行），可能跳过真实执行。
- **证据**：tec-b8b 激活前执行：任务 completed，节点相位仍 `""`；review 表态报 `节点 "n-92" 相位 "" 不接受评审表态`。
- **修复方向**：assign/update 校验节点 state 与 def_node_id 归属（my-task 三向校验同源口径）。

## BUG-006 my-task 与 assign 动作标识口径不一（2026-09-10 全量验收轮登记）

- **严重程度**：轻微（S3）/ P2；**domain**：rd
- **现象**：`wb_my_task` 返回 `activeActions[].actionId="produce"`（裸名）；`wb_assign_task` 入参消费 `"product-design.produce"`（全名）。调用方无法直接回填；裸名恰好也命中 actionTaskTypes 表但派发定位走另一解析路径——双口径并存无文档。
- **证据**：A-13c acts=['produce']；同节点 assign 用 `product-design.produce` 成功。
- **修复方向**：my-task 回显与 assign 同口径（全名），或 assign 明确双口径并写入契约。

## BUG-007 并行 join wait 分支经 API 不可达（2026-09-10 全量验收轮登记）

- **严重程度**：轻微（S3）/ P3；**domain**：rd
- **现象**：注记-C①「本侧完成待对侧（wait）」常规面不可达——fe 侧 exec-CR 节点只在 rd 侧 `last_decision=approved` 后激活，激活后 peerApproved(prev) 恒 true，`AdvanceOutcomeWait` 无 API 触达路径，仅 scanner 兜底巡检语义可达。
- **证据**：qa_drive.py `C-8b`（SKIP 附因）；`rd/service/advance.go` advanceAfterGateLockedTx 分支。
- **修复方向**：记录为已知死代码；或放宽激活链允许双侧并行激活后再议。

## BUG-008 wb_escalate 手动端点通知 kind 投影断链（2026-09-10 全量验收轮登记）

- **严重程度**：轻微（S3）/ P2；**domain**：rd
- **现象**：`POST /api/comments/:id/escalate` 手动上升产生的通知行 `anchor_kind='document'`（文本「[冲突上升待仲裁] …待 User 裁决」）；收件箱 kind 按 anchor_kind 投影 → 手动仲裁请求不呈现为 `escalation` 类，注记-D5 TODO「冲突待裁决←escalation」投影断链。对照 `wb_comment escalate=true` 与 reopen 自动阈值路径均正确落 `anchor_kind='escalation'`。
- **证据（scratch 库 issues 表 notification 行）**：`document|qa|[冲突上升待仲裁] issue cm-…735369 待 User 裁决`（×2 手动）；`escalation|qa-a|冲突上升裁决：评论 cm-…734749…`（comment 路径 ✓）；`escalation|qa-a|冲突上升裁决：评论 cm-…6121…`（自动阈值路径 ✓）。
- **修复方向**：wb_escalate 通知行落 anchor_kind=escalation（或 FE 投影兼容「待仲裁」行并登记偏差）。

## OBS-001 gate 拍板不校验 exec 判据（观察，待拍板）

> 2026-09-10 按 User 反馈改写详尽版：区分「纯 gate 不校验（合理）」与「复合节点不校验（两可）」。

### 1. User 理解确认：纯 gate 不校验是对的

**User 理解正确且与实现一致**：bus-00（待启动）、bus-11（用户验收）、bus-12（完成）、pln-d0/d2/d8、tsk-c0/c6/c7、tec-b0/b10/b11 这类**纯 gate 节点**，定义侧 `expected_outputs=[]`、无 exec 能力挂载——本来就没有任何产物/任务可校验，`wb_decision` 只做「user_gate=1 + collect 通过」校验，天然放行。**最终用户验收环节不校验、也不应校验，本观察项从来不是指它。**

### 2. 观察项实际所指：复合节点（gate+exec 挂载+期望产物）

问题场景 = **user_gate=1 且挂载了 exec 产出判据的复合节点**，seed 里典型是三个对齐节点：

| 节点 | user_gate | exec 挂载 | 期望产物 |
|---|---|---|---|
| bus-01 需求对齐 | 1 | to-brd（pm，skill=to-brd） | brd《业务需求文档》 |
| tec-b1 / tsk-c1 / pln-d1 | 1 | to-brd / 无（迎宾）/ to-alignment | brd / — / alignment |

设计意图是「**先落盘、后确认**」（e2e full 剧本注释原话：entry hold → to-brd 落盘 → 用户确认迁移）。

### 3. 什么场景需要校验（实例）

**实例 1（本轮实测，brd 从未落盘）**：bus-01 上 to-brd 任务未派发/未完成、brd 产物未登记时，`wb_decision approved` 仍 200 并迁移 bus-02。下游产品设计、评审、开发全链对着一个**不存在的对齐文档**推进；事后追溯「用户当时确认的对齐结论是什么」没有任何载体（issues 里也没有讨论记录）。判据在本该拦的节点形同虚设。

**实例 2（防线不对称）**：同一件事「brd 未产出」——走任务层 complete 会被拦（A-17b 实测：`409 blocked + missing:['artifact:brd']` + 任务回 in-progress，spec §2「谎报完成」行）；绕过任务层直接拍板却放行。两条路一拦一放，谎报/漏操作防线不对称。

**实例 3（时序错乱无护栏，未实测、修复时一并界定）**：wb_decision 校验集=user_gate=1+collect 通过，而评审位为空的 gate 节点 collect 恒 approved（`advance.go` 同源公式）。据此推断编排未进入（state=""）时拍板可能同样放行——若属实，bus-01 可在 bus-00 未拍板时被直接拍过。此点诊断轮未实测，修复时一并验证。

### 4. 两可论据（供拍板）

- **不校验方**：gate=用户意志表达；「用户线下/口头已确认，只是 agent 还没把 brd 写完」时强制判据会卡死合法流程；纯 gate 节点（含最终用户验收）本就无判据可校验，wbflow 定位是编排系统不是产出防线，产出质量归任务层与评审层。
- **校验方**：复合节点的拍板语义是「确认这份产物」，产物不存在时「确认」落空；与 exec 侧严判据不对称；实例 1 的追溯空洞在真实项目是实打实的风险。
- **折中（三选项）**：仅对「user_gate=1 且期望产物非空」的节点做**软校验**——拍板时 exec 判据未过则 409 blocked+missing（硬）；或 200+warning 字段（软提示不拦截）；或维持现状+FE 拍板卡上展示「对齐文档未登记」警示。三方案对纯 gate（bus-00/11/12）均零影响。

### 5. 实测证据

run1（环境 docs root 故障侧露）：bus-01 to-brd 任务 complete 被判据拦（409，brd 缺）→ 同状态直接 `wb_decision approved` → 200，flows.current_node_id 迁 bus-02。复现脚本 `qa/qa_drive.py` B-1e 路径。

### 6. 处置（2026-09-10 User 拍板：关闭，非缺陷）

**拍板口径：最小原则——每个环节只关心自己的事情，不做跨环节校验。** approve 拍板不校验 exec 产物（也不应扩展校验投票/澄清遗漏等——均非拍板环节的职责）；「产物必须存在」由 exec 环节自己的出口把守：`wb_task_update complete` 判据（产物缺 → 409 blocked+missing，A-17b 实测）+ `wb_advance` exec 判据（exec 出口，本就存在）。bus-01 无产物可拍板 = 合法行为（用户意志表达），OBS-001 关闭。
（注：本轮曾按「硬校验」方向短暂实现 approve 拦截，已按本拍板撤回。）

---

## BUG-009 reproject 幽灵评审位（2026-09-10 回归轮发现并修复）

- **严重程度**：一般（S2）/ P1；**domain**：rd；**状态**：已修复（复测 PASS）
- **现象**：打回重定窗口内 scanner 兜底巡检（SweepParallelJoins）抢先推进 carrier（by_who=system 审计实证），按**裁剪前**评审位播种下游评审轮；随后 reproject 只裁剪 `reviewers` 字段——被裁角色（fe）的 reviewing 条目永久残留 `review_results`（幽灵位）。
- **证据**：audit_log id 364/366（system 播种 4 位）早于 id 368（user 裁剪 reviewers 至 3 位）；bus-05 review_results 残留 `fe reviewing round 0`。
- **修复（两处，最小原则各自把关）**：①`SweepParallelJoins` 跳过任一节点持有 `pending_scope` 的流程（打回重定窗口不巡检推进）；②reproject 裁剪 reviewers 时同事务同步剔除被裁角色的 review_results 条目（被裁角色不可能再表态，残留恒为陈旧数据）。

## BUG-010 流程树点色语义（2026-09-10 User 亲测发现并修复）

- **严重程度**：轻微（S3）/ P2；**domain**：fe；**状态**：已修复
- **现象**：节点 4「产品设计评审（已通过）」与节点 6「RD 技术评审（待拍板）」左侧点同为琥珀色——点色类名 `phase-${node.state}` 只编码相位（两者均 review 相位），不编码门禁结论。
- **修复**：点色改「门禁结论优先，相位兜底」——绿=已通过 / 琥珀=待拍板 / 红=已打回（deriveGateView 三态覆盖）；无结论时按相位着色，review 相位由琥珀改青色（琥珀让位给待拍板）。改动：`fe/src/components/FlowTree.tsx`（dotVariant）+ `fe/src/index.css`（gate-* 色板）。文字徽标（已通过/待拍板）保持双通道，不单靠色相。

## BUG-011 技术评审上下文断裂（2026-09-10 User 批注轮登记）

- **严重程度**：一般（S2）/ P1；**domain**：rd+fe；**状态**：新建
- **现象**：评审节点（如 RD 技术评审）拿不到被评审产物——`wb_my_task` 的 relatedArtifacts 只查本节点（`rd/service/flow.go:345` `ListArtifactsByRequirementAndNode(reqID, nodeID)`），评审节点自身不产产物故恒空；FE 节点详情产物区同样只过滤本节点（`fe/src/components/NodeDetail.tsx:81`），恒显「本节点暂无登记产物」。评审者（agent 与人）被迫回上一节点翻文档。seed 评审能力 how 写了「读输入：被评审设计文档」，但仅为文字描述，程序不注入；`tasks.context` 注释中的 `dependencyDocs` 位子（`docs/v1.0/arch_design.md:628`）实现零命中。
- **User 原话**：「技术评审，必须看到上一个节点的产物，比如 rd 的技术评审，必须带上 rd 的技术设计，否则我们在评审什么，每次还要到上一个节点去看产物」；定性「重大的实现遗漏」。
- **复现**：任一需求推进到技术评审节点 → 打开节点详情面板 ③ 产物区 → 空；agent 调 wb_my_task → relatedArtifacts=[]。
- **预期**：评审节点直接内嵌上游被评审产物，点击即开阅读器，零跳转。
- **修复方向**（User 已拍板「按清单方向来」）：①RD 沿流程图推导评审覆盖区产物（自评审节点沿 node_order 向前收集有产物节点的产物，遇 gate=1/user_gate=1 节点停止且不含）；RelatedArtifactItem 加 nodeId/upstream 标注；②FE 产物区分「本节点产物」「被评审产物（上游）」两段；③`tasks.context.dependencyDocs` 落实现。

## BUG-012 详情页需求说明缺失（2026-09-10 User 批注轮登记）

- **严重程度**：一般（S2）/ P2；**domain**：fe；**状态**：新建；**原因分类**：需求缺失（实现遗漏）
- **现象**：m4 原型明确「详情页需求描述卡片（Markdown 渲染）插在标题与流程树之间」，实现把描述放在页面最底部补遗卡 `<details>` 默认折叠、且 `<pre>` 裸文本无 Markdown 渲染（`fe/src/pages/RequirementDetailPage.tsx:266-269`）——进详情页看不到需求在说什么。
- **User 原话**：「你要自己完整的 diff 一下差异，比如漏掉了需求说明」。
- **修复方向**：描述卡上移至头部元信息与 FlowTree 之间 + 轻量 Markdown 渲染（粗体/标题/行内代码/代码块/列表/链接，与 ArtifactReader 同口径不引重库）+ 有描述时默认展开；变更历史随补遗区归位（OPT-003）。

## BUG-013 评审评论可游离不锚文档（2026-09-10 User 批注轮登记）

- **严重程度**：轻微（S3）/ P2；**domain**：rd+fe；**状态**：新建
- **现象**：issues 表 document 级锚点允许 `anchor_artifact_id=NULL`（`rd/repo/model.go:269`），评审意见可以不挂任何文档游离在节点上；评审场景交互也没有把被评审文档顶到评审者面前（BUG-011 连带），评论自然挂不上去。
- **User 原话**：「所有的评审都是挂在文档上的」。
- **修复口径**：节点可见产物非空时，wb_comment document 锚必须携带 anchorArtifactId（新评论硬拦，存量数据不回溯）；无产物节点（纯对齐/门禁）document 锚保持合法；FE 评论框在有产物语境默认锚定当前产物。

## BUG-014 branch 协作口径过期（2026-09-10 User 批注轮登记）

- **严重程度**：轻微（S3）/ P2；**domain**：docs（涉及 rd 语义注记）；**状态**：新建；**原因分类**：需求缺失（合并需求时原设计要求丢失，且未随 worktree 实践演化更新）
- **现象**：原设计要求「技术设计/评审阶段敲定分支名，后续 RD/FE/QA/CR 全程同一分支」（铁律⑦、分支约定 `arch_design.md:1147`、节点职责 L1122/L1140「敲定 feature 分支名」）。v1.0 实际执行线已是 worktree 并行模式：17 工单各自分支（`wbflow-impl-t*`）从集成线切出、合回 `wbflow-impl`。「同一分支」字面机制在 git 下与 worktree 物理矛盾（一个分支仅允许一个 worktree checkout），两 agent 也不能共用工作区。
- **拍板口径（2026-09-10 User 确认按建议方向改写）**：对齐动作保留、机制泛化——铁律⑦改写为「**同一 feature 集成线**」：评审阶段敲定集成线名与分支拆合约定（工单分支 `<集成线>-t<N>-<slug>` 切出合回），QA/CR 验收看集成线；`feature_branch` 字段保留，语义注记从「全程共用分支名」改为「集成线/落点约定」；节点职责同步改写；决策理由落 semantica。
- **修复面**：`docs/v1.0/arch_design.md` L1310（铁律⑦）、L1147（分支约定）、L1122/L1140（节点职责）、L577/L604/L942（feature_branch 注记）。

## OPT-001 flow 页左右分栏布局重构（2026-09-10 原型对照 diff 登记）

- **级别**：建议（S4）/ P2；**domain**：fe；**状态**：新建；**User 定性**：优化项（「现在的太丑了」，颜色样式先维持）
- **差距**：m2 原型=左列 220px 流程步骤条常驻（状态点/选中高亮/回滚 ↩ 标记）+ 右列节点详情卡，点步骤即切右栏；实现=SVG 流程树（460px）在上 + NodeDetail 六面板在下垂直堆叠，19 节点树拉长后选节点需滚动找详情。
- **修复方向**：详情页改两栏——左窄列流程导航（保留并行泳道一行两列形态与连接线）+ 右宽列节点详情；颜色样式不动。

## OPT-002 评审药丸可点展开对话区（2026-09-10 原型对照 diff 登记）

- **级别**：建议（S4）/ P3；**domain**：fe；**状态**：新建
- **差距**：m2 原型评审状态行点角色展开对话区（该角色意见线程 + 输入框参与讨论/裁定 + 〔强行通过〕〔强制修改〕快捷入口）；实现 ReviewPills 只读（`fe/src/components/NodeDetail.tsx:295`），意见在④区分离展示。
- **修复方向**：药丸可点、展开该角色意见线程；强行通过/强制修改入口与 DecisionCard 现有裁决动作对齐归位。

## OPT-003 变更历史形态与归位（2026-09-10 原型对照 diff 登记）

- **级别**：建议（S4）/ P3；**domain**：fe；**状态**：新建
- **差距**：m4 原型=字段级 diff 样式（旧值划线 → 新值绿色）随需求描述卡展示；实现=自然语句列表折叠沉底（信息等价，形态弱化）。
- **修复方向**：随 BUG-012 描述卡归位；形态保留自然语句版（可后续升级字段级 diff 样式），不再单列排期。

## OPT-004 评审药丸「强行通过/强制修改」leader 裁定动作（2026-09-10 深夜对齐轮展开；待拍板）

> 本节为审批材料：User 要求展开（原条目一行过略）。已全量核对 m2 原型（02-module-flow-tree prd/rd）、老代码（~/tmp_1/test-flow/workbench-plugin client.js/host.js）与 v1.0 现状。

### 1. 原型口径（应该是什么）

- **prd 评审状态行**：角色药丸行（`Architect ✅ / FE ⚠2 / QA ⚠轮14`）→ 点开该角色「意见↔反馈」对话区（一评论一线程，多轮 roundCount）→ **User 三态：参与讨论（回复）/ 普通 resolve / 强权 override**。业务规则：轮次 ≥10 未解决 → 药丸黄色高亮「疑似卡死」；**override 把该角色所有 open 评论置 overridden**。AC：「点『强行通过 X 全部剩余』→ 该角色 open 评论全 overridden」。
- **rd/design 模型工具**：`wb.override`（评论/整角色 → overridden，User 强权）、`wb.forceRevise`（记强制修改令，产出者不可异议）、`wb.defineScope`（关闭 pendingScope）。
- **老代码实况**：`doOverrideRole`（client.js L109/L726）→ `wb.override {reqId, reviewerRole}` → host.js `opOverride` L370：该角色全部 open 评论置 overridden（resolvedBy=User，UI 标签「已强制通过」）——**强行通过已有完整实现**；`wb.forceRevise` 仅存在于模型工具清单，**client/host 均无实现**（原型只到模型层）。

### 2. v1.0 现状与门禁哲学差异（拆单根因）

- v1.0 已有：`wb_adjudicate`（单条评论 uphold/override，冲突上升后的仲裁路径，-D2）、reopen loop_count≥5 自动 escalate（-C7）。
- v1.0 没有：整角色一键 override、forceRevise、轮次≥10 高亮（口径对应物是 loop_count 阈值，数值也不同 10 vs 5）。
- **核心哲学差异（拆单真正原因）**：老原型 gate=「评论全部闭环才可通过」——老 client.js L598 `gateBlocked = openCnt>0` 直接拦「通过」按钮，override 是绕评论闭环的强权通道；v1.0 gate=**评审表态 collect**（agree/unagine 投票）+ user_gate，评论闭环走 -D1 resolve/retest 链、**不拦拍板**（OBS-001 最小原则拍板：各环节只守自己出口）。同一动作「强行通过 X」在两套模型里语义不同：老=绕评论闭环，v1.0=（若引入评论闭环拦拍板）才有对应的绕行需求。

### 3. 拍板选项

| 选项 | 内容 | 影响 |
|---|---|---|
| A（对齐老门禁） | 引入「评论闭环参与门禁」：gate 判据=评审表态 collect **且** 名下评论无 open/pending-confirm；新增 `wb_override_role`（整角色 open→overridden，overridden 计入闭环）+ forceRevise | 动门禁核心语义，与 OBS-001 最小原则拍板冲突需重拍；改动 rd 判据+fe 双端，大 |
| B（最小对齐，推荐） | 表态制门禁不动；新增整角色 override 端点（批量 overridden，仅作强权清理/表态，标注「不改变门禁结果」）+ 药丸对话区入口；forceRevise 不做（v1.0 打回=wb_decision rejected 已覆盖其语义）；轮次高亮沿用 loop_count≥5 escalate 口径 | 前端药丸区+一个 rd 端点；与既有拍板零冲突 |
| C（维持现状+文档化偏差） | 不做，偏差记入 spec 注记 | 零改动；原型交互持续缺口 |

### 4. 关联

同轮对齐发现的 BUG-016（节点终态缺失）与 OPT-005（Markdown 对齐）见各自条目；飞书文档评论交互调研结论并入 OPT-005。

## BUG-016 流程节点无业务终态——已完成 produce 节点恒显 exec「进行中」（2026-09-10 深夜 User 发现；对齐轮登记，待拍板修复方向）

### A. 基本信息

| 字段 | 值 |
|---|---|
| 缺陷编号 | BUG-016 |
| 标题 | 流程节点无业务终态：produce 节点推进离场后 state 恒保留 exec，点色蓝点被读作「进行中/卡在 exec」 |
| 发现人 | huzilin（req-004 截图质询「为什么 3 节点、5 节点还是进行中，还卡在 exec」） |
| 严重程度 / 优先级 | 一般（S2）/ P1（误导性强，被 User 判为严重 bug 候选；流程实际未卡） |
| domain | rd（状态机语义）+ fe（点色） |

### B. 缺陷详情

**现象**：req-004 推进至节点 7（FE 技术设计，当前），节点 3（产品设计）/5（RD 技术设计）点色仍为蓝（exec 相位色），文字徽标「exec / produce」，被读作「还在进行中、卡在 exec」。流程实际正常（current_node_id=bus-06 活跃推进）。

**实证（联调库）**：bus-02/bus-04/bus-06 `flow_nodes.state` 均为 `exec`（推进离场不复位）。

**根因（老代码对质）**：老模型节点 state=**业务终态集**（未开始/进行中/已通过/未通过），gate 通过时置「已通过」（host.js L393），全部节点类型都有完成表达，dotColor 绿点语义统一；v1.0 状态机 state=**相位**（""/entry/exec/review/respond/gate_pending），wb_advance 推进只初始化下一节点相位，**被离场节点不复位**，且 produce 节点无 gate 无评审位 → `deriveGateView` 恒空 → 点色回落 `phase-exec`（蓝）。**属 v1.0 状态机相对老代码/原型的语义缺口（节点业务终态缺失），非推进故障。**

### C. 修复方向（待拍板二选一）

| 选项 | 做法 | 影响 |
|---|---|---|
| A（数据层终态） | wb_advance 推进离场时把被离开节点置业务终态（如 state="done"；打回重入时清除） | 动状态机+判据+打回重入/reproject 联动，改动大；下游「历史 exec 任务计入判据」（BUG-005 修复口径）需回归 |
| B（展示层推导，推荐起步） | fe `deriveGateView`/NodeDetail 扩展：produce 节点 exec 任务已 completed 且流程已离场（下游评审已通过或非当前）→ 点色绿 + 徽标「已完成」；零数据改动 | 纯前端；与后续 A 不冲突 |

## OPT-005 Markdown 渲染器对齐老代码 + blockquote 支持（2026-09-10 深夜 User 拍板「OPT-005 要支持」并定性文档交互为严重漏实现；对齐轮扩容）

- **级别**：一般（S3）/ P2（User 定性「文档交互不符合预期=严重漏实现」多次发生，升 S3/P2）；**domain**：fe；**状态**：新建（待排期）
- **基准**：老代码 `~/tmp_1/test-flow/workbench-plugin/client.js`（renderMd L292-336 + 阅读器 L737-786）+ m2/m4 原型 + 飞书文档评论交互（调研结论见下）。
- **差距清单（老 → 当前）**：
  1. **表格渲染**：老 renderMd 支持管道表格（isTableSep/splitRow/renderTable）；当前 mini md 无 → req-004 引用的 rd/design.md 含表格渲染为普通行。**必须补**。
  2. **blockquote**：老代码也不支持；User 拍板要支持（新增，对齐飞书引用块）。
  3. **行框选手感**：老=纯单击序列（点起点→点他行=框选范围、点起点=取消、框选后任意点=重新开始，prd「点其他行=框选范围」）；当前=click + shift-click 区间。**交互模型须改成老的单击序列**（飞书式）。
  4. **阅读器筛选栏**：老=作者 pills（全部/各作者/隐藏）+ 状态 pills（全部/未解决/已解决）；当前无。
  5. **锚点展示与回跳**：老=line 评论内嵌对应行下方、range 只在末行+「↔ A~B 行」可点标签回跳高亮（highlightRange 琥珀背景）、block 代码块下方、document 正文末尾分隔线后；当前部分具备，须逐条对齐。
  6. **评论卡信息密度**：老=轮次标签（>5 轮红）+ pending-confirm 超时标签（User 1h/角色 5min 阈值）+ 四态标签（含「已强制通过」）；当前待逐条对齐。
- **修复方式**：以老 client.js 阅读器为交互基准移植到 React 组件（后端零改动或仅微小配套）；Markdown 渲染器升级=mdview 扩表格+blockquote（单测同步）。
- **飞书调研结论**：飞书文档评论=划词即评（选区浮层「评论」按钮）、评论气泡锚定段落右侧/行间、线程回复+已解决折叠、@人協作——老代码交互已覆盖其主干（划选/锚定/线程/解决），主要差异是气泡视觉形态与 @能力，属可选增强不入本轮范围。

## BUG-015 联调库产物全外链占位，文档交互链在用户验收环境不可达（2026-09-10 User 批注登记）

> 本节同时是**审批材料**（待拍板三项在 C 节）：含 User 原文照抄、语境、相关条款关系、选项与影响、性质分类。

### A. 基本信息

| 字段 | 值 |
|---|---|
| 缺陷编号 | BUG-015 |
| 标题 | 联调库全部产物为假域名外链占位（9/9 kind=link），文档阅读→锚定评论→评审交互链在用户验收环境不可达；系统不校验产物期望 kind 放行外链冒充文档并一路推进 |
| 发现人 | huzilin（批注）；ZCode 本轮 SQL/代码取证 |
| 发现日期 | 2026-09-10 |
| 所属版本 / 环境 | wbflow v1.0 联调库 `wbflow`（MySQL @192.168.3.100）+ rd :18080（main 工作区，docs root 缺省 `docs/`） |
| 所属模块 | data（联调数据）× rd（wb_add_artifact / 推进判据）× fe（ArtifactReader 后果）× docs（约束无正本）× qa（无断言/无呈现通道/未对账联调库） |
| 严重程度 / 优先级 | **严重（S1）/ P0**——阻塞用户验收：产品核心交互「评审挂在文档上」（BUG-013 口径）在用户环境零可达 |

**User 原文照抄**：「一个 QA 很严重的问题。之前反复提及，挂载的产物文档，是不能是外链的，必须是系统内部的入库的产物，我看 req-004，还全是外链，这样你真的测试了文档的交互了么？我也没法测试相关数据。」

### B. 缺陷详情

**实证（联调库只读 SQL，2026-09-10）**：

```
SELECT requirement_id, kind, COUNT(*) FROM artifacts GROUP BY requirement_id, kind;
→ req-002 link 2 | req-003 link 4 | req-004 link 3        （document 类：0 行）
req-004 三产物：brd《BRD 对齐纪要》(n-41) / product_doc《产品设计文档》(n-42) / rd_design《RD 技术设计》(n-44)
href 均为 https://docs.example.local/req-004/<product_type>（假域名占位）
```

audit_log：artifacts 表 9 行均 INSERT，by_who=pm/rd（agent 会话扮演角色登记），两批：`1788902xxx`（req-002/003，早间会话）与 `1788977xxx`（req-004，BUG-001 正规重建轮）。

**三层成因**：

1. **数据层**：agent 会话以假域名外链占位替代真实入库文档——document 类登记要求 docs root 下文件真实存在（`rd/service/artifact.go:91-95`），建文件的成本被 link 绕过。
2. **系统层（守卫缺失）**：seed 40 个定义节点的 expected_outputs **全部 `kind:"document"`**，无任何 link 期望；但 `wb_add_artifact` 只校验 kind 在词表 + href 按 kind 格式（`artifact.go:85-97`），**不对照该节点 expected_outputs 的期望 kind**；推进/完成判据只看 product_type 是否有产物、不看 kind（`advance.go` 无 kind 引用）→ 外链可冒充文档产物一路推进到技术评审（req-004 即如此）。
3. **呈现层后果**：`fe/src/components/NodeDetail.tsx:320` 对 link 渲染为 `<a target=_blank>` + 「外链」徽标；`ArtifactReader.tsx:50` `isDocument = kind !== "link"` → link 不拉正文、无行/段选择、无锚定评论。用户点任何产物 = 新标签打开死域名；**文档阅读 → 选段 → 锚定评论 → 评审表态整条链在用户环境不可达**。

**QA 为何没发现（如实）**：

- QA 验收轮在 scratch 库用真实 fixture（document 类，`docs/v1.0/qa/fixtures/`）跑 A-15 正文回读与 D-1 产物矩阵——**API 层文档回读测过，但只测了 QA 自己造的数据**；
- **联调库（用户实际浏览的数据）从未对账**：安全原则「不触 wbflow 库」同时意味着没人校验它的合规；D-1 若跑在联调库会 9/9 FAIL（kind link ≠ expected document）；
- **FE 文档交互链零覆盖**（无呈现通道）；
- **约束不在 oracle**：「文档类产物不得外链」User 反复口头提及，但 spec / arch_design / semantica 均无正本，QA 设计依据自然没有（用户批注未转 oracle）；
- 新盲区类型：**验证环境 ≠ 用户验收环境**（scratch 合规不代表联调库合规）——已追加 `test-design-gate/references/case-library.md` B5 类型与 CASE-002。

**性质分类**：

| 类 | 内容 |
|---|---|
| 文档已写明 | kind 词表 document\|link（arch_design L292、spec L159-160、api.proto L178/865）；document 须 docs root 下存在（artifact.go）；seed expected_outputs 全 document |
| 文档有空白 | 「文档类产物不得 link」无任何条款；期望 kind 是否在登记/推进时校验未定义 |
| 我的推断 | User 意图 = expected_outputs 定义的产物必须 kind=document（与 seed 一致）；link 仅可作 expected_outputs 之外的补充外链（如外部 MR） |

### C. 返工与复验记录

> 2026-09-10 User 拍板：①A、②A、③「可」（注记-K 已落 spec）。

**① 数据修复——已落地（方案 A 微调实施）**：

- docs root 落点：rd :18080 进程 cwd=`rd/`，config 未配 `artifacts.docs_root` → 缺省根=`rd/docs`（不存在）。**未改配置未重启**，建 symlink `rd/docs → ../docs`（`resolveDocsPath` 对 root 每次 EvalSymlinks 归一，symlink 穿透后仍受 root 真实路径越界校验）；
- 入库样例文档 8 份（多级标题/列表供行段锚定）：`docs/v1.0/req-002/{brd/brd.md,prd/prd.md}`、`docs/v1.0/req-003/{brd/brd.md,prd/prd.md,rd/design.md}`、`docs/v1.0/req-004/{brd/brd.md,prd/prd.md,rd/design.md}`（req-003 bus-05 的 rd_design 复用同文件）；
- 数据改写：9 行 `kind='document'` + href 相对路径；audit_log 18 行 UPDATE（field=kind/href，old→new，**by_who=user**）；
- 验证：联调 API 正文回读抽验 5/9 全 OK（22~34 行）。

**② 系统守卫——已落地（方案 A）**：

- 登记口 `rd/service/artifact.go` Add：节点 expected_outputs 对该 product_type 有期望且期望 document 时，kind=link → 400 policyRejected（期望外如 code_mr 不受限）；
- 判据口 `rd/service/task.go` completeMissing：**仅 kind 与期望一致的登记计入判据**；类型已登记但 kind 不符 → 显式缺项 `artifact_kind:<type> 期望 document（已登记为 link）`；推进 exec 判据（execAdvanceReady 数已完成任务）自动同口径——link 冒充物完不成任务，推进保持 blocked；
- 测试资产适配：`qa/qa_drive.py` 新增 **A-15g**（期望 document 拒绝 link 登记，400 blocked）；`qa/qa_live_audit.py` 新增（D-3 巡检机械资产，mysql CLI 零依赖，联调库/scratch 库通用，违规退出码非 0）。

**复测结果（2026-09-10，全 PASS）**：

| 项 | 结果 |
|---|---|
| scratch e2e 全量（qa_env_up + qa_drive） | **196 PASS / 0 FAIL / 1 SKIP**（基线 195 + A-15g；既有用例零破坏） |
| D-3 联调库巡检 | 期望内产物 8 条，违规 0（修复前为 9/9 违规） |
| D-3 scratch 库巡检 | 期望内产物 37 条，违规 0 |
| rd 闸门 | gofmt 清 / vet 清 / go build OK / 全包单测绿 |
| 呈现通道 | API 回读已验；FE 阅读器点开/锚定评论待 User 亲测确认（数据已可达） |

**③ 约束落正本——已拍板落地（2026-09-10 User「可」）**：「落正本」= 把这条规则从「你说过的话」变成**写在 spec 里的条文**。为什么要写：QA 的七源盘点里「用户历史批注」这一源，需要有处可读——规则只存在于对话里，下一轮 QA 盘点时读不到，等于没有；②的守卫代码也需要 spec 条文作为实现依据（代码实现的是文档化的规则，不是口口相传的意图）。**已落 `.plan/wbflow-impl/spec.md` 注记-K**（草案原编号 -J 已被并行 session 的批注轮修复注记占用，顺延为 -K，草案原文一字未动）；`arch_design.md` 同步两处（Artifact 字段说明加约束 bullet；快照示例 href 去 `docs/` 前缀，对齐 api.proto「相对 docs root」口径）。条文正本：

> **注记-K · 产物入库约束（BUG-015 拍板，2026-09-10）**：expected_outputs 期望的产物必须是系统内入库文档——kind=document，href 为相对 docs root 路径且文件真实存在。登记口（wb_add_artifact）：期望 document 的产物以 link 登记 → 400 blocked；判据口（complete / advance exec 同口径）：仅 kind 合规的登记计入判据，kind 不符 → 显式缺项 `artifact_kind:<type>`。link 仅用于期望外的补充外链（如 code_mr）。用户验收环境数据巡检（D-3，`qa/qa_live_audit.py` 对联调库跑同一期望矩阵）违规即回归不通过。

### D. 归属判定

rd（守卫缺失）+ data（联调数据）+ docs（约束无正本）+ qa（设计缺口）。

### E. 最小复现入口

| 字段 | 值 |
|---|---|
| cmd | `mysql -h192.168.3.100 -uroot -p"$QA_MYSQL_PASSWORD" wbflow -e "SELECT requirement_id,node_id,product_type,kind,href FROM artifacts WHERE kind='link'"` |
| 环境变量 | `QA_MYSQL_PASSWORD` |
| 前置 | 联调库可达（无需 env_up） |
| 末次运行输出 | 9 行 link（req-002 ×2 / req-003 ×4 / req-004 ×3），0 行 document |
| 复现率 | 100%（数据态） |
| 预期绿信号 | 该查询 0 行冒充期望 document 的 link；FE 点开 req-004 三产物进阅读器 |

**测试设计缺口回写**：B1（用户批注未转 oracle）+ B2（无呈现通道）+ **B5 新类型（验证环境 ≠ 用户验收环境）** → test-design-gate 增防线「准出前对用户验收环境数据做一次期望矩阵巡检」；to-qa-testcases 骨架 D 组增 D-3「用户验收环境数据合规巡检」。

## 发现源统计（复盘视图，2026-09-10 User 要求分记；同日批注轮 BUG-011~014/OPT-001~003 并入后更新）

> 口径：按「谁先发现问题」归属。User = 用户看板巡检 / 浏览器亲测 / User session 核查；QA 轮 = agent 独立验收驱动（qa_drive.py）与其修复回归轮。

### User 发现（BUG-011~OPT-001 共 9 条 + OBS-001 拍板关闭）

| 编号 | 一句话 | 级别 | 发现渠道 | 现状 |
|---|---|---|---|---|
| BUG-001 | 需求状态与流程推进零联动 | S2/P1 | 看板巡检（req-003 状态矛盾） | 已确认，待拍板修复方案 |
| BUG-011 | 技术评审上下文断裂（评审拿不到被评审产物） | S2/P1 | 批注轮（「重大实现遗漏」） | 新建 |
| BUG-012 | 详情页需求说明缺失（沉底+无 Markdown） | S2/P2 | 批注轮（「漏掉了需求说明」） | 新建 |
| BUG-002 | arch_design.md 未同步注记-A | S3/P2 | User session 核查顺带 | **已修复**（3 文档 11 处 + D-4 巡检固化） |
| BUG-013 | 评审评论可游离不锚文档 | S3/P2 | 批注轮（「评审都挂在文档上」） | 新建 |
| BUG-014 | branch 协作口径过期（铁律⑦） | S3/P2 | 批注轮 | 新建（docs） |
| BUG-010 | 流程树点色不编码门禁结论 | S3/P2 | 浏览器亲测截图反馈 | 已修复 |
| OPT-001 | flow 页左右分栏布局重构 | S4/P2 | 批注轮（「现在的太丑了」） | 新建 |
| OPT-002 / OPT-003 | 评审药丸可点展开对话区 / 变更历史归位 | S4/P3 | User session（批注轮原型对照 diff） | 新建 |
| BUG-015 | 联调库产物 9/9 外链占位，文档交互链用户环境不可达 + 系统无期望 kind 守卫 | **S1/P0** | 批注（「挂载的产物文档不能是外链…req-004 还全是外链」） | **已修复**（①数据②守卫，复测 PASS；③落 spec 注记-K）；设计缺口回写=B1+B2+B5（新类型） |
| （另）OBS-001 由 User 拍板关闭（最小原则口径入 semantica `48c9c4e6`） | — | 观察 | 拍板复核 | 关闭非缺陷 |

### QA 验收轮（agent）发现（7 条，编号 BUG-003~009）

| 编号 | 一句话 | 级别 | 发现手段 | 现状 |
|---|---|---|---|---|
| BUG-003 | plan 流程尾段死锁（create_reqs 不被 exec 判据承认） | S1/P0 | qa_drive.py 黑盒全量（唯一 FAIL 稳定复现） | 已修复 |
| BUG-009 | reproject 幽灵评审位（与 scanner 竞速） | S2/P1 | 修复回归轮（audit_log 实证） | 已修复 |
| BUG-004 | update 强制全量字段 | S3/P2 | A-10 负路径（三类 400 报文） | 新建 |
| BUG-005 | 任务总线不校验节点相位 | S3/P2 | B-3 剧本顺序暴露孤儿任务 | 已修复 |
| BUG-006 | my-task 与 assign 动作标识口径不一 | S3/P2 | A-13c 断言对照 | 新建 |
| BUG-007 | join wait 分支死代码 | S3/P3 | C-8b 分支可达性穷举 | 新建（文档化） |
| BUG-008 | wb_escalate 通知 kind 投影断链 | S3/P2 | scratch 库通知行 DB 对账 | 新建 |

**复盘观察（2026-09-10，含批注轮后更新）**：User 渠道命中「业务语义 / 上下文完整性 / 视觉与交互」类问题（状态联动、评审上下文、点色、布局——看板与真实使用视角，两条 S2 出自 User）；QA 驱动命中「判据完整性 / 接口口径 / 竞速窗口」类问题（全链黑盒 + DB 对账视角）——两条发现渠道互补，缺一不可（S1 死锁仅 QA 驱动能稳定复现）。

## 修复回归结论（2026-09-10）

**195 PASS / 0 FAIL / 1 SKIP**（qa_drive.py 全量，scratch 库 + 最终代码；SKIP=BUG-007 死代码分支）。本轮修复全部回归通过：BUG-001/003/004/005/006/008/009/011/013（rd）+ BUG-010/011fe/012/013fe/OPT-001/002/003（fe，tsc/eslint/vitest 151 用例绿 + vite 热更冒烟）。新增回归断言 9 条（E-1a~h）已固化 qa/qa_drive.py。

## CR 复审处置记录（2026-09-10，两轴并行审查后）

- **回炉已修**：①BUG-001 边界——手动预置「已完成」的需求首次进编排不再被降级「进行中」（终态豁免）；②BUG-011③ `tasks.context.dependencyDocs` 落实现（评审表态任务派发时注入直接上游 produce 段被评审产物；注：现有 seed 评审表态走 wb_review 不经任务总线，注入为任务侧对偶防御路径，无可触发剧本故驱动不断言）；③exec 族判据收敛 `domain.IsExecAdvanceFamily` 单一来源（advance 判据/complete 触发两处同源，防扩族漏改）；④errs.Trace 包裹补齐。
- **误报关闭**：BUG-013「行级裸锚放行」——line/range/block 锚点全局强制 artifact_id（domain/issues.go:263/273/286），评审上下文无裸锚可能；document 级拦截已覆盖唯一缺口。
- **拆单待排**：OPT-002「强行通过/强制修改」leader 裁定动作拆出 **OPT-004**（涉及权限与门禁语义，超出本轮「可点展开+只读线程」范围，arch_design 已改口只读口径为准）。
- **口径销账**：BUG-011①「RelatedArtifactItem 加 nodeId/upstream 标注」按 spec 注记 -J 补记①以 FE 快照反查实现，标注字段不加（单消费方）。
- **范围注记**：工作树中 docs/templates 删除、payload.go TemplatePath 删除、workbench-sql 改写属并行线（standard-authority-refactor）改动，不属本轮缺陷修复范围。

## 待登记对齐

> qa-test-cases.md 头部（2026-09-10）记「User 在并行 session 仍报出 3 个 bug」——该 3 条由验收线并行 session 登记时并入本台账并统一编号；若其中含 req-003 状态矛盾同源问题，并入 BUG-001（关联，不重复计数）。
> 2026-09-10 全量验收轮新增 BUG-003~008 + OBS-001 已按此原则编号并入（与并行 session 的 BUG-001/002 不重复；执行结果见 qa-execution-report.md）。
