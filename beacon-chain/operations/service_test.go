package operations

import (
	"context"
	"errors"
	"testing"

	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = OperationFeeds(&Service{})
var _ = Pool(&Service{})

func TestStop_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	opsService := NewService(context.Background(), &Config{})

	if err := opsService.Stop(); err != nil {
		t.Fatalf("Unable to stop operation service: %v", err)
	}
	// The context should have been canceled.
	if opsService.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	hook.Reset()
}

func TestServiceStatus_Error(t *testing.T) {
	service := NewService(context.Background(), &Config{})
	if service.Status() != nil {
		t.Errorf("service status should be nil to begin with, got: %v", service.error)
	}
	err := errors.New("error error error")
	service.error = err

	if service.Status() != err {
		t.Error("service status did not return wanted err")
	}
}
