package util

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

// NewBeaconBlockMerge creates a beacon block with minimum marshalable fields.
func NewBeaconBlockMerge() *ethpb.SignedBeaconBlockMerge {
	return HydrateSignedBeaconBlockMerge(&ethpb.SignedBeaconBlockMerge{})
}
