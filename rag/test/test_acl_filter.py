import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from retrieval.acl_filter import filter_candidates_by_acl


def resolve_doc_id(candidate):
    return str(candidate.get("source_doc_id") or "")


class AclFilterTest(unittest.TestCase):
    def test_mode_all_passes_through(self):
        candidates = [{"source_doc_id": "d1"}, {"text": "aggregate"}]
        self.assertEqual(filter_candidates_by_acl(candidates, {"mode": "all"}, resolve_doc_id), candidates)

    def test_allow_keeps_only_listed_documents(self):
        scope = {"mode": "allow", "document_ids": ["d1"]}
        kept = filter_candidates_by_acl(
            [{"source_doc_id": "d1"}, {"source_doc_id": "d2"}],
            scope,
            resolve_doc_id,
        )
        self.assertEqual(kept, [{"source_doc_id": "d1"}])

    def test_allow_drops_sql_aggregate_without_document_id(self):
        scope = {"mode": "allow", "document_ids": ["d1"]}
        kept = filter_candidates_by_acl(
            [{"text": "count=42", "sql": "SELECT COUNT(*) FROM t"}],
            scope,
            resolve_doc_id,
        )
        self.assertEqual(kept, [])

    def test_deny_drops_listed_and_unknown(self):
        scope = {"mode": "deny", "document_ids": ["d1"]}
        kept = filter_candidates_by_acl(
            [{"source_doc_id": "d1"}, {"source_doc_id": "d2"}, {"text": "aggregate"}],
            scope,
            resolve_doc_id,
        )
        self.assertEqual(kept, [{"source_doc_id": "d2"}])

    def test_missing_scope_fails_closed(self):
        self.assertEqual(
            filter_candidates_by_acl([{"source_doc_id": "d1"}], None, resolve_doc_id),
            [],
        )


if __name__ == "__main__":
    unittest.main()
