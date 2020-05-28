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
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type Config struct {
	Logger         *logrus.Entry
	HealthHandler  echo.HandlerFunc
	CorsOrigins    []string
	HealthResponse map[string]interface{}
	StatusResponse map[string]interface{}
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
	if cfg.StatusResponse == nil {
		cfg.StatusResponse = map[string]interface{}{
			"version": "not configured",
			"message": "you are a lovely and strong person",
		}
	}
	e := echo.New()
	e.Logger.SetOutput(os.Stdout)
	e.HideBanner = true
	e.HTTPErrorHandler = HTTPErrorHandler
	e.Use(LoggingMiddleware(cfg.Logger))
	if cfg.CorsOrigins != nil {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     cfg.CorsOrigins,
			AllowCredentials: true,
		}))
	}
	e.GET(HealthPath, cfg.HealthHandler)
	e.GET(StatusPath, func(c echo.Context) error {
		return c.JSON(http.StatusOK, cfg.StatusResponse)
	})
	return e
}

const HealthPath = "/healthz"
const StatusPath = "/statusz"
