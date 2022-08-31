package main

import (
	"os"

	"github.com/prysmaticlabs/prysm/v3/cmd/prysmctl/checkpoint"
	"github.com/prysmaticlabs/prysm/v3/cmd/prysmctl/p2p"
	"github.com/prysmaticlabs/prysm/v3/cmd/prysmctl/testnet"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var prysmctlCommands []*cli.Command

func main() {
	app := &cli.App{
		Commands: prysmctlCommands,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	prysmctlCommands = append(prysmctlCommands, checkpoint.Commands...)
	prysmctlCommands = append(prysmctlCommands, testnet.Commands...)
	prysmctlCommands = append(prysmctlCommands, p2p.Commands...)
}
