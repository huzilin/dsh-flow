#!/usr/bin/env python3
"""DSH Web GUI 局域网访问转发代理。

DSH Desktop 的 Web GUI 默认只监听 127.0.0.1:57338（本机回环），
导致局域网内其他设备无法访问 192.168.x.x:57338。
本脚本监听 0.0.0.0:57338 并把连接转发到 127.0.0.1:57338，
使同一局域网内的设备可通过 <本机局域网IP>:57338 访问 DSH Web GUI。

用法:
    python3 tools/dsh-lan-forward.py [listen_port] [target_host] [target_port]
    默认: python3 tools/dsh-lan-forward.py 57338 127.0.0.1 57338

注意: 将 GUI 暴露到局域网后，同一网段内所有设备都能访问；
请仅在可信网络中使用。
"""
import socket
import sys
import threading

LISTEN_ADDR = "0.0.0.0"
TARGET_HOST = "127.0.0.1"
TARGET_PORT = 57338


def banner():
    print(f"[dsh-forward] listening on {LISTEN_ADDR}:{LISTEN_PORT} -> {TARGET_HOST}:{TARGET_PORT}")


def pipe(src, dst):
    try:
        while True:
            data = src.recv(65536)
            if not data:
                break
            dst.sendall(data)
    except OSError:
        pass
    finally:
        try:
            dst.shutdown(socket.SHUT_WR)
        except OSError:
            pass


def handle(client, addr):
    try:
        upstream = socket.create_connection((TARGET_HOST, TARGET_PORT), timeout=10)
    except OSError as e:
        print(f"[dsh-forward] upstream connect failed for {addr}: {e}")
        client.close()
        return
    print(f"[dsh-forward] + {addr} connected")
    t1 = threading.Thread(target=pipe, args=(client, upstream), daemon=True)
    t2 = threading.Thread(target=pipe, args=(upstream, client), daemon=True)
    t1.start()
    t2.start()
    t1.join()
    t2.join()
    client.close()
    upstream.close()
    print(f"[dsh-forward] - {addr} closed")


def main():
    global LISTEN_PORT
    LISTEN_PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 57338
    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind((LISTEN_ADDR, LISTEN_PORT))
    srv.listen(128)
    banner()
    try:
        while True:
            client, addr = srv.accept()
            threading.Thread(target=handle, args=(client, addr), daemon=True).start()
    except KeyboardInterrupt:
        print("\n[dsh-forward] stopping")
    finally:
        srv.close()


if __name__ == "__main__":
    main()