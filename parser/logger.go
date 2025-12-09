package parser

import (
	"context"
	"log/slog"
)

// Logger is the interface that oastools uses for structured logging.
//
// The interface is designed to be minimal yet compatible with popular logging
// libraries including log/slog, zap, and zerolog. It uses variadic key-value
// pairs for structured attributes, following the same convention as log/slog.
//
// Implementations should treat attrs as alternating key-value pairs:
//
//	logger.Debug("resolved reference", "ref", "#/components/schemas/Pet", "depth", 3)
//
// Keys should be strings, and values can be any type that the underlying
// logger can serialize.
//
// # Usage with log/slog
//
// Use [NewSlogAdapter] to wrap a standard library slog.Logger:
//
//	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
//	slogger := slog.New(handler)
//	logger := parser.NewSlogAdapter(slogger)
//
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("api.yaml"),
//	    parser.WithLogger(logger),
//	)
//
// # Usage with zap
//
// Create a simple adapter implementing the Logger interface:
//
//	type ZapAdapter struct {
//	    logger *zap.SugaredLogger
//	}
//
//	func (z *ZapAdapter) Debug(msg string, attrs ...any) { z.logger.Debugw(msg, attrs...) }
//	func (z *ZapAdapter) Info(msg string, attrs ...any)  { z.logger.Infow(msg, attrs...) }
//	func (z *ZapAdapter) Warn(msg string, attrs ...any)  { z.logger.Warnw(msg, attrs...) }
//	func (z *ZapAdapter) Error(msg string, attrs ...any) { z.logger.Errorw(msg, attrs...) }
//	func (z *ZapAdapter) With(attrs ...any) parser.Logger {
//	    return &ZapAdapter{logger: z.logger.With(attrs...)}
//	}
//
// # Usage with zerolog
//
// Create a simple adapter implementing the Logger interface:
//
//	type ZerologAdapter struct {
//	    logger zerolog.Logger
//	}
//
//	func (z *ZerologAdapter) Debug(msg string, attrs ...any) {
//	    e := z.logger.Debug()
//	    for i := 0; i < len(attrs)-1; i += 2 {
//	        e = e.Interface(fmt.Sprint(attrs[i]), attrs[i+1])
//	    }
//	    e.Msg(msg)
//	}
//	// ... implement other methods similarly
type Logger interface {
	// Debug logs at debug level. Use for detailed diagnostic information.
	Debug(msg string, attrs ...any)

	// Info logs at info level. Use for general operational information.
	Info(msg string, attrs ...any)

	// Warn logs at warn level. Use for potentially harmful situations.
	Warn(msg string, attrs ...any)

	// Error logs at error level. Use for error conditions.
	Error(msg string, attrs ...any)

	// With returns a new Logger with the given attributes prepended to every log.
	// This is useful for adding context that applies to multiple log calls.
	With(attrs ...any) Logger
}

// NopLogger is a no-op logger that discards all output.
// It is the default logger used when no logger is configured.
type NopLogger struct{}

// Debug implements Logger.
func (NopLogger) Debug(_ string, _ ...any) {}

// Info implements Logger.
func (NopLogger) Info(_ string, _ ...any) {}

// Warn implements Logger.
func (NopLogger) Warn(_ string, _ ...any) {}

// Error implements Logger.
func (NopLogger) Error(_ string, _ ...any) {}

// With implements Logger.
func (n NopLogger) With(_ ...any) Logger { return n }

// Ensure NopLogger implements Logger at compile time.
var _ Logger = NopLogger{}

// SlogAdapter wraps a *slog.Logger to implement the Logger interface.
// This allows using the standard library's slog package with oastools.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new SlogAdapter from a *slog.Logger.
// If logger is nil, slog.Default() is used.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogAdapter{logger: logger}
}

// Debug implements Logger.
func (s *SlogAdapter) Debug(msg string, attrs ...any) {
	s.logger.Debug(msg, attrs...)
}

// Info implements Logger.
func (s *SlogAdapter) Info(msg string, attrs ...any) {
	s.logger.Info(msg, attrs...)
}

// Warn implements Logger.
func (s *SlogAdapter) Warn(msg string, attrs ...any) {
	s.logger.Warn(msg, attrs...)
}

// Error implements Logger.
func (s *SlogAdapter) Error(msg string, attrs ...any) {
	s.logger.Error(msg, attrs...)
}

// With implements Logger.
func (s *SlogAdapter) With(attrs ...any) Logger {
	return &SlogAdapter{logger: s.logger.With(attrs...)}
}

// Ensure SlogAdapter implements Logger at compile time.
var _ Logger = (*SlogAdapter)(nil)

// ContextLogger wraps a Logger to include context in all operations.
// This is useful for passing request-scoped values through the logging pipeline.
type ContextLogger struct {
	logger Logger
	ctx    context.Context
}

// NewContextLogger creates a new ContextLogger.
func NewContextLogger(logger Logger, ctx context.Context) *ContextLogger {
	return &ContextLogger{logger: logger, ctx: ctx}
}

// Debug implements Logger.
func (c *ContextLogger) Debug(msg string, attrs ...any) {
	c.logger.Debug(msg, attrs...)
}

// Info implements Logger.
func (c *ContextLogger) Info(msg string, attrs ...any) {
	c.logger.Info(msg, attrs...)
}

// Warn implements Logger.
func (c *ContextLogger) Warn(msg string, attrs ...any) {
	c.logger.Warn(msg, attrs...)
}

// Error implements Logger.
func (c *ContextLogger) Error(msg string, attrs ...any) {
	c.logger.Error(msg, attrs...)
}

// With implements Logger.
func (c *ContextLogger) With(attrs ...any) Logger {
	return &ContextLogger{
		logger: c.logger.With(attrs...),
		ctx:    c.ctx,
	}
}

// Context returns the context associated with this logger.
func (c *ContextLogger) Context() context.Context {
	return c.ctx
}

// Ensure ContextLogger implements Logger at compile time.
var _ Logger = (*ContextLogger)(nil)
