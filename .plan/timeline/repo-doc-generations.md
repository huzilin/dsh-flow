# 仓库历史文档多代沉积分析（2026-09-11）

> 本文 = 决策时间线工程的仓库侧盘点。目的：厘清「哪些件是权威、哪些件是多代沉积、哪些件已归档」，配合 decision-timeline.md（会话侧综合，待子代理产出后合成）彻底解决「重复向用户确认已有结论」的问题。

## 1. 文档分层总览（当前状态与权威指针）

| 层 | 位置 | 生成期 | 当前状态 |
|---|---|---|---|
| 设计权威五件套 | `docs-tmp/`：capability-model-final / business-flow-derivation / flow-derivation-tech-task-plan / capability-actions-final / data-tools-final + ui-interaction-final + diagrams/ | 09-02 起，09-08 v3.1 定稿 | **现行权威**。内部多代沉积：capability-model 经 v1→v2.1(trigger 8 词)→v3(两层属性)→v3.1(§1.5 小状态机)→v5 per-agent(§1.2)；09-11 已清 18 处 v4 词残留（71476da）。§8 作废留痕表 15 项=已废结论正典 |
| 技术规格 | `.plan/wbflow-impl/spec.md` | 09-06 起实现期 | 实现翻译层 + 注记 -A~-L4 逐轮累积（09-09~09-10 四轮）。**注记已事实性成为第二真相源**（风险，见 §4） |
| 基线文档 | `docs/v1.0/`（01-module-board 三板 + roadmap + arch_design） | 09-05 剪裁 | 基线（09-10 同步轮更新过） |
| QA 资产 | `qa/` + `docs/v1.0/qa/` + `docs/v1.0/req-002~004/` | 09-09~09-10 | 现行 |
| 历史归档（仓内） | `.archive/`：workbench-plugin（**一版 client.js 829 行**，与 tmp_1/test-flow 同源已 diff 证实）+ docs-tmp-history（10 份历史推演文档，均有「结论时效」横幅 a6521c9）+ tmp/ | 09-08 冻结 | 只读归档，README 有「勿据以实现」指针 |
| 历史正本（仓外） | `/Users/huzilin/tmp_1/test-flow/`（32M） | ~09-02~09-06 | **第一版代码**：含一版交互设计（用户多轮要求参照、履约有缺口——盘点见 testflow-interaction-inventory.md） |
| 会话沉积 | dsh 侧 101 sessions（~09-02~09-07 设计推演主期）+ zcode 侧 23 主 sessions（09-05~09-11 实现期） | — | **此前无台账**——本次补（zcode/dsh-sessions-decisions.md → decision-timeline.md） |
| 决策账本 | semantica | 09-05 起 | 部分关键决策有录，覆盖不全 |
| 规划层 | `.plan/`：capability-chain-finalize map（已关图+时效 banner）/ wbflow-impl / handoffs ×4 / standard-authority-refactor / proposals ×3 | 09-02~09-11 | 现行 |

## 2. 多代沉积热点（同一主题的历代结论所在处）

1. **issue 状态机**：9-07 重设计（澄清/修复两阶段+loop_count）∈ 模型 §1.5 + OpenViking 事件记忆；实现（issues.go v2）另加了 overridden 态与多条非设计通路（09-10 修复轮混入）→ 09-11 状态表批注逐格裁决（五态+override 两格+自循环非法）。**三代结论，最新=09-11 批注链（semantica 513b6a48→920030de→35e0af56）**。
2. **reviewState 词汇**：collecting/clarifying/confirming/agreed(v2/v3)→v4 节点级 7 值(96834d1)→v5 per-agent 三态(§1.2, 09-08)→09-11 User 确认 v5 定稿。历代沉积集中在 OpenViking「节点生命周期 hook 模型」记忆（已更新锚点）与五件套轮转行（已清）。
3. **respond 挂载**：R1 双动作→「一个 state 一个动作」普适约束（647a493，R1 undermined）→CR 异化单挂（41f7c06）→4b602aa 曾回退内嵌 CR 又再定。最终口径见 actions-final 轨道规则。
4. **scope 体系**：task 三值化(e825d76)→回滚(5c25144)→表达式制(3500f77)。最终=渲染集合表达式。
5. **评审节点去留**：「取消评审节点」提案撤回→双轨制定稿（节点内 CR 异化+跨节点评审节点）。
6. **交互层**：ui-interaction-final 增补稿+原型 7 面板（569c28a→963c751→e31e0c4→eb51681→f873d2a 五轮）；**一版 client.js 交互（上下分栏/SVG 泳道/划词锚点/药丸）是用户指定的参照基线**，当前 fe 履约缺口见盘点文件。

## 3. 已有的过时标注机制（不重复建设）

- a6521c9「结论时效」横幅：docs-tmp-history 10 份+工单 01-05（只标废不删原则）。
- `.archive/README.md`：归档指针+勿据以实现。
- capability-model §8 作废留痕表（undermined 正典）。
- map.md 顶部时效 banner。
- **缺口**：①会话层无决策台账（本次补 decision-timeline.md）；②spec 注记与五件套双源无互指；③一版交互参照无履约对照（本次补 inventory）。

## 4. 「重复确认已有结论」根因清单（本次复盘结论）

1. 会话决策无时间线台账，agent 每轮只能靠 OpenViking 记忆检索（被动、覆盖率不全）→ 同一问题（如 issue 状态机、override）多代结论并存时无法机械判定「最新拍板」。
2. 五件套内部多代词汇残留（v5 漏 sweep 18 处即例证），「读到哪里算哪版」不可判定。
3. spec 注记累积成为第二真相源，与五件套无互指、无权威仲裁声明。
4. 一版代码参照要求散落在会话里，无履约对照清单 → 多轮要求未被兑现。
5. OpenViking 记忆为多代沉积体（节点生命周期记忆含 v6 以来全部世代），锚点行此前停在 v4。

**处置**：decision-timeline.md（合成正本，最新拍板=现行权威，旧条目逐条标 superseded-by）+ 本文 §1 分层表 + 三份检索文件，构成「先查台账再开口」的基础。
