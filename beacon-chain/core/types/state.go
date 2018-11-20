package types

import (
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// BeaconState defines the core beacon chain's single
// state containing items pertaining to the validator
// set, recent block hashes, finalized slots, and more.
type BeaconState struct {
	data *pb.BeaconState
}

// Proto returns the underlying protobuf data within a state primitive.
func (b *BeaconState) Proto() *pb.BeaconState {
	return b.data
}

// Marshal encodes state object into the wire format.
func (b *BeaconState) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

// Hash serializes the state object then uses
// blake2b to hash the serialized object.
func (b *BeaconState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(b.data)
	if err != nil {
		return [32]byte{}, err
	}
	return hashutil.Hash(data), nil
}
