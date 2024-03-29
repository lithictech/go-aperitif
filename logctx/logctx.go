package logctx

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

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

func unconfiguredLogger() *logrus.Entry {
	return logrus.New().WithField("unconfigured_logger", "true")
}

// WithLogger returns a new context that adds a logger which
// can be retrieved with Logger(Context).
func WithLogger(c context.Context, logger *logrus.Entry) context.Context {
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
	logger = logger.WithField(string(tkey), trace)
	return context.WithValue(c, LoggerKey, logger)
}

func WithTraceId(c context.Context, key TraceIdKey) context.Context {
	return context.WithValue(c, key, uuid.New().String())
}

func LoggerOrNil(c context.Context) *logrus.Entry {
	logger, _ := c.Value(LoggerKey).(*logrus.Entry)
	return logger
}

func Logger(c context.Context) *logrus.Entry {
	if logger, ok := c.Value(LoggerKey).(*logrus.Entry); ok {
		return logger
	}
	logger := unconfiguredLogger()
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

func AddFieldsAndGet(c context.Context, fields map[string]interface{}) (context.Context, *logrus.Entry) {
	logger := Logger(c)
	logger = logger.WithFields(fields)
	return WithLogger(c, logger), logger
}

func AddFieldAndGet(c context.Context, key string, value interface{}) (context.Context, *logrus.Entry) {
	return AddFieldsAndGet(c, map[string]interface{}{key: value})
}

func AddFields(c context.Context, fields map[string]interface{}) context.Context {
	ctx, _ := AddFieldsAndGet(c, fields)
	return ctx
}

func AddField(c context.Context, key string, value interface{}) context.Context {
	return AddFields(c, map[string]interface{}{key: value})
}

type NewLoggerInput struct {
	Level     string
	Format    string
	File      string
	BuildSha  string
	BuildTime string
	Fields    logrus.Fields
}

func NewLogger(cfg NewLoggerInput) (*logrus.Entry, error) {
	logger := logrus.New()

	// Parse and set level
	lvl, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(lvl)

	// Set format
	if cfg.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else if cfg.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{})
	} else if cfg.File != "" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else if IsTty() {
		logger.SetFormatter(&logrus.TextFormatter{})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Set output to file or stdout/stderr (stderr for tty, stdout otherwise like for 12 factor apps)
	if cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		logger.SetOutput(file)
	} else if IsTty() {
		logger.SetOutput(os.Stderr)
	} else {
		logger.SetOutput(os.Stdout)
	}

	entry := logger.WithFields(nil)
	if len(cfg.Fields) > 0 {
		entry = logger.WithFields(cfg.Fields)
	}
	if cfg.BuildSha != "" {
		entry = entry.WithField("build_sha", cfg.BuildSha)
	}
	if cfg.BuildTime != "" {
		entry = entry.WithField("build_time", cfg.BuildTime)
	}
	return entry, nil
}

func IsTty() bool {
	return terminal.IsTerminal(int(os.Stdout.Fd()))
}

// WithNullLogger adds the logger from test.NewNullLogger into the given context
// (default c to context.Background). Use the hook to get the log messages.
// See https://github.com/sirupsen/logrus#testing for testing with logrus.
func WithNullLogger(c context.Context) (context.Context, *test.Hook) {
	if c == nil {
		c = context.Background()
	}
	logger, hook := test.NewNullLogger()
	c2 := WithLogger(c, logger.WithField("testlogger", true))
	return c2, hook
}
