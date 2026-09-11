#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""wbflow 迁移差集审计（一期②，2026-09-11 拍板）：模型定义动作集 vs QA 驱动已覆盖集。

模型集 = rd/migration/sql/0002_seed.sql 的 flow_def_nodes.node_actions 全量展开：
    kind=capability 条目展开为 <capabilityId>.<action>；kind=node 条目取 actionId 原样
    （注记-J3 口径：kind=node 的 actionId 点号是名字的一部分，不作 capability 引用解）。
已覆盖集 = qa/qa_drive.py + rd/e2e/*.go 源面按模型 id 逐条边界匹配（v1 口径：静态扫描而
    非运行时记录；节点级覆盖由 B 组全走查保证，本脚本管「seed 里的动作一个都不许漏」）。

出口语义（两级）：
  FAIL（exit 1，缺陷类，可作门禁）——
    a. 动作 ID 不可解析：capability 条目的 capabilityId ∉ seed capabilities 或
       action ∉ 该 capability actions 列（actions 列为动作对象数组，取 actionId 字段）；
    b. skill 引用缺失：executor=subagent 的条目 skill 字段 ∉ docs/templates/skills/
       （标准权威化 P1 后 skill 目录即目录；待建 skill 走 EXEMPT 显式豁免）；
    c. 失效豁免：EXEMPT 里的动作实际已被驱动（应移除豁免）。
  补测清单（exit 0 默认 / --strict 时 exit 1）——
    差集 = 模型集 − 驱动集 − 豁免，逐条带 flow/node 出处 = 喂 C 组的补测清单。

用法：python3 qa/qa_branch_audit.py [--strict]   （零外部依赖，离线可跑）
"""
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, ".."))
SEED_SQL = os.path.join(REPO, "rd", "migration", "sql", "0002_seed.sql")
DRIVER_SOURCES = [
    os.path.join(REPO, "qa", "qa_drive.py"),
    *[os.path.join(REPO, "rd", "e2e", f) for f in sorted(os.listdir(os.path.join(REPO, "rd", "e2e"))) if f.endswith(".go")],
]
SKILL_DIR = os.path.join(REPO, "docs", "templates", "skills")

# 显式豁免。纪律：加豁免必须带理由，评审时逐条过。
# EXEMPT_ACTIONS：动作 id 豁免（不参与差集）；SKILL_EXEMPT：subagent 条目 skill 引用豁免。
EXEMPT_ACTIONS = {}
SKILL_EXEMPT = {
    "to-alignment": "待建 skill——wayfinder 工单 03 裁决 to-alignment=公共 skill，在 roadmap P4 待建清单内，建届时移除本豁免",
}

QUOTED = re.compile(r"'((?:[^']|'')*)'")
TUPLE = re.compile(r"\(((?:[^()']|'(?:[^']|'')*')*)\)")


def sql_rows(section, insert_into):
    """取 INSERT IGNORE INTO <table> 段内每条 VALUES 元组的引号字段列表（元组可跨行）。"""
    seg = section.split(f"INSERT IGNORE INTO {insert_into}", 1)[-1].split(";", 1)[0]
    rows = []
    for t in TUPLE.findall(seg):
        if "'" not in t:  # 列名清单括号无引号，跳过
            continue
        rows.append([m.group(1).replace("''", "'") for m in QUOTED.finditer(t)])
    return rows


def load_capabilities(sql):
    """capability_id → actions 列里的 actionId 集合（actions 列=动作对象数组 JSON）。"""
    caps = {}
    for q in sql_rows(sql, "capabilities"):
        try:
            entries = json.loads(q[4]) if len(q) > 4 and q[4] else []
        except json.JSONDecodeError:
            print(f"FATAL: capability {q[0]} actions 列非 JSON")
            sys.exit(2)
        caps[q[0]] = {e.get("actionId") for e in entries if isinstance(e, dict)}
    return caps


def load_model(sql):
    """{action_id: (flow, node_key, entry)}；capability 条目展开，node 条目原样。"""
    model = {}
    for q in sql_rows(sql, "flow_def_nodes"):
        # 引号字段序：flow, template_node_id, node_key, node_type, scope_expr, reviewers, node_actions, expected_outputs
        flow, node_key = q[0], q[2]
        try:
            entries = json.loads(q[6])
        except json.JSONDecodeError:
            print(f"FATAL: {flow}/{node_key} node_actions 非 JSON: {q[6][:120]}")
            sys.exit(2)
        for e in entries or []:
            if e.get("kind") == "capability":
                for a in e.get("actions", []):
                    model[f"{e.get('capabilityId')}.{a}"] = (flow, node_key, e)
            else:
                aid = e.get("actionId", "")
                if aid:
                    model[aid] = (flow, node_key, e)
    return model


def main():
    strict = "--strict" in sys.argv
    sql = open(SEED_SQL, encoding="utf-8").read()
    caps = load_capabilities(sql)
    model = load_model(sql)
    skills = set(os.listdir(SKILL_DIR)) if os.path.isdir(SKILL_DIR) else set()
    corpus = ""
    for p in DRIVER_SOURCES:
        if os.path.exists(p):
            corpus += open(p, encoding="utf-8").read()

    covered = set()

    def in_corpus(aid):
        hit = re.search(r"(?<![\w.-])" + re.escape(aid) + r"(?![\w-])", corpus)
        if hit:
            covered.add(aid)
        return hit

    print(f"模型动作集 {len(model)} 条（seed 定义节点）；驱动语料 {len(DRIVER_SOURCES)} 文件；capabilities {len(caps)} 个；skill 目录 {len(skills)} 项\n")
    # a+b) 可解析性 + skill 引用
    bad = []
    for aid, (flow, node, e) in sorted(model.items()):
        if e.get("kind") == "capability":
            cap_id, action = aid.split(".", 1)
            if cap_id not in caps:
                bad.append(f"{aid}  ({flow}/{node}) capabilityId '{cap_id}' 不在 seed capabilities")
            elif action not in caps[cap_id]:
                bad.append(f"{aid}  ({flow}/{node}) action '{action}' 不在 capability '{cap_id}' actions {sorted(caps[cap_id])}")
        elif e.get("executor") == "subagent" and e.get("skill"):
            if skills and e["skill"] not in skills and e["skill"] not in SKILL_EXEMPT:
                bad.append(f"{aid}  ({flow}/{node}) 引用 skill '{e['skill']}' 不在 docs/templates/skills/（待建则入 SKILL_EXEMPT）")
    if bad:
        print("===== 不可解析/引用缺失（FAIL） =====")
        for b in bad:
            print("  " + b)

    # c) 失效豁免 + 全量覆盖判定
    stale = [aid for aid in sorted(model) if aid in EXEMPT_ACTIONS and in_corpus(aid)]
    if stale:
        print("===== 失效豁免（已被驱动，应移除豁免） =====")
        for aid in stale:
            print(f"  {aid}  {EXEMPT_ACTIONS[aid]}")
    for aid in model:
        in_corpus(aid)

    # 差集（动作级豁免不参与）
    diff = [aid for aid in sorted(model) if aid not in covered and aid not in EXEMPT_ACTIONS]
    print(f"===== 补测清单（模型 {len(model)} − 驱动覆盖 {len(covered)} − 动作豁免 {len(EXEMPT_ACTIONS)}）=====")
    if diff:
        for aid in diff:
            flow, node, _ = model[aid]
            print(f"  {aid}  ({flow}/{node})")
        print(f"\n差集 {len(diff)} 条 = 喂 C 组的补测清单：逐条落剧本，或确认走显式豁免")
    else:
        print("  （空）模型动作全集均被驱动覆盖")

    hard_fail = len(bad) + len(stale)
    soft_fail = len(diff)
    verdict = "FAIL" if hard_fail or (strict and soft_fail) else "PASS"
    print(f"\n结论：{verdict}（不可解析/引用缺失 {len(bad)} / 失效豁免 {len(stale)} / 补测差集 {soft_fail}{'，strict 视为 FAIL' if strict and soft_fail else ''}）")
    sys.exit(1 if verdict == "FAIL" else 0)


if __name__ == "__main__":
    main()
