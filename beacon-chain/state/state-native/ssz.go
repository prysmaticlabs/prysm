package state_native

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

var errAssertionFailed = errors.New("failed to convert interface to proto state")
var errUnsupportedVersion = errors.New("unsupported beacon state version")

func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	proto := b.ToProto()
	switch b.Version() {
	case version.Phase0:
		s, ok := proto.(*ethpb.BeaconState)
		if !ok {
			return nil, errAssertionFailed
		}
		return s.MarshalSSZ()
	case version.Altair:
		s, ok := proto.(*ethpb.BeaconStateAltair)
		if !ok {
			return nil, errAssertionFailed
		}
		return s.MarshalSSZ()
	case version.Bellatrix:
		s, ok := proto.(*ethpb.BeaconStateBellatrix)
		if !ok {
			return nil, errAssertionFailed
		}
		return s.MarshalSSZ()
	default:
		return nil, errUnsupportedVersion
	}
}
