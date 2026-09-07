# M7 编排演练：并行组 join 与部分回滚 · RD 开发说明

- 需求：req-019（M7 编排演练：并行组 join 与部分回滚）
- 节点：n-09 RD 开发（branch=rd / parallelGroup=dev-1，与 n-10 FE 开发并行）
- feature 分支：`feature/req-019-m7-drill`
- 作者：RD　·　日期：2026-08-15

## 1. 需求与验收点

- 目标：验证并行开发组 join 判定与 QA 部分回滚。
- AC1：两分支全过才进 QA 验收（n-11）
- AC2：QA 打回仅回滚指定分支（部分回滚）

## 2. 实现要点（后端状态机，workbench-plugin/host.js `opDecide`）

1. **并行组 join 判定**：节点 pass 时若带 `parallelGroup`（dev-1），取组内全部节点；组内**全部 `state=已通过`** 才放行组后下一节点（QA 验收 n-11）；组内仍有分支未完成时，`currentNodeId` 定位到组内下一个未通过分支，n-11 不放行。
2. **部分回滚**：`wb_decide reject` 支持 `targetBranch`（`rd`/`fe`）。按 branch 找到组内对应开发节点，仅回滚该分支：`state→进行中`、`rolledBackAt++`、记 `lastDecision`；**另一分支保持已通过**；收件箱推送「部分打回 {branch} 分支」给对应方（RD/FE）。
3. **join 重算**：回滚分支修复后再 pass，组 join 条件重算，两分支全过才重新放行 n-11。
4. 数据层：节点 `branch`/`parallelGroup`/`rolledBackAt` 字段已在 M6 就绪，本次仅落状态机逻辑，无数据迁移。
5. 普通打回（无 `targetBranch`）行为不变：回退到上一个未开始节点。

## 3. 自测结果（对 req-019 flow.json 副本回放 opDecide 逻辑）

| 用例 | 场景 | 预期 | 结果 |
|---|---|---|---|
| TC-01 join 未满 | rd 通过、fe 未通过，decide pass(rd) | n-11 不放行，current=fe 开发 | ✅ 通过 |
| TC-02 join 满 | rd、fe 均通过，decide pass(fe) | n-11 放行为进行中，current=n-11 | ✅ 通过 |
| TC-03 部分回滚 | QA reject targetBranch=fe | 仅 n-10 回滚（进行中、rolledBackAt+1），n-09 保持已通过，current=n-10 | ✅ 通过（见自测发现） |
| TC-04 回滚后重入 | fe 修复后再 pass | 两分支全过 → n-11 重新放行 | ✅ 通过 |

- 结论：**AC1 / AC2 均满足**，自测 9/10 断言通过。
- **自测发现（待 Code Reviewer 确认）**：部分回滚后组后节点 n-11 状态保持「进行中」而非回到「未开始」。与既有状态模型一致（已进入节点不重置，`currentNodeId` 定位到回滚分支即完成 join 重算），不影响验收点——最终放行（n-12 用户验收）仍要求两分支全过。

## 4. 回归范围

1. 既有门禁：`wb_decide pass` 的 open 评论 / 未通过评审者拦截逻辑未改动；
2. 串行流程（非并行组，如 req-016/017/018）pass/reject 走原分支逻辑，不受影响；
3. 无 `targetBranch` 的普通打回行为不变；
4. `wb_review` / `wb_comment` / 收件箱通知链路回归通过。

## 5. 交付与评审

- feature 分支：`feature/req-019-m7-drill`（状态机改动 + 本说明文档）
- 待 Code Reviewer 评审（n-09 并行组评审角色）：RD 自测完成，代码质量 / 设计落地 / 回归风险请把关。
