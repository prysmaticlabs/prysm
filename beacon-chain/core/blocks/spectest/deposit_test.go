package spectest

import (
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
		if err = testutil.ConvertToPb(testCase.Pre, preState); err != nil {
			t.Fatal(err)
		}

		deposit := &pb.Deposit{}
		if err = testutil.ConvertToPb(testCase.Deposit, deposit); err != nil {
			t.Fatal(err)
		}

		testPostState := &pb.BeaconState{}
		if err = testutil.ConvertToPb(testCase.Post, testPostState); err != nil {
			t.Fatal(err)
		}
		valMap := stateutils.ValidatorIndexMap(preState)
		_, err := blocks.ProcessDeposit(preState, deposit, valMap, true, true)
		if len(testCase.Post.ValidatorRegistry) == 0 {
			if err == nil {
				t.Errorf("Deposit was processed successfully with deposit %v, when it should have failed", deposit)
			}
		} else {
			if err != nil {
				t.Errorf("Deposit was processed unsuccessfully , when it should have succeeded %v", err)
			}
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
		if err = testutil.ConvertToPb(testCase.Pre, preState); err != nil {
			t.Fatal(err)
		}

		deposit := &pb.Deposit{}
		if err = testutil.ConvertToPb(testCase.Deposit, deposit); err != nil {
			t.Fatal(err)
		}

		testPostState := &pb.BeaconState{}
		if err = testutil.ConvertToPb(testCase.Post, testPostState); err != nil {
			t.Fatal(err)
		}
		valMap := stateutils.ValidatorIndexMap(preState)
		_, err := blocks.ProcessDeposit(preState, deposit, valMap, true, true)
		if len(testCase.Post.ValidatorRegistry) == 0 {
			if err == nil {
				t.Errorf("Deposit was processed successfully with deposit %v, when it should have failed", deposit)
			}
		} else {
			if err != nil {
				t.Errorf("Deposit was processed unsuccessfully , when it should have succeeded %v", err)
			}
		}
	}
}
