package state_native

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
)

var errAssertionFailed = errors.New("failed to convert interface to proto state")

func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	proto := b.ToProto()

	s, ok := proto.(ssz.Marshaler)
	if !ok {
		return nil, errAssertionFailed
	}
	return s.MarshalSSZ()
}
