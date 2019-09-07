package testing

import (
	"bytes"
	"encoding/hex"
	"errors"
	"path"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// SSZRoots --
type SSZRoots struct {
	Root        string `json:"root"`
	SigningRoot string `json:"signing_root"`
}

func runSSZStaticTests(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, _ := testutil.TestFolders(t, config, "ssz_static")
	for _, folder := range testFolders {
		innerPath := path.Join("ssz_static", folder.Name(), "ssz_random")
		innerTestFolders, innerTestsFolderPath := testutil.TestFolders(t, config, innerPath)

		for _, innerFolder := range innerTestFolders {
			t.Run(path.Join(folder.Name(), innerFolder.Name()), func(t *testing.T) {
				serializedBytes, err := testutil.BazelFileBytes(innerTestsFolderPath, innerFolder.Name(), "serialized.ssz")
				if err != nil {
					t.Fatal(err)
				}
				object, err := UnmarshalledSSZ(serializedBytes, folder.Name())
				if err != nil {
					t.Fatalf("Could not unmarshall serialized SSZ: %v", err)
				}

				rootsYamlFile, err := testutil.BazelFileBytes(innerTestsFolderPath, innerFolder.Name(), "roots.yaml")
				if err != nil {
					t.Fatal(err)
				}
				rootsYaml := &SSZRoots{}
				if err := testutil.UnmarshalYaml(rootsYamlFile, rootsYaml); err != nil {
					t.Fatalf("Failed to Unmarshal: %v", err)
				}

				root, err := ssz.HashTreeRoot(object)
				if err != nil {
					t.Fatal(err)
				}
				rootBytes, err := hex.DecodeString(rootsYaml.Root[2:])
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(root[:], rootBytes) {
					t.Fatalf(
						"Did not receive expected hash tree root, received: %#x, expected: %#x",
						root[:],
						rootBytes,
					)
				}

				if rootsYaml.SigningRoot == "" {
					return
				}
				signingRoot, err := ssz.SigningRoot(object)
				if err != nil {
					t.Fatal(err)
				}
				signingRootBytes, err := hex.DecodeString(rootsYaml.SigningRoot[2:])
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(signingRoot[:], signingRootBytes) {
					t.Fatalf(
						"Did not receive expected signing root, received: %#x, expected: %#x",
						signingRoot[:],
						signingRootBytes,
					)
				}
			})
		}
	}
}

func UnmarshalledSSZ(serializedBytes []byte, folderName string) (interface{}, error) {
	var obj interface{}
	switch folderName {
	case "Attestation":
		obj = &ethpb.Attestation{}
	case "AttestationData":
		obj = &ethpb.AttestationData{}
	case "AttestationDataAndCustodyBit":
		obj = &pb.AttestationDataAndCustodyBit{}
	case "AttesterSlashing":
		obj = &ethpb.AttesterSlashing{}
	case "BeaconBlock":
		obj = &ethpb.BeaconBlock{}
	case "BeaconBlockBody":
		obj = &ethpb.BeaconBlockBody{}
	case "BeaconBlockHeader":
		obj = &ethpb.BeaconBlockHeader{}
	case "BeaconState":
		obj = &pb.BeaconState{}
	case "Checkpoint":
		obj = &ethpb.Checkpoint{}
	case "CompactCommittee":
		obj = &pb.CompactCommittee{}
	case "Crosslink":
		obj = &ethpb.Crosslink{}
	case "Deposit":
		obj = &ethpb.Deposit{}
	case "DepositData":
		obj = &ethpb.Deposit_Data{}
	case "Eth1Data":
		obj = &ethpb.Eth1Data{}
	case "Fork":
		obj = &pb.Fork{}
	case "HistoricalBatch":
		obj = &pb.HistoricalBatch{}
	case "IndexedAttestation":
		obj = &ethpb.IndexedAttestation{}
	case "PendingAttestation":
		obj = &pb.PendingAttestation{}
	case "ProposerSlashing":
		obj = &ethpb.ProposerSlashing{}
	case "Transfer":
		obj = &ethpb.Transfer{}
	case "Validator":
		obj = &ethpb.Validator{}
	case "VoluntaryExit":
		obj = &ethpb.VoluntaryExit{}
	default:
		return nil, errors.New("type not found")
	}
	err := ssz.Unmarshal(serializedBytes, obj)
	return obj, err
}
