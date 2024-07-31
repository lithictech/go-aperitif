package stopwatch_test

import (
	"context"
	"github.com/lithictech/go-aperitif/logctx"
	"github.com/lithictech/go-aperitif/stopwatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"log/slog"
	"testing"
)

func TestStopwatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "stopwatch package Suite")
}

var _ = Describe("Stopwatch", func() {
	var logger *slog.Logger
	var hook *logctx.Hook
	var ctx context.Context

	BeforeEach(func() {
		logger, hook = logctx.NewNullLogger()
		ctx = logctx.WithLogger(context.Background(), logger)
	})

	It("logs start and stop", func() {
		sw := stopwatch.Start(ctx, logger, "test")
		sw.Finish(ctx)
		Expect(hook.Records()).To(HaveLen(2))

		Expect(hook.Records()[0].Record.Level).To(Equal(slog.LevelDebug))
		Expect(hook.Records()[0].Record.Message).To(ContainSubstring("test_started"))

		Expect(hook.Records()[1].Record.Level).To(Equal(slog.LevelInfo))
		Expect(hook.Records()[1].Record.Message).To(ContainSubstring("test_finished"))
	})

	It("can custom start and stop", func() {
		sw := stopwatch.StartWith(ctx, logger, "test", stopwatch.StartOpts{Level: slog.LevelWarn, Key: "_begin"})
		sw.FinishWith(ctx, stopwatch.FinishOpts{Level: slog.LevelError, Key: "_end", ElapsedKey: "timing"})
		Expect(hook.Records()).To(HaveLen(2))

		Expect(hook.Records()[0].Record.Level).To(Equal(slog.LevelWarn))
		Expect(hook.Records()[0].Record.Message).To(ContainSubstring("test_begin"))

		Expect(hook.Records()[1].Record.Level).To(Equal(slog.LevelError))
		Expect(hook.Records()[1].Record.Message).To(ContainSubstring("test_end"))
		Expect(hook.Records()[1].AttrMap()).To(HaveKey("timing"))
	})

	It("can use a custom finish logger", func() {
		startLogger, startHook := logctx.NewNullLogger()
		finishLogger, finishHook := logctx.NewNullLogger()

		sw := stopwatch.Start(ctx, startLogger, "test")

		sw.FinishWith(ctx, stopwatch.FinishOpts{Logger: finishLogger})

		Expect(startHook.Records()).To(HaveLen(1))
		Expect(finishHook.Records()).To(HaveLen(1))
	})

	It("can lap", func() {
		sw := stopwatch.Start(ctx, logger, "test")
		sw.Lap(ctx)
		sw.LapWith(ctx, stopwatch.LapOpts{Key: "_split", Level: slog.LevelWarn, ElapsedKey: "timing"})
		Expect(hook.Records()).To(HaveLen(3))

		Expect(hook.Records()[1].Record.Level).To(Equal(slog.LevelInfo))
		Expect(hook.Records()[1].Record.Message).To(ContainSubstring("test_lap"))
		Expect(hook.Records()[1].AttrMap()).To(HaveKey("elapsed"))

		Expect(hook.Records()[2].Record.Level).To(Equal(slog.LevelWarn))
		Expect(hook.Records()[2].Record.Message).To(ContainSubstring("test_split"))
		Expect(hook.Records()[2].AttrMap()).To(HaveKey("timing"))
	})
})
