package types

type UintSize int

const (
	Uint8 UintSize = 8
	Uint16 UintSize = 16
	Uint32 UintSize = 32
	Uint64 UintSize = 64
	Uint128 UintSize = 128
	Uint256 UintSize = 256
)

type ValueUint struct {
	Name string
	Size UintSize
	Package string
}

func (vu *ValueUint) TypeName() string {
	return vu.Name
}

func (vu *ValueUint) PackagePath() string {
	return vu.Package
}

func (vu *ValueUint) FixedSize() int {
	return int(vu.Size)/8
}

func (vu *ValueUint) IsVariableSized() bool {
	return false
}

var _ ValRep = &ValueUint{}