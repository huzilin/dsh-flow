#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
serve.py — 将 docs 目录暴露到局域网，.md 即时渲染为 HTML（UTF-8）。

用法:
    .venv/bin/python serve.py [port]        # 默认 8000，绑定 0.0.0.0
    .venv/bin/python serve.py 8080          # 换端口

局域网访问:
    http://<本机局域网IP>:8000/                          # docs 根索引（自动列出全部文档）
    http://<本机局域网IP>:8000/design/m2-flow-tree.md    # 单个 md 即时渲染
    http://<本机局域网IP>:8000/prototype/m3-board-prototype.html  # prototype 静态页

行为:
  - *.md            → 用 venv 的 markdown 库即时渲染成带 UTF-8 深色样式的 HTML，
                      相对链接（md → md）原样可用，会自动走渲染入口
  - 目录路径        → 生成带面包屑的目录索引页（md/html 均可点击）
  - 其他后缀        → python http.server 默认静态服务
  - 隐藏文件/目录   → 索引中隐藏，直接请求返回 404

字符集:
  - 所有 md 源文件已逐字节验证为合法 UTF-8（无 BOM、无控制字符、无 GBK 混排）
  - 渲染页 <meta charset="UTF-8">，响应头 Content-Type: text/html; charset=utf-8，
    读取源文件固定 encoding='utf-8'，杜绝浏览器按 GBK 解码乱码

安全:
  - web 根目录固定为 docs/，只暴露文档；.venv / .workbench / serve.py 等不暴露
  - 路径穿越由 SimpleHTTPRequestHandler.translate_path 兜底（无法跳出 web 根）
"""
import html as htmlmod
import http.server
import markdown
import os
import re
import sys
import urllib.parse

BASE = os.path.dirname(os.path.abspath(__file__))
DOCS = os.path.join(BASE, 'docs')

CSS = """
:root { --bg:#14161c; --panel:#1b1e27; --panel2:#212530; --border:#2b2f3d;
        --text:#e6e8ef; --muted:#8a90a3; --accent:#e8b34a; --blue:#7ab8ff;
        --green:#58c07a; --red:#d98080; }
* { box-sizing:border-box; }
body { background:var(--bg); color:var(--text); font-family:-apple-system,"PingFang SC","Microsoft YaHei",sans-serif;
       margin:0; padding:32px 20px; line-height:1.75; font-size:15px; }
.wrap { max-width:860px; margin:0 auto; background:var(--panel); border:1px solid var(--border);
        border-radius:12px; padding:32px 40px; }
.crumb { font-size:13px; color:var(--muted); margin-bottom:18px; }
.crumb a { color:var(--blue); text-decoration:none; }
.crumb a:hover { text-decoration:underline; }
h1 { font-size:26px; border-bottom:1px solid var(--border); padding-bottom:12px; margin-top:0; }
h2 { font-size:21px; margin-top:34px; border-left:3px solid var(--accent); padding-left:10px; }
h3 { font-size:18px; margin-top:26px; }
h4 { font-size:16px; }
blockquote { margin:14px 0; padding:10px 16px; background:var(--panel2); border-left:3px solid var(--accent);
            border-radius:6px; color:var(--muted); }
blockquote p { margin:4px 0; }
code { background:var(--panel2); padding:1px 6px; border-radius:4px; font-family:Menlo,Consolas,monospace;
       font-size:0.92em; color:#c8cdd8; }
pre { background:#181b24; border:1px solid var(--border); border-left:3px solid var(--blue); border-radius:8px;
      padding:14px 18px; overflow-x:auto; }
pre code { background:transparent; padding:0; }
table { border-collapse:collapse; width:100%; margin:14px 0; font-size:14px; }
th, td { border:1px solid var(--border); padding:8px 12px; text-align:left; }
th { background:var(--panel2); color:var(--muted); font-weight:600; }
a { color:var(--blue); }
hr { border:none; border-top:1px solid var(--border); margin:22px 0; }
ul, ol { padding-left:22px; }
li { margin:4px 0; }
strong { color:#fff; }
/* 目录索引 */
.idx-dir, .idx-file { display:block; padding:7px 12px; margin:3px 0; border-radius:6px;
        text-decoration:none; color:var(--text); }
.idx-dir { color:var(--accent); font-weight:600; }
.idx-dir::before { content:"📁  "; }
.idx-file { color:var(--text); }
.idx-file::before { content:"📄  "; }
.idx-dir:hover, .idx-file:hover { background:var(--panel2); }
.idx-note { color:var(--muted); font-size:0.9em; margin-left:8px; }
.idx-count { color:var(--muted); font-size:13px; margin:6px 0 14px; }
"""


def render_page(title, inner):
    return ('<!DOCTYPE html>\n<html lang="zh-CN">\n<head>\n<meta charset="UTF-8">\n'
            '<meta name="viewport" content="width=device-width, initial-scale=1.0">\n'
            '<title>%s</title>\n<style>%s</style>\n</head>\n<body>\n<div class="wrap">\n%s\n'
            '</div>\n</body>\n</html>\n' % (htmlmod.escape(title), CSS, inner))


def breadcrumb(rel_path):
    """rel_path: 相对 docs 根（''=根，'design' 等）。生成面包屑导航。"""
    parts = rel_path.split('/') if rel_path else []
    crumbs = ['<span class="crumb"><a href="/">docs</a>']
    acc = ''
    for p in parts:
        acc += '/' + p
        crumbs.append('&nbsp;/&nbsp;<a href="%s/">%s</a>' % (htmlmod.escape(acc), htmlmod.escape(p)))
    crumbs.append('</span>')
    return ''.join(crumbs)


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=DOCS, **kwargs)

    def _send_html(self, data):
        # 所有 HTML 响应（md 渲染页 / 目录索引）都显式声明 UTF-8，
        # 避免浏览器按系统编码（GBK）解码中文
        self.send_response(200)
        self.send_header('Content-Type', 'text/html; charset=utf-8')
        self.send_header('Content-Length', str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    # ---------- md 渲染 ----------
    def serve_md(self, abs_path, rel_path):
        try:
            with open(abs_path, encoding='utf-8') as fh:
                md_text = fh.read()
        except OSError:
            self.send_error(404, 'File not found')
            return
        body = markdown.markdown(md_text, extensions=['tables', 'fenced_code', 'nl2br'])
        m = re.search(r'<h1[^>]*>(.*?)</h1>', body, re.S)
        title = htmlmod.unescape(m.group(1)).strip() if m else os.path.basename(abs_path)[:-3]
        inner = breadcrumb(rel_path) + body
        data = render_page(title, inner).encode('utf-8')
        self._send_html(data)

    # ---------- 目录索引 ----------
    def serve_dir_index(self, rel_path):
        dir_path = os.path.join(DOCS, rel_path.replace('/', os.sep)) if rel_path else DOCS
        try:
            names = sorted(os.listdir(dir_path))
        except OSError:
            self.send_error(404, 'Directory not found')
            return
        dirs, files = [], []
        for n in names:
            if n.startswith('.'):  # 隐藏 .DS_Store 等
                continue
            p = os.path.join(dir_path, n)
            (dirs if os.path.isdir(p) else files).append(n)
        # 链接一律用绝对路径（带前导 /），避免在 /design/ 等子目录页面上
        # 相对解析成 /design/design/... 二级重复路径
        rel_prefix = '/' + (rel_path + '/' if rel_path else '')
        rows = []
        for d in dirs:
            rows.append('<a class="idx-dir" href="%s/">%s</a>'
                        % (htmlmod.escape(rel_prefix + d), htmlmod.escape(d)))
        for f in files:
            rows.append('<a class="idx-file" href="%s">%s</a>'
                        % (htmlmod.escape(rel_prefix + f), htmlmod.escape(f)))
        title = 'docs/' + rel_path if rel_path else 'docs 文档目录'
        inner = (breadcrumb(rel_path)
                 + '<h1>%s</h1>' % htmlmod.escape(title)
                 + '<div class="idx-count">%d 个目录，%d 个文件（md 点击即时渲染）</div>'
                 % (len(dirs), len(files))
                 + ('<p class="idx-note">（空目录）</p>' if not rows else ''.join(rows)))
        data = render_page(title, inner).encode('utf-8')
        self._send_html(data)

    # ---------- 请求路由 ----------
    def do_GET(self):
        path = urllib.parse.urlsplit(self.path).path
        if path == '/':
            self.serve_dir_index('')
            return
        # 隐藏文件一律 404
        if any(seg.startswith('.') for seg in path.split('/') if seg):
            self.send_error(404, 'Hidden file')
            return
        abs_path = self.translate_path(self.path)
        # 目录：带斜杠直接渲染索引；不带斜杠 301 补斜杠
        if path.endswith('/') and os.path.isdir(abs_path):
            self.serve_dir_index(path[1:-1])
            return
        if os.path.isdir(abs_path):
            self.send_response(301, 'Moved Permanently')
            self.send_header('Location', self.path + '/')
            self.end_headers()
            return
        if path.endswith('.md') and os.path.isfile(abs_path):
            self.serve_md(abs_path, path[1:-3])
            return
        if not os.path.isfile(abs_path):
            self.send_error(404, 'File not found')
            return
        # 其余后缀（html / png / css ...）走默认静态服务
        return super().do_GET()


def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8000
    server = http.server.ThreadingHTTPServer(('0.0.0.0', port), Handler)
    print('serving docs on 0.0.0.0:%d  (md auto-render on, charset=utf-8)' % port)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass


if __name__ == '__main__':
    main()