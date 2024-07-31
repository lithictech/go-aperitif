/*
Package api is a standalone API package/pattern built on echo and logrus.
It sets up /statusz and /healthz endpoints,
and sets up logging middleware that takes care of the following important,
and fundamentally (in Go) interconnected tasks:

  - Extract (or add) a trace ID header to the request and response.
  - The trace ID can be retrieved through api.TraceID(context) of the echo.Context for the request.
  - Use that trace ID header as context for the logrus logger.
  - Handle request logging (metadata about the request and response,
    and log at the level appropriate for the status code).
  - The request logger can be retrieved api.Logger(echo.Context).
  - Recover from panics.
  - Coerce all errors into api.Error types, and marshal them.
  - Override echo's HTTPErrorHandler to pass through api.Error types.
*/
package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type Config struct {
	// If not provided, create an echo.New.
	App                    *echo.Echo
	Logger                 *logrus.Entry
	LoggingMiddlwareConfig LoggingMiddlwareConfig
	// Origins for echo's CORS middleware.
	// If it and CorsConfig are empty, do not add the middleware.
	CorsOrigins []string
	// Config for echo's CORS middleware.
	// Supercedes CorsOrigins.
	// If it and CorsOrigins are empty, do not add the middleware.
	CorsConfig *middleware.CORSConfig
	// Return this from the health endpoint.
	// Defaults to {"o":"k"}.
	HealthResponse map[string]interface{}
	// Defaults to /healthz.
	HealthPath string
	// If the health endpoint is not static
	// (for example so it can check whether a database is available),
	// provide this instead of HealthResponse.
	HealthHandler echo.HandlerFunc
	// Return this from the status endpoint.
	// The default is not very useful so you should provide a value.
	StatusResponse map[string]interface{}
	// Defaults to /statusz
	StatusPath string
	// If the status endpoint is not static,
	// provide this instead of StatusRespoinse.
	StatusHandler echo.HandlerFunc
}

func New(cfg Config) *echo.Echo {
	if cfg.Logger == nil {
		cfg.Logger = unconfiguredLogger()
	}
	if cfg.HealthHandler == nil {
		if cfg.HealthResponse == nil {
			cfg.HealthResponse = map[string]interface{}{"o": "k"}
		}
		cfg.HealthHandler = func(c echo.Context) error {
			return c.JSON(http.StatusOK, cfg.HealthResponse)
		}
	}
	if cfg.HealthPath == "" {
		cfg.HealthPath = HealthPath
	}
	if cfg.StatusHandler == nil {
		if cfg.StatusResponse == nil {
			cfg.StatusResponse = map[string]interface{}{
				"version": "not configured",
				"message": "you are a lovely and strong person",
			}
		}
		cfg.StatusHandler = func(c echo.Context) error {
			return c.JSON(http.StatusOK, cfg.StatusResponse)
		}
	}
	if cfg.StatusPath == "" {
		cfg.StatusPath = StatusPath
	}
	e := cfg.App
	if e == nil {
		e = echo.New()
	}
	e.Logger.SetOutput(os.Stdout)
	e.HideBanner = true
	e.HTTPErrorHandler = NewHTTPErrorHandler(e)
	e.Use(LoggingMiddlewareWithConfig(cfg.Logger, cfg.LoggingMiddlwareConfig))
	if cfg.CorsConfig == nil && cfg.CorsOrigins != nil {
		cfg.CorsConfig = &middleware.CORSConfig{AllowOrigins: cfg.CorsOrigins, AllowCredentials: true}
	}
	if cfg.CorsConfig != nil {
		e.Use(middleware.CORSWithConfig(*cfg.CorsConfig))
	}
	e.GET(cfg.HealthPath, cfg.HealthHandler)
	e.GET(cfg.StatusPath, cfg.StatusHandler)
	return e
}

const HealthPath = "/healthz"
const StatusPath = "/statusz"
