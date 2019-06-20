package spectest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestDepositMinimalYaml(t *testing.T) {
	file, err := ioutil.ReadFile("deposit_minimal.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &DepositsMinimal{}
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

		deposit := &pb.Deposit{}
		b, err = json.Marshal(testCase.Deposit)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), deposit)
		if err != nil {
			t.Fatal(err)
		}

		var postState *pb.BeaconState
		_ = postState
		valMap := stateutils.ValidatorIndexMap(preState)
		postState, err = blocks.ProcessDeposit(preState, deposit, valMap, true, true)
		if err != nil {
			t.Errorf("Deposit was processed successfully with deposit %v, when it should have failed", err)
		}
	}
}

func TestDepositMainnetYaml(t *testing.T) {
	file, err := ioutil.ReadFile("deposit_mainnet.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &DepositMainnet{}
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

		deposit := &pb.Deposit{}
		b, err = json.Marshal(testCase.Deposit)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), deposit)
		if err != nil {
			t.Fatal(err)
		}

		var postState *pb.BeaconState
		postState, err = blocks.ProcessDeposit(preState, deposit, nil, true, true)
		if err == nil && postState != nil {
			t.Errorf("Deposit was processed successfully with deposit %v, when it should have failed", deposit)
		}
	}
}
