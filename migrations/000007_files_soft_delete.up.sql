ALTER TABLE files
    ADD COLUMN deleted_at TIMESTAMPTZ NULL;

CREATE INDEX idx_files_user_deleted ON files (user_id, deleted_at);
