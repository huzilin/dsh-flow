# 制度层：流程模板 · 角色 · 能力 · 评审分工

> 本文档 = **制度层（DSH team preset）**：4 类流程模板 + 9+1 角色 + 19 能力 + 评审分工 + 协作铁律——Agent 行为的组织真源。
> 职责边界：流程 / 角色 / 能力 / 评审 / 铁律；**工具清单与编排协议在 `07-tools-and-orchestration.md`**，实体与状态机以 `05-domain-model.md` 为准（本文只引用不重写）。
> 取代旧稿：`m6-team-preset.md`、`m6-flow-templates.md`、`m8-flow-simplify.md` §2.4/§10。

---

## 1. 定位

制度层以 **DSH team preset** 落地：

- 从 **standard** preset 复制，在其上**叠加团队制度**（角色、流程、门禁、协作铁律、skill 指南）；团队内任何 agent 会话（主会话 + 子 agent）加载同一份制度。
- 制度 **只读**：执行角色不得自行改制度；调整仅经「**User 明文告知 → 协助调整 preset 文件**」通道。
- preset 决定：该会话**挂哪些工具**、**系统提示中的制度内容**、**可加载哪些 skill**。
- skill 挂载（随 preset 交付）：`wb-agent-guide`（工作台工具组合流程：拆分版本 / 推进任务 / 评审闭环 / 破坏性操作确认）、`team-conventions`（角色、流程、门禁、评审分工速查）。

## 2. 团队角色（9+1）

| 角色 | 职责（一句话） |
|---|---|
| User | 需求提出方 / 最终决策者，制度唯一修改入口（确认范围、拍板发布） |
| Leader | 团队负责人：拆解任务、分派 owner、门禁决策（wb_decide）推进 |
| Investigator | 调研 / 澄清需求，输出调研结论（竞品/开源：引入 or 重写，为什么） |
| Architect | 技术方案设计与评审：产品设计、技术路线、架构规划、版本规划 |
| RD | 后端研发：技术设计、开发与自测 |
| PM | 项目管理 / 范围管控：需求价值、优先级、排期与依赖 |
| FE | 前端研发：技术设计、开发与自测 |
| QA | 测试与验收：用例设计、验收、缺陷闭环（含跨端根因判定） |
| Code Reviewer | 代码评审：代码质量、设计落地、回归风险 |
| executor（动态占位） | 任务流程（task）的**动态执行者**：方案评审/开发节点 owner，由 User/Leader 指定，实例化时按实际执行者解析（模板默认绑 backend-dev） |

**owner 规则**：需求任务 `scope=仅前端` → **FE**；其余（仅后端、前后端都含）→ **RD**（RD 主导，FE 按需并行）；任务流程执行者由 User/Leader 指定。

## 3. 流程模板（4 类）

按需求 `category`/`reqType` 选择模板（business / tech / task / plan）；每个节点标注：**阶段 / 负责人 / 做什么 / 产物（doc|link）/ 门禁**。

> **通用首阶段「待启动」**：所有流程第一个节点均为「待启动」——需求已创建但尚未投入资源；当团队**开始投入资源**时推进到下一阶段。owner=Leader，**无门禁**（启动动作本身）。以下各表以「待启动」开头，不再重复列出。

### 3.1 业务需求（business）：12 节点 + 待启动

> 节点顺序：①产品设计 → ②需求评审 → ③RD 技术设计 → ④RD 技术评审 → ⑤FE 技术设计 → ⑥FE 设计评审 → ⑦用例设计 → ⑧用例设计评审 → ⑨.1 RD 开发 ∥ ⑨.2 FE 开发（并行）→ ⑩QA 验收 → ⑪用户验收。**无产品设计评审独立节点**（产品方案把关并入②需求评审）；**设计阶段串行；仅开发阶段并行**。

| 阶段 | 负责人 | 做什么 | 产物（doc\|link） | 门禁 |
|---|---|---|---|---|
| ① 产品设计 | Architect | 产品方案：交互、信息架构、页面结构 | 《产品设计文档》(doc) | —（产出后由②把关） |
| ② 需求评审 | PM（Architect/User 参与） | 评审需求合理性 + 产品方案；疑问与 Architect、User 确认拍板 | 评审结论（—） | **PM + Architect + User 拍板** |
| ③ RD 技术设计 | RD（scope=仅前端时无） | 后端技术方案：接口、数据、模块、依赖 | 《RD 技术设计》(doc) | — |
| ④ RD 技术评审 | RD 发起 | 评审 RD 方案；**敲定 feature 分支名**（`feature/<需求id>-<主题>`） | 评审意见（—） | **Architect / RD / FE / QA / PM 各审各的 + User 拍板** |
| ⑤ FE 技术设计 | FE（scope=仅后端时无） | 前端技术方案：组件、状态、交互实现 | 《FE 技术设计》(doc) | — |
| ⑥ FE 设计评审 | FE 发起 | 评审 FE 方案 | 评审意见（—） | **QA** |
| ⑦ 用例设计 | QA | 明确前后端各自自测什么、如何联调 | 《用例/联调设计》(doc) | — |
| ⑧ 用例设计评审 | QA 发起 | 评审用例覆盖与口径 | 评审意见（—） | **RD、FE（按需）** |
| ⑨.1 RD 开发 | RD | 开发 + 自测 + feature 分支 + MR | 代码 MR（RD）(link) | **Code Reviewer**（并行组） |
| ⑨.2 FE 开发 | FE | 开发 + 自测 + feature 分支 + MR | 代码 MR（FE）(link) | **Code Reviewer**（并行组） |
| ⑩ QA 验收 | QA | 联调 + 按用例验收；发现 bug 更新《缺陷文档》并打回对应方 | 《测试用例与验收记录》+《缺陷文档》(doc) | **QA**（打回见 §4.2 部分回滚） |
| ⑪ 用户验收 | User | 最终验收，任务结束 | — | **User** |

**评审要点**：② PM+Architect+User（拍板）；④ Architect/RD/FE/QA/PM 各审各的 + User 拍板；⑥ QA；⑧ RD、FE（按需）；⑨.1/⑨.2 Code Reviewer（并行组）；⑩ QA；⑪ User。

**产物清单**：①《产品设计文档》· ③《RD 技术设计》· ⑤《FE 技术设计》· ⑦《用例/联调设计》· ⑨.1/⑨.2 代码 MR（link）· ⑩《测试用例与验收记录》+《缺陷文档》；其余节点无产物（评审结论 = 门禁状态，疑问走评论）。模板见 `08-doc-standards.md`。

### 3.2 技术需求（tech）：8 节点 + 待启动

> 节点顺序：①需求对齐（align，无产物）→ ②RD 技术设计 → ③RD 技术评审（按需）→ ④FE 技术设计（按需）→ ⑤用例设计 → ⑥并行开发 → ⑦QA 验收 → ⑧用户验收。owner=RD；scope 默认「仅后端」或「前后端都含」。

| 阶段 | 负责人 | 做什么 | 产物（doc\|link） | 门禁 |
|---|---|---|---|---|
| ① 需求对齐（align） | RD（Architect/User 参与） | 对齐目标；疑问与 Architect、User 澄清确认拍板；**无产物、无文档模板** | —（结论落 issues discussion） | **User 拍板确认对齐结论** |
| ② RD 技术设计 | RD | 后端技术方案 | 《RD 技术设计》(doc) | — |
| ③ RD 技术评审（按需） | RD 发起 | 评审 RD 方案；**敲定 feature 分支名**（`feature/<需求id>-<主题>`） | 评审意见（—） | **Architect、FE（按需）、QA 各审各的 + User 拍板** |
| ④ FE 技术设计（按需） | FE | 前端技术方案（无前端则无此节点） | 《FE 技术设计》(doc) | — |
| ⑤ 用例设计 | QA | 自测 / 回归 / 联调口径 | 《用例设计》(doc) | **RD、FE（按需）评审** |
| ⑥ 并行开发 | RD + FE（按需） | 开发 + 自测 + MR | 代码 MR(link) | **Code Reviewer**（分别 CR） |
| ⑦ QA 验收 | QA | 按用例验收；bug 更新《缺陷文档》打回对应方 | 《测试用例与验收记录》+《缺陷文档》(doc) | **QA** |
| ⑧ 用户验收 | User | 验收通过，任务结束 | — | **User** |

**分支约定**：feature 分支名在 ③ RD 技术评审（按需）节点敲定（`feature/<需求id>-<主题>`），后续开发 / CR / QA 验收**均在同一分支**。

**产物清单**：②《RD 技术设计》· ④《FE 技术设计》（按需）· ⑤《用例设计》· ⑥ 代码 MR（link）· ⑦《测试用例与验收记录》+《缺陷文档》。模板见 `08-doc-standards.md`。

### 3.3 任务（task）：5 节点 + 待启动

> 节点顺序：①需求对齐（align）→ ②方案评审 → ③开发 → ④QA 验收 → ⑤用户验收。用途：**非需求场景**的任务（如发现 bug 并修复）——无需求澄清/范围拆解，直接对齐 → 做 → 验。owner=执行者（User/Leader 指定）。

| 阶段 | 负责人 | 做什么 | 产物（doc\|link） | 门禁 |
|---|---|---|---|---|
| ① 需求对齐（align） | 执行者（User/Leader 指定） | 明确目标、范围、验收口径，指定执行者；执行者据任务说明与架构文档核对细节，不清晰处询问确认 | —（最多沟通记录） | —（无门禁） |
| ② 方案评审 | 执行者 | 产出《方案文档》并**约定 fix 分支名**（`fix/<需求id>-<主题>`） | 《方案文档》(doc) | **QA 审核 + User 拍板** |
| ③ 开发 | 执行者 | 编码、自测、fix 分支 + MR | 代码 MR(link) | **Code Reviewer** |
| ④ QA 验收 | QA | 按任务目标测试验证、回归 | 《测试用例与验收记录》+《缺陷文档》(doc) | **QA** |
| ⑤ 用户验收 | User | 最终确认任务关闭 | — | **User** |

**产物清单**：②《方案文档》· ③ 代码 MR（link）· ④《测试用例与验收记录》+《缺陷文档》（发现缺陷时）。模板见 `08-doc-standards.md`。

### 3.4 版本规划（plan）：4 节点 + 待启动

> 节点顺序：①立项（需求对齐）→ ②调研 → ③架构规划 → ④创建需求·启动版本。owner=**Architect**。

| 阶段 | 负责人 | 做什么 | 产物（doc\|link） | 门禁 |
|---|---|---|---|---|
| ① 立项（需求对齐） | Architect（产品/User 参与） | 对齐需求，产出《需求对齐文档》；用户可自建版本号并纳入有价值需求作讨论基础；或基于上版本规划路线 | 《需求对齐文档》(doc) | **PM（产品评审）+ User 拍板** |
| ② 调研 | Investigator | 按立项需求调研竞品（GitHub 等）：哪个更优、建议引入还是重写、为什么 | 《调研文档》(doc) | **Architect** |
| ③ 架构规划 | Architect | 定版本目标（短板 / 本期解决 / 分期）、定引入或实现能力、**拆解需求** | 《架构规划》(doc) | **Leader、PM 评审 + User 拍板** |
| ④ 创建需求·启动版本 | Leader | 按架构拆解创建需求（归入该版本），大版本 待启动→进行中；**创建前查重**：有相似 → 修改原有需求而非新建；版本内有非规划需求 → 移回待分配（v-backlog） | 需求清单(doc) + 版本状态推进 | —（版本状态机单向推进） |

**产物清单**：①《需求对齐文档》· ②《调研文档》· ③《架构规划》· ④ 需求清单（doc）。模板见 `08-doc-standards.md`。

## 4. scope 裁剪与并行

### 4.1 scope 裁剪与实例化

- **模板级声明**（`flow_templates.scope_options`）：business = `["full","fe-only","be-only"]`；tech = `["be-only","full"]`；task / plan 无 scope。
- **节点级裁剪条件**（`flow_template_nodes.scope`）：`''`=全量（任何 scope 下都存在）；`'be'`=仅当需求含后端侧时存在；`'fe'`=仅当需求含前端侧时存在。**保留判定：节点保留 ⇔ scope='' 或 requirement.scope 覆盖该侧**（前后端都含 = 两侧都覆盖）。
- **评审位条目级 scope**（reviewers 内可选字段）：节点 scope 非空时评审位缺省随节点侧；节点 scope=''（全量）时，侧特定评审位显式声明 scope，全量评审位（QA/PM/User/Architect 产品层）缺省两侧都参与。
- **实例化确定性流程**（程序执行，无 persona/人工判断）：

  1. 取模板（`ORDER BY node_order`）→ 2. 按 scope 过滤节点 → 3. 过滤评审位 → 4. 生成 `flow_nodes`（`node_id` **连续编号 n-00…**，复制 key/role_id/capability_id/gate/branch/parallel_group；`flows.current_node_id=n-00`）→ 5. 并行组退化（见 §4.2）→ 6. 此后一切按实例走（wb_my_task / join / 回滚只统计实例行）。

- **fe-only 对称规则**（scope=仅前端）：④ 不实例化、⑧ 剔 **rd** 位、⑥ 仍仅 QA；② 需求评审的 Architect 仍参与（产品可行性属产品层，与前后端范围无关）。
- **be-only 规则**（scope=仅后端）：⑤⑥⑨.2 不实例化（FE 角色零节点、零任务、零评审位）、④⑧ 剔 **fe** 位、dev-1 只剩 RD 分支。
- **事后改范围**：评审/验收打回需重定范围 → `flow_nodes.pending_scope` 记录待定范围 → Leader 改 `requirements.scope` → **重跑实例化**（新增缺失节点、移除多余节点；已通过节点状态保留）。

### 4.2 并行组与 join

- **并行组**：同 `parallel_group` 的节点并行执行，各自独立 `node_id`/`state`（business `dev-1` = ⑨.1 RD 开发 + ⑨.2 FE 开发，组内两分支各自 CR 评审）。
- **join**：组内**全部「已通过」才放行组后节点**（全过 → QA 验收才从待启动进入进行中）；判定只看 `parallel_group` 同组的实例行。
- **branch（rd|fe）只是泳道名**：用于流程图双行布局与部分回滚定位，**不承担隔离**——并行/join 判定依据是 `parallel_group`。
- **部分回滚**：QA 打回 `wb_decide reject + targetBranch=fe` → 仅 FE 分支回滚到开发，RD 分支保持已通过；回滚后 join 重算再放行。
- **并行组退化**：实例化后 `dev-1` 只剩单分支（be-only 仅 RD / fe-only 仅 FE）→ 组内按**串行**推进，join 恒成立，状态机/回滚逻辑不受影响。

## 5. 能力模型（19 能力）

- **role → capability 1 对多，能力唯一归属角色**（`capabilities.role_id`）；角色 = 其拥有的能力集合；同一角色跨流程同一（RD 在业务/技术/任务中同一能力集合，能力一次定义跨流程复用）。
- **评审能力按角色拆分不共享**：arch/rd/fe/qa 各自 tech-review（视角不同），另有 pm-review / code-review / decide-cap / final-approve。
- **节点声明 capability_id**（用什么能力执行）；执行者必须拥有该能力（唯一例外：task「开发」节点 owner=executor 动态解析）。
- **reviewers 条目 `{role, capability_id, scope?}`**：role 决定谁审，capability_id 决定按哪个评审能力审（各审各的固化进 reviewers 数组）；评审维度**不单列 focus，固化在评审能力 how**。**leader 不在 reviewers**（决策者 ≠ 评审表态者，其范围/资源把关体现在 wb_decide 决策 reason）。
- **角色 → 评审能力映射**：arch→arch-tech-review、rd→rd-tech-review、fe→fe-tech-review、qa→qa-tech-review（QA 验收自评位 → acceptance-test）、pm→pm-review、cr→code-review、user→final-approve。
- **mode**：`produce`（有产物，默认）｜`align`（无产物自主对齐，capability_id=NULL、expected_outputs=[]，结论落 `issues`（discussion 型））。
- **products 单列 JSON**：`[{doc_path, template_path, demo_path}]`；**一能力可多产物**（acceptance-test 挂 qa/test.md + qa/defect.md）；评审/决策类 `'[]'`；template/demo 为 null = 模板未固化（backend-dev / frontend-dev / plan-cap）；模板外置 `docs/templates/` 文件，经 template_path 引用。

| capability_id | name | role_id | 类型 | 产出去向（简述） |
|---|---|---|---|---|
| product-design | 产品设计 | arch | 产出 | prd/prd.md（《产品设计文档》） |
| backend-design | 后端技术设计 | rd | 产出 | rd/design.md（《RD 技术设计》） |
| frontend-design | 前端技术设计 | fe | 产出 | fe/design.md（《FE 技术设计》） |
| case-design-cap | 用例/联调设计 | qa | 产出 | qa/cases.md（《用例/联调设计》） |
| backend-dev | 后端开发 | rd | 产出 | rd/dev.md（开发说明 + MR） |
| frontend-dev | 前端开发 | fe | 产出 | fe/dev.md（开发说明 + MR） |
| acceptance-test | 验收与缺陷管理 | qa | 产出 | **2 产物**：qa/test.md（《测试用例与验收记录》）+ qa/defect.md（《缺陷文档》） |
| research-cap | 竞品调研 | inv | 产出 | research.md（《调研文档》） |
| arch-cap | 架构规划 | arch | 产出 | arch_design.md（《架构规划》） |
| alignment-cap | 需求对齐 | arch | 产出 | align.md（版本规划立项《需求对齐文档》） |
| plan-cap | 方案设计（任务） | executor | 产出 | plan.md（《方案文档》） |
| code-review | 代码评审 | cr | 评审 | —（评审意见） |
| arch-tech-review | 技术方案评审（架构视角） | arch | 评审 | —（评审意见） |
| rd-tech-review | 技术方案评审（后端视角） | rd | 评审 | —（评审意见） |
| fe-tech-review | 技术方案评审（前端视角） | fe | 评审 | —（评审意见） |
| qa-tech-review | 技术方案评审（测试视角） | qa | 评审 | —（评审意见） |
| pm-review | 价值/排期评审 | pm | 评审 | —（评审意见） |
| decide-cap | 门禁决策 | leader | 决策 | —（通过/打回） |
| final-approve | 最终拍板 | user | 决策 | —（拍板确认） |

## 6. 评审职责分工（各审各的）

**铁律**：每个评审节点，各评审者只对自己职责内的维度做主要审核与把关；职责外异议**仅供参考，不作为该评审者 pass/block 的判定依据**；职责内问题未解决则该评审者不 pass（计入门禁拦截）。

| 评审者 | 主要审什么（职责内） |
|---|---|
| User | 方向 / 目标是否符合预期、是否可交付 / 可关闭 |
| PM | 需求价值、优先级、范围、排期与依赖 |
| Architect | 技术路线、方案可行性、与总体架构的一致性 |
| RD（设计方） | 设计完整性、接口/数据、自测方案 |
| FE（设计方） | 接口契约、可测性、交互可实现性 |
| QA | 可测性、用例覆盖、验收口径、缺陷闭环 |
| Code Reviewer | 代码质量、设计落地、回归风险 |

## 7. 行为协议

- **评审者 5 步统一协议**（与节点类型无关）：**读产物**（依赖文档 + 本节点产物）→ **按 focus 审**（只审职责内维度，维度来自评审能力 how）→ **评论**（wb_comment 锚到产物对应位置）→ **表态**（无阻塞 wb_review pass；有阻塞先留评论再 block 附理由）→ **复核**（作者 resolve 后 wb_confirm_review / wb_reopen）。
- **被评审者 5 步统一协议**（与节点类型无关）：**读全部意见**（按 reviewerRole 分组）→ **澄清分歧**（wb_reply 说明设计意图/权衡；不认可的意见也澄清）→ **修订 resolve**（wb_resolve note=修复说明；与提评者达成一致走 agree 关闭）→ **复核**（等提评者 confirm/reopen）→ **等推进**（全部闭环后 Leader wb_decide）。
- **差异全部参数化**在评审能力 how 与节点类型里：评审者共享同一套协议、被评审者共享同一套协议，focus 内容才是各自差异。
- **persona 只留**：专业能力卡 + 产物规范 + 3 行协作协议（① 被分配任务先调 wb_my_task，按返回的节点/产物/行为协议执行；② 评论/缺陷只传 reqId+nodeId+内容(+domain)，通知与指派由程序处理；③ 不越权：不改 flow，流程推进归 Leader wb_decide）。

## 8. 协作铁律

1. **文档先行**：PRD → 技术设计 → 原型 → 实现 → 回写文档。
2. **小步多确认**：每里程碑先产出产品文档/原型，User 确认后再实现。
3. **feature branch + MR**：代码/文档改动走分支评审。
4. **审计即证据**：需求字段变更自动写 history；流程节点状态由 decide 驱动，不手改数据。
5. **破坏性操作确认**：删除需求 / 废弃版本 / 强权通过，必须 User 明确指令。
6. **验收闭环**：每个需求有验收点，QA 核对（附《测试用例与验收记录》）后 User 确认才算完成。
7. **同一 feature 分支**：分支名在评审阶段敲定（`feature/<需求id>-<主题>` / `fix/<需求id>-<主题>`），后续 RD、FE、QA、Code Reviewer **全程看同一分支**。
8. **多任务并行用 worktree**：并行开展多个任务时各任务在独立 git worktree（各自分支）中修改，互不阻塞、互不污染工作区；单任务串行可普通分支。

## 9. 演进注记

- 流程节点定义经历多轮精简（业务需求 19 → 12+待启动、版本规划 11 → 4+待启动等），**以本文模板表为准**。