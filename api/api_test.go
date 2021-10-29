package api_test

import (
	"errors"
	"github.com/labstack/echo"
	"github.com/lithictech/go-aperitif/api"
	"github.com/lithictech/go-aperitif/api/apiparams"
	. "github.com/lithictech/go-aperitif/api/echoapitest"
	. "github.com/lithictech/go-aperitif/apitest"
	"github.com/lithictech/go-aperitif/logctx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/rgalanakis/golangal"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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
	var logger *logrus.Logger
	var logHook *test.Hook
	var logEntry *logrus.Entry

	BeforeEach(func() {
		logger, logHook = test.NewNullLogger()
		logEntry = logger.WithFields(nil)
		e = api.New(api.Config{
			Logger:         logEntry,
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

	It("can use a custom health handler", func() {
		e = api.New(api.Config{
			Logger: logEntry,
			HealthHandler: func(c echo.Context) error {
				return c.String(200, "yo")
			},
		})
		rr := Serve(e, GetRequest("/healthz"))
		Expect(rr).To(HaveResponseCode(200))
		Expect(rr.Body.String()).To(Equal("yo"))
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
				Expect(api.Logger(c).Data).ToNot(HaveKey("request_status"))
				return c.String(401, "ok")
			})
			e.GET("/after-first-call", func(c echo.Context) error {
				Expect(api.Logger(c).Data).ToNot(HaveKey("request_status"))
				return c.String(403, "ok")
			})
			Expect(Serve(e, GetRequest("/before-first-call"))).To(HaveResponseCode(401))
			Expect(Serve(e, GetRequest("/after-first-call"))).To(HaveResponseCode(403))
			Expect(logHook.Entries).To(HaveLen(2))
		})
		It("logs normal requests at info", func() {
			e.GET("/", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(200))
			Expect(logHook.Entries).To(HaveLen(1))
			Expect(logHook.Entries[0].Level).To(Equal(logrus.InfoLevel))
		})
		It("logs 500+ at error", func() {
			e.GET("/", func(c echo.Context) error {
				return c.String(500, "oh")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(500))
			Expect(logHook.Entries).To(HaveLen(1))
			Expect(logHook.Entries[0].Level).To(Equal(logrus.ErrorLevel))
		})
		It("logs 400 to 499 as warn", func() {
			e.GET("/", func(c echo.Context) error {
				return c.String(400, "client err")
			})
			Expect(Serve(e, GetRequest("/"))).To(HaveResponseCode(400))
			Expect(logHook.Entries).To(HaveLen(1))
			Expect(logHook.Entries[0].Level).To(Equal(logrus.WarnLevel))
		})
		It("logs status and health as debug", func() {
			logger.SetLevel(logrus.DebugLevel)
			Expect(Serve(e, GetRequest("/healthz"))).To(HaveResponseCode(200))
			Expect(Serve(e, GetRequest("/statusz"))).To(HaveResponseCode(200))
			Expect(logHook.Entries).To(HaveLen(2))
			Expect(logHook.Entries[0].Level).To(Equal(logrus.DebugLevel))
			Expect(logHook.Entries[1].Level).To(Equal(logrus.DebugLevel))
		})
		It("logs options as debug", func() {
			logger.SetLevel(logrus.DebugLevel)
			Expect(Serve(e, NewRequest("OPTIONS", "/foo", nil))).To(HaveResponseCode(404))
			Expect(logHook.Entries).To(HaveLen(1))
			Expect(logHook.Entries[0].Level).To(Equal(logrus.DebugLevel))
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
			logger := logrus.New().WithField("a", 2)
			api.SetLogger(ctx, logger)
			tid := api.TraceId(ctx)

			c := api.StdContext(ctx)
			tkey, tval := logctx.ActiveTraceId(c)
			Expect(tkey).To(Equal(logctx.RequestTraceIdKey))
			Expect(tval).To(Equal(tid))
			Expect(logctx.Logger(c).Data).To(And(
				HaveKeyWithValue("a", 2),
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
		BeforeEach(func() {
			logger.SetLevel(logrus.DebugLevel)
		})
		It("noops if not enabled", func() {
			e.Use(api.DebugMiddleware(api.DebugMiddlewareConfig{Enabled: false, DumpResponseBody: true}))
			e.GET("/foo", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Serve(e, NewRequest("POST", "/endpoint", nil))
			Expect(logHook.Entries).To(HaveLen(1))
			Expect(logHook.Entries[0].Message).To(Equal("request_finished"))
		})
		It("dumps what is enabled", func() {
			e.Use(api.DebugMiddleware(api.DebugMiddlewareConfig{Enabled: true, DumpResponseBody: true, DumpResponseHeaders: true}))
			e.GET("/endpoint", func(c echo.Context) error {
				return c.String(200, "ok")
			})
			Serve(e, NewRequest("GET", "/endpoint", nil))
			Expect(logHook.Entries).To(HaveLen(2))
			Expect(logHook.Entries[0].Message).To(Equal("request_debug"))
			Expect(logHook.Entries[0].Data).To(And(
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
			Expect(logHook.Entries).To(HaveLen(2))
			Expect(logHook.Entries[0].Message).To(Equal("request_debug"))
			Expect(logHook.Entries[0].Data).To(And(
				HaveKeyWithValue("debug_request_headers", HaveKey("Foo")),
				HaveKeyWithValue("debug_response_headers", HaveKey("Content-Type")),
				HaveKeyWithValue("debug_request_body", ""),
				HaveKeyWithValue("debug_response_body", ContainSubstring("ok")),
			))
		})
	})
})
