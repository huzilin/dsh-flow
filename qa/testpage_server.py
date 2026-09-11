#!/usr/bin/env python3
"""wbflow 拍板校验测试页面服务（qa/testpage.html 的承载 + API 代理）。

rd 服务无 CORS 头，浏览器直连会被拦——本进程同源承载静态页并把 /api/* 反代到
rd（默认 http://127.0.0.1:6081，QA_BASE 可覆盖）。零第三方依赖。

用法：
  source /tmp/wbflow-qa-env.env   # 先起验收环境（qa_env_up.sh）
  python3 testpage_server.py      # 打开 http://127.0.0.1:6090
"""
import os
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

HERE = os.path.dirname(os.path.abspath(__file__))
RD_BASE = os.environ.get("QA_BASE", "http://127.0.0.1:6081")
PORT = int(os.environ.get("QA_TESTPAGE_PORT", "6090"))


class Handler(BaseHTTPRequestHandler):
    def _static(self):
        with open(os.path.join(HERE, "testpage.html"), "rb") as f:
            body = f.read()
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _proxy(self):
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else None
        req = urllib.request.Request(RD_BASE + self.path, data=body, method=self.command)
        for h in ("Content-Type", "Authorization"):
            if self.headers.get(h):
                req.add_header(h, self.headers[h])
        try:
            with urllib.request.urlopen(req, timeout=30) as r:
                payload, status = r.read(), r.status
        except urllib.error.HTTPError as e:
            payload, status = e.read(), e.code
        except Exception as e:  # rd 不可达
            payload, status = f'{{"ok":false,"error":"proxy: {e}","code":"PROXY"}}'.encode(), 502
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self):
        self._static() if self.path in ("/", "/index.html") else self._proxy()

    def do_POST(self):
        self._proxy()

    def do_PUT(self):
        self._proxy()

    def do_PATCH(self):
        self._proxy()

    def do_DELETE(self):
        self._proxy()

    def log_message(self, fmt, *args):
        print("[testpage]", fmt % args)


if __name__ == "__main__":
    print(f"testpage → http://127.0.0.1:{PORT}  (proxy /api → {RD_BASE})")
    ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
