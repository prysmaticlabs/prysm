package builder

import (
	"math/big"

	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

// SignedBid is an interface describing the method set of a signed builder bid.
type SignedBid interface {
	Message() (Bid, error)
	Signature() []byte
	Version() int
	IsNil() bool
}

// Bid is an interface describing the method set of a builder bid.
type Bid interface {
	Header() (interfaces.ExecutionData, error)
	Value() []byte
	Pubkey() []byte
	Version() int
	IsNil() bool
	HashTreeRoot() ([32]byte, error)
	HashTreeRootWith(hh *ssz.Hasher) error
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
func (b signedBuilderBid) Message() (Bid, error) {
	return WrappedBuilderBid(b.p.Message)
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
func (b signedBuilderBidCapella) Message() (Bid, error) {
	return WrappedBuilderBidCapella(b.p.Message)
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

// HashTreeRoot --
func (b builderBid) HashTreeRoot() ([32]byte, error) {
	return b.p.HashTreeRoot()
}

// HashTreeRootWith --
func (b builderBid) HashTreeRootWith(hh *ssz.Hasher) error {
	return b.p.HashTreeRootWith(hh)
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

// Header returns the execution data interface.
func (b builderBidCapella) Header() (interfaces.ExecutionData, error) {
	if b.p == nil {
		return nil, errors.New("builder bid is nil")
	}
	// We have to convert big endian to little endian because the value is coming from the execution layer.
	v := big.NewInt(0).SetBytes(bytesutil.ReverseByteOrder(b.p.Value))
	return blocks.WrappedExecutionPayloadHeaderCapella(b.p.Header, math.WeiToGwei(v))
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

// HashTreeRoot --
func (b builderBidCapella) HashTreeRoot() ([32]byte, error) {
	return b.p.HashTreeRoot()
}

// HashTreeRootWith --
func (b builderBidCapella) HashTreeRootWith(hh *ssz.Hasher) error {
	return b.p.HashTreeRootWith(hh)
}
