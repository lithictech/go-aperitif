package spa_test

import (
	"github.com/labstack/echo"
	. "github.com/lithictech/go-aperitif/api/echoapitest"
	"github.com/lithictech/go-aperitif/api/spa"
	. "github.com/lithictech/go-aperitif/apitest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/rgalanakis/golangal"
	"net/http"
	"strings"
	"testing"
)

func TestSPA(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SPA Suite")
}

var _ = Describe("spa Middleware", func() {
	var e *echo.Echo
	skipV1 := func(r *http.Request) (bool, error) {
		return !strings.HasPrefix(r.URL.Path, "/v1/"), nil
	}

	noopMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	BeforeEach(func() {
		e = echo.New()
	})

	It("sends index.html to the Static middleware if predicate matches", func() {
		e.Use(spa.MiddlewareWithConfig(spa.Config{
			Handle: skipV1,
			Static: func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					return c.String(201, "from static")
				}
			},
		}))
		req := GetRequest("/some-callback")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(201))
		Expect(rr.Body.String()).To(Equal("from static"))
	})

	It("skips if path does not match predicate", func() {
		e.Use(spa.Middleware(noopMiddleware, skipV1))
		req := GetRequest("/v1/some-missing-endpoint")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(404))
	})

	It("does not mess with valid static routes", func() {
		e.Use(spa.Middleware(
			func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					return c.String(205, "from static")
				}
			}, skipV1))
		req := GetRequest("/v1/some-missing-endpoint")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(205))
	})

	It("does not mess with registered endpoints", func() {
		e.GET("/handled-route", func(c echo.Context) error {
			return c.String(200, "hi")
		})
		e.Use(spa.Middleware(noopMiddleware, skipV1))
		req := GetRequest("/handled-route")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(200))
		Expect(rr.Body.String()).To(BeEquivalentTo("hi"))
	})

	It("does not error if incorrectly configured (missing required config)", func() {
		e.Use(spa.MiddlewareWithConfig(spa.Config{}))
		req := GetRequest("/some-route")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(404))
	})
})
