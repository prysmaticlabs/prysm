package testing

import (
	"bytes"
	"io/ioutil"
	"path"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"

)

func TestYamlStatic(t *testing.T) {
	topPath := "/eth2_spec_tests/tests/ssz_static/core/"
	yamlFileNames := []string{
		"ssz_minimal_lengthy.yaml",
		"ssz_minimal_max.yaml",
		"ssz_minimal_nil.yaml",
		"ssz_minimal_one.yaml",
		"ssz_minimal_random.yaml",
		"ssz_minimal_random_chaos.yaml",
		"ssz_minimal_zero.yaml",
		"ssz_mainnet_random.yaml",
	}

	for _, f := range yamlFileNames {
		fullPath := path.Join(topPath, f)
		filepath, err := bazel.Runfile(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		file, err := ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatalf("Could not load file %v", err)
		}
		s := &SszProtobufTest{}
		if err := yaml.Unmarshal(file, s); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		runTestCases(t, s)
	}
}

func runTestCases(t *testing.T, s *SszProtobufTest) {
	for _, testCase := range s.TestCases {
		if testCase.Attestation.Value != nil {
			p := &pb.Attestation{}
			if err := testutil.ConvertToPb(testCase.Attestation.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Attestation.Root) {
				t.Errorf("Expected attestation %#x, received %#x", testCase.Attestation.Root, root[:])
			}
		}
		if testCase.AttestationData.Value != nil {
			p := &pb.AttestationData{}
			if err := testutil.ConvertToPb(testCase.AttestationData.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.AttestationData.Root) {
				t.Errorf("Expected attestation data %#x, received %#x", testCase.AttestationData.Root, root[:])
			}
		}
		if testCase.AttestationDataAndCustodyBit.Value != nil {
			p := &pb.AttestationDataAndCustodyBit{}
			if err := testutil.ConvertToPb(testCase.AttestationDataAndCustodyBit.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.AttestationDataAndCustodyBit.Root) {
				t.Errorf("Expected attestation data and custody bit %#x, received %#x", testCase.AttestationDataAndCustodyBit.Root, root[:])
			}
		}
		if testCase.AttesterSlashing.Value != nil {
			p := &pb.AttesterSlashing{}
			if err := testutil.ConvertToPb(testCase.AttesterSlashing.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.AttesterSlashing.Root) {
				t.Errorf("Expected attester slashing bit %#x, received %#x", testCase.AttesterSlashing.Root, root[:])
			}
		}
		if testCase.BeaconBlock.Value != nil {
			p := &pb.BeaconBlock{}
			if err := testutil.ConvertToPb(testCase.BeaconBlock.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.BeaconBlock.Root) {
				t.Errorf("Expected beacon block %#x, received %#x", testCase.BeaconBlock.Root, root[:])
			}
		}
		if testCase.BeaconBlockBody.Value != nil {
			p := &pb.BeaconBlockBody{}
			if err := testutil.ConvertToPb(testCase.BeaconBlockBody.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.BeaconBlockBody.Root) {
				t.Errorf("Expected beacon block body %#x, received %#x", testCase.BeaconBlockBody.Root, root[:])
			}
		}
		if testCase.BeaconBlockHeader.Value != nil {
			p := &pb.BeaconBlockHeader{}
			if err := testutil.ConvertToPb(testCase.BeaconBlockHeader.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.BeaconBlockHeader.Root) {
				t.Errorf("Expected beacon block header %#x, received %#x", testCase.BeaconBlockHeader.Root, root[:])
			}
		}
		//if testCase.BeaconState.Value != nil {
		//	p := &pb.BeaconState{}
		//	if err := testutil.ConvertToPb(testCase.BeaconState.Value, p); err != nil {
		//		t.Fatal(err)
		//	}
		//	root, err := ssz.HashTreeRoot(p)
		//	if err != nil {
		//		t.Fatal(err)
		//	}
		//	if !bytes.Equal(root[:], testCase.BeaconState.Root) {
		//		t.Errorf("Expected beacon state %#x, received %#x", testCase.BeaconState.Root, root[:])
		//	}
		//}
		if testCase.Crosslink.Value != nil {
			c := &pb.Crosslink{}
			if err := testutil.ConvertToPb(testCase.Crosslink.Value, c); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(c)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Crosslink.Root) {
				t.Errorf("Expected crosslink %#x, received %#x", testCase.Crosslink.Root, root[:])
			}
		}
		if testCase.Deposit.Value != nil {
			p := &pb.Deposit{}
			if err := testutil.ConvertToPb(testCase.Deposit.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Deposit.Root) {
				t.Errorf("Expected deposit root %#x, received %#x", testCase.Deposit.Root, root[:])
			}
		}
		if testCase.DepositData.Value != nil {
			p := &pb.DepositData{}
			if err := testutil.ConvertToPb(testCase.DepositData.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.DepositData.Root) {
				t.Errorf("Expected deposit data %#x, received %#x", testCase.DepositData.Root, root[:])
			}
		}
		if testCase.Eth1Data.Value != nil {
			p := &pb.Eth1Data{}
			if err := testutil.ConvertToPb(testCase.Eth1Data.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Eth1Data.Root) {
				t.Errorf("Expected eth1 data %#x, received %#x", testCase.Eth1Data.Root, root[:])
			}
		}
		if testCase.Fork.Value != nil {
			p := &pb.Fork{}
			if err := testutil.ConvertToPb(testCase.Fork.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Fork.Root) {
				t.Errorf("Expected fork %#x, received %#x", testCase.Fork.Root, root[:])
			}
		}
		if testCase.HistoricalBatch.Value != nil {
			p := &pb.HistoricalBatch{}
			if err := testutil.ConvertToPb(testCase.HistoricalBatch.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.HistoricalBatch.Root) {
				t.Errorf("Expected historical batch %#x, received %#x", testCase.HistoricalBatch.Root, root[:])
			}
		}
		if testCase.IndexedAttestation.Value != nil {
			p := &pb.IndexedAttestation{}
			if err := testutil.ConvertToPb(testCase.IndexedAttestation.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.IndexedAttestation.Root) {
				t.Errorf("Expected indexed attestation %#x, received %#x", testCase.IndexedAttestation.Root, root[:])
			}
		}
		if testCase.PendingAttestation.Value != nil {
			p := &pb.PendingAttestation{}
			if err := testutil.ConvertToPb(testCase.PendingAttestation.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.PendingAttestation.Root) {
				t.Errorf("Expected pending attestation %#x, received %#x", testCase.PendingAttestation.Root, root[:])
			}
		}
		if testCase.ProposerSlashing.Value != nil {
			p := &pb.ProposerSlashing{}
			if err := testutil.ConvertToPb(testCase.ProposerSlashing.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.ProposerSlashing.Root) {
				t.Errorf("Expected proposer slashing %#x, received %#x", testCase.ProposerSlashing.Root, root[:])
			}
		}
		if testCase.Transfer.Value != nil {
			p := &pb.Transfer{}
			if err := testutil.ConvertToPb(testCase.Transfer.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Transfer.Root) {
				t.Errorf("Expected trasnfer %#x, received %#x", testCase.Transfer.Root, root[:])
			}
		}
		if testCase.Validator.Value != nil {
			p := &pb.Validator{}
			if err := testutil.ConvertToPb(testCase.Validator.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.Validator.Root) {
				t.Errorf("Expected trasnfer %#x, received %#x", testCase.Validator.Root, root[:])
			}
		}
		if testCase.VoluntaryExit.Value != nil {
			p := &pb.VoluntaryExit{}
			if err := testutil.ConvertToPb(testCase.VoluntaryExit.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.VoluntaryExit.Root) {
				t.Errorf("Expected voluntary exit %#x, received %#x", testCase.VoluntaryExit.Root, root[:])
			}
		}
	}
}
