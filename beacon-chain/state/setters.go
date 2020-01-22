package state

import (
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
)

type fieldIndex int

// Below we define a set of useful enum values for the field
// indices of the beacon state. For example, genesisTime is the
// 0th field of the beacon state. This is helpful when we are
// updating the Merkle branches up the trie representation
// of the beacon state.
const (
	genesisTime fieldIndex = iota
	slot
	fork
	latestBlockHeader
	blockRoots
	stateRoots
	historicalRoots
	eth1Data
	eth1DataVotes
	eth1DepositIndex
	validators
	balances
	randaoMixes
	slashings
	previousEpochAttestations
	currentEpochAttestations
	justificationBits
	previousJustifiedCheckpoint
	currentJustifiedCheckpoint
	finalizedCheckpoint
)

// SetGenesisTime for the beacon state.
func (b *BeaconState) SetGenesisTime(val uint64) error {
	b.state.GenesisTime = val
	root := stateutil.Uint64Root(val)
	b.lock.Lock()
	b.merkleLayers[0][genesisTime] = root[:]
	b.recomputeRoot(int(genesisTime))
	b.lock.Unlock()
	return nil
}

// SetSlot for the beacon state.
func (b *BeaconState) SetSlot(val uint64) error {
	b.state.Slot = val
	root := stateutil.Uint64Root(val)
	b.lock.Lock()
	b.merkleLayers[0][slot] = root[:]
	b.recomputeRoot(int(slot))
	b.lock.Unlock()
	return nil
}

// SetFork version for the beacon chain.
func (b *BeaconState) SetFork(val *pbp2p.Fork) error {
	root, err := stateutil.ForkRoot(val)
	if err != nil {
		return err
	}
	b.state.Fork = val
	b.lock.Lock()
	b.merkleLayers[0][fork] = root[:]
	b.recomputeRoot(int(fork))
	b.lock.Unlock()
	return nil
}

// SetLatestBlockHeader in the beacon state.
func (b *BeaconState) SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) error {
	root, err := stateutil.BlockHeaderRoot(val)
	if err != nil {
		return err
	}
	b.state.LatestBlockHeader = val
	b.lock.Lock()
	b.merkleLayers[0][latestBlockHeader] = root[:]
	b.recomputeRoot(int(latestBlockHeader))
	b.lock.Unlock()
	return nil
}

// SetBlockRoots for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBlockRoots(val [][]byte) error {
	root, err := stateutil.RootsArrayHashTreeRoot(val, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	if err != nil {
		return err
	}
	b.state.BlockRoots = val
	b.lock.Lock()
	b.merkleLayers[0][blockRoots] = root[:]
	b.recomputeRoot(int(blockRoots))
	b.lock.Unlock()
	return nil
}

// UpdateBlockRootAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error {
	if len(b.state.BlockRoots) <= int(idx) {
		return fmt.Errorf("invalid index provided %d", idx)
	}
	b.state.BlockRoots[idx] = blockRoot[:]
	root, err := stateutil.RootsArrayHashTreeRoot(b.state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][blockRoots] = root[:]
	b.recomputeRoot(int(blockRoots))
	b.lock.Unlock()
	return nil
}

// SetStateRoots for the beacon state. This PR updates the entire
// to a new value by overwriting the previous one.
func (b *BeaconState) SetStateRoots(val [][]byte) error {
	root, err := stateutil.RootsArrayHashTreeRoot(val, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	if err != nil {
		return err
	}
	b.state.StateRoots = val
	b.lock.Lock()
	b.merkleLayers[0][stateRoots] = root[:]
	b.recomputeRoot(int(stateRoots))
	b.lock.Unlock()
	return nil
}

// UpdateStateRootAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateStateRootAtIndex(idx uint64, stateRoot [32]byte) error {
	if len(b.state.StateRoots) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.state.StateRoots[idx] = stateRoot[:]
	root, err := stateutil.RootsArrayHashTreeRoot(b.state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][stateRoots] = root[:]
	b.recomputeRoot(int(stateRoots))
	b.lock.Unlock()
	return nil
}

// SetHistoricalRoots for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetHistoricalRoots(val [][]byte) error {
	root, err := stateutil.HistoricalRootsRoot(val)
	if err != nil {
		return err
	}
	b.state.HistoricalRoots = val
	b.lock.Lock()
	b.merkleLayers[0][historicalRoots] = root[:]
	b.recomputeRoot(int(historicalRoots))
	b.lock.Unlock()
	return nil
}

// SetEth1Data for the beacon state.
func (b *BeaconState) SetEth1Data(val *ethpb.Eth1Data) error {
	root, err := stateutil.Eth1Root(val)
	if err != nil {
		return err
	}
	b.state.Eth1Data = val
	b.lock.Lock()
	b.merkleLayers[0][eth1Data] = root[:]
	b.recomputeRoot(int(eth1Data))
	b.lock.Unlock()
	return nil
}

// SetEth1DataVotes for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetEth1DataVotes(val []*ethpb.Eth1Data) error {
	root, err := stateutil.Eth1DataVotesRoot(val)
	if err != nil {
		return err
	}
	b.state.Eth1DataVotes = val
	b.lock.Lock()
	b.merkleLayers[0][eth1DataVotes] = root[:]
	b.recomputeRoot(int(eth1DataVotes))
	b.lock.Unlock()
	return nil
}

// AppendEth1DataVotes for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendEth1DataVotes(val *ethpb.Eth1Data) error {
	b.state.Eth1DataVotes = append(b.state.Eth1DataVotes, val)
	root, err := stateutil.Eth1DataVotesRoot(b.state.Eth1DataVotes)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][eth1DataVotes] = root[:]
	b.recomputeRoot(int(eth1DataVotes))
	b.lock.Unlock()
	return nil
}

// SetEth1DepositIndex for the beacon state.
func (b *BeaconState) SetEth1DepositIndex(val uint64) error {
	b.state.Eth1DepositIndex = val
	root := stateutil.Uint64Root(val)
	b.lock.Lock()
	b.merkleLayers[0][eth1DepositIndex] = root[:]
	b.recomputeRoot(int(eth1DepositIndex))
	b.lock.Unlock()
	return nil
}

// SetValidators for the beacon state. This PR updates the entire
// to a new value by overwriting the previous one.
func (b *BeaconState) SetValidators(val []*ethpb.Validator) error {
	root, err := stateutil.ValidatorRegistryRoot(val)
	if err != nil {
		return err
	}
	b.state.Validators = val
	b.lock.Lock()
	b.merkleLayers[0][validators] = root[:]
	b.recomputeRoot(int(validators))
	b.lock.Unlock()
	return nil
}

// UpdateValidatorAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateValidatorAtIndex(idx uint64, val *ethpb.Validator) error {
	if len(b.state.Validators) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.state.Validators[idx] = val
	root, err := stateutil.ValidatorRegistryRoot(b.state.Validators)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][validators] = root[:]
	b.recomputeRoot(int(validators))
	b.lock.Unlock()
	return nil
}

// SetBalances for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBalances(val []uint64) error {
	root, err := stateutil.ValidatorBalancesRoot(val)
	if err != nil {
		return err
	}
	b.state.Balances = val
	b.lock.Lock()
	b.merkleLayers[0][balances] = root[:]
	b.recomputeRoot(int(balances))
	b.lock.Unlock()
	return nil
}

// UpdateBalancesAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateBalancesAtIndex(idx uint64, val uint64) error {
	if len(b.state.Balances) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.state.Balances[idx] = val
	root, err := stateutil.ValidatorBalancesRoot(b.state.Balances)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][balances] = root[:]
	b.recomputeRoot(int(balances))
	b.lock.Unlock()
	return nil
}

// SetRandaoMixes for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetRandaoMixes(val [][]byte) error {
	root, err := stateutil.RootsArrayHashTreeRoot(val, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
	if err != nil {
		return err
	}
	b.state.RandaoMixes = val
	b.lock.Lock()
	b.merkleLayers[0][randaoMixes] = root[:]
	b.recomputeRoot(int(randaoMixes))
	b.lock.Unlock()
	return nil
}

// UpdateRandaoMixesAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateRandaoMixesAtIndex(val []byte, idx uint64) error {
	if len(b.state.RandaoMixes) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.state.RandaoMixes[idx] = val
	root, err := stateutil.RootsArrayHashTreeRoot(b.state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][randaoMixes] = root[:]
	b.recomputeRoot(int(randaoMixes))
	b.lock.Unlock()
	return nil
}

// SetSlashings for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetSlashings(val []uint64) error {
	root, err := stateutil.SlashingsRoot(val)
	if err != nil {
		return err
	}
	b.state.Slashings = val
	b.lock.Lock()
	b.merkleLayers[0][slashings] = root[:]
	b.recomputeRoot(int(slashings))
	b.lock.Unlock()
	return nil
}

// UpdateSlashingsAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateSlashingsAtIndex(idx uint64, val uint64) error {
	if len(b.state.Slashings) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.state.Slashings[idx] = val
	root, err := stateutil.SlashingsRoot(b.state.Slashings)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][slashings] = root[:]
	b.recomputeRoot(int(slashings))
	b.lock.Unlock()
	return nil
}

// SetPreviousEpochAttestations for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetPreviousEpochAttestations(val []*pbp2p.PendingAttestation) error {
	root, err := stateutil.EpochAttestationsRoot(val)
	if err != nil {
		return err
	}
	b.state.PreviousEpochAttestations = val
	b.lock.Lock()
	b.merkleLayers[0][previousEpochAttestations] = root[:]
	b.recomputeRoot(int(previousEpochAttestations))
	b.lock.Unlock()
	return nil
}

// SetCurrentEpochAttestations for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetCurrentEpochAttestations(val []*pbp2p.PendingAttestation) error {
	root, err := stateutil.EpochAttestationsRoot(val)
	if err != nil {
		return err
	}
	b.state.CurrentEpochAttestations = val
	b.lock.Lock()
	b.merkleLayers[0][currentEpochAttestations] = root[:]
	b.recomputeRoot(int(currentEpochAttestations))
	b.lock.Unlock()
	return nil
}

// AppendHistoricalRoots for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendHistoricalRoots(root [32]byte) error {
	b.state.HistoricalRoots = append(b.state.HistoricalRoots, root[:])
	root, err := stateutil.HistoricalRootsRoot(b.state.HistoricalRoots)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][historicalRoots] = root[:]
	b.recomputeRoot(int(historicalRoots))
	b.lock.Unlock()
	return nil
}

// AppendCurrentEpochAttestations for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentEpochAttestations(val *pbp2p.PendingAttestation) error {
	b.state.CurrentEpochAttestations = append(b.state.CurrentEpochAttestations, val)
	root, err := stateutil.EpochAttestationsRoot(b.state.CurrentEpochAttestations)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][currentEpochAttestations] = root[:]
	b.recomputeRoot(int(currentEpochAttestations))
	b.lock.Unlock()
	return nil
}

// AppendPreviousEpochAttestations for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousEpochAttestations(val *pbp2p.PendingAttestation) error {
	b.state.PreviousEpochAttestations = append(b.state.PreviousEpochAttestations, val)
	root, err := stateutil.EpochAttestationsRoot(b.state.PreviousEpochAttestations)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][previousEpochAttestations] = root[:]
	b.recomputeRoot(int(previousEpochAttestations))
	b.lock.Unlock()
	return nil
}

// AppendValidator for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendValidator(val *ethpb.Validator) error {
	b.state.Validators = append(b.state.Validators, val)
	root, err := stateutil.ValidatorRegistryRoot(b.state.Validators)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][validators] = root[:]
	b.recomputeRoot(int(validators))
	b.lock.Unlock()
	return nil
}

// AppendBalance for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendBalance(bal uint64) error {
	b.state.Balances = append(b.state.Balances, bal)
	root, err := stateutil.ValidatorBalancesRoot(b.state.Balances)
	if err != nil {
		return err
	}
	b.lock.Lock()
	b.merkleLayers[0][balances] = root[:]
	b.recomputeRoot(int(balances))
	b.lock.Unlock()
	return nil
}

// SetJustificationBits for the beacon state.
func (b *BeaconState) SetJustificationBits(val bitfield.Bitvector4) error {
	root := bytesutil.ToBytes32(b.state.JustificationBits)
	b.state.JustificationBits = val
	b.lock.Lock()
	b.merkleLayers[0][justificationBits] = root[:]
	b.recomputeRoot(int(justificationBits))
	b.lock.Unlock()
	return nil
}

// SetPreviousJustifiedCheckpoint for the beacon state.
func (b *BeaconState) SetPreviousJustifiedCheckpoint(val *ethpb.Checkpoint) error {
	root, err := stateutil.CheckpointRoot(val)
	if err != nil {
		return err
	}
	b.state.PreviousJustifiedCheckpoint = val
	b.lock.Lock()
	b.merkleLayers[0][previousJustifiedCheckpoint] = root[:]
	b.recomputeRoot(int(previousJustifiedCheckpoint))
	b.lock.Unlock()
	return nil
}

// SetCurrentJustifiedCheckpoint for the beacon state.
func (b *BeaconState) SetCurrentJustifiedCheckpoint(val *ethpb.Checkpoint) error {
	root, err := stateutil.CheckpointRoot(val)
	if err != nil {
		return err
	}
	b.state.CurrentJustifiedCheckpoint = val
	b.lock.Lock()
	b.merkleLayers[0][currentJustifiedCheckpoint] = root[:]
	b.recomputeRoot(int(currentJustifiedCheckpoint))
	b.lock.Unlock()
	return nil
}

// SetFinalizedCheckpoint for the beacon state.
func (b *BeaconState) SetFinalizedCheckpoint(val *ethpb.Checkpoint) error {
	root, err := stateutil.CheckpointRoot(val)
	if err != nil {
		return err
	}
	b.state.FinalizedCheckpoint = val
	b.lock.Lock()
	b.merkleLayers[0][finalizedCheckpoint] = root[:]
	b.recomputeRoot(int(finalizedCheckpoint))
	b.lock.Unlock()
	return nil
}

// Recomputes the branch up the index in the Merkle trie representation
// of the beacon state. This method performs map reads and the caller MUST
// hold the lock before calling this method.
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
