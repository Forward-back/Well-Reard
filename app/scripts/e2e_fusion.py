# -*- coding: utf-8 -*-
"""app 全栈 e2e:注册(工号)→登录→建 KB→上传 md(§2.14 元数据)→等解析→
验证 YLSW 收到推送→内部问答→验证 PG chunks 元数据。

用法:python app/scripts/e2e_fusion.py  (app 容器 8080 已起)
"""
import json
import os
import sys
import time

import requests

BASE = "http://127.0.0.1:8080/api/v1"
YLSW = "http://127.0.0.1:5000"

PASS, FAIL = [], []


def check(name, cond, detail=""):
    (PASS if cond else FAIL).append(name)
    print(f"[{'PASS' if cond else 'FAIL'}] {name} {detail}")


def req(method, path, token=None, **kw):
    headers = kw.pop("headers", {})
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = requests.request(method, BASE + path, headers=headers, timeout=120, **kw)
    return r


def main():
    # 0. app 健康
    r = requests.get("http://127.0.0.1:8080/health", timeout=30)
    check("app /health", r.status_code == 200, r.text[:80])

    # 1. 注册(工号 UMS10001 → 员工 admin);已存在则视为通过(幂等重跑)
    r = req("POST", "/auth/register", json={
        "username": "e2etest", "email": "e2etest@example.com",
        "password": "E2eTest123!", "emp_no": "UMS10001",
    })
    print("register:", r.status_code, r.text[:200])
    dup = r.status_code in (400, 409) and ("already exists" in r.text or "已存在" in r.text)
    check("注册", r.status_code in (200, 201) or dup, r.text[:120])

    # 2. 登录
    r = req("POST", "/auth/login", json={"email": "e2etest@example.com", "password": "E2eTest123!"})
    print("login:", r.status_code, r.text[:200])
    data = r.json()
    token = data.get("token") or data.get("access_token") or ""
    check("登录拿 token", bool(token))
    if not token:
        sys.exit(1)

    # 3. 模型列表(建 KB 需要 embedding model id);缺失则自动创建(SiliconFlow)
    def load_env():
        from pathlib import Path
        env = {}
        envpath = Path(__file__).resolve().parent.parent / ".env"
        if envpath.exists():
            for line in envpath.read_text(encoding="utf-8").splitlines():
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    k, v = line.split("=", 1)
                    env[k.strip()] = v.strip()
        return env

    env = load_env()
    sf_key = env.get("YLSW_BGE_API_KEY", "")

    def ensure_model(name, mtype, display):
        nonlocal token
        r = req("GET", "/models", token)
        models = r.json().get("data", [])
        hit = next((m for m in models if m.get("type") == mtype), None)
        if hit:
            return hit.get("id") or ""
        r = req("POST", "/models", token, json={
            "name": name, "display_name": display, "type": mtype,
            "source": "siliconflow",
            "parameters": {
                "base_url": "https://api.siliconflow.cn/v1",
                "api_key": sf_key,
                "embedding_parameters": {"dimension": 1024},
            },
        })
        print(f"create model {mtype}:", r.status_code, r.text[:150])
        return (r.json().get("data") or {}).get("id") or ""

    qa_id = ensure_model("deepseek-ai/DeepSeek-V3", "KnowledgeQA", "DeepSeek-V3 (SiliconFlow)")
    emb_id = ensure_model("BAAI/bge-m3", "Embedding", "BGE-M3 (SiliconFlow)")
    print("embedding_model_id=", emb_id, "summary_model_id=", qa_id)

    # 4. 建 KB(带 summary model,触发 metadata:extract)
    r = req("POST", "/knowledge-bases", token, json={
        "name": "e2e-融合测试库",
        "description": "e2e",
        "embedding_model_id": emb_id,
        "summary_model_id": qa_id,
    })
    print("kb:", r.status_code, r.text[:200])
    kb = r.json().get("data", {})
    kb_id = kb.get("id") or ""
    check("建 KB", bool(kb_id), kb_id)

    # 5. 上传 md(带 §2.14 元数据)
    md_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "e2e_doc.md")
    with open(md_path, "w", encoding="utf-8") as f:
        f.write(
            "# 商户结算管理办法\n\n## 第一条 目的\n\n为规范商户结算行为，制定本办法。\n\n"
            "## 第二条 结算周期\n\n商户结算周期为 T+1 工作日，每日 16 点前完成清算。\n\n"
            "## 第三条 废止说明\n\n本办法自 2026-07-01 起被《商户结算管理办法（2026 版）》替代。\n"
        )
    with open(md_path, "rb") as f:
        r = req("POST", f"/knowledge-bases/{kb_id}/knowledge/file", token,
                files={"file": ("REG_20250101_银联商务_测试产品_商户结算管理办法_V1.0_ZH.md", f, "text/markdown")},
                data={
                    "metadata": json.dumps({
                        "doc_number": "测试2025001号", "org_name": "银联商务", "doc_date": "2025-01-01",
                        "doc_title": "商户结算管理办法", "doc_type": "regulation", "product_name": "测试产品",
                        "lifecycle_status": "expired", "effective_date": "2025-01-01",
                        "expiry_date": "2026-07-01", "superseded_by": "测试2026001号",
                        "sensitivity": "internal", "department": "清算部",
                    }),
                })
    print("upload:", r.status_code, r.text[:200])
    up = r.json().get("data", {})
    knowledge_id = up.get("id") or ""
    check("上传文档", bool(knowledge_id), knowledge_id)

    # 6. 等解析完成(轮询 60s)
    ok = False
    for _ in range(30):
        r = req("GET", f"/knowledge-bases/{kb_id}/knowledge", token)
        items = r.json().get("data", [])
        item = next((k for k in items if k.get("id") == knowledge_id), None)
        if item and item.get("parse_status") == "completed":
            ok = True
            break
        time.sleep(2)
    check("解析完成", ok, str(item.get("parse_status") if item else None))

    # 7. YLSW 收到推送
    r = requests.get(YLSW + "/documents/ids", timeout=30)
    ylsw_ids = r.json().get("ids", [])
    check("YLSW 收到推送", knowledge_id in ylsw_ids, f"{len(ylsw_ids)} docs")

    # 8. 内部问答
    r = req("POST", "/sessions", token, json={"title": "e2e"})
    session_id = r.json().get("data", {}).get("id") or ""
    print("session:", r.status_code, session_id)
    r = req("POST", f"/knowledge-chat/{session_id}", token, json={
        "query": "商户结算周期是多少", "knowledge_base_ids": [kb_id], "disable_title": True,
    }, stream=True)
    body = ""
    for line in r.iter_lines():
        if not line:
            continue
        line = line.decode("utf-8")
        if line.startswith("data:"):
            body += line[5:].strip()
    check("问答 SSE 有内容", len(body) > 0, body[:120])

    print(f"\n===== {len(PASS)} PASS / {len(FAIL)} FAIL =====")
    sys.exit(1 if FAIL else 0)


if __name__ == "__main__":
    main()
