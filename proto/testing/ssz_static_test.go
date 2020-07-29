package testing

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"path"
	"testing"

	fssz "github.com/ferranbt/fastssz"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// SSZRoots --
type SSZRoots struct {
	Root        string `json:"root"`
	SigningRoot string `json:"signing_root"`
}

func runSSZStaticTests(t *testing.T, config string) {
	if err := spectest.SetConfig(t, config); err != nil {
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
				object, err := UnmarshalledSSZ(t, serializedBytes, folder.Name())
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

				// Custom hash tree root for beacon state.
				var htr func(interface{}) ([32]byte, error)
				if _, ok := object.(*pb.BeaconState); ok {
					htr = func(s interface{}) ([32]byte, error) {
						beaconState, err := state.InitializeFromProto(s.(*pb.BeaconState))
						require.NoError(t, err)
						return beaconState.HashTreeRoot(context.Background())
					}
				} else {
					htr = ssz.HashTreeRoot
				}

				root, err := htr(object)
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

				var signingRoot [32]byte
				if v, ok := object.(fssz.HashRoot); ok {
					signingRoot, err = v.HashTreeRoot()
				} else {
					t.Fatal("object does not meet fssz.HashRoot")
				}

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

func UnmarshalledSSZ(t *testing.T, serializedBytes []byte, folderName string) (interface{}, error) {
	var obj interface{}
	switch folderName {
	case "Attestation":
		obj = &ethpb.Attestation{}
	case "AttestationData":
		obj = &ethpb.AttestationData{}
	case "AttesterSlashing":
		obj = &ethpb.AttesterSlashing{}
	case "AggregateAndProof":
		obj = &ethpb.AggregateAttestationAndProof{}
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
	case "Deposit":
		obj = &ethpb.Deposit{}
	case "DepositData":
		obj = &ethpb.Deposit_Data{}
	case "DepositMessage":
		t.Skip("Unused type")
		return nil, nil
	case "Eth1Data":
		obj = &ethpb.Eth1Data{}
	case "Eth1Block":
		t.Skip("Unused type")
		return nil, nil
	case "Fork":
		obj = &pb.Fork{}
	case "ForkData":
		obj = &pb.ForkData{}
	case "HistoricalBatch":
		obj = &pb.HistoricalBatch{}
	case "IndexedAttestation":
		obj = &ethpb.IndexedAttestation{}
	case "PendingAttestation":
		obj = &pb.PendingAttestation{}
	case "ProposerSlashing":
		obj = &ethpb.ProposerSlashing{}
	case "SignedAggregateAndProof":
		obj = &pb.SignedAggregateAndProof{}
	case "SignedBeaconBlock":
		obj = &ethpb.SignedBeaconBlock{}
	case "SignedBeaconBlockHeader":
		obj = &ethpb.SignedBeaconBlockHeader{}
	case "SignedVoluntaryExit":
		obj = &ethpb.SignedVoluntaryExit{}
	case "SigningData":
		obj = &pb.SigningData{}
	case "Validator":
		obj = &ethpb.Validator{}
	case "VoluntaryExit":
		obj = &ethpb.VoluntaryExit{}
	default:
		return nil, errors.New("type not found")
	}
	var err error
	if o, ok := obj.(fssz.Unmarshaler); ok {
		err = o.UnmarshalSSZ(serializedBytes)
	} else {
		err = ssz.Unmarshal(serializedBytes, obj)
	}
	return obj, err
}
