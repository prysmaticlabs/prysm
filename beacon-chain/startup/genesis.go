package startup

import (
	"time"

	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// Clock abstracts important time-related concerns in the beacon chain:
// - provides a time.Now() construct that can be overridden in tests
// - synchronization point for code that needs to know the genesis time
// - CurrentSlot: convenience conversion for current time -> slot
//   - support backwards compatibility with the TimeFetcher interface
type Clock interface {
	CurrentSlot() types.Slot
	Now() time.Time
}

// Nower is a function that can return the current time. This will be time.Now by default, but can be overridden for tests.
type Nower func() time.Time

// Genesis represents the genesis time and validator root.
// Genesis also provides a relative concept of chain time via the Clock interface.
type Genesis struct {
	t   time.Time
	vr  []byte
	now Nower
}

func (g *Genesis) Time() time.Time {
	return g.t
}

func (g *Genesis) ValidatorRoot() []byte {
	return g.vr
}

// Clock returns an instance of the Clock interface, using the genesis time used to construct the Genesis value.
func (g *Genesis) Clock() Clock {
	now := g.now
	if now == nil {
		now = time.Now
	}
	return &clock{
		genesis: g.t,
		now:     now,
	}
}

// clock is a type that fulfills the TimeFetcher interface. This can be used in a number of places where
// blockchain.ChainInfoFetcher has historically been used.
type clock struct {
	genesis time.Time
	now     Nower
}

// CurrentSlot returns the current slot relative to the time.Time value clock embeds.
func (gt clock) CurrentSlot() types.Slot {
	return slots.Duration(gt.genesis, gt.now())
}

// Now provides a value for time.Now() that can be overridden in tests.
func (gt clock) Now() time.Time {
	return gt.now()
}

// ClockOpt is a functional option to change the behavior of a clock value made by NewClock.
// It is primarily intended as a way to inject an alternate time.Now() callback (WithNow) for testing.
type GenesisOpt func(*Genesis)

// WithNower allows tests in particular to inject an alternate implementation of time.Now (vs using system time)
func WithNower(n Nower) GenesisOpt {
	return func(g *Genesis) {
		g.now = n
	}
}

// NewGenesis constructs a genesis value, providing the ability to override the
func NewGenesis(t time.Time, vr []byte, opts ...GenesisOpt) *Genesis {
	g := &Genesis{
		t:  t,
		vr: vr,
	}
	for _, o := range opts {
		o(g)
	}
	return g
}
