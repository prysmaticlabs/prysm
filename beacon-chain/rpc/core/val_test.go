package core

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
)

func TestX(t *testing.T) {
	var e error
	e = &AggregateBroadcastFailedError{err: errors.New("foo")}

	ok := errors.As(e, &AggregateBroadcastFailedError{})
	assert.Equal(t, true, ok)
}
