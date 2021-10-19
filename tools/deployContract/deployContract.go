package main

import (
	"bufio"
	"context"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	var keystoreUTCPath string
	var ipcPath string
	var passwordFile string
	var httpPath string
	var privKeyString string
	var drainAddress string

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	log := logrus.WithField("prefix", "main")

	app := cli.App{}
	app.Name = "deployDepositContract"
	app.Usage = "this is a util to deploy deposit contract"
	app.Version = version.Version()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "keystoreUTCPath",
			Usage:       "Location of keystore",
			Destination: &keystoreUTCPath,
		},
		&cli.StringFlag{
			Name:        "ipcPath",
			Usage:       "Filename for IPC socket/pipe within the datadir",
			Destination: &ipcPath,
		},
		&cli.StringFlag{
			Name:        "httpPath",
			Value:       "http://localhost:8545/",
			Usage:       "HTTP-RPC server listening interface",
			Destination: &httpPath,
		},
		&cli.StringFlag{
			Name:        "passwordFile",
			Value:       "./password.txt",
			Usage:       "Password file for unlock account",
			Destination: &passwordFile,
		},
		&cli.StringFlag{
			Name:        "privKey",
			Usage:       "Private key to unlock account",
			Destination: &privKeyString,
		},
		&cli.StringFlag{
			Name:        "drainAddress",
			Value:       "",
			Usage:       "The drain address to specify in the contract. The default will be msg.sender",
			Destination: &drainAddress,
		},
	}

	app.Action = func(c *cli.Context) error {
		// Set up RPC client
		var rpcClient *rpc.Client
		var err error
		var txOps *bind.TransactOpts

		// Uses HTTP-RPC if IPC is not set
		if ipcPath == "" {
			rpcClient, err = rpc.Dial(httpPath)
		} else {
			rpcClient, err = rpc.Dial(ipcPath)
		}
		if err != nil {
			return err
		}

		client := ethclient.NewClient(rpcClient)

		// User inputs private key, sign tx with private key
		if privKeyString != "" {
			privKey, err := crypto.HexToECDSA(privKeyString)
			if err != nil {
				log.Fatal(err)
			}
			txOps, err = bind.NewKeyedTransactorWithChainID(privKey, big.NewInt(1337))
			if err != nil {
				log.Fatal(err)
			}
			txOps.Value = big.NewInt(0)
			txOps.GasLimit = 4000000
			txOps.Context = context.Background()
			// User inputs keystore json file, sign tx with keystore json
		} else {
			// #nosec - Inclusion of file via variable is OK for this tool.
			file, err := os.Open(passwordFile)
			if err != nil {
				return err
			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanWords)
			scanner.Scan()
			password := scanner.Text()

			// #nosec - Inclusion of file via variable is OK for this tool.
			keyJSON, err := ioutil.ReadFile(keystoreUTCPath)
			if err != nil {
				return err
			}
			privKey, err := keystore.DecryptKey(keyJSON, password)
			if err != nil {
				return err
			}

			txOps, err = bind.NewKeyedTransactorWithChainID(privKey.PrivateKey, big.NewInt(1337))
			if err != nil {
				log.Fatal(err)
			}
			txOps.Value = big.NewInt(0)
			txOps.GasLimit = 4000000
			txOps.Context = context.Background()
		}

		txOps.GasPrice = big.NewInt(10 * 1e9 /* 10 gwei */)

		// Deploy validator registration contract
		addr, tx, _, err := contracts.DeployDepositContract(
			txOps,
			client,
		)

		if err != nil {
			return err
		}

		// Wait for contract to mine
		for pending := true; pending; _, pending, err = client.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
		}

		log.WithFields(logrus.Fields{
			"address": addr.Hex(),
		}).Info("New contract deployed")

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
