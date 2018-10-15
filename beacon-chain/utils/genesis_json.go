package utils

import (
	"fmt"
	"os"

	"github.com/golang/protobuf/jsonpb"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InitialValidatorsFromJSON retrieves the validator set that is stored in
// genesis.json.
func InitialValidatorsFromJSON(genesisJSONPath string) ([]*pb.ValidatorRecord, error) {
	// #nosec G304
	// genesisJSONPath is a user input for the path of genesis.json.
	// Ex: /path/to/my/genesis.json.
	f, err := os.Open(genesisJSONPath)
	if err != nil {
		return nil, err
	}

	cState := &pb.CrystallizedState{}
	if err := jsonpb.Unmarshal(f, cState); err != nil {
		return nil, fmt.Errorf("error converting JSON to proto: %v", err)
	}

	return cState.Validators, nil
}
