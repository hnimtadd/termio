package logger

import (
	"io"
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Options struct {
	Buffer io.Writer
	Level  Level
	Type   Type
}

var DefaultLogger = New(Options{os.Stdout, DefaultLevel, TypeText})

type logger struct {
	buffer io.Writer
	*slog.Logger
}

func New(opts Options) Logger {
	var handler slog.Handler
	switch opts.Type {
	case TypeJSON:
		handler = slog.NewJSONHandler(opts.Buffer, &slog.HandlerOptions{
			Level: levels[opts.Level],
		})
	case TypeText:
		fallthrough
	default:
		handler = slog.NewTextHandler(opts.Buffer, &slog.HandlerOptions{
			Level: levels[opts.Level],
		})
	}
	return &logger{
		Logger: slog.New(handler),
	}
}
