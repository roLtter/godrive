ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_storage_quota_positive;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_storage_used_non_negative;
ALTER TABLE users DROP COLUMN IF EXISTS storage_used_bytes;
ALTER TABLE users DROP COLUMN IF EXISTS storage_quota_bytes;
