package spectest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
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
		t.Logf("Testing testcase %s", testCase.Description)
		preState := &pb.BeaconState{}
		b, err := json.Marshal(testCase.Pre)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), preState)
		if err != nil {
			t.Fatal(err)
		}

		proposerSlashing := &pb.ProposerSlashing{}
		b, err = json.Marshal(testCase.ProposerSlashing)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), proposerSlashing)
		if err != nil {
			t.Fatal(err)
		}

		genPostState := &pb.BeaconState{}
		b, err = json.Marshal(testCase.Post)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), genPostState)
		if err != nil {
			t.Fatal(err)
		}

		block := &pb.BeaconBlock{Body: &pb.BeaconBlockBody{ProposerSlashings: []*pb.ProposerSlashing{proposerSlashing}}}
		var postState *pb.BeaconState
		postState, err = blocks.ProcessProposerSlashings(preState, block, true)
		if err != nil && postState != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(postState, genPostState) {
			t.Errorf("test case %s mutated state differently than yaml output", testCase.Description)
		}
	}
}
