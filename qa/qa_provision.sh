#!/usr/bin/env bash
# QA 测试库与测试用户一次性授权（2026-09-12 DM-09 配套）。
# 用 rd/configs/config.local.yaml 的 mysql.root 密码创建：
#   - 数据库 wbflow_qa（及授权 wbflow_qa%.*，覆盖 scratch 库 wbflow_qa_<ts> 命名）
#   - 用户 wbflow_qa@'%'（密码写入 config.local.yaml 的 qa 段，gitignore 不入库）
# 后续 qa_env_up.sh 默认走该测试用户，无需再注入 root 凭据。可重复执行（幂等）。
set -euo pipefail
QA_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$QA_DIR/.." && pwd)"
CFG="$REPO/rd/configs/config.local.yaml"
[ -f "$CFG" ] || { echo "FATAL: $CFG 不存在"; exit 1; }

read_cfg() { # $1=dotted.key  $2=default
  python3 - "$1" "$2" "$CFG" <<'PY' 2>/dev/null || echo "$2"
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

ROOT_HOST=$(read_cfg mysql.host 192.168.3.100)
ROOT_PORT=$(read_cfg mysql.port 3306)
ROOT_USER=$(read_cfg mysql.user root)
ROOT_PASS=$(read_cfg mysql.password "")
[ -n "$ROOT_PASS" ] || { echo "FATAL: config.local.yaml mysql.password 为空"; exit 1; }

# qa 段不存在则生成密码并落盘（gitignore 覆盖，不入库）
QA_PASS=$(read_cfg qa.password "")
if [ -z "$QA_PASS" ]; then
  QA_PASS=$(python3 -c "import secrets; print(secrets.token_hex(12))")
  printf '\n# QA 测试库/用户（qa_provision.sh 生成，gitignore 不入库）\nqa:\n  host: %s\n  port: %s\n  user: wbflow_qa\n  password: %s\n  dbname: wbflow_qa\n' "$ROOT_HOST" "$ROOT_PORT" "$QA_PASS" >> "$CFG"
  echo "[provision] 已生成 wbflow_qa 凭据并写入 config.local.yaml qa 段"
fi

export MYSQL_PWD="$ROOT_PASS"
echo "[provision] 授权 wbflow_qa@'%' + 库 wbflow_qa (host=${ROOT_HOST})"
mysql -h "$ROOT_HOST" -P "$ROOT_PORT" -u "$ROOT_USER" <<SQL
CREATE USER IF NOT EXISTS 'wbflow_qa'@'%' IDENTIFIED BY '$QA_PASS';
CREATE DATABASE IF NOT EXISTS wbflow_qa CHARACTER SET utf8mb4;
GRANT ALL PRIVILEGES ON \`wbflow_qa%\`.* TO 'wbflow_qa'@'%';
FLUSH PRIVILEGES;
SQL

# 冒烟：用新用户建/删探针库
unset MYSQL_PWD
export MYSQL_PWD="$QA_PASS"
mysql -h "$ROOT_HOST" -P "$ROOT_PORT" -u wbflow_qa -e "CREATE DATABASE IF NOT EXISTS wbflow_qa_probe; DROP DATABASE wbflow_qa_probe;" \
  && echo "[provision] 冒烟通过：wbflow_qa 用户可建删 wbflow_qa% 库"
unset MYSQL_PWD
echo "[provision] 完成。后续 qa_env_up.sh 自动走 wbflow_qa 凭据（config.local.yaml qa 段）"
