# Handoff: wbflow CR 清零 → 收尾（裁决 + FE 冒烟 + 合并推送）

> 2026-09-09 · 前会话：CR 第 1 轮修复（17 项）+ 两轴复审**清零达成**（Spec-A 17/17 真修复零新问题；Spec-B 路由/契约/spec 对齐全过；spec 残留 R1-R6 已由主会话修正）。下一会话任务：**User 裁决 → 合并 crfix 分支 → FE 冒烟 → 清理 → 推送**。

## 事实基座（指针，不复制）

- 实现层规格：`.plan/wbflow-impl/spec.md`（已含复审修正：路由表 26 工具/40 节点/flows.scope DDL 回填/request-error 归属标注——commits `3e09e57`）
- 工单：`.plan/wbflow-impl/tickets.md`（17/17 验收勾选）
- **CR 修复分支：`wbflow-impl-crfix-r1` @ `1faf45e`**（worktree `.worktrees/wbflow-impl`，基于 wbflow-impl @ 27df96f，9 commits 未合回）
- 闸门（前会话实现者跑，主会话独立复核代码闸门）：gofmt/build/vet/单测 10 包全绿 ✅；真库 IT 29/29 全绿 ✅；E2E 三剧本 PASS（full 68.5s / rd-only 61.5s / task 56.9s）✅；scratch 库已 DROP 零残留
- 上一轮 handoff（`.plan/handoffs/2026-09-08-wbflow-impl-cr-loop.md`）中「当前状态：CR 第 1 轮进行中」一节已全部完成，**以本文件为准**

## CR 第 1 轮成果摘要（17 项修复，8 commits `9d9c060..6987187`）

- blocker/major：main.go migrate 子命令 + Reviews 装配 / 快照 issues 复用 issueItem + 断言键 / Delete 补删 flows / category 改 blocked 拒绝 / scope 筛选四层（proto scope=7）/ 并发夹具落盘
- minor：progress 同值 noop / 审计 field 全 camel（数据层列名 loop_count 合法保留）/ SQL IF NOT EXISTS ×14 / reproject 校验前置 / 装配守护测试（反射）/ 非锁读注记
- Standards：buildCapabilityInfoMap（4 处收拢）/ withIDAllocRetry（6 处）/ TargetType 死代码删 / *Trigger*→唤起措辞（全库零残留）
- 复审判定：17/17 真修复；「表面修补/引入新问题」均为零；测试卫生（.gocache 误提交已删净 gitignore）

## 待 User 裁决（9 项，CR 只标注未擅改）

**原 A-E**：
- **A** deprecate 语义：现=有引用 blocked；ui-interaction-final=联动取消（semantica `f7463467`）——二选一
- **B** 编排接缝三类无 HTTP 端点（exec complete→下一 review 推进 / 节点内 dev→review 轮转 / QA 机械收口）——T17 e2e 以 scratch SQL 接缝模拟；正式接缝属 wbflow 插件期
- **C** RD∥FE 并行组 join 语义（现按渲染序机械收口）——接受或另定
- **D** scope 切换无 API 端点——补端点 vs 留插件期
- **E** semantica 决策记录 12 条已齐（`3ce761b4`/`21cb8f86`/`50aaefad`/`e1acca31`/`b180101e`/`1fc79a1d`/`f7463467`/`eb31734e`+`3794085c`/`62c492a0`/`47cb6bbb`/`c784503c`/`19f1d47e`）——确认归档即可

**复审新增 4 项**：
- **F** `align.opening` 派发语义：设计口径（program 类 node action 不可经 wb_assign_task 派发，落注记）vs 修路由——**建议前者**（与 wbflow 程序化直启架构一致：node action 由插件执行不经任务总线）
- **G** submit 动作：seed 词表有但 actionTaskTypes 无映射无激活——补映射+激活 vs 落注记（若 produce/complete 判据已覆盖登记场景则一致）——**建议落注记**（登记动作走 exit hook 的 wb_add_artifact+wb_comment，不经任务总线）
- **H** spec §2 GET 缓存快照：已改写为实际口径（降级仅 healthz + 重启恢复，v1.1 预留）——确认即可
- **I** tasks.reply_count 死列：DROP 列 vs 留 TODO 注明插件期接线——**建议留 TODO**（迁移 v4 成本 vs 收益）

## 下一步动作序列（新会话照此执行）

1. **收 User 裁决**（A-D + F/G/I 逐项；E/H 确认）→ 拍板结论 semantica record_decision（skill：semantica-decision）
2. 裁决产生的代码改动（若有）→ 小修直接主会话做或派实现者；无改动则跳过
3. **合并**：`cd .worktrees/wbflow-impl && git checkout wbflow-impl && git merge --no-ff wbflow-impl-crfix-r1`（合并信息注明 CR 第 1 轮清零）；合并后复跑 `go build ./... && go test ./...` 快速闸门
4. **FE 浏览器联调冒烟**（browser-use 主线程做，勿派 subagent）：fe dev + rd 起本地栈（rd 冒烟用 18080，本机 8080 被占）；流程树/看板/阅读器三页冒烟；T14/T15/T16 勾选注记「浏览器联调待编排者验收」即此
5. **清理**：`git worktree remove .worktrees/t1 .. t17`（17 个工单 worktree，逐个 remove + `git branch -d wbflow-impl-t*`——分支已全并入安全删）
6. **AC 终核**：spec §4.1 P1-P5 勾选逐条对照（26 工具可用/seed 初始化/full 端到端/rd-only 9 节点/task 全链）
7. **推送**（若 User 要）：分支未推 origin（远端 GitLab，本机无 glab/gh）——推 `wbflow-impl` + `wbflow-impl-crfix-r1`（或合并后单分支），建 MR 由 User 操作

## 环境事实（不变，新会话必读）

- MySQL 8.0.46 @ 192.168.3.100:3306，root 凭据用户会话中提供（env 注入 `WBFLOW_MYSQL_*`，密码零落仓库）；**禁写 wbflow 库**（scratch `wbflow_*_r<N>` 测后 DROP）
- `export GOPROXY=https://goproxy.cn`；go test 建议加 `-timeout 40m`（service 包全量并发 >20m 撞默认超时）或 `-run` 分片
- 本机 8080 被占（冒烟 18080）；GOCACHE 沙箱不可写时重定向 worktree/.gocache（已 gitignore）
- subagent 约定：prompt 开头 `Load the "caveman" skill and follow it.`
- spec 文档漂移修正授权：文档对齐事实类可直接改（前会话先例）；语义裁决类必须问 User

## Suggested skills

- `semantica-decision`——裁决拍板后记录
- `implement-spec`——若裁决产生代码改动
- `browser-use:control-browser`——FE 冒烟（主线程自己做）
- `caveman`——subagent 注入
