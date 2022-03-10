package util

import (
	v2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// NewBeaconBlockBellatrix creates a beacon block with minimum marshalable fields.
func NewBeaconBlockBellatrix() *ethpb.SignedBeaconBlockBellatrix {
	return HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{})
}

// NewBlindedBeaconBlockBellatrix creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockBellatrix() *v2.SignedBlindedBeaconBlockBellatrix {
	return HydrateSignedBlindedBeaconBlockBellatrix(&v2.SignedBlindedBeaconBlockBellatrix{})
}
