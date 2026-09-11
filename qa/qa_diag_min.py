#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""缺陷诊断最小复现环（qa-diagnosis 证据采集，2026-09-10）。

逐缺陷最小场景驱动真实服务（scratch 库只读对账），每节独立、软失败继续。
用法：source /tmp/wbflow-qa-env.env && python3 qa_diag_min.py
"""
import json
import os
import subprocess
import urllib.error
import urllib.request

BASE = os.environ.get("QA_BASE", "http://127.0.0.1:18081")
TOKEN = os.environ.get("QA_TOKEN", "wbflow-qa-token")
MY = dict(
    host=os.environ.get("QA_MYSQL_HOST", "192.168.3.100"),
    port=os.environ.get("QA_MYSQL_PORT", "3306"),
    user=os.environ.get("QA_MYSQL_USER", "root"),
    pwd=os.environ.get("QA_MYSQL_PASSWORD", ""),
    db=os.environ.get("QA_DBNAME", ""),
)


def api(method, path, body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if data:
        req.add_header("Content-Type", "application/json")
    req.add_header("Authorization", "Bearer " + TOKEN)
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            raw = r.read().decode()
            return r.status, (json.loads(raw) if raw else {})
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw}


def sql(query):
    env = dict(os.environ, MYSQL_PWD=MY["pwd"])
    p = subprocess.run(
        ["mysql", "-h", MY["host"], "-P", MY["port"], "-u", MY["user"], "-N", "-B", MY["db"], "-e", query],
        capture_output=True, text=True, env=env, timeout=30)
    if p.returncode != 0:
        return "SQL_ERR: " + p.stderr.strip()[:200]
    return p.stdout.strip()


def show(tag, v):
    print(f"  [{tag}] {json.dumps(v, ensure_ascii=False)[:300]}")


def new_req(title, category="需求任务", flow=None):
    body = {"title": title, "category": category, "createdBy": "user-a", "owner": "pm"}
    if flow:
        body["flowDefsId"] = flow
    st, b = api("POST", "/api/requirements", body)
    assert st == 200 and b.get("ok"), f"创建需求失败 HTTP {st}: {b}"
    rid = b["requirement"]["requirementId"]
    st, b = api("GET", f"/api/requirements/{rid}")
    nodes = sorted(b["flowNodes"], key=lambda n: int(n["nodeOrder"]))
    return rid, {n["defNodeId"]: n["nodeId"] for n in nodes}, b


def assign(rid, nid, action, by):
    return api("POST", "/api/tasks", {"requirementId": rid, "nodeId": nid, "actionId": action, "assignee": by})


def tupdate(tid, event, progress=None):
    body = {"taskId": tid, "event": event, "updatedAt": 0}
    if progress is not None:
        body["progress"] = progress
    return api("PATCH", f"/api/tasks/{tid}", body)


def advance(rid, nid):
    return api("POST", f"/api/requirements/{rid}/nodes/{nid}/advance", {"requirementId": rid, "nodeId": nid})


def decide(rid, nid, verdict, reason):
    return api("POST", f"/api/requirements/{rid}/nodes/{nid}/decision",
               {"requirementId": rid, "nodeId": nid, "verdict": verdict, "reason": reason})


def review(rid, nid, role, cap, state):
    return api("POST", f"/api/requirements/{rid}/nodes/{nid}/review",
               {"requirementId": rid, "nodeId": nid, "role": role, "capabilityId": cap, "state": state})


def mytask(rid, role, nid):
    return api("POST", f"/api/requirements/{rid}/my-task", {"requirementId": rid, "role": role, "nodeId": nid})


def node_state(rid, nid):
    return sql(f"SELECT state FROM flow_nodes WHERE requirement_id='{rid}' AND node_id='{nid}'")


def current_node(rid):
    return sql(f"SELECT current_node_id FROM flows WHERE requirement_id='{rid}'")


# ============ BUG-003: plan pln-d7 死锁 ============
def bug003():
    print("\n===== BUG-003 plan pln-d7 死锁 =====")
    rid, N, _ = new_req("DIAG-BUG003 规划需求", category="规划审批任务")
    decide(rid, N["pln-d0"], "approved", "启动")
    # pln-d1 对齐：node action to-alignment
    st, b = assign(rid, N["pln-d1"], "to-alignment", "arch")
    show("assign to-alignment", (st, b.get("error", b.get("task", {}).get("taskId"))))
    tid = b["task"]["taskId"]
    tupdate(tid, "ack"); tupdate(tid, "progress", progress="diag")
    tupdate(tid, "complete")
    decide(rid, N["pln-d1"], "approved", "对齐确认")
    decide(rid, N["pln-d2"], "approved", "立项通过")
    # pln-d3 调研 produce（期望产物 research）
    st, b = assign(rid, N["pln-d3"], "research-cap.produce", "inv")
    show("assign research", (st, b.get("error", b.get("task", {}).get("taskId"))))
    tid = b["task"]["taskId"]
    tupdate(tid, "ack"); tupdate(tid, "progress", progress="diag")
    st, b = api("POST", f"/api/requirements/{rid}/nodes/{N['pln-d3']}/artifacts",
        {"productType": "research", "label": "《调研文档》", "kind": "document",
         "href": "v1.0/qa/fixtures/research.md", "addedBy": "inv"})
    show("add research artifact", (st, b.get("ok"), b.get("error", "")))
    st, b = tupdate(tid, "complete")
    show("complete research", (st, b.get("status"), b.get("error", "")))
    st, b = advance(rid, N["pln-d3"]); show("advance pln-d3", (st, b.get("outcome"), b.get("error", "")))
    # pln-d4 调研评审（arch 单评审位 → 机械收口）
    review(rid, N["pln-d4"], "arch", "arch.research-review", "agree")
    st, b = advance(rid, N["pln-d4"]); show("advance pln-d4", (st, b.get("outcome"), b.get("error", "")))
    # pln-d5 架构规划
    st, b = assign(rid, N["pln-d5"], "arch-cap.produce", "arch")
    show("assign arch", (st, b.get("error", b.get("task", {}).get("taskId"))))
    tid = b["task"]["taskId"]
    tupdate(tid, "ack"); tupdate(tid, "progress", progress="diag")
    st, b = api("POST", f"/api/requirements/{rid}/nodes/{N['pln-d5']}/artifacts",
        {"productType": "architecture", "label": "《架构规划》", "kind": "document",
         "href": "v1.0/qa/fixtures/architecture.md", "addedBy": "arch"})
    show("add architecture artifact", (st, b.get("ok"), b.get("error", "")))
    st, b = tupdate(tid, "complete")
    show("complete arch", (st, b.get("status"), b.get("error", "")))
    st, b = advance(rid, N["pln-d5"]); show("advance pln-d5", (st, b.get("outcome"), b.get("error", "")))
    # pln-d6 架构评审（pm 拍板位）
    review(rid, N["pln-d6"], "pm", "pm.architecture-review", "agree")
    decide(rid, N["pln-d6"], "approved", "架构通过")
    st, b = api("GET", f"/api/requirements/{rid}")
    show("current after pln-d6", current_node(rid))
    # BUG-006 对照：my-task 裸名 vs assign 全名
    st, b = mytask(rid, "leader", N["pln-d7"])
    show("my-task acts", [a.get("actionId") for a in b.get("activeActions", [])])
    st, b = assign(rid, N["pln-d7"], "create_reqs", "leader")
    show("assign 裸名 create_reqs", (st, b.get("error", "ok")))
    st, b = assign(rid, N["pln-d7"], "decide-cap.create_reqs", "leader")
    show("assign 全名 decide-cap.create_reqs", (st, b.get("ok")))
    tid = b["task"]["taskId"]
    tupdate(tid, "ack")
    st, b = tupdate(tid, "complete")
    show("complete create_reqs", (st, b.get("status")))
    show("pln-d7 node row", sql(f"SELECT node_id, state FROM flow_nodes WHERE requirement_id='{rid}' AND def_node_id='pln-d7'").replace("\t", " | "))
    show("task row", sql(f"SELECT task_id, action_id, task_type, status FROM tasks WHERE task_id='{tid}'").replace("\t", " | "))
    # BUG-003 判决：advance pln-d7
    st, b = advance(rid, N["pln-d7"])
    show("advance pln-d7", (st, b))
    show("current", current_node(rid))
    print(f"  VERDICT BUG-003: {'REPRODUCED (409 blocked, current!=pln-d8)' if st == 409 else 'NOT REPRODUCED'}")


# ============ BUG-004: update 全量字段 ============
def bug004():
    print("\n===== BUG-004 wb_update_requirement 全量字段 =====")
    rid, N, _ = new_req("DIAG-BUG004 局部更新")
    st, b = api("GET", f"/api/requirements/{rid}")
    req = b["requirement"]
    # 仅传想改字段（缺 category/status/priority）
    st, b = api("PUT", f"/api/requirements/{rid}", {
        "requirementId": rid, "title": req["title"], "priority": "P1",
        "updatedAt": req["updatedAt"]})
    show("PUT 仅 title+priority", (st, b))
    # 全量回显再试
    st, b2 = api("PUT", f"/api/requirements/{rid}", {
        "requirementId": rid, "title": req["title"], "description": req.get("description", ""),
        "category": req["category"], "priority": "P1", "status": req["status"],
        "owner": req.get("owner", ""), "updatedAt": req["updatedAt"]})
    show("PUT 全量回显", (st, b2.get("ok")))
    print(f"  VERDICT BUG-004: {'REPRODUCED (局部 400)' if st == 200 and 'category 必填' in json.dumps(b) else 'CHECK'}")


# ============ BUG-005: 未激活节点派发完成 ============
def bug005():
    print("\n===== BUG-005 任务总线不校验节点相位 =====")
    rid, N, _ = new_req("DIAG-BUG005 相位校验")
    # bus-02（产品设计）未激活（bus-01 未拍板）
    show("bus-02 state before", node_state(rid, N["bus-02"]))
    st, b = assign(rid, N["bus-02"], "product-design.produce", "pm")
    show("assign 未激活 bus-02", (st, b.get("ok")))
    tid = b["task"]["taskId"]
    tupdate(tid, "ack"); tupdate(tid, "progress", progress="diag")
    st, b = api("POST", f"/api/requirements/{rid}/nodes/{N['bus-02']}/artifacts",
        {"productType": "product_doc", "label": "《产品设计文档》", "kind": "document",
         "href": "v1.0/qa/fixtures/product_doc.md", "addedBy": "pm"})
    show("add product_doc artifact（未激活节点）", (st, b.get("ok"), b.get("error", "")))
    st, b = tupdate(tid, "complete")
    show("complete 未激活节点任务", (st, b.get("status"), b.get("error", "")))
    show("bus-02 state after", repr(node_state(rid, N["bus-02"])))
    print(f"  VERDICT BUG-005: {'REPRODUCED (孤儿 completed，节点 state 仍空)' if node_state(rid, N['bus-02']) == '' else 'NOT REPRODUCED'}")


# ============ BUG-008: escalate 通知 anchor_kind ============
def bug008():
    print("\n===== BUG-008 wb_escalate 通知 anchor_kind =====")
    rid, N, _ = new_req("DIAG-BUG008 上升通知")
    # 造一条 issue 评论
    st, b = api("POST", f"/api/requirements/{rid}/comments", {
        "requirementId": rid, "nodeId": N["bus-02"], "author": "qa", "roleId": "qa",
        "issueType": "issue", "text": "diag 冲突评论"})
    cid = b["issue"]["issueId"]
    # 路径1：wb_comment escalate=true
    st, b = api("POST", f"/api/requirements/{rid}/comments", {
        "requirementId": rid, "nodeId": N["bus-02"], "author": "qa", "roleId": "qa",
        "issueType": "issue", "text": "diag 冲突评论2", "escalate": True})
    cid2 = b["issue"]["issueId"]
    # 路径2：手动 wb_escalate
    st, b = api("POST", f"/api/comments/{cid}/escalate", {"operator": "qa", "reason": "diag manual"})
    show("wb_escalate", (st, b.get("escalated")))
    rows = sql(f"SELECT anchor_kind, issue_type, assignee, role_id, text FROM issues "
               f"WHERE requirement_id='{rid}' AND issue_type='notification'").split("\n")
    for r in rows:
        show("notification row", r.replace("\t", " | "))
    doc_rows = [r for r in rows if r.startswith("document\t")]
    esc_rows = [r for r in rows if r.startswith("escalation\t")]
    print(f"  VERDICT BUG-008: manual doc rows={len(doc_rows)} escalation rows(from comment path)={len(esc_rows)}")


# ============ OBS-001: gate 拍板不校验 exec 判据 ============
def obs001():
    print("\n===== OBS-001 bus-01 拍板不校验判据 =====")
    rid, N, _ = new_req("DIAG-OBS001 门禁拍板")
    st, b = decide(rid, N["bus-01"], "approved", "未执行 to-brd 直接拍板")
    show("decide bus-01（无 brd 产物/无任务，拍板口径=200 合法）", (st, b.get("nextNodeId", b.get("error", ""))))
    show("current", current_node(rid))
    n_art = sql(f"SELECT COUNT(*) FROM artifacts WHERE requirement_id='{rid}'")
    n_task = sql(f"SELECT COUNT(*) FROM tasks WHERE requirement_id='{rid}'")
    show("artifacts/tasks count", (n_art, n_task))
    print(f"  VERDICT OBS-001: {'按拍板口径合法（approve 不校验产物，最小原则）' if st == 200 else '行为变化（HTTP %d）——确认是否回归' % st}")


if __name__ == "__main__":
    bug003()
    bug004()
    bug005()
    bug008()
    obs001()
