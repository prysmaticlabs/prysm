package eth

import (
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
)

// CopySignedBeaconBlockHeze copies the provided signed beacon block Heze object.
func CopySignedBeaconBlockHeze(sb *SignedBeaconBlockHeze) *SignedBeaconBlockHeze {
	if sb == nil {
		return nil
	}
	return &SignedBeaconBlockHeze{
		Block:     copyBeaconBlockHeze(sb.Block),
		Signature: bytesutil.SafeCopyBytes(sb.Signature),
	}
}

// copyBeaconBlockHeze copies the provided beacon block Heze object.
func copyBeaconBlockHeze(b *BeaconBlockHeze) *BeaconBlockHeze {
	if b == nil {
		return nil
	}
	return &BeaconBlockHeze{
		Slot:          b.Slot,
		ProposerIndex: b.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(b.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(b.StateRoot),
		Body:          copyBeaconBlockBodyHeze(b.Body),
	}
}

// copyBeaconBlockBodyHeze copies the provided beacon block body Heze object.
func copyBeaconBlockBodyHeze(body *BeaconBlockBodyHeze) *BeaconBlockBodyHeze {
	if body == nil {
		return nil
	}
	copied := &BeaconBlockBodyHeze{
		RandaoReveal:              bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:                  body.Eth1Data.Copy(),
		Graffiti:                  bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:         CopySlice(body.ProposerSlashings),
		AttesterSlashings:         CopySlice(body.AttesterSlashings),
		Attestations:              CopySlice(body.Attestations),
		Deposits:                  CopySlice(body.Deposits),
		VoluntaryExits:            CopySlice(body.VoluntaryExits),
		BlsToExecutionChanges:     CopySlice(body.BlsToExecutionChanges),
		SignedExecutionPayloadBid: copySignedExecutionPayloadBidHeze(body.SignedExecutionPayloadBid),
		PayloadAttestations:       copyPayloadAttestations(body.PayloadAttestations),
		ParentExecutionRequests:   CopyExecutionRequestsGloas(body.ParentExecutionRequests),
	}
	if body.SyncAggregate != nil {
		copied.SyncAggregate = body.SyncAggregate.Copy()
	}
	return copied
}

// Copy creates a deep copy of ExecutionPayloadBidHeze.
func (header *ExecutionPayloadBidHeze) Copy() *ExecutionPayloadBidHeze {
	if header == nil {
		return nil
	}
	return &ExecutionPayloadBidHeze{
		ParentBlockHash:       bytesutil.SafeCopyBytes(header.ParentBlockHash),
		ParentBlockRoot:       bytesutil.SafeCopyBytes(header.ParentBlockRoot),
		BlockHash:             bytesutil.SafeCopyBytes(header.BlockHash),
		PrevRandao:            bytesutil.SafeCopyBytes(header.PrevRandao),
		FeeRecipient:          bytesutil.SafeCopyBytes(header.FeeRecipient),
		GasLimit:              header.GasLimit,
		BuilderIndex:          header.BuilderIndex,
		Slot:                  header.Slot,
		Value:                 header.Value,
		ExecutionPayment:      header.ExecutionPayment,
		BlobKzgCommitments:    bytesutil.SafeCopy2dBytes(header.BlobKzgCommitments),
		ExecutionRequestsRoot: bytesutil.SafeCopyBytes(header.ExecutionRequestsRoot),
		PayloadChunksRoot:     bytesutil.SafeCopyBytes(header.PayloadChunksRoot),
	}
}

// copySignedExecutionPayloadBidHeze copies the provided signed Heze execution payload bid.
func copySignedExecutionPayloadBidHeze(header *SignedExecutionPayloadBidHeze) *SignedExecutionPayloadBidHeze {
	if header == nil {
		return nil
	}
	return &SignedExecutionPayloadBidHeze{
		Message:   header.Message.Copy(),
		Signature: bytesutil.SafeCopyBytes(header.Signature),
	}
}
