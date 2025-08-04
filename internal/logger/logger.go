// Package logger provides a standardized logging interface for the application
package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// contextKey is a private type for context keys
type contextKey int

const (
	// loggerKey is the key for the logger in the context
	loggerKey contextKey = iota
	// requestIDKey is the key for the request ID in the context
	requestIDKey
)

// LogLevel represents log levels
type LogLevel string

// Log levels
const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// Config holds logger configuration
type Config struct {
	// Level is the log level: debug, info, warn, error
	Level LogLevel
	// Format can be "json" or "console"
	Format string
	// ConsoleTimeFormat is the time format for console output
	ConsoleTimeFormat string
	// CallerInfo determines whether to include caller information
	CallerInfo bool
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:             LogInfo,
		Format:            "console",
		ConsoleTimeFormat: time.RFC3339,
		CallerInfo:        true,
	}
}

// EnvConfig loads logger configuration from environment variables
func EnvConfig() Config {
	config := DefaultConfig()

	// Get log level from environment
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = LogLevel(level)
	}

	// Get log format from environment
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = format
	}

	return config
}

// Setup configures the global logger
func Setup(config Config) {
	// Set log level
	var level zerolog.Level
	switch config.Level {
	case LogDebug:
		level = zerolog.DebugLevel
	case LogInfo:
		level = zerolog.InfoLevel
	case LogWarn:
		level = zerolog.WarnLevel
	case LogError:
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	// Configure output format
	if config.Format == "console" {
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: config.ConsoleTimeFormat,
		}
		log.Logger = log.Output(output)
	}

	// Configure caller info
	if config.CallerInfo {
		log.Logger = log.With().Caller().Logger()
	}
}

// FromContext returns the logger from the context or the default logger if not found
func FromContext(ctx context.Context) zerolog.Logger {
	if ctx == nil {
		return log.Logger
	}

	if logger, ok := ctx.Value(loggerKey).(zerolog.Logger); ok {
		return logger
	}

	return log.Logger
}

// WithContext adds a logger to the context
func WithContext(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// WithRequestID adds a request ID to the context and logger
func WithRequestID(ctx context.Context, requestID string) (context.Context, zerolog.Logger) {
	logger := FromContext(ctx).With().Str("request_id", requestID).Logger()
	ctx = WithContext(ctx, logger)
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	return ctx, logger
}

// WithField adds a field to the logger in the context
func WithField(ctx context.Context, key string, value interface{}) (context.Context, zerolog.Logger) {
	logger := FromContext(ctx).With().Interface(key, value).Logger()
	return WithContext(ctx, logger), logger
}

// WithFields adds multiple fields to the logger in the context
func WithFields(ctx context.Context, fields map[string]interface{}) (context.Context, zerolog.Logger) {
	loggerCtx := FromContext(ctx).With()
	for k, v := range fields {
		loggerCtx = loggerCtx.Interface(k, v)
	}
	logger := loggerCtx.Logger()
	return WithContext(ctx, logger), logger
}

// Debug logs a debug message
func Debug(ctx context.Context, message string) {
	logger := FromContext(ctx)
	logger.Debug().Msg(message)
}

// Info logs an info message
func Info(ctx context.Context, message string) {
	logger := FromContext(ctx)
	logger.Info().Msg(message)
}

// Warn logs a warning message
func Warn(ctx context.Context, message string) {
	logger := FromContext(ctx)
	logger.Warn().Msg(message)
}

// Error logs an error message
func Error(ctx context.Context, err error, message string) {
	logger := FromContext(ctx)
	logger.Error().Err(err).Msg(message)
}

// Fatal logs a fatal message and exits
func Fatal(ctx context.Context, err error, message string) {
	logger := FromContext(ctx)
	logger.Fatal().Err(err).Msg(message)
}
