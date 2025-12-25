package logger

import (
	"log/slog"
	"os"
)

// New создает новый логгер с JSON форматом и уровнем INFO
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Default возвращает логгер по умолчанию
func Default() *slog.Logger {
	return slog.Default()
}

// SetDefault устанавливает логгер по умолчанию
func SetDefault(l *slog.Logger) {
	slog.SetDefault(l)
}
