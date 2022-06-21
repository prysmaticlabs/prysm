package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/urfave/cli/v2"
	bolt "go.etcd.io/bbolt"
)

var flags  = struct {
	dbFile string
	tsdbFile string
}{}

const (
	initMMapSize = 536870912
)

var buckets = struct{
	state []byte
	blockSlotRoots []byte
	stateSlotRoots []byte
}{
	state: []byte("state"),
	blockSlotRoots: []byte("block-slot-indices"),
	stateSlotRoots: []byte("state-slot-indices"),
}

var commands = []*cli.Command{
	{
		Name: "bucket-chart",
		Usage: "visualize relative size of buckets",
		Action: bucketSizes,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "db",
				Usage: "path to beaconchain.db",
				Destination: &flags.dbFile,
			},
			&cli.StringFlag{
				Name: "tsdb",
				Usage: "path to database to store stats",
				Destination: &flags.tsdbFile,
				Value: "summarizer.db",
			},
		},
	},
	{
		Name: "summarize",
		Usage: "collect data about the given prysm bolt database file",
		Action: summarize,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "db",
				Usage: "path to beaconchain.db",
				Destination: &flags.dbFile,
			},
			&cli.StringFlag{
				Name: "tsdb",
				Usage: "path to database to store size summaries",
				Destination: &flags.tsdbFile,
				Value: "summarizer.db",
			},
		},
	},
	{
		Name: "dump-summaries",
		Usage: "print out everything in the summaries db for debugging",
		Action: dumpSummaries,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "tsdb",
				Usage: "path to database to store size summaries",
				Destination: &flags.tsdbFile,
				Value: "summarizer.db",
			},
		},
	},
	{
		Name: "summary-chart",
		Usage: "generate a visualization of the utilization summary",
		Action: summaryChart,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "tsdb",
				Usage: "path to database to store size summaries",
				Destination: &flags.tsdbFile,
				Value: "summarizer.db",
			},
		},
	},
	{
		Name: "averages",
		Usage: "report table showing space utilized from highest to lowest",
		Action: printAvg,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "tsdb",
				Usage: "path to database to store size summaries",
				Destination: &flags.tsdbFile,
				Value: "summarizer.db",
			},
		},
	},
}

func main() {
	app := &cli.App{
		Commands: commands,
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("Fatal error, %v", err)
		os.Exit(1)
	}
}

func bucketSizes(_ *cli.Context) error {
	f := flags
	tsdb, err := bolt.Open(f.tsdbFile, 0600, &bolt.Options{
		Timeout:         1 * time.Second,
		InitialMmapSize: initMMapSize,
	})
	if err != nil {
		return err
	}
	defer tsdb.Close()

	stats, err := getBucketStats(tsdb)
	if err != nil {
		return err
	}
	return bucketSizeChart(stats)
}

func lookForSmallerVals(_ *cli.Context) error {
	f := flags
	db, err := bolt.Open(
		f.dbFile,
		params.BeaconIoConfig().ReadWritePermissions,
		&bolt.Options{
			Timeout:         1 * time.Second,
			InitialMmapSize: initMMapSize,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "error opening db=%s", f.dbFile)
	}
	defer db.Close()
	migrationsBucket := []byte("migrations")
	migrationStateValidatorsKey := []byte("migration_state_validator")
	migrationCompleted := []byte("done")
	returnFlag := false
	err = db.View(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		b := mb.Get(migrationStateValidatorsKey)
		returnFlag = bytes.Equal(b, migrationCompleted)
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("migration enabled = %t", returnFlag)
	return nil

	/*
	store, err := kv.NewKVStoreWithDB(ctx, db)
	if err != nil {
		return err
	}
	 */

	/*
	stateBucket := []byte("state")
	//stateValidatorsBucket := []byte("state-validators")
	blockRootValidatorHashesBucket := []byte("block-root-validator-hashes")
	for sr := range slotRootIter(db) {
		err := db.View(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(stateBucket)
			sb := bkt.Get(sr.Root[:])
			idxBkt := tx.Bucket(blockRootValidatorHashesBucket)
			valKey := idxBkt.Get(sr.Root[:])
			fmt.Printf("state bytes = %d, val bytes = %d", len(sb), len(valKey))
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
	*/
}

func dumpSummaries(_ *cli.Context) error {
	f := flags
	tsdb, err := bolt.Open(f.tsdbFile, 0600, &bolt.Options{
		Timeout:         1 * time.Second,
		InitialMmapSize: initMMapSize,
	})
	if err != nil {
		return err
	}
	defer tsdb.Close()
	/*
	for sum := range summaryIter(tsdb) {
		fmt.Printf("%v\n", sum)
	}
	 */
	return summaryDump(tsdb)
}

func summaryChart(_ *cli.Context) error {
	f := flags
	tsdb, err := bolt.Open(f.tsdbFile, 0600, &bolt.Options{
		Timeout:         1 * time.Second,
		InitialMmapSize: initMMapSize,
	})
	if err != nil {
		return err
	}
	defer tsdb.Close()
	sums := make([]SizeSummary, 0)
	for sum := range summaryIter(tsdb) {
		sums = append(sums, sum)
	}
	/*
	fs := forkSummaries(sums)
	for i := range fs {
		fmt.Println(fs[i].String())
	}
	 */
	return renderLineChart(sums)
}

func printAvg(_ *cli.Context) error {
	f := flags
	tsdb, err := bolt.Open(f.tsdbFile, 0600, &bolt.Options{
		Timeout:         1 * time.Second,
		InitialMmapSize: initMMapSize,
	})
	if err != nil {
		return err
	}
	defer tsdb.Close()
	sums := make([]SizeSummary, 0)
	for sum := range summaryIter(tsdb) {
		sums = append(sums, sum)
	}
	fs := forkSummaries(sums)
	for _, s := range fs {
		printRollup(rollupSizes(s))
	}
	return nil
}

func printRollup(stats SizeStats) {
	fmt.Printf("Stats for fork: %s\n", stats.fork)
	fmt.Println("Average:")
	printSummary(stats.avg)
	fmt.Println("Max:")
	printSummary(stats.max)
	fmt.Println("Min:")
	printSummary(stats.min)
}

func printSummary(sum SizeSummary) {
	fmt.Printf("\t- total: %d\n", sum.Total)
	fmt.Printf("\t- validators: %d\n", sum.Validators)
	fmt.Printf("\t- inactivity_scores: %d\n", sum.InactivityScores)
	fmt.Printf("\t- balances: %d\n", sum.Balances)
	fmt.Printf("\t- randao_mixes: %d\n", sum.RandaoMixes)
	fmt.Printf("\t- previous_epoch_participation: %d\n", sum.PreviousEpochParticipation)
	fmt.Printf("\t- current_epoch_participation: %d\n", sum.CurrentEpochParticipation)
	fmt.Printf("\t- block_roots: %d\n", sum.BlockRoots)
	fmt.Printf("\t- state_roots: %d\n", sum.StateRoots)
	fmt.Printf("\t- slashings: %d\n", sum.Slashings)
	fmt.Printf("\t- current_sync_committee: %d\n", sum.CurrentSyncCommittee)
	fmt.Printf("\t- next_sync_committee: %d\n", sum.NextSyncCommittee)
	fmt.Printf("\t- eth1_data_votes: %d\n", sum.Eth1DataVotes)
	fmt.Printf("\t- historical_roots: %d\n", sum.HistoricalRoots)
	fmt.Printf("\t- latest_block_header: %d\n", sum.LatestBlockHeader)
	fmt.Printf("\t- eth1_data: %d\n", sum.Eth1Data)
	fmt.Printf("\t- previously_justified_checkpoint: %d\n", sum.PreviouslyJustifiedCheckpoint)
	fmt.Printf("\t- current_justified_checkpoint: %d\n", sum.CurrentJustifiedCheckpoint)
	fmt.Printf("\t- finalized_checkpoint: %d\n", sum.FinalizedCheckpoint)
	fmt.Printf("\t- fork: %d\n", sum.Fork)
	fmt.Printf("\t- genesis_validators_root: %d\n", sum.GenesisValidatorsRoot)
	fmt.Printf("\t- genesis_time: %d\n", sum.GenesisTime)
	fmt.Printf("\t- slot: %d\n", sum.Slot)
	fmt.Printf("\t- eth1_deposit_index: %d\n", sum.Eth1DepositIndex)
	fmt.Printf("\t- justification_bits: %d\n", sum.JustificationBits)
	fmt.Printf("\t- previous_epoch_attestations: %d\n", sum.PreviousEpochAttestations)
	fmt.Printf("\t- current_epoch_attestations: %d\n", sum.CurrentEpochAttestations)
	fmt.Printf("\t- latest_execution_payload_header: %d\n", sum.LatestExecutionPayloadHeader)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func rollupSizes(fs ForkSummary) SizeStats {
	szs := SizeStats{fork: fs.name}
	ss := fs.sb[0].summaries
	for _, s := range ss {
		szs.min.Total = min(szs.min.Total, s.Total)
		szs.max.Total = max(szs.max.Total, s.Total)
		szs.avg.Total += s.Total
		szs.min.GenesisTime = min(szs.min.GenesisTime, s.GenesisTime)
		szs.max.GenesisTime = max(szs.max.GenesisTime, s.GenesisTime)
		szs.avg.GenesisTime += s.GenesisTime
		szs.min.GenesisValidatorsRoot = min(szs.min.GenesisValidatorsRoot, s.GenesisValidatorsRoot)
		szs.max.GenesisValidatorsRoot = max(szs.max.GenesisValidatorsRoot, s.GenesisValidatorsRoot)
		szs.avg.GenesisValidatorsRoot += s.GenesisValidatorsRoot
		szs.min.Slot = min(szs.min.Slot, s.Slot)
		szs.max.Slot = max(szs.max.Slot, s.Slot)
		szs.avg.Slot += s.Slot
		szs.min.Fork = min(szs.min.Fork, s.Fork)
		szs.max.Fork = max(szs.max.Fork, s.Fork)
		szs.avg.Fork += s.Fork
		szs.min.LatestBlockHeader = min(szs.min.LatestBlockHeader , s.LatestBlockHeader )
		szs.max.LatestBlockHeader = max(szs.max.LatestBlockHeader , s.LatestBlockHeader )
		szs.avg.LatestBlockHeader += s.LatestBlockHeader
		szs.min.BlockRoots = min(szs.min.BlockRoots , s.BlockRoots )
		szs.max.BlockRoots = max(szs.max.BlockRoots , s.BlockRoots )
		szs.avg.BlockRoots += s.BlockRoots
		szs.min.StateRoots = min(szs.min.StateRoots , s.StateRoots )
		szs.max.StateRoots = max(szs.max.StateRoots , s.StateRoots )
		szs.avg.StateRoots += s.StateRoots
		szs.min.HistoricalRoots = min(szs.min.HistoricalRoots , s.HistoricalRoots )
		szs.max.HistoricalRoots = max(szs.max.HistoricalRoots , s.HistoricalRoots )
		szs.avg.HistoricalRoots += s.HistoricalRoots
		szs.min.Eth1Data = min(szs.min.Eth1Data , s.Eth1Data )
		szs.max.Eth1Data = max(szs.max.Eth1Data , s.Eth1Data )
		szs.avg.Eth1Data += s.Eth1Data
		szs.min.Eth1DataVotes = min(szs.min.Eth1DataVotes , s.Eth1DataVotes )
		szs.max.Eth1DataVotes = max(szs.max.Eth1DataVotes , s.Eth1DataVotes )
		szs.avg.Eth1DataVotes += s.Eth1DataVotes
		szs.min.Eth1DepositIndex = min(szs.min.Eth1DepositIndex , s.Eth1DepositIndex )
		szs.max.Eth1DepositIndex = max(szs.max.Eth1DepositIndex , s.Eth1DepositIndex )
		szs.avg.Eth1DepositIndex += s.Eth1DepositIndex
		szs.min.Validators = min(szs.min.Validators , s.Validators )
		szs.max.Validators = max(szs.max.Validators , s.Validators )
		szs.avg.Validators += s.Validators
		szs.min.Balances = min(szs.min.Balances , s.Balances )
		szs.max.Balances = max(szs.max.Balances , s.Balances )
		szs.avg.Balances += s.Balances
		szs.min.RandaoMixes = min(szs.min.RandaoMixes , s.RandaoMixes )
		szs.max.RandaoMixes = max(szs.max.RandaoMixes , s.RandaoMixes )
		szs.avg.RandaoMixes += s.RandaoMixes
		szs.min.Slashings = min(szs.min.Slashings , s.Slashings )
		szs.max.Slashings = max(szs.max.Slashings , s.Slashings )
		szs.avg.Slashings += s.Slashings
		szs.min.PreviousEpochAttestations = min(szs.min.PreviousEpochAttestations , s.PreviousEpochAttestations )
		szs.max.PreviousEpochAttestations = max(szs.max.PreviousEpochAttestations , s.PreviousEpochAttestations )
		szs.avg.PreviousEpochAttestations += s.PreviousEpochAttestations
		szs.min.CurrentEpochAttestations = min(szs.min.CurrentEpochAttestations , s.CurrentEpochAttestations )
		szs.max.CurrentEpochAttestations = max(szs.max.CurrentEpochAttestations , s.CurrentEpochAttestations )
		szs.avg.CurrentEpochAttestations += s.CurrentEpochAttestations
		szs.min.PreviousEpochParticipation = min(szs.min.PreviousEpochParticipation , s.PreviousEpochParticipation )
		szs.max.PreviousEpochParticipation = max(szs.max.PreviousEpochParticipation , s.PreviousEpochParticipation )
		szs.avg.PreviousEpochParticipation += s.PreviousEpochParticipation
		szs.min.CurrentEpochParticipation = min(szs.min.CurrentEpochParticipation , s.CurrentEpochParticipation )
		szs.max.CurrentEpochParticipation = max(szs.max.CurrentEpochParticipation , s.CurrentEpochParticipation )
		szs.avg.CurrentEpochParticipation += s.CurrentEpochParticipation
		szs.min.JustificationBits = min(szs.min.JustificationBits , s.JustificationBits )
		szs.max.JustificationBits = max(szs.max.JustificationBits , s.JustificationBits )
		szs.avg.JustificationBits += s.JustificationBits
		szs.min.PreviouslyJustifiedCheckpoint = min(szs.min.PreviouslyJustifiedCheckpoint , s.PreviouslyJustifiedCheckpoint )
		szs.max.PreviouslyJustifiedCheckpoint = max(szs.max.PreviouslyJustifiedCheckpoint , s.PreviouslyJustifiedCheckpoint )
		szs.avg.PreviouslyJustifiedCheckpoint += s.PreviouslyJustifiedCheckpoint
		szs.min.CurrentJustifiedCheckpoint = min(szs.min.CurrentJustifiedCheckpoint , s.CurrentJustifiedCheckpoint )
		szs.max.CurrentJustifiedCheckpoint = max(szs.max.CurrentJustifiedCheckpoint , s.CurrentJustifiedCheckpoint )
		szs.avg.CurrentJustifiedCheckpoint += s.CurrentJustifiedCheckpoint
		szs.min.FinalizedCheckpoint = min(szs.min.FinalizedCheckpoint , s.FinalizedCheckpoint )
		szs.max.FinalizedCheckpoint = max(szs.max.FinalizedCheckpoint , s.FinalizedCheckpoint )
		szs.avg.FinalizedCheckpoint += s.FinalizedCheckpoint
		szs.min.InactivityScores = min(szs.min.InactivityScores , s.InactivityScores )
		szs.max.InactivityScores = max(szs.max.InactivityScores , s.InactivityScores )
		szs.avg.InactivityScores += s.InactivityScores
		szs.min.CurrentSyncCommittee = min(szs.min.CurrentSyncCommittee , s.CurrentSyncCommittee )
		szs.max.CurrentSyncCommittee = max(szs.max.CurrentSyncCommittee , s.CurrentSyncCommittee )
		szs.avg.CurrentSyncCommittee += s.CurrentSyncCommittee
		szs.min.NextSyncCommittee = min(szs.min.NextSyncCommittee , s.NextSyncCommittee )
		szs.max.NextSyncCommittee = max(szs.max.NextSyncCommittee , s.NextSyncCommittee )
		szs.avg.NextSyncCommittee += s.NextSyncCommittee
		szs.min.LatestExecutionPayloadHeader = min(szs.min.LatestExecutionPayloadHeader , s.LatestExecutionPayloadHeader )
		szs.max.LatestExecutionPayloadHeader = max(szs.max.LatestExecutionPayloadHeader , s.LatestExecutionPayloadHeader )
		szs.avg.LatestExecutionPayloadHeader += s.LatestExecutionPayloadHeader
	}
	szs.avg.Total = szs.avg.Total / len(ss)
	szs.avg.GenesisTime = szs.avg.GenesisTime / len(ss)
	szs.avg.GenesisValidatorsRoot = szs.avg.GenesisValidatorsRoot / len(ss)
	szs.avg.Slot = szs.avg.Slot / len(ss)
	szs.avg.Fork = szs.avg.Fork / len(ss)
	szs.avg.LatestBlockHeader = szs.avg.LatestBlockHeader / len(ss)
	szs.avg.BlockRoots = szs.avg.BlockRoots / len(ss)
	szs.avg.StateRoots = szs.avg.StateRoots / len(ss)
	szs.avg.HistoricalRoots = szs.avg.HistoricalRoots / len(ss)
	szs.avg.Eth1Data = szs.avg.Eth1Data / len(ss)
	szs.avg.Eth1DataVotes = szs.avg.Eth1DataVotes / len(ss)
	szs.avg.Eth1DepositIndex = szs.avg.Eth1DepositIndex / len(ss)
	szs.avg.Validators = szs.avg.Validators / len(ss)
	szs.avg.Balances = szs.avg.Balances / len(ss)
	szs.avg.RandaoMixes = szs.avg.RandaoMixes / len(ss)
	szs.avg.Slashings = szs.avg.Slashings / len(ss)
	szs.avg.PreviousEpochAttestations = szs.avg.PreviousEpochAttestations / len(ss)
	szs.avg.CurrentEpochAttestations = szs.avg.CurrentEpochAttestations / len(ss)
	szs.avg.PreviousEpochParticipation = szs.avg.PreviousEpochParticipation / len(ss)
	szs.avg.CurrentEpochParticipation = szs.avg.CurrentEpochParticipation / len(ss)
	szs.avg.JustificationBits = szs.avg.JustificationBits / len(ss)
	szs.avg.PreviouslyJustifiedCheckpoint = szs.avg.PreviouslyJustifiedCheckpoint / len(ss)
	szs.avg.CurrentJustifiedCheckpoint = szs.avg.CurrentJustifiedCheckpoint / len(ss)
	szs.avg.FinalizedCheckpoint = szs.avg.FinalizedCheckpoint / len(ss)
	szs.avg.InactivityScores = szs.avg.InactivityScores / len(ss)
	szs.avg.CurrentSyncCommittee = szs.avg.CurrentSyncCommittee / len(ss)
	szs.avg.NextSyncCommittee = szs.avg.NextSyncCommittee / len(ss)
	szs.avg.LatestExecutionPayloadHeader = szs.avg.LatestExecutionPayloadHeader / len(ss)
	return szs
}

type SizeStats struct {
	fork string
	min SizeSummary
	max SizeSummary
	avg SizeSummary
}

var fieldLabels = []string{
	"validators",
	"inactivity_scores",
	"balances",
	"randao_mixes",
	"previous_epoch_attestations",
	"current_epoch_attestations",
	"previous_epoch_participation",
	"current_epoch_participation",
	"block_roots",
	"state_roots",
	"slashings",
	"current_sync_committee",
	"next_sync_committee",
	"eth1_data_votes",
	"historical_roots",
	"latest_block_header",
	"eth1_data",
	"previously_justified_checkpoint",
	"current_justified_checkpoint",
	"finalized_checkpoint",
	"genesis_validators_root",
	"fork",
	"genesis_time",
	"slot",
	"eth1_deposit_index",
	"justification_bits",
	"latest_execution_payload_header",
}

func render3dbarChart(fss []ForkSummary) error {
	page := components.NewPage()

	for _, fs := range fss {
		title := fmt.Sprintf("%s: avg field sizes of states", fs.name)
		bucketNames := make([]string, 0)
		for _, b := range fs.sb {
			bucketNames = append(bucketNames, fmt.Sprintf("%d-%d", b.min, b.max))
		}
		bar3d := charts.NewBar3D()
		bar3d.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: title}),
			charts.WithGrid3DOpts(opts.Grid3D{
				BoxWidth: 200,
				BoxDepth: 80,
			}),
		)
		bar3d.SetGlobalOptions(
			charts.WithXAxis3DOpts(opts.XAxis3D{Data: bucketNames}),
			charts.WithYAxis3DOpts(opts.YAxis3D{Data: fieldLabels}),
		)
		page.AddCharts(bar3d)
	}
	f, err := os.Create("bar3d.html")
	if err != nil {
		return nil
	}
	defer f.Close()
	page.Render(io.MultiWriter(f))
	return nil
}

func lineChartData(ss []SizeSummary) map[string][]opts.LineData {
	vectors := make(map[string][]opts.LineData)
	for _, s := range ss {
		vectors["genesis_time"] = append(vectors["genesis_time"], opts.LineData{Value: s.GenesisTime})
		vectors["genesis_validators_root"] = append(vectors["genesis_validators_root"], opts.LineData{Value: s.GenesisValidatorsRoot})
		vectors["slot"] = append(vectors["slot"], opts.LineData{Value: s.Slot})
		vectors["fork"] = append(vectors["fork"], opts.LineData{Value: s.Fork})
		vectors["latest_block_header"] = append(vectors["latest_block_header"], opts.LineData{Value: s.LatestBlockHeader})
		vectors["block_roots"] = append(vectors["block_roots"], opts.LineData{Value: s.BlockRoots})
		vectors["state_roots"] = append(vectors["state_roots"], opts.LineData{Value: s.StateRoots})
		vectors["historical_roots"] = append(vectors["historical_roots"], opts.LineData{Value: s.HistoricalRoots})
		vectors["eth1_data"] = append(vectors["eth1_data"], opts.LineData{Value: s.Eth1Data})
		vectors["eth1_data_votes"] = append(vectors["eth1_data_votes"], opts.LineData{Value: s.Eth1DataVotes})
		vectors["eth1_deposit_index"] = append(vectors["eth1_deposit_index"], opts.LineData{Value: s.Eth1DepositIndex})
		vectors["validators"] = append(vectors["validators"], opts.LineData{Value: s.Validators})
		vectors["balances"] = append(vectors["balances"], opts.LineData{Value: s.Balances})
		vectors["randao_mixes"] = append(vectors["randao_mixes"], opts.LineData{Value: s.RandaoMixes})
		vectors["slashings"] = append(vectors["slashings"], opts.LineData{Value: s.Slashings})
		vectors["previous_epoch_attestations"] = append(vectors["previous_epoch_attestations"], opts.LineData{Value: s.PreviousEpochAttestations})
		vectors["current_epoch_attestations"] = append(vectors["current_epoch_attestations"], opts.LineData{Value: s.CurrentEpochAttestations})
		vectors["previous_epoch_participation"] = append(vectors["previous_epoch_participation"], opts.LineData{Value: s.PreviousEpochParticipation})
		vectors["current_epoch_participation"] = append(vectors["current_epoch_participation"], opts.LineData{Value: s.CurrentEpochParticipation})
		vectors["justification_bits"] = append(vectors["justification_bits"], opts.LineData{Value: s.JustificationBits})
		vectors["previously_justified_checkpoint"] = append(vectors["previously_justified_checkpoint"], opts.LineData{Value: s.PreviouslyJustifiedCheckpoint})
		vectors["current_justified_checkpoint"] = append(vectors["current_justified_checkpoint"], opts.LineData{Value: s.CurrentJustifiedCheckpoint})
		vectors["finalized_checkpoint"] = append(vectors["finalized_checkpoint"], opts.LineData{Value: s.FinalizedCheckpoint})
		vectors["inactivity_scores"] = append(vectors["inactivity_scores"], opts.LineData{Value: s.InactivityScores})
		vectors["current_sync_committee"] = append(vectors["current_sync_committee"], opts.LineData{Value: s.CurrentSyncCommittee})
		vectors["next_sync_committee"] = append(vectors["next_sync_committee"], opts.LineData{Value: s.NextSyncCommittee})
		vectors["latest_execution_payload_header"] = append(vectors["latest_execution_payload_header"], opts.LineData{Value: s.LatestExecutionPayloadHeader})
	}
	return vectors
}

func renderLineChart(ss []SizeSummary) error {
	page := components.NewPage()
	page.SetLayout(components.PageFlexLayout)

	xaxis := make([]string, len(ss))
	for i, s := range ss {
		xaxis[i] = fmt.Sprintf("%d", s.SlotRoot.Slot)
	}
	line := charts.NewLine()
	line.SetXAxis(xaxis)
	lcd := lineChartData(ss)
	for _, name := range fieldLabels {
		points := lcd[name]
		line.AddSeries(name, points)
	}
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "growth of BeaconState components",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "bytes",
			SplitLine: &opts.SplitLine{
				Show: false,
			},
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "slot",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   true,
			Data:   fieldLabels,
			Left:   "0",
			Bottom: "0",
			Orient: "horizontal",
			Type: "scroll",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:        true,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1000px",
			Height: "600px",
		}),
	)

	line.SetSeriesOptions(
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: true,
		}),
		/*
		charts.WithMarkLineNameTypeItemOpts(opts.MarkLineNameTypeItem{
			Name: "Average",
			Type: "average",
		}),
		charts.WithMarkPointStyleOpts(opts.MarkPointStyle{
			Label: &opts.Label{
				Show:      true,
				Formatter: "{a}: {b}",
			},
		}),
		 */
	)

	page.AddCharts(line)
	f, err := os.Create("line.html")
	if err != nil {
		return nil
	}
	defer f.Close()
	page.Render(io.MultiWriter(f))
	return nil
}

func bucketSizeChart(bs map[string]int) error {
	page := components.NewPage()
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "DB Bucket Sizes"}),
	)

	items := make([]opts.PieData, 0)
	for k, v := range bs {
		items = append(items, opts.PieData{Name: k, Value: v})
	}
	pie.AddSeries("pie", items).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:      true,
				Formatter: "{b}: {c}",
			}),
			charts.WithPieChartOpts(opts.PieChart{
				Radius: []string{"40%", "75%"},
			}),
		)
	page.AddCharts(pie)

	f, err := os.Create("db-bucket-sizes.html")
	if err != nil {
		return nil
	}
	defer f.Close()
	return page.Render(io.MultiWriter(f))
}

func summarize(_ *cli.Context) error {
	ctx := context.Background()
	f := flags
	tsdb, err := bolt.Open(f.tsdbFile, 0600, &bolt.Options{
		Timeout:         1 * time.Second,
		InitialMmapSize: initMMapSize,
	})
	if err != nil {
		return err
	}
	defer tsdb.Close()
	if err := dbinit(tsdb); err != nil {
		return errors.Wrapf(err, "error opening tsdb=%s", f.tsdbFile)
	}
	db, err := bolt.Open(
		f.dbFile,
		params.BeaconIoConfig().ReadWritePermissions,
		&bolt.Options{
			Timeout:         1 * time.Second,
			InitialMmapSize: initMMapSize,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "error opening db=%s", f.dbFile)
	}
	defer db.Close()

	store, err := kv.NewKVStoreWithDB(ctx, db)
	if err != nil {
		return err
	}

	// do this first since it reads everything sequentially, which warms up the page cache
	stats := make(map[string]int)
	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if k != nil && v == nil {
				bkt := tx.Bucket(k)
				bc := bkt.Cursor()
				l := 0
				for bk, bv := bc.First(); bk != nil; bk, bv = bc.Next() {
					l += len(bk) + len(bv)
				}
				fmt.Printf("%s: %d bytes\n", string(k), l)
				stats[string(k)] = l
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error while recording db bucket sizes")
	}
	for k, v := range stats {
		if err := writeBucketStat(tsdb, k, v); err != nil {
			return err
		}
	}

	for sr := range slotRootIter(db) {
		st, err := store.State(ctx, sr.Root)
		if err != nil {
			return errors.Wrapf(err, "unable to fetch state for root=%#x", sr.Root)
		}
		sb, err := st.MarshalSSZ()
		if err != nil {
			return errors.Wrapf(err, "unable to marshal state w/ root=%#x", sr.Root)
		}
		sz := &summarizer{s: st, sb: sb, sr: sr}
		sum := sz.Summary()
		if err := writeSummary(tsdb, sum); err != nil {
			return err
		}
		fmt.Printf("wrote summary for slot=%d, root=%#x\n", sr.Slot, sr.Root)
	}
	return nil
}

type SlotRoot struct {
	Slot types.Slot `json:"slot"`
	Root [32]byte `json:"root"`
}

func slotRootIter(db *bolt.DB) chan SlotRoot {
	ch := make(chan SlotRoot)
	go func() {
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(buckets.stateSlotRoots)
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				ch <- SlotRoot{Slot: bytesutil.BytesToSlotBigEndian(k), Root: bytesutil.ToBytes32(v)}
			}
			close(ch)
			return nil
		})
		if err != nil {
			panic(err)
		}
	}()
	return ch
}

type ForkSummary struct {
	name string
	sb []SummaryBucket
}

func (fs ForkSummary) String() string {
	bs := make([]string, len(fs.sb))
	for i, sb := range fs.sb {
		bs[i] = sb.String()
	}
	return fmt.Sprintf("name=%s, buckets=[%s]", fs.name, strings.Join(bs, ","))
}

type SummaryBucket struct {
	summaries []SizeSummary
	min types.Slot
	max types.Slot
}

func (sb SummaryBucket) String() string {
	return fmt.Sprintf("(len=%d, min=%d, max=%d)", len(sb.summaries), sb.min, sb.max)
}

func forkSummaries(sums []SizeSummary) []ForkSummary {
	altairIdx := findAltairOffset(sums)
	return []ForkSummary{
		{
			name: "phase0",
			sb: splitBuckets(sums[0:altairIdx], 1),
		},
		{
			name: "altair",
			sb: splitBuckets(sums[altairIdx:], 1),
		},
	}
}

func findAltairOffset(sums []SizeSummary) int {
	for i := range sums {
		// hardcoded prater value
		if sums[i].SlotRoot.Slot >= 36660*32 {
			return i
		}
	}
	return len(sums)
}

func splitBuckets(sums []SizeSummary, numBuckets int) []SummaryBucket {
	bs := len(sums) / numBuckets
	bkts := make([]SummaryBucket, numBuckets)
	cb := 0
	bkts[0].min = sums[0].SlotRoot.Slot
	for _, sum := range sums {
		if len(bkts[cb].summaries) == bs && cb < numBuckets - 1 {
			cb += 1
			bkts[cb].min = sum.SlotRoot.Slot
		}
		bkts[cb].max = sum.SlotRoot.Slot
		bkts[cb].summaries = append(bkts[cb].summaries, sum)
	}
	return bkts
}

type SizeSummary struct {
	SlotRoot SlotRoot `json:"slot_root"`
	Total int `json:"total"`
	GenesisTime int `json:"genesis_time"`
	GenesisValidatorsRoot int `json:"genesis_validators_root"`
	Slot int `json:"slot"`
	Fork int `json:"fork"`
	LatestBlockHeader int `json:"latest_block_header"`
	BlockRoots int `json:"block_roots"`
	StateRoots int `json:"state_roots"`
	HistoricalRoots int `json:"historical_roots"`
	Eth1Data int `json:"eth1_data"`
	Eth1DataVotes int `json:"eth1_data_votes"`
	Eth1DepositIndex int `json:"eth1_deposit_index"`
	Validators int `json:"validators"`
	Balances int `json:"balances"`
	RandaoMixes int `json:"randao_mixes"`
	Slashings int `json:"slashings"`
	PreviousEpochAttestations int `json:"previous_epoch_attestations"`
	CurrentEpochAttestations int `json:"current_epoch_attestations"`
	PreviousEpochParticipation int `json:"previous_epoch_participation"`
	CurrentEpochParticipation int `json:"current_epoch_participation"`
	JustificationBits int `json:"justification_bits"`
	PreviouslyJustifiedCheckpoint int `json:"previously_justified_checkpoint"`
	CurrentJustifiedCheckpoint int `json:"current_justified_checkpoint"`
	FinalizedCheckpoint int `json:"finalized_checkpoint"`
	InactivityScores int `json:"inactivity_scores"`
	CurrentSyncCommittee int `json:"current_sync_committee"`
	NextSyncCommittee int `json:"next_sync_committee"`
	LatestExecutionPayloadHeader int `json:"latest_execution_payload_header"`
}
