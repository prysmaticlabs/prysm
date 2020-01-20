package state

import (
	"sync"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

type BeaconState struct {
	state        *pbp2p.BeaconState
	lock         sync.RWMutex
	merkleLayers [][][]byte
}

func (b *BeaconState) HashTreeRoot() [32]byte {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0])
}

func (b *BeaconState) GenesisTime() uint64 {
	return b.state.GenesisTime
}

func (b *BeaconState) Slot() uint64 {
	return b.state.Slot
}

func (b *BeaconState) Fork() *pbp2p.Fork {
	return proto.Clone(b.state.Fork).(*pbp2p.Fork)
}

func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	return proto.Clone(b.state.LatestBlockHeader).(*ethpb.BeaconBlockHeader)
}

func (b *BeaconState) BlockRoots() [][]byte {
	res := make([][]byte, len(b.state.BlockRoots))
	copy(res, b.state.BlockRoots)
	return res
}

func (b *BeaconState) StateRoots() [][]byte {
	res := make([][]byte, len(b.state.StateRoots))
	copy(res, b.state.StateRoots)
	return res
}

func (b *BeaconState) HistoricalRoots() [][]byte {
	res := make([][]byte, len(b.state.HistoricalRoots))
	copy(res, b.state.HistoricalRoots)
	return res
}

func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	return proto.Clone(b.state.Eth1Data).(*ethpb.Eth1Data)
}

func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	// TODO: Clone this value.
	return b.state.Eth1DataVotes
}

func (b *BeaconState) Eth1DepositIndex() uint64 {
	return b.state.Eth1DepositIndex
}

func (b *BeaconState) Validators() []*ethpb.Validator {
	// TODO: Clone this value.
	return b.state.Validators
}

func (b *BeaconState) Balances() []uint64 {
	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

func (b *BeaconState) RandaoMixes() [][]byte {
	res := make([][]byte, len(b.state.RandaoMixes))
	copy(res, b.state.RandaoMixes)
	return res
}

func (b *BeaconState) Slashings() []uint64 {
	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	// TODO: Clone this value.
	return b.state.PreviousEpochAttestations
}

func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	// TODO: Clone this value.
	return b.state.CurrentEpochAttestations
}

func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	res := bitfield.Bitvector4{}
	copy(res, b.state.JustificationBits)
	return res
}

func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.PreviousJustifiedCheckpoint.Epoch,
		Root:  b.state.PreviousJustifiedCheckpoint.Root,
	}
}

func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.CurrentJustifiedCheckpoint.Epoch,
		Root:  b.state.CurrentJustifiedCheckpoint.Root,
	}
}

func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.FinalizedCheckpoint.Epoch,
		Root:  b.state.FinalizedCheckpoint.Root,
	}
}

func (b *BeaconState) recomputeRoot(idx int) {
	layers := b.merkleLayers
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	root := b.merkleLayers[0][idx]
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := make([]byte, 32)
		if layers[i] != nil && len(layers[i]) != 0 && neighborIdx < len(layers[i]) {
			neighbor = layers[i][neighborIdx]
		}
		if isLeft {
			parentHash := hashutil.Hash(append(root, neighbor...))
			root = parentHash[:]
		} else {
			parentHash := hashutil.Hash(append(neighbor, root...))
			root = parentHash[:]
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		layers[i+1][parentIdx] = root
		currentIndex = parentIdx
	}
	b.merkleLayers = layers
}

func merkleize(leaves [][]byte) [][][]byte {
	currentLayer := leaves
	layers := make([][][]byte, 5)
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
			hashedChunk := hashutil.Hash(append(currentLayer[i], currentLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		currentLayer = layer
		layers[i] = currentLayer
		i++
	}
	return layers
}
