package logger

import (
	"log/slog"
	"os"
)

func Init(env string) *slog.Logger {
	var logHandler slog.Handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	if env == "production" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	l := slog.New(logHandler)
	slog.SetDefault(l)
	return l
}
