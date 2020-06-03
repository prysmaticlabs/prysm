package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/kr/pretty"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/version"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/d4l3k/messagediff.v1"
)

func main() {
	var blockPath string
	var preStatePath string
	var expectedPostStatePath string
	var sszPath string

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	app := cli.App{}
	app.Name = "pcli"
	app.Usage = "A command line utility to run eth2 specific commands"
	app.Version = version.GetVersion()
	app.Commands = []*cli.Command{
		{
			Name:    "pretty",
			Aliases: []string{"p"},
			Usage:   "pretty-print SSZ data",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "ssz-path",
					Usage:       "Path to file(ssz)",
					Required:    true,
					Destination: &sszPath,
				},
			},
			Subcommands: []*cli.Command{
				{
					Name:  "attestation",
					Usage: "Pretty print a Attestation",
					Action: func(c *cli.Context) error {
						data := &ethpb.Attestation{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "attestation_data",
					Usage: "Pretty print a AttestationData",
					Action: func(c *cli.Context) error {
						data := &ethpb.AttestationData{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "attester_slashing",
					Usage: "Pretty print a AttesterSlashing",
					Action: func(c *cli.Context) error {
						data := &ethpb.AttesterSlashing{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "signed_block",
					Usage: "Pretty print a SignedBeaconBlock",
					Action: func(c *cli.Context) error {
						data := &ethpb.SignedBeaconBlock{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "block",
					Usage: "Pretty print a BeaconBlock",
					Action: func(c *cli.Context) error {
						data := &ethpb.BeaconBlock{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "block_body",
					Usage: "Pretty print a BlockBody",
					Action: func(c *cli.Context) error {
						data := &ethpb.BeaconBlockBody{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "block_header",
					Usage: "Pretty print a BlockHeader",
					Action: func(c *cli.Context) error {
						data := &ethpb.BeaconBlockHeader{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "signed_block_header",
					Usage: "Pretty print a SignedBeaconBlockHeader",
					Action: func(c *cli.Context) error {
						data := &ethpb.SignedBeaconBlockHeader{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "deposit",
					Usage: "Pretty print a Deposit",
					Action: func(c *cli.Context) error {
						data := &ethpb.Deposit{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "deposit_data",
					Usage: "Pretty print a Deposit_Data",
					Action: func(c *cli.Context) error {
						data := &ethpb.Deposit_Data{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "eth1_data",
					Usage: "Pretty print a Eth1Data",
					Action: func(c *cli.Context) error {
						data := &ethpb.Eth1Data{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "fork_data",
					Usage: "Pretty print a Fork",
					Action: func(c *cli.Context) error {
						data := &pb.Fork{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "proposer_slashing",
					Usage: "Pretty print a ProposerSlashing",
					Action: func(c *cli.Context) error {
						data := &ethpb.ProposerSlashing{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "signed_voluntary_exit",
					Usage: "Pretty print a AttestationData",
					Action: func(c *cli.Context) error {
						data := &ethpb.SignedVoluntaryExit{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
				{
					Name:  "voluntary_exit",
					Usage: "Pretty print a VoluntaryExit",
					Action: func(c *cli.Context) error {
						data := &ethpb.VoluntaryExit{}
						prettyPrint(sszPath, data)
						return nil
					},
				},
			},
		},
		{
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

func prettyPrint(sszPath string, data interface{}) {
	if err := dataFetcher(sszPath, data); err != nil {
		log.Fatal(err)
	}
	str := pretty.Sprint(data)
	re := regexp.MustCompile("(?m)[\r\n]+^.*XXX_.*$")
	str = re.ReplaceAllString(str, "")
	fmt.Print(str)
}
