package jwt

import (
	"errors"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/urfave/cli/v2"
)

const (
	secretFileName = "jwt.hex"
)

var Commands = &cli.Command{
	Name:        "generate-auth-secret",
	Usage:       "creates a random, 32 byte hex string in a plaintext file to be used for authenticating JSON-RPC requests. If no --output-file flag is defined, the file will be created in the current working directory",
	Description: `creates a random, 32 byte hex string in a plaintext file to be used for authenticating JSON-RPC requests. If no --output-file flag is defined, the file will be created in the current working directory`,
	Flags: cmd.WrapFlags([]cli.Flag{
		cmd.JwtOutputFileFlag,
	}),
	Action: generateHttpSecretInFile,
}

func generateHttpSecretInFile(c *cli.Context) error {
	fileName := secretFileName
	specifiedFilePath := c.String(cmd.JwtOutputFileFlag.Name)
	if len(specifiedFilePath) > 0 {
		fileName = specifiedFilePath
	}
	secret, err := generateRandom32ByteHexString()
	if err != nil {
		return err
	}
	fileDir := filepath.Dir(fileName)
	exists, err := file.HasDir(fileDir)
	if err != nil {
		return err
	}
	if !exists {
		if err := file.MkdirAll(fileDir); err != nil {
			return err
		}
	}
	return file.WriteFile(fileName, []byte(secret))
}

func generateRandom32ByteHexString() (string, error) {
	blocks := make([]byte, 32)
	randGen := rand.NewGenerator()
	blocksLength, err := randGen.Read(blocks)

	if err != nil {
		return "", err
	} else if blocksLength <= 0 {
		return "", errors.New("rand: unexpected length")
	}

	return hexutil.Encode(blocks), nil
}
