DROP TABLE IF EXISTS audit_logs;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    admin_id UUID NOT NULL,
    action TEXT NOT NULL,
    target_user_id UUID,
    metadata JSONB,

    reason TEXT,

    created_at TIMESTAMP DEFAULT NOW()
);