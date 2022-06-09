package types

type ValueUnion struct {
	Name string
}

func (vu *ValueUnion) TypeName() string {
	return vu.Name
}

func (vu *ValueUnion) PackagePath() string {
	panic("not implemented")
}

func (vu *ValueUnion) FixedSize() int {
	panic("not implemented")
}

func (vu *ValueUnion) IsVariableSized() bool {
	panic("not implemented")
}

var _ ValRep = &ValueUnion{}