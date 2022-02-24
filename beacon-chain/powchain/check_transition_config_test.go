package powchain

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1/mocks"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_checkTransitionConfiguration(t *testing.T) {
	ctx := context.Background()
	hook := logTest.NewGlobal()

	m := &mocks.EngineClient{}
	m.Err = errors.New("something went wrong")

	srv := &Service{}
	srv.engineAPIClient = m
	checkTransitionPollingInterval = time.Millisecond
	ctx, cancel := context.WithCancel(ctx)
	go srv.checkTransitionConfiguration(ctx)
	<-time.After(100 * time.Millisecond)
	cancel()
	require.LogsContain(t, hook, "Could not check configuration values")
}

func TestService_handleExchangeConfigurationError(t *testing.T) {
	hook := logTest.NewGlobal()
	t.Run("clears existing service error", func(t *testing.T) {
		srv := &Service{isRunning: true}
		srv.runError = v1.ErrConfigMismatch
		srv.handleExchangeConfigurationError(nil)
		require.Equal(t, true, srv.Status() == nil)
	})
	t.Run("does not clear existing service error if wrong kind", func(t *testing.T) {
		srv := &Service{isRunning: true}
		err := errors.New("something else went wrong")
		srv.runError = err
		srv.handleExchangeConfigurationError(nil)
		require.ErrorIs(t, err, srv.Status())
	})
	t.Run("sets service error on config mismatch", func(t *testing.T) {
		srv := &Service{isRunning: true}
		srv.handleExchangeConfigurationError(v1.ErrConfigMismatch)
		require.Equal(t, v1.ErrConfigMismatch, srv.Status())
		require.LogsContain(t, hook, configMismatchLog)
	})
	t.Run("does not set service error if unrelated problem", func(t *testing.T) {
		srv := &Service{isRunning: true}
		srv.handleExchangeConfigurationError(errors.New("foo"))
		require.Equal(t, true, srv.Status() == nil)
		require.LogsContain(t, hook, "Could not check configuration values")
	})
}
