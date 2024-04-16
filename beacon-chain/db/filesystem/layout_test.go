package filesystem

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

type mockLayout struct {
	pruneBeforeFunc func(primitives.Epoch) (*pruneSummary, error)
}

func (m *mockLayout) dir(n blobIdent) string {
	return ""
}

func (m *mockLayout) sszPath(n blobIdent) string {
	return ""
}

func (m *mockLayout) partPath(n blobIdent, entropy string) string {
	return ""
}

func (m *mockLayout) iterateIdents(before primitives.Epoch) (*identIterator, error) {
	return nil, nil
}

func (m *mockLayout) ident(root [32]byte, idx uint64) (blobIdent, error) {
	return blobIdent{}, nil
}

func (m *mockLayout) dirIdent(root [32]byte) (blobIdent, error) {
	return blobIdent{}, nil
}

func (m *mockLayout) summary(root [32]byte) BlobStorageSummary {
	return BlobStorageSummary{}
}

func (m *mockLayout) notify(blobIdent) error {
	return nil
}

func (m *mockLayout) pruneBefore(before primitives.Epoch) (*pruneSummary, error) {
	return m.pruneBeforeFunc(before)
}

func (m *mockLayout) remove(ident blobIdent) (int, error) {
	return 0, nil
}

var _ fsLayout = &mockLayout{}
