#!/usr/bin/env bash
# wbflow QA 环境拉起：scratch 库（迁移+seed）+ 真实服务进程（:6081）。
# 用法: qa_env_up.sh   （默认走 config.local.yaml qa 段的 wbflow_qa 测试用户；环境变量可覆盖；零落地）
set -euo pipefail
QA_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$QA_DIR/.." && pwd)"
TS=$(date +%s)
DB="wbflow_qa_${TS}"
PORT=6081
TOKEN="wbflow-qa-token"
LOCAL_CFG="$REPO/rd/configs/config.local.yaml"

read_cfg() { # $1=dotted.key  $2=default
  python3 - "$1" "$2" "$LOCAL_CFG" <<'PY' 2>/dev/null || echo "$2"
import sys
path, default, file = sys.argv[1], sys.argv[2], sys.argv[3]
try:
    import yaml
    d = yaml.safe_load(open(file))
except Exception:
    sys.exit(1)
for k in path.split('.'):
    d = d.get(k) if isinstance(d, dict) else None
    if d is None:
        break
print(d if d is not None else default)
PY
}

MYSQL_HOST_BASE=$(read_cfg mysql.host "192.168.3.100")
: "${QA_MYSQL_HOST:=$(read_cfg qa.host "$MYSQL_HOST_BASE")}"
: "${QA_MYSQL_PORT:=$(read_cfg qa.port "3306")}"
: "${QA_MYSQL_USER:=$(read_cfg qa.user "wbflow_qa")}"
: "${QA_MYSQL_PASSWORD:=$(read_cfg qa.password "")}"
[ -n "$QA_MYSQL_PASSWORD" ] || { echo "FATAL: QA 凭据缺失——先执行 bash qa/qa_provision.sh 生成测试库与用户"; exit 1; }

# docs root 指向 tmp 隔离目录（scratch 同口径）：LLM 产文/运行期产物落 tmp，
# 不污染仓库工作区；fixtures 从仓库资产拷入（href 相对路径同构解析）
QA_DOCS_ROOT="/tmp/wbflow-qa-docs-${TS}"
mkdir -p "${QA_DOCS_ROOT}/v1.0/qa"
cp -R "${REPO}/docs/v1.0/qa/fixtures" "${QA_DOCS_ROOT}/v1.0/qa/" 2>/dev/null || true
CFG="/tmp/wbflow-qa-config-${TS}.yaml"
cat > "$CFG" <<EOF
http:
  port: ${PORT}
mysql:
  host: \${QA_MYSQL_HOST}
  port: \${QA_MYSQL_PORT}
  user: \${QA_MYSQL_USER}
  password: \${QA_MYSQL_PASSWORD}
  dbname: ${DB}
auth:
  token: ${TOKEN}
log:
  level: info
  mode: dev
  path: /tmp/wbflow-qa-server-${TS}.log
artifacts:
  docs_root: ${QA_DOCS_ROOT}
escalate:
  loop_threshold: 5
scanner:
  enabled: true
  interval_sec: 15
  followup_after_sec: 300
  intervene_after_sec: 600
  retry_backoff_sec: 5
llm:
  enabled: true
  base_url: ""
  actions:
    - to-brd
    - arch.prd-review.review
    - arch.rd-design-review.review
EOF

cd "$REPO/rd"
echo "[qa] migrate → ${DB}"
QA_MYSQL_HOST="$QA_MYSQL_HOST" QA_MYSQL_PORT="$QA_MYSQL_PORT" \
QA_MYSQL_USER="$QA_MYSQL_USER" QA_MYSQL_PASSWORD="$QA_MYSQL_PASSWORD" \
  go run ./cmd/rd migrate -config "$CFG" -config-local /dev/null

echo "[qa] start server :${PORT}"
QA_MYSQL_HOST="$QA_MYSQL_HOST" QA_MYSQL_PORT="$QA_MYSQL_PORT" \
QA_MYSQL_USER="$QA_MYSQL_USER" QA_MYSQL_PASSWORD="$QA_MYSQL_PASSWORD" \
  nohup go run ./cmd/rd -config "$CFG" -config-local /dev/null > "/tmp/wbflow-qa-server-${TS}.out" 2>&1 &
SRV_PGID=$!

for i in $(seq 1 60); do
  if curl -sf "http://127.0.0.1:${PORT}/healthz" | grep -q '"dbStatus":"ok"\|"ok":true'; then
    echo "[qa] healthz ok"
    break
  fi
  sleep 1
done

cat > /tmp/wbflow-qa-env.env <<EOF
export QA_BASE=http://127.0.0.1:${PORT}
export QA_TOKEN=${TOKEN}
export QA_MYSQL_HOST=${QA_MYSQL_HOST}
export QA_MYSQL_PORT=${QA_MYSQL_PORT}
export QA_MYSQL_USER=${QA_MYSQL_USER}
export QA_MYSQL_PASSWORD='${QA_MYSQL_PASSWORD}'
export QA_DBNAME=${DB}
export QA_SERVER_PGID=${SRV_PGID}
export QA_CONFIG=${CFG}
export QA_TS=${TS}
export QA_DOCS_ROOT
EOF
echo "[qa] env → /tmp/wbflow-qa-env.env （source 后跑 qa_drive.py；teardown 用 qa_env_down.sh）"
