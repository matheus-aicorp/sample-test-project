package domain

import "time"

// Clock abstracts time so use cases can be tested against fixed instants.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
