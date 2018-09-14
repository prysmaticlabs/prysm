package tracer

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func TestNewTracer(t *testing.T) {
	a, err := New("test", "test:1234", true)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if a == nil {
		t.Error("Expected tracing adapter")
	}

	a, err = New("test", "127.0.0.1:43755", false)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if a == nil {
		t.Error("Expected tracing adapter")
	}

	i := 0
	h := a(func(context.Context, p2p.Message) { i++ })
	h(context.Background(), p2p.Message{})
	if i != 1 {
		t.Error("Expected next handler to be called")
	}
}

func TestNewTracerBad(t *testing.T) {
	_, err := New("", "test:1234", false)
	if err == nil {
		t.Errorf("Expected error with empty name")
	}

	_, err = New("test", "", false)
	if err == nil {
		t.Errorf("Expected error with empty endpoint")
	}
}
