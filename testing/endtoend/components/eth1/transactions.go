package eth1

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	mathRand "math/rand"
	"time"

	"github.com/MariusVanDerWijden/FuzzyVM/filler"
	txfuzz "github.com/MariusVanDerWijden/tx-fuzz"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type TransactionGenerator struct {
	keystore string
	started  chan struct{}
}

func NewTransactionGenerator(keystore string) *TransactionGenerator {
	return &TransactionGenerator{keystore: keystore}
}

func (t *TransactionGenerator) Start(ctx context.Context) error {
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort))
	if err != nil {
		return err
	}
	seed := rand.NewDeterministicGenerator().Int63()
	logrus.Infof("Seed for transaction generator is: %d", seed)
	// Set seed so that all
	mathRand.Seed(seed)

	defer client.Close()

	keystoreBytes, err := ioutil.ReadFile(t.keystore) // #nosec G304
	if err != nil {
		return err
	}
	mineKey, err := keystore.DecryptKey(keystoreBytes, KeystorePassword)
	if err != nil {
		return err
	}
	rnd := make([]byte, 10000)
	// #nosec G404
	_, err = mathRand.Read(rnd)
	if err != nil {
		return err
	}
	f := filler.NewFiller(rnd)
	// Broadcast Transactions every 3 blocks
	txPeriod := time.Duration(params.BeaconConfig().SecondsPerETH1Block*3) * time.Second
	ticker := time.NewTicker(txPeriod)
	gasPrice := big.NewInt(1e18)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := SendTransaction(client, mineKey.PrivateKey, f, gasPrice, mineKey.Address.String(), e2e.NumOfExecEngineTxs, false)
			if err != nil {
				return err
			}
		}
	}
}

// Started checks whether beacon node set is started and all nodes are ready to be queried.
func (s *TransactionGenerator) Started() <-chan struct{} {
	return s.started
}

func SendTransaction(client *rpc.Client, key *ecdsa.PrivateKey, f *filler.Filler, gasPrice *big.Int, addr string, N uint64, al bool) error {
	backend := ethclient.NewClient(client)

	sender := common.HexToAddress(addr)
	chainid, err := backend.ChainID(context.Background())
	if err != nil {
		return err
	}
	nonce, err := backend.NonceAt(context.Background(), sender, big.NewInt(-1))
	if err != nil {
		return err
	}

	g, _ := errgroup.WithContext(context.Background())
	for i := uint64(0); i < N; i++ {
		index := i
		g.Go(func() error {
			tx, err := txfuzz.RandomValidTx(client, f, sender, nonce+index, gasPrice, nil, al)
			if err != nil {
				return nil
			}
			signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainid), key)
			if err != nil {
				return nil
			}
			err = backend.SendTransaction(context.Background(), signedTx)
			return nil
		})

	}
	return g.Wait()
}
