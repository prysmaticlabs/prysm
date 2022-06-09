package types

type ValueByte struct {
	Name string
	Package string
}

func (vb *ValueByte) TypeName() string {
	return vb.Name
}

func (vb *ValueByte) PackagePath() string {
	return vb.Package
}

func (vb *ValueByte) FixedSize() int {
	return 1
}

func (vb *ValueByte) IsVariableSized() bool {
	return false
}

var _ ValRep = &ValueByte{}