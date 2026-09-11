# 团队项目工作台（Team Project Workbench）

DSH 之上「项目管理 + OA 流程 + 评审协作」三合一系统；本文件是**领域词汇单点定义**（现行定义 + `_Avoid_` 退役词）。权威设计 = `docs/v1.0/arch_design.md`（单一权威，2026-09-12 MIG-07 合并）；实现规格见 `.plan/wbflow-impl/spec.md`；术语的**历史沿革与改名链**见 `.plan/timeline/terminology-evolution.md`（本文件不留变更日志）。纪律：每轮方案敲定随台账回写更新本表（alignment-protocol 机制 2「方案敲定三动作」）。

## Language

**工单（ticket）**:
wayfinder 地图上待决策/待研究的问题单元。**≠ Issue（评审待解决问题）**。
_Avoid_: 票、卡片、事项

**Issue 状态机（五态 + 单计数器）**:
`open 待澄清 → pending-confirm 已澄清待复核 →（三出口：resolved / to-fix / open）`；修复子机 `to-fix → fixed →（两出口：resolved / to-fix）`。`loop_count` 单计数器：每次回到前一状态 +1，澄清期读=reopen 次数、修复期读=revise 次数（≥5 自动上升）。权威：capability-model-final §1.5 + 09-11 批注定稿（decision-timeline ISS-08）。
_Avoid_: overridden（作为状态——override 是动作、结果置 resolved）；resolve(note/fix/agree) 旧事件名；processed

**Issue 复核事件（复核位拆分）**:
澄清侧复核位三事件 `settle（协商一致→resolved）/ qualify（定性真问题→to-fix，触发打回）/ reopen（未解释清楚→open）`；修复侧复核位两事件 `confirm（修复成功→resolved）/ revise（修复失败→to-fix）`。owner 申报事件 `clarified`（澄清完成）/ `declare-fixed`（修复完成）。agent 只在被调用时响应——待复核态无自主追加，补充信息走复核者 reopen。
_Avoid_: confirm 作澄清侧事件；resolve(path) 双路径命名；reply 幂等自循环

**override 与上升（escalate）**:
冲突上升到 User 后的仲裁动作，只开两格：pending-confirm→resolved（澄清期上升，issue 先回待复核等仲裁）、fixed→resolved（修复期反复修复不成功上升，判定主体=产出节点 re-exec 按 revise 计数）。open/to-fix 不接受 override。
_Avoid_: overridden 状态；并行仲裁（发回 open 同时裁）

**相位（phase，node.state）**:
可配动作相位：entry（进入/重入迎宾）/ exec（首次主产出、重入返工）/ review（评审表态）/ respond（owner 响应）；`done` = 离场业务终态（激活时前置相位统一置 done，禁挂载，打回复位）。动作以 `on` 挂相位。
_Avoid_: trigger 词表、hook（编排层退役，仅通知扩展位）、collect/reviewed/replied/rework 作相位、re-entry/re-exec（并入 entry/exec，计数位 rolled_back_count 承载）、closing/transform/exit

**reviewState（评审位表态，per-agent 三态）**:
每个评审位独立轮转 `reviewing（评审+表态中）→ agree / unagree（已表态）`，数据在 review_results[].state；全表态由 collect 瞬时判定（不存储），有 unagree → owner 澄清 → reviewer 回 reviewing 重表态；真问题（qualify）→ rejected 打回。门禁结果由 `last_decision` 承载（approved/rejected）。
_Avoid_: collecting、clarifying、confirming、unagreed、pass/block（作表态值）、barrier

**agreed（collect 层专用词）**:
全评审位结论达成一致时 collect 的判定输出——「达成一致」的专用词汇；全部进入 agreed → 进入待确认（拍板等待，gate_pending）。仅限 collect 层使用：不作 reviewState 存储态、不扩大到 collect 之外（09-12 裁决）。
_Avoid_: 作 reviewState 存储态；在 collect 之外（接口/文档/展示）扩大使用

**能力（Capability）与动作（Action）**:
Capability = role 归属 + inputs 数据可见性的结构壳；Action = 可独立调度单元（`<capabilityId>.<action>`），以 `on` 挂相位、挂 skill 或 inline；节点以激活清单（node_actions）声明。**一个 state 只挂一个动作**（普适约束）。
_Avoid_: how 四要素（框架强制）、respond_to_cr/respond_to_qa、步骤/子任务

**动作名（域分立轴）**:
exec 重入按域分立：文档域 `revise`、代码域 `diagnosing-bugs`；respond 按轨道分立：跨节点评审 `clarify`、节点内 CR `fix_on_review`（唯一存活处）；QA 验收 exec=`accept`、重入=`retest`（复验重测）。执行入口 exec 重入看域、respond 看轨道。
_Avoid_: respond_to_cr、rework-return 作现行为（仅预留值）、qa 产用例（旧职责）

**技能（skill）**:
DSH 技能；≠ Capability。how 降级为 skill 写作参考（运行时不引用）。
_Avoid_: 能力（领域词）、插件、behaviors（旧名）

**scope（渲染集合表达式）**:
需求范围 → 节点渲染集合表达式：RD 侧=`rd|full`、FE 侧=`fe|full`、QA 验收=`full`、中性=`''`（any）、task C4=`rd|fe`；flow scope_options：business/tech={full,rd,fe}、task={rd,fe}、plan 无。表达式 ∩ requirement scope 判渲染。
_Avoid_: 单值代号标注（rd/fe 直标）、be（→rd）

**产物（ProductType）**:
入库文档 brd / prd / rd_design / fe_design / test_result / defect / research / arch_design / alignment / dev_notes / plan 等；期望产物必须 kind=document 且 href 相对 docs root 真实存在（注记-K，link 仅期望外补充外链）。
_Avoid_: 外链占位、template_path（已删）、spec（旧概念）

**评论锚（anchor）**:
两级：`range`（划词）/ `document`（文档级）；评审上下文 issue 类评论必须携带 anchorArtifactId。
_Avoid_: line / block（-L3 废弃，存量只读兼容）

**通知可靠性（outbox）**:
notify_outbox 消息事务表：业务事务内 enqueue + 提交后 FlushPending + scanner SweepOutbox 补投；**无 dead 终态、无限重试直至送达**。
_Avoid_: dead 终态、retry 耗尽放弃、Dispatch 吞错口径

**工单地图（wayfinder）与技能分层（direct/dependency/scaffold/discarded）**:
沿用既有定义；to-* 产出 skill 家族（to-brd/to-prd/to-rd-techdesign/to-fe-techdesign）与 implement-techdesign 编排协议（原 implement-spec）不变。

**qa 职责（联调/验收/缺陷）**:
联调 + 验收 + 缺陷判定（验收节点仅 full 渲染）；缺陷响应链=rd/fe clarify 确认归属 + diagnosing-bugs 修复；qa 评审以对象化评审 skill 减少上下文。
_Avoid_: qa 产用例（旧职责）、respond_to_qa（退役动作名）

**subagent 会话（挂载到 flow 节点）**:
每个 subagent 有独立稳定 sessionId，持久化跨重启；UI 经插件页面从 flow 节点查看历史/进行中的 subagent 对话（不经 Leader 中转）。
_Avoid_: 弹窗问答（ask_user_question 是另一通道）
