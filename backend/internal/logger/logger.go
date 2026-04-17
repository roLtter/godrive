package logger

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var (
	global     *zap.Logger
	globalOnce sync.Once
	initErr    error
)

// Init configures a singleton zap logger.
// APP_ENV=prod enables JSON output, everything else uses development output.
func Init(appEnv string) (*zap.Logger, error) {
	globalOnce.Do(func() {
		env := strings.ToLower(strings.TrimSpace(appEnv))
		if env == "prod" || env == "production" {
			global, initErr = zap.NewProduction()
			return
		}
		global, initErr = zap.NewDevelopment()
	})

	if initErr != nil {
		return nil, fmt.Errorf("init logger: %w", initErr)
	}
	return global, nil
}

// L returns initialized global logger.
func L() *zap.Logger {
	if global == nil {
		panic("logger is not initialized: call logger.Init first")
	}
	return global
}

// Sync flushes buffered logger data.
func Sync() error {
	if global == nil {
		return nil
	}
	return global.Sync()
}
