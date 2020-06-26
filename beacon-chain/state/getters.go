package state

import (
	"errors"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// EffectiveBalance returns the effective balance of the
// read only validator.
func (v *ReadOnlyValidator) EffectiveBalance() uint64 {
	if v == nil || v.validator == nil {
		return 0
	}
	return v.validator.EffectiveBalance
}

// ActivationEligibilityEpoch returns the activation eligibility epoch of the
// read only validator.
func (v *ReadOnlyValidator) ActivationEligibilityEpoch() uint64 {
	if v == nil || v.validator == nil {
		return 0
	}
	return v.validator.ActivationEligibilityEpoch
}

// ActivationEpoch returns the activation epoch of the
// read only validator.
func (v *ReadOnlyValidator) ActivationEpoch() uint64 {
	if v == nil || v.validator == nil {
		return 0
	}
	return v.validator.ActivationEpoch
}

// WithdrawableEpoch returns the withdrawable epoch of the
// read only validator.
func (v *ReadOnlyValidator) WithdrawableEpoch() uint64 {
	if v == nil || v.validator == nil {
		return 0
	}
	return v.validator.WithdrawableEpoch
}

// ExitEpoch returns the exit epoch of the
// read only validator.
func (v *ReadOnlyValidator) ExitEpoch() uint64 {
	if v == nil || v.validator == nil {
		return 0
	}
	return v.validator.ExitEpoch
}

// PublicKey returns the public key of the
// read only validator.
func (v *ReadOnlyValidator) PublicKey() [48]byte {
	if v == nil || v.validator == nil {
		return [48]byte{}
	}
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
	if v == nil || v.validator == nil {
		return false
	}
	return v.validator.Slashed
}

// CopyValidator returns the copy of the read only validator.
func (v *ReadOnlyValidator) CopyValidator() *ethpb.Validator {
	if v == nil || v.validator == nil {
		return nil
	}
	return CopyValidator(v.validator)
}

// InnerStateUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) InnerStateUnsafe() *pbp2p.BeaconState {
	if b == nil {
		return nil
	}
	return b.state
}

// CloneInnerState the beacon state into a protobuf for usage.
func (b *BeaconState) CloneInnerState() *pbp2p.BeaconState {
	if b == nil || b.state == nil {
		return nil
	}
	return &pbp2p.BeaconState{
		GenesisTime:                 b.GenesisTime(),
		GenesisValidatorsRoot:       b.GenesisValidatorRoot(),
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
	return b != nil && b.state != nil
}

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	if !b.HasInnerState() {
		return 0
	}
	return b.state.GenesisTime
}

// GenesisValidatorRoot of the beacon state.
func (b *BeaconState) GenesisValidatorRoot() []byte {
	if !b.HasInnerState() {
		return nil
	}

	if b.state.GenesisValidatorsRoot == nil {
		return params.BeaconConfig().ZeroHash[:]
	}

	root := make([]byte, 32)
	copy(root, b.state.GenesisValidatorsRoot)
	return root
}

// GenesisUnixTime returns the genesis time as time.Time.
func (b *BeaconState) GenesisUnixTime() time.Time {
	if !b.HasInnerState() {
		return time.Unix(0, 0)
	}
	return time.Unix(int64(b.state.GenesisTime), 0)
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() uint64 {
	if !b.HasInnerState() {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.state.Slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *pbp2p.Fork {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.Fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return nil
	}
	if b.state.LatestBlockHeader == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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

// ParentRoot is a convenience method to access state.LatestBlockRoot.ParentRoot.
func (b *BeaconState) ParentRoot() [32]byte {
	if !b.HasInnerState() {
		return [32]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	parentRoot := [32]byte{}
	copy(parentRoot[:], b.state.LatestBlockHeader.ParentRoot)
	return parentRoot
}

// BlockRoots kept track of in the beacon state.
func (b *BeaconState) BlockRoots() [][]byte {
	if !b.HasInnerState() {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.BlockRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if uint64(len(b.state.BlockRoots)) <= idx {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	root := make([]byte, 32)
	copy(root, b.state.BlockRoots[idx])
	return root, nil
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() [][]byte {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.StateRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return nil
	}
	if b.state.HistoricalRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.Eth1Data == nil {
		return nil
	}

	return CopyETH1Data(b.state.Eth1Data)
}

// Eth1DataVotes corresponds to votes from eth2 on the canonical proof-of-work chain
// data retrieved from eth1.
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	if !b.HasInnerState() {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.state.Eth1DepositIndex
}

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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

// ValidatorsReadOnly returns validators participating in consensus on the beacon chain. This
// method doesn't clone the respective validators and returns read only references to the validators.
func (b *BeaconState) ValidatorsReadOnly() []*ReadOnlyValidator {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	res := make([]*ReadOnlyValidator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		res[i] = &ReadOnlyValidator{validator: val}
	}
	return res
}

// ValidatorAtIndex is the validator at the provided index.
func (b *BeaconState) ValidatorAtIndex(idx uint64) (*ethpb.Validator, error) {
	if !b.HasInnerState() {
		return nil, ErrNilInnerState
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.Validators == nil {
		return &ethpb.Validator{}, nil
	}

	if uint64(len(b.state.Validators)) <= idx {
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

// ValidatorAtIndexReadOnly is the validator at the provided index. This method
// doesn't clone the validator.
func (b *BeaconState) ValidatorAtIndexReadOnly(idx uint64) (*ReadOnlyValidator, error) {
	if !b.HasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return &ReadOnlyValidator{}, nil
	}
	if uint64(len(b.state.Validators)) <= idx {
		return nil, fmt.Errorf("index %d out of range", idx)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return &ReadOnlyValidator{b.state.Validators[idx]}, nil
}

// ValidatorIndexByPubkey returns a given validator by its 48-byte public key.
func (b *BeaconState) ValidatorIndexByPubkey(key [48]byte) (uint64, bool) {
	if b == nil || b.valIdxMap == nil {
		return 0, false
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	idx, ok := b.valIdxMap[key]
	return idx, ok
}

func (b *BeaconState) validatorIndexMap() map[[48]byte]uint64 {
	if b == nil || b.valIdxMap == nil {
		return map[[48]byte]uint64{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	m := make(map[[48]byte]uint64, len(b.valIdxMap))

	for k, v := range b.valIdxMap {
		m[k] = v
	}
	return m
}

// PubkeyAtIndex returns the pubkey at the given
// validator index.
func (b *BeaconState) PubkeyAtIndex(idx uint64) [48]byte {
	if !b.HasInnerState() {
		return [48]byte{}
	}
	if idx >= uint64(len(b.state.Validators)) {
		return [48]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return bytesutil.ToBytes48(b.state.Validators[idx].PublicKey)
}

// NumValidators returns the size of the validator registry.
func (b *BeaconState) NumValidators() int {
	if !b.HasInnerState() {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.state.Validators)
}

// ReadFromEveryValidator reads values from every validator and applies it to the provided function.
// Warning: This method is potentially unsafe, as it exposes the actual validator registry.
func (b *BeaconState) ReadFromEveryValidator(f func(idx int, val *ReadOnlyValidator) error) error {
	if !b.HasInnerState() {
		return ErrNilInnerState
	}
	if b.state.Validators == nil {
		return errors.New("nil validators in state")
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// BalanceAtIndex of validator with the provided index.
func (b *BeaconState) BalanceAtIndex(idx uint64) (uint64, error) {
	if !b.HasInnerState() {
		return 0, ErrNilInnerState
	}
	if b.state.Balances == nil {
		return 0, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if uint64(len(b.state.Balances)) <= idx {
		return 0, fmt.Errorf("index of %d does not exist", idx)
	}
	return b.state.Balances[idx], nil
}

// BalancesLength returns the length of the balances slice.
func (b *BeaconState) BalancesLength() int {
	if !b.HasInnerState() {
		return 0
	}
	if b.state.Balances == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.state.Balances)
}

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.RandaoMixes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

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
	if !b.HasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.RandaoMixes == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if uint64(len(b.state.RandaoMixes)) <= idx {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	root := make([]byte, 32)
	copy(root, b.state.RandaoMixes[idx])
	return root, nil
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	if !b.HasInnerState() {
		return 0
	}
	if b.state.RandaoMixes == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.state.RandaoMixes)
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

// PreviousEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.PreviousEpochAttestations == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	res := make([]*pbp2p.PendingAttestation, len(b.state.PreviousEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = CopyPendingAttestation(b.state.PreviousEpochAttestations[i])
	}
	return res
}

// CurrentEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.CurrentEpochAttestations == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	res := make([]*pbp2p.PendingAttestation, len(b.state.CurrentEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = CopyPendingAttestation(b.state.CurrentEpochAttestations[i])
	}
	return res
}

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.JustificationBits == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	res := make([]byte, len(b.state.JustificationBits.Bytes()))
	copy(res, b.state.JustificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.PreviousJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return CopyCheckpoint(b.state.PreviousJustifiedCheckpoint)
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.CurrentJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return CopyCheckpoint(b.state.CurrentJustifiedCheckpoint)
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	if !b.HasInnerState() {
		return nil
	}
	if b.state.FinalizedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return CopyCheckpoint(b.state.FinalizedCheckpoint)
}

// FinalizedCheckpointEpoch returns the epoch value of the finalized checkpoint.
func (b *BeaconState) FinalizedCheckpointEpoch() uint64 {
	if !b.HasInnerState() {
		return 0
	}
	if b.state.FinalizedCheckpoint == nil {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.state.FinalizedCheckpoint.Epoch
}
