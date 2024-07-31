package preflight

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"time"
)

type Config struct {
	// The preflight check to execute.
	Check echo.HandlerFunc
	// Preflight checks will never wait longer than this amount of time.
	MaxTotalWait time.Duration
	// Retries will never be further than this far apart.
	MaxRetryWait time.Duration
}

func Middleware(check echo.HandlerFunc) echo.MiddlewareFunc {
	return MiddlewareWithConfig(Config{Check: check})
}

func MiddlewareWithConfig(cfg Config) echo.MiddlewareFunc {
	if cfg.MaxTotalWait == 0 {
		cfg.MaxTotalWait = time.Second * 30
	}
	if cfg.MaxRetryWait == 0 {
		cfg.MaxRetryWait = time.Second * 2
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		if cfg.Check == nil {
			return func(c echo.Context) error {
				return errors.New("preflight check not configured")
			}
		}
		return func(c echo.Context) error {
			// If preflight checks pass, go right on ahead
			if checkErr := cfg.Check(c); checkErr == nil {
				return next(c)
			}
			// If they don't pass, we need to set up some retries.
			// Record the start and end time; then for each retry,
			// double the time we wait (or use the max time if smaller).
			// If we ever get nil for a check, keep going.
			started := time.Now()
			giveUpAt := started.Add(cfg.MaxTotalWait)
			retryWait := 50 * time.Millisecond
			for {
				time.Sleep(retryWait)
				checkErr := cfg.Check(c)
				if checkErr == nil {
					return next(c)
				}
				if time.Now().After(giveUpAt) {
					return errors.Wrap(checkErr, "preflight checks failed")
				}
				retryWait *= 2
				if retryWait > cfg.MaxRetryWait {
					retryWait = cfg.MaxRetryWait
				}
			}
		}
	}

}
