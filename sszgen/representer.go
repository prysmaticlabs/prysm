package sszgen

import (
	"fmt"
	"go/ast"

	"github.com/prysmaticlabs/prysm/sszgen/types"
)

type Representer struct {
	index *PackageIndex
}

func NewRepresenter(pi *PackageIndex) *Representer {
	return &Representer{index: pi}
}

type typeSpecMutator func(*ParseNode)

// this is used to copy a tag from a field down into a declaration
// representation. This is usedTag to push tag data down into declaration parsing,
// so that ssz-size/ssz-max can be applied to list/vector value types.
func typeSpecMutatorCopyTag(source *ParseNode) typeSpecMutator {
	return func(target *ParseNode) {
		target.Tag = source.Tag
	}
}

func (r *Representer) GetDeclaration(packagePath, structName string, mutators ...typeSpecMutator) (types.ValRep, error) {
	ts, err := r.index.GetType(packagePath, structName)
	if err != nil {
		return nil, err
	}
	// apply mutators to replicate any important ParseNode properties
	// from outer ParseNode
	for _, mut := range mutators {
		mut(ts)
	}
	switch ty := ts.typeSpec.Type.(type) {
	case *ast.StructType:
		vr := &types.ValueContainer{
			Name:     ts.Name,
			Package: packagePath,
		}
		for _, f := range ty.Fields.List {
			// this filters out internal protobuf fields, but also serializers like us
			// can safely ignore unexported fields in general. We also ignore embedded
			// fields because I'm not sure if we should support them yet.
			if f.Names == nil || !ast.IsExported(f.Names[0].Name) {
				continue
			}
			fieldName := f.Names[0].Name
			s := &ParseNode{
				FileParser:     ts.FileParser,
				PackageParser:  ts.PackageParser,
				typeExpression: f.Type,
			}
			if f.Tag != nil {
				s.Tag = f.Tag.Value
			}
			rep, err := r.expandRepresentation(s)
			if err != nil {
				return nil, err
			}
			vr.Contents = append(vr.Contents, types.ContainerField{fieldName, rep})
		}
		return vr, nil
	case *ast.Ident:
		// in this case our type is like an "overlay" over a primitive, ie
		// type IntWithMethods int
		// the ValueOverlay value type exists to represent this situation.
		// These values require some special handling in codegen because
		// they must be cast to/from their underlying types when working
		// with their byte representation for un/marshaling, etc
		underlying, err := r.expandIdent(ty, ts)
		if err != nil {
			return nil, err
		}
		// the underlying ValRep will be a primitive value and its .TypeName()
		// will reflect its storage type, not the overlay name
		return &types.ValueOverlay{Name: ts.Name, Package: packagePath, Underlying: underlying}, nil
	case *ast.ArrayType:
		// we can also have an "overlay" array, like the Bitlist types
		// from github.com/prysmaticlabs/go-bitfield
		//underlying, err := r.expandArray()
		underlying, err := r.expandArrayHead(ty, ts)
		if err != nil {
			return nil, err
		}
		return &types.ValueOverlay{Name: ts.Name, Package: packagePath, Underlying: underlying}, nil
	default:
		return nil, fmt.Errorf("Unsupported ast.Expr type for %v", ts.TypeExpression())
	}
}

func (r *Representer) expandRepresentation(ts *ParseNode) (types.ValRep, error) {
	switch ty := ts.typeExpression.(type) {
	case *ast.ArrayType:
		return r.expandArrayHead(ty, ts)
	case *ast.StarExpr:
		referentTS := &ParseNode{
			FileParser:     ts.FileParser,
			PackageParser:  ts.PackageParser,
			typeExpression: ty.X,
		}
		vr, err := r.expandRepresentation(referentTS)
		if err != nil {
			return nil, err
		}
		return &types.ValuePointer{Referent: vr}, nil
	case *ast.SelectorExpr:
		packageAliasIdent := ty.X.(*ast.Ident)
		pa := packageAliasIdent.Name
		path, err := ts.FileParser.ResolveAlias(pa)
		if err != nil {
			return nil, err
		}
		return r.GetDeclaration(path, ty.Sel.Name, typeSpecMutatorCopyTag(ts))
	case *ast.Ident:
		return r.expandIdent(ty, ts)
	default:
		return nil, fmt.Errorf("Unsupported ast.Expr type for %v", ts.TypeExpression())
	}
}

func (r *Representer) expandArrayHead(art *ast.ArrayType, ts *ParseNode) (types.ValRep, error) {
	dims, err := extractSSZDimensions(ts.Tag)
	if err != nil {
		return nil, err
	}
	return r.expandArray(dims, art, ts)
}

func (r *Representer) expandArray(dims []*SSZDimension, art *ast.ArrayType, ts *ParseNode) (types.ValRep, error) {
	if len(dims) == 0 {
		return nil, fmt.Errorf("do not have dimension information for type %v", ts)
	}
	d := dims[0]
	var elv types.ValRep
	var err error
	switch elt := art.Elt.(type) {
	case *ast.ArrayType:
		elv, err = r.expandArray(dims[1:], elt, ts)
		if err != nil {
			return nil, err
		}
	default:
		elv, err = r.expandRepresentation(&ParseNode{
			FileParser: ts.FileParser,
			PackageParser: ts.PackageParser,
			typeExpression: elt,
		})
		if err != nil {
			return nil, err
		}
	}

	if d.IsVector() {
		return &types.ValueVector{
			ElementValue: elv,
			Size: d.VectorLen(),
		}, nil
	}
	if d.IsList() {
		return &types.ValueList{
			ElementValue: elv,
			MaxSize: d.ListLen(),
		}, nil
	}
	return nil, nil
}

func (r *Representer) expandIdent(ident *ast.Ident, ts *ParseNode) (types.ValRep, error) {
	switch ident.Name {
	case "bool":
		return &types.ValueBool{Name: ident.Name}, nil
	case "byte":
		return &types.ValueByte{Name: ident.Name}, nil
	case "uint8":
		return &types.ValueUint{Size: 8, Name: ident.Name}, nil
	case "uint16":
		return &types.ValueUint{Size: 16, Name:ident.Name}, nil
	case "uint32":
		return &types.ValueUint{Size: 32, Name: ident.Name}, nil
	case "uint64":
		return &types.ValueUint{Size: 64, Name: ident.Name}, nil
	case "uint128":
		return &types.ValueUint{Size: 128, Name: ident.Name}, nil
	case "uint256":
		return &types.ValueUint{Size: 256, Name: ident.Name}, nil
	default:
		return r.GetDeclaration(ts.PackageParser.Path(), ident.Name, typeSpecMutatorCopyTag(ts))
	}
}
