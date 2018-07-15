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
	"github.com/prysmaticlabs/geth-sharding/contracts"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	var keystoreUTCPath string
	var ipcPath string
	var passwordFile string
	var httpPath string
	var privKeyString string

	app := cli.NewApp()
	app.Name = "deployVRC"
	app.Usage = "this is a util to deploy validator registration contract"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "keystoreUTCPath",
			Usage:       "Location of keystore",
			Destination: &keystoreUTCPath,
		},
		cli.StringFlag{
			Name:        "ipcPath",
			Usage:       "Filename for IPC socket/pipe within the datadir",
			Destination: &ipcPath,
		},
		cli.StringFlag{
			Name:        "httpPath",
			Value:       "http://localhost:8545/",
			Usage:       "HTTP-RPC server listening interface",
			Destination: &httpPath,
		},
		cli.StringFlag{
			Name:        "passwordFile",
			Value:       "./password.txt",
			Usage:       "Password file for unlock account",
			Destination: &passwordFile,
		},
		cli.StringFlag{
			Name:        "privKey",
			Usage:       "Private key to unlock account",
			Destination: &privKeyString,
		},
	}

	app.Action = func(c *cli.Context) {
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
			log.Fatal(err)
		}

		client := ethclient.NewClient(rpcClient)

		// User inputs private key, sign tx with private key
		if privKeyString != "" {
			privKey, err := crypto.HexToECDSA(privKeyString)
			if err != nil {
				log.Fatal(err)
			}
			txOps = bind.NewKeyedTransactor(privKey)
			txOps.Value = big.NewInt(0)

			// User inputs keystore json file, sign tx with keystore json
		} else {
			file, err := os.Open(passwordFile)
			if err != nil {
				log.Fatal(err)
			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanWords)
			scanner.Scan()
			password := scanner.Text()

			keyJSON, _ := ioutil.ReadFile(keystoreUTCPath)
			privKey, err := keystore.DecryptKey(keyJSON, password)
			if err != nil {
				log.Fatal(err)
			}

			txOps = bind.NewKeyedTransactor(privKey.PrivateKey)
			txOps.Value = big.NewInt(0)
		}

		// Deploy validator registration contract
		addr, tx, _, err := contracts.DeployValidatorRegistration(txOps, client)
		if err != nil {
			log.Fatal(err)
		}

		// Wait for contract to mine
		for pending := true; pending; _, pending, err = client.TransactionByHash(context.Background(), tx.Hash()) {
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(1 * time.Second)
		}

		log.Infof("New contract deployed at %s", addr.Hex())
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
