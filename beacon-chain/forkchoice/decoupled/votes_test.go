package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestProcessVoteWindowDuplicateAndCap(t *testing.T) {
	st := newStore(t)
	st.at(clock(2, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(0), 0, false)
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(0), 3, false)
	require.Equal(t, 0, len(st.poolVotes(0)), "older than s-1")
	require.Equal(t, 0, len(st.poolVotes(3)), "future slot")
	st.ProcessVote([]primitives.ValidatorIndex{1, 2}, root(0), 2, false)
	require.Equal(t, 2, len(st.poolVotes(2)))
	v := Vote{Validator: 1, Slot: 2, Root: root(0)}
	st.at(clock(2, 2))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(0), 2, false)
	at, ok := st.receivedAt(v)
	require.Equal(t, true, ok)
	require.Equal(t, clock(2, 1), at, "duplicate must not restamp")
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(7), 2, false)
	require.Equal(t, true, equivocates(st.poolVotes(2), 1), "second distinct vote establishes equivocation")
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(8), 2, false)
	_, ok = st.receivedAt(Vote{Validator: 1, Slot: 2, Root: root(8)})
	require.Equal(t, false, ok, "third distinct vote is dropped unstamped")
	require.Equal(t, 3, len(st.poolVotes(2)))
	require.Equal(t, true, participates(st.poolVotes(2), 2))
	require.Equal(t, false, participates(st.poolVotes(2), 3))
}

func TestPayloadStatusIsPartOfVoteIdentity(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(0), 1, false)
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(0), 1, true)
	require.Equal(t, true, equivocates(st.poolVotes(1), 1))
}

func TestResolveRejectsLaterSlotRoot(t *testing.T) {
	st := newStore(t)
	a := st.addChild(root(1), st.Node(root(0)), 3)
	require.IsNil(t, st.resolve(Vote{Validator: 1, Slot: 2, Root: root(1)}))
	require.Equal(t, a, st.resolve(Vote{Validator: 1, Slot: 3, Root: root(1)}))
	require.IsNil(t, st.resolve(Vote{Validator: 1, Slot: 3, Root: root(9)}))
}

func TestResolutionTimeIsMaxOfVoteAndBlockReceipt(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 1))
	v := Vote{Validator: 1, Slot: 1, Root: root(1)}
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(1), 1, false)
	_, ok := st.resolutionTime(v)
	require.Equal(t, false, ok, "unresolved while the block is missing")
	a := st.addChild(root(1), st.Node(root(0)), 1)
	a.processed = clock(1, 2)
	at, ok := st.resolutionTime(v)
	require.Equal(t, true, ok)
	require.Equal(t, clock(1, 2), at)
	g := Vote{Validator: 2, Slot: 1, Root: root(0)}
	st.ProcessVote([]primitives.ValidatorIndex{2}, root(0), 1, false)
	at, _ = st.resolutionTime(g)
	require.Equal(t, clock(1, 1), at, "genesis has no receipt time and drops out of the max")
}
