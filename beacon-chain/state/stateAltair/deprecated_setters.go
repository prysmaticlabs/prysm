package stateAltair

import (
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// SetPreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) SetPreviousEpochAttestations(val []*pbp2p.PendingAttestation) error {
	panic("CurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// SetCurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) SetCurrentEpochAttestations(val []*pbp2p.PendingAttestation) error {
	panic("CurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// AppendCurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) AppendCurrentEpochAttestations(val *pbp2p.PendingAttestation) error {
	panic("AppendCurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// AppendPreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) AppendPreviousEpochAttestations(val *pbp2p.PendingAttestation) error {
	panic("AppendPreviousEpochAttestations is not supported for hard fork 1 beacon state")
}
