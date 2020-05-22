package api

import (
	"context"
	"github.com/labstack/echo"
	"github.com/lithictech/go-aperitif/logctx"
)

// StdContext returns a standard context from an echo context.
// Useful when you are in an endpoint, and want to call into some code
// that shouldn't use echo, but you still want to pass along the original endpoint's context
// so you can get stuff back out of it (like a logger).
// This uses the logctx package to set the expected values in the context,
// so the echo context's request trace and logger are passed along.
func StdContext(c echo.Context) context.Context {
	cc := context.Background()
	cc = context.WithValue(cc, logctx.RequestTraceIdKey, TraceId(c))
	cc = logctx.WithLogger(cc, Logger(c))
	cc = logctx.WithTracingLogger(cc)
	return cc
}
