# M7：评论与解决（Discussion & Resolve）完整设计

> 状态：**v0.1（设计稿，待 User review）**
> 背景：当前评论模型只有 `open/resolved/overridden` 三态 + 单线程回复，无法支撑多评审者、多锚点、远端 GitHub CR 等真实场景。本文定义完整对象模型、锚点冲突、状态机、远端同步与通知链路。

## 1. 评论对象模型（comments.json 扩展）

```ts
interface Comment {
  id: string                    // cm-<ts>
  requirementId: string
  nodeId: string                // 挂载的流程节点（场景由此区分）
  anchor: Anchor                // 锚点（定位到文档/代码）
  author: string                // 发表者（角色名）
  role: string                  // 内部角色 key
  reviewerRole?: string         // 评审归属角色（门禁/override 统计）
  text: string
  at: string
  state: 'open' | 'pending-confirm' | 'resolved' | 'overridden'
  assignee?: string             // 解决责任人：缺省=评论锚定文档/产物作者（或节点 owner）；仅 QA 验收缺陷显式指派 RD|FE
  thread: ThreadReply[]         // 多参与者讨论（reply 支持纯沟通，见 §3 关闭路径 B）
  roundCount: number            // 往返轮数
  lastReplyAt?: string
  resolve?: {                   // 关闭信息（双路径）
    by: string                  // 谁关闭（assignee 修复关闭 / 提评者协商一致关闭）
    at: string
    note?: string               // 修复说明 或 一致结论
    path: 'fix' | 'agree'       // 关闭路径：修复关闭 / 协商一致关闭
    reviewerConfirmed?: boolean // 提评者确认（A 路径 assignee resolve 后需 confirm；B 路径直接 true）
  }
  source: 'workbench'              // 恒为 workbench（GitHub 评审结论本地化落 comment）
}

interface Anchor {
  kind: 'document' | 'line' | 'range' | 'block'
  artifactId: string
  // line 级
  line?: number
  lineText?: string
  // range 级
  startLine?: number
  endLine?: number
  // block 级
  blockStart?: number
  blockEnd?: number
  lang?: string
}

interface ThreadReply { by: string; role?: string; text: string; at: string }
```

## 2. 多评审者与锚点冲突（场景 ①②）

**结论：评论按 id 独立存在，锚点只定位不唯一——同一行/同一文档可有多条评论，互不冲突。**

| 场景 | 处理 |
|---|---|
| RD 技术评审多评审者（QA/PM/User/FE） | 每人提独立评论（reviewerRole=各自），门禁按角色统计各自 open 评论 |
| QA/PM 评**同一行** | 同锚点两条独立评论，各自 thread、各自 resolve；UI 聚合显示（锚点计数 badge + 展开列表） |
| 文档级 + 行级 + 块级混用 | anchor.kind 区分；同一 artifact 的评论统一在阅读器按锚点归类 |
| 过滤/审计 | authorFilter / reviewerRole / stateFilter 已支持按作者/角色/状态查看 |

## 3. 解决状态机（场景 ③：评审/CR/缺陷通用）

```
open ──提评──→ open（讨论中：thread 回复、roundCount++）
   │
   ├─ A 修复关闭：assignee（解决责任人）resolve(note=修复说明)
   │     ──→ pending-confirm（待提评者复核）
   │          ├─ 提评者 confirm ──→ resolved（reviewerConfirmed=true，门禁放行）
   │          └─ 提评者 reopen（如 QA 缺陷复核不通过）──→ open（roundCount++，通知 assignee）
   │
   ├─ B 协商一致关闭：thread 交流达成一致 ──→ 提评者 resolve(note=一致结论, path=agree)
   │     ──→ resolved（一致即确认，reviewerConfirmed=true）
   │
   ├─ roundCount > 5 ──→ 通知 Leader/User 介入（拉锯升级）
   │
   └─ 强权 override（User 明确指令）──→ overridden（绕过门禁）

assignee 规则：
  - 缺省：普通评审意见 → 处理者 = 评论锚定文档/产物作者（谁的产品谁改，如 FE 设计评审→FE、RD 技术评审→RD、CR 评审→开发作者）——无需显式指派
  - 显式：仅 QA 验收缺陷 → QA 建缺陷时传 assignee=RD|FE（按缺陷归属端判定，配合 targetBranch 部分回滚）
QA 缺陷特殊化（无 RD 打回、QA 最终裁定）：
  QA 建缺陷（提评者=QA，assignee=指派 RD/FE）
  → RD/FE 处理：真缺陷修复 resolve(note=修复说明)；认为误判/非缺陷 → 直接 resolve(note=理由)，不存在"RD 打回 QA"——带理由解决即异议表达
  → QA 复核（最终裁定）：认可（修复 OK / 理由成立）→ confirm 缺陷关闭；不认可修复结果 → reopen（即"QA 打回对方解决"），RD/FE 继续
  与流程级 reject 区分：wb_decide reject(targetBranch) 是 QA 验收整体打回开发阶段（批量退回），评论 reopen 是单条缺陷不认可修复
```

- **谁 resolve**：A 路径 = assignee（解决责任人，QA 缺陷即缺陷负责人 RD/FE）；B 路径 = 提评者（协商一致直接关闭）；resolve 必须带 note
- **谁 confirm/reopen**：仅提评者（不能自确认；QA 缺陷场景即 QA 复核，不通过 reopen）
- **复核闭环**：A 路径 resolve 后通知提评者复核 → confirm 放行 / reopen 拉锯；B 路径一致即关闭
- **门禁判定**：`wb_decide pass` 要求 ① 无 open 评论 ② 无 pending-confirm 评论 ③ 全部 resolved 且 `reviewerConfirmed=true`（或 overridden）
- **轮数**：roundCount 记录往返次数，防无限拉锯（超阈值 UI 标红提示）

## 4. 远端 GitHub CR（场景 ④，简化方案）

**决策：远端评审结论本地化——无远端评论。**

```
GitHub PR（feature/<需求id>-<主题> 分支）
   │
   ├─ CR 在 GitHub 上评审（代码在远端仓库）
   │
   └─ 评审结论 → Leader / CR 子 agent 在**工作台落一条本地评论**
       （author=Code Reviewer，nodeId=开发节点，anchor=MR link 或开发说明，
        text=评审结论：通过/阻塞问题清单）
       → 开发者按本地评论修复 → resolve(note) → CR 复核确认 → wb_review pass
```

- 工作台评论 `source` 恒为 `workbench`（不设远端字段）——评论模型保持单一来源
- GitHub 仅作为 CR **信息来源**（CR 在远端看完代码，把结论落到工作台），不引入双向同步
- 锚点：本地评论可 anchor 到《开发说明》/MR 链接对应位置；GitHub 行级上下文在评论 text 中体现（如 "path:line"）
## 5. 通知链路（配合 wb_comment/resolve 的 to 语义）

| 动作 | 通知（to=） | 内容 |
|---|---|---|
| 作者提评（交付） | 当前节点评审归属角色 | 请评审 |
| 评审者提评审意见 | 作者（RD/FE） | 问题列表，请修复 |
| 作者回复/修复 | 提评者 | 已修复，请复核 |
| **assignee resolve（A）** | 提评者（reviewer） | 已修复，请复核 |
| **提评者 confirm** | （无，门禁放行） | — |
| **提评者 reopen** | assignee | 复核不通过，请继续 |
| **协商一致关闭（B）** | （无） | 讨论达成一致，直接关闭 |
| QA 缺陷建立 | 缺陷负责人 RD/FE | 缺陷指派，请修复 |
| QA 缺陷打回 | 对应分支 owner（RD/FE） | 部分回滚 + 缺陷详情 |

## 6. 工具变更点

| 项 | 变更 |
|---|---|
| `wb_reply` | 支持任意参与者回复（by=调用者角色），roundCount++；thread 保留多作者 |
| `wb_resolve` | 参数加 `note` + `path`（fix/agree）+ `assignee`（默认节点 owner）；A 路径→pending-confirm、B 路径→resolved |
| 新增 `wb_confirm_review` | 提评者复核确认：pending-confirm → resolved（校验调用者=提评者/assignee 非本人） |
| 新增 `wb_reopen` | 提评者复核不通过：pending-confirm → open（roundCount++，通知 assignee；QA 缺陷场景即 QA reopen） |
| `wb_review` | pass 前置升级：无 open + 无 pending-confirm + 全部 resolved 且 reviewerConfirmed |
| roundCount>5 | 自动通知 Leader/User（拉锯升级） |
| 轮询任务（每 5 分钟） | 扫描 pending-confirm：提评者=User 超时 1 小时、=agent 超时 5 分钟 → 提醒提评者复核（host 定时任务） |
| UI | 阅读器锚点聚合计数；resolve 需 note；待复核标记；roundCount 轮数显示；超时 badge |

## 6.1 UI 操作判定矩阵（一条评论的操作面板动态渲染）

| 评论状态 | 操作者 | 可操作 | 判定 |
|---|---|---|---|
| open（讨论中） | 任何人 | **回复（reply）** | 继续交流/提问/澄清——状态不变，只推进讨论（roundCount++） |
| open | **assignee**（普通评审=文档/产物作者，缺陷=QA 指派 RD/FE；协调者可代） | **resolve（A 修复关闭）** | 已修复 → 填 note=修复说明 → 转 pending-confirm |
| open | **提评者**且讨论已一致 | **resolve（B 协商一致关闭）** | 达成一致 → 填 note=一致结论 → 直接关闭（confirmed=true） |
| open | 非 assignee 非提评者 | 仅 reply | 表达意见，等 assignee/提评者收尾 |
| pending-confirm（已解决待复核） | **提评者** | **confirm（通过）/ reopen（不通过）** | 最终裁定（QA 缺陷即 QA 复核，不通过 reopen） |
| pending-confirm | 非提评者 | 无操作（只读） | 等待提评者复核 |
| resolved / overridden | — | 只读 | 可展开查看 resolve 记录（by/note/path） |

**判定一句话**：
- **reply**：评论未关、还想继续说（提问/澄清/讨论）
- **resolve**：评论要关——你修复了（A，assignee）或双方谈拢了（B，提评者）；必须填 note
- **confirm/reopen**：评论已 resolve 过、你是提评者做最终裁定

**User（协调者）特判**：可代任一身份操作——resolve 时代 assignee（A）或代提评者（B），confirm/reopen 时代提评者；UI 需让 User 指明扮演角色（操作归属用 reviewerRole/assignee 的谁），避免身份混淆。

**UI 动态渲染**：
- 回复输入框：open 态常驻
- resolve 按钮：open 态 + 有资格时（点击前强制填 note，可选择关闭路径 A/B）
- confirm / reopen：pending-confirm 态 + 是提评者时

## 6.2 决策落定（评审确认）

- [x] **复核强制**：resolve 后必须提评者复核确认（reviewerConfirmed=true）才放行门禁
- [x] **reopen 上限**：roundCount > 5 自动通知 Leader/User 介入
- [x] **GitHub CR**：远端评审结论本地化落 comment（无远端评论，无同步）
- [x] **评论不可编辑/删除**：评论不可变（保留审计）；误发用 reply 标注「作废」，不提供编辑/删除
- [x] **待复核超时提醒（轮询，每 5 分钟）**：pending-confirm 挂起超时 → 提醒提评者复核；**提评者为 User → 超时阈值 1 小时；提评者为 agent（子 agent 角色）→ 超时阈值 5 分钟**；挂起期间不阻塞其他评论/流程（仅超时提醒）
- [x] **重复问题不合并**：同锚点 = 不同问题（可能各自独立），各自 thread/resolve，不合并
- [x] **缺陷打回语义**：无"RD 打回 QA"——RD 带理由解决即异议；QA 不认可修复 → reopen（打回对方解决）；流程级 reject(targetBranch) 为验收整体打回（§3）
- [ ] **存储**：v1 用 JSON 文件（单写入者 + git 审计），sqlite 为演进目标
- [x] **UI 操作判定矩阵**：reply / resolve(A·B) / confirm / reopen 按状态+操作者动态渲染（§6.1）

## 7. 待确认项

- [x] 超时阈值：User 1 小时 / agent 5 分钟（§6.2）
- [x] 轮询频率：每 5 分钟（§6）
