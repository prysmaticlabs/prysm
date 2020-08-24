package derived

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/k0kubun/go-ansi"
	"github.com/pkg/errors"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	util "github.com/wealdtech/go-eth2-util"
)

// SendDepositConfig contains all the required information for
// the derived keymanager to submit a 32 ETH deposit from the user's
// eth1 wallet to an eth1 RPC endpoint.
type SendDepositConfig struct {
	DepositContractAddress   string
	DepositDelaySeconds      time.Duration
	DepositPublicKeys        []bls.PublicKey
	Eth1KeystoreUTCFile      string
	Eth1KeystorePasswordFile string
	Eth1PrivateKey           string
	Web3Provider             string
}

// SendDepositTx to the validator deposit contract on the eth1 chain
// using a defined configuration by first unlocking the user's eth1 wallet,
// then generating the deposit data for a desired validator account, finally
// submitting the transaction via an eth1 web3 endpoint.
func (dr *Keymanager) SendDepositTx(conf *SendDepositConfig) error {
	var txOps *bind.TransactOpts
	rpcClient, err := rpc.Dial(conf.Web3Provider)
	if err != nil {
		return err
	}
	client := ethclient.NewClient(rpcClient)
	depositAmountInGwei := params.BeaconConfig().MaxEffectiveBalance

	if conf.Eth1PrivateKey != "" {
		// User inputs private key, sign tx with private key
		privKey, err := crypto.HexToECDSA(conf.Eth1PrivateKey)
		if err != nil {
			return err
		}
		txOps = bind.NewKeyedTransactor(privKey)
		txOps.Value = new(big.Int).Mul(big.NewInt(int64(depositAmountInGwei)), big.NewInt(1e9))
	} else {
		// User inputs keystore json file, sign tx with keystore json
		password, err := fileutil.ReadFileAsBytes(conf.Eth1KeystorePasswordFile)
		if err != nil {
			return err
		}
		// #nosec - Inclusion of file via variable is OK for this tool.
		keyJSON, err := fileutil.ReadFileAsBytes(conf.Eth1KeystoreUTCFile)
		if err != nil {
			return err
		}
		privKey, err := keystore.DecryptKey(keyJSON, strings.TrimRight(string(password), "\r\n"))
		if err != nil {
			return err
		}

		txOps = bind.NewKeyedTransactor(privKey.PrivateKey)
		txOps.Value = new(big.Int).Mul(big.NewInt(int64(depositAmountInGwei)), big.NewInt(1e9))
		txOps.GasLimit = 500000
	}

	depositContract, err := contracts.NewDepositContract(common.HexToAddress(conf.DepositContractAddress), client)
	if err != nil {
		return err
	}
	wantedPubKeys := make(map[[48]byte]bool, len(conf.DepositPublicKeys))
	for _, pk := range conf.DepositPublicKeys {
		wantedPubKeys[bytesutil.ToBytes48(pk.Marshal())] = true
	}
	bar := initializeProgressBar(int(dr.seedCfg.NextAccount), "Sending deposit transactions...")
	for i := uint64(0); i < dr.seedCfg.NextAccount; i++ {
		validatingKeyPath := fmt.Sprintf(ValidatingKeyDerivationPathTemplate, i)
		validatingKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, validatingKeyPath)
		if err != nil {
			return errors.Wrapf(err, "failed to read validating key for account %d", i)
		}
		if ok := wantedPubKeys[bytesutil.ToBytes48(validatingKey.PublicKey().Marshal())]; !ok {
			continue
		}
		withdrawalKeyPath := fmt.Sprintf(WithdrawalKeyDerivationPathTemplate, i)
		withdrawalKey, err := util.PrivateKeyFromSeedAndPath(dr.seed, withdrawalKeyPath)
		if err != nil {
			return errors.Wrapf(err, "failed to read withdrawal key for account %d", i)
		}
		validatingKeyBLS, err := bls.SecretKeyFromBytes(validatingKey.Marshal())
		if err != nil {
			return err
		}
		withdrawalKeyBLS, err := bls.SecretKeyFromBytes(withdrawalKey.Marshal())
		if err != nil {
			return err
		}
		data, depositRoot, err := depositutil.DepositInput(validatingKeyBLS, withdrawalKeyBLS, depositAmountInGwei)
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
			log.Errorf("Unable to send transaction to contract: %v", err)
			continue
		}
		log.WithFields(logrus.Fields{
			"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
		}).Infof(
			"Deposit %d sent to contract address %v for validator with a public key %#x",
			i,
			conf.DepositContractAddress,
			validatingKey.PublicKey().Marshal(),
		)
		log.Infof(
			"You can monitor your transaction on Etherscan here https://goerli.etherscan.io/tx/0x%x",
			tx.Hash(),
		)
		log.Infof("Waiting for a short delay of %v seconds...", conf.DepositDelaySeconds)
		if err := bar.Add(1); err != nil {
			log.Errorf("Could not increase progress bar percentage: %v", err)
		}
		time.Sleep(conf.DepositDelaySeconds)
	}
	log.Infof("Successfully sent all validator deposits!")
	return nil
}

func initializeProgressBar(numItems int, msg string) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		numItems,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetDescription(msg),
	)
}
