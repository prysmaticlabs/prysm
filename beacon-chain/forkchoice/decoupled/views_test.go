package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func hasVote(votes []Vote, v Vote) bool {
	for _, u := range votes {
		if u == v {
			return true
		}
	}
	return false
}

func vote(val primitives.ValidatorIndex, slot primitives.Slot, r [32]byte) Vote {
	return Vote{Validator: val, Slot: slot, Root: r}
}

func TestVoterViewSupportCutoffAndBlockVotes(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}))
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(1), 1, false)
	st.ProcessVote([]primitives.ValidatorIndex{2}, root(5), 1, false)
	st.at(clock(1, 3))
	st.ProcessVote([]primitives.ValidatorIndex{3}, root(1), 1, false)
	early, unknownRoot, afterCutoff := vote(1, 1, root(1)), vote(2, 1, root(5)), vote(3, 1, root(1))
	blockVote, blockVoteUnresolved := vote(4, 1, root(1)), vote(5, 1, root(9))
	st.at(clock(2, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(2), Parent: root(1), Slot: 2, BlockVotes: []Vote{blockVote, afterCutoff, blockVoteUnresolved}, FG: base}))
	v := st.voterView(2)
	require.Equal(t, 5, len(v.participants))
	require.Equal(t, true, hasVote(v.participants, early))
	require.Equal(t, true, hasVote(v.participants, unknownRoot), "an unknown root still participates")
	require.Equal(t, true, hasVote(v.participants, afterCutoff), "a block vote enters even though it was received after the support cutoff")
	require.Equal(t, true, hasVote(v.supported, early))
	require.Equal(t, false, hasVote(v.supported, unknownRoot))
	require.Equal(t, true, hasVote(v.supported, blockVote), "block votes skip the local support cutoff")
	require.Equal(t, true, hasVote(v.supported, afterCutoff))
	require.Equal(t, false, hasVote(v.supported, blockVoteUnresolved), "a block vote whose root is unknown only participates")
}

func TestVoterViewUnionsAllSlotProposals(t *testing.T) {
	st := newStore(t)
	st.at(clock(2, 0))
	a, b, c := vote(1, 1, root(0)), vote(1, 1, root(7)), vote(1, 1, root(8))
	require.NoError(t, st.InsertNode(Block{Root: root(2), Parent: root(0), Slot: 2, BlockVotes: []Vote{a, b}, FG: base}))
	require.NoError(t, st.InsertNode(Block{Root: root(3), Parent: root(0), Slot: 2, BlockVotes: []Vote{c}, FG: base}))
	v := st.voterView(2)
	require.Equal(t, 3, countVotes(v.participants, 1), "the merged view is not capped at two")
	require.Equal(t, 2, len(st.poolVotes(1)), "the pool stays capped")
}

func TestVoterBlocksAndProposerView(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}))
	st.at(clock(1, 3))
	st.ProcessVote([]primitives.ValidatorIndex{9}, root(1), 1, false)
	st.at(clock(2, 0))
	require.NoError(t, st.InsertNode(Block{Root: root(2), Parent: root(1), Slot: 2, FG: base}))
	require.NoError(t, st.InsertNode(Block{Root: root(3), Parent: root(0), Slot: 1, FG: base}))
	in := st.voterBlocks(2)
	require.Equal(t, true, in[st.Node(root(0))])
	require.Equal(t, true, in[st.Node(root(1))])
	require.Equal(t, true, in[st.Node(root(2))], "a slot-2 proposal and its ancestors enter the view")
	require.Equal(t, false, in[st.Node(root(3))], "processed after the slot-1 support cutoff")
	p := st.proposerView(2)
	late := vote(9, 1, root(1))
	require.Equal(t, true, hasVote(p.participants, late), "the proposer applies no cutoff")
	require.Equal(t, true, hasVote(p.supported, late))
}
