#!/usr/bin/env bash
# wbflow QA 环境拉起：scratch 库（迁移+seed）+ 真实服务进程（:18081）。
# 用法: qa_env_up.sh   （凭据经环境变量注入，见 config.local.yaml 口径；零落地）
set -euo pipefail
QA_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$QA_DIR/.." && pwd)"
TS=$(date +%s)
DB="wbflow_qa_${TS}"
PORT=18081
TOKEN="wbflow-qa-token"

: "${QA_MYSQL_HOST:=192.168.3.100}"
: "${QA_MYSQL_PORT:=3306}"
: "${QA_MYSQL_USER:=root}"
: "${QA_MYSQL_PASSWORD:?须注入 QA_MYSQL_PASSWORD}"

QA_DOCS_ROOT="${REPO}/docs"
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
