package logctx

import (
	"context"
	"log/slog"
)

func NewTracingHandler(h slog.Handler) slog.Handler {
	return &TracingHandler{
		h:             h,
		TraceIdLogKey: "trace_id",
		SpanIdLogKey:  "span_id",
		GetTraceId: func(ctx context.Context) any {
			k, t := ActiveTraceId(ctx)
			if k == MissingTraceIdKey {
				return nil
			}
			return t
		},
		GetSpanId: func(ctx context.Context) any {
			return ctx.Value(SpanIdKey)
		},
	}
}

type TracingHandler struct {
	h             slog.Handler
	TraceIdLogKey string
	SpanIdLogKey  string
	GetTraceId    func(context.Context) any
	GetSpanId     func(context.Context) any
}

func (t *TracingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return t.h.Enabled(ctx, level)
}

func (t *TracingHandler) Handle(ctx context.Context, record slog.Record) error {
	if tid := t.GetTraceId(ctx); tid != nil {
		record.Add(t.TraceIdLogKey, tid)
	}
	if sid := t.GetSpanId(ctx); sid != nil {
		record.Add(t.SpanIdLogKey, sid)
	}
	return t.h.Handle(ctx, record)
}

func (t *TracingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewTracingHandler(t.h.WithAttrs(attrs))
}

func (t *TracingHandler) WithGroup(name string) slog.Handler {
	return NewTracingHandler(t.h.WithGroup(name))
}

var _ slog.Handler = &TracingHandler{}
