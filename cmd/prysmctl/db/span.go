package db

import (
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

var (
	f = struct {
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

	slasherDefaultParams = slasher.DefaultParams()
)

var spanCmd = &cli.Command{
	Name:  "span",
	Usage: "visualise values in db span bucket",
	Action: func(c *cli.Context) error {
		if err := spanAction(c); err != nil {
			log.WithError(err).Fatal("error in span command")
		}
		return nil
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "path",
			Usage:       "path to directory containing beaconchain.db",
			Destination: &f.Path,
			Required:    true,
		},
		&cli.Uint64Flag{
			Name:        "validator_index",
			Usage:       "filter by validator index",
			Destination: &f.ValidatorIndex,
			Required:    true,
		},
		&cli.Uint64Flag{
			Name:        "epoch",
			Usage:       "filter by epoch",
			Destination: &f.Epoch,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "chunk_kind",
			Usage:       "chunk kind to query [max|min] (default: min)",
			Destination: &f.ChunkKind,
			Value:       DefaultChunkKind.String(),
		},
		&cli.Uint64Flag{
			Name:        "chunk_size",
			Usage:       fmt.Sprintf("chunk size to query (default: %d)", slasherDefaultParams.ChunkSize()),
			Destination: &f.ChunkSize,
		},
		&cli.Uint64Flag{
			Name:        "validator_chunk_size",
			Usage:       fmt.Sprintf("validator chunk size to query (default: %d)", slasherDefaultParams.ValidatorChunkSize()),
			Destination: &f.ValidatorChunkSize,
		},
		&cli.Uint64Flag{
			Name:        "history_length",
			Usage:       fmt.Sprintf("history length to query (default: %d)", slasherDefaultParams.HistoryLength()),
			Destination: &f.HistoryLength,
		},
		&cli.BoolFlag{
			Name:        "display_all_validators_in_chunk",
			Usage:       "display all validators in chunk (default: false)",
			Destination: &f.IsDisplayAllValidatorsInChunk,
		},
		&cli.BoolFlag{
			Name:        "display_all_epochs_in_chunk",
			Usage:       "display all epochs in chunk (default: false)",
			Destination: &f.IsDisplayAllEpochsInChunk,
		},
	},
}

func spanAction(cliCtx *cli.Context) error {
	var (
		chunk slasher.Chunker

		err error
	)

	// context
	ctx := cliCtx.Context

	// variables
	chunkKind := getChunkKind()
	params := getSlasherParams()
	i := primitives.ValidatorIndex(f.ValidatorIndex)
	epoch := primitives.Epoch(f.Epoch)

	// display configuration
	fmt.Printf("############################# CONFIGURATION ################################\n")
	fmt.Printf("# Slasher Params\n")
	fmt.Printf("# Chunk Size: %d\n", params.ChunkSize())
	fmt.Printf("# Validator Chunk Size: %d\n", params.ValidatorChunkSize())
	fmt.Printf("# History Length: %d\n", params.HistoryLength())
	fmt.Printf("# DB %s\n", f.Path)
	fmt.Printf("# Chunk Kind: %s\n", chunkKind)
	fmt.Printf("# Validator %d\n", i)
	fmt.Printf("# Epoch %d\n", epoch)
	fmt.Printf("############################################################################\n")

	// fetch chunk in database
	if chunk, err = slasher.GetChunkFromDatabase(
		ctx,
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
		return errors.Wrapf(err, "could not get chunk from database")
	}

	// fetch information related to chunk
	validatorChunkIdx := params.ValidatorChunkIndex(i)
	fmt.Printf("\n################################ CHUNK #####################################\n")
	firstValidator := params.ValidatorIndexesInChunk(validatorChunkIdx)[0]
	firstEpoch := epoch - (epoch.Mod(params.ChunkSize()))
	fmt.Printf("# First validator in chunk: %d\n", firstValidator)
	fmt.Printf("# First epoch in chunk: %d\n", firstEpoch)
	fmt.Printf("############################################################################\n\n")

	// display information about all validators in chunk
	if !f.IsDisplayAllValidatorsInChunk {
		if !f.IsDisplayAllEpochsInChunk {
			chunkData, err := slasher.ChunkDataAtEpoch(params, chunk.Chunk(), i, epoch)
			if err != nil {
				return errors.Wrapf(err, "could not get chunk data at epoch %d for validator %d", epoch, i)
			}
			fmt.Printf("Chunk at epoch %d for validator %d: %v\n", epoch, i, chunkData)
		} else {
			// init table
			tw := table.NewWriter()

			// headers
			header := table.Row{"Validator / Epoch"}
			for y := uint64(0); y < params.ChunkSize(); y++ {
				header = append(header, firstEpoch+primitives.Epoch(y))
			}
			tw.AppendHeader(header)

			// rows
			b := chunk.Chunk()
			validatorFirstEpochIdx := uint64(i.Mod(params.ValidatorChunkSize())) * params.ChunkSize()
			subChunk := b[validatorFirstEpochIdx : validatorFirstEpochIdx+params.ChunkSize()]
			row := make(table.Row, params.ChunkSize()+1)
			title := i
			row[0] = title
			for y, span := range subChunk {
				row[y+1] = span
			}
			tw.AppendRow(row)

			// display
			tw.AppendSeparator()
			fmt.Println(tw.Render())
		}
	} else {

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

			// rows
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
				for y, span := range subChunk {
					row[y+1] = span
				}
				tw.AppendRow(row)

				c++
			}

			// display
			tw.AppendSeparator()
			fmt.Println(tw.Render())
		} else {
			indexEpochInChunk := epoch - firstEpoch

			// init table
			tw := table.NewWriter()

			// headers
			header := table.Row{"Validator / Epoch"}
			header = append(header, epoch)
			tw.AppendHeader(header)

			// rows
			b := chunk.Chunk()
			c := uint64(0)
			for z := uint64(0); z < uint64(len(b)); z += params.ChunkSize() {
				end := z + params.ChunkSize()
				if end > uint64(len(b)) {
					end = uint64(len(b))
				}
				subChunk := b[z:end]

				row := make(table.Row, 2)
				title := firstValidator + primitives.ValidatorIndex(c)
				row[0] = title
				row[1] = subChunk[indexEpochInChunk]
				tw.AppendRow(row)

				c++
			}
			tw.AppendSeparator()

			// display
			fmt.Println(tw.Render())
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
	if f.ChunkSize != 0 && f.ChunkSize != slasherDefaultParams.ChunkSize() {
		chunkSize = f.ChunkSize
	} else {
		chunkSize = slasherDefaultParams.ChunkSize()
	}
	if f.ValidatorChunkSize != 0 && f.ValidatorChunkSize != slasherDefaultParams.ValidatorChunkSize() {
		validatorChunkSize = f.ValidatorChunkSize
	} else {
		validatorChunkSize = slasherDefaultParams.ValidatorChunkSize()
	}
	if f.HistoryLength != 0 && f.HistoryLength != uint64(slasherDefaultParams.HistoryLength()) {
		historyLength = primitives.Epoch(f.HistoryLength)
	} else {
		historyLength = slasherDefaultParams.HistoryLength()
	}
	return slasher.NewParams(chunkSize, validatorChunkSize, historyLength)
}
