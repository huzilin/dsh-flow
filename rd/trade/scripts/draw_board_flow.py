import requests
import json
import time

APP_ID = "cli_a9106818dc78dbc6"
APP_SECRET = "gqf0cqECFmf8q9I4Zv6YrbdfADncV4no"
DOC_ID = "KiSDdFHAQo2tDsxZJhbc4sp3nNg"

def get_token():
    url = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
    r = requests.post(url, json={"app_id": APP_ID, "app_secret": APP_SECRET})
    return r.json().get("tenant_access_token")

def api(method, path, data=None):
    token = get_token()
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    url = f"https://open.feishu.cn{path}"
    if method == "GET":
        r = requests.get(url, headers=headers)
    elif method == "POST":
        r = requests.post(url, headers=headers, json=data)
    elif method == "PATCH":
        r = requests.patch(url, headers=headers, json=data)
    result = r.json()
    if result.get("code") != 0:
        print(f"  ❌ API错误: {result.get('code')} - {result.get('msg')}")
        return None
    return result.get("data")

print("=" * 60)
print("飞书画板绘图 - Step 2: 获取/创建画板")
print("=" * 60)

# Step 2: 读取文档 blocks，查找画板
print(f"\n🔍 读取文档 {DOC_ID} 的 blocks...")
blocks = api("GET", f"/open-apis/docx/v1/documents/{DOC_ID}/blocks")
if not blocks:
    print("❌ 无法读取文档")
    exit(1)

items = blocks.get("items", [])
print(f"✅ 文档共 {len(items)} 个 blocks")

board_block = None
for item in items:
    bt = item.get("block_type")
    print(f"  - block_type={bt}, block_id={item.get('block_id')}")
    if bt == 43:  # board type
        board_block = item

if board_block:
    whiteboard_id = board_block.get("board", {}).get("token")
    print(f"\n✅ 找到已有画板: whiteboard_id={whiteboard_id}")
else:
    print("\n📝 文档无画板，正在创建...")
    # 在文档末尾插入画板
    result = api("PATCH", f"/open-apis/docx/v1/documents/{DOC_ID}/blocks/batch_update", {
        "requests": [{
            "insert": {
                "end_of": {"block_id": DOC_ID},
                "nodes": [{"block_type": 43, "board": {"width": 800, "height": 1100}}]
            }
        }]
    })
    if result:
        whiteboard_id = result.get("results", [{}])[0].get("board", {}).get("token")
        if not whiteboard_id:
            # 重新读取获取 whiteboard_id
            time.sleep(2)
            blocks2 = api("GET", f"/open-apis/docx/v1/d