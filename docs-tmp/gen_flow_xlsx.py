# ⚠️ 已过时（2026-09-05 v3 编排模型定稿）：本生成器基于旧模型（旧节点拓扑/trigger 词汇/
# respond_to_cr 等已废止动作）——将按 v3.1 节点配置（business/flow-derivation-tech-task-plan.md）
# 重写（工单 07 seed 重写计划）；配套 能力链路-流程-节点-能力-动作.xlsx 同步过时。勿据本文档生成。
# -*- coding: utf-8 -*-
"""生成 能力链路.xlsx：流程→节点→能力→action→how 回溯表
数据源：/tmp/rows.csv（gen_flow_rows.py 产出，源头 seed.json + capability-review-v2 推演）
遵循 xlsx skill：Arial 字体 / 公式汇总 / 无线错误。
"""
import csv
from openpyxl import Workbook
from openpyxl.styles import Font, PatternFill, Alignment, Border, Side
from openpyxl.utils import get_column_letter

SRC = '/tmp/rows.csv'
OUT = '/Users/huzilin/workdir/test-flow/docs-tmp/能力链路-流程-节点-能力-动作.xlsx'

rows = []
with open(SRC, newline='') as f:
    r = csv.reader(f)
    header = next(r)
    for line in r:
        rows.append(line)

HDR = header  # ['流程','节点order','节点','能力','角色位','动作ID','动作名','trigger','when','how','参考','输出产物','工具']

wb = Workbook()

# ============ Sheet1：链路明细 ============
ws = wb.active
ws.title = '链路明细'
ws.sheet_view.showGridLines = False

# 表头样式
hdr_fill = PatternFill('solid', fgColor='1F4E79')
hdr_font = Font(name='Arial', size=10, bold=True, color='FFFFFF')
body_font = Font(name='Arial', size=9)
thin = Side(style='thin', color='D0D0D0')
border = Border(left=thin, right=thin, top=thin, bottom=thin)
wrap = Alignment(vertical='top', wrap_text=True)
center = Alignment(horizontal='center', vertical='top')

for c, h in enumerate(HDR, 1):
    cell = ws.cell(1, c, h)
    cell.fill = hdr_fill; cell.font = hdr_font
    cell.alignment = Alignment(horizontal='center', vertical='center')
    cell.border = border

# 流程底色（便于快速分层）
flow_fill = {
    'business': PatternFill('solid', fgColor='EAF1F8'),
    'tech':     PatternFill('solid', fgColor='FDF3E7'),
    'task':     PatternFill('solid', fgColor='EAF7EA'),
    'plan':     PatternFill('solid', fgColor='F4EAF6'),
}
# 触发色（trigger 列）
trig_fill = {
    'node_enter': PatternFill('solid', fgColor='DDEBF7'),
    'dev_complete': PatternFill('solid', fgColor='E2EFDA'),
    'reviewer_feedback(open)': PatternFill('solid', fgColor='FFF2CC'),
    'node_review_pending': PatternFill('solid', fgColor='FCE4D6'),
    'decide': PatternFill('solid', fgColor='D9E1F2'),
}

for i, row in enumerate(rows, 2):
    for c, val in enumerate(row, 1):
        cell = ws.cell(i, c, val)
        cell.font = body_font
        cell.border = border
        cell.alignment = wrap if c in (10,) else (center if c in (2,4,5,7,8,13) else wrap)
        # 首列流程底色
        if c == 1 and val in flow_fill:
            cell.fill = flow_fill[val]
        # trigger 列背景
        if c == 8 and val in trig_fill:
            cell.fill = trig_fill[val]
        # how 列加宽
    ws.row_dimensions[i].height = 60

# 列宽
widths = [9, 8, 14, 16, 7, 24, 13, 18, 26, 90, 30, 16, 22]
for c, w in enumerate(widths, 1):
    ws.column_dimensions[get_column_letter(c)].width = w
ws.freeze_panes = 'C2'
ws.auto_filter.ref = f'A1:{get_column_letter(len(HDR))}{len(rows)+1}'

# ============ Sheet2：说明 ============
ws3 = wb.create_sheet('说明')
ws3.sheet_view.showGridLines = False
notes = [
    ('能力链路 · 说明', True),
    ('', False),
    ('目的：回溯「流程 → 节点（产物）→ 能力 → action → how」的完整执行链路。', False),
    ('数据源：', False),
    ('  · 流程/节点/能力绑定：workbench-sql/seed.json（flow_template_nodes）', False),
    ('  · 动作动作/how 推演：docs-tmp/capability-review-v2.md §0.2（backend-dev 完整范例 + 评审协议模板）', False),
    ('  · 产物类型（ProductType）：capability-review-v2.md §0.1 权威表', False),
    ('', False),
    ('列说明：', False),
    ('  · 角色位：能力挂载的角色（产出能力=owner 角色；评审能力=评审者；user 位占位留空），如「能力挂哪个角色就是谁」', False),
    ('  · trigger：统一触发契约（execution-loop.md §3/§11），程序可判定', False),
    ('     node_enter / dev_complete / reviewer_feedback(open) / node_review_pending / decide / issue_replied / reject_rolled_back', False),
    ('  · 参考（reference）：文档/代码类用 productType（标准枚举）；对话/评论类用 query（属性过滤）', False),
    ('  · 输出产物：本动作登记产出（productType），与 wb_add_artifact 校验同源', False),
    ('  · 产物名用「」框起（如「product_doc」），便于识别', False),
    ('  · how 模板化（J7 定稿）：how/ref 中产物为「《标准名》(模板路径)」，如《rd_design》(docs/{{req-version}}/{{req-name}}/rd/design.md)；host 端 renderTpl 替换 {{var}} 即得真实路径', False),
    ('  · 「完成」节点：4 流程末尾的 DONE 结束节点，无能力/动作，标志流程终点（由 Leader 确认收尾）', False),
    ('', False),
    ('使用：链路明细 表可按「流程/节点/能力/trigger」筛选。', False),
]
for i, (txt, bold) in enumerate(notes, 1):
    cell = ws3.cell(i, 1, txt)
    cell.font = Font(name='Arial', size=10, bold=bold)

wb.save(OUT)
print('saved:', OUT)
print('rows:', len(rows))