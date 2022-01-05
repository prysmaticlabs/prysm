// Package helpers contains helper functions outlined in the Ethereum Beacon Chain spec, such as
// computing committees, randao, rewards/penalties, and more.
package helpers

import (
	"bytes"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
)

var (
	syncCommitteeCache = cache.NewSyncCommittee()
)

// IsCurrentPeriodSyncCommittee returns true if the input validator index belongs in the current period sync committee
// along with the sync committee root.
// 1. Checks if the public key exists in the sync committee cache
// 2. If 1 fails, checks if the public key exists in the input current sync committee object
func IsCurrentPeriodSyncCommittee(
	st state.BeaconStateAltair, valIdx types.ValidatorIndex,
) (bool, error) {
	root, err := syncPeriodBoundaryRoot(st)
	if err != nil {
		return false, err
	}
	indices, err := syncCommitteeCache.CurrentPeriodIndexPosition(bytesutil.ToBytes32(root), valIdx)
	if err == cache.ErrNonExistingSyncCommitteeKey {
		val, err := st.ValidatorAtIndex(valIdx)
		if err != nil {
			return false, nil
		}
		committee, err := st.CurrentSyncCommittee()
		if err != nil {
			return false, err
		}

		// Fill in the cache on miss.
		go func() {
			if err := syncCommitteeCache.UpdatePositionsInCommittee(bytesutil.ToBytes32(root), st); err != nil {
				log.Errorf("Could not fill sync committee cache on miss: %v", err)
			}
		}()

		return len(findSubCommitteeIndices(val.PublicKey, committee.Pubkeys)) > 0, nil
	}
	if err != nil {
		return false, err
	}
	return len(indices) > 0, nil
}

// IsNextPeriodSyncCommittee returns true if the input validator index belongs in the next period sync committee
// along with the sync period boundary root.
// 1. Checks if the public key exists in the sync committee cache
// 2. If 1 fails, checks if the public key exists in the input next sync committee object
func IsNextPeriodSyncCommittee(
	st state.BeaconStateAltair, valIdx types.ValidatorIndex,
) (bool, error) {
	root, err := syncPeriodBoundaryRoot(st)
	if err != nil {
		return false, err
	}
	indices, err := syncCommitteeCache.NextPeriodIndexPosition(bytesutil.ToBytes32(root), valIdx)
	if err == cache.ErrNonExistingSyncCommitteeKey {
		val, err := st.ValidatorAtIndex(valIdx)
		if err != nil {
			return false, nil
		}
		committee, err := st.NextSyncCommittee()
		if err != nil {
			return false, err
		}
		return len(findSubCommitteeIndices(val.PublicKey, committee.Pubkeys)) > 0, nil
	}
	if err != nil {
		return false, err
	}
	return len(indices) > 0, nil
}

// CurrentPeriodSyncSubcommitteeIndices returns the subcommittee indices of the
// current period sync committee for input validator.
func CurrentPeriodSyncSubcommitteeIndices(
	st state.BeaconStateAltair, valIdx types.ValidatorIndex,
) ([]types.CommitteeIndex, error) {
	root, err := syncPeriodBoundaryRoot(st)
	if err != nil {
		return nil, err
	}
	indices, err := syncCommitteeCache.CurrentPeriodIndexPosition(bytesutil.ToBytes32(root), valIdx)
	if err == cache.ErrNonExistingSyncCommitteeKey {
		val, err := st.ValidatorAtIndex(valIdx)
		if err != nil {
			return nil, nil
		}
		committee, err := st.CurrentSyncCommittee()
		if err != nil {
			return nil, err
		}

		// Fill in the cache on miss.
		go func() {
			if err := syncCommitteeCache.UpdatePositionsInCommittee(bytesutil.ToBytes32(root), st); err != nil {
				log.Errorf("Could not fill sync committee cache on miss: %v", err)
			}
		}()

		return findSubCommitteeIndices(val.PublicKey, committee.Pubkeys), nil
	}
	if err != nil {
		return nil, err
	}
	return indices, nil
}

// NextPeriodSyncSubcommitteeIndices returns the subcommittee indices of the next period sync committee for input validator.
func NextPeriodSyncSubcommitteeIndices(
	st state.BeaconStateAltair, valIdx types.ValidatorIndex,
) ([]types.CommitteeIndex, error) {
	root, err := syncPeriodBoundaryRoot(st)
	if err != nil {
		return nil, err
	}
	indices, err := syncCommitteeCache.NextPeriodIndexPosition(bytesutil.ToBytes32(root), valIdx)
	if err == cache.ErrNonExistingSyncCommitteeKey {
		val, err := st.ValidatorAtIndex(valIdx)
		if err != nil {
			return nil, nil
		}
		committee, err := st.NextSyncCommittee()
		if err != nil {
			return nil, err
		}
		return findSubCommitteeIndices(val.PublicKey, committee.Pubkeys), nil
	}
	if err != nil {
		return nil, err
	}
	return indices, nil
}

// UpdateSyncCommitteeCache updates sync committee cache.
// It uses `state`'s latest block header root as key. To avoid misuse, it disallows
// block header with state root zeroed out.
func UpdateSyncCommitteeCache(st state.BeaconStateAltair) error {
	nextSlot := st.Slot() + 1
	if nextSlot%params.BeaconConfig().SlotsPerEpoch != 0 {
		return errors.New("not at the end of the epoch to update cache")
	}
	if slots.ToEpoch(nextSlot)%params.BeaconConfig().EpochsPerSyncCommitteePeriod != 0 {
		return errors.New("not at sync committee period boundary to update cache")
	}

	header := st.LatestBlockHeader()
	if bytes.Equal(header.StateRoot, params.BeaconConfig().ZeroHash[:]) {
		return errors.New("zero hash state root can't be used to update cache")
	}

	prevBlockRoot, err := header.HashTreeRoot()
	if err != nil {
		return err
	}

	return syncCommitteeCache.UpdatePositionsInCommittee(prevBlockRoot, st)
}

// Loop through `pubKeys` for matching `pubKey` and get the indices where it matches.
func findSubCommitteeIndices(pubKey []byte, pubKeys [][]byte) []types.CommitteeIndex {
	var indices []types.CommitteeIndex
	for i, k := range pubKeys {
		if bytes.Equal(k, pubKey) {
			indices = append(indices, types.CommitteeIndex(i))
		}
	}
	return indices
}

// Retrieve the current sync period boundary root by calculating sync period start epoch
// and calling `BlockRoot`.
// It uses the boundary slot - 1 for block root. (Ex: SlotsPerEpoch * EpochsPerSyncCommitteePeriod - 1)
func syncPeriodBoundaryRoot(st state.ReadOnlyBeaconState) ([]byte, error) {
	// Can't call `BlockRoot` until the first slot.
	if st.Slot() == params.BeaconConfig().GenesisSlot {
		return params.BeaconConfig().ZeroHash[:], nil
	}

	startEpoch, err := slots.SyncCommitteePeriodStartEpoch(time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}
	startEpochSlot, err := slots.EpochStart(startEpoch)
	if err != nil {
		return nil, err
	}

	// Prevent underflow
	if startEpochSlot >= 1 {
		startEpochSlot--
	}

	return BlockRootAtSlot(st, startEpochSlot)
}
