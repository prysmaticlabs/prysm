package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/prysmaticlabs/prysm/shared/promptutil"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/urfave/cli/v2"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var (
	keystoresFlag = &cli.StringFlag{
		Name:     "keystores",
		Value:    "",
		Usage:    "Path to a file or directory containing keystore files",
		Required: true,
	}
	passwordFlag = &cli.StringFlag{
		Name:  "password",
		Value: "",
		Usage: "Password for the keystore(s)",
	}
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "decrypt",
				Usage: "decrypt a specified keystore file or directory containing keystore files",
				Flags: []cli.Flag{
					keystoresFlag,
					passwordFlag,
				},
				Action: decrypt,
			},
			{
				Name:  "encrypt",
				Usage: "encrypt a specified hex string into a keystore file",
				Flags: []cli.Flag{
					keystoresFlag,
					passwordFlag,
				},
				Action: encrypt,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func decrypt(cliCtx *cli.Context) error {
	keystorePath := cliCtx.String(keystoresFlag.Name)
	if keystorePath == "" {
		panic("need to be set")
	}
	fullPath, err := expandPath(keystorePath)
	if err != nil {
		panic(err)
	}
	password := cliCtx.String(passwordFlag.Name)
	isPasswordSet := cliCtx.IsSet(passwordFlag.Name)
	if !isPasswordSet {
		password, err = promptutil.PasswordPrompt("Input the keystore(s) password", func(s string) error {
			// Any password is valid.
			return nil
		})
	}
	isDir, err := hasDir(fullPath)
	if err != nil {
		panic(err)
	}
	if isDir {
		files, err := ioutil.ReadDir(fullPath)
		if err != nil {
			panic(err)
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if !strings.HasPrefix(f.Name(), "keystore") {
				continue
			}
			readAndDecryptKeystore(filepath.Join(fullPath, f.Name()), password)
		}
	}
	readAndDecryptKeystore(fullPath, password)
	return nil
}

func encrypt(cliCtx *cli.Context) error {
	return nil
}

func readAndDecryptKeystore(fullPath string, password string) {
	file, err := ioutil.ReadFile(fullPath)
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
	privKeyBytes, err := decryptor.Decrypt(keystoreFile.Crypto, password)
	if err != nil {
		panic(err)
	}
	publicKeyBytes, err := hex.DecodeString(keystoreFile.Pubkey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Privkey: %#x\n", privKeyBytes)
	fmt.Printf("Pubkey: %#x\n", publicKeyBytes)
}

// Checks if the item at the specified path exists and is a directory.
func hasDir(dirPath string) (bool, error) {
	fullPath, err := expandPath(dirPath)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return filepath.Abs(path.Clean(os.ExpandEnv(p)))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
