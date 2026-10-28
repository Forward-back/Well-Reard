"""配置 BGE-M3 embedding 模型并重解析存量文档。

用法:
  python app/scripts/setup_embedding.py --email admin@ums.com --password xxx \
      [--app-url http://localhost:8080]
"""

import argparse
import json
import os
import sys
import time

import requests


def login(app_url, email, password):
    resp = requests.post(f"{app_url}/api/v1/auth/login",
                         json={"email": email, "password": password}, timeout=30)
    resp.raise_for_status()
    data = resp.json().get("data", resp.json())
    token = data.get("token")
    if not token:
        raise RuntimeError(f"登录失败: {resp.text[:300]}")
    print("登录成功")
    return token


def create_model(app_url, token, name, dtype, source, params, is_default):
    headers = {"Authorization": f"Bearer {token}"}
    # 复用同名模型
    resp = requests.get(f"{app_url}/api/v1/models", headers=headers, timeout=30)
    resp.raise_for_status()
    for m in resp.json().get("data", []) or []:
        if m.get("name") == name:
            print(f"复用已有模型 {name}: {m['id']}")
            return m["id"]

    body = {
        "name": name,
        "display_name": name,
        "type": dtype,
        "source": source,
        "description": "",
        "parameters": params,
    }
    resp = requests.post(f"{app_url}/api/v1/models", json=body, headers=headers, timeout=30)
    if resp.status_code != 201:
        print(f"创建模型 {name} 失败: {resp.status_code} {resp.text[:400]}")
        sys.exit(1)
    mid = resp.json().get("data", {}).get("id")
    print(f"创建模型 {name}: {mid}")
    if is_default:
        r2 = requests.put(f"{app_url}/api/v1/models/{mid}",
                          json={"is_default": True}, headers=headers, timeout=30)
        print(f"设为默认: {r2.status_code}")
    return mid


def update_kb(app_url, token, kb_id, kb_name, embedding_model_id):
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.put(f"{app_url}/api/v1/knowledge-bases/{kb_id}",
                        json={"name": kb_name, "embedding_model_id": embedding_model_id},
                        headers=headers, timeout=30)
    print(f"更新 KB embedding 模型: {resp.status_code}")
    if resp.status_code != 200:
        print(resp.text[:400])


def batch_reparse(app_url, token, kb_id):
    headers = {"Authorization": f"Bearer {token}"}
    # 收集 KB 下全部 knowledge id
    resp = requests.get(
        f"{app_url}/api/v1/knowledge-bases/{kb_id}/knowledge?page=1&page_size=500",
        headers=headers, timeout=30,
    )
    resp.raise_for_status()
    payload = resp.json()
    items = payload.get("data", payload)
    rows = items if isinstance(items, list) else items.get("items", items.get("list", []))
    ids = [k["id"] for k in rows]
    print(f"待重解析文档: {len(ids)}")
    # 分批（每批 50）
    for i in range(0, len(ids), 50):
        chunk = ids[i:i + 50]
        resp = requests.post(f"{app_url}/api/v1/knowledge/batch-reparse",
                             json={"kb_id": kb_id, "ids": chunk},
                             headers=headers, timeout=60)
        print(f"batch reparse [{i}:{i + len(chunk)}]: {resp.status_code}")
        if resp.status_code != 200:
            print(resp.text[:300])
        time.sleep(1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--email", required=True)
    ap.add_argument("--password", required=True)
    ap.add_argument("--app-url", default="http://localhost:8080")
    ap.add_argument("--kb-name", default="银联商务知识库")
    args = ap.parse_args()

    bge_key = os.getenv("BGE_API_KEY")
    if not bge_key:
        raise SystemExit("BGE_API_KEY is required")
    token = login(args.app_url, args.email, args.password)

    # 1. BGE-M3 embedding（SiliconFlow，OpenAI 兼容）
    emb_id = create_model(
        args.app_url, token, "bge-m3", "Embedding", "remote",
        {
            "base_url": "https://api.siliconflow.cn/v1",
            "api_key": bge_key,
            "model_name": "BAAI/bge-m3",
            "interface_type": "openai",
            "provider": "openai",
        },
        is_default=True,
    )

    # 2. 找 KB 并绑定 embedding 模型
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.get(f"{args.app_url}/api/v1/knowledge-bases", headers=headers, timeout=30)
    resp.raise_for_status()
    kb = None
    for k in resp.json().get("data", []) or []:
        if k.get("name") == args.kb_name:
            kb = k
            break
    if not kb:
        print(f"知识库不存在: {args.kb_name}")
        sys.exit(1)
    update_kb(args.app_url, token, kb["id"], kb["name"], emb_id)

    # 3. 重解析
    batch_reparse(args.app_url, token, kb["id"])
    print("完成。等待解析...")


if __name__ == "__main__":
    main()
