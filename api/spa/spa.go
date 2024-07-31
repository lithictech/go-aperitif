// Package spa creates a set of middlewares for use when serving a Single Page Application
// from an API server.
//
// The problem with `echo.middleware.Static` is that it doesn't handle non-root routes,
// so you end up with 404s/403s/whatever, instead of redirecting to serving the index.html file.
package spa

import (
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

type Matcher func(r *http.Request) (bool, error)

type Config struct {
	// If Handle returns true, the route is treated as a static route (passed to the Static middleware).
	// Required.
	Handle Matcher
	// The middleware used for handling static routes.
	// See echo.middleware.Static for info on configuration.
	// Required.
	Static echo.MiddlewareFunc
	// When Handle returns true, this is the path passed to the Static middleware.
	// Defaults to index.html.
	Path string
}

func Middleware(static echo.MiddlewareFunc, handle Matcher) echo.MiddlewareFunc {
	return MiddlewareWithConfig(Config{Handle: handle, Static: static})
}

func MiddlewareWithConfig(cfg Config) echo.MiddlewareFunc {
	if cfg.Handle == nil {
		cfg.Handle = func(r *http.Request) (bool, error) {
			log.Println("spa middleware Matcher empty, not matching anything")
			return false, nil
		}
	}
	if cfg.Path == "" {
		cfg.Path = "index.html"
	}
	if cfg.Static == nil {
		cfg.Static = func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		// Wrap our custom handler in the static middleware,
		// so we never run our handler if the static middleware matches.
		return cfg.Static(func(c echo.Context) error {
			handle, err := cfg.Handle(c.Request())
			if err != nil {
				return err
			}
			if !handle {
				return next(c)
			}
			// Do not match explicitly registered routes (usually /statusz, etc).
			// In the future we may want to see if we can use echo's actual match logic.
			req := c.Request()
			for _, route := range c.Echo().Routes() {
				if route.Path == req.URL.Path {
					return next(c)
				}
			}
			// At this point, we:
			// - Have not matched an existing static file
			// - Have said we want to handle this with SPA middleware
			// - Have not matched an explicitly registered route
			// So, call the static middleware as if we were requesting index.html originally.
			req.URL.Path = cfg.Path
			c.SetRequest(req)
			return cfg.Static(next)(c)
		})
	}
}
