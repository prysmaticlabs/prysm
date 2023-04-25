package consensus_types

import (
	"errors"
	"fmt"

	errors2 "github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

var (
	// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
	ErrNilObjectWrapped = errors.New("attempted to wrap nil object")
	// ErrUnsupportedGetter is returned when a getter access is not supported for a specific beacon block version.
	ErrUnsupportedGetter = errors.New("unsupported getter")
)

func ErrNotSupported(funcName string, ver int) error {
	return errors2.Wrap(ErrUnsupportedGetter, fmt.Sprintf("%s is not supported for %s", funcName, version.String(ver)))
}
