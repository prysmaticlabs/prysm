package main

import (
	"os"

	"github.com/prysmaticlabs/prysm/cmd/prysmctl/checkpoint"
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
}
