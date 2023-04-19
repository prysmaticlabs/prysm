package backfill

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	blocktest "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks/testing"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/proto/dbval"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

var errEmptyMockDBMethod = errors.New("uninitialized mock db method called")

type mockBackfillDB struct {
	saveBackfillBlockRoot     func(ctx context.Context, blockRoot [32]byte) error
	genesisBlockRoot          func(ctx context.Context) ([32]byte, error)
	originCheckpointBlockRoot func(ctx context.Context) ([32]byte, error)
	block                     func(ctx context.Context, blockRoot [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error)
	saveBackfillStatus        func(ctx context.Context, status *dbval.BackfillStatus) error
	backfillStatus            func(context.Context) (*dbval.BackfillStatus, error)
	status                    *dbval.BackfillStatus
	err                       error
}

var _ BackfillDB = &mockBackfillDB{}

func (db *mockBackfillDB) SaveBackfillStatus(ctx context.Context, status *dbval.BackfillStatus) error {
	if db.saveBackfillStatus != nil {
		return db.saveBackfillStatus(ctx, status)
	}
	db.status = status
	return nil
}

func (db *mockBackfillDB) BackfillStatus(ctx context.Context) (*dbval.BackfillStatus, error) {
	if db.backfillStatus != nil {
		return db.backfillStatus(ctx)
	}
	return db.status, nil
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

func (db *mockBackfillDB) Block(ctx context.Context, blockRoot [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	if db.block != nil {
		return db.block(ctx, blockRoot)
	}
	return nil, errEmptyMockDBMethod
}

func TestSlotCovered(t *testing.T) {
	cases := []struct {
		name   string
		slot   primitives.Slot
		status *StatusUpdater
		result bool
	}{
		{
			name:   "below start true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 1}},
			slot:   0,
			result: true,
		},
		{
			name:   "above end true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{HighSlot: 1}},
			slot:   2,
			result: true,
		},
		{
			name:   "equal end true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{HighSlot: 1}},
			slot:   1,
			result: true,
		},
		{
			name:   "equal start true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 2}},
			slot:   2,
			result: true,
		},
		{
			name:   "between false",
			status: &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 1, HighSlot: 3}},
			slot:   2,
			result: false,
		},
		{
			name:   "genesisSync always true",
			status: &StatusUpdater{genesisSync: true},
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
	s := &StatusUpdater{status: &dbval.BackfillStatus{HighSlot: 100}, store: mdb}
	var root [32]byte
	copy(root[:], []byte{0x23, 0x23})
	require.NoError(t, s.FillFwd(ctx, 90, root))
	require.Equal(t, root, saveBackfillBuf[0])
	not := s.SlotCovered(95)
	require.Equal(t, false, not)

	// this should still be len 1 after failing to advance
	require.Equal(t, 1, len(saveBackfillBuf))
	require.ErrorIs(t, s.FillFwd(ctx, primitives.Slot(s.status.HighSlot)+1, root), ErrFillFwdPastUpper)
	// this has an element in it from the previous test, there shouldn't be an additional one
	require.Equal(t, 1, len(saveBackfillBuf))
}

func goodBlockRoot(root [32]byte) func(ctx context.Context) ([32]byte, error) {
	return func(ctx context.Context) ([32]byte, error) {
		return root, nil
	}
}

func setupTestBlock(slot primitives.Slot) (interfaces.ReadOnlySignedBeaconBlock, error) {
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

	originSlot := primitives.Slot(100)
	var originRoot [32]byte
	copy(originRoot[:], []byte{0x01})
	originBlock, err := setupTestBlock(originSlot)
	require.NoError(t, err)

	backfillSlot := primitives.Slot(50)
	var backfillRoot [32]byte
	copy(originRoot[:], []byte{0x02})
	backfillBlock, err := setupTestBlock(backfillSlot)
	require.NoError(t, err)

	cases := []struct {
		name     string
		db       BackfillDB
		err      error
		expected *StatusUpdater
	}{
		/*{
			name: "origin not found, implying genesis sync ",
			db: &mockBackfillDB{
				genesisBlockRoot: goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, db.ErrNotFoundOriginBlockRoot
				}},
			expected: &StatusUpdater{genesisSync: true},
		},
		{
			name: "genesis not found error",
			err:  db.ErrNotFoundGenesisBlockRoot,
			db: &mockBackfillDB{
				genesisBlockRoot: func(ctx context.Context) ([32]byte, error) {
					return [32]byte{}, db.ErrNotFoundGenesisBlockRoot
				},
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
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
			name: "legacy recovery",
			db: &mockBackfillDB{
				genesisBlockRoot:          goodBlockRoot(params.BeaconConfig().ZeroHash),
				originCheckpointBlockRoot: goodBlockRoot(originRoot),
				block: func(ctx context.Context, root [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
					switch root {
					case originRoot:
						return originBlock, nil
					case backfillRoot:
						return backfillBlock, nil
					}
					return nil, errors.New("not derp")
				},
				backfillStatus: func(context.Context) (*dbval.BackfillStatus, error) { return nil, db.ErrNotFound },
			},
			err:      derp,
			expected: &StatusUpdater{genesisSync: false, status: &dbval.BackfillStatus{LowSlot: 0, HighSlot: uint64(originSlot)}},
		},
	}

	for _, c := range cases {
		s := &StatusUpdater{
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
		require.Equal(t, c.expected.status.LowSlot, s.status.LowSlot)
		require.Equal(t, c.expected.status.HighSlot, s.status.HighSlot)
	}
}
