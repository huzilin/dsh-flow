-- =====================================================================
-- 团队工作台 · 种子数据（seed.sql）
-- 依据：docs/design/m8-flow-simplify.md §3.4（能力模型）
--       §3.6.6（document_templates 固化范围）
--       §10（节点-能力绑定模型重设计 v0.24，v0.28 落库）
--       docs/design/m6-flow-templates.md（tech/task/plan 节点定义）
--       docs/design/m8-doc-templates.md（12 模板 structure/quality_gates，
--         v0.23 起章节骨架/质量门槛外置 docs/templates/*.md 文件承载）
-- 模型（v0.15 修正，User 拍板）：
--   * roles → capabilities 1 对多：capabilities.role_id 唯一归属，
--     不再有 role_capabilities 关联表（能力不共享）
--   * 评审能力按角色拆分：tech-review → arch-tech-review / rd-tech-review /
--     fe-tech-review / qa-tech-review（同一评审节点上各角色看各自维度）
-- v0.23 收缩（User 拍板）：capabilities 删除 reviewer_focus/inputs/quality_gates 三列；
--   新增 template_path/demo_path（指向 docs/templates/xxx-template.md / xxx-demo.md）；
--   how 承载能力本体（含评审维度/自检要点）；依赖文档由派发时 task.context 承载；
--   质量门槛由模板文件承载。产出型 12 项中已建档 9 组模板，其余 3 项
--   （backend-dev/frontend-dev/plan-cap）暂 NULL，模板固化后回填。
-- v0.28（§10 节点-能力绑定模型，User 拍板落库）：
--   * A 点：capabilities.doc_path/template_path/demo_path 三列 → 单列 products
--     （JSON: [{doc_path, template_path, demo_path}]，一能力可多产物；评审/决策类 '[]'）
--   * v0.29（产出标准权威迁 skill，User 拍板 D4，semantica 47c53ed9）：products 项删 template_path，
--     现值 [{doc_path, demo_path}]——标准指针不入 DB，产出/评审端经产出 skill 寻址（docs/templates/skills/）
--   * QA 能力合并（§10.1-3）：acceptance-test + defect-management → 一个能力
--     「验收与缺陷管理」（acceptance-test）挂 2 产物（qa/test.md + qa/defect.md）；
--     defect-management 行删除，how 并入。能力 20 → 19 项。
--   * B 点：flow_template_nodes 加 mode 列（produce|align）+ capability_id 可空；
--     tech①/task① 需求对齐 = mode='align'、capability_id=NULL（无产物自主对齐，
--     需求不清晰→与 User IM 澄清；清晰→认知确认→再开发；结论落 issues discussion，
--     wb_my_task 组装链把全流程讨论一并打包给 subagent）
--   * C 点：reviewers 统一为 JSON: [{role, capability_id, scope?}]（focus 移除，
--     评审维度已在评审能力自身 how）；leader 移出 reviewers（决策者≠评审表态者）
-- 执行：sqlite3 workbench.db < seed.sql
-- =====================================================================

-- ---------------------------------------------------------------------
-- roles 9 行（note 角色工作内容注释，非子 agent persona；摘自 m8 §2.4.4/§2.5；created_at/updated_at 走 DEFAULT）
-- ---------------------------------------------------------------------
INSERT INTO roles (role_id, name, note) VALUES
('rd',          '后端研发',   '后端工程规范/接口数据设计/自测/开发完成检查（对照设计）'),
('fe',          '前端研发',   '前端工程规范/组件状态/联调/自测/开发完成检查（对照设计）'),
('cr',          '代码评审',   '评审维度：代码质量/设计落地/边界与回归风险；结论格式统一'),
('arch',   '架构师',     '技术路线/架构一致/产品方案把关（产品设计、架构规划、需求对齐产出者）'),
('pm',          '产品经理',   '需求价值/优先级/范围/工期依赖把关'),
('qa',          '测试',       '用例设计/验收/缺陷根因判定（domain 归属）'),
('inv','调研员',     '竞品调研/引入 or 重写建议（结论附依据）'),
('leader',      '团队负责人', '编排推进：分派、协调、门禁决策（wb_decide）'),
('user',        '用户',       '需求真实性与最终拍板（破坏性操作授权）');

-- ---------------------------------------------------------------------
-- capabilities 19 项（role_id 唯一归属；how 为能力本体 ≥2 步：读输入→产出/评审→自检→登记）
-- 产出型 11 项（products 有 1–2 项产物）；评审/决策型 8 项 products='[]'
-- products 每项（v0.29）：doc_path + demo_path（填充示例，docs/templates/xxx-demo.md）；
--   无 template_path（D4：产出标准以产出 skill 为权威 docs/templates/skills/，DB 不存标准指针）
--   有 skill 4 类：to-brd / to-prd / to-rd-techdesign / to-fe-techdesign；无 skill 暂由
--   docs/templates/*-template.md 承载（skill 待建，索引 docs/templates/README.md）；
--   demo_path 为 null 即示例未固化（backend-dev/frontend-dev/plan-cap 3 项）
-- QA 合并（§10.1-3）：acceptance-test 挂 2 产物（qa/test.md + qa/defect.md）；defect-management 已删除
-- ---------------------------------------------------------------------
INSERT INTO capabilities (capability_id, name, role_id, how, products) VALUES
('acceptance-test', '验收与缺陷管理', 'qa',
  '["读输入：prd/prd.md（AC）+ qa/cases.md（用例）+ rd/design.md + fe/design.md（接口协议）+ MR/featureBranch（代码）",
    "按 AC + 接口协议创建测试数据构造脚本与种子（qa/data/，幂等可重建）+ 自动化/回归测试用例（qa/code/，含运行命令、依赖、输出）——随需求入库供回测与修复后复验",
    "执行自动化 + 按用例验收：产出《测试用例与验收记录》（qa-acceptance 模板，含 AC 核对/联调/结论 + §3 测试资产必填）→ qa/test.md",
    "发现缺陷：登记《缺陷文档》（qa-defect-management 模板：现象/复现/预期/实际 + 归属判定 domain=rd|fe，缺陷分析依赖代码）→ qa/defect.md",
    "缺陷返工：wb_comment(domain=rd|fe) 自动路由 assignee → 返工 → resolve → 重跑自动化回测复验 → confirm/reopen 闭环（roundCount 拉锯）",
    "登记产物 wb_add_artifact(test.md + qa/data/ + qa/code/ link, nodeId=当前节点) → 缺陷闭环后提评/推进"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/qa/test.md","demo_path":"docs/templates/qa-acceptance-demo.md"},
    {"doc_path":"docs/<版本号>/<需求名>/qa/defect.md","demo_path":"docs/templates/qa-defect-management-demo.md"}]'),
('alignment-cap', '需求对齐', 'arch',
  '["读输入：待启动版本需求池 + roadmap.md（上版本路线）",
    "产出《需求对齐文档》alignment：版本方向/需求池梳理/范围界定",
    "自检：价值评估可溯源、分歧有结论、后续动作明确",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=PM)"]',
  '[{"doc_path":"docs/<版本号>/align.md","demo_path":"docs/templates/arch-version-scope-align-demo.md"}]'),
('arch-tech-review', '技术方案评审（架构视角）', 'arch',
  '["读输入：被评审设计文档（rd-design/fe-design/case-design）",
    "按 focus 评审：技术路线/架构一致/重构影响/调研建议合理性（评审维度见 how 结尾）",
    "按产出 skill（to-rd-techdesign）自检节核对：接口/数据设计与架构方向约束一致",
    "评论 wb_comment(reviewerRole=arch) → wb_review 表态",
    "复核：意见闭环后确认（resolve）"]',
  '[]'),
('arch-cap', '架构规划', 'arch',
  '["读输入：research.md（调研建议）+ align.md（版本方向/范围）",
    "产出《架构规划》architecture：版本目标/能力规划/需求拆解",
    "自检：拆解覆盖版本目标、架构约束可供 rd/fe-design 遵循",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=Leader/PM)"]',
  '[{"doc_path":"docs/<版本号>/arch_design.md","demo_path":"docs/templates/arch-version-design-demo.md"}]'),
('backend-design', '后端技术设计', 'rd',
  '["读输入：prd/prd.md（产品设计）+ 版本 arch_design.md（大方向）",
    "产出《RD 技术设计》：按产出 skill（to-rd-techdesign）标准逐节撰写，章节骨架/门禁/准出以 skill 为权威",
    "按 skill 自检节自检：接口/数据设计对照 prd/prd.md+arch_design.md 一致，准出 blocker 不过不提评",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=评审归属)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/rd/design.md","demo_path":"docs/templates/backend-design-doc-demo.md"}]'),
('backend-dev', '后端开发', 'rd',
  '["读输入：rd/design.md + qa/cases.md（用例口径）+ featureBranch",
    "开发实现 + 自测：按 dev-notes 模板记录实现-设计对照表",
    "自检质量门禁：设计要点逐项落地、边界处理、回归风险说明",
    "登记产物 wb_add_artifact(MR link, nodeId=当前节点) → 提评 wb_comment(to=cr)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/rd/dev.md","demo_path":null}]'),
('case-design-cap', '用例/联调设计', 'qa',
  '["读输入：rd/design.md + fe/design.md（前后端口径）",
    "产出《用例/联调设计》：按模板 qa-testcase-review 章节骨架逐节编写",
    "自检质量门禁：AC 逐条映射用例、联调场景可执行",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=RD/FE)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/qa/cases.md","demo_path":"docs/templates/qa-testcase-review-demo.md"}]'),
('code-review', '代码评审', 'cr',
  '["读输入：MR link + dev-notes（实现-设计对照表）+ 对应设计文档",
    "按质量门禁评审：代码质量/设计落地/边界与回归风险",
    "评论 wb_comment(锚到代码/对照表) → wb_review 表态",
    "复核：作者 resolve 后 confirm 或 reopen"]',
  '[]'),
('decide-cap', '门禁决策', 'leader',
  '["读输入：评审表态汇总（reviewers[].state）+ 未解决评论",
    "门禁判定：全部通过且无 open 评论 → pass；否则 reject（并行组可 targetBranch 部分回滚）",
    "wb_decide 推进状态机（join 判定并行组）",
    "记录决策 last_decision"]',
  '[]'),
('fe-tech-review', '技术方案评审（前端视角）', 'fe',
  '["读输入：被评审设计文档（rd-design/fe-design/case-design）",
    "按 focus 评审：接口契约可对接/前端口径（各审各的，评审维度见 how 结尾）",
    "按产出 skill（to-fe-techdesign）自检节逐条核对：准出 blocker（逐字段对齐 rd 契约/序列化口径随 rd）不过即打回",
    "评论 wb_comment(reviewerRole=fe) → wb_review 表态",
    "复核：意见闭环后确认（resolve）"]',
  '[]'),
('final-approve', '最终拍板', 'user',
  '["读输入：qa/test.md（测试用例与验收记录，含验收结论）+ qa/defect.md",
    "核对需求目标是否达成、是否可交付",
    "wb_review 表态（通过/不通过）→ 收尾或打回",
    "通过后 wb_update_requirement(status=已完成)"]',
  '[]'),
('frontend-design', '前端技术设计', 'fe',
  '["读输入：prd/prd.md（产品方案）+ rd/design.md（接口契约）",
    "产出《FE 技术设计》：按产出 skill（to-fe-techdesign）标准逐节撰写，章节骨架/门禁/准出以 skill 为权威",
    "按 skill 自检节自检：接口消费与 rd 契约逐字段一致、关键路径可验证，准出 blocker 不过不提评",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=QA)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/fe/design.md","demo_path":"docs/templates/frontend-design-doc-demo.md"}]'),
('frontend-dev', '前端开发', 'fe',
  '["读输入：fe/design.md + qa/cases.md（用例口径）+ featureBranch",
    "开发实现 + 自测：按 dev-notes 模板记录实现-设计对照表",
    "自检质量门禁：接口消费与契约一致、关键路径可用、回归风险说明",
    "登记产物 wb_add_artifact(MR link, nodeId=当前节点) → 提评 wb_comment(to=cr)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/fe/dev.md","demo_path":null}]'),
('plan-cap', '方案设计（任务）', 'executor',
  '["读输入：任务说明（缺陷报告/需求描述）+ arch_design.md",
    "产出《方案文档》plan：根因/方案/影响面/验证方案/分支约定",
    "自检：方案可落地、验证可执行、分支名符合 fix/<需求id>-<主题>",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=QA)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/plan.md","demo_path":null}]'),
('pm-review', '价值/排期评审', 'pm',
  '["读输入：prd/prd.md（价值/优先级/范围）+ rd/design.md（工期依赖）",
    "按 focus 评审：需求价值/优先级/范围/工期依赖（各审各的，评审维度见 how 结尾）",
    "按产出 skill（to-prd）自检节核对：目标/范围/AC 无歧义，下游可无歧义消费",
    "评论 wb_comment(reviewerRole=pm) → wb_review 表态",
    "复核：意见闭环后确认（resolve）"]',
  '[]'),
('product-design', '产品设计', 'arch',
  '["读输入：需求池需求描述 + 版本架构文档 arch_design.md（大方向）",
    "产出《产品设计文档》：按产出 skill（to-prd）标准逐节撰写，章节骨架/门禁/准出以 skill 为权威",
    "按 skill 自检节自检：AC 可验证、范围与架构方向不冲突，下游可无歧义消费",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=PM)"]',
  '[{"doc_path":"docs/<版本号>/<需求名>/prd/prd.md","demo_path":"docs/templates/prd-demo.md"}]'),
('qa-tech-review', '技术方案评审（测试视角）', 'qa',
  '["读输入：被评审设计文档（rd-design/fe-design/case-design）",
    "按 focus 评审：可测性/联调口径/用例覆盖（各审各的，评审维度见 how 结尾）",
    "按产出 skill（to-rd-techdesign / to-fe-techdesign）自检节核对：数据可测性、边界态处理说明齐备",
    "评论 wb_comment(reviewerRole=qa) → wb_review 表态",
    "复核：意见闭环后确认（resolve）"]',
  '[]'),
('rd-tech-review', '技术方案评审（后端视角）', 'rd',
  '["读输入：被评审设计文档（rd-design/case-design）+ prd/prd.md",
    "按 focus 评审：设计完整性/自测方案/后端口径（各审各的，评审维度见 how 结尾）",
    "按产出 skill（to-rd-techdesign）自检节逐条核对：准出 blocker（接口与契约真相源一致/序列化口径一致/覆盖无遗漏）不过即打回",
    "评论 wb_comment(reviewerRole=rd) → wb_review 表态",
    "复核：意见闭环后确认（resolve）"]',
  '[]'),
('research-cap', '竞品调研', 'inv',
  '["读输入：align.md（立项范围）+ roadmap.md（上版本路线）",
    "调研竞品/开源方案：GitHub 等，产出《调研文档》research",
    "自检：候选覆盖 align 范围、结论附依据、建议明确（引入/重写）",
    "登记产物 wb_add_artifact(nodeId=当前节点) → 提评 wb_comment(to=Architect)"]',
  '[{"doc_path":"docs/<版本号>/research.md","demo_path":"docs/templates/research-open-source-tech-demo.md"}]');

-- ---------------------------------------------------------------------
-- flow_templates 4 种（flow_templates_id + scope_options；节点拓扑见 flow_template_nodes；时间戳走 DEFAULT）
-- ---------------------------------------------------------------------
INSERT INTO flow_templates (flow_templates_id, scope_options) VALUES
('business', '["full","fe-only","be-only"]'),
('tech', '["be-only","full"]'),
('task', '[]'),
('plan', '[]');

-- ---------------------------------------------------------------------
-- flow_template_nodes
-- capability_id 语义：本节点执行者（role_id）使用的能力；
--   NULL = 无产出能力（mode='align' 需求对齐节点：不绑产出能力，subagent 自主决策对齐）
-- mode：'produce'=有产物常规节点｜'align'=无产物自主对齐节点（tech/task 需求对齐）
-- reviewers 语义（v0.28，C 点）：JSON [{role, capability_id, scope?}] 评审者集合，各审各的——
--   role 决定谁审（wb_review 表态者），capability_id 决定按哪个评审能力审（how 内含评审维度）；
--   leader 不进 reviewers（决策者 wb_decide，非评审表态者）；focus 字段已移除
-- 业务需求 13 节点（待启动 + 12，并行开发 RD/FE 各占一行 = 12 业务节点），按 m8 §3.2 nodeDefs
-- scope：''=全量｜'be'=仅后端侧｜'fe'=仅前端侧（实例化时按 requirement.scope 过滤，见 m8 ★scope 裁剪 A2/A3；reviewers 条目 scope 同语义）
-- ---------------------------------------------------------------------
INSERT INTO flow_template_nodes
  (flow_templates_id, node_order, key, role_id, capability_id, mode, gate, branch, parallel_group, scope, reviewers, expected_outputs) VALUES
('business', 0,  '待启动',        'leader',     'decide-cap',      'produce', 0, '', '',
  '', '[]',
  '[]'),
('business', 1,  '产品设计',      'arch',  'product-design',  'produce', 0, '', '',
  '', '[]',
  '[{"capability_id":"product-design","label":"《产品设计文档》","kind":"document"}]'),
('business', 2,  '需求评审',      'pm',         'pm-review',       'produce', 1, '', '',
  '', '[{"role":"pm","capability_id":"pm-review"},{"role":"arch","capability_id":"arch-tech-review"},{"role":"user","capability_id":"final-approve"}]',
  '[]'),
('business', 3,  'RD 技术设计',   'rd',         'backend-design',  'produce', 0, '', '',
  'be', '[]',
  '[{"capability_id":"backend-design","label":"《RD 技术设计》","kind":"document"}]'),
('business', 4,  'RD 技术评审',   'rd',         'rd-tech-review',  'produce', 1, '', '',
  'be',
  '[{"role":"arch","capability_id":"arch-tech-review","scope":"be"},{"role":"rd","capability_id":"rd-tech-review","scope":"be"},{"role":"fe","capability_id":"fe-tech-review","scope":"fe"},{"role":"qa","capability_id":"qa-tech-review"},{"role":"pm","capability_id":"pm-review"},{"role":"user","capability_id":"final-approve"}]',
  '[]'),
('business', 5,  'FE 技术设计',   'fe',         'frontend-design','produce', 0, '', '',
  'fe', '[]',
  '[{"capability_id":"frontend-design","label":"《FE 技术设计》","kind":"document"}]'),
('business', 6,  'FE 设计评审',   'fe',         'fe-tech-review',  'produce', 1, '', '',
  'fe', '[{"role":"qa","capability_id":"qa-tech-review"}]',
  '[]'),
('business', 7,  '用例设计',      'qa',         'case-design-cap', 'produce', 0, '', '',
  '', '[]',
  '[{"capability_id":"case-design-cap","label":"《用例/联调设计》","kind":"document"}]'),
('business', 8,  '用例设计评审',  'qa',         'qa-tech-review',  'produce', 1, '', '',
  '',
  '[{"role":"rd","capability_id":"rd-tech-review","scope":"be"},{"role":"fe","capability_id":"fe-tech-review","scope":"fe"}]',
  '[]'),
('business', 9,  'RD 开发',       'rd',         'backend-dev',     'produce', 1, 'rd', 'dev-1',
  'be', '[{"role":"cr","capability_id":"code-review"}]',
  '[{"capability_id":"backend-dev","label":"《RD 开发说明》","kind":"document"}]'),
('business', 10, 'FE 开发',       'fe',         'frontend-dev',    'produce', 1, 'fe', 'dev-1',
  'fe', '[{"role":"cr","capability_id":"code-review"}]',
  '[{"capability_id":"frontend-dev","label":"《FE 开发说明》","kind":"document"}]'),
('business', 11, 'QA 验收',       'qa',         'acceptance-test', 'produce', 1, '', '',
  '', '[{"role":"qa","capability_id":"acceptance-test"}]',
  '[{"capability_id":"acceptance-test","label":"《测试用例与验收记录》","kind":"document"},
    {"capability_id":"acceptance-test","label":"《缺陷文档》","kind":"document"}]'),
('business', 12, '用户验收',      'user',       'final-approve',   'produce', 1, '', '',
  '', '[{"role":"user","capability_id":"final-approve"}]',
  '[]');

-- 技术需求 9 节点（待启动 + 8，按 m6 §二；m8 §4.2 标题"7 节点"为笔误，按 m6 9 行落）
-- ①需求对齐：mode='align'、capability_id=NULL（无产物自主对齐；对齐结论落 issues discussion；
--   gate 保留 = User 拍板确认对齐结论；rd/arch 参与走 IM 讨论不入 reviewers）
INSERT INTO flow_template_nodes
  (flow_templates_id, node_order, key, role_id, capability_id, mode, gate, branch, parallel_group, scope, reviewers, expected_outputs) VALUES
('tech', 0, '待启动',            'leader',   'decide-cap',      'produce', 0, '', '',
  '', '[]',
  '[]'),
('tech', 1, '需求对齐',          'rd',       NULL,              'align',   1, '', '',
  '', '[{"role":"user","capability_id":"final-approve"}]',
  '[]'),
('tech', 2, 'RD 技术设计',       'rd',       'backend-design',  'produce', 0, '', '',
  'be', '[]',
  '[{"capability_id":"backend-design","label":"《RD 技术设计》","kind":"document"}]'),
('tech', 3, 'RD 技术评审（按需）','rd',      'rd-tech-review',  'produce', 1, '', '',
  'be',
  '[{"role":"arch","capability_id":"arch-tech-review","scope":"be"},{"role":"rd","capability_id":"rd-tech-review","scope":"be"},{"role":"fe","capability_id":"fe-tech-review","scope":"fe"},{"role":"qa","capability_id":"qa-tech-review"},{"role":"user","capability_id":"final-approve"}]',
  '[]'),
('tech', 4, 'FE 技术设计（按需）','fe',     'frontend-design', 'produce', 0, '', '',
  'fe', '[]',
  '[{"capability_id":"frontend-design","label":"《FE 技术设计》","kind":"document"}]'),
('tech', 5, '用例设计',          'qa',       'case-design-cap', 'produce', 1, '', '',
  '',
  '[{"role":"rd","capability_id":"rd-tech-review","scope":"be"},{"role":"fe","capability_id":"fe-tech-review","scope":"fe"}]',
  '[{"capability_id":"case-design-cap","label":"《用例设计》","kind":"document"}]'),
('tech', 6, '并行开发',          'rd',       'backend-dev',     'produce', 1, 'rd', 'dev-1',
  '', '[{"role":"cr","capability_id":"code-review"}]',
  '[{"capability_id":"backend-dev","label":"《RD 开发说明》","kind":"document"}]'),
('tech', 7, 'QA 验收',           'qa',       'acceptance-test', 'produce', 1, '', '',
  '', '[{"role":"qa","capability_id":"acceptance-test"}]',
  '[{"capability_id":"acceptance-test","label":"《测试用例与验收记录》","kind":"document"},
    {"capability_id":"acceptance-test","label":"《缺陷文档》","kind":"document"}]'),
('tech', 8, '用户验收',          'user',     'final-approve',   'produce', 1, '', '',
  '', '[{"role":"user","capability_id":"final-approve"}]',
  '[]');

-- 任务 6 节点（待启动 + 5，按 m6 §三；owner=executor 执行者，User/Leader 指定；无 scope 裁剪，全 ''）
-- ①需求对齐：mode='align'、capability_id=NULL（原误用 plan-cap 已修正；任务场景无审批者，gate=0）
INSERT INTO flow_template_nodes
  (flow_templates_id, node_order, key, role_id, capability_id, mode, gate, branch, parallel_group, scope, reviewers, expected_outputs) VALUES
('task', 0, '待启动',   'leader',   'decide-cap',      'produce', 0, '', '',
  '', '[]',
  '[]'),
('task', 1, '需求对齐', 'executor', NULL,              'align',   0, '', '',
  '', '[]',
  '[]'),
('task', 2, '方案评审', 'executor', 'plan-cap',        'produce', 1, '', '',
  '', '[{"role":"qa","capability_id":"qa-tech-review"},{"role":"user","capability_id":"final-approve"}]',
  '[{"capability_id":"plan-cap","label":"《方案文档》","kind":"document"}]'),
('task', 3, '开发',     'executor', 'backend-dev',     'produce', 1, '', '',
  '', '[{"role":"cr","capability_id":"code-review"}]',
  '[{"capability_id":"backend-dev","label":"《开发说明》","kind":"document"}]'),
('task', 4, 'QA 验收',  'qa',       'acceptance-test', 'produce', 1, '', '',
  '', '[{"role":"qa","capability_id":"acceptance-test"}]',
  '[{"capability_id":"acceptance-test","label":"《测试用例与验收记录》","kind":"document"}]'),
('task', 5, '用户验收', 'user',     'final-approve',   'produce', 1, '', '',
  '', '[{"role":"user","capability_id":"final-approve"}]',
  '[]');

-- 版本规划 5 节点（待启动 + 4，按 m6 §四；无 scope 裁剪，全 ''）
-- ③架构规划：reviewers 移除 leader（决策者≠评审表态者，其「范围/资源把关」体现在 wb_decide 决策 reason）
INSERT INTO flow_template_nodes
  (flow_templates_id, node_order, key, role_id, capability_id, mode, gate, branch, parallel_group, scope, reviewers, expected_outputs) VALUES
('plan', 0, '待启动',              'leader',      'decide-cap',      'produce', 0, '', '',
  '', '[]',
  '[]'),
('plan', 1, '立项（需求对齐）',    'arch',   'alignment-cap',   'produce', 1, '', '',
  '', '[{"role":"pm","capability_id":"pm-review"},{"role":"user","capability_id":"final-approve"}]',
  '[{"capability_id":"alignment-cap","label":"《需求对齐文档》","kind":"document"}]'),
('plan', 2, '调研',                'inv','research-cap',    'produce', 1, '', '',
  '', '[{"role":"arch","capability_id":"arch-tech-review"}]',
  '[{"capability_id":"research-cap","label":"《调研文档》","kind":"document"}]'),
('plan', 3, '架构规划',            'arch',   'arch-cap','produce', 1, '', '',
  '', '[{"role":"pm","capability_id":"pm-review"},{"role":"user","capability_id":"final-approve"}]',
  '[{"capability_id":"arch-cap","label":"《架构规划》","kind":"document"}]'),
('plan', 4, '创建需求·启动版本',   'leader',      'decide-cap',      'produce', 0, '', '',
  '', '[]',
  '[]');

-- ---------------------------------------------------------------------
-- meta：schema 版本（v0.28：capabilities.products 单列（一能力多产物）+ QA 合并（acceptance-test 挂 2 产物，defect-management 删除）
--   + flow_template_nodes.mode 列（produce|align）与 capability_id 可空（需求对齐无产物节点）
--   + reviewers 统一 [{role,capability_id,scope?}]（focus 移除、leader 移出））
-- ---------------------------------------------------------------------
INSERT INTO meta (meta_key, meta_value) VALUES ('schema_version', '29');