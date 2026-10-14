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
)

// ExecutionPayloadEnvelopeFromConsensus converts a proto envelope to the API struct. The
// beacon block root is carried out of band since EIP-8411 removed it from the container;
// pass a zero root when the block is not yet known (e.g. pre-signing block contents).
func ExecutionPayloadEnvelopeFromConsensus(e *eth.ExecutionPayloadEnvelope, beaconBlockRoot [fieldparams.RootLength]byte) (*ExecutionPayloadEnvelope, error) {
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

// SignedExecutionPayloadEnvelopeFromConsensus converts a signed proto envelope to the API struct.
func SignedExecutionPayloadEnvelopeFromConsensus(e *eth.SignedExecutionPayloadEnvelope, beaconBlockRoot [fieldparams.RootLength]byte) (*SignedExecutionPayloadEnvelope, error) {
	envelope, err := ExecutionPayloadEnvelopeFromConsensus(e.Message, beaconBlockRoot)
	if err != nil {
		return nil, err
	}
	return &SignedExecutionPayloadEnvelope{
		Message:   envelope,
		Signature: hexutil.Encode(e.Signature),
	}, nil
}

// BlockContentsGloasFromConsensus converts a proto Gloas block, envelope, and
// blob data to the API struct.
func BlockContentsGloasFromConsensus(block *eth.BeaconBlockGloas, envelope *eth.ExecutionPayloadEnvelope, kzgProofs [][]byte, blobs [][]byte) (*BlockContentsGloas, error) {
	b, err := BeaconBlockGloasFromConsensus(block)
	if err != nil {
		return nil, err
	}
	// The block is unsigned at contents-production time, so the beacon block
	// root is not yet known; the envelope is bound to the bundled block.
	env, err := ExecutionPayloadEnvelopeFromConsensus(envelope, [fieldparams.RootLength]byte{})
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
	return &BlockContentsGloas{
		Block:                    b,
		ExecutionPayloadEnvelope: env,
		KzgProofs:                encodedProofs,
		Blobs:                    encodedBlobs,
	}, nil
}

// ToConsensus converts the API struct to a proto ExecutionPayloadEnvelope.
func (e *ExecutionPayloadEnvelope) ToConsensus() (*eth.ExecutionPayloadEnvelope, error) {
	if e == nil {
		return nil, server.NewDecodeError(errNilValue, "ExecutionPayloadEnvelope")
	}
	payload, err := e.Payload.ToConsensus()
	if err != nil {
		return nil, server.NewDecodeError(err, "Payload")
	}
	var requests *enginev1.ExecutionRequestsGloas
	if e.ExecutionRequests != nil {
		requests, err = e.ExecutionRequests.ToConsensus()
		if err != nil {
			return nil, server.NewDecodeError(err, "ExecutionRequests")
		}
	}
	builderIndex, err := strconv.ParseUint(e.BuilderIndex, 10, 64)
	if err != nil {
		return nil, server.NewDecodeError(err, "BuilderIndex")
	}
	parentBeaconBlockRoot, err := bytesutil.DecodeHexWithLength(e.ParentBeaconBlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "ParentBeaconBlockRoot")
	}
	return &eth.ExecutionPayloadEnvelope{
		Payload:               payload,
		ExecutionRequests:     requests,
		BuilderIndex:          primitives.BuilderIndex(builderIndex),
		ParentBeaconBlockRoot: parentBeaconBlockRoot,
	}, nil
}

// ParsedBeaconBlockRoot decodes the out-of-band beacon_block_root carried by the API struct.
// It returns a zero root when the field is empty.
func (e *ExecutionPayloadEnvelope) ParsedBeaconBlockRoot() ([fieldparams.RootLength]byte, error) {
	if e == nil || e.BeaconBlockRoot == "" {
		return [fieldparams.RootLength]byte{}, nil
	}
	root, err := bytesutil.DecodeHexWithLength(e.BeaconBlockRoot, fieldparams.RootLength)
	if err != nil {
		return [fieldparams.RootLength]byte{}, server.NewDecodeError(err, "BeaconBlockRoot")
	}
	return [fieldparams.RootLength]byte(root), nil
}

// ToConsensus converts the API struct to a proto SignedExecutionPayloadEnvelope.
func (e *SignedExecutionPayloadEnvelope) ToConsensus() (*eth.SignedExecutionPayloadEnvelope, error) {
	if e == nil {
		return nil, server.NewDecodeError(errNilValue, "SignedExecutionPayloadEnvelope")
	}
	msg, err := e.Message.ToConsensus()
	if err != nil {
		return nil, server.NewDecodeError(err, "Message")
	}
	sig, err := bytesutil.DecodeHexWithLength(e.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, server.NewDecodeError(err, "Signature")
	}
	return &eth.SignedExecutionPayloadEnvelope{
		Message:   msg,
		Signature: sig,
	}, nil
}

// SignedExecutionPayloadEnvelopeContentsFromConsensus builds the API struct
// used for stateless envelope publishing from native components.
func SignedExecutionPayloadEnvelopeContentsFromConsensus(signed *eth.SignedExecutionPayloadEnvelope, beaconBlockRoot [fieldparams.RootLength]byte, kzgProofs [][]byte, blobs [][]byte) (*SignedExecutionPayloadEnvelopeContents, error) {
	signedJSON, err := SignedExecutionPayloadEnvelopeFromConsensus(signed, beaconBlockRoot)
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

// ToConsensus decodes the API struct into the signed envelope plus raw blob and
// KZG proof bytes used by the stateless publish path.
func (c *SignedExecutionPayloadEnvelopeContents) ToConsensus() (*eth.SignedExecutionPayloadEnvelope, [][]byte, [][]byte, error) {
	if c == nil {
		return nil, nil, nil, server.NewDecodeError(errNilValue, "SignedExecutionPayloadEnvelopeContents")
	}
	signed, err := c.SignedExecutionPayloadEnvelope.ToConsensus()
	if err != nil {
		return nil, nil, nil, server.NewDecodeError(err, "SignedExecutionPayloadEnvelope")
	}
	proofs := make([][]byte, len(c.KzgProofs))
	for i, p := range c.KzgProofs {
		proof, err := bytesutil.DecodeHexWithLength(p, 48)
		if err != nil {
			return nil, nil, nil, server.NewDecodeError(err, fmt.Sprintf("KzgProofs[%d]", i))
		}
		proofs[i] = proof
	}
	blobs := make([][]byte, len(c.Blobs))
	for i, b := range c.Blobs {
		blob, err := bytesutil.DecodeHexWithLength(b, fieldparams.BlobSize)
		if err != nil {
			return nil, nil, nil, server.NewDecodeError(err, fmt.Sprintf("Blobs[%d]", i))
		}
		blobs[i] = blob
	}
	return signed, proofs, blobs, nil
}
