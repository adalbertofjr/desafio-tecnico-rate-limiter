package ratelimiter

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("client IP not found")

type ClientIPData struct {
	Count        int
	Time         time.Time
	DisableUntil time.Time
}
