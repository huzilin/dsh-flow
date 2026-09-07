# fe · 前端（Web）

wbflow 前端。React 19 + Vite 8 + TypeScript（严格模式）+ react-router 7 + TanStack Query 5；契约类型由 `docs/api/v1/api.proto`（单一真相源）经 buf + ts-proto 生成，禁手写镜像类型。

本期范围（工单 T14，模块 I）：需求池（筛选/分页/新建）、看板（状态四泳道）、版本管理（创建/改名/推进/废弃）、需求详情占位页（流程树/节点详情五区分栏由 T15 接管）。模块 III/IV 收件箱/TODO 属 T16，未实现。

交互基线与增补明细：`docs-tmp/ui-interaction-final.md` · 原型：`docs-tmp/diagrams/ui-prototype-v3.html`

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
- **删除需求**：DELETE 需显式 `confirm=true`（User 明确指令口径）——client 已封装，UI 入口留待详情页（T15）。
- **看板分页上限**：后端强制分页（page_size 上限 100），看板取单页 100；超出时提示条说明。
