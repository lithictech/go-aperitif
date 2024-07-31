package logctx_test

import (
	"context"
	"github.com/lithictech/go-aperitif/v2/logctx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"log/slog"
	"testing"
)

func TestLogtools(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logtools Suite")
}

var _ = Describe("logtools", func() {
	var logger *slog.Logger
	var hook *logctx.Hook
	var ctx context.Context

	BeforeEach(func() {
		logger, hook = logctx.NewNullLogger()
		ctx = logctx.WithLogger(context.Background(), logger)
	})

	Describe("WithTraceId", func() {
		It("adds a new trace id", func() {
			c := logctx.WithTraceId(ctx, logctx.ProcessTraceIdKey)
			Expect(c.Value(logctx.ProcessTraceIdKey)).To(HaveLen(36))
		})
	})

	Describe("ActiveTraceId", func() {
		It("returns a request trace id", func() {
			c := context.WithValue(ctx, logctx.RequestTraceIdKey, "abc")
			key, val := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.RequestTraceIdKey))
			Expect(val).To(Equal("abc"))
			Expect(logctx.ActiveTraceIdValue(c)).To(Equal("abc"))
		})
		It("returns a process trace id", func() {
			c := context.WithValue(ctx, logctx.ProcessTraceIdKey, "abc")
			key, val := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.ProcessTraceIdKey))
			Expect(val).To(Equal("abc"))
		})
		It("returns a process trace id", func() {
			c := context.WithValue(ctx, logctx.JobTraceIdKey, "abc")
			key, val := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.JobTraceIdKey))
			Expect(val).To(Equal("abc"))
		})
		It("prefers request->job->process trace id", func() {
			c := ctx

			c = context.WithValue(c, logctx.ProcessTraceIdKey, "proc")
			key, _ := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.ProcessTraceIdKey))

			c = context.WithValue(c, logctx.JobTraceIdKey, "job")
			key, _ = logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.JobTraceIdKey))

			c = context.WithValue(c, logctx.RequestTraceIdKey, "req")
			key, _ = logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.RequestTraceIdKey))
		})
		It("defaults to a missing trace id", func() {
			key, val := logctx.ActiveTraceId(ctx)
			Expect(key).To(Equal(logctx.MissingTraceIdKey))
			Expect(val).To(Equal("no-trace-id-in-context"))
		})
	})

	Describe("WithLogger", func() {
		It("adds the logger", func() {
			c := logctx.WithLogger(context.Background(), logger)
			Expect(c.Value(logctx.LoggerKey)).To(BeAssignableToTypeOf(logger))
		})
	})

	Describe("WithTracingLogger", func() {
		It("adds a trace id to the logger", func() {
			c := logctx.WithTracingLogger(logctx.WithTraceId(ctx, logctx.RequestTraceIdKey))
			logctx.Logger(c).Info("hi")
			Expect(hook.LastRecord().AttrMap()).To(HaveKeyWithValue(BeEquivalentTo(logctx.RequestTraceIdKey), BeAssignableToTypeOf("")))
		})
	})

	Describe("AddTo", func() {
		It("returns a new context where the given fields have been added to the context logger", func() {
			c := logctx.AddTo(ctx, "x", "y")
			Expect(logctx.Logger(c).Handler().(*logctx.Hook).AttrMap()).To(HaveKeyWithValue("x", "y"))
			logctx.Logger(c).Info("hi")
			Expect(hook.LastRecord().AttrMap()).To(HaveKeyWithValue("x", "y"))
		})
	})

	Describe("AddToR", func() {
		It("returns the new context, and the logger that was added", func() {
			c, logger := logctx.AddToR(ctx, "x", "y")
			logctx.Logger(c).Info("hi")
			Expect(hook.LastRecord().AttrMap()).To(HaveKeyWithValue("x", "y"))
			Expect(logger).To(BeIdenticalTo(logctx.Logger(c)))
		})
	})

	Describe("WithNullLogger", func() {
		It("inserts the null logger", func() {
			c, hook := logctx.WithNullLogger(nil)
			logctx.Logger(c).Info("hi")
			Expect(hook.Records()).To(HaveLen(1))
			Expect(hook.LastRecord().Record.Message).To(Equal("hi"))
		})
	})
})
