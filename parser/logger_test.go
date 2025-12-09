package parser

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
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
		if !ok {
			t.Error("With should return NopLogger")
		}
	})
}

func TestSlogAdapter(t *testing.T) {
	t.Run("implements Logger interface", func(t *testing.T) {
		var _ Logger = (*SlogAdapter)(nil)
	})

	t.Run("NewSlogAdapter with nil uses default", func(t *testing.T) {
		adapter := NewSlogAdapter(nil)
		if adapter.logger == nil {
			t.Error("adapter.logger should not be nil")
		}
	})

	t.Run("NewSlogAdapter with custom logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Debug("debug message", "key", "value")
		if !strings.Contains(buf.String(), "debug message") {
			t.Errorf("expected buffer to contain 'debug message', got: %s", buf.String())
		}
	})

	t.Run("Debug logs at debug level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Debug("test debug", "foo", "bar")
		output := buf.String()
		if !strings.Contains(output, "DEBUG") {
			t.Errorf("expected DEBUG level, got: %s", output)
		}
		if !strings.Contains(output, "foo=bar") {
			t.Errorf("expected foo=bar attribute, got: %s", output)
		}
	})

	t.Run("Info logs at info level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Info("test info", "count", 42)
		output := buf.String()
		if !strings.Contains(output, "INFO") {
			t.Errorf("expected INFO level, got: %s", output)
		}
		if !strings.Contains(output, "count=42") {
			t.Errorf("expected count=42 attribute, got: %s", output)
		}
	})

	t.Run("Warn logs at warn level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Warn("test warn", "problem", "something")
		output := buf.String()
		if !strings.Contains(output, "WARN") {
			t.Errorf("expected WARN level, got: %s", output)
		}
	})

	t.Run("Error logs at error level", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		adapter.Error("test error", "err", "failed")
		output := buf.String()
		if !strings.Contains(output, "ERROR") {
			t.Errorf("expected ERROR level, got: %s", output)
		}
	})

	t.Run("With adds attributes", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		adapter := NewSlogAdapter(logger)

		withAdapter := adapter.With("component", "parser")
		withAdapter.Debug("test with", "extra", "data")
		output := buf.String()
		if !strings.Contains(output, "component=parser") {
			t.Errorf("expected component=parser attribute, got: %s", output)
		}
		if !strings.Contains(output, "extra=data") {
			t.Errorf("expected extra=data attribute, got: %s", output)
		}
	})

	t.Run("With returns new SlogAdapter", func(t *testing.T) {
		adapter := NewSlogAdapter(nil)
		withAdapter := adapter.With("key", "value")
		_, ok := withAdapter.(*SlogAdapter)
		if !ok {
			t.Error("With should return *SlogAdapter")
		}
	})
}

func TestContextLogger(t *testing.T) {
	t.Run("implements Logger interface", func(t *testing.T) {
		var _ Logger = (*ContextLogger)(nil)
	})

	t.Run("NewContextLogger stores context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey("test"), "value")
		logger := NewContextLogger(NopLogger{}, ctx)
		if logger.Context() != ctx {
			t.Error("Context() should return the stored context")
		}
	})

	t.Run("Debug delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Debug("debug via context", "key", "val")
		if !strings.Contains(buf.String(), "debug via context") {
			t.Errorf("expected message in output, got: %s", buf.String())
		}
	})

	t.Run("Info delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Info("info via context")
		if !strings.Contains(buf.String(), "info via context") {
			t.Errorf("expected message in output, got: %s", buf.String())
		}
	})

	t.Run("Warn delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Warn("warn via context")
		if !strings.Contains(buf.String(), "warn via context") {
			t.Errorf("expected message in output, got: %s", buf.String())
		}
	})

	t.Run("Error delegates to wrapped logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
		slogger := slog.New(handler)
		adapter := NewSlogAdapter(slogger)
		ctxLogger := NewContextLogger(adapter, context.Background())

		ctxLogger.Error("error via context")
		if !strings.Contains(buf.String(), "error via context") {
			t.Errorf("expected message in output, got: %s", buf.String())
		}
	})

	t.Run("With preserves context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey("req_id"), "123")
		ctxLogger := NewContextLogger(NopLogger{}, ctx)

		withLogger := ctxLogger.With("key", "value")
		ctxLogger2, ok := withLogger.(*ContextLogger)
		if !ok {
			t.Fatal("With should return *ContextLogger")
		}
		if ctxLogger2.Context() != ctx {
			t.Error("With should preserve context")
		}
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
		if !strings.Contains(output, "package=parser") {
			t.Errorf("expected package=parser, got: %s", output)
		}
		if !strings.Contains(output, "operation=resolve") {
			t.Errorf("expected operation=resolve, got: %s", output)
		}
		if !strings.Contains(output, "ref=#/schemas/Pet") {
			t.Errorf("expected ref=#/schemas/Pet, got: %s", output)
		}
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
		if !strings.Contains(output, "component=parser") {
			t.Errorf("expected component=parser, got: %s", output)
		}
		if !strings.Contains(output, "component=validator") {
			t.Errorf("expected component=validator, got: %s", output)
		}
	})
}
