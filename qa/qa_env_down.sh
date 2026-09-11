#!/usr/bin/env bash
# wbflow QA 环境拆除：按端口杀服务进程 + DROP scratch 库（绝不触 wbflow 库）。
set -uo pipefail
source /tmp/wbflow-qa-env.env 2>/dev/null || true
PIDS=$(lsof -ti tcp:6081 2>/dev/null || true)
if [ -n "${PIDS}" ]; then
  echo "${PIDS}" | xargs kill 2>/dev/null || true
  sleep 1
  echo "${PIDS}" | xargs kill -9 2>/dev/null || true
fi
pkill -f "wbflow-qa-config" 2>/dev/null || true
if [ -n "${QA_DBNAME:-}" ]; then
  mysql -h "${QA_MYSQL_HOST}" -P "${QA_MYSQL_PORT}" -u "${QA_MYSQL_USER}" -p"${QA_MYSQL_PASSWORD}" \
    -e "DROP DATABASE IF EXISTS \`${QA_DBNAME}\`;" 2>/dev/null || true
  echo "[qa] dropped ${QA_DBNAME}"
fi
echo "[qa] teardown done"
