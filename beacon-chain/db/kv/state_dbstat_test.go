package kv

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dustin/go-humanize"
	types "github.com/prysmaticlabs/eth2-types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/status-im/keycard-go/hexutils"
)

const (
	dbDirEnvVar = "PRYSM_DB_DIR"
	dbFileName  = "beaconchain.db"
	rowLimit    = uint64(1000) // should be okay for a normal mainnet DB.
)

type RegistryValidatorEntry struct {
	StateSlot         types.Slot
	ValidatorIdx      int
	ValidatorEntryKey string
}

type ValidatorEntry struct {
	WithdrawalCredentials      string
	EffectiveBalance           uint64
	Slashed                    bool
	ActivationEligibilityEpoch types.Epoch
	ActivationEpoch            types.Epoch
	ExitEpoch                  types.Epoch
	WithdrawableEpoch          types.Epoch
}

func TestStore_DisplayStates(t *testing.T) {
	dbDirectory := os.Getenv(dbDirEnvVar)

	// check if the supplied db file name exists, otherwise silently return.
	dbFileWithPath := filepath.Join(dbDirectory, dbFileName)
	if _, err := os.Stat(dbFileWithPath); os.IsNotExist(err) {
		return
	}

	// get all the buckets in the DB
	buckets := keysInBucket(t, dbFileWithPath)
	if _, ok := buckets[string(stateBucket)]; !ok {
		log.Fatal("state bucket not in database")
	}

	// get all the keys of the "state" bucket.
	keys := keysOfBucket(t, dbFileWithPath, string(stateBucket), rowLimit)

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
		fmt.Println("key                           :", key)
		fmt.Println("value                         : compressed size = ", humanize.Bytes(uint64(valueLen)))
		fmt.Println("genesis_time                  :", st.GenesisTime())
		fmt.Println("genesis_validators_root       :", hexutils.BytesToHex(st.GenesisValidatorRoot()))
		fmt.Println("slot                          :", st.Slot())
		fmt.Println("fork                          : previous_version: ", st.Fork().PreviousVersion, ",  current_version: ", st.Fork().CurrentVersion)
		fmt.Println("latest_block_header           : sizeSSZ = ", humanize.Bytes(uint64(st.LatestBlockHeader().SizeSSZ())))
		size, count := sizeAndCountOfByteList(st.BlockRoots())
		fmt.Println("block_roots                   : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.StateRoots())
		fmt.Println("state_roots                   : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.HistoricalRoots())
		fmt.Println("historical_roots              : size =  ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("eth1_data                     : sizeSSZ =  ", humanize.Bytes(uint64(st.Eth1Data().SizeSSZ())))
		size, count = sizeAndCountGeneric(st.Eth1DataVotes(), nil)
		fmt.Println("eth1_data_votes               : sizeSSZ = ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("eth1_deposit_index            :", st.Eth1DepositIndex())
		size, count = sizeAndCountGeneric(st.Validators(), nil)
		fmt.Println("validators                    : sizeSSZ = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfUin64List(st.Balances())
		fmt.Println("balances                      : size = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.RandaoMixes())
		fmt.Println("randao_mixes                  : size = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfUin64List(st.Slashings())
		fmt.Println("slashings                     : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountGeneric(st.PreviousEpochAttestations())
		fmt.Println("previous_epoch_attestations   : sizeSSZ ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountGeneric(st.CurrentEpochAttestations())
		fmt.Println("current_epoch_attestations    : sizeSSZ =  ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("justification_bits            : size =  ", humanize.Bytes(st.JustificationBits().Len()), ", count =  ", st.JustificationBits().Count())
		fmt.Println("previous_justified_checkpoint : sizeSSZ =  ", humanize.Bytes(uint64(st.PreviousJustifiedCheckpoint().SizeSSZ())))
		fmt.Println("current_justified_checkpoint  : sizeSSZ =  ", humanize.Bytes(uint64(st.CurrentJustifiedCheckpoint().SizeSSZ())))
		fmt.Println("finalized_checkpoint          : sizeSSZ =  ", humanize.Bytes(uint64(st.FinalizedCheckpoint().SizeSSZ())))
		rowCount++
	}
}

func TestStore_DisplayValidators(t *testing.T) {
	dbDirectory := os.Getenv(dbDirEnvVar)

	// check if the supplied db file name exists, otherwise silently return.
	dbFileWithPath := filepath.Join(dbDirectory, dbFileName)
	if _, err := os.Stat(dbFileWithPath); os.IsNotExist(err) {
		return
	}

	keys := keysOfBucket(t, dbFileWithPath, string(stateBucket), rowLimit)

	// create a new KV Store.
	db, err := NewKVStore(context.Background(), dbDirectory, &Config{})
	require.NoError(t, err, "Failed to instantiate DB")
	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	rowCount := uint64(0)

	registryValidatorMap := make(map[string][]*RegistryValidatorEntry)
	validatorMap := make(map[string]*ValidatorEntry)

	for key := range keys {
		st, stateErr := db.State(ctx, bytesutil.ToBytes32(hexutils.HexToBytes(key)))
		require.NoError(t, stateErr)

		if readValErr := st.ReadFromEveryValidator(func(idx int, val iface.ReadOnlyValidator) error {
			publicKey := fmt.Sprintf("%#x", val.PublicKey())

			// construct the validator entry.
			validatorEntry := &ValidatorEntry{
				WithdrawalCredentials:      hexutils.BytesToHex(val.WithdrawalCredentials()),
				EffectiveBalance:           val.EffectiveBalance(),
				Slashed:                    val.Slashed(),
				ActivationEligibilityEpoch: val.ActivationEligibilityEpoch(),
				ActivationEpoch:            val.ActivationEpoch(),
				ExitEpoch:                  val.ExitEpoch(),
				WithdrawableEpoch:          val.WithdrawableEpoch(),
			}

			// create the hash of the entry to check it is a new entry.
			// if the hash is not present in validator map, then add it.
			valEntryKey := validatorEntryKey(validatorEntry)
			if _, ok := validatorMap[valEntryKey]; !ok {
				validatorMap[valEntryKey] = validatorEntry
			}

			// create a registry entry.
			registryEntry := &RegistryValidatorEntry{
				StateSlot:         st.Slot(),
				ValidatorIdx:      idx,
				ValidatorEntryKey: valEntryKey,
			}

			// add the registryEntry entry to its list
			regValList := registryValidatorMap[publicKey]
			regValList = append(regValList, registryEntry)
			registryValidatorMap[publicKey] = regValList

			return nil
		}); readValErr != nil {
			require.NoError(t, readValErr)
		}
		rowCount++
	}

	for pubKey, valList := range registryValidatorMap {
		valKeys := make(map[string]string)
		for _, val := range valList {
			valKeys[val.ValidatorEntryKey] = ""
		}
		fmt.Println(pubKey, ",", len(valList), ",", len(valKeys))
	}
}

func keysInBucket(t *testing.T, absoluteDBFileName string) map[string]*bolt.Bucket {
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

func keysOfBucket(t *testing.T, absoluteDBFileName, bucketName string, limit uint64) map[string]int {
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
	for i := 0; i < len(list); i++ {
		size += uint64(8)
		count += 1
	}
	return size, count
}

func sizeAndCountGeneric(genericItems interface{}, err error) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	if err != nil {
		return size, count
	}

	switch items := genericItems.(type) {
	case []*ethpb.Eth1Data:
		for _, item := range items {
			size += uint64(item.SizeSSZ())
		}
		count = uint64(len(items))
	case []*ethpb.Validator:
		for _, item := range items {
			size += uint64(item.SizeSSZ())
		}
		count = uint64(len(items))
	case []*pbp2p.PendingAttestation:
		for _, item := range items {
			size += uint64(item.SizeSSZ())
		}
		count = uint64(len(items))
	default:
		return 0, 0
	}

	return size, count
}

func validatorEntryKey(valEntry *ValidatorEntry) string {
	var allBytes []byte

	allBytes = append(allBytes, []byte(valEntry.WithdrawalCredentials)...)

	effBal := make([]byte, 8)
	binary.LittleEndian.PutUint64(effBal, valEntry.EffectiveBalance)
	allBytes = append(allBytes, effBal...)

	slashed := uint8(0)
	if valEntry.Slashed {
		slashed = uint8(1)
	}
	allBytes = append(allBytes, slashed)

	aeb := make([]byte, 8)
	binary.LittleEndian.PutUint64(aeb, uint64(valEntry.ActivationEligibilityEpoch))
	allBytes = append(allBytes, aeb...)

	ae := make([]byte, 8)
	binary.LittleEndian.PutUint64(ae, uint64(valEntry.ActivationEpoch))
	allBytes = append(allBytes, ae...)

	ee := make([]byte, 8)
	binary.LittleEndian.PutUint64(ee, uint64(valEntry.ExitEpoch))
	allBytes = append(allBytes, ee...)

	we := make([]byte, 8)
	binary.LittleEndian.PutUint64(we, uint64(valEntry.WithdrawableEpoch))
	allBytes = append(allBytes, we...)

	hash := hashutil.Hash(allBytes)
	return hexutils.BytesToHex(hash[:])
}
