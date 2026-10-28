"""Document-level ACL filtering shared by all retrieval branches.

The resolver is injected so this module stays importable without loading
FAISS/MySQL/model singletons in unit tests.
"""


def filter_candidates_by_acl(candidates, acl_scope, resolve_doc_id):
    """Keep only candidates whose app document id is allowed by acl_scope.

    - mode=all: pass through
    - allow: keep listed document ids; unknown ids are dropped
    - deny: drop listed document ids; unknown ids are dropped
    """
    if not acl_scope:
        return []
    mode = acl_scope.get("mode")
    if mode == "all":
        return list(candidates or [])
    if mode not in ("allow", "deny"):
        return []
    allowed = set(acl_scope.get("document_ids") or [])
    kept = []
    for candidate in candidates or []:
        doc_id = resolve_doc_id(candidate)
        if mode == "allow":
            if doc_id and doc_id in allowed:
                kept.append(candidate)
        else:
            if doc_id and doc_id not in allowed:
                kept.append(candidate)
    return kept
