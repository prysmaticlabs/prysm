package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

func (st *Store) votes(slot primitives.Slot) []Vote {
	if b := st.pool[slot]; b != nil {
		return b.votes
	}
	return nil
}

// votesBefore counts distinct votes per validator strictly before instant.
func (b *bucket) votesBefore(instant time.Time) map[primitives.ValidatorIndex]int {
	seen := make(map[primitives.ValidatorIndex]int, len(b.byValidator))
	for i, u := range b.votes {
		if b.at[i].Before(instant) {
			seen[u.Validator]++
		}
	}
	return seen
}

// receivedAt returns the vote's first local processing time.
func (st *Store) receivedAt(v Vote) (time.Time, bool) {
	b := st.pool[v.Slot]
	if b == nil {
		return time.Time{}, false
	}
	for _, i := range b.byValidator[v.Validator] {
		if b.votes[i] == v {
			return b.at[i], true
		}
	}
	return time.Time{}, false
}

// ProcessVote records verified votes at now, after running due ticks.
// A vote processed exactly at a cutoff is excluded from that cutoff's view.
func (st *Store) ProcessVote(validators []primitives.ValidatorIndex, root [32]byte, slot primitives.Slot, now time.Time) {
	st.OnTick(now)
	for _, val := range validators {
		st.addVote(Vote{Validator: val, Slot: slot, Root: root}, now)
	}
}

// addVote retains current- and previous-slot votes without refreshing duplicates.
// Two distinct votes per validator suffice to detect equivocation.
func (st *Store) addVote(v Vote, now time.Time) {
	current := st.slotAt(now)
	if v.Slot+1 < current || current < v.Slot {
		return
	}
	b := st.pool[v.Slot]
	if b == nil {
		b = &bucket{byValidator: make(map[primitives.ValidatorIndex][]int)}
		st.pool[v.Slot] = b
	}
	pos := b.byValidator[v.Validator]
	for _, i := range pos {
		if b.votes[i] == v {
			return
		}
	}
	if len(pos) >= 2 {
		return
	}
	b.byValidator[v.Validator] = append(pos, len(b.votes))
	b.votes = append(b.votes, v)
	b.at = append(b.at, now)
}

// block excludes unknown blocks and blocks newer than the vote slot.
func (st *Store) block(v Vote) *Node {
	n := st.nodes[v.Root]
	if n == nil || n.Slot > v.Slot {
		return nil
	}
	return n
}

// processedAt is when both the vote and its block were available locally.
func (st *Store) processedAt(v Vote) (time.Time, bool) {
	at, ok := st.receivedAt(v)
	if !ok {
		return time.Time{}, false
	}
	n := st.block(v)
	if n == nil {
		return time.Time{}, false
	}
	if n.processed.After(at) {
		return n.processed, true
	}
	return at, true
}
