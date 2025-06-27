package logger

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogSettings struct {
	Filename   string
	Level      string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

var LogLevel = new(slog.LevelVar)

func InitLogger(config *LogSettings) {
	log := lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		LocalTime:  true,
	}

	LogLevel.Set(GetLogLevel(config.Level))
	var logHandler *slog.Logger
	if isLocal, _ := strconv.ParseBool(os.Getenv("LOCAL_ENV")); isLocal {
		logHandler = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      LogLevel, // 设置最低日志级别
			AddSource:  true,
			TimeFormat: time.DateTime,
		}))
	} else {
		logHandler = slog.New(slog.NewJSONHandler(&log, &slog.HandlerOptions{
			Level:     LogLevel,
			AddSource: true,
		}))
	}

	slog.SetDefault(logHandler)
}

func NewLogger(opts *LogSettings) (*slog.Logger, error) {
	log := lumberjack.Logger{
		Filename:   opts.Filename,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAge,
		LocalTime:  true,
	}

	LogLevel.Set(GetLogLevel(opts.Level))
	var logHandler *slog.Logger
	if isLocal, _ := strconv.ParseBool(os.Getenv("LOCAL_ENV")); isLocal {
		logHandler = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      LogLevel, // 设置最低日志级别
			AddSource:  true,
			TimeFormat: time.DateTime,
		}))
	} else {
		logHandler = slog.New(slog.NewJSONHandler(&log, &slog.HandlerOptions{
			Level:     LogLevel,
			AddSource: true,
		}))
	}

	slog.SetDefault(logHandler)
	return logHandler, nil
}

func GetLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	return slog.LevelInfo
}
