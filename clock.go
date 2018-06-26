package assumerole

import (
	"time"
)

// Clock is the clock interface we expect.
type Clock interface {
	Now() time.Time
}

type defaultClock struct{}

func (c *defaultClock) Now() time.Time {
	return time.Now()
}
