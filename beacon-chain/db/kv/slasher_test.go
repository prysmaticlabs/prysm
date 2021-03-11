package kv

import (
	"context"
	"reflect"
	"testing"

	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_AttestationRecordForValidator_SaveRetrieve(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	valIdx := types.ValidatorIndex(1)
	target := types.Epoch(5)
	source := types.Epoch(4)
	attRecord, err := beaconDB.AttestationRecordForValidator(ctx, valIdx, target)
	require.NoError(t, err)
	require.Equal(t, true, attRecord == nil)

	sr := [32]byte{1}
	err = beaconDB.SaveAttestationRecordsForValidators(ctx, []*slashertypes.IndexedAttestationWrapper{
		createAttestationWrapper(source, target, []uint64{uint64(valIdx)}, sr[:]),
	})
	require.NoError(t, err)
	attRecord, err = beaconDB.AttestationRecordForValidator(ctx, valIdx, target)
	require.NoError(t, err)
	assert.DeepEqual(t, target, attRecord.IndexedAttestation.Data.Target.Epoch)
	assert.DeepEqual(t, source, attRecord.IndexedAttestation.Data.Source.Epoch)
	assert.DeepEqual(t, sr, attRecord.SigningRoot)
}

func TestStore_LastEpochWrittenForValidators(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	indices := []types.ValidatorIndex{1, 2, 3}
	epoch := types.Epoch(5)

	attestedEpochs, err := beaconDB.LastEpochWrittenForValidators(ctx, indices)
	require.NoError(t, err)
	require.Equal(t, true, len(attestedEpochs) == 0)

	err = beaconDB.SaveLastEpochWrittenForValidators(ctx, indices, epoch)
	require.NoError(t, err)

	retrievedEpochs, err := beaconDB.LastEpochWrittenForValidators(ctx, indices)
	require.NoError(t, err)
	require.Equal(t, len(indices), len(retrievedEpochs))

	for i, retrievedEpoch := range retrievedEpochs {
		want := &slashertypes.AttestedEpochForValidator{
			Epoch:          epoch,
			ValidatorIndex: indices[i],
		}
		require.DeepEqual(t, want, retrievedEpoch)
	}
}

func TestStore_CheckAttesterDoubleVotes(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	err := beaconDB.SaveAttestationRecordsForValidators(ctx, []*slashertypes.IndexedAttestationWrapper{
		createAttestationWrapper(2, 3, []uint64{0, 1}, []byte{1}),
		createAttestationWrapper(3, 4, []uint64{2, 3}, []byte{1}),
	})
	require.NoError(t, err)

	slashableAtts := []*slashertypes.IndexedAttestationWrapper{
		createAttestationWrapper(2, 3, []uint64{0, 1}, []byte{2}), // Different signing root.
		createAttestationWrapper(3, 4, []uint64{2, 3}, []byte{2}), // Different signing root.
	}

	wanted := []*slashertypes.AttesterDoubleVote{
		{
			ValidatorIndex:         0,
			Target:                 3,
			PrevAttestationWrapper: createAttestationWrapper(2, 3, []uint64{0, 1}, []byte{1}),
			AttestationWrapper:     createAttestationWrapper(2, 3, []uint64{0, 1}, []byte{2}),
		},
		{
			ValidatorIndex:         1,
			Target:                 3,
			PrevAttestationWrapper: createAttestationWrapper(2, 3, []uint64{0, 1}, []byte{1}),
			AttestationWrapper:     createAttestationWrapper(2, 3, []uint64{0, 1}, []byte{2}),
		},
		{
			ValidatorIndex:         2,
			Target:                 4,
			PrevAttestationWrapper: createAttestationWrapper(3, 4, []uint64{2, 3}, []byte{1}),
			AttestationWrapper:     createAttestationWrapper(3, 4, []uint64{2, 3}, []byte{2}),
		},
		{
			ValidatorIndex:         3,
			Target:                 4,
			PrevAttestationWrapper: createAttestationWrapper(3, 4, []uint64{2, 3}, []byte{1}),
			AttestationWrapper:     createAttestationWrapper(3, 4, []uint64{2, 3}, []byte{2}),
		},
	}
	doubleVotes, err := beaconDB.CheckAttesterDoubleVotes(ctx, slashableAtts)
	require.NoError(t, err)
	require.DeepEqual(t, wanted, doubleVotes)
}

func TestStore_SlasherChunk_SaveRetrieve(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	elemsPerChunk := 16
	totalChunks := 64
	chunkKeys := make([][]byte, totalChunks)
	chunks := make([][]uint16, totalChunks)
	for i := 0; i < totalChunks; i++ {
		chunk := make([]uint16, elemsPerChunk)
		for j := 0; j < len(chunk); j++ {
			chunk[j] = uint16(0)
		}
		chunks[i] = chunk
		chunkKeys[i] = ssz.MarshalUint64(make([]byte, 0), uint64(i))
	}

	// We save chunks for min spans.
	err := beaconDB.SaveSlasherChunks(ctx, slashertypes.MinSpan, chunkKeys, chunks)
	require.NoError(t, err)

	// We expect no chunks to be stored for max spans.
	_, chunksExist, err := beaconDB.LoadSlasherChunks(
		ctx, slashertypes.MaxSpan, chunkKeys,
	)
	require.NoError(t, err)
	require.Equal(t, len(chunks), len(chunksExist))
	for _, exists := range chunksExist {
		require.Equal(t, false, exists)
	}

	// We check we saved the right chunks.
	retrievedChunks, chunksExist, err := beaconDB.LoadSlasherChunks(
		ctx, slashertypes.MinSpan, chunkKeys,
	)
	require.NoError(t, err)
	require.Equal(t, len(chunks), len(retrievedChunks))
	require.Equal(t, len(chunks), len(chunksExist))
	for i, exists := range chunksExist {
		require.Equal(t, true, exists)
		require.DeepEqual(t, chunks[i], retrievedChunks[i])
	}

	// We save chunks for max spans.
	err = beaconDB.SaveSlasherChunks(ctx, slashertypes.MaxSpan, chunkKeys, chunks)
	require.NoError(t, err)

	// We check we saved the right chunks.
	retrievedChunks, chunksExist, err = beaconDB.LoadSlasherChunks(
		ctx, slashertypes.MaxSpan, chunkKeys,
	)
	require.NoError(t, err)
	require.Equal(t, len(chunks), len(retrievedChunks))
	require.Equal(t, len(chunks), len(chunksExist))
	for i, exists := range chunksExist {
		require.Equal(t, true, exists)
		require.DeepEqual(t, chunks[i], retrievedChunks[i])
	}
}

func TestStore_ExistingBlockProposals(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	proposals := []*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(1, 1, []byte{1}),
		createProposalWrapper(2, 1, []byte{1}),
		createProposalWrapper(3, 1, []byte{1}),
	}
	// First time checking should return empty existing proposals.
	doubleProposals, err := beaconDB.CheckDoubleBlockProposals(ctx, proposals)
	require.NoError(t, err)
	require.Equal(t, 0, len(doubleProposals))

	// We then save the block proposals to disk.
	err = beaconDB.SaveBlockProposals(ctx, proposals)
	require.NoError(t, err)

	// Second time checking same proposals but all with different signing root should
	// return all double proposals.
	proposals[0].SigningRoot = bytesutil.ToBytes32([]byte{2})
	proposals[1].SigningRoot = bytesutil.ToBytes32([]byte{2})
	proposals[2].SigningRoot = bytesutil.ToBytes32([]byte{2})

	doubleProposals, err = beaconDB.CheckDoubleBlockProposals(ctx, proposals)
	require.NoError(t, err)
	require.Equal(t, len(proposals), len(doubleProposals))
	for i, existing := range doubleProposals {
		require.DeepNotEqual(t, doubleProposals[i].PrevBeaconBlockWrapper.SigningRoot, existing.BeaconBlockWrapper.SigningRoot)
	}
}

func Test_encodeDecodeProposalRecord(t *testing.T) {
	tests := []struct {
		name    string
		blkHdr  *slashertypes.SignedBlockHeaderWrapper
		wantErr bool
	}{
		{
			name:   "empty standard encode/decode",
			blkHdr: createProposalWrapper(0, 0, nil /* signingRoot */),
		},
		{
			name:   "standard encode/decode",
			blkHdr: createProposalWrapper(15, 6, []byte("1") /* signingRoot */),
		},
		{
			name: "failing encode/decode",
			blkHdr: &slashertypes.SignedBlockHeaderWrapper{
				SignedBeaconBlockHeader: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{},
				},
			},
			wantErr: true,
		},
		{
			name:    "failing empty encode/decode",
			blkHdr:  &slashertypes.SignedBlockHeaderWrapper{},
			wantErr: true,
		},
		{
			name:    "failing nil",
			blkHdr:  nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeProposalRecord(tt.blkHdr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("encodeProposalRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
			decoded, err := decodeProposalRecord(got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeProposalRecord() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(tt.blkHdr, decoded) {
				t.Errorf("Did not match got = %v, want %v", tt.blkHdr, decoded)
			}
		})
	}
}

func Test_encodeDecodeAttestationRecord(t *testing.T) {
	tests := []struct {
		name       string
		attWrapper *slashertypes.IndexedAttestationWrapper
		wantErr    bool
	}{
		{
			name:       "empty standard encode/decode",
			attWrapper: createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */),
		},
		{
			name:       "standard encode/decode",
			attWrapper: createAttestationWrapper(15, 6, []uint64{2, 4}, []byte("1") /* signingRoot */),
		},
		{
			name: "failing encode/decode",
			attWrapper: &slashertypes.IndexedAttestationWrapper{
				IndexedAttestation: &ethpb.IndexedAttestation{
					Data: &ethpb.AttestationData{},
				},
			},
			wantErr: true,
		},
		{
			name:       "failing empty encode/decode",
			attWrapper: &slashertypes.IndexedAttestationWrapper{},
			wantErr:    true,
		},
		{
			name:       "failing nil",
			attWrapper: nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeAttestationRecord(tt.attWrapper)
			if (err != nil) != tt.wantErr {
				t.Fatalf("encodeAttestationRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
			decoded, err := decodeAttestationRecord(got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeAttestationRecord() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(tt.attWrapper, decoded) {
				t.Errorf("Did not match got = %v, want %v", tt.attWrapper, decoded)
			}
		})
	}
}

func TestStore_PruneProposals(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name          string
		proposalsInDB []*slashertypes.SignedBlockHeaderWrapper
		afterPruning  []*slashertypes.SignedBlockHeaderWrapper
		epoch         types.Epoch
		wantErr       bool
	}{
		{
			name: "should delete all proposals under epoch 2",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(types.Slot(2), 0, []byte{1}),
				createProposalWrapper(types.Slot(8), 0, []byte{1}),
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			epoch: 5,
		},
		{
			name: "should delete all proposals under epoch 4",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*0, 0, []byte{1}),
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*1, 0, []byte{1}),
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*2, 0, []byte{1}),
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{},
			epoch:        4,
		},
		{
			name: "no proposal to delete under epoch 1",
			proposalsInDB: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*2, 0, []byte{1}),
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			afterPruning: []*slashertypes.SignedBlockHeaderWrapper{
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*2, 0, []byte{1}),
				createProposalWrapper(params.BeaconConfig().SlotsPerEpoch*3, 0, []byte{1}),
			},
			epoch: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconDB := setupDB(t)
			require.NoError(t, beaconDB.SaveBlockProposals(ctx, tt.proposalsInDB))
			if err := beaconDB.PruneProposals(ctx, tt.epoch, 0); (err != nil) != tt.wantErr {
				t.Errorf("PruneProposals() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Second time checking same proposals but all with different signing root should
			// return all double proposals.
			for i := 0; i < len(tt.afterPruning); i++ {
				prop := tt.afterPruning[i]
				tt.afterPruning[i].SigningRoot = createProposalWrapper(
					prop.SignedBeaconBlockHeader.Header.Slot,
					prop.SignedBeaconBlockHeader.Header.ProposerIndex,
					[]byte{2},
				).SigningRoot
			}

			doubleProposals, err := beaconDB.CheckDoubleBlockProposals(ctx, tt.afterPruning)
			require.NoError(t, err)
			require.Equal(t, len(tt.afterPruning), len(doubleProposals))
			for i, existing := range doubleProposals {
				require.DeepNotEqual(t, doubleProposals[i].PrevBeaconBlockWrapper.SigningRoot, existing.BeaconBlockWrapper.SigningRoot)
			}
		})
	}
}

func TestStore_PruneAttestations(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name             string
		attestationsInDB []*slashertypes.IndexedAttestationWrapper
		afterPruning     []*slashertypes.IndexedAttestationWrapper
		epoch            types.Epoch
		wantErr          bool
	}{
		{
			name: "should delete all attestations under epoch 2",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
			},
			epoch: 5000,
		},
		{
			name: "should delete all proposals under epoch 4",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 2, []uint64{1}, []byte{1}),
				createAttestationWrapper(0, 3, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{},
			epoch:        4,
		},
		{
			name: "no proposal to delete under epoch 1",
			attestationsInDB: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
			},
			afterPruning: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, []byte{1}),
			},
			epoch: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconDB := setupDB(t)
			require.NoError(t, beaconDB.SaveAttestationRecordsForValidators(ctx, tt.attestationsInDB))
			if err := beaconDB.PruneProposals(ctx, tt.epoch, 0); (err != nil) != tt.wantErr {
				t.Errorf("PruneProposals() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Second time checking same proposals but all with different signing root should
			// return all double proposals.
			for i := 0; i < len(tt.afterPruning); i++ {
				att := tt.afterPruning[i]
				tt.afterPruning[i].SigningRoot = createAttestationWrapper(
					att.IndexedAttestation.Data.Source.Epoch,
					att.IndexedAttestation.Data.Target.Epoch,
					att.IndexedAttestation.AttestingIndices,
					[]byte{2},
				).SigningRoot
			}

			doubleAtts, err := beaconDB.CheckAttesterDoubleVotes(ctx, tt.afterPruning)
			require.NoError(t, err)
			require.Equal(t, len(tt.afterPruning), len(doubleAtts))
			for i, existing := range doubleAtts {
				require.DeepNotEqual(t, doubleAtts[i].PrevAttestationWrapper.SigningRoot, existing.AttestationWrapper.SigningRoot)
			}
		})
	}
}

func createProposalWrapper(slot types.Slot, proposerIndex types.ValidatorIndex, signingRoot []byte) *slashertypes.SignedBlockHeaderWrapper {
	signRoot := bytesutil.ToBytes32(signingRoot)
	if signingRoot == nil {
		signRoot = params.BeaconConfig().ZeroHash
	}
	return &slashertypes.SignedBlockHeaderWrapper{
		SignedBeaconBlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:          slot,
				ProposerIndex: proposerIndex,
				ParentRoot:    params.BeaconConfig().ZeroHash[:],
				StateRoot:     params.BeaconConfig().ZeroHash[:],
				BodyRoot:      params.BeaconConfig().ZeroHash[:],
			},
			Signature: params.BeaconConfig().EmptySignature[:],
		},
		SigningRoot: signRoot,
	}
}

func createAttestationWrapper(source, target types.Epoch, indices []uint64, signingRoot []byte) *slashertypes.IndexedAttestationWrapper {
	signRoot := bytesutil.ToBytes32(signingRoot)
	if signingRoot == nil {
		signRoot = params.BeaconConfig().ZeroHash
	}
	data := &ethpb.AttestationData{
		BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
		Source: &ethpb.Checkpoint{
			Epoch: source,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: target,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
	}
	return &slashertypes.IndexedAttestationWrapper{
		IndexedAttestation: &ethpb.IndexedAttestation{
			AttestingIndices: indices,
			Data:             data,
			Signature:        params.BeaconConfig().EmptySignature[:],
		},
		SigningRoot: signRoot,
	}
}
