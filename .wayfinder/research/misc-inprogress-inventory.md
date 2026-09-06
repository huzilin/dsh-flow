# misc / in-progress 桶盘点（12 个技能）

> 来源：本地技能一手源 `~/.dsh/skills/mp-<name>/SKILL.md`（逐篇全文读取，未凭名字推断）
> 所属票：`.wayfinder/tickets/03-misc-inprogress-inventory.md`（research）
> 判定标准（Q4=A）：按当前阶段「能否改进流程/能力/产物/动作推演工作」为第一标准；编码向技能标注「插件实现期激活」。
> 判定档位：**当前推演期采纳** / **插件实现期激活** / **不适用**

---

## git-guardrails-claude-code（misc）

**功能**：在 Claude Code 中安装 PreToolUse hook，在危险 git 命令真正执行前拦截并阻止（`git push` 含 `--force`、`reset --hard`、`clean -f/-fd`、`branch -D`、`checkout .` / `restore .`），被拦时 Claude 会看到"无权限访问这些命令"的提示。关键流程：①问作用域（项目 `.claude/settings.json` 或全局 `~/.claude/settings.json`）；②把捆绑脚本 `scripts/block-dangerous-git.sh` 复制到目标 hooks 目录并 `chmod +x`；③把 hook 合入 settings 文件的 `hooks.PreToolUse` 数组（已有配置则合并、不覆盖其他设置）；④问是否增删拦截模式；⑤用管道 JSON 喂脚本验证（期望 exit 2 + stderr 输出 BLOCKED）。

**触发方式**：model-invoked（frontmatter 无 `disable-model-invocation`，靠 description 语义触发：用户提 git 安全 / 加 git 防护 hook / 禁止 push、reset 时）。

**输入输出**：输入 = 用户意图 + 作用域选择 +（可选）自定义拦截清单；输出 = hook 脚本 + 对应 settings.json 的 hook 配置。

**依赖**：Claude Code 环境（`.claude/` 目录结构、PreToolUse hook 机制）；技能内置脚本 `scripts/block-dangerous-git.sh`。

**初步复用判定**：**插件实现期激活**（低优先）。DSH 环境本身不是 Claude Code，推演期无 git 破坏操作问题；且本环境已有 DSH 沙箱/审批机制承担同类拦截，功能重叠。插件实现期若用 Claude Code 写代码，该 guardrail 可直接照抄。价值点在"明确拦截清单 + 可验证（喂 JSON 测 exit code）"的防护写法，可作为实现期质量基线参考。

---

## migrate-to-shoehorn（misc）

**功能**：把测试代码中的 `as Type` / `as unknown as Type` 类型断言迁移为 `@total-typescript/shoehorn` 的类型安全替代（`fromPartial()` 传部分数据、`fromAny()` 故意传错数据、`fromExact()` 强制全量），解决测试里"为一个大对象伪造全部字段"与"double-as 故意错数据"两个痛点。关键机制：只在测试代码用、绝不进生产；迁移流程 = 问需求（哪些文件有 `as`、是否大对象只关心部分字段、是否要故意错数据）→ `npm i` → `grep -r " as [A-Z]"` 找断言 → 分类替换 → 补 import → 跑 typecheck 验证。

**触发方式**：model-invoked（无 `disable-model-invocation`；用户提 shoehorn、想替换测试里的 `as`、或需要部分测试数据时触发）。

**输入输出**：输入 = 含 `as` 断言的测试文件与需求；输出 = 迁移后的测试代码（+ 依赖安装）。

**依赖**：`@total-typescript/shoehorn` npm 包；TS 测试代码库中的 `*.test.ts` / `*.spec.ts`。

**初步复用判定**：**插件实现期激活**。推演期无 TS 测试代码可迁移，不适用；但插件实现期（本项目 dev）写插件测试时，"部分数据 mock 保持类型安全"是高频需求，届时按此迁移模式执行。当前不深入。

---

## scaffold-exercises（misc）

**功能**：创建练习目录结构（`exercises/XX-section-name/` 下 `XX.YY-exercise-name/`，每个练习含 `problem/`、`solution/`、`explainer/` 变体子目录，readme 存根），并通过 `pnpm ai-hero-cli internal lint` 校验后 `git commit`。关键机制：约定目录命名（编号 dash-case）；stub 默认 `explainer/`；每个变体子目录的 `readme.md` 必须非空且无坏链，有代码时需 `main.ts`（>1 行），readme-only 也可；移动/重编号用 `git mv` 保历史；linter 规则清单明确（禁 `.gitkeep`、`speaker-notes.md`、坏链、`pnpm run exercise` 命令等）。

**触发方式**：model-invoked（无 `disable-model-invocation`；用户要 scaffold 练习、建练习存根、开新课节时触发）。

**输入输出**：输入 = 课程计划（章节/练习/变体类型列表）；输出 = exercises/ 目录树 + readme 存根（+ lint 通过 + git commit）。

**依赖**：强绑定特定仓库工具链 `pnpm ai-hero-cli internal lint`（mattpocock 课程仓库自有 CLI）+ `exercises/` 目录约定。

**初步复用判定**：**不适用**（技能本体）。本项目无课程/练习结构，linter 也无法迁移。但机制可借鉴：目录骨架化 + 机器校验闭环的"建结构 → 自动验证 → 修到过"方法，与推演产物的骨架化（先立节点骨架再填内容）同构，可写进 how 规范作参考，不作为技能复用。

---

## setup-pre-commit（misc）

**功能**：在当前仓库装上 Husky pre-commit hook + lint-staged（Prettier）+ typecheck/test，做到提交时自动格式化、类型检查、跑测试。关键流程：①按 lock 文件探测包管理器（npm/pnpm/yarn/bun，默认 npm）；②装 `husky lint-staged prettier`（devDependencies）；③`npx husky init`（建 `.husky/` 并加 `prepare: "husky"`）；④写 `.husky/pre-commit`（`npx lint-staged` + `npm run typecheck` + `npm run test`，无对应 script 则省略该行并告知用户）；⑤建 `.lintstagedrc`（`"*": "prettier --ignore-unknown --write"`）；⑥缺 `.prettierrc` 才建（默认配置）；⑦按验证清单核对；⑧提交冒烟测试（提交本身跑一遍新 hook）。

**触发方式**：model-invoked（无 `disable-model-invocation`；用户要加 pre-commit hook、配 Husky/lint-staged、加提交时格式化/类型检查/测试时触发）。

**输入输出**：输入 = 当前仓库；输出 = `.husky/`、`.lintstagedrc`、（可选）`.prettierrc`、package.json 的 prepare script。

**依赖**：npm/pnpm/yarn/bun 环境；husky / lint-staged / prettier 包；仓库需有 typecheck/test script（缺失则降级处理）。

**初步复用判定**：**插件实现期激活**。当前推演期产物是 md/xlsx，无代码提交门禁需求，不适用；插件实现期启动代码仓库时即为基线工程设置。注意：commit 门禁与 DSH 别处机制（approve/沙箱）不重叠，属仓库自身体系，实现期直接采纳。

---

## claude-handoff（in-progress）

**功能**：把当前会话写成交接摘要，不落盘保存而是直接以摘要为 prompt 启动一个后台新 agent 继续干活（`claude --bg --name "<描述性名字>" "<摘要>"`，在当前工作目录启动并立即返回，用户用 `claude agents` 管理）。关键机制：摘要必须含 **suggested skills** 节（指明下一 agent 该调哪些 skill）；不重复已存在于其他产物（specs/plans/ADRs/issues/commits/diffs）的内容，用路径或 URL 引用；脱敏（API key、密码、PII，因为摘要会成为 agent 的 prompt）；用户传参时把参数当"下一会话要做什么"来定制摘要。

**触发方式**：**user-invoked**（`disable-model-invocation: true`；`argument-hint: "What will the next session be used for?"` 表明用户主动传参触发）。

**输入输出**：输入 = 当前会话上下文 +（可选）下一阶段目标描述；输出 = 后台 `claude --bg` 进程（以脱敏摘要为 prompt）。

**依赖**：Claude Code 的 `claude --bg` 后台模式 + `claude agents` 管理界面；需现有产物文件/URL 可供引用。

**初步复用判定**：**当前推演期采纳**。本项目推演链路本身多会话/多 agent 接力（子代理任务、长推演拆段），"交接摘要"动作直接对应。三条机制可直接进 how 规范 v2 的登记/交接环节：①摘要只引用产物路径、不重复内容（防上下文膨胀）；②必含 suggested skills；③脱敏。具体命令 `claude --bg` 绑定 Claude Code，替换为 DSH 对应物（本环境 subagent / goal 接力机制）即可复用全部写法。

---

## implement-spec（in-progress）

**功能**：按 spec（带关联票）在**单一分支**上产出一个实现整个 spec 的 PR，把票当作**任务图**（含阻塞关系）而非步骤列表，始终存在可抓取的 **frontier**（就绪票前沿），用多个 implementer 子代理并发实现以获得最大并发度。关键流程：①读 spec 和票理解任务图；②（可选）exploration 子代理做探索并把 markdown 笔记存到 **repo 外**、所有后续子代理可达的目录（让 implementer 专注实现）；③建分支 + **draft PR**（标注 closing spec issue 和票）；④每个 implementer 在自己的 worktree/分支上实现票；⑤完成后 merger 子代理并入 PR 分支；⑥frontier 变化就再开 implementer 并发；⑦全部完成跑 `/code-review` 修问题；⑧标记 ready；⑨清理所有 worktree。通信原则：稀疏，主要通过**上下文指针**（指向 spec/票/研究笔记/既往 commit）传递，不重复信息。

**触发方式**：**user-invoked**（`disable-model-invocation: true`）。

**输入输出**：输入 = spec + 关联票；输出 = 单分支 PR（draft → 合并全部实现 → code review 通过 → ready）。

**依赖**：git worktree 机制；子代理（implementer/exploration/merger 三类）；`/code-review` 技能。

**初步复用判定**：**插件实现期激活**（主体）+ 推演期部分借鉴。实现期是主流：spec → 票 → 并发实现正是插件开发主流程，届时全流程照用。推演期可借鉴的是 **task-graph / frontier** 编排思想：本项目 flow 推演按节点批量推进、节点间有依赖，frontier（"当前就绪可抓取的节点集合"）概念可直接借用描述推演任务推进顺序与批量作业分派，写进 how 规范的推进策略。

---

## loop-me（in-progress）

**功能**：在 workspace 里跑一场**有状态 `/grilling` 会话**，唯一产出是 **workflow 规格**（`workflows/*.md`），用 "loop" 镜头找出用户生活中值得委托的重复模式并规格化。关键机制与词汇：**loop** = 用户生活中的反复模式（职业/一周/早晨/单个重复活动），把生活看作层层嵌套的 loop 能暴露其可预测性——这正是值得委托的原因；**workflow** = 一个 loop 的规格，运行于 loop 之上，`workflows/*.md` 是唯一事实源；**trigger** 决定每次运行由什么触发——event（新邮件/新 issue）通常比 schedule 高效；**checkpoint** = 人在环验证点，有些流程没有、可全自动，甚至不用 AI；**push right** = 把 checkpoint 尽量后移，先做完最大工作量，让人只被问一次、问得晚、且一切都备好；**brief** = checkpoint 呈现的"决策就绪摘要"（产出了什么、为什么、链接到底层产物），用户读 brief 而非原始输出，评审速度是硬要求。**Definition of done**：workflow 规格做到实现者 agent 能无疑问地构建为止，问题未清不算完。`NOTES.md` 存用户世界的原始笔记/工具/术语，薄时就先访谈。

**触发方式**：**user-invoked**（`disable-model-invocation: true`；`argument-hint: "A workflow to design, or nothing to go find one"`）。

**输入输出**：输入 = 用户想设计的 workflow 或空（技能去找）；输出 = `workflows/*.md` 规格 + `NOTES.md`。

**依赖**：`/grilling` 技能（有状态 grilling 会话）作为执行机制。

**初步复用判定**：**当前推演期采纳**（高相关，优先级靠前）。与本项目高度同构：项目就是在做 flow/capability/action 推演，loop-me 是"生活/流程层面规格化"的同类物。可直接吸收进 how 规范 v2（响应/输出环节）：**①brief 原则**——给用户呈现决策就绪摘要而非原始输出，正对应本项目推演稿的呈现纪律；②**push right**——检查点后移、一次问清，正是推演批注/评审的节奏设计；③trigger（event 优先于 schedule）对应流程触发设计；④"做到无疑问才 done"对应推演产物的完成标准。（词汇层：trigger/checkpoint/push right/brief 可直接成为推演动作的标准术语。）

---

## retro（in-progress）

**功能**：对一次编码会话做**回顾**，产出对"编码 agent 的**环境**"（而非代码）的改进建议，以提升未来运行。关键流程：①先调 `writing-for-agents` 技能拿写作规范；②读用户指定会话的一手源（可能翻本机会话日志，未指定则默认当前会话）；③按六个类别找改进候选——**Navigation**（agent 找文件难不难、有无隐藏依赖、要不要导航指针；会话花很久找信息时用）、**Automated checks**（能否加 lint/类型/测试/文件系统 linter 拦住犯过的错）、**Coding standards**（是否该给 reviewer agent 加/改/删规则；review 漏错时用）、**Global AGENTS.md**（有无该下沉为编码标准或自动检查的指令；文件过大时用）、**Tool economy**（昂贵的工具调用、token 低效的自制工具）、**No-ops**（steering 文件里不改变行为的指令）、**Information access**（扩大 agent 信息通道，如 dev server 日志）；④按严重度排序呈现。参考框架：implementation vs review——实现 agent context 压力最大（探索+写码+调试），review agent 压力最小（只收 diff），所以**编码标准应由 review agent 强制而非实现 agent**；文件职责——`CLAUDE.md`/`AGENTS.md` 只放导航指针、`CODING_STANDARDS.md` review 时读、docs 作被指针引用的参考、skills 用于文档或用户调用命令。

**触发方式**：**user-invoked**（`disable-model-invocation: true`）。

**输入输出**：输入 = 指定编码会话（默认当前）；输出 = 按严重度排序的环境改进候选清单（含每条的"何时用"触发条件）。

**依赖**：`writing-for-agents` 技能（写作规范）；会话日志（本机可检索）。

**初步复用判定**：**当前推演期采纳**。retro 的"环境改进候选分类"= 一套对 agent 工作环境的审计维度，本项目推演期正在优化自身工作方式（AGENTS.md 裁剪、how 规范迭代），这七个类别恰是"推演 agent 自己的动作链"的检查清单，直接可用作**动作推演/流程审查**的维度表（navigation/checks/standards/no-ops/tool economy 每项都对应一个可推演的动作改进点）。插件实现期则按原意对照码会话定期执行。另：其文件职责原则（AGENTS.md 只放导航指针）与本项目 AGENTS.md 治理方向一致，写 how 规范时引用。

---

## setup-ts-deep-modules（in-progress）

**功能**：把 `dependency-cruiser` 接进 TS 仓库，让每个 package 成为 **deep module**（小接口背后大量行为）：公开面 = 包根**入口文件**，子文件夹内一切实现对外隐藏，外部只能经入口访问。关键机制：四条 `error` 规则——①入口边界（外部只能 import 包根文件，不得进子目录）；②包内自由互引；③测试走入口（`tests/` 可引任意包入口与自身 fixtures，但绝不引任何包子目录内部）；④无循环依赖。**入口是多文件、非 barrel**（可暴露多个小入口 `index.ts`/`client.ts`/`server.ts`，不鼓励巨型 index 重导出整棵子树）。公私由**深度**决定（根文件公开、任何子目录私有，"任何子目录都私有"的泛规则让新加目录无需改配置）。流程：①探测环境（包管理器、包根 `src/packages` 或 `packages`、已有 `.dependency-cruiser.*` 则**合并不覆盖**）；②装依赖；③复制内置 `dependency-cruiser.config.cjs`（用 `.cjs` 兼容 type:module）、设 `PACKAGES_ROOT`；④加 `lint:boundaries` 并入已有的 typecheck 伞命令；⑤搭 `example/` 模板包（入口委托内部 `lib/impl.ts`，测试只引入口）；⑥**证明规则生效**（完成判据：clean pass → 临时加 deep import 必须 fail → 还原再 pass）；⑦写 `<packages-root>/README.md` 记录约定 + 在 `CLAUDE.md`/`AGENTS.md` 加一行上下文指针。词汇（deep module/interface/seam/depth）调用 `codebase-design` 技能。

**触发方式**：**user-invoked**（`disable-model-invocation: true`，description 明示 "User-invoked"）。

**输入输出**：输入 = TS 仓库；输出 = `.dependency-cruiser.cjs` + `lint:boundaries` + example 包模板 + `packages/README.md` + AGENTS.md/CLAUDE.md 指针。

**依赖**：`dependency-cruiser` npm 包；`codebase-design` 技能的深度模块词汇；`CLAUDE.md`/`AGENTS.md` 文件。

**初步复用判定**：**插件实现期激活**。作为工具接入无当前用途（推演期无 TS 包边界问题）。但 deep-module 词汇（公开面小而窄、实现藏深、接缝）在推演期有**隐喻价值**：本项目把能力/产物模块化时，"小入口 + 藏实现"是同一原则，可与 codebase-design 一起用作产物/能力边界划分的推演语言。实现期则全流程照用（含"证明规则咬人"的完成判据写法，值得写进实现规范）。

---

## writing-beats（in-progress · ❌ 废弃 2026-09-02）

**功能**：**exploit** 写作——把一份固定的 raw material 文件组装成以 **beat**（旅程中的一个行动单位）为步的**概念旅程**，任何术语在被某个 beat 依赖之前必须先 **grounding**（落地）。关键机制：①先与用户确立 **prerequisites**（读者进门就懂的概念集合，固定不变）；②从素材里给 2-3 个候选**起始 beat**（不同文章入口，只能依赖已 grounded 概念，注明各自新 ground 什么），用户选一个并预览其解锁路径；③**一次只写一个 beat** 到文章文件（一句到几段皆可，自然大小）即停；④**重读磁盘文件**再给 2-3 个候选 next beat（必须从当前 grounded 集可达，注明各 ground 什么）；⑤循环至旅程自然终结。grounding 单位是**概念**而非词——无术语的陌生想法同样算未 grounded；进度跟踪"已 grounded 清单"。大杠杆 = 什么设为 prerequisite vs 什么在文内 ground（前置要太多关掉读者，文内 ground 太多开头溺死在定义里）。文章以旅程完整为终，不要求清空素材堆（pile 是采石场，可改写/拆/拼/引）。

**初步复用判定**：❌ **废弃**（用不太到；需求对齐直接对话已覆盖访谈碎片收集、产出 skill 模板驱动已覆盖文档成型）。

---

---

## writing-fragments（in-progress · ❌ 废弃 2026-09-02）

**功能**：**explore** 写作——不做任何结构承诺，跑一场 grilling 式访谈（relentless 追问），产出 **fragments**（可能幸存进最终文章的文本片段）追加进单个 markdown 文件，把"写作探索"与"结构成型"（exploit，别的技能的活）分离。关键机制：phase/outline/文章结构一律越界；fragment 标准 = "作者自己读得懂"（不必对冷读者自足），如金句、带一句论证的论断、小景/代码片/类比的 vignette、半想法、引语、按感觉成簇的观察、**leading word**——最值钱的片段，一个负载性隐喻/造词命名整篇（如 tracer bullets）；文件格式 = 首写放一个 H1 工作标题 + 无元数据，片段间 `\n---\n` 分隔，体内无小标题；写作节奏 = 静默追加（不逐条请示）、每次写前**重读磁盘**（用户可能编辑/重排/删除）、只追加不覆盖，用户可随时"砍掉/改锐/合并"。

**初步复用判定**：❌ **废弃**（用不太到；需求对齐直接对话已覆盖访谈碎片收集、产出 skill 模板驱动已覆盖文档成型）。

---

---

## writing-shape（in-progress · ❌ 废弃 2026-09-02）

**功能**：**exploit** 写作——把一份 raw material 文件（从规整片段列表到无结构文本墙均可）当作固定 input pile，**逐段**塑造成一篇文章（独立文章文件，raw 文件只读）。关键机制：①**通读 pile** 再动笔；②确立 prerequisites（读者进门已知）；③给 2-3 个**候选开头**（各暗示不同论点/角度），逼用户选或杂交，所选开头定义全文义务；④循环"**给定这个开头，读者接下来需要听到什么**"，从 pile 里取料回答，下一块只能依赖已 grounded 概念并逐个 ground 新的，**格式论证**（prose vs list、inline vs callout、table vs 重复结构、quote vs paraphrase、code block vs inline——每种格式选择都要有意识、可辩护）；⑤逐块追加文章文件不批量；⑥用户决定何时完成。grounding 同 beats（概念单位、prerequisite/introduced 两路、维护已 grounded 清单）；**gap-naming** = pile 缺素材就直接说"这里需要例子而 pile 没有，现在给我一个，否则砍掉这节"；对话纪律 = 反 grilling："这段为读者做了什么上段没做的？""删了它什么会坏？""开头承诺 X，漂到 Y 了，重穿线或改开头"。

**初步复用判定**：❌ **废弃**（用不太到；需求对齐直接对话已覆盖访谈碎片收集、产出 skill 模板驱动已覆盖文档成型）。

---

---

## writing 系列三件套 × writing-for-agents 关系与项目关联（单列说明）

- **两个写作域**：writing-beats / writing-fragments / writing-shape 属**给人读的文章/文稿**的写作域（fragments = explore 素材收集，beats = exploit 概念级旅程成型，shape = exploit 段落级文章成型）；writing-for-agents 属**给 agent 读的指令文档**的写作域（技能/AGENTS.md/CLAUDE.md/被指针引用文档，机制为上下文指针、两层负载、信息层级/渐进披露、聚合词、完成判据、剪枝）。两者不是竞争而是**互补**：本项目推演稿（capability-review-v2 等，人读）用三件套，agent 指令文档（how 规范 v2、AGENTS.md、技能）用 writing-for-agents。
- **共享概念**：①**leading word** 两域同源——writing-fragments 中"命名整篇的隐喻词"，writing-for-agents 中"聚合行为的 pretrained 词"，说明"一个好词锚定一大片行为/结构"是通用写作原理；②**前置知识/grounding** 与 writing-for-agents 的 **branches/信息层级** 同构——都是"读者/agent 进门已有什么、后续如何按需补"，三件套对读者、writing-for-agents 对 agent。
- **项目文档写作关联**：`writing-shape`（或论证深时叠 `writing-beats`）可直接作为 capability-review-v2 等推演稿的成型协议；`writing-fragments` 作推演素材的探索期沉淀；`writing-for-agents` 作 how 规范 v2 与后续技能/AGENTS.md 的写作基准。三件套的 explore→exploit 流水线 = 推演稿标准生产流程。

---

## 汇总表

| 技能 | 桶 | 触发 | 判定 |
| --- | --- | --- | --- |
| git-guardrails-claude-code | misc | model-invoked | 插件实现期激活（低优先） |
| migrate-to-shoehorn | misc | model-invoked | 插件实现期激活 |
| scaffold-exercises | misc | model-invoked | 不适用（机制可借鉴） |
| setup-pre-commit | misc | model-invoked | 插件实现期激活 |
| claude-handoff | in-progress | user-invoked | **当前推演期采纳** |
| implement-spec | in-progress | user-invoked | 插件实现期激活（frontier 概念推演期借鉴） |
| loop-me | in-progress | user-invoked | **当前推演期采纳**（高相关） |
| retro | in-progress | user-invoked | **当前推演期采纳** |
| setup-ts-deep-modules | in-progress | user-invoked | 插件实现期激活（deep-module 词汇推演期参考） |
| writing-beats | in-progress | user-invoked | **当前推演期采纳** |
| writing-fragments | in-progress | user-invoked | **当前推演期采纳** |
| writing-shape | in-progress | user-invoked | **当前推演期采纳** |

misc 桶 4 个全部为编码/环境基建向 → 均标注「插件实现期激活」或「不适用」；in-progress 桶 8 个中 6 个判「当前推演期采纳」（claude-handoff / loop-me / retro / writing-beats / writing-fragments / writing-shape），2 个编码向判「插件实现期激活」（implement-spec / setup-ts-deep-modules）。in-progress 桶整体质量高于 misc 桶，与项目推演场景贴合更紧，建议评估稿优先消化 in-progress 桶的 6 个采纳项。