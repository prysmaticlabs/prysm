package tracer

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func TestNewTracer(t *testing.T) {
	a, err := New("test", "test:1234", 0.25, false)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if a == nil {
		t.Error("Expected tracing adapter")
	}

	a, err = New("test", "127.0.0.1:43755", 0.25, true)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if a == nil {
		t.Error("Expected tracing adapter")
	}

	i := 0
	h := a(func(p2p.Message) { i++ })
	h(p2p.Message{Ctx: context.Background()})
	if i != 1 {
		t.Error("Expected next handler to be called")
	}
}

func TestNewTracerBad(t *testing.T) {
	_, err := New("", "test:1234", 0.25, true)
	if err == nil {
		t.Errorf("Expected error with empty name")
	}

	_, err = New("test", "", 0.25, true)
	if err == nil {
		t.Errorf("Expected error with empty endpoint")
	}
}
