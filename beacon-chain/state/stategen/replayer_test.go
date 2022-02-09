package stategen

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"

	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/testing/require"
)

// combines a mock history sanity check w/ tests on canonicalBlockForSlot
// to show how canonicalBlockForSlot relates to HighestSlotBlocksBelow
func TestMockHistoryCanonicalBlockForSlot(t *testing.T) {
	ctx := context.Background()
	var begin, middle, end types.Slot = 100, 150, 155
	specs := []mockHistorySpec{
		{slot: begin},
		{slot: middle, savedState: true},
		{slot: end, canonicalBlock: true},
	}
	hist := newMockHistory(t, specs, end+1)
	cc := canonicalChainer{hist, hist, hist}

	// since only the end block and genesis are canonical, once the slot drops below
	// end, we should always get genesis
	cases := []struct {
		slot    types.Slot
		highest types.Slot
		canon   types.Slot
		name    string
	}{
		{slot: hist.current, highest: end, canon: end, name: "slot > end"},
		{slot: end, highest: end, canon: end, name: "slot == end"},
		{slot: end - 1, highest: middle, canon: 0, name: "middle < slot < end"},
		{slot: middle, highest: middle, canon: 0, name: "slot == middle"},
		{slot: middle - 1, highest: begin, canon: 0, name: "begin < slot < middle"},
		{slot: begin, highest: begin, canon: 0, name: "slot == begin"},
		{slot: begin - 1, highest: 0, canon: 0, name: "genesis < slot < begin"},
		{slot: 0, highest: 0, canon: 0, name: "slot == genesis"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bs, err := hist.HighestSlotBlocksBelow(ctx, c.slot+1)
			require.NoError(t, err)
			require.Equal(t, len(bs), 1)
			r, err := bs[0].Block().HashTreeRoot()
			require.NoError(t, err)
			require.Equal(t, hist.slotMap[c.highest], r)
			cr, _, err := cc.canonicalBlockForSlot(ctx, c.slot)
			require.NoError(t, err)
			require.Equal(t, hist.slotMap[c.canon], cr)
		})
	}
}

type mockCanonicalChainer struct {
	State  state.BeaconState
	Blocks []block.SignedBeaconBlock
	Err    error
}

func (m mockCanonicalChainer) chainForSlot(_ context.Context, _ types.Slot) (state.BeaconState, []block.SignedBeaconBlock, error) {
	return m.State, m.Blocks, m.Err
}

var _ chainer = &mockCanonicalChainer{}

type mockCurrentSlotter struct {
	Slot types.Slot
}

func (c *mockCurrentSlotter) CurrentSlot() types.Slot {
	return c.Slot
}

var _ CurrentSlotter = &mockCurrentSlotter{}

func TestCanonicalChainerFuture(t *testing.T) {
	r := &canonicalChainer{
		cs: &mockCurrentSlotter{Slot: 0},
	}
	_, _, err := r.chainForSlot(context.Background(), 1)
	require.ErrorIs(t, err, ErrFutureSlotRequested)
}

// TODO
func TestCanonicalChainerOK(t *testing.T) {
}

func TestAncestorChainOK(t *testing.T) {
	ctx := context.Background()
	var begin, middle, end types.Slot = 100, 150, 155
	specs := []mockHistorySpec{
		{slot: begin},
		{slot: middle, savedState: true},
		{slot: end, canonicalBlock: true},
	}
	hist := newMockHistory(t, specs, end+1)
	cc := &canonicalChainer{h: hist, c: hist, cs: hist}

	endBlock := hist.blocks[hist.slotMap[end]]
	st, bs, err := cc.ancestorChain(ctx, endBlock)
	require.NoError(t, err)

	// middle is the most recent slot where savedState == true
	expectedSt := hist.states[hist.slotMap[middle]]
	require.Equal(t, 1, len(bs))
	require.DeepEqual(t, endBlock, bs[0])
	require.Equal(t, expectedSt, st)

	middleBlock := hist.blocks[hist.slotMap[middle]]
	st, bs, err = cc.ancestorChain(ctx, middleBlock)
	require.NoError(t, err)
	require.Equal(t, 0, len(bs))
	require.Equal(t, expectedSt, st)
}

// TODO
func TestChainForSlotOK(t *testing.T) {
}

func TestAncestorChainOrdering(t *testing.T) {
	ctx := context.Background()
	var zero, one, two, three, four, five types.Slot = 50, 51, 150, 151, 152, 200
	specs := []mockHistorySpec{
		{slot: zero},
		{slot: one, savedState: true},
		{slot: two},
		{slot: three},
		{slot: four},
		{slot: five},
	}

	hist := newMockHistory(t, specs, five+1)
	endRoot := hist.slotMap[specs[len(specs)-1].slot]
	endBlock := hist.blocks[endRoot]

	cc := &canonicalChainer{h: hist, c: hist, cs: hist}
	st, bs, err := cc.ancestorChain(ctx, endBlock)
	require.NoError(t, err)
	require.DeepEqual(t, hist.states[hist.slotMap[one]], st)
	// we asked for the chain leading up to five
	// one has the savedState. one is applied to the savedState, so it should be omitted
	// that means we should get two, three, four, five (length of 4)
	require.Equal(t, 4, len(bs))
	for i, slot := range []types.Slot{two, three, four, five} {
		require.Equal(t, slot, bs[i].Block().Slot(), fmt.Sprintf("wrong value at index %d", i))
	}

	// do the same query, but with the final state saved
	// we should just get the final state w/o block to apply
	specs[5].savedState = true
	hist = newMockHistory(t, specs, five+1)
	endRoot = hist.slotMap[specs[len(specs)-1].slot]
	endBlock = hist.blocks[endRoot]

	cc = &canonicalChainer{h: hist, c: hist, cs: hist}
	st, bs, err = cc.ancestorChain(ctx, endBlock)
	require.NoError(t, err)
	require.DeepEqual(t, hist.states[hist.slotMap[one]], st)
	require.Equal(t, 0, len(bs))

	// slice off the last element for an odd size list (to cover odd/even in the reverseChain func)
	specs = specs[:len(specs)-1]
	require.Equal(t, 5, len(specs))
	hist = newMockHistory(t, specs, five+1)

	cc = &canonicalChainer{h: hist, c: hist, cs: hist}
	endRoot = hist.slotMap[specs[len(specs)-1].slot]
	endBlock = hist.blocks[endRoot]
	st, bs, err = cc.ancestorChain(ctx, endBlock)
	require.NoError(t, err)
	require.DeepEqual(t, hist.states[hist.slotMap[one]], st)
	require.Equal(t, 3, len(bs))
	for i, slot := range []types.Slot{two, three, four} {
		require.Equal(t, slot, bs[i].Block().Slot(), fmt.Sprintf("wrong value at index %d", i))
	}
}

// TODO
func TestBestForSlotValidation(t *testing.T) {
}
