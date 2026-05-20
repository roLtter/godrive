package worker

import (
	"context"
	"fmt"
	"time"

	"cloudstore/backend/internal/config"
	"cloudstore/backend/internal/db/postgres"
	"cloudstore/backend/internal/logger"
	minioClient "cloudstore/backend/internal/storage/minio"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// DeletedFilesCleanup periodically removes MinIO objects for soft-deleted files and hard-deletes rows.
type DeletedFilesCleanup struct {
	db       *postgres.Client
	storage  *minioClient.Client
	interval time.Duration
	batch    int
	minAge   time.Duration
}

// NewDeletedFilesCleanup builds a worker from config (disabled if interval is zero).
func NewDeletedFilesCleanup(db *postgres.Client, storage *minioClient.Client, cfg config.Config) *DeletedFilesCleanup {
	return &DeletedFilesCleanup{
		db:       db,
		storage:  storage,
		interval: time.Duration(cfg.WorkerCleanupIntervalSec) * time.Second,
		batch:    cfg.WorkerCleanupBatch,
		minAge:   time.Duration(cfg.TrashMinAgeMinutes) * time.Minute,
	}
}

// Run blocks until ctx is cancelled. Runs the first purge immediately, then on each tick.
func (w *DeletedFilesCleanup) Run(ctx context.Context) {
	if w.interval <= 0 || w.batch <= 0 {
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		n, err := w.purgeBatch(ctx)
		if err != nil {
			logger.L().Warn("deleted files cleanup batch failed", zap.Error(err))
		} else if n > 0 {
			logger.L().Info("deleted files cleanup completed", zap.Int("purged", n))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *DeletedFilesCleanup) purgeBatch(ctx context.Context) (int, error) {
	cutoff := time.Now().UTC().Add(-w.minAge)

	const query = `
		SELECT id, user_id, s3_key, size
		FROM files
		WHERE deleted_at IS NOT NULL
		  AND deleted_at <= $1
		ORDER BY deleted_at ASC
		LIMIT $2
	`
	rows, err := w.db.QueryContext(ctx, query, cutoff, w.batch)
	if err != nil {
		return 0, fmt.Errorf("select trashed files: %w", err)
	}
	defer rows.Close()

	purged := 0
	for rows.Next() {
		var (
			id     int64
			userID string
			s3Key  string
			size   int64
		)
		if err := rows.Scan(&id, &userID, &s3Key, &size); err != nil {
			return purged, fmt.Errorf("scan trashed file: %w", err)
		}
		if err := w.purgeOne(ctx, id, userID, s3Key, size); err != nil {
			logger.L().Warn("purge trashed file failed",
				zap.Int64("file_id", id),
				zap.String("s3_key", s3Key),
				zap.Error(err),
			)
			continue
		}
		purged++
	}
	if err := rows.Err(); err != nil {
		return purged, err
	}
	return purged, nil
}

func (w *DeletedFilesCleanup) purgeOne(ctx context.Context, fileID int64, userID, s3Key string, size int64) error {
	if err := w.storage.RemoveObject(ctx, s3Key); err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
			logger.L().Info("minio object already missing, continuing with db purge",
				zap.Int64("file_id", fileID),
				zap.String("s3_key", s3Key),
			)
		} else {
			return fmt.Errorf("minio remove: %w", err)
		}
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		DELETE FROM files
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL
	`, fileID, userID)
	if err != nil {
		return fmt.Errorf("delete file row: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("file row not deleted (race or missing)")
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET storage_used_bytes = GREATEST(0, storage_used_bytes - $1)
		WHERE id = $2
	`, size, userID); err != nil {
		return fmt.Errorf("decrement user storage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit purge: %w", err)
	}
	return nil
}
