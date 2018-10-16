package metric

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func TestNewMetric(t *testing.T) {
	adapter := New()
	if adapter == nil {
		t.Error("Expected metric adapter")
	}

	i := 0
	h := adapter(func(p2p.Message) { i++ })
	h(p2p.Message{Ctx: context.Background()})
	if i != 1 {
		t.Errorf("Expected next handler to be called once, but it was called %d times", i)
	}
}
