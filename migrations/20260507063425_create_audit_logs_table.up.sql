CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    admin_id UUID NOT NULL,
    action TEXT NOT NULL,
    target_user_id UUID,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);