CREATE TABLE folders (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    parent_id BIGINT NULL,
    name TEXT NOT NULL,
    CONSTRAINT fk_folders_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_folders_parent
        FOREIGN KEY (parent_id)
        REFERENCES folders (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_folders_user_id ON folders (user_id);
