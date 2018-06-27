package utils

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/internal"
)

func TestHandleServiceErrors(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)
	done := make(chan struct{})
	errChan := make(chan error)

	go HandleServiceErrors(done, errChan)

	errChan <- errors.New("something wrong")
	done <- struct{}{}
	h.VerifyLogMsg("something wrong")
}
