package ssz_static

import (
	"context"
	"encoding/hex"
	"errors"
	"path"
	"testing"

	fssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	state "github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/utils"
)

// SSZRoots --
type SSZRoots struct {
	Root        string `json:"root"`
	SigningRoot string `json:"signing_root"`
}

// RunSSZStaticTests executes "ssz_static" tests.
func RunSSZStaticTests(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, _ := utils.TestFolders(t, config, "merge", "ssz_static")
	for _, folder := range testFolders {
		innerPath := path.Join("ssz_static", folder.Name(), "ssz_random")
		innerTestFolders, innerTestsFolderPath := utils.TestFolders(t, config, "merge", innerPath)
		for _, innerFolder := range innerTestFolders {
			t.Run(path.Join(folder.Name(), innerFolder.Name()), func(t *testing.T) {
				if folder.Name() == "BeaconBlock" || folder.Name() == "BeaconBlockBody" {
					t.Skip()
				}
				serializedBytes, err := testutil.BazelFileBytes(innerTestsFolderPath, innerFolder.Name(), "serialized.ssz_snappy")
				require.NoError(t, err)
				serializedSSZ, err := snappy.Decode(nil /* dst */, serializedBytes)
				require.NoError(t, err, "Failed to decompress")
				object, err := UnmarshalledSSZ(t, serializedSSZ, folder.Name())
				require.NoError(t, err, "Could not unmarshall serialized SSZ")

				rootsYamlFile, err := testutil.BazelFileBytes(innerTestsFolderPath, innerFolder.Name(), "roots.yaml")
				require.NoError(t, err)
				rootsYaml := &SSZRoots{}
				require.NoError(t, utils.UnmarshalYaml(rootsYamlFile, rootsYaml), "Failed to Unmarshal")

				// Custom hash tree root for beacon state.
				var htr func(interface{}) ([32]byte, error)
				if _, ok := object.(*pb.BeaconState); ok {
					htr = func(s interface{}) ([32]byte, error) {
						beaconState, err := state.InitializeFromProto(s.(*pb.BeaconState))
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

				if _, ok := object.(*ethpb.ExecutionPayload); ok {
					htr = func(s interface{}) ([32]byte, error) {
						return HashTreeRootExecutionPayload(s.(*ethpb.ExecutionPayload))
					}
				}

				root, err := htr(object)
				require.NoError(t, err)
				rootBytes, err := hex.DecodeString(rootsYaml.Root[2:])
				require.NoError(t, err)
				assert.DeepEqual(t, rootBytes, root[:], "Did not receive expected hash tree root")

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
				assert.DeepEqual(t, signingRootBytes, signingRoot[:], "Did not receive expected signing root")
			})
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
	case "DepositMessage":
		obj = &pb.DepositMessage{}
	case "DepositData":
		obj = &ethpb.Deposit_Data{}
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
		obj = &ethpb.SignedAggregateAttestationAndProof{}
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
	case "ExecutionPayload":
		obj = &ethpb.ExecutionPayload{}
	case "ExecutionPayloadHeader":
		obj = &pb.ExecutionPayloadHeader{}
	case "ContributionAndProof":
		t.Skip("Unused type")
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

func HashTreeRootExecutionPayload(e *ethpb.ExecutionPayload) ([32]byte, error) {
	hh := fssz.DefaultHasherPool.Get()
	if err := HashTreeRootWithExecutionPayload(hh, e); err != nil {
		fssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	fssz.DefaultHasherPool.Put(hh)
	return root, err
}

func HashTreeRootWithExecutionPayload(hh *fssz.Hasher, e *ethpb.ExecutionPayload) (err error) {
	indx := hh.Index()

	// Field (0) 'BlockHash'
	if len(e.BlockHash) != 32 {
		err = fssz.ErrBytesLength
		return
	}
	hh.PutBytes(e.BlockHash)

	// Field (1) 'ParentHash'
	if len(e.ParentHash) != 32 {
		err = fssz.ErrBytesLength
		return
	}
	hh.PutBytes(e.ParentHash)

	// Field (2) 'Coinbase'
	if len(e.Coinbase) != 20 {
		err = fssz.ErrBytesLength
		return
	}
	hh.PutBytes(e.Coinbase)

	// Field (3) 'StateRoot'
	if len(e.StateRoot) != 32 {
		err = fssz.ErrBytesLength
		return
	}
	hh.PutBytes(e.StateRoot)

	// Field (4) 'Number'
	hh.PutUint64(e.Number)

	// Field (5) 'GasLimit'
	hh.PutUint64(e.GasLimit)

	// Field (6) 'GasUsed'
	hh.PutUint64(e.GasUsed)

	// Field (7) 'Timestamp'
	hh.PutUint64(e.Timestamp)

	// Field (8) 'ReceiptRoot'
	if len(e.ReceiptRoot) != 32 {
		err = fssz.ErrBytesLength
		return
	}
	hh.PutBytes(e.ReceiptRoot)

	// Field (9) 'LogsBloom'
	if len(e.LogsBloom) != 256 {
		err = fssz.ErrBytesLength
		return
	}
	hh.PutBytes(e.LogsBloom)

	// Field (10) 'Transactions'
	{
		subIndx := hh.Index()
		num := uint64(len(e.Transactions))
		if num > 16384 {
			err = fssz.ErrIncorrectListSize
			return
		}
		for i := uint64(0); i < num; i++ {
			txSubIndx := hh.Index()
			hh.PutBytes(e.Transactions[i])
			numItems := uint64(len(e.Transactions[i]))
			hh.MerkleizeWithMixin(txSubIndx, numItems, fssz.CalculateLimit(1048576, numItems, 8))
		}
		hh.MerkleizeWithMixin(subIndx, num, 16384)
	}
	//txRoot, err := htrutils.TransactionsRoot(e.Transactions)
	//if err != nil {
	//	return
	//}
	//hh.PutBytes(txRoot[:])

	hh.Merkleize(indx)
	return
}
