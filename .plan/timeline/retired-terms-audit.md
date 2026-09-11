# 退役技术名词现行引用清查（四档台账）· 2026-09-11

> 候选清单来源三处合并：①`.plan/proposals/state-tables-draft.md` 词汇单点定义黑名单；②`docs-tmp/capability-model-final.md` §8 作废留痕表（实为 20 行，以文件为准）；③`.plan/timeline/decision-timeline.md` superseded-by 条目词汇 + `.archive/README.md` 旧概念。
> 判读基准锚点：`CONTEXT.md`（词汇单点定义 + _Avoid_ 表，2026-09-11 版）为现行词表正本；`state-tables-draft.md` 2026-09-11 三轮批注（ISS-08/OVR-06 裁决）为 issue 状态机权威。
> 四档：A=现行权威文档/代码当现行概念用（需改）｜B=合法历史语境（留痕/对照/演进叙述，保留）｜C=零引用｜D=代码/DDL/seed/测试在用（返工项）。

---

## 0. 方法说明（按用户指令口径）

- **本轮按用户指令全量直读、不计成本**：docs-tmp 七件、`.plan/wbflow-impl`（spec/tickets）、`.plan/capability-chain-finalize/map.md`、`.plan/handoffs`×4、workbench-sql 全部（schema/seed.sql/seed.json/ER/README/demo-data/init.sh）、rd 迁移 SQL×4、api.proto、rd 核心 .go（domain 全部、service 核心 review/comment/escalation/advance/reject、repo model/issue/payload、handler 路由与 comment/escalation、e2e 断言区）、fe/src 核心（flowview/ArtifactReader/DecisionCard/CommentCard/Badges/constants/comments api）、qa_drive.py 关键断言区、docs/v1.0/arch_design.md 全文、02-module-flow-tree prd+rd/design、qa 四件套命中区、README/CONTEXT/Makefile/scripts/githooks——逐行判读「该行在用的概念是否已退役」。
- **grep 降级为终验复核**（21 个模式跑全现行面）：与直读结论对账，差异两条已并入正文（ui-prototype-v3.html 的 clarifying；0002_seed 的 templatePath）。该口径仅本轮适用；常态轮可回归 grep+抽样，但方案敲定后的词汇表核对（CONTEXT.md _Avoid_ 对照）必做。
- **疑义声明（未逐行直读、按扫描+命中区精读处理，非静默跳过）**：①`fe/src/api/proto/api.ts`（13777 行，protoc 生成物，无 prose 语境，已按 21 模式扫描，零命中退役词）；②rd/fe 的 `*_test.go`/`*.test.tsx` 全量（扫描+命中区精读，未逐行）；③`qa/qa-results.json`、`docs/docs/v1.0/qa/fixtures/`（与 `docs/v1.0/qa/fixtures/` 重复的镜像目录）、`docs/v1.0/_archive/`（9 份，归档区按 B 档整体豁免）、req-002/003/004 三需求文档、01/03 模块 prd/design（扫描零命中）；④`docs/v1.0/diagrams/*.json|html`。

---

## 1. 总表（词 | 候选来源 | 四档命中 | 代表位置 | 建议动作）

| 词 | 来源 | A | B | C | D | 代表位置（现行面） | 建议动作 |
|---|---|---|---|---|---|---|---|
| `overridden`（作状态） | 黑名单+§8 | 6 | 多 | — | 12 文件 | 见 §3 D1 全图 | D1 返工（9-11 裁决：override=动作→resolved） |
| `agreed`（作状态） | 黑名单（CONTEXT reviewState _Avoid_） | 7 | 4 | — | 4 文件 | spec L64/L473/L488；collect.go:46；flowview.ts:106 | 需拍板：改名 gate_pending/pending-decision 或在 CONTEXT 正名（见 §3） |
| `resolve(note/fix/agree)` 事件名 | 黑名单 | — | — | — | 代码内为现行 API | CONTEXT 已裁「resolve(path) 双路径命名=_Avoid_」但实现/契约仍是 path=fix\|agree | 随 D5 事件命名提案一并定（先改表后改码） |
| `clarifying`/`confirming`/`unagreed` | 黑名单 | 1 | 多 | — | 0 | ui-prototype-v3.html:192/627（相位角标 clarifying） | 原型 HTML 改 reviewing/agree/unagree |
| `agreed`（v4 节点级链） | REV-05 | — | ✓ | — | 0 | — | 无 |
| `pass`/`block`（作表态值） | 黑名单+REV-04 | 2 | 1 | — | 0 | arch_design §8.6/§9.4；e2e README 措辞 | arch_design 随基线标注处理；代码零残留 |
| `pass`/`block`（作裁决值） | data-tools L121「语义保留 service 层」 | — | — | — | 0 | 实现用 approved/rejected（collect.go:38-41） | 无（旧词已被新词取代干净） |
| `trigger`（词表/字段） | §8+ORCH | — | ✓ | — | 0 | 代码仅 e2e 曾有、已改「唤起」（closeout 记录全库零残留）；残余全在 B 语境 | 无 |
| hook / after-exec hook | §8 | 1 | ✓ | — | 0 | handoff-4 裁决项 G「走 exit hook」（历史提案文本） | B 保留（结论以 spec -F 为准） |
| re-entry / re-exec（相位） | §8 | 1 | ✓ | — | 0 | state-tables-draft 自己的行文（re-exec 侧判定）+ CONTEXT _Avoid_ 自指 | 黑名单文件自指，免改 |
| `retest` | §8 作废行 + actions L70「已移除」 | 2（留痕侧） | — | — | 现行（seed/task.go/通知子型） | capability-model §8 行 vs seed 0002 L131、CONTEXT「重入=retest」 | **留痕侧更新**：§8 行与「已移除」清单补「09-09 复活」注记，否则文档自相矛盾 |
| how 四要素 | §8+SK | — | ✓ | — | 0 | 全部为降级/退役声明语境；spec -L4「how=prompt 素材」为新拍板用词 | 无（建议 -L4 补一句与「how 退出框架」留痕的关系） |
| collect/reviewed/replied/complete/ready 作相位 | §8 | 2 | ✓ | — | 0 | capability-actions L30/L71「replied 事件触发」；data-tools L130 同 | A：改「respond 相位复核环节触发」，删 replied 事件语 |
| respond_to_cr / respond_to_qa | §8 | — | ✓ | — | 0 | 全部退役声明 | 无 |
| retest 独立动作（§8 语义） | §8 | 见 retest 行 | | | | | |
| 取消独立评审节点 / pending-done-gate 占位 / pm-review@评审 / decide-cap.decide / final-approve / R1 双动作 / 内嵌 CR 两轮摇摆 | §8 | — | ✓ | — | 0 | 全部在 §8/退役声明/B 语境 | 无 |
| plan-cap / alignment-cap / plan-review-arch / alignment-review-pm | §8+actions | 1 | ✓ | — | 0 | **flow-derivation L369 D6 yaml `pm.alignment-review`**（退役能力名残留）vs L375 已用 pm.architecture-review | A：L369 改 pm.architecture-review |
| case-design-cap / case_design | §8+actions「已移除」 | — | ✓ | — | 0 | 全部为「移除/无 case_design」声明与旧 seed（workbench-sql=B 参考区） | 无 |
| review-protocol 公共 skill | actions L34 | — | ✓ | — | 0 | 「否决全局 review-protocol」声明 | 无 |
| align.closing / closing / transform / exit | §8+actions | — | ✓ | — | 0 | 「已移除」声明+CONTEXT _Avoid_ | 无 |
| 用例设计（独立节点/capability） | §8+timeline QA-01/02 | — | ✓ | — | 0 | qa-acceptance-template 的「用例设计」=内含于 accept 的新口径（合法） | 无 |
| `processed` | how-spec §4 + CONTEXT _Avoid_ | — | ✓ | — | 0 | how-spec-v2 已整体降级（B）；实现用 pending-confirm | 无 |
| line / block 锚（-L3 废弃） | CONTEXT+spec -L3 | 4 | ✓ | — | 契约内存量兼容 | 见 §3 锚点专项 | A：注释/文案/用例文本 4 处；行为已合规 |
| `be`/`be-only`/`fe-only`（scope 代号） | 09-07 术语统一 be→rd | — | ✓ | — | 0（现行面） | 仅 workbench-sql 旧 seed（B 参考区）+arch_design §8.3 | arch_design 随基线标注处理 |
| template_path / templatePath | handoff-3 D4 拍板删除 | — | — | — | 2 文件 | rd/migration/sql/0001 L17 注释、0002 L37/42/47/52/57（5 能力行仍带值）；payload.go 已删字段、workbench-sql 已删 | D：0002 seed 删 templatePath 键值、0001 注释对齐 |
| `flow_templates`/`flow_template_nodes` 旧表名 | data-tools 更名 | — | ✓ | — | 0（现行 rd 为 flow_defs/flow_def_nodes） | 仅 workbench-sql/arch_design 旧 DDL（B/基线区） | 无 |
| barrier / collecting / rework-return 作现行为 | REV/COL/CONTEXT | — | ✓ | — | 0 | rework-return=预留词表有注释声明（reentry.go:30） | 无 |

---

## 2. A 类逐条清单（现行权威文档误用残留，供主线修）

1. **`retest` 作废留痕未对账（最重——文档三处自相矛盾）**
   - `docs-tmp/capability-model-final.md` §8 行「retest 独立动作｜复验=accept 重跑组成」与 `docs-tmp/capability-actions-final.md` L70「已移除：…retest…」宣告退役；但同批文档 L66（actions §2 acceptance-test 行）、L157（model §5.1）、business-flow L225/L229、flow-derivation L148、ui-interaction §3.1/§3.2/N1/N2/N3、spec -D1/-L 系、seed 0002 L131、domain/reentry.go、task.go、CONTEXT.md（「QA exec 重入=retest」）全部当现行动作。
   - 判读：09-09 起复活的现行词；**改留痕不改用词**——§8 行与「已移除」清单补复活注记（引用 2026-09-09 N1 定稿）。
2. **`agreed` 作门禁侧状态（黑名单「作存储态」边界争议，需拍板）**
   - `spec.md` L64（user_gate DDL 注释「1=agreed 后唤醒 User」）、L473（AC-V-03）、L488（AC-M-02）；`business-flow-derivation` L94/L133/L189/L227（「本节点维持 agreed」「agreed→approved」）；`flow-derivation` L84/L133/L237/L254/L379 同；`ui-interaction` L95（「门禁 agreed + user_gate」）；`tickets.md` L104。
   - 语义已非 v4 reviewState 存储态（v5 下 collect 瞬时判定不入库），实现里 gate_state=agreed 是**响应字段/展示态**（collect.go GateStateAgreed）。但「agreed 作存储态」在 CONTEXT _Avoid_ 与 state-tables 黑名单中仍列死。两条出路：①改名（如 gate_pending 同义化）+全量改 spec/derivation 文档；②CONTEXT 正名「agreed 仅作拍板卡展示态、不存储」。**属词汇制度裁决，不由本轮擅自定**。
3. **capability-model §8 L197「reviewState 7 值」**：取代列指向 v5 前旧口径（现行=per-agent 三态+'-'，REV-06）。改「相位 4 值 + reviewState per-agent 三态」。
4. **flow-derivation L369 plan D6 yaml** `reviewers: [{pm: pm.alignment-review}]`——已退役能力名；L375 正文已用 pm.architecture-review。改 yaml 行。
5. **capability-actions L30/L71「replied 事件触发」**：replied 为 §8 退役 hook/trigger 词，现行模型无此事件。改「respond 相位复核环节触发」。
6. **data-tools L74 issues.state 枚举含 `overridden`**：9-07 状态机 v2 权威（模型 §1.5+ISS-08）无此态；实现漂移词写进了设计文档。随 D1 改（设计侧先改）。
7. **capability-model L157 + business-flow L225/L229 + actions L66 + flow-derivation L148** 的 retest 现行用法本身合法（见第 1 条，改的是留痕侧）。
8. **ui-prototype-v3.html L192/L627** 相位角标用 `clarifying`（v4 退役词）。原型=丢弃式视觉参照（-D6 先例），可改可不改，建议顺手改 reviewing。
9. **arch_design.md 基线旧词密集区（73 处命中）**：§3.3 #4（overridden 状态机）/#5（四级锚点）、§7.3（wb_decide pass/resolve(note)/overridden 强权/未开始-进行中-待评审-已通过节点态）、§8.4（19 能力含 final-approve/decide-cap/plan-cap/alignment-cap/case-design-cap）、§8.6（wb_review pass/block）、§9.1（22 工具含 wb_override/wb_decide）、§9.4（wb_decide pass）、§12。该文档已部分随新拍板更新（-A/-K/-J5/J6/§10.3 重写）呈「半新半旧」。建议动作二选一：①对被 docs-tmp v3.1 取代的章节加「已被取代，见 capability-model-final §x」标注（去变更日志负担，正文留基线）；②整段迁 archive。属文档定位裁决，提请主线拍板。
10. **02-module-flow-tree prd/prd.md L89/L127/L132/L163/L194/L199/L249 + rd/design.md L24/25/73/101/119/129/137-141/162-163/196**：四级锚点、wb.override、overridden、resolve(note) 作现行 AC/设计描述。同 arch_design 处理（PRD 基线被 ui-interaction 增补层覆盖处加标注；但 L127「评审通过=全部 resolved/overridden」已被 -L1 收口，L132 强行通过 AC 与 9-11 override 裁决冲突，**建议逐条标注取代**而非整段归档）。
11. **qa-test-cases.md A-18 行**「anchor 四级（document/line/range/block）」——驱动 qa_drive A-18b/d 已断言 line/block=400（-L3），用例文本未同步。改「anchor 两级（range/document）+line/block 400 拒绝」。
12. **rd/service/comment.go L150** 错误文案「或使用 line/range/block 级锚点」——引导使用已废弃锚点，与 ValidateIssueAnchor 自相矛盾。改「或携带 anchorArtifactId（document 级）」。

## 3. D 类逐条清单（代码/DDL/seed/测试残留 = 返工项）

**D1：`overridden` 全部落点（9-11 裁决=偏差 D1，逐文件列全）**
- 契约：`docs/api/v1/api.proto` L191/L408/L429/L824/L829/L1267/L1273；`rd/api/api.pb.go`（生成物随 proto）。
- DDL/seed：`rd/migration/sql/0001_schema.sql` L183（state 注释）；`.plan/wbflow-impl/spec.md` L191（同源正本）；workbench-sql schema L273/ER L138/README（B 参考区顺带）。
- repo：`rd/repo/issue.go` L24/L34（常量+未闭合集）、`rd/repo/model.go` L295（注释）。
- domain：`rd/domain/issues.go` L16（wb_reply 终态拒绝）、L287（IssueClosed）、L334（ValidIssueState）、L218-233（issues_test 同源断言）。
- service：`rd/service/comment.go` L399-452（OverrideRole 批量置 overridden）；`rd/service/escalation.go` L104/L235（终态拦截）、L348-391（OverrideIssueTx：L357/L369/L382）、`rd/service/review.go` L124/L137/L372（agree/approve 前置文案）；`rd/service/flow.go` L213/L243、`rd/service/scanner.go` L101（注释）；`rd/handler/comment.go` L28/L40/L133-141（路由 /override-role）。
- 测试/e2e/qa：`escalation_it_test.go` L104/195-240、`audit_it_test.go` L283+、`issues_test.go` L23/218-233、`qa/qa_drive.py` L650-660（L1b-3）、L1112（C-6e）、`qa-test-cases.md` A-24/C-6、`qa/qa_diag_min` 相关组。
- FE：`api/constants.ts` L70（overridden:「已裁决」）、`components/Badges.tsx` L80、`CommentCard.tsx` L51、`DecisionCard.tsx` L44/L96、`flowview.ts` L122、`ConflictCard.test`/`module3.test` 断言。
- 返工口径（ISS-08/OVR-06）：override=动作→置 resolved；涉及枚举收敛（迁移不 DROP，新增值不再写入）+ 通知/文案 + 双端断言。

**D2：`agreed` 运行时落点（与 A-2 同题，代码侧清单）**
- `rd/domain/collect.go` L45-48（GateStateAgreed 常量）、`rd/service/review.go` L59/L190-191/L234/L245-247（gate_state=agreed 响应+gate_pending 通知触发）、proto L1114（gate_state 契约）、e2e `script_test.go` L216-294、`full_test.go` L85/L168、IT `review_it_test.go` L109-168/L341、`escalation_it_test.go` L221-232、`inbox_it_test.go` L2、qa_drive L617/L723、FE `flowview.ts` L104-118 deriveGateView、`NodeDetail.tsx` L142/L212、`FlowTree.tsx` L242、`DecisionCard` 全套、module2/module3.test。

**D3：line/block 锚存量兼容落点（-L3 已废弃、只读兼容，行为已合规，列全供核对）**
- rd：`domain/issues.go` L261-264/L279（400 拒绝+废弃注记）、L419-435（SummarizeIssueAnchor 存量展示）；proto L197-275（契约字段保留）；DDL 0001 L169-176+spec L177-184（列与注释保留）。
- fe：`api/comments.ts` L99-117（类型仍含 line/block）、`domain/flowview.ts` L259-265（存量摘要展示）、`ArtifactReader.tsx` L19-24（**头注释仍描述块选按钮/line 展示规则**——行为已只 range/document，注释漂移，A 类顺手改）。
- qa：qa_drive L1059-1068（A-18b/d 断言 400，已合规）。

**D4：templatePath 双源残留（handoff-3 D4「seed 双源删 template_path」未覆盖 rd seed）**
- `rd/migration/sql/0002_seed.sql` L37/L42/L47/L52/L57（product-design/backend-design/frontend-design/research-cap/arch-cap 五行 products 仍带 templatePath 键值）；`rd/migration/sql/0001_schema.sql` L17（products 注释仍写 templatePath）。payload.go ProductDef 已无此字段（未知键被静默忽略，暂无运行时危害，但双源漂移违反「JSON 键以 payload.go 为单一口径」）。

**D5：workbench-sql/ 旧 sqlite 参考区（已声明「不迁移仅参考」，非返工但建议处置）**
- schema.sql L173（节点态 未开始/进行中/待评审/已通过）、L273（overridden）、L70（capabilities.how 列）、seed.json/seed.sql（19 旧能力全量：final-approve/decide-cap/plan-cap/alignment-cap/case-design-cap/*-tech-review、pass/reject how、scope be/fe-only）、demo-data.md（block 锚示例、overridden 注释）。建议：目录头加「仅历史参考，现行 seed=rd/migration/sql」横幅或整体迁 .archive（该目录曾于 09-09 被反向更新过 template_path/评审 how，归属信号混乱，宜一次性定性）。

**D6：杂项**
- `qa/qa_diag_min.py` L222：wb_resolve 传 `path:"clarify"`（非法值，响应弃置）——旧接口残留探测行，删或改 fix。
- `model.go` L453-455 Outbox 注释「retry 耗尽置 dead 终态」「上限 5」与 0004/CONTEXT「无限补投无 dead」矛盾（注释漂移，非退役词，顺带登记）。

## 4. 直读新发现（grep 黑名单拿不到的部分）

1. retest 复活未对账（A-1，三文档自相矛盾——本条只有逐行对读 §8 vs §5.1/seed 才能暴露）。
2. agreed 词的「黑名单 vs 实现/文档大规模使用」制度级冲突（A-2/D2）——grep 只给数量，直读才定性「展示态非存储态」。
3. capability-model §8 取代列「reviewState 7 值」旧口径（A-3）。
4. flow-derivation D6 yaml 退役能力名 pm.alignment-review（A-4）。
5. capability-actions「replied 事件触发」残留（A-5）。
6. comment.go 错误文案引导废弃锚点（A-12）；ArtifactReader 头注释块选按钮漂移（D3-fe）。
7. 0002_seed templatePath 双源漂移（D4）——handoff-3 宣称「seed 双源已删」与实物不符。
8. model.go Outbox dead/上限 5 注释与 0004 无限补投拍板矛盾（D6）。
9. arch_design「半新半旧」状态：数据模型节已带 -A/-K/-J 注记而 §3.3/§7.3/§8/§9 整体仍是 pkg-16 时代词汇——文档定位需一次性裁决，否则每轮增补都会继续分层沉积。
10. qa_diag_min 的 path=clarify 死调用（D6）。

## 5. C 类（零引用确认）

`collecting`、`barrier`（现行面零引用，仅 timeline/CONTEXT _Avoid_ 自指）；`respond_to_qa`、`respond_to_cr`、`align.closing`、`case_design`（现行实现/DDL/FE 全零，仅声明语境）；`pass`/`block` 作表态值与裁决值（rd/fe 代码零残留，统一 approved/rejected）；`processed`（实现零引用）。废弃成立，保持归档即可。
