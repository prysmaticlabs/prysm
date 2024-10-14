package audit

import "time"

// Failure struct to track the reason and the time of each failure
type Failure struct {
	Reason string
	Time   time.Time
}
