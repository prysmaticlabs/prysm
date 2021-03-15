package stateV0

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// ReadOnlyValidator returns a wrapper that only allows fields from a validator
// to be read, and prevents any modification of internal validator fields.
type ReadOnlyValidator struct {
	validator *ethpb.Validator
}

// EffectiveBalance returns the effective balance of the
// read only Validator.
func (v ReadOnlyValidator) EffectiveBalance() uint64 {
	if v.IsNil() {
		return 0
	}
	return v.Validator.EffectiveBalance
}

// ActivationEligibilityEpoch returns the activation eligibility epoch of the
// read only Validator.
func (v ReadOnlyValidator) ActivationEligibilityEpoch() types.Epoch {
	if v.IsNil() {
		return 0
	}
	return v.Validator.ActivationEligibilityEpoch
}

// ActivationEpoch returns the activation epoch of the
// read only Validator.
func (v ReadOnlyValidator) ActivationEpoch() types.Epoch {
	if v.IsNil() {
		return 0
	}
	return v.Validator.ActivationEpoch
}

// WithdrawableEpoch returns the withdrawable epoch of the
// read only Validator.
func (v ReadOnlyValidator) WithdrawableEpoch() types.Epoch {
	if v.IsNil() {
		return 0
	}
	return v.Validator.WithdrawableEpoch
}

// ExitEpoch returns the exit epoch of the
// read only Validator.
func (v ReadOnlyValidator) ExitEpoch() types.Epoch {
	if v.IsNil() {
		return 0
	}
	return v.Validator.ExitEpoch
}

// PublicKey returns the public key of the
// read only Validator.
func (v ReadOnlyValidator) PublicKey() [48]byte {
	if v.IsNil() {
		return [48]byte{}
	}
	var pubkey [48]byte
	copy(pubkey[:], v.Validator.PublicKey)
	return pubkey
}

// WithdrawalCredentials returns the withdrawal credentials of the
// read only Validator.
func (v ReadOnlyValidator) WithdrawalCredentials() []byte {
	creds := make([]byte, len(v.Validator.WithdrawalCredentials))
	copy(creds, v.Validator.WithdrawalCredentials)
	return creds
}

// Slashed returns the read only Validator is slashed.
func (v ReadOnlyValidator) Slashed() bool {
	if v.IsNil() {
		return false
	}
	return v.Validator.Slashed
}

// CopyValidator returns the copy of the read only Validator.
func (v ReadOnlyValidator) IsNil() bool {
	return v.Validator == nil
}
