CREATE TABLE files (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    folder_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    size BIGINT NOT NULL,
    mime TEXT NOT NULL,
    s3_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_files_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_files_folder
        FOREIGN KEY (folder_id)
        REFERENCES folders (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_files_user_id ON files (user_id);
