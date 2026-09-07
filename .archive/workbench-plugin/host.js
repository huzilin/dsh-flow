// Host 半：团队项目工作台（wbflow-1）— pkg-24（agent 工具注册版）
// 职责：读写 <workspace>/.workbench/ 领域数据，暴露 RPC 供 Client 调用；
//       并将全部 wb.* 领域操作注册为模型工具（agent 可直接驱动工作台）。
// 本文件内容 = cordis_define 的 code.host（Plugin 函数体，返回 Cordis Plugin）。
// 变更记录：
//   pkg-3   +wb.listVersions
//   pkg-5   loadRequirement 缺 flow 返回 null（lossless 守卫）
//   pkg-7   wb.decide 加 flow.__err 防护
//   pkg-8   +wb.createRequirement / wb.updateRequirement（字段 diff + history 审计）/ wb.deleteRequirement
//   pkg-9   +wb.createVersion / wb.updateVersion / wb.advanceVersion / wb.deprecateVersion / wb.deleteVersion
//   pkg-10  废弃不联动取消需求：有需求引用禁止废弃（blocked+refCount）
//   pkg-11  移除 wb.deleteVersion（版本不支持删除，废弃是唯一移除手段）
//   pkg-13  listVersions 归一化缺失 state 为「待启动」
//   pkg-24  重构：16 个 wb.* handler 提取为 op* 领域函数，RPC 与 16 个模型工具共用；
//           harness.defineTool + harness.registerTool(ctx, def)（ctx.effect 包裹，Fiber 生命周期）
//   pkg-35  缺口补齐 S1：+wb_add_artifact（产物登记）/ +wb_review（评审者表态，reviewers[].state）；
//           wb_comment 支持 author/role/reviewerRole；wb_decide pass 联动拦截「未通过」评审者（18 工具）
//   pkg-36  缺口补齐 S4：inbox（JSON 文件版，schema 与 sqlite 表一致）+ wb_list_inbox；
//           wb_comment/wb_review/wb_decide 触发写收件箱（19 工具）；sqlite 迁移留 preset 环境验证
//   pkg-37  M7 状态机：并行组 join 判定（组内全分支已通过才放行下一节点）；
//           wb_decide reject 支持 targetBranch=rd|fe 部分回滚（仅回滚指定分支，另一分支保持已通过）
//   pkg-38  修复部分回滚：按 targetBranch 定位分支所在并行组回滚（不依赖打回节点自身 parallelGroup）
//   pkg-39  修复 wb_comment 收件语义：新增 to 参数（收件提醒人，默认 reviewerRole 或 Leader）；
//           reviewerRole 只作评审归属（门禁/override 匹配），不再兼任收件人（CR 评审评论可 to=作者）
//   pkg-40  opResolve 增加复核通知：评论解决后 pushInbox to=评论作者（提评者），提醒复核；
//           persona to 语义泛化为「当前节点评审归属角色」（开发→CR、FE 技术设计→QA 等场景）
//   pkg-41  +wb_my_task（22 工具）：确定性任务上下文（roleObjective/behaviors/onFeedback +
//           reviewers/artifactsToReview/dependencyDocs/outputs + sessionsByNode —— 每个 node 的
//           全部会话（含锚定在文档上的会话）按 node 分组打包，每条带 nodeId/nodeKey 归属）；
//           行为协议表 ROLE_PROTOCOLS（m8 §2.4.4）内联、产出映射 NODE_OUTPUTS、
//           align 判定=key 含「需求对齐」（sqlite 迁移后切 mode 字段）
return {
  apply(ctx) {
    const fs = ctx.get('fs')
    const sp = ctx.get('sandboxPolicy')
    const PROJECT = '/Users/huzilin/workdir/test-flow'
    const WB = PROJECT + '/.workbench'
    const VER_STATES = { '待启动': '进行中', '进行中': '已发布' }
    function writePolicy() { var mode = 'workspace-write'; if (sp) { try { mode = sp.resolve().mode || 'workspace-write' } catch (e) {} } return { mode: mode, workspaceRoot: PROJECT } }
    function readJson(rel) { if (!fs) return Promise.resolve(undefined); return fs.resolve(WB + '/' + rel).then(function (t) { return fs.stat(t).then(function (info) { if (!info) return undefined; return fs.readText(t).then(function (text) { return JSON.parse(text) }) }) }).catch(function (e) { return { __err: String(e) } }) }
    function writeJson(rel, data) { if (!fs) return Promise.resolve({ ok: false, err: 'no fs' }); return fs.resolve(WB + '/' + rel).then(function (t) { return fs.writeText(t, JSON.stringify(data, null, 2), undefined, undefined, writePolicy()).then(function () { return { ok: true, err: null } }) }).catch(function (e) { return { ok: false, err: String(e) } }) }
    function listRequirements() { return readJson('requirements.json').then(function (r) { return Array.isArray(r) ? r : [] }) }
    function listVersions() {
      return readJson('versions.json').then(function (v) {
        if (!Array.isArray(v)) return []
        v.forEach(function (x) { if (!x.state) x.state = '待启动' })
        return v
      })
    }
    function loadRequirement(reqId) { return readJson('requirements.json').then(function (reqs) { if (!reqs || !Array.isArray(reqs) || reqs.__err) return null; const req = reqs.filter(function (r) { return r.id === reqId })[0]; if (!req) return null; return Promise.all([readJson(reqId + '/flow.json'), readJson(reqId + '/artifacts.json'), readJson(reqId + '/comments.json')]).then(function (parts) { const flow = parts[0] && !parts[0].__err ? parts[0] : null; if (!flow) return null; return { requirement: req, flow: flow, artifacts: parts[1] || [], comments: parts[2] || [] } }) }) }
    function openComments(comments, nodeId) { return (comments || []).filter(function (c) { return c.nodeId === nodeId && c.state === 'open' && c.type !== 'notification' && c.type !== 'discussion' }).length }
    function pendingComments(comments, nodeId) { return (comments || []).filter(function (c) { return c.nodeId === nodeId && c.state === 'pending-confirm' }).length }

    // ---- wb_my_task（pkg-41，m8 §11）：确定性任务上下文供给 ----
    // 行为协议表：m8 §2.4.4 全角色（roleObjective/behaviors/onFeedback）；executor 为任务流程动态执行者占位
    const ROLE_PROTOCOLS = {
      leader: { roleObjective: '编排推进：分派、协调、门禁决策', behaviors: ['wb_list_* 全局视图', '派活给子 agent（reqId+role）', 'wb_decide 推进/打回', '破坏性操作需 User 指令'], onFeedback: '收到返工报告 → 重新派活' },
      arch: { roleObjective: '产品设计/技术方案/版本规划把关', behaviors: ['产《产品设计》《架构规划》《需求对齐文档》', '评审 focus=技术路线/可行性/架构一致', 'wb_review 表态'], onFeedback: '设计意见被质疑 → 澄清设计意图 → 修订文档 → resolve' },
      pm: { roleObjective: '需求价值/优先级/范围/排期把关', behaviors: ['评审 focus=价值/优先级/范围/工期依赖', 'wb_comment 提评', 'wb_review 表态', '版本规划参与'], onFeedback: '评审意见被质疑 → 澄清 → 确认或修订' },
      rd: { roleObjective: '后端设计/开发/自测', behaviors: ['产《RD 技术设计》《RD 开发说明》+MR', '评审 focus=设计完整性/自测/后端口径', 'wb_comment', 'wb_review'], onFeedback: '收评审/CR/缺陷反馈 → 澄清 → 修订 → resolve → 复核' },
      fe: { roleObjective: '前端设计/开发/自测', behaviors: ['产《FE 技术设计》《FE 开发说明》+MR', '评审 focus=接口契约/前端口径', 'wb_comment', 'wb_review'], onFeedback: '收评审/CR/缺陷反馈 → 澄清 → 修订 → resolve → 复核' },
      qa: { roleObjective: '用例设计/验收/缺陷闭环', behaviors: ['产《用例设计》《测试用例与验收记录》《缺陷文档》', '评审 focus=可测性/验收', 'L3 根因判定（domain=rd|fe）', 'wb_review'], onFeedback: '用例设计意见被质疑 → 澄清 → 修订 → resolve；缺陷返工后重跑验收' },
      cr: { roleObjective: '代码质量/设计落地/回归风险把关', behaviors: ['评审 focus=质量/设计落地/回归', '评论 → 表态 → 复核（confirm/reopen）'], onFeedback: '评审结论被作者质疑 → thread 澄清 → 确认或坚持（reopen）' },
      inv: { roleObjective: '竞品调研/建议', behaviors: ['产《调研文档》', '按需求对齐文档调研 → 结论（引入/重写）'], onFeedback: '调研意见被质疑 → 补充依据 → 修订 → resolve' },
      user: { roleObjective: '需求真实性与最终拍板', behaviors: ['各 gate 节点拍板（wb_review 表态）', '最终验收', '破坏性操作授权'], onFeedback: '（最终裁决者，无被评审物）' },
      executor: { roleObjective: '任务流程动态执行者：对齐/方案/开发', behaviors: ['需求对齐：与 User 沟通澄清范围（IM 式），结论落 discussion', '方案/开发：按模板产出 → 登记 → 提评'], onFeedback: '收评审/CR/缺陷反馈 → 澄清 → 修订 → resolve → 复核' }
    }
    // 产出映射：按节点 key（JSON 载体过渡；sqlite 迁移后由 capabilities.products 展开，v0.28/m8 §11.2）
    const NODE_OUTPUTS = {
      '产品设计': ['《产品设计文档》prd.md'],
      'RD 技术设计': ['《RD 技术设计》rd/design.md'],
      'FE 技术设计': ['《FE 技术设计》fe/design.md'],
      '用例设计': ['《用例设计》qa/cases.md'],
      'RD 开发': ['《RD 开发说明》rd/dev.md', 'MR'],
      'FE 开发': ['《FE 开发说明》fe/dev.md', 'MR'],
      '并行开发': ['《RD 开发说明》rd/dev.md', '《FE 开发说明》fe/dev.md', 'MR'],
      'QA 验收': ['《测试用例与验收记录》qa/test.md', '《缺陷文档》qa/defect.md'],
      '方案评审': ['《方案文档》plan.md'],
      '调研': ['《调研文档》research.md'],
      '架构规划': ['《架构规划》arch_design.md']
    }
    function isAlignNode(nodeKey) { return /需求对齐|立项/.test(nodeKey || '') }
    function nodeOutputs(nodeKey, gate) { if (gate) return []; return NODE_OUTPUTS[nodeKey] || [] }

    // wb_my_task(reqId, role, nodeId?)：m8 §11 返回结构
    // sessionsByNode = 泛化（v0.30）：每个 node 的所有会话（含锚定到产物文档的会话）都作为上下文产物，
    //   按 node 分组返回，每组内按 at 升序；每条会话带 nodeId/nodeKey 明确归属，文档锚定解析 anchorArtifact
    async function opMyTask(args) {
      const rid = args && args.reqId
      const role = args && args.role
      if (!rid || !role) return { ok: false, error: 'missing reqId/role', task: null }
      const req = await loadRequirement(rid)
      if (!req) return { ok: false, error: 'requirement not found', task: null }
      const flow = req.flow
      const nodes = flow.nodes || []
      const nid = (args && args.nodeId) || flow.currentNodeId
      const node = nodes.filter(function (n) { return n.id === nid })[0]
      if (!node) return { ok: false, error: 'node not found: ' + nid, task: null }
      const gate = !!node.gate
      const align = isAlignNode(node.key)
      const proto = ROLE_PROTOCOLS[role] || { roleObjective: '', behaviors: [], onFeedback: '' }
      const nodeOrder = node.order !== undefined ? node.order : nodes.indexOf(node)
      // 依赖文档：order 较小（前置）节点已登记产物
      const dependencyDocs = (req.artifacts || []).filter(function (a) {
        const owner = nodes.filter(function (n) { return n.id === a.nodeId })[0]
        return owner && (owner.order !== undefined ? owner.order : nodes.indexOf(owner)) < nodeOrder
      }).map(function (a) { return a.href })
      // 待审/已产产物：gate 节点=前置节点产物；产出节点=本节点产物
      const artifactsToReview = gate
        ? (req.artifacts || []).filter(function (a) {
            const owner = nodes.filter(function (n) { return n.id === a.nodeId })[0]
            return owner && (owner.order !== undefined ? owner.order : nodes.indexOf(owner)) < nodeOrder
          })
        : (req.artifacts || []).filter(function (a) { return a.nodeId === nid })
      // 全会话按 node 分组（泛化）：每个 node 组 = 该 node 下全部评论线程（issue/discussion/notification 全含，
      // 按 at 升序）；锚到产物文档的会话解析出 anchorArtifact；组内每条携带 nodeId/nodeKey 明确归属
      const sessionsByNode = nodes.map(function (n) {
        const list = (req.comments || []).filter(function (c) { return c.nodeId === n.id }).sort(function (a, b) { return new Date(a.at) - new Date(b.at) }).map(function (c) {
          const art = (c.anchor && c.anchor.artifactId) ? (req.artifacts || []).filter(function (a) { return a.id === c.anchor.artifactId })[0] : undefined
          return {
            issue_id: c.id, nodeId: n.id, nodeKey: n.key,   // ★ 明确归属哪个 node
            type: c.type || 'issue', author: c.author, role: c.role, text: c.text,
            at: c.at, state: c.state,
            anchor: c.anchor || { kind: 'document' },
            anchorArtifact: art ? { label: art.label, href: art.href } : null,   // 文档里的会话：锚定产物解析
            replies: c.thread || []
          }
        })
        return { nodeId: n.id, nodeKey: n.key, nodeOrder: n.order !== undefined ? n.order : nodes.indexOf(n), sessions: list }
      })
      return {
        ok: true,
        reqId: rid, role: role, nodeId: node.id, nodeKey: node.key,
        mode: align ? 'align' : 'produce', gate: gate, state: node.state,
        task: gate ? '评审+表态：' + node.key : '产出：' + node.key,
        roleObjective: proto.roleObjective, behaviors: proto.behaviors, onFeedback: proto.onFeedback,
        reviewers: (node.reviewers || []).map(function (r) { return { role: r.role, state: r.state || '' } }),
        artifactsToReview: artifactsToReview,
        dependencyDocs: dependencyDocs,
        outputs: align ? [] : nodeOutputs(node.key, gate),
        sessionsByNode: sessionsByNode
      }
    }

    // ---- inbox（IM 收件箱 v1：JSON 文件版，schema 与 sqlite 表一致；sqlite 迁移留 preset 环境验证）----
    function pushInbox(msg) {
      if (!fs) return Promise.resolve()
      return readJson('inbox.json').then(function (cur) {
        const list = Array.isArray(cur) ? cur : []
        list.push(Object.assign({ id: 'in-' + Date.now(), at: new Date().toISOString(), read: 0 }, msg))
        return writeJson('inbox.json', list)
      }).catch(function () {})
    }
    async function opListInbox(args) { const list = await readJson('inbox.json'); const arr = Array.isArray(list) ? list : []; const unread = args && args.unread; return unread ? arr.filter(function (m) { return !m.read }) : arr }

    async function opListRequirements() { return await listRequirements() }
    async function opListVersions() { return await listVersions() }
    async function opGetRequirement(args) { const id = args && args.reqId; if (!id) return null; return await loadRequirement(id) }
    async function opCreateVersion(args) {
      const name = (args && args.name || '').trim()
      if (!name) return { ok: false, error: 'name required', verList: null }
      const list = await listVersions()
      let maxN = 0
      list.forEach(function (v) { const m = String(v.id || '').match(/^v-(\d+)$/); if (m) { const n = parseInt(m[1], 10); if (n > maxN) maxN = n } })
      const id = 'v-' + (maxN + 1)
      list.push({ id: id, name: name, state: '待启动', isBacklog: false })
      const w = await writeJson('versions.json', list)
      return { ok: w.ok, error: w.err, verList: list }
    }
    async function opUpdateVersion(args) {
      const vid = args && args.versionId
      const name = (args && args.name || '').trim()
      if (!vid || !name) return { ok: false, error: 'missing args', verList: null }
      const list = await listVersions()
      const v = list.filter(function (x) { return x.id === vid })[0]
      if (!v) return { ok: false, error: 'not found', verList: list }
      if (v.isBacklog) return { ok: false, error: 'backlog locked', verList: list }
      v.name = name
      const w = await writeJson('versions.json', list)
      return { ok: w.ok, error: w.err, verList: list }
    }
    async function opAdvanceVersion(args) {
      const vid = args && args.versionId
      if (!vid) return { ok: false, error: 'missing versionId', verList: null }
      const list = await listVersions()
      const v = list.filter(function (x) { return x.id === vid })[0]
      if (!v) return { ok: false, error: 'not found', verList: list }
      if (v.isBacklog) return { ok: false, error: 'backlog locked', verList: list }
      const next = VER_STATES[v.state]
      if (!next) return { ok: false, error: 'cannot advance from ' + v.state, verList: list }
      v.state = next
      let unfinished = 0
      if (next === '已发布') {
        const reqs = await listRequirements()
        unfinished = reqs.filter(function (r) { return r.versionId === vid && (r.status === '待启动' || r.status === '进行中') }).length
      }
      const w = await writeJson('versions.json', list)
      return { ok: w.ok, error: w.err, verList: list, unfinished: unfinished }
    }
    async function opDeprecateVersion(args) {
      const vid = args && args.versionId
      if (!vid) return { ok: false, error: 'missing versionId', verList: null }
      const vlist = await listVersions()
      const v = vlist.filter(function (x) { return x.id === vid })[0]
      if (!v) return { ok: false, error: 'not found', verList: vlist }
      if (v.isBacklog) return { ok: false, error: 'backlog locked', verList: vlist }
      if (v.state === '已废弃') return { ok: false, error: 'already deprecated', verList: vlist }
      const reqs = await listRequirements()
      const refCount = reqs.filter(function (r) { return r.versionId === vid }).length
      if (refCount > 0) return { ok: false, blocked: true, refCount: refCount, error: '版本下还有需求', verList: vlist }
      v.state = '已废弃'
      const w = await writeJson('versions.json', vlist)
      return { ok: w.ok, error: w.err, verList: vlist }
    }
    async function opCreateRequirement(args) {
      const title = (args && args.title || '').trim()
      if (!title) return { ok: false, error: 'title required', reqList: null }
      const list = await listRequirements()
      const maxN = list.reduce(function (m, r) { var n = parseInt(String(r.id || '').replace(/^req-/, ''), 10); return isNaN(n) ? m : Math.max(m, n) }, 0)
      const id = 'req-' + ('00' + (maxN + 1)).slice(-3)
      const now = new Date().toISOString()
      list.push({ id: id, projectId: PROJECT, title: title, description: args.description || '', category: args.category || '需求任务', versionId: args.versionId || 'v-backlog', priority: args.priority || 'P2', status: args.status || '待启动', owner: args.owner || 'Leader', createdAt: now, updatedAt: now, history: [] })
      const w = await writeJson('requirements.json', list)
      return { ok: w.ok, error: w.err, reqList: list }
    }
    async function opUpdateRequirement(args) {
      const rid = args && args.reqId
      if (!rid) return { ok: false, error: 'missing reqId', reqList: null }
      const list = await listRequirements()
      const req = list.filter(function (r) { return r.id === rid })[0]
      if (!req) return { ok: false, error: 'not found', reqList: list }
      const fields = ['title', 'description', 'category', 'versionId', 'priority', 'status', 'owner']
      const changes = []
      fields.forEach(function (f) {
        if (args[f] !== undefined && String(args[f]) !== String(req[f] || '')) {
          changes.push({ field: f, from: req[f] || '', to: args[f] })
          req[f] = args[f]
        }
      })
      const now = new Date().toISOString()
      req.updatedAt = now
      if (changes.length) {
        req.history = req.history || []
        req.history.push({ at: now, by: 'User', changes: changes })
      }
      const w = await writeJson('requirements.json', list)
      return { ok: w.ok, error: w.err, reqList: list }
    }
    async function opDeleteRequirement(args) {
      const rid = args && args.reqId
      if (!rid) return { ok: false, error: 'missing reqId', reqList: null }
      const list = await listRequirements()
      const next = list.filter(function (r) { return r.id !== rid })
      const w = await writeJson('requirements.json', next)
      return { ok: w.ok, error: w.err, reqList: next }
    }
    async function opGetArtifactContent(args) {
      const aid = args && args.artifactId
      const rid = args && args.reqId
      if (!fs || !aid) return null
      const arts = await readJson(rid + '/artifacts.json')
      const art = Array.isArray(arts) ? arts.filter(function (a) { return a.id === aid })[0] : undefined
      if (!art) return null
      try {
        const t = await fs.resolve(PROJECT + '/' + art.href)
        const info = await fs.stat(t)
        if (!info) return null
        const text = await fs.readText(t)
        return { art: art, content: text }
      } catch (err) {
        return null
      }
    }

    async function mutateComments(reqId, fn) { const comments = await readJson(reqId + '/comments.json'); const list = Array.isArray(comments) ? comments : []; fn(list); const w = await writeJson(reqId + '/comments.json', list); const snap = await loadRequirement(reqId); return { ok: w.ok, error: w.err, snapshot: snap } }

    async function opComment(args) { const rid = args.reqId; const nid = args.nodeId; const text = (args.text || '').trim(); const anchor = args.anchor; if (!rid || !nid || !text) return { ok: false, error: 'missing args' }; const author = args.author || 'User'; const role = args.role || 'user'; const to = args.to || args.reviewerRole || 'Leader'; const type = args.type === 'notification' || args.type === 'discussion' ? args.type : 'issue'; return await mutateComments(rid, function (list) { list.push({ id: 'cm-' + Date.now(), requirementId: rid, nodeId: nid, anchor: anchor || { kind: 'document' }, author: author, role: role, reviewerRole: args.reviewerRole || '', type: type, assignee: args.assignee || '', text: text, at: new Date().toISOString(), state: 'open', thread: [] }) }).then(function (r) { pushInbox({ reqId: rid, type: 'mention', from: author, to: to, text: text.slice(0, 60) }); return r }) }
    async function opAddArtifact(args) {
      const rid = args.reqId
      const nid = args.nodeId
      const label = (args.label || '').trim()
      const href = (args.href || '').trim()
      const kind = args.kind === 'link' ? 'link' : 'document'
      if (!rid || !nid || !label || !href) return { ok: false, error: 'missing args' }
      const arts = await readJson(rid + '/artifacts.json')
      const list = Array.isArray(arts) ? arts : []
      list.push({ id: 'art-' + Date.now(), nodeId: nid, label: label, href: href, kind: kind, note: args.note || '', addedBy: args.by || 'User', addedAt: new Date().toISOString() })
      const w = await writeJson(rid + '/artifacts.json', list)
      return { ok: w.ok, error: w.err, artifacts: list }
    }
    async function opReview(args) {
      const rid = args.reqId
      const nid = args.nodeId
      const role = args.role
      const verdict = args.verdict
      if (!rid || !nid || !role || (verdict !== 'pass' && verdict !== 'block')) return { ok: false, error: 'missing args' }
      const flow = await readJson(rid + '/flow.json')
      if (!flow || !flow.nodes || flow.__err) return { ok: false, error: 'no flow' }
      const node = flow.nodes.filter(function (n) { return n.id === nid })[0]
      if (!node) return { ok: false, error: 'node not found' }
      const rv = (node.reviewers || []).filter(function (r) { return r.role === role })[0]
      if (!rv) return { ok: false, error: 'role not in reviewers' }
      rv.state = verdict === 'pass' ? '已通过' : '未通过'
      rv.reviewedAt = new Date().toISOString()
      rv.reviewedBy = args.by || role
      if (verdict === 'block') { rv.blockReason = args.reason || '' }
      await writeJson(rid + '/flow.json', flow)
      pushInbox({ reqId: rid, type: 'review', from: args.by || role, to: node.ownerRole || 'Leader', text: '评审 ' + role + ' ' + (verdict === 'pass' ? '通过' : '未通过') + '：' + node.key })
      return { ok: true, snapshot: await loadRequirement(rid) }
    }
    async function opReply(args) { const rid = args.reqId; const cid = args.commentId; const text = (args.text || '').trim(); const by = args.by || 'User'; const role = args.role || 'user'; if (!rid || !cid || !text) return { ok: false, error: 'missing args' }; return await mutateComments(rid, function (list) { var c = list.filter(function (x) { return x.id === cid })[0]; if (c) { c.thread = c.thread || []; c.thread.push({ by: by, role: role, text: text, at: new Date().toISOString() }); c.roundCount = (c.roundCount || 0) + 1; c.lastReplyAt = new Date().toISOString(); if (c.author) pushInbox({ reqId: rid, type: 'mention', from: by, to: c.author, text: '回复评论：' + text.slice(0, 40) }) } }) }
    async function opResolve(args) {
      const rid = args.reqId
      const cid = args.commentId
      const note = (args.note || '').trim()
      const path = args.path === 'agree' ? 'agree' : 'fix'
      const by = args.by || 'User'
      if (!rid || !cid || !note) return { ok: false, error: 'missing args（note 必填：修复说明或一致结论）' }
      return await mutateComments(rid, function (list) {
        var c = list.filter(function (x) { return x.id === cid })[0]
        if (!c) return
        c.resolve = { by: by, at: new Date().toISOString(), note: note, path: path, reviewerConfirmed: false }
        if (path === 'agree') {
          // B 协商一致关闭：提评者 resolve，一致即确认
          c.state = 'resolved'
          c.resolve.reviewerConfirmed = true
        } else {
          // A 修复关闭：转待复核，通知提评者
          c.state = 'pending-confirm'
          if (c.author) pushInbox({ reqId: rid, type: 'mention', from: by, to: c.author, text: '已修复，请复核：' + note.slice(0, 40) })
        }
      })
    }
    async function opConfirmReview(args) {
      const rid = args.reqId
      const cid = args.commentId
      const by = args.by || 'User'
      if (!rid || !cid) return { ok: false, error: 'missing args' }
      return await mutateComments(rid, function (list) {
        var c = list.filter(function (x) { return x.id === cid })[0]
        if (!c) return
        if (c.state === 'pending-confirm') {
          c.state = 'resolved'
          c.resolve = c.resolve || {}
          c.resolve.reviewerConfirmed = true
          c.resolve.confirmedBy = by
          c.resolve.confirmedAt = new Date().toISOString()
        }
      })
    }
    async function opReopen(args) {
      const rid = args.reqId
      const cid = args.commentId
      const reason = (args.reason || '').trim()
      const by = args.by || 'User'
      if (!rid || !cid || !reason) return { ok: false, error: 'missing args（reason 必填：复核不通过理由）' }
      return await mutateComments(rid, function (list) {
        var c = list.filter(function (x) { return x.id === cid })[0]
        if (!c) return
        if (c.state === 'pending-confirm') {
          c.state = 'open'
          c.roundCount = (c.roundCount || 0) + 1
          c.thread = c.thread || []
          c.thread.push({ by: by, role: '', text: '复核不通过，重新打开：' + reason, at: new Date().toISOString() })
          if (c.roundCount > 5) pushInbox({ reqId: rid, type: 'gate', from: by, to: 'Leader', text: '评论拉锯超过 5 轮，请介入：' + String(c.text || '').slice(0, 40) })
          if (c.assignee || c.reviewerRole) pushInbox({ reqId: rid, type: 'mention', from: by, to: c.assignee || c.reviewerRole || 'Leader', text: '复核不通过，请继续：' + reason.slice(0, 40) })
        }
      })
    }
    async function opOverride(args) { const rid = args.reqId; const cid = args.commentId; const role = args.reviewerRole; return await mutateComments(rid, function (list) { list.forEach(function (c) { if (cid && c.id === cid) { c.state = 'overridden'; c.resolvedBy = 'User'; c.resolvedAt = new Date().toISOString() } else if (role && c.reviewerRole === role && c.state === 'open') { c.state = 'overridden'; c.resolvedBy = 'User'; c.resolvedAt = new Date().toISOString() } }) }) }
    async function opDecide(args) {
      const rid = args.reqId
      const nid = args.nodeId
      const verdict = args.verdict
      const reason = args.reason || ''
      const newScope = !!args.newScope
      const targetBranch = args.targetBranch || ''
      if (!rid || !nid) return { ok: false, error: 'missing args' }
      const flow = await readJson(rid + '/flow.json')
      if (!flow || !flow.nodes || flow.__err) return { ok: false, error: 'no flow' }
      const nodes = flow.nodes
      const idx = nodes.map(function (n) { return n.id }).indexOf(nid)
      if (idx < 0) return { ok: false, error: 'node not found' }
      const node = nodes[idx]
      const comments = await readJson(rid + '/comments.json')
      if (verdict === 'pass') {
        const open = openComments(comments, nid)
        if (open > 0) return { ok: false, error: '还有 ' + open + ' 条评论未解决，不能通过', blocked: true, open: open }
        const pend = pendingComments(comments, nid)
        if (pend > 0) return { ok: false, error: '还有 ' + pend + ' 条评论待提评者复核（confirm），不能通过', blocked: true, pendingConfirm: pend }
        const rvBlocked = (node.reviewers || []).filter(function (r) { return r.state === '未通过' }).length
        if (rvBlocked > 0) return { ok: false, error: '还有 ' + rvBlocked + ' 位评审者未通过（wb_review block），不能通过', blocked: true, blockedReviewers: rvBlocked }
        node.state = '已通过'
        var next = null
        if (node.parallelGroup) {
          // 并行组 join：组内全部已通过才放行到组后节点，否则 current 指向组内下一个未通过分支
          const grp = nodes.filter(function (n) { return n.parallelGroup === node.parallelGroup })
          const allDone = grp.every(function (n) { return n.state === '已通过' })
          for (var gi = idx + 1; gi < nodes.length; gi++) { if (nodes[gi].state !== '已通过') { next = nodes[gi]; break } }
          if (!allDone) {
            // 组内还有分支未完成：current 定位到组内下一个未通过分支节点
            const gPending = grp.filter(function (n) { return n.state !== '已通过' })[0]
            next = gPending || next
          }
        } else {
          for (var i = idx + 1; i < nodes.length; i++) { if (nodes[i].state !== '已通过') { next = nodes[i]; break } }
        }
        if (next) { flow.currentNodeId = next.id; if (next.state === '未开始') next.state = '进行中'; if (!next.enteredAt) next.enteredAt = new Date().toISOString() }
        pushInbox({ reqId: rid, type: 'gate', from: 'wb_decide', to: next ? next.ownerRole || 'Leader' : 'User', text: '节点已通过，进入：' + (next ? next.key : '结束') })
      } else {
        if (!reason) return { ok: false, error: '打回必须填写理由' }
        if (targetBranch) {
          // 部分回滚：按 targetBranch 找到分支所在并行组，仅回滚该分支节点（QA 打回对应方），另一分支保持已通过
          const grp = nodes.filter(function (n) { return n.branch === targetBranch && n.parallelGroup })
          grp.forEach(function (n) {
            n.state = '进行中'
            n.rolledBackAt = n.rolledBackAt || []
            n.rolledBackAt.push(new Date().toISOString())
            n.lastDecision = { verdict: 'reject', reason: reason, at: new Date().toISOString(), branch: targetBranch }
          })
          const tgt = grp[grp.length - 1]
          if (tgt) flow.currentNodeId = tgt.id
          pushInbox({ reqId: rid, type: 'gate', from: 'QA', to: targetBranch === 'rd' ? 'RD' : 'FE', text: '部分打回 ' + targetBranch + ' 分支：' + reason.slice(0, 40) })
        } else {
          node.state = '进行中'
          node.rolledBackAt = node.rolledBackAt || []
          node.rolledBackAt.push(new Date().toISOString())
          node.lastDecision = { verdict: 'reject', reason: reason, at: new Date().toISOString() }
          if (newScope) node.pendingScope = reason
          var prev = null
          for (var j = idx - 1; j >= 0; j--) { if (nodes[j].state !== '未开始') { prev = nodes[j]; break } }
          if (prev) { flow.currentNodeId = prev.id; prev.state = '进行中' }
        }
      }
      await writeJson(rid + '/flow.json', flow)
      return { ok: true, snapshot: await loadRequirement(rid) }
    }

    const rpc = {
      'wb.listRequirements': opListRequirements,
      'wb.listVersions': opListVersions,
      'wb.getRequirement': opGetRequirement,
      'wb.myTask': opMyTask,
      'wb.createVersion': opCreateVersion,
      'wb.updateVersion': opUpdateVersion,
      'wb.advanceVersion': opAdvanceVersion,
      'wb.deprecateVersion': opDeprecateVersion,
      'wb.createRequirement': opCreateRequirement,
      'wb.updateRequirement': opUpdateRequirement,
      'wb.deleteRequirement': opDeleteRequirement,
      'wb.getArtifactContent': opGetArtifactContent,
      'wb.addArtifact': opAddArtifact,
      'wb.comment': opComment,
      'wb.reply': opReply,
      'wb.resolve': opResolve,
      'wb.confirmReview': opConfirmReview,
      'wb.reopen': opReopen,
      'wb.override': opOverride,
      'wb.review': opReview,
      'wb.listInbox': opListInbox,
      'wb.decide': opDecide
    }
    Object.keys(rpc).forEach(function (m) { harness.handle(m, rpc[m]) })

    function toolDef(name, description, parameters, execute) {
      return harness.defineTool({
        name: name,
        description: description,
        parameters: parameters,
        output: { schema: { type: 'object', additionalProperties: true }, render: function (_args, value) { return [{ type: 'text', text: JSON.stringify(value, null, 2) }] } },
        execute: execute
      })
    }
    const S = { type: 'string' }
    const SB = { type: 'boolean' }
    const J = { type: 'object', additionalProperties: true }
    // 模型工具 output schema 要求对象：列表/空值统一包成 {items:[...]} 或 {ok:false}
    function asObj(p) { return p.then(function (v) { if (v === null || v === undefined) return { ok: false }; return Array.isArray(v) ? { items: v } : v }) }
    const tools = [
      { name: 'wb_list_requirements', description: '列出工作台全部需求（需求池），返回 {items:[{id,title,category,versionId,priority,status,owner,description}]}。agent 推进任务或拆分版本前先查此列表。', parameters: {}, execute: function (a) { return asObj(opListRequirements(a)) } },
      { name: 'wb_list_versions', description: '列出全部版本（含待分配池），返回 {items:[{id,name,state,isBacklog}]}。拆分大版本或把需求归入版本时先查此列表。', parameters: {}, execute: function (a) { return asObj(opListVersions(a)) } },
      { name: 'wb_get_requirement', description: '读取单个需求聚合详情（需求字段 + 流程树 nodes/currentNodeId + 产物 artifacts + 评论 comments）。评审、推进节点前先读详情。', parameters: { reqId: Object.assign({ required: true, description: '需求 id，如 req-001' }, S) }, execute: function (a) { return asObj(opGetRequirement(a)) } },
      { name: 'wb_my_task', description: '获取确定性任务上下文（pkg-41，m8 §11）：入参 reqId+role（+可选 nodeId，缺省=当前节点）。返回 {nodeKey, mode(produce|align), gate, state, task, roleObjective, behaviors, onFeedback, reviewers, artifactsToReview, dependencyDocs, outputs, sessionsByNode}。★ sessionsByNode = 按 node 分组返回该需求**每个 node 的全部会话**（issue/discussion/notification 全含，含锚定在产物文档上的会话——anchorArtifact 已解析出 label/href；每条携带 nodeId/nodeKey 明确归属，组内时间升序、replies 完整）——设计/开发阶段 subagent 被派活后先调本工具拿全部上下文，无需自行检索讨论。', parameters: { reqId: Object.assign({ required: true, description: '需求 id，如 req-016' }, S), role: Object.assign({ required: true, description: '角色 key：rd/fe/qa/arch/pm/user/leader/cr/inv/executor' }, S), nodeId: Object.assign({ description: '节点 id（缺省=当前节点 current）' }, S) }, execute: opMyTask },
      { name: 'wb_get_artifact_content', description: '读取某个产物（文档）的正文内容。评审产物时用：先 wb_get_requirement 拿 artifactId，再读正文定位评论锚点。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), artifactId: Object.assign({ required: true, description: '产物 id' }, S) }, execute: function (a) { return asObj(opGetArtifactContent(a)) } },
      { name: 'wb_create_requirement', description: '新建需求（title 必填；description 支持 Markdown；category 取 需求任务/规划审批任务/版本规划任务；versionId 取 wb_list_versions 的 id 或 v-backlog；priority P0-P3；status 待启动/进行中/已完成/已取消；owner 取 User/Leader/Investigator/Architect/RD/PM/FE/QA）。自动带审计起点 history=[]。', parameters: { title: Object.assign({ required: true, description: '需求标题' }, S), description: Object.assign({ description: 'Markdown 描述' }, S), category: Object.assign({ description: '类别' }, S), versionId: Object.assign({ description: '版本 id' }, S), priority: Object.assign({ description: '优先级' }, S), status: Object.assign({ description: '状态' }, S), owner: Object.assign({ description: '负责人角色' }, S) }, execute: opCreateRequirement },
      { name: 'wb_update_requirement', description: '更新需求字段（title/description/category/versionId/priority/status/owner 中任意）。Host 自动计算 diff 并写变更历史（谁将什么从什么变更为什么）。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), title: Object.assign({ description: '标题' }, S), description: Object.assign({ description: '描述' }, S), category: Object.assign({ description: '类别' }, S), versionId: Object.assign({ description: '版本 id' }, S), priority: Object.assign({ description: '优先级' }, S), status: Object.assign({ description: '状态' }, S), owner: Object.assign({ description: '负责人' }, S) }, execute: opUpdateRequirement },
      { name: 'wb_delete_requirement', description: '删除需求（从需求池移除，不删除磁盘目录）。破坏性操作，需 User 明确指令后调用。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S) }, execute: opDeleteRequirement },
      { name: 'wb_create_version', description: '新建版本（state=待启动）。版本名自由文本。', parameters: { name: Object.assign({ required: true, description: '版本名，如 v1.2 / 2.0 迭代' }, S) }, execute: opCreateVersion },
      { name: 'wb_update_version', description: '修改版本名（待分配池禁止改名）。', parameters: { versionId: Object.assign({ required: true, description: '版本 id' }, S), name: Object.assign({ required: true, description: '新版本名' }, S) }, execute: opUpdateVersion },
      { name: 'wb_advance_version', description: '推进版本状态：待启动→进行中→已发布（单向不可回退）。发布到已发布时会返回未完成需求数 unfinished，有未完成需求时发布属重大动作，需 User 确认后调用。', parameters: { versionId: Object.assign({ required: true, description: '版本 id' }, S) }, execute: opAdvanceVersion },
      { name: 'wb_deprecate_version', description: '废弃版本（废弃后不可恢复）。版本下仍有需求时 blocked（需先把需求移出）。破坏性操作，需 User 明确指令后调用。', parameters: { versionId: Object.assign({ required: true, description: '版本 id' }, S) }, execute: opDeprecateVersion },
      { name: 'wb_comment', description: '在需求某个流程节点追加评论。type=issue（默认，阻塞门禁，走 resolve 闭环）| notification（交付/通知，不阻塞）| discussion（纯沟通，reply 达成一致即关闭）。anchor 可省略（document 级），或传 {kind:"document"|"line"|"range"|"block", artifactId, ...}。author=发表者；reviewerRole=评审归属；to=收件提醒人（默认 reviewerRole 或 Leader）；assignee=解决责任人（缺省=文档/产物作者；仅 QA 验收缺陷显式指派 RD|FE）。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), nodeId: Object.assign({ required: true, description: '流程节点 id' }, S), text: Object.assign({ required: true, description: '评论内容' }, S), anchor: Object.assign({ description: '锚点对象' }, J), author: Object.assign({ description: '发表者（角色名）' }, S), role: Object.assign({ description: '发表者角色 key' }, S), reviewerRole: Object.assign({ description: '评审归属角色（门禁/override 匹配）' }, S), to: Object.assign({ description: '收件提醒人（默认 reviewerRole 或 Leader）' }, S), type: Object.assign({ description: 'issue|notification|discussion（默认 issue）' }, S), assignee: Object.assign({ description: '解决责任人（缺省=文档作者；QA 缺陷显式指派 RD|FE）' }, S) }, execute: opComment },
      { name: 'wb_add_artifact', description: '登记产物到某个流程节点（artifacts.json）。文档正文由作者写入项目目录后，用本工具登记元数据（label/href/kind=document|link/note）。对应 wb_add_artifact 领域操作。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), nodeId: Object.assign({ required: true, description: '流程节点 id' }, S), label: Object.assign({ required: true, description: '产物名，如《RD 技术设计》' }, S), href: Object.assign({ required: true, description: '相对项目路径（doc）或外链 URL（link）' }, S), kind: Object.assign({ description: 'document 或 link' }, S), note: Object.assign({ description: '备注' }, S), by: Object.assign({ description: '登记人' }, S) }, execute: opAddArtifact },
      { name: 'wb_reply', description: '在已有评论下追加对话（roundCount+1，通知评论作者）。by/role 指定回复者。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), commentId: Object.assign({ required: true, description: '评论 id' }, S), text: Object.assign({ required: true, description: '回复内容' }, S), by: Object.assign({ description: '回复者（角色名）' }, S), role: Object.assign({ description: '回复者角色 key' }, S) }, execute: opReply },
      { name: 'wb_override', description: '强行通过：把单条评论或某评审角色的全部 open 评论标记为 overridden（强权，绕过门禁，需 User 明确指令）。传 reviewerRole 时作用于该角色全部剩余 open 评论。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), commentId: Object.assign({ description: '评论 id' }, S), reviewerRole: Object.assign({ description: '评审角色' }, S) }, execute: opOverride },
      { name: 'wb_resolve', description: '关闭评论（双路径）：A 修复关闭（assignee/作者 resolve，note=修复说明，path=fix）→ 转待复核 pending-confirm 并通知提评者复核；B 协商一致关闭（提评者 resolve，note=一致结论，path=agree）→ 直接关闭。note 必填。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), commentId: Object.assign({ required: true, description: '评论 id' }, S), note: Object.assign({ required: true, description: '修复说明 或 一致结论' }, S), path: Object.assign({ description: 'fix（默认，修复关闭）或 agree（协商一致关闭）' }, S), by: Object.assign({ description: '关闭人' }, S) }, execute: opResolve },
      { name: 'wb_confirm_review', description: '提评者复核确认：pending-confirm → resolved（reviewerConfirmed=true，门禁放行）。仅提评者可确认（QA 缺陷场景即 QA）。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), commentId: Object.assign({ required: true, description: '评论 id' }, S), by: Object.assign({ description: '确认人（提评者）' }, S) }, execute: opConfirmReview },
      { name: 'wb_reopen', description: '提评者复核不通过：pending-confirm → open（roundCount++，通知 assignee 继续；QA 缺陷场景即 QA 打回对方解决）。roundCount>5 时自动通知 Leader/User 介入。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), commentId: Object.assign({ required: true, description: '评论 id' }, S), reason: Object.assign({ required: true, description: '复核不通过理由' }, S), by: Object.assign({ description: '复核人（提评者）' }, S) }, execute: opReopen },
      { name: 'wb_review', description: '评审者正式表态：更新某节点的指定评审角色 reviewers[].state 为 已通过(pass) 或 未通过(block)。wb_decide pass 时若有评审者未通过会被拦截。block 建议先 wb_comment 留理由并传 reason。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), nodeId: Object.assign({ required: true, description: '流程节点 id' }, S), role: Object.assign({ required: true, description: '评审角色，如 Architect/RD/FE/QA/PM/Code Reviewer/User' }, S), verdict: Object.assign({ required: true, description: 'pass 或 block' }, S), reason: Object.assign({ description: 'block 理由' }, S), by: Object.assign({ description: '表态人' }, S) }, execute: opReview },
      { name: 'wb_list_inbox', description: '查询项目收件箱（门禁待处理/评审表态/评论@），返回 {items:[{id,reqId,type,from,to,at,read}]}。unread=true 只看未读。', parameters: { unread: Object.assign({ description: '只看未读' }, SB) }, execute: function (a) { return asObj(opListInbox(a)) } },
      { name: 'wb_decide', description: '门禁裁决：verdict=pass 通过当前节点（未解决的 open 评论、或有 wb_review 未通过的评审者会 blocked 拦截）；并行组节点（业务需求开发）需组内全部分支均通过才放行下一节点。verdict=reject 打回上一节点（reason 必填）；对并行组节点传 targetBranch=rd|fe 实现部分回滚（仅回滚指定分支）。与 UI 同一套状态机。', parameters: { reqId: Object.assign({ required: true, description: '需求 id' }, S), nodeId: Object.assign({ required: true, description: '节点 id' }, S), verdict: Object.assign({ required: true, description: 'pass 或 reject' }, S), reason: Object.assign({ description: '打回理由（reject 必填）' }, S), newScope: Object.assign({ description: '是否需重新界定新范围' }, SB), targetBranch: Object.assign({ description: '部分回滚目标分支 rd|fe（reject+并行组时用）' }, S) }, execute: opDecide }
    ]
    tools.forEach(function (t) {
      const def = toolDef(t.name, t.description, t.parameters, t.execute)
      ctx.effect(function () { return harness.registerTool(ctx, def) }, 'wb-tool:' + t.name)
    })

    // ---- pending-confirm 超时轮询（每 5 分钟；User 超时 1h / agent 超时 5min）----
    if (ctx.setInterval) {
      ctx.setInterval(function () {
        listRequirements().then(function (reqs) {
          (reqs || []).forEach(function (rq) {
            readJson(rq.id + '/comments.json').then(function (cmts) {
              if (!Array.isArray(cmts)) return
              cmts.forEach(function (c) {
                if (c.state !== 'pending-confirm' || !c.resolve || !c.resolve.at) return
                if (c.resolve.remindedAt) return
                var el = Date.now() - new Date(c.resolve.at).getTime()
                var threshold = (c.author === 'User') ? 60 * 60 * 1000 : 5 * 60 * 1000
                if (el >= threshold) {
                  c.resolve.remindedAt = new Date().toISOString()
                  writeJson(rq.id + '/comments.json', cmts)
                  pushInbox({ reqId: rq.id, type: 'mention', from: 'timeout', to: c.author, text: '待复核超时，请及时复核：' + String(c.text || '').slice(0, 40) })
                }
              })
            }).catch(function () {})
          })
        }).catch(function () {})
      }, 5 * 60 * 1000)
    }

    ctx.effect(function () {})
  }
}
