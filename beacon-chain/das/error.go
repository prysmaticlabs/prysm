package das

import (
	"fmt"
	"strings"
)

func NewMissingIndicesError(root [32]byte, missing []uint64) *MissingIndicesError {
	return &MissingIndicesError{root: root, indices: missing}
}

type MissingIndicesError struct {
	root    [32]byte
	indices []uint64
}

func (m *MissingIndicesError) Error() string {
	is := make([]string, 0, len(m.indices))
	for i := range m.indices {
		is = append(is, fmt.Sprintf("%d", m.indices[i]))
	}
	return fmt.Sprintf("%s missing blob indices %s", errDAIncomplete.Error(), strings.Join(is, ","))
}

func (m *MissingIndicesError) Missing() []uint64 {
	return m.indices
}

var _ error = &MissingIndicesError{}
