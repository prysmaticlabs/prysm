package types

type ValRep interface {
	TypeName() string
	FixedSize() int
	PackagePath() string
	IsVariableSized() bool
}