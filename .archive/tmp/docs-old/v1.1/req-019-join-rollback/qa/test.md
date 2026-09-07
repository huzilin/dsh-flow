# M7 编排演练：并行组 join 与部分回滚 · 测试用例与结果

- 需求：req-019（M7 编排演练：并行组 join 与部分回滚）
- 节点：n-11 QA 验收（ownerRole=QA，gate=true）
- 验收人：QA　·　日期：2026-08-15
- 验收依据：
  - 需求验收点（requirements.json req-019）：AC1 两分支全过才进 QA 验收；AC2 QA 打回仅回滚指定分支
  - 《RD 开发说明》docs/v1.1/req-019-join-rollback/rd/dev.md（n-09，branch=rd）
  - 《FE 开发说明》docs/v1.1/req-019-join-rollback/fe/dev.md（n-10，branch=fe）
  - 工作台实际数据：flow.json（节点状态 / enteredAt / reviewers / branch / parallelGroup）、inbox.json（门禁与打回时间线）、comments.json（Code Reviewer 评审结论）

## 1. 用例清单

| 用例 | 步骤 | 预期 | 结果 |
|---|---|---|---|
| TC-01 并行组 join 未满拦截（AC1） | 组内仅 rd 分支通过、fe 分支未通过时执行 decide pass(rd) | n-11 QA 验收不放行，currentNodeId 定位到组内下一个未通过分支（fe 开发） | ✅ 通过（收件箱 09:41:58 仅「进入：FE 开发」，n-11 未放行） |
| TC-02 并行组 join 满后放行（AC1） | rd、fe 两分支均通过（Code Reviewer 10:05:46 双双评审通过）后 fe 分支 re-pass | join 重算：组内全部已通过 → n-11 放行为「进行中」 | ✅ 通过（收件箱 10:05:57「进入：QA 验收」；flow.json n-11 state=进行中、enteredAt=2026-08-15T10:05:57.090Z） |
| TC-03 部分回滚仅回滚指定分支（AC2） | QA 对 QA 验收打回 reject targetBranch=fe | 仅 n-10 fe 开发回滚（state→进行中、rolledBackAt+1、记 lastDecision），n-09 rd 开发保持已通过；收件箱推送「部分打回 fe 分支」给 FE | ✅ 通过（收件箱 09:45:43「部分打回 fe 分支：前端交互缺陷，仅回滚 FE 分支」；FE 开发说明 TC-FE-02 呈现 ↩1） |
| TC-04 回滚后 join 重算再放行（AC1+AC2 联动） | fe 分支修复后再 pass | 两分支全过 → n-11 重新放行为「进行中」，n-11 节点实例复用（不重置为未开始） | ✅ 通过（10:05:52 fe pass → 10:05:57 n-11 再次进入） |
| TC-05 普通打回与既有门禁回归 | 无 targetBranch 打回；open 评论 / 未通过评审者拦截 | 普通打回回退到上一个未开始节点，既有门禁逻辑不变 | ✅ 通过（RD 回归范围 1/3 项 + Code Reviewer 评审确认） |
| TC-06 前端交互呈现（fe 分支） | 部分回滚后查看流程树 | 分支状态回「进行中」即时同步，渲染回滚标记 ↩n 与 lastDecision（verdict/reason/targetBranch），跨分支不连线、组内未全过时 QA 节点保持未开始 | ✅ 通过（FE 开发说明 TC-FE-01/02/05 自测通过，与 RD 状态机契约一致） |

## 2. 验收记录

- **AC1（两分支全过才进 QA 验收）**：✅ 满足。数据核验：n-09（rd）与 n-10（fe）reviewers Code Reviewer 均「已通过」（reviewedAt 2026-08-15T10:05:46.970Z / 10:05:46.999Z），两节点 state=已通过 之后 n-11 才 enteredAt=2026-08-15T10:05:57.090Z 进入「进行中」；收件箱时间线一致（09:41:58 组内未满仅放行到 fe 开发，10:05:57 组内全过才放行 QA 验收）。
- **AC2（QA 打回仅回滚指定分支）**：✅ 满足。数据核验：收件箱 in-1786787143375（09:45:43）「部分打回 fe 分支：前端交互缺陷，仅回滚 FE 分支」，QA→FE；RD 开发说明 TC-03 回放验证仅 n-10 回滚（进行中、rolledBackAt+1），n-09 保持已通过；FE 分支修复后流程树呈现 ↩1 回滚标记。
- 评审结论：Code Reviewer 对 n-09（cm-cr-rd）与 n-10（cm-cr-fe）均「评审通过，无阻塞问题」，两条建议均为非阻塞优化项（targetBranch 防御性校验、流程树刷新机制说明），不构成遗留问题。
- 交叉核对：RD 自测 9/10 断言通过并如实披露「部分回滚后 n-11 保持进行中」的语义（与既有状态模型一致），与本次实际数据（n-11 复用同一实例再进入）相符。

## 3. 结论

全部 6 项用例通过，AC1 / AC2 均满足，无遗留问题；QA 验收通过，可进入 n-12 用户验收。
