# test-flow（wbflow 第一版）交互设计盘点 + 与当前 dsh-flow fe/ 履约对照

> 取证日期：2026-09-11。只读盘点，未改两个仓库任何代码。
> 一版正本：`/Users/huzilin/tmp_1/test-flow/`（2026-09-06 迁移前项目正本）。
> 一版主体代码：`workbench-plugin/client.js`（829 行，React 单文件 UI）+ `workbench-plugin/host.js`（534 行，领域状态机 + 22 个 wb_* 模型工具）。
> 设计文档：`tmp/docs-tmp/02-module-board.md`（看板）、`03-module-flow-tree.md`（流程树/评论/门禁，权威定稿）、`04-module-inbox-todo.md`（收件箱/TODO，仅设计稿）。
> 冲突裁决：以下一律以代码为准；文档与代码不一致处已标注。
> 当前 fe 根目录：`/Users/huzilin/workdir/dsh-flow/fe/src/`。

---

## A. 交互设计清单（一版，逐条）

### A1. 外层：版本看板 / 需求池（模块 I）

| # | 交互名 | 一版实现方式与代码位置（file:line） | 设计亮点 / 用户看重原因推测 |
|---|---|---|---|
| A1-1 | 看板/列表双视图切换 | client.js:360-361（顶栏「看板」「列表」按钮，选中金色高亮）、512-545 | 一屏切换不丢筛选上下文；看板看进度、列表看全字段 |
| A1-2 | 状态四泳道看板 | client.js:338-339（statusCols/statusColor 常量）、513（泳道渲染：色点+状态名+计数，空列也显容器） | 待启动灰/进行中琥珀/已完成绿/已取消红，颜色语义全站统一 |
| A1-3 | 需求卡要素 | client.js:504-508（标题+类别徽标+优先级徽标+版本名+负责人） | 卡上信息密度刚好够判断要不要点进去 |
| A1-4 | 优先级徽标 P0-P3 | client.js:341（priColor：P0 红/P1 橙/P2 蓝/P3 灰）、506、562（卡/行/详情三处同款） | 同一徽标三处复用，认知零成本 |
| A1-5 | 顶栏筛选：版本×优先级交集 | client.js:343（filter）、350-359（两个 select，「全部」=虚拟选项 __all__） | 看板与列表共用同一过滤结果（client.js:343 单一 filteredReqs） |
| A1-6 | 列表视图：公共表头+按状态分组+虚线分割 | client.js:515-518（表头一次渲染，列 flex basis-0 严格比例，pkg-25）、517（空组隐藏）、537（组间 dashed 分割线）、539（组头色点+「n 项」） | 表头只渲染一次；空状态组整组消失不留白 |
| A1-7 | 卡片/行 hover 出 ✎🗑 | client.js:500-503（卡）、523-526（行）：opacity 0→1 过渡，stopPropagation 不拦截进详情 | 操作藏在 hover 里不占版面；编辑删除都不用进详情页 |
| A1-8 | 删除二次确认 | client.js:401-411（delModal，「删除后不可恢复，确定删除需求「xxx」？」）、146-157 | 删除当前详情需求确认后自动返回看板（client.js:154） |
| A1-9 | 新建/编辑需求弹窗 | client.js:373-400（标题*/描述 Markdown textarea/类别/版本/优先级/状态/负责人+datalist 角色候选）、128-145 | 新建编辑同一弹窗复用（mode 字段）；负责人用 datalist 限制角色集合 |
| A1-10 | 需求信息面板 | client.js:553-568（负责人/版本/优先级/状态/类别/创建/更新时间，kv 行式，pkg-15） | 详情页一眼看清元信息，字段带徽标着色 |
| A1-11 | 描述折叠 | client.js:569-583（>4 行折叠 150px + 「展开全部 ▾ / 收起 ▴」，pkg-16~21）；openDetail 时重置折叠（client.js:82） | 折叠阈值按行数不按字数（pkg-20）；切换需求重置折叠态 |
| A1-12 | 变更历史自然语句 | client.js:788-816（倒序，「谁将「字段」从「X」变更为「Y」」，描述字段特判「谁填写/清空/修改了需求说明」，pkg-23）；host.js:226-248（7 字段 diff 审计） | 审计说人话，不是 JSON diff |
| A1-13 | 版本管理弹窗 | client.js:413-459（版本行：名称+状态徽标+四色进度条+计数+操作钮）、449-452（新建版本输入框 Enter 提交） | 一行一版本，四状态计数 + 进度条段色（client.js:418-421、434 图例点） |
| A1-14 | 版本状态机：单向推进 | client.js:169-186（▶ 推进为进行中/已发布）；host.js:38（VER_STATES）、182-199 | 单向不可回退；按钮文案直接写「推进为X」消除歧义（pkg-12） |
| A1-15 | 版本发布确认 | client.js:169-176 + 476-486（推进为已发布时若版本下有未完成需求，弹确认框列未完成数，pkg-14） | 发布是重大动作，软约束但必须过人手 |
| A1-16 | 版本废弃拦截 | client.js:461-475（有需求引用→只给「知道了」无确认钮）；host.js:200-214（blocked+refCount） | 废弃不可恢复，有引用先移出；无删除接口（pkg-11，废弃是唯一移除手段） |
| A1-17 | 待分配池锁定 | client.js:424（「待分配池」徽标）、432（「池 · 只读」）；host.js:177/188/206（backlog locked） | v-backlog 只可入不可操作，防误动 |
| A1-18 | 版本改名行内确认 | client.js:487-497（改名弹窗，Enter 提交）、199-206 | 待分配池禁改名（host.js:177） |

### A2. 需求详情页：SVG 流程图（模块 II 主体）

| # | 交互名 | 一版实现方式与代码位置 | 设计亮点 / 用户看重原因推测 |
|---|---|---|---|
| A2-1 | 详情页上下分栏 | client.js:824（流程图 flowwrap 在上、详情面板纵列在下，pkg-28）、585（detailTop 返回看板+标题+类别/优先级/状态徽标+编辑钮） | 流程图常驻页首，先看全局再看节点 |
| A2-2 | SVG 横向流程图 | client.js:602-710（renderFlow：节点 118x38 方块、折线连接、左→右布局，pkg-27/28/31/32 三轮调优尺寸） | 沙箱无 import 自绘 SVG；横向读图符合「流程推进」直觉 |
| A2-3 | 并行双泳道 | client.js:610-618（按 parallelGroup 分组）、645-648（branch=rd → topY 上泳道，fe → botY 下泳道）、692+697（RD 泳道蓝 #7ab8ff、FE 泳道绿 #58c07a，节点上方「RD ⇄ 后端」「FE ⇄ 前端」泳道标签） | 颜色+文字双编码泳道归属；pkg-26/30 |
| A2-4 | 组级分叉/汇聚连线 | client.js:666-689（主链→并行组：一分叉多折线；组→主链：多汇聚一；组间 1:1 直连；pkg-29） | 连线语义=控制流分叉汇聚，不是逐对连线 |
| A2-5 | 组内只连同行 | client.js:653-656（同行 y 相等才画水平线）；pkg-30 明确「跨分支（RD/FE）之间不画连线」 | 防止泳道间拉斜线视觉噪声 |
| A2-6 | 节点状态点色 | client.js:600（dotColor：已通过绿/待评审·进行中琥珀/其余灰）+701（节点内圆点） | 一眼看进度；与全站状态色系一致 |
| A2-7 | 选中节点金色描边 | client.js:690-695（isSel：#e8b34a 描边 2.4px + 10% 金底） | 选中态强烈但不刺眼 |
| A2-8 | 回滚角标 ↩N | client.js:703（rolledBackAt.length>0 显红色 ↩N）+713（元信息「回滚 N 次」） | 返工次数直接在图上可见 |
| A2-9 | 横向自动布局 | client.js:619-635（并行组宽=最长分支；单节点组宽=节点宽；x 累加，SVG 宽自适应） | 节点任意多不重叠，容器横向滚动（client.js:711 overflowX） |
| A2-10 | 点节点切详情 | client.js:700（rect onClick onSelect）、711（选中并收起展开的评审角色区） | 图即导航 |

### A3. 节点详情面板（右/下列）

| # | 交互名 | 一版实现方式与代码位置 | 设计亮点 / 用户看重原因推测 |
|---|---|---|---|
| A3-1 | 节点元信息卡 | client.js:713（节点名大字+负责人/进入时间/回滚次数） | docs 03 §3.2 四区之首 |
| A3-2 | 角色药丸（评审状态行） | client.js:724（每 reviewer 一药丸：状态点+角色名+N未解决；绿=已通过/红=有 open/灰=无；**轮次≥10 且未解决 → 琥珀加粗「轮N」卡死高亮**） | docs 03 §3.3「QA ⚠轮14」形态；拉锯一眼可见 |
| A3-3 | 点药丸展开角色对话区 | client.js:726（openRole 切换，展开该角色全部评论串） | 意见↔反馈按角色聚合，不混时间线 |
| A3-4 | 强行通过 override | client.js:726 底部「强行通过 X 全部剩余」红钮；client.js:109 doOverrideRole；host.js:370（整角色 open→overridden） | User 三态参与之强权态；绕过门禁但留痕 |
| A3-5 | 产物列表 | client.js:715-719（文件名蓝字+note+💬N 评论数；无产物显占位「该节点无产物」；kind=link 新窗口不进阅读器，pkg-33） | 💬N 预告评论密度再决定点不点 |
| A3-6 | 门禁进度+拦截 | client.js:730-735（「评论解决进度 已解决/总数」+（未全部解决，「通过」被拦截）提示；通过钮置灰 not-allowed + 文案变「通过（被拦截）」） | 门禁=全部评论闭环（host.js:386-392 三重拦截：open/pending-confirm/评审者未通过）；置灰而非隐藏，用户知道为什么不能过 |
| A3-7 | 打回表单 | client.js:733（「打回（带理由）」展开 textarea 必填理由 + 「需重新界定新范围」checkbox → host.js:429 pendingScope） | 打回理由必填+新范围界定标记 |
| A3-8 | 全量快照刷新 | 所有 wb.* 写操作返回最新聚合 snapshot 直接 setReq（client.js:103-108 等）；host.js:50 loadRequirement | 前端无本地状态机，永远渲染服务端真相 |

### A4. 产物阅读器与锚点评论

| # | 交互名 | 一版实现方式与代码位置 | 设计亮点 / 用户看重原因推测 |
|---|---|---|---|
| A4-1 | 阅读器大弹窗 | client.js:785（72vw×85vh，正文满宽无右侧栏，底部固定输入框，点遮罩关闭） | docs 03 §3.5 飞书式 |
| A4-2 | 轻量 md 渲染（阅读器内） | client.js:208-336（renderMd/renderInline/renderTable/renderLineContent：标题 1-6 级放大、粗体、行内代码、围栏代码块蓝左边条+语言标签、表格、▪列表、链接新窗口；pkg-17 pre-wrap） | 文档可读性；mermaid 按代码块展示不画图（docs 03 §3.5） |
| A4-3 | 行号 + 整行可点 | client.js:772-779（1-based 行号、整行 hover/选中反馈） | 行号=锚点坐标系 |
| A4-4 | 四级锚点 | client.js:112-124（doComment：block/line/range/document 四级锚构造）；host.js:492（wb_comment anchor 参数） | docs 03 §6-3 定稿四级；heading 级已废弃 |
| A4-5 | 行选择四规则 | client.js:123（pickLine）：单击=单行选中；点其他行=框选区间；重复点起点=取消；已有区间再点任意行=重新开始 | docs 03 §3.6 权威规则；选中黄高亮（client.js:776-778） |
| A4-6 | 代码块整块评论 | client.js:757-770（围栏扫描，点块=选整块，金边高亮；「评论代码块」钮） | 对一整段代码提意见不必逐行框 |
| A4-7 | 锚定评论就地渲染 | line=行下方内嵌（client.js:779）；range=只在末行下方一次+「↔ A~B 行」金标，点标高亮整范围（client.js:236+775 highlightRange）；block=块下方（client.js:770）；document=正文末尾（client.js:783） | docs 03 §3.7 展示位置表 |
| A4-8 | 有评论行高亮 | client.js:777-778（浅蓝底+左蓝条；选中黄条优先） | 扫一眼定位讨论热点 |
| A4-9 | 输入框文案随选中态变化 | client.js:784（「对整个代码块评论」/「对第 N 行评论」/「对第 A~B 行评论」/「写文档评论…（点正文行可选）」） | 所见即所评，防锚点发错 |
| A4-10 | 评论卡状态体系 | client.js:250+237-238（未解决红/待复核琥珀/已解决绿/已强制通过金；类型标：通知=蓝/讨论=灰；轮次标 N 轮，>5 轮红色；超时 ⏱ 标：pending-confirm 超阈值红框） | 状态、类型、轮次、超时四维一卡打尽 |
| A4-11 | 已解决评论折叠 | client.js:245-247（resolved/overridden 默认折叠一行摘要：作者+截断文本+状态标+▸）、252（展开后「收起 ▴」） | open 态常驻、closed 态收纳，列表不爆炸 |
| A4-12 | thread 回复 | client.js:249（左边线缩进 [by] text）+105 doReply+host.js:311（roundCount++ 并 pushInbox 通知作者） | 拉锯轮次计数是门禁与卡死判定的数据源 |
| A4-13 | resolve 双路径 | client.js:257-268（「解决(修复)」绿/「解决(一致)」蓝，note 必填输入框）；host.js:312-333（fix→pending-confirm 通知提评者；agree→直接 resolved） | docs 03 §3.7 判定矩阵：reply=还想说；resolve=要关；confirm/reopen=提评者终裁 |
| A4-14 | 复核 confirm/reopen | client.js:262-264+270-273（复核通过绿钮/复核不通过红钮+理由必填）；host.js:334-369（reopen→open+roundCount++；>5 轮 pushInbox gate 请 Leader 介入） | 修复关闭必须提评者复核才放行门禁 |
| A4-15 | resolve 信息行 | client.js:253-256（[协商一致/修复] note — by） | 关闭原因留痕可展开回看 |
| A4-16 | 阅读器筛选条 | client.js:748-750（作者药丸：全部/各作者/隐藏；状态药丸：全部/未解决/已解决） | 大文档评论多时按人/按状态过滤 |
| A4-17 | 行级锚点防漂移（设计稿） | docs 03 §4.4：line+lineText 去空白原文指纹，四步定位，失效标「⚠ 锚点可能失效」。**一版 client.js 未实现渲染层反查**（仅数据层 lineText 字段传入） | 文档有设计、代码未落；属半成品 |

### A5. 状态机 / 通知 / agent 协同（host 半）

| # | 交互名 | 一版实现方式与代码位置 | 设计亮点 / 用户看重原因推测 |
|---|---|---|---|
| A5-1 | 收件箱（数据层） | host.js:146-154 pushInbox + 154 wb_list_inbox；触发点：评论@（277）、评审表态（308）、修复请复核（330）、拉锯>5 介入（365）、复核超时（523）、节点推进（409）、部分打回（423） | 通知=领域事件副作用，UI 与 agent 工具同源 |
| A5-2 | 复核超时轮询 | host.js:509-530（每 5 分钟轮询 pending-confirm；User 阈值 1h / agent 5min，remindedAt 防重）+ client.js:239-244（UI ⏱ 超时标同阈值） | 人慢机器快，阈值差异化 |
| A5-3 | 并行组 join 判定 | host.js:395-404（组内全分支已通过才放行下一节点，否则 current 停组内下一个未通过分支） | pkg-37 M7 状态机；图上汇聚与数据层 join 同口径 |
| A5-4 | 部分回滚 targetBranch | host.js:412-423（reject+targetBranch=rd|fe：仅回滚该分支，另一分支保持已通过，rollBackAt/lastDecision 落分支，通知对应方） | QA 打回只需返工一方——并行不互相拖死 |
| A5-5 | wb_my_task 任务上下文 | host.js:54-143（ROLE_PROTOCOLS 11 角色 + NODE_OUTPUTS + sessionsByNode：每节点全会话含文档锚定会话打包，条条带 nodeId/nodeKey 归属） | subagent 派活即拿全上下文，不用自行检索 |
| A5-6 | UI 与 agent 工具同源 | host.js:439-463（22 个 wb.* RPC）+ 479-502（同名 wb_* 模型工具）共用 op* 领域函数；破坏性工具描述标「需 User 明确指令」 | 同一审计/门禁/状态机；UI 点的和 agent 调的必然一致 |

### A6. 线索核对备注

- 任务线索中「scope 切换动态渲染」：**一版代码无 scope 概念**（client.js/host.js 全文无 scope；一版动态渲染指类别/版本筛选联动）。scope 是 dsh-flow 新概念，fe 侧已自行实现（见 B 表 B-19）。
- 「四级锚点评论」：代码与文档均证实（A4-4）。
- docs 04（收件箱 UI、TODO 列表）**仅设计稿，一版 client.js 未实现收件箱 UI**（只有数据层+超时标），fe 的 InboxPage 属超出 Proto 的新增。

---

## B. 与当前 fe/ 履约对照表

判定口径：已履约（交互实质存在，允许形式差异）/ 部分履约（核心在但关键形态缺失）/ 未履约（无对应实现）。标注「刻意裁剪」= fe 有注释证明是有意决策而非遗漏。

| 交互名 | 一版形态 | 当前 fe 现状（组件 file:line） | 判定 |
|---|---|---|---|
| B-1 看板/列表双视图 | 顶栏切换钮 | pages/BoardPage.tsx:118-135（sbtn on 态）| 已履约 |
| B-2 状态四泳道看板 | 色点+状态名+计数列 | BoardPage.tsx:148-184（lane/lane-head/lane-dot+计数，空列显「空」）| 已履约 |
| B-3 需求卡要素 | 标题+类别+优先级+版本+负责人 | BoardPage.tsx:164-177（+scope 徽标+依赖角标，超出一版）| 已履约 |
| B-4 优先级徽标三处同款 | 卡/行/详情 | components/Badges.tsx PriorityBadge；BoardPage.tsx:170、RequirementDetailPage.tsx:142 | 已履约 |
| B-5 版本×优先级筛选交集 | 顶栏双 select | components/FilterBar.tsx:9（版本×优先级×状态，pool/board 共用）| 已履约 |
| B-6 列表分组+虚线分割+空组隐藏 | 表头一次+组间 dashed | BoardPage.tsx:186-222、259-309（组头 colspan 色系 GROUP_HEAD_CLASS:24-29、空组 return null:203）；虚线分隔未见（组头色块替代）| 已履约 |
| B-7 hover ✎🗑 双操作 | 卡/行 hover 出编辑+删除 | BoardPage.tsx:88-106 仅 ✎（card-edit）；🗑 只在详情页（RequirementDetailPage.tsx:237-241）；列表行无 hover 操作 | 部分履约 |
| B-8 删除二次确认+自动返回 | delModal | RequirementDetailPage.tsx:242-255（Modal 确认+onSuccess navigate("/pool")）| 已履约（入口位置不同） |
| B-9 新建/编辑同弹窗 | formModal mode 复用 | components/CreateRequirementDialog.tsx（RequirementModal mode="edit" 复用于 BoardPage:227、DetailPage:185）| 已履约 |
| B-10 需求信息面板 | kv 行式七字段 | RequirementDetailPage.tsx:133-152（detail-grid 九字段+操作身份）| 已履约 |
| B-11 描述折叠（>4 行 150px+重置） | 行数阈值折叠+切换重置 | RequirementDetailPage.tsx:265-292（RequirementBriefCard：手风琴展开/收起，有描述默认展开，无行数阈值、无自动 150px、无切换重置）| 部分履约 |
| B-12 变更历史自然语句 | 倒序自然语句面板 | RequirementDetailPage.tsx:302-325 + domain/historyview.ts describeHistory（自然语句+by+时间，折叠区呈现）| 已履约 |
| B-13 版本管理（进度条/推进/改名/新建） | 弹窗四色进度条 | pages/VersionsPage.tsx:69（四色进度条 vbar）、275-322（推进/改名/废弃/终态）| 已履约（页面替代弹窗） |
| B-14 版本发布确认 | 未完成需求弹确认框 | VersionsPage.tsx:242-244（发布软约束提示条列 unfinished.length）| 部分履约（提示条 vs 强确认弹窗） |
| B-15 版本废弃拦截 | blocked+refCount 禁止 | VersionsPage.tsx:410-412（确认弹窗注明拦截规则，blocked 错误原样呈现）| 已履约 |
| B-16 待分配池锁定 | 只读徽标 | VersionsPage.tsx:275、288（「待分配池锁定，无操作」）| 已履约 |
| B-17 详情页上下分栏 | 流程图在上详情在下 | RequirementDetailPage.tsx:160-182（flow-detail-columns 左右两栏：左 FlowTree 常驻+右 NodeDetail，注释引 OPT-001；窄屏退化堆叠）| 部分履约（改为左右栏，属 fe 自主决策非缺失，但与用户多次点名的「上下分栏」形态不符） |
| B-18 SVG 流程图 | 横向（左→右）折线图 | components/FlowTree.tsx:56-123（**纵向**单列 460px 宽，垂直 linkIn 短线）| 未履约（方向相反：一版横向读图） |
| B-19 并行双泳道 RD 上蓝/FE 下绿 | 上蓝下绿+泳道标签 | FlowTree.tsx:89-121（并行组一行两列并排+join 横线）；泳道卡无 RD 蓝/FE 绿分支着色、无「RD ⇄ 后端」泳道标签（badge-scope-rd/fe 蓝绿只用于 scope 徽标，index.css:174-179）| 部分履约（有双泳道结构，无色彩/标签语义） |
| B-20 组级分叉/汇聚连线 | 主链分叉→组、组汇聚→主链折线 | FlowTree.tsx:66-71（仅垂直短线 linkIn）+116-119（join 水平线）；无分叉/汇聚折线路径 | 未履约 |
| B-21 组内不画跨分支连线 | 同 y 才连线 | FlowTree.tsx 无跨泳道连线（结构上天然满足）| 已履约（结构等价） |
| B-22 节点状态点色 | 绿/琥珀/灰 | FlowTree.tsx:152-155、186-191（phase-dot：gate 结论优先+相位兜底，绿/琥珀/红——含「已打回」红，超出一版）| 已履约（语义升级） |
| B-23 选中金色描边 | 金描边+金底 | FlowTree.tsx:183-184（is-selected class）+index.css:540 | 已履约 |
| B-24 回滚角标 ↩N | 图上红 ↩N+元信息次数 | FlowTree.tsx:230（nodeSubLine ↩N）+RequirementDetailPage.tsx:300（回滚次数）| 已履约 |
| B-25 点节点切详情 | 图即导航 | FlowTree.tsx:164（onClick onSelectNode）→RequirementDetailPage effectiveNodeId | 已履约 |
| B-26 节点元信息卡 | 负责人/进入时间/回滚 | RequirementDetailPage.tsx:293-301 + NodeDetail.tsx:133-153（① 状态卡：相位+执行动作+表态计数+进入时间）| 已履约（信息更全） |
| B-27 角色药丸+轮≥10 卡死高亮 | 药丸含状态点+N未解决+「轮N」琥珀高亮 | NodeDetail.tsx:155-162、373-439（ReviewPills：ReviewStatePill 可点）；**轮≥10 卡死高亮不在药丸上**，在①状态卡 badge-stall（NodeDetail.tsx:143-148）；药丸上无「N 未解决」计数（openCount 显示在药丸行下方文字：NodeDetail.tsx:417）| 部分履约（信息都在，药丸本体信息密度低于一版） |
| B-28 点药丸展开角色意见区 | 展开该角色评论串+强权钮 | NodeDetail.tsx:391-436（OPT-002：展开该角色 issues 只读，讨论引导走回复通道）；无强权钮（见 B-29）| 部分履约（只读，一版可在对话区内直接回复/resolve/override） |
| B-29 强行通过 override（角色级） | 对话区底部「强行通过 X 全部剩余」 | 无 wb.override。components/ConflictCard.tsx:12 注释明示「取代 override 强权」：escalate → wb_adjudicate uphold/override 表单（ConflictCard.tsx:118-160）| 已履约（刻意重构：强权改走冲突裁决卡，语义更重但路径变长） |
| B-30 产物列表+💬N+占位+link 外链 | 文件行+note+评论数+空占位 | NodeDetail.tsx:264-347（ArtifactGroups：评审节点分「被评审产物（上游）/本节点产物」两段——BUG-011 升级；ArtifactRowList：label+productType+外链徽标+💬N 阅读+note；空占位:280、296）| 已履约（升级） |
| B-31 门禁进度+通过置灰 | 已解决/总数+置灰+拦截原因文案 | DecisionCard.tsx:53-77（表态 N/M+工单统计+「拍板置灰」文案）；置灰口径改=评审轮未齐（user_gate 终审制，DecisionActions 通过钮不置灰但打回必填理由）| 已履约（门禁模型重构为评审轮聚合，非评论闭环直算——后端口径） |
| B-32 打回必填理由+新范围 checkbox | 理由必填+「需重新界定新范围」 | DecisionCard.tsx:214-230（打回 reason 必填 400 拦截+评审敲定分支名可选）；「重新界定新范围」checkbox 无（由 T9 重入语义承接，DecisionResult:160-164 rejected 重入提示）| 部分履约（理由必约已履约；新范围标记被重入语义替代，无显式 UI） |
| B-33 阅读器大弹窗+底部输入框 | 72vw×85vh 满宽+固定输入 | components/ArtifactReader.tsx:116-263（reader-modal+reader-footer 固定输入+点遮罩关闭）| 已履约 |
| B-34 阅读器内 md 渲染 | 轻量 md（标题/粗体/代码块/表格） | ArtifactReader.tsx:17-19 刻意不做（行号直映原文是锚点防漂移前提）；MarkdownView.tsx 仅用于描述卡 | 未履约（刻意裁剪，理由成立但阅读体验低于一版——值得复核取舍） |
| B-35 行号+整行可点 | 1-based 行号 | ArtifactReader.tsx:131-169（line-no+整行 onClick）+懒加载分段 200 行/段（一版一次全渲）| 已履约（形式不同：懒加载 vs 全量） |
| B-36 四级锚点 | document/line/range/block | ArtifactReader.tsx:216-227（**仅 range/document 两级**；line/block 后端 400 废弃，注记-L3 定稿；单行=range(n,n)）| 部分履约（刻意裁剪：新定稿两级；与一版四级形态不兼容） |
| B-37 行选择四规则 | 单击/框选/重复取消/重开 | ArtifactReader.tsx:142-157（注释引 OPT-005「对齐老阅读器手感」，四规则逐条实现）| 已履约 |
| B-38 代码块整块评论 | 点块选块+金边 | ArtifactReader.tsx:61+132+162（scanFenceBlocks+fence-mark 仅视觉标记，无选块钮——block 锚已废弃）| 未履约（随 block 锚裁剪） |
| B-39 锚定评论就地渲染 | line 行下/range 末行+↔标/block 块下/document 末尾 | ArtifactReader.tsx:288-314（AnchorCommentsAt：line/range 末行下）+317-347（document 级末尾 CommentThread）；「↔ A~B 行」可点高亮标记无（锚点摘要在 CommentCard.tsx:75-83 anchorSummary 文本）| 部分履约（就位渲染有；range 反向高亮交互无） |
| B-40 有评论行高亮 | 浅蓝底+左蓝条 | ArtifactReader.tsx:136-140（in-selection 黄高亮）；有评论行蓝底高亮无（仅 fence-start class）| 部分履约（选中高亮有，评论热点高亮无） |
| B-41 输入框文案随选中变化 | 四态 placeholder | ArtifactReader.tsx:201-205 + 195 selectionLabel | 已履约 |
| B-42 评论卡状态/类型/轮次/超时四标 | 状态+类型+轮次（>5 红）+⏱超时 | CommentCard.tsx:64-73（IssueStateBadge+issueType 徽标+escalate 徽标+轮 roundCount+回退 loopCount）；轮次>5 变红无；超时 ⏱ 标无 | 部分履约 |
| B-43 已解决评论折叠一行摘要 | 折叠+展开/收起 | CommentCard.tsx 无折叠（所有状态全展开渲染）| 未履约 |
| B-44 thread 回复（左边线） | 左边线+[by] text | CommentCard.tsx:93-103（comment-replies 列表+voided 作废样式）；回复含误发作废 checkbox（:156-163，超出一版「reply 标注作废」设计稿）| 已履约 |
| B-45 resolve 双路径+复核 confirm/reopen | 修复/一致+复核通过/不通过 | CommentCard.tsx:105-145（fix/agree/复核确认关闭/reopen，按身份收敛可见性 canResolve/canConfirm/canReopen）+hooks/queries.ts 四 mutation | 已履约（升级） |
| B-46 resolve 信息行 | [路径] note — by | CommentCard.tsx:85-91（resolvePath 标签+resolveBy+resolveNote+复核确认人）| 已履约 |
| B-47 阅读器作者/状态筛选条 | 作者药丸+状态药丸 | ArtifactReader.tsx 无（全量按时间渲染）| 未履约 |
| B-48 收件箱 UI | 一版无 UI（仅数据层+超时标） | pages/InboxPage.tsx:67-69（待拍板/待澄清/待重表态 TODO 投影）、95-97（待拍板提醒条）、107（轮询 5s 未读优先）、118-154（类型/角色/已读三筛选）、250（冲突入口聚合）| 已履约（fe 超出一版） |
| B-49 复核超时提醒（User 1h/agent 5min） | host 轮询+UI ⏱ | fe 端无对应 UI 标；后端是否轮询未在本次范围（be/ 未盘点）| 未履约（fe 侧） |
| B-50 部分回滚 targetBranch 交互 | reject+targetBranch 打回单分支 | DecisionActions 有 branch 输入但语义=「评审敲定分支名」（DecisionCard.tsx:231-235），非「部分打回指定分支」；DecisionResult 显示 decision.branch（:158）| 部分履约（数据有分支位，无「只打回 RD/FE 一方」显式交互） |
| B-51 wb_my_task 上下文 | agent 工具 | hooks/queries.ts useMyTaskQuery；RequirementDetailPage.tsx:79 + NodeDetail activeActions 投影（NodeDetail.tsx:350-366）| 已履约 |
| B-52 scope 切换动态渲染 | 一版无此概念 | fe：FlowTree.tsx:44-50（当前渲染 scope 徽标+被裁节点不渲染不占位）、api/constants.ts sessionFindRoleForScope、RequirementDetailPage.tsx:121-129（会话找 X 随 scope 动态）| 已履约（fe 自有设计，超出 Proto） |

统计：52 项对照——已履约 33、部分履约 11、未履约 8。

---

## C. 值得回归的设计点 TOP（按用户多轮点名 × 缺口大小排序）

1. **SVG 横向流程图 + 组级分叉/汇聚折线 + RD 上蓝/FE 下绿泳道标签**（A2-2/3/4/5；B-18/19/20 未/部分履约）——用户点名的主体。fe 现为纵向单列+短竖线，读图时序感与并行语义弱一截。回归要点：左→右布局、并行组上下双泳道、分支色（rd 蓝/fe 绿）+泳道标签、主链↔组之间折线分叉/汇聚、组内只连同行。
2. **已解决评论默认折叠一行摘要**（A4-11；B-43 未履约）——评论区长列表的核心降噪手段；open 常驻、closed 一行摘要+展开收起。
3. **阅读器轻量 md 渲染**（A4-2；B-34 未履约但属刻意裁剪）——fe 以锚点防漂移为由行号裸渲。一版证明二者可兼得（按行渲染+行内 md 样式）。建议复核取舍：至少标题/代码块视觉分层。
4. **阅读器作者/状态筛选药丸**（A4-16；B-47 未履约）——评审意见多时按人过滤是刚需；fe 意见区/阅读器均无过滤。
5. **角色药丸信息密度**（A3-2；B-27 部分履约）——「N 未解决」与「轮N≥10 卡死」高亮直接上药丸（fe 分散在药丸行下方文字与状态卡），一版单药丸自足。
6. **描述 >4 行自动折叠 150px + 切换重置**（A1-11；B-11 部分履约）——fe 手风琴无行数阈值，长描述默认全展开拉长详情页。
7. **评论区轮次>5 红 + 待复核 ⏱ 超时标**（A4-10、A5-2；B-42/49 部分履约）——拉锯可视化的时间维度，fe 只有轮次数无阈值着色/超时提示。
8. **打回「需重新界定新范围」显式 checkbox**（A3-7；B-32 部分履约）——fe 用 T9 重入语义承接但无显式 UI 标记「本次打回需重新界定范围」。
9. **hover ✎🗑 双操作**（A1-7；B-7 部分履约）——看板卡片/列表行删除入口缺失，删一条要先点进详情。
10. **range 评论「↔ A~B 行」标记点击反高亮整个范围**（A4-7；B-39 部分履约）——从评论跳回原文上下文的一键路径。

另注（非缺口、属 fe 超出一版的增强，回归时勿回退）：评审产物上游/本节点分段（BUG-011）、scope 三色徽标与被裁节点渲染、收件箱 TODO 投影与待拍板提醒条、冲突裁决卡（取代 override）、误发回复作废、乐观锁 409 冲突提示、拍板幂等 no-op 提示。

---

## 附：取证文件清单

一版：`/Users/huzilin/tmp_1/test-flow/workbench-plugin/client.js`、`host.js`、`README.md`；`/Users/huzilin/tmp_1/test-flow/tmp/docs-tmp/02-module-board.md`、`03-module-flow-tree.md`、`04-module-inbox-todo.md`（05-domain-model.md 为领域真相源未逐行盘点）。workbench-sql/（schema.sql/ER.md/seed）为 sqlite 迁移素材，数据模型与一版 JSON 同构，未影响交互结论；data/ 目录为空；tools/ 为会话日志修复脚本，与本盘点无关。
当前 fe：`/Users/huzilin/workdir/dsh-flow/fe/src/` 下 pages/BoardPage.tsx、RequirementDetailPage.tsx、VersionsPage.tsx、InboxPage.tsx、RequirementPoolPage.tsx、components/FlowTree.tsx、NodeDetail.tsx、ArtifactReader.tsx、CommentCard.tsx、DecisionCard.tsx、ConflictCard.tsx、CreateRequirementDialog.tsx、FilterBar.tsx、MarkdownView.tsx、domain/flowview.ts、historyview.ts、index.css。
