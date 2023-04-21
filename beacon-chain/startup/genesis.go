package startup

import (
	"time"

	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// Nower is a function that can return the current time.
// In Clock, Now() will use time.Now by default, but a Nower can be set using WithNower in NewClock
// to customize the return value for Now() in tests.
type Nower func() time.Time

// Clock abstracts important time-related concerns in the beacon chain:
//   - provides a time.Now() construct that can be overridden in tests
//   - GenesisTime() to know the genesis time or use genesis time determination as a syncronization point.
//   - CurrentSlot: convenience conversion for current time -> slot
//     (support backwards compatibility with the TimeFetcher interface)
//   - GenesisValidatorsRoot: is determined at the same point as genesis time and is needed by some of the same code,
//     so it is also bundled for convenience.
type Clock struct {
	t   time.Time
	vr  []byte
	now Nower
}

// GenesisTime returns the genesis timestamp.
func (g *Clock) GenesisTime() time.Time {
	return g.t
}

// GenesisValidatorsRoot returns the genesis state validator root
func (g *Clock) GenesisValidatorsRoot() []byte {
	return g.vr
}

// CurrentSlot returns the current slot relative to the time.Time value that Clock embeds.
func (g *Clock) CurrentSlot() types.Slot {
	return slots.Duration(g.t, g.now())
}

// Now provides a value for time.Now() that can be overridden in tests.
func (g *Clock) Now() time.Time {
	return g.now()
}

// ClockOpt is a functional option to change the behavior of a clock value made by NewClock.
// It is primarily intended as a way to inject an alternate time.Now() callback (WithNow) for testing.
type ClockOpt func(*Clock)

// WithNower allows tests in particular to inject an alternate implementation of time.Now (vs using system time)
func WithNower(n Nower) ClockOpt {
	return func(g *Clock) {
		g.now = n
	}
}

// NewClock constructs a genesis value, providing the ability to override the
func NewClock(t time.Time, vr []byte, opts ...ClockOpt) *Clock {
	g := &Clock{
		t:  t,
		vr: vr,
	}
	for _, o := range opts {
		o(g)
	}
	return g
}
