package parser

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contextKey is a custom type for context keys to satisfy staticcheck SA1029
type contextKey string

func TestNopLogger(t *testing.T) {
	t.Run("implements Logger interface", func(t *testing.T) {
		var _ Logger = NopLogger{}
	})

	t.Run("Debug does nothing", func(t *testing.T) {
		l := NopLogger{}
		// Should not panic
		l.Debug("test message", "key", "value")
	})

	t.Run("Info does nothing", func(t *testing.T) {
		l := NopLogger{}
		l.Info("test message", "key", "value")
	})

	t.Run("Warn does nothing", func(t *testing.T) {
		l := NopLogger{}
		l.Warn("test message", "key", "value")
	})

	t.Run("Error does nothing", func(t *testing.T) {
		l := NopLogger{}
		l.Error("test message", "key", "value")
	})

	t.Run("With returns same NopLogger", func(t *testing.T) {
		l := NopLogger{}
		l2 := l.With("key", "value")
		_, ok := l2.(NopLogger)
		assert.True(t, ok, "With should return NopLogger")
	})
}

func TestSlogAdapter(t *testing.T) {
	t.Run("implements Logger interface", func(t *testing.T) {
		var _ Logger = (*SlogAdapter)(nil)
	})

	t.Run("NewSlogAdapter with nil uses default", func(t *testing.T) {
		adapter := NewSlogAdapter(nil)
		assert.NotNil(t, adapter.logger, "adapter.logger should not be nil")
	})

	t.Run("NewSlogAdapter with custom logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Debug("debug message", "key", "value")
		assert.Contains(t, buf.String(), "debug message")
	})

	t.Run("Debug logs at debug level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Debug("test debug", "foo", "bar")
		output := buf.String()
		assert.Contains(t, output, "DEBUG")
		assert.Contains(t, output, "foo=bar")
	})

	t.Run("Info logs at info level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Info("test info", "count", 42)
		output := buf.String()
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "count=42")
	})

	t.Run("Warn logs at warn level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Warn("test warn", "problem", "something")
		output := buf.String()
		assert.Contains(t, output, "WARN")
	})

	t.Run("Error logs at error level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Error("test error", "err", "failed")
		output := buf.String()
		assert.Contains(t, output, "ERROR")
	})

	t.Run("With adds attributes", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		withAdapter := adapter.With("component", "parser")
		withAdapter.Debug("test with", "extra", "data")
		output := buf.String()
		assert.Contains(t, output, "component=parser")
		assert.Contains(t, output, "extra=data")
	})

	t.Run("With returns new SlogAdapter", func(t *testing.T) {
		adapter := NewSlogAdapter(nil)
		withAdapter := adapter.With("key", "value")
		_, ok := withAdapter.(*SlogAdapter)
		assert.True(t, ok, "With should return *SlogAdapter")
	})
}

func TestContextLogger(t *testing.T) {
	t.Run("implements Logger interface", func(t *testing.T) {
		var _ Logger = (*ContextLogger)(nil)
	})

	t.Run("NewContextLogger stores context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey("test"), "value")
		logger := NewContextLogger(NopLogger{}, ctx)
		assert.Equal(t, ctx, logger.Context(), "Context() should return the stored context")
	})

	t.Run("Debug delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Debug("debug via context", "key", "val")
		assert.Contains(t, buf.String(), "debug via context")
	})

	t.Run("Info delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Info("info via context")
		assert.Contains(t, buf.String(), "info via context")
	})

	t.Run("Warn delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Warn("warn via context")
		assert.Contains(t, buf.String(), "warn via context")
	})

	t.Run("Error delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Error("error via context")
		assert.Contains(t, buf.String(), "error via context")
	})

	t.Run("With preserves context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey("req_id"), "123")
		ctxLogger := NewContextLogger(NopLogger{}, ctx)

		withLogger := ctxLogger.With("key", "value")
		ctxLogger2, ok := withLogger.(*ContextLogger)
		require.True(t, ok, "With should return *ContextLogger")
		assert.Equal(t, ctx, ctxLogger2.Context(), "With should preserve context")
	})
}

func TestLoggerUsagePatterns(t *testing.T) {
	t.Run("chained With calls", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		l := adapter.
			With("package", "parser").
			With("operation", "resolve")
		l.Debug("resolving reference", "ref", "#/schemas/Pet")

		output := buf.String()
		assert.Contains(t, output, "package=parser")
		assert.Contains(t, output, "operation=resolve")
		assert.Contains(t, output, "ref=#/schemas/Pet")
	})

	t.Run("multiple loggers from same base", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		parserLogger := adapter.With("component", "parser")
		validatorLogger := adapter.With("component", "validator")

		parserLogger.Debug("parsing")
		validatorLogger.Debug("validating")

		output := buf.String()
		assert.Contains(t, output, "component=parser")
		assert.Contains(t, output, "component=validator")
	})
}
