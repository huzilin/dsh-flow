# Productivity 桶盘点（7 个技能）

> 来源：`~/.dsh/skills/mp-{grill-me,grilling,handoff,teach,to-questionnaire,wait-what,writing-for-agents}/SKILL.md` 全文逐一读取（一手源）。
> 判定标准（Q4=A）：按当前阶段「能否改进流程/能力/产物/动作推演工作」为第一标准；「当前推演期采纳 / 插件实现期激活 / 不适用」三档。
> how 写法实例依据：`docs-tmp/capability-review-v2.md`（19 能力 how，尤其 §0.2 backend-dev 动作 JSON）、`docs-tmp/execution-loop.md`（触发→派发→执行→返回闭环、how 获取链路 §9）、`docs-tmp/my-task-response-model.md`（wb_my_task 返回模型含 actions[].how）、`docs-tmp/capability-artifact-analysis.md`（how 推演方法 §2.5、登记闭环 §4.3、自审表 §4.2）。
> 关联票：`.wayfinder/tickets/02-productivity-bucket-inventory.md`（本盘点）、`04-how-writing-comparison.md`（writing-for-agents × 五段式全量对照，另票）。

---

## grill-me（✅ 保留，需求对齐阶段用户可选调用 · 2026-09-02 裁定）

**功能**：一对一的「不松口」访谈——沿计划决策树逐分支下行，一次只问一个问题，每题先给推荐答案；事实自查、决策归用户、达共识前不动手。

**触发方式**：user-invoked。

**初步复用判定**：✅ **保留**（A 组裁定）。需求对齐阶段 subagent 主动开场后，用户可自行决定：一句话阐述上下文，或调用 grill-me 单题盘问（一问一答轻量节奏）。与 grilling 同构，仅节奏不同；作为「轻量单题档」随需求对齐交互使用。

---

---

## grilling（✅ 保留，底层原语 · 不直接使用 · 2026-09-02 裁定）

**功能**：对计划/决策做压力测试的递归访谈：设计树 + frontier 轮次制（每轮问当前能答的决策题、附推荐答案）、事实自查、决策归用户、达成共享理解前不执行。

**触发方式**：model-invoked。

**初步复用判定**：✅ **保留为底层依赖**（A 组裁定）。grill-me / wayfinder 的底层原语；需求对齐阶段经调用方间接生效，不直接被 user/subagent 触发。配套协议：frontier 机制、事实 vs 决策分离、完成判据=无静默假设。

---

---

## handoff

**功能**：把当前会话压缩成**续接文档**，让一个全新 agent 能接续干活；落盘 `.plan/handoffs/<YYYY-MM-DD>-<slug>.md`（仓库根，`.plan/` 提交进版本控制、作为项目共享规划记忆；不在仓库内则落 OS 临时目录并告知绝对路径）；特有的 **suggested skills** 节——列出下一个 agent 应按名调用的技能、每项一句理由，并注明仅在其环境可用时生效；**不重复**其他产物（specs/plans/ADRs/issues/commits/diffs）已捕获的内容，改按路径/URL 引用；**脱敏**（API key、密码、PII）。

- 关键机制：压缩对象是**会话**不是项目（与文档台账互补）；「下一会话用途」作为参数传入时决定文档侧重。

**触发方式**：user-invoked（`disable-model-invocation: true`；`argument-hint: "What will the next session be used for?"`）。

**输入输出**：输入 = 当前会话全部上下文（被压缩对象）+ 可选「下一会话用途」参数；输出 = 一篇 markdown 续接文档（落盘 `.plan/handoffs/` 或临时目录）。

**依赖**：仓库（`.plan/` 目录，需在版本库内才有共享规划记忆语义）；无外部服务。

**初步复用判定**：**当前推演期采纳（裁剪）**。跨会话续接是现实需求——技能包评估本身就是多会话长任务（地图已注明「每次会话 consult…」的习惯、ticket 分批盘点），会话收尾时把「进度 + 下一会话该做什么 + suggested skills」写成续接文档，能显著降低下次会话重读 docs-tmp/seed 的成本。裁剪两点：①落盘位置按项目约定改 `.wayfinder/`（map/tickets 已是台账，handoff 侧重「会话级续接摘要」，如 `research/` 同目录或 ticket 内附续接节）；②吸收两条纪律——suggested skills（续接会话先加载哪些技能，与本文其它技能的 consult 约定呼应）、「不重复产物、引用路径」（与本项目「how 禁止写死路径、引用章节名」同精神）。插件实现期把该协议固化为「会话切换任务」的动作 how 亦顺理成章。

---

## teach（❌ 废弃 · 2026-08-27 裁定）

**功能**：教学生态 workspace（MISSION/lessons/reference/learning-records + 记忆科学设计）。

**初步复用判定**：❌ **废弃**（教学产出与 flow/capability/artifact 推演无交集，推演期无教学环节）。

---

---

## to-questionnaire（❌ 废弃 · 2026-08-27 裁定）

**功能**：把「用户独自答不了的一个决策」变成一张异步问卷（grill the send, not the subject）。

**初步复用判定**：❌ **废弃**（与流程无关——「向人取知识」已由需求对齐阶段 subagent 直接对话 + grill-me 当面盘问覆盖）。

---

---

## wait-what（✅ 保留，底层原语 · 不直接使用 · 2026-09-02 裁定）

**功能**：沟通失准时的「急刹车重述」——受控语言 + 项目共享术语重讲。

**初步复用判定**：✅ **保留为底层依赖**（A 组裁定）。失准重述是会话侧通用协议，随交互类 conversations 间接生效；可与 AGENTS.md 沟通准则合并一句约定，不单独注册。

---

---

## writing-for-agents

**功能**：写作任何「给 agent 消费的文档」的**写作规范参考**——skill、AGENTS.md/CLAUDE.md、被指针指向的文档同源同法（包装不同、写作不变），因为 agent 每次运行走的是同一套**过程**而非产出同一结果。机制性纪律共八块：context pointer（指针措辞决定命中）、两 load（常驻上下文成本 vs 人类认知负荷）、信息层级（阶梯 + 渐进披露 + 同址 + 防 sprawl）、step 完成标准（clarity 防 premature completion、demand 驱动 legwork）、按序列/触发切分、leading words（借预训练概念词锚定行为）、negation（禁制是失败模式，正向引导）、pruning（单一真相源 / 环境即真相源 / relevance / no-op 测试）。

- 关键机制：以「指针 + 阶梯 + 完成标准」三角为骨架——指针做按需取用、阶梯做放置决策、完成标准做过程锚定；剪枝纪律保证长期不沉积。

**触发方式**：model-invoked 允许（无 `disable-model-invocation`；description 绑定场景：创建/编辑 skill、修改 AGENTS.md 或 CLAUDE.md 时自动生效；写其它 agent 消费文档时也可显式采用）。

**输入输出**：输入 = 待写/待改的 agent 消费文档（skill、AGENTS.md、how/ref 文本等）；输出 = 文档本体 + 应用各纪律后的写法决策（无固定落盘格式）。

**依赖**：`SKILL-MECHANICS.md`（写 skill 时需读 frontmatter/调用选择/router skills——**本地技能目录缺失该文件**，仅 SKILL.md，需自上游 mattpocock/skills 仓库补齐后方可写 skill 类文档）；无其它外部资源。

**初步复用判定**：**当前推演期采纳（靶心）**。writing-for-agents 直接决定 how 编写规范 v2 的候选纪律来源（地图 Destination 第 2 项、触发根因「技能包直接影响 action 的 how 如何编写」），且本项目 how 本来就是给 subagent 消费的执行文档——其写作纪律与本项目 how 五段式的结合是本评估的靶心，逐条分析见下节。

### 与 how 写法的结合点

先立事实（一手源，供后文引用）：

- **how 的载体**：`capabilities.actions[].how`（字符串，动作内嵌），随 `wb_my_task` 每次注入 subagent（execution-loop §9：how 唯一真相源 = actions[]，prompt 不承载 how 全文——「程序路由，agent 执行」）。即 how 是**按次注入、用后即弃**的执行文档，不是常驻上下文。
- **五段式骨架**：读产物[productType] → 按模板[章节名] → 自检[quality_gates] → 登记/提评/响应[wb_* 工具] → 输出[productType]（map.md 定义；执行处见 capability-artifact-analysis §4.3「产出文档/代码 → 按模板章节骨架撰写 + quality_gates 自检 → 写盘 → wb_add_artifact 登记 → wb_comment 提评」）。
- **引用规约**：文档/代码类产物 → productType（受控枚举，§0.1 权威表）；对话/评论类产物 → query 属性过滤（§0.1.1：reviewerRole/assignee/state/anchor/thread）；模板路径经 `renderTpl` 渲染 `{{var}}`，how 禁止写死相对路径。
- **实例**：`backend-dev` 四动作（dev/self_test/submit/respond_to_cr）内嵌 how（capability-review-v2 §0.2），如 `backend-dev.dev`「读《rd_design》(…/rd/design.md) + 《architecture》(…/arch_design.md) + 《product_doc》(…/prd/prd.md) → 按设计契约实现（不造接口），对齐架构；产出代码（待自测）」；`respond_to_cr` 用 ①②③④ 编号分步；自审表（覆盖/一致/偏差/边界/测序，blocker/major severity，§4.2）。

以下逐条分析 writing-for-agents 每条纪律对 how 写作的影响。

#### (1) context pointer——how 里谁是「指针」，谁吃「措辞决定命中」这条纪律

- how 内的《标准名》(模板路径) 是**已被机器解析的指针**：`renderTpl` 注入时变真实路径，subagent 拿到的指针即目标——「措辞决定命中率」的经典风险在这里被降级。真正吃这条纪律的是另外三处：
  - **`when_to_use` 字段（指向动作的指针）**：subagent 靠它从 actions[] 里选动作（my-task-response-model：「每条 when_to_use/trigger 说明使 subagent 能判断当前场景用哪个动作」）。分支条件 = trigger（程序算好，如 `reviewer_feedback(open)`），when_to_use 是自然语言补充——若其措辞与 trigger 语义不一致（同义叠床或背道而驰），会制造「一条分支两种说法」的 variance。纪律落地：when_to_use 复用 trigger 词作 leading word 前载、每分支只写一个触发表述、不重复 trigger 已覆盖的语义。
  - **`reference.query`（指向对话产物的指针）**：query 的字段值与属性过滤条件（reviewerRole/assignee/state）就是分支条件，必须与 mytask 评论产物的**属性名完全一致**——拼写/大小写漂移（"Code Reviewer" vs "code_reviewer"）即指针措辞 bug，会整批漏采评论（respond_to_cr 依赖此过滤，漏采=没响应 CR）。
  - **ProductType 受控表（指针目标的稳定器）**：「命名受控、禁止自造」保证指针目标永不移位——这是 pointer 纪律在数据层的实现（目标唯一、枚举受控），how v2 应继续维护而不动摇。
- 同义改写的反面：how 正文不要用「《产品设计文档》/ product_doc / prd」三种写法指同一产物（一个分支写三遍）——统一标准名 + 模板路径。

#### (2) 两 load——how 不常驻，主要矛盾在认知负荷与注入体积

- **context load**：how 按次注入（execution-loop §9），常驻成本趋近 0——「最小化常驻上下文」不是 how 的主要矛盾。真正的 context load 是**单次注入体积**：wb_my_task 返回的是能力**全部动作**的 actions[]（how/reference/focus/tools 串），长能力（acceptance-test 类多动作）一次注入量大。纪律应用：把每个动作的 how 写短，长参考（模板骨架、quality_gates 明细）下推（见 (3)），不要「把整本书塞进单 action how」。
- **cognitive load**：how 的读者 = 编写推演者（人）+ 执行的 subagent。本项目已做了大量减负：受控枚举（ProductType/trigger）、renderTpl 路径渲染、「引用模板用章节名不用编号」——都是在把「写 how 要记的约定」转成程序校验。新增纪律：每引一个新受控词（含 future leading word，见 (6)）都要评估「写 how 的人记不记得住」——词只在反复用、语义稳定的场景引入。
- **指针无指针的对照**：模板文件永远不常驻，经《标准名》指针按需拉取——正中「material reached only through a pointer escapes context load at the price of the pointer's own line」；而 ProductType 表 / trigger 枚举这类「索引」是 human 的认知负荷，恰好花在人力判断价值点上（枚举怎么做、判定标准定什么——归用户），符合「spend it where human judgement matters」。

#### (3) 信息层级——how 已经是阶梯结构，但产物清单三处重复

- 现状映射：五段 = **steps 层**（有序动作序列：读→写→检→登记→输出）；`reference[]`/`focus[]` = **in-file reference**；模板文件 = **disclosed reference**（被《标准名》指针指向、按需加载）；ProductType 表 = 体量最小的顶层 reference。
- **问题**：产物清单在三个层级重复出现——how 散文里（「读《rd_design》+《architecture》+《product_doc》」）、reference[] 里（productType 列表）、focus[]/输入声明里。按分层测试：「读哪些产物」是每次分支都做的事，应留在 step 层；「重点章节、评审维度」是分支相关（评审型才用 focus），可下推。capability-artifact-analysis §2.5 的 A1–A4 四列表（动作/参考产物/重点章节/实施）已是正确分层的雏形——how v2 可正式采纳「产物清单（reference 层）与动作序（steps 层）语义分离」。
- **co-location（同址）**：章节名应紧随其所属产物（《rd_design》+「接口设计（契约）」相邻），现状章节名散在 focus[] 与模板表里——违反「读一部带邻居」测试。落地：how 内产物引用后直接括注重点章节，focus[] 只做机器可见性/校验。
- **sprawl 防线**：单动作 how 超过「一段话/一屏」即触发蔓延风险；长参考一律下推进 disclosed reference（模板/表格），how 顶部只留指针。

#### (4) Steps & completion criteria——自检步是满分样板，其余步缺 criterion

- **现状对比**：自检步（模板 quality_gates + dev 自审表）完成标准最完备——「对照表覆盖《rd_design》《fe_design》全部设计要点，无遗漏，缺一行即降级声明」「接口实现与契约一致，抽样 ≥3」= checkable + exhaustive 的教科书式 criterion；登记步有程序化标准（wb_add_artifact 校验 product_type 与动作 outputProductTypes 一致）；**读产物步、按模板撰写步没有显式 criterion**——premature completion 高风险：subagent「读了产物列表没读正文」「写了文档没覆盖全部必填节」就宣布完成。
- 落地建议：给非自检步补可检条件——读步 =「从《rd_design》提取接口清单与数据表清单」；写步 =「按模板章节名逐节落文，缺节即降级声明」（可与 §4.2 的 blocker 机制共用）。
- **post-completion 诱惑**：自测步后面跟着提交步，把「wb_add_artifact/wb_comment 细节」暴露在后，subagent 可能 rush 自测。respond_to_cr 的 ①②③④ 编号已是「按序列分段」的雏形；纪律要求藏后段——把 submit 的 tool 参数细节与自测完成标准分离，而不是平铺进一句话。
- **demand 杠杆**：现有「按设计契约实现」demand 弱（只靠「不造接口」约束）；「覆盖全部设计要点」demand 强。写 how 时用 demand 词（逐一/逐条/全部…）驱动 legwork，不靠「认真/仔细」这类 no-op。

#### (5) When to split——how 已按触发拆动作，边界真实有效

- how 已经是**按调用拆分**（backend-dev 拆 dev/self_test/submit/respond_to_cr，对应 node_enter/dev_complete/dev_complete/reviewer_feedback(open) 四个 trigger），动作边界 = 真实上下文边界（不同轮次调起、经 wb_my_task 重新注入）——符合「split earns it, 且只在跨真实上下文边界时隐藏后段有效」。
- 两个提示：①镜像风险——**不要反向合并**：dev 与 self_test 触发虽同为 node_enter，但完成标准不同、诱惑面不同，合成一个 how 会让「自测」的完成标准被「开发」淹没；②inline 不藏段——若 submit 细节会让当前步 premature completion，宁可拆成动作（submit 有自己 trigger）也不在 how 里提前写「提交时记得 wb_add_artifact」。

#### (6) Leading words——已有三个隐式 leading word，可转正式词条

- 现状盘点：**「契约」**（「按设计契约实现」）、**「各审各的」**（评审只审职责内维度，arch_design §8.5 铁律）、**「降级声明」**（dev.md「偏差与风险说明」节的显式声明机制）——都已是反复出现的短语锚点，且都是预训练里语义扎实的词，正是 leading word 的形态。how v2 可把它们转成正式词条：一次定义、全域（how/reference/模板）复用，省 tokens 又稳行为。
- **指针侧锚定**：让「契约」同词出现在 how（按设计契约实现）、reference（productType=rd_design）、模板（「接口设计（契约）」节名）三处——共享语言让 subagent 更可靠地把引用与行为连起来（这正是 (1) 里 when_to_use 复用 trigger 词的同类机制）。
- 自造词要克制：自造词不借预训练先验、定义成本高。例：为「实现-设计对照表的完整性」造新词，不如直接用模板节名「实现-设计对照表」——已有词、已出现于模板，现成 leading word。
- 反面现状：how 里「对齐架构」「按设计契约实现」并用是两句——若统一收缩为「契约」担当架构一致与接口一致两个锚点（或各归一词），可省句且锚点更锐。

#### (7) Negation——how 与规约里的否定式逐句过审

- 否定式盘点（现存）：**「不造接口」**（dev）、**「禁止自造」**（ProductType 命名受控）、**「不自行改设计→停止并报告 Leader」**（自审表偏差行）、**「不评排期」**（pm-review，J23 批注已删该相关实现）、**「无遗漏行」**（自审覆盖行）、**「不硬切分支」**（worktree 读代码）。
- 逐条处理：
  - 「不造接口」→ 正向：**「接口仅实现《rd_design》「接口设计（契约）」所列」**——正向目标多带一条 demand（把要做的写出来），比禁令更可用。
  - 「无遗漏行」→ 正向：**「对照表逐条打勾，与设计要点一一对应」**。
  - 「禁止自造」→ 硬守卫（受控枚举难以完全正向化），但纪律允许守卫存在——且现状已配正向句「新增流程/节点只绑定上表已有 productType」，守卫+正向目标并存，符合规范。
  - 「不自行改设计→停止并报告 Leader」→ 禁令前半可以保留（是安全阀式的 hard guardrail），后半「停止并报告 Leader」本身就是正向动作——守卫带正向配对，同样合规。
- 结论：how v2 审稿时逐句过否定式，能正向化的都转正向；否定式仅在「改变行为的安全阀」处保留且必须带正向配对。（对照：think-of-an-elephant 机制——「不造接口」把「造接口」拉进上下文，弱修饰盖不过强激活的概念，禁令半读成指示。）

#### (8) Pruning——最大裁点是 how 内嵌路径 vs 权威表

- **单一真相源（本评估最大裁点）**：ProductType 0.1 表把 productType→标准名→路径模板绑定为**唯一权威**；但 J7 定稿要求 how/ref 里每个产物直接写成「《标准名》(模板路径)」——等于把 0.1 表的内容复制进每一个 how：**重复 + 缓存**。按 cache 纪律：路径的查找很便宜（0.1 表一次查），不值得随 how 复制；复制必漂移（改一次路径要改 N 处 how）。落地建议：**how 里只写《标准名》，路径由宿主 renderTpl 从 0.1 权威表取值渲染**（一次定义、全 how 复用）——这其实是 J7「host 端 renderTpl 替换」设计意图的正确延伸：渲染源 = 权威表而非 how 字符串。（该修正是对既有定稿的偏离建议，需用户在 how v2 票里确认。）
- **环境即真相源**：模板文件（docs/templates/*）不抄进 how——已遵守（how 只引用章节名，不抄模板正文）；章节名 = 对模板的 cache 指针，存在单向漂移风险（模板结构改了、how 的章节名引用过期）——建议登记时程序校验「how 引用的章节名存在于对应模板」（与 §4.3 登记闭环的预检同源，capability-artifact-analysis 发现 4 的增强项可顺手覆盖）。
- **relevance/sediment 实例**：pm-review 简介里「不评排期」经历的 J23 批注删除，正是「行为变了、文档没跟上」的沉积史样本——盘点应定期核对 how 时效；建议在动作元数据加「最后核对日期」，让过期 how 可被扫描出来。
- **no-op 测试**：how 里「对齐架构」「产出代码（待自测）」等句——若 subagent 默认行为已含（看到《architecture》在 reference 就会读、产出代码是任务本义），该句不改变行为 = no-op，删整句；改变行为的句子（「按设计契约实现」约束了本来会自造接口的行为）保留。测试是模型相对的，靠运行验证而非争论（纪律原话：settle it by running the document）。

#### (9) 小结：纪律 × 五段式的落点

五段式提供 **steps 骨架**（顺序与工具位），writing-for-agents 提供**每段内部写法**（完成标准、正向措辞、层级放置、剪枝）。骨架不冲突；冲突/待裁点集中在两处：①**重复权威**（how vs reference[] vs 0.1 权威表；路径渲染源 vs how 内嵌路径）；②**否定式措辞**（不造接口/无遗漏行等可正向化）。how v2 优先裁这两处，其次补非自检步的完成标准、把「契约/各审各的/降级声明」转正式词条、章节名与产物同址。完整「每条纪律 × 五段式 = 互补/冲突/分层」的系统对照是 ticket 04 的范围，本文只做 writing-for-agents 单技能深挖，供 04/05 票引用。

---

## 汇总表

| 技能 | 触发方式 | 初步复用判定 | 一句话理由 |
|---|---|---|---|
| grill-me | user-invoked | 插件实现期激活 | 与 grilling 同构，仅节奏不同，推演期无增量 |
| grilling | model-invoked | 当前推演期采纳 | 地图已定 consult；决策收敛协议 |
| handoff | user-invoked | 当前推演期采纳（裁剪） | 跨会话续接 + suggested skills；落点改 .wayfinder/ |
| teach | user-invoked | 不适用 | 教学产出与推演无交集 |
| to-questionnaire | user-invoked | 当前推演期采纳 | 向人取知识的异步问卷；grill the send |
| wait-what | user-invoked | 当前推演期采纳（裁剪） | 失准重述协议；STE→中文受控、CONTEXT→域模型词表 |
| writing-for-agents | model-invoked | 当前推演期采纳（靶心） | how 编写规范 v2 的纪律来源，见结合点小节 |