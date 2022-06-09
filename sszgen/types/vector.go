package types

type ValueVector struct {
	ElementValue ValRep
	Size int
}

func (vv *ValueVector) TypeName() string {
	return "[]" + vv.ElementValue.TypeName()
}

func (vv *ValueVector) FixedSize() int {
	return vv.Size * vv.ElementValue.FixedSize()
}

func (vv *ValueVector) PackagePath() string {
	return vv.ElementValue.PackagePath()
}

func (vv *ValueVector) IsVariableSized() bool {
	return vv.ElementValue.IsVariableSized()
}

var _ ValRep = &ValueVector{}