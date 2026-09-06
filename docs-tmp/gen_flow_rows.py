# ⚠️ 已过时（2026-09-05 v3 编排模型定稿）：本生成器基于旧模型（旧节点拓扑/trigger 词汇/
# respond_to_cr 等已废止动作）——将按 v3.1 节点配置（business/flow-derivation-tech-task-plan.md）
# 重写（工单 07 seed 重写计划）；配套 能力链路-流程-节点-能力-动作.xlsx 同步过时。勿据本文档生成。
# -*- coding: utf-8 -*-
"""工作台能力链路 xlsx 生成数据：流程 -> 节点 -> 能力 -> action -> how
来源：workbench-sql/seed.json（流程/节点/能力绑定）+ docs-tmp/capability-review-v2.md（action/how 推演）

━━━━━ how 路径模板（技术可实施，J7 定稿）━━━━━
how/ref 中每个产物都以「标准名 + Python/JS 通用模板路径」直接呈现，如：
    《rd_design》 (docs/{{req-version}}/{{req-name}}/rd/design.md)
模板语法为 {{var}} 双花括号（Python str 与 JS 正则均可直接替换，语法一次定义两端复用）。

模板变量元数据（ctx 来源）：
    {{req-version}}  需求所属版本号（requirements.version，如 v1.1）
    {{req-name}}     需求 slug（requirements.slug/req_name，如 req-019-join-rollback）
    {{side}}         开发侧（rd|fe，仅 dev_notes 路径使用；来自能力 role_id）

宿主端（JS/TS 插件）渲染实现（变量名可含连字符，如 req-version）：

    // 把 how/ref 中的 {{var}} 替换为真实值；未知变量抛错防止静默出错
    function renderTpl(s: string, ctx: Record<string, string>): string {
      return s.replace(/\{\{([^{}]+)\}\}/g, (_m, k) => {
        const key = k.trim();
        if (!(key in ctx)) throw new Error(`how 模板变量未提供: {{{key}}}`);
        return ctx[key];
      });
    }

    // 调用示例
    const ctx = { 'req-version': 'v1.1', 'req-name': 'req-019-join-rollback', 'side': 'rd' };
    renderTpl('docs/{{req-version}}/{{req-name}}/rd/design.md', ctx);
    // -> 'docs/v1.1/req-019-join-rollback/rd/design.md'

渲染时机：capabilities.actions[].how / inputs.reference 定义层存模板；
wb_my_task 返回时按当前需求 ctx 渲染后再发给 subagent（host 侧 renderTpl 调用点）。
（本脚本底部 render_tpl() 为 Python 同构实现，仅用于生成期校验模板可替换。）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"""

# ProductType → 路径模板（与 capability-review-v2 §0.1 权威表一致；docs/ 前缀直接写进模板）
PTYPE_TPL = {
    'product_doc':  'docs/{{req-version}}/{{req-name}}/prd/prd.md',
    'rd_design':    'docs/{{req-version}}/{{req-name}}/rd/design.md',
    'fe_design':    'docs/{{req-version}}/{{req-name}}/fe/design.md',
    'case_design':  'docs/{{req-version}}/{{req-name}}/qa/cases.md',
    'test_result':  'docs/{{req-version}}/{{req-name}}/qa/test.md',
    'defect':       'docs/{{req-version}}/{{req-name}}/qa/defect.md',
    'dev_notes':    'docs/{{req-version}}/{{req-name}}/{{side}}/dev.md',
    'plan':         'docs/{{req-version}}/{{req-name}}/plan.md',
    'research':     'docs/{{req-version}}/research.md',
    'architecture': 'docs/{{req-version}}/arch_design.md',
    'alignment':    'docs/{{req-version}}/align.md',
    'roadmap':      'docs/{{req-version}}/roadmap.md',
    # code_mr / feature_branch 无文件路径模板（MR 外链 / flow_nodes 字段）
}
# 模板文件 → 固定路径（无变量）
TPLFILE_TPL = {
    'prd-template.md': 'docs/templates/prd-template.md',
    'backend-design-doc-template.md': 'docs/templates/backend-design-doc-template.md',
    'frontend-design-doc-template.md': 'docs/templates/frontend-design-doc-template.md',
    'qa-testcase-review-template.md': 'docs/templates/qa-testcase-review-template.md',
    'qa-acceptance-template.md': 'docs/templates/qa-acceptance-template.md',
    'qa-defect-management-template.md': 'docs/templates/qa-defect-management-template.md',
    'dev-notes-template.md': 'docs/templates/dev-notes-template.md',
    'plan-template.md': 'docs/templates/plan-template.md',
    'research-open-source-tech-template.md': 'docs/templates/research-open-source-tech-template.md',
    'arch-version-design-template.md': 'docs/templates/arch-version-design-template.md',
    'arch-version-scope-align-template.md': 'docs/templates/arch-version-scope-align-template.md',
}

def render_tpl(s, ctx):
    """把 how/ref 中的 {{var}} 替换为真实值；未知变量抛 KeyError 防止静默出错。
    变量名可含连字符（req-version 等），故用 [^{}]+ 匹配。
    （Python 同构实现，仅生成期校验模板可替换；宿主端正式实现为 JS renderTpl）"""
    import re
    def _sub(m):
        k = m.group(1).strip()
        if k not in ctx:
            raise KeyError(f'how 模板变量未提供: {{{k}}}')
        return ctx[k]
    return re.sub(r'\{\{([^{}]+)\}\}', _sub, s)

# 四流程节点定义（order, key, capability_id, 产出 productType 列表, 评审 reviewers）
# reviewers: [(role, capability_id)]
# 2026-08-24 对齐（flow-alignment.md）：business 增加「需求对齐」前置；「产品设计评审」独立节点（评审 product_doc）
FLOWS = {
    'business': [
        (0,  '待启动',        'decide-cap',       [], []),
        (1,  '需求对齐',       'ALIGN',            [], []),
        (2,  '产品设计',       'product-design',   ['product_doc'], []),
        (3,  '产品设计评审',   'pm-review',        [], [('pm','pm-review',''),('arch','arch-tech-review',''),('rd','rd-tech-review','be'),('fe','fe-tech-review','fe'),('user','final-approve','')]),
        (4,  'RD 技术设计',    'backend-design',   ['rd_design'], []),
        (5,  'RD 技术评审',    'rd-tech-review',   [], [('arch','arch-tech-review','be'),('rd','rd-tech-review','be'),('fe','fe-tech-review','fe'),('qa','qa-tech-review',''),('pm','pm-review',''),('user','final-approve','')]),
        (6,  'FE 技术设计',    'frontend-design',  ['fe_design'], []),
        (7,  'FE 设计评审',    'fe-tech-review',   [], [('qa','qa-tech-review','')]),
        (8,  '用例设计',       'case-design-cap',  ['case_design'], []),
        (9,  '用例设计评审',   'qa-tech-review',   [], [('rd','rd-tech-review','be'),('fe','fe-tech-review','fe')]),
        (10, 'RD 开发',       'backend-dev',      ['dev_notes','code_mr'], [('cr','code-review')]),
        (11, 'FE 开发',       'frontend-dev',     ['dev_notes','code_mr'], [('cr','code-review')]),
        (12, 'QA 验收',       'acceptance-test',  ['test_result','defect'], []),
        (13, '用户验收',       'final-approve',    [], [('user','final-approve')]),
        (14, '完成',          'DONE',             [], []),
    ],
    'tech': [
        (0, '待启动',         'decide-cap',       [], []),
        (1, '需求对齐',       'ALIGN',            [], []),
        (2, 'RD 技术设计',    'backend-design',   ['rd_design'], []),
        (3, 'RD 技术评审(按需)', 'rd-tech-review',  [], [('arch','arch-tech-review','be'),('rd','rd-tech-review','be'),('fe','fe-tech-review','fe'),('qa','qa-tech-review',''),('user','final-approve','')]),
        (4, 'FE 技术设计(按需)', 'frontend-design', ['fe_design'], []),
        (5, 'FE 技术评审(按需)', 'fe-tech-review',  [], [('qa','qa-tech-review','')]),
        (6, '用例设计',       'case-design-cap',  ['case_design'], []),
        (7, '用例设计评审',   'qa-tech-review',   [], [('rd','rd-tech-review','be'),('fe','fe-tech-review','fe')]),
        (8, '并行开发(RD)',   'backend-dev',      ['dev_notes','code_mr'], [('cr','code-review','')]),
        (9, '并行开发(FE)',   'frontend-dev',     ['dev_notes','code_mr'], [('cr','code-review','')]),
        (10, 'QA 验收',      'acceptance-test',  ['test_result','defect'], []),
        (11, '用户验收',      'final-approve',    [], [('user','final-approve','')]),
        (12, '完成',          'DONE',             [], []),
    ],
    'task': [
        (0, '待启动',         'decide-cap',       [], []),
        (1, '需求对齐',       'ALIGN',            [], []),
        (2, '技术设计',       'plan-cap',         ['plan'], []),
        (3, '技术评审',       'rd-tech-review',   [], [('arch','arch-tech-review',''),('user','final-approve','')]),
        (4, '开发',          'DEV_SET',         ['dev_notes','code_mr'], [('cr','code-review','')]),
        (5, 'QA 验收',       'acceptance-test',  ['test_result','defect'], []),
        (6, '用户验收',       'final-approve',    [], [('user','final-approve','')]),
        (7, '完成',          'DONE',             [], []),
    ],
    'plan': [
        (0, '待启动',         'decide-cap',       [], []),
        (1, '立项(需求对齐)',  'alignment-cap',    ['alignment'], []),
        (2, '立项评审',       'pm-review',        [], [('pm','pm-review',''),('user','final-approve','')]),
        (3, '调研',          'research-cap',     ['research'], []),
        (4, '调研评审',       'arch-tech-review', [], [('arch','arch-tech-review','')]),
        (5, '架构规划',       'arch-cap',         ['architecture'], []),
        (6, '架构规划评审',   'pm-review',        [], [('pm','pm-review',''),('user','final-approve','')]),
        (7, '创建需求·启动版本', 'decide-cap',      ['roadmap'], []),
        (8, '完成',          'DONE',             [], []),
    ],
}

# 动作库：capability_id -> [ {action_id, name, trigger, when, how, ref, out, tools} ]
ACTIONS = {
    # ---------- 产出型 ----------
    'product-design': [
        dict(aid='product-design.produce', name='产出产品设计', trigger='node_enter',
             when='节点进入（产品设计，owner=pm 执行产出组织）',
             how='读「product_doc」需求对齐结论(discussion) + 「architecture」版本架构 + 需求描述 → 按模板「prd-template.md」（问题陈述/目标与成功指标/范围与非范围/用户与场景/功能需求/验收标准）编写 → 自检 quality_gates → wb_add_artifact(「product_doc」) → 提出评审等待 hook/扫描推进',
             ref='「product_doc」; 「architecture」; 需求对齐结论',
             out=['product_doc'], tools='wb_my_task, wb_add_artifact, wb_comment, fs'),
        dict(aid='product-design.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='产品设计评审阶段，被评审者收到 open 评审意见时（评审计论中）',
             how='对被指出的评审意见 reply 澄清：说明设计意图/权衡/依据，引用对应章节（可修改「product_doc」对应位置并在回复中给出定位：行号/章节）；澄清在评审阶段完成，不进入打回',
             ref='query{assignee:实例化评审者集合(scope 决定),state:open}',
             out=['product_doc(可选修订)'], tools='wb_my_task, wb_reply, wb_resolve'),
        dict(aid='product-design.fix_on_review', name='按评审意见修订(打回·真问题修复)', trigger='reject_rolled_back',
             when='产品设计评审确认无法在评审阶段解决（User 裁决打回）→ 回退到产品设计节点（此时 open 均为已确认真问题，无澄清环节）',
             how='读评审节点的已确认真问题（text+anchor+thread）→ 逐条修订「product_doc」对应位置（resolve 带修复定位）→ 全部处理后=设计阶段完成 → 重新提评 → 回评审阶段（临时任务态视图不再展示）',
             ref='query{state:confirmed(真问题), 锚定被打回产物}; 评审节点评论',
             out=['product_doc'], tools='wb_my_task, wb_reply, wb_resolve, fs'),
    ],
    'backend-design': [
        dict(aid='backend-design.produce', name='产出RD技术设计', trigger='node_enter',
             when='节点进入（RD 技术设计）',
             how='读 product_doc（功能/AC）+ architecture（大方向）→ 按 backend-design-doc 模板（背景与动机/目标与非目标/总体方案设计/备选方案与取舍/接口设计(契约)/数据与存储设计/自测方案）逐节编写 → 自检（接口/数据对照 product_doc+architecture 一致）→ wb_add_artifact(rd_design) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='product_doc; architecture',
             out=['rd_design'], tools='wb_my_task, wb_add_artifact, wb_comment, fs'),
        dict(aid='backend-design.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='RD 技术评审节点有 assignee=rd 的 open 评论（arch/fe/qa/pm 各自意见）',
             how='query 过滤 assignee=rd & state=open → 读 text+anchor+thread → 按评审者分组核对（arch 技术路线/fe 接口契约/qa 可测性/pm 工期）→ 修订对应章节 或 wb_reply 澄清 → wb_resolve(note) → 复核',
             ref='query{assignee:rd,state:open}',
             out=['rd_design'], tools='wb_my_task, wb_reply, wb_resolve'),
        dict(aid='backend-design.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='RD 技术评审打回（decide reject）→ 回退到 RD 技术设计节点',
             how='读评审节点（RD 技术评审）的 open comment/issue（text+anchor+thread）→ 逐条修订 rd_design（接口/数据/自测按评审分组处理）→ resolve → 重新提评 → 新一轮评审',
             ref='query{state:open, 锚定被打回产物}; 评审节点评论',
             out=['rd_design'], tools='wb_my_task, wb_reply, wb_resolve, fs'),
    ],
    'frontend-design': [
        dict(aid='frontend-design.produce', name='产出FE技术设计', trigger='node_enter',
             when='节点进入（FE 技术设计）',
             how='读 product_doc（页面/交互）+ rd_design（接口契约）→ 按 frontend-design-doc 模板（背景与目标/页面与交互设计/组件设计/数据与状态管理/接口设计(前后端契约)/风险与验收）逐节编写 → 自检（接口消费与 rd 契约一致、关键路径可验证）→ wb_add_artifact(fe_design) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='product_doc; rd_design',
             out=['fe_design'], tools='wb_my_task, wb_add_artifact, wb_comment, fs'),
        dict(aid='frontend-design.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='FE 设计评审节点有 assignee=fe 的 open 评论（qa 意见）',
             how='query 过滤 assignee=fe & state=open → 读 text+anchor+thread → 修订（qa 评交互可测性/关键路径）→ wb_resolve(note) → 复核',
             ref='query{assignee:fe,state:open}',
             out=['fe_design'], tools='wb_my_task, wb_reply, wb_resolve'),
        dict(aid='frontend-design.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='FE 设计评审打回（decide reject）→ 回退到 FE 技术设计节点',
             how='读评审节点（FE 设计评审）的 open comment/issue（text+anchor+thread）→ 逐条修订 fe_design（页面/交互/接口对接按 qa 意见）→ resolve → 重新提评 → 新一轮评审',
             ref='query{state:open, 锚定被打回产物}; 评审节点评论',
             out=['fe_design'], tools='wb_my_task, wb_reply, wb_resolve, fs'),
    ],
    'case-design-cap': [
        dict(aid='case-design-cap.produce', name='产用例/联调设计', trigger='node_enter',
             when='节点进入（用例设计）',
             how='读 rd_design（后端口径）+ fe_design（前端口径）→ 按 qa-testcase-review 模板骨架（用例总览与覆盖矩阵/后端自测/联调用例/前端自测/联调用例/联调场景与数据准备/回归范围/验收用例映射AC）编写 → 自检（AC 逐条映射用例、联调场景可执行）→ wb_add_artifact(case_design) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='rd_design; fe_design',
             out=['case_design'], tools='wb_my_task, wb_add_artifact, wb_comment, fs'),
        dict(aid='case-design-cap.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='用例设计评审节点有 assignee=qa 的 open 评论（rd/fe 口径意见）',
             how='query 过滤 assignee=qa & state=open → 读 text+anchor+thread → 修订用例/口径 → wb_resolve(note) → 复核',
             ref='query{assignee:qa,state:open}',
             out=['case_design'], tools='wb_my_task, wb_reply, wb_resolve'),
        dict(aid='case-design-cap.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='用例设计评审打回（decide reject）→ 回退到用例设计节点',
             how='读评审节点（用例设计评审）的 open comment/issue（text+anchor+thread）→ 逐条修订 case_design（口径按 rd/fe 意见）→ resolve → 重新提评 → 新一轮评审',
             ref='query{state:open, 锚定被打回产物}; 评审节点评论',
             out=['case_design'], tools='wb_my_task, wb_reply, wb_resolve, fs'),
    ],
    'backend-dev': [
        dict(aid='backend-dev.dev', name='开发实现', trigger='node_enter',
             when='节点进入（RD 开发）',
             how='读 rd_design（接口/数据契约）+ product_doc（功能/AC）+ architecture（大方向）→ 按设计契约实现（不得自造接口），技术路线对齐架构 → 产出代码（待自测）',
             ref='rd_design; product_doc; architecture; feature_branch',
             out=['code_mr'], tools='wb_my_task, fs, git, 代码工具'),
        dict(aid='backend-dev.self_test', name='开发自测', trigger='dev_complete',
             when='dev 动作产出代码后，提交前',
             how='读 rd_design（自测方案）+ case_design（后端口径）→ 按 case 后端口径逐项自测 → 记录自测结果（场景→依据→结果→证据）→ 待写 dev_notes 对照表',
             ref='rd_design; case_design',
             out=[], tools='wb_my_task, 测试工具'),
        dict(aid='backend-dev.submit', name='登记提评', trigger='dev_complete',
             when='自测通过后，产生产物（推进由插件 hook 处理）',
             how='写《dev_notes》（dev-notes 模板：实现-设计对照表+自测记录+偏差与风险说明，路径由 mytask 按《dev_notes》渲染）→ wb_add_artifact(dev_notes, product_type=dev_notes) + wb_add_artifact(代码MR, product_type=code_mr) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='dev_notes(本节点产物)',
             out=['dev_notes','code_mr'], tools='wb_add_artifact, wb_comment, fs'),
        dict(aid='backend-dev.respond_to_cr', name='响应CR', trigger='reviewer_feedback(open)',
             when='RD 开发节点有 assignee=rd 且 reviewerRole=Code Reviewer 的 open 评论',
             how='query 过滤 reviewerRole=Code Reviewer & assignee=rd & state=open → 逐条读 text（要改什么）+ anchor（锚代码/文档哪行哪块）→ 读 thread 理解上下文 → 改代码 或 wb_reply 澄清（分歧）→ wb_resolve(note=修复说明) → CR 复核 confirm/reopen',
             ref='query{reviewerRole:Code Reviewer,assignee:rd,state:open}',
             out=['code_mr'], tools='wb_my_task, wb_reply, wb_resolve, 代码工具'),
    ],
    'frontend-dev': [
        dict(aid='frontend-dev.dev', name='前端开发实现', trigger='node_enter',
             when='节点进入（FE 开发）',
             how='读 fe_design（组件/状态/交互）+ rd_design（接口契约）+ product_doc（页面/AC）→ 按设计实现，接口消费与 rd 契约一致 → 产出代码（待自测）',
             ref='fe_design; rd_design; product_doc; feature_branch',
             out=['code_mr'], tools='wb_my_task, fs, git, 代码工具'),
        dict(aid='frontend-dev.self_test', name='前端自测', trigger='dev_complete',
             when='dev 动作产出代码后，提交前',
             how='读 fe_design（自测口径）+ case_design（前端口径）→ 按 case 前端口径自测（关键路径/交互/异常态）→ 记录结果',
             ref='fe_design; case_design',
             out=[], tools='wb_my_task, 测试工具'),
        dict(aid='frontend-dev.submit', name='登记提评', trigger='dev_complete',
             when='自测通过后，产生产物（推进由插件 hook 处理）',
             how='写《dev_notes》（dev-notes 模板）→ wb_add_artifact(dev_notes) + wb_add_artifact(代码MR, code_mr) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='dev_notes(本节点产物)',
             out=['dev_notes','code_mr'], tools='wb_add_artifact, wb_comment, fs'),
        dict(aid='frontend-dev.respond_to_cr', name='响应CR', trigger='reviewer_feedback(open)',
             when='FE 开发节点有 assignee=fe 且 reviewerRole=Code Reviewer 的 open 评论',
             how='query 过滤 reviewerRole=Code Reviewer & assignee=fe & state=open → 读 text+anchor+thread → 改代码 或澄清 → wb_resolve(note) → 复核',
             ref='query{reviewerRole:Code Reviewer,assignee:fe,state:open}',
             out=['code_mr'], tools='wb_my_task, wb_reply, wb_resolve'),
    ],
    'acceptance-test': [
        dict(aid='acceptance-test.produce', name='验收并登记缺陷', trigger='node_enter',
             when='节点进入（QA 验收）',
             how='读 product_doc(AC) + case_design(用例口径) + rd_design/fe_design(接口协议) + feature_branch(代码) → worktree 检出分支读真实代码 → 建测试数据/自动化用例（《test_result》对应测试资产：qa/data/ + qa/code/）→ 按 AC 验收 → 写《test_result》+ 发现缺陷写《defect》→ wb_add_artifact(test_result+defect) → 缺陷经 wb_comment(domain=rd|fe) 指派返工',
             ref='product_doc; case_design; rd_design; fe_design; feature_branch',
             out=['test_result','defect'], tools='wb_my_task, wb_add_artifact, 测试工具, git worktree'),
        dict(aid='acceptance-test.defect_followup', name='缺陷返工闭环', trigger='reviewer_feedback(open)',
             when='缺陷指派后 assignee 修复，等待 QA 复核',
             how='query 过滤 pending-confirm 评论（提评者=qa）→ 复核：confirm（通过）或 reopen（不通过→回滚重做）→ 重跑自动化回测（qa/code/）→ 复验闭环',
             ref='query{state:pending-confirm,提评者:qa}',
             out=['test_result'], tools='wb_confirm_review, wb_reopen, 测试工具'),
    ],
    'research-cap': [
        dict(aid='research-cap.produce', name='产出调研文档', trigger='node_enter',
             when='节点进入（调研）',
             how='读 alignment（立项范围）+ roadmap（上版本路线）→ 调研竞品/开源（GitHub 等）→ 按 research 模板（调研背景与目标/候选方案盘点/开源项目评估/候选方案对比/结论与建议）编写，结论附依据 → wb_add_artifact(research) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='alignment; roadmap',
             out=['research'], tools='wb_my_task, wb_add_artifact, web_search'),
        dict(aid='research-cap.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='调研评审打回 → 修订《research》',
             how='读评审节点的已确认真问题（text+anchor+thread）→ 逐条修订对应文档（resolve 带修复定位）→ 全部处理后=完成 → 重新提评 → 回评审阶段（临时任务态视图撤销）',
             ref='query{state:confirmed(真问题), 锚定被打回产物}; 评审节点评论',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, fs'),
        dict(aid='research-cap.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='调研评审阶段收到评审意见时（评审讨论中，未打回）',
             how='对被指出的评审意见 reply 澄清：说明设计意图/权衡/依据，引用对应章节（可修改文档并在回复中给出定位）；澄清在评审阶段完成',
             ref='query{assignee:当前实例化评审者,state:open}',
             out=[], tools='wb_my_task, wb_reply, wb_resolve'),
    ],
    'arch-cap': [
        dict(aid='arch-cap.produce', name='产出架构规划', trigger='node_enter',
             when='节点进入（架构规划）',
             how='读 research（调研建议）+ alignment（版本方向/范围）→ 按 arch-version-design 模板（执行摘要/版本背景与范围/架构设计原则与决策/整体架构/组件与模块设计/需求拆解/数据架构）编写 → wb_add_artifact(architecture) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='research; alignment',
             out=['architecture'], tools='wb_my_task, wb_add_artifact'),
        dict(aid='arch-cap.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='架构规划评审打回 → 修订《architecture》',
             how='读评审节点的已确认真问题（text+anchor+thread）→ 逐条修订对应文档（resolve 带修复定位）→ 全部处理后=完成 → 重新提评 → 回评审阶段（临时任务态视图撤销）',
             ref='query{state:confirmed(真问题), 锚定被打回产物}; 评审节点评论',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, fs'),
        dict(aid='arch-cap.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='架构规划评审阶段收到评审意见时（评审讨论中，未打回）',
             how='对被指出的评审意见 reply 澄清：说明设计意图/权衡/依据，引用对应章节（可修改文档并在回复中给出定位）；澄清在评审阶段完成',
             ref='query{assignee:当前实例化评审者,state:open}',
             out=[], tools='wb_my_task, wb_reply, wb_resolve'),
    ],
    'alignment-cap': [
        dict(aid='alignment-cap.produce', name='产出需求对齐文档', trigger='node_enter',
             when='节点进入（立项/需求对齐）',
             how='读版本需求池 + roadmap（上版本路线）→ 按 arch-version-scope-align 模板（版本主题与目标/需求对齐清单/范围边界/工作量与人力评估/版本排期/风险与待决问题）编写 → wb_add_artifact(alignment) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='roadmap; 需求池',
             out=['alignment'], tools='wb_my_task, wb_add_artifact, wb_list_requirements'),
        dict(aid='alignment-cap.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='立项评审打回 → 修订《alignment》',
             how='读评审节点的已确认真问题（text+anchor+thread）→ 逐条修订对应文档（resolve 带修复定位）→ 全部处理后=完成 → 重新提评 → 回评审阶段（临时任务态视图撤销）',
             ref='query{state:confirmed(真问题), 锚定被打回产物}; 评审节点评论',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, fs'),
        dict(aid='alignment-cap.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='立项评审阶段收到评审意见时（评审讨论中，未打回）',
             how='对被指出的评审意见 reply 澄清：说明设计意图/权衡/依据，引用对应章节（可修改文档并在回复中给出定位）；澄清在评审阶段完成',
             ref='query{assignee:当前实例化评审者,state:open}',
             out=[], tools='wb_my_task, wb_reply, wb_resolve'),
    ],
    'plan-cap': [
        dict(aid='plan-cap.produce', name='产出方案文档', trigger='node_enter',
             when='节点进入（方案评审，任务流程）',
             how='读任务说明（缺陷报告/需求描述）+ architecture（大方向）→ 按 plan 模板（问题定位与根因/方案设计/影响面与回归范围/验证方案/风险与回滚/分支约定）编写 → wb_add_artifact(plan) → 推进由插件 hook 自动处理（非 agent 主动提评）',
             ref='任务说明; architecture',
             out=['plan'], tools='wb_my_task, wb_add_artifact'),
        dict(aid='plan-cap.fix_on_review', name='按评审意见修订(打回)', trigger='reject_rolled_back',
             when='方案评审打回 → 修订《plan》',
             how='读评审节点的已确认真问题（text+anchor+thread）→ 逐条修订对应文档（resolve 带修复定位）→ 全部处理后=完成 → 重新提评 → 回评审阶段（临时任务态视图撤销）',
             ref='query{state:confirmed(真问题), 锚定被打回产物}; 评审节点评论',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, fs'),
        dict(aid='plan-cap.clarify', name='评审阶段澄清', trigger='reviewer_feedback(open)',
             when='方案评审阶段收到评审意见时（评审讨论中，未打回）',
             how='对被指出的评审意见 reply 澄清：说明设计意图/权衡/依据，引用对应章节（可修改文档并在回复中给出定位）；澄清在评审阶段完成',
             ref='query{assignee:当前实例化评审者,state:open}',
             out=[], tools='wb_my_task, wb_reply, wb_resolve'),
    ],
    # ---------- 评审型（统一协议）----------
    'pm-review': [
        dict(aid='pm-review.review', name='价值/范围评审+澄清质疑', trigger='node_review_pending',
             when='需求对齐/产品设计评审/RD技术评审待评审，PM 位',
             how='读 product_doc（目标与成功指标/范围与非范围）评价值/优先级/范围；对产品方案提出澄清质疑（向下游追问业务理解，与 User/产出者沟通），不评排期（agent 无排期）→ wb_comment(锚到对应章节, reviewerRole=pm) → wb_review pass/block',
             ref='product_doc; 需求对齐结论',
             out=[], tools='wb_my_task, wb_comment, wb_review'),
        dict(aid='pm-review.review_response', name='处理回复(澄清/修复复核)', trigger='issue_replied',
             when='评审阶段收到被评审者的回复（澄清或修复，含定位信息）',
             how='读取该 issue 的回复（澄清说明/修复定位：行号/章节）→ 基于当前文档对照审核：合理/已修 → confirm/resolve（闭环）；不成立 → 坚持 reopen（继续讨论或升级 User 拍板）',
             ref='issue 回复(含定位) + 当前文档对应位置',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, wb_confirm_review, wb_reopen'),
    ],
    'arch-tech-review': [
        dict(aid='arch-tech-review.review', name='架构视角评审', trigger='node_review_pending',
             when='需求评审/RD技术评审场景待评审，arch 位',
             how='读 rd_design/fe_design（技术方案）+ architecture（整体架构/架构决策）评技术路线/架构一致性/重构影响 → wb_comment(reviewerRole=arch) → wb_review pass/block',
             ref='rd_design; fe_design; architecture',
             out=[], tools='wb_my_task, wb_comment, wb_review'),
        dict(aid='arch-tech-review.review_response', name='处理回复(澄清/修复复核)', trigger='issue_replied',
             when='评审阶段收到被评审者的回复（澄清或修复，含定位信息）',
             how='读取该 issue 的回复（澄清说明/修复定位：行号/章节）→ 基于当前文档对照审核：合理/已修 → confirm/resolve（闭环）；不成立 → 坚持 reopen（继续讨论或升级 User 拍板）',
             ref='issue 回复(含定位) + 当前文档对应位置',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, wb_confirm_review, wb_reopen'),
    ],
    'rd-tech-review': [
        dict(aid='rd-tech-review.review', name='后端视角评审', trigger='node_review_pending',
             when='RD技术评审/用例设计评审待评审，rd 位',
             how='读 rd_design（总体方案/接口设计(契约)/数据与存储设计）评设计完整性/自测方案；读 case_design（后端口径）评后端口径 → wb_comment(reviewerRole=rd) → wb_review pass/block',
             ref='rd_design; case_design; product_doc',
             out=[], tools='wb_my_task, wb_comment, wb_review'),
        dict(aid='rd-tech-review.review_response', name='处理回复(澄清/修复复核)', trigger='issue_replied',
             when='评审阶段收到被评审者的回复（澄清或修复，含定位信息）',
             how='读取该 issue 的回复（澄清说明/修复定位：行号/章节）→ 基于当前文档对照审核：合理/已修 → confirm/resolve（闭环）；不成立 → 坚持 reopen（继续讨论或升级 User 拍板）',
             ref='issue 回复(含定位) + 当前文档对应位置',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, wb_confirm_review, wb_reopen'),
    ],
    'fe-tech-review': [
        dict(aid='fe-tech-review.review', name='前端视角评审', trigger='node_review_pending',
             when='RD技术评审/FE设计评审/用例设计评审待评审，fe 位',
             how='读 fe_design（接口对接）+ rd_design（接口契约）评接口可对接（不造契约）；读 case_design（前端口径）评前端自测口径 → wb_comment(reviewerRole=fe) → wb_review pass/block',
             ref='fe_design; rd_design; case_design',
             out=[], tools='wb_my_task, wb_comment, wb_review'),
        dict(aid='fe-tech-review.review_response', name='处理回复(澄清/修复复核)', trigger='issue_replied',
             when='评审阶段收到被评审者的回复（澄清或修复，含定位信息）',
             how='读取该 issue 的回复（澄清说明/修复定位：行号/章节）→ 基于当前文档对照审核：合理/已修 → confirm/resolve（闭环）；不成立 → 坚持 reopen（继续讨论或升级 User 拍板）',
             ref='issue 回复(含定位) + 当前文档对应位置',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, wb_confirm_review, wb_reopen'),
    ],
    'qa-tech-review': [
        dict(aid='qa-tech-review.review', name='测试视角评审', trigger='node_review_pending',
             when='RD技术评审/FE设计评审/用例设计评审待评审，qa 位',
             how='读 product_doc（AC/功能需求）评可测性；读 rd_design（接口设计）评接口可测性/规范性（幂等/错误码/边界）；读 case_design（用例覆盖/联调）评验收口径 → wb_comment(reviewerRole=qa) → wb_review pass/block',
             ref='product_doc; rd_design; fe_design; case_design',
             out=[], tools='wb_my_task, wb_comment, wb_review'),
        dict(aid='qa-tech-review.review_response', name='处理回复(澄清/修复复核)', trigger='issue_replied',
             when='评审阶段收到被评审者的回复（澄清或修复，含定位信息）',
             how='读取该 issue 的回复（澄清说明/修复定位：行号/章节）→ 基于当前文档对照审核：合理/已修 → confirm/resolve（闭环）；不成立 → 坚持 reopen（继续讨论或升级 User 拍板）',
             ref='issue 回复(含定位) + 当前文档对应位置',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, wb_confirm_review, wb_reopen'),
    ],
    'code-review': [
        dict(aid='code-review.review', name='代码评审', trigger='node_review_pending',
             when='开发节点待评审（有 MR 产出），cr 位',
             how='worktree 检出 feature_branch 读真实代码 → 对照 dev_notes（实现-设计对照表）核对设计落地（逐条：设计要点/来源文档/实现位置/一致与偏差）→ 按偏差与风险说明评回归风险 → wb_comment(锚到代码/对照表) → wb_review pass/block',
             ref='code_mr; dev_notes; rd_design; fe_design',
             out=[], tools='wb_my_task, git worktree, wb_comment, wb_review'),
        dict(aid='code-review.review_response', name='处理回复(澄清/修复/缺陷复核)', trigger='issue_replied',
             when='评审阶段收到被评审者的回复（澄清/修复/缺陷修复，含定位信息）——含 CR 提出的代码缺陷被修复后复核',
             how='读取该 issue 的回复（澄清说明/修复定位/缺陷修复说明：行号/章节）→ 基于当前文档或代码对照审核：合理/已修 → confirm/resolve（闭环）；不成立 → 坚持 reopen（继续讨论或升级 User 拍板）',
             ref='issue 回复(含定位) + 当前文档/代码对应位置',
             out=[], tools='wb_my_task, wb_reply, wb_resolve, wb_confirm_review, wb_reopen'),
    ],
    'final-approve': [
        dict(aid='final-approve.review', name='最终拍板', trigger='node_review_pending',
             when='用户验收节点待评审，user 位',
             how='读 test_result（AC 逐条核对结果/结论与遗留）+ defect（遗留/挂起缺陷）+ product_doc（目标）评目标达成/可交付 → wb_review pass/block → 通过则 wb_update_requirement(status=已完成)',
             ref='test_result; defect; product_doc',
             out=[], tools='wb_my_task, wb_review, wb_update_requirement'),
    ],
    'decide-cap': [
        dict(aid='decide-cap.decide', name='门禁决策', trigger='decide',
             when='全部评审者已通过且无 open 评论（或需打回）',
             how='汇总 reviewers[].state + open 评论 → 全通过且无 open → wb_decide pass（并行组 join 判定；后置节点放行）→ 否则 wb_decide reject(reason 必填, targetBranch 可选部分回滚) → 记录 last_decision',
             ref='reviewers[].state; open 评论',
             out=[], tools='wb_decide'),
    ],
}

# 每节点上要多展几行评审者（评审型能力：每 reviewers 一人一行）
# 节点级 scope 推断（arch_design §8.3-A2）：RD/FE 侧节点
# 注意：tech「并行开发」节点为全量（''，RD+FE 都在），不标单侧
def node_scope(key):
    if key == '并行开发':
        return ''   # tech 并行开发 = RD+FE 全量，无单侧 scope
    if 'RD 技术' in key or key == 'RD 开发':
        return 'be'
    if 'FE 技术' in key or key == 'FE 开发':
        return 'fe'
    return ''

def build_rows():
    rows = []
    for flow, nodes in FLOWS.items():
        for order, key, cap, out_pts, reviewers in nodes:
            if cap == 'DONE':
                # 流程结束节点：无能力/动作/产物，占位行说明流程终点
                rows.append(dict(flow=flow, order=order, node=key, cap='', role='', aid='', name='',
                                 trigger='', when=f'{key} 节点 · 流程终点（{flow} 流程全部节点已完成，无动作；由 Leader 确认收尾）',
                                 how='', ref='', out='', tools=''))
                continue
            if cap == 'ALIGN':
                # 需求对齐：business/tech = PM 主导；task = 执行者（开发者，按 scope fe/rd）向 User 澄清
                align_role = '执行者(fe/rd by scope)' if flow == 'task' else 'pm'
                align_name = '需求对齐(task执行者)' if flow == 'task' else '需求对齐(PM主导)'
                align_how = ('执行者（按 scope 指定 fe/rd）与 User 直接对齐目标/验收口径：澄清任务说明（如改页面/修 bug）'
                             if flow == 'task' else
                             'PM 向 User 直接提问澄清范围/目标/价值；User 也可向 PM 问业务理解，确认是否对齐；结论落 issues discussion，无文档产物')
                rows.append(dict(flow=flow, order=order, node=key, cap='(align)', role=align_role, aid='align.discuss', name=align_name,
                                 trigger='node_enter', when='节点进入',
                                 how=align_how,
                                 ref='需求描述(任务配置 scope)', out='-', tools='wb_my_task, wb_comment(discussion)'))
                for role, rcap in reviewers:
                    if rcap in ACTIONS:
                        for a in ACTIONS[rcap]:
                            rows.append(dict(flow=flow, order=order, node=key, cap=rcap, role=role, aid=a['aid'], name=a['name'],
                                             trigger=a['trigger'], when=f"{a['when']}（{role} 位·需求对齐）", how=a['how'], ref=a['ref'], out='-', tools=a['tools']))
                continue
            if cap == 'decide-cap':
                if key == '待启动':
                    # 占位行：待启动由 User 推进，无 action（User 动作不记录），留说明
                    rows.append(dict(flow=flow, order=order, node=key, cap='', role='user', aid='', name='',
                                     trigger='', when='User 确认投入资源后推进（User 动作不记 action，占位说明）',
                                     how='', ref='', out='', tools=''))
                elif key == '创建需求·启动版本':
                    rows.append(dict(flow=flow, order=order, node=key, cap=cap, role='leader', aid='decide-cap.create_reqs',
                                     name='创建需求·启动版本', trigger='node_enter', when='架构规划通过后',
                                     how='按架构拆解创建需求（查重：相似则修改原需求）→ 归入版本 → 版本待启动→进行中 → roadmap 产物登记',
                                     ref='architecture（拆解清单）', out=['roadmap'], tools='wb_create_requirement, wb_advance_version'))
                else:
                    rows.append(dict(flow=flow, order=order, node=key, cap=cap, role='leader', aid='decide-cap.decide',
                                     name='门禁决策', trigger='decide', when='全部评审者通过且无 open 评论',
                                     how=ACTIONS[cap][0]['how'], ref=ACTIONS[cap][0]['ref'], out='-', tools=ACTIONS[cap][0]['tools']))
                continue
            # 评审节点：无「发起评审」动作（推进=hook/定时扫描；评审者行承担）；owner 行不单独列出
            # 被评审者 clarify（评审阶段澄清）：挂在评审节点，角色=产出者
            REVIEWEES = {'产品设计评审':'pm','RD 技术评审':'rd','RD 技术评审(按需)':'rd','FE 设计评审':'fe',
                         'FE 技术评审(按需)':'fe','用例设计评审':'qa','方案评审':'executor','技术评审':'执行者(rd|fe by scope)',
                         '立项(需求对齐)':'arch','立项评审':'arch','架构规划':'arch','架构规划评审':'arch','调研':'inv','调研评审':'inv'}
            CLAR_CAPS = {'产品设计评审':'product-design','RD 技术评审':'backend-design','RD 技术评审(按需)':'backend-design',
                         'FE 设计评审':'frontend-design','FE 技术评审(按需)':'frontend-design','用例设计评审':'case-design-cap',
                         '方案评审':'plan-cap','技术评审':'plan-cap',
                         '立项(需求对齐)':'alignment-cap','立项评审':'alignment-cap','架构规划':'arch-cap','架构规划评审':'arch-cap','调研':'research-cap','调研评审':'research-cap'}
            if key in REVIEWEES and REVIEWEES[key] != '-' and not out_pts:
                clar_cap = CLAR_CAPS.get(key, '')
                if clar_cap in ACTIONS:
                    for a in ACTIONS[clar_cap]:
                        if a['aid'].endswith('.clarify'):
                            rows.append(dict(flow=flow, order=order, node=key, cap=clar_cap, role=REVIEWEES[key], aid=a['aid'], name=a['name'],
                                             trigger=a['trigger'], when=f"{a['when']}（{key}·{REVIEWEES[key]} 澄清）", how=a['how'], ref=a['ref'], out='-', tools=a['tools']))
            # 产出型：该能力全部动作（产出节点不含 clarify——澄清在评审阶段）
            if cap == 'DEV_SET':
                # task 开发节点：能力候选集（executor=rd→backend-dev / fe→frontend-dev），实例化期解析
                for a in ACTIONS['backend-dev']:
                    if a['aid'].endswith('.clarify'):
                        continue
                    rows.append(dict(flow=flow, order=order, node=key, cap='backend-dev|frontend-dev',
                                     role='执行者(rd|fe by scope)', aid=a['aid'], name=a['name'],
                                     trigger=a['trigger'],
                                     when=a['when'].replace('RD 开发节点','开发节点').replace('RD 开发','开发'),
                                     how=a['how'], ref=a['ref'], out=';'.join(a['out']), tools=a['tools']))
            elif cap in ACTIONS and out_pts:
                for a in ACTIONS[cap]:
                    if a['aid'].endswith('.clarify'):
                        continue  # clarify 已挂评审节点
                    rows.append(dict(flow=flow, order=order, node=key, cap=cap, role='-', aid=a['aid'], name=a['name'],
                                     trigger=a['trigger'], when=a['when'], how=a['how'], ref=a['ref'], out=';'.join(a['out']), tools=a['tools']))
            # 评审者（评审型能力：每 reviewers 一人一行；仅产出+评审合一节点跳过 rcap==cap 自评重复；评审节点 rcap==cap 正常展开）
            for item in reviewers:
                role, rcap = item[0], item[1]
                rscope = item[2] if len(item) > 2 else ''
                if rcap in ACTIONS and not (rcap == cap and out_pts):
                    for a in ACTIONS[rcap]:
                        scopenote = f"，scope={rscope} 时参加" if rscope else ''
                        rows.append(dict(flow=flow, order=order, node=key, cap=rcap, role=role, aid=a['aid'], name=a['name'],
                                         trigger=a['trigger'], when=f"{a['when']}（{role} 位{scopenote}）", how=a['how'], ref=a['ref'], out='-', tools=a['tools']))
    return rows

if __name__ == '__main__':
    rows = build_rows()
    # 产物标准名清单（ProductType + 模板文件）——how/ref/out 输出时用「」框起来
    PTYPES = ['product_doc','rd_design','fe_design','case_design','test_result','defect','dev_notes',
              'research','architecture','alignment','roadmap','code_mr','feature_branch',
              'prd-template.md','backend-design-doc-template.md','frontend-design-doc-template.md',
              'qa-testcase-review-template.md','qa-acceptance-template.md','qa-defect-management-template.md',
              'dev-notes-template.md','plan-template.md','research-open-source-tech-template.md',
              'arch-version-design-template.md','arch-version-scope-align-template.md']
    def bracket(s):
        """产物标识符统一展开为「《标准名》(模板路径)」（J7 how 模板化定稿）：
        - 有路径模板的 PTYPE（product_doc/rd_design/…）→ `《product_doc》(docs/{{req-version}}/{{req-name}}/prd/prd.md)`；
        - 模板文件（prd-template.md/…）→ `《prd-template.md》(docs/templates/prd-template.md)` 固定路径；
        - 无路径模板产物（code_mr/feature_branch）→ 保留「标准名」书名号（逻辑引用，非文件路径）。
        模板路径是 JS/Python 通用双花括号语法，代码加载后替换 {{req-version}}/{{req-name}}/{{side}} 即得真实相对路径。
        统一在展开项后补一个空格，避免与后续中文粘连；幂等（先去旧框再展开）。"""
        if s is None: return ''
        if isinstance(s, list): s = ';'.join(s)
        all_keys = list(PTYPE_TPL) + list(TPLFILE_TPL)
        for t in sorted(all_keys, key=len, reverse=True):
            s = s.replace('「'+t+'」', t)   # 去旧框（幂等基础）
        for t, tpl in sorted(PTYPE_TPL.items(), key=lambda kv: -len(kv[0])):
            s = s.replace(t, f'《{t}》({tpl}) ')   # 标准名(模板路径)
        for t, tpl in TPLFILE_TPL.items():
            s = s.replace(t, f'《{t}》({tpl}) ')   # 模板文件 → 固定路径
        for t in ('code_mr', 'feature_branch'):
            s = s.replace(t, f'「{t}」 ')          # 无路径产物保留书名号
        return s.rstrip()
    # 后处理：①节点级 scope 标注 ②user 位行清空动作列 ③角色列补 owner ④产物名加书名号
    OWNER_MAP = {'product-design':'pm','backend-design':'rd','frontend-design':'fe','case-design-cap':'qa',
                 'acceptance-test':'qa','research-cap':'inv','arch-cap':'arch','alignment-cap':'arch','plan-cap':'executor',
                 'backend-dev':'rd','frontend-dev':'fe'}
    for r in rows:
        ns = node_scope(r['node'])
        if ns:
            r['node'] = f"{r['node']}[{ns}]"
        # 产出型能力：角色=owner（能力挂哪个角色；task 流程的执行者动态 rd|fe by scope）
        if r['cap'] in OWNER_MAP and r['role'] == '-':
            if r['flow'] == 'task' and r['cap'] in ('backend-dev','frontend-dev','plan-cap'):
                r['role'] = '执行者(rd|fe by scope)'
            else:
                r['role'] = OWNER_MAP[r['cap']]
        # user 位：占位行（能力/动作列全留空——user 是拍板交互方，无 capability/action；行保留表示参与）
        if r['role'] == 'user':
            r['cap'] = ''; r['aid'] = ''; r['name'] = ''; r['trigger'] = ''; r['how'] = ''; r['ref'] = ''; r['out'] = ''; r['tools'] = ''
            r['when'] = f"{r['node'].split('[')[0]} 节点 · User 拍板/交互（占位，无 action；确认完毕由 User 显式完成）"
        # 产物标准名加书名号
        r['how'] = bracket(r['how'])
        r['ref'] = bracket(r['ref'])
        r['out'] = bracket(r['out'])
    print('总行数(节点×能力×动作):', len(rows))
    from collections import Counter
    print('按流程:', dict(Counter(r['flow'] for r in rows)))
    # 输出到 csv 暂存（供 xlsx 脚本读取）
    import csv, io
    with open('/tmp/rows.csv','w',newline='') as f:
        w = csv.writer(f)
        w.writerow(['流程','节点order','节点','能力','角色位','动作ID','动作名','trigger','when','how','参考','输出产物','工具'])
        for r in rows:
            w.writerow([r['flow'],r['order'],r['node'],r['cap'],r['role'],r['aid'],r['name'],r['trigger'],r['when'],r['how'],r['ref'],r['out'],r['tools']])
    print('已写 /tmp/rows.csv')