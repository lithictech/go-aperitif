package stringutil_test

import (
	"github.com/lithictech/go-aperitif/stringutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"
	"testing"
)

func TestStringUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "stringutil Suite")
}

var _ = Describe("Map", func() {
	It("maps the input slice", func() {
		s := []string{"a", "b"}
		res := stringutil.Map(s, strings.ToUpper)
		Expect(res).To(Equal([]string{"A", "B"}))
	})
})

var _ = Describe("Contains", func() {
	It("is true if the slice contains the string", func() {
		s := []string{"a", "b"}
		Expect(stringutil.Contains(s, "a")).To(BeTrue())
		Expect(stringutil.Contains(s, "A")).To(BeFalse())
	})
})
