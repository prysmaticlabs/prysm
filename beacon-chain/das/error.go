package das

import (
	"fmt"
	"strings"
)

func NewMissingIndicesError(missing []uint64) *MissingIndicesError {
	return &MissingIndicesError{indices: missing}
}

type MissingIndicesError struct {
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
