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
