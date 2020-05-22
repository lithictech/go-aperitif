package parallel_test

import (
	"github.com/lithictech/aperitif/parallel"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync"
	"testing"
)

func TestParallel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "parallel package Suite")
}

var _ = Describe("ParallelFor", func() {
	It("processes in parallel", func() {
		mux := sync.Mutex{}
		active := 0
		called := 0
		err := parallel.ForEach(1000, 2, func(idx int) error {
			mux.Lock()
			active += 1
			called += 1
			mux.Unlock()

			mux.Lock()
			Expect(active).To(BeNumerically("<=", 2))
			mux.Unlock()

			mux.Lock()
			active -= 1
			mux.Unlock()
			return nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(called).To(Equal(1000))
		Expect(active).To(Equal(0))
	})
})
