package stateV0

import (
	"bytes"
	"errors"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// InnerStateUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) InnerStateUnsafe() interface{} {
	if b == nil {
		return nil
	}
	return b.state
}

// CloneInnerState the beacon state into a protobuf for usage.
func (b *BeaconState) CloneInnerState() interface{} {
	if b == nil || b.state == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	return &pbp2p.BeaconState{
		GenesisTime:                 b.genesisTime(),
		GenesisValidatorsRoot:       b.genesisValidatorRoot(),
		Slot:                        b.slot(),
		Fork:                        b.fork(),
		LatestBlockHeader:           b.latestBlockHeader(),
		BlockRoots:                  b.blockRoots(),
		StateRoots:                  b.stateRoots(),
		HistoricalRoots:             b.historicalRoots(),
		Eth1Data:                    b.eth1Data(),
		Eth1DataVotes:               b.eth1DataVotes(),
		Eth1DepositIndex:            b.eth1DepositIndex(),
		Validators:                  b.validators(),
		Balances:                    b.balances(),
		RandaoMixes:                 b.randaoMixes(),
		Slashings:                   b.slashings(),
		PreviousEpochAttestations:   b.previousEpochAttestations(),
		CurrentEpochAttestations:    b.currentEpochAttestations(),
		JustificationBits:           b.justificationBits(),
		PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint(),
		FinalizedCheckpoint:         b.finalizedCheckpoint(),
	}
}

// hasInnerState detects if the internal reference to the state data structure
// is populated correctly. Returns false if nil.
func (b *BeaconState) hasInnerState() bool {
	return b != nil && b.state != nil
}

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisTime()
}

// genesisTime of the beacon state as a uint64.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisTime() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	return b.state.GenesisTime
}

// GenesisValidatorRoot of the beacon state.
func (b *BeaconState) GenesisValidatorRoot() []byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.GenesisValidatorsRoot == nil {
		return params.BeaconConfig().ZeroHash[:]
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorRoot()
}

// genesisValidatorRoot of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisValidatorRoot() []byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.GenesisValidatorsRoot == nil {
		return params.BeaconConfig().ZeroHash[:]
	}

	root := make([]byte, 32)
	copy(root, b.state.GenesisValidatorsRoot)
	return root
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slot()
}

// slot of the current beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slot() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	return b.state.Slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *pbp2p.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.fork()
}

// fork version of the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) fork() *pbp2p.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Fork == nil {
		return nil
	}

	prevVersion := make([]byte, len(b.state.Fork.PreviousVersion))
	copy(prevVersion, b.state.Fork.PreviousVersion)
	currVersion := make([]byte, len(b.state.Fork.CurrentVersion))
	copy(currVersion, b.state.Fork.CurrentVersion)
	return &pbp2p.Fork{
		PreviousVersion: prevVersion,
		CurrentVersion:  currVersion,
		Epoch:           b.state.Fork.Epoch,
	}
}

// LatestBlockHeader stored within the beacon state.
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.LatestBlockHeader == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestBlockHeader()
}

// latestBlockHeader stored within the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestBlockHeader() *ethpb.BeaconBlockHeader {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.LatestBlockHeader == nil {
		return nil
	}

	hdr := &ethpb.BeaconBlockHeader{
		Slot:          b.state.LatestBlockHeader.Slot,
		ProposerIndex: b.state.LatestBlockHeader.ProposerIndex,
	}

	parentRoot := make([]byte, len(b.state.LatestBlockHeader.ParentRoot))
	bodyRoot := make([]byte, len(b.state.LatestBlockHeader.BodyRoot))
	stateRoot := make([]byte, len(b.state.LatestBlockHeader.StateRoot))

	copy(parentRoot, b.state.LatestBlockHeader.ParentRoot)
	copy(bodyRoot, b.state.LatestBlockHeader.BodyRoot)
	copy(stateRoot, b.state.LatestBlockHeader.StateRoot)
	hdr.ParentRoot = parentRoot
	hdr.BodyRoot = bodyRoot
	hdr.StateRoot = stateRoot
	return hdr
}

// BlockRoots kept track of in the beacon state.
func (b *BeaconState) BlockRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.BlockRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.blockRoots()
}

// blockRoots kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) blockRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	return b.safeCopy2DByteSlice(b.state.BlockRoots)
}

// BlockRootAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) BlockRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.BlockRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.blockRootAtIndex(idx)
}

// blockRootAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) blockRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	return b.safeCopyBytesAtIndex(b.state.BlockRoots, idx)
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.StateRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.stateRoots()
}

// StateRoots kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	return b.safeCopy2DByteSlice(b.state.StateRoots)
}

// StateRootAtIndex retrieves a specific state root based on an
// input index value.
func (b *BeaconState) StateRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.StateRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.stateRootAtIndex(idx)
}

// stateRootAtIndex retrieves a specific state root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	return b.safeCopyBytesAtIndex(b.state.StateRoots, idx)
}

// HistoricalRoots based on epochs stored in the beacon state.
func (b *BeaconState) HistoricalRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.HistoricalRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.historicalRoots()
}

// historicalRoots based on epochs stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) historicalRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	return b.safeCopy2DByteSlice(b.state.HistoricalRoots)
}

// Eth1Data corresponding to the proof-of-work chain information stored in the beacon state.
func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Eth1Data == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.eth1Data()
}

// eth1Data corresponding to the proof-of-work chain information stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) eth1Data() *ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Eth1Data == nil {
		return nil
	}

	return CopyETH1Data(b.state.Eth1Data)
}

// Eth1DataVotes corresponds to votes from eth2 on the canonical proof-of-work chain
// data retrieved from eth1.
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Eth1DataVotes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.eth1DataVotes()
}

// eth1DataVotes corresponds to votes from eth2 on the canonical proof-of-work chain
// data retrieved from eth1.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) eth1DataVotes() []*ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Eth1DataVotes == nil {
		return nil
	}

	res := make([]*ethpb.Eth1Data, len(b.state.Eth1DataVotes))
	for i := 0; i < len(res); i++ {
		res[i] = CopyETH1Data(b.state.Eth1DataVotes[i])
	}
	return res
}

// Eth1DepositIndex corresponds to the index of the deposit made to the
// validator deposit contract at the time of this state's eth1 data.
func (b *BeaconState) Eth1DepositIndex() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.eth1DepositIndex()
}

// eth1DepositIndex corresponds to the index of the deposit made to the
// validator deposit contract at the time of this state's eth1 data.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) eth1DepositIndex() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	return b.state.Eth1DepositIndex
}

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validators()
}

// validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) validators() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		if val == nil {
			continue
		}
		res[i] = CopyValidator(val)
	}
	return res
}

// references of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState. This does not
// copy fully and instead just copies the reference.
func (b *BeaconState) validatorsReferences() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		validator := b.state.Validators[i]
		if validator == nil {
			continue
		}
		// copy validator reference instead.
		res[i] = validator
	}
	return res
}

// ValidatorAtIndex is the validator at the provided index.
func (b *BeaconState) ValidatorAtIndex(idx types.ValidatorIndex) (*ethpb.Validator, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return &ethpb.Validator{}, nil
	}
	if uint64(len(b.state.Validators)) <= uint64(idx) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	val := b.state.Validators[idx]
	return CopyValidator(val), nil
}

// ValidatorAtIndexReadOnly is the validator at the provided index. This method
// doesn't clone the validator.
func (b *BeaconState) ValidatorAtIndexReadOnly(idx types.ValidatorIndex) (iface.ReadOnlyValidator, error) {
	if !b.hasInnerState() {
		return ReadOnlyValidator{}, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return ReadOnlyValidator{}, nil
	}
	if uint64(len(b.state.Validators)) <= uint64(idx) {
		return ReadOnlyValidator{}, fmt.Errorf("index %d out of range", idx)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return ReadOnlyValidator{b.state.Validators[idx]}, nil
}

// ValidatorIndexByPubkey returns a given validator by its 48-byte public key.
func (b *BeaconState) ValidatorIndexByPubkey(key [48]byte) (types.ValidatorIndex, bool) {
	if b == nil || b.valMapHandler == nil || b.valMapHandler.IsNil() {
		return 0, false
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	idx, ok := b.valMapHandler.Get(key)
	return idx, ok
}

func (b *BeaconState) validatorIndexMap() map[[48]byte]types.ValidatorIndex {
	if b == nil || b.valMapHandler == nil || b.valMapHandler.IsNil() {
		return map[[48]byte]types.ValidatorIndex{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.valMapHandler.Copy().ValidatorIndexMap()
}

// PubkeyAtIndex returns the pubkey at the given
// validator index.
func (b *BeaconState) PubkeyAtIndex(idx types.ValidatorIndex) [48]byte {
	if !b.hasInnerState() {
		return [48]byte{}
	}
	if uint64(idx) >= uint64(len(b.state.Validators)) {
		return [48]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.Validators[idx] == nil {
		return [48]byte{}
	}
	return bytesutil.ToBytes48(b.state.Validators[idx].PublicKey)
}

// NumValidators returns the size of the validator registry.
func (b *BeaconState) NumValidators() int {
	if !b.hasInnerState() {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.state.Validators)
}

// ReadFromEveryValidator reads values from every validator and applies it to the provided function.
// Warning: This method is potentially unsafe, as it exposes the actual validator registry.
func (b *BeaconState) ReadFromEveryValidator(f func(idx int, val iface.ReadOnlyValidator) error) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	if b.state.Validators == nil {
		return errors.New("nil validators in state")
	}
	b.lock.RLock()
	validators := b.state.Validators
	b.lock.RUnlock()

	for i, v := range validators {
		err := f(i, ReadOnlyValidator{validator: v})
		if err != nil {
			return err
		}
	}
	return nil
}

// Balances of validators participating in consensus on the beacon chain.
func (b *BeaconState) Balances() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balances()
}

// balances of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balances() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}

	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// BalanceAtIndex of validator with the provided index.
func (b *BeaconState) BalanceAtIndex(idx types.ValidatorIndex) (uint64, error) {
	if !b.hasInnerState() {
		return 0, ErrNilInnerState
	}
	if b.state.Balances == nil {
		return 0, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if uint64(len(b.state.Balances)) <= uint64(idx) {
		return 0, fmt.Errorf("index of %d does not exist", idx)
	}
	return b.state.Balances[idx], nil
}

// BalancesLength returns the length of the balances slice.
func (b *BeaconState) BalancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.Balances == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balancesLength()
}

// balancesLength returns the length of the balances slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.Balances == nil {
		return 0
	}

	return len(b.state.Balances)
}

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.RandaoMixes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixes()
}

// randaoMixes of block proposers on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixes() [][]byte {
	if !b.hasInnerState() {
		return nil
	}

	return b.safeCopy2DByteSlice(b.state.RandaoMixes)
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.RandaoMixes == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixAtIndex(idx)
}

// randaoMixAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}

	return b.safeCopyBytesAtIndex(b.state.RandaoMixes, idx)
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.RandaoMixes == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixesLength()
}

// randaoMixesLength returns the length of the randao mixes slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.RandaoMixes == nil {
		return 0
	}

	return len(b.state.RandaoMixes)
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slashings()
}

// slashings of validators on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slashings() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

// PreviousEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.PreviousEpochAttestations == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochAttestations()
}

// previousEpochAttestations corresponding to blocks on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochAttestations() []*pbp2p.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}

	return b.safeCopyPendingAttestationSlice(b.state.PreviousEpochAttestations)
}

// CurrentEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.CurrentEpochAttestations == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochAttestations()
}

// currentEpochAttestations corresponding to blocks on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochAttestations() []*pbp2p.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}

	return b.safeCopyPendingAttestationSlice(b.state.CurrentEpochAttestations)
}

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.JustificationBits == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.justificationBits()
}

// justificationBits marking which epochs have been justified in the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) justificationBits() bitfield.Bitvector4 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.JustificationBits == nil {
		return nil
	}

	res := make([]byte, len(b.state.JustificationBits.Bytes()))
	copy(res, b.state.JustificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.PreviousJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousJustifiedCheckpoint()
}

// previousJustifiedCheckpoint denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}

	return b.safeCopyCheckpoint(b.state.PreviousJustifiedCheckpoint)
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.CurrentJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentJustifiedCheckpoint()
}

// currentJustifiedCheckpoint denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}

	return b.safeCopyCheckpoint(b.state.CurrentJustifiedCheckpoint)
}

// MatchCurrentJustifiedCheckpoint returns true if input justified checkpoint matches
// the current justified checkpoint in state.
func (b *BeaconState) MatchCurrentJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if !b.hasInnerState() {
		return false
	}
	if b.state.CurrentJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.state.CurrentJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.state.CurrentJustifiedCheckpoint.Root)
}

// MatchPreviousJustifiedCheckpoint returns true if the input justified checkpoint matches
// the previous justified checkpoint in state.
func (b *BeaconState) MatchPreviousJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if !b.hasInnerState() {
		return false
	}
	if b.state.PreviousJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.state.PreviousJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.state.PreviousJustifiedCheckpoint.Root)
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.FinalizedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.finalizedCheckpoint()
}

// finalizedCheckpoint denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) finalizedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}

	return b.safeCopyCheckpoint(b.state.FinalizedCheckpoint)
}

// FinalizedCheckpointEpoch returns the epoch value of the finalized checkpoint.
func (b *BeaconState) FinalizedCheckpointEpoch() types.Epoch {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.FinalizedCheckpoint == nil {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.state.FinalizedCheckpoint.Epoch
}

func (b *BeaconState) safeCopy2DByteSlice(input [][]byte) [][]byte {
	if input == nil {
		return nil
	}

	dst := make([][]byte, len(input))
	for i, r := range input {
		tmp := make([]byte, len(r))
		copy(tmp, r)
		dst[i] = tmp
	}
	return dst
}

func (b *BeaconState) safeCopyBytesAtIndex(input [][]byte, idx uint64) ([]byte, error) {
	if input == nil {
		return nil, nil
	}

	if uint64(len(input)) <= idx {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	root := make([]byte, 32)
	copy(root, input[idx])
	return root, nil
}

func (b *BeaconState) safeCopyPendingAttestationSlice(input []*pbp2p.PendingAttestation) []*pbp2p.PendingAttestation {
	if input == nil {
		return nil
	}

	res := make([]*pbp2p.PendingAttestation, len(input))
	for i := 0; i < len(res); i++ {
		res[i] = CopyPendingAttestation(input[i])
	}
	return res
}

func (b *BeaconState) safeCopyCheckpoint(input *ethpb.Checkpoint) *ethpb.Checkpoint {
	if input == nil {
		return nil
	}

	return CopyCheckpoint(input)
}

// MarshalSSZ marshals the underlying beacon state to bytes.
func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, errors.New("nil beacon state")
	}
	return b.state.MarshalSSZ()
}

// ProtobufBeaconState transforms an input into beacon state in the form of protobuf.
// Error is returned if the input is not type protobuf beacon state.
func ProtobufBeaconState(s interface{}) (*pbp2p.BeaconState, error) {
	pbState, ok := s.(*pbp2p.BeaconState)
	if !ok {
		return nil, errors.New("input is not type pb.BeaconState")
	}
	return pbState, nil
}
