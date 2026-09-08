# Handoff: 原型补齐 P8-P11/N1-N7 实现 + 全链路联调收官（2026-09-09）

> 本轮接续 `2026-09-08-wbflow-impl-cr-loop.md`（拍板 A-E 落地）之后：ui-interaction-final §6 N1-N7 定稿 + ui-prototype-v3 P8-P11（main 工作区未提交修订即本轮「新 spec」）全量实现、双 CR 循环清零、浏览器联调跑通。

## 终态

- **分支**：`wbflow-impl-crfix-r1` @ `9a85f57`（worktree `.worktrees/wbflow-impl`），基点 `d1c3731`，**待推 origin**（同上轮，远端 GitLab 本机无 glab/gh）
- **提交链**：`914a7dd` 后端配套（spec 注记-D1..D6）→ `2b89d15` 详情页域 → `5b74416` 收件箱五分类（子分支合并）→ `bca6d59` 看板/版本页（子分支合并 `889d828`）→ `d97f7ff` 依赖树封装收编 → `a072188` 看板编辑接线 → `4573cd6` 疑似卡死高亮 → `9a85f57` FE CR-R8 修复
- **CR 循环**：后端 R6（9 发现：3 应修=spec 计数 27→28 全局同步/retest 幂等守卫/TargetID 注释）→ R7 复审清零 + Low 项收口（`f7f89c2`+`1caa340`）；前端 R8（硬违规 1=CSS merge 残缺括号 / 应修 2 / judgement 8，另直测暴露依赖树悬挂根渲染缺陷已修）→ R9 复审清零（`9a85f57`）
- **闸门**：rd gofmt/build/vet/test 全绿 + ProtoD 真库 IT PASS（service 包含新 2 用例）；fe 17 文件 117 测试 / eslint / build 全绿
- **联调**（真实库 wbflow + rd:18080 + fe:5173，browser-use 主线程实测）：编排驱动（对齐拍板→产品设计→评审三位 agree→agreed 停等）→ UI 拍板卡迎宾态+通过迁移；收件箱 badge/提醒条/gate_pending/「待复验」五分类（retest targetId 锚点跳转）；版本页四色聚合/依赖树空态/调度卡就绪池/编辑挂版本；废弃 blocked（注记-A「已开始」真库生效）。证据注记已落 `.plan/wbflow-impl/tickets.md` 头部。

## 本轮关键裁决（semantica `c8984043`）

- retest 子型不加通知 kind（守七类枚举），复用 re_review + target_id 空/非空区分子型
- sessions 进快照（同屏数据一次拉全）；history 独立端点（审计无界不进快照）；端点 27→28
- P9「删除版本」不实现（路由表为功能边界权威，原型=丢弃式视觉参照）
- 编辑弹窗 category/scope 只读（PUT 无重投影 + scope 主动切换暂不支持拍板）

## 待 User / 下一步

1. **推 origin / 建 MR**（同上轮遗留）
2. main 工作区三个未提交文件（ui-prototype-v3.html / ui-interaction-final.md / fe/README.md）为本轮 spec 来源，确认后随本轮交付一并提交 main
3. 观察项：Dispatch 吞错下通知偶发静默丢失（spec §1.3 既有拍板），未来补重试/审计
4. 暂缓项不变：P2/P3 评审轮完整视图、P6 会话聊天（DSH 嵌入期）、N5「最近消息摘要」字段（随会话端点补齐）

## 环境备忘（新会话必读）

- 同上轮：MySQL 8.0.46 @192.168.3.100（凭据 `rd/configs/config.local.yaml`，gitignore）；`GOPROXY=https://goproxy.cn`；本机 8080 被占（rd 冒烟 18080）；fe dev `WBFLOW_API_PORT=18080 pnpm dev` + `.env.local`（VITE_WBFLOW_AUTH_TOKEN 同 rd）
- 本轮清理：`.worktrees/fe-inbox`、`.worktrees/fe-board`（分支 `wbflow-impl-fe-inbox`/`wbflow-impl-fe-board` 已合并保留）；IT 复跑仍需 `-timeout 900s`
