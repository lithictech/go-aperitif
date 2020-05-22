package api

import (
	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/lithictech/aperitif/logctx"
)

const TraceIdHeader = "Trace-Id"

// TraceId returns the trace id for the request.
// If it's already in the echo context, use that as this has already been called.
// Otherwise, if it's provided in the request, set the response header trace id and cache in context.
// Otherwise, generate a new trace id, set the response header trace id and cache it in context.
func TraceId(c echo.Context) string {
	traceIdKey := string(logctx.RequestTraceIdKey)
	idInCtx := c.Get(traceIdKey)
	if idInCtx != nil {
		return idInCtx.(string)
	}

	idInHeader := c.Request().Header.Get(TraceIdHeader)
	if idInHeader != "" {
		c.Set(traceIdKey, idInHeader)
		c.Response().Header().Set(TraceIdHeader, idInHeader)
		return idInHeader
	}

	newId := uuid.New().String()
	c.Set(traceIdKey, newId)
	c.Response().Header().Set(TraceIdHeader, newId)
	return newId
}
