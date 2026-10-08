package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// confirmationCounts counts participation before 6*delta, but support only
// before 2*delta. Equivocators count toward participation, never support.
func (st *Store) confirmationCounts(slot primitives.Slot) *supportCounts {
	c := &supportCounts{subtree: make(map[*Node]int)}
	b := st.pool[slot]
	if b == nil {
		return c
	}
	votesBy := b.votesBefore(st.evaluationTime(slot))
	c.participants = len(votesBy)
	cutoff := st.confirmationCutoff(slot)
	for _, u := range b.votes {
		if votesBy[u.Validator] != 1 {
			continue
		}
		if at, ok := st.processedAt(u); !ok || !at.Before(cutoff) {
			continue
		}
		for n := st.block(u); n != nil; n = n.Parent {
			c.subtree[n]++
		}
	}
	return c
}

// advance preserves a deeper confirmation when candidate is its ancestor.
func (st *Store) advance(candidate *Node) {
	if st.inTree(st.latestConfirmed) && isAncestor(candidate, st.latestConfirmed) {
		return
	}
	st.latestConfirmed = candidate
}

// updateConfirmation records the majority result, applies the lock veto, then
// updates confirmation, locks and FG source candidates.
func (st *Store) updateConfirmation(slot primitives.Slot, now time.Time) {
	st.fastConfirmed = st.fastConfirmed[:0]
	if st.restartTime.After(st.slotStart(slot)) {
		st.confirmations[slot] = nil
		return
	}
	c := st.confirmationCounts(slot)
	// Confirmation ignores the head-selection root and height floor.
	candidate := ghost(st.finalized, everyNode, c.score, c.hasMajority)
	if !c.hasMajority(candidate) {
		candidate = nil
	}
	// Vetoed results still count toward slow-lock streaks.
	st.confirmations[slot] = candidate
	if candidate == nil || st.vetoed(candidate, now) {
		return
	}
	st.advance(candidate)
	streak := st.streakAncestor(candidate, slot)
	slow := false
	var locked *Node
	for n := candidate; n != nil; n = n.Parent {
		// Fast locking requires at least half the full committee.
		fast := 2*uint64(c.score(n)) >= st.committeeSize
		if fast {
			st.fastConfirmed = append(st.fastConfirmed, n)
		}
		// Every ancestor of the streak's common ancestor also passes.
		slow = slow || n == streak
		if locked == nil && (fast || slow) {
			locked = n
		}
	}
	if locked != nil {
		st.locks = append(st.locks, lock{block: locked, deadline: st.unlockTime(slot + lockStreak())})
	}
}

// streakAncestor finds the deepest block covered by all k consecutive results.
// A missing or pruned result breaks the streak.
func (st *Store) streakAncestor(candidate *Node, slot primitives.Slot) *Node {
	streak := candidate
	for i := primitives.Slot(1); i < lockStreak(); i++ {
		if i > slot {
			return nil
		}
		c := st.confirmations[slot-i]
		if !st.inTree(c) {
			return nil
		}
		streak = commonAncestor(streak, c)
	}
	return streak
}

// Confirmed ticks to now, then returns the common ancestor of the confirmation record and the head.
// It can back off without changing the record. A pruned record falls back to F.
func (st *Store) Confirmed(now time.Time) *Node {
	st.OnTick(now)
	return st.confirmed(now)
}

func (st *Store) confirmed(now time.Time) *Node {
	c := st.latestConfirmed
	if !st.inTree(c) {
		c = st.finalized
	}
	return commonAncestor(c, st.head(now))
}
