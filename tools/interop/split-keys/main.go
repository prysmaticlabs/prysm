package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	mnemonicsFileFlag      = flag.String("mnemonics-file", "", "File containing mnemonics, one mnemonic per line")
	keysPerMnemonicFlag    = flag.Int("keys-per-mnemonic", 0, "The number of keys per mnemonic to generate")
	numberOfWalletsFlag    = flag.Int("num-wallets", 0, "Number of wallets to generate")
	walletOutDirFlag       = flag.String("out-dir", "", "Output directory for wallet files")
	walletPasswordFileFlag = flag.String("wallet-password-file", "", "File containing the password to encrypt all generated wallets")
)

// This application is run to generate keystores for testnets.
func main() {
	flag.Parse()
	f, err := os.Open(*mnemonicsFileFlag)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	privKeys := make([][]byte, 0)
	pubKeys := make([][]byte, 0)
	s := bufio.NewScanner(f)
	for s.Scan() {
		fmt.Printf("Generating %d keys from mneomic\n", *keysPerMnemonicFlag)
		mnemonic := s.Text()
		seed, err := seedFromMnemonic(mnemonic, "" /* 25th word*/)
		if err != nil {
			panic(err)
		}
		for i := 0; i < *keysPerMnemonicFlag; i++ {
			if i%250 == 0 && i > 0 {
				fmt.Printf("%d/%d keys generated\n", i, *keysPerMnemonicFlag)
			}
			privKey, err := util.PrivateKeyFromSeedAndPath(
				seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i),
			)
			if err != nil {
				panic(err)
			}
			privKeys = append(privKeys, privKey.Marshal())
			pubKeys = append(pubKeys, privKey.PublicKey().Marshal())
		}
	}

	fmt.Printf("Splitting %d keys across %d wallets\n", len(privKeys), *numberOfWalletsFlag)
	
	wPass, err := ioutil.ReadFile(*walletPasswordFileFlag)
	if err != nil {
		panic(err)
	}

	keysPerWallet := len(privKeys) / *numberOfWalletsFlag

	ctx := context.Background()
	for i := 0; i < *numberOfWalletsFlag; i++ {
		w := wallet.New(&wallet.Config{
			WalletDir:      path.Join(*walletOutDirFlag, fmt.Sprintf("wallet_%d", i)),
			KeymanagerKind: keymanager.Imported,
			WalletPassword: string(wPass),
		})
		km, err := imported.NewKeymanager(ctx, &imported.SetupConfig{
			Wallet: w,
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Importing %d keys into wallet %d\n", keysPerWallet, i)
		if err := km.ImportKeypairs(ctx, privKeys[i*keysPerWallet:(i+1)*keysPerWallet], pubKeys[i*keysPerWallet:(i+1)*keysPerWallet]); err != nil {
			panic(err)
		}
	}
	fmt.Println("done")
}

// Uses the provided mnemonic seed phrase to generate the
// appropriate seed file for recovering a derived wallets.
func seedFromMnemonic(mnemonic, mnemonicPassphrase string) ([]byte, error) {
	if ok := bip39.IsMnemonicValid(mnemonic); !ok {
		return nil, bip39.ErrInvalidMnemonic
	}
	return bip39.NewSeed(mnemonic, mnemonicPassphrase), nil
}
