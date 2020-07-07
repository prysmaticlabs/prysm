package main

import (
	"bufio"
	"encoding/hex"
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
	"github.com/pkg/errors"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	log = logrus.WithField("prefix", "main")
)

func main() {
	var keystoreUTCPath string
	var prysmKeystorePath string
	var ipcPath string
	var passwordFile string
	var httpPath string
	var privKeyString string
	var depositContractAddr string
	var numberOfDeposits int64
	var depositAmount int64
	var depositDelay int64
	var randomKey bool

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	app := cli.App{}
	app.Name = "sendDepositTx"
	app.Usage = "this is a util to send deposit transactions"
	app.Version = version.GetVersion()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "keystoreUTCPath",
			Usage:       "Location of keystore",
			Destination: &keystoreUTCPath,
		},
		&cli.StringFlag{
			Name:        "prysm-keystore",
			Usage:       "The path to the existing prysm keystore. This flag is ignored if used with --random-key",
			Destination: &prysmKeystorePath,
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
			Usage:       "Private key to send ETH transaction",
			Destination: &privKeyString,
		},
		&cli.StringFlag{
			Name:        "depositContract",
			Usage:       "Address of the deposit contract",
			Destination: &depositContractAddr,
		},
		&cli.IntFlag{
			Name:        "numberOfDeposits",
			Value:       1,
			Usage:       "number of deposits to send to the contract",
			Destination: &numberOfDeposits,
		},
		&cli.IntFlag{
			Name:        "depositAmount",
			Value:       int64(params.BeaconConfig().MaxEffectiveBalance),
			Usage:       "Maximum deposit value allowed in contract(in gwei)",
			Destination: &depositAmount,
		},
		&cli.IntFlag{
			Name:        "depositDelay",
			Value:       5,
			Usage:       "The time delay between sending the deposits to the contract(in seconds)",
			Destination: &depositDelay,
		},
		&cli.BoolFlag{
			Name:        "random-key",
			Usage:       "Use a randomly generated keystore key",
			Destination: &randomKey,
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
		depositAmountInGwei := uint64(depositAmount)

		if privKeyString != "" {
			// User inputs private key, sign tx with private key
			privKey, err := crypto.HexToECDSA(privKeyString)
			if err != nil {
				return err
			}
			txOps = bind.NewKeyedTransactor(privKey)
			txOps.Value = new(big.Int).Mul(big.NewInt(depositAmount), big.NewInt(1e9))
		} else {
			// User inputs keystore json file, sign tx with keystore json
			password := loadTextFromFile(passwordFile)

			// #nosec - Inclusion of file via variable is OK for this tool.
			keyJSON, err := ioutil.ReadFile(keystoreUTCPath)
			if err != nil {
				return err
			}
			privKey, err := keystore.DecryptKey(keyJSON, password)
			if err != nil {
				return err
			}

			txOps = bind.NewKeyedTransactor(privKey.PrivateKey)
			txOps.Value = new(big.Int).Mul(big.NewInt(depositAmount), big.NewInt(1e9))
			txOps.GasLimit = 500000
		}

		depositContract, err := contracts.NewDepositContract(common.HexToAddress(depositContractAddr), client)
		if err != nil {
			return err
		}

		validatorKeys := make(map[string]*prysmKeyStore.Key)
		if randomKey {
			validatorKey, err := prysmKeyStore.NewKey()
			if err != nil {
				return errors.Wrap(err, "Could not generate random key")
			}
			validatorKeys[hex.EncodeToString(validatorKey.PublicKey.Marshal())] = validatorKey
		} else {
			// Load from keystore
			store := prysmKeyStore.NewKeystore(prysmKeystorePath)
			rawPassword := loadTextFromFile(passwordFile)
			prefix := params.BeaconConfig().ValidatorPrivkeyFileName
			validatorKeys, err = store.GetKeys(prysmKeystorePath, prefix, rawPassword, false /* warnOnFail */)
			if err != nil {
				log.WithField("path", prysmKeystorePath).WithField("password", rawPassword).Errorf("Could not get keys: %v", err)
			}
		}

		keyCounter := int64(0)
		for _, validatorKey := range validatorKeys {
			data, depositRoot, err := depositutil.DepositInput(validatorKey.SecretKey, validatorKey.SecretKey, depositAmountInGwei)
			if err != nil {
				log.Errorf("Could not generate deposit input data: %v", err)
				continue
			}
			for j := int64(0); j < numberOfDeposits; j++ {
				tx, err := depositContract.Deposit(txOps, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoot)
				if err != nil {
					log.Errorf("unable to send transaction to contract: %v", err)
					continue
				}

				log.WithFields(logrus.Fields{
					"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
				}).Infof("Deposit %d sent to contract address %v for validator with a public key %#x", (keyCounter*numberOfDeposits)+j, depositContractAddr, validatorKey.PublicKey.Marshal())

				time.Sleep(time.Duration(depositDelay) * time.Second)
			}
			keyCounter++
		}

		return nil
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
