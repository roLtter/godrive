ALTER TABLE shares
    DROP COLUMN IF EXISTS last_accessed_at,
    DROP COLUMN IF EXISTS download_count;
