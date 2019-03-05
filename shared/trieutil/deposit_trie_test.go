package trieutil

import (
	"bytes"
	"encoding/binary"
	"testing"

	dt "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestDepositTrie_UpdateDeposit(t *testing.T) {
	tests := []struct {
		deposits [][]byte
	}{
		{
			[][]byte{
				{1, 2, 3, 4},
				{5, 6, 7, 8},
			},
		},
		{
			[][]byte{
				{0, 0, 0, 0},
				{0, 0, 0, 0},
			},
		},
	}
	for _, tt := range tests {
		d := NewDepositTrie()
		d.UpdateDepositTrie(tt.deposits[0])
		if d.depositCount != 1 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}

		hashedData := hashutil.Hash(tt.deposits[0])

		for i := 0; i < 32; i++ {
			hashedData = hashutil.Hash(append(hashedData[:], d.zeroHashes[i][:]...))
		}
		if d.Root() != hashedData {
			t.Errorf("Expected %#x but got %#x", hashedData, d.Root())
		}

		d.UpdateDepositTrie(tt.deposits[1])
		if d.depositCount != 2 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}

		hash1 := hashutil.Hash(tt.deposits[0])
		hash2 := hashutil.Hash(tt.deposits[1])

		hashedData = hashutil.Hash(append(hash1[:], hash2[:]...))

		for i := 0; i < 31; i++ {
			hashedData = hashutil.Hash(append(hashedData[:], d.zeroHashes[i+1][:]...))
		}
		if d.Root() != hashedData {
			t.Errorf("Expected %#x but got %#x", hashedData, d.Root())
		}

	}
}

func TestVerifyMerkleBranch_OK(t *testing.T) {
	d := NewDepositTrie()
	deposit1 := []byte{1, 2, 3}
	d.UpdateDepositTrie(deposit1)
	deposit2 := []byte{5, 6, 7}
	d.UpdateDepositTrie(deposit2)
	deposit3 := []byte{8, 9, 10}
	d.UpdateDepositTrie(deposit3)
	index := make([]byte, 8)
	binary.LittleEndian.PutUint64(index, d.depositCount-1)
	branch := d.Branch()
	root := d.Root()
	if ok := VerifyMerkleBranch(
		branch,
		root,
		index,
	); !ok {
		t.Error("Expected Merkle branch to verify, received false")
	}
}
func TestToProtoDepositTrie_OK(t *testing.T) {
	d := NewDepositTrie()
	pdt := d.ToProtoDepositTrie()
	for i := 0; i < len(pdt.Branch); i++ {
		if !bytes.Equal(pdt.Branch[i], d.branch[i][:]) {
			t.Errorf("Expected zero values %#x of proto deposit trie but got %#x", d.branch[i][:], pdt.Branch[i])
		}
		if !bytes.Equal(pdt.ZeroHashes[i], d.zeroHashes[i][:]) {
			t.Errorf("Expected zero values %#x of proto deposit trie but got %#x", pdt.ZeroHashes[i], d.zeroHashes[i][:])
		}
	}
	if pdt.DepositCount != 0 {
		t.Errorf("Expected DepositCount to be 0 but got %v", pdt.DepositCount)
	}
	deposit1 := []byte{1, 2, 3}
	d.UpdateDepositTrie(deposit1)
	pdt = d.ToProtoDepositTrie()
	if pdt.DepositCount != 1 {
		t.Errorf("Expected DepositCount to be 1 but got %v", pdt.DepositCount)
	}

}

func TestFromProtoDepositTrie_OK(t *testing.T) {

	pr := &dt.DepositTrie{}
	d, err := FromProtoDepositTrie(pr)
	if err == nil {
		t.Errorf("expected error as deposit are not initianlizes")

	}
	if d.depositCount != 0 {
		t.Errorf("Expected DepositCount to be 0 but got %v", d.depositCount)
	}
	pr.DepositCount = 111
	d, err = FromProtoDepositTrie(pr)
	if d.depositCount != 111 {
		t.Errorf("Expected DepositCount to be 1 but got %v", pr.DepositCount)
	}
	d = NewDepositTrie()
	deposit1 := []byte{1, 2, 3}
	d.UpdateDepositTrie(deposit1)
	pr = d.ToProtoDepositTrie()
	d, err = FromProtoDepositTrie(pr)
	if err != nil {
		t.Errorf("FromProtoDepositTrie failed with error: %v", err)
	}
	if d.depositCount != 1 {
		t.Errorf("Expected DepositCount to be 1 but got %v", d.depositCount)
	}

}
