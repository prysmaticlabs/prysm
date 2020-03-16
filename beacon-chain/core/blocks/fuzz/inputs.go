package fuzz

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

type InputBlockHeader struct {
	StateID uint16
	Block   *ethpb.BeaconBlock
}

type InputAttesterSlashingWrapper struct {
	StateID          uint16
	AttesterSlashing *ethpb.AttesterSlashing
}

type InputAttestationWrapper struct {
	StateID     uint16
	Attestation *ethpb.Attestation
}

type InputDepositWrapper struct {
	StateID uint16
	Deposit *ethpb.Deposit
}

type InputVoluntaryExitWrapper struct {
	StateID       uint16
	VoluntaryExit *ethpb.VoluntaryExit
}

type InputProposerSlashingWrapper struct {
	StateID          uint16
	ProposerSlashing *ethpb.ProposerSlashing
}
