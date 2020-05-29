package logctx

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

const LoggerKey = "logger"

type TraceIdKey string

// Trace ID key for requests.
const RequestTraceIdKey TraceIdKey = "trace_id"

// Trace ID key for when we run jobs in the background, like cron jobs.
const JobTraceIdKey TraceIdKey = "job_trace_id"

// Trace ID key for the overall process.
const ProcessTraceIdKey TraceIdKey = "process_trace_id"

// Trace ID key that will be present to indicate something is misconfigured.
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
