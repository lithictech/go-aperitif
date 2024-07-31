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
	"context"
	"log/slog"
	"time"
)

type Stopwatch struct {
	start     time.Time
	operation string
	logger    *slog.Logger
}

type StartOpts struct {
	Key   string
	Level slog.Level
}

func StartWith(ctx context.Context, logger *slog.Logger, operation string, opts StartOpts) *Stopwatch {
	if opts.Key == "" {
		opts.Key = "_started"
	}
	if opts.Level == 0 {
		opts.Level = slog.LevelDebug
	}
	sw := &Stopwatch{
		start:     time.Now(),
		operation: operation,
		logger:    logger,
	}

	sw.logger.Log(ctx, opts.Level, operation+opts.Key)
	return sw
}

func Start(ctx context.Context, logger *slog.Logger, operation string) *Stopwatch {
	return StartWith(ctx, logger, operation, StartOpts{})
}

type FinishOpts struct {
	Logger       *slog.Logger
	Key          string
	ElapsedKey   string
	Milliseconds bool
	Level        slog.Level
}

func (sw *Stopwatch) FinishWith(ctx context.Context, opts FinishOpts) {
	if opts.Key == "" {
		opts.Key = "_finished"
	}
	if opts.ElapsedKey == "" {
		opts.ElapsedKey = "elapsed"
	}
	if opts.Level == 0 {
		opts.Level = slog.LevelInfo
	}
	if opts.Logger == nil {
		opts.Logger = sw.logger
	}
	logger := opts.Logger
	if opts.Milliseconds {
		logger = logger.With(opts.ElapsedKey, time.Since(sw.start).Milliseconds())
	} else {
		logger = logger.With(opts.ElapsedKey, time.Since(sw.start).Seconds())
	}
	logger.Log(ctx, opts.Level, sw.operation+opts.Key)
}

func (sw *Stopwatch) Finish(ctx context.Context) {
	sw.FinishWith(ctx, FinishOpts{})
}

type LapOpts FinishOpts

func (sw *Stopwatch) LapWith(ctx context.Context, opts LapOpts) {
	if opts.Key == "" {
		opts.Key = "_lap"
	}
	if opts.ElapsedKey == "" {
		opts.ElapsedKey = "elapsed"
	}
	if opts.Level == 0 {
		opts.Level = slog.LevelInfo
	}
	if opts.Logger == nil {
		opts.Logger = sw.logger
	}
	sw.FinishWith(ctx, FinishOpts(opts))
}

func (sw *Stopwatch) Lap(ctx context.Context) {
	sw.LapWith(ctx, LapOpts{})
}
