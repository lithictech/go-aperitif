package quiz

import (
	"github.com/onsi/ginkgo"
	"math/rand"
	"os"
	"strconv"
	"time"
)

// TestingT is a testing.T compatible interface,
// like used for libraries like cupaloy.
type TestingT struct {
	ginkgo.GinkgoTInterface
	desc ginkgo.GinkgoTestDescription
}

func NewTestingT() TestingT {
	return TestingT{ginkgo.GinkgoT(), ginkgo.CurrentGinkgoTestDescription()}
}

func (i TestingT) Helper() {
}

func (i TestingT) Name() string {
	return i.desc.FullTestText
}

var Rand *rand.Rand

func init() {
	seed, err := strconv.ParseInt(os.Getenv("RAND_SEED"), 10, 64)
	if err != nil {
		seed = time.Now().UnixNano()
	}
	Rand = rand.New(rand.NewSource(seed))
}
