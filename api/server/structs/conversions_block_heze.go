package structs

import (
	"fmt"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/api/server"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

// ----------------------------------------------------------------------------
// Heze
// ----------------------------------------------------------------------------

func SignedBeaconBlockHezeFromConsensus(b *eth.SignedBeaconBlockHeze) (*SignedBeaconBlockHeze, error) {
	block, err := BeaconBlockHezeFromConsensus(b.Block)
	if err != nil {
		return nil, fmt.Errorf("convert Heze block: %w", err)
	}
	return &SignedBeaconBlockHeze{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BeaconBlockHezeFromConsensus(b *eth.BeaconBlockHeze) (*BeaconBlockHeze, error) {
	payloadAttestations := make([]*PayloadAttestation, len(b.Body.PayloadAttestations))
	for i, pa := range b.Body.PayloadAttestations {
		payloadAttestations[i] = PayloadAttestationFromConsensus(pa)
	}

	return &BeaconBlockHeze{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyHeze{
			RandaoReveal:              hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:                  Eth1DataFromConsensus(b.Body.Eth1Data),
			Graffiti:                  hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings:         ProposerSlashingsFromConsensus(b.Body.ProposerSlashings),
			AttesterSlashings:         AttesterSlashingsGloasFromConsensus(b.Body.AttesterSlashings),
			Attestations:              AttsGloasFromConsensus(b.Body.Attestations),
			Deposits:                  DepositsFromConsensus(b.Body.Deposits),
			VoluntaryExits:            SignedExitsFromConsensus(b.Body.VoluntaryExits),
			SyncAggregate:             SyncAggregateFromConsensus(b.Body.SyncAggregate),
			BLSToExecutionChanges:     SignedBLSChangesFromConsensus(b.Body.BlsToExecutionChanges),
			SignedExecutionPayloadBid: SignedExecutionPayloadBidHezeFromConsensus(b.Body.SignedExecutionPayloadBid),
			PayloadAttestations:       payloadAttestations,
			ParentExecutionRequests:   ExecutionRequestsGloasFromConsensus(b.Body.ParentExecutionRequests),
		},
	}, nil
}

func (b *SignedBeaconBlockHeze) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}
	signed, err := b.ToConsensus()
	if err != nil {
		return nil, fmt.Errorf("convert block to consensus: %w", err)
	}
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Heze{Heze: signed},
	}, nil
}

func (b *SignedBeaconBlockHeze) ToConsensus() (*eth.SignedBeaconBlockHeze, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := bytesutil.DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "Signature")
	}
	block, err := b.Message.ToConsensus()
	if err != nil {
		return nil, server.NewDecodeError(err, "Message")
	}
	return &eth.SignedBeaconBlockHeze{
		Block:     block,
		Signature: sig,
	}, nil
}

func (b *BeaconBlockHeze) ToConsensus() (*eth.BeaconBlockHeze, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, server.NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, server.NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, server.NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.SignedExecutionPayloadBid == nil {
		return nil, server.NewDecodeError(errNilValue, "Body.SignedExecutionPayloadBid")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, server.NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, server.NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := bytesutil.DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := bytesutil.DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "StateRoot")
	}
	body, err := b.Body.ToConsensus()
	if err != nil {
		return nil, server.NewDecodeError(err, "Body")
	}
	return &eth.BeaconBlockHeze{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body:          body,
	}, nil
}

func (b *BeaconBlockBodyHeze) ToConsensus() (*eth.BeaconBlockBodyHeze, error) {
	if b == nil {
		return nil, errNilValue
	}

	randaoReveal, err := bytesutil.DecodeHexWithLength(b.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "RandaoReveal")
	}
	depositRoot, err := bytesutil.DecodeHexWithLength(b.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, server.NewDecodeError(err, "Eth1Data.DepositCount")
	}
	blockHash, err := bytesutil.DecodeHexWithLength(b.Eth1Data.BlockHash, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "Eth1Data.BlockHash")
	}
	graffiti, err := bytesutil.DecodeHexWithLength(b.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.ProposerSlashings)
	if err != nil {
		return nil, server.NewDecodeError(err, "ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsGloasToConsensus(b.AttesterSlashings)
	if err != nil {
		return nil, server.NewDecodeError(err, "AttesterSlashings")
	}
	atts, err := AttsGloasToConsensus(b.Attestations)
	if err != nil {
		return nil, server.NewDecodeError(err, "Attestations")
	}
	deposits, err := DepositsToConsensus(b.Deposits)
	if err != nil {
		return nil, server.NewDecodeError(err, "Deposits")
	}
	exits, err := SignedExitsToConsensus(b.VoluntaryExits)
	if err != nil {
		return nil, server.NewDecodeError(err, "VoluntaryExits")
	}
	syncCommitteeBits, err := bytesutil.DecodeHexWithLength(b.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := bytesutil.DecodeHexWithLength(b.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "SyncAggregate.SyncCommitteeSignature")
	}
	blsChanges, err := SignedBLSChangesToConsensus(b.BLSToExecutionChanges)
	if err != nil {
		return nil, server.NewDecodeError(err, "BLSToExecutionChanges")
	}
	signedBid, err := b.SignedExecutionPayloadBid.ToConsensusHeze()
	if err != nil {
		return nil, server.NewDecodeError(err, "SignedExecutionPayloadBid")
	}
	payloadAttestations, err := PayloadAttestationsToConsensus(b.PayloadAttestations)
	if err != nil {
		return nil, server.NewDecodeError(err, "PayloadAttestations")
	}
	var parentExecutionRequests *enginev1.ExecutionRequestsGloas
	if b.ParentExecutionRequests != nil {
		parentExecutionRequests, err = b.ParentExecutionRequests.ToConsensus()
		if err != nil {
			return nil, server.NewDecodeError(err, "ParentExecutionRequests")
		}
	}

	return &eth.BeaconBlockBodyHeze{
		RandaoReveal: randaoReveal,
		Eth1Data: &eth.Eth1Data{
			DepositRoot:  depositRoot,
			DepositCount: depositCount,
			BlockHash:    blockHash,
		},
		Graffiti:          graffiti,
		ProposerSlashings: proposerSlashings,
		AttesterSlashings: attesterSlashings,
		Attestations:      atts,
		Deposits:          deposits,
		VoluntaryExits:    exits,
		SyncAggregate: &eth.SyncAggregate{
			SyncCommitteeBits:      syncCommitteeBits,
			SyncCommitteeSignature: syncCommitteeSig,
		},
		BlsToExecutionChanges:     blsChanges,
		SignedExecutionPayloadBid: signedBid,
		PayloadAttestations:       payloadAttestations,
		ParentExecutionRequests:   parentExecutionRequests,
	}, nil
}

func (b *BeaconBlockHeze) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, errors.Wrap(err, "could not convert heze block to consensus")
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Heze{Heze: block}}, nil
}

// BlockContentsHezeFromConsensus converts a proto Heze block, envelope, and
// blob data to the API struct.
func BlockContentsHezeFromConsensus(block *eth.BeaconBlockHeze, envelope *eth.ExecutionPayloadEnvelopeHeze, kzgProofs [][]byte, blobs [][]byte) (*BlockContentsHeze, error) {
	b, err := BeaconBlockHezeFromConsensus(block)
	if err != nil {
		return nil, fmt.Errorf("convert Heze block: %w", err)
	}
	// The block is unsigned at contents-production time, so the beacon block
	// root is not yet known; the envelope is bound to the bundled block
	// (EIP-8411: the root travels out of band).
	env, err := ExecutionPayloadEnvelopeHezeFromConsensus(envelope, [fieldparams.RootLength]byte{})
	if err != nil {
		return nil, fmt.Errorf("convert execution payload envelope: %w", err)
	}
	encodedProofs := make([]string, len(kzgProofs))
	for i, p := range kzgProofs {
		encodedProofs[i] = hexutil.Encode(p)
	}
	encodedBlobs := make([]string, len(blobs))
	for i, b := range blobs {
		encodedBlobs[i] = hexutil.Encode(b)
	}
	return &BlockContentsHeze{
		Block:                    b,
		ExecutionPayloadEnvelope: env,
		KzgProofs:                encodedProofs,
		Blobs:                    encodedBlobs,
	}, nil
}

// SignedExecutionPayloadBidHezeFromConsensus converts a Heze signed bid proto to the API struct.
// The shared bid struct carries payload_chunks_root, populated only for Heze bids (EIP-8411).
func SignedExecutionPayloadBidHezeFromConsensus(b *eth.SignedExecutionPayloadBidHeze) *SignedExecutionPayloadBid {
	return &SignedExecutionPayloadBid{
		Message:   ExecutionPayloadBidHezeFromConsensus(b.Message),
		Signature: hexutil.Encode(b.Signature),
	}
}

// ExecutionPayloadBidHezeFromConsensus converts a Heze bid proto to the API struct.
func ExecutionPayloadBidHezeFromConsensus(b *eth.ExecutionPayloadBidHeze) *ExecutionPayloadBid {
	blobKzgCommitments := make([]string, len(b.BlobKzgCommitments))
	for i, commitment := range b.BlobKzgCommitments {
		blobKzgCommitments[i] = hexutil.Encode(commitment)
	}
	return &ExecutionPayloadBid{
		ParentBlockHash:       hexutil.Encode(b.ParentBlockHash),
		ParentBlockRoot:       hexutil.Encode(b.ParentBlockRoot),
		BlockHash:             hexutil.Encode(b.BlockHash),
		PrevRandao:            hexutil.Encode(b.PrevRandao),
		FeeRecipient:          hexutil.Encode(b.FeeRecipient),
		GasLimit:              fmt.Sprintf("%d", b.GasLimit),
		BuilderIndex:          fmt.Sprintf("%d", b.BuilderIndex),
		Slot:                  fmt.Sprintf("%d", b.Slot),
		Value:                 fmt.Sprintf("%d", b.Value),
		ExecutionPayment:      fmt.Sprintf("%d", b.ExecutionPayment),
		BlobKzgCommitments:    blobKzgCommitments,
		ExecutionRequestsRoot: hexutil.Encode(b.ExecutionRequestsRoot),
		PayloadChunksRoot:     hexutil.Encode(b.PayloadChunksRoot),
	}
}

// ToConsensusHeze converts the API struct to a Heze signed bid proto.
func (s *SignedExecutionPayloadBid) ToConsensusHeze() (*eth.SignedExecutionPayloadBidHeze, error) {
	if s == nil {
		return nil, server.NewDecodeError(errNilValue, "SignedExecutionPayloadBid")
	}
	msg, err := s.Message.ToConsensusHeze()
	if err != nil {
		return nil, server.NewDecodeError(err, "Message")
	}
	sig, err := bytesutil.DecodeHexWithLength(s.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "Signature")
	}
	return &eth.SignedExecutionPayloadBidHeze{Message: msg, Signature: sig}, nil
}

// ToConsensusHeze converts the API struct to a Heze bid proto (EIP-8411).
func (b *ExecutionPayloadBid) ToConsensusHeze() (*eth.ExecutionPayloadBidHeze, error) {
	gloasBid, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	payloadChunksRoot, err := bytesutil.DecodeHexWithLength(b.PayloadChunksRoot, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "PayloadChunksRoot")
	}
	return &eth.ExecutionPayloadBidHeze{
		ParentBlockHash:       gloasBid.ParentBlockHash,
		ParentBlockRoot:       gloasBid.ParentBlockRoot,
		BlockHash:             gloasBid.BlockHash,
		PrevRandao:            gloasBid.PrevRandao,
		FeeRecipient:          gloasBid.FeeRecipient,
		GasLimit:              gloasBid.GasLimit,
		BuilderIndex:          gloasBid.BuilderIndex,
		Slot:                  gloasBid.Slot,
		Value:                 gloasBid.Value,
		ExecutionPayment:      gloasBid.ExecutionPayment,
		BlobKzgCommitments:    gloasBid.BlobKzgCommitments,
		ExecutionRequestsRoot: gloasBid.ExecutionRequestsRoot,
		PayloadChunksRoot:     payloadChunksRoot,
	}, nil
}

// ExecutionPayloadEnvelopeHezeFromConsensus converts a Heze envelope proto to the API struct.
// The Heze container carries no beacon block root (EIP-8411); the binding travels out of band
// and surfaces in the shared struct's beacon_block_root field.
func ExecutionPayloadEnvelopeHezeFromConsensus(e *eth.ExecutionPayloadEnvelopeHeze, beaconBlockRoot [fieldparams.RootLength]byte) (*ExecutionPayloadEnvelope, error) {
	payload, err := ExecutionPayloadGloasFromConsensus(e.Payload)
	if err != nil {
		return nil, err
	}
	var requests *ExecutionRequestsGloas
	if e.ExecutionRequests != nil {
		requests = ExecutionRequestsGloasFromConsensus(e.ExecutionRequests)
	}
	return &ExecutionPayloadEnvelope{
		Payload:               payload,
		ExecutionRequests:     requests,
		BuilderIndex:          fmt.Sprintf("%d", e.BuilderIndex),
		BeaconBlockRoot:       hexutil.Encode(beaconBlockRoot[:]),
		ParentBeaconBlockRoot: hexutil.Encode(e.ParentBeaconBlockRoot),
	}, nil
}

// SignedExecutionPayloadEnvelopeHezeFromConsensus converts a signed Heze envelope proto to the API struct.
func SignedExecutionPayloadEnvelopeHezeFromConsensus(e *eth.SignedExecutionPayloadEnvelopeHeze, beaconBlockRoot [fieldparams.RootLength]byte) (*SignedExecutionPayloadEnvelope, error) {
	envelope, err := ExecutionPayloadEnvelopeHezeFromConsensus(e.Message, beaconBlockRoot)
	if err != nil {
		return nil, err
	}
	return &SignedExecutionPayloadEnvelope{
		Message:   envelope,
		Signature: hexutil.Encode(e.Signature),
	}, nil
}

// ToConsensusHeze converts the API struct to a Heze envelope proto plus the out-of-band
// beacon block root carried in the struct.
func (e *ExecutionPayloadEnvelope) ToConsensusHeze() (*eth.ExecutionPayloadEnvelopeHeze, [fieldparams.RootLength]byte, error) {
	if e == nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(errNilValue, "ExecutionPayloadEnvelope")
	}
	payload, err := e.Payload.ToConsensus()
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "Payload")
	}
	var requests *enginev1.ExecutionRequestsGloas
	if e.ExecutionRequests != nil {
		requests, err = e.ExecutionRequests.ToConsensus()
		if err != nil {
			return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "ExecutionRequests")
		}
	}
	builderIndex, err := strconv.ParseUint(e.BuilderIndex, 10, 64)
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "BuilderIndex")
	}
	beaconBlockRoot, err := bytesutil.DecodeHexWithLength(e.BeaconBlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "BeaconBlockRoot")
	}
	parentBeaconBlockRoot, err := bytesutil.DecodeHexWithLength(e.ParentBeaconBlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "ParentBeaconBlockRoot")
	}
	return &eth.ExecutionPayloadEnvelopeHeze{
		Payload:               payload,
		ExecutionRequests:     requests,
		BuilderIndex:          primitives.BuilderIndex(builderIndex),
		ParentBeaconBlockRoot: parentBeaconBlockRoot,
	}, [fieldparams.RootLength]byte(beaconBlockRoot), nil
}

// ToConsensusHeze converts the API struct to a signed Heze envelope proto plus the
// out-of-band beacon block root.
func (e *SignedExecutionPayloadEnvelope) ToConsensusHeze() (*eth.SignedExecutionPayloadEnvelopeHeze, [fieldparams.RootLength]byte, error) {
	if e == nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(errNilValue, "SignedExecutionPayloadEnvelope")
	}
	msg, root, err := e.Message.ToConsensusHeze()
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "Message")
	}
	sig, err := bytesutil.DecodeHexWithLength(e.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "Signature")
	}
	return &eth.SignedExecutionPayloadEnvelopeHeze{
		Message:   msg,
		Signature: sig,
	}, root, nil
}

// SignedExecutionPayloadEnvelopeContentsHezeFromConsensus builds the API struct used for
// stateless Heze envelope publishing from native components.
func SignedExecutionPayloadEnvelopeContentsHezeFromConsensus(signed *eth.SignedExecutionPayloadEnvelopeHeze, beaconBlockRoot [fieldparams.RootLength]byte, kzgProofs [][]byte, blobs [][]byte) (*SignedExecutionPayloadEnvelopeContents, error) {
	signedJSON, err := SignedExecutionPayloadEnvelopeHezeFromConsensus(signed, beaconBlockRoot)
	if err != nil {
		return nil, err
	}
	encodedProofs := make([]string, len(kzgProofs))
	for i, p := range kzgProofs {
		encodedProofs[i] = hexutil.Encode(p)
	}
	encodedBlobs := make([]string, len(blobs))
	for i, b := range blobs {
		encodedBlobs[i] = hexutil.Encode(b)
	}
	return &SignedExecutionPayloadEnvelopeContents{
		SignedExecutionPayloadEnvelope: signedJSON,
		KzgProofs:                      encodedProofs,
		Blobs:                          encodedBlobs,
	}, nil
}

// ToConsensusHeze decodes the API struct into the signed Heze envelope, its out-of-band
// beacon block root, and the raw blob and KZG proof bytes used by the stateless publish path.
func (c *SignedExecutionPayloadEnvelopeContents) ToConsensusHeze() (*eth.SignedExecutionPayloadEnvelopeHeze, [fieldparams.RootLength]byte, [][]byte, [][]byte, error) {
	if c == nil {
		return nil, [fieldparams.RootLength]byte{}, nil, nil, server.NewDecodeError(errNilValue, "SignedExecutionPayloadEnvelopeContents")
	}
	signed, root, err := c.SignedExecutionPayloadEnvelope.ToConsensusHeze()
	if err != nil {
		return nil, [fieldparams.RootLength]byte{}, nil, nil, server.NewDecodeError(err, "SignedExecutionPayloadEnvelope")
	}
	proofs := make([][]byte, len(c.KzgProofs))
	for i, p := range c.KzgProofs {
		proof, err := bytesutil.DecodeHexWithLength(p, 48)
		if err != nil {
			return nil, [fieldparams.RootLength]byte{}, nil, nil, server.NewDecodeError(err, fmt.Sprintf("KzgProofs[%d]", i))
		}
		proofs[i] = proof
	}
	blobs := make([][]byte, len(c.Blobs))
	for i, b := range c.Blobs {
		blob, err := bytesutil.DecodeHexWithLength(b, fieldparams.BlobSize)
		if err != nil {
			return nil, [fieldparams.RootLength]byte{}, nil, nil, server.NewDecodeError(err, fmt.Sprintf("Blobs[%d]", i))
		}
		blobs[i] = blob
	}
	return signed, root, proofs, blobs, nil
}
