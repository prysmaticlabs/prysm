package v1

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	"go.opencensus.io/trace"
)

var (
	leavesCache = make(map[string][][32]byte, params.BeaconConfig().BeaconStateFieldCount)
	layersCache = make(map[string][][][32]byte, params.BeaconConfig().BeaconStateFieldCount)
	lock        sync.RWMutex
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

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func computeFieldRoots(ctx context.Context, state *BeaconState) ([][]byte, error) {
	if features.Get().EnableSSZCache {
		return cachedHasher.computeFieldRootsWithHasher(ctx, state)
	}
	return nocachedHasher.computeFieldRootsWithHasher(ctx, state)
}

func (h *stateRootHasher) computeFieldRootsWithHasher(ctx context.Context, state *BeaconState) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.computeFieldRootsWithHasher")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	fieldRoots := make([][]byte, params.BeaconConfig().BeaconStateFieldCount)

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.genesisTimeInternal())
	fieldRoots[0] = genesisRoot[:]

	// Genesis validator root.
	genesisValidatorsRoot := state.genesisValidatorRoot()
	fieldRoots[1] = genesisValidatorsRoot[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.slotInternal()))
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.fork())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.latestBlockHeader())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	bRoots := make([][]byte, len(state.blockRoots()))
	for i := range bRoots {
		bRoots[i] = state.blockRoots()[i][:]
	}
	blockRootsRoot, err := h.arraysRoot(bRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "BlockRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	sRoots := make([][]byte, len(state.stateRoots()))
	for i := range sRoots {
		sRoots[i] = state.stateRoots()[i][:]
	}
	stateRootsRoot, err := h.arraysRoot(sRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "StateRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	hRoots := make([][]byte, len(state.historicalRoots()))
	for i := range hRoots {
		hRoots[i] = state.historicalRoots()[i][:]
	}
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(hRoots, params.BeaconConfig().HistoricalRootsLimit)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := eth1Root(hasher, state.eth1Data())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[8] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.eth1DataVotes())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[9] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.eth1DepositIndex())
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[10] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := h.validatorRegistryRoot(state.validators())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[11] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.balances())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	mixes := make([][]byte, len(state.randaoMixes()))
	for i := range mixes {
		mixes[i] = state.randaoMixes()[i][:]
	}
	randaoRootsRoot, err := h.arraysRoot(mixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector), "RandaoMixes")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.slashings())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoot, err := h.epochAttestationsRoot(state.previousEpochAttestations())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[15] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := h.epochAttestationsRoot(state.currentEpochAttestations())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
	}
	fieldRoots[16] = currAttsRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.justificationBits())
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.previousJustifiedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.currentJustifiedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.finalizedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]
	return fieldRoots, nil
}
