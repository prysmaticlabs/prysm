// Package stateutil defines utility functions to compute state roots
// using advanced merkle branch caching techniques.
package stateutil

import (
	"encoding/binary"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const cacheSize = 100000

var nocachedHasher *stateRootHasher
var cachedHasher *stateRootHasher

func init() {
	rootsCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 22,   // maximum cost of cache (3MB).
		// 100,000 roots will take up approximately 3 MB in memory.
		BufferItems: 64, // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
	// Temporarily disable roots cache until cache issues can be resolved.
	cachedHasher = &stateRootHasher{rootsCache: rootsCache}
	nocachedHasher = &stateRootHasher{}
}

type stateRootHasher struct {
	rootsCache *ristretto.Cache
}

// HashTreeRootState provides a fully-customized version of ssz.HashTreeRoot
// for the BeaconState type of the official Ethereum Serenity specification.
// The reason for this particular function is to optimize for speed and memory allocation
// at the expense of complete specificity (that is, this function can only be used
// on the Prysm BeaconState data structure).
func HashTreeRootState(state *pb.BeaconState) ([32]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.hashTreeRootState(state)
	}
	return nocachedHasher.hashTreeRootState(state)
}

// ComputeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func ComputeFieldRoots(state *pb.BeaconState) ([][]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.computeFieldRoots(state)
	}
	return nocachedHasher.computeFieldRoots(state)
}

func (h *stateRootHasher) hashTreeRootState(state *pb.BeaconState) ([32]byte, error) {
	var fieldRoots [][]byte
	var err error
	if featureconfig.Get().EnableSSZCache {
		fieldRoots, err = cachedHasher.computeFieldRoots(state)
		if err != nil {
			return [32]byte{}, err
		}
	} else {
		fieldRoots, err = nocachedHasher.computeFieldRoots(state)
		if err != nil {
			return [32]byte{}, err
		}
	}
	return htrutils.BitwiseMerkleize(hashutil.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func (h *stateRootHasher) computeFieldRoots(state *pb.BeaconState) ([][]byte, error) {
	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hashutil.CustomSHA256Hasher()
	// There are 21 fields in the beacon state.
	fieldRoots := make([][]byte, 21)

	// Genesis time root.
	genesisRoot := htrutils.Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Genesis validator root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot)
	fieldRoots[1] = r[:]

	// Slot root.
	slotRoot := htrutils.Uint64Root(state.Slot)
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := htrutils.ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := BlockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := h.arraysRoot(state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := h.arraysRoot(state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := htrutils.HistoricalRootsRoot(state.HistoricalRoots)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := Eth1Root(hasher, state.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[8] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := Eth1DataVotesRoot(state.Eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[9] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[10] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := h.validatorRegistryRoot(state.Validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[11] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := ValidatorBalancesRoot(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := h.arraysRoot(state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := htrutils.SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoot, err := h.epochAttestationsRoot(state.PreviousEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[15] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := h.epochAttestationsRoot(state.CurrentEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
	}
	fieldRoots[16] = currAttsRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := htrutils.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := htrutils.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := htrutils.CheckpointRoot(hasher, state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]
	return fieldRoots, nil
}
