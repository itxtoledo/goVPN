// filepath: /Computers/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/logger/logger.go
package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	once   sync.Once
)

// LogLevel represents the level of logging
type LogLevel string

// Log levels
const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// Init initializes the logger with the specified log level
func Init() {
	once.Do(func() {
		// Create encoder configuration
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseColorLevelEncoder, // Use color for console
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Create console encoder for terminal output
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

		// Always set log level to Debug to capture all messages
		zapLevel := zapcore.DebugLevel

		// Create core for console output
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel),
		)

		// Create logger
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		sugar = logger.Sugar()

		// Log initialization
		sugar.Infow("Logger initialized (all levels to console)",
			"time", time.Now().Format(time.RFC3339),
		)
	})
}

// Debug logs a message at debug level with structured fields
func Debug(msg string, fields ...interface{}) {
	sugar.Debugw(msg, fields...)
}

// Info logs a message at info level with structured fields
func Info(msg string, fields ...interface{}) {
	sugar.Infow(msg, fields...)
}

// Warn logs a message at warn level with structured fields
func Warn(msg string, fields ...interface{}) {
	sugar.Warnw(msg, fields...)
}

// Error logs a message at error level with structured fields
func Error(msg string, fields ...interface{}) {
	sugar.Errorw(msg, fields...)
}

// Fatal logs a message at fatal level with structured fields and exits
func Fatal(msg string, fields ...interface{}) {
	sugar.Fatalw(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return logger.Sync()
}

// GetLogger returns the underlying zap logger
func GetLogger() *zap.Logger {
	return logger
}

// GetSugaredLogger returns the underlying sugared zap logger
func GetSugaredLogger() *zap.SugaredLogger {
	return sugar
}
