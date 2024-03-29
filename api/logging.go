package api

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/lithictech/go-aperitif/api/apiparams"
	"github.com/lithictech/go-aperitif/logctx"
	"github.com/sirupsen/logrus"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

func unconfiguredLogger() *logrus.Entry {
	return logrus.New().WithField("unconfigured_logger", "true")
}

func Logger(c echo.Context) *logrus.Entry {
	logger, ok := c.Get(logctx.LoggerKey).(*logrus.Entry)
	if !ok {
		logger = unconfiguredLogger()
		logger.Error("No logger configured for request!")
	}
	return logger
}

func SetLogger(c echo.Context, logger *logrus.Entry) {
	c.Set(logctx.LoggerKey, logger)
}

type LoggingMiddlwareConfig struct {
	// If true, log request headers.
	RequestHeaders bool
	// If true, log response headers.
	ResponseHeaders bool
	// If provided, the returned logger is stored in the context
	// which is eventually passed to the handler.
	// Use to add additional fields to the logger based on the request.
	BeforeRequest func(echo.Context, *logrus.Entry) *logrus.Entry
	// If provided, the returned logger is used for response logging.
	// Use to add additional fields to the logger based on the request or response.
	AfterRequest func(echo.Context, *logrus.Entry) *logrus.Entry
	// The function that does the actual logging.
	// By default, it will log at a certain level based on the status code of the response.
	DoLog func(echo.Context, *logrus.Entry)
}

func LoggingMiddleware(outerLogger *logrus.Entry) echo.MiddlewareFunc {
	return LoggingMiddlewareWithConfig(outerLogger, LoggingMiddlwareConfig{})
}

func LoggingMiddlewareWithConfig(outerLogger *logrus.Entry, cfg LoggingMiddlwareConfig) echo.MiddlewareFunc {
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

			logger := outerLogger.WithFields(logrus.Fields{
				"request_started_at":             start.Format(time.RFC3339),
				"request_remote_ip":              c.RealIP(),
				"request_method":                 req.Method,
				"request_uri":                    req.RequestURI,
				"request_protocol":               req.Proto,
				"request_host":                   req.Host,
				"request_path":                   path,
				"request_query":                  req.URL.RawQuery,
				"request_referer":                req.Referer(),
				"request_user_agent":             req.UserAgent(),
				"request_bytes_in":               bytesIn,
				string(logctx.RequestTraceIdKey): TraceId(c),
			})
			if cfg.RequestHeaders {
				for k, v := range req.Header {
					if len(v) > 0 && k != "Authorization" && k != "Cookie" {
						logger = logger.WithField("request_header."+k, v[0])
					}
				}
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

			logger = Logger(c).WithFields(logrus.Fields{
				"request_finished_at": stop.Format(time.RFC3339),
				"request_status":      res.Status,
				"request_latency_ms":  int(stop.Sub(start)) / 1000 / 1000,
				"request_bytes_out":   strconv.FormatInt(res.Size, 10),
			})
			if cfg.ResponseHeaders {
				for k, v := range res.Header() {
					if len(v) > 0 && k != "Set-Cookie" {
						logger = logger.WithField("response_header."+k, v[0])
					}
				}
			}
			if err != nil {
				logger = logger.WithField("request_error", err)
			}
			if cfg.BeforeRequest != nil {
				logger = cfg.AfterRequest(c, logger)
			}
			cfg.DoLog(c, logger)
			// c.Error is already called
			return nil
		}
	}
}

func LoggingMiddlewareDefaultDoLog(c echo.Context, logger *logrus.Entry) {
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
func safeInvokeNext(logger *logrus.Entry, next echo.HandlerFunc, c echo.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
			stack := make([]byte, 4<<10) // 4kb
			length := runtime.Stack(stack, true)
			logger.WithFields(logrus.Fields{
				"error": err,
				"stack": string(stack[:length]),
			}).Error("panic_recover")
		}
	}()
	err = next(c)
	return
}

func adaptToError(e error) error {
	if e == nil {
		return nil
	}
	if apiErr, ok := e.(Error); ok {
		return apiErr
	}
	if ee, ok := e.(*echo.HTTPError); ok {
		apiErr := NewError(ee.Code, "echo", ee.Internal)
		apiErr.Message = fmt.Sprintf("%v", ee.Message)
		return apiErr
	}
	if ae, ok := e.(apiparams.HTTPError); ok {
		apiErr := NewError(ae.Code(), "validation", ae)
		apiErr.Message = ae.Error()
		return apiErr
	}
	return NewInternalError(e)
}

// Deprecated: Use NewHTTPErrorHandler instead.
func HTTPErrorHandler(err error, c echo.Context) {
	e := echo.New()
	NewHTTPErrorHandler(e)(err, c)
}

func NewHTTPErrorHandler(e *echo.Echo) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		apiErr, ok := err.(Error)
		if !ok {
			e.DefaultHTTPErrorHandler(err, c)
			return
		}
		// This is copied from echo's default error handler.
		if !c.Response().Committed {
			var err error
			if c.Request().Method == http.MethodHead {
				err = c.NoContent(apiErr.HTTPStatus)
			} else {
				err = c.JSON(apiErr.HTTPStatus, apiErr)
			}
			if err != nil {
				Logger(c).WithField("error", err).Error("http_error_handler_error")
			}
		}
	}
}
