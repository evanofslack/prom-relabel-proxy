package proxy

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
)

func NewLogger(level, env string) *slog.Logger {
	loglevel := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		loglevel = slog.LevelDebug
	case "info":
		loglevel = slog.LevelInfo
	case "warn":
		loglevel = slog.LevelWarn
	case "error":
		loglevel = slog.LevelError
	}

	var handler slog.Handler
	if strings.ToLower(env) == "debug" {
		opts := &slog.HandlerOptions{
			Level:     loglevel,
			AddSource: true,
		}
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		opts := &slog.HandlerOptions{
			Level: loglevel,
		}
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)

	buildInfo, _ := debug.ReadBuildInfo()
	logger = logger.With(
		slog.Int("pid", os.Getpid()),
		slog.String("go_version", buildInfo.GoVersion),
	)

	return logger
}
