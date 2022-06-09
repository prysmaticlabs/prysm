package backend

import "github.com/prysmaticlabs/prysm/sszgen/types"

type visitor func(vr types.ValRep)

func visit(vr types.ValRep, v visitor) {
	v(vr)
	switch t := vr.(type) {
	case *types.ValueContainer:
		for _, f := range t.Contents {
			visit(f.Value, v)
		}
	case *types.ValueVector:
		visit(t.ElementValue, v)
	case *types.ValueList:
		visit(t.ElementValue, v)
	case *types.ValuePointer:
		visit(t.Referent, v)
	case *types.ValueOverlay:
		visit(t.Underlying, v)
	}
}