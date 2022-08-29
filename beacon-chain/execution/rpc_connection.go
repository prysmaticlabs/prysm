package execution

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	contracts "github.com/prysmaticlabs/prysm/v3/contracts/deposit"
	"github.com/prysmaticlabs/prysm/v3/io/logs"
	"github.com/prysmaticlabs/prysm/v3/network"
	"github.com/prysmaticlabs/prysm/v3/network/authorization"
)

func (s *Service) setupExecutionClientConnections(ctx context.Context, currEndpoint network.Endpoint) error {
	client, err := s.newRPCClientWithAuth(ctx, currEndpoint)
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
	return nil
}

// Every N seconds, defined as a backoffPeriod, attempts to re-establish an execution client
// connection and if this does not work, we fallback to the next endpoint if defined.
func (s *Service) pollConnectionStatus(ctx context.Context) {
	// Use a custom logger to only log errors
	logCounter := 0
	errorLogger := func(err error, msg string) {
		if logCounter > logThreshold {
			log.WithError(err).Error(msg)
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
			currClient := s.rpcClient
			if err := s.setupExecutionClientConnections(ctx, s.cfg.currHttpEndpoint); err != nil {
				errorLogger(err, "Could not connect to execution client endpoint")
				continue
			}
			// Close previous client, if connection was successful.
			if currClient != nil {
				currClient.Close()
			}
			log.Infof("Connected to new endpoint: %s", logs.MaskCredentialsLogging(s.cfg.currHttpEndpoint.Url))
			return
		case <-s.ctx.Done():
			log.Debug("Received cancelled context,closing existing powchain service")
			return
		}
	}
}

// Forces to retry an execution client connection.
func (s *Service) retryExecutionClientConnection(ctx context.Context, err error) {
	s.runError = err
	s.updateConnectedETH1(false)
	// Back off for a while before redialing.
	time.Sleep(backOffPeriod)
	currClient := s.rpcClient
	if err := s.setupExecutionClientConnections(ctx, s.cfg.currHttpEndpoint); err != nil {
		s.runError = err
		return
	}
	// Close previous client, if connection was successful.
	if currClient != nil {
		currClient.Close()
	}
	// Reset run error in the event of a successful connection.
	s.runError = nil
}

// Initializes an RPC connection with authentication headers.
func (s *Service) newRPCClientWithAuth(ctx context.Context, endpoint network.Endpoint) (*gethRPC.Client, error) {
	// Need to handle ipc and http
	var client *gethRPC.Client
	u, err := url.Parse(endpoint.Url)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		client, err = gethRPC.DialHTTPWithClient(endpoint.Url, endpoint.HttpClient())
		if err != nil {
			return nil, err
		}
	case "", "ipc":
		client, err = gethRPC.DialIPC(ctx, endpoint.Url)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("no known transport for URL scheme %q", u.Scheme)
	}
	if endpoint.Auth.Method != authorization.None {
		header, err := endpoint.Auth.ToHeaderValue()
		if err != nil {
			return nil, err
		}
		client.SetHeader("Authorization", header)
	}
	for _, h := range s.cfg.headers {
		if h != "" {
			keyValue := strings.Split(h, "=")
			if len(keyValue) < 2 {
				log.Warnf("Incorrect HTTP header flag format. Skipping %v", keyValue[0])
				continue
			}
			client.SetHeader(keyValue[0], strings.Join(keyValue[1:], "="))
		}
	}
	return client, nil
}

// Checks the chain ID of the execution client to ensure
// it matches local parameters of what Prysm expects.
func ensureCorrectExecutionChain(ctx context.Context, client *ethclient.Client) error {
	cID, err := client.ChainID(ctx)
	if err != nil {
		return err
	}
	wantChainID := params.BeaconConfig().DepositChainID
	if cID.Uint64() != wantChainID {
		return fmt.Errorf("wanted chain ID %d, got %d", wantChainID, cID.Uint64())
	}
	return nil
}
