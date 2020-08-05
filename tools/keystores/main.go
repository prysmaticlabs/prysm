package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

func main() {
	filePath := flag.String("keystore-path", "", "Path to keystore file")
	password := flag.String("password", "", "Keystore file password")
	flag.Parse()
	file, err := ioutil.ReadFile(*filePath)
	if err != nil {
		panic(err)
	}
	decryptor := keystorev4.New()
	keystoreFile := &v2keymanager.Keystore{}

	if err := json.Unmarshal(file, keystoreFile); err != nil {
		panic(err)
	}
	// We extract the validator signing private key from the keystore
	// by utilizing the password.
	privKeyBytes, err := decryptor.Decrypt(keystoreFile.Crypto, *password)
	if err != nil {
		panic(err)
	}
	publicKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Privkey: %#x, length = %d\n", privKeyBytes, len(privKeyBytes))
	fmt.Printf("Pubkey: %#x, length = %d\n", publicKeyBytes, len(publicKeyBytes))
}
