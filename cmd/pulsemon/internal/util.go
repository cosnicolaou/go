package internal

import "time"

// UntilHHMM returns the duration until the specified time (in 24 hours and
// minutes) will be reached.
func UntilHHMM(hhmm time.Time) time.Duration {
	now := time.Now()
	then := time.Date(now.Year(), now.Month(), now.Day(), hhmm.Hour(), hhmm.Minute(), 0, 0, time.Local)
	until := then.Sub(now)
	if until < 0 {
		until += 24 * time.Hour
	}
	return until
}
