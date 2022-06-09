package types

type ValuePointer struct {
	Referent ValRep
}

func (vp *ValuePointer) TypeName() string {
	return "*" + vp.Referent.TypeName()
}

func (vp *ValuePointer) PackagePath() string {
	return vp.Referent.PackagePath()
}

func (vp *ValuePointer) FixedSize() int {
	return vp.Referent.FixedSize()
}

func (vp *ValuePointer) IsVariableSized() bool {
	return vp.Referent.IsVariableSized()
}

var _ ValRep = &ValuePointer{}