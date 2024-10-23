package builder

import (
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
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
	BlobKzgCommitments() ([][]byte, error)
	Value() primitives.Wei
	Pubkey() []byte
	Version() int
	IsNil() bool
	HashTreeRoot() ([32]byte, error)
	HashTreeRootWith(hh *ssz.Hasher) error
}

// BidElectra is an interface that exposes the newly added execution requests on top of the builder bid
type BidElectra interface {
	Bid
	ExecutionRequests() (*v1.ExecutionRequests, error)
}

type signedBuilderBid struct {
	p *ethpb.SignedBuilderBid
}

// WrappedSignedBuilderBid is a constructor which wraps a protobuf signed bit into an interface.
func WrappedSignedBuilderBid(p *ethpb.SignedBuilderBid) (SignedBid, error) {
	w := signedBuilderBid{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
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
		return nil, consensus_types.ErrNilObjectWrapped
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
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Header --
func (b builderBid) Header() (interfaces.ExecutionData, error) {
	return blocks.WrappedExecutionPayloadHeader(b.p.Header)
}

// BlobKzgCommitments --
func (b builderBid) BlobKzgCommitments() ([][]byte, error) {
	return [][]byte{}, errors.New("blob kzg commitments not available before Deneb")
}

// Version --
func (b builderBid) Version() int {
	return version.Bellatrix
}

// Value --
func (b builderBid) Value() primitives.Wei {
	return primitives.LittleEndianBytesToWei(b.p.Value)
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
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Header returns the execution data interface.
func (b builderBidCapella) Header() (interfaces.ExecutionData, error) {
	// We have to convert big endian to little endian because the value is coming from the execution layer.
	return blocks.WrappedExecutionPayloadHeaderCapella(b.p.Header)
}

// BlobKzgCommitments --
func (b builderBidCapella) BlobKzgCommitments() ([][]byte, error) {
	return [][]byte{}, errors.New("blob kzg commitments not available before Deneb")
}

// Version --
func (b builderBidCapella) Version() int {
	return version.Capella
}

// Value --
func (b builderBidCapella) Value() primitives.Wei {
	return primitives.LittleEndianBytesToWei(b.p.Value)
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

type builderBidDeneb struct {
	p *ethpb.BuilderBidDeneb
}

// WrappedBuilderBidDeneb is a constructor which wraps a protobuf bid into an interface.
func WrappedBuilderBidDeneb(p *ethpb.BuilderBidDeneb) (Bid, error) {
	w := builderBidDeneb{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Version --
func (b builderBidDeneb) Version() int {
	return version.Deneb
}

// Value --
func (b builderBidDeneb) Value() primitives.Wei {
	return primitives.LittleEndianBytesToWei(b.p.Value)
}

// Pubkey --
func (b builderBidDeneb) Pubkey() []byte {
	return b.p.Pubkey
}

// IsNil --
func (b builderBidDeneb) IsNil() bool {
	return b.p == nil
}

// HashTreeRoot --
func (b builderBidDeneb) HashTreeRoot() ([32]byte, error) {
	return b.p.HashTreeRoot()
}

// HashTreeRootWith --
func (b builderBidDeneb) HashTreeRootWith(hh *ssz.Hasher) error {
	return b.p.HashTreeRootWith(hh)
}

// Header --
func (b builderBidDeneb) Header() (interfaces.ExecutionData, error) {
	// We have to convert big endian to little endian because the value is coming from the execution layer.
	return blocks.WrappedExecutionPayloadHeaderDeneb(b.p.Header)
}

// BlobKzgCommitments --
func (b builderBidDeneb) BlobKzgCommitments() ([][]byte, error) {
	return b.p.BlobKzgCommitments, nil
}

type signedBuilderBidDeneb struct {
	p *ethpb.SignedBuilderBidDeneb
}

// WrappedSignedBuilderBidDeneb is a constructor which wraps a protobuf signed bit into an interface.
func WrappedSignedBuilderBidDeneb(p *ethpb.SignedBuilderBidDeneb) (SignedBid, error) {
	w := signedBuilderBidDeneb{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Message --
func (b signedBuilderBidDeneb) Message() (Bid, error) {
	return WrappedBuilderBidDeneb(b.p.Message)
}

// Signature --
func (b signedBuilderBidDeneb) Signature() []byte {
	return b.p.Signature
}

// Version --
func (b signedBuilderBidDeneb) Version() int {
	return version.Deneb
}

// IsNil --
func (b signedBuilderBidDeneb) IsNil() bool {
	return b.p == nil
}

type builderBidElectra struct {
	p *ethpb.BuilderBidElectra
}

// WrappedBuilderBidElectra is a constructor which wraps a protobuf bid into an interface.
func WrappedBuilderBidElectra(p *ethpb.BuilderBidElectra) (Bid, error) {
	w := builderBidElectra{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Version --
func (b builderBidElectra) Version() int {
	return version.Electra
}

// Value --
func (b builderBidElectra) Value() primitives.Wei {
	return primitives.LittleEndianBytesToWei(b.p.Value)
}

// Pubkey --
func (b builderBidElectra) Pubkey() []byte {
	return b.p.Pubkey
}

// IsNil --
func (b builderBidElectra) IsNil() bool {
	return b.p == nil
}

// HashTreeRoot --
func (b builderBidElectra) HashTreeRoot() ([32]byte, error) {
	return b.p.HashTreeRoot()
}

// HashTreeRootWith --
func (b builderBidElectra) HashTreeRootWith(hh *ssz.Hasher) error {
	return b.p.HashTreeRootWith(hh)
}

// Header --
func (b builderBidElectra) Header() (interfaces.ExecutionData, error) {
	// We have to convert big endian to little endian because the value is coming from the execution layer.
	return blocks.WrappedExecutionPayloadHeaderElectra(b.p.Header)
}

// ExecutionRequests --
func (b builderBidElectra) ExecutionRequests() (*v1.ExecutionRequests, error) {
	if b.p == nil {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	if b.p.ExecutionRequests == nil {
		return nil, errors.New("ExecutionRequests is nil")
	}
	return b.p.ExecutionRequests, nil
}

// BlobKzgCommitments --
func (b builderBidElectra) BlobKzgCommitments() ([][]byte, error) {
	return b.p.BlobKzgCommitments, nil
}

type signedBuilderBidElectra struct {
	p *ethpb.SignedBuilderBidElectra
}

// WrappedSignedBuilderBidElectra is a constructor which wraps a protobuf signed bit into an interface.
func WrappedSignedBuilderBidElectra(p *ethpb.SignedBuilderBidElectra) (SignedBid, error) {
	w := signedBuilderBidElectra{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Message --
func (b signedBuilderBidElectra) Message() (Bid, error) {
	return WrappedBuilderBidElectra(b.p.Message)
}

// Signature --
func (b signedBuilderBidElectra) Signature() []byte {
	return b.p.Signature
}

// Version --
func (b signedBuilderBidElectra) Version() int {
	return version.Electra
}

// IsNil --
func (b signedBuilderBidElectra) IsNil() bool {
	return b.p == nil
}
