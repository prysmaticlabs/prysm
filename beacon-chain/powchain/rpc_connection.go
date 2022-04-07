package powchain

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/io/logs"
	"github.com/prysmaticlabs/prysm/network"
	"github.com/prysmaticlabs/prysm/network/authorization"
)

func (s *Service) setupExecutionClientConnections(ctx context.Context) error {
	client, err := newRPCClientWithAuth(s.cfg.currHttpEndpoint)
	if err != nil {
		return errors.Wrap(err, "could not dial execution node")
	}
	fetcher := ethclient.NewClient(client)
	depositContractCaller, err := contracts.NewDepositContractCaller(s.cfg.depositContractAddr, fetcher)
	if err != nil {
		client.Close()
		return errors.Wrap(err, "could not initialize deposit contract caller")
	}
	if err := ensureCorrectExecutionChain(ctx, fetcher); err != nil {
		client.Close()
		return errors.Wrap(err, "could not make initial request to verify execution chain ID")
	}
	s.rpcClient = client
	s.httpLogger = fetcher
	s.eth1DataFetcher = fetcher
	s.depositContractCaller = depositContractCaller
	s.updateConnectedETH1(true)
	s.runError = nil
	return nil
}

func newRPCClientWithAuth(endpoint network.Endpoint) (*gethRPC.Client, error) {
	client, err := gethRPC.Dial(endpoint.Url)
	if err != nil {
		return nil, err
	}
	if endpoint.Auth.Method != authorization.None {
		header, err := endpoint.Auth.ToHeaderValue()
		if err != nil {
			return nil, err
		}
		client.SetHeader("Authorization", header)
	}
	return client, nil
}

func ensureCorrectExecutionChain(ctx context.Context, client *ethclient.Client) error {
	cID, err := client.ChainID(ctx)
	if err != nil {
		return err
	}
	nID, err := client.NetworkID(ctx)
	if err != nil {
		return err
	}
	wantChainID := params.BeaconConfig().DepositChainID
	wantNetworkID := params.BeaconConfig().DepositNetworkID
	if cID.Uint64() != wantChainID {
		return fmt.Errorf("wanted chain ID %d, got %d", wantChainID, cID.Uint64())
	}
	if nID.Uint64() != wantNetworkID {
		return fmt.Errorf("wanted network ID %d, got %d", wantNetworkID, nID.Uint64())
	}
	return nil
}

func (s *Service) pollConnectionStatus() {
	// Use a custom logger to only log errors
	logCounter := 0
	errorLogger := func(err error, msg string) {
		if logCounter > logThreshold {
			log.Errorf("%s: %v", msg, err)
			logCounter = 0
		}
		logCounter++
	}
	ticker := time.NewTicker(backOffPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Debugf("Trying to dial endpoint: %s", logs.MaskCredentialsLogging(s.cfg.currHttpEndpoint.Url))
			errConnect := s.connectToPowChain()
			if errConnect != nil {
				errorLogger(errConnect, "Could not connect to powchain endpoint")
				s.runError = errConnect
				s.fallbackToNextEndpoint()
				continue
			}
		case <-s.ctx.Done():
			log.Debug("Received cancelled context,closing existing powchain service")
			return
		}
	}
}

// Reconnect to eth1 node in case of any failure.
func (s *Service) retryETH1Node(err error) {
	s.runError = err
	s.updateConnectedETH1(false)
	// Back off for a while before
	// resuming dialing the eth1 node.
	time.Sleep(backOffPeriod)
	if err := s.connectToPowChain(); err != nil {
		s.runError = err
		return
	}
	// Reset run error in the event of a successful connection.
	s.runError = nil
}
