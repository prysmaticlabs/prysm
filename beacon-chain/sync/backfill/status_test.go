package backfill

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
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
	states                    map[[32]byte]state.BeaconState
	blocks                    map[[32]byte]interfaces.ReadOnlySignedBeaconBlock
}

var _ BackfillDB = &mockBackfillDB{}

func (d *mockBackfillDB) StateOrError(_ context.Context, blockRoot [32]byte) (state.BeaconState, error) {
	st, ok := d.states[blockRoot]
	if !ok {
		return nil, db.ErrNotFoundState
	}
	return st, nil
}

func (d *mockBackfillDB) SaveBackfillStatus(ctx context.Context, status *dbval.BackfillStatus) error {
	if d.saveBackfillStatus != nil {
		return d.saveBackfillStatus(ctx, status)
	}
	d.status = status
	return nil
}

func (d *mockBackfillDB) BackfillStatus(ctx context.Context) (*dbval.BackfillStatus, error) {
	if d.backfillStatus != nil {
		return d.backfillStatus(ctx)
	}
	return d.status, nil
}

func (d *mockBackfillDB) GenesisBlockRoot(ctx context.Context) ([32]byte, error) {
	if d.genesisBlockRoot != nil {
		return d.genesisBlockRoot(ctx)
	}
	return [32]byte{}, errEmptyMockDBMethod
}

func (d *mockBackfillDB) OriginCheckpointBlockRoot(ctx context.Context) ([32]byte, error) {
	if d.originCheckpointBlockRoot != nil {
		return d.originCheckpointBlockRoot(ctx)
	}
	return [32]byte{}, errEmptyMockDBMethod
}

func (d *mockBackfillDB) Block(ctx context.Context, blockRoot [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	if d.block != nil {
		return d.block(ctx, blockRoot)
	}
	b, ok := d.blocks[blockRoot]
	if !ok {
		return nil, db.ErrNotFound
	}
	return b, nil
}

func (d *mockBackfillDB) SaveBlock(ctx context.Context, signed interfaces.ReadOnlySignedBeaconBlock) error {
	if d.blocks == nil {
		d.blocks = make(map[[32]byte]interfaces.ReadOnlySignedBeaconBlock)
	}
	r, err := signed.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	d.blocks[r] = signed
	return nil
}

func TestSlotCovered(t *testing.T) {
	cases := []struct {
		name   string
		slot   primitives.Slot
		status *StatusUpdater
		result bool
	}{
		{
			name:   "genesis true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 10}},
			slot:   0,
			result: true,
		},
		{
			name:   "above end true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 1}},
			slot:   2,
			result: true,
			{}},
		{
			name:   "equal end true",
			status: &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 1}},
			slot:   1,
			result: true,
		},
		{
			name:   "genesisSync always true",
			status: &StatusUpdater{genesisSync: true},
			slot:   100,
			result: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.status.SlotCovered(c.slot)
			require.Equal(t, c.result, result)
		})
	}
}

func TestStatusUpdater_FillBack(t *testing.T) {
	ctx := context.Background()
	mdb := &mockBackfillDB{}
	s := &StatusUpdater{status: &dbval.BackfillStatus{LowSlot: 100}, store: mdb}
	b, err := setupTestBlock(90)
	require.NoError(t, err)
	rob, err := blocks.NewROBlock(b)
	require.NoError(t, err)
	require.NoError(t, s.FillBack(ctx, rob))
	require.Equal(t, true, s.SlotCovered(95))
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
			expected: &StatusUpdater{genesisSync: false, status: &dbval.BackfillStatus{LowSlot: uint64(originSlot)}},
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
	}
}
