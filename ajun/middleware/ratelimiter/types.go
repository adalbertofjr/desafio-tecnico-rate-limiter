package ratelimiter

import "time"

type ClientIPData struct {
	count        int
	time         time.Time
	disableUntil time.Time
}
