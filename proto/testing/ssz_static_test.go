package testing

import (
	"bytes"
	"encoding/hex"
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
	if folderName == "Attestation" {
		obj = &ethpb.Attestation{}
	} else if folderName == "AttestationData" {
		obj = &ethpb.AttestationData{}
	} else if folderName == "AttestationDataAndCustodyBit" {
		obj = &pb.AttestationDataAndCustodyBit{}
	} else if folderName == "AttesterSlashing" {
		obj = &ethpb.AttesterSlashing{}
	} else if folderName == "BeaconBlock" {
		obj = &ethpb.BeaconBlock{}
	} else if folderName == "BeaconBlockBody" {
		obj = &ethpb.BeaconBlockBody{}
	} else if folderName == "BeaconBlockHeader" {
		obj = &ethpb.BeaconBlockHeader{}
	} else if folderName == "BeaconState" {
		obj = &pb.BeaconState{}
	} else if folderName == "Checkpoint" {
		obj = &ethpb.Checkpoint{}
	} else if folderName == "CompactCommittee" {
		obj = &pb.CompactCommittee{}
	} else if folderName == "Crosslink" {
		obj = &ethpb.Crosslink{}
	} else if folderName == "Deposit" {
		obj = &ethpb.Deposit{}
	} else if folderName == "DepositData" {
		obj = &ethpb.Deposit_Data{}
	} else if folderName == "Eth1Data" {
		obj = &ethpb.Eth1Data{}
	} else if folderName == "Fork" {
		obj = &pb.Fork{}
	} else if folderName == "HistoricalBatch" {
		obj = &pb.HistoricalBatch{}
	} else if folderName == "IndexedAttestation" {
		obj = &ethpb.IndexedAttestation{}
	} else if folderName == "PendingAttestation" {
		obj = &pb.PendingAttestation{}
	} else if folderName == "ProposerSlashing" {
		obj = &ethpb.ProposerSlashing{}
	} else if folderName == "Transfer" {
		obj = &ethpb.Transfer{}
	} else if folderName == "Validator" {
		obj = &ethpb.Validator{}
	} else if folderName == "VoluntaryExit" {
		obj = &ethpb.VoluntaryExit{}
	}
	var err = ssz.Unmarshal(serializedBytes, obj)
	return obj, err
}
