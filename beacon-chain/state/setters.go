package state

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.GenesisTime = val
	b.markFieldAsDirty(genesisTime)
	return nil
}

// SetSlot for the beacon state.
func (b *BeaconState) SetSlot(val uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Slot = val
	b.markFieldAsDirty(slot)
	return nil
}

// SetFork version for the beacon chain.
func (b *BeaconState) SetFork(val *pbp2p.Fork) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Fork = proto.Clone(val).(*pbp2p.Fork)
	b.markFieldAsDirty(fork)
	return nil
}

// SetLatestBlockHeader in the beacon state.
func (b *BeaconState) SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.LatestBlockHeader = CopyBeaconBlockHeader(val)
	b.markFieldAsDirty(latestBlockHeader)
	return nil
}

// SetBlockRoots for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBlockRoots(val [][]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[blockRoots].refs--
	b.sharedFieldReferences[blockRoots] = &reference{refs: 1}

	b.state.BlockRoots = val
	b.markFieldAsDirty(blockRoots)
	return nil
}

// UpdateBlockRootAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if len(b.state.BlockRoots) <= int(idx) {
		return fmt.Errorf("invalid index provided %d", idx)
	}

	b.lock.RLock()
	r := b.state.BlockRoots
	if ref := b.sharedFieldReferences[blockRoots]; ref.refs > 1 {
		// Copy on write since this is a shared array.
		r = b.BlockRoots()

		ref.refs--
		b.sharedFieldReferences[blockRoots] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	// Must secure lock after copy or hit a deadlock.
	b.lock.Lock()
	defer b.lock.Unlock()

	r[idx] = blockRoot[:]
	b.state.BlockRoots = r

	b.markFieldAsDirty(blockRoots)
	return nil
}

// SetStateRoots for the beacon state. This PR updates the entire
// to a new value by overwriting the previous one.
func (b *BeaconState) SetStateRoots(val [][]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[stateRoots].refs--
	b.sharedFieldReferences[stateRoots] = &reference{refs: 1}

	b.state.StateRoots = val
	b.markFieldAsDirty(stateRoots)
	return nil
}

// UpdateStateRootAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateStateRootAtIndex(idx uint64, stateRoot [32]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if len(b.state.StateRoots) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}

	b.lock.RLock()
	// Check if we hold the only reference to the shared state roots slice.
	r := b.state.StateRoots
	if ref := b.sharedFieldReferences[stateRoots]; ref.refs > 1 {
		// Perform a copy since this is a shared reference and we don't want to mutate others.
		r = b.StateRoots()

		ref.refs--
		b.sharedFieldReferences[stateRoots] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	// Must secure lock after copy or hit a deadlock.
	b.lock.Lock()
	defer b.lock.Unlock()

	r[idx] = stateRoot[:]
	b.state.StateRoots = r

	b.markFieldAsDirty(stateRoots)
	return nil
}

// SetHistoricalRoots for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetHistoricalRoots(val [][]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[historicalRoots].refs--
	b.sharedFieldReferences[historicalRoots] = &reference{refs: 1}

	b.state.HistoricalRoots = val
	b.markFieldAsDirty(historicalRoots)
	return nil
}

// SetEth1Data for the beacon state.
func (b *BeaconState) SetEth1Data(val *ethpb.Eth1Data) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Eth1Data = val
	b.markFieldAsDirty(eth1Data)
	return nil
}

// SetEth1DataVotes for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetEth1DataVotes(val []*ethpb.Eth1Data) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[eth1DataVotes].refs--
	b.sharedFieldReferences[eth1DataVotes] = &reference{refs: 1}

	b.state.Eth1DataVotes = val
	b.markFieldAsDirty(eth1DataVotes)
	return nil
}

// AppendEth1DataVotes for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendEth1DataVotes(val *ethpb.Eth1Data) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()
	votes := b.state.Eth1DataVotes
	if b.sharedFieldReferences[eth1DataVotes].refs > 1 {
		votes = b.Eth1DataVotes()
		b.sharedFieldReferences[eth1DataVotes].refs--
		b.sharedFieldReferences[eth1DataVotes] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Eth1DataVotes = append(votes, val)
	b.markFieldAsDirty(eth1DataVotes)
	return nil
}

// SetEth1DepositIndex for the beacon state.
func (b *BeaconState) SetEth1DepositIndex(val uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Eth1DepositIndex = val
	b.markFieldAsDirty(eth1DepositIndex)
	return nil
}

// SetValidators for the beacon state. This PR updates the entire
// to a new value by overwriting the previous one.
func (b *BeaconState) SetValidators(val []*ethpb.Validator) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Validators = val
	b.sharedFieldReferences[validators].refs--
	b.sharedFieldReferences[validators] = &reference{refs: 1}
	b.markFieldAsDirty(validators)
	return nil
}

// ApplyToEveryValidator applies the provided callback function to each validator in the
// validator registry.
func (b *BeaconState) ApplyToEveryValidator(f func(idx int, val *ethpb.Validator) error) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()
	v := b.state.Validators
	if ref := b.sharedFieldReferences[validators]; ref.refs > 1 {
		// Perform a copy since this is a shared reference and we don't want to mutate others.
		v = b.Validators()

		ref.refs--
		b.sharedFieldReferences[validators] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	for i, val := range v {
		err := f(i, val)
		if err != nil {
			return err
		}
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Validators = v
	b.markFieldAsDirty(validators)
	return nil
}

// UpdateValidatorAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateValidatorAtIndex(idx uint64, val *ethpb.Validator) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if len(b.state.Validators) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}

	b.lock.RLock()
	v := b.state.Validators
	if ref := b.sharedFieldReferences[validators]; ref.refs > 1 {
		// Perform a copy since this is a shared reference and we don't want to mutate others.
		v = b.Validators()

		ref.refs--
		b.sharedFieldReferences[validators] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	v[idx] = val
	b.state.Validators = v
	b.markFieldAsDirty(validators)
	return nil
}

// SetValidatorIndexByPubkey updates the validator index mapping maintained internally to
// a given input 48-byte, public key.
func (b *BeaconState) SetValidatorIndexByPubkey(pubKey [48]byte, validatorIdx uint64) {
	// Copy on write since this is a shared map.
	m := b.validatorIndexMap()

	b.lock.Lock()
	defer b.lock.Unlock()

	m[pubKey] = validatorIdx
	b.valIdxMap = m
}

// SetBalances for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBalances(val []uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[balances].refs--
	b.sharedFieldReferences[balances] = &reference{refs: 1}

	b.state.Balances = val
	b.markFieldAsDirty(balances)
	return nil
}

// UpdateBalancesAtIndex for the beacon state. This method updates the balance
// at a specific index to a new value.
func (b *BeaconState) UpdateBalancesAtIndex(idx uint64, val uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if len(b.state.Balances) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}

	b.lock.RLock()
	bals := b.state.Balances
	if b.sharedFieldReferences[balances].refs > 1 {
		bals = b.Balances()
		b.sharedFieldReferences[balances].refs--
		b.sharedFieldReferences[balances] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	bals[idx] = val
	b.state.Balances = bals
	b.markFieldAsDirty(balances)
	return nil
}

// SetRandaoMixes for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetRandaoMixes(val [][]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[randaoMixes].refs--
	b.sharedFieldReferences[randaoMixes] = &reference{refs: 1}

	b.state.RandaoMixes = val
	b.markFieldAsDirty(randaoMixes)
	return nil
}

// UpdateRandaoMixesAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateRandaoMixesAtIndex(val []byte, idx uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if len(b.state.RandaoMixes) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}

	b.lock.RLock()
	mixes := b.state.RandaoMixes
	if refs := b.sharedFieldReferences[randaoMixes].refs; refs > 1 {
		mixes = b.RandaoMixes()
		b.sharedFieldReferences[randaoMixes].refs--
		b.sharedFieldReferences[randaoMixes] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	mixes[idx] = val
	b.state.RandaoMixes = mixes
	b.markFieldAsDirty(randaoMixes)
	return nil
}

// SetSlashings for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetSlashings(val []uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[slashings].refs--
	b.sharedFieldReferences[slashings] = &reference{refs: 1}

	b.state.Slashings = val
	b.markFieldAsDirty(slashings)
	return nil
}

// UpdateSlashingsAtIndex for the beacon state. This PR updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateSlashingsAtIndex(idx uint64, val uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if len(b.state.Slashings) <= int(idx) {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.RLock()
	s := b.state.Slashings

	if b.sharedFieldReferences[slashings].refs > 1 {
		s = b.Slashings()
		b.sharedFieldReferences[slashings].refs--
		b.sharedFieldReferences[slashings] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	s[idx] = val

	b.state.Slashings = s

	b.markFieldAsDirty(slashings)
	return nil
}

// SetPreviousEpochAttestations for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetPreviousEpochAttestations(val []*pbp2p.PendingAttestation) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[previousEpochAttestations].refs--
	b.sharedFieldReferences[previousEpochAttestations] = &reference{refs: 1}

	b.state.PreviousEpochAttestations = val
	b.markFieldAsDirty(previousEpochAttestations)
	return nil
}

// SetCurrentEpochAttestations for the beacon state. This PR updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetCurrentEpochAttestations(val []*pbp2p.PendingAttestation) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[currentEpochAttestations].refs--
	b.sharedFieldReferences[currentEpochAttestations] = &reference{refs: 1}

	b.state.CurrentEpochAttestations = val
	b.markFieldAsDirty(currentEpochAttestations)
	return nil
}

// AppendHistoricalRoots for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendHistoricalRoots(root [32]byte) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()
	roots := b.state.HistoricalRoots
	if b.sharedFieldReferences[historicalRoots].refs > 1 {
		roots = b.HistoricalRoots()
		b.sharedFieldReferences[historicalRoots].refs--
		b.sharedFieldReferences[historicalRoots] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.HistoricalRoots = append(roots, root[:])
	b.markFieldAsDirty(historicalRoots)
	return nil
}

// AppendCurrentEpochAttestations for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentEpochAttestations(val *pbp2p.PendingAttestation) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()

	atts := b.state.CurrentEpochAttestations
	if b.sharedFieldReferences[currentEpochAttestations].refs > 1 {
		atts = b.CurrentEpochAttestations()
		b.sharedFieldReferences[currentEpochAttestations].refs--
		b.sharedFieldReferences[currentEpochAttestations] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.CurrentEpochAttestations = append(atts, val)
	b.markFieldAsDirty(currentEpochAttestations)
	return nil
}

// AppendPreviousEpochAttestations for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousEpochAttestations(val *pbp2p.PendingAttestation) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()
	atts := b.state.PreviousEpochAttestations
	if b.sharedFieldReferences[previousEpochAttestations].refs > 1 {
		atts = b.PreviousEpochAttestations()
		b.sharedFieldReferences[previousEpochAttestations].refs--
		b.sharedFieldReferences[previousEpochAttestations] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.PreviousEpochAttestations = append(atts, val)
	b.markFieldAsDirty(previousEpochAttestations)
	return nil
}

// AppendValidator for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendValidator(val *ethpb.Validator) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()
	vals := b.state.Validators
	if b.sharedFieldReferences[validators].refs > 1 {
		vals = b.Validators()
		b.sharedFieldReferences[validators].refs--
		b.sharedFieldReferences[validators] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Validators = append(vals, val)
	b.markFieldAsDirty(validators)
	return nil
}

// AppendBalance for the beacon state. This PR appends the new value
// to the the end of list.
func (b *BeaconState) AppendBalance(bal uint64) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.RLock()

	bals := b.state.Balances
	if b.sharedFieldReferences[balances].refs > 1 {
		bals = b.Balances()
		b.sharedFieldReferences[balances].refs--
		b.sharedFieldReferences[balances] = &reference{refs: 1}
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Balances = append(bals, bal)
	b.markFieldAsDirty(balances)
	return nil
}

// SetJustificationBits for the beacon state.
func (b *BeaconState) SetJustificationBits(val bitfield.Bitvector4) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.JustificationBits = val
	b.markFieldAsDirty(justificationBits)
	return nil
}

// SetPreviousJustifiedCheckpoint for the beacon state.
func (b *BeaconState) SetPreviousJustifiedCheckpoint(val *ethpb.Checkpoint) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.PreviousJustifiedCheckpoint = val
	b.markFieldAsDirty(previousJustifiedCheckpoint)
	return nil
}

// SetCurrentJustifiedCheckpoint for the beacon state.
func (b *BeaconState) SetCurrentJustifiedCheckpoint(val *ethpb.Checkpoint) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.CurrentJustifiedCheckpoint = val
	b.markFieldAsDirty(currentJustifiedCheckpoint)
	return nil
}

// SetFinalizedCheckpoint for the beacon state.
func (b *BeaconState) SetFinalizedCheckpoint(val *ethpb.Checkpoint) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.FinalizedCheckpoint = val
	b.markFieldAsDirty(finalizedCheckpoint)
	return nil
}

// Recomputes the branch up the index in the Merkle trie representation
// of the beacon state. This method performs map reads and the caller MUST
// hold the lock before calling this method.
func (b *BeaconState) recomputeRoot(idx int) {
	hashFunc := hashutil.CustomSHA256Hasher()
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
			parentHash := hashFunc(append(root, neighbor...))
			root = parentHash[:]
		} else {
			parentHash := hashFunc(append(neighbor, root...))
			root = parentHash[:]
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		layers[i+1][parentIdx] = root
		currentIndex = parentIdx
	}
	b.merkleLayers = layers
}

func (b *BeaconState) markFieldAsDirty(field fieldIndex) {
	_, ok := b.dirtyFields[field]
	if !ok {
		b.dirtyFields[field] = true
	}
	// do nothing if field already exists
}
