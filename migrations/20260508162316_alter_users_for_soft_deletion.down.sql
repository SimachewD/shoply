-- 1. Remove the partial indexes
DROP INDEX IF EXISTS users_email_unique_active;
DROP INDEX IF EXISTS idx_users_deleted_at;

-- 2. Remove the soft-delete columns
ALTER TABLE users 
    DROP COLUMN IF EXISTS deleted_at, 
    DROP COLUMN IF EXISTS deleted_by;

-- 3. Restore the original UNIQUE CONSTRAINT
-- We use ADD CONSTRAINT so it behaves exactly like the original
ALTER TABLE users 
    ADD CONSTRAINT users_email_key UNIQUE (email);
