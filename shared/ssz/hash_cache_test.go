package ssz

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHashKeyFn_OK(t *testing.T) {
	mRoot := &root{
		Hash: common.HexToHash("0x0123456"),
	}

	key, err := hashKeyFn(mRoot)
	if err != nil {
		t.Fatal(err)
	}
	if key != mRoot.Hash.Hex() {
		t.Errorf("Incorrect hash key: %s, expected %s", key, mRoot.Hash.Hex())
	}
}

func TestHashKeyFn_InvalidObj(t *testing.T) {
	_, err := hashKeyFn("bad")
	if err != ErrNotMarakleRoot {
		t.Errorf("Expected error %v, got %v", ErrNotMarakleRoot, err)
	}
}

func TestObjCache_byHash(t *testing.T) {
	cache := newHashCache()

	byteSl := [][]byte{{0, 0}, {1, 1}}
	mr, err := merkleHash(byteSl)
	if err != nil {
		t.Fatal(err)
	}
	hs, err := hashedEncoding(reflect.ValueOf(byteSl))
	if err != nil {
		t.Fatal(err)
	}
	exists, _, err := cache.RootByHash(bytesutil.ToBytes32(hs))

	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected block info not to exist in empty cache")
	}

	if _, err := cache.AddRetriveMarkleRoot(byteSl); err != nil {
		t.Fatal(err)
	}

	exists, fetchedInfo, err := cache.RootByHash(bytesutil.ToBytes32(hs))

	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected blockInfo to exist")
	}
	if bytes.Compare(mr, fetchedInfo.MarkleRoot) != 0 {
		t.Errorf(
			"Expected fetched info number to be %v, got %v",
			mr,
			fetchedInfo.MarkleRoot,
		)
	}
	if fetchedInfo.Hash != bytesutil.ToBytes32(hs) {
		t.Errorf(
			"Expected fetched info hash to be %v, got %v",
			hs,
			fetchedInfo.Hash,
		)
	}
}

func TestMerkleHashWithCache(t *testing.T) {
	cache := newHashCache()
	for i := 0; i < 200; i++ {

		runMerkleHashTests(t, func(val [][]byte) ([]byte, error) {
			return merkleHash(val)
		})

	}

	for i := 0; i < 200; i++ {

		runMerkleHashTests(t, func(val [][]byte) ([]byte, error) {
			return cache.AddRetriveMarkleRoot(val)
		})

	}

}

// EncodeDepositData converts a deposit input proto into an a byte slice
// of Simple Serialized deposit input followed by 8 bytes for a deposit value
// and 8 bytes for a unix timestamp, all in LittleEndian format.
func EncodeDepositData(
	depositInput *pb.DepositInput,
	depositValue uint64,
	depositTimestamp int64,
) ([]byte, error) {
	wBuf := new(bytes.Buffer)
	if err := Encode(wBuf, depositInput); err != nil {
		return nil, fmt.Errorf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	depositData := make([]byte, 0, 512)
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, depositValue)
	timestamp := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTimestamp))

	depositData = append(depositData, value...)
	depositData = append(depositData, timestamp...)
	depositData = append(depositData, encodedInput...)

	return depositData, nil
}

// generateInitialSimulatedDeposits generates initial deposits for creating a beacon state in the simulated
// backend based on the yaml configuration.
func generateInitialSimulatedDeposits(numDeposits uint64) ([]*pb.Deposit, []*bls.SecretKey, error) {
	genesisTime := time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC).Unix()
	deposits := make([]*pb.Deposit, numDeposits)
	privKeys := make([]*bls.SecretKey, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("could not initialize key: %v", err)
		}
		depositInput := &pb.DepositInput{
			Pubkey:                      priv.PublicKey().Marshal(),
			WithdrawalCredentialsHash32: make([]byte, 32),
			ProofOfPossession:           make([]byte, 96),
		}
		depositData, err := EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
			genesisTime,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("could not encode genesis block deposits: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys, nil
}

func TestDepositsBLSHash(t *testing.T) {
	cache := newHashCache()
	initialDeposits, blsg, err := generateInitialSimulatedDeposits(1000)
	if err != nil {
		t.Errorf("test: unexpected error: %v\n", err)
	}
	type tree struct {
		Dep []*pb.Deposit
		BLS []*bls.SecretKey
	}

	startTime := time.Now().UnixNano()
	output, err := cache.AddRetrieveTrieRoot(&tree{
		Dep: initialDeposits,
		BLS: blsg,
	})

	fmt.Printf("time it took: %v \n", time.Now().UnixNano()-startTime)
	startTime = time.Now().UnixNano()
	output2, err := cache.AddRetrieveTrieRoot(&tree{
		Dep: initialDeposits,
		BLS: blsg,
	})
	fmt.Printf("time it took when cached: %v \n", time.Now().UnixNano()-startTime)

	// Check expected output
	if err == nil && !bytes.Equal(output[:], output2[:]) {
		t.Errorf("output mismatch:\ngot   %X\nwant  %X\n",
			output2, output)
	}
}

func TestBlockCache_maxSize(t *testing.T) {
	cache := newHashCache()
	maxCacheSize = 10000
	for i := uint64(0); i < uint64(maxCacheSize+10); i++ {

		if err := cache.AddRoot(bytesutil.ToBytes32(bytesutil.Bytes4(i)), []byte{1}); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.hashCache.ListKeys()) != maxCacheSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxCacheSize,
			len(cache.hashCache.ListKeys()),
		)
	}

}
