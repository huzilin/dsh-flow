# FE 开发说明（req-019：M7 编排演练 — 并行组 join 与部分回滚）

> 节点：n-10 FE 开发（ownerRole=FE，branch=fe，parallelGroup=dev-1）
> 分支：**feature/req-019-m7-drill**（与 RD 同一分支；工作台按 `branch=fe` 记录）
> 状态：FE 自测完成，待 Code Reviewer 评审（CR）

## 1. 需求与目标

- 需求：`req-019`「M7 编排演练：并行组 join 与部分回滚」，P3，范围=前后端都含。
- 目标：验证并行开发组 join 判定与 QA 部分回滚。
- 验收点：
  1. 两分支（RD/FE）全过才进 QA 验收（并行组 join 判定）；
  2. QA 打回仅回滚指定分支（部分回滚，另一分支保持已通过）。

## 2. 开发背景

- 本节点 n-10 与 RD 开发节点 n-09 构成并行组 `dev-1`，两分支并行开发，组内全通过（join）后才放行 n-11 QA 验收。
- 演练已发生一次 **QA 部分打回 fe 分支**（打回理由：前端交互缺陷，仅回滚 FE 分支），RD 分支不受影响；本次为 FE 分支回滚后重开发（回滚记录 `rolledBackAt` 递增，节点状态回到「进行中」）。

## 3. 实现要点

1. **前端交互缺陷修复（QA 打回项）**
   - 修复并行组泳道视图下「部分回滚」的交互呈现：分支被打回后节点状态恢复为「进行中」，流程树需即时同步展示，避免停留在旧的「已通过」态。
   - 回滚标记与理由展示：节点渲染 `rolledBackAt` 回滚次数（↩n）与 `lastDecision`（verdict=reject、reason、targetBranch），让打回方与原因在流程树上可见。
2. **并行组 join 前端呈现**
   - RD/FE 双泳道并排渲染（`parallelGroup`/`branch`），跨分支不画连线；组内任一分支未通过时，后续 QA 验收节点保持「未开始」，组内全部「已通过」才汇聚放行。
3. **角色身份落库（协作铁律）**
   - 以 FE 身份产出《FE 开发说明》并 `wb_add_artifact` 登记到 n-10；`wb_comment`（author=FE，reviewerRole=Code Reviewer）提交评审，评论 state=open 阻塞门禁直至 resolve/override。
4. **与 RD 的契约**
   - 与 RD 共用同一 feature 分支 `feature/req-019-m7-drill`；本需求为编排演练，无新增业务接口，FE 侧仅消费流程树状态机数据（`branch`/`parallelGroup`/`rolledBackAt`/`lastDecision`）做交互呈现，接口契约不变。

## 4. 自测结果

| 用例 | 步骤 | 预期 | 结果 |
|---|---|---|---|
| TC-FE-01 并行组 join | 组内 RD/FE 任一分支未通过 | QA 验收（n-11）保持未开始，不放行 | ✅ 通过 |
| TC-FE-02 部分回滚呈现 | QA 打回 fe 分支（targetBranch=fe） | 仅 n-10 状态回「进行中」、`rolledBackAt`+1 且显示 ↩1、lastDecision 含打回理由；n-09 保持已通过 | ✅ 通过 |
| TC-FE-03 评论落库 | FE 在 n-10 发评论（reviewerRole=Code Reviewer） | 评论 state=open、作者=FE、入收件箱提醒 Code Reviewer | ✅ 通过 |
| TC-FE-04 产物登记 | 登记《FE 开发说明》到 n-10 | artifacts.json 含 art-fe（kind=document，href 指向本文档） | ✅ 通过 |
| TC-FE-05 评审流 | Code Reviewer 表态 pass 后组内全通过 | join 判定放行 n-11 QA 验收 | ✅ 通过（状态机逻辑） |

## 5. 回归范围

- 流程树视图：并行泳道布局、门禁状态行、评审区、回滚标记与理由展示。
- 需求详情页：产物列表、评论/收件箱联动。
- 不涉及后端服务改动（并行组 join / 部分回滚状态机在 Host 侧 `wb_decide` 实现，FE 仅做交互呈现与验证）。

## 6. 结论

FE 分支自测全部通过，无遗留问题；提交 Code Reviewer 评审（author=FE，reviewerRole=Code Reviewer）。
