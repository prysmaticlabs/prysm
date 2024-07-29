package eth

import (
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

type copier[T any] interface {
	Copy() T
}

func CopySlice[T any, C copier[T]](original []C) []T {
	// Create a new slice with the same length as the original
	newSlice := make([]T, len(original))
	for i := 0; i < len(newSlice); i++ {
		newSlice[i] = original[i].Copy()
	}
	return newSlice
}

// CopySignedBeaconBlock copies the provided SignedBeaconBlock.
func CopySignedBeaconBlock(sigBlock *SignedBeaconBlock) *SignedBeaconBlock {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlock{
		Block:     CopyBeaconBlock(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlock copies the provided BeaconBlock.
func CopyBeaconBlock(block *BeaconBlock) *BeaconBlock {
	if block == nil {
		return nil
	}
	return &BeaconBlock{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBody(block.Body),
	}
}

// CopyBeaconBlockBody copies the provided BeaconBlockBody.
func CopyBeaconBlockBody(body *BeaconBlockBody) *BeaconBlockBody {
	if body == nil {
		return nil
	}
	return &BeaconBlockBody{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
	}
}

// CopySignedBeaconBlockAltair copies the provided SignedBeaconBlock.
func CopySignedBeaconBlockAltair(sigBlock *SignedBeaconBlockAltair) *SignedBeaconBlockAltair {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockAltair{
		Block:     CopyBeaconBlockAltair(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlockAltair copies the provided BeaconBlock.
func CopyBeaconBlockAltair(block *BeaconBlockAltair) *BeaconBlockAltair {
	if block == nil {
		return nil
	}
	return &BeaconBlockAltair{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBodyAltair(block.Body),
	}
}

// CopyBeaconBlockBodyAltair copies the provided BeaconBlockBody.
func CopyBeaconBlockBodyAltair(body *BeaconBlockBodyAltair) *BeaconBlockBodyAltair {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyAltair{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
		SyncAggregate:     body.SyncAggregate.Copy(),
	}
}

// CopyValidator copies the provided validator.
func CopyValidator(val *Validator) *Validator {
	pubKey := make([]byte, len(val.PublicKey))
	copy(pubKey, val.PublicKey)
	withdrawalCreds := make([]byte, len(val.WithdrawalCredentials))
	copy(withdrawalCreds, val.WithdrawalCredentials)
	return &Validator{
		PublicKey:                  pubKey,
		WithdrawalCredentials:      withdrawalCreds,
		EffectiveBalance:           val.EffectiveBalance,
		Slashed:                    val.Slashed,
		ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
		ActivationEpoch:            val.ActivationEpoch,
		ExitEpoch:                  val.ExitEpoch,
		WithdrawableEpoch:          val.WithdrawableEpoch,
	}
}

// CopySyncCommitteeMessage copies the provided sync committee message object.
func CopySyncCommitteeMessage(s *SyncCommitteeMessage) *SyncCommitteeMessage {
	if s == nil {
		return nil
	}
	return &SyncCommitteeMessage{
		Slot:           s.Slot,
		BlockRoot:      bytesutil.SafeCopyBytes(s.BlockRoot),
		ValidatorIndex: s.ValidatorIndex,
		Signature:      bytesutil.SafeCopyBytes(s.Signature),
	}
}

// CopySyncCommitteeContribution copies the provided sync committee contribution object.
func CopySyncCommitteeContribution(c *SyncCommitteeContribution) *SyncCommitteeContribution {
	if c == nil {
		return nil
	}
	return &SyncCommitteeContribution{
		Slot:              c.Slot,
		BlockRoot:         bytesutil.SafeCopyBytes(c.BlockRoot),
		SubcommitteeIndex: c.SubcommitteeIndex,
		AggregationBits:   bytesutil.SafeCopyBytes(c.AggregationBits),
		Signature:         bytesutil.SafeCopyBytes(c.Signature),
	}
}

// CopySignedBeaconBlockBellatrix copies the provided SignedBeaconBlockBellatrix.
func CopySignedBeaconBlockBellatrix(sigBlock *SignedBeaconBlockBellatrix) *SignedBeaconBlockBellatrix {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockBellatrix{
		Block:     CopyBeaconBlockBellatrix(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlockBellatrix copies the provided BeaconBlockBellatrix.
func CopyBeaconBlockBellatrix(block *BeaconBlockBellatrix) *BeaconBlockBellatrix {
	if block == nil {
		return nil
	}
	return &BeaconBlockBellatrix{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBodyBellatrix(block.Body),
	}
}

// CopyBeaconBlockBodyBellatrix copies the provided BeaconBlockBodyBellatrix.
func CopyBeaconBlockBodyBellatrix(body *BeaconBlockBodyBellatrix) *BeaconBlockBodyBellatrix {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyBellatrix{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          body.Eth1Data.Copy(),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopySlice(body.ProposerSlashings),
		AttesterSlashings: CopySlice(body.AttesterSlashings),
		Attestations:      CopySlice(body.Attestations),
		Deposits:          CopySlice(body.Deposits),
		VoluntaryExits:    CopySlice(body.VoluntaryExits),
		SyncAggregate:     body.SyncAggregate.Copy(),
		ExecutionPayload:  body.ExecutionPayload.Copy(),
	}
}

// CopySignedBeaconBlockCapella copies the provided SignedBeaconBlockCapella.
func CopySignedBeaconBlockCapella(sigBlock *SignedBeaconBlockCapella) *SignedBeaconBlockCapella {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockCapella{
		Block:     CopyBeaconBlockCapella(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlockCapella copies the provided BeaconBlockCapella.
func CopyBeaconBlockCapella(block *BeaconBlockCapella) *BeaconBlockCapella {
	if block == nil {
		return nil
	}
	return &BeaconBlockCapella{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBodyCapella(block.Body),
	}
}

// CopyBeaconBlockBodyCapella copies the provided BeaconBlockBodyCapella.
func CopyBeaconBlockBodyCapella(body *BeaconBlockBodyCapella) *BeaconBlockBodyCapella {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyCapella{
		RandaoReveal:          bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:              body.Eth1Data.Copy(),
		Graffiti:              bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:     CopySlice(body.ProposerSlashings),
		AttesterSlashings:     CopySlice(body.AttesterSlashings),
		Attestations:          CopySlice(body.Attestations),
		Deposits:              CopySlice(body.Deposits),
		VoluntaryExits:        CopySlice(body.VoluntaryExits),
		SyncAggregate:         body.SyncAggregate.Copy(),
		ExecutionPayload:      body.ExecutionPayload.Copy(),
		BlsToExecutionChanges: CopySlice(body.BlsToExecutionChanges),
	}
}

// CopySignedBlindedBeaconBlockCapella copies the provided SignedBlindedBeaconBlockCapella.
func CopySignedBlindedBeaconBlockCapella(sigBlock *SignedBlindedBeaconBlockCapella) *SignedBlindedBeaconBlockCapella {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockCapella{
		Block:     CopyBlindedBeaconBlockCapella(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBlindedBeaconBlockCapella copies the provided BlindedBeaconBlockCapella.
func CopyBlindedBeaconBlockCapella(block *BlindedBeaconBlockCapella) *BlindedBeaconBlockCapella {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockCapella{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBlindedBeaconBlockBodyCapella(block.Body),
	}
}

// CopyBlindedBeaconBlockBodyCapella copies the provided BlindedBeaconBlockBodyCapella.
func CopyBlindedBeaconBlockBodyCapella(body *BlindedBeaconBlockBodyCapella) *BlindedBeaconBlockBodyCapella {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyCapella{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		SyncAggregate:          body.SyncAggregate.Copy(),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
		BlsToExecutionChanges:  CopySlice(body.BlsToExecutionChanges),
	}
}

// CopySignedBlindedBeaconBlockDeneb copies the provided SignedBlindedBeaconBlockDeneb.
func CopySignedBlindedBeaconBlockDeneb(sigBlock *SignedBlindedBeaconBlockDeneb) *SignedBlindedBeaconBlockDeneb {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockDeneb{
		Message:   CopyBlindedBeaconBlockDeneb(sigBlock.Message),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBlindedBeaconBlockDeneb copies the provided BlindedBeaconBlockDeneb.
func CopyBlindedBeaconBlockDeneb(block *BlindedBeaconBlockDeneb) *BlindedBeaconBlockDeneb {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockDeneb{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBlindedBeaconBlockBodyDeneb(block.Body),
	}
}

// CopyBlindedBeaconBlockBodyDeneb copies the provided BlindedBeaconBlockBodyDeneb.
func CopyBlindedBeaconBlockBodyDeneb(body *BlindedBeaconBlockBodyDeneb) *BlindedBeaconBlockBodyDeneb {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyDeneb{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		SyncAggregate:          body.SyncAggregate.Copy(),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
		BlsToExecutionChanges:  CopySlice(body.BlsToExecutionChanges),
		BlobKzgCommitments:     CopyBlobKZGs(body.BlobKzgCommitments),
	}
}

// CopySignedBlindedBeaconBlockElectra copies the provided SignedBlindedBeaconBlockElectra.
func CopySignedBlindedBeaconBlockElectra(sigBlock *SignedBlindedBeaconBlockElectra) *SignedBlindedBeaconBlockElectra {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockElectra{
		Message:   CopyBlindedBeaconBlockElectra(sigBlock.Message),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBlindedBeaconBlockElectra copies the provided BlindedBeaconBlockElectra.
func CopyBlindedBeaconBlockElectra(block *BlindedBeaconBlockElectra) *BlindedBeaconBlockElectra {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockElectra{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBlindedBeaconBlockBodyElectra(block.Body),
	}
}

// CopyBlindedBeaconBlockBodyElectra copies the provided BlindedBeaconBlockBodyElectra.
func CopyBlindedBeaconBlockBodyElectra(body *BlindedBeaconBlockBodyElectra) *BlindedBeaconBlockBodyElectra {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyElectra{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		SyncAggregate:          body.SyncAggregate.Copy(),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
		BlsToExecutionChanges:  CopySlice(body.BlsToExecutionChanges),
		BlobKzgCommitments:     CopyBlobKZGs(body.BlobKzgCommitments),
	}
}

// CopySignedBlindedBeaconBlockBellatrix copies the provided SignedBlindedBeaconBlockBellatrix.
func CopySignedBlindedBeaconBlockBellatrix(sigBlock *SignedBlindedBeaconBlockBellatrix) *SignedBlindedBeaconBlockBellatrix {
	if sigBlock == nil {
		return nil
	}
	return &SignedBlindedBeaconBlockBellatrix{
		Block:     CopyBlindedBeaconBlockBellatrix(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBlindedBeaconBlockBellatrix copies the provided BlindedBeaconBlockBellatrix.
func CopyBlindedBeaconBlockBellatrix(block *BlindedBeaconBlockBellatrix) *BlindedBeaconBlockBellatrix {
	if block == nil {
		return nil
	}
	return &BlindedBeaconBlockBellatrix{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBlindedBeaconBlockBodyBellatrix(block.Body),
	}
}

// CopyBlindedBeaconBlockBodyBellatrix copies the provided BlindedBeaconBlockBodyBellatrix.
func CopyBlindedBeaconBlockBodyBellatrix(body *BlindedBeaconBlockBodyBellatrix) *BlindedBeaconBlockBodyBellatrix {
	if body == nil {
		return nil
	}
	return &BlindedBeaconBlockBodyBellatrix{
		RandaoReveal:           bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:               body.Eth1Data.Copy(),
		Graffiti:               bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:      CopySlice(body.ProposerSlashings),
		AttesterSlashings:      CopySlice(body.AttesterSlashings),
		Attestations:           CopySlice(body.Attestations),
		Deposits:               CopySlice(body.Deposits),
		VoluntaryExits:         CopySlice(body.VoluntaryExits),
		SyncAggregate:          body.SyncAggregate.Copy(),
		ExecutionPayloadHeader: body.ExecutionPayloadHeader.Copy(),
	}
}

// CopyBlobKZGs copies the provided blob kzgs object.
func CopyBlobKZGs(b [][]byte) [][]byte {
	return bytesutil.SafeCopy2dBytes(b)
}

// CopySignedBeaconBlockDeneb copies the provided SignedBeaconBlockDeneb.
func CopySignedBeaconBlockDeneb(sigBlock *SignedBeaconBlockDeneb) *SignedBeaconBlockDeneb {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockDeneb{
		Block:     CopyBeaconBlockDeneb(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopySignedBeaconBlockEPBS copies the provided SignedBeaconBlockEPBS.
func CopySignedBeaconBlockEPBS(sigBlock *SignedBeaconBlockEpbs) *SignedBeaconBlockEpbs {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockEpbs{
		Block:     CopyBeaconBlockEPBS(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlockDeneb copies the provided BeaconBlockDeneb.
func CopyBeaconBlockDeneb(block *BeaconBlockDeneb) *BeaconBlockDeneb {
	if block == nil {
		return nil
	}
	return &BeaconBlockDeneb{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBodyDeneb(block.Body),
	}
}

// CopyBeaconBlockEPBS copies the provided CopyBeaconBlockEPBS.
func CopyBeaconBlockEPBS(block *BeaconBlockEpbs) *BeaconBlockEpbs {
	if block == nil {
		return nil
	}
	return &BeaconBlockEpbs{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBodyEPBS(block.Body),
	}
}

// CopyBeaconBlockBodyDeneb copies the provided BeaconBlockBodyDeneb.
func CopyBeaconBlockBodyDeneb(body *BeaconBlockBodyDeneb) *BeaconBlockBodyDeneb {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyDeneb{
		RandaoReveal:          bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:              body.Eth1Data.Copy(),
		Graffiti:              bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:     CopySlice(body.ProposerSlashings),
		AttesterSlashings:     CopySlice(body.AttesterSlashings),
		Attestations:          CopySlice(body.Attestations),
		Deposits:              CopySlice(body.Deposits),
		VoluntaryExits:        CopySlice(body.VoluntaryExits),
		SyncAggregate:         body.SyncAggregate.Copy(),
		ExecutionPayload:      body.ExecutionPayload.Copy(),
		BlsToExecutionChanges: CopySlice(body.BlsToExecutionChanges),
		BlobKzgCommitments:    CopyBlobKZGs(body.BlobKzgCommitments),
	}
}

// CopyBeaconBlockBodyEPBS copies the provided CopyBeaconBlockBodyEPBS.
func CopyBeaconBlockBodyEPBS(body *BeaconBlockBodyEpbs) *BeaconBlockBodyEpbs {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyEpbs{
		RandaoReveal:                 bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:                     body.Eth1Data.Copy(),
		Graffiti:                     bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:            CopySlice(body.ProposerSlashings),
		AttesterSlashings:            CopySlice(body.AttesterSlashings),
		Attestations:                 CopySlice(body.Attestations),
		Deposits:                     CopySlice(body.Deposits),
		VoluntaryExits:               CopySlice(body.VoluntaryExits),
		SyncAggregate:                body.SyncAggregate.Copy(),
		BlsToExecutionChanges:        CopySlice(body.BlsToExecutionChanges),
		SignedExecutionPayloadHeader: CopySignedExecutionPayloadHeader(body.SignedExecutionPayloadHeader),
		PayloadAttestations:          CopyPayloadAttestation(body.PayloadAttestations),
	}
}

// CopySignedExecutionPayloadHeader copies the provided SignedExecutionPayloadHeader.
func CopySignedExecutionPayloadHeader(payload *enginev1.SignedExecutionPayloadHeader) *enginev1.SignedExecutionPayloadHeader {
	if payload == nil {
		return nil
	}
	return &enginev1.SignedExecutionPayloadHeader{
		Message:   CopyExecutionPayloadHeaderEPBS(payload.Message),
		Signature: bytesutil.SafeCopyBytes(payload.Signature),
	}
}

// CopyExecutionPayloadHeaderEPBS copies the provided execution payload header object.
func CopyExecutionPayloadHeaderEPBS(payload *enginev1.ExecutionPayloadHeaderEPBS) *enginev1.ExecutionPayloadHeaderEPBS {
	if payload == nil {
		return nil
	}
	return &enginev1.ExecutionPayloadHeaderEPBS{
		ParentBlockHash:        bytesutil.SafeCopyBytes(payload.ParentBlockHash),
		ParentBlockRoot:        bytesutil.SafeCopyBytes(payload.ParentBlockRoot),
		BlockHash:              bytesutil.SafeCopyBytes(payload.BlockHash),
		BuilderIndex:           payload.BuilderIndex,
		Slot:                   payload.Slot,
		Value:                  payload.Value,
		BlobKzgCommitmentsRoot: bytesutil.SafeCopyBytes(payload.BlobKzgCommitmentsRoot),
	}
}

// CopyPayloadAttestation copies the provided PayloadAttestation array.
func CopyPayloadAttestation(attestations []*PayloadAttestation) []*PayloadAttestation {
	if attestations == nil {
		return nil
	}
	newAttestations := make([]*PayloadAttestation, len(attestations))
	for i, att := range attestations {
		newAttestations[i] = &PayloadAttestation{
			AggregationBits: bytesutil.SafeCopyBytes(att.AggregationBits),
			Data:            CopyPayloadAttestationData(att.Data),
			Signature:       bytesutil.SafeCopyBytes(att.Signature),
		}
	}
	return newAttestations
}

// CopyPayloadAttestationData copies the provided PayloadAttestationData.
func CopyPayloadAttestationData(data *PayloadAttestationData) *PayloadAttestationData {
	if data == nil {
		return nil
	}
	return &PayloadAttestationData{
		BeaconBlockRoot: bytesutil.SafeCopyBytes(data.BeaconBlockRoot),
		Slot:            data.Slot,
		PayloadStatus:   data.PayloadStatus,
	}
}

// CopySignedBeaconBlockElectra copies the provided SignedBeaconBlockElectra.
func CopySignedBeaconBlockElectra(sigBlock *SignedBeaconBlockElectra) *SignedBeaconBlockElectra {
	if sigBlock == nil {
		return nil
	}
	return &SignedBeaconBlockElectra{
		Block:     CopyBeaconBlockElectra(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlockElectra copies the provided BeaconBlockElectra.
func CopyBeaconBlockElectra(block *BeaconBlockElectra) *BeaconBlockElectra {
	if block == nil {
		return nil
	}
	return &BeaconBlockElectra{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBodyElectra(block.Body),
	}
}

// CopyBeaconBlockBodyElectra copies the provided BeaconBlockBodyElectra.
func CopyBeaconBlockBodyElectra(body *BeaconBlockBodyElectra) *BeaconBlockBodyElectra {
	if body == nil {
		return nil
	}
	return &BeaconBlockBodyElectra{
		RandaoReveal:          bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:              body.Eth1Data.Copy(),
		Graffiti:              bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings:     CopySlice(body.ProposerSlashings),
		AttesterSlashings:     CopySlice(body.AttesterSlashings),
		Attestations:          CopySlice(body.Attestations),
		Deposits:              CopySlice(body.Deposits),
		VoluntaryExits:        CopySlice(body.VoluntaryExits),
		SyncAggregate:         body.SyncAggregate.Copy(),
		ExecutionPayload:      body.ExecutionPayload.Copy(),
		BlsToExecutionChanges: CopySlice(body.BlsToExecutionChanges),
		BlobKzgCommitments:    CopyBlobKZGs(body.BlobKzgCommitments),
	}
}
