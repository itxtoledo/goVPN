// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/logger/logger.go
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
func Init(level LogLevel) {
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
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Create JSON encoder
		// jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

		// Create console encoder for terminal output
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

		// Convert log level
		var zapLevel zapcore.Level
		switch level {
		case DebugLevel:
			zapLevel = zapcore.DebugLevel
		case InfoLevel:
			zapLevel = zapcore.InfoLevel
		case WarnLevel:
			zapLevel = zapcore.WarnLevel
		case ErrorLevel:
			zapLevel = zapcore.ErrorLevel
		case FatalLevel:
			zapLevel = zapcore.FatalLevel
		default:
			zapLevel = zapcore.InfoLevel
		}

		// Create log file
		// logFile, _ := os.Create("./server.log")

		// Create core for both console and file output
		core := zapcore.NewTee(
			// zapcore.NewCore(jsonEncoder, zapcore.AddSync(logFile), zapLevel),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel),
		)

		// Create logger
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		sugar = logger.Sugar()

		// Log initialization
		sugar.Infow("Logger initialized",
			"level", level,
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
