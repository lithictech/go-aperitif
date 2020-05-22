package api

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/lithictech/aperitif/logctx"
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

func LoggingMiddleware(outerLogger *logrus.Entry) echo.MiddlewareFunc {
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
			//for k, v := range req.Header {
			//	if len(v) > 0 && k != "Authorization" && k != "Cookie" {
			//		logger = logger.WithField("header."+k, v[0])
			//	}
			//}

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
			if err != nil {
				logger = logger.WithField("request_error", err)
			}

			logMethod := logger.Info
			if res.Status >= 500 {
				logMethod = logger.Error
			} else if res.Status >= 400 {
				logMethod = logger.Warn
			}
			logMethod("request_finished")

			// c.Error is already called
			return nil
		}
	}
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
	return NewInternalError(e)
}

// HTTPErrorHandler is a custom error handler, at this point we should always have an api.Error
// because it's been adapted by the logging middleware.
func HTTPErrorHandler(e error, c echo.Context) {
	apiErr := e.(Error)
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
