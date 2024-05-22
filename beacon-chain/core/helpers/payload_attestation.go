package helpers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/math"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

var (
	errNilMessage            = errors.New("nil PayloadAttestationMessage")
	errNilData               = errors.New("nil PayloadAttestationData")
	errNilBeaconBlockRoot    = errors.New("nil BeaconBlockRoot")
	errNilPayloadAttestation = errors.New("nil PayloadAttestation")
	errNilSignature          = errors.New("nil Signature")
	errNilAggregationBits    = errors.New("nil AggregationBits")
	errPreEPBSState          = errors.New("beacon state pre ePBS fork")
	errCommitteeOverflow     = errors.New("beacon committee of insufficient size")
)

// ValidateNilPayloadAttestationData checks if any composite field of the
// payload attestation data is nil
func ValidateNilPayloadAttestationData(data *eth.PayloadAttestationData) error {
	if data == nil {
		return errNilData
	}
	if data.BeaconBlockRoot == nil {
		return errNilBeaconBlockRoot
	}
	return nil
}

// ValidateNilPayloadAttestationMessage checks if any composite field of the
// payload attestation message is nil
func ValidateNilPayloadAttestationMessage(att *eth.PayloadAttestationMessage) error {
	if att == nil {
		return errNilMessage
	}
	if att.Signature == nil {
		return errNilSignature
	}
	return ValidateNilPayloadAttestationData(att.Data)
}

// ValidateNilPayloadAttestation checks if any composite field of the
// payload attestation is nil
func ValidateNilPayloadAttestation(att *eth.PayloadAttestation) error {
	if att == nil {
		return errNilPayloadAttestation
	}
	if att.AggregationBits == nil {
		return errNilAggregationBits
	}
	if att.Signature == nil {
		return errNilSignature
	}
	return ValidateNilPayloadAttestationData(att.Data)
}

// InPayloadTimelinessCommittee returns whether the given index belongs to the
// PTC computed from the passed state.
func InPayloadTimelinessCommittee(ctx context.Context, state state.ReadOnlyBeaconState, slot primitives.Slot, idx primitives.ValidatorIndex) (bool, error) {
	ptc, err := GetPayloadTimelinessCommittee(ctx, state, slot)
	if err != nil {
		return false, err
	}
	for _, i := range ptc {
		if i == idx {
			return true, nil
		}
	}
	return false, nil
}

// GetPayloadTimelinessCommittee returns the PTC for the given slot, computed from the passed state as in the
// spec function `get_ptc`.
func GetPayloadTimelinessCommittee(ctx context.Context, state state.ReadOnlyBeaconState, slot primitives.Slot) (indices []primitives.ValidatorIndex, err error) {
	if state.Version() < version.EPBS {
		return nil, errPreEPBSState
	}
	epoch := slots.ToEpoch(slot)
	activeCount, err := ActiveValidatorCount(ctx, state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute active validator count")
	}
	committeesPerSlot := math.LargestPowerOfTwo(math.Min(SlotCommitteeCount(activeCount), fieldparams.PTCSize))
	membersPerCommittee := fieldparams.PTCSize / committeesPerSlot
	for i := uint64(0); i < committeesPerSlot; i++ {
		committee, err := BeaconCommitteeFromState(ctx, state, slot, primitives.CommitteeIndex(i))
		if err != nil {
			return nil, err
		}
		if uint64(len(committee)) < membersPerCommittee {
			return nil, errCommitteeOverflow
		}
		start := uint64(len(committee)) - membersPerCommittee
		indices = append(indices, committee[start:]...)
	}
	return
}

// ValidatePayloadAttestationMessageSignature verifies the signature of a
// payload attestation message.
func ValidatePayloadAttestationMessageSignature(ctx context.Context, st state.ReadOnlyBeaconState, msg *eth.PayloadAttestationMessage) error {
	if err := ValidateNilPayloadAttestationMessage(msg); err != nil {
		return err
	}
	val, err := st.ValidatorAtIndex(msg.ValidatorIndex)
	if err != nil {
		return err
	}
	pub, err := bls.PublicKeyFromBytes(val.PublicKey)
	if err != nil {
		return err
	}
	sig, err := bls.SignatureFromBytes(msg.Signature)
	if err != nil {
		return err
	}
	currentEpoch := slots.ToEpoch(st.Slot())
	domain, err := signing.Domain(st.Fork(), currentEpoch, params.BeaconConfig().DomainPTCAttester, st.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	root, err := signing.ComputeSigningRoot(msg.Data, domain)
	if err != nil {
		return err
	}
	if !sig.Verify(pub, root[:]) {
		return signing.ErrSigFailedToVerify
	}
	return nil
}
