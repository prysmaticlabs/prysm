package jwt

import (
	"encoding/hex"
	"math/rand"
	"os"
	"time"

	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/runtime/tos"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "jwt")

// Commands for interacting with jwt tokens that support the beacon node.
var Commands = &cli.Command{
	Name:     "jwt",
	Category: "jwt",
	Usage:    "defines commands for interacting with jwt tokens that support the beacon node",
	Subcommands: []*cli.Command{
		{
			Name:        "generate-jwt-secret",
			Description: `creates a random 32 byte hex string in a plaintext file within the root /prysm directory`,
			Flags:       cmd.WrapFlags([]cli.Flag{}),
			Before:      tos.VerifyTosAcceptedOrPrompt,
			Action: func(cliCtx *cli.Context) error {
				if err := generateHttpSecretInFile(); err != nil {
					log.Printf("Could not generate secret: %v", err)
				}
				return nil
			},
		},
	},
}

func generateHttpSecretInFile() error {
	f, err := os.Create("secret.jwt")
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

	return nil
}

func generateRandom32ByteHexString() (string, error) {
	b := make([]byte, 32)

	src := rand.New(rand.NewSource(time.Now().UnixNano()))

	if _, err := src.Read(b); err != nil {
		return "", err // <- todo: this seems wrong, but I don't want to deal with pointers... recommended pattern?
	}

	return hex.EncodeToString(b)[:64], nil
}
