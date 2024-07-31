package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lithictech/go-aperitif/v2/logctx"
	"net/http"
	"runtime"
	"sync/atomic"
)

type DebugMiddlewareConfig struct {
	Enabled             bool
	DumpRequestBody     bool
	DumpResponseBody    bool
	DumpRequestHeaders  bool
	DumpResponseHeaders bool
	DumpAll             bool
	// Log out memory stats every 'n' requests.
	// If <= 0, do not log them.
	DumpMemoryEvery int
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
	var requestCounter uint64
	dumpEveryUint := uint64(cfg.DumpMemoryEvery)
	bd := middleware.BodyDump(func(c echo.Context, reqBody []byte, resBody []byte) {
		atomic.AddUint64(&requestCounter, 1)
		log := logctx.Logger(StdContext(c))
		if cfg.DumpRequestBody {
			log = log.With("debug_request_body", string(reqBody))
		}
		if cfg.DumpResponseBody {
			log = log.With("debug_response_body", string(resBody))
		}
		if cfg.DumpRequestHeaders {
			log = log.With("debug_request_headers", headerToMap(c.Request().Header))
		}
		if cfg.DumpResponseHeaders {
			log = log.With("debug_response_headers", headerToMap(c.Response().Header()))
		}
		if cfg.DumpMemoryEvery > 0 && (requestCounter%dumpEveryUint) == 0 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			log = log.With(
				"memory_alloc", ms.Alloc,
				"memory_total_alloc", ms.TotalAlloc,
				"memory_sys", ms.Sys,
				"memory_mallocs", ms.Mallocs,
				"memory_frees", ms.Frees,
				"memory_heap_alloc", ms.HeapAlloc,
				"memory_heap_sys", ms.HeapSys,
				"memory_heap_idle", ms.HeapIdle,
				"memory_heap_inuse", ms.HeapInuse,
				"memory_heap_released", ms.HeapReleased,
				"memory_heap_objects", ms.HeapObjects,
				"memory_stack_inuse", ms.StackInuse,
				"memory_stack_sys", ms.StackSys,
				"memory_other_sys", ms.OtherSys,
				"memory_next_gc", ms.NextGC,
				"memory_last_gc", ms.LastGC,
				"memory_pause_total_ns", ms.PauseTotalNs,
				"memory_num_gc", ms.NumGC,
			)
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
