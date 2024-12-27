package logctx

import (
	"context"
	"github.com/google/uuid"
	"github.com/phsym/console-slog"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log/slog"
	"os"
)

type IdProviderT func() string

func DefaultIdProvider() string {
	return uuid.New().String()
}

var IdProvider IdProviderT = DefaultIdProvider

const LoggerKey = "logger"

type TraceIdKey string

// RequestTraceIdKey is the trace ID key for requests.
const RequestTraceIdKey TraceIdKey = "trace_id"

// JobTraceIdKey is the trace ID key for when we run jobs in the background, like cron jobs.
const JobTraceIdKey TraceIdKey = "job_trace_id"

// ProcessTraceIdKey is the trace ID key for the overall process.
const ProcessTraceIdKey TraceIdKey = "process_trace_id"

// MissingTraceIdKey is the key that will be present to indicate tracing is misconfigured.
const MissingTraceIdKey TraceIdKey = "missing_trace_id"

const SpanIdKey TraceIdKey = "span_id"

func UnconfiguredLogger() *slog.Logger {
	return slog.Default().With("unconfigured_logger", "true")
}

// WithLogger returns a new context that adds a logger which
// can be retrieved with Logger(Context).
func WithLogger(c context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(c, LoggerKey, logger)
}

// WithTracingLogger stiches together WithTraceId and WithLogger.
// It extracts the ActiveTraceId and sets it on the logger.
// In this way you can do WithTracingLogger(WithTraceId(WithLogger(ctx, logger)))
// to get a logger in the context with a trace id,
// and then Logger to get the logger back.
func WithTracingLogger(c context.Context) context.Context {
	logger := Logger(c)
	tkey, trace := ActiveTraceId(c)
	logger = logger.With(string(tkey), trace)
	return context.WithValue(c, LoggerKey, logger)
}

func WithTraceId(c context.Context, key TraceIdKey) context.Context {
	return context.WithValue(c, key, IdProvider())
}

func LoggerOrNil(c context.Context) *slog.Logger {
	logger, _ := c.Value(LoggerKey).(*slog.Logger)
	return logger
}

func Logger(c context.Context) *slog.Logger {
	if logger, ok := c.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	logger := UnconfiguredLogger()
	logger.Warn(
		"Logger called with no logger in context. " +
			"It should always be there to ensure consistent logs from a single logger")
	return logger
}

// ActiveTraceId returns the first valid trace value and type from the given context,
// or MissingTraceIdKey if there is none.
func ActiveTraceId(c context.Context) (TraceIdKey, string) {
	if trace, ok := c.Value(RequestTraceIdKey).(string); ok {
		return RequestTraceIdKey, trace
	}
	if trace, ok := c.Value(JobTraceIdKey).(string); ok {
		return JobTraceIdKey, trace
	}
	if trace, ok := c.Value(ProcessTraceIdKey).(string); ok {
		return ProcessTraceIdKey, trace
	}
	return MissingTraceIdKey, "no-trace-id-in-context"
}

// ActiveTraceIdValue returns the value part of ActiveTraceId (does not return the TradeIdKey type part).
func ActiveTraceIdValue(c context.Context) string {
	_, v := ActiveTraceId(c)
	return v
}

func AddTo(c context.Context, args ...any) context.Context {
	ctx, _ := AddToR(c, args...)
	return ctx
}

func AddToR(c context.Context, args ...any) (context.Context, *slog.Logger) {
	logger := Logger(c)
	logger = logger.With(args...)
	return WithLogger(c, logger), logger
}

type NewLoggerInput struct {
	Level     string
	Format    string
	File      string
	BuildSha  string
	BuildTime string
	// Called with the derived handler options,
	// and the result of the default handler logic.
	// Allows the replacement or wrapping of the calculated handler
	// with a custom handler.
	// For example, use NewTracingHandler(h) to wrap the handler
	// in one that will log the span and trace ids in the context.
	MakeHandler func(*slog.HandlerOptions, slog.Handler) slog.Handler
	Fields      []any
}

func NewLogger(cfg NewLoggerInput) (*slog.Logger, error) {
	// Set output to file or stdout/stderr (stderr for tty, stdout otherwise like for 12 factor apps)
	var out io.Writer
	if cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		out = file
	} else if IsTty() {
		out = os.Stderr
	} else {
		out = os.Stdout
	}

	hopts := &slog.HandlerOptions{}
	lvl, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	hopts.Level = lvl

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(out, hopts)
	} else if cfg.Format == "text" {
		handler = slog.NewTextHandler(out, hopts)
	} else if cfg.File != "" {
		handler = slog.NewJSONHandler(out, hopts)
	} else if IsTty() {
		handler = console.NewHandler(out, &console.HandlerOptions{
			AddSource: hopts.AddSource,
			Level:     hopts.Level,
		})
	} else {
		handler = slog.NewJSONHandler(out, hopts)
	}
	if cfg.MakeHandler != nil {
		handler = cfg.MakeHandler(hopts, handler)
	}

	logger := slog.New(handler)
	if len(cfg.Fields) > 0 {
		logger = logger.With(cfg.Fields...)
	}
	if cfg.BuildSha != "" {
		logger = logger.With("build_sha", cfg.BuildSha)
	}
	if cfg.BuildTime != "" {
		logger = logger.With("build_time", cfg.BuildTime)
	}
	return logger, nil
}

func IsTty() bool {
	return terminal.IsTerminal(int(os.Stdout.Fd()))
}

// WithNullLogger adds the logger from test.NewNullLogger into the given context
// (default c to context.Background). Use the hook to get the log messages.
// See https://github.com/sirupsen/logrus#testing for examples,
// though this doesn't use logrus the ideas still apply.
func WithNullLogger(c context.Context) (context.Context, *Hook) {
	if c == nil {
		c = context.Background()
	}
	logger, hook := NewNullLogger()
	c2 := WithLogger(c, logger.With("testlogger", true))
	return c2, hook
}

func ParseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}

func NewNullLogger() (*slog.Logger, *Hook) {
	hook := NewHook()
	logger := slog.New(hook)
	return logger, hook
}
