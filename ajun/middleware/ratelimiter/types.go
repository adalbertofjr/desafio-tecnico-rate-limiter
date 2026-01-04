package ratelimiter

import "time"

type ClientIPData struct {
	Count        int
	Time         time.Time
	DisableUntil time.Time
}
