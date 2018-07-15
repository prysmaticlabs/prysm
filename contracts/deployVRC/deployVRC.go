package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"time"
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/geth-sharding/contracts"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// ClientIdentifier tells us what client the node we interact with over RPC is running.
const ClientIdentifier = "geth"

func main() {
	var dataDirPath string
	var ipcPath string
	var passwordFile string
	var httpPath string

	app := cli.NewApp()
	app.Name = "deployVRC"
	app.Usage = "this is a util to deploy validator registration contract"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "dataDirPath",
			Value:       "./datadir",
			Usage:       "Data directory for the databases and keystore",
			Destination: &dataDirPath,
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
	}

	app.Action = func(c *cli.Context) {
		// Set up RPC client
		var rpcClient *rpc.Client
		var err error

		// uses HTTP-RPC if IPC is not set
		if ipcPath == "" {
			rpcClient, err = rpc.Dial(httpPath)
		} else  {
			rpcClient, err = rpc.Dial(ipcPath)
		}

		client := ethclient.NewClient(rpcClient)

		config := &node.Config{
			DataDir: dataDirPath,
		}

		// Set up keystore
		scryptN, scryptP, keydir, err := config.AccountConfig()
		if err != nil {
			log.Fatal(err)
		}

		ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

		file, err := os.Open(passwordFile)
		if err != nil {
			log.Fatal(err)
		}

		// Retrieve password to unlock account
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanWords)
		scanner.Scan()
		password := scanner.Text()
		ks.Unlock(ks.Accounts()[0], password)

		// Construct transaction
		txOps := &bind.TransactOpts{
			From:  ks.Accounts()[0].Address,
			Value: big.NewInt(0),
			Signer: func(signer types.Signer, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
				networkID, err := client.NetworkID(context.Background())
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				return ks.SignTx(ks.Accounts()[0], tx, networkID)
			},
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
