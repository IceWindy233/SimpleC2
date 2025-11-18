package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// globalLogger is the global zap logger instance
	globalLogger *zap.Logger
	// sugarLogger is the sugared version of the global logger for convenience
	sugarLogger *zap.SugaredLogger
)

// Init initializes the global logger with the specified log level
func Init(level string) error {
	// Parse log level
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create config
	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: true,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderConfig,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		InitialFields:     nil,
	}

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return err
	}

	globalLogger = logger
	sugarLogger = globalLogger.Sugar()
	return nil
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Debug logs a debug message with the given key-value pairs
func Debug(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Debug(msg, fields...)
	}
}

// Debugf logs a formatted debug message
func Debugf(template string, args ...interface{}) {
	if sugarLogger != nil {
		sugarLogger.Debugf(template, args...)
	}
}

// Info logs an info message with the given key-value pairs
func Info(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Info(msg, fields...)
	}
}

// Infof logs a formatted info message
func Infof(template string, args ...interface{}) {
	if sugarLogger != nil {
		sugarLogger.Infof(template, args...)
	}
}

// Warn logs a warning message with the given key-value pairs
func Warn(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Warn(msg, fields...)
	}
}

// Warnf logs a formatted warning message
func Warnf(template string, args ...interface{}) {
	if sugarLogger != nil {
		sugarLogger.Warnf(template, args...)
	}
}

// Error logs an error message with the given key-value pairs
func Error(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Error(msg, fields...)
	}
}

// Errorf logs a formatted error message
func Errorf(template string, args ...interface{}) {
	if sugarLogger != nil {
		sugarLogger.Errorf(template, args...)
	}
}

// Fatal logs a fatal message and then calls os.Exit(1)
func Fatal(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Fatal(msg, fields...)
	}
}

// Fatalf logs a formatted fatal message and then calls os.Exit(1)
func Fatalf(template string, args ...interface{}) {
	if sugarLogger != nil {
		sugarLogger.Fatalf(template, args...)
	}
}

// WithFields creates a child logger with the given fields
func WithFields(fields ...zap.Field) *zap.Logger {
	if globalLogger != nil {
		return globalLogger.With(fields...)
	}
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	return globalLogger
}

// GetSugar returns the global sugared logger instance
func GetSugar() *zap.SugaredLogger {
	return sugarLogger
}
