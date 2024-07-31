package async_test

import (
	"github.com/lithictech/go-aperitif/v2/async"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAsync(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "async package Suite")
}

var _ = Describe("async", func() {
	Describe("SpyingGoer", func() {
		It("records the calls to the wrapped Goer", func() {
			g := async.NewSpying(async.Sync)
			async.NewSpying(g.Go) // Assert Spying.Go is async.Goer type
			innerCalls := 0
			g.Go("x", func() { innerCalls++ })
			g.Go("y", func() {})
			Expect(g.Calls).To(ConsistOf("x", "y"))
			Expect(g.CallCount).To(Equal(2))
			Expect(innerCalls).To(Equal(1))
		})
	})
})
