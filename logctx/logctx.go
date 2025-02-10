package logctx

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/phsym/console-slog"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log/slog"
	"os"
	"reflect"
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
// The returned trace value will always be a string; if the value is string-like it'll be used,
// but if it is not string-like, it will have '!BADVALUE-' prepended.
// For example, ActiveTraceId(context.WithValue(ctx, RequestTraceIdKey, 5)) would return
// (RequestTraceIdKey, "!BADVALUE-5").
func ActiveTraceId(c context.Context) (TraceIdKey, string) {
	if tv := c.Value(RequestTraceIdKey); tv != nil {
		return RequestTraceIdKey, toTraceVal(tv)
	}
	if tv := c.Value(JobTraceIdKey); tv != nil {
		return JobTraceIdKey, toTraceVal(tv)
	}
	if tv := c.Value(ProcessTraceIdKey); tv != nil {
		return ProcessTraceIdKey, toTraceVal(tv)
	}
	return MissingTraceIdKey, "no-trace-id-in-context"
}

func toTraceVal(v any) string {
	s, ok := AsString(v)
	if ok {
		return s
	}
	return fmt.Sprintf("!BADVALUE-%v", v)
}

// AsString returns o as a string and true if o is a string,
// a fmt.Stringer, or a reflect.String kind (subtype of string).
// Otherwise, return "" and false.
func AsString(o any) (string, bool) {
	if o == nil {
		return "", false
	} else if s, ok := o.(string); ok {
		return s, true
	} else if s, ok := o.(fmt.Stringer); ok {
		return s.String(), true
	}
	r := reflect.ValueOf(o)
	if r.Kind() == reflect.String {
		return r.String(), true
	}
	return "", false
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
	// Level is the logging level name. Should match slog.Level strings
	// ('debug', 'info', 'warning', 'error').
	// Case independent.
	Level string
	// Format should be empty, 'json' or 'text'.
	// If empty, use 'json' if File is set, colored text/console if IsTty,
	// or 'json' otherwise.
	Format string
	// File is the filename to log to.
	File string
	// Out specifies the stream to log to.
	// If File is set, log to that file.
	// If IsTty, log to os.Stderr.
	// Otherwise, log to os.Stdout.
	Out io.Writer
	// BuildSha will add "build_sha" to the logger fields, if not empty.
	BuildSha string
	// BuildTime will add "build_time" to the logger fields, it not empty.
	BuildTime string
	// MakeHandler can override the slog.Handler assigned to the logger.
	// Called with the derived handler options,
	// and the result of the default handler logic.
	// Allows the replacement or wrapping of the calculated handler
	// with a custom handler.
	// For example, use NewTracingHandler(h) to wrap the handler
	// in one that will log the span and trace ids in the context.
	MakeHandler func(*slog.HandlerOptions, slog.Handler) slog.Handler
	// Fields are additional fields to add to the logger.
	Fields []any
}

func NewLogger(cfg NewLoggerInput) (*slog.Logger, error) {
	// Set output to file or stdout/stderr (stderr for tty, stdout otherwise like for 12 factor apps)
	var out io.Writer
	if cfg.Out != nil {
		out = cfg.Out
	} else if cfg.File != "" {
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
