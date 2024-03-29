package stopwatch_test

import (
	"github.com/lithictech/go-aperitif/stopwatch"
	. "github.com/onsi/ginkgo/v2"
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
	var logger *logrus.Logger
	var entry *logrus.Entry
	var hook *test.Hook

	BeforeEach(func() {
		logger, hook = test.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		entry = logger.WithFields(nil)
	})
	It("logs start and stop", func() {
		sw := stopwatch.Start(entry, "test")
		sw.Finish()
		Expect(hook.Entries).To(HaveLen(2))

		Expect(hook.Entries[0].Level).To(Equal(logrus.DebugLevel))
		Expect(hook.Entries[0].Message).To(ContainSubstring("test_started"))

		Expect(hook.Entries[1].Level).To(Equal(logrus.InfoLevel))
		Expect(hook.Entries[1].Message).To(ContainSubstring("test_finished"))
	})

	It("can custom start and stop", func() {
		sw := stopwatch.StartWith(entry, "test", stopwatch.StartOpts{Level: logrus.WarnLevel, Key: "_begin"})
		sw.FinishWith(stopwatch.FinishOpts{Level: logrus.ErrorLevel, Key: "_end", ElapsedKey: "timing"})
		Expect(hook.Entries).To(HaveLen(2))

		Expect(hook.Entries[0].Level).To(Equal(logrus.WarnLevel))
		Expect(hook.Entries[0].Message).To(ContainSubstring("test_begin"))

		Expect(hook.Entries[1].Level).To(Equal(logrus.ErrorLevel))
		Expect(hook.Entries[1].Message).To(ContainSubstring("test_end"))
		Expect(hook.Entries[1].Data).To(HaveKey("timing"))
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

	It("can lap", func() {
		sw := stopwatch.Start(entry, "test")
		sw.Lap()
		sw.LapWith(stopwatch.LapOpts{Key: "_split", Level: logrus.WarnLevel, ElapsedKey: "timing"})
		Expect(hook.Entries).To(HaveLen(3))

		Expect(hook.Entries[1].Level).To(Equal(logrus.InfoLevel))
		Expect(hook.Entries[1].Message).To(ContainSubstring("test_lap"))
		Expect(hook.Entries[1].Data).To(HaveKey("elapsed"))

		Expect(hook.Entries[2].Level).To(Equal(logrus.WarnLevel))
		Expect(hook.Entries[2].Message).To(ContainSubstring("test_split"))
		Expect(hook.Entries[2].Data).To(HaveKey("timing"))
	})
})
