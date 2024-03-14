package db

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/slasher"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const DefaultChunkKind = types.MinSpan

var f = struct {
	Path                          string
	ValidatorIndex                uint64
	Epoch                         uint64
	ChunkKind                     string
	ChunkSize                     uint64
	ValidatorChunkSize            uint64
	HistoryLength                 uint64
	IsDisplayAllValidatorsInChunk bool
	IsDisplayAllEpochsInChunk     bool
}{}

var spanCmd = &cli.Command{
	Name:  "span",
	Usage: "visualise values in db span bucket",
	Action: func(c *cli.Context) error {
		if err := spanAction(c); err != nil {
			log.WithError(err).Fatal("Could not query db")
		}
		return nil
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "path",
			Usage:       "path to directory containing beaconchain.db",
			Destination: &f.Path,
		},
		&cli.Uint64Flag{
			Name:        "validator_index",
			Usage:       "filter by validator index",
			Destination: &f.ValidatorIndex,
		},
		&cli.Uint64Flag{
			Name:        "epoch",
			Usage:       "filter by epoch",
			Destination: &f.Epoch,
		},
		&cli.StringFlag{
			Name:        "chunk_kind",
			Usage:       "chunk kind to query [max|min] default: min",
			Destination: &f.ChunkKind,
			Value:       DefaultChunkKind.String(),
		},
		// C - defines how many elements are in a chunk for a validator min or max span slice.
		&cli.Uint64Flag{
			Name:        "chunk_size",
			Usage:       "chunk size to query (optional)",
			Destination: &f.ChunkSize,
		},
		// K - defines how many validators' chunks we store in a single flat byte slice on disk.
		&cli.Uint64Flag{
			Name:        "validator_chunk_size",
			Usage:       "validator chunk size to query (optional)",
			Destination: &f.ValidatorChunkSize,
		},
		// H - defines how many epochs we keep of min or max spans.
		&cli.Uint64Flag{
			Name:        "history_length",
			Usage:       "history length to query (optional)",
			Destination: &f.HistoryLength,
		},
		&cli.BoolFlag{
			Name:        "display_all_validators_in_chunk",
			Usage:       "display all validators in chunk",
			Destination: &f.IsDisplayAllValidatorsInChunk,
		},
		&cli.BoolFlag{
			Name:        "display_all_epochs_in_chunk",
			Usage:       "display all epochs in chunk",
			Destination: &f.IsDisplayAllEpochsInChunk,
		},
	},
}

func spanAction(_ *cli.Context) error {
	var (
		chunk slasher.Chunker

		err error
	)

	// variables
	chunkKind := getChunkKind()
	params := getSlasherParams()
	i := primitives.ValidatorIndex(f.ValidatorIndex)
	epoch := primitives.Epoch(f.Epoch)

	// display configuration
	fmt.Printf("Slasher Params %v\n", params)
	fmt.Printf("DB %v\n", f.Path)
	fmt.Printf("Chunk Kind %v\n", chunkKind)
	fmt.Printf("Validator %d\n", i)
	fmt.Printf("Epoch %d\n", epoch)

	// fetch chunk in database
	if chunk, err = slasher.GetChunkFromDatabase(
		context.Background(),
		f.Path,
		slasher.GetChunkFromDatabaseFilters{
			ChunkKind:                     chunkKind,
			ValidatorIndex:                i,
			SourceEpoch:                   epoch,
			IsDisplayAllValidatorsInChunk: f.IsDisplayAllValidatorsInChunk,
			IsDisplayAllEpochsInChunk:     f.IsDisplayAllEpochsInChunk,
		},
		params,
	); err != nil {
		return err
	}

	validatorChunkIdx := params.ValidatorChunkIndex(i)
	if !f.IsDisplayAllValidatorsInChunk {
		if !f.IsDisplayAllEpochsInChunk {
			chunkData, err := slasher.ChunkDataAtEpoch(params, chunk.Chunk(), i, epoch)
			if err != nil {
				return errors.Wrapf(err, "could not get chunk data at epoch %d for validator %d", epoch, i)
			}
			fmt.Printf("Chunk at epoch %d for validator %d: %v\n", epoch, i, chunkData)
		} else {
			vci := params.ValidatorChunkIndex(i)
			// TODO: should check index avoiding panics
			fmt.Printf("Chunk with epochs for validator %d: %d\n", i, chunk.Chunk()[vci:vci+params.ChunkSize()])
		}
	} else {
		// find first val and epoch in chunk
		firstValidator := params.ValidatorIndexesInChunk(validatorChunkIdx)[0]
		firstEpoch := epoch - (epoch.Mod(params.ChunkSize()))
		fmt.Printf("First validator in chunk: %d\n", firstValidator)
		fmt.Printf("First epoch in chunk: %d\n", firstEpoch)

		// display all validators and epochs in chunk
		if f.IsDisplayAllEpochsInChunk {
			// init table
			tw := table.NewWriter()

			// headers
			header := table.Row{"Validator / Epoch"}
			for y := uint64(0); y < params.ChunkSize(); y++ {
				header = append(header, firstEpoch+primitives.Epoch(y))
			}
			tw.AppendHeader(header)

			b := chunk.Chunk()
			c := uint64(0)
			for z := uint64(0); z < uint64(len(b)); z += params.ChunkSize() {
				end := z + params.ChunkSize()
				if end > uint64(len(b)) {
					end = uint64(len(b))
				}
				subChunk := b[z:end]

				row := make(table.Row, params.ChunkSize()+1)
				title := firstValidator + primitives.ValidatorIndex(c)
				row[0] = title
				for y, minspan := range subChunk {
					row[y+1] = minspan
				}
				tw.AppendRow(row)
				c++
			}
			tw.AppendSeparator()
			fmt.Println(tw.Render())
		} else {
			// all validators but just one epoch
			// TODO
		}
	}

	return nil
}

func getChunkKind() types.ChunkKind {
	chunkKind := types.MinSpan
	if f.ChunkKind == "max" {
		chunkKind = types.MaxSpan
	}
	return chunkKind
}

func getSlasherParams() *slasher.Parameters {
	var (
		chunkSize, validatorChunkSize uint64
		historyLength                 primitives.Epoch
	)
	params := slasher.DefaultParams()
	if f.ChunkSize != 0 && f.ChunkSize != params.ChunkSize() {
		chunkSize = f.ChunkSize
	} else {
		chunkSize = params.ChunkSize()
	}
	if f.ValidatorChunkSize != 0 && f.ValidatorChunkSize != params.ValidatorChunkSize() {
		validatorChunkSize = f.ValidatorChunkSize
	} else {
		validatorChunkSize = params.ValidatorChunkSize()
	}
	if f.HistoryLength != 0 && f.HistoryLength != uint64(params.HistoryLength()) {
		historyLength = primitives.Epoch(f.HistoryLength)
	} else {
		historyLength = params.HistoryLength()
	}
	return slasher.NewParams(chunkSize, validatorChunkSize, historyLength)
}
