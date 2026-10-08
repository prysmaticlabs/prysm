package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func twoChildren(t *testing.T) *Store {
	st := newStore(t)
	st.at(clock(1, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}))
	require.NoError(t, st.InsertNode(Block{Root: root(2), Parent: root(0), Slot: 1, FG: base}))
	return st
}

// Three of five slot-1 voters back A, so the slot-2 proposer and voter both pick A.
func TestProposalHeadAndVoteHeadFollowPreviousSlotMajority(t *testing.T) {
	st := twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1, 2, 3}, root(1), 1, false)
	st.ProcessVote([]primitives.ValidatorIndex{4, 5}, root(2), 1, false)
	st.at(clock(2, 0))
	require.Equal(t, st.Node(root(1)), st.ProposalHead(2))
	require.Equal(t, 5, len(st.BlockVotes(2)))
	st.at(clock(2, 1))
	require.Equal(t, st.Node(root(1)), st.VoteHead(2))
}

// Votes received after the support cutoff are invisible to the voter until a slot-2 block carries them.
func TestVoteHeadUsesBlockVotesAfterSupportCutoff(t *testing.T) {
	st := twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(2), 1, false)
	st.at(clock(1, 3))
	st.ProcessVote([]primitives.ValidatorIndex{2, 3}, root(1), 1, false)
	st.at(clock(2, 1))
	require.Equal(t, st.Node(root(2)), st.VoteHead(2), "only validator 1 is inside the cutoff")
	late := []Vote{vote(2, 1, root(1)), vote(3, 1, root(1))}
	require.NoError(t, st.InsertNode(Block{Root: root(3), Parent: root(1), Slot: 2, BlockVotes: late, FG: base}))
	require.Equal(t, st.Node(root(3)), st.VoteHead(2), "block votes flip to A, then the current-slot clause descends to the proposal")
}

func TestHeadFollowsSlotPhase(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}))
	require.Equal(t, st.Node(root(1)), st.Head(), "a current-slot block is eligible without votes")
	require.NoError(t, st.InsertNode(Block{Root: root(2), Parent: root(0), Slot: 1, FG: base}))
	require.Equal(t, st.Node(root(2)), st.Head(), "tie by greatest root")
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(1), 1, false)
	require.Equal(t, st.Node(root(1)), st.Head(), "from the vote time the slot-1 pool decides")
	st.at(clock(2, 0))
	require.Equal(t, st.Node(root(1)), st.Head(), "before slot 2's vote time the proposer view of slot 2 reads the slot-1 pool")
	st.at(clock(2, 1))
	require.Equal(t, st.Node(root(0)), st.Head(), "the slot-2 pool is empty and no slot-1 block has a majority, so the anchor")
}

func TestVoteHeadIgnoresBlocksOutsideTheVoterView(t *testing.T) {
	st := twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1, 2, 3}, root(1), 1, false)
	st.at(clock(1, 3))
	require.NoError(t, st.InsertNode(Block{Root: root(3), Parent: root(1), Slot: 1, FG: base}))
	st.at(clock(2, 1))
	require.Equal(t, st.Node(root(1)), st.VoteHead(2), "root3 was processed at the cutoff, not before it, and is not a slot-2 proposal")
	require.Equal(t, st.Node(root(1)), st.ProposalHead(2), "root3 has no votes and is not a current-slot block")
}
