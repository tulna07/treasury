// Package logger provides structured logging with OpenTelemetry trace correlation.
package logger

import (
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
)

type loggerKey struct{}

// New creates a new Zap logger configured for the given environment.
func New(env string) (*zap.Logger, error) {
	var cfg zap.Config
	if strings.ToLower(env) == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Override log level from env
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		var zapLvl zapcore.Level
		if err := zapLvl.UnmarshalText([]byte(lvl)); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(zapLvl)
		}
	}

	return cfg.Build(zap.AddCallerSkip(0))
}

// WithContext stores a logger in the context.
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext retrieves the logger from context, adding trace fields if available.
func FromContext(ctx context.Context) *zap.Logger {
	logger, _ := ctx.Value(loggerKey{}).(*zap.Logger)
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Enrich with trace context
	fields := make([]zap.Field, 0, 4)
	if traceID := ctxutil.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if spanID := ctxutil.GetSpanID(ctx); spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}
	if reqID := ctxutil.GetRequestID(ctx); reqID != "" {
		fields = append(fields, zap.String("request_id", reqID))
	}
	if userID := ctxutil.GetUserID(ctx); userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	if len(fields) > 0 {
		return logger.With(fields...)
	}
	return logger
}
