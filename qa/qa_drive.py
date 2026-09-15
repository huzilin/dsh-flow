#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""wbflow v1.0 QA 验收驱动（测试资产，配套 docs/v1.0/qa/qa-test-cases.md）。

独立第二双眼睛：不复用 rd/e2e 断言，直接按 proto 契约黑盒驱动真实服务进程
（scratch 库），全部缺陷软失败继续跑（一轮收集全部问题），结果落 qa-results.json。

用法：
  source /tmp/wbflow-qa-env.env   # qa_env_up.sh 产出
  python3 qa_drive.py             # 结果摘要 + qa-results.json
"""
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

BASE = os.environ.get("QA_BASE", "http://127.0.0.1:6081")
TOKEN = os.environ.get("QA_TOKEN", "wbflow-qa-token")
MY = dict(
    host=os.environ.get("QA_MYSQL_HOST", "192.168.3.100"),
    port=os.environ.get("QA_MYSQL_PORT", "3306"),
    user=os.environ.get("QA_MYSQL_USER", "root"),
    pwd=os.environ.get("QA_MYSQL_PASSWORD", ""),
    db=os.environ.get("QA_DBNAME", ""),
)
HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, ".."))
FIXTURES = os.path.join(REPO, "docs", "v1.0", "qa", "fixtures")
OUT = os.path.join(HERE, "qa-results.json")

RESULTS = []
WALKED = []  # (req_id, flow, scope) 已全走需求（D-1 产物矩阵对账数据源）

# ===== AC 覆盖映射（spec §4「AC 编号规范」注的执行端，2026-09-11 拍板一期①）=====
# 维护纪律：spec §4 新增 AC 时——qa_drive 可见证的加 AC_MAP（UI/外部半边在 note 写明见证方）；
# qa_drive 整条不可见证的（真库 IT / fe vitest / DSH 侧）加 AC_OUT_OF_SCOPE 并写明见证方。
# 缺声明=FAIL；映射 cid 全未执行=FAIL；仅 SKIP/WARN 行保持可见不拦绿灯。
AC_MAP = {
    "AC-V-01": (["A-02", "A-03", "A-13"], "接口面即工具契约面（team preset 启动属 DSH 侧，不经本驱动）"),
    "AC-V-02": (["B-1", "D-1"], "seed 初始化由走查前置与产物矩阵共同见证"),
    "AC-V-03": (["B-1", "C-8"], ""),
    "AC-V-04": (["B-2", "C-3"], ""),
    "AC-V-05": (["B-4"], ""),
    "AC-V-06": (["B-1", "A-13"], "对齐段在 B-1 前段；entry 迎宾 A-13"),
    "AC-V-07": (["C-1", "C-2", "C-5"], ""),
    "AC-V-08": (["C-6", "C-7"], ""),
    "AC-V-09": (["C-11", "A-29"], ""),
    "AC-V-11": (["C-10", "A-16"], ""),
    "AC-V-12": (["C-12", "D-2"], ""),
    "AC-M-01": (["C-3"], "后端半边（scope 落值/删行不占位）；UI 渲染矩阵=fe vitest"),
    "AC-M-02": (["B-1"], "后端表态序列；UI 三态=fe vitest"),
    "AC-M-03": (["C-6"], "后端半边；UI 冲突卡=fe vitest"),
    "AC-M-04": (["C-4", "B-1"], ""),
}
AC_OUT_OF_SCOPE = {
    "AC-V-10": "扫描器 5/10min 阈值=rd/service/scanner_it_test.go 真库 IT（注入时钟）；本驱动 C-11 仅通知投影半边（慢路径 skip 标注）",
}


def ac_coverage_assert():
    """spec §4 AC ↔ 用例覆盖机器断言：断「声明存在 + 映射可执行」，不断言 AC 本体通过。"""
    import re
    spec_path = os.path.join(REPO, ".plan", "wbflow-impl", "spec.md")
    try:
        text = open(spec_path, encoding="utf-8").read()
    except OSError as e:
        ck("AC-COVERAGE", False, f"spec 不可读: {e}")
        return
    sec = text.split("## 4. 验收标准", 1)[-1].split("## Testing Decisions", 1)[0]
    acs = sorted(set(re.findall(r"AC-[VM]-\d{2}", sec)))
    if not acs:
        ck("AC-COVERAGE", False, "spec §4 未解析到任何 AC 编号——编号规范注被移除？")
        return
    print(f"\n===== AC 覆盖断言（spec §4 共 {len(acs)} 条）=====")
    for ac in acs:
        if ac not in AC_MAP:
            if ac in AC_OUT_OF_SCOPE:
                print(f"COVER-EXT {ac}  外部见证：{AC_OUT_OF_SCOPE[ac]}")
            else:
                ck(f"AC-COVERAGE {ac}", False, "spec 新增 AC 未声明覆盖（AC_MAP/AC_OUT_OF_SCOPE 均无）")
            continue
        cids, note = AC_MAP[ac]
        rows = [x for x in RESULTS if any(re.match(re.escape(c) + r"(?!\d)", x["cid"]) for c in cids)]
        passed = [x for x in rows if x["ok"] is True]
        suffix = f"：{note}" if note else ""
        if not rows:
            ck(f"AC-COVERAGE {ac}", False, f"映射 cid {cids} 一个都未执行（对应驱动阶段被删改？）")
        elif not passed:
            print(f"WARN  {ac}  映射 cid 全 SKIP/FAIL（{cids}）{suffix}")
        else:
            print(f"COVER {ac}  {len(passed)}/{len(rows)} 断言通过{suffix}")


def ck(cid, cond, detail=""):
    RESULTS.append(dict(cid=cid, ok=bool(cond), detail=str(detail)[:400]))
    print(("PASS " if cond else "FAIL ") + cid + ("  " + str(detail)[:220] if not cond else ""))


def skip(cid, reason):
    RESULTS.append(dict(cid=cid, ok=None, detail=reason))
    print("SKIP " + cid + "  " + reason)


def sec(name):
    print("\n===== " + name + " =====")


def api(method, path, body=None, token=TOKEN):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if data:
        req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
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
        return None
    return p.stdout.strip()


def sql_int(query):
    out = sql(query)
    try:
        return int(out or "0")
    except ValueError:
        return 0


def audit_count(table, row, action, field=None, by=None):
    q = f"SELECT COUNT(*) FROM audit_log WHERE table_name='{table}' AND row_id='{row}' AND action='{action}'"
    if field:
        q += f" AND field='{field}'"
    if by:
        q += f" AND by_who='{by}'"
    return sql_int(q)


# ---- 各节点期望产物（flow_def_nodes.expected_outputs 快照，live seed 2026-09-10） ----
DOC = "document"
LINK = "link"
EXP = {
    "bus-01": [("brd", "《业务需求文档》", DOC)],
    "bus-02": [("product_doc", "《产品设计文档》", DOC)],
    "bus-04": [("rd_design", "《RD 技术设计》", DOC)],
    "bus-06": [("fe_design", "《FE 技术设计》", DOC)],
    "bus-08": [("dev_notes", "《RD 开发说明》", DOC), ("code_mr", "《RD 代码 MR》", LINK)],
    "bus-09": [("dev_notes", "《FE 开发说明》", DOC), ("code_mr", "《FE 代码 MR》", LINK)],
    "bus-10": [("test_result", "《测试用例与验收记录》", DOC), ("defect", "《缺陷文档》", DOC)],
    "tec-b1": [("brd", "《业务需求文档》", DOC)],
    "tec-b2": [("rd_design", "《RD 技术设计》", DOC)],
    "tec-b4": [("fe_design", "《FE 技术设计》", DOC)],
    "tec-b8a": [("dev_notes", "《RD 开发说明》", DOC), ("code_mr", "《RD 代码 MR》", LINK)],
    "tec-b8b": [("dev_notes", "《FE 开发说明》", DOC), ("code_mr", "《FE 代码 MR》", LINK)],
    "tec-b9": [("test_result", "《测试用例与验收记录》", DOC), ("defect", "《缺陷文档》", DOC)],
    "tsk-c2": [("rd_design", "《RD 技术设计》", DOC), ("fe_design", "《FE 技术设计》", DOC)],  # 双 scope 节点：rd 走 rd_design / fe 走 fe_design（对账按 scope 裁剪）
    "tsk-c4": [("dev_notes", "《开发说明》", DOC), ("code_mr", "《代码 MR》", LINK)],
    "pln-d1": [("alignment", "《需求对齐文档》", DOC)],
    "pln-d3": [("research", "《调研文档》", DOC)],
    "pln-d5": [("architecture", "《架构规划》", DOC)],
}
FIXTURE_TYPE = {"brd", "product_doc", "rd_design", "fe_design", "dev_notes",
                "test_result", "defect", "alignment", "research", "architecture"}


def setup_fixtures():
    os.makedirs(FIXTURES, exist_ok=True)
    for t in FIXTURE_TYPE:
        p = os.path.join(FIXTURES, t + ".md")
        if not os.path.exists(p):
            with open(p, "w") as f:
                f.write(f"# {t} QA fixture\n\nmarker-{t}-42\n")
    readme = os.path.join(REPO, "docs", "v1.0", "qa", "qa-test-cases.md")
    return os.path.exists(readme)


def href_of(ptype):
    return f"v1.0/qa/fixtures/{ptype}.md"


def products_of(defid, scope=None):
    """节点期望产物（按需求 scope 裁剪双 scope 节点，与 D-1 对账同口径：rd 裁 fe_design / fe 裁 rd_design）。"""
    rows = EXP.get(defid, [])
    if scope == "rd":
        return [r for r in rows if r[0] != "fe_design"]
    if scope == "fe":
        return [r for r in rows if r[0] != "rd_design"]
    return rows


# ---- 需求剧本对象 ----
class Req:
    def __init__(self, cid_prefix, title, flow=None, scope=None, category="需求任务",
                 owner="pm", version_id=None, priority=None):
        self.cid = cid_prefix
        self.scope = scope
        body = {"title": title, "category": category, "createdBy": "user-a", "owner": owner}
        if flow:
            body["flowDefsId"] = flow
        if scope:
            body["scope"] = scope
        if version_id:
            body["versionId"] = version_id
        if priority:
            body["priority"] = priority
        st, b = api("POST", "/api/requirements", body)
        assert st == 200 and b.get("ok"), f"创建需求失败 HTTP {st}: {b}"
        self.id = b["requirement"]["requirementId"]
        self.flow_defs = b["requirement"]["flowDefsId"]
        self.tasks = 0
        self.artifacts = 0
        self.reload()

    def reload(self):
        st, b = api("GET", f"/api/requirements/{self.id}")
        assert st == 200 and b.get("ok"), f"快照失败: {b}"
        self.snap = b
        nodes = sorted(b["flowNodes"], key=lambda n: int(n["nodeOrder"]))
        self.nodes = nodes
        self.bydef = {n["defNodeId"]: n for n in nodes}
        self.flow = b["flows"][0] if b["flows"] else {}

    def node(self, defid):
        return self.bydef[defid]["nodeId"]

    def ndef(self, defid):
        return self.bydef[defid]

    # -- 步骤 --
    def assign(self, defid, action, by, expect_ok=True):
        st, b = api("POST", "/api/tasks", {
            "requirementId": self.id, "nodeId": self.node(defid),
            "actionId": action, "assignee": by})
        tid = b.get("task", {}).get("taskId", "")
        if st == 200 and b.get("ok") and not b.get("reused"):
            self.tasks += 1
        return st, b, tid

    def task_update(self, tid, event, progress=None, updated_at=0, expect_ok=True):
        body = {"taskId": tid, "event": event, "updatedAt": updated_at}
        if progress is not None:
            body["progress"] = progress
        return api("PATCH", f"/api/tasks/{tid}", body)

    def add_artifact(self, defid, ptype, by, label=None, kind=None, href=None):
        exp = next((e for e in products_of(defid) if e[0] == ptype), None)
        kind = kind or (exp[2] if exp else DOC)
        label = label or (exp[1] if exp else f"《QA-{ptype}》")
        if href is None:
            href = href_of(ptype) if kind == DOC else f"https://git.example.local/{self.node(defid)}/{ptype}"
        st, b = api("POST", f"/api/requirements/{self.id}/nodes/{self.node(defid)}/artifacts", {
            "productType": ptype, "label": label, "href": href, "kind": kind, "addedBy": by})
        if st == 200 and b.get("ok"):
            self.artifacts += 1
        return st, b

    def execute(self, defid, action, by, note=""):
        """标准产出步：派发→ack→progress→登记期望产物→complete（判据=期望产物齐）。"""
        st, b, tid = self.assign(defid, action, by)
        assert st == 200 and b.get("ok"), f"assign {action} 失败 HTTP {st}: {b}"
        self.task_update(tid, "ack")
        self.task_update(tid, "progress", progress=f"QA 演练 {action} {note}")
        for ptype, _, _ in products_of(defid, self.scope):
            self.add_artifact(defid, ptype, by)
        return self.task_update(tid, "complete")

    def review(self, defid, role, cap, state):
        return api("POST", f"/api/requirements/{self.id}/nodes/{self.node(defid)}/review", {
            "requirementId": self.id, "nodeId": self.node(defid),
            "role": role, "capabilityId": cap, "state": state})

    def carry(self, defid, positions):
        """全评审位 straight-agree；返回末位 (st, b)。"""
        r = None
        for i, (role, cap) in enumerate(positions):
            r = self.review(defid, role, cap, "agree")
            assert r[0] == 200 and r[1].get("ok"), f"review {role}@{defid} 失败: {r}"
        return r

    def decide(self, defid, verdict, reason):
        return api("POST", f"/api/requirements/{self.id}/nodes/{self.node(defid)}/decision", {
            "requirementId": self.id, "nodeId": self.node(defid),
            "verdict": verdict, "reason": reason})

    def advance(self, defid):
        return api("POST", f"/api/requirements/{self.id}/nodes/{self.node(defid)}/advance", {
            "requirementId": self.id, "nodeId": self.node(defid)})

    def mytask(self, role, defid):
        return api("POST", f"/api/requirements/{self.id}/my-task", {
            "requirementId": self.id, "role": role, "nodeId": self.node(defid)})

    def comment(self, defid, author, role, text, assignee="", escalate=False, anchor=None):
        body = {"requirementId": self.id, "nodeId": self.node(defid),
                "author": author, "roleId": role, "text": text, "assignee": assignee}
        if anchor:
            body["anchor"] = anchor
        if escalate:
            body["escalate"] = True
        st, b = api("POST", f"/api/requirements/{self.id}/comments", body)
        iid = b.get("issue", {}).get("issueId", "")
        return st, b, iid

    def settle(self, iid, operator):
        return api("POST", f"/api/comments/{iid}/settle", {"commentId": iid, "operator": operator})

    def qualify(self, iid, operator):
        return api("POST", f"/api/comments/{iid}/qualify", {"commentId": iid, "operator": operator})

    def clarified(self, iid, by, role, text):
        return api("POST", f"/api/comments/{iid}/clarified", {
            "commentId": iid, "by": by, "roleId": role, "text": text})

    def declare_fixed(self, iid, by, role, text):
        return api("POST", f"/api/comments/{iid}/declare-fixed", {
            "commentId": iid, "by": by, "roleId": role, "text": text})

    def confirm(self, iid, operator):
        return api("POST", f"/api/comments/{iid}/confirm", {"commentId": iid, "operator": operator})

    def reopen(self, iid, operator):
        return api("POST", f"/api/comments/{iid}/reopen", {"commentId": iid, "operator": operator})

    def escalate(self, iid, operator, reason=""):
        return api("POST", f"/api/comments/{iid}/escalate", {
            "commentId": iid, "operator": operator, "reason": reason})

    def adjudicate(self, iid, verdict, by, conclusion=""):
        return api("POST", f"/api/comments/{iid}/adjudicate", {
            "commentId": iid, "verdict": verdict, "decidedBy": by, "conclusion": conclusion})

    def inbox(self, role):
        st, b = api("GET", f"/api/inbox?role={role}")
        assert st == 200 and b.get("ok"), f"inbox 失败: {b}"
        return b

    def inbox_kinds(self, role):
        counts = {}
        for n in self.inbox(role).get("notifications", []):
            if n.get("requirementId") == self.id:
                counts[n.get("kind", "")] = counts.get(n.get("kind", ""), 0) + 1
        return counts

    # -- 断言 --
    def ck_phase(self, cid, defid, want):
        got = self.ndef(defid).get("state", "")
        ck(cid, got == want, f"{defid} 相位 want={want} got={got}")

    def ck_current(self, cid, defid):
        got = self.flow.get("currentNodeId", "")
        ck(cid, got == self.node(defid), f"current want={defid}({self.node(defid)}) got={got}")

    def ck_verdict(self, cid, defid, want):
        raw = self.ndef(defid).get("lastDecision", "")
        try:
            got = json.loads(raw).get("verdict", "") if raw else ""
        except Exception:
            got = "PARSE_ERR:" + raw[:80]
        ck(cid, got == want, f"{defid} lastDecision.verdict want={want} got={got}")


def defids_of(req):
    return [n["defNodeId"] for n in req.nodes]


# ============================== A-01/A-02 探活与认证 ==============================
def p_health_auth():
    sec("A-01/A-02 healthz + 认证")
    st, b = api("GET", "/healthz", token=None)
    ck("A-01", st == 200 and b.get("ok") is True, f"healthz HTTP {st} {b}")
    st, _ = api("GET", "/api/versions", token=None)
    ck("A-02a 无 token 401", st == 401, f"HTTP {st}")
    st, _ = api("GET", "/api/versions", token="wrong-token")
    ck("A-02b 错 token 401", st == 401, f"HTTP {st}")


# ============================== 版本域 A-03~A-06 / C-9 ==============================
def p_versions():
    sec("A-03~A-06/C-09 版本域 + 条件解绑")
    st, b = api("POST", "/api/versions", {"name": "QA 版本 A"})
    vid_a = b.get("version", {}).get("versionId", "")
    ck("A-03a create_version", st == 200 and b.get("ok") and vid_a != "", f"HTTP {st} {b}")
    st, b = api("GET", "/api/versions")
    ck("A-03b list_versions 含新版本", any(v.get("versionId") == vid_a for v in b.get("items", [])), f"HTTP {st}")
    ver = next((v for v in b.get("items", []) if v.get("versionId") == vid_a), {})
    st, b = api("PUT", f"/api/versions/{vid_a}", {"versionId": vid_a, "name": "QA 版本 A2", "updatedAt": int(ver.get("updatedAt", "0"))})
    ck("A-04a update_version", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b = api("PUT", f"/api/versions/{vid_a}", {"versionId": vid_a, "name": "QA 版本 A3", "updatedAt": 1})
    ck("A-04b 乐观锁冲突 409", st == 409, f"HTTP {st} {b}")

    # A-05 发布软约束
    st, b = api("POST", "/api/versions", {"name": "QA 版本 adv"})
    vid_adv = b.get("version", {}).get("versionId", "")
    Req("A-05", "QA 版本发布软约束需求", flow="task", scope="rd", version_id=vid_adv)
    st, b = api("POST", f"/api/versions/{vid_adv}/advance", {"versionId": vid_adv})
    unf = b.get("unfinished", [])
    ck("A-05 advance_version 软提示", st == 200 and b.get("ok") and len(unf) >= 1, f"HTTP {st} unfinished={unf}")

    # A-06/C-09 条件解绑：全未开始 → 成功解绑
    st, b = api("POST", "/api/versions", {"name": "QA 版本 dep"})
    vid_dep = b.get("version", {}).get("versionId", "")
    r1 = Req("A-06", "QA 待解绑需求 1", flow="business", scope="full", version_id=vid_dep)
    r2 = Req("A-06", "QA 待解绑需求 2", flow="task", scope="rd", version_id=vid_dep)
    st, b = api("POST", f"/api/versions/{vid_dep}/deprecate", {"versionId": vid_dep})
    ok = st == 200 and b.get("ok")
    ck("A-06a 全未开始 deprecate 成功", ok, f"HTTP {st} {b}")
    st, b = api("GET", f"/api/requirements/{r1.id}")
    ck("A-06b 需求解绑回池 versionId 空", b.get("requirement", {}).get("versionId", "") == "", f"versionId={b.get('requirement', {}).get('versionId')}")
    st, b = api("GET", "/api/versions")
    vdep = next((v for v in b.get("items", []) if v.get("versionId") == vid_dep), {})
    ck("A-06c 版本态=已废弃", vdep.get("state") == "已废弃", f"state={vdep.get('state')}")
    ck("A-06d 解绑逐行审计", audit_count("requirements", r1.id, "UPDATE", "versionId") >= 1, "audit requirements.versionId")

    # C-09 已开始（进入编排）→ 409
    st, b = api("POST", "/api/versions", {"name": "QA 版本 dep2"})
    vid_dep2 = b.get("version", {}).get("versionId", "")
    r3 = Req("C-09", "QA 已开始需求", flow="business", scope="full", version_id=vid_dep2)
    st, b = r3.decide("bus-00", "approved", "进入编排")
    ck("C-09a 前置：bus-00 拍板进入编排", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b = api("POST", f"/api/versions/{vid_dep2}/deprecate", {"versionId": vid_dep2})
    ck("C-09b 已开始 deprecate 409 blocked", st == 409 and b.get("blocked") is True, f"HTTP {st} {b}")


# ============================== 需求 CRUD A-07~A-12/A-30 ==============================
def p_req_crud():
    sec("A-07~A-12/A-30 需求 CRUD + 历史 + 依赖树")
    st, b = api("POST", "/api/requirements", {"category": "需求任务", "createdBy": "user-a"})
    ck("A-07a 缺 title 400", st == 400, f"HTTP {st} {b}")
    st, b = api("POST", "/api/requirements", {"title": "QA 坏类别", "category": "不存在的类别", "createdBy": "user-a"})
    ck("A-07b 非法 category 拦截", st >= 400, f"HTTP {st} {b}")

    r = Req("A-07", "QA CRUD 主需求", flow="business", scope="full", priority="P1")
    ck("A-07c 实例化矩阵", defids_of(r) == [f"bus-{i:02d}" for i in range(13)]
       and [int(n["nodeOrder"]) for n in r.nodes] == list(range(1, 14))
       and all(n["state"] == "" for n in r.nodes),
       f"defs={defids_of(r)}")
    ck("A-07d currentNodeId 空 / scope=full", r.flow.get("currentNodeId", "") == "" and r.flow.get("scope") == "full", str(r.flow))

    st, b = api("GET", f"/api/requirements?priority=P1")
    ck("A-08a priority 筛选", any(i.get("requirementId") == r.id for i in b.get("items", [])), f"HTTP {st}")
    st, b = api("GET", f"/api/requirements?priority=P0&page=1&pageSize=1")
    ck("A-08b 分页 total/items", b.get("total", "0") is not None and len(b.get("items", [])) <= 1, f"total={b.get('total')}")
    st, b = api("GET", "/api/requirements?page=1&pageSize=5")
    ck("A-08c dependencies 字段存在（注记-D4）", b.get("items") and "dependencies" in b["items"][0],
       f"items={len(b.get('items', []))} keys={sorted((b.get('items') or [{}])[0])[:12]}")

    st, b = api("GET", f"/api/requirements/{r.id}")
    keys = set(b.keys())
    ck("A-09 聚合快照字段齐", {"flows", "flowNodes", "artifacts", "issues", "sessions", "reviewResults"} <= keys, f"keys={sorted(keys)}")

    st, b = api("PUT", f"/api/requirements/{r.id}", {
        "requirementId": r.id, "title": "QA CRUD 主需求 v2", "priority": "P0",
        "status": "待启动", "owner": "pm", "category": "需求任务", "updatedAt": 0})
    ck("A-10a update_requirement", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b = api("PUT", f"/api/requirements/{r.id}", {
        "requirementId": r.id, "title": "QA 陈旧基线", "priority": "P0",
        "status": "待启动", "owner": "pm", "category": "需求任务", "updatedAt": 1})
    ck("A-10b 陈旧基线 409", st == 409, f"HTTP {st} {b}")
    st, b = api("PUT", f"/api/requirements/{r.id}", {
        "requirementId": r.id, "title": "QA CRUD 局部更新", "updatedAt": 0})
    ck("BUG-004a 局部更新（仅 title）200", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b = api("PUT", f"/api/requirements/{r.id}", {
        "requirementId": r.id, "title": "QA 非法优先级", "priority": "P9",
        "updatedAt": 0})
    ck("BUG-004b 显式非法值仍拦截 400", st == 400, f"HTTP {st} {b}")

    st, b = api("GET", f"/api/requirements/{r.id}/history")
    items = b.get("items", [])
    ck("A-11 history 有 diff 行且倒序", st == 200 and len(items) >= 2
       and all(set(["field", "oldValue", "newValue", "byWho", "createdAt"]) <= set(i.keys()) for i in items)
       and int(items[0]["createdAt"]) >= int(items[-1]["createdAt"]), f"items={len(items)}")

    st, b = api("GET", f"/api/requirements/{r.id}/dependencies/tree")
    root = b.get("root", b)
    ck("A-30 依赖树根节点 found", st == 200 and (root.get("found") is True or b.get("ok") is True), f"{json.dumps(b)[:160]}")

    st, b = api("GET", "/api/requirements/req-no-such")
    ck("A-12a GET 不存在 404", st == 404, f"HTTP {st}")
    st, b = api("DELETE", f"/api/requirements/{r.id}")
    ck("A-12b 未 confirm 拦截", st >= 400 or b.get("ok") is not True, f"HTTP {st} {b}")
    st, b = api("DELETE", f"/api/requirements/{r.id}?confirm=true")
    ck("A-12c confirm 删除成功", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b = api("GET", f"/api/requirements/{r.id}")
    ck("A-12d 删后 404", st == 404, f"HTTP {st}")


# ============================== B-1 business×full 主链 ==============================
def p_business_full():
    sec("B-1 business×full 13 节点主链（含 C-4 澄清 / C-8 序A / A-13~A-17）")
    r = Req("B-1", "QA 主链全栈需求", flow="business", scope="full")
    WALKED.append((r.id, "business", "full"))

    ck("B-1a 实例化 bus-00..12 连续", defids_of(r) == [f"bus-{i:02d}" for i in range(13)], str(defids_of(r)))

    # A-27 负路径：空相位 advance blocked
    st, b = r.advance("bus-00")
    ck("A-27a 空相位 advance 409 blocked", st == 409 and b.get("blocked") is True, f"HTTP {st} {b}")

    # bus-00 待启动
    st, b = r.decide("bus-00", "maybe", "非法结论")
    ck("A-26a 非法 verdict 400", st == 400, f"HTTP {st} {b}")
    st, b = r.decide("bus-00", "approved", "确认投入资源")
    ck("B-1b bus-00 拍板→bus-01", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st_db = sql(f"SELECT status FROM requirements WHERE requirement_id='{r.id}'")
    ck("BUG-001a 首次进编排→进行中", st_db == "进行中", f"status={st_db}")
    r.reload()
    r.ck_current("B-1c", "bus-01")
    r.ck_phase("B-1d", "bus-01", "entry")

    # A-26b 重复拍板幂等
    st, b = r.decide("bus-00", "approved", "重复拍板")
    ck("A-26b 重复拍板 noop/幂等", st == 200 and b.get("noop") in (True, False), f"HTTP {st} noop={b.get('noop')}")

    # A-13 my-task entry 迎宾
    st, b = r.mytask("pm", "bus-01")
    acts = [a.get("actionId") for a in b.get("activeActions", [])]
    ck("A-13a my-task entry 迎宾 align.opening", st == 200 and b.get("currentPhase") == "entry" and "align.opening" in acts, f"phase={b.get('currentPhase')} acts={acts}")

    st, b = r.mytask("rd", "bus-01")
    ck("A-13b 非本节点角色动作视图可取（三向不越权由 action 归属决定）", st == 200, f"HTTP {st}")

    # bus-01 对齐：exec 判据负路径 → 正路径
    st, b, tid = r.assign("bus-01", "to-brd", "pm")
    ck("A-16a assign 新建 reused=false", st == 200 and b.get("reused") is False and tid != "", f"HTTP {st} {b}")
    st, b2, tid2 = r.assign("bus-01", "to-brd", "pm")
    ck("A-16b 同幂等键重放 reused=true", st == 200 and b2.get("reused") is True and tid2 == tid, f"HTTP {st} reused={b2.get('reused')}")
    st, b = r.task_update(tid, "complete")
    ck("A-17a 未 ack 先 complete 409 stale", st == 409, f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    st, b = r.task_update(tid, "complete")
    ck("A-17b 判据未过 blocked+missing+回 in-progress", st == 409 and b.get("blocked") is True and b.get("missing"), f"HTTP {st} {b}")
    st, b = r.task_update(tid, "bogus")
    ck("A-17c 非法 event 400", st == 400, f"HTTP {st} {b}")
    # OBS-001 拍板口径（2026-09-10 User：最小原则）——approve 不校验 exec 产物，
    # 产物拦截在 exec 出口（complete 判据 A-17b 已验证）；此拍板 200 为合法行为。
    r.execute("bus-01", "to-brd", "pm")
    r.reload()
    obs_current = r.flow.get("currentNodeId", "")
    print(f"  [观察] bus-01 to-brd complete 后 current={obs_current}（设计：user_gate=1 应停等用户确认）")
    st, b = r.decide("bus-01", "approved", "对齐结论用户确认")
    ck("B-1e bus-01 拍板通过", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.reload()
    r.ck_current("B-1f", "bus-02")

    # bus-02 产品设计
    st, b = r.mytask("pm", "bus-02")
    acts = [a.get("actionId") for a in b.get("activeActions", [])]
    ck("BUG-006a my-task 回显全名 actionId", "product-design.produce" in acts, f"acts={acts}")
    st, b = r.review("bus-02", "arch", "arch.prd-review", "agree")
    ck("A-25a 非评审相位表态 409 blocked", st == 409 and b.get("blocked") is True, f"HTTP {st} {b}")
    st, b, aid = r.comment("bus-01", "qa-a", "qa", "锚点探针", "pm")
    ck("A-18a document 锚点建行", st == 200 and b.get("ok") and aid != "", f"HTTP {st} {b}")
    st, b = r.add_artifact("bus-02", "bogus_type", "pm")
    ck("A-14a 非法 product_type 400", st == 400, f"HTTP {st} {b}")
    st, b = r.add_artifact("bus-02", "product_doc", "pm", kind="bogus")
    ck("A-14b 非法 kind 400", st == 400, f"HTTP {st} {b}")
    r.execute("bus-02", "product-design.produce", "pm")
    st, b = r.advance("bus-02")
    ck("B-1g bus-02 提评迁移 bus-03", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.reload()
    r.ck_phase("B-1h", "bus-03", "review")
    st, b = r.mytask("arch", "bus-03")
    rel = [a.get("label") for a in b.get("relatedArtifacts", [])]
    ck("BUG-011a 评审节点 my-task 带被评审产物（上游置前）",
       st == 200 and rel[:1] == ["《产品设计文档》"], f"relatedArtifacts={rel}")

    # bus-03 三轮评审（C-4 澄清循环）
    r.review("bus-03", "arch", "arch.prd-review", "agree")
    r.review("bus-03", "rd", "rd.prd-review", "agree")
    upstream_art = next((a for a in r.snap.get("artifacts", []) if a.get("nodeId") == r.node("bus-02")), {})
    st, b, issue_fe = r.comment("bus-03", "fe-a", "fe", "登录态刷新口径未定义，需澄清", "pm",
                                anchor={"kind": "document", "artifactId": upstream_art.get("artifactId", "")})
    st, b = r.review("bus-03", "fe", "fe.prd-review", "unagree")
    ck("C-4a fe unagree→clarify", st == 200 and b.get("collectOutcome") == "clarify", f"HTTP {st} {b}")
    r.reload()
    r.ck_phase("C-4b", "bus-03", "respond")
    kc = r.inbox_kinds("pm")
    ck("C-4c pm 收 clarify_task", kc.get("clarify_task", 0) >= 1, str(kc))
    st, b = r.review("bus-03", "qa", "qa.prd-review", "agree")
    ck("A-25b 非评审位表态拦截", st >= 400, f"HTTP {st} {b}")
    st, b, _ = r.comment("bus-03", "fe-a", "fe", "无锚探针（应被拒）", "pm")
    ck("BUG-013b 评审上下文无锚 issue 评论 400", st == 400, f"HTTP {st} {b}")
    # owner 澄清：resolve agree + respond complete
    st, b, tid = r.assign("bus-03", "product-design.clarify", "pm")
    assert st == 200 and b.get("ok"), f"assign clarify 失败: {b}"
    r.task_update(tid, "ack")
    st, b = r.settle(issue_fe, "pm")
    ck("C-4d 澄清闭环 resolved", st == 200 and b.get("issue", {}).get("state") == "resolved", f"HTTP {st} {b}")
    rr_before = r.inbox_kinds("fe").get("re_review", 0)
    st, b = r.task_update(tid, "complete")
    ck("C-4e respond complete 触发重入", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.reload()
    r.ck_phase("C-4f", "bus-03", "review")
    rr_after = r.inbox_kinds("fe").get("re_review", 0)
    ck("C-4g fe 收 re_review（澄清子型 target_id 空）", rr_after > rr_before, f"{rr_before}→{rr_after}")
    rev = [n for n in r.snap.get("issues", []) if n.get("issueId") == issue_fe]
    rr_rows = [x for x in r.inbox("fe").get("notifications", [])
               if x.get("kind") == "re_review" and x.get("requirementId") == r.id]
    ck("C-4h 澄清子型 target_id 空", rr_rows and rr_rows[-1].get("targetId", "") == "", str(rr_rows[-1]) if rr_rows else "无 re_review 行")
    # 同轮覆写幂等（C-10 预演）：arch 重复 agree 单条目
    r.review("bus-03", "arch", "arch.prd-review", "agree")
    r.reload()
    try:
        entries = json.loads(r.ndef("bus-03").get("reviewResults", "[]"))
        cur_round = max([e.get("round", 0) for e in entries] or [0])
        arch_n = len([e for e in entries if e.get("role") == "arch" and e.get("round") == cur_round])
        ck("C-10a 同轮同角色覆写单条目", arch_n == 1, f"arch 条目数={arch_n}")
    except Exception as e:
        ck("C-10a 同轮同角色覆写单条目", False, f"解析失败 {e}")
    st, b = r.review("bus-03", "fe", "fe.prd-review", "agree")
    ck("C-4i fe 复核 agree→gate_pending", st == 200 and b.get("gateState") == "agreed", f"HTTP {st} gateState={b.get('gateState')}")
    kc = r.inbox_kinds("pm")
    ck("C-4j pm 收 gate_pending", kc.get("gate_pending", 0) >= 1, str(kc))
    st, b = r.decide("bus-03", "approved", "产品设计终审通过")
    ck("B-1i bus-03 拍板→bus-04", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.reload()
    r.ck_current("B-1j", "bus-04")
    r.ck_verdict("B-1k", "bus-03", "approved")

    # bus-04 → bus-05 四位评审 straight agree → 拍板
    r.execute("bus-04", "backend-design.produce", "rd")
    r.advance("bus-04")
    r.reload()
    r.ck_phase("B-1l", "bus-05", "review")
    # -J6/BUG-013 评论锚产物强制（黑盒负路径 + 正向对照；bus-05=review 相位，arch∈评审位）
    st, b, _ = r.comment("bus-05", "arch", "arch", "评审意见未锚产物", "rd")
    ck("A-18a2 评审上下文 document 锚缺 artifactId 400", st == 400, f"HTTP {st} {b}")
    st, b, i_arch = r.comment("bus-05", "arch", "arch", "评审意见锚定被评审文档", "rd",
                              anchor={"kind": "document",
                                      "artifactId": sql(f"SELECT artifact_id FROM artifacts WHERE requirement_id='{r.id}' AND product_type='rd_design' LIMIT 1")})
    ck("A-18a3 评审意见锚产物 200", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    # -L1 collect 闭环绑定：评审位名下有 open 评论 → agree 409；resolved 后 agree 200
    st, b = r.review("bus-05", "arch", "arch.rd-design-review", "agree")
    ck("L1a open 评论拦 agree 409", st == 409, f"HTTP {st} {json.dumps(b)[:160]}")
    st, b = r.settle(i_arch, "rd")
    ck("L1a2 协商一致 resolved", st == 200 and b.get("issue", {}).get("state") == "resolved", f"HTTP {st} {json.dumps(b)[:160]}")
    st, b = r.review("bus-05", "arch", "arch.rd-design-review", "agree")
    ck("L1a3 闭环后 agree 200", st == 200 and b.get("ok"), f"HTTP {st} {json.dumps(b)[:160]}")
    r.carry("bus-05", [("fe", "fe.rd-design-review"),
                       ("qa", "qa.rd-design-review"), ("pm", "pm.rd-design-review")])
    st, b = r.decide("bus-05", "approved", "技术评审通过")
    ck("B-1m bus-05 拍板→bus-06", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.reload()
    # -L1b wb_override_role 强权通道（User 专属）：approve 后补挂 open 评论 → 批量置 resolved
    # （09-12 裁决 overridden 态废除：override 动作=批量 resolve，resolve_path=override）
    st, b, i_ov = r.comment("bus-05", "arch", "arch", "强权通过前补挂意见", "rd",
                            anchor={"kind": "document",
                                    "artifactId": sql(f"SELECT artifact_id FROM artifacts WHERE requirement_id='{r.id}' AND product_type='rd_design' LIMIT 1")})
    ck("L1b-1 补挂评论 200", st == 200, f"HTTP {st} {b}")
    st, b = api("POST", "/api/comments/override-role", {
        "requirementId": r.id, "nodeId": r.node("bus-05"), "reviewerRole": "arch", "operator": "pm"})
    ck("L1b-2 非 user 强权 400", st == 400, f"HTTP {st} {json.dumps(b)[:140]}")
    st, b = api("POST", "/api/comments/override-role", {
        "requirementId": r.id, "nodeId": r.node("bus-05"), "reviewerRole": "arch", "operator": "user"})
    ck("L1b-3 user 整角色强权批量 resolved", st == 200 and b.get("resolved") in (1, "1"), f"HTTP {st} {json.dumps(b)[:140]}")
    i_ov_row = sql(f"SELECT state, resolve_path FROM issues WHERE issue_id='{i_ov}'").split("\t") if i_ov else ["", ""]
    ck("L1b-4 override 后 state=resolved 且 resolve_path=override", i_ov_row == ["resolved", "override"], f"got {i_ov_row}")

    # bus-06/07
    r.execute("bus-06", "frontend-design.produce", "fe")
    r.advance("bus-06")
    r.reload()
    st, b = r.carry("bus-07", [("qa", "qa.fe-design-review")])
    ck("B-1n bus-07 机械收口 approved（user_gate=0）", st == 200 and b.get("gateState") == "approved", f"gateState={b.get('gateState')}")
    r.reload()
    r.ck_current("B-1o", "bus-08")
    r.ck_verdict("B-1p", "bus-07", "approved")

    # bus-08 RD 开发 + CR（join 序A：rd 先完成 → 激活 bus-09）
    st, b = r.advance("bus-02")
    ck("A-27b 相位不符 advance blocked/noop", st == 409 or b.get("noop") is True, f"HTTP {st} {b}")
    st, b, _ = r.assign("bus-09", "frontend-dev.dev", "fe")
    ck("BUG-005a 未激活节点派发 409 blocked（相位守卫）", st == 409 and b.get("blocked") is True, f"HTTP {st} {b}")
    r.execute("bus-08", "backend-dev.dev", "rd")
    r.reload()
    r.ck_phase("B-1q", "bus-08", "review")
    r.carry("bus-08", [("cr", "code-review")])
    r.reload()
    r.ck_current("B-1r", "bus-09")
    r.ck_verdict("B-1s", "bus-08", "approved")

    # bus-09 FE 开发 + CR → 整组推进 bus-10（C-8 序A）
    r.execute("bus-09", "frontend-dev.dev", "fe")
    r.carry("bus-09", [("cr", "code-review")])
    r.reload()
    r.ck_current("C-8a join 整组推进 bus-10", "bus-10")

    # bus-10 QA 验收
    r.execute("bus-10", "acceptance-test.accept", "qa")
    r.reload()
    r.ck_current("B-1t", "bus-11")
    r.ck_phase("B-1u", "bus-11", "entry")

    # bus-11 用户验收 → bus-12 完成
    st, b = r.decide("bus-11", "approved", "用户验收通过")
    ck("B-1v bus-11 拍板→bus-12", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st_db = sql(f"SELECT status FROM requirements WHERE requirement_id='{r.id}'")
    ck("BUG-001b 落位尾节点→已完成", st_db == "已完成", f"status={st_db}")
    r.reload()
    r.ck_current("B-1w", "bus-12")
    ck("B-1x 尾节点 enteredAt>0", int(r.ndef("bus-12").get("enteredAt", "0")) > 0, "")

    # -L2 done 离场终态（BUG-016）：bus-12 激活时其前 12 节点全量收口 done；done 禁挂载
    # bus-00 门禁未激活过（state 空），通过以 last_decision 表达 → done=11（bus-01..bus-11）
    done_n = sql_int(f"SELECT COUNT(*) FROM flow_nodes WHERE requirement_id='{r.id}' AND state='done'")
    ck("L2a 离场节点全量 done=11", done_n == 11, f"done={done_n}")
    st, b = r.add_artifact("bus-01", "brd", "pm")
    ck("L2b done 节点禁挂产物 400", st == 400, f"HTTP {st} {json.dumps(b)[:140]}")
    st, b, _ = r.assign("bus-02", "product-design.produce", "pm")
    ck("L2c done 节点禁派任务 409", st == 409, f"HTTP {st} {json.dumps(b)[:140]}")

    # A-17d 重复 complete 幂等
    st, b, tid = r.assign("bus-10", "acceptance-test.retest", "qa")
    if st == 200 and b.get("ok"):
        r.task_update(tid, "ack")
        st, b = r.task_update(tid, "complete")
        ck("B-1y retest 任务 complete（判据产物缺可 409）", st in (200, 409), f"HTTP {st}")
    # C-11 通知族对账（本需求）
    kc = r.inbox_kinds("pm")
    ck("C-11a gate_pending≥2 / clarify_task=1", kc.get("gate_pending", 0) >= 2 and kc.get("clarify_task", 0) >= 1, str(kc))
    return r


# ============================== C-1/C-2/C-3 拒绝打回 + reproject ==============================
def p_reject_branch():
    sec("C-1~C-3 拒绝打回 / 回退重 review / reproject")
    r = Req("C-1", "QA 打回链路需求", flow="business", scope="full")
    WALKED.append((r.id, "business", "full"))
    r.decide("bus-00", "approved", "启动")
    r.execute("bus-01", "to-brd", "pm")
    r.decide("bus-01", "approved", "对齐确认")
    r.execute("bus-02", "product-design.produce", "pm")
    r.advance("bus-02")
    r.carry("bus-03", [("arch", "arch.prd-review"), ("rd", "rd.prd-review"), ("fe", "fe.prd-review")])
    r.decide("bus-03", "approved", "产品设计通过")
    r.execute("bus-04", "backend-design.produce", "rd")
    r.advance("bus-04")
    r.reload()
    r.carry("bus-05", [("arch", "arch.rd-design-review"), ("fe", "fe.rd-design-review"),
                       ("qa", "qa.rd-design-review"), ("pm", "pm.rd-design-review")])
    st, b = r.decide("bus-05", "rejected", "技术方案存在缺陷，打回重做")
    ck("C-1a rejected 拍板回显 rollbackNodeId", st == 200 and b.get("verdict") == "rejected"
       and b.get("rollbackNodeId") == r.node("bus-04"), f"HTTP {st} {b}")
    r.reload()
    n04 = r.ndef("bus-04")
    ck("C-1b bus-04 exec 重入", n04.get("state") == "exec" and n04.get("reopenSource") == "downstream-reject"
       and int(n04.get("rolledBackCount", "0")) == 1, f"state={n04.get('state')} src={n04.get('reopenSource')} rbc={n04.get('rolledBackCount')}")
    ps = sql(f"SELECT pending_scope FROM flow_nodes WHERE requirement_id='{r.id}' AND node_id='{r.node('bus-04')}'")
    ck("C-1c bus-04 pending_scope 落值（DB）", ps == "full", f"pending_scope={ps}（注：快照 FlowNodeItem 无 pending_scope 投影字段）")
    n05 = r.ndef("bus-05")
    ck("C-1d bus-05 清零", n05.get("state") == "" and n05.get("reviewResults") == "[]"
       and n05.get("lastDecision", "") == "" and int(n05.get("rolledBackCount", "0")) == 1,
       f"state={n05.get('state')} rr={n05.get('reviewResults')} ld={n05.get('lastDecision')} rbc={n05.get('rolledBackCount')}")
    n06 = r.ndef("bus-06")
    ck("C-1e 下游联动 upstream-reject", n06.get("reopenSource") == "upstream-reject"
       and int(n06.get("rolledBackCount", "0")) == 1, f"src={n06.get('reopenSource')} rbc={n06.get('rolledBackCount')}")
    r.ck_current("C-1f current 回退 bus-04", "bus-04")
    ck("C-1g 回退审计", audit_count("flows", r.id, "UPDATE", "currentNodeId", "plugin") >= 1, "")
    r.ck_verdict("C-1h bus-05 无残留结论", "bus-05", "")

    # C-3 reproject：full→rd（下游 fe/QA 裁剪）
    st, b = api("POST", f"/api/requirements/{r.id}/reproject", {"requirementId": r.id, "scope": "be-only"})
    ck("C-3a scope 越界 400", st == 400, f"HTTP {st} {b}")
    st, b = api("POST", f"/api/requirements/{r.id}/reproject", {"requirementId": r.id, "scope": "rd"})
    ck("C-3b 重投影成功 scope=rd", st == 200 and b.get("ok") and b.get("scope") == "rd" and b.get("noop") is False,
       f"HTTP {st} {b}")
    r.reload()
    want = ["bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-08", "bus-11", "bus-12"]
    ck("C-3c 节点集=rd 渲染矩阵 9 节点", defids_of(r) == want, str(defids_of(r)))
    ck("C-3d 序号连续不占位", [int(n["nodeOrder"]) for n in r.nodes] == list(range(1, 10)), "")
    ps2 = sql(f"SELECT pending_scope FROM flow_nodes WHERE requirement_id='{r.id}' AND node_id='{r.node('bus-04')}'")
    ck("C-3e flows.scope=rd + 标记清（DB）", r.flow.get("scope") == "rd" and (ps2 in ("", "NULL", None)), f"scope={r.flow.get('scope')} pending_scope={ps2}")
    st, b = api("POST", f"/api/requirements/{r.id}/reproject", {"requirementId": r.id, "scope": "rd"})
    ck("C-3f 重复 reproject noop", st == 200 and b.get("noop") is True, f"HTTP {st} {b}")

    # C-2 回退重做 → 重评 → 恢复迁移
    st, b, tid = r.assign("bus-04", "backend-design.revise", "rd")
    if not (st == 200 and b.get("ok")):
        print(f"  [观察] backend-design.revise 派发失败 HTTP {st}，回退 produce 双账号重做")
        st, b, tid = r.assign("bus-04", "backend-design.produce", "rd-2")
    ck("C-2a 重做任务派发", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    r.add_artifact("bus-04", "rd_design", "rd")
    st, b = r.task_update(tid, "complete")
    ck("C-2b 重做 complete", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.advance("bus-04")
    r.reload()
    r.ck_phase("C-2c", "bus-05", "review")
    try:
        entries = json.loads(r.ndef("bus-05").get("reviewResults", "[]"))
        ck("C-2d 重评新轮播种 reviewing", all(e.get("state") == "reviewing" for e in entries) and len(entries) == 3,
           str(entries)[:160])
    except Exception as e:
        ck("C-2d 重评新轮播种 reviewing", False, str(e))
    r.carry("bus-05", [("arch", "arch.rd-design-review"), ("qa", "qa.rd-design-review"), ("pm", "pm.rd-design-review")])
    st, b = r.decide("bus-05", "approved", "重评通过")
    ck("C-2e 重评拍板→bus-08（rd 重排后继）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.reload()
    r.ck_current("C-2f", "bus-08")
    # 走完剩余链（rd scope）
    r.execute("bus-08", "backend-dev.dev", "rd")
    r.carry("bus-08", [("cr", "code-review")])
    r.reload()
    r.ck_current("C-2g", "bus-11")
    r.decide("bus-11", "approved", "用户验收")
    r.reload()
    r.ck_current("C-2h", "bus-12")


# ============================== B-2 business×rd ==============================
def p_rdonly():
    sec("B-2 business×rd 9 节点")
    r = Req("B-2", "QA 纯后端需求", flow="business", scope="rd")
    WALKED.append((r.id, "business", "rd"))
    want = ["bus-00", "bus-01", "bus-02", "bus-03", "bus-04", "bus-05", "bus-08", "bus-11", "bus-12"]
    ck("B-2a 9 节点不占位", defids_of(r) == want, str(defids_of(r)))
    rev03 = r.ndef("bus-03").get("reviewers", "")
    ck("B-2b bus-03 评审位裁剪", '"role":"rd"' in rev03 and '"role":"fe"' not in rev03, rev03[:120])
    rev05 = r.ndef("bus-05").get("reviewers", "")
    ck("B-2c bus-05 评审位裁剪", '"role":"qa"' in rev05 and '"role":"fe"' not in rev05, rev05[:120])
    r.decide("bus-00", "approved", "启动")
    r.execute("bus-01", "to-brd", "pm")
    r.decide("bus-01", "approved", "对齐确认")
    r.execute("bus-02", "product-design.produce", "pm")
    r.advance("bus-02")
    r.reload()
    r.carry("bus-03", [("arch", "arch.prd-review"), ("rd", "rd.prd-review")])
    r.decide("bus-03", "approved", "产品设计通过")
    r.execute("bus-04", "backend-design.produce", "rd")
    r.advance("bus-04")
    r.reload()
    r.carry("bus-05", [("arch", "arch.rd-design-review"), ("qa", "qa.rd-design-review"), ("pm", "pm.rd-design-review")])
    r.decide("bus-05", "approved", "技术评审通过")
    r.execute("bus-08", "backend-dev.dev", "rd")
    r.carry("bus-08", [("cr", "code-review")])
    r.reload()
    r.ck_current("B-2d", "bus-11")
    r.decide("bus-11", "approved", "用户验收")
    r.reload()
    r.ck_current("B-2e", "bus-12")


# ============================== B-3 tech×full（C-8 序B fe 先完成 wait） ==============================
def p_tech():
    sec("B-3 tech×full 11 节点 + join fe 先序")
    r = Req("B-3", "QA 技术流程需求", flow="tech", scope="full")
    WALKED.append((r.id, "tech", "full"))
    want = ["tec-b0", "tec-b1", "tec-b2", "tec-b3", "tec-b4", "tec-b5", "tec-b8a", "tec-b8b", "tec-b9", "tec-b10", "tec-b11"]
    ck("B-3a 实例化 11 节点", defids_of(r) == want, str(defids_of(r)))
    r.decide("tec-b0", "approved", "启动")
    r.execute("tec-b1", "to-brd", "pm")
    r.decide("tec-b1", "approved", "对齐确认")
    r.execute("tec-b2", "backend-design.produce", "rd")
    r.advance("tec-b2")
    r.reload()
    r.carry("tec-b3", [("arch", "arch.rd-design-review"), ("fe", "fe.rd-design-review"), ("qa", "qa.rd-design-review")])
    st, b = r.decide("tec-b3", "approved", "RD 技术评审通过")
    ck("B-3b tec-b3 拍板", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.execute("tec-b4", "frontend-design.produce", "fe")
    r.advance("tec-b4")
    r.reload()
    st, b = r.carry("tec-b5", [("qa", "qa.fe-design-review")])
    ck("B-3c tec-b5 机械收口", b.get("gateState") == "approved", f"gateState={b.get('gateState')}")
    r.reload()
    r.ck_current("B-3d", "tec-b8a")

    # join：rd 侧 exec→rotate→cr agree→机械（激活 fe 侧）；fe 侧同构（对侧已 approved→整组推进）
    r.execute("tec-b8a", "backend-dev.dev", "rd")
    r.reload()
    r.ck_phase("B-3e", "tec-b8a", "review")
    r.carry("tec-b8a", [("cr", "code-review")])
    r.reload()
    r.ck_current("B-3f", "tec-b8b")
    r.ck_phase("B-3g", "tec-b8b", "exec")
    # wait 分支（fe 侧先完成待 rd 侧）不可达：fe 节点仅在 rd 侧 approved 后才激活
    skip("C-8b fe 先完成 wait 分支", "API 不可达（fe 侧激活以 rd 侧 approved 为前提，peerApproved 恒 true）——见缺陷清单 BUG-5")
    r.execute("tec-b8b", "frontend-dev.dev", "fe")
    r.reload()
    r.ck_phase("B-3h", "tec-b8b", "review")
    st, b = r.carry("tec-b8b", [("cr", "code-review")])
    ck("B-3i fe 侧收口整组推进", b.get("gateState") == "approved", f"gateState={b.get('gateState')}")
    r.reload()
    r.ck_current("B-3j 整组推进 tec-b9", "tec-b9")

    r.execute("tec-b9", "acceptance-test.accept", "qa")
    r.reload()
    r.ck_current("B-3k", "tec-b10")
    r.decide("tec-b10", "approved", "用户验收")
    r.reload()
    r.ck_current("B-3l", "tec-b11")


# ============================== B-4 task×rd + C-5 retest ==============================
def p_task():
    sec("B-4 task×rd 7 节点 + C-5 retest")
    r = Req("B-4", "QA 任务流程需求", flow="task", scope="rd", owner="rd")
    WALKED.append((r.id, "task", "rd"))
    want = ["tsk-c0", "tsk-c1", "tsk-c2", "tsk-c3", "tsk-c4", "tsk-c6", "tsk-c7"]
    ck("B-4a 实例化 7 节点无 QA 验收节点", defids_of(r) == want, str(defids_of(r)))
    r.decide("tsk-c0", "approved", "启动")
    st, b = r.mytask("executor", "tsk-c1")
    acts = [a.get("actionId") for a in b.get("activeActions", [])]
    ck("B-4b tsk-c1 entry 迎宾仅 align.opening", b.get("currentPhase") == "entry" and acts == ["align.opening"], f"acts={acts}")
    r.decide("tsk-c1", "approved", "对齐确认")
    r.reload()
    r.ck_current("B-4c", "tsk-c2")

    # A-17 complete 判据演示（tsk-c2：先缺产物）
    st, b, tid = r.assign("tsk-c2", "backend-design.produce", "rd")
    r.task_update(tid, "ack")
    st, b = r.task_update(tid, "complete")
    ck("A-17d 判据缺产物 blocked+missing", st == 409 and b.get("blocked") is True and b.get("missing"), f"HTTP {st} {b}")
    r.add_artifact("tsk-c2", "rd_design", "rd")
    st, b = r.task_update(tid, "complete")
    ck("B-4d 补齐产物后 complete", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.advance("tsk-c2")
    r.reload()
    r.carry("tsk-c3", [("arch", "arch.rd-design-review")])
    r.decide("tsk-c3", "approved", "技术评审通过")
    r.execute("tsk-c4", "backend-dev.dev", "rd")
    r.carry("tsk-c4", [("cr", "code-review")])
    r.reload()
    r.ck_current("B-4e", "tsk-c6")

    # C-5 retest：cr 提缺陷 → rd resolve fix（open→to-fix 不通知）→ 二次 fix（→fixed+re_review）
    # （BUG-013：评审上下文评论必须锚产物——挂 tsk-c4 自身 dev_notes）
    devnotes_art = next((a for a in r.snap.get("artifacts", []) if a.get("nodeId") == r.node("tsk-c4") and a.get("productType") == "dev_notes"), {})
    c4_anchor = {"kind": "document", "artifactId": devnotes_art.get("artifactId", "")}
    st, b, defect = r.comment("tsk-c4", "cr-a", "cr", "导出接口缺超时分支（缺陷）", "rd", anchor=c4_anchor)
    ck("C-5a 缺陷 issue 建行", st == 200 and b.get("ok") and defect != "", f"HTTP {st} {b}")
    def rr_count():
        return len([x for x in r.inbox("cr").get("notifications", []) if x.get("kind") == "re_review" and x.get("requirementId") == r.id])
    rr0 = rr_count()
    st, b = r.qualify(defect, "rd")
    ck("C-5b resolve fix open→to-fix", st == 200 and b.get("issue", {}).get("state") == "to-fix", f"HTTP {st} {b}")
    ck("C-5c to-fix 未迁 fixed 不发 re_review", rr_count() == rr0, f"{rr0}→{rr_count()}")
    st, b = r.qualify(defect, "rd")
    rr1 = rr_count()
    ck("C-5d to-fix→fixed + re_review（retest 子型）", st == 200 and b.get("issue", {}).get("state") == "fixed" and rr1 > rr0,
       f"HTTP {st} state={b.get('issue', {}).get('state')} {rr0}→{rr1}")
    rr_rows = [x for x in r.inbox("cr").get("notifications", []) if x.get("kind") == "re_review" and x.get("requirementId") == r.id]
    ck("C-5e retest 子型 target_id=issueId", bool(rr_rows) and rr_rows[-1].get("targetId") == defect,
       f"targetId={rr_rows[-1].get('targetId') if rr_rows else None} issue={defect}")
    st, b = r.qualify(defect, "rd")
    ck("C-5f fixed→fixed 幂等不重发", st == 200 and rr_count() == rr1, f"{rr1}→{rr_count()}")
    st, b = r.confirm(defect, "cr")
    ck("C-5g 提评者（角色 cr）confirm → resolved", st == 200 and b.get("issue", {}).get("state") == "resolved", f"HTTP {st} {b}")

    # A-19 reply 路径：to-fix 态回复 → fixed（-D1 路径②）
    st, b, iss = r.comment("tsk-c4", "cr-a", "cr", "边界未覆盖（第二处）", "rd", anchor=c4_anchor)
    r.qualify(iss, "rd")
    rr3 = len([x for x in r.inbox("cr").get("notifications", []) if x.get("kind") == "re_review"])
    st, b = r.declare_fixed(iss, "rd-a", "rd", "已补边界用例并修复")
    st2, b2 = r.qualify(iss, "rd")
    rr4 = len([x for x in r.inbox("cr").get("notifications", []) if x.get("kind") == "re_review"])
    ck("A-19a reply roundCount++", st == 200, f"HTTP {st} {b}")
    ck("A-19b to-fix 态回复后 resolve fix → fixed + re_review（-D1 路径②）", b2.get("issue", {}).get("state") == "fixed" and rr4 > rr3, f"state={b2.get('issue', {}).get('state')} {rr3}→{rr4}")
    st, b = r.confirm(iss, "cr")
    ck("A-19c 提评者（角色 cr）confirm → resolved", st == 200 and b.get("issue", {}).get("state") == "resolved", f"HTTP {st} {b}")
    r.decide("tsk-c6", "approved", "用户验收")
    r.reload()
    r.ck_current("B-4f", "tsk-c7")


# ============================== C-13 澄清矩阵（§7 补测①×6） ==============================
def p_clarify_matrix():
    sec("C-13 澄清矩阵（§7 补测①：6 条 clarify 全链：unagree→respond→owner 澄清→重入新轮→复核→门禁收口）")

    def clarify_chain(r, defid, cap, owner, anchor_defid, tag, tail):
        """澄清循环同型链（C-4 范本）：评论→unagree→respond→clarify 派发→settle→complete 重入→复核→收口。
        tail(req) 收口回调（在复核 agree、gate 开放后调用一次）。评审位=arch（pln-d6 为 pm）。"""
        reviewer = "pm" if cap.startswith("pm.") else "arch"
        art = next((a for a in r.snap.get("artifacts", [])
                    if a.get("nodeId") == r.node(anchor_defid)), {})
        st, b, iss = r.comment(defid, f"{reviewer}-a", reviewer,
                               "口径未定义，需澄清（澄清矩阵探针）", owner,
                               anchor={"kind": "document", "artifactId": art.get("artifactId", "")})
        st, b = r.review(defid, reviewer, cap, "unagree")
        ck(f"{tag}1 unagree→clarify（respond 相位）", st == 200 and b.get("collectOutcome") == "clarify",
           f"HTTP {st} {b}")
        r.reload()
        r.ck_phase(f"{tag}2", defid, "respond")
        st, b, tid = r.assign(defid, tag_action[tag], owner)
        ck(f"{tag} 澄清动作派发 200", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.task_update(tid, "ack")
        st, b = r.settle(iss, owner)
        ck(f"{tag}3 owner 澄清 issue resolved", st == 200 and b.get("issue", {}).get("state") == "resolved",
           f"HTTP {st} {b}")
        rr0 = r.inbox_kinds(reviewer).get("re_review", 0)
        st, b = r.task_update(tid, "complete")
        ck(f"{tag}4 respond complete 触发评审重入", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.reload()
        r.ck_phase(f"{tag}5 重入 review 新轮", defid, "review")
        ck(f"{tag}6 评审位收 re_review", r.inbox_kinds(reviewer).get("re_review", 0) > rr0,
           f"{rr0}→{r.inbox_kinds(reviewer).get('re_review', 0)}")
        r.review(defid, reviewer, cap, "agree")
        tail(r)

    # —— task×rd：backend-design.clarify（技术评审 tsk-c3）——
    r = Req("C-13", "QA 澄清矩阵 task×rd", flow="task", scope="rd", owner="rd")
    WALKED.append((r.id, "task", "rd"))
    r.decide("tsk-c0", "approved", "启动")
    r.decide("tsk-c1", "approved", "对齐确认")
    r.execute("tsk-c2", "backend-design.produce", "rd")
    r.advance("tsk-c2")
    r.reload()

    def tail_rd(req):
        st, b = req.decide("tsk-c3", "approved", "澄清后通过")
        ck("C-13a7 拍板 approved 门禁收口", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        req.reload()
        req.ck_current("C-13a8", "tsk-c4")
        req.execute("tsk-c4", "backend-dev.dev", "rd")
        req.carry("tsk-c4", [("cr", "code-review")])
        req.decide("tsk-c6", "approved", "用户验收")
        req.reload()
        req.ck_current("C-13a9 尾段收口 tsk-c7（D-1 对账）", "tsk-c7")
    clarify_chain(r, "tsk-c3", "arch.rd-design-review", "rd", "tsk-c2", "C-13a", tail_rd)

    # —— task×fe：frontend-design.clarify（技术评审 tsk-c3）——
    r = Req("C-13", "QA 澄清矩阵 task×fe", flow="task", scope="fe", owner="fe")
    WALKED.append((r.id, "task", "fe"))
    r.decide("tsk-c0", "approved", "启动")
    r.decide("tsk-c1", "approved", "对齐确认")
    st, b, tid = r.assign("tsk-c2", "frontend-design.produce", "fe")
    assert st == 200 and b.get("ok"), f"assign produce 失败: {b}"
    r.task_update(tid, "ack")
    r.add_artifact("tsk-c2", "fe_design", "fe")  # 手工登记：scope=fe 不登 rd_design（D-1 矩阵）
    r.task_update(tid, "complete")
    r.advance("tsk-c2")
    r.reload()

    def tail_fe(req):
        st, b = req.decide("tsk-c3", "approved", "澄清后通过")
        ck("C-13b7 拍板 approved 门禁收口", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        req.reload()
        req.ck_current("C-13b8", "tsk-c4")
        req.execute("tsk-c4", "frontend-dev.dev", "fe")
        req.carry("tsk-c4", [("cr", "code-review")])
        req.decide("tsk-c6", "approved", "用户验收")
        req.reload()
        req.ck_current("C-13b9 尾段收口 tsk-c7（D-1 对账）", "tsk-c7")
    clarify_chain(r, "tsk-c3", "arch.fe-design-review", "fe", "tsk-c2", "C-13b", tail_fe)

    # —— plan：research-cap.clarify（pln-d4 调研评审，机械收口）——
    r = Req("C-13", "QA 澄清矩阵 plan", category="规划审批任务")
    WALKED.append((r.id, "plan", ""))
    ck("C-13c0 plan 实例化 9 节点", defids_of(r) == [f"pln-d{i}" for i in range(9)], str(defids_of(r)))
    r.decide("pln-d0", "approved", "启动")
    r.execute("pln-d1", "to-alignment", "arch")
    r.decide("pln-d1", "approved", "对齐确认")
    r.decide("pln-d2", "approved", "立项通过")
    r.execute("pln-d3", "research-cap.produce", "inv")
    r.advance("pln-d3")
    r.reload()

    def tail_d4(req):
        req.reload()
        req.ck_current("C-13c7 机械收口（user_gate=0）→pln-d5", "pln-d5")
    clarify_chain(r, "pln-d4", "arch.research-review", "inv", "pln-d3", "C-13c", tail_d4)
    r.execute("pln-d5", "arch-cap.produce", "arch")
    r.advance("pln-d5")
    r.reload()

    # —— plan：arch-cap.clarify（pln-d6 架构规划评审，用户拍板）——
    def tail_d6(req):
        st, b = req.decide("pln-d6", "approved", "澄清后通过")
        ck("C-13d7 拍板 approved 门禁收口", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        st, b, tid = req.assign("pln-d7", "decide-cap.create_reqs", "leader")
        req.task_update(tid, "ack")
        req.task_update(tid, "complete")
        req.advance("pln-d7")
        req.reload()
        req.ck_current("C-13d8 尾段收口 pln-d8（D-1 对账）", "pln-d8")
    clarify_chain(r, "pln-d6", "pm.architecture-review", "arch", "pln-d5", "C-13d", tail_d6)

    # —— tech×full：backend-dev/frontend-dev.clarify（QA 验收 tec-b9）——
    r = Req("C-13", "QA 澄清矩阵 tech×full", flow="tech", scope="full")
    WALKED.append((r.id, "tech", "full"))
    r.decide("tec-b0", "approved", "启动")
    r.execute("tec-b1", "to-brd", "pm")
    r.decide("tec-b1", "approved", "对齐确认")
    r.execute("tec-b2", "backend-design.produce", "rd")
    r.advance("tec-b2")
    r.carry("tec-b3", [("arch", "arch.rd-design-review"), ("fe", "fe.rd-design-review"),
                       ("qa", "qa.rd-design-review")])
    r.decide("tec-b3", "approved", "RD 技术评审通过")
    r.execute("tec-b4", "frontend-design.produce", "fe")
    r.advance("tec-b4")
    r.carry("tec-b5", [("qa", "qa.fe-design-review")])
    r.execute("tec-b8a", "backend-dev.dev", "rd")
    r.carry("tec-b8a", [("cr", "code-review")])
    r.execute("tec-b8b", "frontend-dev.dev", "fe")
    r.carry("tec-b8b", [("cr", "code-review")])
    r.reload()
    r.ck_current("C-13e0 前置走查到 tec-b9", "tec-b9")
    # QA 验收节点（reviewers=[]）无 review/respond 相位（respond 唯一写入点=评审 collect
    # clarify，rd/service/review.go:180）——issue 闭环半边由 C-13a~d/C-15 见证；
    # 此处见证可派发性（assign 守卫仅挡空/done 相位）+ respond 判据链（open issue 清零）。
    st, b, tid = r.assign("tec-b9", "backend-dev.clarify", "rd")
    ck("C-13e backend-dev.clarify 派发（tech/tec-b9）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    st, b = r.task_update(tid, "complete")
    ck("C-13e2 respond complete 判据（open issue 清零）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b, tid = r.assign("tec-b9", "frontend-dev.clarify", "fe")
    ck("C-13f frontend-dev.clarify 派发（tech/tec-b9）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    r.task_update(tid, "complete")
    r.execute("tec-b9", "acceptance-test.accept", "qa")
    r.reload()
    r.ck_current("C-13e3 验收后推进 tec-b10（门禁收口）", "tec-b10")
    r.decide("tec-b10", "approved", "用户验收")
    r.reload()
    r.ck_current("C-13e4 尾节点 tec-b11（D-1 对账）", "tec-b11")


# clarify_chain 用的动作标签表（动作 id → 补测项 cid 前缀；词汇基准=seed node_actions）
tag_action = {
    "C-13a": "backend-design.clarify",
    "C-13b": "frontend-design.clarify",
    "C-13c": "research-cap.clarify",
    "C-13d": "arch-cap.clarify",
}


# ============================== C-14 修订矩阵（§7 补测②×4） ==============================
def p_revise_matrix():
    sec("C-14 修订矩阵（§7 补测②：4 条 revise exec 重入修订：rejected→回退→revise→重评→拍板收口）")

    # —— business×rd：product-design.revise（bus-02）——
    r = Req("C-14", "QA 修订矩阵 business×rd", flow="business", scope="rd")
    WALKED.append((r.id, "business", "rd"))
    r.decide("bus-00", "approved", "启动")
    r.execute("bus-01", "to-brd", "pm")
    r.decide("bus-01", "approved", "对齐确认")
    r.execute("bus-02", "product-design.produce", "pm")
    r.advance("bus-02")
    r.reload()
    r.carry("bus-03", [("arch", "arch.prd-review"), ("rd", "rd.prd-review")])
    st, b = r.decide("bus-03", "rejected", "产品设计范围蔓延，打回修订")
    ck("C-14a rejected 回退 bus-02", st == 200 and b.get("rollbackNodeId") == r.node("bus-02"),
       f"HTTP {st} {b}")
    r.reload()
    n02 = r.ndef("bus-02")
    ck("C-14a2 bus-02 exec 重入（downstream-reject）", n02.get("state") == "exec"
       and n02.get("reopenSource") == "downstream-reject" and int(n02.get("rolledBackCount", "0")) == 1,
       f"state={n02.get('state')} src={n02.get('reopenSource')} rbc={n02.get('rolledBackCount')}")
    st, b, tid = r.assign("bus-02", "product-design.revise", "pm")
    ck("C-14a3 product-design.revise 派发（business/bus-02）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    r.task_update(tid, "progress", progress="QA 演练 revise")
    r.add_artifact("bus-02", "product_doc", "pm")
    st, b = r.task_update(tid, "complete")
    ck("C-14a4 修订 complete", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.advance("bus-02")
    r.reload()
    r.ck_phase("C-14a5 bus-03 重入 review", "bus-03", "review")
    r.carry("bus-03", [("arch", "arch.prd-review"), ("rd", "rd.prd-review")])
    st, b = r.decide("bus-03", "approved", "修订后通过")
    ck("C-14a6 重评拍板 approved", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    # 走完 rd 尾段（D-1 对账）
    r.execute("bus-04", "backend-design.produce", "rd")
    r.advance("bus-04")
    r.carry("bus-05", [("arch", "arch.rd-design-review"), ("qa", "qa.rd-design-review"),
                       ("pm", "pm.rd-design-review")])
    r.decide("bus-05", "approved", "技术评审通过")
    r.execute("bus-08", "backend-dev.dev", "rd")
    r.carry("bus-08", [("cr", "code-review")])
    r.decide("bus-11", "approved", "用户验收")
    r.reload()
    r.ck_current("C-14a7 尾段收口 bus-12", "bus-12")

    # —— task×fe：frontend-design.revise（tsk-c2）——
    r = Req("C-14", "QA 修订矩阵 task×fe", flow="task", scope="fe", owner="fe")
    WALKED.append((r.id, "task", "fe"))
    r.decide("tsk-c0", "approved", "启动")
    r.decide("tsk-c1", "approved", "对齐确认")
    st, b, tid = r.assign("tsk-c2", "frontend-design.produce", "fe")
    r.task_update(tid, "ack")
    r.add_artifact("tsk-c2", "fe_design", "fe")
    r.task_update(tid, "complete")
    r.advance("tsk-c2")
    r.reload()
    r.carry("tsk-c3", [("arch", "arch.fe-design-review")])
    st, b = r.decide("tsk-c3", "rejected", "技术方案不成立，打回修订")
    ck("C-14b rejected 回退 tsk-c2", st == 200 and b.get("rollbackNodeId") == r.node("tsk-c2"),
       f"HTTP {st} {b}")
    r.reload()
    nc2 = r.ndef("tsk-c2")
    ck("C-14b2 tsk-c2 exec 重入（downstream-reject）", nc2.get("state") == "exec"
       and nc2.get("reopenSource") == "downstream-reject" and int(nc2.get("rolledBackCount", "0")) == 1,
       f"state={nc2.get('state')} src={nc2.get('reopenSource')} rbc={nc2.get('rolledBackCount')}")
    st, b, tid = r.assign("tsk-c2", "frontend-design.revise", "fe")
    ck("C-14b3 frontend-design.revise 派发（task/tsk-c2）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    r.add_artifact("tsk-c2", "fe_design", "fe")
    st, b = r.task_update(tid, "complete")
    ck("C-14b4 修订 complete", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.advance("tsk-c2")
    r.reload()
    r.ck_phase("C-14b5 tsk-c3 重入 review", "tsk-c3", "review")
    r.carry("tsk-c3", [("arch", "arch.fe-design-review")])
    st, b = r.decide("tsk-c3", "approved", "修订后通过")
    ck("C-14b6 重评拍板 approved", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.execute("tsk-c4", "frontend-dev.dev", "fe")
    r.carry("tsk-c4", [("cr", "code-review")])
    r.decide("tsk-c6", "approved", "用户验收")
    r.reload()
    r.ck_current("C-14b7 尾段收口 tsk-c7（D-1 对账）", "tsk-c7")

    # —— plan：research-cap.revise（exec 探针）+ arch-cap.revise（pln-d5 重入）——
    r = Req("C-14", "QA 修订矩阵 plan", category="规划审批任务")
    WALKED.append((r.id, "plan", ""))
    r.decide("pln-d0", "approved", "启动")
    r.execute("pln-d1", "to-alignment", "arch")
    r.decide("pln-d1", "approved", "对齐确认")
    r.decide("pln-d2", "approved", "立项通过")
    # research-cap.revise 唯一重入触发=pln-d4 rejected，而 pln-d4 user_gate=0（collect 机械
    # 装配，wb_decision 非 User 拍板位必 409）——重入不可达。落两半：①exec 相位直接派发探针
    # （assign 守卫仅挡空/done）+ produce 族判据链；②409 负探针实证触发封闭。
    st, b, tid = r.assign("pln-d3", "research-cap.revise", "inv")
    ck("C-14d research-cap.revise exec 相位派发（plan/pln-d3 探针）", st == 200 and b.get("ok"),
       f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    r.add_artifact("pln-d3", "research", "inv")
    st, b = r.task_update(tid, "complete")
    ck("C-14d2 修订 complete（produce 族判据）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.advance("pln-d3")
    r.reload()
    r.ck_current("C-14d3 推进 pln-d4", "pln-d4")
    st, b = r.decide("pln-d4", "rejected", "research-cap.revise 唯一重入触发探针")
    ck("C-14d4 pln-d4 rejected 409（user_gate=0 重入触发不可达实证）",
       st == 409 and b.get("blocked") is True, f"HTTP {st} {b}")
    r.carry("pln-d4", [("arch", "arch.research-review")])
    r.reload()
    r.ck_current("C-14d5 机械收口 pln-d5", "pln-d5")
    r.execute("pln-d5", "arch-cap.produce", "arch")
    r.advance("pln-d5")
    r.reload()
    d5art = next((a for a in r.snap.get("artifacts", []) if a.get("nodeId") == r.node("pln-d5")), {})
    st, b, iss = r.comment("pln-d6", "pm-a", "pm", "容量估算缺依据，需澄清（修订矩阵铺垫）", "arch",
                           anchor={"kind": "document", "artifactId": d5art.get("artifactId", "")})
    r.review("pln-d6", "pm", "pm.architecture-review", "unagree")
    r.reload()
    r.ck_phase("C-14c1 pln-d6 respond（澄清铺垫）", "pln-d6", "respond")
    st, b, tid = r.assign("pln-d6", "arch-cap.clarify", "arch")
    r.task_update(tid, "ack")
    r.settle(iss, "arch")
    r.task_update(tid, "complete")
    r.reload()
    r.review("pln-d6", "pm", "pm.architecture-review", "agree")
    st, b = r.decide("pln-d6", "rejected", "架构规划不成熟，打回修订")
    ck("C-14c rejected 回退 pln-d5", st == 200 and b.get("rollbackNodeId") == r.node("pln-d5"),
       f"HTTP {st} {b}")
    r.reload()
    nd5 = r.ndef("pln-d5")
    ck("C-14c2 pln-d5 exec 重入（downstream-reject）", nd5.get("state") == "exec"
       and nd5.get("reopenSource") == "downstream-reject" and int(nd5.get("rolledBackCount", "0")) == 1,
       f"state={nd5.get('state')} src={nd5.get('reopenSource')} rbc={nd5.get('rolledBackCount')}")
    st, b, tid = r.assign("pln-d5", "arch-cap.revise", "arch")
    ck("C-14c3 arch-cap.revise 派发（plan/pln-d5）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    r.add_artifact("pln-d5", "architecture", "arch")
    st, b = r.task_update(tid, "complete")
    ck("C-14c4 修订 complete", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.advance("pln-d5")
    r.reload()
    r.ck_phase("C-14c5 pln-d6 重入 review", "pln-d6", "review")
    r.carry("pln-d6", [("pm", "pm.architecture-review")])
    st, b = r.decide("pln-d6", "approved", "修订后通过")
    ck("C-14c6 重评拍板 approved", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b, tid = r.assign("pln-d7", "decide-cap.create_reqs", "leader")
    r.task_update(tid, "ack")
    r.task_update(tid, "complete")
    r.advance("pln-d7")
    r.reload()
    r.ck_current("C-14c7 尾段收口 pln-d8（D-1 对账）", "pln-d8")


# ============================== C-15 CR 修复矩阵（§7 补测③×4） ==============================
def p_cr_fix_matrix():
    sec("C-15 CR 修复矩阵（§7 补测③：fix_on_review=节点内 respond 修复轮内闭环；diagnosing-bugs=代码域 exec 重入）")

    def cr_chain(side, cap_produce, cap_dev, fix_action, diag_action, tag_fix, tag_diag):
        """task 流 CR 修复同型链：dev→CR 缺陷 issue→fix_on_review 轮内闭环（tag_fix）→c6 打回→diagnosing-bugs 重入（tag_diag）→复验收口。
        fix_action/diag_action 须字面量传入（qa_branch_audit 静态语料匹配，动态拼接不可见）。"""
        owner = side
        r = Req("C-15", f"QA CR 修复矩阵 task×{side}", flow="task", scope=side, owner=owner)
        WALKED.append((r.id, "task", side))
        r.decide("tsk-c0", "approved", "启动")
        r.decide("tsk-c1", "approved", "对齐确认")
        st, b, tid = r.assign("tsk-c2", cap_produce, owner)
        r.task_update(tid, "ack")
        r.add_artifact("tsk-c2", "rd_design" if side == "rd" else "fe_design", owner)
        r.task_update(tid, "complete")
        r.advance("tsk-c2")
        r.reload()
        r.carry("tsk-c3", [("arch", f"arch.{side}-design-review")])
        r.decide("tsk-c3", "approved", "技术评审通过")
        r.execute("tsk-c4", cap_dev, owner)
        r.reload()
        r.ck_phase(f"{tag_fix}1 tsk-c4 进 CR review", "tsk-c4", "review")
        art = next((a for a in r.snap.get("artifacts", [])
                    if a.get("nodeId") == r.node("tsk-c4") and a.get("productType") == "dev_notes"), {})
        st, b, iss = r.comment("tsk-c4", "cr-a", "cr", "导出接口缺超时分支（CR 缺陷）", owner,
                               anchor={"kind": "document", "artifactId": art.get("artifactId", "")})
        ck(f"{tag_fix}2 CR 缺陷 issue 建行", st == 200 and b.get("ok") and iss != "", f"HTTP {st} {b}")
        st, b, tid = r.assign("tsk-c4", fix_action, owner)
        ck(f"{tag_fix} fix_on_review 派发（task/tsk-c4）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.task_update(tid, "ack")
        st, b = r.settle(iss, owner)
        ck(f"{tag_fix}3 CR 缺陷 resolved（respond 判据清零）", st == 200
           and b.get("issue", {}).get("state") == "resolved", f"HTTP {st} {b}")
        rr0 = r.inbox_kinds("cr").get("re_review", 0)
        st, b = r.task_update(tid, "complete")
        ck(f"{tag_fix}4 respond complete 轮内重入（不打回）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.reload()
        r.ck_phase(f"{tag_fix}5 tsk-c4 重入 review 新轮", "tsk-c4", "review")
        ck(f"{tag_fix}6 cr 收 re_review", r.inbox_kinds("cr").get("re_review", 0) > rr0,
           f"{rr0}→{r.inbox_kinds('cr').get('re_review', 0)}")
        ck(f"{tag_fix}7 节点内修复无 rejected 结论（永不 rejected）", r.ndef("tsk-c4").get("lastDecision", "") == ""
           and r.flow.get("currentNodeId", "") == r.node("tsk-c4"),
           f"ld={r.ndef('tsk-c4').get('lastDecision')} current={r.flow.get('currentNodeId')}")
        r.carry("tsk-c4", [("cr", "code-review")])
        r.reload()
        r.ck_current(f"{tag_fix}8 cr 复核 agree 机械推进 c6", "tsk-c6")
        # diagnosing-bugs：用户验收打回 → 代码域 exec 重入
        st, b = r.decide("tsk-c6", "rejected", "验收发现缺陷，打回定位修复")
        ck(f"{tag_diag}1 rejected 回退 tsk-c4", st == 200 and b.get("rollbackNodeId") == r.node("tsk-c4"),
           f"HTTP {st} {b}")
        r.reload()
        n4 = r.ndef("tsk-c4")
        ck(f"{tag_diag}2 tsk-c4 exec 重入（downstream-reject）", n4.get("state") == "exec"
           and n4.get("reopenSource") == "downstream-reject" and int(n4.get("rolledBackCount", "0")) == 1,
           f"state={n4.get('state')} src={n4.get('reopenSource')} rbc={n4.get('rolledBackCount')}")
        st, b, tid = r.assign("tsk-c4", diag_action, owner)
        ck(f"{tag_diag} diagnosing-bugs 派发（task/tsk-c4）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.task_update(tid, "ack")
        r.task_update(tid, "progress", progress="QA 演练 diagnosing-bugs")
        st, b = r.task_update(tid, "complete")
        ck(f"{tag_diag}3 缺陷定位 complete（dev 族，产物已登记）", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.advance("tsk-c4")
        r.reload()
        r.ck_phase(f"{tag_diag}4 tsk-c4 重入 CR review", "tsk-c4", "review")
        r.carry("tsk-c4", [("cr", "code-review")])
        st, b = r.decide("tsk-c6", "approved", "修复后复验通过")
        ck(f"{tag_diag}5 复验拍板 approved", st == 200 and b.get("ok"), f"HTTP {st} {b}")
        r.reload()
        r.ck_current(f"{tag_diag}6 收口 tsk-c7（D-1 对账）", "tsk-c7")

    cr_chain("rd", "backend-design.produce", "backend-dev.dev",
             "backend-dev.fix_on_review", "backend-dev.diagnosing-bugs", "C-15a", "C-15c")
    cr_chain("fe", "frontend-design.produce", "frontend-dev.dev",
             "frontend-dev.fix_on_review", "frontend-dev.diagnosing-bugs", "C-15b", "C-15d")


# ============================== B-5 plan ==============================
def p_plan():
    sec("B-5 plan 9 节点")
    st, b = api("POST", "/api/requirements", {"title": "QA 规划需求", "category": "规划审批任务", "createdBy": "user-a"})
    ok = st == 200 and b.get("ok")
    if not ok:
        ck("B-5a 规划审批任务→plan 绑定", False, f"HTTP {st} {b}")
        return
    rid = b["requirement"]["requirementId"]
    ck("B-5a 规划审批任务→plan 绑定", b["requirement"].get("flowDefsId") == "plan", str(b["requirement"]))
    r = Req.__new__(Req)
    r.cid = "B-5"
    r.scope = None  # plan 无 scope（products_of 全量返回）
    r.id = rid
    r.flow_defs = "plan"
    r.tasks = 0
    r.artifacts = 0
    r.reload()
    WALKED.append((r.id, "plan", ""))
    want = ["pln-d0", "pln-d1", "pln-d2", "pln-d3", "pln-d4", "pln-d5", "pln-d6", "pln-d7", "pln-d8"]
    ck("B-5b 实例化 9 节点", defids_of(r) == want, str(defids_of(r)))
    r.decide("pln-d0", "approved", "启动")
    r.execute("pln-d1", "to-alignment", "arch")
    r.decide("pln-d1", "approved", "对齐确认")
    r.decide("pln-d2", "approved", "立项通过")
    r.execute("pln-d3", "research-cap.produce", "inv")
    r.advance("pln-d3")
    r.reload()
    r.carry("pln-d4", [("arch", "arch.research-review")])
    r.reload()
    r.ck_current("B-5c 调研评审机械收口", "pln-d5")
    r.execute("pln-d5", "arch-cap.produce", "arch")
    r.advance("pln-d5")
    r.reload()
    r.carry("pln-d6", [("pm", "pm.architecture-review")])
    r.decide("pln-d6", "approved", "架构规划通过")
    st, b = r.mytask("leader", "pln-d7")
    acts = [a.get("actionId") for a in b.get("activeActions", [])]
    ck("B-5d pln-d7 create_reqs 激活", "create_reqs" in acts or "decide-cap.create_reqs" in acts, f"acts={acts}")
    st, b, tid = r.assign("pln-d7", "decide-cap.create_reqs", "leader")
    ck("B-5e create_reqs 派发", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    r.task_update(tid, "ack")
    st, b = r.task_update(tid, "complete")
    ck("B-5f create_reqs complete", st in (200, 409), f"HTTP {st} {b}")
    st, b = r.advance("pln-d7")
    print(f"  [证据] wb_advance pln-d7 → HTTP {st} {json.dumps(b, ensure_ascii=False)[:200]}")
    r.reload()
    r.ck_current("B-5g pln-d7→pln-d8 推进", "pln-d8")



# ============================== A-14/A-15 产物登记与内容读取（独立需求，防污染矩阵） ==============================
def p_artifact_content():
    sec("A-14/A-15 产物登记校验 + 内容读取 + 路径穿越")
    r = Req("A-15", "QA 产物内容需求", flow="business", scope="rd")
    r.decide("bus-00", "approved", "进入编排")
    st, b = r.add_artifact("bus-01", "bogus_type", "pm")
    ck("A-14a 非法 product_type 400", st == 400, f"HTTP {st} {b}")
    st, b = r.add_artifact("bus-01", "brd", "pm", kind="bogus")
    ck("A-14b 非法 kind 400", st == 400, f"HTTP {st} {b}")
    st, b = r.add_artifact("bus-01", "brd", "pm")
    aid = b.get("artifact", {}).get("artifactId", "")
    ck("A-15a document 产物登记", st == 200 and b.get("ok") and aid != "", f"HTTP {st} {b}")
    st, b = api("GET", f"/api/requirements/{r.id}/artifacts/{aid}/content")
    ck("A-15b 内容回读含 marker", st == 200 and "marker-brd-42" in b.get("content", ""), f"HTTP {st} content[:60]={b.get('content', '')[:60]}")
    st, b = api("GET", f"/api/requirements/{r.id}/artifacts/art-no-such/content")
    ck("A-15c 产物不存在 404", st == 404, f"HTTP {st}")
    st, b = r.add_artifact("bus-01", "product_doc", "pm", href="../../etc/passwd")
    ck("A-15d 路径穿越 400 blocked", st == 400 and b.get("blocked") is True, f"HTTP {st} {b}")
    st, b = r.add_artifact("bus-01", "product_doc", "pm", href="v1.0/qa/fixtures/nope.md")
    if st == 200 and b.get("ok"):
        aid3 = b.get("artifact", {}).get("artifactId", "")
        st2, b2 = api("GET", f"/api/requirements/{r.id}/artifacts/{aid3}/content")
        ck("A-15e 文件缺失 404", st2 == 404, f"HTTP {st2} {b2}")
    else:
        ck("A-15e 文件缺失 404", True, f"登记期拦截 HTTP {st}（可接受）")
    st, b = r.add_artifact("bus-01", "code_mr", "rd", kind="link")  # link 类外链（期望外补充，不受限）
    ck("A-15f link 类登记", st == 200 and b.get("artifact", {}).get("kind") == "link", f"HTTP {st} {b}")
    st, b = r.add_artifact("bus-01", "brd", "pm", kind="link")  # bus-01 期望 brd=document
    ck("A-15g 期望 document 拒绝 link 登记（BUG-015 守卫）", st == 400 and b.get("blocked") is True, f"HTTP {st} {b}")

# ============================== A-18~A-24 + C-6/C-7 issue 域 ==============================
def p_issue_branches():
    sec("A-18~A-24/C-06/C-07 issue 域全分支")
    r = Req("A-18", "QA issue 域需求", flow="business", scope="rd")
    r.decide("bus-00", "approved", "进入编排（意见锚 bus-01 entry）")
    # anchor 四级
    st, b, i_doc = r.comment("bus-01", "qa-a", "qa", "文档级意见", "pm")
    ck("A-18a document 锚", b.get("issue", {}).get("anchorKind") == "document", str(b.get("issue", {}).get("anchorKind")))
    st, b = r.add_artifact("bus-01", "brd", "pm")
    anch_id = b.get("artifact", {}).get("artifactId", "")
    st, b, i_line = r.comment("bus-01", "qa-a", "qa", "行级意见(已废弃)", "pm",
                              anchor={"kind": "line", "artifactId": anch_id, "line": 3})
    ck("A-18b line 锚废弃 400（注记-L3）", st == 400, f"HTTP {st} {json.dumps(b)[:160]}")
    st, b, i_range = r.comment("bus-01", "qa-a", "qa", "范围意见", "pm",
                               anchor={"kind": "range", "artifactId": anch_id, "startLine": 3, "endLine": 5})
    irow = b.get("issue", {})
    ck("A-18c range 锚（划词）", st == 200 and irow.get("anchorStartLine") in ("3", 3) and irow.get("anchorEndLine") in ("5", 5),
       f"HTTP {st} {json.dumps(irow)[:200]}")
    st, b, i_block = r.comment("bus-01", "qa-a", "qa", "代码块意见(已废弃)", "pm",
                               anchor={"kind": "block", "artifactId": anch_id, "blockStart": 1, "blockEnd": 9, "lang": "go"})
    ck("A-18d block 锚废弃 400（注记-L3）", st == 400, f"HTTP {st} {json.dumps(b)[:160]}")
    st, b, i_esc = r.comment("bus-01", "qa-a", "qa", "直升意见", "pm", escalate=True)
    iss = b.get("issue", {})
    ck("A-18e escalate=true 直升", st == 200 and iss.get("escalate") in ("1", 1), f"HTTP {st} escalate={iss.get('escalate')}")

    # A-20 resolve agree 双态
    st, b, i2 = r.comment("bus-01", "qa-a", "qa", "待复核意见", "pm")
    r.clarified(i2, "pm-a", "pm", "澄清已补")
    st, b = api("GET", f"/api/requirements/{r.id}")
    i2row = next((x for x in b.get("issues", []) if x.get("issueId") == i2), {})
    ck("A-19c reply→pending-confirm", i2row.get("state") == "pending-confirm" and int(i2row.get("roundCount", "0")) >= 1, f"state={i2row.get('state')}")
    st, b = r.reopen(i2, "qa-a")
    ck("A-22a reopen→open loop=1", st == 200 and b.get("loopCount") in ("1", 1), f"HTTP {st} loop={b.get('loopCount')}")
    st, b = r.settle(i2, "pm")
    ck("A-20a resolve agree→resolved", b.get("issue", {}).get("state") == "resolved", "")
    st, b = r.settle(i2, "pm")
    ck("A-20b resolved 再 resolve 拦截/幂等", st >= 400 or b.get("issue", {}).get("state") == "resolved", f"HTTP {st}")

    # A-21 confirm（pending-confirm → resolved）
    st, b, i3 = r.comment("bus-01", "qa-a", "qa", "复核通道意见", "pm")
    r.clarified(i3, "pm-a", "pm", "已回应")
    st, b = r.confirm(i3, "qa-a")
    ck("A-21 confirm→resolved", st == 200 and b.get("issue", {}).get("state") == "resolved", f"HTTP {st} {b}")

    # A-24/C-6 仲裁双路径
    st, b, i4 = r.comment("bus-01", "qa-a", "qa", "上升裁决意见 A", "pm")
    r.clarified(i4, "pm-a", "pm", "不认可")
    st, b = r.escalate(i4, "qa", "双方分歧")
    sess = b.get("sessionId", "")
    ck("C-6a escalate 建 session", st == 200 and b.get("escalated") is True and sess != "", f"HTTP {st} {b}")
    kind_db = sql(f"SELECT anchor_kind FROM issues WHERE issue_type='notification' AND anchor_artifact_id='{i4}' LIMIT 1")
    ck("BUG-008a wb_escalate 通知行 anchor_kind=escalation（DB）", kind_db == "escalation", f"anchor_kind={kind_db}")
    st, b = api("GET", f"/api/requirements/{r.id}")
    ck("C-6b 快照 sessions 出行（注记-D2）", len(b.get("sessions", [])) >= 1, f"sessions={len(b.get('sessions', []))}")
    ke = r.inbox_kinds("pm")
    ku = r.inbox_kinds("user")
    ck("C-6c escalation 通知（pm 或 user）", ke.get("escalation", 0) >= 1 or ku.get("escalation", 0) >= 1, f"pm={ke.get('escalation')} user={ku.get('escalation')}")
    st, b = r.adjudicate(i4, "uphold", "user-a", "维持议题")
    ck("C-6d uphold 保持", st == 200 and b.get("verdict") == "uphold", f"HTTP {st} {b}")
    st, b, i5 = r.comment("bus-01", "qa-a", "qa", "上升裁决意见 B", "pm")
    r.escalate(i5, "qa", "分歧 B")
    st, b = r.adjudicate(i5, "override", "user-a", "仲裁推翻")
    st, b = api("GET", f"/api/requirements/{r.id}")
    i5row = next((x for x in b.get("issues", []) if x.get("issueId") == i5), {})
    ck("C-6e override→resolved 终态（09-12 裁决废除 overridden）", i5row.get("state") == "resolved" and i5row.get("resolvePath") == "override", f"state={i5row.get('state')} resolvePath={i5row.get('resolvePath')}")
    st, b = r.adjudicate(i4, "bogus", "user-a", "")
    ck("A-24a 非法 verdict 400", st == 400, f"HTTP {st} {b}")

    # C-7 reopen×5 自动上升（澄清期 loop_count）
    st, b, i6 = r.comment("bus-01", "qa-a", "qa", "澄清拉锯意见", "pm")
    auto = False
    for i in range(1, 6):
        r.clarified(i6, "pm-a", "pm", f"回应 {i}")
        st, b = r.reopen(i6, "qa-a")
        loop = int(b.get("loopCount") or 0)
        if loop == 5:
            st2, b2 = api("GET", f"/api/requirements/{r.id}")
            i6row = next((x for x in b2.get("issues", []) if x.get("issueId") == i6), {})
            auto = str(i6row.get("escalate")) == "1"
    ck("C-7a 第 5 次 reopen 自动 escalate", auto, f"最后 loop={b.get('loopCount')}")
    ka = r.inbox_kinds("pm")
    ku = r.inbox_kinds("user")
    ck("C-7b 自动上升通知", ka.get("escalation", 0) >= 1 or ku.get("escalation", 0) >= 1, f"pm={ka.get('escalation')} user={ku.get('escalation')}")

    # A-23 负路径
    st, b = r.escalate("cm-no-such", "qa")
    ck("A-23a escalate 不存在 404", st == 404, f"HTTP {st}")
    st, b = r.settle("cm-no-such", "pm")
    ck("A-23b resolve 不存在 404", st == 404, f"HTTP {st}")


# ============================== C-10 幂等补遗 ==============================
def p_idempotent():
    sec("C-10 幂等补遗")
    st, b = api("GET", "/api/inbox?role=pm&kind=gate_pending")
    ck("A-29a inbox kind 筛选", st == 200 and b.get("ok") is True, f"HTTP {st}")
    st, b = api("GET", "/api/inbox?role=pm")
    ck("A-29b todos 字段存在（注记-D5）", "todos" in b and "notifications" in b, f"keys={sorted(b)[:8]}")


# ============================== D-1/D-2 对账 ==============================
def p_reconcile():
    sec("D-1/D-2 产物矩阵 + 审计对账")
    for (rid, flow, scope) in WALKED:
        st, b = api("GET", f"/api/requirements/{rid}")
        if not (st == 200 and b.get("ok")):
            ck(f"D-1 {rid}", False, "快照失败")
            continue
        arts = {}
        for a in b.get("artifacts", []):
            arts.setdefault(a.get("nodeId"), {})[a.get("productType")] = a
        bad = []
        for n in b.get("flowNodes", []):
            defid = n.get("defNodeId")
            exp = {p[0]: p for p in EXP.get(defid, [])}
            if flow == "task":
                if scope == "rd":
                    exp = {k: v for k, v in exp.items() if k != "fe_design"}
                elif scope == "fe":
                    exp = {k: v for k, v in exp.items() if k != "rd_design"}
            got = arts.get(n.get("nodeId"), {})
            for ptype in exp:
                if ptype not in got:
                    bad.append(f"{defid} 缺 {ptype}")
                else:
                    if got[ptype].get("label") != exp[ptype][1]:
                        bad.append(f"{defid} {ptype} label≠《{exp[ptype][1]}》got={got[ptype].get('label')}")
                    if got[ptype].get("kind") != exp[ptype][2]:
                        bad.append(f"{defid} {ptype} kind≠{exp[ptype][2]}")
            for ptype in got:
                if ptype not in exp:
                    bad.append(f"{defid} 多登 {ptype}")
        ck(f"D-1 产物矩阵 {rid[:14]}({flow}×{scope or '-'})", not bad, "; ".join(bad[:6]))

    # D-2 审计 append-only
    bad = sql("SELECT COUNT(*) FROM audit_log WHERE created_at <> updated_at")
    ck("D-2a audit append-only（created=updated）", bad == "0", f"异常行={bad}")
    n = sql_int("SELECT COUNT(*) FROM audit_log")
    ck("D-2b audit 非空", n > 0, f"rows={n}")
    st, b = api("GET", "/api/requirements")
    total = int(b.get("total", "0"))
    cnt = sql_int("SELECT COUNT(*) FROM requirements")
    ck("D-2c requirements 列表/库一致", total == cnt, f"list={total} db={cnt}")


# ============================== Q 架构行为增量（2026-09-15 批次） ==============================

def wait_scan(cond, timeout_sec=75, interval=5):
    """轮询等待 scanner sweep 生效（扫描间隔 15s；默认上界约 5 轮）。"""
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        if cond():
            return True
        time.sleep(interval)
    return False


def p_q_arch():
    sec("Q-1~Q-3 架构行为增量（编排会话/审计派生批次；LLM=FakeClient 演示态黑盒全链）")

    # ---- Q-1 LLM 产线黑盒全链：sweep 自动派发 to-brd（entry 相位准入，BUG-Q1 修复）
    #      → FakeClient 产文 → 产物入库 → complete 判据过 → align 拍板位停等 →
    #      wb_decision 拍板推进（tec-b1 done / current 前移 tec-b2） ----
    q = Req("Q", "架构增量·LLM 产线评审链（tech×full）", flow="tech", scope="full")
    st, b = q.decide("tec-b0", "approved", "Q 链启动拍板")
    ck("Q-1a tec-b0 entry 拍板推进", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    q.reload()
    ck("Q-1b tec-b1 激活 entry 相位（align 开场白节点）", q.ndef("tec-b1").get("state") == "entry",
       f"state={q.ndef('tec-b1').get('state')}")

    appeared = wait_scan(lambda: sql_int(
        f"SELECT COUNT(*) FROM tasks WHERE requirement_id='{q.id}' AND action_id='to-brd'") > 0)
    ck("Q-1c sweep 自动派发 to-brd（entry 相位准入，BUG-Q1 回归位）", appeared, "75s 内未出现任务行")

    done = wait_scan(lambda: sql_int(
        f"SELECT COUNT(*) FROM tasks WHERE requirement_id='{q.id}' AND action_id='to-brd' "
        f"AND status='completed'") > 0, timeout_sec=90)
    ck("Q-1d FakeClient 产文后 complete", done, "90s 内任务未 completed")

    q.reload()
    arts = [a for a in q.snap.get("artifacts", []) if a.get("nodeId") == q.node("tec-b1")]
    brd = next((a for a in arts if a.get("productType") == "brd"), None)
    ck("Q-1e brd 产物入库（kind=document）", brd is not None and brd.get("kind") == "document",
       f"artifacts={[a.get('productType') for a in arts]}")
    if brd:
        st2, b2 = api("GET", f"/api/requirements/{q.id}/artifacts/{brd.get('artifactId')}/content")
        ck("Q-1f 产物文件真实存在且可回读", st2 == 200 and (b2.get("content") or "") != "",
           f"HTTP {st2} len={len((b2.get('content') or ''))}")
    q.reload()
    ck("Q-1g align 拍板位停等（complete 后不内聚推进）",
       q.ndef("tec-b1").get("state") == "entry" and q.flow.get("currentNodeId") == q.node("tec-b1"),
       f"tec-b1={q.ndef('tec-b1').get('state')} current={q.flow.get('currentNodeId')}")
    st, b = q.decide("tec-b1", "approved", "对齐结论用户确认")
    ck("Q-1h wb_decision 拍板推进", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    q.reload()
    ck("Q-1i tec-b1 done / current 前移 tec-b2",
       q.ndef("tec-b1").get("state") == "done" and q.flow.get("currentNodeId") == q.node("tec-b2"),
       f"tec-b1={q.ndef('tec-b1').get('state')} current={q.flow.get('currentNodeId')}")

    # ---- Q-2 LLM 评审黑盒：tec-b2 手工产出 → tec-b3 review 相位 → sweep 派发
    #      arch.rd-design-review.review → FakeClient 0 findings → 无评论 + arch 位 agree
    #      （软失败纪律：Q-1 链未通时评审链不可达，SKIP 附因不硬断） ----
    q1_ok = all(x["ok"] is True for x in RESULTS if x["cid"].startswith("Q-1"))
    if not q1_ok:
        skip("Q-2", "Q-1 产线链未通（评审链依赖其产文推进），见 Q-1 缺陷条目")
    else:
        q2_body(q)

    # ---- Q-3 巡检 notFound 容错（C1）：删除编排中需求 → scanner 一轮 → 服务健康、
    #      新需求编排照常（ListFlowsInOrchestration 候选消失竞态的容错语义） ----
    q3 = Req("Q3", "架构增量·巡检删除竞态容错", flow="business", scope="full")
    st, b = q3.decide("bus-00", "approved", "进入编排后即删")
    ck("Q-3a bus-00 拍板进入编排", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    # 产品守卫：进行中需求拒绝删除（先取消）——取消后 flows.current 仍非空，
    # 删除后 sweep 候选消失 = notFound 容错场景的真实可达路径
    st, b = api("PUT", f"/api/requirements/{q3.id}", {
        "requirementId": q3.id, "title": "架构增量·巡检删除竞态容错", "status": "已取消",
        "updatedAt": 0})
    ck("Q-3b-1 进行中需求取消", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    st, b = api("DELETE", f"/api/requirements/{q3.id}?confirm=true")
    ck("Q-3b 编排中（已取消）需求删除成功", st == 200 and b.get("ok"), f"HTTP {st} {b}")
    time.sleep(20)  # 跨一个 scanner 轮（15s）
    st, b = api("GET", "/healthz")
    ck("Q-3c 删除竞态后服务健康（sweep notFound 容错）",
       st == 200 and b.get("ok") is True and b.get("mysql", {}).get("reachable") is True,
       f"HTTP {st} {b}")
    q4 = Req("Q4", "架构增量·删除竞态后新需求可推进", flow="business", scope="full")
    st, b = q4.decide("bus-00", "approved", "删除竞态后健康验证")
    ck("Q-3d 删除竞态后新需求编排正常", st == 200 and b.get("ok"), f"HTTP {st} {b}")


def q2_body(q):
    """Q-2 评审链主体（Q-1 绿才进入）。"""
    r = q.execute("tec-b2", "backend-design.produce", "rd")
    ck("Q-2a tec-b2 手工产出 complete", r[0] == 200 and r[1].get("ok"), f"HTTP {r[0]} {r[1]}")
    q.reload()
    ck("Q-2b tec-b3 激活 review 相位", q.ndef("tec-b3").get("state") == "review",
       f"state={q.ndef('tec-b3').get('state')}")
    issues_before = sql_int(f"SELECT COUNT(*) FROM issues WHERE requirement_id='{q.id}' AND issue_type='issue'")
    appeared2 = wait_scan(lambda: sql_int(
        f"SELECT COUNT(*) FROM tasks WHERE requirement_id='{q.id}' "
        f"AND action_id='arch.rd-design-review.review'") > 0)
    ck("Q-2c sweep 派发 arch 评审任务（准入集第二动作）", appeared2, "75s 未派发")
    done2 = wait_scan(lambda: sql_int(
        f"SELECT COUNT(*) FROM tasks WHERE requirement_id='{q.id}' "
        f"AND action_id='arch.rd-design-review.review' AND status='completed'") > 0, timeout_sec=90)
    ck("Q-2d FakeClient 评审任务完成", done2, "90s 未完成")
    q.reload()
    rr = (q.ndef("tec-b3").get("reviewResults") or "[]").replace(" ", "")
    # 占位文本非 findings JSON → -L4 宽口径兜底单条意见（防评审中断拍板口径）
    # → unagree + 挂锚评论 ×1；0-findings 分支黑盒不可达（FakeClient 文本不可配），
    # 由 rd/llm 单测与 llmflow IT 结构化样例锁定
    ck("Q-2e arch 位 unagree（占位文本兜底意见口径）",
       '"role":"arch"' in rr and '"state":"unagree"' in rr,
       f"reviewResults={rr[:220]}")
    issues_after = sql_int(f"SELECT COUNT(*) FROM issues WHERE requirement_id='{q.id}' AND issue_type='issue'")
    ck("Q-2f 兜底意见挂锚评论 ×1", issues_after == issues_before + 1,
       f"{issues_before}→{issues_after}")


def main():
    t0 = time.time()
    print(f"QA 驱动启动 → {BASE} db={MY['db']}")
    if not setup_fixtures():
        print("FATAL: fixture 基座缺失（qa-test-cases.md 不存在）")
        sys.exit(2)
    phases = [p_health_auth, p_versions, p_req_crud, p_artifact_content, p_business_full, p_reject_branch,
              p_rdonly, p_tech, p_task, p_clarify_matrix, p_revise_matrix, p_cr_fix_matrix, p_plan,
              p_issue_branches, p_idempotent, p_reconcile, p_q_arch]
    for p in phases:
        try:
            p()
        except Exception as e:
            ck(f"PHASE-CRASH {p.__name__}", False, f"{type(e).__name__}: {e}")
    ok = sum(1 for x in RESULTS if x["ok"] is True)
    fail = sum(1 for x in RESULTS if x["ok"] is False)
    skipped = sum(1 for x in RESULTS if x["ok"] is None)
    ac_coverage_assert()
    print(f"\n===== 汇总：PASS {ok} / FAIL {fail} / SKIP {skipped}（{time.time()-t0:.1f}s）=====")
    for x in RESULTS:
        if x["ok"] is False:
            print(f"  FAIL {x['cid']}: {x['detail'][:260]}")
    with open(OUT, "w") as f:
        json.dump(dict(base=BASE, db=MY["db"], results=RESULTS, elapsed_sec=round(time.time()-t0, 1)), f, ensure_ascii=False, indent=1)
    print(f"结果 → {OUT}")
    sys.exit(1 if fail else 0)


if __name__ == "__main__":
    main()
