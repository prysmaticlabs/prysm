package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage:   "ssz support for prysm",
		Commands: []*cli.Command{benchmark, generate, ir},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
