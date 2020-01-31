package state

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// EffectiveBalance returns the effective balance of the
// read only validator.
func (v *ReadOnlyValidator) EffectiveBalance() uint64 {
	return v.validator.EffectiveBalance
}

// ActivationEligibilityEpoch returns the activation eligibility epoch of the
// read only validator.
func (v *ReadOnlyValidator) ActivationEligibilityEpoch() uint64 {
	return v.validator.ActivationEligibilityEpoch
}

// ActivationEpoch returns the activation epoch of the
// read only validator.
func (v *ReadOnlyValidator) ActivationEpoch() uint64 {
	return v.validator.ActivationEpoch
}

// WithdrawableEpoch returns the withdrawable epoch of the
// read only validator.
func (v *ReadOnlyValidator) WithdrawableEpoch() uint64 {
	return v.validator.WithdrawableEpoch
}

// ExitEpoch returns the exit epoch of the
// read only validator.
func (v *ReadOnlyValidator) ExitEpoch() uint64 {
	return v.validator.ExitEpoch
}

// PublicKey returns the public key of the
// read only validator.
func (v *ReadOnlyValidator) PublicKey() [48]byte {
	var pubkey [48]byte
	copy(pubkey[:], v.validator.PublicKey)
	return pubkey
}

// WithdrawalCredentials returns the withdrawal credentials of the
// read only validator.
func (v *ReadOnlyValidator) WithdrawalCredentials() []byte {
	creds := make([]byte, len(v.validator.WithdrawalCredentials))
	copy(creds[:], v.validator.WithdrawalCredentials)
	return creds
}

// Slashed returns the read only validator is slashed.
func (v *ReadOnlyValidator) Slashed() bool {
	return v.validator.Slashed
}

// InnerStateUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) InnerStateUnsafe() *pbp2p.BeaconState {
	if b.state == nil {
		return &pbp2p.BeaconState{}
	}
	return b.state
}

// CloneInnerState the beacon state into a protobuf for usage.
func (b *BeaconState) CloneInnerState() *pbp2p.BeaconState {
	if b.state == nil {
		return nil
	}
	return &pbp2p.BeaconState{
		GenesisTime:                 b.GenesisTime(),
		Slot:                        b.Slot(),
		Fork:                        b.Fork(),
		LatestBlockHeader:           b.LatestBlockHeader(),
		BlockRoots:                  b.BlockRoots(),
		StateRoots:                  b.StateRoots(),
		HistoricalRoots:             b.HistoricalRoots(),
		Eth1Data:                    b.Eth1Data(),
		Eth1DataVotes:               b.Eth1DataVotes(),
		Eth1DepositIndex:            b.Eth1DepositIndex(),
		Validators:                  b.Validators(),
		Balances:                    b.Balances(),
		RandaoMixes:                 b.RandaoMixes(),
		Slashings:                   b.Slashings(),
		PreviousEpochAttestations:   b.PreviousEpochAttestations(),
		CurrentEpochAttestations:    b.CurrentEpochAttestations(),
		JustificationBits:           b.JustificationBits(),
		PreviousJustifiedCheckpoint: b.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  b.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:         b.FinalizedCheckpoint(),
	}
}

// HasInnerState detects if the internal reference to the state data structure
// is populated correctly. Returns false if nil.
func (b *BeaconState) HasInnerState() bool {
	return b.state != nil
}

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	return b.state.GenesisTime
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() uint64 {
	return b.state.Slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *pbp2p.Fork {
	if b.state.Fork == nil {
		return nil
	}
	prevVersion := make([]byte, len(b.state.Fork.PreviousVersion))
	copy(prevVersion, b.state.Fork.PreviousVersion)
	currVersion := make([]byte, len(b.state.Fork.PreviousVersion))
	copy(currVersion, b.state.Fork.PreviousVersion)
	return &pbp2p.Fork{
		PreviousVersion: prevVersion,
		CurrentVersion:  currVersion,
		Epoch:           b.state.Fork.Epoch,
	}
}

// LatestBlockHeader stored within the beacon state.
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	if b.state.LatestBlockHeader == nil {
		return nil
	}
	hdr := &ethpb.BeaconBlockHeader{
		Slot: b.state.LatestBlockHeader.Slot,
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
	if b.state.BlockRoots == nil {
		return nil
	}
	roots := make([][]byte, len(b.state.BlockRoots))
	for i, r := range b.state.BlockRoots {
		tmpRt := make([]byte, len(r))
		copy(tmpRt, r)
		roots[i] = tmpRt
	}
	return roots
}

// BlockRootAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) BlockRootAtIndex(idx uint64) ([]byte, error) {
	if b.state.BlockRoots == nil {
		return nil, nil
	}
	if len(b.state.BlockRoots) <= int(idx) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	root := make([]byte, 32)
	copy(root, b.state.BlockRoots[idx])
	return root, nil
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() [][]byte {
	if b.state.StateRoots == nil {
		return nil
	}
	roots := make([][]byte, len(b.state.StateRoots))
	for i, r := range b.state.StateRoots {
		tmpRt := make([]byte, len(r))
		copy(tmpRt, r)
		roots[i] = tmpRt
	}
	return roots
}

// HistoricalRoots based on epochs stored in the beacon state.
func (b *BeaconState) HistoricalRoots() [][]byte {
	if b.state.HistoricalRoots == nil {
		return nil
	}
	roots := make([][]byte, len(b.state.HistoricalRoots))
	for i, r := range b.state.HistoricalRoots {
		tmpRt := make([]byte, len(r))
		copy(tmpRt, r)
		roots[i] = tmpRt
	}
	return roots
}

// Eth1Data corresponding to the proof-of-work chain information stored in the beacon state.
func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	if b.state.Eth1Data == nil {
		return nil
	}
	return CopyETH1Data(b.state.Eth1Data)
}

// Eth1DataVotes corresponds to votes from eth2 on the canonical proof-of-work chain
// data retrieved from eth1.
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
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
	return b.state.Eth1DepositIndex
}

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	if b.state.Validators == nil {
		return nil
	}
	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		if val == nil {
			continue
		}
		pubKey := make([]byte, len(val.PublicKey))
		copy(pubKey, val.PublicKey)
		withdrawalCreds := make([]byte, len(val.WithdrawalCredentials))
		copy(withdrawalCreds, val.WithdrawalCredentials)
		res[i] = &ethpb.Validator{
			PublicKey:                  pubKey[:],
			WithdrawalCredentials:      withdrawalCreds,
			EffectiveBalance:           val.EffectiveBalance,
			Slashed:                    val.Slashed,
			ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
			ActivationEpoch:            val.ActivationEpoch,
			ExitEpoch:                  val.ExitEpoch,
			WithdrawableEpoch:          val.WithdrawableEpoch,
		}
	}
	return res
}

// ValidatorsReadOnly returns validators participating in consensus on the beacon chain. This
// method doesn't clone the respective validators and returns read only references to the validators.
func (b *BeaconState) ValidatorsReadOnly() []*ReadOnlyValidator {
	if b.state.Validators == nil {
		return nil
	}
	res := make([]*ReadOnlyValidator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		res[i] = &ReadOnlyValidator{validator: val}
	}
	return res
}

// ValidatorAtIndex is the validator at the provided index.
func (b *BeaconState) ValidatorAtIndex(idx uint64) (*ethpb.Validator, error) {
	if b.state.Validators == nil {
		return &ethpb.Validator{}, nil
	}
	if len(b.state.Validators) <= int(idx) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	val := b.state.Validators[idx]
	pubKey := make([]byte, len(val.PublicKey))
	copy(pubKey, val.PublicKey)
	withdrawalCreds := make([]byte, len(val.WithdrawalCredentials))
	copy(withdrawalCreds, val.WithdrawalCredentials)
	return &ethpb.Validator{
		PublicKey:                  pubKey,
		WithdrawalCredentials:      withdrawalCreds,
		EffectiveBalance:           val.EffectiveBalance,
		Slashed:                    val.Slashed,
		ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
		ActivationEpoch:            val.ActivationEpoch,
		ExitEpoch:                  val.ExitEpoch,
		WithdrawableEpoch:          val.WithdrawableEpoch,
	}, nil
}

// ValidatorAtIndexReadOnly is the validator at the provided index.This method
// doesn't clone the validator.
func (b *BeaconState) ValidatorAtIndexReadOnly(idx uint64) (*ReadOnlyValidator, error) {
	if b.state.Validators == nil {
		return &ReadOnlyValidator{}, nil
	}
	if len(b.state.Validators) <= int(idx) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	return &ReadOnlyValidator{b.state.Validators[idx]}, nil
}

// ValidatorIndexByPubkey returns a given validator by its 48-byte public key.
func (b *BeaconState) ValidatorIndexByPubkey(key [48]byte) (uint64, bool) {
	b.lock.RLock()
	b.lock.RUnlock()
	idx, ok := b.valIdxMap[key]
	return idx, ok
}

// PubkeyAtIndex returns the pubkey at the given
// validator index.
func (b *BeaconState) PubkeyAtIndex(idx uint64) [48]byte {
	return bytesutil.ToBytes48(b.state.Validators[idx].PublicKey)
}

// NumValidators returns the size of the validator registry.
func (b *BeaconState) NumValidators() int {
	return len(b.state.Validators)
}

// ReadFromEveryValidator reads values from every validator and applies it to the provided function.
// Warning: This method is potentially unsafe, as it exposes the actual validator registry.
func (b *BeaconState) ReadFromEveryValidator(f func(idx int, val *ReadOnlyValidator) error) error {
	for i, v := range b.state.Validators {
		err := f(i, &ReadOnlyValidator{validator: v})
		if err != nil {
			return err
		}
	}
	return nil
}

// Balances of validators participating in consensus on the beacon chain.
func (b *BeaconState) Balances() []uint64 {
	if b.state.Balances == nil {
		return nil
	}
	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// BalanceAtIndex of validator with the provided index.
func (b *BeaconState) BalanceAtIndex(idx uint64) (uint64, error) {
	if b.state.Balances == nil {
		return 0, nil
	}
	if len(b.state.Balances) <= int(idx) {
		return 0, fmt.Errorf("index of %d does not exist", idx)
	}
	return b.state.Balances[idx], nil
}

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if b.state.RandaoMixes == nil {
		return nil
	}
	mixes := make([][]byte, len(b.state.RandaoMixes))
	for i, r := range b.state.RandaoMixes {
		tmpRt := make([]byte, len(r))
		copy(tmpRt, r)
		mixes[i] = tmpRt
	}
	return mixes
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([]byte, error) {
	if b.state.RandaoMixes == nil {
		return nil, nil
	}
	if len(b.state.RandaoMixes) <= int(idx) {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	root := make([]byte, 32)
	copy(root, b.state.RandaoMixes[idx])
	return root, nil
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if b.state.Slashings == nil {
		return nil
	}
	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

// PreviousEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	if b.state.PreviousEpochAttestations == nil {
		return nil
	}
	res := make([]*pbp2p.PendingAttestation, len(b.state.PreviousEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = proto.Clone(b.state.PreviousEpochAttestations[i]).(*pbp2p.PendingAttestation)
	}
	return res
}

// CurrentEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	if b.state.CurrentEpochAttestations == nil {
		return nil
	}
	res := make([]*pbp2p.PendingAttestation, len(b.state.CurrentEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = proto.Clone(b.state.CurrentEpochAttestations[i]).(*pbp2p.PendingAttestation)
	}
	return res
}

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	if b.state.JustificationBits == nil {
		return nil
	}
	res := make([]byte, len(b.state.JustificationBits.Bytes()))
	copy(res, b.state.JustificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.state.PreviousJustifiedCheckpoint == nil {
		return nil
	}
	cp := &ethpb.Checkpoint{
		Epoch: b.state.PreviousJustifiedCheckpoint.Epoch,
	}
	root := make([]byte, len(b.state.PreviousJustifiedCheckpoint.Root))
	copy(root, b.state.PreviousJustifiedCheckpoint.Root)
	cp.Root = root
	return cp
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.state.CurrentJustifiedCheckpoint == nil {
		return nil
	}
	cp := &ethpb.Checkpoint{
		Epoch: b.state.CurrentJustifiedCheckpoint.Epoch,
	}
	root := make([]byte, len(b.state.CurrentJustifiedCheckpoint.Root))
	copy(root, b.state.CurrentJustifiedCheckpoint.Root)
	cp.Root = root
	return cp
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	if b.state.FinalizedCheckpoint == nil {
		return nil
	}
	cp := &ethpb.Checkpoint{
		Epoch: b.state.FinalizedCheckpoint.Epoch,
	}
	root := make([]byte, len(b.state.FinalizedCheckpoint.Root))
	copy(root, b.state.FinalizedCheckpoint.Root)
	cp.Root = root
	return cp
}

// CopyETH1Data copies the provided eth1data object.
func CopyETH1Data(data *ethpb.Eth1Data) *ethpb.Eth1Data {
	newETH1 := &ethpb.Eth1Data{
		DepositCount: data.DepositCount,
	}
	depositRoot := make([]byte, len(data.DepositRoot))
	blockHash := make([]byte, len(data.BlockHash))

	copy(depositRoot, data.DepositRoot)
	copy(blockHash, data.BlockHash)

	newETH1.DepositRoot = depositRoot
	newETH1.BlockHash = blockHash

	return newETH1
}
