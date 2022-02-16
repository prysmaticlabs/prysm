package powchain

import (
	"context"
	"errors"
	"testing"
	"time"

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
