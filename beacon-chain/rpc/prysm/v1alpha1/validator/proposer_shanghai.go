package validator

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/sirupsen/logrus"
)

func (vs *Server) getShanghaiBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockAndBlobs, error) {
	bellatrixBlk, err := vs.getBellatrixBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}
	payload, err := vs.getExecutionPayload(ctx, req.Slot)
	if err != nil {
		return nil, err
	}
	blobs, err := vs.ExecutionEngineCaller.GetBlobs(ctx, [8]byte{})
	if err != nil {
		return nil, err
	}
	blobTxs, err := mockBlobTransactions(256)
	if err != nil {
		return nil, err
	}
	_ = blobTxs
	// TODO: Save blobs, broadcast, and convert to KZGs
	log.WithFields(logrus.Fields{
		"hash":       fmt.Sprintf("%#x", payload.BlockHash),
		"parentHash": fmt.Sprintf("%#x", payload.ParentHash),
		"number":     payload.BlockNumber,
		"txCount":    len(payload.Transactions),
		"blobCount":  len(blobs.Blob),
	}).Info("Received payload")

	blk := &ethpb.BeaconBlockWithBlobKZGs{
		Slot:          bellatrixBlk.Slot,
		ProposerIndex: bellatrixBlk.ProposerIndex,
		ParentRoot:    bellatrixBlk.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyWithBlobKZGs{
			RandaoReveal:      bellatrixBlk.Body.RandaoReveal,
			Eth1Data:          bellatrixBlk.Body.Eth1Data,
			Graffiti:          bellatrixBlk.Body.Graffiti,
			ProposerSlashings: bellatrixBlk.Body.ProposerSlashings,
			AttesterSlashings: bellatrixBlk.Body.AttesterSlashings,
			Attestations:      bellatrixBlk.Body.Attestations,
			Deposits:          bellatrixBlk.Body.Deposits,
			VoluntaryExits:    bellatrixBlk.Body.VoluntaryExits,
			SyncAggregate:     bellatrixBlk.Body.SyncAggregate,
			BlobKzgs:          nil, // TODO: Add blob KZGs here.
			ExecutionPayload:  payload,
		},
	}
	// Compute state root with the newly constructed block.
	wsb, err := wrapper.WrappedSignedBeaconBlock(
		&ethpb.SignedBeaconBlockAndBlobs{
			Block: &ethpb.SignedBeaconBlockWithBlobKZGs{
				Block:     blk,
				Signature: make([]byte, 96),
			},
			Blobs: []*ethpb.Blob{blobs},
		},
	)
	if err != nil {
		return nil, err
	}
	stateRoot, err := vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot
	blockWithBlobs := &ethpb.BeaconBlockAndBlobs{
		Block: blk,
		Blobs: []*ethpb.Blob{blobs},
	}
	return blockWithBlobs, nil
}

// Returns a list of SSZ-encoded,
func mockBlobTransactions(numItems uint64) ([][]byte, error) {
	txs := make([][]byte, numItems) // TODO: Add some mock txs.
	foo := [32]byte{1}
	addr := [20]byte{}
	for i := uint64(0); i < numItems; i++ {
		blobTx := &pb.SignedBlobTransaction{
			Header: &pb.BlobTransaction{
				Nonce:               i,
				Gas:                 1,
				MaxBasefee:          foo[:],
				PriorityFee:         foo[:],
				Address:             addr[:],
				Value:               foo[:],
				Data:                []byte("foo"),
				BlobVersionedHashes: [][]byte{foo[:]},
			},
			Signatures: &pb.ECDSASignature{
				V: []byte{1},
				R: make([]byte, 32),
				S: make([]byte, 32),
			},
		}
		enc, err := blobTx.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		prefixByte := "0x05"
		prefixByteEnc, err := hexutil.Decode(prefixByte)
		if err != nil {
			return nil, err
		}
		txs[i] = append(prefixByteEnc, enc...)
	}
	return txs, nil
}
