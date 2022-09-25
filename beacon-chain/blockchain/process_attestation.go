package blockchain

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// OnAttestation is called whenever an attestation is received, verifies the attestation is valid and saves
// it to the DB. As a stateless function, this does not hold nor delay attestation based on the spec descriptions.
// The delay is handled by the caller in `processAttestations`.
//
// Spec pseudocode definition:
//   def on_attestation(store: Store, attestation: Attestation) -> None:
//    """
//    Run ``on_attestation`` upon receiving a new ``attestation`` from either within a block or directly on the wire.
//
//    An ``attestation`` that is asserted as invalid may be valid at a later time,
//    consider scheduling it for later processing in such case.
//    """
//    validate_on_attestation(store, attestation)
//    store_target_checkpoint_state(store, attestation.data.target)
//
//    # Get state at the `target` to fully validate attestation
//    target_state = store.checkpoint_states[attestation.data.target]
//    indexed_attestation = get_indexed_attestation(target_state, attestation)
//    assert is_valid_indexed_attestation(target_state, indexed_attestation)
//
//    # Update latest messages for attesting indices
//    update_latest_messages(store, indexed_attestation.attesting_indices, attestation)
func (s *Service) OnAttestation(ctx context.Context, a *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onAttestation")
	defer span.End()

	if err := helpers.ValidateNilAttestation(a); err != nil {
		return err
	}
	if err := helpers.ValidateSlotTargetEpoch(a.Data); err != nil {
		return err
	}
	tgt := ethpb.CopyCheckpoint(a.Data.Target)

	// Note that target root check is ignored here because it was performed in sync's validation pipeline:
	// validate_aggregate_proof.go and validate_beacon_attestation.go
	// If missing target root were to fail in this method, it would have just failed in `getAttPreState`.

	// Retrieve attestation's data beacon block pre state. Advance pre state to latest epoch if necessary and
	// save it to the cache.
	baseState, err := s.getAttPreState(ctx, tgt)
	if err != nil {
		return err
	}

	genesisTime := uint64(s.genesisTime.Unix())

	// Verify attestation target is from current epoch or previous epoch.
	if err := verifyAttTargetEpoch(ctx, genesisTime, uint64(time.Now().Unix()), tgt); err != nil {
		return err
	}

	// Verify attestation beacon block is known and not from the future.
	if err := s.verifyBeaconBlock(ctx, a.Data); err != nil {
		return errors.Wrap(err, "could not verify attestation beacon block")
	}

	// Note that LMG GHOST and FFG consistency check is ignored because it was performed in sync's validation pipeline:
	// validate_aggregate_proof.go and validate_beacon_attestation.go

	// Verify attestations can only affect the fork choice of subsequent slots.
	if err := slots.VerifyTime(genesisTime, a.Data.Slot+1, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return err
	}

	// Use the target state to verify attesting indices are valid.
	committee, err := helpers.BeaconCommitteeFromState(ctx, baseState, a.Data.Slot, a.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	indexedAtt, err := attestation.ConvertToIndexed(ctx, a, committee)
	if err != nil {
		return err
	}
	if err := attestation.IsValidAttestationIndices(ctx, indexedAtt); err != nil {
		return err
	}

	// Note that signature verification is ignored here because it was performed in sync's validation pipeline:
	// validate_aggregate_proof.go and validate_beacon_attestation.go
	// We assume trusted attestation in this function has verified signature.

	// Update forkchoice store with the new attestation for updating weight.
	s.cfg.ForkChoiceStore.ProcessAttestation(ctx, indexedAtt.AttestingIndices, bytesutil.ToBytes32(a.Data.BeaconBlockRoot), a.Data.Target.Epoch)

	return nil
}

func (s *Service) OnAggregate(ctx context.Context, a *ethpb.SignedAggregateAttestationAndProof) error {
	ag := a.Message

	if s.ForkChoicer().IsSlashed(ag.AggregatorIndex) {
		return fmt.Errorf("aggregator %d is slashed", ag.AggregatorIndex)
	}

	ce := slots.ToEpoch(s.CurrentSlot())
	ae := slots.ToEpoch(ag.Aggregate.Data.Slot)

	if ae != ce && ae != ce-1 { // TODO: underflow
		return fmt.Errorf("aggregate epoch %d is not within bound %d %d", ae, ce, ce-1)
	}

	if s.ForkChoicer().HasCurrentAggregate(a.Message) {
		return errors.New("aggregate already exists for current epoch")
	}

	if s.ForkChoicer().HasPreviousAggregate(a.Message) {
		return errors.New("aggregate already exists for previous epoch")
	}

	// TODO: validate attestation

	jc := &ethpb.Checkpoint{
		Epoch: s.ForkChoicer().JustifiedCheckpoint().Epoch,
		Root:  s.ForkChoicer().JustifiedCheckpoint().Root[:],
	}
	st, err := s.AttestationTargetState(ctx, jc)
	if err != nil {
		return err
	}
	err = validateAggregateSignatures(st, a)
	if err != nil {
		return err
	}

	if ae == ce {
		err := s.ForkChoicer().InsertCurrentAggregate(a.Message)
		if err != nil {
			return err
		}
	}
	return s.ForkChoicer().InsertPrevAggregate(a.Message)
}

func (s *Service) OnAggregatorEquivocation(ctx context.Context, a *ethpb.AggregatorEquivocation) error {
	a1 := a.SignedAggregate_1
	a2 := a.SignedAggregate_2

	err := slashableAggregateAndProof(a1.Message, a2.Message)
	if err != nil {
		return err
	}

	jc := &ethpb.Checkpoint{
		Epoch: s.ForkChoicer().JustifiedCheckpoint().Epoch,
		Root:  s.ForkChoicer().JustifiedCheckpoint().Root[:],
	}
	st, err := s.AttestationTargetState(ctx, jc)
	if err != nil {
		return err
	}
	err = validateAggregateSignatures(st, a1)
	if err != nil {
		return err
	}
	err = validateAggregateSignatures(st, a2)
	if err != nil {
		return err
	}

	i := a.SignedAggregate_1.Message.AggregatorIndex
	s.ForkChoicer().InsertSlashedIndex(ctx, i)

	currentAtts := s.ForkChoicer().CurrentAttsByAggregator(i)
	prevAtts := s.ForkChoicer().PrevAttsByAggregator(i)

	// TODO: Too slow, optimize this algorithm for union
	m := make(map[[32]byte]bool)
	unexpiredAtts := make([]*ethpb.AggregateAttestationAndProof, 0)
	for _, att := range currentAtts {
		r, err := att.HashTreeRoot()
		if err != nil {
			return err
		}
		ok := m[r]
		if !ok {
			m[r] = true
			unexpiredAtts = append(unexpiredAtts, att)
		}
	}
	for _, att := range prevAtts {
		r, err := att.HashTreeRoot()
		if err != nil {
			return err
		}
		ok := m[r]
		if !ok {
			m[r] = true
			unexpiredAtts = append(unexpiredAtts, att)
		}
	}

	for _, att := range unexpiredAtts {
		data := att.Aggregate.Data
		st, err := s.AttestationTargetState(ctx, data.Target)
		if err != nil {
			return err
		}
		committee, err := helpers.BeaconCommitteeFromState(ctx, st, data.Slot, data.CommitteeIndex)
		if err != nil {
			return err
		}
		indices, err := attestation.AttestingIndices(att.Aggregate.AggregationBits, committee)
		if err != nil {
			return err
		}
		for _, index := range indices {
			r, e, err := s.ForkChoiceStore().CurrentLatestMessage(types.ValidatorIndex(index))
			if err != nil {
				return err
			}
			if bytes.Equal(r[:], data.BeaconBlockRoot) && e == slots.ToEpoch(data.Slot) {
				s.ForkChoicer().MinusCurrentReferenceCount(types.ValidatorIndex(index))
			}
			r, e, err = s.ForkChoiceStore().PrevLatestMessage(types.ValidatorIndex(index))
			if err != nil {
				return err
			}
			if bytes.Equal(r[:], data.BeaconBlockRoot) && e == slots.ToEpoch(data.Slot) {
				s.ForkChoicer().MinusPrevReferenceCount(types.ValidatorIndex(index))
			}
		}
	}

	return nil
}

func slashableAggregateAndProof(a1 *ethpb.AggregateAttestationAndProof, a2 *ethpb.AggregateAttestationAndProof) error {
	if a1.AggregatorIndex != a2.AggregatorIndex {
		return fmt.Errorf("missmatch aggregator index %d != %d", a1.AggregatorIndex, a2.AggregatorIndex)
	}
	if a1.Aggregate.Data.Slot != a2.Aggregate.Data.Slot {
		return fmt.Errorf("missmatch aggregate slot %d != %d", a1.Aggregate.Data.Slot, a2.Aggregate.Data.Slot)
	}
	r1, err := a1.HashTreeRoot()
	if err != nil {
		return err
	}
	r2, err := a2.HashTreeRoot()
	if err != nil {
		return err
	}
	if r1 == r2 {
		return errors.New("aggregate 1 is the same as aggregate 2")
	}
	return nil
}

func validateAggregateSignatures(st state.BeaconState, a *ethpb.SignedAggregateAttestationAndProof) error {
	msg := a.Message
	data := msg.Aggregate.Data
	epoch := slots.ToEpoch(data.Slot)

	v, err := st.ValidatorAtIndex(msg.AggregatorIndex)
	if err != nil {
		return err
	}
	pk, err := bls.PublicKeyFromBytes(v.PublicKey)
	if err != nil {
		return err
	}

	d, err := signing.Domain(st.Fork(), epoch, params.BeaconConfig().DomainSelectionProof, st.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	sszUint := types.SSZUint64(data.Slot)
	root, err := signing.ComputeSigningRoot(&sszUint, d)
	if err != nil {
		return err
	}

	b1 := &bls.SignatureBatch{
		Signatures: [][]byte{msg.SelectionProof},
		PublicKeys: []bls.PublicKey{pk},
		Messages:   [][32]byte{root},
	}

	d, err = signing.Domain(st.Fork(), epoch, params.BeaconConfig().DomainAggregateAndProof, st.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	root, err = signing.ComputeSigningRoot(a.Message, d)
	if err != nil {
		return err
	}

	b2 := &bls.SignatureBatch{
		Signatures: [][]byte{a.Signature},
		PublicKeys: []bls.PublicKey{pk},
		Messages:   [][32]byte{root},
	}

	set := bls.NewSet()
	set.Join(b1).Join(b2)
	verified, err := set.Verify()
	if err != nil {
		return err
	}
	if !verified {
		return errors.New("could not validate signature")
	}
	return nil
}
