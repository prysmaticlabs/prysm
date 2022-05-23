package jwt

import (
	"errors"
	"fmt"

	"path/filepath"

	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/io/file"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/runtime/tos"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "jwt")

var Commands = &cli.Command{
	Name:        "generate-jwt-secret",
	Usage:       "creates a random 32 byte hex string in a plaintext file to be used for authenticating JSON-RPC requests. If no --output-file flag is defined, the file will be created in the current working directory",
	Description: `creates a random 32 byte hex string in a plaintext file to be used for authenticating JSON-RPC requests. If no --output-file flag is defined, the file will be created in the current working directory`,
	Flags: cmd.WrapFlags([]cli.Flag{
		cmd.JwtOutputFileFlag,
	}),
	Before: tos.VerifyTosAcceptedOrPrompt,
	Action: func(cliCtx *cli.Context) error {
		specifiedFilePath := cliCtx.String(cmd.JwtOutputFileFlag.Name)
		if err := generateHttpSecretInFile(specifiedFilePath); err != nil {
			log.Printf("Could not generate secret: %v", err)
		}
		return nil
	},
}

func generateHttpSecretInFile(specifiedFilePath string) error {
	jwtFileName := "secret.jwt"
	if len(specifiedFilePath) > 0 {
		jwtFileName = specifiedFilePath
	}

	secret, err := generateRandom32ByteHexString()
	if err != nil {
		return err
	}

	// decided to convert to string then back to bytes for easy debugging
	err = file.WriteFile(jwtFileName, []byte(secret))
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

	// TODO: Remove these before merging after unit + confirming IRL

	//encoded := hexutil.Encode(blocks)[:66]

	//sliced := encoded[2:66] // remove 0x

	return hexutil.Encode(blocks), nil
}
