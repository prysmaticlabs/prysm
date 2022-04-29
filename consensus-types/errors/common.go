package typeerrors

import "github.com/pkg/errors"

var (
	// ErrUnsupportedField is returned when a field is not supported by a specific beacon block type.
	// This allows us to create a generic beacon block interface that is implemented by different
	// fork versions of beacon blocks.
	ErrUnsupportedField = errors.New("unsupported field for block type")
	// ErrUnsupportedSignedBeaconBlock is returned when the struct type is not a supported signed
	// beacon block type.
	ErrUnsupportedSignedBeaconBlock = errors.New("unsupported signed beacon block")
	// ErrUnsupportedBeaconBlock is returned when the struct type is not a supported beacon block
	// type.
	ErrUnsupportedBeaconBlock = errors.New("unsupported beacon block")
	// ErrUnsupportedPhase0Block is returned when accessing a phase0 block from a non-phase0 wrapped
	// block.
	ErrUnsupportedPhase0Block = errors.New("unsupported phase0 block")
	// ErrUnsupportedAltairBlock is returned when accessing an altair block from non-altair wrapped
	// block.
	ErrUnsupportedAltairBlock = errors.New("unsupported altair block")
	// ErrUnsupportedBellatrixBlock is returned when accessing a bellatrix block from a non-bellatrix wrapped
	// block.
	ErrUnsupportedBellatrixBlock = errors.New("unsupported bellatrix block")
	// ErrUnsupportedBlindedBellatrixBlock is returned when accessing a blinded bellatrix block from unsupported method.
	ErrUnsupportedBlindedBellatrixBlock = errors.New("unsupported blinded bellatrix block")
	// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
	ErrNilObjectWrapped = errors.New("attempted to wrap nil object")
)
