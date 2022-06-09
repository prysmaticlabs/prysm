package sszgen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestFindStruct(t *testing.T) {
	packageName := "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	sourceFiles := []string{"testdata/types.pb.go"}
	pp, err := newTestPackageParser(packageName, sourceFiles)
	require.NoError(t, err)
	_, err = pp.GetType("BeaconState")
	require.NoError(t, err)
}

func newTestPackageParser(packageName string, files []string) (*packageParser, error) {
	pp := &packageParser{packagePath: packageName, files: make(map[string]*ast.File)}
	for _, f := range files {
		syn, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.AllErrors)
		if err != nil {
				return nil, err
			}
		pp.files[f] = syn
	}
	return pp, nil
}

func TestResolveImport(t *testing.T) {
	packageName := "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	sourceFiles := []string{"testdata/types.pb.go"}
	pp, err := newTestPackageParser(packageName, sourceFiles)
	require.NoError(t, err)
	ts, err := pp.GetType("BeaconState")
	require.NoError(t, err)
	alias := "github_com_prysmaticlabs_eth2_types"
	path, err := ts.FileParser.ResolveAlias(alias)
	require.NoError(t, err)
	expectedPath := "github.com/prysmaticlabs/eth2-types"
	require.Equal(t, expectedPath, path)
}
