package validator

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

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

func TestAdd(t *testing.T) {
	slot := primitives.Slot(1)
	bundle := &enginev1.BlobsBundle{KzgCommitments: [][]byte{{'a'}}}
	bundleCache.add(slot, bundle)
	require.Equal(t, bundleCache.bundle, bundle)

	slot = primitives.Slot(2)
	bundle = &enginev1.BlobsBundle{KzgCommitments: [][]byte{{'b'}}}
	bundleCache.add(slot, bundle)
	require.Equal(t, bundleCache.bundle, bundle)
}

func TestGet(t *testing.T) {
	slot := primitives.Slot(3)
	bundle := &enginev1.BlobsBundle{KzgCommitments: [][]byte{{'a'}}}
	bundleCache.add(slot, bundle)
	require.Equal(t, bundleCache.get(slot), bundle)
}

func TestPrune(t *testing.T) {
	slot1 := primitives.Slot(4)
	bundle1 := &enginev1.BlobsBundle{KzgCommitments: [][]byte{{'a'}}}

	bundleCache.add(slot1, bundle1)
	bundleCache.prune(slot1 + 1)

	if bundleCache.get(slot1) != nil {
		t.Errorf("Prune did not remove the bundle at slot1")
	}
}

func TestServer_buildBlobSidecars(t *testing.T) {
	kzgCommitments := [][]byte{bytesutil.PadTo([]byte{'a'}, 48), bytesutil.PadTo([]byte{'b'}, 48)}
	bundle := &enginev1.BlobsBundle{
		KzgCommitments: kzgCommitments,
		Proofs:         [][]byte{{0x03}, {0x04}},
		Blobs:          [][]byte{{0x05}, {0x06}},
	}
	bundleCache.add(0, bundle)
	blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockDeneb())
	require.NoError(t, err)
	require.NoError(t, blk.SetBlobKzgCommitments(kzgCommitments))
	scs, err := buildBlobSidecars(blk)
	require.NoError(t, err)
	require.Equal(t, 2, len(scs))

	inclusionProof0, err := blocks.MerkleProofKZGCommitment(blk.Block().Body(), 0)
	require.NoError(t, err)
	require.DeepEqual(t, inclusionProof0, scs[0].CommitmentInclusionProof)

	inclusionProof1, err := blocks.MerkleProofKZGCommitment(blk.Block().Body(), 1)
	require.NoError(t, err)
	require.DeepEqual(t, inclusionProof1, scs[1].CommitmentInclusionProof)
}
