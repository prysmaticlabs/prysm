package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	var keystoreUTCPath string
	var ipcPath string
	var passwordFile string
	var httpPath string
	var privKeyString string
	var k8sConfigMapName string
	var depositsForChainStart int64
	var minDepositAmount int64
	var maxDepositAmount int64
	var customChainstartDelay uint64
	var drainAddress string

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	log := logrus.WithField("prefix", "main")

	app := cli.NewApp()
	app.Name = "deployDepositContract"
	app.Usage = "this is a util to deploy deposit contract"
	app.Version = version.GetVersion()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "keystoreUTCPath",
			Usage:       "Location of keystore",
			Destination: &keystoreUTCPath,
		},
		cli.Uint64Flag{
			Name:        "customChainstartDelay",
			Usage:       "Number of seconds to delay the ChainStart genesis timestamp",
			Value:       0,
			Destination: &customChainstartDelay,
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
		cli.StringFlag{
			Name:        "k8sConfig",
			Usage:       "Name of kubernetes config map to update with the contract address",
			Destination: &k8sConfigMapName,
		},
		cli.Int64Flag{
			Name:        "chainStart",
			Value:       params.ContractConfig().DepositsForChainStart.Int64(),
			Usage:       "Number of validators required for chain start",
			Destination: &depositsForChainStart,
		},
		cli.Int64Flag{
			Name:        "minDeposit",
			Value:       params.ContractConfig().MinDepositAmount.Int64(),
			Usage:       "Minimum deposit value allowed in contract",
			Destination: &minDepositAmount,
		},
		cli.Int64Flag{
			Name:        "maxDeposit",
			Value:       params.ContractConfig().MaxDepositAmount.Int64(),
			Usage:       "Maximum deposit value allowed in contract",
			Destination: &maxDepositAmount,
		},
		cli.StringFlag{
			Name:        "drainAddress",
			Value:       "",
			Usage:       "The drain address to specify in the contract. The default will be msg.sender",
			Destination: &drainAddress,
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
			txOps.GasLimit = 4000000
			// User inputs keystore json file, sign tx with keystore json
		} else {
			// #nosec - Inclusion of file via variable is OK for this tool.
			file, err := os.Open(passwordFile)
			if err != nil {
				log.Fatal(err)
			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanWords)
			scanner.Scan()
			password := scanner.Text()

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
		}

		drain := txOps.From
		if drainAddress != "" {
			drain = common.HexToAddress(drainAddress)
		}

		txOps.GasPrice = big.NewInt(10 * 1e9 /* 10 gwei */)

		// Deploy validator registration contract
		addr, tx, _, err := contracts.DeployDepositContract(
			txOps,
			client,
			big.NewInt(depositsForChainStart),
			big.NewInt(minDepositAmount),
			big.NewInt(maxDepositAmount),
			big.NewInt(int64(customChainstartDelay)),
			drain,
		)

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

		log.WithFields(logrus.Fields{
			"address": addr.Hex(),
		}).Info("New contract deployed")

		if k8sConfigMapName != "" {
			if err := updateKubernetesConfigMap(k8sConfigMapName, addr.Hex()); err != nil {
				log.Fatalf("Failed to update kubernetes config map: %v", err)
			} else {
				log.Printf("Updated config map %s", k8sConfigMapName)
			}
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// updateKubernetesConfigMap in the beacon-chain namespace. This specifically
// updates the data value for DEPOSIT_CONTRACT_ADDRESS.
func updateKubernetesConfigMap(configMapName string, contractAddr string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	client, err := k8s.NewForConfig(config)
	if err != nil {
		return err
	}

	cm, err := client.CoreV1().ConfigMaps("beacon-chain").Get("beacon-config", metav1.GetOptions{})
	if err != nil {
		return err
	}

	if cm.Data["DEPOSIT_CONTRACT_ADDRESS"] != "0x0" {
		return fmt.Errorf("existing vcr address in config map = %v", cm.Data["DEPOSIT_CONTRACT_ADDRESS"])
	}
	cm.Data["DEPOSIT_CONTRACT_ADDRESS"] = contractAddr

	_, err = client.CoreV1().ConfigMaps("beacon-chain").Update(cm)

	return err
}
