package casper

import (
	"math/big"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

// ValidatorRecord defines the validator object which
// contains all the information of a single validator
type ValidatorRecord struct {
	data *pb.ValidatorRecord
}

// Proto returns the underlying protobuf data within a state primitive.
func (v *ValidatorRecord) Proto() *pb.ValidatorRecord {
	return v.data
}

// Marshal encodes crystallized state object into the wire format.
func (v *ValidatorRecord) Marshal() ([]byte, error) {
	return proto.Marshal(v.data)
}

// Hash serializes the validator record object
// blake2b to hash the serialized object.
func (v *ValidatorRecord) Hash() ([32]byte, error) {
	data, err := proto.Marshal(v.data)
	if err != nil {
		return [32]byte{}, err
	}
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash, nil
}

func (v *ValidatorRecord) GetBalance() (*big.Int, error) {
	balance := big.NewInt(0)
	err := balance.UnmarshalJSON(v.data.Balance)
	if err != nil {
		return nil, err
	}
	return balance, nil
}
