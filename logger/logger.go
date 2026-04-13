package logger

import (
	"io"
	"log/slog"
	"time"
)

func NewLogger(w io.Writer) *slog.Logger {
	textHandler := slog.NewTextHandler(w, nil)
	return slog.New(textHandler)
}

func LogError(s *slog.Logger, msg string) {
	slog.Error(
		msg,
		slog.Time("timestamp", now()),
	)
}

func LogInfo(s *slog.Logger, msg string) {
	slog.Info(
		msg,
		slog.Time("timestamp", now()),
	)
}

func now() time.Time {
	return time.Now().UTC()
}
