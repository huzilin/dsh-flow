-- =====================================================================
-- wbflow seed 基线（迁移 v2: seed_baseline）
-- 口径真相源（docs-tmp 定稿，2026-09-08）：
--   capability-model-final.md v3.1（能力模型 + 21 活跃能力）
--   capability-actions-final.md（动作→skill 挂载）
--   business-flow-derivation.md / flow-derivation-tech-task-plan.md（四流程节点配置）
-- 行数：roles 10 + capabilities 21 + flow_defs 4 + flow_def_nodes 40
--   （business 13 / tech 11 / task 7 / plan 9；工单文案「41」为 task C5 删除前陈旧计数，
--    按 docs-tmp 定稿为 40，缺口已在交付报告记录）
-- 幂等：全部 INSERT IGNORE，唯一键（role_id/capability_id/flow_defs_id/template_node_id）
--   命中即跳过；重入无害。
-- =====================================================================

-- ---------------------------------------------------------------------
-- roles 全集 10 行（note=角色工作内容注释，非 persona）
-- ---------------------------------------------------------------------
INSERT IGNORE INTO roles (role_id, name, note, created_at, updated_at) VALUES
('rd',       '后端研发',   '后端工程规范/接口数据设计/自测/开发完成检查（对照设计）；业务·技术需求的 rd 侧执行者', 0, 0),
('fe',       '前端研发',   '前端工程规范/组件状态/联调/自测/开发完成检查（对照设计）；业务·技术需求的 fe 侧执行者', 0, 0),
('qa',       '测试',       '联调 + 验收 + 缺陷判定（用例设计内含于 accept）；缺陷响应链=rd/fe clarify 确认归属 + 修复 diagnosing-bugs', 0, 0),
('arch',     '架构师',     '技术路线/架构一致把关；产品设计评审、技术设计评审、调研评审、FE 设计评审（架构位）产出者；plan 立项对齐与架构规划产出者', 0, 0),
('pm',       '产品经理',   '需求价值/优先级/范围/工期依赖把关；业务需求对齐与产品设计产出者', 0, 0),
('cr',       '代码评审',   '代码质量/设计落地/边界与回归风险（开发节点内嵌 CR 异化评审位，永不输出 rejected）', 0, 0),
('inv',      '调研员',     '竞品调研/引入 or 重写建议（结论附依据）', 0, 0),
('leader',   '团队负责人', 'plan D7 创建需求·启动版本（create_reqs 直执行，不起 subagent）', 0, 0),
('user',     '用户',       '需求真实性与最终拍板（user_gate 终审位；破坏性操作授权）', 0, 0),
('executor', '执行者',     'task 流程指定开发者（by scope=rd|fe 单执行者；C6 报障直连，无 QA 中转）', 0, 0);

-- ---------------------------------------------------------------------
-- capabilities 21 活跃（capability-actions-final §2；JSON 键以 rd/repo/payload.go Go struct 为单一口径）
-- inputs 只列 useThreads=true 条目；评审类动作产出=issue，outputProductTypes 空=正确语义
-- ---------------------------------------------------------------------
INSERT IGNORE INTO capabilities (capability_id, name, role_id, inputs, actions, products, created_at, updated_at) VALUES
('product-design', '产品设计', 'pm',
  '[{"productType":"brd","useThreads":true}]',
  '[{"actionId":"produce","name":"产出产品设计","on":"exec","skill":"to-prd","outputProductTypes":["product_doc"]},{"actionId":"revise","name":"修订产品设计","on":"exec","inline":"基于评审结论逐条修订〈产物〉对应章节 → 模板自检 → 重新提评","outputProductTypes":["product_doc"]},{"actionId":"clarify","name":"澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/{{req-name}}/prd/prd.md","demoPath":"docs/templates/prd-demo.md"}]',
  0, 0),
('backend-design', '后端技术设计', 'rd',
  '[{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"produce","name":"产出后端技术设计","on":"exec","skill":"to-rd-techdesign","outputProductTypes":["rd_design"]},{"actionId":"revise","name":"修订后端技术设计","on":"exec","inline":"基于评审结论逐条修订〈产物〉对应章节 → 模板自检 → 重新提评","outputProductTypes":["rd_design"]},{"actionId":"clarify","name":"澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/{{req-name}}/rd/design.md","demoPath":"docs/templates/backend-design-doc-demo.md"}]',
  0, 0),
('frontend-design', '前端技术设计', 'fe',
  '[{"productType":"product_doc","useThreads":true},{"productType":"rd_design","useThreads":true}]',
  '[{"actionId":"produce","name":"产出前端技术设计","on":"exec","skill":"to-fe-techdesign","outputProductTypes":["fe_design"]},{"actionId":"revise","name":"修订前端技术设计","on":"exec","inline":"基于评审结论逐条修订〈产物〉对应章节 → 模板自检 → 重新提评","outputProductTypes":["fe_design"]},{"actionId":"clarify","name":"澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/{{req-name}}/fe/design.md","demoPath":"docs/templates/frontend-design-doc-demo.md"}]',
  0, 0),
('research-cap', '竞品调研', 'inv',
  '[{"productType":"alignment","useThreads":true}]',
  '[{"actionId":"produce","name":"产出调研文档","on":"exec","skill":"research","outputProductTypes":["research"]},{"actionId":"revise","name":"修订调研文档","on":"exec","inline":"基于评审结论逐条修订〈产物〉对应章节 → 模板自检 → 重新提评","outputProductTypes":["research"]},{"actionId":"clarify","name":"澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/research.md","demoPath":"docs/templates/research-open-source-tech-demo.md"}]',
  0, 0),
('arch-cap', '架构规划', 'arch',
  '[{"productType":"research","useThreads":true}]',
  '[{"actionId":"produce","name":"产出架构规划","on":"exec","skill":"to-arch","outputProductTypes":["architecture"]},{"actionId":"revise","name":"修订架构规划","on":"exec","inline":"基于评审结论逐条修订〈产物〉对应章节 → 模板自检 → 重新提评","outputProductTypes":["architecture"]},{"actionId":"clarify","name":"澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/arch_design.md","demoPath":"docs/templates/arch-version-design-demo.md"}]',
  0, 0),
('backend-dev', '后端开发', 'rd',
  '[{"productType":"rd_design","useThreads":true},{"productType":"test_result","useThreads":true}]',
  '[{"actionId":"dev","name":"开发实现","on":"exec","skill":"implement-techdesign","outputProductTypes":["dev_notes","code_mr"]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]},{"actionId":"fix_on_review","name":"评审修复","on":"respond","inline":"读 CR 意见 → 逐条修复 → resolve → 重新进入复核（轮内闭环，不打回、不重入）","outputProductTypes":[]},{"actionId":"diagnosing-bugs","name":"缺陷定位修复","on":"exec","skill":"diagnosing-bugs","outputProductTypes":["dev_notes","code_mr"]},{"actionId":"clarify","name":"缺陷确认澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/{{req-name}}/rd/dev.md"}]',
  0, 0),
('frontend-dev', '前端开发', 'fe',
  '[{"productType":"fe_design","useThreads":true},{"productType":"test_result","useThreads":true}]',
  '[{"actionId":"dev","name":"开发实现","on":"exec","skill":"implement-techdesign","outputProductTypes":["dev_notes","code_mr"]},{"actionId":"submit","name":"登记请评","on":"exec","inline":"产物 wb_add_artifact 登记 → wb_comment 请评；登记齐、请评已发即完成","outputProductTypes":[]},{"actionId":"fix_on_review","name":"评审修复","on":"respond","inline":"读 CR 意见 → 逐条修复 → resolve → 重新进入复核（轮内闭环，不打回、不重入）","outputProductTypes":[]},{"actionId":"diagnosing-bugs","name":"缺陷定位修复","on":"exec","skill":"diagnosing-bugs","outputProductTypes":["dev_notes","code_mr"]},{"actionId":"clarify","name":"缺陷确认澄清","on":"respond","inline":"读 assignee=本角色 且 open 意见/缺陷 → 逐条说明意图/权衡/依据或确认/争议归属（引定位），缺陷性意见修订产物 → resolve","outputProductTypes":[]}]',
  '[{"docPath":"docs/{{req-version}}/{{req-name}}/fe/dev.md"}]',
  0, 0),
('arch.prd-review', '产品设计评审（架构视角）', 'arch',
  '[{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"prd-review-arch-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('rd.prd-review', '产品设计评审（后端视角）', 'rd',
  '[{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"prd-review-rd-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('fe.prd-review', '产品设计评审（前端视角）', 'fe',
  '[{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"prd-review-fe-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('arch.rd-design-review', '技术设计评审（架构视角）', 'arch',
  '[{"productType":"rd_design","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"rd-design-review-arch-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('fe.rd-design-review', '技术设计评审（前端视角）', 'fe',
  '[{"productType":"rd_design","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"rd-design-review-fe-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('qa.rd-design-review', '技术设计评审（测试视角）', 'qa',
  '[{"productType":"rd_design","useThreads":true},{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"rd-design-review-qa-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('pm.rd-design-review', '技术设计评审（产品视角）', 'pm',
  '[{"productType":"rd_design","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"rd-design-review-pm-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('arch.fe-design-review', 'FE 设计评审（架构视角）', 'arch',
  '[{"productType":"fe_design","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"fe-design-review-arch-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('qa.fe-design-review', 'FE 设计评审（测试视角）', 'qa',
  '[{"productType":"fe_design","useThreads":true},{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"fe-design-review-qa-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('arch.research-review', '调研评审（架构视角）', 'arch',
  '[{"productType":"research","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"research-review-arch-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('pm.architecture-review', '架构规划评审（产品视角）', 'pm',
  '[{"productType":"architecture","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"architecture-review-pm-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('code-review', '代码评审', 'cr',
  '[{"productType":"dev_notes","useThreads":true},{"productType":"code_mr","useThreads":true}]',
  '[{"actionId":"review","name":"评审表态","on":"review","skill":"code-review-skill","outputProductTypes":[]},{"actionId":"confirm","name":"复核","on":"respond","inline":"读本人 pending-confirm 意见 → 核对澄清/修订 → wb_confirm_review 或 wb_reopen(附理由)","outputProductTypes":[]}]',
  '[]',
  0, 0),
('acceptance-test', '验收与缺陷管理', 'qa',
  '[{"productType":"product_doc","useThreads":true}]',
  '[{"actionId":"accept","name":"验收","on":"exec","skill":"qa-acceptance","outputProductTypes":["test_result","defect"]},{"actionId":"retest","name":"复验重测","on":"exec","inline":"针对重新提测的代码重新执行测试 → 核对是否修复 → 更新 defect 文档（已解决标记/新增缺陷登记并分派）","outputProductTypes":["test_result","defect"]}]',
  '[{"docPath":"docs/{{req-version}}/{{req-name}}/qa/test.md","demoPath":"docs/templates/qa-acceptance-demo.md"},{"docPath":"docs/{{req-version}}/{{req-name}}/qa/defect.md","demoPath":"docs/templates/qa-defect-management-demo.md"}]',
  0, 0),
('decide-cap', '规划决策', 'leader',
  '[]',
  '[{"actionId":"create_reqs","name":"创建需求·启动版本","on":"exec","skill":"to-tickets","outputProductTypes":[]}]',
  '[]',
  0, 0);

-- ---------------------------------------------------------------------
-- flow_defs 4 行（scope_options 按 spec §0 注释 + data-tools-final §1.2）
-- ---------------------------------------------------------------------
INSERT IGNORE INTO flow_defs (flow_defs_id, scope_options, created_at, updated_at) VALUES
('business', '["full","rd","fe"]', 0, 0),
('tech',     '["full","rd","fe"]', 0, 0),
('task',     '["rd","fe"]',        0, 0),
('plan',     '[]',                 0, 0);

-- ---------------------------------------------------------------------
-- flow_def_nodes 40 行（business 13 / tech 11 / task 7 / plan 9；docs-tmp 定稿逐节点配置）
-- created_at/updated_at=0 占位：seed 为基线常量数据，由迁移账本承载版本语义
-- ---------------------------------------------------------------------
INSERT IGNORE INTO flow_def_nodes (flow_defs_id, template_node_id, node_order, node_key, node_type, gate, user_gate, scope_expr, reviewers, node_actions, expected_outputs, created_at, updated_at) VALUES
-- ===== business 13 节点 =====
('business', 'bus-00',  1,  '待启动',       'gate',    0, 1, '',        '[]', '[]', '[]', 0, 0),
('business', 'bus-01',  2,  '需求对齐',     'align',   0, 1, '',        '[]', '[{"kind":"node","actionId":"align.opening","on":"entry","executor":"program","executorRole":"pm","inline":"渲染节点开场白并等待用户指令（内置强制等待）"},{"kind":"node","actionId":"to-brd","on":"exec","executor":"subagent","executorRole":"pm","skill":"to-brd"}]', '[{"productType":"brd","label":"《业务需求文档》","kind":"document"}]', 0, 0),
('business', 'bus-02',  3,  '产品设计',     'produce', 0, 0, '',        '[]', '[{"kind":"capability","capabilityId":"product-design","actions":["produce","revise"]}]', '[{"productType":"product_doc","label":"《产品设计文档》","kind":"document"}]', 0, 0),
('business', 'bus-03',  4,  '产品设计评审', 'review',  1, 1, '',        '[{"role":"arch","capabilityId":"arch.prd-review"},{"role":"rd","capabilityId":"rd.prd-review","scopeExpr":"rd|full"},{"role":"fe","capabilityId":"fe.prd-review","scopeExpr":"full"}]', '[{"kind":"capability","capabilityId":"product-design","actions":["clarify"]}]', '[]', 0, 0),
('business', 'bus-04',  5,  'RD 技术设计',  'produce', 0, 0, 'rd|full', '[]', '[{"kind":"capability","capabilityId":"backend-design","actions":["produce","revise"]}]', '[{"productType":"rd_design","label":"《RD 技术设计》","kind":"document"}]', 0, 0),
('business', 'bus-05',  6,  'RD 技术评审',  'review',  1, 1, 'rd|full', '[{"role":"arch","capabilityId":"arch.rd-design-review"},{"role":"fe","capabilityId":"fe.rd-design-review","scopeExpr":"full"},{"role":"qa","capabilityId":"qa.rd-design-review"},{"role":"pm","capabilityId":"pm.rd-design-review"}]', '[{"kind":"capability","capabilityId":"backend-design","actions":["clarify"]}]', '[]', 0, 0),
('business', 'bus-06',  7,  'FE 技术设计',  'produce', 0, 0, 'fe|full', '[]', '[{"kind":"capability","capabilityId":"frontend-design","actions":["produce","revise"]}]', '[{"productType":"fe_design","label":"《FE 技术设计》","kind":"document"}]', 0, 0),
('business', 'bus-07',  8,  'FE 设计评审',  'review',  1, 0, 'fe|full', '[{"role":"qa","capabilityId":"qa.fe-design-review"}]', '[{"kind":"capability","capabilityId":"frontend-design","actions":["clarify"]}]', '[]', 0, 0),
('business', 'bus-08',  9,  'RD 开发',      'produce', 1, 0, 'rd|full', '[{"role":"cr","capabilityId":"code-review"}]', '[{"kind":"capability","capabilityId":"backend-dev","actions":["dev","diagnosing-bugs","fix_on_review"]}]', '[{"productType":"dev_notes","label":"《RD 开发说明》","kind":"document"},{"productType":"code_mr","label":"《RD 代码 MR》","kind":"link"}]', 0, 0),
('business', 'bus-09',  10, 'FE 开发',      'produce', 1, 0, 'fe|full', '[{"role":"cr","capabilityId":"code-review"}]', '[{"kind":"capability","capabilityId":"frontend-dev","actions":["dev","diagnosing-bugs","fix_on_review"]}]', '[{"productType":"dev_notes","label":"《FE 开发说明》","kind":"document"},{"productType":"code_mr","label":"《FE 代码 MR》","kind":"link"}]', 0, 0),
('business', 'bus-10',  11, 'QA 验收（含联调）', 'produce', 1, 0, 'full', '[]', '[{"kind":"capability","capabilityId":"acceptance-test","actions":["accept","retest"]},{"kind":"capability","capabilityId":"backend-dev","actions":["clarify"]},{"kind":"capability","capabilityId":"frontend-dev","actions":["clarify"]}]', '[{"productType":"test_result","label":"《测试用例与验收记录》","kind":"document"},{"productType":"defect","label":"《缺陷文档》","kind":"document"}]', 0, 0),
('business', 'bus-11',  12, '用户验收',     'gate',    1, 1, '',        '[]', '[]', '[]', 0, 0),
('business', 'bus-12',  13, '完成',         'gate',    0, 0, '',        '[]', '[]', '[]', 0, 0),
-- ===== tech 11 节点 =====
('tech', 'tec-b0',   1,  '待启动',              'gate',    0, 1, '',        '[]', '[]', '[]', 0, 0),
('tech', 'tec-b1',   2,  '需求对齐',            'align',   0, 1, '',        '[]', '[{"kind":"node","actionId":"align.opening","on":"entry","executor":"program","executorRole":"pm","inline":"渲染节点开场白并等待用户指令（内置强制等待）"},{"kind":"node","actionId":"to-brd","on":"exec","executor":"subagent","executorRole":"pm","skill":"to-brd"}]', '[{"productType":"brd","label":"《业务需求文档》","kind":"document"}]', 0, 0),
('tech', 'tec-b2',   3,  'RD 技术设计',         'produce', 0, 0, 'rd|full', '[]', '[{"kind":"capability","capabilityId":"backend-design","actions":["produce","revise"]}]', '[{"productType":"rd_design","label":"《RD 技术设计》","kind":"document"}]', 0, 0),
('tech', 'tec-b3',   4,  'RD 技术评审（按需）', 'review',  1, 1, 'rd|full', '[{"role":"arch","capabilityId":"arch.rd-design-review"},{"role":"fe","capabilityId":"fe.rd-design-review","scopeExpr":"full"},{"role":"qa","capabilityId":"qa.rd-design-review"}]', '[{"kind":"capability","capabilityId":"backend-design","actions":["clarify"]}]', '[]', 0, 0),
('tech', 'tec-b4',   5,  'FE 技术设计（按需）', 'produce', 0, 0, 'fe|full', '[]', '[{"kind":"capability","capabilityId":"frontend-design","actions":["produce","revise"]}]', '[{"productType":"fe_design","label":"《FE 技术设计》","kind":"document"}]', 0, 0),
('tech', 'tec-b5',   6,  'FE 技术评审（按需）', 'review',  1, 0, 'fe|full', '[{"role":"qa","capabilityId":"qa.fe-design-review"}]', '[{"kind":"capability","capabilityId":"frontend-design","actions":["clarify"]}]', '[]', 0, 0),
('tech', 'tec-b8a',  7,  'RD 开发',             'produce', 1, 0, 'rd|full', '[{"role":"cr","capabilityId":"code-review"}]', '[{"kind":"capability","capabilityId":"backend-dev","actions":["dev","diagnosing-bugs","fix_on_review"]}]', '[{"productType":"dev_notes","label":"《RD 开发说明》","kind":"document"},{"productType":"code_mr","label":"《RD 代码 MR》","kind":"link"}]', 0, 0),
('tech', 'tec-b8b',  8,  'FE 开发',             'produce', 1, 0, 'fe|full', '[{"role":"cr","capabilityId":"code-review"}]', '[{"kind":"capability","capabilityId":"frontend-dev","actions":["dev","diagnosing-bugs","fix_on_review"]}]', '[{"productType":"dev_notes","label":"《FE 开发说明》","kind":"document"},{"productType":"code_mr","label":"《FE 代码 MR》","kind":"link"}]', 0, 0),
('tech', 'tec-b9',   9,  'QA 验收（含联调）',   'produce', 1, 0, 'full',    '[]', '[{"kind":"capability","capabilityId":"acceptance-test","actions":["accept","retest"]},{"kind":"capability","capabilityId":"backend-dev","actions":["clarify"]},{"kind":"capability","capabilityId":"frontend-dev","actions":["clarify"]}]', '[{"productType":"test_result","label":"《测试用例与验收记录》","kind":"document"},{"productType":"defect","label":"《缺陷文档》","kind":"document"}]', 0, 0),
('tech', 'tec-b10',  10, '用户验收',            'gate',    1, 1, '',        '[]', '[]', '[]', 0, 0),
('tech', 'tec-b11',  11, '完成',                'gate',    0, 0, '',        '[]', '[]', '[]', 0, 0),
-- ===== task 7 节点（无 QA；C5 已删——工单「41 节点」为删除前计数） =====
('task', 'tsk-c0',  1,  '待启动',   'gate',    0, 1, '',      '[]', '[]', '[]', 0, 0),
('task', 'tsk-c1',  2,  '需求对齐', 'align',   0, 1, '',      '[]', '[{"kind":"node","actionId":"align.opening","on":"entry","executor":"program","executorRole":"executor","inline":"渲染节点开场白并等待用户指令（内置强制等待）"}]', '[]', 0, 0),
('task', 'tsk-c2',  3,  '技术设计', 'produce', 0, 0, 'rd|fe', '[]', '[{"kind":"capability","capabilityId":"backend-design","actions":["produce","revise"],"scopeExpr":"rd"},{"kind":"capability","capabilityId":"frontend-design","actions":["produce","revise"],"scopeExpr":"fe"}]', '[{"productType":"rd_design","label":"《RD 技术设计》","kind":"document"},{"productType":"fe_design","label":"《FE 技术设计》","kind":"document"}]', 0, 0),
('task', 'tsk-c3',  4,  '技术评审', 'review',  1, 1, 'rd|fe', '[{"role":"arch","capabilityId":"arch.rd-design-review","scopeExpr":"rd"},{"role":"arch","capabilityId":"arch.fe-design-review","scopeExpr":"fe"}]', '[{"kind":"capability","capabilityId":"backend-design","actions":["clarify"],"scopeExpr":"rd"},{"kind":"capability","capabilityId":"frontend-design","actions":["clarify"],"scopeExpr":"fe"}]', '[]', 0, 0),
('task', 'tsk-c4',  5,  '开发',     'produce', 1, 0, 'rd|fe', '[{"role":"cr","capabilityId":"code-review"}]', '[{"kind":"capability","capabilityId":"backend-dev","actions":["dev","diagnosing-bugs","fix_on_review"],"scopeExpr":"rd"},{"kind":"capability","capabilityId":"frontend-dev","actions":["dev","diagnosing-bugs","fix_on_review"],"scopeExpr":"fe"}]', '[{"productType":"dev_notes","label":"《开发说明》","kind":"document"},{"productType":"code_mr","label":"《代码 MR》","kind":"link"}]', 0, 0),
('task', 'tsk-c6',  6,  '用户验收', 'gate',    1, 1, '',      '[]', '[]', '[]', 0, 0),
('task', 'tsk-c7',  7,  '完成',     'gate',    0, 0, '',      '[]', '[]', '[]', 0, 0),
-- ===== plan 9 节点（无 scope） =====
('plan', 'pln-d0', 1, '待启动',           'gate',    0, 1, '', '[]', '[]', '[]', 0, 0),
('plan', 'pln-d1', 2, '立项需求对齐',     'align',   0, 1, '', '[]', '[{"kind":"node","actionId":"align.opening","on":"entry","executor":"program","executorRole":"arch","inline":"渲染节点开场白并等待用户指令（内置强制等待）"},{"kind":"node","actionId":"to-alignment","on":"exec","executor":"subagent","executorRole":"arch","skill":"to-alignment"}]', '[{"productType":"alignment","label":"《需求对齐文档》","kind":"document"}]', 0, 0),
('plan', 'pln-d2', 3, '立项评审',         'gate',    1, 1, '', '[]', '[]', '[]', 0, 0),
('plan', 'pln-d3', 4, '调研',             'produce', 0, 0, '', '[]', '[{"kind":"capability","capabilityId":"research-cap","actions":["produce","revise"]}]', '[{"productType":"research","label":"《调研文档》","kind":"document"}]', 0, 0),
('plan', 'pln-d4', 5, '调研评审',         'review',  1, 0, '', '[{"role":"arch","capabilityId":"arch.research-review"}]', '[{"kind":"capability","capabilityId":"research-cap","actions":["clarify"]}]', '[]', 0, 0),
('plan', 'pln-d5', 6, '架构规划',         'produce', 0, 0, '', '[]', '[{"kind":"capability","capabilityId":"arch-cap","actions":["produce","revise"]}]', '[{"productType":"architecture","label":"《架构规划》","kind":"document"}]', 0, 0),
('plan', 'pln-d6', 7, '架构规划评审',     'review',  1, 1, '', '[{"role":"pm","capabilityId":"pm.architecture-review"}]', '[{"kind":"capability","capabilityId":"arch-cap","actions":["clarify"]}]', '[]', 0, 0),
('plan', 'pln-d7', 8, '创建需求·启动版本', 'produce', 0, 0, '', '[]', '[{"kind":"capability","capabilityId":"decide-cap","actions":["create_reqs"]}]', '[]', 0, 0),
('plan', 'pln-d8', 9, '完成',             'gate',    0, 0, '', '[]', '[]', '[]', 0, 0);
