package state_native

import (
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
)

var errAssertionFailed = errors.New("failed to convert interface to proto state")

func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	proto, err := b.ToProto()
	if err != nil {
		return nil, err
	}

	s, ok := proto.(ssz.Marshaler)
	if !ok {
		return nil, errAssertionFailed
	}
	return s.MarshalSSZ()
}
