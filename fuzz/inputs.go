package fuzz

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InputBlockHeader for fuzz testing beacon block headers.
type InputBlockHeader struct {
	State *pb.BeaconState
	Block *ethpb.SignedBeaconBlock
}

// InputAttesterSlashingWrapper for fuzz testing attester slashing.
type InputAttesterSlashingWrapper struct {
	StateID          uint16
	AttesterSlashing *ethpb.AttesterSlashing
}

// InputAttestationWrapper for fuzz testing attestations.
type InputAttestationWrapper struct {
	StateID     uint16
	Attestation *ethpb.Attestation
}

// InputDepositWrapper for fuzz testing deposits.
type InputDepositWrapper struct {
	StateID uint16
	Deposit *ethpb.Deposit
}

// InputVoluntaryExitWrapper for fuzz testing voluntary exits.
type InputVoluntaryExitWrapper struct {
	StateID       uint16
	VoluntaryExit *ethpb.VoluntaryExit
}

// InputProposerSlashingWrapper for fuzz testing proposer slashings.
type InputProposerSlashingWrapper struct {
	StateID          uint16
	ProposerSlashing *ethpb.ProposerSlashing
}
