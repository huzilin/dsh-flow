# M6：缺口补齐方案（review 结论 → 落地）

> 状态：**v0.1（方案稿，待 User 确认后实施）**
> 来源：流程 review（下游角色视角：每阶段信息是否够、接口是否够）→ 7 个缺口；本文给出每个缺口的补齐方案、归属里程碑（M6 工具层 / M7 编排层 / 文档约定层）与实施步骤。

## 0. 缺口清单与归属总览

| # | 缺口 | 方案 | 归属 | 涉及 |
|---|---|---|---|---|
| 1 | 产物登记（doc/link 无工具挂 artifacts.json） | 新增 `wb_add_artifact` | **M6** | host.js / pkg |
| 2 | 评审者正式表态（reviewers[].state 无工具维护） | 新增 `wb_review`；wb_decide 门禁联动 reviewers | **M6** | host.js / pkg |
| 3 | 评论无身份/角色（wb_comment 固定 author=User） | wb_comment 支持 author/role 参数 | **M6** | host.js / pkg |
| 4 | CR 读代码（link 类无法读内容） | Code Reviewer 子 agent 用 fs/代码工具评审真实代码 | **M7** | 子 agent 编排 |
| 5 | 工期/依赖信息无约定 | RD 技术设计文档模板（含接口/数据/工期与依赖/自测）写入 skill | **文档约定** | m6-flow-templates / skill |
| 6 | 部分回滚（QA 打回仅回滚对应分支） | 并行组 join 判定 + reject 支持 targetBranch | **M7** | 状态机升级 |
| 7 | 通知提醒（门禁待处理/评审@） | sqlite inbox（落库 v1，UI 后置） | **M6** | Host inbox.db |

> 已同步落地（本文之前）：feature 分支敲定（RD 评审/方案评审节点）+ worktree 并行（协作铁律 7/8），flow.json 新增 `featureBranch` 字段。

## 1. M6 工具层补齐（本次实施）

### 1.1 产物登记 `wb_add_artifact`

- **签名**：`wb_add_artifact(reqId, nodeId, label, href, kind, note?)`
  - `kind`: `document` | `link`
  - 服务端 `opAddArtifact`：读 `artifacts.json` → push `{id: 'art-<ts>', nodeId, label, href, kind, note, addedBy, addedAt}` → 写回 → 返回最新 artifact 列表
- **用途**：产品设计/RD 设计/FE 设计/用例设计/测试用例/缺陷/验收/需求清单等产出后登记；MR 链接由开发登记
- **说明**：文档正文由作者用文件工具写入项目目录，登记只写元数据（href 指向文件）；与「文档先行」铁律一致

### 1.2 评审者表态 `wb_review`

- **签名**：`wb_review(reqId, nodeId, role, verdict, commentIds?)`
  - `verdict`: `pass` | `block`（block 需给出理由——建议强制附 commentId 或 reason）
  - 服务端 `opReview`：在 `flow.nodes[].reviewers[]` 中找到 `role` 匹配项 → 置 `state='已通过'|'未通过'`，记录 `reviewedAt/by`；block 时可将传入的 open 评论留在该 reviewer 名下
- **门禁联动（wb_decide pass 升级）**：pass 前置条件 = ① 无 open 评论 ② **所有 reviewers[].state === '已通过'**；任一 reviewer 未通过 → blocked 提示（与评论拦截并列）
- **UI**（可选 M6）：评审状态胶囊旁加「通过/打回」操作，复用 wb_review

### 1.3 评论身份 `wb_comment` 扩展

- **签名**：`wb_comment(..., author?, role?, reviewerRole?)`（默认 author=User / role=user）
- 评论记录写入 `author/role/reviewerRole`；既有 `wb_override`（按 reviewerRole 匹配）与 `wb_review`（统计角色评论）依赖该字段
- UI 评论者身份默认 User（后续按登录态/角色切换）

### 1.4 通知落库 `wb_inbox`（sqlite）

- 表：`inbox(id, reqId, type, from, to, at, read)`；`type`: `gate`（门禁待处理）/ `review`（评审待表态）/ `mention`（评论@角色）
- 触发点：`wb_review`（生成评审待办）、`wb_comment`（@角色）、`wb_decide`（推进通知下一节点 owner）
- 存储路径验证：Host `bash`/`subprocess` 执行 `sqlite3` CLI（`<workspace>/.workbench/inbox.db`）；若无 sqlite3 则回退 DSH 存储服务或 JSON 文件（v1 可退化为 inbox.json，schema 不变）
- UI 收件箱面板**后置**（M6 只落库 + 工具可查）

## 2. M7 编排层补齐（子 agent 真实协作）

### 2.1 并行组 join + 部分回滚（状态机升级）

- **join 判定**：业务需求 `dev-1` 组内 RD 开发、FE 开发 **两分支均 state=已通过** 后才放行下一节点（QA 验收）；`wb_decide` 对并行组节点 pass 时检查组内其余分支
- **部分回滚**：`wb_decide reject` 增加 `targetBranch`（`rd`/`fe`）；QA 打回时仅将该分支的开发节点回滚（state→进行中、rolledBackAt++），另一分支保持已通过；组 join 状态重算
- 数据已支持（节点 `branch`/`parallelGroup`/`rolledBackAt`），逻辑 M7 实现

### 2.2 CR 读代码

- Code Reviewer 由真实子 agent 承担：用 fs/代码/搜索工具读取 MR 对应分支代码（featureBranch 已约定），对照设计文档与用例评审；评论经 wb_comment（author=Code Reviewer）落库
- 前提：M7 编排按角色拉起子 agent，各子 agent 具备对应工具集

### 2.3 真实子 agent 编排（req-019 演练验证中）

**编排链（角色子 agent 通过 wb_* 工具协作，不直接改领域数据）**：
1. **RD / FE 子 agent（并行）**：产出开发说明文档 → `wb_add_artifact` 登记到对应开发节点 → `wb_comment`（author=RD/FE, reviewerRole=Code Reviewer）请评审
2. **Code Reviewer 子 agent**：读产物文档（`wb_get_artifact_content`）→ `wb_comment`（author=Code Reviewer）留评审意见 → `wb_review`（role=Code Reviewer, verdict=pass）
3. **协调者（主 agent）**：`wb_decide pass` 推进两分支（并行组 join 自动判定）
4. **QA 子 agent**：产出《测试用例与结果》→ `wb_add_artifact` → `wb_review`（role=QA, pass）
5. **协调者**：`wb_decide pass` QA 验收 → 用户验收 → 完成

**产物/评论身份铁律**：子 agent 以角色身份通过 wb_add_artifact / wb_comment(author) / wb_review 落数据；主 agent 只做流程推进（wb_decide）与协调，不替角色表态。

## 3. 文档约定层补齐

- **RD 技术设计文档模板**（写入 m6-flow-templates.md 附录 + skill `team-conventions`）：
  1. 方案概述与目标
  2. 接口/数据设计（FE 评审接口契约依据）
  3. **工期与依赖**（PM 评审排期依据；涉及外部系统/第三方依赖明列）
  4. 自测方案（QA 联调口径）
  5. 风险与回滚
- 同类模板：产品设计文档（需求评审输入）、FE 技术设计（QA 可测性输入）、用例设计（RD/FE 自测口径）、方案文档（任务 QA 可测性）

## 4. 实施步骤（顺序）

| 步 | 内容 | 验证 |
|---|---|---|
| S1 | host.js：`opAddArtifact` + `opReview` + `wb_comment` 扩展 author/role + `wb_decide` 门禁联动 reviewers；注册 `wb_add_artifact`/`wb_review`（16→18 工具） | node --check + wb_* 工具调用验证 |
| S2 | 提交 pkg（host 变更，client 不动）→ 批准后刷新 | 工具列表含新工具；评审胶囊显示 |
| S3 | 文档：RD 技术设计等产物模板写入 m6-flow-templates.md 附录 + skill 草案更新 | 文档 review |
| S4 | sqlite inbox：验证 sqlite3 CLI 路径 → 落库 + 触发点接入 | inbox.db 生成、查得消息 |
| S5 | （M7 前置）数据层补：req-017 示例补 featureBranch、reviewers 状态 | flow.json review |
| M7 | 并行组 join/部分回滚状态机 + CR 子 agent + 全角色子 agent 编排 | 真实演练一条需求走完 |

## 5. 范围外（明确不做）

- 评论富文本/附件：不做
- UI 产物在线编辑：不做（文档作者在项目内编辑，工作台只读展示）
- 权限体系（谁可 pass 谁）：M7 按角色子 agent 约束，不做登录/ACL
