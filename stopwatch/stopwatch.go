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

func Start(logger *logrus.Entry, operation string) *Stopwatch {
	sw := &Stopwatch{
		start:     time.Now(),
		operation: operation,
		logger:    logger,
	}

	sw.logger.Debug(operation + "_started")
	return sw
}

type FinishOpts struct {
	Logger *logrus.Entry
}

func (sw *Stopwatch) FinishWith(opts FinishOpts) {
	logger := sw.logger
	if opts.Logger != nil {
		logger = opts.Logger
	}
	logger = logger.WithField("elapsed", time.Since(sw.start).Seconds())
	logger.Info(sw.operation + "_finished")
}

func (sw *Stopwatch) Finish() {
	sw.FinishWith(FinishOpts{})
}
