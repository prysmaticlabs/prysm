package database

import (
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
)

// Verifies that ShardDB implements the sharding Service inteface.
var _ = sharding.Service(&ShardDB{})

var testDB *ShardDB

func init() {
	shardDB, _ := NewShardDB("/tmp/datadir", "shardchaindata", false)
	testDB = shardDB
	testDB.Start()
}

func TestLifecycle(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	s, err := NewShardDB("/tmp/datadir", "shardchaindb", false)
	if err != nil {
		t.Fatalf("Could not initialize a new sb: %v", err)
	}

	s.Start()
	h.VerifyLogMsg("Starting shardDB service")
	// ethdb.NewLDBDatabase logs the following
	h.VerifyLogMsg("Allocated cache and file handles")

	s.Stop()
	h.VerifyLogMsg("Stopping shardDB service")

	// Access DB after it's stopped, this should fail
	_, err = s.db.Get([]byte("ralph merkle"))

	if err.Error() != "leveldb: closed" {
		t.Fatalf("shardDB close function did not work")
	}
}

// Testing the concurrency of the shardDB with multiple processes attempting to write.
func Test_DBConcurrent(t *testing.T) {
	for i := 0; i < 100; i++ {
		go func(val string) {
			if err := testDB.db.Put([]byte("ralph merkle"), []byte(val)); err != nil {
				t.Errorf("could not save value in db: %v", err)
			}
		}(strconv.Itoa(i))
	}
}

func Test_DBPut(t *testing.T) {
	if err := testDB.db.Put([]byte("ralph merkle"), []byte{1, 2, 3}); err != nil {
		t.Errorf("could not save value in db: %v", err)
	}
}

func Test_DBHas(t *testing.T) {
	key := []byte("ralph merkle")

	if err := testDB.db.Put(key, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in db: %v", err)
	}

	has, err := testDB.db.Has(key)
	if err != nil {
		t.Errorf("could not check if db has key: %v", err)
	}
	if !has {
		t.Errorf("db should have key: %v", key)
	}

	key2 := []byte{}
	has2, err := testDB.db.Has(key2)
	if err != nil {
		t.Errorf("could not check if db has key: %v", err)
	}
	if has2 {
		t.Errorf("db should not have non-existent key: %v", key2)
	}
}

func Test_DBGet(t *testing.T) {
	key := []byte("ralph merkle")

	if err := testDB.db.Put(key, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in db: %v", err)
	}

	val, err := testDB.db.Get(key)
	if err != nil {
		t.Errorf("get failed: %v", err)
	}
	if len(val) == 0 {
		t.Errorf("no value stored for key")
	}

	key2 := []byte{}
	val2, err := testDB.db.Get(key2)
	if len(val2) != 0 {
		t.Errorf("non-existent key should not have a value. key=%v, value=%v", key2, val2)
	}
}

func Test_DBDelete(t *testing.T) {
	key := []byte("ralph merkle")

	if err := testDB.db.Put(key, []byte{1, 2, 3}); err != nil {
		t.Fatalf("could not save value in db: %v", err)
	}

	if err := testDB.db.Delete(key); err != nil {
		t.Errorf("could not delete key: %v", key)
	}
}
