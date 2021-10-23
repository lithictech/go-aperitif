/*
Package stopwatch is used to time things.
Create a stopwatch with Start,
then on success record the timing with Finish.

It's recommended you do not record errors,
since they can have vastly different timings
(maybe we add an Error method in the future and record a different event).
*/
package stopwatch

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Stopwatch struct {
	start     time.Time
	operation string
	logger    *logrus.Entry
}

type StartOpts struct {
	Key   string
	Level logrus.Level
}

func StartWith(logger *logrus.Entry, operation string, opts StartOpts) *Stopwatch {
	if opts.Key == "" {
		opts.Key = "_started"
	}
	if opts.Level == 0 {
		opts.Level = logrus.DebugLevel
	}
	sw := &Stopwatch{
		start:     time.Now(),
		operation: operation,
		logger:    logger,
	}

	sw.logger.Log(opts.Level, operation+opts.Key)
	return sw
}

func Start(logger *logrus.Entry, operation string) *Stopwatch {
	return StartWith(logger, operation, StartOpts{})
}

type FinishOpts struct {
	Logger       *logrus.Entry
	Key          string
	ElapsedKey   string
	Milliseconds bool
	Level        logrus.Level
}

func (sw *Stopwatch) FinishWith(opts FinishOpts) {
	if opts.Key == "" {
		opts.Key = "_finished"
	}
	if opts.ElapsedKey == "" {
		opts.ElapsedKey = "elapsed"
	}
	if opts.Level == 0 {
		opts.Level = logrus.InfoLevel
	}
	if opts.Logger == nil {
		opts.Logger = sw.logger
	}
	logger := opts.Logger
	if opts.Milliseconds {
		logger = logger.WithField(opts.ElapsedKey, time.Since(sw.start).Milliseconds())
	} else {
		logger = logger.WithField(opts.ElapsedKey, time.Since(sw.start).Seconds())
	}
	logger.Log(opts.Level, sw.operation+opts.Key)
}

func (sw *Stopwatch) Finish() {
	sw.FinishWith(FinishOpts{})
}

type LapOpts FinishOpts

func (sw *Stopwatch) LapWith(opts LapOpts) {
	if opts.Key == "" {
		opts.Key = "_lap"
	}
	if opts.ElapsedKey == "" {
		opts.ElapsedKey = "elapsed"
	}
	if opts.Level == 0 {
		opts.Level = logrus.InfoLevel
	}
	if opts.Logger == nil {
		opts.Logger = sw.logger
	}
	sw.FinishWith(FinishOpts(opts))
}

func (sw *Stopwatch) Lap() {
	sw.LapWith(LapOpts{})
}
