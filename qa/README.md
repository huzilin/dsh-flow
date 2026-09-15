# qa/ · wbflow QA 测试资产（固化复用）

> wbflow 验收线的**可执行测试资产**正本（2026-09-10 固化）。配套文档在 `docs/v1.0/qa/`：
> 用例设计 [qa-test-cases.md](docs/v1.0/qa/qa-test-cases.md) · 验收报告 [qa-execution-report.md](docs/v1.0/qa/qa-execution-report.md)
> · 缺陷台账 [qa-defects.md](docs/v1.0/qa/qa-defects.md) · 诊断报告 [qa-diagnosis.md](docs/v1.0/qa/qa-diagnosis.md)。

## 资产清单

| 文件 | 用途 |
|---|---|
| `qa_env_up.sh` | 一键拉起验收环境：scratch 库（`wbflow_qa_<ts>`，迁移 v4+seed）+ 真实服务进程 :6081；凭据经环境变量注入零落地 |
| `qa_env_down.sh` | 拆环境：按端口杀进程 + DROP scratch 库（绝不触 wbflow 联调库） |
| `qa_drive.py` | 全量验收驱动：28 wb_* 端点 × 4 流程 40 节点 × 全分支 + 产物矩阵/审计对账（基线 **196 PASS / 0 FAIL / 1 SKIP**，含 A-15g 产物入库守卫断言） |
| `qa_diag_min.py` | 诊断最小复现环：逐缺陷独立节实测（~8s，软失败继续） |
| `qa_live_audit.py` | D-3 用户验收环境数据合规巡检：对联调库跑期望矩阵（期望 document 产物 link 冒充/文件缺失 → FAIL，退出码非 0；BUG-015 衍生） |
| `qa_doc_audit.py` | D-4 文档-实现一致性巡检：裁决废止的旧口径禁用短语扫活文档（BUG-002 衍生；规则随新拍板增补） |
| `qa-results.json` | 末轮逐用例结果（每次运行覆盖） |

产物 fixture 由 `qa_drive.py` 自动生成到 `docs/v1.0/qa/fixtures/`（wb_get_artifact_content 的 docs root 解析要求 fixture 必须在 docs/ 下，故不随本目录）。

## 用法（复跑/回归）

```bash
# 1. 拉环境（密码注入）
QA_MYSQL_PASSWORD='<密码>' bash qa_env_up.sh
# 2. 跑全量验收（约 40s）
source /tmp/wbflow-qa-env.env && python3 qa_drive.py
# 3. 拆环境
bash qa_env_down.sh
```

- 环境变量：`QA_BASE` / `QA_TOKEN` / `QA_DBNAME` 等，全部由 `qa_env_up.sh` 写入 `/tmp/wbflow-qa-env.env`。
- 退出码：FAIL>0 时非 0，可直接接 CI。
- 回归判据：**196 PASS / 0 FAIL / 1 SKIP**（SKIP=BUG-007 死代码分支，附因跳过）；另跑 `QA_MYSQL_PASSWORD='<密码>' python3 qa_live_audit.py` 巡检联调库（D-3）、`python3 qa_doc_audit.py`（D-4），违规即回归。FAIL>0 即回归。
- **rd 单测/IT 分层（注记-L 单测空跑修复）**：`go test ./...` 仅纯函数单测（IT 以 `WBFLOW_MYSQL_IT=1` env 门控自跳过——这就是「单测全绿但状态推进没覆盖」假象的根源）；完整回归须再跑真库 IT：
  ```bash
  cd rd && WBFLOW_MYSQL_IT=1 WBFLOW_MYSQL_HOST=192.168.3.100 WBFLOW_MYSQL_USER=root \
    WBFLOW_MYSQL_PASSWORD='<密码>' go test ./service/ -timeout=25m
  ```

## 契约变更 SOP（C3 跨语言契约快照）

FE 手写镜像面（flowview.ts 判定/constants.ts 词表/payload 形状）以 Go 真相源导出的
golden 快照做漂移检测（双测试互锁）：改后端判定/词表/payload struct 后**必须**：

```bash
cd rd && go test ./contract/ -run TestUnit_ContractSnapshot -update   # 重生成 golden（并提交）
cd fe && npx vitest run src/api/contract                              # FE 消费面验证
```

- 不重生 → Go 侧 `TestUnit_ContractSnapshot` 比对红灯（忘跑 FE 测试也会被拦）；
- 后端改判定 FE 未跟 → FE 侧 contract.test.ts 真值表红灯；
- golden 中 `feSkip` 样例=已知双端分歧的显式记录（FE 无 scope 校验，相邻成对约束
  替代，flowview.ts:359 注），FE 测试跳过并断言其存在；
- 断言口径锁「拦截与否+结果值」，不锁文案措辞（呈现层自由）。

## 扩展约定

- 新增用例：用例编号进 `docs/v1.0/qa/qa-test-cases.md` 覆盖矩阵，断言用 `qa_drive.py` 的 `ck(cid, cond, detail)`（软失败，一轮收全缺陷）。
- 新增剧本：`Req` 对象（create/execute/review/decide/advance 全步骤封装）+ 走完的需求 append 进 `WALKED`（自动进 D-1 产物矩阵对账）。
