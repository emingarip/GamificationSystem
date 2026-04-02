package metrics

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Logger provides structured logging with request ID tracing
type Logger struct {
	logger zerolog.Logger
}

// Global logger instance
var globalLogger *Logger

// RequestIDKey is the context key for request ID
const RequestIDKey = "request_id"

// Init initializes the global logger
func Init(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	zerolog.TimeFieldFormat = time.RFC3339
	logger := zerolog.New(output).With().Timestamp().Caller().Logger()

	switch level {
	case LevelDebug:
		logger = logger.Level(zerolog.DebugLevel)
	case LevelInfo:
		logger = logger.Level(zerolog.InfoLevel)
	case LevelWarn:
		logger = logger.Level(zerolog.WarnLevel)
	case LevelError:
		logger = logger.Level(zerolog.ErrorLevel)
	default:
		logger = logger.Level(zerolog.InfoLevel)
	}

	globalLogger = &Logger{logger: logger}
	return globalLogger
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default settings if not initialized
		return Init(LevelInfo, os.Stdout)
	}
	return globalLogger
}

// WithContext creates a logger with request ID from context
func (l *Logger) WithContext(ctx context.Context) *zerolog.Logger {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		logger := l.logger.With().Str("request_id", reqID).Logger()
		return &logger
	}
	return &l.logger
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string) {
	l.WithContext(ctx).Debug().Msg(msg)
}

// Debugf logs a debug message with format
func (l *Logger) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.WithContext(ctx).Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string) {
	l.WithContext(ctx).Info().Msg(msg)
}

// Infof logs an info message with format
func (l *Logger) Infof(ctx context.Context, format string, args ...interface{}) {
	l.WithContext(ctx).Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string) {
	l.WithContext(ctx).Warn().Msg(msg)
}

// Warnf logs a warning message with format
func (l *Logger) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.WithContext(ctx).Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, msg string) {
	l.WithContext(ctx).Error().Msg(msg)
}

// Errorf logs an error message with format
func (l *Logger) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.WithContext(ctx).Error().Msgf(format, args...)
}

// With adds contextual fields to the logger
func (l *Logger) With(fields map[string]interface{}) *Logger {
	logger := l.logger.With()
	for k, v := range fields {
		logger = logger.Interface(k, v)
	}
	return &Logger{logger: logger.Logger()}
}

// API request logging helpers

// LogAPIRequest logs an API request
func (l *Logger) LogAPIRequest(ctx context.Context, method, path, clientIP string, statusCode int, durationMs float64) {
	l.WithContext(ctx).Info().
		Str("method", method).
		Str("path", path).
		Str("client_ip", clientIP).
		Int("status_code", statusCode).
		Float64("duration_ms", durationMs).
		Msg("API request")
}

// LogRuleEvaluation logs a rule evaluation
func (l *Logger) LogRuleEvaluation(ctx context.Context, ruleID, ruleName string, matched bool, evalTimeMs float64) {
	lvl := zerolog.InfoLevel
	if !matched {
		lvl = zerolog.DebugLevel
	}

	l.WithContext(ctx).WithLevel(lvl).
		Str("rule_id", ruleID).
		Str("rule_name", ruleName).
		Bool("matched", matched).
		Float64("eval_time_ms", evalTimeMs).
		Msg("Rule evaluation")
}

// LogBadgeGrant logs a badge grant event
func (l *Logger) LogBadgeGrant(ctx context.Context, userID, badgeID, badgeName string, points int) {
	l.WithContext(ctx).Info().
		Str("user_id", userID).
		Str("badge_id", badgeID).
		Str("badge_name", badgeName).
		Int("points", points).
		Msg("Badge granted")
}

// LogWebSocketConnection logs a WebSocket connection event
func (l *Logger) LogWebSocketConnection(ctx context.Context, userID string, connected bool) {
	lvl := zerolog.InfoLevel
	if !connected {
		lvl = zerolog.WarnLevel
	}

	l.WithContext(ctx).WithLevel(lvl).
		Str("user_id", userID).
		Bool("connected", connected).
		Msg("WebSocket connection")
}

// LogKafkaMessage logs a Kafka message processed
func (l *Logger) LogKafkaMessage(ctx context.Context, topic, partition string, offset int64, eventType string) {
	l.WithContext(ctx).Info().
		Str("topic", topic).
		Str("partition", partition).
		Int64("offset", offset).
		Str("event_type", eventType).
		Msg("Kafka message processed")
}

// Helper functions that work with global logger

// Debug logs using global logger
func Debug(ctx context.Context, msg string) {
	GetLogger().Debug(ctx, msg)
}

// Info logs using global logger
func Info(ctx context.Context, msg string) {
	GetLogger().Info(ctx, msg)
}

// Warn logs using global logger
func Warn(ctx context.Context, msg string) {
	GetLogger().Warn(ctx, msg)
}

// Error logs using global logger
func Error(ctx context.Context, msg string) {
	GetLogger().Error(ctx, msg)
}

// AddRequestID adds a request ID to context
func AddRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
