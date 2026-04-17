CREATE TABLE shares (
    id BIGSERIAL PRIMARY KEY,
    file_id BIGINT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    password_hash TEXT NULL,
    CONSTRAINT fk_shares_file
        FOREIGN KEY (file_id)
        REFERENCES files (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_shares_file_id ON shares (file_id);
CREATE INDEX idx_shares_token ON shares (token);
