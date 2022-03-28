package powchain

import (
	"context"
	"errors"
	"testing"
	"time"

	mockChain2 "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1/mocks"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_checkTransitionConfiguration(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = 0
	params.OverrideBeaconConfig(cfg)
	hook := logTest.NewGlobal()

	t.Run("context canceled", func(t *testing.T) {
		ctx := context.Background()
		m := &mocks.EngineClient{}
		m.Err = errors.New("something went wrong")

		mockChain := &mockChain2.MockStateNotifier{}
		srv := &Service{
			cfg: &config{stateNotifier: mockChain},
		}
		srv.engineAPIClient = m
		checkTransitionPollingInterval = time.Millisecond
		ctx, cancel := context.WithCancel(ctx)
		go srv.checkTransitionConfiguration(ctx, make(chan *statefeed.BlockProcessedData, 1))
		<-time.After(100 * time.Millisecond)
		cancel()
		require.LogsContain(t, hook, "Could not check configuration values")
	})

	t.Run("block containing execution payload exits routine", func(t *testing.T) {
		ctx := context.Background()
		m := &mocks.EngineClient{}
		m.Err = errors.New("something went wrong")

		mockChain := &mockChain2.MockStateNotifier{}
		srv := &Service{
			cfg: &config{stateNotifier: mockChain},
		}
		srv.engineAPIClient = m
		checkTransitionPollingInterval = time.Millisecond
		ctx, cancel := context.WithCancel(ctx)
		exit := make(chan bool)
		notification := make(chan *statefeed.BlockProcessedData)
		go func() {
			srv.checkTransitionConfiguration(ctx, notification)
			exit <- true
		}()
		payload := emptyPayload()
		payload.GasUsed = 21000
		wrappedBlock, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{
			Block: &ethpb.BeaconBlockBellatrix{
				Body: &ethpb.BeaconBlockBodyBellatrix{
					ExecutionPayload: payload,
				},
			}},
		)
		require.NoError(t, err)
		notification <- &statefeed.BlockProcessedData{
			SignedBlock: wrappedBlock,
		}
		<-exit
		cancel()
		require.LogsContain(t, hook, "PoS transition is complete, no longer checking")
	})
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
func emptyPayload() *enginev1.ExecutionPayload {
	return &enginev1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		ExtraData:     make([]byte, 0),
	}
}
