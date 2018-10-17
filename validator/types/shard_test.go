package types

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	sharedDB "github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

type mockShardDB struct {
	kv map[common.Hash][]byte
}

func (m *mockShardDB) Get(k []byte) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockShardDB) Has(k []byte) (bool, error) {
	return false, nil
}

func (m *mockShardDB) Put(k []byte, v []byte) error {
	return fmt.Errorf("error updating db")
}

func (m *mockShardDB) Delete(k []byte) error {
	return fmt.Errorf("error deleting value in db")
}

func (m *mockShardDB) Close() {
}

func (m *mockShardDB) NewBatch() ethdb.Batch {
	return nil
}

// Hash returns the hash of a collation's entire contents. Useful for comparison tests.
func (c *Collation) Hash() common.Hash {
	encoded, err := rlp.EncodeToBytes(c)
	if err != nil {
		log.Errorf("Failed to RLP encode data: %v", err)
	}
	return hashutil.Hash(encoded)
}

func TestShard_ValidateShardID(t *testing.T) {
	emptyHash := common.BytesToHash([]byte{})
	emptyAddr := common.BytesToAddress([]byte{})
	header := NewCollationHeader(big.NewInt(1), &emptyHash, big.NewInt(1), &emptyAddr, [32]byte{})
	shardDB := sharedDB.NewKVStore()
	shard := NewShard(big.NewInt(3), shardDB)

	if err := shard.ValidateShardID(header); err == nil {
		t.Errorf("ShardID validation incorrect. Function should throw error when ShardID's do not match. want=%d. got=%d", header.ShardID().Int64(), shard.ShardID().Int64())
	}

	header2 := NewCollationHeader(big.NewInt(100), &emptyHash, big.NewInt(1), &emptyAddr, [32]byte{})
	shard2 := NewShard(big.NewInt(100), shardDB)

	if err := shard2.ValidateShardID(header2); err != nil {
		t.Errorf("ShardID validation incorrect. Function should not throw error when ShardID's match. want=%d. got=%d", header2.ShardID().Int64(), shard2.ShardID().Int64())
	}
}

func TestShard_HeaderByHash(t *testing.T) {
	emptyHash := common.BytesToHash([]byte{})
	emptyAddr := common.BytesToAddress([]byte{})
	header := NewCollationHeader(big.NewInt(1), &emptyHash, big.NewInt(1), &emptyAddr, [32]byte{})

	// creates a mockDB that always returns nil values from .Get and errors in every other method.
	mockDB := &mockShardDB{kv: make(map[common.Hash][]byte)}

	// creates a well-functioning shardDB.
	shardDB := sharedDB.NewKVStore()

	// creates a shard with a functioning DB and another one with a faulty DB.
	shard := NewShard(big.NewInt(1), shardDB)
	errorShard := NewShard(big.NewInt(1), mockDB)

	if err := shard.SaveHeader(header); err != nil {
		t.Fatalf("cannot save collation header: %v", err)
	}
	hash := header.Hash()

	dbHeader, err := shard.HeaderByHash(&hash)
	if err != nil {
		t.Fatalf("could not fetch collation header by hash: %v", err)
	}

	// checks for errors if shardDB value decoding fails (in the case of the
	// shard with the faulty DB).
	if _, err := errorShard.HeaderByHash(&hash); err == nil {
		t.Errorf("if a faulty db is used, RLP decoding should fail")
	}

	// Compare the hashes.
	if header.Hash() != dbHeader.Hash() {
		t.Errorf("headers do not match. want=%v. got=%v", header, dbHeader)
	}
}

func TestShard_CollationByHeaderHash(t *testing.T) {
	emptyAddr := common.BytesToAddress([]byte{})

	// Empty chunk root.
	header := NewCollationHeader(big.NewInt(1), nil, big.NewInt(1), &emptyAddr, [32]byte{})

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	shardDB := sharedDB.NewKVStore()
	shard := NewShard(big.NewInt(1), shardDB)

	// should throw error if saving the collation before setting the chunk root
	// in header.
	if err := shard.SaveCollation(collation); err == nil {
		t.Errorf("should not be able to save collation before setting header chunk root")
	}

	// we set the chunk root.
	collation.CalculateChunkRoot()

	hash := collation.Header().Hash()

	// calculate a new hash now that chunk root is set.
	newHash := collation.Header().Hash()

	// should not be able to fetch collation without saving first.
	if _, err := shard.CollationByHeaderHash(&newHash); err == nil {
		t.Errorf("should not be able to fetch collation before saving first")
	}

	// should not be able to fetch collation if the header has been saved but body has not.
	if err := shard.SaveHeader(header); err != nil {
		t.Fatalf("could not save header: %v", err)
	}

	if _, err := shard.CollationByHeaderHash(&hash); err == nil {
		t.Errorf("should not be able to fetch collation if body has not been saved")
	}

	// properly saves the collation.
	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("cannot save collation: %v", err)
	}

	dbCollation, err := shard.CollationByHeaderHash(&hash)
	if err != nil {
		t.Fatalf("could not fetch collation by hash: %v", err)
	}

	// Compare the hashes.
	if collation.Hash() != dbCollation.Hash() {
		t.Errorf("collations do not match. want=%v. got=%v", collation, dbCollation)
	}
}

func TestShard_ChunkRootfromHeaderHash(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.BytesToAddress([]byte{})
	proposerSignature := [32]byte{}
	header := NewCollationHeader(shardID, nil, period, &proposerAddress, proposerSignature)

	collation := NewCollation(header, []byte{1, 2, 3}, nil)
	collation.CalculateChunkRoot()
	shardDB := sharedDB.NewKVStore()
	shard := NewShard(shardID, shardDB)

	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("failed to save collation to shardDB: %v", err)
	}

	if err := shard.SetCanonical(header); err != nil {
		t.Fatalf("failed to set header as canonical: %v", err)
	}

	chunkroot := collation.header.ChunkRoot()
	headerHash := collation.header.Hash()

	newchunkroot, err := shard.ChunkRootfromHeaderHash(&headerHash)

	if err != nil {
		t.Errorf("Unable to retrieve chunk root from collation header hash: %v", err)
	}

	if !bytes.Equal(newchunkroot.Bytes(), chunkroot.Bytes()) {
		t.Errorf("Calculated chunk root, %v, and chunk root, %v, retrieved from headerHash is different", chunkroot, newchunkroot)
	}

}

func TestShard_CanonicalHeaderHash(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.BytesToAddress([]byte{})
	proposerSignature := [32]byte{}
	header := NewCollationHeader(shardID, nil, period, &proposerAddress, proposerSignature)

	collation := NewCollation(header, []byte{1, 2, 3}, nil)

	collation.CalculateChunkRoot()

	shardDB := sharedDB.NewKVStore()
	shard := NewShard(shardID, shardDB)

	// should not be able to set as canonical before saving the header and body first.
	if err := shard.SetCanonical(header); err == nil {
		t.Errorf("cannot set as canonical before saving header and body first")
	}

	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("failed to save collation to shardDB: %v", err)
	}

	if err := shard.SetCanonical(header); err != nil {
		t.Fatalf("failed to set header as canonical: %v", err)
	}

	headerHash := header.Hash()

	canonicalHeaderHash, err := shard.CanonicalHeaderHash(shardID, period)
	if err != nil {
		t.Fatalf("failed to get canonical header hash from shardDB: %v", err)
	}

	if _, err := shard.CanonicalHeaderHash(big.NewInt(100), big.NewInt(300)); err == nil {
		t.Errorf("should throw error if a non-existent period, shardID pair is used")
	}

	if canonicalHeaderHash.Hex() != headerHash.Hex() {
		t.Errorf("header hashes do not match. want=%s. got=%s", headerHash.Hex(), canonicalHeaderHash.Hex())
	}
}

func TestShard_CanonicalCollation(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.BytesToAddress([]byte{})
	proposerSignature := [32]byte{}
	emptyHash := common.BytesToHash([]byte{})
	header := NewCollationHeader(shardID, &emptyHash, period, &proposerAddress, proposerSignature)

	shardDB := sharedDB.NewKVStore()
	shard := NewShard(shardID, shardDB)

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	// we set the chunk root.
	collation.CalculateChunkRoot()

	// saves the full collation in the shardDB.
	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("failed to save collation to shardDB: %v", err)
	}

	// fetching non-existent shardID, period pair should throw error.
	if _, err := shard.CanonicalCollation(big.NewInt(100), big.NewInt(100)); err == nil {
		t.Errorf("fetching a")
	}

	// sets the correct information as canonical once the collation has been saved to shardDB.
	if err := shard.SetCanonical(header); err != nil {
		t.Fatalf("was not able to set header as canonical: %v", err)
	}

	canonicalCollation, err := shard.CanonicalCollation(shardID, period)
	if err != nil {
		t.Fatalf("failed to get canonical collation from shardDB: %v", err)
	}

	if canonicalCollation.Hash() != collation.Hash() {
		t.Errorf("collations are not equal. want=%v. got=%v.", collation, canonicalCollation)
	}
}

func TestShard_SetCanonical(t *testing.T) {
	chunkRoot := common.BytesToHash([]byte{})
	header := NewCollationHeader(big.NewInt(1), &chunkRoot, big.NewInt(1), nil, [32]byte{})

	shardDB := sharedDB.NewKVStore()
	shard := NewShard(big.NewInt(1), shardDB)
	otherShard := NewShard(big.NewInt(2), shardDB)

	// saving the header but not the full collation, and then trying to fetch
	// a canonical collation should throw an error.
	if err := shard.SaveHeader(header); err != nil {
		t.Fatalf("failed to save header to shardDB: %v", err)
	}

	if err := shard.SetCanonical(header); err == nil {
		t.Errorf("should not be able to set collation as canonical if header has no corresponding saved body")
	}

	// should not be allowed to set as canonical in a different shard.
	if err := otherShard.SetCanonical(header); err == nil {
		t.Errorf("should not be able to set header with ShardID=%v as canonical in other shard=%v", header.ShardID(), big.NewInt(2))
	}
}

func TestShard_BodyByChunkRoot(t *testing.T) {
	body := []byte{1, 2, 3, 4, 5}
	shardID := big.NewInt(1)
	shardDB := sharedDB.NewKVStore()
	shard := NewShard(shardID, shardDB)

	if err := shard.SaveBody(body); err != nil {
		t.Fatalf("cannot save body: %v", err)
	}

	chunks := Chunks(body)                   // wrapper allowing us to merklizing the chunks.
	chunkRoot := gethTypes.DeriveSha(chunks) // merklize the serialized blobs.

	dbBody, err := shard.BodyByChunkRoot(&chunkRoot)
	if err != nil {
		t.Errorf("cannot fetch body by chunk root: %v", err)
	}

	// it should throw error if fetching non-existent chunk root.
	emptyHash := common.BytesToHash([]byte{})
	if _, err := shard.BodyByChunkRoot(&emptyHash); err == nil {
		t.Errorf("non-existent chunk root should throw error: %v", err)
	}

	if !bytes.Equal(body, dbBody) {
		t.Errorf("bodies not equal. want=%v. got=%v", body, dbBody)
	}

	// setting the val of the key to nil.
	if err := shard.shardDB.Put([]byte{}, nil); err != nil {
		t.Fatalf("could not update shardDB: %v", err)
	}
	if _, err := shard.BodyByChunkRoot(&emptyHash); err == nil {
		t.Errorf("value set as nil in shardDB should return error from BodyByChunkRoot")
	}

}

func TestShard_CheckAvailability(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.BytesToAddress([]byte{})
	proposerSignature := [32]byte{}
	emptyHash := common.BytesToHash([]byte{})
	header := NewCollationHeader(shardID, &emptyHash, period, &proposerAddress, proposerSignature)

	shardDB := sharedDB.NewKVStore()
	shard := NewShard(shardID, shardDB)

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	// we set the chunk root.
	collation.CalculateChunkRoot()

	// should throw error if checking availability before header is even saved.
	if _, err := shard.CheckAvailability(header); err == nil {
		t.Errorf("error should be thrown: cannot check availability before saving header first")
	}

	if err := shard.SaveBody(collation.body); err != nil {
		t.Fatalf("cannot save body: %v", err)
	}

	available, err := shard.CheckAvailability(header)
	if err != nil {
		t.Errorf("could not check availability: %v", err)
	}
	if !available {
		t.Errorf("collation body is not available: chunkRoot=%s, body=%v", header.ChunkRoot().Hex(), collation.body)
	}
}

func TestShard_SetAvailability(t *testing.T) {
	chunkRoot := common.BytesToHash([]byte{})
	header := NewCollationHeader(big.NewInt(1), &chunkRoot, big.NewInt(1), nil, [32]byte{})

	// creates a mockDB that always returns nil values from .Get and errors in every other method.
	mockDB := &mockShardDB{kv: make(map[common.Hash][]byte)}

	// creates a well-functioning shardDB.
	shardDB := sharedDB.NewKVStore()

	// creates a shard with a functioning DB and another one with a faulty DB.
	shard := NewShard(big.NewInt(1), shardDB)
	errorShard := NewShard(big.NewInt(1), mockDB)

	if err := errorShard.SetAvailability(&chunkRoot, false); err == nil {
		t.Errorf("should not be able to set availability when using a faulty DB")
	}

	// sets the availability in a well-functioning shard with a good shardDB.
	if err := shard.SetAvailability(&chunkRoot, false); err != nil {
		t.Errorf("could not set availability for chunk root")
	}

	available, err := shard.CheckAvailability(header)
	if err != nil {
		t.Fatalf("unable to check availability for header: %v", err)
	}

	// availability should have been set to false.
	if available {
		t.Errorf("collation header should have been set to unavailable, instead was saved as available")
	}
}

func TestShard_SaveCollation(t *testing.T) {
	headerShardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.BytesToAddress([]byte{})
	proposerSignature := [32]byte{}
	emptyHash := common.BytesToHash([]byte{})
	header := NewCollationHeader(headerShardID, &emptyHash, period, &proposerAddress, proposerSignature)

	shardDB := sharedDB.NewKVStore()
	shard := NewShard(big.NewInt(2), shardDB)

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	// We set the chunk root.
	collation.CalculateChunkRoot()

	if err := shard.SaveCollation(collation); err == nil {
		t.Errorf("cannot save collation in shard with wrong shardID")
	}
}

func TestShard_SaveHeader(t *testing.T) {
	// creates a mockDB that always returns nil values from .Get and errors in every other method.
	mockDB := &mockShardDB{kv: make(map[common.Hash][]byte)}
	emptyHash := common.BytesToHash([]byte{})
	errorShard := NewShard(big.NewInt(1), mockDB)

	header := NewCollationHeader(big.NewInt(1), &emptyHash, big.NewInt(1), nil, [32]byte{})
	if err := errorShard.SaveHeader(header); err == nil {
		t.Errorf("should not be able to save header if a faulty shardDB is used")
	}
}

func TestShard_SaveBody(t *testing.T) {
	// creates a mockDB that always returns nil values from .Get and errors in every other method.
	mockDB := &mockShardDB{kv: make(map[common.Hash][]byte)}
	errorShard := NewShard(big.NewInt(1), mockDB)

	if err := errorShard.SaveBody([]byte{1, 2, 3}); err == nil {
		t.Errorf("should not be able to save body if a faulty shardDB is used")
	}
}
