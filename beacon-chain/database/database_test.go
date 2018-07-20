package database

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/prysmaticlabs/geth-sharding/shared"
	logTest "github.com/sirupsen/logrus/hooks/test"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
)

// Verifies that BeaconDB implements the sharding Service inteface.
var _ = shared.Service(&BeaconDB{})

var testDB *BeaconDB

func init() {
	tmp := fmt.Sprintf("%s/datadir", os.TempDir())
	config := &BeaconDBConfig{DataDir: tmp, Name: "beaconchaindata", InMemory: false}
	beaconDB, _ := NewBeaconDB(config)
	testDB = beaconDB
	testDB.Start()
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/lifecycledir", os.TempDir())
	config := &BeaconDBConfig{DataDir: tmp, Name: "beaconchaindata", InMemory: false}
	b, err := NewBeaconDB(config)
	if err != nil {
		t.Fatalf("could not initialize a new DB: %v", err)
	}

	b.Start()
	msg := hook.LastEntry().Message
	if msg != "Starting beaconDB service" {
		t.Errorf("incorrect log, expected %s, got %s", "Starting beaconDB service", msg)
	}

	b.Stop()
	msg = hook.LastEntry().Message
	if msg != "Stopping beaconDB service" {
		t.Errorf("incorrect log, expected %s, got %s", "Stopping beaconDB service", msg)
	}

	// Access DB after it's stopped, this should fail.
	_, err = b.db.Get([]byte("ralph merkle"))

	if err.Error() != "leveldb: closed" {
		t.Fatalf("beaconDB close function did not work")
	}
}

// Testing the concurrency with multiple processes attempting to write.
func Test_DBConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(val string) {
			defer wg.Done()
			if err := testDB.db.Put([]byte("ralph merkle"), []byte(val)); err != nil {
				t.Errorf("could not save value in db: %v", err)
			}
		}(strconv.Itoa(i))
	}
	wg.Wait()
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
	if err == nil || err.Error() != leveldberrors.ErrNotFound.Error() {
		t.Errorf("Expected error %v but got %v", leveldberrors.ErrNotFound, err)
	}
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
