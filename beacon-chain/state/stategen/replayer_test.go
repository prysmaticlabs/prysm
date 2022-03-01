package stategen

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block/mock"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
)

// happy path tests
func TestCanonicalBlockForSlotHappy(t *testing.T) {
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

func TestCanonicalBlockForSlotNonHappy(t *testing.T) {
	ctx := context.Background()
	var begin, middle, end types.Slot = 100, 150, 155
	specs := []mockHistorySpec{
		{slot: begin},
		{slot: middle, savedState: true},
		{slot: end, canonicalBlock: true},
	}
	hist := newMockHistory(t, specs, end+1)

	slotOrderObserved := make([]types.Slot, 0)
	derp := errors.New("HighestSlotBlocksBelow don't work")
	// since only the end block and genesis are canonical, once the slot drops below
	// end, we should always get genesis
	cases := []struct {
		name              string
		slot              types.Slot
		canon             CanonicalChecker
		overrideHighest   func(context.Context, types.Slot) ([]block.SignedBeaconBlock, error)
		slotOrderExpected []types.Slot
		err               error
		root              [32]byte
	}{
		{
			name: "HigestSlotBlocksBelow not called for genesis",
			overrideHighest: func(_ context.Context, _ types.Slot) ([]block.SignedBeaconBlock, error) {
				return nil, derp
			},
			root: hist.slotMap[0],
		},
		{
			name: "wrapped error from HigestSlotBlocksBelow returned",
			err:  derp,
			overrideHighest: func(_ context.Context, _ types.Slot) ([]block.SignedBeaconBlock, error) {
				return nil, derp
			},
			slot: end,
		},
		{
			name: "HigestSlotBlocksBelow empty list",
			err:  ErrNoBlocksBelowSlot,
			overrideHighest: func(_ context.Context, _ types.Slot) ([]block.SignedBeaconBlock, error) {
				return []block.SignedBeaconBlock{}, nil
			},
			slot: end,
		},
		{
			name:  "HigestSlotBlocksBelow no canonical",
			err:   ErrNoCanonicalBlockForSlot,
			canon: &mockCanonicalChecker{is: false},
			slot:  end,
		},
		{
			name: "slot ordering correct - only genesis canonical",
			canon: &mockCanonicalChecker{isCanon: func(root [32]byte) (bool, error) {
				if root == hist.slotMap[0] {
					return true, nil
				}
				return false, nil
			}},
			overrideHighest: func(_ context.Context, s types.Slot) ([]block.SignedBeaconBlock, error) {
				slotOrderObserved = append(slotOrderObserved, s)
				// this allows the mock HighestSlotBlocksBelow to continue to execute now that we've recorded
				// the slot in our channel
				return nil, errFallThroughOverride
			},
			slotOrderExpected: []types.Slot{156, 155, 150, 100},
			slot:              end,
			root:              hist.slotMap[0],
		},
		{
			name: "slot ordering correct - slot 100 canonical",
			canon: &mockCanonicalChecker{isCanon: func(root [32]byte) (bool, error) {
				if root == hist.slotMap[100] {
					return true, nil
				}
				return false, nil
			}},
			overrideHighest: func(_ context.Context, s types.Slot) ([]block.SignedBeaconBlock, error) {
				slotOrderObserved = append(slotOrderObserved, s)
				// this allows the mock HighestSlotBlocksBelow to continue to execute now that we've recorded
				// the slot in our channel
				return nil, errFallThroughOverride
			},
			slotOrderExpected: []types.Slot{156, 155, 150},
			slot:              end,
			root:              hist.slotMap[100],
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var canon CanonicalChecker = hist
			if c.canon != nil {
				canon = c.canon
			}
			cc := canonicalChainer{hist, canon, hist}
			hist.overrideHighestSlotBlocksBelow = c.overrideHighest
			r, _, err := cc.canonicalBlockForSlot(ctx, c.slot)
			if c.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, c.err)
			}
			if len(c.slotOrderExpected) > 0 {
				require.Equal(t, len(c.slotOrderExpected), len(slotOrderObserved), "HighestSlotBlocksBelow not called the expected number of times")
				for i := range c.slotOrderExpected {
					require.Equal(t, c.slotOrderExpected[i], slotOrderObserved[i])
				}
			}
			if c.root != [32]byte{} {
				require.Equal(t, c.root, r)
			}
			slotOrderObserved = make([]types.Slot, 0)
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

func TestChainForSlot(t *testing.T) {
	ctx := context.Background()
	var zero, one, two, three types.Slot = 50, 51, 150, 151
	specs := []mockHistorySpec{
		{slot: zero, canonicalBlock: true, savedState: true},
		{slot: one, canonicalBlock: true},
		{slot: two},
		{slot: three, canonicalBlock: true},
	}
	hist := newMockHistory(t, specs, three+10)
	cc := &canonicalChainer{h: hist, c: hist, cs: hist}
	firstNonGenesisRoot := hist.slotMap[zero]
	nonGenesisStateRoot, err := hist.states[firstNonGenesisRoot].HashTreeRoot(ctx)
	require.NoError(t, err)

	cases := []struct {
		name       string
		slot       types.Slot
		stateRoot  [32]byte
		blockRoots [][32]byte
	}{
		{
			name:       "above latest slot (but before current slot)",
			slot:       three + 1,
			stateRoot:  nonGenesisStateRoot,
			blockRoots: [][32]byte{hist.slotMap[one], hist.slotMap[two], hist.slotMap[three]},
		},
		{
			name:       "last canonical slot - two treated as canonical because it is parent of three",
			slot:       three,
			stateRoot:  nonGenesisStateRoot,
			blockRoots: [][32]byte{hist.slotMap[one], hist.slotMap[two], hist.slotMap[three]},
		},
		{
			name:       "non-canonical slot skipped",
			slot:       two,
			stateRoot:  nonGenesisStateRoot,
			blockRoots: [][32]byte{hist.slotMap[one]},
		},
		{
			name:       "first canonical slot",
			slot:       one,
			stateRoot:  nonGenesisStateRoot,
			blockRoots: [][32]byte{hist.slotMap[one]},
		},
		{
			name:       "slot at saved state",
			slot:       zero,
			stateRoot:  nonGenesisStateRoot,
			blockRoots: [][32]byte{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, blocks, err := cc.chainForSlot(ctx, c.slot)
			require.NoError(t, err)
			actualStRoot, err := st.HashTreeRoot(ctx)
			require.NoError(t, err)
			require.Equal(t, c.stateRoot, actualStRoot)
			require.Equal(t, len(c.blockRoots), len(blocks))
			for i, b := range blocks {
				root, err := b.Block().HashTreeRoot()
				require.NoError(t, err)
				require.Equal(t, c.blockRoots[i], root)
			}
		})
	}
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

func TestBestForSlot(t *testing.T) {
	nilBlock, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlock{})
	require.NoError(t, err)
	nilBody, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}})
	require.NoError(t, err)
	derp := errors.New("fake hash tree root method no hash good")
	badHTR := &mock.SignedBeaconBlock{BeaconBlock: &mock.BeaconBlock{HtrErr: derp, BeaconBlockBody: &mock.BeaconBlockBody{}}}
	var goodHTR [32]byte
	copy(goodHTR[:], []byte{23})
	var betterHTR [32]byte
	copy(betterHTR[:], []byte{42})
	good := &mock.SignedBeaconBlock{BeaconBlock: &mock.BeaconBlock{BeaconBlockBody: &mock.BeaconBlockBody{}, Htr: goodHTR}}
	better := &mock.SignedBeaconBlock{BeaconBlock: &mock.BeaconBlock{BeaconBlockBody: &mock.BeaconBlockBody{}, Htr: betterHTR}}

	cases := []struct {
		name   string
		err    error
		blocks []block.SignedBeaconBlock
		best   block.SignedBeaconBlock
		root   [32]byte
		cc     CanonicalChecker
	}{
		{
			name:   "empty list",
			err:    ErrNoCanonicalBlockForSlot,
			blocks: []block.SignedBeaconBlock{},
		},
		{
			name:   "empty SignedBeaconBlock",
			err:    ErrNoCanonicalBlockForSlot,
			blocks: []block.SignedBeaconBlock{nil},
		},
		{
			name:   "empty BeaconBlock",
			err:    ErrNoCanonicalBlockForSlot,
			blocks: []block.SignedBeaconBlock{nilBlock},
		},
		{
			name:   "empty BeaconBlockBody",
			err:    ErrNoCanonicalBlockForSlot,
			blocks: []block.SignedBeaconBlock{nilBody},
		},
		{
			name:   "bad HTR",
			err:    ErrInvalidDBBlock,
			blocks: []block.SignedBeaconBlock{badHTR},
		},
		{
			name:   "IsCanonical fail",
			blocks: []block.SignedBeaconBlock{good, better},
			cc:     &mockCanonicalChecker{is: true, err: derp},
			err:    derp,
		},
		{
			name:   "all non-canonical",
			err:    ErrNoCanonicalBlockForSlot,
			blocks: []block.SignedBeaconBlock{good, better},
			cc:     &mockCanonicalChecker{is: false},
		},
		{
			name:   "one canonical",
			blocks: []block.SignedBeaconBlock{good},
			cc:     &mockCanonicalChecker{is: true},
			root:   goodHTR,
			best:   good,
		},
		{
			name:   "all canonical",
			blocks: []block.SignedBeaconBlock{better, good},
			cc:     &mockCanonicalChecker{is: true},
			root:   betterHTR,
			best:   better,
		},
		{
			name:   "first wins",
			blocks: []block.SignedBeaconBlock{good, better},
			cc:     &mockCanonicalChecker{is: true},
			root:   goodHTR,
			best:   good,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			chk := CanonicalChecker(&mockCanonicalChecker{is: true})
			if c.cc != nil {
				chk = c.cc
			}
			cc := &canonicalChainer{c: chk}
			r, b, err := cc.bestForSlot(context.Background(), c.blocks)
			if c.err == nil {
				require.NoError(t, err)
				require.DeepEqual(t, c.best, b)
				require.Equal(t, c.root, r)
			} else {
				require.ErrorIs(t, err, c.err)
			}
		})
	}
}

type mockCanonicalChecker struct {
	isCanon func([32]byte) (bool, error)
	is      bool
	err     error
}

func (m *mockCanonicalChecker) IsCanonical(_ context.Context, root [32]byte) (bool, error) {
	if m.isCanon != nil {
		return m.isCanon(root)
	}
	return m.is, m.err
}

func TestReverseChain(t *testing.T) {
	// test 0,1,2,3 elements to handle: zero case; single element; even number; odd number
	for i := 0; i < 4; i ++ {
		t.Run(fmt.Sprintf("reverseChain with %d elements", i), func(t *testing.T){
			actual := mockBlocks(i, incrFwd)
			expected := mockBlocks(i, incrBwd)
			reverseChain(actual)
			if len(actual) != len(expected) {
				t.Errorf("different list lengths")
			}
			for i := 0; i < len(actual); i++ {
				sblockA, ok := actual[i].(*mock.SignedBeaconBlock)
				require.Equal(t, true, ok)
				blockA, ok := sblockA.BeaconBlock.(*mock.BeaconBlock)
				require.Equal(t, true, ok)
				sblockE, ok := expected[i].(*mock.SignedBeaconBlock)
				require.Equal(t, true, ok)
				blockE, ok := sblockE.BeaconBlock.(*mock.BeaconBlock)
				require.Equal(t, true, ok)
				require.Equal(t, blockA.Htr, blockE.Htr)
			}
		})
	}
}

func incrBwd(n int, c chan uint32) {
	for i := n-1; i >= 0; i-- {
		c <- uint32(i)
	}
	close(c)
}

func incrFwd(n int, c chan uint32) {
	for i := 0; i < n; i++ {
		c <- uint32(i)
	}
	close(c)
}

func mockBlocks(n int, iter func(int, chan uint32)) []block.SignedBeaconBlock {
	bchan := make(chan uint32)
	go iter(n, bchan)
	mb := make([]block.SignedBeaconBlock, 0)
	for i := range bchan {
		h := [32]byte{}
		binary.LittleEndian.PutUint32(h[:], i)
		b := &mock.SignedBeaconBlock{BeaconBlock: &mock.BeaconBlock{BeaconBlockBody: &mock.BeaconBlockBody{}, Htr: h}}
		mb = append(mb, b)
	}
	return mb
}
