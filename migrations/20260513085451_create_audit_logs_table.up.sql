CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    actor_id UUID,
    actor_name TEXT,

    action TEXT NOT NULL,
    resource TEXT NOT NULL,

    user_id UUID,

    metadata JSONB DEFAULT '{}'::jsonb,

    ip_address TEXT,
    user_agent TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);