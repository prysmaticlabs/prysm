package sszgen

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func newTestIndexer() *PackageIndex {
	return &PackageIndex{
		index: make(map[string]PackageParser),
		structCache: make(map[[2]string]*ParseNode),
	}
}

func TestAddGet(t *testing.T) {
	packageName := "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pi := newTestIndexer()
	sourceFiles := []string{"testdata/simple.go"}
	pp, err := newTestPackageParser(packageName, sourceFiles)
	require.NoError(t, err)
	pi.index[packageName] = pp
	parser, err := pi.getParser(packageName)
	require.NoError(t, err)
	_, err = parser.GetType("NoImports")
	require.NoError(t, err)
	_, err = pi.GetType(packageName, "NoImports")
	require.NoError(t, err)
}
