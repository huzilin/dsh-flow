# wbflow E2E 演练（T17，build tag `e2e`）

roadmap P5「seed 注入 → HTTP 驱动全流程」：真实 MySQL **scratch 库**（`wbflow_e2e_*` 前缀，
每条剧本每次运行新建 → 迁移+seed → 演练 → DROP，禁写 `wbflow` 库，天然可重复执行）+
`httptest.NewServer` 启动完整 Gin 栈（26 个 `wb_*` 端点，T1-T16 全量装配）+ 真实 HTTP
client（Bearer token）。演练者扮演全部 agent/User 角色，按节点剧本 HTTP 驱动推进。

## 运行命令

```bash
cd rd
GOPROXY=https://goproxy.cn \
WBFLOW_E2E=1 \
WBFLOW_MYSQL_HOST=192.168.3.100 \
WBFLOW_MYSQL_PORT=3306 \
WBFLOW_MYSQL_USER=root \
WBFLOW_MYSQL_PASSWORD='<密码>' \
go test -tags=e2e ./e2e/ -run TestE2E -v
```

- env 清单：`WBFLOW_E2E=1`（门控，缺省 skip）；`WBFLOW_MYSQL_HOST/PORT/USER/PASSWORD`
  （HOST/PORT/USER 有缺省值，密码必注入，零落地）。
- 日常 `go build ./... && go vet ./... && go test ./...` **不受影响**：本目录全部文件挂
  `//go:build e2e`，无 tag 时整包排除。
- 单条剧本：`-run TestE2E_BusinessFull_Flow | TestE2E_BusinessRDOnly_Flow | TestE2E_Task_Flow`。

## 剧本索引（步骤一览）

共同步骤词汇：`decision`=User 拍板（wb_decision）；`produce`=派发→ack→progress→登记
期望产物→complete（wb_assign_task / wb_task_update / wb_add_artifact）；`review`=wb_review
per-agent 表态（agree/unagree，collect 判定）；`clarify`=澄清循环（wb_comment 建意见 →
owner wb_resolve 闭环 → respond complete → review 重入新轮 re_review）。

### TestE2E_BusinessFull_Flow（business × full，13 节点 bus-00→bus-12）

| # | 节点 | 步骤 |
|---|---|---|
| 1 | bus-00 待启动 | decision approved（确认投入资源） |
| 2 | bus-01 需求对齐 | entry hold → produce to-brd（brd）→ decision approved |
| 3 | bus-02 产品设计 | produce product-design.produce（product_doc）→ 提评接缝 |
| 4 | bus-03 产品设计评审 | arch/rd agree；fe 建 open 意见+unagree（collect=clarify）→ pm 澄清闭环（re_review）→ fe 复核 agree（agreed）→ decision approved |
| 5 | bus-04 RD 技术设计 | produce backend-design.produce（rd_design）→ 提评接缝 |
| 6 | bus-05 RD 技术评审 | arch/fe/qa/pm 四位 agree（agreed）→ decision approved |
| 7 | bus-06 FE 技术设计 | produce frontend-design.produce（fe_design）→ 提评接缝 |
| 8 | bus-07 FE 设计评审 | qa 单位 agree（user_gate=0 机械 approved，即时推进） |
| 9 | bus-08 RD 开发 | produce backend-dev.dev（dev_notes+code_mr）→ 节点内 CR 轮转接缝 → cr agree（机械 approved） |
| 10 | bus-09 FE 开发 | 同构（frontend-dev.dev → cr agree） |
| 11 | bus-10 QA 验收 | produce acceptance-test.accept（test_result+defect）→ 收口接缝 |
| 12 | bus-11 用户验收 | decision approved → bus-12 完成（尾节点驻留） |

终态断言：flows.current_node_id=bus-12、各 gate last_decision、审计条数对账
（tasks/artifacts INSERT 与步骤计数、plugin 编排迁移与 user 拍板留痕）、收件箱
gate_pending×2 / clarify_task×1 / re_review×1、TODO 投影不残留。

### TestE2E_BusinessRDOnly_Flow（business × rd，9 节点）

节点集 bus-00/01/02/03/04/05/08/11/12；断言 FE 侧（bus-06/07/09）与 QA 验收（bus-10）
不渲染不占位（node_order 连续 1..9）、bus-03/bus-05 评审位 fe 位被裁。步骤同上（缺 FE
侧与 QA），走通至 bus-11 用户验收 decision approved。

### TestE2E_Task_Flow（task × rd，7 节点无 QA）

| # | 节点 | 步骤 |
|---|---|---|
| 1 | tsk-c0 待启动 | decision approved |
| 2 | tsk-c1 需求对齐 | entry 迎宾投影（align.opening，响应者 executor）；exec 跳过（D2 无产物）→ decision approved |
| 3 | tsk-c2 技术设计 | produce backend-design.produce（rd_design，scope=rd 激活）→ 提评接缝 |
| 4 | tsk-c3 技术评审 | arch 单位 agree（agreed）→ decision approved |
| 5 | tsk-c4 开发 | produce backend-dev.dev（dev_notes+code_mr）→ 节点内 CR 轮转接缝 → cr agree（机械 approved） |
| 6 | tsk-c6 用户验收 | **报障直连断言**：issue 挂 exec 节点 tsk-c4（无显式指派）→ assignee=rd（T7 L2 推导）→ rd wb_resolve 闭环 → decision approved |
| 7 | tsk-c7 完成 | 尾节点驻留 |

## 编排接缝口径（重要）

exec 完成后的三类节点轮转——**提评迁移（exec→review / exec→gate entry）**、**节点内 CR
轮转（exec→review 同节点）**——属 plugin 编排职责，本期无 HTTP 端点承载。演练以 scratch
库内 SQL 接缝模拟（T9/T12 IT 同口径），激活口径复用 `domain.ActivationPhase` /
`domain.InitReviewRound`（与 service 层 `advanceApprovedNode` 一致）。业务推进断言全部经
HTTP 出参与聚合快照，不依赖接缝内部数据。
