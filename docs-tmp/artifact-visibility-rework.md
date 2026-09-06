> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：action 字段（trigger 枚举、when_to_use/reference/focus）与 trigger 词汇被 v3 相位模型（on=可配相位 4 值）取代。
**仍有效**：inputs[]（ProductType+use_threads 默认 true）、sessionsByNode 废弃裁决、relatedNodes 最小返回集。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。


# 产物可见性建模 v2：节点产物视图 + 评论增量 + how 声明依赖

> **状态**：方案稿 v2（未改任何架构/代码文档）。v2 吸收用户三点纠正：①产物含评论/评审增量；②产物区分当前/历史节点；③可见性由 **capability.how 声明「看定义节点 X 的产物」** 驱动，取代 `artifactsToReview` 特设字段。
> **一句话**：每个节点有一份「节点产物视图」（文档/代码 + 其上评论线程）；角色如何获得可见性，由 **how 中声明依赖的定义节点** 决定；代码以「MR link + 可检出 branch」表达，经 **git worktree（session 级隔离）** 读取，不硬切分支。

---

## 1. 问题（现状与不满点）

| 现状 | 问题 |
|---|---|
| `artifactsToReview` 特设字段列「文档列表」 | ❌ 不好用：可见性应由 **how 声明**驱动，而非 wb_my_task 拍脑壳填字段 |
| `dependencyDocs` 只是 href 数组 | ❌ 太窄：**产物应包含文档 + 其上的评论/评审增量**（QA 评审意见 → RD 修订回复/澄清，都是产物的一部分） |
| 产物不分当前/历史 | ❌ 无法快速定位「当前节点产出 vs 前置节点产物」 |
| 实例节点无定义节点关联 | ❌ its 无法从定义表推导「看哪个节点 id 的什么文档」 |

**用户定义的正确语义**：
- **每个节点会注册的产物** = 文档（或代码 MR）+ **该产物上的评论线程**（评审意见、修订回复、澄清）——评审增量信息也是产物；
- 扁平结构化产物列表**区分当前节点产物与历史节点产物**；
- 可见性由 **capability.how 声明**：how 里写「看定义表中节点 X（如 `RD 技术设计`）的什么文档」；**实例节点带定义节点关联 id** → 程序即可推导「我该看哪个实例节点的哪份产物」。

---

## 2. 建模基础：节点产物视图 + how 声明依赖

### 2.1 实例节点关联定义节点（新增字段）

- **现状**：`flow_nodes`（实例）只有 `key`（中文名），`flow_template_nodes`（定义）有 `id + flow_templates_id + node_order + key`——**无显式关联**。
- **方案**：`flow_nodes` 增加 `template_node_id`（定义节点 id 引用），实例化时从定义节点复制。此后可从「能力 how 里声明的定义节点 id」反向定位实例节点。

```
定义表 flow_template_nodes:  { id: 4, flow_templates_id: 'business', node_order: 4, key: 'RD 技术评审', capability_id: 'rd-tech-review', reviewers: [...] }
实例表 flow_nodes:           { node_id: 'n-04', template_node_id: 4, key: 'RD 技术评审', capability_id: 'rd-tech-review', ... }
```

### 2.2 节点产物视图（产物分层：文档挂评论；顶层 doc/link/节点评论平等）

每个节点维护一份「节点产物视图」。产物**分层组织**：

- **文档的评论（行级/块级/文档级锚点评论）挂在文档下面**（作为 `document` 的子结构，不单独平级在顶层）；
- **顶层平级**的是：`document`（含其评论）、`link`（代码/外链）、**节点级评论**（不锚定文档的对话）——三者暂时视为平等成员。

```
NodeArtifactView(node_id):
  products: [
    { kind: 'document', id, label, href, producedBy, readMode,
      comments: [ { …Issue（锚定本文档的 行/块/文档级 评论） } ] },   // ★ 文档的评论挂在文档下
    { kind: 'link',     id, label, href, producedBy, readMode, featureBranch? },
    { kind: 'comment',  …Issue（节点级对话，anchor 不锚定文档） }      // ★ 顶层：节点评论（平等成员）
  ]
  按当前节点分组视角（见 §2.4）：
    currentNode  → ownerProducts（节点 owner 交付：文档[含其评论]/代码）
                   / reviewerProducts（评审成员产物：文档评论挂在文档下；节点级对话为顶层 comment）
    relatedNodes → products[]（document[含评论]/link/节点级 comment 平等混排）
  status:    state / enteredAt / rolledBackCount     // 节点状态（快速定位）
```

**评论作为产物的本质**：QA 审 RD 技术设计 → 在文档锚点提意见（挂到该文档的 comments）→ RD 修订 → 回复「已修订」或澄清「设计意图如此」——这条**对话链**记录需求为何这样演化，是评审判断的重要输入（不只读文档定稿，还要读"改了什么、为什么"，即文档下的评论增量）。节点级对话（不锚定文档的澄清/介入）则作为顶层 comment 产物。

### 2.3 可见性由 capability.how 声明（取代 artifactsToReview）

**核心思路**：`capabilities[capability_id].how` 中声明「本能力要看定义表中哪些节点的什么产物」；实例节点凭 `template_node_id` 关联，程序即可推导可见内容。

**how 声明格式（扩展）**——在 `how` 之外增加结构化 `inputs` 字段（显式依赖声明）：

```
capabilities 行（评审能力示例，如 qa-tech-review）:
  {
    "capability_id": "qa-tech-review",
    "how": [ "读输入：产品设计文档（prd）+ RD 技术设计（rd/design）…", … ],
    "inputs": [                       // ★ 结构化依赖声明（新增）
      { "node_key": "产品设计",        // 定义节点 key → 实例经 template_node_id 找到对应实例节点
        "artifacts": ["prd/prd.md"],  // 该节点的哪些产物（相对 doc_path）
        "kind": "document",
        "use_threads": true },        // 是否连该产物上的评论增量一起给
      { "node_key": "RD 技术设计",
        "artifacts": ["rd/design.md"],
        "kind": "document",
        "use_threads": true }
    ]
  }
```

> **说明（2026-08-26 对齐）**：§2.3 的 `inputs` 示例为早期 `node_key` 形式；**以 §2.6 ProductType 标准产物类型为最终形态**（`inputs: [{ productType: "rd_design", use_threads: true }]`，不写 node_key/order）——能力跨流程复用；how/ref 中产物引用为「《标准名》(路径模板)」，真实路径由 host 端 `renderTpl(s, ctx)`（替换 `{{req-version}}`/`{{req-name}}`/`{{side}}`，JS 实现见 capability-review-v2 §0.1）渲染（J7 how 模板化定稿）。

**推导流程（opMyTask）**：

```
1. 取当前节点 flow_nodes[node].capability_id + template_node_id
2. 读 capabilities[capability_id].inputs（结构化依赖声明）
3. 对每条 input：
   a. 在实例 flow_nodes 中按 node_key（或 template_node_id）找到对应实例节点
   b. 取其 NodeArtifactView（artifacts + threads）
4. 组装可见产物（区分当前节点产物 vs 历史节点产物，见 §2.4）
```

> 说明：`inputs` 是「how」的结构化补充，两者同源（how 的自然语言是从 inputs 可读性转述）；只读依赖声明在评审能力上，产出能力也可声明其前置输入。

### 2.4 节点产物视图（分层产物模型 + 当前节点按产出者分组）

**产物分层组织**：
- **文档的评论（行级/块级/文档级锚点评论）挂在文档下面**——`document` 产物内嵌 `comments[]`，不单独平级在顶层；
- **顶层平等成员**：`document`（含其评论）、`link`（代码/外链）、**节点级评论**（不锚定文档的对话）。

```
产物（product）统一模型：
  document: { kind, id, label, href, producedBy, readMode,
              comments: [ Issue（锚定本文档的 行/块/文档级 评论） ] }
  link:     { kind, id, label, href, producedBy, readMode?, featureBranch? }
  comment:  { kind:'comment', …Issue（节点级对话，anchor 不锚定文档） }
```

**currentNode 的产物按「产出者」分组**（而非按文档/链接分类）：
- `ownerProducts`：**节点 owner（产出者）**的产物——如评审节点被审的《RD 技术设计》（含其上评审评论，挂在文档下）、开发节点 RD 的代码 MR；
- `reviewerProducts`：**评审成员**的产物——文档评论挂在对应文档的 `comments` 下；节点级对话（不锚定文档）作为顶层 comment 产物。

完整示例（RD 技术评审节点，评审成员 PM/FE/Arch/QA）：

```jsonc
{
  "ok": true, "reqId": "req-019", "role": "qa", "nodeId": "n-04", "nodeKey": "RD 技术评审",

  "currentNode": {
    "instanceId": "n-04", "templateNodeId": 4, "key": "RD 技术评审",

    "ownerProducts": [                                            // ★ 节点 owner（RD）的产物
      { "kind": "document", "id": "art-rd", "label": "《RD 技术设计》",
        "href": "docs/…/rd/design.md", "producedBy": "RD", "readMode": "document",
        "comments": [                                             // ★ 文档的评论挂在文档下
          { "issue_id": "cm-qa-line",                             // 行级锚点评论
            "anchor": { "kind": "line", "artifactId": "art-rd", "line": 42,
                        "lineText": "接口 /settle/commit 缺幂等键" },
            "author": "QA", "reviewerRole": "QA", "type": "issue", "state": "open",
            "text": "幂等性：重试是否会重复扣款？建议加 idempotency-key。", "roundCount": 2,
            "thread": [ { "by": "RD", "role": "rd", "text": "已补幂等键，见接口 §3.2", "at": "…" } ] },
          { "issue_id": "cm-fe-block",                            // 块级锚点评论
            "anchor": { "kind": "block", "artifactId": "art-rd", "blockStart": 22,
                        "blockEnd": 30, "lang": "sql" },
            "author": "FE", "reviewerRole": "FE", "type": "issue", "state": "open",
            "text": "这段 SQL 的联调用例没覆盖", "roundCount": 1, "thread": [] },
          { "issue_id": "cm-arch-doc",                            // 文档级评论
            "anchor": { "kind": "document", "artifactId": "art-rd" },
            "author": "Architect", "reviewerRole": "Architect", "type": "issue", "state": "open",
            "text": "技术路线与版本架构一致，但并行 join 需补回滚说明", "roundCount": 0, "thread": [] }
        ] }
    ],

    "reviewerProducts": [                                        // ★ 评审成员：节点级对话（顶层 comment 产物）
      { "kind": "comment", "issue_id": "cm-pm-node",            // 节点级对话（不锚定文档）
        "anchor": { "kind": "document" },
        "author": "PM", "reviewerRole": "PM", "type": "discussion", "state": "open",
        "text": "如 RD/FE 并行，联调窗口怎么排期？", "roundCount": 0, "thread": [] }
    ],

    "outputs": [ /* 本节点应产出（若为产出型能力） */ ]
  },

  "relatedNodes": [                                             // 其他/上下节点：顶层 doc/link/节点评论平等
    { "node": { "instanceId": "n-01", "templateNodeId": 1, "key": "产品设计" },
      "products": [
        { "kind": "document", "id": "art-prd", "label": "《产品设计文档》",
          "href": "docs/…/prd/prd.md", "producedBy": "Architect", "readMode": "document",
          "comments": [                                         // ★ 文档评论挂文档下
            { "issue_id": "cm-prd-note",
              "anchor": { "kind": "document", "artifactId": "art-prd" },
              "author": "PM", "reviewerRole": "PM", "type": "discussion", "state": "resolved",
              "text": "范围补充确认…", "roundCount": 1, "thread": [] }
          ] }
      ] },
    { "node": { "instanceId": "n-09", "templateNodeId": 9, "key": "RD 开发" },
      "products": [
        { "kind": "link", "id": "art-mr-rd", "label": "代码 MR（RD）",
          "href": "https://…/pull/123", "producedBy": "RD",
          "readMode": "worktree", "featureBranch": "feature/req-019-m7-drill" }
      ] }
  ]
}
```

**命名/结构语义**：
- `currentNode`：当前节点，产物按产出者分两组——`ownerProducts`（节点 owner 交付：文档[含其评论]/代码）+ `reviewerProducts`（评审成员的节点级对话）；
- `relatedNodes`（其他/上下节点）：每个节点一个 `products[]`，顶层 **document（含评论）/ link / 节点级 comment 平等**混排——它们是"我评审/产出需要参照的产物"；
- 不用 `visibleNodes`：可见性是**推演阶段**的东西，不是数据本身的属性（数据是"哪些节点的产物"，推演是"我此刻能看到哪些"）。

**统一结构要点**：
- **文档的评论挂在文档下**：行/块/文档级锚点评论是 `document.comments[]` 的子结构，不单独平级在顶层；
- **顶层平等**的是 document（含其评论）/ link / 节点级 comment（不锚定文档）——三个平级成员；
- currentNode 中：锚定文档的评审意见在 `ownerProducts[].comments` 内（挂在被审文档下，按 reviewerRole 区分提评者），节点级对话在 `reviewerProducts[]`；
- 代码产物（kind=link）带 `readMode: worktree` + `featureBranch`（见 §3）。

### 2.5 评论产物结构体模型（Issue / Anchor / Thread）

> 评审评论统一走 `issues` 统一模型（评论 = 问题），挂载位置分三类：**文档级**（document）、**锚点级**（line/range/block，定位到文档具体位置）、**节点级**（直接挂节点，不锚定文档）。评审节点（如 RD 技术评审）多个评审成员（PM/FE/Arch/QA）各自提意见，经 `reviewerRole` 归属。

```ts
// ============ 锚点（评论挂在哪里） ============

type AnchorKind = 'document' | 'line' | 'range' | 'block'

interface Anchor {
  kind: AnchorKind
  artifactId?: string      // 锚定产物（document/line/range/block 均指向某 artifact）；
                           // 缺省 = 节点级对话（不锚定具体文档）
  // --- line 级 ---
  line?: number            // 单行号（1-based）
  lineText?: string        // 该行去首尾空白原文（定位指纹，防漂移）
  // --- range 级 ---
  startLine?: number
  endLine?: number
  // --- block 级 ---
  blockStart?: number
  blockEnd?: number
  lang?: string            // 代码块语言（如 sql/mermaid）
}

// ============ 线程回复（不可变追加） ============

interface ThreadReply {
  reply_id: string         // r-<ts>
  by: string               // 发言者（角色名 / User）
  role?: string            // 角色 key（rd/fe/qa/arch/pm/cr/user…）
  text: string
  at: string               // ISO 时间
}

// ============ 评论/问题 ============

type IssueType = 'issue' | 'discussion' | 'notification'
type IssueState = 'open' | 'pending-confirm' | 'resolved' | 'overridden'

interface Issue {
  issue_id: string         // cm-<ts>
  requirement_id: string
  node_id: string          // 挂载节点（如 n-04 RD 技术评审）
  anchor: Anchor           // ★ 挂载位置：document / line / range / block / 节点级（缺 artifactId）

  author: string           // 发表者显示名（QA / PM / FE / Architect / RD / User）
  role: string             // 作者角色 key
  reviewerRole: string     // ★ 评审归属（门禁/override 统计；如 'QA'）——评审成员各自意见的归属
  type: IssueType          // issue（阻塞门禁）| discussion（纯沟通）| notification（通知）
  assignee: string         // 解决责任人（L1/L2/L3；缺省=锚点产物作者；QA 缺陷显式指派）

  text: string
  state: IssueState
  roundCount: number       // 来回轮次（>5 触发拉锯升级）
  lastReplyAt?: string
  thread: ThreadReply[]    // ★ 子对话（修订回复 / 澄清 / 讨论）

  // resolve 双路径关闭
  resolve?: {
    by: string             // 谁关闭（assignee 修复关闭 / 提评者协商一致关闭）
    at: string
    note?: string          // 修复说明 或 一致结论
    path: 'fix' | 'agree'  // 修复关闭 / 协商一致关闭
    reviewerConfirmed?: boolean   // 提评者复核确认（A 路径 resolve 后需 confirm）
  }
  at: string               // 根评论创建时间
}
```

**挂载位置语义**：

| anchor.kind | artifactId | 典型场景 |
|---|---|---|
| `document` | 有 | **文档级**评论（挂在文档正文末尾的整篇讨论） |
| `line` / `range` / `block` | 有 | **锚点级**评论（定位到文档某行/范围/代码块，如 QA 在接口定义某行提幂等意见） |
| `document`（或省略） | 无 | **节点级对话**（直接挂在节点上的沟通，不锚定文档——如评审者与 RD 的澄清、User 介入发言） |

**评论产物的组织（评审节点，多评审成员）**：

```
文档的评论（行/块/文档级锚点）→ 挂在对应文档下：ownerProducts[].document.comments[] / relatedNodes[].document.comments[]
  └─ 每条评论的 reviewerRole 标识提评的评审成员（PM/FE/Arch/QA…）——按成员聚合即「各审各的」的意见组

节点级对话（不锚定文档）→ 顶层 comment 产物：
  currentNode.reviewerProducts[] / relatedNodes[].products[]（kind=comment）

relatedNodes[].products: (document[含评论] | link | 节点级 comment)[]   // ★ 上下节点：顶层三者平等混排
```

> 对应落库：`arch_design.md §7.2 Issue 实体`（anchor_kind/anchor_artifact_id/anchor_line…/author/role_id/reviewer_role/issue_type/assignee/text/state/round_count/thread（replies JSON）/resolve 族）。本模型为传输视图，字段与落库一致，仅结构化为嵌套（anchor 对象 + thread 数组）。

---

## 3. 代码读取：基于 branch 的 git worktree（session 级隔离）

### 3.1 原则

- **不硬切分支**：绝不在主工作区 `git checkout` 切到 feature 分支（会污染他人/主分支）；
- **session 级隔离**：用 `git worktree add <临时目录> <branch>` 检出到独立目录，用完 `git worktree remove`；
- 检出目录与 session 绑定，不影响其他 agent/用户的默认工作区。

### 3.2 流程（QA 验收 / CR 读代码为例）

```
1. wb_my_task 返回 relatedNodes：含代码产物（kind=link）+ fromNode.featureBranch + readMode=worktree
2. Leader/QA 按 worktree 检出：git worktree add .worktown/req-019.qa <feature/req-019-m7-drill>
3. QA 在主 worktree 目录读代码（目录绑定本 session，无全局副作用）
4. 拉取最新：git -C <worktree> pull origin <branch>（可选，按需）
5. 验收/评审完成 → git worktree remove <目录>（清理，或会话结束时统一清理）
```

### 3.3 安全与边界

- **MR 地址不直接读**：`href` 仅作人可点击跳转（review 记录、审计），程序不抓取 MR 网页内容；
- **代码可读性来源** = worktree 检出的 feature 分支目录（真实代码），供 fs/代码工具读取、搜索、锚点；
- worktree 目录约定：`<workspace>/.worktown/<reqId>.<role>`（不可见目录，避免污染项目根）；
- **readMode 语义**：`document`（阅读器/直读）｜`worktree`（检出分支读代码）｜`link`（仅跳转，不读内容——用于 MR 地址）。

---

### 2.6 标准产物类型枚举（ProductType，解决能力复用性）

> **问题（严重）**：能力是可复用的（4 个流程共享同一能力），但此前 how/action 用 `template_node_id`（或 node_order）定位产物，**跨流程失效**——同一能力在不同流程绑定的是不同的定义节点/节点位置：
> - `backend-dev`：business `RD 开发`(order9) / tech `并行开发`(order6) / task `开发`(order3)；
> - `acceptance-test`：business QA验收(order11) / tech QA验收(order7) / task QA验收(order4)；
> - 且产物命名不统一：`case-design-cap` 产出 business《用例/联调设计》vs tech《用例设计》；`backend-dev` 产出 business《RD 开发说明》vs task《开发说明》。
>
> **根因**：能力真正关心的是「**我要读/产哪种类型的产物**」，而非「流程中的哪个节点/哪个序号」。用节点位置定位 = 把"能力对产物的依赖"错写成"能力对流程结构的依赖"。

**方案：标准产物类型枚举（ProductType）**——能力用标准类型声明依赖，wb_my_task 把当前上下文映射为标准类型，能力据此跨流程复用。

```
ProductType（标准产物类型，全局统一、跨流程）：
  // 文档类
  product_doc      // 《产品设计文档》（product-design 产出）
  rd_design        // 《RD 技术设计》（backend-design）
  fe_design        // 《FE 技术设计》（frontend-design）
  case_design      // 《用例/联调设计》（case-design-cap，tech 称《用例设计》）
  dev_notes        // 《开发说明》（backend-dev/frontend-dev，business 带 RD/FE 前缀）
  test_result      // 《测试用例与验收记录》（acceptance-test 产出1）
  defect           // 《缺陷文档》（acceptance-test 产出2）
  plan             // 《方案文档》（plan-cap）
  research         // 《调研文档》（research-cap）
  architecture     // 《架构规划》（arch-cap，版本级 arch_design.md）
  alignment        // 《需求对齐文档》（alignment-cap，版本级 align.md）
  roadmap          // 《版本规划》（roadmap.md，版本级）
  // 代码/链接类
  code_mr          // 代码 MR（开发节点 kind=link 产物）
  feature_branch   // 特性分支（tech 设计阶段敲定，与 code_mr 关联）
```

**能力如何用 ProductType（替代节点位置）**：
```
inputs: [
  { productType: "rd_design", use_threads: true },   // 找当前流程上下文里的《RD 技术设计》产物
  { productType: "product_doc", use_threads: true }
]
action 的 reference 同理：[{ "productType": "rd_design" }, …]   // 不写 node_key/order
```

**wb_my_task 如何保证 ProductType 可命中**：
1. `flow_template_nodes[节点].expected_outputs` 的 label/模板 → **绑定 ProductType**（定义层机器可读）；
2. 产物登记 `wb_add_artifact` 时带 `product_type`（标准枚举字段）——即使 label 或节点位置跨流程不同，`product_type` 始终稳定；
3. 展开可见产物时，按 `product_type` 分组/索引，能力按类型精确取「哪一个产物」。

**防填错名词**：ProductType 是集中定义的受控枚举（`docs/v1.0/arch_design.md` 或独立表），新增 flow 时产出节点只能"绑定某标准类型"，**不允许自造**（如把「产品设计文档」写成「产品文档」）→ 从命名上杜绝错填、保证能力跨流程复用。

**复盘（为何先前漏检）**：推演 how/action 时潜意识只基于 business 流程的 node_order（node3=RD设计、node7=用例设计），把 business 的节点映射当默认/通用；未做「同一能力在 4 个流程中实例化时 how 是否仍有效」的跨流程复用验证，导致漏检。

---

## 4. 与现有设计的衔接（无破坏）

| 现有机制 | 变化 |
|---|---|
| `flow_template_nodes` 定义节点 | 不变；`expected_outputs` 绑定 **ProductType**（替代 label 文本识别） |
| `flow_nodes` 实例节点 | **新增 `template_node_id`**；产物经 `product_type` 索引 |
| `capabilities` | **新增 `inputs`/`actions` 声明依赖 = ProductType 枚举**（不再写 node_key/order） |
| 评论即产物 | 不新增表：`artifacts`（document/link）+ `issues`（comment）——统一进 `products[]`，各带 `product_type`（comment 可为 node 级） |
| `wb_my_task` 返回字段 | `relatedNodes[].products[]` 每项带 `productType`；能力按类型取产物，跨流程复用 |
| 协作铁律 ⑦ 同一 feature 分支 / ⑧ worktree | 天然一致，无需改 |

## 5. 待办（方案确认后落地）

1. **`flow_nodes.template_node_id` 字段**：schema + 实例化复制（seed/实例数据回填）；
2. **ProductType 枚举定义**：建立受控枚举（`arch_design.md` 或独立 def），`flow_template_nodes.expected_outputs` 与 `wb_add_artifact` 登记均绑定 product_type；
3. **`capabilities.inputs`/`actions` 声明依赖 = ProductType**：为 19 个能力补输入/动作（替代 node_key 定位，解决跨流程复用）；
4. **评论统一为产物（kind=comment）**：issues 输出模型对齐 `products[]`；
5. **`opMyTask` 推导升级**：按 productType → currentNode/relatedNodes，不再硬编码节点位置；
6. **`feature_branch` 写入时机实装**：RD 技术评审/方案评审节点 `wb_decide pass` 时敲定写入；
7. **worktree 工具/指南**：`git worktree add/remove` 标准用法与清理策略（落 wb-agent-guide skill）；
8. **跨流程复用验证用例**：同一能力（qa-tech-review / backend-dev / acceptance-test）在 business/tech/task 各流程实例化，校验 inputs 按 productType 命中产物。