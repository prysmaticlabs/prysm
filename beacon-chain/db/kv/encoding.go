package kv

import (
	"errors"
	"reflect"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func decode(data []byte, dst proto.Message) error {
	data, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	if isWhitelisted(dst) {
		return dst.(fastssz.Unmarshaler).UnmarshalSSZ(data)
	}
	return proto.Unmarshal(data, dst)
}

func encode(msg proto.Message) ([]byte, error) {
	if msg == nil || reflect.ValueOf(msg).IsNil() {
		return nil, errors.New("cannot encode nil message")
	}
	var enc []byte
	var err error
	if isWhitelisted(msg) {
		enc, err = msg.(fastssz.Marshaler).MarshalSSZ()
		if err != nil {
			return nil, err
		}
	} else {
		enc, err = proto.Marshal(msg)
		if err != nil {
			return nil, err
		}
	}
	return snappy.Encode(nil, enc), nil
}

func isWhitelisted(obj interface{}) bool {
	switch obj.(type) {
	case *pb.BeaconState:
		return true
	case *ethpb.SignedBeaconBlock:
		return true
	case *ethpb.SignedAggregateAttestationAndProof:
		return true
	case *ethpb.BeaconBlock:
		return true
	case *ethpb.Attestation:
		return true
	case *ethpb.Deposit:
		return true
	case *ethpb.AttesterSlashing:
		return true
	case *ethpb.ProposerSlashing:
		return true
	case *ethpb.VoluntaryExit:
		return true
	default:
		return false
	}
}
