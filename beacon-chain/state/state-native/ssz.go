package state_native

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var errAssertionFailed = errors.New("failed to convert interface to proto state")
var errUnsupportedVersion = errors.New("unsupported beacon state version")

func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	proto := b.ToProto()
	switch b.version {
	case Phase0:
		s, ok := proto.(*ethpb.BeaconState)
		if !ok {
			return nil, errAssertionFailed
		}
		return s.MarshalSSZ()
	case Altair:
		s, ok := proto.(*ethpb.BeaconStateAltair)
		if !ok {
			return nil, errAssertionFailed
		}
		return s.MarshalSSZ()
	case Bellatrix:
		s, ok := proto.(*ethpb.BeaconStateBellatrix)
		if !ok {
			return nil, errAssertionFailed
		}
		return s.MarshalSSZ()
	default:
		return nil, errUnsupportedVersion
	}
}
