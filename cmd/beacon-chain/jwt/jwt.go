package jwt

import (
	"errors"
	"fmt"
	"os"

	"path/filepath"

	"github.com/prysmaticlabs/prysm/crypto/rand"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/runtime/tos"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "jwt")

var Commands = &cli.Command{
	Name:        "generate-jwt-secret",
	Usage:       "creates a random 32 byte hex string in a jwt.secret plaintext file within the root /prysm directory",
	Description: `creates a random 32 byte hex string in a jwt.secret plaintext file within the root /prysm directory`,
	Flags:       cmd.WrapFlags([]cli.Flag{}),
	Before:      tos.VerifyTosAcceptedOrPrompt,
	Action: func(cliCtx *cli.Context) error {
		if err := generateHttpSecretInFile(); err != nil {
			log.Printf("Could not generate secret: %v", err)
		}
		return nil
	},
}

func generateHttpSecretInFile() error {
	jwtFileName := "secret.jwt"
	f, err := os.Create(jwtFileName)
	if err != nil {
		return err
	}

	defer f.Close()

	secret, err := generateRandom32ByteHexString()
	if err != nil {
		return err
	}

	_, err = f.WriteString(secret)
	if err != nil {
		return err
	}

	jwtPath, err := filepath.Abs(jwtFileName)
	if err == nil {
		fmt.Println("JWT token file path:", jwtPath)
	} else {
		return err
	}

	return nil
}

func generateRandom32ByteHexString() (string, error) {

	blocks := make([]byte, 32)
	randGen := rand.NewGenerator()
	blocksLength, err := randGen.Read(blocks)

	if err != nil {
		return "", errors.New("rand: unexpected length")
	} else if blocksLength <= 0 {
		return "", err
	}

	encoded := hexutil.Encode(blocks)[:66]

	sliced := encoded[2:66] // remove 0x

	return sliced, nil
}
