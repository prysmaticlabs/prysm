package validator

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func Test_blobsBundleToSidecars(t *testing.T) {
	b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockDeneb())
	require.NoError(t, err)

	b.SetSlot(1)
	b.SetProposerIndex(2)
	b.SetParentRoot(bytesutil.PadTo([]byte("parentRoot"), 32))

	kcs := [][]byte{[]byte("kzg"), []byte("kzg1"), []byte("kzg2")}
	proofs := [][]byte{[]byte("proof"), []byte("proof1"), []byte("proof2")}
	blobs := [][]byte{[]byte("blob"), []byte("blob1"), []byte("blob2")}
	bundle := &enginev1.BlobsBundle{KzgCommitments: kcs, Proofs: proofs, Blobs: blobs}

	sidecars, err := blobsBundleToSidecars(bundle, b.Block())
	require.NoError(t, err)

	r, err := b.Block().HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, len(sidecars), 3)
	for i := 0; i < len(sidecars); i++ {
		require.DeepEqual(t, sidecars[i].BlockRoot, r[:])
		require.Equal(t, sidecars[i].Index, uint64(i))
		require.Equal(t, sidecars[i].Slot, b.Block().Slot())
		pr := b.Block().ParentRoot()
		require.DeepEqual(t, sidecars[i].BlockParentRoot, pr[:])
		require.Equal(t, sidecars[i].ProposerIndex, b.Block().ProposerIndex())
		require.DeepEqual(t, sidecars[i].Blob, blobs[i])
		require.DeepEqual(t, sidecars[i].KzgProof, proofs[i])
		require.DeepEqual(t, sidecars[i].KzgCommitment, kcs[i])
	}
}

func Test_blindBlobsBundleToSidecars(t *testing.T) {
	b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockDeneb())
	require.NoError(t, err)

	b.SetSlot(1)
	b.SetProposerIndex(2)
	b.SetParentRoot(bytesutil.PadTo([]byte("parentRoot"), 32))

	kcs := [][]byte{[]byte("kzg"), []byte("kzg1"), []byte("kzg2")}
	proofs := [][]byte{[]byte("proof"), []byte("proof1"), []byte("proof2")}
	blobRoots := [][]byte{[]byte("blob"), []byte("blob1"), []byte("blob2")}
	bundle := &enginev1.BlindedBlobsBundle{KzgCommitments: kcs, Proofs: proofs, BlobRoots: blobRoots}

	sidecars, err := blindBlobsBundleToSidecars(bundle, b.Block())
	require.NoError(t, err)

	r, err := b.Block().HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, len(sidecars), 3)
	for i := 0; i < len(sidecars); i++ {
		require.DeepEqual(t, sidecars[i].BlockRoot, r[:])
		require.Equal(t, sidecars[i].Index, uint64(i))
		require.Equal(t, sidecars[i].Slot, b.Block().Slot())
		pr := b.Block().ParentRoot()
		require.DeepEqual(t, sidecars[i].BlockParentRoot, pr[:])
		require.Equal(t, sidecars[i].ProposerIndex, b.Block().ProposerIndex())
		require.DeepEqual(t, sidecars[i].BlobRoot, blobRoots[i])
		require.DeepEqual(t, sidecars[i].KzgProof, proofs[i])
		require.DeepEqual(t, sidecars[i].KzgCommitment, kcs[i])
	}
}
