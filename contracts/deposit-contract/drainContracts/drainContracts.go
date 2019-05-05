package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	var keystoreUTCPath string
	var passwordFile string
	var httpPath string
	var privKeyString string

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	app := cli.NewApp()
	app.Name = "drainContracts"
	app.Usage = "this is a util to drain all (testing) deposit contracts of their ETH."
	app.Version = version.GetVersion()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "keystoreUTCPath",
			Usage:       "Location of keystore",
			Destination: &keystoreUTCPath,
		},
		cli.StringFlag{
			Name:        "httpPath",
			Value:       "https://goerli.infura.io/v3/be3fb7ed377c418087602876a40affa1",
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
			Usage:       "Private key to send ETH transaction",
			Destination: &privKeyString,
		},
	}

	app.Action = func(c *cli.Context) {
		// Set up RPC client
		var rpcClient *rpc.Client
		var err error
		var txOps *bind.TransactOpts

		// Uses HTTP-RPC if IPC is not set
		rpcClient, err = rpc.Dial(httpPath)
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
			txOps.GasLimit = 4000000
			nonce, err := client.NonceAt(context.Background(), crypto.PubkeyToAddress(privKey.PublicKey), nil)
			if err != nil {
				log.Fatalf("could not get account nonce: %v", err)
			}
			txOps.Nonce = big.NewInt(int64(nonce))
			fmt.Printf("current address is %s\n", crypto.PubkeyToAddress(privKey.PublicKey).String())
			fmt.Printf("nonce is %d\n", nonce)
			// User inputs keystore json file, sign tx with keystore json
		} else {
			password := loadTextFromFile(passwordFile)

			// #nosec - Inclusion of file via variable is OK for this tool.
			keyJSON, err := ioutil.ReadFile(keystoreUTCPath)
			if err != nil {
				log.Fatal(err)
			}
			privKey, err := keystore.DecryptKey(keyJSON, password)
			if err != nil {
				log.Fatal(err)
			}

			txOps = bind.NewKeyedTransactor(privKey.PrivateKey)
			txOps.Value = big.NewInt(0)
			txOps.GasLimit = 4000000
			nonce, err := client.NonceAt(context.Background(), privKey.Address, nil)
			if err != nil {
				log.Fatalf("could not get account nonce: %v", err)
			}
			txOps.Nonce = big.NewInt(int64(nonce))
			fmt.Printf("current address is %s\n", privKey.Address.String())
			fmt.Printf("nonce is %d\n", nonce)
		}

		addresses, err := allDepositContractAddresses(client)
		if err != nil {
			log.Fatalf("Could not get all deposit contract address: %v", err)
		}

		fmt.Printf("%d contracts ready to drain found\n", len(addresses))

		for _, address := range addresses {
			bal, err := client.BalanceAt(context.Background(), address, nil /*blockNum*/)
			if err != nil {
				log.Fatal(err)
			}
			if bal.Cmp(big.NewInt(0)) < 1 {
				continue
			}
			depositContract, err := contracts.NewDepositContract(address, client)
			if err != nil {
				log.Fatal(err)
			}
			tx, err := depositContract.Drain(txOps)
			if err != nil {
				log.Fatalf("unable to send transaction to contract: %v", err)
			}

			txOps.Nonce = txOps.Nonce.Add(txOps.Nonce, big.NewInt(1))

			fmt.Printf("Contract address %s drained in TX hash: %s\n", address.String(), tx.Hash().String())
			time.Sleep(time.Duration(1) * time.Second)
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func loadTextFromFile(filepath string) string {
	// #nosec - Inclusion of file via variable is OK for this tool.
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanner.Scan()
	return scanner.Text()
}

func allDepositContractAddresses(client *ethclient.Client) ([]common.Address, error) {
	log.Print("Looking up contracts")
	addresses := make(map[common.Address]bool)

	depositEventSignature := []byte("Deposit(bytes32,bytes,bytes,bytes32[32])")
	depositTopicHash := crypto.Keccak256Hash(depositEventSignature)
	fmt.Println(depositTopicHash.Hex())

	query := ethereum.FilterQuery{
		Addresses: []common.Address{},
		Topics: [][]common.Hash{
			{depositTopicHash},
		},
		FromBlock: big.NewInt(400000), // Contracts before this may not have drain().
	}

	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("could not get all deposit logs: %v", err)
	}

	fmt.Printf("%d deposit logs found\n", len(logs))
	for i := len(logs)/2 - 1; i >= 0; i-- {
		opp := len(logs) - 1 - i
		logs[i], logs[opp] = logs[opp], logs[i]
	}

	for _, ll := range logs {
		addresses[ll.Address] = true
	}

	keys := make([]common.Address, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}

	return keys, nil
}
