# -*- coding: utf-8 -*-
"""调试:检查 chunk_store resolve 输出的 knowledge_base_id。"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "src"))

os.environ.setdefault("MYSQL_HOST", "127.0.0.1")
os.environ.setdefault("MYSQL_PORT", "3306")
os.environ.setdefault("MYSQL_USER", "root")
os.environ.setdefault("MYSQL_PASSWORD", "")
os.environ.setdefault("MYSQL_DATABASE", "YLSW_Service_Table")

from dbutils.pooled_db import PooledDB
from config import MYSQL_CONFIG
from retrieval.chunk_store import ChunkStore

pool = PooledDB(creator=__import__("pymysql"), maxconnections=10, **MYSQL_CONFIG)
cs = ChunkStore(pool)
print("== resolve test ==")
for r in cs.resolve([(3, 0.9), (6, 0.8), (7, 0.7), (1, 0.6)]):
    print("kb=", repr(r.get("knowledge_base_id")), "chunk=", r["chunk_id"], "doc=", r["document_id"])
print("== doc_meta ==")
print(cs.doc_meta(4))
