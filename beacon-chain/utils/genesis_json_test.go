package utils

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestInitGenesisJsonFailure(t *testing.T) {
	fname := "/genesis.json"
	pwd, _ := os.Getwd()
	fnamePath := pwd + fname

	_, err := InitialValidatorRegistryFromJSON(fnamePath)
	if err == nil {
		t.Fatalf("genesis.json should have failed %v", err)
	}
}

func TestInitGenesisJson(t *testing.T) {
	// Support running this test via bazel or go test.
	var fNamePath string
	if os.Getenv("RUNFILES_DIR") != "" {
		fNamePath = fmt.Sprintf("%s/%s/%s",
			os.Getenv("RUNFILES_DIR"),
			os.Getenv("TEST_WORKSPACE"),
			"/genesis.json",
		)
	} else {
		fNamePath = "../../genesis.json"
	}

	params.UseDemoBeaconConfig()
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Pubkey: []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), Balance: 32000000000},
		},
	}

	validators, err := InitialValidatorRegistryFromJSON(fNamePath)
	if err != nil {
		t.Fatalf("Reading validatory registry from genesis.json failed %v", err)
	}

	if !reflect.DeepEqual(state.ValidatorRegistry[0], validators[0]) {
		t.Error("Validator registry mismatched")
	}
}
