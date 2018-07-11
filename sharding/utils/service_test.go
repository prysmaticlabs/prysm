package utils

import (
	"errors"
	"testing"

	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestHandleServiceErrors(t *testing.T) {
	hook := logTest.NewGlobal()
	done := make(chan struct{})
	errChan := make(chan error)

	go HandleServiceErrors(done, errChan)

	errChan <- errors.New("something wrong")
	done <- struct{}{}
	msg := hook.LastEntry().Message
	want := "something wrong"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}
}
