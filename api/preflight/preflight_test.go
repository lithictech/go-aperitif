package preflight_test

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/lithictech/go-aperitif/v2/api"
	. "github.com/lithictech/go-aperitif/v2/api/echoapitest"
	"github.com/lithictech/go-aperitif/v2/api/preflight"
	. "github.com/lithictech/go-aperitif/v2/apitest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rgalanakis/golangal"
	"testing"
	"time"
)

func TestPreflight(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "preflight Suite")
}

var _ = Describe("preflight", func() {
	var e *echo.Echo

	noop := func(c echo.Context) error {
		return c.NoContent(204)
	}

	BeforeEach(func() {
		e = api.New(api.Config{})
	})

	It("retries for the specified time if the preflight check errors", func() {
		calls := 0
		e.GET("/", noop, preflight.MiddlewareWithConfig(preflight.Config{
			Check: func(c echo.Context) error {
				calls++
				return errors.New("nope")
			},
			MaxTotalWait: time.Millisecond * 100,
			MaxRetryWait: time.Millisecond,
		}))
		req := GetRequest("/")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(500))
		Expect(rr.Body.String()).To(ContainSubstring("nope"))
		Expect(rr.Body.String()).To(ContainSubstring("preflight checks failed"))
		Expect(calls).To(BeNumerically(">=", 2))
	})
	It("calls through if the preflight check does not error", func() {
		calls := 0
		e.GET("/", noop, preflight.MiddlewareWithConfig(preflight.Config{
			Check: func(c echo.Context) error {
				calls++
				return nil
			},
		}))
		req := GetRequest("/")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(204))
		Expect(calls).To(BeEquivalentTo(1))
	})
	It("recovers if the preflight check starts passing", func() {
		calls := 0
		e.GET("/", noop, preflight.MiddlewareWithConfig(preflight.Config{
			Check: func(c echo.Context) error {
				calls++
				if calls > 4 {
					return nil
				}
				return errors.New("nope")
			},
			MaxRetryWait: time.Millisecond,
		}))
		req := GetRequest("/")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(204))
		Expect(calls).To(BeEquivalentTo(5))
	})
	It("errors if the check is not defined", func() {
		e.GET("/", noop, preflight.Middleware(nil))
		req := GetRequest("/")
		rr := Serve(e, req)
		Expect(rr).To(HaveResponseCode(500))
		Expect(rr.Body.String()).To(ContainSubstring("preflight check not configured"))
	})
})
