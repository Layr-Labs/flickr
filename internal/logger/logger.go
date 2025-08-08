package logger

import (
	"context"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for logging
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	
	With(fields ...zap.Field) Logger
	Sugar() *zap.SugaredLogger
}

// logger wraps zap.Logger to implement our interface
type logger struct {
	*zap.Logger
}

// With returns a new logger with additional fields
func (l *logger) With(fields ...zap.Field) Logger {
	return &logger{Logger: l.Logger.With(fields...)}
}

var globalLogger Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(verbose bool) {
	InitGlobalLoggerWithWriter(verbose, os.Stderr)
}

// InitGlobalLoggerWithWriter initializes the global logger with a custom writer
func InitGlobalLoggerWithWriter(verbose bool, writer io.Writer) {
	var cfg zap.Config
	if verbose {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.TimeKey = ""
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = ""
		cfg.EncoderConfig.EncodeTime = nil
	}

	// Use the provided writer
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg.EncoderConfig),
		zapcore.AddSync(writer),
		cfg.Level,
	)

	zapLogger := zap.New(core)
	globalLogger = &logger{Logger: zapLogger}
}

// GetLogger returns the global logger
func GetLogger() Logger {
	if globalLogger == nil {
		InitGlobalLogger(false)
	}
	return globalLogger
}

// NewLoggerWithWriter creates a new logger with a custom writer
func NewLoggerWithWriter(verbose bool, writer io.Writer) Logger {
	var cfg zap.Config
	if verbose {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.TimeKey = ""
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = ""
		cfg.EncoderConfig.EncodeTime = nil
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg.EncoderConfig),
		zapcore.AddSync(writer),
		cfg.Level,
	)

	zapLogger := zap.New(core)
	return &logger{Logger: zapLogger}
}

// FromContext retrieves the logger from context
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value("logger").(Logger); ok {
		return l
	}
	return GetLogger()
}