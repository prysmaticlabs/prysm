package decoupled

import (
	"slices"
	"time"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// delta is the AC timing interval: one quarter of a slot.
func delta() time.Duration { return params.BeaconConfig().SlotDuration() / 4 }

func lockStreak() primitives.Slot { return primitives.Slot(params.BeaconConfig().DecoupledLockStreak) }

func (st *Store) slotStart(s primitives.Slot) time.Time {
	return st.genesis.Add(4 * delta() * time.Duration(s))
}

func (st *Store) slotAt(now time.Time) primitives.Slot {
	if now.Before(st.genesis) {
		return 0
	}
	return primitives.Slot(now.Sub(st.genesis) / (4 * delta()))
}

func (st *Store) voteTime(s primitives.Slot) time.Time { return st.slotStart(s).Add(delta()) }

// confirmationCutoff applies to both the vote and its block.
func (st *Store) confirmationCutoff(s primitives.Slot) time.Time {
	return st.slotStart(s).Add(2 * delta())
}

// supportCutoff freezes the local view; proposal view merge can add later evidence.
func (st *Store) supportCutoff(s primitives.Slot) time.Time {
	return st.slotStart(s).Add(3 * delta())
}

// unlockTime precedes the next slot's vote at the same timestamp.
func (st *Store) unlockTime(s primitives.Slot) time.Time {
	return st.slotStart(s).Add(5 * delta())
}

// evaluationTime closes participation and equivocation counting for slot s.
func (st *Store) evaluationTime(s primitives.Slot) time.Time {
	return st.slotStart(s).Add(6 * delta())
}

// joined requires observation since the previous slot started.
// A restart midway through slot 10 permits duties from slot 12.
func (st *Store) joined(slot primitives.Slot) bool {
	return slot > 0 && !st.restartTime.After(st.slotStart(slot-1))
}

// OnTick runs every phase scheduled since the last call, each at its own instant.
// New slot: drop old pools and confirmations. Δ: back locks up, capture the vote head.
// 2Δ: evaluate the previous slot, on an action slot capture the FG source and head.
// Call before processing messages at now. Late calls catch up, each phase runs once.
func (st *Store) OnTick(now time.Time) {
	from := st.slotAt(st.ticked)
	for s := from; s <= st.slotAt(now); s++ {
		if s > from {
			for k := range st.pool {
				if k+1 < s {
					delete(st.pool, k)
				}
			}
			for k := range st.confirmations {
				if k+lockStreak() < s {
					delete(st.confirmations, k)
				}
			}
		}
		if s == 0 {
			continue
		}
		if at := st.voteTime(s); st.due(at, now) {
			if st.joined(s) {
				st.unlock(s-1, at)
			}
			st.duty.voteSlot, st.duty.vote = s, st.voteHead(s, at)
		}
		if at := st.confirmationCutoff(s); st.due(at, now) {
			st.updateConfirmation(s-1, at)
			if params.BeaconConfig().DecoupledActionSlot(s) {
				st.duty.fgSlot = s
				st.duty.source, st.duty.head = st.source(at)
			}
		}
	}
	st.locks = slices.DeleteFunc(st.locks, func(l lock) bool { return !st.live(l, now) })
	if now.After(st.ticked) {
		st.ticked = now
	}
}

// True once per instant, on the first tick at or after it.
func (st *Store) due(at, now time.Time) bool { return at.After(st.ticked) && !at.After(now) }

// Spec: ac_restart. Every retained block is restamped at now and its carried votes replayed in parent order into fresh pools, saved confirmations and the fast set are cleared, locks keep their deadlines and the confirmation record stays. Observation counts from now.
func (st *Store) Restart(now time.Time) {
	st.restartTime = now
	st.pool = make(map[primitives.Slot]*bucket)
	st.confirmations = make(map[primitives.Slot]*Node)
	st.fastConfirmed = st.fastConfirmed[:0]
	st.duty = duty{}
	blocks := append([]*Node(nil), st.finalized.Children...)
	for i := 0; i < len(blocks); i++ {
		blocks[i].processed = now
		for _, v := range blocks[i].Votes {
			st.addVote(v, now)
		}
		blocks = append(blocks, blocks[i].Children...)
	}
}
