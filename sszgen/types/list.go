package types

type ValueList struct {
	ElementValue ValRep
	MaxSize int
}

func (vl *ValueList) TypeName() string {
	return "[]" + vl.ElementValue.TypeName()
}

func (vl *ValueList) PackagePath() string {
	return vl.ElementValue.PackagePath()
}

func (vl *ValueList) FixedSize() int {
	return 4
}

func (vl *ValueList) IsVariableSized() bool {
	return true
}

var _ ValRep = &ValueList{}
