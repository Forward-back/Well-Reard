"""
银联商务融合项目：存量文档批量入库脚本。

流程：登录管理员 → 创建/复用"银联商务知识库"（父子分块 300/1200）
→ 批量上传 data/rag/Data 下的文档 → 轮询解析状态 → 触发 YLSW /reload。

用法（从仓库根目录执行）：
  python scripts/ingest_data.py --email admin@ums.com --password xxx \
      [--data-dir data/rag/Data] [--app-url http://localhost:8080] \
      [--ylsw-url http://localhost:5000] [--kb-name 银联商务知识库]
"""

import argparse
import os
import sys
import time

import requests

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_DATA_DIR = os.path.join(ROOT_DIR, "data", "rag", "Data")


def login(app_url: str, email: str, password: str) -> str:
    resp = requests.post(
        f"{app_url}/api/v1/auth/login",
        json={"email": email, "password": password},
        timeout=30,
    )
    resp.raise_for_status()
    data = resp.json().get("data", resp.json())
    token = data.get("token")
    if not token:
        raise RuntimeError(f"登录失败: {resp.text[:300]}")
    print(f"登录成功: {email}")
    return token


def find_or_create_kb(app_url: str, token: str, name: str) -> dict:
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.get(f"{app_url}/api/v1/knowledge-bases", headers=headers, timeout=30)
    resp.raise_for_status()
    payload = resp.json()
    items = payload.get("data", payload)
    for kb in items or []:
        if kb.get("name") == name:
            print(f"复用已有知识库: {kb['id']}")
            return kb

    body = {
        "name": name,
        "description": "银联商务文档库（YLSW 融合版）",
        "chunking_config": {
            "chunk_size": 300,
            "chunk_overlap": 128,
            "enable_parent_child": True,
            "parent_chunk_size": 1200,
            "child_chunk_size": 300,
            "separators": ["\n\n", "\n", "。", "！", "？", "；", " "],
        },
    }
    resp = requests.post(
        f"{app_url}/api/v1/knowledge-bases", json=body, headers=headers, timeout=30
    )
    resp.raise_for_status()
    kb = resp.json().get("data", resp.json())
    print(f"创建知识库: {kb['id']}")
    return kb


def upload_file(app_url: str, token: str, kb_id: str, path: str) -> bool:
    headers = {"Authorization": f"Bearer {token}"}
    with open(path, "rb") as f:
        resp = requests.post(
            f"{app_url}/api/v1/knowledge-bases/{kb_id}/knowledge/file",
            headers=headers,
            files={"file": (os.path.basename(path), f, "text/markdown")},
            timeout=120,
        )
    return resp.status_code == 200


def wait_processing(app_url: str, token: str, kb_id: str, total: int, timeout_s: int = 1800):
    headers = {"Authorization": f"Bearer {token}"}
    start = time.time()
    while time.time() - start < timeout_s:
        resp = requests.get(
            f"{app_url}/api/v1/knowledge-bases/{kb_id}/knowledge?page=1&page_size=200",
            headers=headers, timeout=30,
        )
        resp.raise_for_status()
        payload = resp.json()
        items = payload.get("data", payload)
        rows = items if isinstance(items, list) else items.get("items", items.get("list", []))
        done = sum(1 for k in rows if k.get("parse_status") == "completed")
        failed = sum(1 for k in rows if k.get("parse_status") == "failed")
        print(f"\r解析进度: {done}/{total} 完成, {failed} 失败", end="", flush=True)
        if done + failed >= total and done + failed > 0:
            print()
            return done, failed
        time.sleep(5)
    print("\n等待超时")
    return -1, -1


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--email", required=True)
    ap.add_argument("--password", required=True)
    ap.add_argument("--data-dir", default=DEFAULT_DATA_DIR)
    ap.add_argument("--app-url", default="http://localhost:8080")
    ap.add_argument("--ylsw-url", default="http://localhost:5000")
    ap.add_argument("--kb-name", default="银联商务知识库")
    args = ap.parse_args()

    files = sorted(
        f for f in os.listdir(args.data_dir)
        if f.lower().endswith((".md", ".pdf", ".docx", ".doc", ".txt", ".xlsx"))
    )
    if not files:
        print(f"数据目录无文档: {args.data_dir}")
        sys.exit(1)
    print(f"待上传文档: {len(files)} 份")

    token = login(args.app_url, args.email, args.password)
    kb = find_or_create_kb(args.app_url, token, args.kb_name)

    ok, skip = 0, 0
    for i, name in enumerate(files, 1):
        path = os.path.join(args.data_dir, name)
        if upload_file(args.app_url, token, kb["id"], path):
            ok += 1
        else:
            skip += 1
        if i % 10 == 0:
            print(f"已上传 {i}/{len(files)}（成功 {ok}，失败 {skip}）")
        time.sleep(0.3)
    print(f"上传完成: 成功 {ok} / 失败 {skip}")

    print("等待解析（父子分块 + embedding，耗时较长）...")
    wait_processing(args.app_url, token, kb["id"], ok)

    print("触发 YLSW 内存索引重建...")
    try:
        resp = requests.post(f"{args.ylsw_url}/reload", timeout=300)
        print(f"YLSW /reload: {resp.status_code} {resp.text[:200]}")
    except Exception as e:
        print(f"YLSW /reload 失败: {e}")


if __name__ == "__main__":
    main()
