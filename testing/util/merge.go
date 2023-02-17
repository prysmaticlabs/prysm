package util

import (
	v2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// NewBeaconBlockBellatrix creates a beacon block with minimum marshalable fields.
func NewBeaconBlockBellatrix() *ethpb.SignedBeaconBlockBellatrix {
	return HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{})
}

// NewBlindedBeaconBlockBellatrix creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockBellatrix() *ethpb.SignedBlindedBeaconBlockBellatrix {
	return HydrateSignedBlindedBeaconBlockBellatrix(&ethpb.SignedBlindedBeaconBlockBellatrix{})
}

// NewBlindedBeaconBlockBellatrixV2 creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockBellatrixV2() *v2.SignedBlindedBeaconBlockBellatrix {
	return HydrateV2SignedBlindedBeaconBlockBellatrix(&v2.SignedBlindedBeaconBlockBellatrix{})
}

// NewBeaconBlockCapella creates a beacon block with minimum marshalable fields.
func NewBeaconBlockCapella() *ethpb.SignedBeaconBlockCapella {
	return HydrateSignedBeaconBlockCapella(&ethpb.SignedBeaconBlockCapella{})
}

// NewBlindedBeaconBlockCapella creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockCapella() *ethpb.SignedBlindedBeaconBlockCapella {
	return HydrateSignedBlindedBeaconBlockCapella(&ethpb.SignedBlindedBeaconBlockCapella{})
}

// NewBlindedBeaconBlockCapellaV2 creates a blinded beacon block with minimum marshalable fields.
func NewBlindedBeaconBlockCapellaV2() *v2.SignedBlindedBeaconBlockCapella {
	return HydrateV2SignedBlindedBeaconBlockCapella(&v2.SignedBlindedBeaconBlockCapella{})
}
