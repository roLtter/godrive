ALTER TABLE users
    ADD COLUMN storage_quota_bytes BIGINT NOT NULL DEFAULT 5368709120,
    ADD COLUMN storage_used_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE users
    ADD CONSTRAINT chk_users_storage_used_non_negative CHECK (storage_used_bytes >= 0);

ALTER TABLE users
    ADD CONSTRAINT chk_users_storage_quota_positive CHECK (storage_quota_bytes > 0);
