package api

import (
	"github.com/labstack/echo/v4"
	"github.com/lithictech/go-aperitif/v2/logctx"
)

const TraceIdHeader = "Trace-Id"

var candidateTraceHeaders = []string{
	TraceIdHeader,
	"X-Request-Id",
}

// TraceId returns the trace id for the request.
//
// If it's already in the echo context, use that as this has already been called.
//
// Otherwise, if it's provided in the request through one of the supported headers,
// set the response header trace id and cache in context.
// See SupportedTraceIdHeaders for where the trace id will be pulled from,
// in order of preference.
//
// Otherwise, generate a new trace id, set the response header trace id and cache it in context.
func TraceId(c echo.Context) string {
	traceIdKey := string(logctx.RequestTraceIdKey)
	idInCtx := c.Get(traceIdKey)
	if idInCtx != nil {
		return idInCtx.(string)
	}

	for _, header := range candidateTraceHeaders {
		idInHeader := c.Request().Header.Get(header)
		if idInHeader != "" {
			c.Set(traceIdKey, idInHeader)
			c.Response().Header().Set(TraceIdHeader, idInHeader)
			return idInHeader
		}
	}

	newId := logctx.IdProvider()
	c.Set(traceIdKey, newId)
	c.Response().Header().Set(TraceIdHeader, newId)
	return newId
}
