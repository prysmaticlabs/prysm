package stategen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	beacondb "github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

type ActiveValidatorCounter interface {
	ActiveValidatorCount(ctx context.Context) (uint64, error)
}

type LastFinalizedValidatorCounter struct {
	count uint64
	db beacondb.HeadAccessDatabase
	sm StateManager
}

func (lf *LastFinalizedValidatorCounter) ActiveValidatorCount(ctx context.Context) (uint64, error) {
	if lf.count != 0 {
		return lf.count, nil
	}
	cp, err := lf.db.FinalizedCheckpoint(ctx)
	if err != nil {
		return 0, err
	}

	r := bytesutil.ToBytes32(cp.Root)
	// Consider edge case where finalized root are zeros instead of genesis root hash.
	if r == params.BeaconConfig().ZeroHash {
		genesisBlock, err := lf.db.GenesisBlock(ctx)
		if err != nil {
			return 0, err
		}
		if genesisBlock != nil && !genesisBlock.IsNil() {
			r, err = genesisBlock.Block().HashTreeRoot()
			if err != nil {
				return 0, err
			}
		}
	}
	st, err := lf.sm.StateByRoot(ctx, r)
	if err != nil {
		return 0, err
	}
	if st == nil || st.IsNil() {
		return 0, errors.Wrapf(errUnknownState, "could not retrieve state with root=%#x", r)
	}
	vc, err := helpers.ActiveValidatorCount(context.Background(), st, coreTime.CurrentEpoch(st))
	if err != nil {
		return 0, err
	}
	lf.count = vc
	return lf.count, nil
}

func NewLastFinalizedValidatorCounter(count uint64, db beacondb.HeadAccessDatabase, sg StateManager) *LastFinalizedValidatorCounter {
	return &LastFinalizedValidatorCounter{
		count: count,
		db:    db,
		sm:    sg,
	}
}
