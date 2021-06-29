package kv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dustin/go-humanize"
	types "github.com/prysmaticlabs/eth2-types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/status-im/keycard-go/hexutils"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)

	r := [32]byte{'A'}

	require.Equal(t, false, db.HasState(context.Background(), r))

	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(100))

	require.NoError(t, db.SaveState(context.Background(), st, r))
	assert.Equal(t, true, db.HasState(context.Background(), r))

	savedS, err := db.State(context.Background(), r)
	require.NoError(t, err)

	if !reflect.DeepEqual(st.InnerStateUnsafe(), savedS.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(st.InnerStateUnsafe(), savedS.InnerStateUnsafe())
		t.Errorf("Did not retrieve saved state: %v", diff)
	}

	savedS, err = db.State(context.Background(), [32]byte{'B'})
	require.NoError(t, err)
	assert.Equal(t, iface.ReadOnlyBeaconState(nil), savedS, "Unsaved state should've been nil")
}

func TestGenesisState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)

	headRoot := [32]byte{'B'}

	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), headRoot))
	require.NoError(t, db.SaveState(context.Background(), st, headRoot))

	savedGenesisS, err := db.GenesisState(context.Background())
	require.NoError(t, err)
	assert.DeepSSZEqual(t, st.InnerStateUnsafe(), savedGenesisS.InnerStateUnsafe(), "Did not retrieve saved state")
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), [32]byte{'C'}))
}

func TestStore_StatesBatchDelete(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	numBlocks := 100
	totalBlocks := make([]interfaces.SignedBeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	evenBlockRoots := make([][32]byte, 0)
	for i := 0; i < len(totalBlocks); i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = types.Slot(i)
		totalBlocks[i] = interfaces.WrappedPhase0SignedBeaconBlock(b)
		r, err := totalBlocks[i].Block().HashTreeRoot()
		require.NoError(t, err)
		st, err := testutil.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(types.Slot(i)))
		require.NoError(t, db.SaveState(context.Background(), st, r))
		blockRoots = append(blockRoots, r)
		if i%2 == 0 {
			evenBlockRoots = append(evenBlockRoots, r)
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	// We delete all even indexed states.
	require.NoError(t, db.DeleteStates(ctx, evenBlockRoots))
	// When we retrieve the data, only the odd indexed state should remain.
	for _, r := range blockRoots {
		s, err := db.State(context.Background(), r)
		require.NoError(t, err)
		if s == nil {
			continue
		}
		assert.Equal(t, types.Slot(1), s.Slot()%2, "State with slot %d should have been deleted", s.Slot())
	}
}

func TestStore_DeleteGenesisState(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisBlockRoot := [32]byte{'A'}
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))
	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(100))
	require.NoError(t, db.SaveState(ctx, st, genesisBlockRoot))
	wantedErr := "cannot delete genesis, finalized, or head state"
	assert.ErrorContains(t, wantedErr, db.DeleteState(ctx, genesisBlockRoot))
}

func TestStore_DeleteFinalizedState(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))

	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = genesis[:]
	blk.Block.Slot = 100

	require.NoError(t, db.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(blk)))

	finalizedBlockRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	finalizedState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, finalizedState.SetSlot(100))
	require.NoError(t, db.SaveState(ctx, finalizedState, finalizedBlockRoot))
	finalizedCheckpoint := &ethpb.Checkpoint{Root: finalizedBlockRoot[:]}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, finalizedCheckpoint))
	wantedErr := "cannot delete genesis, finalized, or head state"
	assert.ErrorContains(t, wantedErr, db.DeleteState(ctx, finalizedBlockRoot))
}

func TestStore_DeleteHeadState(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))

	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = genesis[:]
	blk.Block.Slot = 100
	require.NoError(t, db.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(blk)))

	headBlockRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(100))
	require.NoError(t, db.SaveState(ctx, st, headBlockRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headBlockRoot))
	wantedErr := "cannot delete genesis, finalized, or head state"
	assert.ErrorContains(t, wantedErr, db.DeleteState(ctx, headBlockRoot))
}

func TestStore_SaveDeleteState_CanGetHighestBelow(t *testing.T) {
	db := setupDB(t)

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 1
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))
	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(1))
	s0 := st.InnerStateUnsafe()
	require.NoError(t, db.SaveState(context.Background(), st, r))

	b.Block.Slot = 100
	r1, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))
	st, err = testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(100))
	s1 := st.InnerStateUnsafe()
	require.NoError(t, db.SaveState(context.Background(), st, r1))

	b.Block.Slot = 1000
	r2, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))
	st, err = testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(1000))
	s2 := st.InnerStateUnsafe()

	require.NoError(t, db.SaveState(context.Background(), st, r2))

	highest, err := db.HighestSlotStatesBelow(context.Background(), 2)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, highest[0].InnerStateUnsafe(), s0)

	highest, err = db.HighestSlotStatesBelow(context.Background(), 101)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, highest[0].InnerStateUnsafe(), s1)

	highest, err = db.HighestSlotStatesBelow(context.Background(), 1001)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, highest[0].InnerStateUnsafe(), s2)
}

func TestStore_GenesisState_CanGetHighestBelow(t *testing.T) {
	db := setupDB(t)

	genesisState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	genesisRoot := [32]byte{'a'}
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), genesisRoot))
	require.NoError(t, db.SaveState(context.Background(), genesisState, genesisRoot))

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 1
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))

	st, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, db.SaveState(context.Background(), st, r))

	highest, err := db.HighestSlotStatesBelow(context.Background(), 2)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, highest[0].InnerStateUnsafe(), st.InnerStateUnsafe())

	highest, err = db.HighestSlotStatesBelow(context.Background(), 1)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, highest[0].InnerStateUnsafe(), genesisState.InnerStateUnsafe())
	highest, err = db.HighestSlotStatesBelow(context.Background(), 0)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, highest[0].InnerStateUnsafe(), genesisState.InnerStateUnsafe())
}

func TestStore_CleanUpDirtyStates_AboveThreshold(t *testing.T) {
	db := setupDB(t)

	genesisState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	genesisRoot := [32]byte{'a'}
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), genesisRoot))
	require.NoError(t, db.SaveState(context.Background(), genesisState, genesisRoot))

	bRoots := make([][32]byte, 0)
	slotsPerArchivedPoint := types.Slot(128)
	prevRoot := genesisRoot
	for i := types.Slot(1); i <= slotsPerArchivedPoint; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = prevRoot[:]
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))
		bRoots = append(bRoots, r)
		prevRoot = r

		st, err := testutil.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(i))
		require.NoError(t, db.SaveState(context.Background(), st, r))
	}

	require.NoError(t, db.SaveFinalizedCheckpoint(context.Background(), &ethpb.Checkpoint{
		Root:  bRoots[len(bRoots)-1][:],
		Epoch: types.Epoch(slotsPerArchivedPoint / params.BeaconConfig().SlotsPerEpoch),
	}))
	require.NoError(t, db.CleanUpDirtyStates(context.Background(), slotsPerArchivedPoint))

	for i, root := range bRoots {
		if types.Slot(i) >= slotsPerArchivedPoint.SubSlot(slotsPerArchivedPoint.Div(3)) {
			require.Equal(t, true, db.HasState(context.Background(), root))
		} else {
			require.Equal(t, false, db.HasState(context.Background(), root))
		}
	}
}

func TestStore_CleanUpDirtyStates_Finalized(t *testing.T) {
	db := setupDB(t)

	genesisState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	genesisRoot := [32]byte{'a'}
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), genesisRoot))
	require.NoError(t, db.SaveState(context.Background(), genesisState, genesisRoot))

	for i := types.Slot(1); i <= params.BeaconConfig().SlotsPerEpoch; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))

		st, err := testutil.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(i))
		require.NoError(t, db.SaveState(context.Background(), st, r))
	}

	require.NoError(t, db.SaveFinalizedCheckpoint(context.Background(), &ethpb.Checkpoint{Root: genesisRoot[:]}))
	require.NoError(t, db.CleanUpDirtyStates(context.Background(), params.BeaconConfig().SlotsPerEpoch))
	require.Equal(t, true, db.HasState(context.Background(), genesisRoot))
}

func TestStore_CleanUpDirtyStates_DontDeleteNonFinalized(t *testing.T) {
	db := setupDB(t)

	genesisState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	genesisRoot := [32]byte{'a'}
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), genesisRoot))
	require.NoError(t, db.SaveState(context.Background(), genesisState, genesisRoot))

	var unfinalizedRoots [][32]byte
	for i := types.Slot(1); i <= params.BeaconConfig().SlotsPerEpoch; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(b)))
		unfinalizedRoots = append(unfinalizedRoots, r)

		st, err := testutil.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(i))
		require.NoError(t, db.SaveState(context.Background(), st, r))
	}

	require.NoError(t, db.SaveFinalizedCheckpoint(context.Background(), &ethpb.Checkpoint{Root: genesisRoot[:]}))
	require.NoError(t, db.CleanUpDirtyStates(context.Background(), params.BeaconConfig().SlotsPerEpoch))

	for _, rt := range unfinalizedRoots {
		require.Equal(t, true, db.HasState(context.Background(), rt))
	}
}



func TestStore_DisplayStates(t *testing.T) {
	dbDirectory := "/Users/jmozah/Desktop"
	dbFileName := "beaconchain.db"
	rowLimit := uint64(160)

	// get all the buckets in the DB
	dbFileWithPath := filepath.Join(dbDirectory, dbFileName)
	buckets := getKeysInBucket(t, dbFileWithPath)
	if _, ok := buckets[string(stateBucket)]; !ok {
		log.Fatal("state bucket not in database")
	}

	// get all the keys of the "state" bucket.
	keys := getKeysOfBucket(t, dbFileWithPath, string(stateBucket), rowLimit)

	// create a new KV Store.
	db, err := NewKVStore(context.Background(), dbDirectory, &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	rowCount := uint64(0)
	for key, valueLen := range keys {
		st, stateErr := db.State(ctx, bytesutil.ToBytes32(hexutils.HexToBytes(key)))
		require.NoError(t, stateErr)
		rowStr := fmt.Sprintf("---- row = %04d ----", rowCount)
		fmt.Println(rowStr)
		fmt.Println("key                           :",key)
		fmt.Println("value                         : compressed size = ", humanize.Bytes(uint64(valueLen)))
		fmt.Println("genesis_time                  :",st.GenesisTime())
		fmt.Println("genesis_validators_root       :",hexutils.BytesToHex(st.GenesisValidatorRoot()))
		fmt.Println("slot                          :",st.Slot())
		fmt.Println("fork                          : previous_version: ", st.Fork().PreviousVersion , ",  current_version: ",  st.Fork().CurrentVersion)
		fmt.Println("latest_block_header           : sizeSSZ = ", humanize.Bytes(uint64(st.LatestBlockHeader().SizeSSZ())))
		size, count := sizeAndCountOfByteList(st.BlockRoots())
		fmt.Println("block_roots                   : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.StateRoots())
		fmt.Println("state_roots                   : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.HistoricalRoots())
		fmt.Println("historical_roots              : size =  ",  humanize.Bytes(size), ", count =  ", count)
		fmt.Println("eth1_data                     : sizeSSZ =  ", humanize.Bytes(uint64(st.Eth1Data().SizeSSZ())))
		size, count = sizeAndCountOfEth1DataVotes(st.Eth1DataVotes())
		fmt.Println("eth1_data_votes               : sizeSSZ = ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("eth1_deposit_index            :",st.Eth1DepositIndex())
		size, count = sizeAndCountOfValidators(st.Validators())
		fmt.Println("validators                    : sizeSSZ = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfUin64List(st.Balances())
		fmt.Println("balances                      : size = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.RandaoMixes())
		fmt.Println("randao_mixes                  : size = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfUin64List(st.Slashings())
		fmt.Println("slashings                     : size =  ",  humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfPendingAttestations(st.PreviousEpochAttestations())
		fmt.Println("previous_epoch_attestations   : sizeSSZ ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfPendingAttestations(st.CurrentEpochAttestations())
		fmt.Println("current_epoch_attestations    : sizeSSZ =  ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("justification_bits            : size =  ", humanize.Bytes(st.JustificationBits().Len()), ", count =  ", st.JustificationBits().Count())
		fmt.Println("previous_justified_checkpoint : sizeSSZ =  ", humanize.Bytes(uint64(st.PreviousJustifiedCheckpoint().SizeSSZ())))
		fmt.Println("current_justified_checkpoint  : sizeSSZ =  ",  humanize.Bytes(uint64(st.CurrentJustifiedCheckpoint().SizeSSZ())))
		fmt.Println("finalized_checkpoint          : sizeSSZ =  ", humanize.Bytes(uint64(st.FinalizedCheckpoint().SizeSSZ())))
		rowCount++
	}
}

func TestStore_DisplayValidators(t *testing.T) {

}


func getKeysInBucket(t *testing.T, absoluteDBFileName string) map[string]*bolt.Bucket {
	// check if the supplied db file name exists.
	if _, err := os.Stat(absoluteDBFileName); os.IsNotExist(err) {
		require.NoError(t, err)
	}

	// open the bolt db file. If we could not open the file in 5 seconds, the probably
	// another process has opend it already.
	boltDB, err := bolt.Open(absoluteDBFileName, 0600, &bolt.Options{Timeout: 5 * time.Second})
	require.NoError(t, err)

	// close the DB at the end.
	defer func() {
		err := boltDB.Close()
		require.NoError(t, err)
	}()

	// find all the bucket names and bucket structures.
	buckets := make(map[string]*bolt.Bucket)
	if err = boltDB.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, buc *bolt.Bucket) error {
			buckets[string(name)] = buc
			return nil
		})
	}); err != nil {
		require.NoError(t, err)
	}

	return buckets
}


func getKeysOfBucket(t *testing.T, absoluteDBFileName, bucketName string, limit uint64) map[string]int {
	// check if the supplied db file name exists.
	if _, err := os.Stat(absoluteDBFileName); os.IsNotExist(err) {
		require.NoError(t, err)
	}

	// open the bolt db file. If we could not open the file in 5 seconds, the probably
	// another process has opend it already.
	boltDB, err := bolt.Open(absoluteDBFileName, 0600, &bolt.Options{Timeout: 5 * time.Second})
	require.NoError(t, err)

	// close the DB at the end.
	defer func() {
		err := boltDB.Close()
		require.NoError(t, err)
	}()

	keys := make(map[string]int)
	if err := boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		c := b.Cursor()
		count := uint64(0)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if count >= limit {
				return nil
			}
			keys[hexutils.BytesToHex(k)] = len(v)
			count++
		}
		return nil
	}); err != nil {
		require.NoError(t, err)
	}

	return keys
}

func sizeAndCountOfByteList(list [][]byte) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for _, root := range list {
		size += uint64(len(root))
		count += 1
	}
	return size, count
}

func sizeAndCountOfUin64List(list []uint64) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for i:=0; i < len(list); i++ {
		size += uint64(8)
		count += 1
	}
	return size, count
}

func sizeAndCountOfEth1DataVotes(votes []*ethpb.Eth1Data) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for _, vote := range votes {
		size += uint64(vote.SizeSSZ())
		count += 1
	}
	return size, count
}

func sizeAndCountOfValidators(validators []*ethpb.Validator) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for _, validator := range validators {
		size += uint64(validator.SizeSSZ())
		count += 1
	}
	return size, count
}

func sizeAndCountOfPendingAttestations(attestations []*pbp2p.PendingAttestation, err error ) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	if err != nil {
		return size, count
	}
	for _, attestation := range attestations {
		size += uint64(attestation.SizeSSZ())
		count += 1
	}
	return size, count
}

func sizeAndCountOfCheckpoints(validators []*ethpb.Validator) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for _, validator := range validators {
		size += uint64(validator.SizeSSZ())
		count += 1
	}
	return size, count
}