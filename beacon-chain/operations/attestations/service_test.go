package attestations

import (
	"context"
	"errors"
	"testing"
)

func TestStop_OK(t *testing.T) {
	s, err := NewService(context.Background(), &Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Unable to stop attestation pool service: %v", err)
	}

	if s.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
}

func TestStatus_Error(t *testing.T) {
	err := errors.New("bad bad bad")
	s := &Service{error: err}

	if err := s.Status(); err != s.error {
		t.Errorf("Wanted: %v, got: %v", s.error, s.Status())
	}
}
