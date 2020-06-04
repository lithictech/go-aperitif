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

// Between returns a slice of Times between the given start and end at each interval.
// start and end are inclusive.
// If end is before start, nil is returned.
// If end is earlier than an interval from start, a slice containing only start is returned.
func Between(start, end time.Time, interval time.Duration) []time.Time {
	// Pre-allocate the slice. We should be able to exactly know how big the slice needs to be by
	// dividing the total distance by the interval/step distance.
	result := makeTimeSlice(int(end.Sub(start)), int(interval))
	BetweenEach(start, end, interval, func(t time.Time) {
		result = append(result, t)
	})
	return result
}

// BetweenEach calls each for every time between start and end.
// See Between for more information.
func BetweenEach(start, end time.Time, interval time.Duration, each func(time.Time)) {
	for t := start; !t.After(end); t = t.Add(interval) {
		each(t)
	}
}

// BetweenDates returns a slice of Times between the given start and end dates,
// adding the given years/months/days between each iteration.
// start and end are inclusive.
// If end is before start, nil is returned.
// If end is earlier than an interval after start, a slice containing only start is returned.
func BetweenDates(start, end time.Time, offsetYear, offsetMonth, offsetDay int) []time.Time {
	// Pre-allocate a slice by dividing the the total time/distance in days
	// by the step in days, and adding 1 for the start date.
	// We use 31-day months so we allocate the max possible slice by default.
	// It is likely better to over-allocate by a few, than underestimate and
	// grow the underlying array which would way overshoot the necessary size.
	distanceInDays := end.Sub(start) / (24 * time.Hour)
	stepInDays := (offsetYear * 12 * 30) + (offsetMonth * 30) + offsetDay
	result := makeTimeSlice(int(distanceInDays), stepInDays)

	BetweenDatesEach(start, end, offsetYear, offsetMonth, offsetDay, func(t time.Time) {
		result = append(result, t)
	})
	return result
}

// BetweenDatesEach calls each for every time between start and end.
// See BetweenDates for more information.
func BetweenDatesEach(start, end time.Time, y, m, d int, each func(time.Time)) {
	for t := start; !t.After(end); t = t.AddDate(y, m, d) {
		each(t)
	}
}

func Compare(t1, t2 time.Time) int {
	if t1.Equal(t2) {
		return 0
	}
	if t1.Before(t2) {
		return -1
	}
	return 1
}

func makeTimeSlice(totalDuration, stepDuration int) []time.Time {
	// if end is before start, this can be negative.
	if totalDuration < 0 {
		return make([]time.Time, 0)
	}
	size := totalDuration / stepDuration
	// 0 size will need 1 entry for the start, since start is inclusive.
	size++
	return make([]time.Time, 0, size)
}

// DaysInMonth returns the number of days in the month of t.
func DaysInMonth(t time.Time) int {
	startOfNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	lastOfThisMonth := startOfNextMonth.AddDate(0, 0, -1)
	return lastOfThisMonth.Day()
}

// RollMonth adds months number of months to t (months can be negative).
// Unlike Go's time.AddDate, this works on a calendar basis.
// For example, (October 31).AddDate(0, 1, 0) with Go's time package returns (December 1).
// RollMonth(October 31, 1) returns (November 30).
func RollMonth(t time.Time, months int) time.Time {
	offsetDate := t.AddDate(0, months, 0)
	targetMonth := offsetMonth(t.Month(), months)
	if offsetDate.Month() != targetMonth {
		// AddDate may return values that are 'too far ahead' when adding months,
		// or 'not far enough back' when subtracting months.
		// In both cases, the calculated date is too far in the future
		// (in other words: even though the wrongness is in the opposite direction-
		// too far forward, not enough back- in absolute terms the wrongness is always biased towards the future).
		// So we can subtract the number of days remaining in the month so we end up in the previous month.
		offsetDate = offsetDate.AddDate(0, 0, -offsetDate.Day())
	}
	return offsetDate
}

// Get the new month after offsetting month m by offset.
//
//     offsetMonth(January, 1) => February
//     offsetMonth(January, 13) => February
//     offsetMonth(January, -1) => December
func offsetMonth(m time.Month, offset int) time.Month {
	zeroBased := int(m - 1)
	offsetMonth := zeroBased + offset
	if offsetMonth < 0 {
		offsetMonth = 12 + offsetMonth
	}
	mod := offsetMonth % 12
	return time.Month(1 + mod)
}
