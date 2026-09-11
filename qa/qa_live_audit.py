#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""D-3 用户验收环境数据合规巡检（BUG-015 衍生，to-qa-testcases 骨架 D-3 固化）。

对联调库（用户实际浏览的环境）跑同一套期望矩阵：期望 document 的产物必须
kind=document 且文件在 docs root 下真实存在（外链冒充/文件缺失均为 FAIL）；
期望 link 的产物 kind 必须为 link。期望外产物不校验。SQL 经 mysql CLI（与
qa_drive.py 同模式，零新增依赖）。

用法：
  QA_MYSQL_PASSWORD='<密码>' python3 qa_live_audit.py            # 默认库 wbflow
  QA_DBNAME=wbflow_qa_xxx ... python3 qa_live_audit.py           # 或巡检 scratch 库
退出码：存在违规行非 0（可接 CI / 回归口径）。
"""
import json
import os
import subprocess
import sys

MY = dict(
    host=os.environ.get("QA_MYSQL_HOST", "192.168.3.100"),
    port=os.environ.get("QA_MYSQL_PORT", "3306"),
    user=os.environ.get("QA_MYSQL_USER", "root"),
    pwd=os.environ.get("QA_MYSQL_PASSWORD", ""),
    db=os.environ.get("QA_DBNAME", "wbflow"),
)
DOCROOT = os.path.realpath(os.environ.get(
    "QA_DOCS_ROOT", os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "docs")))
VIOLATIONS = []


def sql_rows(query):
    """mysql CLI -N -B 取数：TSV 行列表；失败抛异常（巡检不允许静默空结果）。"""
    env = dict(os.environ, MYSQL_PWD=MY["pwd"])
    p = subprocess.run(
        ["mysql", "-h", MY["host"], "-P", MY["port"], "-u", MY["user"], "-N", "-B", MY["db"], "-e", query],
        capture_output=True, text=True, env=env, timeout=30)
    if p.returncode != 0:
        raise RuntimeError(f"mysql 查询失败: {p.stderr.strip()[:200]}")
    out = p.stdout.strip()
    return [line.split("\t") for line in out.splitlines()] if out else []


QUERY = """
SELECT a.artifact_id, a.requirement_id, a.node_id, a.product_type, a.kind,
       IFNULL(a.href,''), d.expected_outputs
FROM artifacts a
JOIN flow_nodes fn ON a.node_id = fn.node_id
JOIN requirements r ON a.requirement_id = r.requirement_id
JOIN flow_def_nodes d ON d.flow_defs_id = r.flow_defs_id AND d.template_node_id = fn.def_node_id
ORDER BY a.requirement_id, a.created_at"""


def main():
    rows = sql_rows(QUERY)
    checked = 0
    for cells in rows:
        aid, req, node, ptype, kind, href, raw_exp = (cells + [""] * 7)[:7]
        try:
            expected = {e.get("productType"): (e.get("kind") or "document")
                        for e in json.loads(raw_exp or "[]")}
        except (ValueError, TypeError):
            print(f"[WARN] {req}/{node} expected_outputs 非法 JSON，该行跳过")
            continue
        exp_kind = expected.get(ptype)
        if exp_kind is None:
            continue  # 期望外产物（如 code_mr 补充外链）不校验
        checked += 1
        if kind != exp_kind:
            VIOLATIONS.append(f"{aid} {req}/{node} {ptype}: kind={kind} 期望 {exp_kind}（外链冒充/类型错位）")
            continue
        if kind == "document":
            path = os.path.realpath(os.path.join(DOCROOT, href))
            if not path.startswith(DOCROOT + os.sep) or not os.path.isfile(path):
                VIOLATIONS.append(f"{aid} {req}/{node} {ptype}: 文档文件缺失或越界 docs root: {href}")

    print(f"D-3 巡检（库 {MY['db']}，docs root {DOCROOT}）：期望内产物 {checked} 条，违规 {len(VIOLATIONS)} 条")
    for v in VIOLATIONS:
        print(f"  [FAIL] {v}")
    sys.exit(1 if VIOLATIONS else 0)


if __name__ == "__main__":
    main()
