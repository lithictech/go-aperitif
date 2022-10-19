package auth0jwt_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAuth0Jwt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("auth0jwt", func() {
	//var e *echo.Echo

	BeforeEach(func() {
		//e = echo.New()
	})

	It("validates against iss", func() {
	})
	It("validates against aud string", func() {
	})
	It("validates against aud array", func() {
	})
	It("adds the user to the echo context", func() {
	})
	It("validates against the PEM cert", func() {})
})
