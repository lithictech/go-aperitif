package api_test

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/lithictech/go-aperitif/api"
	"github.com/lithictech/go-aperitif/api/apiparams"
	. "github.com/lithictech/go-aperitif/api/echoapitest"
	. "github.com/lithictech/go-aperitif/apitest"
	"github.com/lithictech/go-aperitif/logctx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rgalanakis/golangal"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("API", func() {
	var e *echo.Echo

	var logger *slog.Logger
	var logHook *logctx.Hook

	BeforeEach(func() {
		logger, logHook = logctx.NewNullLogger()
		e = api.New(api.Config{
			Logger:         logger,
			HealthResponse: map[string]interface{}{"o": "k"},
			StatusResponse: map[string]interface{}{"it": "me"},
		})
	})

	It("has a health endpoint", func() {
		req := GetRequest("/healthz")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(200))
		Expect(rr).To(HaveJsonBody(HaveKeyWithValue("o", "k")))
	})

	It("can use custom health and status fields", func() {
		e = api.New(api.Config{
			Logger: logger,
			HealthHandler: func(c echo.Context) error {
				return c.String(200, "yo")
			},
			HealthPath: "/health",
			StatusHandler: func(c echo.Context) error {
				return c.String(202, "hai")
			},
			StatusPath: "/status",
		})
		rr := Serve(e, GetRequest("/health"))
		Expect(rr).To(HaveResponseCode(200))
		Expect(rr.Body.String()).To(Equal("yo"))
		rr = Serve(e, GetRequest("/status"))
		Expect(rr).To(HaveResponseCode(202))
		Expect(rr.Body.String()).To(Equal("hai"))
	})

	It("has a status endpoint", func() {
		req := GetRequest("/statusz")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(200))
		Expect(rr).To(HaveJsonBody(HaveKeyWithValue("it", "me")))
	})

	It("defaults all config values", func() {
		e = api.New(api.Config{
			HealthResponse: map[string]interface{}{"o": "k"},
			StatusResponse: map[string]interface{}{"it": "me"},
		})

		Expect(Serve(e, GetRequest("/healthz"))).To(HaveResponseCode(200))
		Expect(Serve(e, GetRequest("/statusz"))).To(HaveResponseCode(200))
	})

	It("can use the provided echo instance", func() {
		e1 := echo.New()
		e2 := api.New(api.Config{App: e1})
		Expect(e2).To(BeIdenticalTo(e1))
	})

	Describe("tracing", func() {
		It("uses the trace id in the Trace-Id header", func() {
			req := GetRequest("/healthz")
			req.Header.Set(api.TraceIdHeader, "abcd")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(200))
			Expect(rr).To(HaveHeader("TRACE-ID", Equal("abcd")))
		})

		It("calculates a returns a new trace id header", func() {
			req := GetRequest("/healthz")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(200))
			Expect(rr).To(HaveHeader("TRACE-ID", Not(BeEmpty())))
		})

		It("will use an existing X-Request-Id and copy it into Trace-Id", func() {
			req := GetRequest("/healthz")
			req.Header.Set("X-Request-ID", "abcd")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(200))
			Expect(rr).To(HaveHeader("TRACE-ID", Equal("abcd")))
		})
	})

	Describe("logging", func() {
		It("does not corrupt the input logger (by reassigning the closure)", func() {
			e.GET("/before-first-call", func(c echo.Context) error {
				Expect(api.Logger(c).Handler().(*logctx.Hook).AttrMap()).ToNot(HaveKey("request_status"))
				return c.String(401, "ok")
			})
			e.GET("/after-first-call", func(c echo.Context) error {
				Expect(api.Logger(c).Handler().(*logctx.Hook).AttrMap()).ToNot(HaveKey("request_status"))
				return c.String(403, "ok")
			})
			Expect(Serve(e, GetRequest("/before-first-call"))).To(HaveResponseCode(401))
			Expect(Serve(e, GetRequest("/after-first-call"))).To(HaveResponseCode(403))
			Expect(logHook.Records()).To(HaveLen(2))
		})
		It("logs normal requests at info", func() {
			e.GET("/", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(200))
			Expect(logHook.Records()).To(HaveLen(1))
			Expect(logHook.Records()[0].Record.Level).To(Equal(slog.LevelInfo))
		})
		It("logs 500+ at error", func() {
			e.GET("/", func(c echo.Context) error {
				return c.String(500, "oh")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(500))
			Expect(logHook.Records()).To(HaveLen(1))
			Expect(logHook.Records()[0].Record.Level).To(Equal(slog.LevelError))
		})
		It("logs 400 to 499 as warn", func() {
			e.GET("/", func(c echo.Context) error {
				return c.String(400, "client err")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(400))
			Expect(logHook.Records()).To(HaveLen(1))
			Expect(logHook.Records()[0].Record.Level).To(Equal(slog.LevelWarn))
		})
		It("logs status and health as debug", func() {
			Expect(Serve(e, GetRequest("/healthz"))).To(HaveResponseCode(200))
			Expect(Serve(e, GetRequest("/statusz"))).To(HaveResponseCode(200))
			Expect(logHook.Records()).To(HaveLen(2))
			Expect(logHook.Records()[0].Record.Level).To(Equal(slog.LevelDebug))
			Expect(logHook.Records()[1].Record.Level).To(Equal(slog.LevelDebug))
		})
		It("logs options as debug", func() {
			Expect(Serve(e, NewRequest("OPTIONS", "/foo", nil))).To(HaveResponseCode(404))
			Expect(logHook.Records()).To(HaveLen(1))
			Expect(logHook.Records()[0].Record.Level).To(Equal(slog.LevelDebug))
		})
		It("can log request and response headers", func() {
			e = api.New(api.Config{
				Logger: logger,
				LoggingMiddlwareConfig: api.LoggingMiddlwareConfig{
					RequestHeaders:  true,
					ResponseHeaders: true,
				},
			})
			e.GET("/", func(c echo.Context) error {
				c.Response().Header().Set("ResHead", "ResHeadVal")
				return c.String(200, "ok")
			})
			Expect(Serve(e, GetRequest("/", SetReqHeader("ReqHead", "ReqHeadVal")))).To(HaveResponseCode(200))
			Expect(logHook.Records()).To(HaveLen(1))
			Expect(logHook.Records()[0].AttrMap()).To(And(
				HaveKeyWithValue("request_header.Reqhead", "ReqHeadVal"),
				HaveKeyWithValue("response_header.Reshead", "ResHeadVal"),
			))
		})
		It("can use custom DoLog, BeforeRequest, and AfterRequest hooks", func() {
			doLogCalled := false
			e = api.New(api.Config{
				Logger: logger,
				LoggingMiddlwareConfig: api.LoggingMiddlwareConfig{
					BeforeRequest: func(_ echo.Context, e *slog.Logger) *slog.Logger {
						return e.With("before", 1)
					},
					AfterRequest: func(_ echo.Context, e *slog.Logger) *slog.Logger {
						return e.With("after", 2)
					},
					DoLog: func(c echo.Context, e *slog.Logger) {
						doLogCalled = true
						api.LoggingMiddlewareDefaultDoLog(c, e)
					},
				},
			})
			e.GET("/", func(c echo.Context) error {
				return c.String(400, "")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(400))
			Expect(doLogCalled).To(BeTrue())
			Expect(logHook.LastRecord().AttrMap()).To(And(
				HaveKeyWithValue("before", BeEquivalentTo(1)),
				HaveKeyWithValue("after", BeEquivalentTo(2)),
			))
		})
	})

	Describe("error handling", func() {
		It("handles panics", func() {
			e.GET("/test", func(c echo.Context) error {
				panic("hello")
			})
			req := GetRequest("/test")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(500))
			Expect(rr).To(HaveJsonBody(And(
				HaveKeyWithValue("http_status", BeEquivalentTo(500)),
				HaveKeyWithValue("error_code", BeEquivalentTo("internal_error")),
				HaveKeyWithValue("message", BeEquivalentTo("Internal Server Error")),
			)))
		})
		It("adapts unhandled errors", func() {
			e.GET("/test", func(c echo.Context) error {
				return errors.New("internal error")
			})
			req := GetRequest("/test")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(500))
			Expect(rr).To(HaveJsonBody(And(
				HaveKeyWithValue("http_status", BeEquivalentTo(500)),
				HaveKeyWithValue("error_code", BeEquivalentTo("internal_error")),
				HaveKeyWithValue("message", BeEquivalentTo("Internal Server Error")),
			)))
		})
		It("passes through api.Error instances", func() {
			e.GET("/test", func(c echo.Context) error {
				return api.NewError(429, "hello_teapot")
			})
			req := GetRequest("/test")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(429))
			Expect(rr).To(HaveJsonBody(And(
				HaveKeyWithValue("http_status", BeEquivalentTo(429)),
				HaveKeyWithValue("error_code", BeEquivalentTo("hello_teapot")),
			)))
		})
		It("adapts echo errors", func() {
			e.GET("/test", func(c echo.Context) error {
				return echo.NewHTTPError(428, "echo msg")
			})
			req := GetRequest("/test")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(428))
			Expect(rr).To(HaveJsonBody(And(
				HaveKeyWithValue("http_status", BeEquivalentTo(428)),
				HaveKeyWithValue("message", BeEquivalentTo("echo msg")),
			)))
		})
		It("adapts apiparams errors", func() {
			e.GET("/test", func(c echo.Context) error {
				return apiparams.NewHTTPError(428, "apiparams msg")
			})
			req := GetRequest("/test")
			rr := Serve(e, req)
			Expect(rr).To(HaveResponseCode(428))
			Expect(rr).To(HaveJsonBody(And(
				HaveKeyWithValue("http_status", BeEquivalentTo(428)),
				HaveKeyWithValue("message", BeEquivalentTo("apiparams msg")),
			)))
		})
	})

	Describe("adapting to standard context", func() {
		It("can adapt an echo.Context to a context.Context for portability", func() {
			r, err := http.NewRequest("GET", "", nil)
			Expect(err).ToNot(HaveOccurred())
			ctx := e.NewContext(r, httptest.NewRecorder())
			logger, _ := logctx.NewNullLogger()
			logger = logger.With("a", 2)
			api.SetLogger(ctx, logger)
			tid := api.TraceId(ctx)

			c := api.StdContext(ctx)
			tkey, tval := logctx.ActiveTraceId(c)
			Expect(tkey).To(Equal(logctx.RequestTraceIdKey))
			Expect(tval).To(Equal(tid))
			Expect(logctx.Logger(c).Handler().(*logctx.Hook).AttrMap()).To(And(
				HaveKeyWithValue("a", BeEquivalentTo(2)),
				HaveKeyWithValue(BeEquivalentTo(logctx.RequestTraceIdKey), tid),
			))
		})
	})

	Describe("CacheControl", func() {
		It("adds a cache-control header", func() {
			e.POST("/endpoint", func(c echo.Context) error {
				api.SetCacheControl(c)
				return c.String(200, "ok")
			}, api.WithCacheControl(true, "max-age=60"))
			resp := Serve(e, NewRequest("POST", "/endpoint", nil))
			Expect(resp).To(HaveResponseCode(200))
			Expect(resp.Header().Get("Cache-Control")).To(Equal("max-age=60"))
		})
		It("does not add a header if not configured", func() {
			e.POST("/endpoint", func(c echo.Context) error {
				return c.String(200, "ok")
			}, api.WithCacheControl(false, "max-age=60"))
			resp := Serve(e, NewRequest("POST", "/endpoint", nil))
			Expect(resp).To(HaveResponseCode(200))
			Expect(resp.Header().Get("Cache-Control")).To(BeEmpty())
		})
	})

	Describe("DebugMiddleware", func() {
		It("noops if not enabled", func() {
			e.Use(api.DebugMiddleware(api.DebugMiddlewareConfig{Enabled: false, DumpResponseBody: true}))
			e.GET("/foo", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Serve(e, NewRequest("POST", "/endpoint", nil))
			Expect(logHook.Records()).To(HaveLen(1))
			Expect(logHook.Records()[0].Record.Message).To(Equal("request_finished"))
		})
		It("dumps what is enabled", func() {
			e.Use(api.DebugMiddleware(api.DebugMiddlewareConfig{Enabled: true, DumpResponseBody: true, DumpResponseHeaders: true}))
			e.GET("/endpoint", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Serve(e, NewRequest("GET", "/endpoint", nil))
			Expect(logHook.Records()).To(HaveLen(2))
			Expect(logHook.Records()[0].Record.Message).To(Equal("request_debug"))
			Expect(logHook.Records()[0].AttrMap()).To(And(
				HaveKeyWithValue("debug_response_headers", HaveKey("Content-Type")),
				HaveKeyWithValue("debug_response_body", ContainSubstring("ok")),
			))
		})
		It("can dump everything", func() {
			e.Use(api.DebugMiddleware(api.DebugMiddlewareConfig{Enabled: true, DumpAll: true}))
			e.GET("/endpoint", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Serve(e, NewRequest("GET", "/endpoint", nil, SetReqHeader("Foo", "x")))
			Expect(logHook.Records()).To(HaveLen(2))
			Expect(logHook.Records()[0].Record.Message).To(Equal("request_debug"))
			Expect(logHook.Records()[0].AttrMap()).To(And(
				HaveKeyWithValue("debug_request_headers", HaveKey("Foo")),
				HaveKeyWithValue("debug_response_headers", HaveKey("Content-Type")),
				HaveKeyWithValue("debug_request_body", ""),
				HaveKeyWithValue("debug_response_body", ContainSubstring("ok")),
			))
		})
		It("can print memory stats every n requests", func() {
			e.Use(api.DebugMiddleware(api.DebugMiddlewareConfig{Enabled: true, DumpMemoryEvery: 2}))
			e.GET("/endpoint", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Serve(e, NewRequest("GET", "/endpoint", nil, SetReqHeader("Foo", "x")))
			Serve(e, NewRequest("GET", "/endpoint", nil, SetReqHeader("Foo", "x")))
			Expect(logHook.Records()).To(HaveLen(4))
			Expect(logHook.Records()[0].Record.Message).To(Equal("request_debug"))
			Expect(logHook.Records()[0].AttrMap()).ToNot(HaveKey("memory_sys"))
			Expect(logHook.Records()[1].Record.Message).To(Equal("request_finished"))
			Expect(logHook.Records()[2].Record.Message).To(Equal("request_debug"))
			Expect(logHook.Records()[2].AttrMap()).To(HaveKey("memory_sys"))
			Expect(logHook.Records()[3].Record.Message).To(Equal("request_finished"))
		})
	})
})
