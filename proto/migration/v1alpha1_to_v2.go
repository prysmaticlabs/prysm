package migration

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// V1Alpha1BeaconBlockAltairToV2 converts a v1alpha1 Altair beacon block to a v2 Altair block.
func V1Alpha1BeaconBlockAltairToV2(v1alpha1Block *ethpbalpha.BeaconBlockAltair) (*ethpbv2.BeaconBlockAltair, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockAltair{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockAltairToV2Signed converts a v1alpha1 Altair signed beacon block to a v2 Altair block.
func V1Alpha1BeaconBlockAltairToV2Signed(v1alpha1Block *ethpbalpha.SignedBeaconBlockAltair) (*ethpbv2.SignedBeaconBlockAltair, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.SignedBeaconBlockAltair{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockBellatrixToV2 converts a v1alpha1 Bellatrix beacon block to a v2
// Bellatrix block.
func V1Alpha1BeaconBlockBellatrixToV2(v1alpha1Block *ethpbalpha.BeaconBlockBellatrix) (*ethpbv2.BeaconBlockBellatrix, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockBellatrix{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockCapellaToV2 converts a v1alpha1 Capella beacon block to a v2
// Capella block.
func V1Alpha1BeaconBlockCapellaToV2(v1alpha1Block *ethpbalpha.BeaconBlockCapella) (*ethpbv2.BeaconBlockCapella, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockCapella{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockDenebToV2 converts a v1alpha1 Deneb beacon block to a v2
// Deneb block.
func V1Alpha1BeaconBlockDenebToV2(v1alpha1Block *ethpbalpha.BeaconBlockDeneb) (*ethpbv2.BeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1SignedBeaconBlockDenebToV2 converts a v1alpha1 signed Deneb beacon block to a v2
// Deneb block.
func V1Alpha1SignedBeaconBlockDenebToV2(v1alpha1Block *ethpbalpha.SignedBeaconBlockDeneb) (*ethpbv2.SignedBeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.SignedBeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BlobSidecarsToV2 converts an array of v1alpha1 blinded blob sidecars to its v2 equivalent.
func V1Alpha1BlobSidecarsToV2(v1alpha1Blobs []*ethpbalpha.BlobSidecar) ([]*ethpbv2.BlobSidecar, error) {
	v2Blobs := make([]*ethpbv2.BlobSidecar, len(v1alpha1Blobs))
	for index, v1Blob := range v1alpha1Blobs {
		marshaledBlob, err := proto.Marshal(v1Blob)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal blob sidecar")
		}
		v2Blob := &ethpbv2.BlobSidecar{}
		if err := proto.Unmarshal(marshaledBlob, v2Blob); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal blob sidecar")
		}
		v2Blobs[index] = v2Blob
	}
	return v2Blobs, nil
}

// V1Alpha1BlindedBlobSidecarsToV2 converts an array of v1alpha1 blinded blob sidecars to its v2 equivalent.
func V1Alpha1BlindedBlobSidecarsToV2(v1alpha1Blobs []*ethpbalpha.BlindedBlobSidecar) ([]*ethpbv2.BlindedBlobSidecar, error) {
	v2Blobs := make([]*ethpbv2.BlindedBlobSidecar, len(v1alpha1Blobs))
	for index, v1Blob := range v1alpha1Blobs {
		marshaledBlob, err := proto.Marshal(v1Blob)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal blob sidecar")
		}
		v2Blob := &ethpbv2.BlindedBlobSidecar{}
		if err := proto.Unmarshal(marshaledBlob, v2Blob); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal blob sidecar")
		}
		v2Blobs[index] = v2Blob
	}
	return v2Blobs, nil
}

// V1Alpha1SignedBlindedBlobSidecarsToV2 converts an array of v1alpha1 objects to its v2 SignedBlindedBlobSidecar equivalent.
func V1Alpha1SignedBlindedBlobSidecarsToV2(sidecars []*ethpbalpha.SignedBlindedBlobSidecar) []*ethpbv2.SignedBlindedBlobSidecar {
	result := make([]*ethpbv2.SignedBlindedBlobSidecar, len(sidecars))
	for i, sc := range sidecars {
		result[i] = &ethpbv2.SignedBlindedBlobSidecar{
			Message: &ethpbv2.BlindedBlobSidecar{
				BlockRoot:       bytesutil.SafeCopyBytes(sc.Message.BlockRoot),
				Index:           sc.Message.Index,
				Slot:            sc.Message.Slot,
				BlockParentRoot: bytesutil.SafeCopyBytes(sc.Message.BlockParentRoot),
				ProposerIndex:   sc.Message.ProposerIndex,
				BlobRoot:        bytesutil.SafeCopyBytes(sc.Message.BlobRoot),
				KzgCommitment:   bytesutil.SafeCopyBytes(sc.Message.KzgCommitment),
				KzgProof:        bytesutil.SafeCopyBytes(sc.Message.KzgProof),
			},
			Signature: bytesutil.SafeCopyBytes(sc.Signature),
		}
	}
	return result
}

// V1Alpha1SignedBlobsToV2 converts an array of v1alpha1 objects to its v2 SignedBlobSidecar equivalent.
func V1Alpha1SignedBlobsToV2(sidecars []*ethpbalpha.SignedBlobSidecar) []*ethpbv2.SignedBlobSidecar {
	result := make([]*ethpbv2.SignedBlobSidecar, len(sidecars))
	for i, sc := range sidecars {
		result[i] = &ethpbv2.SignedBlobSidecar{
			Message: &ethpbv2.BlobSidecar{
				BlockRoot:       bytesutil.SafeCopyBytes(sc.Message.BlockRoot),
				Index:           sc.Message.Index,
				Slot:            sc.Message.Slot,
				BlockParentRoot: bytesutil.SafeCopyBytes(sc.Message.BlockParentRoot),
				ProposerIndex:   sc.Message.ProposerIndex,
				Blob:            bytesutil.SafeCopyBytes(sc.Message.Blob),
				KzgCommitment:   bytesutil.SafeCopyBytes(sc.Message.KzgCommitment),
				KzgProof:        bytesutil.SafeCopyBytes(sc.Message.KzgProof),
			},
			Signature: bytesutil.SafeCopyBytes(sc.Signature),
		}
	}
	return result
}

// V1Alpha1BeaconBlockDenebAndBlobsToV2 converts a v1alpha1 Deneb beacon block and blobs to a v2
// Deneb block.
func V1Alpha1BeaconBlockDenebAndBlobsToV2(v1alpha1Block *ethpbalpha.BeaconBlockAndBlobsDeneb) (*ethpbv2.BeaconBlockContentsDeneb, error) {
	v2Block, err := V1Alpha1BeaconBlockDenebToV2(v1alpha1Block.Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block")
	}
	v2Blobs, err := V1Alpha1BlobSidecarsToV2(v1alpha1Block.Blobs)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert blobs")
	}
	return &ethpbv2.BeaconBlockContentsDeneb{Block: v2Block, BlobSidecars: v2Blobs}, nil
}

// V1Alpha1SignedBeaconBlockDenebAndBlobsToV2 converts a signed v1alpha1 Deneb beacon block and blobs to a v2
// Deneb block.
func V1Alpha1SignedBeaconBlockDenebAndBlobsToV2(v1alpha1Block *ethpbalpha.SignedBeaconBlockAndBlobsDeneb) (*ethpbv2.SignedBeaconBlockContentsDeneb, error) {
	v2Block, err := V1Alpha1SignedBeaconBlockDenebToV2(v1alpha1Block.Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block")
	}
	v2Blobs := V1Alpha1SignedBlobsToV2(v1alpha1Block.Blobs)
	return &ethpbv2.SignedBeaconBlockContentsDeneb{
		SignedBlock:        v2Block,
		SignedBlobSidecars: v2Blobs,
	}, nil
}

// V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded converts a v1alpha1 Blinded Bellatrix beacon block to a v2 Blinded Bellatrix block.
func V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(v1alpha1Block *ethpbalpha.BlindedBeaconBlockBellatrix) (*ethpbv2.BlindedBeaconBlockBellatrix, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BlindedBeaconBlockBellatrix{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockBlindedCapellaToV2Blinded converts a v1alpha1 Blinded Capella beacon block to a v2 Blinded Capella block.
func V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(v1alpha1Block *ethpbalpha.BlindedBeaconBlockCapella) (*ethpbv2.BlindedBeaconBlockCapella, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BlindedBeaconBlockCapella{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockBlindedDenebToV2Blinded converts a v1alpha1 Blinded Deneb beacon block to a v2 Blinded Deneb block.
func V1Alpha1BeaconBlockBlindedDenebToV2Blinded(v1alpha1Block *ethpbalpha.BlindedBeaconBlockDeneb) (*ethpbv2.BlindedBeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BlindedBeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1SignedBeaconBlockBlindedDenebToV2Blinded converts a v1alpha1 Signed Blinded Deneb beacon block to a v2 Blinded Deneb block.
func V1Alpha1SignedBeaconBlockBlindedDenebToV2Blinded(v1alpha1Block *ethpbalpha.SignedBlindedBeaconBlockDeneb) (*ethpbv2.SignedBlindedBeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.SignedBlindedBeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// V1Alpha1BlindedBlockAndBlobsDenebToV2Blinded converts a v1alpha1 Deneb blinded beacon block and blobs to v2 blinded block contents.
func V1Alpha1BlindedBlockAndBlobsDenebToV2Blinded(
	v1Alpha1BlkAndBlobs *ethpbalpha.BlindedBeaconBlockAndBlobsDeneb,
) (*ethpbv2.BlindedBeaconBlockContentsDeneb, error) {
	v2Block, err := V1Alpha1BeaconBlockBlindedDenebToV2Blinded(v1Alpha1BlkAndBlobs.Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block")
	}
	v2Blobs, err := V1Alpha1BlindedBlobSidecarsToV2(v1Alpha1BlkAndBlobs.Blobs)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert blobs")
	}
	return &ethpbv2.BlindedBeaconBlockContentsDeneb{BlindedBlock: v2Block, BlindedBlobSidecars: v2Blobs}, nil
}

// V1Alpha1SignedBlindedBlockAndBlobsDenebToV2Blinded converts a v1alpha1 signed Deneb blinded beacon block and blobs to v2 blinded block contents.
func V1Alpha1SignedBlindedBlockAndBlobsDenebToV2Blinded(
	v1Alpha1BlkAndBlobs *ethpbalpha.SignedBlindedBeaconBlockAndBlobsDeneb,
) (*ethpbv2.SignedBlindedBeaconBlockContentsDeneb, error) {
	v2Block, err := V1Alpha1SignedBeaconBlockBlindedDenebToV2Blinded(v1Alpha1BlkAndBlobs.SignedBlindedBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block")
	}
	v2Blobs := V1Alpha1SignedBlindedBlobSidecarsToV2(v1Alpha1BlkAndBlobs.SignedBlindedBlobSidecars)
	return &ethpbv2.SignedBlindedBeaconBlockContentsDeneb{
		SignedBlindedBlock:        v2Block,
		SignedBlindedBlobSidecars: v2Blobs,
	}, nil
}

// V1Alpha1BeaconBlockBellatrixToV2Blinded converts a v1alpha1 Bellatrix beacon block to a v2
// blinded Bellatrix block.
func V1Alpha1BeaconBlockBellatrixToV2Blinded(v1alpha1Block *ethpbalpha.BeaconBlockBellatrix) (*ethpbv2.BlindedBeaconBlockBellatrix, error) {
	sourceProposerSlashings := v1alpha1Block.Body.ProposerSlashings
	resultProposerSlashings := make([]*ethpbv1.ProposerSlashing, len(sourceProposerSlashings))
	for i, s := range sourceProposerSlashings {
		resultProposerSlashings[i] = &ethpbv1.ProposerSlashing{
			SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
				Message: &ethpbv1.BeaconBlockHeader{
					Slot:          s.Header_1.Header.Slot,
					ProposerIndex: s.Header_1.Header.ProposerIndex,
					ParentRoot:    bytesutil.SafeCopyBytes(s.Header_1.Header.ParentRoot),
					StateRoot:     bytesutil.SafeCopyBytes(s.Header_1.Header.StateRoot),
					BodyRoot:      bytesutil.SafeCopyBytes(s.Header_1.Header.BodyRoot),
				},
				Signature: bytesutil.SafeCopyBytes(s.Header_1.Signature),
			},
			SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
				Message: &ethpbv1.BeaconBlockHeader{
					Slot:          s.Header_2.Header.Slot,
					ProposerIndex: s.Header_2.Header.ProposerIndex,
					ParentRoot:    bytesutil.SafeCopyBytes(s.Header_2.Header.ParentRoot),
					StateRoot:     bytesutil.SafeCopyBytes(s.Header_2.Header.StateRoot),
					BodyRoot:      bytesutil.SafeCopyBytes(s.Header_2.Header.BodyRoot),
				},
				Signature: bytesutil.SafeCopyBytes(s.Header_2.Signature),
			},
		}
	}

	sourceAttesterSlashings := v1alpha1Block.Body.AttesterSlashings
	resultAttesterSlashings := make([]*ethpbv1.AttesterSlashing, len(sourceAttesterSlashings))
	for i, s := range sourceAttesterSlashings {
		att1Indices := make([]uint64, len(s.Attestation_1.AttestingIndices))
		copy(att1Indices, s.Attestation_1.AttestingIndices)
		att2Indices := make([]uint64, len(s.Attestation_2.AttestingIndices))
		copy(att2Indices, s.Attestation_2.AttestingIndices)
		resultAttesterSlashings[i] = &ethpbv1.AttesterSlashing{
			Attestation_1: &ethpbv1.IndexedAttestation{
				AttestingIndices: att1Indices,
				Data: &ethpbv1.AttestationData{
					Slot:            s.Attestation_1.Data.Slot,
					Index:           s.Attestation_1.Data.CommitteeIndex,
					BeaconBlockRoot: bytesutil.SafeCopyBytes(s.Attestation_1.Data.BeaconBlockRoot),
					Source: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_1.Data.Source.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_1.Data.Source.Root),
					},
					Target: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_1.Data.Target.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_1.Data.Target.Root),
					},
				},
				Signature: bytesutil.SafeCopyBytes(s.Attestation_1.Signature),
			},
			Attestation_2: &ethpbv1.IndexedAttestation{
				AttestingIndices: att2Indices,
				Data: &ethpbv1.AttestationData{
					Slot:            s.Attestation_2.Data.Slot,
					Index:           s.Attestation_2.Data.CommitteeIndex,
					BeaconBlockRoot: bytesutil.SafeCopyBytes(s.Attestation_2.Data.BeaconBlockRoot),
					Source: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_2.Data.Source.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_2.Data.Source.Root),
					},
					Target: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_2.Data.Target.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_2.Data.Target.Root),
					},
				},
				Signature: bytesutil.SafeCopyBytes(s.Attestation_2.Signature),
			},
		}
	}

	sourceAttestations := v1alpha1Block.Body.Attestations
	resultAttestations := make([]*ethpbv1.Attestation, len(sourceAttestations))
	for i, a := range sourceAttestations {
		resultAttestations[i] = &ethpbv1.Attestation{
			AggregationBits: bytesutil.SafeCopyBytes(a.AggregationBits),
			Data: &ethpbv1.AttestationData{
				Slot:            a.Data.Slot,
				Index:           a.Data.CommitteeIndex,
				BeaconBlockRoot: bytesutil.SafeCopyBytes(a.Data.BeaconBlockRoot),
				Source: &ethpbv1.Checkpoint{
					Epoch: a.Data.Source.Epoch,
					Root:  bytesutil.SafeCopyBytes(a.Data.Source.Root),
				},
				Target: &ethpbv1.Checkpoint{
					Epoch: a.Data.Target.Epoch,
					Root:  bytesutil.SafeCopyBytes(a.Data.Target.Root),
				},
			},
			Signature: bytesutil.SafeCopyBytes(a.Signature),
		}
	}

	sourceDeposits := v1alpha1Block.Body.Deposits
	resultDeposits := make([]*ethpbv1.Deposit, len(sourceDeposits))
	for i, d := range sourceDeposits {
		resultDeposits[i] = &ethpbv1.Deposit{
			Proof: bytesutil.SafeCopy2dBytes(d.Proof),
			Data: &ethpbv1.Deposit_Data{
				Pubkey:                bytesutil.SafeCopyBytes(d.Data.PublicKey),
				WithdrawalCredentials: bytesutil.SafeCopyBytes(d.Data.WithdrawalCredentials),
				Amount:                d.Data.Amount,
				Signature:             bytesutil.SafeCopyBytes(d.Data.Signature),
			},
		}
	}

	sourceExits := v1alpha1Block.Body.VoluntaryExits
	resultExits := make([]*ethpbv1.SignedVoluntaryExit, len(sourceExits))
	for i, e := range sourceExits {
		resultExits[i] = &ethpbv1.SignedVoluntaryExit{
			Message: &ethpbv1.VoluntaryExit{
				Epoch:          e.Exit.Epoch,
				ValidatorIndex: e.Exit.ValidatorIndex,
			},
			Signature: bytesutil.SafeCopyBytes(e.Signature),
		}
	}

	transactionsRoot, err := ssz.TransactionsRoot(v1alpha1Block.Body.ExecutionPayload.Transactions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate transactions root")
	}

	resultBlockBody := &ethpbv2.BlindedBeaconBlockBodyBellatrix{
		RandaoReveal: bytesutil.SafeCopyBytes(v1alpha1Block.Body.RandaoReveal),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(v1alpha1Block.Body.Eth1Data.DepositRoot),
			DepositCount: v1alpha1Block.Body.Eth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(v1alpha1Block.Body.Eth1Data.BlockHash),
		},
		Graffiti:          bytesutil.SafeCopyBytes(v1alpha1Block.Body.Graffiti),
		ProposerSlashings: resultProposerSlashings,
		AttesterSlashings: resultAttesterSlashings,
		Attestations:      resultAttestations,
		Deposits:          resultDeposits,
		VoluntaryExits:    resultExits,
		SyncAggregate: &ethpbv1.SyncAggregate{
			SyncCommitteeBits:      bytesutil.SafeCopyBytes(v1alpha1Block.Body.SyncAggregate.SyncCommitteeBits),
			SyncCommitteeSignature: bytesutil.SafeCopyBytes(v1alpha1Block.Body.SyncAggregate.SyncCommitteeSignature),
		},
		ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
			ParentHash:       bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.ParentHash),
			FeeRecipient:     bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.FeeRecipient),
			StateRoot:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.StateRoot),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.ReceiptsRoot),
			LogsBloom:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.LogsBloom),
			PrevRandao:       bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.PrevRandao),
			BlockNumber:      v1alpha1Block.Body.ExecutionPayload.BlockNumber,
			GasLimit:         v1alpha1Block.Body.ExecutionPayload.GasLimit,
			GasUsed:          v1alpha1Block.Body.ExecutionPayload.GasUsed,
			Timestamp:        v1alpha1Block.Body.ExecutionPayload.Timestamp,
			ExtraData:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.ExtraData),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.BaseFeePerGas),
			BlockHash:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.BlockHash),
			TransactionsRoot: transactionsRoot[:],
		},
	}
	v2Block := &ethpbv2.BlindedBeaconBlockBellatrix{
		Slot:          v1alpha1Block.Slot,
		ProposerIndex: v1alpha1Block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(v1alpha1Block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(v1alpha1Block.StateRoot),
		Body:          resultBlockBody,
	}
	return v2Block, nil
}

// V1Alpha1BeaconBlockCapellaToV2Blinded converts a v1alpha1 Capella beacon block to a v2
// blinded Capella block.
func V1Alpha1BeaconBlockCapellaToV2Blinded(v1alpha1Block *ethpbalpha.BeaconBlockCapella) (*ethpbv2.BlindedBeaconBlockCapella, error) {
	sourceProposerSlashings := v1alpha1Block.Body.ProposerSlashings
	resultProposerSlashings := make([]*ethpbv1.ProposerSlashing, len(sourceProposerSlashings))
	for i, s := range sourceProposerSlashings {
		resultProposerSlashings[i] = &ethpbv1.ProposerSlashing{
			SignedHeader_1: &ethpbv1.SignedBeaconBlockHeader{
				Message: &ethpbv1.BeaconBlockHeader{
					Slot:          s.Header_1.Header.Slot,
					ProposerIndex: s.Header_1.Header.ProposerIndex,
					ParentRoot:    bytesutil.SafeCopyBytes(s.Header_1.Header.ParentRoot),
					StateRoot:     bytesutil.SafeCopyBytes(s.Header_1.Header.StateRoot),
					BodyRoot:      bytesutil.SafeCopyBytes(s.Header_1.Header.BodyRoot),
				},
				Signature: bytesutil.SafeCopyBytes(s.Header_1.Signature),
			},
			SignedHeader_2: &ethpbv1.SignedBeaconBlockHeader{
				Message: &ethpbv1.BeaconBlockHeader{
					Slot:          s.Header_2.Header.Slot,
					ProposerIndex: s.Header_2.Header.ProposerIndex,
					ParentRoot:    bytesutil.SafeCopyBytes(s.Header_2.Header.ParentRoot),
					StateRoot:     bytesutil.SafeCopyBytes(s.Header_2.Header.StateRoot),
					BodyRoot:      bytesutil.SafeCopyBytes(s.Header_2.Header.BodyRoot),
				},
				Signature: bytesutil.SafeCopyBytes(s.Header_2.Signature),
			},
		}
	}

	sourceAttesterSlashings := v1alpha1Block.Body.AttesterSlashings
	resultAttesterSlashings := make([]*ethpbv1.AttesterSlashing, len(sourceAttesterSlashings))
	for i, s := range sourceAttesterSlashings {
		att1Indices := make([]uint64, len(s.Attestation_1.AttestingIndices))
		copy(att1Indices, s.Attestation_1.AttestingIndices)
		att2Indices := make([]uint64, len(s.Attestation_2.AttestingIndices))
		copy(att2Indices, s.Attestation_2.AttestingIndices)
		resultAttesterSlashings[i] = &ethpbv1.AttesterSlashing{
			Attestation_1: &ethpbv1.IndexedAttestation{
				AttestingIndices: att1Indices,
				Data: &ethpbv1.AttestationData{
					Slot:            s.Attestation_1.Data.Slot,
					Index:           s.Attestation_1.Data.CommitteeIndex,
					BeaconBlockRoot: bytesutil.SafeCopyBytes(s.Attestation_1.Data.BeaconBlockRoot),
					Source: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_1.Data.Source.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_1.Data.Source.Root),
					},
					Target: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_1.Data.Target.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_1.Data.Target.Root),
					},
				},
				Signature: bytesutil.SafeCopyBytes(s.Attestation_1.Signature),
			},
			Attestation_2: &ethpbv1.IndexedAttestation{
				AttestingIndices: att2Indices,
				Data: &ethpbv1.AttestationData{
					Slot:            s.Attestation_2.Data.Slot,
					Index:           s.Attestation_2.Data.CommitteeIndex,
					BeaconBlockRoot: bytesutil.SafeCopyBytes(s.Attestation_2.Data.BeaconBlockRoot),
					Source: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_2.Data.Source.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_2.Data.Source.Root),
					},
					Target: &ethpbv1.Checkpoint{
						Epoch: s.Attestation_2.Data.Target.Epoch,
						Root:  bytesutil.SafeCopyBytes(s.Attestation_2.Data.Target.Root),
					},
				},
				Signature: bytesutil.SafeCopyBytes(s.Attestation_2.Signature),
			},
		}
	}

	sourceAttestations := v1alpha1Block.Body.Attestations
	resultAttestations := make([]*ethpbv1.Attestation, len(sourceAttestations))
	for i, a := range sourceAttestations {
		resultAttestations[i] = &ethpbv1.Attestation{
			AggregationBits: bytesutil.SafeCopyBytes(a.AggregationBits),
			Data: &ethpbv1.AttestationData{
				Slot:            a.Data.Slot,
				Index:           a.Data.CommitteeIndex,
				BeaconBlockRoot: bytesutil.SafeCopyBytes(a.Data.BeaconBlockRoot),
				Source: &ethpbv1.Checkpoint{
					Epoch: a.Data.Source.Epoch,
					Root:  bytesutil.SafeCopyBytes(a.Data.Source.Root),
				},
				Target: &ethpbv1.Checkpoint{
					Epoch: a.Data.Target.Epoch,
					Root:  bytesutil.SafeCopyBytes(a.Data.Target.Root),
				},
			},
			Signature: bytesutil.SafeCopyBytes(a.Signature),
		}
	}

	sourceDeposits := v1alpha1Block.Body.Deposits
	resultDeposits := make([]*ethpbv1.Deposit, len(sourceDeposits))
	for i, d := range sourceDeposits {
		resultDeposits[i] = &ethpbv1.Deposit{
			Proof: bytesutil.SafeCopy2dBytes(d.Proof),
			Data: &ethpbv1.Deposit_Data{
				Pubkey:                bytesutil.SafeCopyBytes(d.Data.PublicKey),
				WithdrawalCredentials: bytesutil.SafeCopyBytes(d.Data.WithdrawalCredentials),
				Amount:                d.Data.Amount,
				Signature:             bytesutil.SafeCopyBytes(d.Data.Signature),
			},
		}
	}

	sourceExits := v1alpha1Block.Body.VoluntaryExits
	resultExits := make([]*ethpbv1.SignedVoluntaryExit, len(sourceExits))
	for i, e := range sourceExits {
		resultExits[i] = &ethpbv1.SignedVoluntaryExit{
			Message: &ethpbv1.VoluntaryExit{
				Epoch:          e.Exit.Epoch,
				ValidatorIndex: e.Exit.ValidatorIndex,
			},
			Signature: bytesutil.SafeCopyBytes(e.Signature),
		}
	}

	transactionsRoot, err := ssz.TransactionsRoot(v1alpha1Block.Body.ExecutionPayload.Transactions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate transactions root")
	}

	withdrawalsRoot, err := ssz.WithdrawalSliceRoot(v1alpha1Block.Body.ExecutionPayload.Withdrawals, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate transactions root")
	}

	changes := make([]*ethpbv2.SignedBLSToExecutionChange, len(v1alpha1Block.Body.BlsToExecutionChanges))
	for i, change := range v1alpha1Block.Body.BlsToExecutionChanges {
		changes[i] = &ethpbv2.SignedBLSToExecutionChange{
			Message: &ethpbv2.BLSToExecutionChange{
				ValidatorIndex:     change.Message.ValidatorIndex,
				FromBlsPubkey:      bytesutil.SafeCopyBytes(change.Message.FromBlsPubkey),
				ToExecutionAddress: bytesutil.SafeCopyBytes(change.Message.ToExecutionAddress),
			},
			Signature: bytesutil.SafeCopyBytes(change.Signature),
		}
	}

	resultBlockBody := &ethpbv2.BlindedBeaconBlockBodyCapella{
		RandaoReveal: bytesutil.SafeCopyBytes(v1alpha1Block.Body.RandaoReveal),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(v1alpha1Block.Body.Eth1Data.DepositRoot),
			DepositCount: v1alpha1Block.Body.Eth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(v1alpha1Block.Body.Eth1Data.BlockHash),
		},
		Graffiti:          bytesutil.SafeCopyBytes(v1alpha1Block.Body.Graffiti),
		ProposerSlashings: resultProposerSlashings,
		AttesterSlashings: resultAttesterSlashings,
		Attestations:      resultAttestations,
		Deposits:          resultDeposits,
		VoluntaryExits:    resultExits,
		SyncAggregate: &ethpbv1.SyncAggregate{
			SyncCommitteeBits:      bytesutil.SafeCopyBytes(v1alpha1Block.Body.SyncAggregate.SyncCommitteeBits),
			SyncCommitteeSignature: bytesutil.SafeCopyBytes(v1alpha1Block.Body.SyncAggregate.SyncCommitteeSignature),
		},
		ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:       bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.ParentHash),
			FeeRecipient:     bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.FeeRecipient),
			StateRoot:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.StateRoot),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.ReceiptsRoot),
			LogsBloom:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.LogsBloom),
			PrevRandao:       bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.PrevRandao),
			BlockNumber:      v1alpha1Block.Body.ExecutionPayload.BlockNumber,
			GasLimit:         v1alpha1Block.Body.ExecutionPayload.GasLimit,
			GasUsed:          v1alpha1Block.Body.ExecutionPayload.GasUsed,
			Timestamp:        v1alpha1Block.Body.ExecutionPayload.Timestamp,
			ExtraData:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.ExtraData),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.BaseFeePerGas),
			BlockHash:        bytesutil.SafeCopyBytes(v1alpha1Block.Body.ExecutionPayload.BlockHash),
			TransactionsRoot: transactionsRoot[:],
			WithdrawalsRoot:  withdrawalsRoot[:],
		},
		BlsToExecutionChanges: changes,
	}
	v2Block := &ethpbv2.BlindedBeaconBlockCapella{
		Slot:          v1alpha1Block.Slot,
		ProposerIndex: v1alpha1Block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(v1alpha1Block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(v1alpha1Block.StateRoot),
		Body:          resultBlockBody,
	}
	return v2Block, nil
}

// BeaconStateAltairToProto converts a state.BeaconState object to its protobuf equivalent.
func BeaconStateAltairToProto(altairState state.BeaconState) (*ethpbv2.BeaconState, error) {
	sourceFork := altairState.Fork()
	sourceLatestBlockHeader := altairState.LatestBlockHeader()
	sourceEth1Data := altairState.Eth1Data()
	sourceEth1DataVotes := altairState.Eth1DataVotes()
	sourceValidators := altairState.Validators()
	sourceJustificationBits := altairState.JustificationBits()
	sourcePrevJustifiedCheckpoint := altairState.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := altairState.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := altairState.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*ethpbv1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(vote.DepositRoot),
			DepositCount: vote.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(vote.BlockHash),
		}
	}
	resultValidators := make([]*ethpbv1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &ethpbv1.Validator{
			Pubkey:                     bytesutil.SafeCopyBytes(validator.PublicKey),
			WithdrawalCredentials:      bytesutil.SafeCopyBytes(validator.WithdrawalCredentials),
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	sourcePrevEpochParticipation, err := altairState.PreviousEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch participation")
	}
	sourceCurrEpochParticipation, err := altairState.CurrentEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch participation")
	}
	sourceInactivityScores, err := altairState.InactivityScores()
	if err != nil {
		return nil, errors.Wrap(err, "could not get inactivity scores")
	}
	sourceCurrSyncCommittee, err := altairState.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	sourceNextSyncCommittee, err := altairState.NextSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next sync committee")
	}

	hrs, err := altairState.HistoricalRoots()
	if err != nil {
		return nil, errors.Wrap(err, "could not get historical roots")
	}

	result := &ethpbv2.BeaconState{
		GenesisTime:           altairState.GenesisTime(),
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(altairState.GenesisValidatorsRoot()),
		Slot:                  altairState.Slot(),
		Fork: &ethpbv1.Fork{
			PreviousVersion: bytesutil.SafeCopyBytes(sourceFork.PreviousVersion),
			CurrentVersion:  bytesutil.SafeCopyBytes(sourceFork.CurrentVersion),
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &ethpbv1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    bytesutil.SafeCopyBytes(sourceLatestBlockHeader.ParentRoot),
			StateRoot:     bytesutil.SafeCopyBytes(sourceLatestBlockHeader.StateRoot),
			BodyRoot:      bytesutil.SafeCopyBytes(sourceLatestBlockHeader.BodyRoot),
		},
		BlockRoots:      bytesutil.SafeCopy2dBytes(altairState.BlockRoots()),
		StateRoots:      bytesutil.SafeCopy2dBytes(altairState.StateRoots()),
		HistoricalRoots: bytesutil.SafeCopy2dBytes(hrs),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(sourceEth1Data.DepositRoot),
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(sourceEth1Data.BlockHash),
		},
		Eth1DataVotes:              resultEth1DataVotes,
		Eth1DepositIndex:           altairState.Eth1DepositIndex(),
		Validators:                 resultValidators,
		Balances:                   altairState.Balances(),
		RandaoMixes:                bytesutil.SafeCopy2dBytes(altairState.RandaoMixes()),
		Slashings:                  altairState.Slashings(),
		PreviousEpochParticipation: bytesutil.SafeCopyBytes(sourcePrevEpochParticipation),
		CurrentEpochParticipation:  bytesutil.SafeCopyBytes(sourceCurrEpochParticipation),
		JustificationBits:          bytesutil.SafeCopyBytes(sourceJustificationBits),
		PreviousJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourcePrevJustifiedCheckpoint.Root),
		},
		CurrentJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceCurrJustifiedCheckpoint.Root),
		},
		FinalizedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceFinalizedCheckpoint.Root),
		},
		InactivityScores: sourceInactivityScores,
		CurrentSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceCurrSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceCurrSyncCommittee.AggregatePubkey),
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceNextSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceNextSyncCommittee.AggregatePubkey),
		},
	}

	return result, nil
}

// BeaconStateBellatrixToProto converts a state.BeaconState object to its protobuf equivalent.
func BeaconStateBellatrixToProto(st state.BeaconState) (*ethpbv2.BeaconStateBellatrix, error) {
	sourceFork := st.Fork()
	sourceLatestBlockHeader := st.LatestBlockHeader()
	sourceEth1Data := st.Eth1Data()
	sourceEth1DataVotes := st.Eth1DataVotes()
	sourceValidators := st.Validators()
	sourceJustificationBits := st.JustificationBits()
	sourcePrevJustifiedCheckpoint := st.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := st.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := st.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*ethpbv1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(vote.DepositRoot),
			DepositCount: vote.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(vote.BlockHash),
		}
	}
	resultValidators := make([]*ethpbv1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &ethpbv1.Validator{
			Pubkey:                     bytesutil.SafeCopyBytes(validator.PublicKey),
			WithdrawalCredentials:      bytesutil.SafeCopyBytes(validator.WithdrawalCredentials),
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	sourcePrevEpochParticipation, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch participation")
	}
	sourceCurrEpochParticipation, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch participation")
	}
	sourceInactivityScores, err := st.InactivityScores()
	if err != nil {
		return nil, errors.Wrap(err, "could not get inactivity scores")
	}
	sourceCurrSyncCommittee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	sourceNextSyncCommittee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next sync committee")
	}
	executionPayloadHeaderInterface, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest execution payload header")
	}
	sourceLatestExecutionPayloadHeader, ok := executionPayloadHeaderInterface.Proto().(*enginev1.ExecutionPayloadHeader)
	if !ok {
		return nil, errors.New("execution payload header has incorrect type")
	}

	hRoots, err := st.HistoricalRoots()
	if err != nil {
		return nil, errors.Wrap(err, "could not get historical roots")
	}

	result := &ethpbv2.BeaconStateBellatrix{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(st.GenesisValidatorsRoot()),
		Slot:                  st.Slot(),
		Fork: &ethpbv1.Fork{
			PreviousVersion: bytesutil.SafeCopyBytes(sourceFork.PreviousVersion),
			CurrentVersion:  bytesutil.SafeCopyBytes(sourceFork.CurrentVersion),
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &ethpbv1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    bytesutil.SafeCopyBytes(sourceLatestBlockHeader.ParentRoot),
			StateRoot:     bytesutil.SafeCopyBytes(sourceLatestBlockHeader.StateRoot),
			BodyRoot:      bytesutil.SafeCopyBytes(sourceLatestBlockHeader.BodyRoot),
		},
		BlockRoots:      bytesutil.SafeCopy2dBytes(st.BlockRoots()),
		StateRoots:      bytesutil.SafeCopy2dBytes(st.StateRoots()),
		HistoricalRoots: bytesutil.SafeCopy2dBytes(hRoots),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(sourceEth1Data.DepositRoot),
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(sourceEth1Data.BlockHash),
		},
		Eth1DataVotes:              resultEth1DataVotes,
		Eth1DepositIndex:           st.Eth1DepositIndex(),
		Validators:                 resultValidators,
		Balances:                   st.Balances(),
		RandaoMixes:                bytesutil.SafeCopy2dBytes(st.RandaoMixes()),
		Slashings:                  st.Slashings(),
		PreviousEpochParticipation: bytesutil.SafeCopyBytes(sourcePrevEpochParticipation),
		CurrentEpochParticipation:  bytesutil.SafeCopyBytes(sourceCurrEpochParticipation),
		JustificationBits:          bytesutil.SafeCopyBytes(sourceJustificationBits),
		PreviousJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourcePrevJustifiedCheckpoint.Root),
		},
		CurrentJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceCurrJustifiedCheckpoint.Root),
		},
		FinalizedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceFinalizedCheckpoint.Root),
		},
		InactivityScores: sourceInactivityScores,
		CurrentSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceCurrSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceCurrSyncCommittee.AggregatePubkey),
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceNextSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceNextSyncCommittee.AggregatePubkey),
		},
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
			ParentHash:       bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ParentHash),
			FeeRecipient:     bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.FeeRecipient),
			StateRoot:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.StateRoot),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ReceiptsRoot),
			LogsBloom:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.LogsBloom),
			PrevRandao:       bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.PrevRandao),
			BlockNumber:      sourceLatestExecutionPayloadHeader.BlockNumber,
			GasLimit:         sourceLatestExecutionPayloadHeader.GasLimit,
			GasUsed:          sourceLatestExecutionPayloadHeader.GasUsed,
			Timestamp:        sourceLatestExecutionPayloadHeader.Timestamp,
			ExtraData:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ExtraData),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.BaseFeePerGas),
			BlockHash:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.BlockHash),
			TransactionsRoot: bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.TransactionsRoot),
		},
	}

	return result, nil
}

// BeaconStateCapellaToProto converts a state.BeaconState object to its protobuf equivalent.
func BeaconStateCapellaToProto(st state.BeaconState) (*ethpbv2.BeaconStateCapella, error) {
	sourceFork := st.Fork()
	sourceLatestBlockHeader := st.LatestBlockHeader()
	sourceEth1Data := st.Eth1Data()
	sourceEth1DataVotes := st.Eth1DataVotes()
	sourceValidators := st.Validators()
	sourceJustificationBits := st.JustificationBits()
	sourcePrevJustifiedCheckpoint := st.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := st.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := st.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*ethpbv1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(vote.DepositRoot),
			DepositCount: vote.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(vote.BlockHash),
		}
	}
	resultValidators := make([]*ethpbv1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &ethpbv1.Validator{
			Pubkey:                     bytesutil.SafeCopyBytes(validator.PublicKey),
			WithdrawalCredentials:      bytesutil.SafeCopyBytes(validator.WithdrawalCredentials),
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	sourcePrevEpochParticipation, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch participation")
	}
	sourceCurrEpochParticipation, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch participation")
	}
	sourceInactivityScores, err := st.InactivityScores()
	if err != nil {
		return nil, errors.Wrap(err, "could not get inactivity scores")
	}
	sourceCurrSyncCommittee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	sourceNextSyncCommittee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next sync committee")
	}
	executionPayloadHeaderInterface, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest execution payload header")
	}
	sourceLatestExecutionPayloadHeader, ok := executionPayloadHeaderInterface.Proto().(*enginev1.ExecutionPayloadHeaderCapella)
	if !ok {
		return nil, errors.New("execution payload header has incorrect type")
	}
	sourceNextWithdrawalIndex, err := st.NextWithdrawalIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next withdrawal index")
	}
	sourceNextWithdrawalValIndex, err := st.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next withdrawal validator index")
	}
	summaries, err := st.HistoricalSummaries()
	if err != nil {
		return nil, errors.Wrap(err, "could not get historical summaries")
	}
	sourceHistoricalSummaries := make([]*ethpbv2.HistoricalSummary, len(summaries))
	for i, summary := range summaries {
		sourceHistoricalSummaries[i] = &ethpbv2.HistoricalSummary{
			BlockSummaryRoot: summary.BlockSummaryRoot,
			StateSummaryRoot: summary.StateSummaryRoot,
		}
	}
	hRoots, err := st.HistoricalRoots()
	if err != nil {
		return nil, errors.Wrap(err, "could not get historical roots")
	}

	result := &ethpbv2.BeaconStateCapella{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(st.GenesisValidatorsRoot()),
		Slot:                  st.Slot(),
		Fork: &ethpbv1.Fork{
			PreviousVersion: bytesutil.SafeCopyBytes(sourceFork.PreviousVersion),
			CurrentVersion:  bytesutil.SafeCopyBytes(sourceFork.CurrentVersion),
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &ethpbv1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    bytesutil.SafeCopyBytes(sourceLatestBlockHeader.ParentRoot),
			StateRoot:     bytesutil.SafeCopyBytes(sourceLatestBlockHeader.StateRoot),
			BodyRoot:      bytesutil.SafeCopyBytes(sourceLatestBlockHeader.BodyRoot),
		},
		BlockRoots: bytesutil.SafeCopy2dBytes(st.BlockRoots()),
		StateRoots: bytesutil.SafeCopy2dBytes(st.StateRoots()),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(sourceEth1Data.DepositRoot),
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(sourceEth1Data.BlockHash),
		},
		Eth1DataVotes:              resultEth1DataVotes,
		Eth1DepositIndex:           st.Eth1DepositIndex(),
		Validators:                 resultValidators,
		Balances:                   st.Balances(),
		RandaoMixes:                bytesutil.SafeCopy2dBytes(st.RandaoMixes()),
		Slashings:                  st.Slashings(),
		PreviousEpochParticipation: bytesutil.SafeCopyBytes(sourcePrevEpochParticipation),
		CurrentEpochParticipation:  bytesutil.SafeCopyBytes(sourceCurrEpochParticipation),
		JustificationBits:          bytesutil.SafeCopyBytes(sourceJustificationBits),
		PreviousJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourcePrevJustifiedCheckpoint.Root),
		},
		CurrentJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceCurrJustifiedCheckpoint.Root),
		},
		FinalizedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceFinalizedCheckpoint.Root),
		},
		InactivityScores: sourceInactivityScores,
		CurrentSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceCurrSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceCurrSyncCommittee.AggregatePubkey),
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceNextSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceNextSyncCommittee.AggregatePubkey),
		},
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:       bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ParentHash),
			FeeRecipient:     bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.FeeRecipient),
			StateRoot:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.StateRoot),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ReceiptsRoot),
			LogsBloom:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.LogsBloom),
			PrevRandao:       bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.PrevRandao),
			BlockNumber:      sourceLatestExecutionPayloadHeader.BlockNumber,
			GasLimit:         sourceLatestExecutionPayloadHeader.GasLimit,
			GasUsed:          sourceLatestExecutionPayloadHeader.GasUsed,
			Timestamp:        sourceLatestExecutionPayloadHeader.Timestamp,
			ExtraData:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ExtraData),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.BaseFeePerGas),
			BlockHash:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.BlockHash),
			TransactionsRoot: bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.TransactionsRoot),
			WithdrawalsRoot:  bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.WithdrawalsRoot),
		},
		NextWithdrawalIndex:          sourceNextWithdrawalIndex,
		NextWithdrawalValidatorIndex: sourceNextWithdrawalValIndex,
		HistoricalSummaries:          sourceHistoricalSummaries,
		HistoricalRoots:              hRoots,
	}

	return result, nil
}

// BeaconStateDenebToProto converts a state.BeaconState object to its protobuf equivalent.
func BeaconStateDenebToProto(st state.BeaconState) (*ethpbv2.BeaconStateDeneb, error) {
	sourceFork := st.Fork()
	sourceLatestBlockHeader := st.LatestBlockHeader()
	sourceEth1Data := st.Eth1Data()
	sourceEth1DataVotes := st.Eth1DataVotes()
	sourceValidators := st.Validators()
	sourceJustificationBits := st.JustificationBits()
	sourcePrevJustifiedCheckpoint := st.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := st.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := st.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*ethpbv1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(vote.DepositRoot),
			DepositCount: vote.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(vote.BlockHash),
		}
	}
	resultValidators := make([]*ethpbv1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &ethpbv1.Validator{
			Pubkey:                     bytesutil.SafeCopyBytes(validator.PublicKey),
			WithdrawalCredentials:      bytesutil.SafeCopyBytes(validator.WithdrawalCredentials),
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	sourcePrevEpochParticipation, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch participation")
	}
	sourceCurrEpochParticipation, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch participation")
	}
	sourceInactivityScores, err := st.InactivityScores()
	if err != nil {
		return nil, errors.Wrap(err, "could not get inactivity scores")
	}
	sourceCurrSyncCommittee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	sourceNextSyncCommittee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next sync committee")
	}
	executionPayloadHeaderInterface, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest execution payload header")
	}
	sourceLatestExecutionPayloadHeader, ok := executionPayloadHeaderInterface.Proto().(*enginev1.ExecutionPayloadHeaderDeneb)
	if !ok {
		return nil, errors.New("execution payload header has incorrect type")
	}
	sourceNextWithdrawalIndex, err := st.NextWithdrawalIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next withdrawal index")
	}
	sourceNextWithdrawalValIndex, err := st.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next withdrawal validator index")
	}
	summaries, err := st.HistoricalSummaries()
	if err != nil {
		return nil, errors.Wrap(err, "could not get historical summaries")
	}
	sourceHistoricalSummaries := make([]*ethpbv2.HistoricalSummary, len(summaries))
	for i, summary := range summaries {
		sourceHistoricalSummaries[i] = &ethpbv2.HistoricalSummary{
			BlockSummaryRoot: summary.BlockSummaryRoot,
			StateSummaryRoot: summary.StateSummaryRoot,
		}
	}

	hr, err := st.HistoricalRoots()
	if err != nil {
		return nil, errors.Wrap(err, "could not get historical roots")
	}

	result := &ethpbv2.BeaconStateDeneb{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(st.GenesisValidatorsRoot()),
		Slot:                  st.Slot(),
		Fork: &ethpbv1.Fork{
			PreviousVersion: bytesutil.SafeCopyBytes(sourceFork.PreviousVersion),
			CurrentVersion:  bytesutil.SafeCopyBytes(sourceFork.CurrentVersion),
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &ethpbv1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    bytesutil.SafeCopyBytes(sourceLatestBlockHeader.ParentRoot),
			StateRoot:     bytesutil.SafeCopyBytes(sourceLatestBlockHeader.StateRoot),
			BodyRoot:      bytesutil.SafeCopyBytes(sourceLatestBlockHeader.BodyRoot),
		},
		BlockRoots:      bytesutil.SafeCopy2dBytes(st.BlockRoots()),
		StateRoots:      bytesutil.SafeCopy2dBytes(st.StateRoots()),
		HistoricalRoots: bytesutil.SafeCopy2dBytes(hr),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(sourceEth1Data.DepositRoot),
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(sourceEth1Data.BlockHash),
		},
		Eth1DataVotes:              resultEth1DataVotes,
		Eth1DepositIndex:           st.Eth1DepositIndex(),
		Validators:                 resultValidators,
		Balances:                   st.Balances(),
		RandaoMixes:                bytesutil.SafeCopy2dBytes(st.RandaoMixes()),
		Slashings:                  st.Slashings(),
		PreviousEpochParticipation: bytesutil.SafeCopyBytes(sourcePrevEpochParticipation),
		CurrentEpochParticipation:  bytesutil.SafeCopyBytes(sourceCurrEpochParticipation),
		JustificationBits:          bytesutil.SafeCopyBytes(sourceJustificationBits),
		PreviousJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourcePrevJustifiedCheckpoint.Root),
		},
		CurrentJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceCurrJustifiedCheckpoint.Root),
		},
		FinalizedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceFinalizedCheckpoint.Root),
		},
		InactivityScores: sourceInactivityScores,
		CurrentSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceCurrSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceCurrSyncCommittee.AggregatePubkey),
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceNextSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceNextSyncCommittee.AggregatePubkey),
		},
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderDeneb{
			ParentHash:       bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ParentHash),
			FeeRecipient:     bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.FeeRecipient),
			StateRoot:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.StateRoot),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ReceiptsRoot),
			LogsBloom:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.LogsBloom),
			PrevRandao:       bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.PrevRandao),
			BlockNumber:      sourceLatestExecutionPayloadHeader.BlockNumber,
			GasLimit:         sourceLatestExecutionPayloadHeader.GasLimit,
			GasUsed:          sourceLatestExecutionPayloadHeader.GasUsed,
			Timestamp:        sourceLatestExecutionPayloadHeader.Timestamp,
			ExtraData:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.ExtraData),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.BaseFeePerGas),
			BlobGasUsed:      sourceLatestExecutionPayloadHeader.BlobGasUsed,
			ExcessBlobGas:    sourceLatestExecutionPayloadHeader.ExcessBlobGas,
			BlockHash:        bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.BlockHash),
			TransactionsRoot: bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.TransactionsRoot),
			WithdrawalsRoot:  bytesutil.SafeCopyBytes(sourceLatestExecutionPayloadHeader.WithdrawalsRoot),
		},
		NextWithdrawalIndex:          sourceNextWithdrawalIndex,
		NextWithdrawalValidatorIndex: sourceNextWithdrawalValIndex,
		HistoricalSummaries:          sourceHistoricalSummaries,
	}

	return result, nil
}

// V1Alpha1SignedContributionAndProofToV2 converts a v1alpha1 SignedContributionAndProof object to its v2 equivalent.
func V1Alpha1SignedContributionAndProofToV2(alphaContribution *ethpbalpha.SignedContributionAndProof) *ethpbv2.SignedContributionAndProof {
	result := &ethpbv2.SignedContributionAndProof{
		Message: &ethpbv2.ContributionAndProof{
			AggregatorIndex: alphaContribution.Message.AggregatorIndex,
			Contribution: &ethpbv2.SyncCommitteeContribution{
				Slot:              alphaContribution.Message.Contribution.Slot,
				BeaconBlockRoot:   alphaContribution.Message.Contribution.BlockRoot,
				SubcommitteeIndex: alphaContribution.Message.Contribution.SubcommitteeIndex,
				AggregationBits:   alphaContribution.Message.Contribution.AggregationBits,
				Signature:         alphaContribution.Message.Contribution.Signature,
			},
			SelectionProof: alphaContribution.Message.SelectionProof,
		},
		Signature: alphaContribution.Signature,
	}
	return result
}

// V2SignedBLSToExecutionChangeToV1Alpha1 converts a V2 SignedBLSToExecutionChange to its v1alpha1 equivalent.
func V2SignedBLSToExecutionChangeToV1Alpha1(change *ethpbv2.SignedBLSToExecutionChange) *ethpbalpha.SignedBLSToExecutionChange {
	return &ethpbalpha.SignedBLSToExecutionChange{
		Message: &ethpbalpha.BLSToExecutionChange{
			ValidatorIndex:     change.Message.ValidatorIndex,
			FromBlsPubkey:      bytesutil.SafeCopyBytes(change.Message.FromBlsPubkey),
			ToExecutionAddress: bytesutil.SafeCopyBytes(change.Message.ToExecutionAddress),
		},
		Signature: bytesutil.SafeCopyBytes(change.Signature),
	}
}

// V1Alpha1SignedBLSToExecChangeToV2 converts a v1alpha1 SignedBLSToExecutionChange object to its v2 equivalent.
func V1Alpha1SignedBLSToExecChangeToV2(alphaChange *ethpbalpha.SignedBLSToExecutionChange) *ethpbv2.SignedBLSToExecutionChange {
	result := &ethpbv2.SignedBLSToExecutionChange{
		Message: &ethpbv2.BLSToExecutionChange{
			ValidatorIndex:     alphaChange.Message.ValidatorIndex,
			FromBlsPubkey:      bytesutil.SafeCopyBytes(alphaChange.Message.FromBlsPubkey),
			ToExecutionAddress: bytesutil.SafeCopyBytes(alphaChange.Message.ToExecutionAddress),
		},
		Signature: bytesutil.SafeCopyBytes(alphaChange.Signature),
	}
	return result
}
