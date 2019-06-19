package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestRegistryProcessingYaml(t *testing.T) {
	file, err := ioutil.ReadFile("proposer_slashing_minimal_formatted.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &ProposerSlashingMinimal{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	t.Logf("Running spec test vectors for %s", s.Title)
	for _, testCase := range s.TestCases {
		t.Logf("Testing test case %s", testCase.Description)
		preState := &pb.BeaconState{}
		err := convertToPb(testCase.Pre, preState)
		if err != nil {
			t.Fatal(err)
		}

		proposerSlashing := &pb.ProposerSlashing{}
		err = convertToPb(testCase.ProposerSlashing, proposerSlashing)
		if err != nil {
			t.Fatal(err)
		}

		genPostState := &pb.BeaconState{}
		err = convertToPb(testCase.Post, genPostState)
		if err != nil {
			t.Fatal(err)
		}

		block := &pb.BeaconBlock{Body: &pb.BeaconBlockBody{ProposerSlashings: []*pb.ProposerSlashing{proposerSlashing}}}
		var postState *pb.BeaconState
		postState, err = blocks.ProcessProposerSlashings(preState, block, true)
		// Note: This doesn't test anything worthwhile. It essentially tests
		// that *any* error has occurred, not any specific error.
		if len(genPostState.ValidatorRegistry) == 0 {
			if err == nil {
				t.Fatal("Did not fail when expected")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(postState, genPostState) {
			t.Errorf("test case %s mutated state differently than yaml output", testCase.Description)
		}
	}
}
