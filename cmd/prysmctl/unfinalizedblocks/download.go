package unfinalizedblocks

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var downloadFlags = struct {
	BeaconNodeHost string
	Timeout        time.Duration
	Dir            string
}{}

const (
	ArchiveFilename = "unfinalized_blocks.zip"
)

var downloadCmd = &cli.Command{
	Name:    "download",
	Aliases: []string{"dl"},
	Usage:   "Download all the unfinalized blocks in ssz form to zipfile.",
	Action:  cliActionDownload,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node connection",
			Destination: &downloadFlags.BeaconNodeHost,
			Value:       "http://localhost:3500",
		},
		&cli.DurationFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 20s",
			Destination: &downloadFlags.Timeout,
			Value:       time.Second * 20,
		},
		&cli.StringFlag{
			Name:        "outdir",
			Usage:       "location on disk to save the output zipfile to. default: /tmp",
			Destination: &downloadFlags.Dir,
			Value:       "/tmp",
		},
	},
}

func cliActionDownload(_ *cli.Context) error {
	ctx := context.Background()
	f := downloadFlags

	opts := []beacon.ClientOpt{beacon.WithTimeout(f.Timeout), beacon.WithRetry(10)}
	client, err := beacon.NewClient(downloadFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	blocks, err := beacon.DownloadUnfinalizedBlocks(ctx, client)
	if err != nil {
		return err
	}

	err = zipBlocks(blocks, f.Dir)
	if err != nil {
		return err
	}

	return nil
}

// Zips a list of blocks in ssz format and outputs to current directory
func zipBlocks(blocks []interfaces.BeaconBlock, outputDir string) error {
	if len(blocks) == 0 {
		return errors.New("nothing to zip")
	}

	archivePath := filepath.Join(outputDir, ArchiveFilename)
	if file.FileExists(archivePath) {
		return errors.Errorf("zip file already exists in directory: %s", archivePath)
	}

	zipfile, err := os.Create(filepath.Clean(archivePath))
	if err != nil {
		return errors.Wrapf(err, "could not create zip file with path: %s", archivePath)
	}
	defer func() {
		if err := zipfile.Close(); err != nil {
			log.Printf("could not close zipfile")
		}
	}()

	writer := zip.NewWriter(zipfile)
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("could not close zip file after writing")
		}
	}()

	for _, b := range blocks {
		htr, err := b.HashTreeRoot()
		f, err := writer.Create(fmt.Sprintf("%d-%s.ssz", b.Slot(), beacon.IdFromRoot(htr)))
		if err != nil {
			return errors.Wrap(err, "could not write block file to zip")
		}

		if _, err = f.Write(htr[:]); err != nil {
			return errors.Wrap(err, "could not write block file contents")
		}
	}
	log.Printf("output saved to %s", archivePath)
	return nil
}
