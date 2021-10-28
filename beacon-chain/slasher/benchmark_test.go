package slasher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/io/file"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func TestSlasherTimes(t *testing.T) {
	slasherDB := dbtest.SetupSlasherDB(t)
	ctx := context.Background()

	// Initializing the service
	srv, err := New(ctx, &ServiceConfig{
		Database: slasherDB,
	})
	require.NoError(t, err)

	dirs := map[string]types.Slot{"/tmp/attestations": 2380670, "/tmp/attestations2": 2380671}
	for dir, slot := range dirs {
		indexedAttWrappers, err := fetchIndexedAttestationWrapperFixtures(dir)
		require.NoError(t, err)
		require.NoError(t, srv.serviceCfg.Database.SaveAttestationRecordsForValidators(ctx, indexedAttWrappers))

		// Set the current epoch to the epoch the block where the attestations originated was extracted from + 1.
		currentEpoch := slots.ToEpoch(slot) + 1
		start := time.Now()
		_, err = srv.checkSlashableAttestations(ctx, currentEpoch, indexedAttWrappers)
		require.NoError(t, err)
		t.Logf("Took %v to process %d atts", time.Since(start), len(indexedAttWrappers))
		t.Logf("---------------------------------------------")
	}
}

func fetchIndexedAttestationWrapperFixtures(dirPath string) ([]*slashertypes.IndexedAttestationWrapper, error) {
	attestations, err := readAttestationsFromDisk(dirPath)
	if err != nil {
		return nil, err
	}

	// Converting the attestations from the block into indexed format.
	indexedAttWrappers := make([]*slashertypes.IndexedAttestationWrapper, len(attestations))
	for i, att := range attestations {
		signingRoot, err := att.Data.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		indexedAttWrappers[i] = &slashertypes.IndexedAttestationWrapper{
			IndexedAttestation: att,
			SigningRoot:        signingRoot,
		}
	}
	return indexedAttWrappers, nil
}

func readAttestationsFromDisk(dirPath string) ([]*ethpb.IndexedAttestation, error) {
	dirItems, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	atts := make([]*ethpb.IndexedAttestation, 0)
	for _, item := range dirItems {
		attFilePath := filepath.Join(dirPath, item.Name())
		enc, err := file.ReadFileAsBytes(attFilePath)
		if err != nil {
			return nil, err
		}
		idxAtt := &ethpb.IndexedAttestation{}
		if err := idxAtt.UnmarshalSSZ(enc); err != nil {
			return nil, err
		}
		atts = append(atts, idxAtt)
	}
	return atts, nil
}
