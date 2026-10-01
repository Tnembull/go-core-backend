package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(env string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		AddSource: env == "development", // show file:line only in development
	}

	if env == "development" {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)
}
