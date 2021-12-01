package migration

import (
	"github.com/pkg/errors"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// V1Alpha1BeaconBlockMergeToV2 converts a v1alpha1 Merge beacon block to a v2
// Merge block.
func V1Alpha1BeaconBlockMergeToV2(v1alpha1Block *ethpbalpha.BeaconBlockMerge) (*ethpbv2.BeaconBlockMerge, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockMerge{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}
