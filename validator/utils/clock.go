package utils

import "time"

// BlockingWait sleeps until a specific time is reached after
// a certain duration. For example, if the genesis block
// was at 12:00:00PM and the current time is 12:00:03PM,
// we want the next slot to tick at 12:00:08PM so we can use
// this helper method to achieve that purpose.
func BlockingWait(duration time.Duration) {
	d := time.Until(time.Now().Add(duration).Truncate(duration))
	time.Sleep(d)
}
