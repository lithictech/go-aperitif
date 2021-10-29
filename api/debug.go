package api

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/lithictech/go-aperitif/logctx"
	"net/http"
)

type DebugMiddlewareConfig struct {
	Enabled             bool
	DumpRequestBody     bool
	DumpResponseBody    bool
	DumpRequestHeaders  bool
	DumpResponseHeaders bool
	DumpAll             bool
}

func DebugMiddleware(cfg DebugMiddlewareConfig) echo.MiddlewareFunc {
	if !cfg.Enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}
	if cfg.DumpAll {
		cfg.DumpRequestHeaders = true
		cfg.DumpRequestBody = true
		cfg.DumpResponseHeaders = true
		cfg.DumpResponseBody = true
	}
	bd := middleware.BodyDump(func(c echo.Context, reqBody []byte, resBody []byte) {
		log := logctx.Logger(StdContext(c))
		if cfg.DumpRequestBody {
			log = log.WithField("debug_request_body", string(reqBody))
		}
		if cfg.DumpResponseBody {
			log = log.WithField("debug_response_body", string(resBody))
		}
		if cfg.DumpRequestHeaders {
			log = log.WithField("debug_request_headers", headerToMap(c.Request().Header))
		}
		if cfg.DumpResponseHeaders {
			log = log.WithField("debug_response_headers", headerToMap(c.Response().Header()))
		}
		log.Debug("request_debug")
	})
	return bd
}

func headerToMap(h http.Header) map[string]string {
	r := make(map[string]string, len(h))
	for k := range h {
		r[k] = h.Get(k)
	}
	return r
}
