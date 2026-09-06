// Client 半：团队项目工作台（wbflow-1）— M5/M4 最终版（pkg-16）
// 职责：conversation.view「项目」tab UI：外层看板/分组列表 + 内层 M2 流程树 + M4 增删改 + M5 版本管理 + 需求信息面板 + 描述折叠。
// 本文件内容 = cordis_define 的 code.client（Plugin 函数体，返回 Cordis Plugin）。
// 变更记录：
//   pkg-4  外层看板（select 版本筛选 + 看板/列表切换）+ 详情复用 + 返回看板
//   pkg-5  详情缺 flow 显示「暂无流程数据」（req.missing 早退）
//   pkg-6  优先级 P0-P3：顶栏筛选 + 卡/行/详情三处展示
//   pkg-7  列表视图分组：公共表头一次 + 按状态分组 + 组间虚线分割
//   pkg-8  新建/编辑弹窗（描述 textarea）+ 卡片/行 hover ✎🗑 + 删除确认 + 详情描述 md 渲染 + 变更历史面板 + 编辑按钮
//   pkg-9  M5 版本管理弹窗（状态徽标/进度条/推进/废弃/改名）+ 版本 CRUD
//   pkg-10 废弃不联动取消需求：有需求引用弹「请先移出」+ 去掉 v1 不可恢复文案
//   pkg-11 版本不支持删除（移除 🗑 与删除弹窗）
//   pkg-12 版本行操作改为明显按钮（▶ 推进为X / ✖ 废弃 / ✎ 改名）
//   pkg-13 listVersions 归一化 state（旧数据缺字段兜底）
//   pkg-14 推进为已发布时若版本下有未完成需求 → 弹发布确认框
//   pkg-15 详情页新增「需求信息」面板（负责人/版本/优先级/状态/类别/创建/更新时间）
//   pkg-16 需求描述折叠：>300 字默认固定高度 180px + 「展开全部 ▾ / 收起 ▴」
//   pkg-17 修复 md 不换行：renderMd 段落容器加 whiteSpace:pre-wrap
//   pkg-20 折叠阈值改为行数：≤4 行完整显示无折叠；>4 行折叠（高度 110px 约 4 行可见）
//   pkg-21 折叠高度 110 → 150px（标题大字号占空间，150px 展示更多）
//   pkg-23 变更历史自然语句：谁将「字段」从「X」变更为「Y」；描述显示谁填写/清空/修改了需求说明
//   pkg-25 列表列对齐：所有列 flex 改 basis-0 严格按比例；表头补操作列占位；行操作列固定 60px
//   pkg-26 流程树并行泳道：parallelGroup/branch 节点渲染 RD/FE 双泳道并排（M7 状态机再补并行汇合语义）
//   pkg-27 流程树改 SVG 方块流程图：节点方块+折线连接，并行分支双列分叉/汇聚（沙箱无 import，自绘 SVG）
//   pkg-28 流程图横向化（节点从左到右、并行分支上下泳道），详情页改上下分栏（流程图在上、详情在下）
//   pkg-29 连线改组级分叉/汇聚（主链→并行组分叉、并行组→主链汇聚、行内串联）；MR 评审并入开发节点（数据层）
//   pkg-30 并行组内只连同行节点，跨分支（RD/FE）之间不画连线
//   pkg-31 流程图缩小两号：节点 140x44、间距/泳道/字号同步缩小
//   pkg-32 流程图再小一号：节点 118x38、间距/泳道/字号再缩
//   pkg-33 产物面板：无产物节点显示占位；link 类型产物点击新窗口跳外链（不进阅读器）
return {
  apply(ctx) {
    const slots = ctx.get('slots')
    if (slots === undefined) return
    slots.inject('conversation.view', function () {
      return slots.register(
        { name: 'conversation.view', id: 'workbench', order: 20, label: '项目' },
        function WorkbenchView(props) {
          var reqListState = React.useState([]); var reqList = reqListState[0]; var setReqList = reqListState[1]
          var verListState = React.useState([]); var verList = verListState[0]; var setVerList = verListState[1]
          var viewState = React.useState('board'); var view = viewState[0]; var setView = viewState[1]
          var detailIdState = React.useState(null); var detailId = detailIdState[0]; var setDetailId = detailIdState[1]
          var verFilterState = React.useState('__all__'); var verFilter = verFilterState[0]; var setVerFilter = verFilterState[1]
          var priFilterState = React.useState('__all__'); var priFilter = priFilterState[0]; var setPriFilter = priFilterState[1]
          var kindState = React.useState('board'); var kind = kindState[0]; var setKind = kindState[1]
          var reqState = React.useState(null); var req = reqState[0]; var setReq = reqState[1]
          var selState = React.useState(''); var sel = selState[0]; var setSel = selState[1]
          var gateModeState = React.useState('idle'); var gateMode = gateModeState[0]; var setGateMode = gateModeState[1]
          var rejectState = React.useState({ reason: '', newScope: false }); var rejectVal = rejectState[0]; var setRejectVal = rejectState[1]
          var errState = React.useState(''); var gateErr = errState[0]; var setGateErr = errState[1]
          var openRoleState = React.useState(''); var openRole = openRoleState[0]; var setOpenRole = openRoleState[1]
          var replyTextState = React.useState({}); var replyText = replyTextState[0]; var setReplyText = replyTextState[1]
          var viewerState = React.useState(null); var viewerId = viewerState[0]; var setViewerId = viewerState[1]
          var viewerDataState = React.useState(null); var viewerData = viewerDataState[0]; var setViewerData = viewerDataState[1]
          var newCmtState = React.useState(''); var newCmt = newCmtState[0]; var setNewCmt = newCmtState[1]
          var rangeStartState = React.useState(null); var rangeStart = rangeStartState[0]; var setRangeStart = rangeStartState[1]
          var rangeEndState = React.useState(null); var rangeEnd = rangeEndState[0]; var setRangeEnd = rangeEndState[1]
          var pendingBlockState = React.useState(null); var pendingBlock = pendingBlockState[0]; var setPendingBlock = pendingBlockState[1]
          var viewSelState = React.useState(null); var viewSel = viewSelState[0]; var setViewSel = viewSelState[1]
          var expandedState = React.useState({}); var expanded = expandedState[0]; var setExpanded = expandedState[1]
          var authorFilterState = React.useState('__all__'); var authorFilter = authorFilterState[0]; var setAuthorFilter = authorFilterState[1]
          var stateFilterState = React.useState('all'); var stateFilter = stateFilterState[0]; var setStateFilter = stateFilterState[1]
          var formState = React.useState(null); var form = formState[0]; var setForm = formState[1]
          var delState = React.useState(null); var delReq = delState[0]; var setDelReq = delState[1]
          var hoverState = React.useState(null); var hoverId = hoverState[0]; var setHoverId = hoverState[1]
          var verMgrState = React.useState(false); var verMgrOpen = verMgrState[0]; var setVerMgrOpen = verMgrState[1]
          var verNameState = React.useState(''); var verNameInput = verNameState[0]; var setVerNameInput = verNameState[1]
          var verDepState = React.useState(null); var verDepTarget = verDepState[0]; var setVerDepTarget = verDepState[1]
          var verRenameState = React.useState(null); var verRenameTarget = verRenameState[0]; var setVerRenameTarget = verRenameState[1]
          var verPubState = React.useState(null); var verPubTarget = verPubState[0]; var setVerPubTarget = verPubState[1]
          var verMsgState = React.useState(''); var verMsg = verMsgState[0]; var setVerMsg = verMsgState[1]
          var descOpenState = React.useState(false); var descOpen = descOpenState[0]; var setDescOpen = descOpenState[1]
          var resolveNoteState = React.useState(''); var resolveNote = resolveNoteState[0]; var setResolveNote = resolveNoteState[1]
          var resolveTargetState = React.useState(null); var resolveTarget = resolveTargetState[0]; var setResolveTarget = resolveTargetState[1]
          var reopenTargetState = React.useState(null); var reopenTarget = reopenTargetState[0]; var setReopenTarget = reopenTargetState[1]

          function loadLists() {
            host.call('wb.listRequirements').then(function (l) { setReqList(l || []) })
            host.call('wb.listVersions').then(function (v) { setVerList(v || []) })
          }
          function openDetail(id) {
            setDetailId(id); setView('detail'); setSel(''); setDescOpen(false)
            host.call('wb.getRequirement', { reqId: id }).then(function (v) {
              if (!v) { var meta = reqList.filter(function (r) { return r.id === id })[0] || null; setReq({ missing: true, requirement: meta }); return }
              setReq(v); if (v.flow && v.flow.currentNodeId) setSel(v.flow.currentNodeId)
            })
          }
          function backToBoard() { setView('board'); setDetailId(null); setReq(null); setViewerId(null); setViewerData(null) }
          React.useEffect(function () { loadLists() }, [])

          const panel = { background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 8, padding: '16px 18px' }
          const stepStyle = { display: 'flex', alignItems: 'center', gap: 8, padding: '9px 12px', borderRadius: 6, cursor: 'pointer', fontSize: 14 }
          const btn = { fontSize: 14, padding: '8px 16px', borderRadius: 6, cursor: 'pointer', border: '1px solid #2b2f3d', background: 'transparent', color: '#e6e8ef' }
          const btnGreen = Object.assign({}, btn, { borderColor: 'rgba(88,192,122,0.6)', color: '#58c07a', background: 'rgba(88,192,122,0.08)' })
          const btnRed = Object.assign({}, btn, { borderColor: 'rgba(217,128,128,0.6)', color: '#d98080', background: 'rgba(217,128,128,0.08)' })
          const btnSmall = Object.assign({}, btn, { padding: '5px 12px', fontSize: 13 })
          const btnSmallRed = Object.assign({}, btnSmall, { borderColor: 'rgba(217,128,128,0.6)', color: '#d98080', background: 'rgba(217,128,128,0.08)' })
          const btnSmallGreen = Object.assign({}, btnSmall, { borderColor: 'rgba(88,192,122,0.6)', color: '#58c07a', background: 'rgba(88,192,122,0.08)' })
          const btnSmallBlue = Object.assign({}, btnSmall, { borderColor: 'rgba(122,184,255,0.5)', color: '#7ab8ff', background: 'rgba(122,184,255,0.08)' })
          const selStyle = { width: '100%', fontSize: 13, padding: '7px 10px', borderRadius: 6, background: '#212530', border: '1px solid #2b2f3d', color: '#e6e8ef', outline: 'none' }
          const inputStyle = { width: '100%', fontSize: 13, padding: '7px 10px', borderRadius: 6, background: '#212530', border: '1px solid #2b2f3d', color: '#e6e8ef', outline: 'none' }

          function doPass() { host.call('wb.decide', { reqId: req.requirement.id, nodeId: sel, verdict: 'pass' }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setSel(r.snapshot.flow.currentNodeId); setGateMode('idle'); setGateErr('') } else if (r && r.blocked) setGateErr(r.error); else setGateErr((r && r.error) || '失败') }) }
          function doReject() { var reason = rejectVal.reason.trim(); host.call('wb.decide', { reqId: req.requirement.id, nodeId: sel, verdict: 'reject', reason: reason, newScope: rejectVal.newScope }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setSel(r.snapshot.flow.currentNodeId); setGateMode('idle'); setRejectVal({ reason: '', newScope: false }); setGateErr('') } else setGateErr((r && r.error) || '失败') }) }
          function doReply(commentId) { var t = (replyText[commentId] || '').trim(); if (!t) return; host.call('wb.reply', { reqId: req.requirement.id, commentId: commentId, text: t, by: 'User' }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setReplyText(Object.assign({}, replyText, { [commentId]: '' })) } }) }
          function doResolve(commentId) { var note = resolveNote.trim(); if (!note) return; var path = (resolveTarget && resolveTarget.path) || 'fix'; host.call('wb.resolve', { reqId: req.requirement.id, commentId: commentId, note: note, path: path, by: 'User' }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setResolveTarget(null); setResolveNote('') } }) }
          function doConfirmReview(commentId) { host.call('wb.confirmReview', { reqId: req.requirement.id, commentId: commentId, by: 'User' }).then(function (r) { if (r && r.ok) setReq(r.snapshot) }) }
          function doReopen(commentId) { var reason = resolveNote.trim(); if (!reason) return; host.call('wb.reopen', { reqId: req.requirement.id, commentId: commentId, reason: reason, by: 'User' }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setReopenTarget(null); setResolveNote('') } }) }
          function doOverrideRole(role) { host.call('wb.override', { reqId: req.requirement.id, reviewerRole: role }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setOpenRole(role) } }) }
          function openViewer(artId) { setViewerId(artId); setRangeStart(null); setRangeEnd(null); setPendingBlock(null); setViewSel(null); setNewCmt(''); setAuthorFilter('__all__'); setStateFilter('all'); host.call('wb.getArtifactContent', { reqId: req.requirement.id, artifactId: artId }).then(function (v) { setViewerData(v) }) }
          function closeViewer() { setViewerId(null); setViewerData(null); setRangeStart(null); setRangeEnd(null); setPendingBlock(null); setViewSel(null) }
          function doComment() {
            var t = newCmt.trim(); if (!t) return
            var anchor
            if (pendingBlock) { anchor = { kind: 'block', artifactId: viewerId, blockStart: pendingBlock.start, blockEnd: pendingBlock.end, lang: pendingBlock.lang } }
            else if (rangeStart !== null) {
              if (rangeStart === rangeEnd) { anchor = { kind: 'line', artifactId: viewerId, line: rangeStart, lineText: '' } }
              else { anchor = { kind: 'range', artifactId: viewerId, startLine: Math.min(rangeStart, rangeEnd), endLine: Math.max(rangeStart, rangeEnd) } }
            }
            else { anchor = { kind: 'document', artifactId: viewerId } }
            host.call('wb.comment', { reqId: req.requirement.id, nodeId: sel, anchor: anchor, text: t }).then(function (r) { if (r && r.ok) { setReq(r.snapshot); setNewCmt(''); setRangeStart(null); setRangeEnd(null); setPendingBlock(null); setViewSel(null) } })
          }
          function pickLine(lineNo) { setPendingBlock(null); setNewCmt(''); setViewSel(null); if (rangeStart === null) { setRangeStart(lineNo); setRangeEnd(lineNo) } else if (rangeStart === rangeEnd) { if (lineNo === rangeStart) { setRangeStart(null); setRangeEnd(null) } else { setRangeEnd(lineNo) } } else { setRangeStart(lineNo); setRangeEnd(lineNo) } }
          function pickBlock(b) { setPendingBlock(pendingBlock && pendingBlock.start === b.start ? null : b); setRangeStart(null); setRangeEnd(null); setViewSel(null); setNewCmt('') }
          function highlightRange(c) { setViewSel({ lo: c.anchor.startLine, hi: c.anchor.endLine }); setRangeStart(null); setRangeEnd(null); setPendingBlock(null) }
          function toggleExpand(id) { var n = Object.assign({}, expanded); n[id] = !n[id]; setExpanded(n) }

          function openNewForm() { setForm({ mode: 'new', title: '', description: '', category: '需求任务', versionId: 'v-backlog', priority: 'P2', status: '待启动', owner: 'Leader' }) }
          function openEditForm(r) { setForm({ mode: 'edit', reqId: r.id, title: r.title || '', description: r.description || '', category: r.category || '需求任务', versionId: r.versionId || 'v-backlog', priority: r.priority || 'P2', status: r.status || '待启动', owner: r.owner || 'Leader' }) }
          function setFormField(k, v) { var n = Object.assign({}, form); n[k] = v; setForm(n) }
          function closeForm() { setForm(null) }
          function saveForm() {
            var t = (form.title || '').trim(); if (!t) return
            var payload = { title: t, description: form.description || '', category: form.category, versionId: form.versionId, priority: form.priority, status: form.status, owner: form.owner }
            var p = form.mode === 'new' ? host.call('wb.createRequirement', payload) : host.call('wb.updateRequirement', Object.assign({ reqId: form.reqId }, payload))
            p.then(function (r) {
              if (r && r.ok) {
                setReqList(r.reqList || [])
                setForm(null)
                if (form.mode === 'edit' && req && req.requirement && req.requirement.id === form.reqId) {
                  host.call('wb.getRequirement', { reqId: form.reqId }).then(function (v) { if (v) setReq(v) })
                }
              }
            })
          }
          function askDelete(r) { setDelReq({ reqId: r.reqId || r.id, title: r.title }) }
          function cancelDelete() { setDelReq(null) }
          function confirmDelete() {
            if (!delReq) return
            host.call('wb.deleteRequirement', { reqId: delReq.reqId }).then(function (r) {
              if (r && r.ok) {
                setReqList(r.reqList || [])
                setDelReq(null)
                if (view === 'detail' && req && req.requirement && req.requirement.id === delReq.reqId) backToBoard()
              }
            })
          }

          function openVerMgr() { setVerMgrOpen(true); setVerMsg('') }
          function closeVerMgr() { setVerMgrOpen(false); setVerDepTarget(null); setVerRenameTarget(null); setVerPubTarget(null); setVerMsg('') }
          function createVer() {
            var name = verNameInput.trim(); if (!name) { setVerMsg('版本名不能为空'); return }
            host.call('wb.createVersion', { name: name }).then(function (r) {
              if (r && r.ok) { setVerList(r.verList || []); setVerNameInput(''); setVerMsg('') }
              else setVerMsg((r && r.error) || '创建失败')
            })
          }
          function verUnfinished(vid) { return reqList.filter(function (r) { return r.versionId === vid && (r.status === '待启动' || r.status === '进行中') }).length }
          function advanceVer(v) {
            var nxt = v.state === '待启动' ? '进行中' : '已发布'
            if (nxt === '已发布') {
              var un = verUnfinished(v.id)
              if (un > 0) { setVerPubTarget({ id: v.id, name: v.name, unfinished: un }); return }
            }
            doAdvance(v)
          }
          function doAdvance(v) {
            host.call('wb.advanceVersion', { versionId: v.id }).then(function (r) {
              if (r && r.ok) {
                setVerList(r.verList || [])
                var nxt = v.state === '待启动' ? '进行中' : '已发布'
                setVerMsg(v.name + ' 已推进：' + v.state + ' → ' + nxt)
                setVerPubTarget(null)
              } else setVerMsg((r && r.error) || '推进失败')
            })
          }
          function doDeprecate() {
            if (!verDepTarget) return
            host.call('wb.deprecateVersion', { versionId: verDepTarget.id }).then(function (r) {
              if (r && r.ok) {
                setVerList(r.verList || [])
                setVerMsg('已废弃 ' + verDepTarget.name)
                setVerDepTarget(null)
              } else if (r && r.blocked) {
                setVerMsg('禁止废弃：' + verDepTarget.name + ' 下还有 ' + r.refCount + ' 条需求，请先将需求移出该版本')
              } else setVerMsg((r && r.error) || '废弃失败')
            })
          }
          function doRenameVer() {
            if (!verRenameTarget) return
            var name = verRenameTarget.newName.trim(); if (!name) { setVerMsg('版本名不能为空'); return }
            host.call('wb.updateVersion', { versionId: verRenameTarget.id, name: name }).then(function (r) {
              if (r && r.ok) { setVerList(r.verList || []); setVerRenameTarget(null); setVerMsg('') }
              else setVerMsg((r && r.error) || '改名失败')
            })
          }

          function renderInline(text) {
            var parts = []
            var regex = /(\*\*[^*]+\*\*|`[^`]+`|\[[^\]\n]+\]\([^)\n]+\))/g
            var last = 0
            var m
            while ((m = regex.exec(text)) !== null) {
              if (m.index > last) parts.push(text.slice(last, m.index))
              var tok = m[0]
              if (tok.charAt(0) === '*') parts.push(React.createElement('strong', { key: 'b' + m.index }, tok.slice(2, -2)))
              else if (tok.charAt(0) === '[') {
                var lm = tok.match(/^\[([^\]]+)\]\(([^)]+)\)/)
                parts.push(React.createElement('a', { key: 'a' + m.index, href: lm[2], target: '_blank', rel: 'noreferrer', style: { color: '#7ab8ff', textDecoration: 'underline' } }, lm[1]))
              }
              else parts.push(React.createElement('code', { key: 'c' + m.index, style: { background: '#212530', padding: '0 6px', borderRadius: 3, fontFamily: 'Menlo, monospace', fontSize: '0.95em' } }, tok.slice(1, -1)))
              last = m.index + tok.length
            }
            if (last < text.length) parts.push(text.slice(last))
            return parts
          }
          function renderLineContent(ln) {
            var hm = ln.match(/^(#{1,6})\s+(.*)$/)
            if (hm) { var lvl = hm[1].length; var size = [26,23,21,19,17,15][lvl-1] || 15; return React.createElement('div', { style: { fontSize: size, fontWeight: 700, margin: lvl <= 2 ? '14px 0 6px' : '8px 0 3px', color: '#fff', lineHeight: 1.3 } }, renderInline(hm[2])) }
            return React.createElement('span', { style: { whiteSpace: 'pre-wrap' } }, renderInline(ln))
          }
          function renderComment(c) {
            var isRange = c.anchor && c.anchor.kind === 'range'
            var isSolved = c.state !== 'open'
            var showFull = c.state === 'open' || c.state === 'pending-confirm' || !!expanded[c.id]
            var rangeTag = isRange ? React.createElement('span', { key: 'rt', onClick: function (e) { e.stopPropagation(); highlightRange(c) }, style: { fontSize: 12, color: '#e8b34a', border: '1px solid rgba(232,179,74,0.5)', borderRadius: 4, padding: '0 6px', marginLeft: 6, cursor: 'pointer' } }, '↔ ' + Math.min(c.anchor.startLine, c.anchor.endLine) + '~' + Math.max(c.anchor.startLine, c.anchor.endLine) + ' 行') : null
            var typeTag = c.type && c.type !== 'issue' ? React.createElement('span', { key: 'ty', style: { fontSize: 11, color: c.type === 'notification' ? '#7ab8ff' : '#8a90a3', marginLeft: 6 } }, c.type === 'notification' ? '通知' : '讨论') : null
            var roundTag = (c.roundCount || 0) > 0 ? React.createElement('span', { key: 'rd', style: { fontSize: 11, color: (c.roundCount || 0) > 5 ? '#d98080' : '#8a90a3', marginLeft: 6 } }, (c.roundCount || 0) + ' 轮') : null
            var overdueTag = null
            if (c.state === 'pending-confirm' && c.resolve && c.resolve.at) {
              var _el = Date.now() - new Date(c.resolve.at).getTime()
              var _th = (c.author === 'User') ? 60 * 60 * 1000 : 5 * 60 * 1000
              if (_el >= _th) overdueTag = React.createElement('span', { key: 'od', style: { fontSize: 11, color: '#d98080', border: '1px solid rgba(217,128,128,0.6)', borderRadius: 4, padding: '0 6px', marginLeft: 6 } }, '⏱ 超时')
            }
            if (!showFull) {
              var solTag = c.state === 'resolved' ? React.createElement('span', { key: 'sg', style: { fontSize: 12, color: '#58c07a', marginLeft: 6 } }, '已解决') : c.state === 'overridden' ? React.createElement('span', { key: 'sg', style: { fontSize: 12, color: '#e8b34a', marginLeft: 6 } }, '已强制通过') : React.createElement('span', { key: 'sg', style: { fontSize: 12, color: '#e8b34a', marginLeft: 6 } }, '已解决')
              return React.createElement('div', { key: c.id, style: { margin: '4px 0', padding: '6px 12px', background: '#1b1e27', borderRadius: 5, fontSize: 13, color: '#8a90a3', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }, onClick: function () { toggleExpand(c.id) } }, [React.createElement('span', { key: 'a', style: { color: '#7ab8ff' } }, c.author), typeTag, React.createElement('span', { key: 't', style: { flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, c.text), rangeTag, roundTag, overdueTag, solTag, React.createElement('span', { key: 'x', style: { color: '#8a90a3', fontSize: 13 } }, '▸')])
            }
            var thread = (c.thread || []).map(function (m, i) { return React.createElement('div', { key: i, style: { margin: '4px 0', fontSize: 13, color: '#c8cdd8', paddingLeft: 8, borderLeft: '2px solid #3a4356' } }, '[' + m.by + '] ' + m.text) })
            var stTag = c.state === 'open' ? React.createElement('span', { key: 's', style: { fontSize: 12, color: '#d98080', marginLeft: 6 } }, '未解决') : c.state === 'pending-confirm' ? React.createElement('span', { key: 's', style: { fontSize: 12, color: '#e8b34a', marginLeft: 6 } }, '待复核') : c.state === 'resolved' ? React.createElement('span', { key: 's', style: { fontSize: 12, color: '#58c07a', marginLeft: 6 } }, '已解决') : React.createElement('span', { key: 's', style: { fontSize: 12, color: '#e8b34a', marginLeft: 6 } }, '已强制通过')
            var head = [React.createElement('span', { key: 'a', style: { color: '#7ab8ff' } }, c.author), typeTag, React.createElement('span', { key: 't', style: { marginLeft: 6 } }, c.text), rangeTag, roundTag, overdueTag, stTag]
            if (isSolved) head.push(React.createElement('span', { key: 'fld', style: { marginLeft: 'auto', color: '#8a90a3', fontSize: 13, cursor: 'pointer' }, onClick: function () { toggleExpand(c.id) } }, '收起▴'))
            var resolveInfo = null
            if (c.resolve) {
              resolveInfo = React.createElement('div', { key: 'ri', style: { margin: '6px 0', padding: '6px 10px', background: '#1b1e27', borderRadius: 4, fontSize: 12, color: '#8a90a3' } }, ['[' + (c.resolve.path === 'agree' ? '协商一致' : '修复') + '] ' + (c.resolve.note || '') + ' — ' + (c.resolve.by || '')])
            }
            var actBtns = [React.createElement('button', { key: 's', style: btnSmall, onClick: function () { doReply(c.id) } }, '回复')]
            if (c.state === 'open') {
              actBtns.push(React.createElement('button', { key: 'rs', style: btnSmallGreen, onClick: function () { setResolveTarget({ id: c.id, path: 'fix' }) } }, '解决(修复)'))
              actBtns.push(React.createElement('button', { key: 'ra', style: btnSmallBlue, onClick: function () { setResolveTarget({ id: c.id, path: 'agree' }) } }, '解决(一致)'))
            }
            if (c.state === 'pending-confirm') {
              actBtns.push(React.createElement('button', { key: 'cf', style: btnSmallGreen, onClick: function () { doConfirmReview(c.id) } }, '复核通过'))
              actBtns.push(React.createElement('button', { key: 'rp', style: btnSmallRed, onClick: function () { setReopenTarget(c.id) } }, '复核不通过'))
            }
            var resolveBox = null
            if (resolveTarget && resolveTarget.id === c.id) {
              resolveBox = React.createElement('div', { key: 'rb', style: { display: 'flex', gap: 6, marginTop: 6, alignItems: 'center' } }, [React.createElement('input', { key: 'in', placeholder: resolveTarget.path === 'agree' ? '一致结论（必填）' : '修复说明（必填）', value: resolveNote, onChange: function (e) { setResolveNote(e.target.value) }, style: { flex: 1, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 5, padding: '6px 10px', color: '#e6e8ef', fontSize: 13 } }), React.createElement('button', { key: 'go', style: btnSmallGreen, onClick: function () { doResolve(c.id) } }, '提交')])
            }
            var reopenBox = null
            if (reopenTarget === c.id) {
              reopenBox = React.createElement('div', { key: 'rb2', style: { display: 'flex', gap: 6, marginTop: 6, alignItems: 'center' } }, [React.createElement('input', { key: 'in', placeholder: '复核不通过理由（必填）', value: resolveNote, onChange: function (e) { setResolveNote(e.target.value) }, style: { flex: 1, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 5, padding: '6px 10px', color: '#e6e8ef', fontSize: 13 } }), React.createElement('button', { key: 'go', style: btnSmallRed, onClick: function () { doReopen(c.id) } }, '重开')])
            }
            return React.createElement('div', { key: c.id, style: { margin: '5px 0', padding: '8px 12px', background: '#212530', borderRadius: 6, fontSize: 14 } }, [
              React.createElement('div', { key: 'h', style: { display: 'flex', alignItems: 'center', flexWrap: 'wrap' } }, head),
              resolveInfo,
              thread,
              React.createElement('div', { key: 'act', style: { display: 'flex', gap: 6, alignItems: 'center', marginTop: 6, flexWrap: 'wrap' } }, actBtns),
              resolveBox,
              reopenBox
            ])
          }
          function isTableSep(s) { return s.indexOf('-') >= 0 && /^\s*\|?[\s:|-]+\|?\s*$/.test(s) }
          function splitRow(s) { return String(s).trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map(function (c) { return c.trim() }) }
          function renderTable(rows, key) {
            var head = rows[0] || []
            var body = rows.slice(1)
            var thead = React.createElement('thead', { key: 'th' }, React.createElement('tr', { key: 'tr' }, head.map(function (c, ci) { return React.createElement('th', { key: ci, style: { padding: '6px 10px', border: '1px solid #2b2f3d', textAlign: 'left', fontSize: 13, color: '#c8cdd8', background: '#212530', fontWeight: 600 } }, renderInline(c)) })))
            var tbody = React.createElement('tbody', { key: 'tb' }, body.map(function (row, ri) { return React.createElement('tr', { key: 'r' + ri }, row.map(function (c, ci) { return React.createElement('td', { key: ci, style: { padding: '6px 10px', border: '1px solid #2b2f3d', fontSize: 13, color: '#e6e8ef' } }, renderInline(c)) })) }))
            return React.createElement('table', { key: key, style: { borderCollapse: 'collapse', margin: '8px 0', width: '100%' } }, [thead, tbody])
          }
          function renderMd(text) {
            var lines = String(text || '').split('\n')
            var nodes = []
            var para = []
            function flushPara() {
              if (para.length) { nodes.push(React.createElement('div', { key: 'p' + nodes.length, style: { margin: '6px 0', whiteSpace: 'pre-wrap' } }, para.slice())); para = [] }
            }
            var i = 0
            while (i < lines.length) {
              var ln = lines[i]
              var bm = ln.match(/^```(\w*)\s*$/)
              if (bm) {
                flushPara()
                var code = []
                i++
                while (i < lines.length && !/^```\s*$/.test(lines[i])) { code.push(lines[i]); i++ }
                i++
                nodes.push(React.createElement('div', { key: 'b' + nodes.length, style: { margin: '8px 0', padding: '10px 14px', background: '#181b24', border: '1px solid #2b2f3d', borderLeft: '3px solid #7ab8ff', borderRadius: 6, fontFamily: 'Menlo, Consolas, monospace', fontSize: 13, color: '#c8cdd8', whiteSpace: 'pre-wrap' } }, code.join('\n')))
              } else if (/^\s*\|/.test(ln) && i + 1 < lines.length && isTableSep(lines[i + 1])) {
                flushPara()
                var tRows = [splitRow(ln)]
                i += 2
                while (i < lines.length && /^\s*\|/.test(lines[i])) { tRows.push(splitRow(lines[i])); i++ }
                nodes.push(renderTable(tRows, 't' + nodes.length))
                continue
              } else if (/^#{1,6}\s/.test(ln)) {
                flushPara()
                var hm = ln.match(/^(#{1,6})\s+(.*)$/)
                var lvl = hm[1].length
                nodes.push(React.createElement('div', { key: 'h' + nodes.length, style: { fontSize: [26,23,21,19,17,15][lvl-1] || 15, fontWeight: 700, margin: '10px 0 4px', color: '#fff' } }, renderInline(hm[2])))
              } else if (ln.trim() === '') {
                flushPara()
              } else if (/^\s*[-*▪]\s+/.test(ln)) {
                flushPara()
                var item = ln.replace(/^\s*[-*▪]\s+/, '')
                nodes.push(React.createElement('div', { key: 'l' + nodes.length, style: { margin: '2px 0', paddingLeft: 16 } }, '▪ ', renderInline(item)))
              } else {
                para.push(React.createElement('span', { key: 'l' + para.length }, renderInline(ln)))
                para.push(React.createElement('span', { key: 'br' + para.length }, '\n'))
              }
              i++
            }
            flushPara()
            return nodes
          }

          var statusCols = ['待启动', '进行中', '已完成', '已取消']
          var statusColor = { '待启动': '#8a90a3', '进行中': '#e8b34a', '已完成': '#58c07a', '已取消': '#d98080' }
          var verStateColor = { '待启动': '#8a90a3', '进行中': '#e8b34a', '已发布': '#58c07a', '已废弃': '#d98080' }
          var priColor = { 'P0': '#d98080', 'P1': '#e8a34a', 'P2': '#7ab8ff', 'P3': '#8a90a3' }
          var roles = ['User', 'Leader', 'Investigator', 'Architect', 'RD', 'PM', 'FE', 'QA']
          var filteredReqs = reqList.filter(function (r) { return (verFilter === '__all__' || r.versionId === verFilter) && (priFilter === '__all__' || r.priority === priFilter) })
          var verNameOf = function (vid) { var v = verList.filter(function (x) { return x.id === vid })[0]; return v ? v.name : (vid || '待分配') }
          var verStats = function (vid) { var st = { '待启动': 0, '进行中': 0, '已完成': 0, '已取消': 0 }; reqList.forEach(function (r) { if (r.versionId === vid && st[r.status] !== undefined) st[r.status]++ }); return st }
          var verCount = function (st) { return st['待启动'] + st['进行中'] + st['已完成'] + st['已取消'] }
          var stColors = [['待启动', '#8a90a3'], ['进行中', '#e8b34a'], ['已完成', '#58c07a'], ['已取消', '#d98080']]
          var fmtTime = function (t) { var s = String(t || '').replace('T', ' '); return s.length > 16 ? s.slice(0, 16) : (s || '—') }

          var verOptions = [React.createElement('option', { key: '__all__', value: '__all__' }, '全部')].concat(verList.map(function (v) { return React.createElement('option', { key: v.id, value: v.id }, v.name) }))
          var priOptions = ['__all__', 'P0', 'P1', 'P2', 'P3'].map(function (p) { return React.createElement('option', { key: p, value: p }, p === '__all__' ? '全部' : p) })
          var topbar = React.createElement('div', { style: { display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 } }, [
            React.createElement('span', { key: 't', style: { fontSize: 19, fontWeight: 600 } }, '项目'),
            React.createElement('span', { key: 'w', style: { fontSize: 13, padding: '3px 10px', borderRadius: 12, background: '#212530', border: '1px solid #2b2f3d', color: '#8a90a3' } }, 'test-flow'),
            React.createElement('span', { key: 'fl', style: { marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 } }, [
              React.createElement('span', { key: 'vl', style: { fontSize: 13, color: '#8a90a3' } }, '版本'),
              React.createElement('select', { key: 'vs', value: verFilter, onChange: function (e) { setVerFilter(e.target.value) }, style: { fontSize: 13, padding: '5px 10px', borderRadius: 6, background: '#212530', border: '1px solid #2b2f3d', color: '#e6e8ef', outline: 'none' } }, verOptions),
              React.createElement('span', { key: 'pl', style: { fontSize: 13, color: '#8a90a3' } }, '优先级'),
              React.createElement('select', { key: 'ps', value: priFilter, onChange: function (e) { setPriFilter(e.target.value) }, style: { fontSize: 13, padding: '5px 10px', borderRadius: 6, background: '#212530', border: '1px solid #2b2f3d', color: '#e6e8ef', outline: 'none' } }, priOptions),
              React.createElement('button', { key: 'kb', style: kind === 'board' ? Object.assign({}, btnSmall, { background: 'rgba(232,179,74,0.15)', borderColor: '#e8b34a', color: '#e8b34a' }) : btnSmall, onClick: function () { setKind('board') } }, '看板'),
              React.createElement('button', { key: 'kl', style: kind === 'list' ? Object.assign({}, btnSmall, { background: 'rgba(232,179,74,0.15)', borderColor: '#e8b34a', color: '#e8b34a' }) : btnSmall, onClick: function () { setKind('list') } }, '列表'),
              React.createElement('button', { key: 'vm', style: btnSmallBlue, onClick: openVerMgr }, '版本管理'),
              React.createElement('button', { key: 'nb', style: btnSmallGreen, onClick: openNewForm }, '＋ 新建需求')
            ])
          ])

          function field(k, label, control) {
            return React.createElement('div', { key: k, style: { marginBottom: 12 } }, [
              React.createElement('div', { key: 'l', style: { fontSize: 12, color: '#8a90a3', marginBottom: 4 } }, label),
              control
            ])
          }
          var formModal = null
          if (form) {
            var fTitle = React.createElement('input', { key: 'i', placeholder: '需求标题（必填）', value: form.title, onChange: function (e) { setFormField('title', e.target.value) }, style: inputStyle })
            var fDesc = React.createElement('textarea', { key: 'i', placeholder: '支持 Markdown，可写背景/目标/验收点，或贴文档链接…', value: form.description, onChange: function (e) { setFormField('description', e.target.value) }, style: Object.assign({}, inputStyle, { minHeight: 84, resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.5 }) })
            var fCat = React.createElement('select', { key: 's', value: form.category, onChange: function (e) { setFormField('category', e.target.value) }, style: selStyle }, ['需求任务', '规划审批任务', '版本规划任务'].map(function (c) { return React.createElement('option', { key: c, value: c }, c) }))
            var fVer = React.createElement('select', { key: 's', value: form.versionId, onChange: function (e) { setFormField('versionId', e.target.value) }, style: selStyle }, verList.length ? verList.map(function (v) { return React.createElement('option', { key: v.id, value: v.id }, v.name) }) : React.createElement('option', { key: 'vb', value: 'v-backlog' }, '待分配'))
            var fPri = React.createElement('select', { key: 's', value: form.priority, onChange: function (e) { setFormField('priority', e.target.value) }, style: selStyle }, ['P0', 'P1', 'P2', 'P3'].map(function (p) { return React.createElement('option', { key: p, value: p }, p) }))
            var fStatus = React.createElement('select', { key: 's', value: form.status, onChange: function (e) { setFormField('status', e.target.value) }, style: selStyle }, statusCols.map(function (s) { return React.createElement('option', { key: s, value: s }, s) }))
            var fOwner = React.createElement('input', { key: 'i', list: 'wb-roles', value: form.owner, onChange: function (e) { setFormField('owner', e.target.value) }, style: inputStyle })
            var roleList = React.createElement('datalist', { key: 'dl', id: 'wb-roles' }, roles.map(function (r) { return React.createElement('option', { key: r, value: r }) }))
            formModal = React.createElement('div', { key: 'fm', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 } }, React.createElement('div', { key: 'm', style: { width: 480, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, overflow: 'hidden' } }, [
              React.createElement('div', { key: 'h', style: { display: 'flex', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid #2b2f3d', fontSize: 15, fontWeight: 600 } }, [form.mode === 'new' ? '新建需求' : '编辑需求', React.createElement('span', { key: 'x', style: { marginLeft: 'auto', cursor: 'pointer', color: '#8a90a3' }, onClick: closeForm }, '✕')]),
              React.createElement('div', { key: 'b', style: { padding: 16 } }, [
                field('f1', '标题 *', fTitle),
                field('f2', '描述（支持 Markdown）', fDesc),
                field('f3', '类别', fCat),
                React.createElement('div', { key: 'r1', style: { display: 'flex', gap: 10 } }, [field('f4a', '版本', fVer), field('f4b', '优先级', fPri)]),
                React.createElement('div', { key: 'r2', style: { display: 'flex', gap: 10 } }, [field('f5a', '状态', fStatus), field('f5b', '负责人', fOwner)]),
                roleList
              ]),
              React.createElement('div', { key: 'f', style: { display: 'flex', gap: 8, padding: '12px 16px', borderTop: '1px solid #2b2f3d', alignItems: 'center' } }, [
                form.mode === 'edit' ? React.createElement('button', { key: 'd', style: btnSmallRed, onClick: function () { askDelete({ reqId: form.reqId, title: form.title }) } }, '删除') : React.createElement('span', { key: 'sp' }),
                React.createElement('span', { key: 'sp2', style: { flex: 1 } }),
                React.createElement('button', { key: 'c', style: btnSmall, onClick: closeForm }, '取消'),
                React.createElement('button', { key: 's', style: btnSmallGreen, onClick: saveForm }, '保存')
              ])
            ]))
          }
          var delModal = null
          if (delReq) {
            delModal = React.createElement('div', { key: 'dm', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 210 } }, React.createElement('div', { key: 'm', style: { width: 400, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, overflow: 'hidden' } }, [
              React.createElement('div', { key: 'h', style: { padding: '12px 16px', borderBottom: '1px solid #2b2f3d', fontSize: 15, fontWeight: 600 } }, '删除需求'),
              React.createElement('div', { key: 'b', style: { padding: 16, fontSize: 13, lineHeight: 1.7 } }, '删除后不可恢复，确定删除需求 ', React.createElement('b', { key: 'q', style: { color: '#d98080' } }, '「' + delReq.title + '」'), '？'),
              React.createElement('div', { key: 'f', style: { display: 'flex', gap: 8, justifyContent: 'flex-end', padding: '12px 16px', borderTop: '1px solid #2b2f3d' } }, [
                React.createElement('button', { key: 'c', style: btnSmall, onClick: cancelDelete }, '取消'),
                React.createElement('button', { key: 'd', style: btnSmallRed, onClick: confirmDelete }, '确认删除')
              ])
            ]))
          }

          var verMgrModal = null
          if (verMgrOpen) {
            var verRows = verList.map(function (v) {
              var st = verStats(v.id)
              var total = verCount(st)
              var segs = []
              if (total === 0) segs.push(React.createElement('i', { key: 'e', style: { width: '100%', background: '#2b2f3d' } }))
              else stColors.forEach(function (sc) { if (st[sc[0]] > 0) segs.push(React.createElement('i', { key: sc[0], style: { width: (st[sc[0]] / total * 100) + '%', background: sc[1] } })) })
              var vsc = verStateColor[v.state] || '#8a90a3'
              var badgeBg = v.state === '待启动' ? 'rgba(138,144,163,0.2)' : v.state === '进行中' ? 'rgba(232,179,74,0.18)' : v.state === '已发布' ? 'rgba(88,192,122,0.18)' : 'rgba(217,128,128,0.18)'
              var stateBadge = v.isBacklog
                ? React.createElement('span', { key: 'st', style: { fontSize: 11, padding: '2px 9px', borderRadius: 10, background: 'rgba(138,144,163,0.2)', color: '#8a90a3', border: '1px solid rgba(138,144,163,0.4)' } }, '待分配池')
                : React.createElement('span', { key: 'st', style: { fontSize: 11, padding: '2px 9px', borderRadius: 10, minWidth: 52, textAlign: 'center', background: badgeBg, color: vsc, border: '1px solid ' + vsc } }, v.state)
              var actNodes = []
              if (!v.isBacklog) {
                if (v.state === '待启动' || v.state === '进行中') actNodes.push(React.createElement('button', { key: 'a', onClick: function () { advanceVer(v) }, style: btnSmallBlue }, '▶ 推进为' + (v.state === '待启动' ? '进行中' : '已发布')))
                if (v.state !== '已废弃') actNodes.push(React.createElement('button', { key: 'd', onClick: function () { setVerDepTarget({ id: v.id, name: v.name }) }, style: btnSmallRed }, '✖ 废弃'))
                actNodes.push(React.createElement('button', { key: 'r', onClick: function () { setVerRenameTarget({ id: v.id, name: v.name, newName: v.name }) }, style: btnSmall }, '✎ 改名'))
              } else {
                actNodes.push(React.createElement('span', { key: 'lk', style: { fontSize: 11, color: '#8a90a3' } }, '池 · 只读'))
              }
              var progItems = stColors.map(function (sc) { return React.createElement('span', { key: sc[0], style: { display: 'inline-flex', alignItems: 'center', gap: 4 } }, [React.createElement('span', { key: 'd', style: { width: 8, height: 8, borderRadius: '50%', background: sc[1] } }), sc[0] + ' ' + st[sc[0]]]) })
              return React.createElement('div', { key: v.id, style: { display: 'flex', alignItems: 'center', gap: 12, padding: '12px 14px', border: '1px solid #2b2f3d', borderRadius: 8, marginBottom: 8, background: '#212530', opacity: v.state === '已废弃' ? 0.65 : 1 } }, [
                React.createElement('span', { key: 'n', style: { fontSize: 14, fontWeight: 600, minWidth: 60 } }, v.name),
                stateBadge,
                React.createElement('span', { key: 'p', style: { display: 'flex', gap: 10, fontSize: 12, color: '#8a90a3', flex: 1, flexWrap: 'wrap', alignItems: 'center' } }, [
                  React.createElement('span', { key: 'items', style: { display: 'flex', gap: 10 } }, progItems),
                  React.createElement('div', { key: 'bar', style: { height: 6, borderRadius: 3, background: '#1b1e27', overflow: 'hidden', width: 120, display: 'flex', marginTop: 4 } }, segs),
                  React.createElement('span', { key: 'c', style: { color: '#4a4f60' } }, total + ' 条')
                ]),
                React.createElement('span', { key: 'act', style: { display: 'flex', gap: 6, alignItems: 'center' } }, actNodes)
              ])
            })
            verMgrModal = React.createElement('div', { key: 'vm', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 } }, React.createElement('div', { key: 'm', style: { width: 700, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, overflow: 'hidden' } }, [
              React.createElement('div', { key: 'h', style: { display: 'flex', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid #2b2f3d', fontSize: 15, fontWeight: 600 } }, ['版本管理', React.createElement('span', { key: 'c', style: { color: '#8a90a3', fontWeight: 400, fontSize: 12, marginLeft: 8 } }, verList.length + ' 个版本 · ' + reqList.length + ' 条需求'), React.createElement('span', { key: 'x', style: { marginLeft: 'auto', cursor: 'pointer', color: '#8a90a3' }, onClick: closeVerMgr }, '✕')]),
              React.createElement('div', { key: 'b', style: { padding: 16, maxHeight: '55vh', overflowY: 'auto' } }, [
                React.createElement('div', { key: 'nv', style: { display: 'flex', gap: 8, marginBottom: 14 } }, [
                  React.createElement('input', { key: 'i', placeholder: '新版本名，如 v1.2 / 2.0 迭代', value: verNameInput, onChange: function (e) { setVerNameInput(e.target.value) }, onKeyDown: function (e) { if (e.key === 'Enter') createVer() }, style: Object.assign({}, inputStyle, { flex: 1 }) }),
                  React.createElement('button', { key: 'b', style: btnSmallGreen, onClick: createVer }, '＋ 新建版本')
                ]),
                verRows,
                verMsg ? React.createElement('div', { key: 'msg', style: { marginTop: 10, padding: '8px 12px', borderRadius: 6, background: 'rgba(232,179,74,0.08)', border: '1px solid rgba(232,179,74,0.4)', color: '#e8b34a', fontSize: 13 } }, verMsg) : null
              ]),
              React.createElement('div', { key: 'f', style: { display: 'flex', gap: 8, justifyContent: 'flex-end', padding: '12px 16px', borderTop: '1px solid #2b2f3d' } }, [
                React.createElement('button', { key: 'c', style: btnSmall, onClick: closeVerMgr }, '关闭')
              ])
            ]))
          }
          var verDepModal = null
          if (verDepTarget) {
            var dcount = verStats(verDepTarget.id)
            var dtotal = verCount(dcount)
            verDepModal = React.createElement('div', { key: 'vd', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 220 } }, React.createElement('div', { key: 'm', style: { width: 440, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, overflow: 'hidden' } }, [
              React.createElement('div', { key: 'h', style: { padding: '12px 16px', borderBottom: '1px solid #2b2f3d', fontSize: 15, fontWeight: 600 } }, '废弃版本'),
              React.createElement('div', { key: 'b', style: { padding: 16, fontSize: 13, lineHeight: 1.7 } }, dtotal > 0
                ? '该版本下还有 ' + dtotal + ' 条需求，请先将需求移出该版本，才能废弃。'
                : React.createElement('span', null, '确认废弃 ', React.createElement('b', { key: 'q', style: { color: '#d98080' } }, '「' + verDepTarget.name + '」'), '？废弃后不可恢复。')),
              React.createElement('div', { key: 'f', style: { display: 'flex', gap: 8, justifyContent: 'flex-end', padding: '12px 16px', borderTop: '1px solid #2b2f3d' } }, [
                React.createElement('button', { key: 'c', style: btnSmall, onClick: function () { setVerDepTarget(null) } }, dtotal > 0 ? '知道了' : '取消'),
                dtotal > 0 ? null : React.createElement('button', { key: 'd', style: btnSmallRed, onClick: doDeprecate }, '确认废弃')
              ])
            ]))
          }
          var verPubModal = null
          if (verPubTarget) {
            verPubModal = React.createElement('div', { key: 'vp', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 220 } }, React.createElement('div', { key: 'm', style: { width: 440, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, overflow: 'hidden' } }, [
              React.createElement('div', { key: 'h', style: { padding: '12px 16px', borderBottom: '1px solid #2b2f3d', fontSize: 15, fontWeight: 600 } }, '发布版本'),
              React.createElement('div', { key: 'b', style: { padding: 16, fontSize: 13, lineHeight: 1.7 } }, '该版本下还有 ', React.createElement('b', { key: 'n', style: { color: '#e8b34a' } }, verPubTarget.unfinished), ' 条未完成需求（待启动/进行中）。', '确认发布 ', React.createElement('b', { key: 'q', style: { color: '#d98080' } }, '「' + verPubTarget.name + '」'), '？'),
              React.createElement('div', { key: 'f', style: { display: 'flex', gap: 8, justifyContent: 'flex-end', padding: '12px 16px', borderTop: '1px solid #2b2f3d' } }, [
                React.createElement('button', { key: 'c', style: btnSmall, onClick: function () { setVerPubTarget(null) } }, '取消'),
                React.createElement('button', { key: 'd', style: btnSmallGreen, onClick: function () { var v = verList.filter(function (x) { return x.id === verPubTarget.id })[0]; if (v) doAdvance(v) } }, '确认发布')
              ])
            ]))
          }
          var verRenameModal = null
          if (verRenameTarget) {
            verRenameModal = React.createElement('div', { key: 'vr', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 220 } }, React.createElement('div', { key: 'm', style: { width: 400, background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, overflow: 'hidden' } }, [
              React.createElement('div', { key: 'h', style: { padding: '12px 16px', borderBottom: '1px solid #2b2f3d', fontSize: 15, fontWeight: 600 } }, '版本改名'),
              React.createElement('div', { key: 'b', style: { padding: 16 } }, React.createElement('input', { key: 'i', value: verRenameTarget.newName, onChange: function (e) { var n = Object.assign({}, verRenameTarget); n.newName = e.target.value; setVerRenameTarget(n) }, onKeyDown: function (e) { if (e.key === 'Enter') doRenameVer() }, style: inputStyle })),
              React.createElement('div', { key: 'f', style: { display: 'flex', gap: 8, justifyContent: 'flex-end', padding: '12px 16px', borderTop: '1px solid #2b2f3d' } }, [
                React.createElement('button', { key: 'c', style: btnSmall, onClick: function () { setVerRenameTarget(null) } }, '取消'),
                React.createElement('button', { key: 's', style: btnSmallGreen, onClick: doRenameVer }, '保存')
              ])
            ]))
          }

          function reqCard(r) {
            var act = React.createElement('span', { key: 'act', style: { position: 'absolute', top: 6, right: 8, display: 'flex', gap: 4, opacity: hoverId === r.id ? 1 : 0, transition: 'opacity 0.15s' } }, [
              React.createElement('span', { key: 'e', onClick: function (e) { e.stopPropagation(); openEditForm(r) }, style: { fontSize: 11, padding: '1px 7px', borderRadius: 4, cursor: 'pointer', background: '#212530', border: '1px solid #2b2f3d', color: '#8a90a3' } }, '✎'),
              React.createElement('span', { key: 'd', onClick: function (e) { e.stopPropagation(); askDelete(r) }, style: { fontSize: 11, padding: '1px 7px', borderRadius: 4, cursor: 'pointer', background: '#212530', border: '1px solid rgba(217,128,128,0.5)', color: '#d98080' } }, '🗑')
            ])
            return React.createElement('div', { key: r.id, onClick: function () { openDetail(r.id) }, onMouseEnter: function () { setHoverId(r.id) }, onMouseLeave: function () { setHoverId(null) }, style: { position: 'relative', background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 6, padding: '10px 12px', marginBottom: 8, cursor: 'pointer' } }, [
              React.createElement('div', { key: 't', style: { fontSize: 13, fontWeight: 500 } }, r.title),
              React.createElement('div', { key: 'm', style: { fontSize: 11, color: '#8a90a3', marginTop: 5 } }, [React.createElement('span', { key: 'c', style: { display: 'inline-block', fontSize: 10, padding: '1px 6px', borderRadius: 6, background: 'rgba(122,184,255,0.15)', color: '#7ab8ff', marginRight: 6 } }, r.category), React.createElement('span', { key: 'p', style: { display: 'inline-block', fontSize: 10, padding: '1px 6px', borderRadius: 6, background: 'rgba(232,179,74,0.15)', color: priColor[r.priority] || '#8a90a3', marginRight: 6 } }, r.priority || '—'), verNameOf(r.versionId), ' · ', r.owner]),
              act
            ])
          }

          var boardBody = null
          if (kind === 'board') {
            boardBody = React.createElement('div', { style: { display: 'flex', gap: 12, alignItems: 'flex-start', overflowX: 'auto' } }, statusCols.map(function (s) { var rs = filteredReqs.filter(function (r) { return r.status === s }); return React.createElement('div', { key: s, style: { flex: 1, minWidth: 180, background: '#212530', borderRadius: 8, padding: 10 } }, [React.createElement('div', { key: 'h', style: { fontSize: 13, fontWeight: 600, padding: '4px 6px 8px', display: 'flex', alignItems: 'center', gap: 6 } }, [React.createElement('span', { key: 'd', style: { width: 8, height: 8, borderRadius: '50%', background: statusColor[s] } }), s, React.createElement('span', { key: 'n', style: { marginLeft: 'auto', color: '#8a90a3', fontSize: 11 } }, rs.length)]), rs.map(reqCard)]) }))
          } else {
            var listHeaders = ['标题', '类别', '优先级', '版本', '状态', '负责人']
            var colFlex = [3, 1, 0.7, 1, 1, 1]
            var groups = statusCols.map(function (s) { return { status: s, items: filteredReqs.filter(function (r) { return r.status === s }) } }).filter(function (g) { return g.items.length > 0 })
            var headerRow = React.createElement('div', { key: 'hr', style: { display: 'flex', padding: '10px 12px', borderBottom: '1px solid #2b2f3d', color: '#8a90a3', fontWeight: 500, fontSize: 12 } }, listHeaders.map(function (h, i) { return React.createElement('div', { key: h, style: { flex: colFlex[i] + ' 1 0', textAlign: 'left' } }, h) }).concat(React.createElement('div', { key: 'op', style: { flex: '0 0 60px' } })))
            var groupNodes = groups.map(function (g, gi) {
              var rows = g.items.map(function (r) {
                var stc = { '待启动': 'rgba(138,144,163,0.2)', '进行中': 'rgba(232,179,74,0.18)', '已完成': 'rgba(88,192,122,0.18)', '已取消': 'rgba(217,128,128,0.18)' }[r.status]
                var stcol = statusColor[r.status]
                var rAct = React.createElement('span', { key: 'act', style: { flex: '0 0 60px', justifyContent: 'flex-end', display: 'flex', gap: 4, opacity: hoverId === r.id ? 1 : 0, transition: 'opacity 0.15s' } }, [
                  React.createElement('span', { key: 'e', onClick: function (e) { e.stopPropagation(); openEditForm(r) }, style: { fontSize: 11, padding: '1px 7px', borderRadius: 4, cursor: 'pointer', background: '#212530', border: '1px solid #2b2f3d', color: '#8a90a3' } }, '✎'),
                  React.createElement('span', { key: 'd', onClick: function (e) { e.stopPropagation(); askDelete(r) }, style: { fontSize: 11, padding: '1px 7px', borderRadius: 4, cursor: 'pointer', background: '#212530', border: '1px solid rgba(217,128,128,0.5)', color: '#d98080' } }, '🗑')
                ])
                return React.createElement('div', { key: r.id, onClick: function () { openDetail(r.id) }, onMouseEnter: function () { setHoverId(r.id) }, onMouseLeave: function () { setHoverId(null) }, style: { display: 'flex', padding: '10px 12px', borderBottom: '1px solid #1e2230', cursor: 'pointer', alignItems: 'center' } }, [
                  React.createElement('div', { key: 't', style: { flex: colFlex[0] + ' 1 0' } }, r.title),
                  React.createElement('div', { key: 'c', style: { flex: colFlex[1] + ' 1 0' } }, React.createElement('span', { style: { fontSize: 10, padding: '1px 6px', borderRadius: 6, background: 'rgba(122,184,255,0.15)', color: '#7ab8ff' } }, r.category)),
                  React.createElement('div', { key: 'p', style: { flex: colFlex[2] + ' 1 0' } }, React.createElement('span', { style: { fontSize: 10, padding: '1px 6px', borderRadius: 6, background: 'rgba(232,179,74,0.15)', color: priColor[r.priority] || '#8a90a3' } }, r.priority || '—')),
                  React.createElement('div', { key: 'v', style: { flex: colFlex[3] + ' 1 0' } }, verNameOf(r.versionId)),
                  React.createElement('div', { key: 's', style: { flex: colFlex[4] + ' 1 0' } }, React.createElement('span', { style: { fontSize: 11, padding: '2px 8px', borderRadius: 10, background: stc, color: stcol } }, r.status)),
                  React.createElement('div', { key: 'o', style: { flex: colFlex[5] + ' 1 0' } }, r.owner),
                  rAct
                ])
              })
              var sep = gi < groups.length - 1 ? React.createElement('div', { key: 'sep', style: { borderTop: '1px dashed #3a4356', margin: '14px 0 6px' } }) : null
              return React.createElement('div', { key: g.status }, [
                React.createElement('div', { key: 'st', style: { display: 'flex', alignItems: 'center', gap: 6, padding: '12px 12px 6px', fontSize: 13, fontWeight: 600, color: statusColor[g.status] } }, [React.createElement('span', { key: 'd', style: { width: 8, height: 8, borderRadius: '50%', background: statusColor[g.status] } }), g.status, React.createElement('span', { key: 'n', style: { color: '#8a90a3', fontWeight: 400, fontSize: 12 } }, g.items.length + ' 项')]),
                rows,
                sep
              ])
            })
            boardBody = React.createElement('div', { key: 'lst' }, [headerRow].concat(groupNodes))
          }

          if (view === 'board') {
            return React.createElement('div', { style: { padding: 22, height: '100%', overflow: 'auto', color: '#e6e8ef' } }, [topbar, boardBody, formModal, delModal, verMgrModal, verDepModal, verPubModal, verRenameModal])
          }

          if (!req) return React.createElement('div', { style: { padding: 30, color: '#888' } }, '加载中…')

          var infoPanel = null
          if (req.requirement) {
            var rq = req.requirement
            var kvStyle = { display: 'flex', alignItems: 'center', gap: 8, padding: '8px 0', borderBottom: '1px solid #242a38', fontSize: 13 }
            var kvLabel = { color: '#8a90a3', width: 60, flex: '0 0 60px' }
            var kvVal = { color: '#e6e8ef' }
            var infoRows = []
            infoRows.push(React.createElement('div', { key: 'owner', style: kvStyle }, [React.createElement('span', { key: 'l', style: kvLabel }, '负责人'), React.createElement('span', { key: 'v', style: kvVal }, rq.owner || '—')]))
            infoRows.push(React.createElement('div', { key: 'ver', style: kvStyle }, [React.createElement('span', { key: 'l', style: kvLabel }, '版本'), React.createElement('span', { key: 'v', style: kvVal }, verNameOf(rq.versionId))]))
            infoRows.push(React.createElement('div', { key: 'pri', style: kvStyle }, [React.createElement('span', { key: 'l', style: kvLabel }, '优先级'), React.createElement('span', { key: 'v', style: kvVal }, rq.priority ? React.createElement('span', { style: { fontSize: 11, padding: '2px 9px', borderRadius: 10, background: 'rgba(232,179,74,0.15)', color: priColor[rq.priority] || '#8a90a3', border: '1px solid ' + (priColor[rq.priority] || '#2b2f3d') } }, rq.priority) : '—')]))
            infoRows.push(React.createElement('div', { key: 'st', style: kvStyle }, [React.createElement('span', { key: 'l', style: kvLabel }, '状态'), React.createElement('span', { key: 'v', style: kvVal }, rq.status ? React.createElement('span', { style: { fontSize: 11, padding: '2px 9px', borderRadius: 10, background: { '待启动': 'rgba(138,144,163,0.2)', '进行中': 'rgba(232,179,74,0.18)', '已完成': 'rgba(88,192,122,0.18)', '已取消': 'rgba(217,128,128,0.18)' }[rq.status], color: statusColor[rq.status], border: '1px solid ' + statusColor[rq.status] } }, rq.status) : '—')]))
            infoRows.push(React.createElement('div', { key: 'cat', style: kvStyle }, [React.createElement('span', { key: 'l', style: kvLabel }, '类别'), React.createElement('span', { key: 'v', style: kvVal }, rq.category || '—')]))
            infoRows.push(React.createElement('div', { key: 'ca', style: kvStyle }, [React.createElement('span', { key: 'l', style: kvLabel }, '创建时间'), React.createElement('span', { key: 'v', style: kvVal }, fmtTime(rq.createdAt))]))
            infoRows.push(React.createElement('div', { key: 'ua', style: Object.assign({}, kvStyle, { borderBottom: 'none' }) }, [React.createElement('span', { key: 'l', style: kvLabel }, '更新时间'), React.createElement('span', { key: 'v', style: kvVal }, fmtTime(rq.updatedAt))]))
            infoPanel = React.createElement('div', { key: 'info', style: panel }, [React.createElement('div', { key: 'h', style: { fontSize: 15, fontWeight: 600, marginBottom: 6 } }, '需求信息'), infoRows])
          }
          var descPanel = null
          if (req.requirement) {
            var descText = (req.requirement.description || '').trim()
            var descLines = descText ? String(descText).split('\n').length : 0
            var needFold = descLines > 4
            var descBody = descText ? React.createElement('div', { key: 'b', style: { fontSize: 14, lineHeight: 1.8, maxHeight: needFold && !descOpen ? 150 : 'none', overflow: needFold && !descOpen ? 'hidden' : 'visible', position: 'relative' } }, renderMd(descText)) : React.createElement('div', { key: 'e', style: { color: '#8a90a3', fontSize: 13 } }, '暂无描述')
            var descFooter = null
            if (descText && needFold) {
              descFooter = React.createElement('div', { key: 'ft', style: { marginTop: 10, textAlign: 'center' } }, React.createElement('span', { onClick: function () { setDescOpen(!descOpen) }, style: { fontSize: 13, color: '#7ab8ff', cursor: 'pointer', userSelect: 'none' } }, descOpen ? '收起 ▴' : '展开全部 ▾'))
            }
            descPanel = React.createElement('div', { key: 'desc', style: panel }, [
              React.createElement('div', { key: 'h', style: { fontSize: 15, fontWeight: 600, marginBottom: 10 } }, '需求描述'),
              descBody,
              descFooter
            ])
          }
          var detailTop = React.createElement('div', { style: { display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16, flexWrap: 'wrap' } }, [React.createElement('span', { key: 'back', style: { color: '#7ab8ff', cursor: 'pointer', fontSize: 14 }, onClick: backToBoard }, '← 返回看板'), React.createElement('span', { key: 't', style: { fontSize: 19, fontWeight: 600 } }, req.requirement ? req.requirement.title : ''), req.requirement && req.requirement.category ? React.createElement('span', { key: 'c', style: { fontSize: 13, padding: '3px 10px', borderRadius: 12, background: '#212530', border: '1px solid #2b2f3d', color: '#8a90a3' } }, req.requirement.category) : null, req.requirement && req.requirement.priority ? React.createElement('span', { key: 'p', style: { fontSize: 13, padding: '3px 10px', borderRadius: 12, background: '#212530', border: '1px solid ' + (priColor[req.requirement.priority] || '#2b2f3d'), color: priColor[req.requirement.priority] || '#8a90a3' } }, req.requirement.priority) : null, req.requirement && req.requirement.status ? React.createElement('span', { key: 's', style: { fontSize: 13, padding: '3px 10px', borderRadius: 12, background: '#212530', border: '1px solid rgba(232,179,74,0.4)', color: '#e8b34a' } }, req.requirement.status) : null, req.requirement ? React.createElement('button', { key: 'ed', style: btnSmall, onClick: function () { openEditForm(req.requirement) } }, '编辑') : null])

          if (req.missing) {
            return React.createElement('div', { style: { padding: 22, height: '100%', overflow: 'auto', color: '#e6e8ef' } }, [detailTop, infoPanel, descPanel, React.createElement('div', { key: 'm', style: panel }, [React.createElement('div', { key: 't', style: { color: '#8a90a3', fontSize: 14, marginBottom: 12 } }, '该需求暂无流程数据（当前 mock 仅 req-001 有完整流程）'), React.createElement('button', { key: 'b', style: btnSmall, onClick: backToBoard }, '← 返回看板')]), formModal, delModal, verMgrModal, verDepModal, verPubModal, verRenameModal])
          }

          const nodes = (req.flow && req.flow.nodes) || []
          const curNodeId = sel || (req.flow && req.flow.currentNodeId) || (nodes.length && nodes[0].id)
          const curNode = nodes.filter(function (n) { return n.id === curNodeId })[0] || nodes[0]
          const allComments = req.comments || []
          const nodeComments = allComments.filter(function (c) { return c.nodeId === curNodeId })
          const openCnt = nodeComments.filter(function (c) { return c.state === 'open' }).length
          const totalCnt = nodeComments.length
          const gateBlocked = openCnt > 0
          const nodeArtifacts = (req.artifacts || []).filter(function (a) { return a.nodeId === curNodeId })
          function dotColor(n) { if (n.state === '已通过') return '#58c07a'; if (n.state === '待评审' || n.state === '进行中') return '#e8b34a'; return '#4a4f60' }

          function renderFlow(nodes, curNodeId, onSelect) {
            const NW = 118, NH = 38, GAP = 36, VGAP = 30
            const padX = 10, padY = 12
            const centerY = 78
            const topY = centerY - 66
            const botY = centerY + 28
            const groups = []
            let cur = null
            nodes.forEach(function (n) {
              if (n.parallelGroup) {
                if (!cur || cur.group !== n.parallelGroup) { cur = { group: n.parallelGroup, items: [] }; groups.push(cur) }
                cur.items.push(n)
              } else {
                cur = null
                groups.push({ group: null, items: [n] })
              }
            })
            let x = padX
            groups.forEach(function (g) {
              if (g.items.length <= 1) {
                g.width = NW; g.x = x; g.byB = null; g.branches = []
                x += NW + GAP
              } else {
                const byB = {}
                g.items.forEach(function (n) { (byB[n.branch] = byB[n.branch] || []).push(n) })
                g.byB = byB
                g.branches = Object.keys(byB)
                const maxC = Math.max.apply(null, g.branches.map(function (b) { return byB[b].length }))
                g.width = maxC * (NW + GAP) - GAP
                g.x = x
                x += g.width + GAP
              }
            })
            const W = x + padX
            const H = botY + NH + padY
            const placed = []
            const paths = []
            const gPlaced = []
            groups.forEach(function (g) {
              const items = []
              if (g.items.length <= 1) {
                items.push({ n: g.items[0], x: g.x, y: centerY - NH / 2, w: NW, h: NH, col: null })
              } else {
                g.branches.forEach(function (b) {
                  const y = b === 'rd' ? topY : botY
                  g.byB[b].forEach(function (n, i) {
                    items.push({ n: n, x: g.x + i * (NW + GAP), y: y, w: NW, h: NH, col: b })
                  })
                })
              }
              items.forEach(function (p) { placed.push(p) })
              for (let k = 0; k < items.length - 1; k++) {
                const a = items[k], b = items[k + 1]
                if (a.y === b.y) paths.push('M' + (a.x + a.w) + ',' + (a.y + a.h / 2) + ' H' + b.x)
              }
              gPlaced.push(items)
            })
            function centerOf(p) { return { x: p.x + p.w / 2, y: p.y + p.h / 2 } }
            function rightOf(p) { return { x: p.x + p.w, y: p.y + p.h / 2 } }
            function leftOf(p) { return { x: p.x, y: p.y + p.h / 2 } }
            function pathLine(a, b) {
              if (a.y === b.y) paths.push('M' + a.x + ',' + a.y + ' H' + b.x)
              else { const jx = Math.round((a.x + b.x) / 2); paths.push('M' + a.x + ',' + a.y + ' H' + jx + ' V' + b.y + ' H' + b.x) }
            }
            gPlaced.forEach(function (items, gi) {
              if (gi === 0) return
              const prevItems = gPlaced[gi - 1]
              if (prevItems.length === 1 && items.length === 1) {
                pathLine(rightOf(prevItems[0]), leftOf(items[0]))
              } else if (prevItems.length === 1 && items.length > 1) {
                const from = rightOf(prevItems[0])
                items.forEach(function (p) {
                  const jx = Math.round((from.x + leftOf(p).x) / 2)
                  paths.push('M' + from.x + ',' + from.y + ' H' + jx + ' V' + leftOf(p).y + ' H' + leftOf(p).x)
                })
              } else if (prevItems.length > 1 && items.length === 1) {
                const to = leftOf(items[0])
                prevItems.forEach(function (p) {
                  const jx = Math.round((rightOf(p).x + to.x) / 2)
                  paths.push('M' + rightOf(p).x + ',' + rightOf(p).y + ' H' + jx + ' V' + to.y + ' H' + to.x)
                })
              } else {
                items.forEach(function (p, k) {
                  const from = rightOf(prevItems[k])
                  pathLine(from, leftOf(p))
                })
              }
            })
            const nodeEls = placed.map(function (p) {
              const isSel = p.n.id === curNodeId
              const colEdge = p.col === 'rd' ? '#7ab8ff' : (p.col === 'fe' ? '#58c07a' : null)
              const stroke = isSel ? '#e8b34a' : (colEdge || '#2b2f3d')
              const strokeW = isSel ? 2.4 : 1
              const fill = isSel ? 'rgba(232,179,74,0.10)' : '#1b1e27'
              const color = dotColor(p.n)
              const label = p.col === 'rd' ? 'RD ⇄ 后端' : (p.col === 'fe' ? 'FE ⇄ 前端' : null)
              return React.createElement('g', { key: 'g' + p.n.id }, [
                label ? React.createElement('text', { key: 'lb', x: p.x + p.w / 2, y: p.y - 8, textAnchor: 'middle', fontSize: 10.5, fontWeight: 600, fill: colEdge }) : null,
                React.createElement('rect', { key: 'r', x: p.x, y: p.y, width: p.w, height: p.h, rx: 7, fill: fill, stroke: stroke, strokeWidth: strokeW, cursor: 'pointer', onClick: function () { onSelect(p.n.id) } }),
                React.createElement('circle', { key: 'd', cx: p.x + 12, cy: p.y + p.h / 2, r: 4, fill: color }),
                React.createElement('text', { key: 't', x: p.x + p.w / 2 + 6, y: p.y + p.h / 2 + 4, textAnchor: 'middle', fontSize: 11.5, fill: '#e6e8ef', pointerEvents: 'none' }, p.n.key),
                p.n.rolledBackAt && p.n.rolledBackAt.length ? React.createElement('text', { key: 'rb', x: p.x + p.w - 10, y: p.y + p.h / 2 + 4, textAnchor: 'end', fontSize: 9.5, fill: '#d98080', pointerEvents: 'none' }, '↩' + p.n.rolledBackAt.length) : null
              ])
            })
            const pathEls = paths.map(function (d, i) {
              return React.createElement('path', { key: 'p' + i, d: d, stroke: '#3a4356', strokeWidth: 1.6, fill: 'none' })
            })
            return React.createElement('svg', { width: W, height: H, viewBox: '0 0 ' + W + ' ' + H, style: { display: 'block' } }, pathEls.concat(nodeEls))
          }
          var stepList = React.createElement('div', { style: { width: '100%', background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 8, padding: 12, overflowX: 'auto' } }, renderFlow(nodes, curNodeId, function (id) { setSel(id); setOpenRole('') }))

          var meta = React.createElement('div', { style: panel }, [React.createElement('div', { key: 't', style: { fontSize: 18, fontWeight: 600, marginBottom: 12 } }, (curNode && curNode.key) || ''), React.createElement('div', { key: 'kv', style: { display: 'flex', gap: 22, flexWrap: 'wrap', fontSize: 14, color: '#8a90a3' } }, [React.createElement('span', { key: 'b' }, React.createElement('b', { style: { color: '#e6e8ef', fontWeight: 500 } }, '负责人 '), curNode && curNode.ownerRole), React.createElement('span', { key: 'c' }, React.createElement('b', { style: { color: '#e6e8ef', fontWeight: 500 } }, '进入时间 '), (curNode && curNode.enteredAt) || '—'), React.createElement('span', { key: 'd' }, React.createElement('b', { style: { color: '#e6e8ef', fontWeight: 500 } }, '回滚 '), (curNode && curNode.rolledBackAt && curNode.rolledBackAt.length) || 0, ' 次')])])

          var artifactRow = null
          {
            var artRows = nodeArtifacts.length ? nodeArtifacts.map(function (a) { var cnt = allComments.filter(function (c) { return c.anchor && c.anchor.artifactId === a.id }).length; var labelEl = a.kind === 'link' ? React.createElement('a', { key: 'n', href: a.href, target: '_blank', rel: 'noreferrer', style: { color: '#7ab8ff', cursor: 'pointer', fontSize: 15 } }, a.label) : React.createElement('span', { key: 'n', style: { color: '#7ab8ff', cursor: 'pointer', fontSize: 15 }, onClick: function () { openViewer(a.id) } }, a.label); return React.createElement('div', { key: a.id, style: { display: 'flex', alignItems: 'center', gap: 8, padding: '10px 0', borderBottom: '1px solid #2b2f3d' } }, [labelEl, a.note ? React.createElement('span', { key: 'nt', style: { color: '#8a90a3', fontSize: 13, flex: 1 } }, a.note) : React.createElement('span', { key: 'sp', style: { flex: 1 } }), a.kind === 'link' ? null : React.createElement('span', { key: 'c', style: { color: '#7ab8ff', fontSize: 13, cursor: 'pointer' }, onClick: function () { openViewer(a.id) } }, '💬' + cnt)]) }) : [React.createElement('div', { key: 'empty', style: { color: '#4a4f60', fontSize: 13, padding: '8px 0' } }, '该节点无产物')]
            artifactRow = React.createElement('div', { key: 'art', style: panel }, [React.createElement('div', { key: 'h', style: { fontSize: 15, fontWeight: 600, marginBottom: 10 } }, '产物'), artRows])
          }

          var reviewerStrip = null
          var reviewers = (curNode && curNode.reviewers) || []
          if (reviewers.length) {
            var pills = reviewers.map(function (rv) { var rvComments = rv.commentIds.map(function (cid) { return allComments.filter(function (c) { return c.id === cid })[0] }).filter(Boolean); var openN = rvComments.filter(function (c) { return c.state === 'open' }).length; var maxRounds = 0; rvComments.forEach(function (c) { if ((c.roundCount || 0) > maxRounds) maxRounds = c.roundCount }); var ok = rv.state === '已通过'; var hasOpen = rvComments.some(function (c) { return c.state === 'open' }); var pillBg = ok ? 'rgba(88,192,122,0.15)' : (hasOpen ? 'rgba(217,128,128,0.15)' : 'rgba(138,144,163,0.15)'); var pillCol = ok ? '#58c07a' : (hasOpen ? '#d98080' : '#8a90a3'); var ps = { display: 'inline-flex', alignItems: 'center', gap: 5, fontSize: 13, padding: '3px 10px', borderRadius: 12, background: pillBg, border: '1px solid ' + pillCol, cursor: 'pointer' }; var dotc = { width: 8, height: 8, borderRadius: '50%', background: pillCol }; return React.createElement('span', { key: rv.role, style: ps, onClick: function () { setOpenRole(openRole === rv.role ? '' : rv.role) } }, [React.createElement('span', { key: 'd', style: dotc }), rv.role, openN > 0 ? React.createElement('span', { key: 'n', style: { color: '#8a90a3' } }, openN + '未解决') : null, maxRounds >= 10 && openN > 0 ? React.createElement('span', { key: 'r', style: { color: '#e8b34a', fontWeight: 600 } }, '轮' + maxRounds) : null]) })
            var disc = null
            if (openRole) { var rvSel = reviewers.filter(function (rv) { return rv.role === openRole })[0]; if (rvSel) { var rvComments = rvSel.commentIds.map(function (cid) { return allComments.filter(function (c) { return c.id === cid })[0] }).filter(Boolean); var cmtNodes = rvComments.map(renderComment); cmtNodes.push(React.createElement('div', { key: 'ovr', style: { marginTop: 12, paddingTop: 10, borderTop: '1px solid #2b2f3d' } }, React.createElement('button', { key: 'ob', style: btnSmallRed, onClick: function () { doOverrideRole(openRole) } }, '强行通过 ' + openRole + ' 全部剩余'))); disc = React.createElement('div', { key: 'disc', style: { marginTop: 12, padding: '10px 12px', background: '#212530', borderRadius: 6 } }, cmtNodes) } }
            reviewerStrip = React.createElement('div', { key: 'rv', style: panel }, [React.createElement('div', { key: 'h', style: { fontSize: 15, fontWeight: 600, marginBottom: 10 } }, '评审状态'), React.createElement('div', { key: 'pills', style: { display: 'flex', gap: 8, flexWrap: 'wrap' } }, pills), disc])
          }

          var gate = null
          if (curNode && curNode.gate) {
            var rejectedUI = null
            if (gateMode === 'reject') { rejectedUI = React.createElement('div', { key: 'rf', style: { marginTop: 12, border: '1px dashed rgba(217,128,128,0.5)', borderRadius: 6, padding: '12px 14px' } }, [React.createElement('textarea', { key: 'ta', placeholder: '打回理由（必填）', value: rejectVal.reason, onChange: function (e) { setRejectVal({ reason: e.target.value, newScope: rejectVal.newScope }) }, style: { width: '100%', background: '#212530', border: '1px solid #2b2f3d', borderRadius: 6, padding: '10px 12px', color: '#e6e8ef', fontSize: 14, minHeight: 64, resize: 'vertical' } }), React.createElement('label', { key: 'chk', style: { display: 'flex', gap: 8, alignItems: 'center', margin: '10px 0', fontSize: 14, color: '#8a90a3' } }, [React.createElement('input', { key: 'cb', type: 'checkbox', checked: rejectVal.newScope, onChange: function (e) { setRejectVal({ reason: rejectVal.reason, newScope: e.target.checked }) } }), '需重新界定新范围']), React.createElement('button', { key: 'go', style: btnRed, onClick: doReject }, '提交打回')]) }
            gate = React.createElement('div', { key: 'gate', style: panel }, [React.createElement('div', { key: 'cnt', style: { display: 'flex', alignItems: 'center', gap: 8, fontSize: 14, marginBottom: 12 } }, [React.createElement('span', { key: 'l', style: { color: '#8a90a3' } }, '评论解决进度'), React.createElement('span', { key: 'n', style: { color: gateBlocked ? '#e8b34a' : '#58c07a', fontWeight: 600 } }, (totalCnt - openCnt) + '/' + totalCnt), gateBlocked ? React.createElement('span', { key: 'w', style: { color: '#e8b34a', fontSize: 13 } }, '（未全部解决，「通过」被拦截）') : React.createElement('span', { key: 'w', style: { color: '#58c07a', fontSize: 13 } }, '（可通过）')]), gateErr ? React.createElement('div', { key: 'e', style: { color: '#d98080', fontSize: 13, marginBottom: 10 } }, gateErr) : null, React.createElement('div', { key: 'btns', style: { display: 'flex', gap: 8 } }, [React.createElement('button', { key: 'p', style: gateBlocked ? Object.assign({}, btn, { opacity: 0.4, cursor: 'not-allowed' }) : btnGreen, onClick: gateBlocked ? null : doPass }, gateBlocked ? '通过（被拦截）' : '通过'), React.createElement('button', { key: 'r', style: btnRed, onClick: function () { setGateMode(gateMode === 'reject' ? 'idle' : 'reject') } }, '打回（带理由）')])].concat(rejectedUI ? [rejectedUI] : []))
          }

          var viewer = null
          if (viewerId) {
            var artComments = allComments.filter(function (c) { return c.anchor && c.anchor.artifactId === viewerId })
            var authors = ['User']
            artComments.forEach(function (c) { if (c.author && authors.indexOf(c.author) < 0) authors.push(c.author) })
            var filtered = artComments.filter(function (c) { if (authorFilter === '__hide__') return false; if (authorFilter !== '__all__' && c.author !== authorFilter) return false; if (stateFilter === 'open' && c.state !== 'open') return false; if (stateFilter === 'resolved' && c.state === 'open') return false; return true })
            var lineComments = filtered.filter(function (c) { return c.anchor && c.anchor.kind === 'line' })
            var rangeComments = filtered.filter(function (c) { return c.anchor && c.anchor.kind === 'range' })
            var blockComments = filtered.filter(function (c) { return c.anchor && c.anchor.kind === 'block' })
            var docComments = filtered.filter(function (c) { return c.anchor && c.anchor.kind === 'document' })
            var rawLines = (viewerData && viewerData.content || '').split('\n')
            var authorPills = [React.createElement('span', { key: '__all__', onClick: function () { setAuthorFilter('__all__') }, style: { fontSize: 12, padding: '2px 10px', borderRadius: 10, cursor: 'pointer', background: authorFilter === '__all__' ? 'rgba(122,184,255,0.2)' : 'transparent', border: '1px solid ' + (authorFilter === '__all__' ? '#7ab8ff' : '#2b2f3d'), color: authorFilter === '__all__' ? '#7ab8ff' : '#8a90a3' } }, '全部')].concat(authors.map(function (a) { return React.createElement('span', { key: a, onClick: function () { setAuthorFilter(a) }, style: { fontSize: 12, padding: '2px 10px', borderRadius: 10, cursor: 'pointer', background: authorFilter === a ? 'rgba(122,184,255,0.2)' : 'transparent', border: '1px solid ' + (authorFilter === a ? '#7ab8ff' : '#2b2f3d'), color: authorFilter === a ? '#7ab8ff' : '#8a90a3' } }, a) })).concat([React.createElement('span', { key: '__hide__', onClick: function () { setAuthorFilter('__hide__') }, style: { fontSize: 12, padding: '2px 10px', borderRadius: 10, cursor: 'pointer', background: authorFilter === '__hide__' ? 'rgba(217,128,128,0.2)' : 'transparent', border: '1px solid ' + (authorFilter === '__hide__' ? '#d98080' : '#2b2f3d'), color: authorFilter === '__hide__' ? '#d98080' : '#8a90a3' } }, '隐藏')])
            var statePills = ['all', 'open', 'resolved'].map(function (s) { var label = s === 'all' ? '全部状态' : (s === 'open' ? '未解决' : '已解决'); return React.createElement('span', { key: s, onClick: function () { setStateFilter(s) }, style: { fontSize: 12, padding: '2px 10px', borderRadius: 10, cursor: 'pointer', background: stateFilter === s ? 'rgba(232,179,74,0.2)' : 'transparent', border: '1px solid ' + (stateFilter === s ? '#e8b34a' : '#2b2f3d'), color: stateFilter === s ? '#e8b34a' : '#8a90a3' } }, label) })
            var filterBar = React.createElement('div', { key: 'fb', style: { display: 'flex', gap: 8, alignItems: 'center', padding: '8px 18px', borderBottom: '1px solid #2b2f3d', flexWrap: 'wrap' } }, [React.createElement('span', { key: 'al', style: { fontSize: 12, color: '#4a4f60' } }, '作者'), authorPills, React.createElement('span', { key: 'sl', style: { fontSize: 12, color: '#4a4f60', marginLeft: 16 } }, '状态'), statePills])
            var elementNodes = []
            var idx = 0
            var selLo = null
            var selHi = null
            if (viewSel) { selLo = viewSel.lo; selHi = viewSel.hi }
            else if (rangeStart !== null && rangeEnd !== null) { selLo = Math.min(rangeStart, rangeEnd); selHi = Math.max(rangeStart, rangeEnd) }
            while (idx < rawLines.length) {
              var lm = rawLines[idx].match(/^```(\w*)/)
              if (lm) {
                const lang = lm[1] || ''
                const bstart = idx + 1
                const codeLines = [rawLines[idx]]
                idx++
                while (idx < rawLines.length && !/^```\s*$/.test(rawLines[idx])) { codeLines.push(rawLines[idx]); idx++ }
                if (idx < rawLines.length) { codeLines.push(rawLines[idx]); idx++ }
                const bend = idx
                const codeText = codeLines.join('\n')
                const blkComments = blockComments.filter(function (c) { return c.anchor.blockStart === bstart })
                const isPickedBlk = pendingBlock && pendingBlock.start === bstart
                elementNodes.push(React.createElement('div', { key: 'b' + bstart }, [React.createElement('div', { key: 'cb', onClick: function () { pickBlock({ start: bstart, end: bend, lang: lang }) }, style: { margin: '10px 0', padding: '12px 16px', background: isPickedBlk ? 'rgba(232,179,74,0.10)' : '#181b24', border: '1px solid ' + (isPickedBlk ? '#e8b34a' : '#2b2f3d'), borderLeft: '3px solid ' + (isPickedBlk ? '#e8b34a' : '#7ab8ff'), borderRadius: 6, cursor: 'pointer', fontFamily: 'Menlo, Consolas, monospace', fontSize: 13, color: '#c8cdd8' } }, [lang ? React.createElement('div', { key: 'lg', style: { fontSize: 12, color: '#e8b34a', marginBottom: 8 } }, lang) : null, React.createElement('span', { key: 'code', style: { whiteSpace: 'pre-wrap' } }, codeText)]), blkComments.map(renderComment)]))
              } else {
                const lineNo = idx + 1
                const ln = rawLines[idx]
                const lc = lineComments.filter(function (c) { return c.anchor.line === lineNo })
                const rc = rangeComments.filter(function (c) { return c.anchor.endLine === lineNo })
                const inSel = selLo !== null && lineNo >= selLo && lineNo <= selHi
                const bg = inSel ? 'rgba(232,179,74,0.14)' : (lc.length || rc.length ? 'rgba(122,184,255,0.07)' : 'transparent')
                const leftBar = inSel ? '3px solid #e8b34a' : (lc.length || rc.length) ? '3px solid #7ab8ff' : '3px solid transparent'
                elementNodes.push(React.createElement('div', { key: 'l' + lineNo }, [React.createElement('div', { key: 'ln', onClick: function () { pickLine(lineNo) }, style: { padding: '3px 12px', borderRadius: 4, background: bg, borderLeft: leftBar, cursor: 'pointer', lineHeight: '1.8' } }, React.createElement('div', { key: 'c', style: { display: 'flex', gap: 10 } }, [React.createElement('span', { key: 'n', style: { color: '#4a4f60', fontSize: 12, width: 30, textAlign: 'right', flex: '0 0 30px', userSelect: 'none', lineHeight: '1.8' } }, lineNo), React.createElement('div', { key: 't', style: { flex: 1, fontSize: 15 } }, renderLineContent(ln))])), lc.map(renderComment), rc.map(renderComment)]))
                idx++
              }
            }
            var docList = docComments.map(renderComment)
            var inputLabel = pendingBlock ? '对整个代码块评论' : (rangeStart !== null && rangeStart !== rangeEnd ? '对第 ' + Math.min(rangeStart, rangeEnd) + '~' + Math.max(rangeStart, rangeEnd) + ' 行评论' : (rangeStart !== null ? '对第 ' + rangeStart + ' 行评论' : '写文档评论…（点正文行可选）'))
            viewer = React.createElement('div', { key: 'vw', style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }, onClick: function (e) { if (e.target === e.currentTarget) closeViewer() } }, React.createElement('div', { key: 'vl', style: { width: '72vw', height: '85vh', background: '#1b1e27', border: '1px solid #2b2f3d', borderRadius: 10, display: 'flex', flexDirection: 'column', overflow: 'hidden' } }, [React.createElement('div', { key: 'h', style: { display: 'flex', alignItems: 'center', gap: 10, padding: '12px 18px', borderBottom: '1px solid #2b2f3d' } }, [React.createElement('span', { key: 't', style: { fontSize: 16, fontWeight: 600 } }, (viewerData && viewerData.art && viewerData.art.label) || '文档'), React.createElement('span', { key: 'x', style: { marginLeft: 'auto', cursor: 'pointer', color: '#8a90a3', fontSize: 18 }, onClick: closeViewer }, '✕')]), filterBar, React.createElement('div', { key: 'doc', style: { flex: 1, overflow: 'auto', padding: '18px 26px', color: '#d5d8e0', fontFamily: 'inherit', fontSize: 15 } }, [elementNodes, React.createElement('div', { key: 'sep', style: { margin: '18px 0', borderTop: '1px solid #2b2f3d' } }), docList]), React.createElement('div', { key: 'inputwrap', style: { borderTop: '1px solid #2b2f3d', padding: '14px 18px', display: 'flex', gap: 8, alignItems: 'flex-end' } }, [React.createElement('textarea', { key: 'cm', placeholder: inputLabel, value: newCmt, onChange: function (e) { setNewCmt(e.target.value) }, style: { flex: 1, background: '#212530', border: '1px solid #2b2f3d', borderRadius: 6, padding: '10px 12px', color: '#e6e8ef', fontSize: 15, minHeight: 50, resize: 'vertical' } }), React.createElement('button', { key: 'go', style: btnGreen, onClick: doComment }, pendingBlock ? '评论代码块' : '发表评论')])]))
          }

          var histPanel = null
          if (req.requirement && (req.requirement.history || []).length) {
            var hist = req.requirement.history
            var fieldName = { title: '标题', description: '描述', category: '类别', versionId: '版本', priority: '优先级', status: '状态', owner: '负责人' }
            function auditLine(by, c) {
              var label = fieldName[c.field] || c.field
              if (c.field === 'description') {
                var fEmpty = !String(c.from || '').trim()
                var tEmpty = !String(c.to || '').trim()
                if (fEmpty && !tEmpty) return by + ' 填写了需求说明'
                if (!fEmpty && tEmpty) return by + ' 清空了需求说明'
                return by + ' 修改了需求说明'
              }
              var fromV = c.field === 'versionId' ? verNameOf(c.from) : (String(c.from || '') === '' ? '（空）' : String(c.from))
              var toV = c.field === 'versionId' ? verNameOf(c.to) : (String(c.to || '') === '' ? '（空）' : String(c.to))
              return by + ' 将「' + label + '」从「' + fromV + '」变更为「' + toV + '」'
            }
            var histNodes = hist.slice().reverse().map(function (h, hi) {
              var by = h.by || 'User'
              var chgNodes = (h.changes || []).map(function (c, ci) {
                return React.createElement('div', { key: 'c' + ci, style: { fontSize: 13, lineHeight: 1.6 } }, auditLine(by, c))
              })
              return React.createElement('div', { key: 'h' + hi, style: { margin: '5px 0', padding: '8px 10px', background: '#212530', borderRadius: 5 } }, [
                React.createElement('div', { key: 'm', style: { fontSize: 12, color: '#8a90a3', marginBottom: 4 } }, String(h.at || '').replace('T', ' ').slice(0, 16)),
                chgNodes
              ])
            })
            histPanel = React.createElement('div', { key: 'hist', style: panel }, [React.createElement('div', { key: 'h', style: { fontSize: 15, fontWeight: 600, marginBottom: 10 } }, '变更历史'), histNodes])
          }

          var rightCol = [meta]
          if (artifactRow) rightCol.push(artifactRow)
          if (reviewerStrip) rightCol.push(reviewerStrip)
          if (gate) rightCol.push(gate)
          if (histPanel) rightCol.push(histPanel)

          return React.createElement('div', { style: { padding: 22, height: '100%', overflow: 'auto', color: '#e6e8ef' } }, [detailTop, infoPanel, descPanel, React.createElement('div', { key: 'flowwrap', style: { marginBottom: 14 } }, stepList), React.createElement('div', { key: 'right', style: { display: 'flex', flexDirection: 'column', gap: 14 } }, rightCol), viewer, formModal, delModal, verMgrModal, verDepModal, verPubModal, verRenameModal])
        }
      )
    })
  }
}
