package kv

import (
	"errors"
	"testing"
)

func TestWrappedSentinelError(t *testing.T) {
	e := ErrNotFoundOriginBlockRoot
	if !errors.Is(e, ErrNotFoundOriginBlockRoot) {
		t.Error("expected that a copy of ErrNotFoundOriginBlockRoot would have an is-a relationship")
	}

	outer := errors.New("wrapped error")
	e2 := DBError{Wraps: ErrNotFoundOriginBlockRoot, Outer: outer}
	if !errors.Is(e2, ErrNotFoundOriginBlockRoot) {
		t.Error("expected that errors.Is would know DBError wraps ErrNotFoundOriginBlockRoot")
	}

	// test that the innermost not found error is detected
	if !errors.Is(e2, ErrNotFound) {
		t.Error("expected that errors.Is would know ErrNotFoundOriginBlockRoot wraps ErrNotFound")
	}
}
