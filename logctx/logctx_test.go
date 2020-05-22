package logctx_test

import (
	"context"
	"github.com/lithictech/aperitif/logctx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestLogtools(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logtools Suite")
}

var _ = Describe("logtools", func() {
	bg := context.Background()

	Describe("WithTraceId", func() {
		It("adds a new trace id", func() {
			c := logctx.WithTraceId(bg, logctx.ProcessTraceIdKey)
			Expect(c.Value(logctx.ProcessTraceIdKey)).To(HaveLen(36))
		})
	})

	Describe("ActiveTraceId", func() {
		It("returns a request trace id", func() {
			c := context.WithValue(bg, logctx.RequestTraceIdKey, "abc")
			key, val := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.RequestTraceIdKey))
			Expect(val).To(Equal("abc"))
		})
		It("returns a process trace id", func() {
			c := context.WithValue(bg, logctx.ProcessTraceIdKey, "abc")
			key, val := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.ProcessTraceIdKey))
			Expect(val).To(Equal("abc"))
		})
		It("returns a process trace id", func() {
			c := context.WithValue(bg, logctx.JobTraceIdKey, "abc")
			key, val := logctx.ActiveTraceId(c)
			Expect(key).To(Equal(logctx.JobTraceIdKey))
			Expect(val).To(Equal("abc"))
		})
		It("prefers request->job->process trace id", func() {
			c := bg

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
			key, val := logctx.ActiveTraceId(bg)
			Expect(key).To(Equal(logctx.MissingTraceIdKey))
			Expect(val).To(Equal("no-trace-id-in-context"))
		})
	})

	Describe("WithLogger", func() {
		It("adds the logger", func() {
			logger := &logrus.Entry{}
			c := logctx.WithLogger(context.Background(), logger)
			Expect(c.Value(logctx.LoggerKey)).To(BeAssignableToTypeOf(logger))
		})
	})

	Describe("WithTracingLogger", func() {
		It("adds a trace id to the logger", func() {
			c := logctx.WithTracingLogger(logctx.WithTraceId(bg, logctx.RequestTraceIdKey))
			logger := c.Value(logctx.LoggerKey).(*logrus.Entry)
			Expect(logger.Data).To(HaveKeyWithValue(BeEquivalentTo(logctx.RequestTraceIdKey), BeAssignableToTypeOf("")))
		})
	})

	Describe("AddFields", func() {
		It("returns a new context where the given fields have been added to the context logger", func() {
			c := logctx.AddField(bg, "x", "y")
			logger := logctx.Logger(c)
			Expect(logger.Data).To(HaveKeyWithValue("x", "y"))
		})
	})

	Describe("AddFieldsAndGet", func() {
		It("returns the new context, and the logger that was added", func() {
			c, logger := logctx.AddFieldAndGet(bg, "x", "y")
			Expect(logger.Data).To(HaveKeyWithValue("x", "y"))
			Expect(logger).To(BeIdenticalTo(logctx.Logger(c)))
		})
	})
})
