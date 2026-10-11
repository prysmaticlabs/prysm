package kv

import (
	"bytes"
	"fmt"
	"testing"

	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	light_client "github.com/OffchainLabs/prysm/v7/consensus-types/light-client"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/golang/snappy"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// Fork epochs for every test in this file: each fork starts 10 epochs after the previous one.
var backfillTestForkEpochs = map[int]primitives.Epoch{
	version.Altair:    0,
	version.Bellatrix: 10,
	version.Capella:   20,
	version.Deneb:     30,
	version.Electra:   40,
	version.Fulu:      50,
	version.Gloas:     60,
}

// The expected stored key per fork, written out so a wrong mapping in the code cannot hide.
var backfillTestForkKeys = map[int][]byte{
	version.Altair:    altairKey,
	version.Bellatrix: bellatrixKey,
	version.Capella:   capellaKey,
	version.Deneb:     denebKey,
	version.Electra:   ElectraKey,
	version.Fulu:      fuluKey,
	version.Gloas:     gloasKey,
}

var backfillTestForks = []int{
	version.Altair, version.Bellatrix, version.Capella, version.Deneb, version.Electra, version.Fulu, version.Gloas,
}

func setupBackfillForks(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.AltairForkEpoch = backfillTestForkEpochs[version.Altair]
	cfg.BellatrixForkEpoch = backfillTestForkEpochs[version.Bellatrix]
	cfg.CapellaForkEpoch = backfillTestForkEpochs[version.Capella]
	cfg.DenebForkEpoch = backfillTestForkEpochs[version.Deneb]
	cfg.ElectraForkEpoch = backfillTestForkEpochs[version.Electra]
	cfg.FuluForkEpoch = backfillTestForkEpochs[version.Fulu]
	cfg.GloasForkEpoch = backfillTestForkEpochs[version.Gloas]
	params.OverrideBeaconConfig(cfg)
}

// epochInsideFork returns an epoch whose whole window lies inside fork v.
func epochInsideFork(v int) primitives.Epoch {
	return backfillTestForkEpochs[v] + 5
}

func filledRoots(n int, seed byte) [][]byte {
	roots := make([][]byte, n)
	for i := range roots {
		roots[i] = bytesutil.PadTo([]byte{seed, byte(i + 1)}, 32)
	}
	return roots
}

func testBlockHeader(slot primitives.Slot) *pb.BeaconBlockHeader {
	return &pb.BeaconBlockHeader{
		Slot:          slot,
		ProposerIndex: 7,
		ParentRoot:    bytesutil.PadTo([]byte{1}, 32),
		StateRoot:     bytesutil.PadTo([]byte{2}, 32),
		BodyRoot:      bytesutil.PadTo([]byte{3}, 32),
	}
}

func testBlockData(seed byte) []*pb.LightClientBlockData {
	bd := make([]*pb.LightClientBlockData, fieldparams.SlotsPerEpoch)
	for i := range bd {
		bd[i] = &pb.LightClientBlockData{
			ProposerIndex:              primitives.ValidatorIndex(i),
			StateRoot:                  bytesutil.PadTo([]byte{seed, byte(i + 1)}, 32),
			SyncCommitteeBits:          make([]byte, fieldparams.SyncCommitteeLength/8),
			SyncCommitteeSignatureRoot: bytesutil.PadTo([]byte{seed, 9}, 32),
			SyncAggregateBranch:        filledRoots(4, seed),
		}
	}
	return bd
}

func testBlockDataGloas(seed byte) []*pb.LightClientBlockDataGloas {
	bd := make([]*pb.LightClientBlockDataGloas, fieldparams.SlotsPerEpoch)
	for i := range bd {
		bd[i] = &pb.LightClientBlockDataGloas{
			ProposerIndex:              primitives.ValidatorIndex(i),
			StateRoot:                  bytesutil.PadTo([]byte{seed, byte(i + 1)}, 32),
			SyncCommitteeBits:          make([]byte, fieldparams.SyncCommitteeLength/8),
			SyncCommitteeSignatureRoot: bytesutil.PadTo([]byte{seed, 9}, 32),
			SyncAggregateBranch:        filledRoots(8, seed),
		}
	}
	return bd
}

func testExecutionHeaderCapella() *enginev1.ExecutionPayloadHeaderCapella {
	return &enginev1.ExecutionPayloadHeaderCapella{
		ParentHash: make([]byte, 32), FeeRecipient: make([]byte, 20), StateRoot: make([]byte, 32),
		ReceiptsRoot: make([]byte, 32), LogsBloom: make([]byte, 256), PrevRandao: make([]byte, 32),
		BaseFeePerGas: make([]byte, 32), BlockHash: make([]byte, 32), TransactionsRoot: make([]byte, 32),
		WithdrawalsRoot: make([]byte, 32),
	}
}

func testExecutionHeaderDeneb() *enginev1.ExecutionPayloadHeaderDeneb {
	return &enginev1.ExecutionPayloadHeaderDeneb{
		ParentHash: make([]byte, 32), FeeRecipient: make([]byte, 20), StateRoot: make([]byte, 32),
		ReceiptsRoot: make([]byte, 32), LogsBloom: make([]byte, 256), PrevRandao: make([]byte, 32),
		BaseFeePerGas: make([]byte, 32), BlockHash: make([]byte, 32), TransactionsRoot: make([]byte, 32),
		WithdrawalsRoot: make([]byte, 32),
	}
}

// testEpochData builds a complete, valid record of fork v for epoch. seed makes
// records with the same epoch differ.
func testEpochData(t *testing.T, v int, epoch primitives.Epoch, seed byte) light_client.EpochData {
	start, err := slots.EpochStart(epoch)
	require.NoError(t, err)
	header := testBlockHeader(start - primitives.Slot(fieldparams.SlotsPerEpoch))
	finalizedRoot := bytesutil.PadTo([]byte{seed, 0xff}, 32)

	switch v {
	case version.Altair, version.Bellatrix:
		return &pb.LightClientEpochDataAltair{
			Epoch: epoch, ParentBlockHeader: header, BlockData: testBlockData(seed),
			BootstrapData:  &pb.LightClientBootstrapDataAltair{CurrentSyncCommitteeBranch: filledRoots(5, seed)},
			FinalizedRoot:  finalizedRoot,
			FinalityBranch: filledRoots(6, seed),
		}
	case version.Capella:
		return &pb.LightClientEpochDataCapella{
			Epoch: epoch, ParentBlockHeader: header, BlockData: testBlockData(seed),
			BootstrapData: &pb.LightClientBootstrapDataCapella{
				CurrentSyncCommitteeBranch: filledRoots(5, seed),
				Execution:                  testExecutionHeaderCapella(),
				ExecutionBranch:            filledRoots(4, seed),
			},
			FinalizedRoot:  finalizedRoot,
			FinalityBranch: filledRoots(6, seed),
		}
	case version.Deneb:
		return &pb.LightClientEpochDataDeneb{
			Epoch: epoch, ParentBlockHeader: header, BlockData: testBlockData(seed),
			BootstrapData: &pb.LightClientBootstrapDataDeneb{
				CurrentSyncCommitteeBranch: filledRoots(5, seed),
				Execution:                  testExecutionHeaderDeneb(),
				ExecutionBranch:            filledRoots(4, seed),
			},
			FinalizedRoot:  finalizedRoot,
			FinalityBranch: filledRoots(6, seed),
		}
	case version.Electra, version.Fulu:
		return &pb.LightClientEpochDataElectra{
			Epoch: epoch, ParentBlockHeader: header, BlockData: testBlockData(seed),
			BootstrapData: &pb.LightClientBootstrapDataElectra{
				CurrentSyncCommitteeBranch: filledRoots(6, seed),
				Execution:                  testExecutionHeaderDeneb(),
				ExecutionBranch:            filledRoots(4, seed),
			},
			FinalizedRoot:  finalizedRoot,
			FinalityBranch: filledRoots(7, seed),
		}
	case version.Gloas:
		return &pb.LightClientEpochDataGloas{
			Epoch: epoch, ParentBlockHeader: header, BlockData: testBlockDataGloas(seed),
			BootstrapData: &pb.LightClientBootstrapDataGloas{
				CurrentSyncCommitteeBranch: filledRoots(11, seed),
				ExecutionBlockHash:         bytesutil.PadTo([]byte{seed, 0xee}, 32),
				ExecutionBranch:            filledRoots(11, seed),
			},
			FinalizedRoot:  finalizedRoot,
			FinalityBranch: filledRoots(9, seed),
		}
	default:
		t.Fatalf("no fixture for fork %s", version.String(v))
		return nil
	}
}

func testSyncCommittee(seed byte) *pb.SyncCommittee {
	pubkeys := make([][]byte, fieldparams.SyncCommitteeLength)
	for i := range pubkeys {
		pubkeys[i] = bytesutil.PadTo([]byte{seed, byte(i)}, fieldparams.BLSPubkeyLength)
	}
	return &pb.SyncCommittee{Pubkeys: pubkeys, AggregatePubkey: bytesutil.PadTo([]byte{seed}, fieldparams.BLSPubkeyLength)}
}

func TestStore_LightClientEpochData_RoundTripEveryFork(t *testing.T) {
	setupBackfillForks(t)
	db := setupDB(t)
	ctx := t.Context()

	for _, v := range backfillTestForks {
		t.Run(version.String(v), func(t *testing.T) {
			epoch := epochInsideFork(v)
			want := testEpochData(t, v, epoch, 1)
			require.NoError(t, db.SaveLightClientEpochData(ctx, []light_client.EpochData{want}))

			got, err := db.LightClientEpochData(ctx, epoch)
			require.NoError(t, err)
			// Bellatrix and Fulu share a type with a neighbour; the exact fork must survive.
			require.Equal(t, fmt.Sprintf("%T", want), fmt.Sprintf("%T", got))
			gotVersion, err := light_client.EpochDataVersion(got)
			require.NoError(t, err)
			require.Equal(t, version.String(v), version.String(gotVersion))
			require.DeepSSZEqual(t, want, got)

			// The stored key must be the exact fork, not just a type that decodes.
			key := backfillTestForkKeys[v]
			require.NoError(t, db.db.View(func(tx *bolt.Tx) error {
				enc := tx.Bucket(lightClientEpochDataBucket).Get(bytesutil.Uint64ToBytesBigEndian(uint64(epoch)))
				raw, err := snappy.Decode(nil, enc)
				require.NoError(t, err)
				require.Equal(t, true, bytes.HasPrefix(raw, key), "stored key is not %s", key)
				return nil
			}))
		})
	}
}

func TestStore_LightClientEpochData_MissingIsNilNil(t *testing.T) {
	db := setupDB(t)
	got, err := db.LightClientEpochData(t.Context(), 42)
	require.NoError(t, err)
	require.IsNil(t, got)
}

func TestStore_SaveLightClientEpochData_InsertIfAbsent(t *testing.T) {
	setupBackfillForks(t)
	db := setupDB(t)
	ctx := t.Context()
	epoch := epochInsideFork(version.Deneb)
	original := testEpochData(t, version.Deneb, epoch, 1)
	require.NoError(t, db.SaveLightClientEpochData(ctx, []light_client.EpochData{original}))

	t.Run("same content again is a no-op", func(t *testing.T) {
		again := proto.Clone(original).(light_client.EpochData)
		require.NoError(t, db.SaveLightClientEpochData(ctx, []light_client.EpochData{again}))
	})

	t.Run("different content is a conflict and the original stays", func(t *testing.T) {
		changed := testEpochData(t, version.Deneb, epoch, 2)
		err := db.SaveLightClientEpochData(ctx, []light_client.EpochData{changed})
		require.ErrorIs(t, err, ErrLightClientEpochDataConflict)

		got, err := db.LightClientEpochData(ctx, epoch)
		require.NoError(t, err)
		require.DeepSSZEqual(t, original, got)
	})

	t.Run("one conflict rolls back the whole batch", func(t *testing.T) {
		batch := []light_client.EpochData{
			testEpochData(t, version.Deneb, epoch+1, 1),
			testEpochData(t, version.Deneb, epoch, 2), // conflicts
			testEpochData(t, version.Deneb, epoch+2, 1),
		}
		err := db.SaveLightClientEpochData(ctx, batch)
		require.ErrorIs(t, err, ErrLightClientEpochDataConflict)
		for _, e := range []primitives.Epoch{epoch + 1, epoch + 2} {
			got, err := db.LightClientEpochData(ctx, e)
			require.NoError(t, err)
			require.IsNil(t, got, "epoch %d must not be written", e)
		}
	})
}

func TestStore_SaveLightClientEpochData_Rejects(t *testing.T) {
	setupBackfillForks(t)
	db := setupDB(t)
	ctx := t.Context()
	epoch := epochInsideFork(version.Deneb)

	save := func(d light_client.EpochData) error {
		return db.SaveLightClientEpochData(ctx, []light_client.EpochData{d})
	}

	t.Run("nil record", func(t *testing.T) {
		require.ErrorContains(t, "nil light client epoch data", save(nil))
	})
	t.Run("typed nil record", func(t *testing.T) {
		require.ErrorContains(t, "nil light client epoch data", save((*pb.LightClientEpochDataDeneb)(nil)))
	})
	t.Run("committee inside the record", func(t *testing.T) {
		d := testEpochData(t, version.Deneb, epoch, 1).(*pb.LightClientEpochDataDeneb)
		d.BootstrapData.CurrentSyncCommittee = []*pb.SyncCommittee{testSyncCommittee(1)}
		require.ErrorContains(t, "current_sync_committee must be empty", save(d))
	})
	t.Run("wrong block_data length", func(t *testing.T) {
		d := testEpochData(t, version.Deneb, epoch, 1).(*pb.LightClientEpochDataDeneb)
		d.BlockData = d.BlockData[:4]
		require.ErrorContains(t, "block_data entries", save(d))
	})
	t.Run("nil block_data entry", func(t *testing.T) {
		d := testEpochData(t, version.Deneb, epoch, 1).(*pb.LightClientEpochDataDeneb)
		d.BlockData[3] = nil
		require.ErrorContains(t, "block_data[3]", save(d))
	})
	t.Run("type does not match the fork of the content", func(t *testing.T) {
		// A Gloas type whose window lies in Fulu would be decoded as the Electra type.
		d := testEpochData(t, version.Gloas, epochInsideFork(version.Fulu), 1)
		require.ErrorContains(t, "must be", save(d))
	})
	t.Run("a failed save does not change the caller's record", func(t *testing.T) {
		d := testEpochData(t, version.Deneb, epoch, 1).(*pb.LightClientEpochDataDeneb)
		d.ParentBlockHeader = nil // generated MarshalSSZ would fill this in place
		require.NotNil(t, save(d))
		require.IsNil(t, d.ParentBlockHeader, "ParentBlockHeader was filled in")
	})

	got, err := db.LightClientEpochData(ctx, epoch)
	require.NoError(t, err)
	require.IsNil(t, got, "a rejected record must not be written")
}

func TestStore_LightClientEpochData_KeyMustMatchContent(t *testing.T) {
	setupBackfillForks(t)
	db := setupDB(t)
	ctx := t.Context()
	epoch := epochInsideFork(version.Deneb)
	raw, err := encodeLightClientEpochData(testEpochData(t, version.Deneb, epoch, 1))
	require.NoError(t, err)

	// Store the record of `epoch` under the key of `epoch+1`.
	require.NoError(t, db.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(lightClientEpochDataBucket).Put(bytesutil.Uint64ToBytesBigEndian(uint64(epoch+1)), snappy.Encode(nil, raw))
	}))
	_, err = db.LightClientEpochData(ctx, epoch+1)
	require.ErrorContains(t, "stored under epoch", err)
}

func TestStore_LightClientPeriodSyncCommittee(t *testing.T) {
	db := setupDB(t)
	ctx := t.Context()

	t.Run("missing is nil, nil", func(t *testing.T) {
		got, err := db.LightClientPeriodSyncCommittee(ctx, 3)
		require.NoError(t, err)
		require.IsNil(t, got)
	})
	t.Run("nil committee is rejected", func(t *testing.T) {
		require.ErrorContains(t, "nil sync committee", db.SaveLightClientPeriodSyncCommittee(ctx, 3, nil))
	})

	original := testSyncCommittee(1)
	require.NoError(t, db.SaveLightClientPeriodSyncCommittee(ctx, 3, original))

	t.Run("round trip", func(t *testing.T) {
		got, err := db.LightClientPeriodSyncCommittee(ctx, 3)
		require.NoError(t, err)
		require.DeepSSZEqual(t, original, got)
	})
	t.Run("same committee again is a no-op", func(t *testing.T) {
		require.NoError(t, db.SaveLightClientPeriodSyncCommittee(ctx, 3, testSyncCommittee(1)))
	})
	t.Run("different committee is a conflict and the original stays", func(t *testing.T) {
		err := db.SaveLightClientPeriodSyncCommittee(ctx, 3, testSyncCommittee(2))
		require.ErrorIs(t, err, ErrLightClientPeriodSyncCommitteeConflict)
		got, err := db.LightClientPeriodSyncCommittee(ctx, 3)
		require.NoError(t, err)
		require.DeepSSZEqual(t, original, got)
	})
}

func TestStore_LightClientBackfill_SurvivesReopen(t *testing.T) {
	setupBackfillForks(t)
	dir := t.TempDir()
	ctx := t.Context()
	epoch := epochInsideFork(version.Electra)
	record := testEpochData(t, version.Electra, epoch, 1)
	committee := testSyncCommittee(1)

	db, err := NewKVStore(ctx, dir)
	require.NoError(t, err)
	require.NoError(t, db.SaveLightClientEpochData(ctx, []light_client.EpochData{record}))
	require.NoError(t, db.SaveLightClientPeriodSyncCommittee(ctx, 9, committee))
	require.NoError(t, db.Close())

	db, err = NewKVStore(ctx, dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	gotRecord, err := db.LightClientEpochData(ctx, epoch)
	require.NoError(t, err)
	require.DeepSSZEqual(t, record, gotRecord)
	gotCommittee, err := db.LightClientPeriodSyncCommittee(ctx, 9)
	require.NoError(t, err)
	require.DeepSSZEqual(t, committee, gotCommittee)
}
