
CREATE TYPE status AS ENUM ('active', 'inactive', 'suspended', 'banned');

ALTER TABLE users
DROP COLUMN is_active,
ADD COLUMN status status NOT NULL DEFAULT 'active';