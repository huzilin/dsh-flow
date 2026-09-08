# Handoff: wbflow 17 工单实现收官 → CR 循环直至清零

> 2026-09-08 · 前会话：implement-spec 编排 17 张工单全部开发合并完成，进入第 1 轮两轴 code-review。下一会话任务：**收拢 Spec 轴评审 → 派单实现者修复全部发现 → 复审循环直至清零 → 收尾**。

## 事实基座（指针，不复制）

- 实现层规格：`.plan/wbflow-impl/spec.md`（41→40 节点计数已修正；`flow_nodes.last_decision`/`flows.scope` 已由迁移 v3 补齐）
- 工单：`.plan/wbflow-impl/tickets.md`（17/17 验收已勾选）
- PR 分支：`wbflow-impl` @ `27df96f`（worktree `.worktrees/wbflow-impl`），基点 `main` @ `223c65b`；**未推 origin**（远端 GitLab，本机无 glab/gh）
- E2E：`rd/e2e/`（build tag `e2e`），三条剧本真库两轮 PASS，运行方式见 `rd/e2e/README.md`
- 各工单分支：`wbflow-impl-t<N>-<slug>` 扁平名，已全部并入（worktree `.worktrees/t*` 待清理——**收尾阶段做，勿现在删**）

## 环境事实（新会话必读）

- MySQL 8.0.46 @ 192.168.3.100:3306，root 凭据由用户在会话中提供（**本文件不记录密码**；一律 env 注入 `WBFLOW_MYSQL_*`，密码零落仓库，用 `mysql.NewConfig()` 构造 DSN）
- `export GOPROXY=https://goproxy.cn`（proxy.golang.org 不可达，仅 env 勿写 go.mod）
- Docker 不可用（IT 用真实库 + `wbflow_it_*`/`wbflow_e2e_*` scratch 库测后 DROP；**禁写 wbflow 库**）
- 本机 8080 被占（冒烟用 18080）；分支名必须扁平（带 `/` 与 `wbflow-impl` 嵌套冲突）
- subagent 约定：prompt 开头 `Load the "caveman" skill and follow it.`；实现者各占独立 worktree/分支，合并者串行占 `.worktrees/wbflow-impl`；api.proto/生成物冲突=proto 取并集后 `cd docs/api/v1 && buf generate` 重生成，生成物不手解

## 当前状态：CR 第 1 轮进行中

### Standards 轴（已完成）
- 硬违规 1：`rd/service/scanner.go:85` `Scanned int` 违反「int 统一 int64」
- judgement call 8，最重：**capability→`domain.CapabilityInfo` 装配循环同形 4 处**（`service/flow.go:425`/`task.go:539`/`comment.go:585`/`notifier.go:223`）应收拢 helper；其余：取号重试循环 6 处同形可提 `withIDAllocRetry`、`NotificationSpec.TargetType` 覆写无调用方可删、`domain/escalation.go` 的 `*Trigger*` 词族擦边退役词建议改「唤起/触达」措辞、proto 投影 message 与「string 承载」张力判可接受需声明、json tag `ID→Id` 判刻意收敛

### Spec 轴（两个评审 agent 在前会话后台运行，结果不跨会话——**新会话重跑**）
按以下两份 prompt 要点重派（Explore 或 general-purpose，只读）：
- **A（数据层/异常/验收）**：对照 spec §0/§2/§4 核实已知发现 1/3/4/5/10/11（编号见下表）+ 找新问题
- **B（路由表/分层契约/失败 IT）**：对照 spec §1.1/§1.2/§1.3 逐行核对端点（含移出清单 wb_decide/wb_override）+ 核实已知发现 2/6/7/8/9/12/13 + 找新问题

## 已知发现全清单（CR 修复实现者的输入，两轴汇总）

**代码缺陷（应修）**：
1. main.go 无迁移调用（需 boot-migrate 或 `migrate` 子命令）
2. `wb_list_requirements` 未实现 scope 筛选（spec §1.2 注明；`flows.scope` 列已在）
3. 需求 PUT 改 category 后流程实例不重投影
4. T7 两处 IT：断言键 `resolveReviewConfirmed` 应为 `resolveReviewerConfirmed`；`IssueConcurrentReply` 依赖的快照投影缺 replies/resolve 族字段（`service/requirement.go` issueItems）
5. main.go 未装配 `Deps.Reviews`（wb_review/wb_decision 生产路由未注册，一行接线）
6. `align.opening` node action id 含点号，`SplitCapabilityAction` 解析失败恒 400
7. 聚合快照 issueItems 缺 Assignee/ReviewerRole/IssueType（FE 冲突卡消费方受损）
8. 需求 Delete 留孤儿 flows/flow_nodes → req-NNN 复用撞 `uk_flows_req`（处置口径见裁决清单）
9. `TaskUpdate_IT` complete 审计行数断言 7≠6（曾现，未稳定复现）
10. `Artifacts_IdempotentAndConcurrentAlloc` 真库夹具竞态（高负载 flaky：并发登记报 docs root 无 note-0.md）
11. 审计 field 命名混用（issues 状态机 snake vs 其余 camel）
12. AuditAppendOnly IT 高负载偶发一次（观察项）

**judgement call（Standards 轴 8 项，见上节）**。

**提请 User 裁决（CR 只标注不擅改）**：
- A. deprecate 语义：现=有引用 blocked；ui-interaction-final=联动取消（semantica `f7463467`）
- B. 编排接缝三类无 HTTP 端点（exec complete→下一 review 节点推进 / 节点内 dev→review 轮转 / QA 机械收口）——T17 e2e 以 scratch SQL 接缝模拟
- C. RD∥FE 并行组无 join 语义（按渲染序机械收口）
- D. scope 切换无 API 端点（路由表本无此行）
- E. semantica 决策记录：`3ce761b4` FE 栈 / `21cb8f86` seed 口径 / `50aaefad` 凭据与 PR 降级 / `e1acca31` seed 翻译取舍 / `b180101e` category 绑定 / `1fc79a1d` int64-string / `f7463467` deprecate 冲突 / `eb31734e`+`3794085c` ErrPolicyRejected / `62c492a0` issue 状态机取舍 / `47cb6bbb` collect 口径 / `c784503c` FE TS 链 / `19f1d47e` by_who=plugin

## 下一步动作序列（新会话照此执行）

1. 重派 Spec A/B 两轴评审（prompt 要点见上），与已有 Standards 报告聚合（skill：code-review 第 5 步——两轴分列不合并排序）
2. 派**单个**实现者 subagent 修复全部代码缺陷（1-11）+ judgement call 中低风险项（重复收拢/删死代码/Trigger 改词）；裁决项 A-E 不动，写入报告。工作约定同环境事实节；修复分支建议 `wbflow-impl-crfix-r1`（worktree 内直接建分支或新 worktree）
3. 修复后全闸门：`gofmt -l .` / `go build ./...` / `go vet ./...` / `go test ./...` 全绿 + 集中 IT 复跑（`WBFLOW_MYSQL_IT=1 WBFLOW_MYSQL_HOST=192.168.3.100 WBFLOW_MYSQL_PORT=3306 WBFLOW_MYSQL_USER=root WBFLOW_MYSQL_PASSWORD='<凭据 env 注入>' WBFLOW_MYSQL_DBNAME=wbflow go test ./...`，期望 0 失败）+ `go test -tags=e2e ./e2e/` 三剧本 PASS
4. 复审循环：重跑两轴（或抽轴）直至清零
5. 收尾：FE 浏览器联调冒烟（fe dev + rd 起本地栈，browser-use 主线程做；T14/T15/T16 勾选注记「浏览器联调待编排者验收」即此）→ 清理 `.worktrees/t*` → AC 终核 → 向 User 提请裁决 A-E →（若 User 要）推 origin/建 MR

## Suggested skills（下一会话按需调用）

- `implement-spec`——继续编排修复/收尾循环（其第 7 步即当前所在阶段）
- `code-review`——复审轮（基点固定 `main`，两轴并行）
- `caveman`——启动任何 subagent 时注入（用户全局约定）
- `semantica-decision`——裁决项 A-E 拍板后记录
- `browser-use:control-browser`——FE 浏览器冒烟（主线程自己做，勿派 subagent）
