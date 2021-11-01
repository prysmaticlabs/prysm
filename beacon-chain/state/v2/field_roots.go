package v2

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
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

var (
	leavesCache = make(map[string][][32]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
	layersCache = make(map[string][][][32]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
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
func computeFieldRoots(ctx context.Context, state *ethpb.BeaconStateAltair) ([][]byte, error) {
	if features.Get().EnableSSZCache {
		return cachedHasher.computeFieldRootsWithHasher(ctx, state)
	}
	return nocachedHasher.computeFieldRootsWithHasher(ctx, state)
}

func (h *stateRootHasher) computeFieldRootsWithHasher(ctx context.Context, state *ethpb.BeaconStateAltair) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.computeFieldRootsWithHasher")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	fieldRoots := make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Genesis validator root.
	fieldRoots[1] = state.GenesisValidatorsRoot[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.Slot))
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := h.arraysRoot(state.BlockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "BlockRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := h.arraysRoot(state.StateRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "StateRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(state.HistoricalRoots, params.BeaconConfig().HistoricalRootsLimit)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := eth1Root(hasher, state.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[8] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.Eth1DataVotes)
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
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := h.arraysRoot(state.RandaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector), "RandaoMixes")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochParticipation slice root.
	prevParticipationRoot, err := stateutil.ParticipationBitsRoot(state.PreviousEpochParticipation)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
	}
	fieldRoots[15] = prevParticipationRoot[:]

	// CurrentEpochParticipation slice root.
	currParticipationRoot, err := stateutil.ParticipationBitsRoot(state.CurrentEpochParticipation)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
	}
	fieldRoots[16] = currParticipationRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]

	// Inactivity scores root.
	inactivityScoresRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.InactivityScores)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
	}
	fieldRoots[21] = inactivityScoresRoot[:]

	// Current sync committee root.
	currentSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.CurrentSyncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute sync committee merkleization")
	}
	fieldRoots[22] = currentSyncCommitteeRoot[:]

	// Next sync committee root.
	nextSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.NextSyncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute sync committee merkleization")
	}
	fieldRoots[23] = nextSyncCommitteeRoot[:]

	return fieldRoots, nil
}
