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
