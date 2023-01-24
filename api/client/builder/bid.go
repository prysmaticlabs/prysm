package builder

import (
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

type SignedBid interface {
	Message() Bid
	Signature() []byte
	Version() int
	IsNil() bool
}

type Bid interface {
	Header() (interfaces.ExecutionData, error)
	Value() []byte
	Pubkey() []byte
	Version() int
	IsNil() bool
}

type signedBuilderBid struct {
	p *ethpb.SignedBuilderBid
}

// WrappedSignedBuilderBid is a constructor which wraps a protobuf signed bit into an interface.
func WrappedSignedBuilderBid(p *ethpb.SignedBuilderBid) (SignedBid, error) {
	w := signedBuilderBid{p: p}
	if w.IsNil() {
		return nil, blocks.ErrNilObjectWrapped
	}
	return w, nil
}

// Message --
func (b signedBuilderBid) Message() Bid {
	return b.Message()
}

// Signature --
func (b signedBuilderBid) Signature() []byte {
	return b.p.Signature
}

// Version --
func (b signedBuilderBid) Version() int {
	return version.Bellatrix
}

// IsNil --
func (b signedBuilderBid) IsNil() bool {
	return b.p == nil
}

type signedBuilderBidCapella struct {
	p *ethpb.SignedBuilderBidCapella
}

// WrappedSignedBuilderBidCapella is a constructor which wraps a protobuf signed bit into an interface.
func WrappedSignedBuilderBidCapella(p *ethpb.SignedBuilderBidCapella) (SignedBid, error) {
	w := signedBuilderBidCapella{p: p}
	if w.IsNil() {
		return nil, blocks.ErrNilObjectWrapped
	}
	return w, nil
}

// Message --
func (b signedBuilderBidCapella) Message() Bid {
	return b.Message()
}

// Signature --
func (b signedBuilderBidCapella) Signature() []byte {
	return b.p.Signature
}

// Version --
func (b signedBuilderBidCapella) Version() int {
	return version.Capella
}

// IsNil --
func (b signedBuilderBidCapella) IsNil() bool {
	return b.p == nil
}

type builderBid struct {
	p *ethpb.BuilderBid
}

// WrappedBuilderBid is a constructor which wraps a protobuf bid into an interface.
func WrappedBuilderBid(p *ethpb.BuilderBid) (Bid, error) {
	w := builderBid{p: p}
	if w.IsNil() {
		return nil, blocks.ErrNilObjectWrapped
	}
	return w, nil
}

// Header --
func (b builderBid) Header() (interfaces.ExecutionData, error) {
	return blocks.WrappedExecutionPayloadHeader(b.p.Header)
}

// Version --
func (b builderBid) Version() int {
	return version.Bellatrix
}

// Value --
func (b builderBid) Value() []byte {
	return b.p.Value
}

// Pubkey --
func (b builderBid) Pubkey() []byte {
	return b.p.Pubkey
}

// IsNil --
func (b builderBid) IsNil() bool {
	return b.p == nil
}

type builderBidCapella struct {
	p *ethpb.BuilderBidCapella
}

// WrappedBuilderBidCapella is a constructor which wraps a protobuf bid into an interface.
func WrappedBuilderBidCapella(p *ethpb.BuilderBidCapella) (Bid, error) {
	w := builderBidCapella{p: p}
	if w.IsNil() {
		return nil, blocks.ErrNilObjectWrapped
	}
	return w, nil
}

// Header --
func (b builderBidCapella) Header() (interfaces.ExecutionData, error) {
	return blocks.WrappedExecutionPayloadHeaderCapella(b.p.Header)
}

// Version --
func (b builderBidCapella) Version() int {
	return version.Capella
}

// Value --
func (b builderBidCapella) Value() []byte {
	return b.p.Value
}

// Pubkey --
func (b builderBidCapella) Pubkey() []byte {
	return b.p.Pubkey
}

// IsNil --
func (b builderBidCapella) IsNil() bool {
	return b.p == nil
}
