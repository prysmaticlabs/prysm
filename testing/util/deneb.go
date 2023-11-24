package util

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func GenerateTestDenebBlockWithSidecar(t *testing.T, parent [32]byte, slot primitives.Slot, nblobs int) (blocks.ROBlock, []blocks.ROBlob) {
	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
	receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
	logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
	parentHash := bytesutil.PadTo([]byte("parentHash"), fieldparams.RootLength)
	tx := gethTypes.NewTransaction(
		0,
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), 0, big.NewInt(0),
		nil,
	)
	txs := []*gethTypes.Transaction{tx}
	encodedBinaryTxs := make([][]byte, 1)
	var err error
	encodedBinaryTxs[0], err = txs[0].MarshalBinary()
	require.NoError(t, err)
	blockHash := bytesutil.ToBytes32([]byte("foo"))
	payload := &enginev1.ExecutionPayloadDeneb{
		ParentHash:    parentHash,
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    blockHash[:],
		BlockNumber:   0,
		GasLimit:      0,
		GasUsed:       0,
		Timestamp:     0,
		ExtraData:     make([]byte, 0),
		BaseFeePerGas: bytesutil.PadTo([]byte("baseFeePerGas"), fieldparams.RootLength),
		ExcessBlobGas: 0,
		BlobGasUsed:   0,
		BlockHash:     blockHash[:],
		Transactions:  encodedBinaryTxs,
	}
	block := NewBeaconBlockDeneb()
	block.Block.Body.ExecutionPayload = payload
	block.Block.Slot = slot
	block.Block.ParentRoot = parent[:]
	commitments := make([][48]byte, nblobs)
	block.Block.Body.BlobKzgCommitments = make([][]byte, nblobs)
	for i := range commitments {
		binary.LittleEndian.PutUint16(commitments[i][0:16], uint16(i))
		binary.LittleEndian.PutUint16(commitments[i][16:32], uint16(slot))
		block.Block.Body.BlobKzgCommitments[i] = commitments[i][:]
	}

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	sidecars := make([]blocks.ROBlob, len(commitments))
	sbb, err := blocks.NewSignedBeaconBlock(block)
	require.NoError(t, err)
	sh, err := sbb.Header()
	require.NoError(t, err)
	for i, c := range block.Block.Body.BlobKzgCommitments {
		sidecars[i] = GenerateTestDenebBlobSidecar(t, root, sh, i, c)
	}

	rob, err := blocks.NewROBlock(sbb)
	require.NoError(t, err)
	return rob, sidecars
}

func GenerateTestDenebBlobSidecar(t *testing.T, root [32]byte, header *ethpb.SignedBeaconBlockHeader, index int, commitment []byte) blocks.ROBlob {
	blob := make([]byte, fieldparams.BlobSize)
	binary.LittleEndian.PutUint64(blob, uint64(index))
	pb := &ethpb.BlobSidecar{
		SignedBlockHeader: header,
		Index:             uint64(index),
		Blob:              blob,
		KzgCommitment:     commitment,
		KzgProof:          commitment,
	}
	pb.CommitmentInclusionProof = fakeEmptyProof(t, pb)
	r, err := blocks.NewROBlobWithRoot(pb, root)
	require.NoError(t, err)
	return r
}

func fakeEmptyProof(_ *testing.T, _ *ethpb.BlobSidecar) [][]byte {
	r := make([][]byte, fieldparams.KzgCommitmentInclusionProofDepth)
	for i := range r {
		r[i] = make([]byte, fieldparams.RootLength)
	}
	return r
}

func GenerateTestDeprecatedBlobSidecar(root [32]byte, block *ethpb.SignedBeaconBlockDeneb, index int, commitment []byte) *ethpb.DeprecatedBlobSidecar {
	blob := make([]byte, fieldparams.BlobSize)
	binary.LittleEndian.PutUint64(blob, uint64(index))
	pb := &ethpb.DeprecatedBlobSidecar{
		BlockRoot:       root[:],
		Index:           uint64(index),
		Slot:            block.Block.Slot,
		BlockParentRoot: block.Block.ParentRoot,
		ProposerIndex:   block.Block.ProposerIndex,
		Blob:            blob,
		KzgCommitment:   commitment,
		KzgProof:        commitment,
	}
	return pb
}

func ExtendBlocksPlusBlobs(t *testing.T, blks []blocks.ROBlock, size int) ([]blocks.ROBlock, []blocks.ROBlob) {
	blobs := make([]blocks.ROBlob, 0)
	if len(blks) == 0 {
		blk, blb := GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 0, 6)
		blobs = append(blobs, blb...)
		blks = append(blks, blk)
	}

	for i := 0; i < size; i++ {
		prev := blks[len(blks)-1]
		blk, blb := GenerateTestDenebBlockWithSidecar(t, prev.Root(), prev.Block().Slot()+1, 6)
		blobs = append(blobs, blb...)
		blks = append(blks, blk)
	}

	return blks, blobs
}
