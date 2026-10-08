package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// Spec: Δ, the slot is 4Δ.
func delta() time.Duration { return params.BeaconConfig().SlotDuration() / 4 }

func (st *Store) slotStart(s primitives.Slot) time.Time {
	return slots.UnsafeStartTime(st.genesis, s)
}

func (st *Store) currentSlot() primitives.Slot { return slots.At(st.genesis, st.now()) }

func (st *Store) voteTime(s primitives.Slot) time.Time { return st.slotStart(s).Add(delta()) }
func (st *Store) confirmationCutoff(s primitives.Slot) time.Time {
	return st.slotStart(s).Add(2 * delta())
}
func (st *Store) supportCutoff(s primitives.Slot) time.Time { return st.slotStart(s).Add(3 * delta()) }
func (st *Store) evaluationTime(s primitives.Slot) time.Time {
	return st.slotStart(s).Add(6 * delta())
}

// Slot s-1 is still read by the slot-s voter view and by the slot-(s-1) confirmation at 2Δ, older slots are not.
func (st *Store) NewSlot(slot primitives.Slot) {
	for k := range st.pool {
		if k+1 < slot {
			delete(st.pool, k)
		}
	}
}
