-- 1. These MUST run outside a transaction block if your tool wraps migrations in 'BEGIN/COMMIT'
ALTER TYPE status ADD VALUE IF NOT EXISTS 'pending_verification';
ALTER TYPE status ADD VALUE IF NOT EXISTS 'deleted';

-- 2. Modify the table
ALTER TABLE users 
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL, 
    ADD COLUMN IF NOT EXISTS deleted_by UUID NULL;

-- 3. DROP THE CONSTRAINT, NOT THE INDEX
-- If you created the table with 'email VARCHAR UNIQUE', the constraint name is usually 'users_email_key'
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

-- 4. Recreate as partial unique index
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_active 
    ON users(email) 
    WHERE deleted_at IS NULL;

-- 5. Optional performance index
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
