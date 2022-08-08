package blockchain

import (
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/time/slots"
	"time"
)

// Clock abstracts important time-related concerns in the beacon chain:
// - genesis time
// - provides a time.Now() construct that can be overriden in tests
// - syncronization point for code that needs to know the genesis time
// - CurrentSlot: convenience conversion for current time -> slot
//   - support backwards compatibility with the TimeFetcher interface
type Clock interface {
	GenesisTime() time.Time
	CurrentSlot() types.Slot
	Now() time.Time
}

// clock is a type that fulfills the TimeFetcher interface. This can be used in a number of places where
// blockchain.ChainInfoFetcher has historically been used.
type clock struct {
	time.Time
	now Now
}
var _ Clock = &clock{}

// clock provides an accessor to the embedded time, also fulfilling the blockchain.TimeFetcher interface.
func (gt clock) GenesisTime() time.Time {
	return gt.Time
}

// CurrentSlot returns the current slot relative to the time.Time value clock embeds.
func (gt clock) CurrentSlot() types.Slot {
	return slots.Duration(gt.Time, gt.now())
}

// Now provides a value for time.Now() that can be overriden in tests.
func (gt clock) Now() time.Time {
	return gt.now()
}

// ClockOpt is a functional option to change the behavior of a clock value made by NewClock.
// It is primarily intended as a way to inject an alternate time.Now() callback (WithNow) for testing.
type ClockOpt func(*clock)

// WithNow allows tests in particular to inject an alternate implementation of time.Now (vs using system time)
func WithNow(n Now) ClockOpt {
	return func(gt *clock) {
		gt.now = n
	}
}

// NewClock constructs a clock value using the given time value. Optional ClockOpt can be provided.
// If an implementation of the Now function type is not provided (via WithNow), time.Now (system time) will be used by default.
func NewClock(t time.Time, opts ...ClockOpt) clock {
	gt := clock{Time: t}
	for _, o :=  range opts {
		o(&gt)
	}
	if gt.now == nil {
		gt.now = time.Now
	}
	return gt
}

// Now is a function that can return the current time. This will be time.Now by default, but can be overridden for tests.
type Now func() time.Time