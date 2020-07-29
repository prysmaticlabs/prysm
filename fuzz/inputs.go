package fuzz

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// InputBlockHeader for fuzz testing beacon block headers.
type InputBlockHeader struct {
	StateID uint64
	Block   *ethpb.BeaconBlock
}

// InputAttesterSlashingWrapper for fuzz testing attester slashing.
type InputAttesterSlashingWrapper struct {
	StateID          uint64
	AttesterSlashing *ethpb.AttesterSlashing
}

// InputAttestationWrapper for fuzz testing attestations.
type InputAttestationWrapper struct {
	StateID     uint64
	Attestation *ethpb.Attestation
}

// InputDepositWrapper for fuzz testing deposits.
type InputDepositWrapper struct {
	StateID uint64
	Deposit *ethpb.Deposit
}

// InputVoluntaryExitWrapper for fuzz testing voluntary exits.
type InputVoluntaryExitWrapper struct {
	StateID       uint16
	VoluntaryExit *ethpb.VoluntaryExit
}

// InputProposerSlashingWrapper for fuzz testing proposer slashings.
type InputProposerSlashingWrapper struct {
	StateID          uint64
	ProposerSlashing *ethpb.ProposerSlashing
}
