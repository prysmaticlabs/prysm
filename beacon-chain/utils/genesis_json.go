package utils

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InitialValidatorRegistryFromJSON retrieves the validator set that is stored in
// genesis.json.
func InitialValidatorRegistryFromJSON(genesisJSONPath string) ([]*pb.ValidatorRecord, error) {
	// genesisJSONPath is a user input for the path of genesis.json.
	// Ex: /path/to/my/genesis.json.
	f, err := os.Open(genesisJSONPath) // #nosec
	if err != nil {
		return nil, err
	}

	beaconState := &pb.BeaconState{}
	if err := jsonpb.Unmarshal(f, beaconState); err != nil {
		return nil, fmt.Errorf("error converting JSON to proto: %v", err)
	}

	return beaconState.ValidatorRegistry, nil
}
