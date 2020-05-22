// Package kronos are time utilities.
package kronos

import "time"

// TMax returns the later of two time instants.
func TMax(t, u time.Time) time.Time {
	if t.After(u) {
		return t
	}
	return u
}

// TMin returns the earlier of two time instants.
func TMin(t, u time.Time) time.Time {
	if t.Before(u) {
		return t
	}
	return u
}
