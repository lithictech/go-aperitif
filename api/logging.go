package api

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/lithictech/go-aperitif/v2/api/apiparams"
	"github.com/lithictech/go-aperitif/v2/logctx"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

func Logger(c echo.Context) *slog.Logger {
	logger, ok := c.Get(logctx.LoggerKey).(*slog.Logger)
	if !ok {
		logger = logctx.UnconfiguredLogger()
		logger.Error("No logger configured for request!")
	}
	return logger
}

func SetLogger(c echo.Context, logger *slog.Logger) {
	c.Set(logctx.LoggerKey, logger)
}

type LoggingMiddlwareConfig struct {
	// If true, log request headers.
	RequestHeaders bool
	// If true, log response headers.
	ResponseHeaders bool
	// If true, do not log trace_id to the logs.
	// Use this when doing your own trace logging, like with logctx.TracingHandler.
	// Note that the trace ID for the request is still available in the request.
	SkipTraceAttrs bool

	// If provided, the returned logger is stored in the context
	// which is eventually passed to the handler.
	// Use to add additional fields to the logger based on the request.
	BeforeRequest func(echo.Context, *slog.Logger) *slog.Logger
	// If provided, the returned logger is used for response logging.
	// Use to add additional fields to the logger based on the request or response.
	AfterRequest func(echo.Context, *slog.Logger) *slog.Logger
	// The function that does the actual logging.
	// By default, it will log at a certain level based on the status code of the response.
	DoLog func(echo.Context, *slog.Logger)
}

func LoggingMiddleware(outerLogger *slog.Logger) echo.MiddlewareFunc {
	return LoggingMiddlewareWithConfig(outerLogger, LoggingMiddlwareConfig{})
}

func LoggingMiddlewareWithConfig(outerLogger *slog.Logger, cfg LoggingMiddlwareConfig) echo.MiddlewareFunc {
	if cfg.DoLog == nil {
		cfg.DoLog = LoggingMiddlewareDefaultDoLog
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			path := req.URL.Path
			if path == "" {
				path = "/"
			}
			bytesIn := req.Header.Get(echo.HeaderContentLength)
			if bytesIn == "" {
				bytesIn = "0"
			}

			logger := outerLogger
			if !cfg.SkipTraceAttrs {
				logger = logger.With(string(logctx.RequestTraceIdKey), TraceId(c))
			}
			if cfg.BeforeRequest != nil {
				logger = cfg.BeforeRequest(c, logger)
			}

			SetLogger(c, logger)

			err := safeInvokeNext(logger, next, c)
			err = adaptToError(err)
			if err != nil {
				c.Error(err)
			}

			stop := time.Now()
			res := c.Response()

			logger = Logger(c).With(
				"request_started_at", start.Format(time.RFC3339),
				"request_remote_ip", c.RealIP(),
				"request_method", req.Method,
				"request_uri", req.RequestURI,
				"request_protocol", req.Proto,
				"request_host", req.Host,
				"request_path", path,
				"request_query", req.URL.RawQuery,
				"request_referer", req.Referer(),
				"request_user_agent", req.UserAgent(),
				"request_bytes_in", bytesIn,

				"request_finished_at", stop.Format(time.RFC3339),
				"request_status", res.Status,
				"request_latency_ms", int(stop.Sub(start))/1000/1000,
				"request_bytes_out", strconv.FormatInt(res.Size, 10),
			)
			if cfg.RequestHeaders {
				for k, v := range req.Header {
					if len(v) > 0 && k != "Authorization" && k != "Cookie" {
						logger = logger.With("request_header."+k, v[0])
					}
				}
			}
			if cfg.ResponseHeaders {
				for k, v := range res.Header() {
					if len(v) > 0 && k != "Set-Cookie" {
						logger = logger.With("response_header."+k, v[0])
					}
				}
			}
			if err != nil {
				logger = logger.With("request_error", err)
			}
			if cfg.AfterRequest != nil {
				logger = cfg.AfterRequest(c, logger)
			}
			if logger != nil {
				cfg.DoLog(c, logger)
			}
			// c.Error is already called
			return nil
		}
	}
}

func LoggingMiddlewareDefaultDoLog(c echo.Context, logger *slog.Logger) {
	req := c.Request()
	res := c.Response()
	logMethod := logger.Info
	if req.Method == http.MethodOptions {
		logMethod = logger.Debug
	} else if res.Status >= 500 {
		logMethod = logger.Error
	} else if res.Status >= 400 {
		logMethod = logger.Warn
	} else if req.URL.Path == HealthPath || req.URL.Path == StatusPath {
		logMethod = logger.Debug
	}
	logMethod("request_finished")
}

// Invoke next(c) within a function wrapped with defer,
// so that if it panics, we can recover from it and pass on a 500.
// Use the "named return parameter can be set in defer" trick so we can
// return the error we create from the panic.
func safeInvokeNext(logger *slog.Logger, next echo.HandlerFunc, c echo.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
			stack := make([]byte, 4<<10) // 4kb
			length := runtime.Stack(stack, true)
			logger.With(
				"error", err,
				"stack", string(stack[:length]),
			).Error("panic_recover")
		}
	}()
	err = next(c)
	return
}

func adaptToError(e error) error {
	if e == nil {
		return nil
	}
	var apiErr Error
	if errors.As(e, &apiErr) {
		return apiErr
	}
	var ee *echo.HTTPError
	if errors.As(e, &ee) {
		apiErr := NewError(ee.Code, "echo", ee.Internal)
		apiErr.Message = fmt.Sprintf("%v", ee.Message)
		return apiErr
	}
	var ae apiparams.HTTPError
	if errors.As(e, &ae) {
		apiErr := NewError(ae.Code(), "validation", ae)
		apiErr.Message = ae.Error()
		return apiErr
	}
	return NewInternalError(e)
}

func NewHTTPErrorHandler(e *echo.Echo) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var apiErr Error
		if ok := errors.As(err, &apiErr); !ok {
			e.DefaultHTTPErrorHandler(err, c)
			return
		}
		// This is based on echo's default error handler,
		if !c.Response().Committed {
			// We can have api errors that are using a non-error status code.
			// We should still return a spec-correct response,
			// using no body for 204, 304, and HEAD requests.
			noContent := c.Request().Method == http.MethodHead ||
				apiErr.HTTPStatus == http.StatusNoContent ||
				apiErr.HTTPStatus == http.StatusNotModified
			var err error
			if noContent {
				err = c.NoContent(apiErr.HTTPStatus)
			} else {
				err = c.JSON(apiErr.HTTPStatus, apiErr)
			}
			if err != nil {
				Logger(c).With("error", err).Error("http_error_handler_error")
			}
		}
	}
}
