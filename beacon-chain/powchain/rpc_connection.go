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
	"github.com/sirupsen/logrus"
)

func (s *Service) setupExecutionClientConnections(ctx context.Context) error {
	client, err := newRPCClientWithAuth(s.cfg.currHttpEndpoint)
	if err != nil {
		return errors.Wrap(err, "could not dial execution node")
	}
	// Attach the clients to the service struct.
	fetcher := ethclient.NewClient(client)
	s.rpcClient = client
	s.httpLogger = fetcher
	s.eth1DataFetcher = fetcher

	depositContractCaller, err := contracts.NewDepositContractCaller(s.cfg.depositContractAddr, fetcher)
	if err != nil {
		client.Close()
		return errors.Wrap(err, "could not initialize deposit contract caller")
	}
	s.depositContractCaller = depositContractCaller

	// Ensure we have the correct chain and deposit IDs.
	if err := ensureCorrectExecutionChain(ctx, fetcher); err != nil {
		client.Close()
		return errors.Wrap(err, "could not make initial request to verify execution chain ID")
	}
	s.updateConnectedETH1(true)
	s.runError = nil

	log.WithFields(logrus.Fields{
		"endpoint": logs.MaskCredentialsLogging(s.cfg.currHttpEndpoint.Url),
	}).Info("Connected to Ethereum execution client RPC")
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

func (s *Service) pollConnectionStatus(ctx context.Context) {
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
			log.Warnf("Trying to dial endpoint: %s", logs.MaskCredentialsLogging(s.cfg.currHttpEndpoint.Url))
			if err := s.setupExecutionClientConnections(ctx); err != nil {
				log.Infof("GOT ISSUE DIALING: %v", err)
				errorLogger(err, "Could not connect to execution client endpoint")
				s.runError = err
				s.fallbackToNextEndpoint()
			}
			log.Info("Managed to dial")
		case <-s.ctx.Done():
			log.Debug("Received cancelled context,closing existing powchain service")
			return
		}
	}
}

//// Reconnect to eth1 node in case of any failure.
//func (s *Service) retryETH1Node(err error) {
//	s.runError = err
//	s.updateConnectedETH1(false)
//	// Back off for a while before
//	// resuming dialing the eth1 node.
//	time.Sleep(backOffPeriod)
//	if err := s.connectToPowChain(); err != nil {
//		s.runError = err
//		return
//	}
//	// Reset run error in the event of a successful connection.
//	s.runError = nil
//}
