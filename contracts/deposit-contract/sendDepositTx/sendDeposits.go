package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"math"
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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/x-cray/logrus-prefixed-formatter"
	rand2 "golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

func main() {
	var keystoreUTCPath string
	var ipcPath string
	var passwordFile string
	var httpPath string
	var privKeyString string
	var depositContractAddr string
	var numberOfDeposits int64
	var depositAmount int64
	var depositDelay int64
	var variableTx bool

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	log := logrus.WithField("prefix", "main")

	app := cli.NewApp()
	app.Name = "sendDepositTx"
	app.Usage = "this is a util to send deposit transactions"
	app.Version = version.GetVersion()
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
		cli.StringFlag{
			Name:        "depositContract",
			Usage:       "Address of the deposit contract",
			Destination: &depositContractAddr,
		},
		cli.Int64Flag{
			Name:        "numberOfDeposits",
			Value:       8,
			Usage:       "number of deposits to send to the contract",
			Destination: &numberOfDeposits,
		},
		cli.Int64Flag{
			Name:        "depositAmount",
			Value:       3200,
			Usage:       "Maximum deposit value allowed in contract(in gwei)",
			Destination: &depositAmount,
		},
		cli.Int64Flag{
			Name:        "depositDelay",
			Value:       5,
			Usage:       "The time delay between sending the deposits to the contract(in seconds)",
			Destination: &depositDelay,
		},
		cli.BoolFlag{
			Name:        "variableTx",
			Usage:       "This enables variable transaction latencies to simulate real-world transactions",
			Destination: &variableTx,
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
		depositAmount = depositAmount * 1e9

		// User inputs private key, sign tx with private key
		if privKeyString != "" {
			privKey, err := crypto.HexToECDSA(privKeyString)
			if err != nil {
				log.Fatal(err)
			}
			txOps = bind.NewKeyedTransactor(privKey)
			txOps.Value = big.NewInt(depositAmount)
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
			txOps.Value = big.NewInt(depositAmount)
			txOps.GasLimit = 4000000
		}

		depositContract, err := contracts.NewDepositContract(common.HexToAddress(depositContractAddr), client)
		if err != nil {
			log.Fatal(err)
		}

		statDist := buildStatisticalDist(depositDelay, numberOfDeposits)

		for i := int64(0); i < numberOfDeposits; i++ {

			validatorKey, err := prysmKeyStore.NewKey(rand.Reader)

			data := &pb.DepositInput{
				Pubkey:                      validatorKey.PublicKey.BufferedPublicKey(),
				ProofOfPossession:           []byte("pop"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			}

			serializedData := new(bytes.Buffer)
			if err := ssz.Encode(serializedData, data); err != nil {
				log.Errorf("could not serialize deposit data: %v", err)
			}

			tx, err := depositContract.Deposit(txOps, serializedData.Bytes())
			if err != nil {
				log.Error("unable to send transaction to contract")
			}

			log.WithFields(logrus.Fields{
				"Transaction Hash": tx.Hash(),
			}).Infof("Deposit %d sent to contract for validator with a public key %#x", i, validatorKey.PublicKey.BufferedPublicKey())

			// If flag is enabled make transaction times variable
			if variableTx {
				time.Sleep(time.Duration(math.Abs(statDist.Rand())) * time.Second)
				continue
			}

			time.Sleep(time.Duration(depositDelay) * time.Second)
		}

	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func buildStatisticalDist(depositDelay int64, numberOfDeposits int64) *distuv.StudentsT {

	src := rand2.NewSource(uint64(time.Now().Unix()))
	dist := &distuv.StudentsT{
		Mu:    float64(depositDelay),
		Sigma: 2,
		Nu:    float64(numberOfDeposits - 1),
		Src:   src,
	}

	return dist
}
