#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""D-4 文档-实现一致性巡检（BUG-002 衍生，to-qa-testcases 骨架 D-4 固化）。

对活文档扫描「已被裁决废止的旧口径」禁用短语——文档与裁决正本（spec 注记）漂移
即 FAIL。范围 docs/v1.0/**，排除 _archive/（历史归档）与 qa/（缺陷台账引用原文
属合法历史记录）。规则随裁决增补：每条新拍板若废止某文档口径，把旧措辞加进
FORBIDDEN 并在 FIXTURE 注明正本出处。

用法：python3 qa_doc_audit.py    # 仓库根执行或任意 cwd；退出码违规非 0
"""
import os
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCAN_ROOT = os.path.join(REPO, "docs", "v1.0")
EXCLUDE_DIRS = {"_archive", "qa", "node_modules"}

# (禁用短语, 裁决正本出处)
FORBIDDEN = [
    ("废弃联动其下需求", "spec §1.2 注记-A（条件解绑，2026-09-09）——BUG-002"),
    ("联动需求已取消", "spec §1.2 注记-A（条件解绑，2026-09-09）——BUG-002"),
    ("status→已取消", "spec §1.2 注记-A（条件解绑，2026-09-09）——BUG-002"),
    ("status → 已取消", "spec §1.2 注记-A（条件解绑，2026-09-09）——BUG-002"),
    ("需求全部已取消", "spec §1.2 注记-A（条件解绑，2026-09-09）——BUG-002"),
]

VIOLATIONS = []
for dirpath, dirnames, filenames in os.walk(SCAN_ROOT):
    dirnames[:] = [d for d in dirnames if d not in EXCLUDE_DIRS]
    for name in filenames:
        if not name.endswith((".md", ".tsx", ".ts")):
            continue
        path = os.path.join(dirpath, name)
        rel = os.path.relpath(path, REPO)
        try:
            with open(path, encoding="utf-8") as f:
                for lineno, line in enumerate(f, 1):
                    for phrase, source in FORBIDDEN:
                        if phrase in line:
                            VIOLATIONS.append(f"{rel}:{lineno}: 「{phrase}」 → 正本 {source}")
        except (UnicodeDecodeError, OSError):
            continue

print(f"D-4 文档一致性巡检（{os.path.relpath(SCAN_ROOT, REPO)}/**，排除 {'/'.join(sorted(EXCLUDE_DIRS))}）：违规 {len(VIOLATIONS)} 条")
for v in VIOLATIONS:
    print(f"  [FAIL] {v}")
sys.exit(1 if VIOLATIONS else 0)
