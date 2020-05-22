package api

import (
	"github.com/labstack/echo"
)

func WithCacheControl(enabled bool, value string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if enabled {
				c.Set("cache-control-value", value)
			}
			return next(c)
		}
	}
}

// CacheControl sets the Cache-Control header to the given value,
// if it was configured in WithCacheControl.
// Because response headers must be written before the body is written,
// we cannot handle this like normal middleware, and write the header after handling the call.
// Yet, we do not want to write the header in the case of error,
// so the endpoint itself MUST write the cache-control header.
// Thus, we can configure middleware, and call CacheControl unconditionally in our handlers,
// but it will noop if not configured.
func SetCacheControl(c echo.Context) {
	value, ok := c.Get("cache-control-value").(string)
	if ok {
		c.Response().Header().Set("Cache-Control", value)
	}
}
