package ssz_static

import (
	"context"
	"encoding/hex"
	"errors"
	"path"
	"testing"

	fssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/testing/util"
)

// SSZRoots --
type SSZRoots struct {
	Root        string `json:"root"`
	SigningRoot string `json:"signing_root"`
}

// RunSSZStaticTests executes "ssz_static" tests.
func RunSSZStaticTests(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, _ := utils.TestFolders(t, config, "altair", "ssz_static")
	for _, folder := range testFolders {
		modePath := path.Join("ssz_static", folder.Name())
		modeFolders, _ := utils.TestFolders(t, config, "altair", modePath)

		for _, modeFolder := range modeFolders {
			innerPath := path.Join(modePath, modeFolder.Name())
			innerTestFolders, innerTestsFolderPath := utils.TestFolders(t, config, "altair", innerPath)

			for _, innerFolder := range innerTestFolders {
				t.Run(path.Join(modeFolder.Name(), folder.Name(), innerFolder.Name()), func(t *testing.T) {
					serializedBytes, err := util.BazelFileBytes(innerTestsFolderPath, innerFolder.Name(), "serialized.ssz_snappy")
					require.NoError(t, err)
					serializedSSZ, err := snappy.Decode(nil /* dst */, serializedBytes)
					require.NoError(t, err, "Failed to decompress")
					object, err := UnmarshalledSSZ(t, serializedSSZ, folder.Name())
					require.NoError(t, err, "Could not unmarshall serialized SSZ")

					rootsYamlFile, err := util.BazelFileBytes(innerTestsFolderPath, innerFolder.Name(), "roots.yaml")
					require.NoError(t, err)
					rootsYaml := &SSZRoots{}
					require.NoError(t, utils.UnmarshalYaml(rootsYamlFile, rootsYaml), "Failed to Unmarshal")

					// Custom hash tree root for beacon state.
					var htr func(interface{}) ([32]byte, error)
					if _, ok := object.(*ethpb.BeaconStateAltair); ok {
						htr = func(s interface{}) ([32]byte, error) {
							beaconState, err := stateAltair.InitializeFromProto(s.(*ethpb.BeaconStateAltair))
							require.NoError(t, err)
							return beaconState.HashTreeRoot(context.Background())
						}
					} else {
						htr = func(s interface{}) ([32]byte, error) {
							sszObj, ok := s.(fssz.HashRoot)
							if !ok {
								return [32]byte{}, errors.New("could not get hash root, not compatible object")
							}
							return sszObj.HashTreeRoot()
						}
					}

					root, err := htr(object)
					require.NoError(t, err)
					rootBytes, err := hex.DecodeString(rootsYaml.Root[2:])
					require.NoError(t, err)
					require.DeepEqual(t, rootBytes, root[:], "Did not receive expected hash tree root")

					if rootsYaml.SigningRoot == "" {
						return
					}

					var signingRoot [32]byte
					if v, ok := object.(fssz.HashRoot); ok {
						signingRoot, err = v.HashTreeRoot()
					} else {
						t.Fatal("object does not meet fssz.HashRoot")
					}

					require.NoError(t, err)
					signingRootBytes, err := hex.DecodeString(rootsYaml.SigningRoot[2:])
					require.NoError(t, err)
					require.DeepEqual(t, signingRootBytes, signingRoot[:], "Did not receive expected signing root")
				})
			}
		}
	}
}

// UnmarshalledSSZ unmarshalls serialized input.
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
		obj = &ethpb.BeaconBlockAltair{}
	case "BeaconBlockBody":
		obj = &ethpb.BeaconBlockBodyAltair{}
	case "BeaconBlockHeader":
		obj = &ethpb.BeaconBlockHeader{}
	case "BeaconState":
		obj = &ethpb.BeaconStateAltair{}
	case "Checkpoint":
		obj = &ethpb.Checkpoint{}
	case "Deposit":
		obj = &ethpb.Deposit{}
	case "DepositMessage":
		obj = &ethpb.DepositMessage{}
	case "DepositData":
		obj = &ethpb.Deposit_Data{}
	case "Eth1Data":
		obj = &ethpb.Eth1Data{}
	case "Eth1Block":
		t.Skip("Unused type")
		return nil, nil
	case "Fork":
		obj = &ethpb.Fork{}
	case "ForkData":
		obj = &ethpb.ForkData{}
	case "HistoricalBatch":
		obj = &ethpb.HistoricalBatch{}
	case "IndexedAttestation":
		obj = &ethpb.IndexedAttestation{}
	case "PendingAttestation":
		obj = &ethpb.PendingAttestation{}
	case "ProposerSlashing":
		obj = &ethpb.ProposerSlashing{}
	case "SignedAggregateAndProof":
		obj = &ethpb.SignedAggregateAttestationAndProof{}
	case "SignedBeaconBlock":
		obj = &ethpb.SignedBeaconBlockAltair{}
	case "SignedBeaconBlockHeader":
		obj = &ethpb.SignedBeaconBlockHeader{}
	case "SignedVoluntaryExit":
		obj = &ethpb.SignedVoluntaryExit{}
	case "SigningData":
		obj = &ethpb.SigningData{}
	case "Validator":
		obj = &ethpb.Validator{}
	case "VoluntaryExit":
		obj = &ethpb.VoluntaryExit{}
	case "SyncCommitteeMessage":
		obj = &ethpb.SyncCommitteeMessage{}
	case "SyncCommitteeContribution":
		obj = &ethpb.SyncCommitteeContribution{}
	case "ContributionAndProof":
		obj = &ethpb.ContributionAndProof{}
	case "SignedContributionAndProof":
		obj = &ethpb.SignedContributionAndProof{}
	case "SyncAggregate":
		obj = &ethpb.SyncAggregate{}
	case "SyncAggregatorSelectionData":
		obj = &ethpb.SyncAggregatorSelectionData{}
	case "SyncCommittee":
		obj = &ethpb.SyncCommittee{}
	case "LightClientSnapshot":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	case "LightClientUpdate":
		t.Skip("not a beacon node type, this is a light node type")
		return nil, nil
	default:
		return nil, errors.New("type not found")
	}
	var err error
	if o, ok := obj.(fssz.Unmarshaler); ok {
		err = o.UnmarshalSSZ(serializedBytes)
	} else {
		err = errors.New("could not unmarshal object, not a fastssz compatible object")
	}
	return obj, err
}
