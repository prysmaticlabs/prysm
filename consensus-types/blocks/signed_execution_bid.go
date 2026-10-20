package blocks

import (
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	consensus_types "github.com/OffchainLabs/prysm/v7/consensus-types"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// signedExecutionPayloadBid wraps the protobuf signed execution payload bid
// and implements the ROSignedExecutionPayloadBid interface.
type signedExecutionPayloadBid struct {
	bid *ethpb.SignedExecutionPayloadBid
}

// executionPayloadBidGloas wraps the protobuf execution payload bid for Gloas fork
// and implements the ROExecutionPayloadBidGloas interface.
type executionPayloadBidGloas struct {
	payload *ethpb.ExecutionPayloadBid
}

// IsNil checks if the signed execution payload bid is nil or invalid.
func (s signedExecutionPayloadBid) IsNil() bool {
	if s.bid == nil {
		return true
	}

	if _, err := WrappedROExecutionPayloadBid(s.bid.Message); err != nil {
		return true
	}

	return len(s.bid.Signature) != 96
}

// IsNil checks if the execution payload bid is nil or has invalid fields.
func (h executionPayloadBidGloas) IsNil() bool {
	if h.payload == nil {
		return true
	}

	if len(h.payload.ParentBlockHash) != 32 ||
		len(h.payload.ParentBlockRoot) != 32 ||
		len(h.payload.BlockHash) != 32 ||
		len(h.payload.PrevRandao) != 32 ||
		len(h.payload.FeeRecipient) != 20 {
		return true
	}

	for _, commitment := range h.payload.BlobKzgCommitments {
		if len(commitment) != 48 {
			return true
		}
	}

	return false
}

// WrappedROSignedExecutionPayloadBid creates a new read-only signed execution payload bid
// wrapper from the given protobuf message.
func WrappedROSignedExecutionPayloadBid(pb *ethpb.SignedExecutionPayloadBid) (interfaces.ROSignedExecutionPayloadBid, error) {
	wrapper := signedExecutionPayloadBid{bid: pb}
	if wrapper.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return wrapper, nil
}

// WrappedROExecutionPayloadBid creates a new read-only execution payload bid
// wrapper for the Gloas fork from the given protobuf message.
func WrappedROExecutionPayloadBid(pb *ethpb.ExecutionPayloadBid) (interfaces.ROExecutionPayloadBid, error) {
	wrapper := executionPayloadBidGloas{payload: pb}
	if wrapper.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return wrapper, nil
}

// Bid returns the execution payload bid as a read-only interface.
func (s signedExecutionPayloadBid) Bid() (interfaces.ROExecutionPayloadBid, error) {
	return WrappedROExecutionPayloadBid(s.bid.Message)
}

// Proto returns the underlying protobuf message.
func (s signedExecutionPayloadBid) Proto() proto.Message {
	return s.bid
}

// HashTreeRoot returns the hash tree root of the signed bid container.
func (s signedExecutionPayloadBid) HashTreeRoot() ([32]byte, error) {
	return s.bid.HashTreeRoot()
}

// SigningRoot computes the signing root for the execution payload bid with the given domain.
func (s signedExecutionPayloadBid) SigningRoot(domain []byte) ([32]byte, error) {
	return signing.ComputeSigningRoot(s.bid.Message, domain)
}

// Signature returns the BLS signature as a 96-byte array.
func (s signedExecutionPayloadBid) Signature() [96]byte {
	return [96]byte(s.bid.Signature)
}

// ParentBlockHash returns the hash of the parent execution block.
func (h executionPayloadBidGloas) ParentBlockHash() [32]byte {
	return [32]byte(h.payload.ParentBlockHash)
}

// ParentBlockRoot returns the beacon block root of the parent block.
func (h executionPayloadBidGloas) ParentBlockRoot() [32]byte {
	return [32]byte(h.payload.ParentBlockRoot)
}

// PrevRandao returns the previous randao value for the execution block.
func (h executionPayloadBidGloas) PrevRandao() [32]byte {
	return [32]byte(h.payload.PrevRandao)
}

// BlockHash returns the hash of the execution block.
func (h executionPayloadBidGloas) BlockHash() [32]byte {
	return [32]byte(h.payload.BlockHash)
}

// GasLimit returns the gas limit for the execution block.
func (h executionPayloadBidGloas) GasLimit() uint64 {
	return h.payload.GasLimit
}

// BuilderIndex returns the builder index of the builder who created this bid.
func (h executionPayloadBidGloas) BuilderIndex() primitives.BuilderIndex {
	return h.payload.BuilderIndex
}

// Slot returns the beacon chain slot for which this bid was created.
func (h executionPayloadBidGloas) Slot() primitives.Slot {
	return h.payload.Slot
}

// Value returns the payment value offered by the builder in Gwei.
func (h executionPayloadBidGloas) Value() primitives.Gwei {
	return primitives.Gwei(h.payload.Value)
}

// ExecutionPayment returns the execution payment offered by the builder.
func (h executionPayloadBidGloas) ExecutionPayment() primitives.Gwei {
	return primitives.Gwei(h.payload.ExecutionPayment)
}

// BlobKzgCommitments returns the KZG commitments for blobs.
func (h executionPayloadBidGloas) BlobKzgCommitments() [][]byte {
	return bytesutil.SafeCopy2dBytes(h.payload.BlobKzgCommitments)
}

// BlobKzgCommitmentCount returns the number of blob KZG commitments.
func (h executionPayloadBidGloas) BlobKzgCommitmentCount() uint64 {
	return uint64(len(h.payload.BlobKzgCommitments))
}

// FeeRecipient returns the execution address that will receive the builder payment.
func (h executionPayloadBidGloas) FeeRecipient() [20]byte {
	return [20]byte(h.payload.FeeRecipient)
}

// PayloadChunksRoot returns the EIP-8411 commitment to the chunked envelope
// bytes. Gloas bids predate the field, so it is always zero here.
func (h executionPayloadBidGloas) PayloadChunksRoot() [32]byte {
	return [32]byte{}
}

// ExecutionRequestsRoot returns the hash tree root of the execution requests.
func (h executionPayloadBidGloas) ExecutionRequestsRoot() [32]byte {
	if h.payload == nil || len(h.payload.ExecutionRequestsRoot) < 32 {
		return [32]byte{}
	}
	return [32]byte(h.payload.ExecutionRequestsRoot)
}

// signedExecutionPayloadBidHeze wraps the Heze signed execution payload bid.
type signedExecutionPayloadBidHeze struct {
	bid *ethpb.SignedExecutionPayloadBidHeze
}

// executionPayloadBidHeze wraps the Heze execution payload bid (EIP-8411).
type executionPayloadBidHeze struct {
	payload *ethpb.ExecutionPayloadBidHeze
}

// IsNil checks if the signed execution payload bid is nil or invalid.
func (s signedExecutionPayloadBidHeze) IsNil() bool {
	if s.bid == nil {
		return true
	}
	if _, err := WrappedROExecutionPayloadBidHeze(s.bid.Message); err != nil {
		return true
	}
	return len(s.bid.Signature) != 96
}

// IsNil checks if the execution payload bid is nil or has invalid fields.
func (h executionPayloadBidHeze) IsNil() bool {
	if h.payload == nil {
		return true
	}
	if len(h.payload.ParentBlockHash) != 32 ||
		len(h.payload.ParentBlockRoot) != 32 ||
		len(h.payload.BlockHash) != 32 ||
		len(h.payload.PrevRandao) != 32 ||
		len(h.payload.FeeRecipient) != 20 ||
		len(h.payload.PayloadChunksRoot) != 32 {
		return true
	}
	for _, commitment := range h.payload.BlobKzgCommitments {
		if len(commitment) != 48 {
			return true
		}
	}
	return false
}

// WrappedROSignedExecutionPayloadBidHeze wraps a Heze signed bid read-only.
func WrappedROSignedExecutionPayloadBidHeze(pb *ethpb.SignedExecutionPayloadBidHeze) (interfaces.ROSignedExecutionPayloadBid, error) {
	wrapper := signedExecutionPayloadBidHeze{bid: pb}
	if wrapper.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return wrapper, nil
}

// WrappedROExecutionPayloadBidHeze wraps a Heze bid read-only.
func WrappedROExecutionPayloadBidHeze(pb *ethpb.ExecutionPayloadBidHeze) (interfaces.ROExecutionPayloadBid, error) {
	wrapper := executionPayloadBidHeze{payload: pb}
	if wrapper.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return wrapper, nil
}

// Bid returns the execution payload bid as a read-only interface.
func (s signedExecutionPayloadBidHeze) Bid() (interfaces.ROExecutionPayloadBid, error) {
	return WrappedROExecutionPayloadBidHeze(s.bid.Message)
}

// Proto returns the underlying protobuf message.
func (s signedExecutionPayloadBidHeze) Proto() proto.Message {
	return s.bid
}

// HashTreeRoot returns the hash tree root of the signed bid container.
func (s signedExecutionPayloadBidHeze) HashTreeRoot() ([32]byte, error) {
	return s.bid.HashTreeRoot()
}

// SigningRoot computes the signing root for the bid with the given domain.
func (s signedExecutionPayloadBidHeze) SigningRoot(domain []byte) ([32]byte, error) {
	return signing.ComputeSigningRoot(s.bid.Message, domain)
}

// Signature returns the BLS signature as a 96-byte array.
func (s signedExecutionPayloadBidHeze) Signature() [96]byte {
	return [96]byte(s.bid.Signature)
}

// ParentBlockHash returns the hash of the parent execution block.
func (h executionPayloadBidHeze) ParentBlockHash() [32]byte {
	return [32]byte(h.payload.ParentBlockHash)
}

// ParentBlockRoot returns the beacon block root of the parent block.
func (h executionPayloadBidHeze) ParentBlockRoot() [32]byte {
	return [32]byte(h.payload.ParentBlockRoot)
}

// PrevRandao returns the previous randao value for the execution block.
func (h executionPayloadBidHeze) PrevRandao() [32]byte {
	return [32]byte(h.payload.PrevRandao)
}

// BlockHash returns the hash of the execution block.
func (h executionPayloadBidHeze) BlockHash() [32]byte {
	return [32]byte(h.payload.BlockHash)
}

// GasLimit returns the gas limit for the execution block.
func (h executionPayloadBidHeze) GasLimit() uint64 {
	return h.payload.GasLimit
}

// BuilderIndex returns the builder index of the builder who created this bid.
func (h executionPayloadBidHeze) BuilderIndex() primitives.BuilderIndex {
	return h.payload.BuilderIndex
}

// Slot returns the beacon chain slot for which this bid was created.
func (h executionPayloadBidHeze) Slot() primitives.Slot {
	return h.payload.Slot
}

// Value returns the payment value offered by the builder in Gwei.
func (h executionPayloadBidHeze) Value() primitives.Gwei {
	return primitives.Gwei(h.payload.Value)
}

// ExecutionPayment returns the execution payment offered by the builder.
func (h executionPayloadBidHeze) ExecutionPayment() primitives.Gwei {
	return primitives.Gwei(h.payload.ExecutionPayment)
}

// BlobKzgCommitments returns the KZG commitments for blobs.
func (h executionPayloadBidHeze) BlobKzgCommitments() [][]byte {
	return bytesutil.SafeCopy2dBytes(h.payload.BlobKzgCommitments)
}

// BlobKzgCommitmentCount returns the number of blob KZG commitments.
func (h executionPayloadBidHeze) BlobKzgCommitmentCount() uint64 {
	return uint64(len(h.payload.BlobKzgCommitments))
}

// FeeRecipient returns the execution address that receives the builder payment.
func (h executionPayloadBidHeze) FeeRecipient() [20]byte {
	return [20]byte(h.payload.FeeRecipient)
}

// PayloadChunksRoot returns the EIP-8411 commitment to the chunked envelope bytes.
func (h executionPayloadBidHeze) PayloadChunksRoot() [32]byte {
	return [32]byte(h.payload.PayloadChunksRoot)
}

// ExecutionRequestsRoot returns the hash tree root of the execution requests.
func (h executionPayloadBidHeze) ExecutionRequestsRoot() [32]byte {
	if h.payload == nil || len(h.payload.ExecutionRequestsRoot) < 32 {
		return [32]byte{}
	}
	return [32]byte(h.payload.ExecutionRequestsRoot)
}
