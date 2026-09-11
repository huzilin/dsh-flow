# fe · 前端（Web）

wbflow 前端。React 19 + Vite 8 + TypeScript（严格模式）+ react-router 7 + TanStack Query 5；契约类型由 `docs/api/v1/api.proto`（单一真相源）经 buf + ts-proto 生成，禁手写镜像类型。

本期范围（工单 T14 模块 I + T15 模块 II）：

- **T14 模块 I**：需求池（筛选/分页/新建）、看板（状态四泳道）、版本管理（创建/改名/推进/废弃）。
- **T15 模块 II**：需求详情页=流程树（SVG 纵向全节点渲染：nodeKey/相位/gate/user_gate/评审轮计数/current 高亮，被裁节点不渲染不占位，scope 徽标呈现 `flows.scope` 实值）+ **节点详情五区分栏**（① 状态卡 ② 评审状态药丸 ③ 产物区 ④ 意见区 ⑤ 门禁/拍板卡只读态）+ **产物阅读器**（行号渲染 + 懒加载分段 200 行/段 + link 外链卡）+ **锚点评论**（document/line/range/block 四级装配，T15 起四级全部可用：block 按围栏扫描代码块「⬚ 选代码块」整块选取）+ 评论操作矩阵（回复恒可、resolve=assignee、confirm/reopen=提评者，Web 身份=`user`）+ 删除需求入口（confirm=true 显式弹窗）+ 会话通道占位按钮（disabled，DSH 嵌入期开放）。
- **T16 模块 III/IV**：收件箱页（`/inbox`：七类通知 kind 图标/标签对齐 `constants.ts` 单一来源、未读优先、角色×类别×已读三筛选、分页、跳转定位 node→节点高亮 / task→任务区 / issue→意见区锚点）+ 顶栏入口未读 badge（MVP 轮询口径 30s，`VITE_INBOX_POLL_MS` 可配，TanStack Query `refetchInterval` 失效）+ TODO 投影区（tasks 在途任务同源投影，completed/rejected 不残留——后端同源，FE 只呈现）+ 全局冲突入口（escalation 未读聚合区）。
- **T16 拍板卡三态**（复用 T15 `DecisionCard` 的 `actions` 插槽）：进度只读 → agreed 迎宾呈现（各评审位结论 + resolved 摘要 + 产物链接 + 建议 + 通过/打回按钮组 + ↗ 流程树跳转）→ 已裁决（verdict 徽标 + 结论摘要；rejected 呈 T9 重入相位提示）；wb_decision 409 改判冲突提示 + 失效刷新。
- **T16 冲突卡**：escalate=1 工单聚合于节点详情意见区顶部；正反对照两栏（正方=review_results 最新反对位 / 反方=issue replies 普通条目）；裁决动作（wb_escalate 拉起幂等 / wb_adjudicate uphold|override 表单，user 角色可见）；裁决回填 replies 留痕呈现 + ↗ 节点/issue 跳转。

五区分栏口径对照（`docs/v1.0/arch_design.md` §5.5）：① 状态卡=双行（执行行：相位徽标+my_task 激活动作；审批行：表态计数）② 评审状态行=per-agent 药丸（轮内完整时间线属暂缓项 P2）③ 产物区=文件行+💬N+外链 ④ 意见区=评论列表+escalate 标记+在途任务投影（冲突卡聚合属 T16）⑤ 拍板卡=进度只读态（评审轮进行中/agreed 待拍板提示/已裁决 verdict）。数据源全部走聚合快照 + my_task（role=user），不自拼接口。

交互基线与增补明细：`docs/v1.0/arch_design.md` §5（交互增补层）· 原型：`.archive/docs-tmp-final/diagrams/ui-prototype-v3.html`（P1–P11：P8 看板/需求池、P9 版本管理、P10 收件箱、P11 TODO 为模块 I/III/IV 原型，2026-09-09 补齐）

## 常用命令

```bash
pnpm install        # 安装依赖（pnpm-lock.yaml 入库；node_modules 不入库）
pnpm dev            # 开发服务器（/api 代理到 rd，见下）
pnpm lint           # eslint
pnpm build          # tsc -b && vite build
pnpm test           # vitest
pnpm gen:proto      # 从 docs/api/v1/api.proto 重新生成 src/api/proto/api.ts（生成物入库）
```

## proto 生成链（契约零漂移）

- 真相源：`docs/api/v1/api.proto`（与后端 rd/api/api.pb.go 同一 proto）。
- 复现：`cd fe && pnpm gen:proto`，等价于 `PATH="$PWD/node_modules/.bin:$PATH" buf generate ../docs/api/v1 --template buf.gen.yaml`。
- 工具链：buf（`brew install bufbuild/buf/buf`）+ ts-proto v2（`protoc-gen-ts_proto` 从 fe/node_modules/.bin 解析）。
- 关键选项：`forceLong=string`——**int64 字段（createdAt/updatedAt/total/page 等）一律按 string 承接**（后端 protojson 出参口径，2^53 精度安全）；生成物 `src/api/proto/api.ts` 入库，改契约即重新生成。

## 本地联调指引（真实 rd 后端）

1. **起 rd**（仓库根）：

   ```bash
   export GOPROXY=https://goproxy.cn
   # 1) 先对目标库跑迁移（rd 启动不自动建表；rd/migration/migration_it_test.go env 门控）
   cd rd
   WBFLOW_MYSQL_IT=1 \
   WBFLOW_MYSQL_HOST=127.0.0.1 WBFLOW_MYSQL_PORT=3306 \
   WBFLOW_MYSQL_USER=root WBFLOW_MYSQL_PASSWORD='真实密码' \
   WBFLOW_MYSQL_DB=wbflow \
   go test ./migration/ -run TestIntegration_Migration -v
   # 2) 起服务
   export WBFLOW_AUTH_TOKEN='本地自定 token'   # 未设则 rd fail closed，API 全 401
   export WBFLOW_MYSQL_PASSWORD='真实密码'     # 或复制 configs/config.local.yaml 覆盖 host/user/dbname
   go run ./cmd/rd                             # 监听 127.0.0.1:8080（rd/configs/config.yaml）
   ```

   `curl 127.0.0.1:8080/healthz` 豁免 token 可探活（db 未就绪时 API 路由降级不注册）。

2. **起 fe**（本目录）：

   ```bash
   cp .env.example .env.local    # 填 VITE_WBFLOW_AUTH_TOKEN=<与 rd 同值>
   pnpm install && pnpm dev      # http://localhost:5173
   ```

   dev 代理：`/api → http://127.0.0.1:<port>`，端口经环境变量 `WBFLOW_API_PORT` 覆盖（默认 8080）。浏览器访问 `http://localhost:5173`（Vite 8 默认绑定 localhost）。

3. **seed 里有哪些可看数据**：rd 迁移种子（`rd/migration/sql/0002_seed.sql`）只覆盖**定义层**——roles 10 / capabilities 21 / flow_defs 4（business·tech scope_options=[full,rd,fe]，task=[rd,fe]，plan=[]）/ flow_def_nodes 40；**versions 与 requirements 无种子行**，需求池/看板/版本管理首次为空，直接在 UI 里「＋新建需求 / ＋新建版本」造数。如需「待分配池标记行」，新建版本时勾选 is_backlog（不勾时待分配池需求以 version_id 为空承载，列表照常可用）。

## 口径备注（后端为行为真相源）

- **scope 徽标**：需求 scope 实值由流程实例化按 scope_options 决定（T4 消费）；T14 需求卡/行的 scope 徽标按 flow_defs 的 scope_options **示意呈现**（business/tech→full、task→rd|fe、plan→无徽标），见 `src/api/constants.ts`。
- **乐观锁 409（OPTIMISTIC_LOCK）**：版本改名/推进/废弃提交时必回传 GET 所得 `updated_at` 基线（后端宽松模式也恒回传）；409 时提示冲突并 invalidate 刷新基线，请重试。
- **advance 软约束**：发布成功但版本内有 unfinished（status 非 已完成/已取消）需求时，响应 `unfinished` 清单以提示条呈现（仅提示不拦截）；重复发布幂等。
- **deprecate 有引用 blocked**：存在未完成需求引用时后端返回 `{ok:false, blocked:true, code:GATE_BLOCKED}`，错误文本原样呈现在确认弹窗内。
- **删除需求**：DELETE 需显式 `confirm=true`（User 明确指令口径）——详情页右上「删除需求」按钮（确认弹窗后携带 confirm 提交）。

## 模块 II 联调指引（T15：流程树 / 五区分栏 / 阅读器 / 锚点评论）

### 造数路径（curl，亦见 rd 各 IT 的 httptest 造数方式）

rd 起服后（`WBFLOW_AUTH_TOKEN` 与 fe `.env.local` 同值；`docs` 为仓库根目录，document 产物 href 相对它解析）：

```bash
T=<token>; H="-H \"Authorization: Bearer $T\" -H Content-Type:application/json"

# 1) 创建需求（创建即实例化流程；scope 空=取 scope_options 首值，business 默认 full）
curl -s $H -X POST localhost:8080/api/requirements -d '{
  "title":"模块II联调","category":"需求任务","createdBy":"user",
  "description":"流程树/五区分栏/阅读器冒烟"}'
# → requirement.requirementId，记为 REQ

# 2) 聚合快照：读 flows.scope / flow_nodes（节点集与序号即流程树渲染集）/ current_node_id
curl -s $H localhost:8080/api/requirements/$REQ | jq '{scope:.flows[0].scope, current:.flows[0].currentNodeId, nodes:[.flowNodes[]|{nodeId,nodeOrder,nodeKey,state}]}'

# 3) 给节点登记产物（nodeId 取快照，如 n-05）；document 需仓库根下真实文件
curl -s $H -X POST localhost:8080/api/requirements/$REQ/nodes/n-05/artifacts -d '{
  "productType":"rd_design","label":"《RD 技术设计》","kind":"document",
  "href":"docs/v1.0/arch_design.md","addedBy":"rd"}'
curl -s $H -X POST localhost:8080/api/requirements/$REQ/nodes/n-05/artifacts -d '{
  "productType":"code_mr","label":"《RD 代码 MR》","kind":"link",
  "href":"https://git.example.com/mr/1","addedBy":"rd"}'
# 亦可 rd 侧 go test ./service/ -run TestIntegration 的造数函数（如有 IT 门控）

# 4) 评论五端点（document / line / range / block 四级锚点）
curl -s $H -X POST localhost:8080/api/requirements/$REQ/comments -d '{
  "nodeId":"n-05","author":"user","roleId":"user",
  "anchor":{"kind":"line","artifactId":"<artId>","line":"42"},"text":"第 42 行口径?"}'
curl -s $H -X POST localhost:8080/api/comments/<commentId>/replies -d '{
  "by":"rd","roleId":"rd","text":"已修订"}'
# resolve（须=assignee）/ confirm、reopen（须=提评者）同理，非法迁移 409 INVALID_TRANSITION

# 5) my_task（详情页以 role=user 消费节点摘要/激活动作/在途任务）
curl -s $H -X POST localhost:8080/api/requirements/$REQ/my-task -d '{"role":"user","nodeId":"n-05"}'
```

UI 侧冒烟路径：`需求池 → 点击需求进详情`：核对流程树节点集与快照一致（scope 徽标=快照 `flows.scope` 实值）→ 点节点出五区分栏 → 产物区点「阅读」开阅读器（行号/懒加载分段；外链产物出外链卡）→ 点行/Shift 点区间/「⬚ 选代码块」/不选行发表评论 → 评论按锚点就位渲染 → 意见区核对状态徽标与操作按钮可见性。**进度态推进**（相位/评审轮/拍板卡只读态变化）需任务总线与评审表态造数（`POST /api/tasks` + `PATCH /api/tasks/:taskId` + `POST .../nodes/:nodeId/review`），属 T17 演练脚本范围，UI 冒烟可暂以 step 4 的评论造数验阅读器闭环。

### 口径备注（模块 II）

- **Web 操作身份**：详情页/评论/删除均以 `user` 角色操作（`src/api/constants.ts` `WEB_USER_ROLE`）；resolve/confirm/reopen 按钮按 T7 操作者约束收敛可见性（后端裁决仍是真相源，409 兜底）。
- **流程树形态**：SVG 纵向单主链——实例节点无并行组/泳道字段（`flow_nodes` 无该列），原型横向泳道不适用；渲染矩阵一致性由后端 T4 实例化断言兜底。
- **阅读器渲染**：原文按行号直映（不做 Markdown 转换渲染——行号↔内容对应是锚点防漂移前提）；大文档按 200 行/段懒加载（滚动自动续载 + 兜底按钮）。
- **block 锚点**：按围栏（```/~~~）扫描代码块，块首行「⬚ 选代码块」整块选取；无围栏文档无 block 入口（后端要求 lang 非空）。
- **拍板卡只读态**：agreed 判定为「评审位齐且无反对无待表态」（`deriveGateView`）；三态切换按钮组属 T16（`DecisionCard` 已留 `actions` 插槽）。
- **看板分页上限**：后端强制分页（page_size 上限 100），看板取单页 100；超出时提示条说明。

## 模块 III/IV 联调指引（T16：收件箱 / 拍板卡三态 / 冲突卡）

### 造数路径（curl，接续模块 II 的 REQ/节点/产物/评论造数）

```bash
T=<token>; H="-H "Authorization: Bearer $T" -H Content-Type:application/json"

# 1) 评审表态造数（对某 review 相位 + user_gate=1 节点，如 business 的 n-04 产品设计评审）：
#    逐角色 agree → collect=approved 触发 gate_pending 通知（收件箱「待你拍板」）+ 拍板卡 agreed 迎宾态
curl -s $H -X POST localhost:8080/api/requirements/$REQ/nodes/n-04/review -d '{
  "role":"arch","capabilityId":"arch.prd-review","state":"agree"}'
#    同理补齐该节点 reviewers 声明的其余评审位；unagree 则触发 clarify_task 通知（owner 澄清循环）

# 2) 收件箱读端点（七类通知 + TODO 投影双通道；未读优先，int64 计数为 string）
curl -s $H "localhost:8080/api/inbox?role=user&read_filter=unread" | jq '{total, kinds:[.notifications[].kind], todos}'

# 3) 拍板（agreed 后）：approved 前移下一节点；rejected 必填 reason（T9 重入语义，
#    拍板卡呈现「已打回，等待 exec 重入」——重入状态从快照相位读取）
curl -s $H -X POST localhost:8080/api/requirements/$REQ/nodes/n-04/decision -d '{
  "verdict":"approved","reason":"评审齐备","branch":"feat/x"}'
#    重复拍板同 verdict 幂等 noop=true；改判 409 IDEMPOTENT_CONFLICT（UI 提示+刷新）
#    评审未全 agree 时拍板 → 409 GATE_BLOCKED（进度态按钮组不渲染，后端兜底）

# 4) 冲突上升裁决链：评论 → 反复 reopen（或显式上升）→ escalate 置位 + escalation 通知
curl -s $H -X POST localhost:8080/api/comments/$CM/escalate -d '{
  "operator":"user","reason":"reopen 拉锯不收敛"}'
#    幂等：已有进行中裁决 session 时 noop=true 回显 session_id
curl -s $H "localhost:8080/api/inbox?kind=escalation&read_filter=unread" | jq '.notifications'

# 5) User 仲裁回填：uphold=维持议题（回澄清/修复链）/ override=推翻（issue 以 resolve_path=override 置 resolved——
#    无 overridden 态，09-12 裁决；escalate 复位 + reviewer 重表态新轮 reentered=true）；结论经 replies kind=adjudication 留痕
curl -s $H -X POST localhost:8080/api/comments/$CM/adjudicate -d '{
  "verdict":"uphold","conclusion":"按含降级路径口径执行","decidedBy":"user"}'
#    escalate=0 时回填 → 409 GATE_BLOCKED
```

UI 冒烟路径：`顶栏收件箱（badge 未读数）→ /inbox`：七类 kind 标签/图标、未读优先、筛选与分页 →
点「待你拍板」通知跳详情并选中节点（拍板卡呈 agreed 迎宾态：评审位结论/工单摘要/产物/建议）→
「通过/打回」提交（409 改判时提示并刷新为已裁决态）→ 「冲突上升裁决」通知跳意见区顶部冲突卡
（正反对照两栏）→ 仲裁表单提交 → 回填条目与轮重入提示呈现。TODO 投影区核对在途任务
（completed/rejected 不残留由后端同源保证）。

### 口径备注（模块 III/IV）

- **通知载体**：issues 行 issue_type=notification（anchor_kind=类别复用列）；已读/未读=state open/resolved；
  排序与分页全在后端（未读优先、同组新→旧），FE 不重排。
- **badge 轮询**：独立轻量查询 `read_filter=unread&page_size=1` 只取 total；与收件箱页同频率独立轮询，
  mutate/裁决后经 `invalidateQueries({queryKey:["inbox"]})` 族失效即时刷新。
- **拍板双通道**：收件箱 gate_pending=系统投递（plugin）、拍板卡迎宾态=快照派生呈现，二者同源不双写。
- **冲突卡数据装配**：正方=快照 flow_nodes.review_results 该位最新表态=unagree（后轮 agree 覆盖）；
  反方=replies 非 voided 普通条目；裁决回填=replies kind=adjudication（含 verdict/sessionId）。
