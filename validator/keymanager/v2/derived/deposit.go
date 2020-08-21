package derived

import (
	"bufio"
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
	"github.com/sirupsen/logrus"

	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func (dr *Keymanager) SendDepositTx() error {
	var rpcClient *rpc.Client
	var err error
	var txOps *bind.TransactOpts
	privKeyString := "hello"
	passwordFile := "h"
	keystoreUTCPath := "h"
	depositDelay := time.Second

	// Uses HTTP-RPC if IPC is not set
	//if ipcPath == "" {
	rpcClient, err = rpc.Dial("https://goerli.prylabs.net")
	//} else {
	//	rpcClient, err = rpc.Dial(ipcPath)
	//}
	if err != nil {
		return err
	}
	client := ethclient.NewClient(rpcClient)
	depositAmountInGwei := params.BeaconConfig().MinDepositAmount

	if privKeyString != "" {
		// User inputs private key, sign tx with private key
		privKey, err := crypto.HexToECDSA(privKeyString)
		if err != nil {
			return err
		}
		txOps = bind.NewKeyedTransactor(privKey)
		txOps.Value = new(big.Int).Mul(big.NewInt(int64(depositAmountInGwei)), big.NewInt(1e9))
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
		txOps.Value = new(big.Int).Mul(big.NewInt(int64(depositAmountInGwei)), big.NewInt(1e9))
		txOps.GasLimit = 500000
	}

	depositContract, err := contracts.NewDepositContract(common.HexToAddress(""), client)
	if err != nil {
		return err
	}
	keyCounter := int64(0)
	for _, validatorKey := range dr.keysCache {
		data, depositRoot, err := depositutil.DepositInput(validatorKey, validatorKey, depositAmountInGwei)
		if err != nil {
			log.Errorf("Could not generate deposit input data: %v", err)
			continue
		}
		tx, err := depositContract.Deposit(
			txOps,
			data.PublicKey,
			data.WithdrawalCredentials,
			data.Signature,
			depositRoot,
		)
		if err != nil {
			log.Errorf("unable to send transaction to contract: %v", err)
			continue
		}

		log.WithFields(logrus.Fields{
			"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
		}).Infof(
			"Deposit %d sent to contract address %v for validator with a public key %#x",
			keyCounter,
			"",
			validatorKey.PublicKey().Marshal(),
		)
		time.Sleep(time.Duration(depositDelay) * time.Second)
		keyCounter++
	}
	return nil
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
