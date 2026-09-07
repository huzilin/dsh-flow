# M7：角色协议（Agent 编排落地文档）

> 状态：**v0.1（Leader 已定 = 主 agent；角色子 agent 逐个落地中）**
> 目的：为团队工作台的每个角色定义**扮演者 / 职责 / 输入 / 产物 / 输出格式 / 协作规则**，使主 agent（Leader）与角色子 agent 的编排可复用、可验证。
> 关联：m6-team-preset.md（制度：流程/门禁/评审分工）、m6-flow-templates.md（节点/产物模板）、m6-gap-closure.md（缺口补齐与编排链）。

## 1. 角色归属总表

| 角色 | 扮演者 | 职责要点 | 落地状态 |
|---|---|---|---|
| User | 用户（Human） | 需求提出、最终拍板、制度修改入口 | ✅ 固定 |
| **Leader** | **主 agent** | 拆解分派、协调推进、wb_* 落账、处理返工 | ✅ 已定（本文 §2） |
| Architect | 子 agent | 产品设计、技术方案把关、版本规划 | ⏳ 待落地 |
| RD | 子 agent | 后端设计/开发/自测 | ✅ M7 已验证（§4.1） |
| PM | 子 agent | 需求评审、排期与依赖把关 | ⏳ 待落地 |
| FE | 子 agent | 前端设计/开发/自测 | ✅ M7 已验证（§4.2） |
| QA | 子 agent | 用例设计、验收测试 | ✅ M7 已验证（§4.5） |
| Investigator | 子 agent | 竞品调研（版本规划 ②） | ⏳ 待落地 |
| Code Reviewer | 子 agent | 开发产物评审（代码/说明） | ✅ M7 已验证（§4.3） |

## 2. Leader（主 agent）行为协议

Leader 由主 agent 扮演，是编排的唯一协调者，负责：

1. **全局视图**：用 `wb_list_requirements` / `wb_list_versions` / `wb_get_requirement` 掌握需求池与流程状态
2. **拆解分派**：按流程节点把工作分派给对应角色子 agent（prompt 自包含：需求 id、节点、产物路径、期望输出格式）
3. **子 agent 并行**：无依赖的角色（如 RD/FE 开发）并行起后台子 agent；有依赖的串行等结果
4. **正式落账**：子 agent 的产物/评论/评审通过 `wb_*` 工具落账（add_artifact / comment / resolve / review / decide）——子 agent 无 wb_* 工具时直接写 `.workbench/`（按 host.js 格式），Leader 负责校验与用工具正式落账
5. **门禁推进**：resolve 全部 open 评论 → 相关角色 `wb_review pass` → `wb_decide pass` 推进（并行组 join 自动判定）
6. **返工循环**：评审 block 时（wb_review 未通过 或 open 评论未解决）→ 用 `send_message` 续话对应子 agent 返工 → 重评审；QA 部分回滚用 `wb_decide reject + targetBranch`
7. **收尾**：最后节点通过后 `wb_update_requirement status=已完成`

**铁律**：Leader 不替角色表态（review 由对应角色子 agent 或以 by=角色 落账）；只做协调与流程推进。

## 3. 角色子 agent 通用约定

- **prompt 自包含**：子 agent 看不到 Leader 对话，prompt 必须含：需求 id、节点 id/key、角色职责、输入文档路径、产物路径与格式、期望输出格式、协作铁律（各审各的、不越权改 flow.json）
- **无 wb_* 工具**（宿主注册的工具子会话不可见）：按 host.js 的 opAddArtifact/opComment 格式直接写 `.workbench/` JSON；写入后报告 artifactId/commentId
- **不越权**：不改 flow.json 的节点状态/currentNodeId（流程推进归 Leader）；只产文档、登记产物、发评论
- **feature 分支**：开发类角色按 featureBranch 约定（评审节点敲定分支名）

## 4. 角色卡（逐个落地）

### 4.1 RD（后端研发）✅
- 职责：后端设计（业务需求③/技术需求②）、开发（⑨.1）、自测
- 输入：需求描述、产品设计/RD 设计文档
- 产物：《RD 技术设计》/《RD 开发说明》（`docs/<版本号>/<需求名>/rd/design.md` / `rd/dev.md`）
- 输出格式：报告 artifactId / commentId / 自测结论 / 阻塞问题
- 协作：登记产物 by=RD；评论 author=RD reviewerRole=Code Reviewer

### 4.2 FE（前端研发）✅
- 职责：前端设计（⑤）、开发（⑨.2）、自测
- 输入：产品设计 + RD 设计（接口契约）
- 产物：《FE 技术设计》/《FE 开发说明》
- 输出格式：同 RD

### 4.3 Code Reviewer（代码评审）✅
- 职责：评审开发产物（质量/设计落地/边界与回归风险，各审各的）
- 输入：开发说明文档（+ 真实代码，若可用）
- 产物：评审结论评论（author=Code Reviewer，state=open；通过/阻塞）
- 输出格式：pass/block + 结论摘要 + 非阻塞建议

### 4.4 Architect（架构师）⏳
- 职责：产品设计（①）、RD 技术评审把关、版本规划（立项/架构规划）
- 输入：需求描述、架构文档
- 产物：《产品设计文档》/《架构规划》/《需求对齐文档》
- 输出格式：待定（落地时按 §4.1 风格定义）

### 4.5 QA（测试验收）✅
- 职责：用例设计（⑦）、验收（⑩ 按用例 + 缺陷闭环）
- 输入：需求描述（验收点）、设计文档、开发说明
- 产物：《用例/联调设计》《测试用例与结果》《缺陷文档》《验收记录》
- 输出格式：AC 逐条核对结论 + 用例结果表 + 通过/遗留

### 4.6 PM（项目管理）⏳
- 职责：需求评审（② 价值/优先级/范围）、排期与依赖把关（RD 技术评审）
- 输入：需求描述、产品设计、RD 技术设计（工期/依赖小节）
- 产物：评审结论评论
- 输出格式：待定

### 4.7 Investigator（调研员）⏳
- 职责：版本规划 ② 竞品调研（GitHub 等）：引入 or 重写、为什么
- 输入：《需求对齐文档》
- 产物：《调研文档》
- 输出格式：待定

## 5. 编排序列（按流程类型）

### 业务需求（11 节点 + 待启动）
Leader 依次：待启动→(Architect 产品设计)→(PM/Architect/User 需求评审)→(RD 设计)→(Architect/RD/FE/QA/PM 评审)→(FE 设计)→(QA 评审)→(QA 用例)→(RD/FE 用例评审)→**并行 (RD 开发 | FE 开发) → CR 分别评审**→(QA 验收)→(User 验收)
其中括号内角色由对应子 agent 承担，Leader 负责落账与推进。

### 技术需求 / 任务 / 版本规划
同 §4 角色卡映射，Leader 按 m6-flow-templates 的节点顺序推进。

## 6. 验证记录

| 日期 | 场景 | 参与角色 | 结果 |
|---|---|---|---|
| 2026-08-15 | req-019 业务需求开发→评审→验收全流程 | RD/FE/Code Reviewer/QA（子 agent）+ Leader | ✅ 全通过（join/部分回滚验证） |

## 7. 待落地

- [ ] Architect 角色卡 + 验证（产品设计 / 版本规划）
- [ ] PM 角色卡 + 验证（需求评审 / 排期把关）
- [ ] Investigator 角色卡 + 验证（调研）
- [ ] 沉淀为 team preset 的 skill（wb-agent-guide 编排章节）
- [ ] wb_* 工具随 preset 挂载后，子 agent 直接使用工具（替代文件直写）
