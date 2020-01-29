package state

import (
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/protolambda/zssz/merkle"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	coreutils "github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
)

// BeaconState defines a struct containing utilities for the eth2 chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	state        *pbp2p.BeaconState
	lock         sync.RWMutex
	dirtyFields  map[fieldIndex]interface{}
	valIdxMap    map[[48]byte]uint64
	merkleLayers [][][]byte
}

// ReadOnlyValidator returns a wrapper that only allows fields from a validator
// to be read, and prevents any modification of internal validator fields.
type ReadOnlyValidator struct {
	validator *ethpb.Validator
}

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *pbp2p.BeaconState) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*pbp2p.BeaconState))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *pbp2p.BeaconState) (*BeaconState, error) {
	fieldRoots, err := stateutil.ComputeFieldRoots(st)
	if err != nil {
		return nil, err
	}
	layers := merkleize(fieldRoots)
	valMap := coreutils.ValidatorIndexMap(st.Validators)
	return &BeaconState{
		state:        st,
		merkleLayers: layers,
		dirtyFields:  make(map[fieldIndex]interface{}),
		valIdxMap:    valMap,
	}, nil
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the eth2 Simple Serialize specification.
func (b *BeaconState) HashTreeRoot() ([32]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if len(b.merkleLayers) == 0 {
		return [32]byte{}, errors.New("state merkle layers not initialized")
	}

	for field := range b.dirtyFields {
		root, err := b.rootSelector(field)
		if err != nil {
			return [32]byte{}, err
		}
		b.merkleLayers[0][field] = root[:]
		b.recomputeRoot(int(field))
		delete(b.dirtyFields, field)
	}
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0]), nil
}

// Merkleize 32-byte leaves into a Merkle trie for its adequate depth, returning
// the resulting layers of the trie based on the appropriate depth. This function
// pads the leaves to a power-of-two length.
func merkleize(leaves [][]byte) [][][]byte {
	hashFunc := hashutil.CustomSHA256Hasher()
	layers := make([][][]byte, merkle.GetDepth(uint64(len(leaves)))+1)
	for len(leaves) != 32 {
		leaves = append(leaves, make([]byte, 32))
	}
	currentLayer := leaves
	layers[0] = currentLayer

	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	for len(currentLayer) > 1 && i < len(layers) {
		layer := make([][]byte, 0)
		for i := 0; i < len(currentLayer); i += 2 {
			hashedChunk := hashFunc(append(currentLayer[i], currentLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		currentLayer = layer
		layers[i] = currentLayer
		i++
	}
	return layers
}

func (b *BeaconState) rootSelector(field fieldIndex) ([32]byte, error) {
	switch field {
	case genesisTime:
		return stateutil.Uint64Root(b.state.GenesisTime), nil
	case slot:
		return stateutil.Uint64Root(b.state.Slot), nil
	case eth1DepositIndex:
		return stateutil.Uint64Root(b.state.Eth1DepositIndex), nil
	case fork:
		return stateutil.ForkRoot(b.state.Fork)
	case latestBlockHeader:
		return stateutil.BlockHeaderRoot(b.state.LatestBlockHeader)
	case blockRoots:
		return stateutil.RootsArrayHashTreeRoot(b.state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	case stateRoots:
		return stateutil.RootsArrayHashTreeRoot(b.state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	case historicalRoots:
		return stateutil.HistoricalRootsRoot(b.state.HistoricalRoots)
	case eth1Data:
		return stateutil.Eth1Root(b.state.Eth1Data)
	case eth1DataVotes:
		return stateutil.Eth1DataVotesRoot(b.state.Eth1DataVotes)
	case validators:
		return stateutil.ValidatorRegistryRoot(b.state.Validators)
	case balances:
		return stateutil.ValidatorBalancesRoot(b.state.Balances)
	case randaoMixes:
		return stateutil.RootsArrayHashTreeRoot(b.state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
	case slashings:
		return stateutil.SlashingsRoot(b.state.Slashings)
	case previousEpochAttestations:
		return stateutil.EpochAttestationsRoot(b.state.PreviousEpochAttestations)
	case currentEpochAttestations:
		return stateutil.EpochAttestationsRoot(b.state.CurrentEpochAttestations)
	case justificationBits:
		return bytesutil.ToBytes32(b.state.JustificationBits), nil
	case previousJustifiedCheckpoint:
		return stateutil.CheckpointRoot(b.state.PreviousJustifiedCheckpoint)
	case currentJustifiedCheckpoint:
		return stateutil.CheckpointRoot(b.state.CurrentJustifiedCheckpoint)
	case finalizedCheckpoint:
		return stateutil.CheckpointRoot(b.state.FinalizedCheckpoint)
	}
	return [32]byte{}, errors.New("invalid field index provided")
}
