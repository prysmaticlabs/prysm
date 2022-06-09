package types

type ValueBool struct {
	Name string
	Package string
}

func (vb *ValueBool) TypeName() string {
	return vb.Name
}

func (vb *ValueBool) PackagePath() string {
	return vb.Package
}

func (vb *ValueBool) FixedSize() int {
	return 1
}

func (vb *ValueBool) IsVariableSized() bool {
	return false
}

var _ ValRep = &ValueBool{}