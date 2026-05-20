package files

import (
	"os"
	"testing"

	"cloudstore/backend/internal/logger"
)

func TestMain(m *testing.M) {
	_, _ = logger.Init("dev")
	os.Exit(m.Run())
}
