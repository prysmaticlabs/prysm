package eth

import (
	"slices"

	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
)

// Copy deep copies the checkpoint.
func (c *HeightCheckpoint) Copy() *HeightCheckpoint {
	if c == nil {
		return nil
	}
	return &HeightCheckpoint{Height: c.Height, Root: bytesutil.SafeCopyBytes(c.Root)}
}

// Copy deep copies the pending finality.
func (p *PendingFinalityEIP8453) Copy() *PendingFinalityEIP8453 {
	if p == nil {
		return nil
	}
	return &PendingFinalityEIP8453{
		Checkpoint:    p.Checkpoint.Copy(),
		Slot:          p.Slot,
		Participation: bytesutil.SafeCopyBytes(p.Participation),
	}
}

// Copy deep copies the data.
func (d *HeadVoteData) Copy() *HeadVoteData {
	if d == nil {
		return nil
	}
	return &HeadVoteData{Slot: d.Slot, BeaconBlockRoot: bytesutil.SafeCopyBytes(d.BeaconBlockRoot)}
}

// Copy deep copies the head vote.
func (v *HeadVote) Copy() *HeadVote {
	if v == nil {
		return nil
	}
	return &HeadVote{
		AggregationBits: bytesutil.SafeCopyBytes(v.AggregationBits),
		Data:            v.Data.Copy(),
		Signature:       bytesutil.SafeCopyBytes(v.Signature),
	}
}

// Copy deep copies the data.
func (a *AttestationDataEIP8453) Copy() *AttestationDataEIP8453 {
	if a == nil {
		return nil
	}
	return &AttestationDataEIP8453{Target: a.Target.Copy(), Timeout: a.Timeout, Finality: a.Finality.Copy()}
}

// Copy deep copies the attestation.
func (a *AttestationEIP8453) Copy() *AttestationEIP8453 {
	if a == nil {
		return nil
	}
	return &AttestationEIP8453{
		AggregationBits: bytesutil.SafeCopyBytes(a.AggregationBits),
		Data:            a.Data.Copy(),
		Signature:       bytesutil.SafeCopyBytes(a.Signature),
		CommitteeBits:   bytesutil.SafeCopyBytes(a.CommitteeBits),
		Epoch:           a.Epoch,
	}
}

// Copy deep copies the indexed attestation.
func (a *IndexedAttestationEIP8453) Copy() *IndexedAttestationEIP8453 {
	if a == nil {
		return nil
	}
	return &IndexedAttestationEIP8453{
		AttestingIndices: slices.Clone(a.AttestingIndices),
		Data:             a.Data.Copy(),
		Signature:        bytesutil.SafeCopyBytes(a.Signature),
		Epoch:            a.Epoch,
	}
}

// Copy deep copies the slashing.
func (s *AttesterSlashingEIP8453) Copy() *AttesterSlashingEIP8453 {
	if s == nil {
		return nil
	}
	return &AttesterSlashingEIP8453{Attestation_1: s.Attestation_1.Copy(), Attestation_2: s.Attestation_2.Copy()}
}

// Copy deep copies the bid.
func (b *ExecutionPayloadBidHeze) Copy() *ExecutionPayloadBidHeze {
	if b == nil {
		return nil
	}
	return &ExecutionPayloadBidHeze{
		ParentBlockHash:       bytesutil.SafeCopyBytes(b.ParentBlockHash),
		ParentBlockRoot:       bytesutil.SafeCopyBytes(b.ParentBlockRoot),
		BlockHash:             bytesutil.SafeCopyBytes(b.BlockHash),
		PrevRandao:            bytesutil.SafeCopyBytes(b.PrevRandao),
		FeeRecipient:          bytesutil.SafeCopyBytes(b.FeeRecipient),
		GasLimit:              b.GasLimit,
		BuilderIndex:          b.BuilderIndex,
		Slot:                  b.Slot,
		Value:                 b.Value,
		ExecutionPayment:      b.ExecutionPayment,
		BlobKzgCommitments:    bytesutil.SafeCopy2dBytes(b.BlobKzgCommitments),
		ExecutionRequestsRoot: bytesutil.SafeCopyBytes(b.ExecutionRequestsRoot),
		InclusionListBits:     bytesutil.SafeCopyBytes(b.InclusionListBits),
	}
}

// Gloas returns the bid without its inclusion list bits, sharing the byte slices.
func (b *ExecutionPayloadBidHeze) Gloas() *ExecutionPayloadBid {
	if b == nil {
		return nil
	}
	return &ExecutionPayloadBid{
		ParentBlockHash:       b.ParentBlockHash,
		ParentBlockRoot:       b.ParentBlockRoot,
		BlockHash:             b.BlockHash,
		PrevRandao:            b.PrevRandao,
		FeeRecipient:          b.FeeRecipient,
		GasLimit:              b.GasLimit,
		BuilderIndex:          b.BuilderIndex,
		Slot:                  b.Slot,
		Value:                 b.Value,
		ExecutionPayment:      b.ExecutionPayment,
		BlobKzgCommitments:    b.BlobKzgCommitments,
		ExecutionRequestsRoot: b.ExecutionRequestsRoot,
	}
}

// HezeBid adds inclusion list bits to a Gloas bid, sharing the byte slices.
func HezeBid(b *ExecutionPayloadBid, inclusionListBits []byte) *ExecutionPayloadBidHeze {
	if b == nil {
		return nil
	}
	return &ExecutionPayloadBidHeze{
		ParentBlockHash:       b.ParentBlockHash,
		ParentBlockRoot:       b.ParentBlockRoot,
		BlockHash:             b.BlockHash,
		PrevRandao:            b.PrevRandao,
		FeeRecipient:          b.FeeRecipient,
		GasLimit:              b.GasLimit,
		BuilderIndex:          b.BuilderIndex,
		Slot:                  b.Slot,
		Value:                 b.Value,
		ExecutionPayment:      b.ExecutionPayment,
		BlobKzgCommitments:    b.BlobKzgCommitments,
		ExecutionRequestsRoot: b.ExecutionRequestsRoot,
		InclusionListBits:     inclusionListBits,
	}
}

// Copy deep copies the signed bid.
func (b *SignedExecutionPayloadBidHeze) Copy() *SignedExecutionPayloadBidHeze {
	if b == nil {
		return nil
	}
	return &SignedExecutionPayloadBidHeze{Message: b.Message.Copy(), Signature: bytesutil.SafeCopyBytes(b.Signature)}
}

// CopySignedBeaconBlockEIP8453 deep copies the signed block.
func CopySignedBeaconBlockEIP8453(sb *SignedBeaconBlockEIP8453) *SignedBeaconBlockEIP8453 {
	if sb == nil {
		return nil
	}
	return &SignedBeaconBlockEIP8453{Block: CopyBeaconBlockEIP8453(sb.Block), Signature: bytesutil.SafeCopyBytes(sb.Signature)}
}

// CopyBeaconBlockEIP8453 deep copies the block.
func CopyBeaconBlockEIP8453(b *BeaconBlockEIP8453) *BeaconBlockEIP8453 {
	if b == nil {
		return nil
	}
	return &BeaconBlockEIP8453{
		Slot:          b.Slot,
		ProposerIndex: b.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(b.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(b.StateRoot),
		Body:          CopyBeaconBlockBodyEIP8453(b.Body),
	}
}

// CopyBeaconBlockBodyEIP8453 deep copies the body.
func CopyBeaconBlockBodyEIP8453(body *BeaconBlockBodyEIP8453) *BeaconBlockBodyEIP8453 {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyEIP8453{
		RandaoReveal:              bytesutil.SafeCopyBytes(body.RandaoReveal),
		Graffiti:                  bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:         CopySlice(body.ProposerSlashings),
		AttesterSlashings:         CopySlice(body.AttesterSlashings),
		Attestations:              CopySlice(body.Attestations),
		VoluntaryExits:            CopySlice(body.VoluntaryExits),
		SyncAggregate:             body.SyncAggregate.Copy(),
		BlsToExecutionChanges:     CopySlice(body.BlsToExecutionChanges),
		SignedExecutionPayloadBid: body.SignedExecutionPayloadBid.Copy(),
		PayloadAttestations:       copyPayloadAttestations(body.PayloadAttestations),
		ParentExecutionRequests:   CopyExecutionRequestsGloas(body.ParentExecutionRequests),
		HeadVotes:                 CopySlice(body.HeadVotes),
		HeadVoteSupportBits:       bytesutil.SafeCopyBytes(body.HeadVoteSupportBits),
		LegacyAttesterSlashings:   CopySlice(body.LegacyAttesterSlashings),
	}
}
