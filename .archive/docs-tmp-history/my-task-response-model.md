> ⚠️ **结论时效（2026-09-05 v3 编排模型定稿后标注）**：本文为历史推演层——**已取代**：返回结构全表（sessionsByNode、roleObjective/behaviors/onFeedback、capability.how[]）——被 data-tools-final 的 wb_my_task v3 契约取代（currentPhase + activeActions{action_id,on,skill|inline,上下文} + relatedNodes 最小集 + three-way 校验）。 现行权威：`capability-model-final.md` v3（两层属性状态机）+ `business-flow-derivation.md` / `flow-derivation-tech-task-plan.md` v3.1 + `capability-actions-final.md` + `data-tools-final.md`。

> **追加废弃指针（2026-09-08）**：本文件之外，最新模型已迭代为 v3.1+（两层属性状态机 node.state 4 相位 + **per-agent 评审轮** reviewing/agree/unagree + collect 内置聚合 + review_results 单字段 + 评审能力对象化 {role}.{object}-review + 技术设计无内嵌 CR + task 无 QA 节点 + scope 渲染集合表达式 rd|full/fe|full/full/rd|fe + QA 验收仅 full 渲染）——任何与本文件一致的旧记忆引用，一律以 `docs-tmp/capability-model-final.md` v3.1 现行版为准。


# `wb_my_task` 预期返回 JSON 模型（示例）

> **依据**：`artifact-visibility-rework.md`（产物可见性建模 v2）+ `.workbench/req-019` 真实数据。
> 以下为 **`wb_my_task({ reqId: 'req-019', role: 'qa', nodeId: 'n-04' })`**（RD 技术评审节点，gate=true，reviewers=Arch/RD/FE/QA/PM）的**目标态完整返回**。
> 结构遵循 rework §2.4/§2.5：**文档的评论（行/块/文档级锚点）挂在文档下**（`document.comments[]`）；**顶层平等**的是 document（含评论）/ link / 节点级 comment；`currentNode` 产物按产出者分 `ownerProducts` / `reviewerProducts`。

```json
{
  "ok": true,
  "reqId": "req-019",
  "role": "qa",
  "nodeId": "n-04",
  "nodeKey": "RD 技术评审",

  "mode": "produce",
  "gate": true,
  "state": "待评审",

  "task": "评审+表态：RD 技术评审",

  "roleObjective": "评审 focus=可测性/联调口径/用例覆盖（各审各的）",
  "action": {
    "action_id": "qa-tech-review.review",
    "name": "评审（RD 技术评审）",
    "trigger": "node_review_pending",
    "when_to_use": "节点是 gate 且处于待评审（产物已提交，评审者待表态）",
    "how": [
      "读输入：产品设计文档 + RD 技术设计",
      "按 qa-tech-review 维度评审：接口可测性、联调口径、验收用例覆盖",
      "评论 wb_comment(reviewerRole=QA)",
      "表态 wb_review(pass|block)"
    ],
    "reference": [{ "productType": "product_doc" }, { "productType": "rd_design" }],
    "focus": ["可测性/联调口径/用例覆盖"],
    "output": "评审意见/表态", "tools": ["wb_my_task", "wb_comment", "wb_review"]
  },
  "onFeedback": "评审意见被作者修订后 → 复核 confirm/reopen；分歧线 wb_reply 澄清",

  "reviewers": [
    { "role": "Architect", "capabilityId": "arch-tech-review", "state": "待评审" },
    { "role": "RD",        "capabilityId": "rd-tech-review",   "state": "待评审" },
    { "role": "FE",        "capabilityId": "fe-tech-review",   "state": "待评审" },
    { "role": "QA",        "capabilityId": "qa-tech-review",   "state": "待评审" },
    { "role": "PM",        "capabilityId": "pm-review",        "state": "待评审" },
    { "role": "User",      "capabilityId": "final-approve",    "state": "待拍板" }
  ],

  "currentNode": {
    "instanceId": "n-04", "templateNodeId": 4, "key": "RD 技术评审",

    "ownerProducts": [
      {
        "kind": "document",
        "id": "art-rd",
        "label": "《RD 技术设计》",
        "href": "docs/v1.1/req-019-join-rollback/rd/design.md",
        "producedBy": "RD",
        "readMode": "document",
        "comments": [
          {
            "issue_id": "cm-qa-line",
            "anchor": { "kind": "line", "artifactId": "art-rd", "line": 42, "lineText": "接口 /settle/commit 缺幂等键" },
            "author": "QA", "role": "qa", "reviewerRole": "QA",
            "type": "issue", "assignee": "RD",
            "text": "幂等性：/settle/commit 重试是否会重复扣款？建议加 idempotency-key。",
            "state": "open", "roundCount": 2,
            "thread": [
              { "reply_id": "r-1", "by": "RD", "role": "rd", "text": "已补充 idempotency-key 字段，见接口定义 §3.2。", "at": "2026-08-15T10:18:00.000Z" },
              { "reply_id": "r-2", "by": "QA", "role": "qa", "text": "确认，接口结构一致，可关闭。", "at": "2026-08-15T10:20:00.000Z" }
            ],
            "at": "2026-08-15T10:15:00.000Z"
          },
          {
            "issue_id": "cm-fe-block",
            "anchor": { "kind": "block", "artifactId": "art-rd", "blockStart": 22, "blockEnd": 30, "lang": "sql" },
            "author": "FE", "role": "fe", "reviewerRole": "FE",
            "type": "issue", "assignee": "RD",
            "text": "这段 SQL 的联调用例没覆盖（前端造数场景）。",
            "state": "open", "roundCount": 1, "thread": [],
            "at": "2026-08-15T10:16:00.000Z"
          },
          {
            "issue_id": "cm-arch-doc",
            "anchor": { "kind": "document", "artifactId": "art-rd" },
            "author": "Architect", "role": "arch", "reviewerRole": "Architect",
            "type": "issue", "assignee": "RD",
            "text": "技术路线与版本架构一致，但并行 join 状态机需补充回滚说明。",
            "state": "open", "roundCount": 1, "thread": [],
            "at": "2026-08-15T10:17:00.000Z"
          }
        ]
      }
    ],

    "reviewerProducts": [
      {
        "kind": "comment",
        "issue_id": "cm-pm-node",
        "anchor": { "kind": "document" },
        "author": "PM", "role": "pm", "reviewerRole": "PM",
        "type": "discussion",
        "text": "若 RD/FE 并行开发，联调窗口如何排期？（节点级对话，不锚定文档）",
        "state": "open", "roundCount": 0, "thread": [],
        "at": "2026-08-15T10:18:00.000Z"
      }
    ],

    "outputs": []
  },

  "relatedNodes": [
    {
      "node": { "instanceId": "n-01", "templateNodeId": 1, "key": "产品设计" },
      "products": [
        {
          "kind": "document",
          "id": "art-prd",
          "label": "《产品设计文档》",
          "href": "docs/v1.1/req-019-join-rollback/prd/prd.md",
          "producedBy": "Architect",
          "readMode": "document",
          "comments": [
            {
              "issue_id": "cm-prd-note",
              "anchor": { "kind": "document", "artifactId": "art-prd" },
              "author": "PM", "role": "pm", "reviewerRole": "PM", "type": "discussion", "state": "resolved",
              "text": "范围补充确认：报表导出按模块拆分。", "roundCount": 1, "thread": [],
              "at": "2026-08-15T09:10:00.000Z"
            }
          ]
        }
      ]
    },
    {
      "node": { "instanceId": "n-03", "templateNodeId": 3, "key": "RD 技术设计" },
      "products": [
        {
          "kind": "document",
          "id": "art-rd-v1",
          "label": "《RD 技术设计（初版）》",
          "href": "docs/v1.1/req-019-join-rollback/rd/design.md",
          "producedBy": "RD",
          "readMode": "document",
          "comments": [
            {
              "issue_id": "cm-early-qa",
              "anchor": { "kind": "range", "artifactId": "art-rd-v1", "startLine": 10, "endLine": 15 },
              "author": "QA", "role": "qa", "reviewerRole": "QA", "type": "issue", "assignee": "RD", "state": "resolved",
              "text": "接口契约区：联调口径需补充（前置评审遗留意见）。", "roundCount": 3, "thread": [],
              "at": "2026-08-15T09:30:00.000Z"
            }
          ]
        }
      ]
    },
    {
      "node": { "instanceId": "n-09", "templateNodeId": 9, "key": "RD 开发" },
      "products": [
        {
          "kind": "link",
          "id": "art-mr-rd",
          "label": "代码 MR（RD）",
          "href": "https://github.com/huzilin/dsh-plugin/pull/123",
          "producedBy": "RD",
          "readMode": "worktree",
          "featureBranch": "feature/req-019-m7-drill"
        }
      ]
    }
  ]
}
```

## 产物模型要点（对应 rework §2.4/§2.5）

| 位置 | 结构 | 说明 |
|---|---|---|
| `document.comments[]` | 行/块/文档级锚点评论 | **文档的评论挂在文档下**（子结构），`reviewerRole` 标识提评者（各审各的） |
| `currentNode.ownerProducts[]` | 节点 owner 交付的产物 | 被审文档（含其评论）/ 开发代码 MR——按产出者归类 |
| `currentNode.reviewerProducts[]` | 节点级对话（不锚定文档） | 评审成员的顶层 comment 产物 |
| `relatedNodes[].products[]` | document（含评论）/ link / 节点级 comment **平等混排** | 上下节点的顶层产物一视同仁 |

**锚点挂载（评论产物）**：

| anchor.kind | artifactId | 归属 | 示例 |
|---|---|---|---|
| `line` | 有 | → 挂在文档 comments | QA 幂等意见（第 42 行） |
| `block` | 有 | → 挂在文档 comments | FE SQL 代码块意见 |
| `range` | 有 | → 挂在文档 comments | QA 早期范围意见 |
| `document` | 有 | → 挂在文档 comments | Architect 文档级意见 |
| `document`（无） | 无 | → 顶层 comment（reviewerProducts / products） | PM 节点级对话（排期讨论） |

## 补充说明

1. **`relatedNodes` 由 capability `inputs` 推导**：`qa-tech-review` 的 inputs 声明「产品设计 + RD 技术设计 + RD 开发（代码）」→ 实例经 `template_node_id` 找到对应节点 → 取其 `products[]`。
2. **文档评论挂文档下**：行/块/文档级锚点评论是 `document.comments[]` 子结构；顶层平等成员 = document（含评论）/ link / 节点级 comment。
3. **代码产物**（QA 验收场景）：`kind=link` + `readMode: worktree` + `featureBranch`，QA 经 git worktree 检出分支读真实代码（rework §3，不硬切分支）。
4. host.js 实装当前仍返回旧结构（`artifactsToReview + sessionsByNode`），本文为目标态，待按 rework §5 落地。

## action（动作定位，最小返回集 · 2026-08-27 更新）

> **更新说明（2026-08-27）**：①顶层 `behaviors` 字段**并入 `action.how`**（行为列表即 how 的前身，统一术语）；②`actions` 数组改为**最小返回集**——`wb_my_task` 只返回**程序判定的当前动作**（单个 `action` 对象，含完整 how），**不返回其他动作**（旧模型返回当前节点能力全部 actions 数组，已按用户裁定收敛：减少不必要信息、防 rush 由程序返回层完成；引用链路稿 D3 定稿）。`activeActions` 保留为程序判定结（审计/日志用，只含被激活的 action_id）。

```jsonc
// 顶层（基于程序决策树判定的当前动作——链路环节⑤：who + when → currentTrigger → 选中动作）
"currentTrigger": "reviewer_feedback(open)",
"activeActions": ["backend-dev.respond_to_cr"],        // 程序判定结（审计用，仅激活项 id）
"action": {                                             // ← 仅当前动作（最小返回集，其他动作不返回）
  "action_id": "backend-dev.respond_to_cr",
  "name": "响应CR",
  "trigger": "reviewer_feedback(open)",
  "when_to_use": "评论产物中存在 reviewerRole=Code Reviewer 且 assignee=rd 且 state=open 的评论时（人读注释，程序不据此判定）",
  "how": "① 过滤 assignee=rd 且 state=open 的评论（reviewerRole=Code Reviewer，挂在本节点《dev_notes》(docs/{{req-version}}/{{req-name}}/{{side}}/dev.md)/code 上）
         ② 逐条读 text（要改什么）+ anchor（锚哪行/块）定位改动点 ③ 读 thread 理解上下文 → 改代码或 wb_reply 澄清
         ④ 修订后 wb_resolve(note) → CR 复核 confirm/reopen
         完成标准：全部 open CR 已处理（resolve 或澄清），无遗漏",
  "reference": [{ "query": { "reviewerRole": "Code Reviewer", "assignee": "rd", "state": "open" } }],
  "focus": ["dev_notes.实现-设计对照表"], "output": "修订代码/resolve", "tools": ["wb_my_task", "wb_reply", "wb_resolve"]
}
```

- **引用规约**：文档/代码类产物 → `reference.productType`（标准枚举）；**对话/评论类产物 → `reference.query`（按 reviewerRole/assignee/state/anchor 属性过滤）**——两者都不依赖流程内节点位置/名称，跨流程成立；
- **how 模板化渲染（J7 定稿 · 技术可实施）**：`how` 里每个产物直接写「《标准名》(路径模板)」——如 `《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md)`；`wb_my_task` 返回前 host 端调 `renderTpl(s, ctx)` 将 `{{var}}` 替换为当前需求真实值（ctx：`req-version`=版本号、`req-name`=需求 slug、`side`=rd/fe），subagent 拿到即真实路径：
> ```ts
> // host 侧实现（JS/TS，一次定义全 how/ref 复用；变量名可含连字符，故用 [^{}]+ 匹配）
> function renderTpl(s: string, ctx: Record<string, string>): string {
>   return s.replace(/\{\{([^{}]+)\}\}/g, (_m, k) => {
>     const key = k.trim();
>     if (!(key in ctx)) throw new Error(`how 模板变量未提供: {{{key}}}`);
>     return ctx[key];
>   });
> }
> renderTpl('docs/{{req-version}}/{{req-name}}/rd/design.md', { 'req-version': 'v1.1', 'req-name': 'req-019-join-rollback' })
> // -> 'docs/v1.1/req-019-join-rollback/rd/design.md'
> ```
- **subagent 定位**：respond_to_cr 时，RD 从 mytask 返回的评论产物（`document.comments[]`/`reviewerProducts[]`）里用 `query` 过滤出自己的 open CR 意见（带 text + anchor + thread），据此改代码或澄清；

- **调度定位**：Leader 对 RD「用 backend-dev.respond_to_cr 响应 CR」→ RD 调 wb_my_task → 返回**当前动作**（action_id + how/reference/focus/tools）→ 精确执行；程序已判定动作，subagent 零选择；
- **when_to_use/trigger 的定位**（链路推演 2026-08-27 定稿）：`trigger` = 程序契约值（wb_my_task 计算 currentTrigger 用）；`when_to_use` = **人读注释**（评审者/Leader 看激活语义），程序不读它判定；
- action 由 `capabilities[capability_id].actions` 按决策树选中（回填 seed 后）；`trigger` 标注何时触发（node_enter / dev_complete / reviewer_feedback / reject 打回）；
- 与 currentNode.ownerProducts/reviewerProducts 的关系：action 描述「做什么 + 参考哪些可见产物」；products 提供可见产物本身。