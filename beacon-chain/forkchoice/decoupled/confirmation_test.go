package decoupled

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

// Slot 1: confirmation cutoff at (1,2), evaluation at (2,2).
func TestConfirmationNeedsSupportBeforeCutoffOverLateParticipants(t *testing.T) {
	st := twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1, 2, 3}, root(1), 1, false)
	st.ProcessVote([]primitives.ValidatorIndex{4}, root(2), 1, false)
	st.at(clock(1, 3))
	st.ProcessVote([]primitives.ValidatorIndex{5}, root(2), 1, false)
	st.at(clock(2, 1))
	st.ProcessVote([]primitives.ValidatorIndex{6}, root(2), 1, false)
	st.at(clock(2, 2))
	st.UpdateConfirmation(1)
	require.Equal(t, st.Node(root(0)), st.LiveConfirmed(), "3 for A over 6 participants, 6 < 6 fails, so the FG root")
	require.Equal(t, st.Node(root(0)), st.LatestConfirmed())

	st = twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1, 2, 3, 4}, root(1), 1, false)
	st.at(clock(1, 3))
	st.ProcessVote([]primitives.ValidatorIndex{5}, root(2), 1, false)
	st.at(clock(2, 2))
	st.UpdateConfirmation(1)
	require.Equal(t, st.Node(root(1)), st.LiveConfirmed(), "4 for A over 5 participants")
	require.Equal(t, st.Node(root(1)), st.LatestConfirmed())
	require.Equal(t, st.Node(root(1)), st.Confirmed())
}

func TestConfirmationVoteBeforeCutoffBlockAtCutoff(t *testing.T) {
	st := newStore(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1}, root(1), 1, false)
	st.at(clock(1, 2))
	require.NoError(t, st.InsertNode(Block{Root: root(1), Parent: root(0), Slot: 1, FG: base}))
	st.at(clock(2, 2))
	st.UpdateConfirmation(1)
	require.Equal(t, st.Node(root(0)), st.LiveConfirmed(), "resolved exactly at the cutoff, not before it")
	st.at(clock(2, 1))
	require.Equal(t, st.Node(root(1)), st.VoteHead(2), "resolved before the support cutoff, so the next vote sees it")
}

func TestConfirmationDropsSupportFromALateSecondVote(t *testing.T) {
	st := twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1, 2}, root(1), 1, false)
	st.at(clock(2, 0))
	st.ProcessVote([]primitives.ValidatorIndex{2}, root(2), 1, false)
	st.at(clock(2, 2))
	st.UpdateConfirmation(1)
	require.Equal(t, st.Node(root(0)), st.LiveConfirmed(), "validator 2 is dropped, A has 1 of 2 participants and 2 < 2 fails")
}

func TestLatestConfirmedSurvivesAnUnconfirmedSlot(t *testing.T) {
	st := twoChildren(t)
	st.at(clock(1, 1))
	st.ProcessVote([]primitives.ValidatorIndex{1, 2, 3}, root(1), 1, false)
	st.at(clock(2, 2))
	st.UpdateConfirmation(1)
	require.Equal(t, st.Node(root(1)), st.LiveConfirmed())
	st.at(clock(3, 2))
	st.UpdateConfirmation(2)
	require.Equal(t, st.Node(root(0)), st.LiveConfirmed(), "an empty slot confirms nothing, live retreats to the FG root")
	require.Equal(t, st.Node(root(1)), st.LatestConfirmed(), "the record keeps the last confirmation")
	require.Equal(t, st.Node(root(1)), st.Confirmed())
}

func TestLatestOfAndClampToStable(t *testing.T) {
	st := newStore(t)
	g := st.Node(root(0))
	a := st.addChild(root(1), g, 1)
	b := st.addChild(root(2), a, 2)
	c := st.addChild(root(3), g, 1)
	require.Equal(t, b, latestOf(b, a), "an ancestor candidate keeps the record")
	require.Equal(t, b, latestOf(a, b))
	require.Equal(t, c, latestOf(b, c), "a conflict replaces")
	require.Equal(t, b, clampToStable(a, b, c), "candidate extends stable")
	require.Equal(t, b, clampToStable(a, c, b), "candidate off the stable branch, keep the record")
	require.Equal(t, a, clampToStable(a, c, c), "both off, move to stable")
}
