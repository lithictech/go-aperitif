// Package mariobros is useful for monitoring Goroutine *leaks* from your code (get it? leaks?).
//
// To use it, call mariobros.Start early in your process. mariobros.Start(mariobros.NewOptions()) should work
// fine locally, but you may want to supply a custom writer if you want the reports in your logs.
//
// Then wherever you start a Goroutine, call Mariobros.Yo with a unique identifier for
// the operation, like 'level1.area4.lavapit'. Then defer the callback returned from that function,
// like:
//
//	go func() {
//	    mario := mariobros.Yo("level1.area4.lavapit")
//	    defer mario()
//	    // Do more stuff...
//	}
//
// Every 5 seconds (or whatever is configured), mariobros will report on the active goroutines.
//
// You can configure mariobros when you call Start. NewOptions by default writes to stdout,
// and will read an integer value from the MARIOBROS envvar if specified
// (ie, use MARIOBROS=1 to poll every second).
// You can specify your own overrides.
//
// If Mariobros is not active, calls to Yo noop and the timer that prints does not run.
// It's important to call Mariobros.Start() early, or import mariobros/autoload.
package mariobros

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type GoroutineId uint
type Writer func(totalActive uint, activePerName map[string][]GoroutineId)

// StreamWriter is used when you want to write mariobros output to a stream.
// The output you get is like:
//
//	active goroutines (1):
//	  my.job: 1, 5
//	  other.job: 6, 7
func StreamWriter(w io.Writer) Writer {
	return func(totalActive uint, activePerName map[string][]GoroutineId) {
		w := func(s string, a ...interface{}) {
			_, _ = w.Write([]byte(fmt.Sprintf(s, a...)))
		}
		w("active goroutines (%d):\n", totalActive)
		for name, active := range activePerName {
			if len(active) > 0 {
				keys := make([]string, 0, len(active))
				for goroutineId := range active {
					keys = append(keys, fmt.Sprintf("%d", goroutineId))
				}
				w("  %s: %s\n", name, strings.Join(keys, ","))
			}
		}
	}
}

// KeyValueWriter is helpful when you want to log a structured message.
// For example:
//
//	    mariobros.Start(mariobros.NewOptions(func(o *mariobros.Options) {
//		       o.Writer = mariobros.KeyValueWriter("mariobros_", func(m map[string]interface{}) {
//	            logger.WithFields(m).Info("mariobros")
//	        })
//	    }))
func KeyValueWriter(keyPrefix string, write func(map[string]interface{})) Writer {
	return func(totalActive uint, activePerName map[string][]GoroutineId) {
		result := make(map[string]interface{}, len(activePerName)+1)
		result[keyPrefix+"total"] = totalActive
		for name, active := range activePerName {
			result[keyPrefix+name] = active
		}
		write(result)
	}
}

func noop() {}

var instance *mariobros

type mariobros struct {
	mutex             *sync.Mutex
	goroutineIndex    GoroutineId
	activeGoroutines  uint
	goroutineRegistry map[string]map[GoroutineId]struct{}
	enabledFast       int64
	writer            Writer
	interval          time.Duration
}

func newMariobros() *mariobros {
	return &mariobros{
		mutex:             &sync.Mutex{},
		goroutineIndex:    0,
		activeGoroutines:  0,
		goroutineRegistry: make(map[string]map[GoroutineId]struct{}, 16),
		enabledFast:       0,
	}
}

func (mb *mariobros) Start(opts Options) {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	if mb.enabledFast == 1 {
		return
	}
	atomic.StoreInt64(&mb.enabledFast, 1)
	mb.interval = opts.Interval
	mb.writer = opts.Writer
	t := time.NewTicker(mb.interval)
	go func() {
		for {
			select {
			case <-t.C:
			}
			activePerName := make(map[string][]GoroutineId, len(mb.goroutineRegistry)+1)
			for name, active := range mb.goroutineRegistry {
				if len(active) > 0 {
					for id := range active {
						activePerName[name] = append(activePerName[name], id)
					}
				}
			}
			mb.writer(mb.activeGoroutines, activePerName)
		}
	}()
}

func (mb *mariobros) Yo(name string) func() {
	if atomic.LoadInt64(&mb.enabledFast) == 0 {
		return noop
	}
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	if atomic.LoadInt64(&mb.enabledFast) == 0 {
		return noop
	}
	mb.goroutineIndex++
	mb.activeGoroutines++
	thisId := mb.goroutineIndex
	nameRegistry := mb.goroutineRegistry[name]
	if nameRegistry == nil {
		nameRegistry = make(map[GoroutineId]struct{}, 16)
		mb.goroutineRegistry[name] = nameRegistry
	}
	nameRegistry[thisId] = struct{}{}
	return func() {
		mb.mutex.Lock()
		delete(nameRegistry, thisId)
		mb.activeGoroutines--
		mb.mutex.Unlock()
	}
}

func init() {
	instance = newMariobros()
}

func Start(opts Options) {
	instance.Start(opts)
}

func Yo(name string) func() {
	return instance.Yo(name)
}

type Options struct {
	Interval time.Duration
	Writer   Writer
}

type OptionModifier func(*Options)

func NewOptions(mods ...OptionModifier) Options {
	opts := Options{}
	opts.Interval = time.Second * 5
	if iv, err := strconv.Atoi(os.Getenv("MARIOBROS")); err == nil {
		opts.Interval = time.Second * time.Duration(iv)
	}
	opts.Writer = StreamWriter(os.Stdout)
	for _, m := range mods {
		m(&opts)
	}
	return opts
}
