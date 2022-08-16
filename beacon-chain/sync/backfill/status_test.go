package backfill

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	blocktest "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks/testing"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

var errEmptyMockDBMethod = errors.New("uninitialized mock db method called")

type mockBackfillDB struct {
	saveBackfillBlockRoot     func(ctx context.Context, blockRoot [32]byte) error
	genesisBlockRoot          func(ctx context.Context) ([32]byte, error)
	originCheckpointBlockRoot func(ctx context.Context) ([32]byte, error)
	backfillBlockRoot         func(ctx context.Context) ([32]byte, error)
	block                     func(ctx context.Context, blockRoot [32]byte) (interfaces.SignedBeaconBlock, error)
}

var _ BackfillDB = &mockBackfillDB{}

func (db *mockBackfillDB) SaveBackfillBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	if db.saveBackfillBlockRoot != nil {
		return db.saveBackfillBlockRoot(ctx, blockRoot)
	}
	return errEmptyMockDBMethod
}

func (db *mockBackfillDB) GenesisBlockRoot(ctx context.Context) ([32]byte, error) {
	if db.genesisBlockRoot != nil {
		return db.genesisBlockRoot(ctx)
	}
	return [32]byte{}, errEmptyMockDBMethod
}

func (db *mockBackfillDB) OriginCheckpointBlockRoot(ctx context.Context) ([32]byte, error) {
	if db.originCheckpointBlockRoot != nil {
		return db.originCheckpointBlockRoot(ctx)
	}
	return [32]byte{}, errEmptyMockDBMethod
}

func (db *mockBackfillDB) BackfillBlockRoot(ctx context.Context) ([32]byte, error) {
	if db.backfillBlockRoot != nil {
		return db.backfillBlockRoot(ctx)
	}
	return [32]byte{}, errEmptyMockDBMethod
}

func (db *mockBackfillDB) Block(ctx context.Context, blockRoot [32]byte) (interfaces.SignedBeaconBlock, error) {
	if db.block != nil {
		return db.block(ctx, blockRoot)
	}
	return nil, errEmptyMockDBMethod
}

func TestSlotCovered(t *testing.T) {
	cases := []struct {
		name   string
		slot   types.Slot
		status *Status
		result bool
	}{
		{
			name:   "below start true",
			status: &Status{start: 1},
			slot:   0,
			result: true,
		},
		{
			name:   "above end true",
			status: &Status{end: 1},
			slot:   2,
			result: true,
		},
		{
			name:   "equal end true",
			status: &Status{end: 1},
			slot:   1,
			result: true,
		},
		{
			name:   "equal start true",
			status: &Status{start: 2},
			slot:   2,
			result: true,
		},
		{
			name:   "between false",
			status: &Status{start: 1, end: 3},
			slot:   2,
			result: false,
		},
		{
			name:   "genesisSync always true",
			status: &Status{genesisSync: true},
			slot:   100,
			result: true,
		},
	}
	for _, c := range cases {
		result := c.status.SlotCovered(c.slot)
		require.Equal(t, c.result, result)
	}
}

func TestAdvance(t *testing.T) {
	ctx := context.Background()
	saveBackfillBuf := make([][32]byte, 0)
	mdb := &mockBackfillDB{
		saveBackfillBlockRoot: func(ctx context.Context, root [32]byte) error {
			saveBackfillBuf = append(saveBackfillBuf, root)
			return nil
		},
	}
	s := &Status{end: 100, store: mdb}
	var root [32]byte
	copy(root[:], []byte{0x23, 0x23})
	require.NoError(t, s.Advance(ctx, 90, root))
	require.Equal(t, root, saveBackfillBuf[0])
	not := s.SlotCovered(95)
	require.Equal(t, false, not)

	// this should still be len 1 after failing to advance
	require.Equal(t, 1, len(saveBackfillBuf))
	require.ErrorIs(t, s.Advance(ctx, s.end+1, root), ErrAdvancePastOrigin)
	// this has an element in it from the previous test, there shouldn't be an additional one
	require.Equal(t, 1, len(saveBackfillBuf))
}

func goodBlockRoot(root [32]byte) func(ctx context.Context) ([32]byte, error) {
	return func(ctx context.Context) ([32]byte, error) {
		return root, nil
	}
}

func setupTestBlock(slot types.Slot) (interfaces.SignedBeaconBlock, error) {
	bRaw := util.NewBeaconBlock()
	b, err := blocks.NewSignedBeaconBlock(bRaw)
	if err != nil {
		return nil, err
	}
	return blocktest.SetBlockSlot(b, slot)
}

func TestReload(t *testing.T) {
	ctx := context.Background()
	derp := errors.New("derp")

	originSlot := types.Slot(100)
	var originRoot [32]byte
	copy(originRoot[:], []byte{0x01})
	originBlock, err := setupTestBlock(originSlot)
	require.NoError(t, err)

	backfillSlot := types.Slot(50)
	var backfillRoot [32]byte
	copy(originRoot[:], []byte{0x02})
	backfillBlock, err := setupTestBlock(backfillSlot)
	require.NoError(t, err)

	cases := []struct {
		name     string
		db       BackfillDB
		err      error
		expected *Status
	}{
		/*{
			name: "origin not found, implying genesis sync ",
			db: &mockBackfillDB{
				genesisBlockRoot: goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, db.ErrNotFoundOriginBlockRoot
				}},
			expected: &Status{genesisSync: true},
		},
		{
			name: "genesis not found error",
			err:  db.ErrNotFoundGenesisBlockRoot,
			db: &mockBackfillDB{
				genesisBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, db.ErrNotFoundGenesisBlockRoot
				},
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					}
					return nil, nil
				},
			},
		},
		{
			name: "other genesis error",
			err:  derp,
			db: &mockBackfillDB{
				genesisBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, derp
				},
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					}
					return nil, nil
				},
			},
		},
		{
			name: "origin other error",
			db: &mockBackfillDB{
				genesisBlockRoot: goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, derp
				}},
			err: derp,
		},
		{
			name: "origin root found, block missing",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					return nil, nil
				},
			},
			err: blocks.ErrNilSignedBeaconBlock,
		},
		{
			name: "origin root found, block error",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					return nil, derp
				},
			},
			err: derp,
		},
		{
			name: "origin root found, block found, backfill root not found",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					return originBlock, nil
				},
				backfillBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, db.ErrNotFoundBackfillBlockRoot
				},
			},
			err: db.ErrNotFoundBackfillBlockRoot,
		},
		{
			name: "origin root found, block found, random backfill root err",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					case backfillRoot:
						return nil, nil
					}
					return nil, derp
				},
				backfillBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, derp
				},
			},
			err: derp,
		},
		{
			name: "origin root found, block found, backfill root found, backfill block not found",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					case backfillRoot:
						return nil, nil
					}
					return nil, derp
				},
				backfillBlockRoot: goodBlockRoot(backfillRoot),
			},
			err: blocks.ErrNilSignedBeaconBlock,
		},
		{
			name: "origin root found, block found, backfill root found, backfill block random err",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					case backfillRoot:
						return nil, derp
					}
					return nil, errors.New("not derp")
				},
				backfillBlockRoot: goodBlockRoot(backfillRoot),
			},
			err: derp,
		},*/
		{
			name: "complete happy path",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.SignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					case backfillRoot:
						return backfillBlock, nil
					}
					return nil, errors.New("not derp")
				},
				backfillBlockRoot: goodBlockRoot(backfillRoot),
			},
			err:      derp,
			expected: &Status{genesisSync: false, start: backfillSlot, end: originSlot},
		},
	}

	for _, c := range cases {
		s := &Status{
			store: c.db,
		}
		err := s.Reload(ctx)
		if err != nil {
			require.ErrorIs(t, err, c.err)
			continue
		}
		require.NoError(t, err)
		if c.expected == nil {
			continue
		}
		require.Equal(t, c.expected.genesisSync, s.genesisSync)
		require.Equal(t, c.expected.start, s.start)
		require.Equal(t, c.expected.end, s.end)
	}
}
