# 两项待拍板展开材料——`agreed` 词属裁定 与 arch_design 修订优先级

> 日期：2026-09-12。性质标注：【实测】=全量直读清查证实；【推断】=分析判断；【历史拍板】=台账可查的原有结论。
> 关联：retired-terms-audit.md（A2/D2 条目）、terminology-evolution.md、decision-timeline.md REV 组。

---

## 事项一：`agreed` 正名还是改名

### 原文照抄（你的裁决，2026-09-11）

> 「每个评审位独立 reviewing/agree/unagree，clarifying/confirming/agreed 不再作为存储态」——你认可 v5 定稿时的原话。

据此 CONTEXT.md 黑名单把 **`agreed`（作存储态）** 列为退役词。

### 冲突是什么【实测】

全量直读发现 `agreed` 在现行面仍有十余处在用：spec.md L64/L473/L488、business-flow-derivation 与 flow-derivation-tech-task-plan 十余处、`rd/service/collect.go:46`、**proto 契约 gate_state 枚举**、FE 拍板卡整套组件。

逐一判读后结论【推断，证据在 audit A2/D2】：这些用法**不是**你废除的那个东西。你废除的是 v4 模型里「agreed 作为节点级 reviewState 的一个**存储状态**」；现在这些 `agreed` 的实际身份是——**collect 看到「全 agree」时的瞬时判定输出 + 拍板卡迎宾呈现的展示词汇**（gate_state 承载），不落 reviewState 存储。也就是说：语义上它们已经合规，只是**词面**与黑名单撞车。

### 选项与影响

| 选项 | 内容 | 影响面 | 风险 |
|---|---|---|---|
| **A 正名（推荐）** | 保留全部现行用法；在 spec gate_state 处与 CONTEXT 黑名单加注记：「agreed=collect 瞬时判定输出/门禁展示词汇，非 reviewState 存储态」；黑名单措辞精确化为「agreed 作 reviewState 存储态」 | 零代码，纯文档两处措辞 | 词面仍与黑名单同形，后人需靠注记区分（靠台账与映射表兜底） |
| **B 改名** | 门禁侧换新词（如 `all-agreed`/`consensus`），改 proto gate_state 枚举 + collect.go + FE 拍板卡 + 两份推导文档十余处 + 相关测试 | 一处契约枚举变更（buf breaking 会拦，须同步）；全链测试跑一轮 | 改动为纯词面工程，不产生行为收益；撞 pre-push 契约闸门需同步重生成 api.pb.go |

### 推荐【推断】

**A 正名**。理由：你的废除本意是「不存储」，现用法确实不存储，语义已合规；B 是纯词面工程却要动契约和全链测试，不值。若你更在意「词面零歧义」（与「名与义必须零距离」纪律一致），选 B，我按 B 出返工单。

---

## 事项二：arch_design.md 修订优先级

### 原文照抄（两条历史拍板的张力）

> 「**不支持将能力推演合并到主文档 arch_design.md**（保持现状基线）」（2026-09-02，wayfinder 工单 06）

> 「版本废弃改条件解绑（注记-A）/需求状态编排联动（§7.3）/产物入库约束（注记-K）」——2026-09-10 同步轮，arch_design **数据节**已按新口径更新

### 现状是什么【实测】

全量直读：arch_design.md 现在**半新半旧**——数据模型节已带 09-09/09-10 新注记，但 **§7.3/§8/§9 仍整套是 pkg-16 时代词汇**：`wb_decide`、`resolve(note)`、`overridden`、`pass|block` 表态、19 项旧能力清单、`agreed` 旧用法。我已按你的「标注」授权加了定位横幅（声明其为 v1 基线历史底稿、现行权威=docs-tmp 五件套）。

### 为什么需要你定

「保持现状基线」的既有拍板 vs「过时信息要标注/清理」的新纪律，在 arch_design 身上打架：它既不能大改（09-02 拍板），也不能不标（新纪律已标）。剩下的真问题是——**§7-§9 的旧词汇节，什么时候、以什么力度刷新**。

### 选项与影响

| 选项 | 内容 | 影响面 | 适用条件 |
|---|---|---|---|
| **A 维持横幅（现状）** | 基线留档 + 横幅声明，刷新不排期；任何人要引用 §7-9 先查 docs-tmp | 零成本 | 近期没有要引用 arch_design §7-9 的场景 |
| **B 定向刷新 §7-§9** | 按 v5 口径重写三个节（wb_decide→user_gate decision 路由、pass/block→agree/unagree、overridden 段删除、19 旧能力→现行 18 能力+动作挂载表） | 约 1 个 session；须同步核对 01-module-board 三板引用 | 有文档/工单要正式引用 arch_design（如对外交付、模板基线引用） |
| **C 整体迁 .archive** | 文件移归档，docs/ 留指针 | 需改 roadmap/三板对它的引用；与「保持基线」拍板冲突，不推荐 | 你认为基线本身已无留档价值 |

### 推荐【推断】

**维持 A，触发条件明确**：一旦出现「需要引用 arch_design 权威性」的场景（对外交付/模板链引用），立刻升级为 B 做定向刷新。C 不推荐——与 09-02 拍板冲突，且三板 PRD 的历史依据仍指着它。

---

## 你需要回复的

1. 事项一：A 正名 / B 改名？
2. 事项二：A 维持横幅 / B 定向刷新（若 B，我排 1 个 session 的工单）？
