package blocks

import (
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/gohashtree"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	consensus_blocks "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/container/trie"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func Test_MerkleProofKZGCommitment(t *testing.T) {
	kzgs := make([][]byte, 3)
	kzgs[0] = make([]byte, 48)
	_, err := rand.Read(kzgs[0])
	require.NoError(t, err)
	kzgs[1] = make([]byte, 48)
	_, err = rand.Read(kzgs[1])
	require.NoError(t, err)
	kzgs[2] = make([]byte, 48)
	_, err = rand.Read(kzgs[2])
	require.NoError(t, err)
	pbBody := &ethpb.BeaconBlockBodyDeneb{
		SyncAggregate: &ethpb.SyncAggregate{
			SyncCommitteeBits:      make([]byte, fieldparams.SyncAggregateSyncCommitteeBytesLength),
			SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
		},
		ExecutionPayload: &enginev1.ExecutionPayloadDeneb{
			ParentHash:    make([]byte, fieldparams.RootLength),
			FeeRecipient:  make([]byte, 20),
			StateRoot:     make([]byte, fieldparams.RootLength),
			ReceiptsRoot:  make([]byte, fieldparams.RootLength),
			LogsBloom:     make([]byte, 256),
			PrevRandao:    make([]byte, fieldparams.RootLength),
			BaseFeePerGas: make([]byte, fieldparams.RootLength),
			BlockHash:     make([]byte, fieldparams.RootLength),
			Transactions:  make([][]byte, 0),
			ExtraData:     make([]byte, 0),
		},
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, fieldparams.RootLength),
			BlockHash:   make([]byte, fieldparams.RootLength),
		},
		BlobKzgCommitments: kzgs,
	}

	body, err := consensus_blocks.NewBeaconBlockBody(pbBody)
	require.NoError(t, err)
	index := 1
	proof, err := MerkleProofKZGCommitment(body, index)
	require.NoError(t, err)

	chunk := make([][32]byte, 2)
	copy(chunk[0][:], kzgs[index])
	copy(chunk[1][:], kzgs[index][32:])
	gohashtree.HashChunks(chunk, chunk)
	root, err := body.HashTreeRoot()
	require.NoError(t, err)
	kzgOffset := 54 * fieldparams.MaxBlobCommitmentsPerBlock
	require.Equal(t, true, trie.VerifyMerkleProof(root[:], chunk[0][:], uint64(index+kzgOffset), proof))
}
