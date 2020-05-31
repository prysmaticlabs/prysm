package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/version"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/d4l3k/messagediff.v1"
	"github.com/urfave/cli/v2"
)

func main() {
	var blockPath string
	var preStatePath string
	var expectedPostStatePath string

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	app := cli.App{}
	app.Name = "pcli"
	app.Usage = "A command line utility to run eth2 specific commands"
	app.Version = version.GetVersion()
	app.Commands = []*cli.Command{{
		Name:     "state-transition",
		Category: "state-transition",
		Usage:    "Subcommand to run manual state transitions",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "block-path",
				Usage:       "Path to block file(ssz)",
				Destination: &blockPath,
			},
			&cli.StringFlag{
				Name:        "pre-state-path",
				Usage:       "Path to pre state file(ssz)",
				Destination: &preStatePath,
			},
			&cli.StringFlag{
				Name:        "expected-post-state-path",
				Usage:       "Path to expected post state file(ssz)",
				Destination: &expectedPostStatePath,
			},
		},
		Action: func(c *cli.Context) error {
			if blockPath == "" {
				log.Info("Block path not provided for state transition. " +
					"Please provide path")
				reader := bufio.NewReader(os.Stdin)
				text, err := reader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				}
				if text = strings.Replace(text, "\n", "", -1); text == "" {
					log.Fatal("Empty block path given")
				}
				blockPath = text
			}
			block := &ethpb.SignedBeaconBlock{}
			if err := dataFetcher(blockPath, block); err != nil {
				log.Fatal(err)
			}
			blkRoot, err := stateutil.BlockRoot(block.Block)
			if err != nil {
				log.Fatal(err)
			}
			if preStatePath == "" {
				log.Info("Pre State path not provided for state transition. " +
					"Please provide path")
				reader := bufio.NewReader(os.Stdin)
				text, err := reader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				}
				if text = strings.Replace(text, "\n", "", -1); text == "" {
					log.Fatal("Empty state path given")
				}
				preStatePath = text
			}
			preState := &pb.BeaconState{}
			if err := dataFetcher(preStatePath, preState); err != nil {
				log.Fatal(err)
			}
			stateObj, err := stateTrie.InitializeFromProto(preState)
			if err != nil {
				log.Fatal(err)
			}
			preStateRoot, err := stateObj.HashTreeRoot(context.Background())
			if err != nil {
				log.Fatal(err)
			}
			log.WithFields(log.Fields{
				"blockSlot":    fmt.Sprintf("%d", block.Block.Slot),
				"preStateSlot": fmt.Sprintf("%d", stateObj.Slot()),
			}).Infof(
				"Performing state transition with a block root of %#x and pre state root of %#x",
				blkRoot,
				preStateRoot,
			)
			postState, err := state.ExecuteStateTransition(context.Background(), stateObj, block)
			if err != nil {
				log.Fatal(err)
			}
			postRoot, err := postState.HashTreeRoot(context.Background())
			log.Infof("Finished state transition with post state root of %#x", postRoot)

			// Diff the state if a post state is provided.
			if expectedPostStatePath != "" {
				expectedState := &pb.BeaconState{}
				if err := dataFetcher(expectedPostStatePath, expectedState); err != nil {
					log.Fatal(err)
				}
				if !ssz.DeepEqual(expectedState, postState.InnerStateUnsafe()) {
					diff, _ := messagediff.PrettyDiff(expectedState, postState.InnerStateUnsafe())
					log.Errorf("Derived state differs from provided post state: %s", diff)
				}
			}
			return nil
		},
	},
	}
	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

// dataFetcher fetches and unmarshals data from file to provided data structure.
func dataFetcher(fPath string, data interface{}) error {
	rawFile, err := ioutil.ReadFile(fPath)
	if err != nil {
		return err
	}
	return ssz.Unmarshal(rawFile, data)
}
