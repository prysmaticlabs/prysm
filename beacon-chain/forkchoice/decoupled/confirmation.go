package decoupled

import "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"

// Spec: advance_confirmed. The candidate replaces the record unless it is an ancestor of it.
func latestOf(record, candidate *Node) *Node {
	if isAncestor(candidate, record) {
		return record
	}
	return candidate
}

// Spec: floor_on_stable. The result never sits below the stable block.
func clampToStable(stable, candidate, record *Node) *Node {
	if isAncestor(stable, candidate) {
		return candidate
	}
	if isAncestor(stable, record) {
		return record
	}
	return stable
}

// Spec: update_confirmation. Called at the evaluation time of slot, which is the next slot's confirmation cutoff. Support must resolve before the confirmation cutoff, the denominator and the second-vote test use votes received before the evaluation time.
func (st *Store) UpdateConfirmation(slot primitives.Slot) {
	cutoff := st.confirmationCutoff(slot)
	evaluation := st.evaluationTime(slot)
	var late, supported []Vote
	for _, e := range st.pool[slot] {
		if e.at.Before(evaluation) {
			late = append(late, e.vote)
		}
	}
	for _, e := range st.pool[slot] {
		if at, ok := st.resolutionTime(e.vote); ok && at.Before(cutoff) && countVotes(late, e.vote.Validator) < 2 {
			supported = append(supported, e.vote)
		}
	}
	// Spec passes support_votes as both score arguments and counts participants over late_votes.
	counts := st.countSupport(view{participants: supported, supported: supported}, slot)
	seen := make(map[primitives.ValidatorIndex]struct{}, len(late))
	for _, u := range late {
		seen[u.Validator] = struct{}{}
	}
	counts.participants = len(seen)
	head := ghost(st.anchor(), st.candidates(everyNode), counts.score, counts.hasMajority)
	if counts.hasMajority(head) {
		st.liveConfirmed = head
		st.latestConfirmed = clampToStable(st.stableBlock(), latestOf(st.latestConfirmed, head), st.latestConfirmed)
		return
	}
	st.liveConfirmed = st.fgRoot()
	st.latestConfirmed = clampToStable(st.stableBlock(), st.latestConfirmed, st.latestConfirmed)
}

// Spec: live_confirmed, what the FG duty reads. Retreats to the FG root on an unconfirmed slot.
func (st *Store) LiveConfirmed() *Node { return st.liveConfirmed }

// Spec: latest_confirmed, the record users see.
func (st *Store) LatestConfirmed() *Node { return st.latestConfirmed }

// Spec: get_confirmed, the record if it extends the stable block, otherwise the stable block.
func (st *Store) Confirmed() *Node {
	if isAncestor(st.stableBlock(), st.latestConfirmed) {
		return st.latestConfirmed
	}
	return st.stableBlock()
}
