package common

import "time"

// Rate represents a rate.
type Rate struct {
	Value    int
	Interval time.Duration
}
