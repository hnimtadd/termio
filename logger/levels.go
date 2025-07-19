package logger

import "log/slog"

type Level int

const (
	InfoLevel Level = iota
	DebugLevel
	WarnLevel
	ErrorLevel
	DefaultLevel Level = InfoLevel
)

var levels = map[Level]slog.Level{
	DebugLevel: slog.LevelDebug,
	InfoLevel:  slog.LevelInfo,
	WarnLevel:  slog.LevelWarn,
	ErrorLevel: slog.LevelError,
}
