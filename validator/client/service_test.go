package client

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&ValidatorService{})

func TestStop_cancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	vs := &ValidatorService{
		ctx:    ctx,
		cancel: cancel,
	}

	if err := vs.Stop(); err != nil {
		t.Error(err)
	}

	select {
	case <-time.After(1 * time.Second):
		t.Error("ctx not cancelled within 1s")
	case <-vs.ctx.Done():
	}
}
