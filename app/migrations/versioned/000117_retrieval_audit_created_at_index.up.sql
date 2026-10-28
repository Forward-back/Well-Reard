-- 000117: index for retrieval-audit retention sweeps.
CREATE INDEX IF NOT EXISTS idx_retrieval_audit_created_at
    ON retrieval_audit_logs (created_at);
