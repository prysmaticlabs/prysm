package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

func countVotes(votes []Vote, val primitives.ValidatorIndex) int {
	n := 0
	for _, v := range votes {
		if v.Validator == val {
			n++
		}
	}
	return n
}

// Spec: equivocates, two distinct votes by one validator in the set.
func equivocates(votes []Vote, val primitives.ValidatorIndex) bool {
	return countVotes(votes, val) >= 2
}

func participates(votes []Vote, val primitives.ValidatorIndex) bool {
	return countVotes(votes, val) >= 1
}

func (st *Store) poolVotes(k primitives.Slot) []Vote {
	entries := st.pool[k]
	out := make([]Vote, len(entries))
	for i, e := range entries {
		out[i] = e.vote
	}
	return out
}

func (st *Store) receivedAt(v Vote) (time.Time, bool) {
	for _, e := range st.pool[v.Slot] {
		if e.vote == v {
			return e.at, true
		}
	}
	return time.Time{}, false
}

// Votes are stamped on receipt, so the gossip validator must call this directly instead of parking votes in the attestation pool.
func (st *Store) ProcessVote(validators []primitives.ValidatorIndex, root [32]byte, slot primitives.Slot, payloadStatus bool) {
	now := st.now()
	for _, val := range validators {
		st.addVote(Vote{Validator: val, Slot: slot, Root: root, PayloadStatus: payloadStatus}, now)
	}
}

// Spec: on_goldfish_vote. The slot window is [s-1, s], duplicates and a third distinct vote are dropped unstamped.
func (st *Store) addVote(v Vote, now time.Time) {
	current := st.currentSlot()
	if v.Slot+1 < current || current < v.Slot {
		return
	}
	if _, dup := st.receivedAt(v); dup {
		return
	}
	if equivocates(st.poolVotes(v.Slot), v.Validator) {
		return
	}
	st.pool[v.Slot] = append(st.pool[v.Slot], poolEntry{vote: v, at: now})
}

// Spec: resolved. A root from a later slot than the vote never resolves.
func (st *Store) resolve(v Vote) *Node {
	n := st.nodes[v.Root]
	if n == nil || n.Slot > v.Slot {
		return nil
	}
	return n
}

// Spec: resolution_time τ = max(receipt(vote), receipt(block)), unresolved otherwise. Genesis has a zero receipt.
func (st *Store) resolutionTime(v Vote) (time.Time, bool) {
	at, ok := st.receivedAt(v)
	if !ok {
		return time.Time{}, false
	}
	n := st.resolve(v)
	if n == nil {
		return time.Time{}, false
	}
	if n.processed.After(at) {
		return n.processed, true
	}
	return at, true
}
