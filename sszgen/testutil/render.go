package testutil

import (
	"go/format"

	jen "github.com/dave/jennifer/jen"
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

func RenderIntermediate(vr types.ValRep) (string, error){
	file := jen.NewFile("values")
	evr, err := encodeValRep(vr)
	if err != nil {
		return "", err
	}
	v := jen.Var().Id(vr.TypeName()).Id("types").Dot("ValRep").Op("=").Add(evr)
	file.Add(v)

	gs := file.GoString()
	b, err := format.Source([]byte(gs))
	return string(b), err
}

func encodeValRep(vr types.ValRep) (jen.Code, error) {
	var c jen.Code
	switch ty := vr.(type) {
	case *types.ValueByte:
		values := []jen.Code{jen.Id("Name").Op(":").Lit(ty.Name)}
		if ty.Package != "" {
			values = append(values, jen.Id("Package").Op(":").Lit(ty.Package))
		}
		s := jen.Op("&").Id("types").Dot("ValueByte").Values(values...)
		return s, nil
	case *types.ValueBool:
		values := []jen.Code{jen.Id("Name").Op(":").Lit(ty.Name)}
		if ty.Package != "" {
			values = append(values, jen.Id("Package").Op(":").Lit(ty.Package))
		}
		s := jen.Op("&").Id("types").Dot("ValueBool").Values(values...)
		return s, nil
	case *types.ValueUint:
		s := jen.Op("&").Id("types").Dot("ValueUint").Values(
			jen.Id("Name").Op(":").Lit(ty.Name),
			jen.Id("Size").Op(":").Lit(int(ty.Size)),
		)
		return s, nil
	case *types.ValueVector:
		ev, err := encodeValRep(ty.ElementValue)
		if err != nil {
			return nil, err
		}
		s := jen.Op("&").Id("types").Dot("ValueVector").Values(
			jen.Id("Size").Op(":").Lit(ty.Size),
			jen.Id("ElementValue").Op(":").Add(ev),
		)
		return s, nil
	case *types.ValueList:
		ev, err := encodeValRep(ty.ElementValue)
		if err != nil {
			return nil, err
		}
		s := jen.Op("&").Id("types").Dot("ValueList").Values(
			jen.Id("MaxSize").Op(":").Lit(ty.MaxSize),
			jen.Id("ElementValue").Op(":").Add(ev),
		)
		return s, nil
	case *types.ValueOverlay:
		underlying, err := encodeValRep(ty.Underlying)
		if err != nil {
			return nil, err
		}
		s := jen.Op("&").Id("types").Dot("ValueOverlay").Values(
			jen.Id("Name").Op(":").Lit(ty.Name),
			jen.Id("Package").Op(":").Lit(ty.Package),
			jen.Id("Underlying").Op(":").Add(underlying),
		)
		return s, nil
	case *types.ValuePointer:
		referent, err := encodeValRep(ty.Referent)
		if err != nil {
			return nil, err
		}
		s := jen.Op("&").Id("types").Dot("ValuePointer").Values(
			jen.Id("Referent").Op(":").Add(referent),
		)
		return s, nil
	case *types.ValueContainer:
		contents := make([]jen.Code, 0)
		for _, c := range ty.Contents {
			cvr, err := encodeValRep(c.Value)
			if err != nil {
				return nil, err
			}
			kv := jen.Values(jen.Id("Key").Op(":").Lit(c.Key),
				jen.Id("Value").Op(":").Add(cvr))
			contents = append(contents, kv)
		}
		fields := []jen.Code{
			jen.Id("Name").Op(":").Lit(ty.Name),
			jen.Id("Package").Op(":").Lit(ty.Package),
			jen.Id("Contents").Op(":").Index().Id("types").Dot("ContainerField").
				Values(contents...),
		}
		c = jen.Op("&").Id("types").Dot("ValueContainer").Values(fields...)
	case *types.ValueUnion:
		panic("not implemented")
	}
	return c, nil
}
