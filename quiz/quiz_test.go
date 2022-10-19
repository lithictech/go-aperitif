package quiz_test

import (
	"github.com/lithictech/go-aperitif/quiz"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestQuiz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "quiz package Suite")
}

var _ = Describe("Quiz", func() {
	It("sets Rand", func() {
		Expect(quiz.Rand).ToNot(BeNil())
	})
})
