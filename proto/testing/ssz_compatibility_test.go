package testing

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestYaml(t *testing.T) {
	file, err := ioutil.ReadFile("ssz_minimal_one.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &SszMinimalTest{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range s.TestCases {
		if !isEmpty(testCase.Attestation.Value) {
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
		if !isEmpty(testCase.AttestationData.Value) {
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
		if !isEmpty(testCase.AttestationDataAndCustodyBit.Value) {
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
		if !isEmpty(testCase.AttesterSlashing.Value) {
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
		if !isEmpty(testCase.BeaconBlock.Value) {
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
		if !isEmpty(testCase.BeaconBlockBody.Value) {
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
		if !isEmpty(testCase.BeaconBlockHeader.Value) {
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
		if !isEmpty(testCase.BeaconState.Value) {
			p := &pb.BeaconState{}
			if err := testutil.ConvertToPb(testCase.BeaconState.Value, p); err != nil {
				t.Fatal(err)
			}
			root, err := ssz.HashTreeRoot(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(root[:], testCase.BeaconState.Root) {
				t.Errorf("Expected beacon state %#x, received %#x", testCase.BeaconState.Root, root[:])
			}
		}
		if !isEmpty(testCase.Crosslink.Value) {
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
		if !isEmpty(testCase.Deposit.Value) {
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
		if !isEmpty(testCase.DepositData.Value) {
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
		if !isEmpty(testCase.Eth1Data.Value) {
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
		if !isEmpty(testCase.Fork.Value) {
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
		if !isEmpty(testCase.HistoricalBatch.Value) {
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
		if !isEmpty(testCase.IndexedAttestation.Value) {
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
		if !isEmpty(testCase.PendingAttestation.Value) {
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
		if !isEmpty(testCase.ProposerSlashing.Value) {
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
		if !isEmpty(testCase.Transfer.Value) {
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
		if !isEmpty(testCase.Validator.Value) {
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
		if !isEmpty(testCase.VoluntaryExit.Value) {
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

func isEmpty(item interface{}) bool {
	val := reflect.ValueOf(item)
	for i := 0; i < val.NumField(); i++ {
		if !reflect.DeepEqual(val.Field(i).Interface(), reflect.Zero(val.Field(i).Type()).Interface()) {
			return false
		}
	}
	return true
}
