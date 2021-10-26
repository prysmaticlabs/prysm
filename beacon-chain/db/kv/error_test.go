package kv

import (
	"errors"
	"testing"
)

func TestWrappedSentinelError(t *testing.T) {
	e := ErrNotFoundOriginCheckpoint
	if !errors.Is(e, ErrNotFoundOriginCheckpoint) {
		t.Error("expected that a copy of ErrNotFoundOriginCheckpoint would have an is-a relationship")
	}

	outer := errors.New("wrapped error")
	e2 := DBError{Wraps: ErrNotFoundOriginCheckpoint, Outer: outer}
	if !errors.Is(e2, ErrNotFoundOriginCheckpoint) {
		t.Error("expected that errors.Is would know DBError wraps ErrNotFoundOriginCheckpoint")
	}

	// test that the innermost not found error is detected
	if !errors.Is(e2, ErrNotFound) {
		t.Error("expected that errors.Is would know ErrNotFoundOriginCheckpoint wraps ErrNotFound")
	}
}
