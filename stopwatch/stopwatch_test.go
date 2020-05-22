package stopwatch_test

import (
	"github.com/lithictech/go-aperitif/stopwatch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"testing"
)

func TestStopwatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "stopwatch package Suite")
}

var _ = Describe("Stopwatch", func() {
	It("logs start and stop", func() {
		logger, hook := test.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		sw := stopwatch.Start(logger.WithFields(nil), "test")
		sw.Finish()
		Expect(hook.Entries).To(HaveLen(2))

		Expect(hook.Entries[0].Level).To(Equal(logrus.DebugLevel))
		Expect(hook.Entries[0].Message).To(ContainSubstring("test_started"))

		Expect(hook.Entries[1].Level).To(Equal(logrus.InfoLevel))
		Expect(hook.Entries[1].Message).To(ContainSubstring("test_finished"))
	})

	It("can use a custom finish logger", func() {
		startLogger, startHook := test.NewNullLogger()
		startLogger.SetLevel(logrus.DebugLevel)

		finishLogger, finishHook := test.NewNullLogger()
		finishLogger.SetLevel(logrus.DebugLevel)

		sw := stopwatch.Start(startLogger.WithFields(nil), "test")

		sw.FinishWith(stopwatch.FinishOpts{Logger: finishLogger.WithFields(nil)})

		Expect(startHook.Entries).To(HaveLen(1))
		Expect(finishHook.Entries).To(HaveLen(1))
	})
})
