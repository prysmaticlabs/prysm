package tree

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"github.com/protolambda/ztyp/view"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/io/file"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestProof_SimpleField(t *testing.T) {
	data, err := file.ReadFileAsBytes("/tmp/state.ssz")
	require.NoError(t, err)

	dec := codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data)))
	ztypBeaconState, err := BeaconStateAltairType.Deserialize(dec)
	require.NoError(t, err)
	ztypBeaconCont := ztypBeaconState.(*view.ContainerView)

	// Get a proof of the genesis time.
	proof, gIndex, err := Proof(ztypBeaconState, 0 /* genesis time */)
	require.NoError(t, err)

	hFn := tree.GetHashFn()
	root := ztypBeaconCont.HashTreeRoot(hFn)
	leaf, err := ztypBeaconCont.Backing().Getter(gIndex)
	require.NoError(t, err)

	// Verify the Merkle proof using the state root, leaf for the genesis time,
	// and the generalized index of the genesis time in the state.
	valid := VerifyProof(root, proof, leaf.MerkleRoot(hFn), gIndex)
	require.Equal(t, true, valid)
}

func TestProof_FinalizedCheckpoint(t *testing.T) {
	data, err := file.ReadFileAsBytes("/tmp/state.ssz")
	require.NoError(t, err)

	dec := codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data)))
	ztypBeaconState, err := BeaconStateAltairType.Deserialize(dec)
	require.NoError(t, err)
	ztypBeaconCont := ztypBeaconState.(*view.ContainerView)

	// Get a proof of the finalized checkpoint.
	proof, gIndex, err := Proof(ztypBeaconState, 20 /* finalized checkpoint */)
	require.NoError(t, err)

	hFn := tree.GetHashFn()
	root := ztypBeaconCont.HashTreeRoot(hFn)
	leaf, err := ztypBeaconCont.Backing().Getter(gIndex)
	require.NoError(t, err)

	// Verify the Merkle proof using the state root, leaf for the finalized checkpoint,
	// and the generalized index of the field in the state.
	valid := VerifyProof(root, proof, leaf.MerkleRoot(hFn), gIndex)
	require.Equal(t, true, valid)
}

func TestPrysmSSZComparison(t *testing.T) {
	data, err := file.ReadFileAsBytes("/tmp/state.ssz")
	require.NoError(t, err)

	protoState := &ethpb.BeaconStateAltair{}
	require.NoError(t, protoState.UnmarshalSSZ(data))
	prysmBeaconState, err := stateAltair.InitializeFromProto(protoState)
	require.NoError(t, err)

	dec := codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data)))
	ztypBeaconState, err := BeaconStateAltairType.Deserialize(dec)
	require.NoError(t, err)
	hFn := tree.GetHashFn()
	ztypItem := ztypBeaconState.(*view.ContainerView)
	ztypRoot := ztypItem.HashTreeRoot(hFn)
	prysmRoot, err := prysmBeaconState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	require.Equal(
		t,
		fmt.Sprintf("%#x", prysmRoot),
		ztypRoot.String(),
	)
}
