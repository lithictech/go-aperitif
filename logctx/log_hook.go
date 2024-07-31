package logctx

import (
	"context"
	"log/slog"
	"sync"
)

type HookRecord struct {
	Record slog.Record
	Attrs  []slog.Attr
	Group  string
}

func (r HookRecord) AttrMap() map[string]any {
	return attrMap(r.Attrs)
}

func NewHook() *Hook {
	return &Hook{records: &hookRecords{r: make([]HookRecord, 0, 4)}}
}

// Hook is a hook designed for dealing with logs in test scenarios.
type Hook struct {
	records *hookRecords
	attrs   []slog.Attr
	group   string
}

var _ slog.Handler = &Hook{}

func (t *Hook) Enabled(context.Context, slog.Level) bool {
	return true
}

func (t *Hook) Handle(_ context.Context, r slog.Record) error {
	t.records.Add(HookRecord{Record: r, Attrs: t.attrs, Group: t.group})
	return nil
}

func (t *Hook) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Hook{
		records: t.records,
		attrs:   append(t.attrs, attrs...),
		group:   t.group,
	}
}

func (t *Hook) WithGroup(group string) slog.Handler {
	return &Hook{
		records: t.records,
		attrs:   t.attrs,
		group:   group,
	}
}

// LastRecord returns the last record that was logged or nil.
func (t *Hook) LastRecord() *HookRecord {
	return t.records.LastRecord()
}

// Records returns all records that were logged.
func (t *Hook) Records() []HookRecord {
	return t.records.Records()
}

func (t *Hook) AttrMap() map[string]any {
	return attrMap(t.attrs)
}

type hookRecords struct {
	r   []HookRecord
	mux sync.RWMutex
}

func (h *hookRecords) Add(r HookRecord) {
	h.mux.Lock()
	defer h.mux.Unlock()
	h.r = append(h.r, r)
}

func (h *hookRecords) Records() []HookRecord {
	h.mux.RLock()
	defer h.mux.RUnlock()
	entries := make([]HookRecord, len(h.r))
	for i, rec := range h.r {
		entries[i] = rec
	}
	return entries
}

func (h *hookRecords) LastRecord() *HookRecord {
	h.mux.RLock()
	defer h.mux.RUnlock()
	i := len(h.r) - 1
	if i < 0 {
		return nil
	}
	return &h.r[i]
}

func attrMap(attrs []slog.Attr) map[string]any {
	result := make(map[string]any, len(attrs))
	for _, a := range attrs {
		result[a.Key] = a.Value.Any()
	}
	return result
}
