package filter

import (
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

func (f BloomFilter) String() string {
	return fmt.Sprintf("%08b", f)
}

func TestBloomFilter_OK(t *testing.T) {
	attData := &ethpb.AttestationData{
		Slot:           4,
		CommitteeIndex: 2,
		Target: &ethpb.Checkpoint{
			Epoch: 4,
			Root:  []byte("wow"),
		},
		Source: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  []byte("eth2"),
		},
		BeaconBlockRoot: []byte("is great"),
	}
	dataRoot, err := ssz.HashTreeRoot(attData)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewBloomFilter(dataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	got := f.String()
	want := "[00001000 01101010]"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	found, err := f.Contains(dataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("Did not expect filter to not contain entered data")
	}

	diffAttData := &ethpb.AttestationData{
		Slot:           4,
		CommitteeIndex: 2,
		Target: &ethpb.Checkpoint{
			Epoch: 4,
			Root:  []byte("wowzers"),
		},
		Source: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  []byte("eth2.0"),
		},
		BeaconBlockRoot: []byte("is really great"),
	}
	diffDataRoot, err := ssz.HashTreeRoot(diffAttData)
	if err != nil {
		t.Fatal(err)
	}
	found, err = f.Contains(diffDataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("Did not expect filter to contain non-existent root")
	}

	diffAttData = &ethpb.AttestationData{
		Slot:           4,
		CommitteeIndex: 2,
		Target: &ethpb.Checkpoint{
			Epoch: 4,
			Root:  []byte("hahaha"),
		},
		Source: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  []byte("i love"),
		},
		BeaconBlockRoot: []byte("eth2"),
	}
	diffDataRoot, err = ssz.HashTreeRoot(diffAttData)
	if err != nil {
		t.Fatal(err)
	}
	found, err = f.Contains(diffDataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("Did not expect filter to contain non-existent root")
	}
}

func TestBloomFilter_NoCollisions(t *testing.T) {
	attData := &ethpb.AttestationData{
		Slot:           4,
		CommitteeIndex: 2,
		Target: &ethpb.Checkpoint{
			Epoch: 4,
			Root:  []byte("wow"),
		},
		Source: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  []byte("eth2"),
		},
		BeaconBlockRoot: []byte("is great"),
	}
	dataRoot, err := ssz.HashTreeRoot(attData)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewBloomFilter(dataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	got := f.String()

	for i := uint64(0); i < 1000; i++ {
		attData.Source.Epoch = i + 5
		dataRoot, err = ssz.HashTreeRoot(attData)
		if err != nil {
			t.Fatal(err)
		}
		filter, err := NewBloomFilter(dataRoot[:])
		if err != nil {
			t.Fatal(err)
		}
		if filter.String() == got {
			t.Fatalf("Unexpected coliision at %d", i)
		}
	}
}

func TestBloomFilter_Output(t *testing.T) {
	testCases := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "large key",
			key:  "A very very large key, almost too large. This key is wayyyy too large.",
			want: "[00110000 00101100]",
		},
		{
			name: "small key",
			key:  "A small key",
			want: "[00000000 11100100]",
		},
		{
			name: "tiny key",
			key:  "Tiny",
			want: "[10000000 00111000]",
		},
		{
			name: "empty key",
			key:  "",
			want: "[00101011 00000000]",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewBloomFilter([]byte(tc.key))
			if err != nil {
				t.Fatal(err)
			}
			if filter.String() != tc.want {
				t.Errorf("Unexpected filter result, received %s, expected %s", filter.String(), tc.want)
			}
			found, err := filter.Contains([]byte(tc.key))
			if err != nil {
				t.Fatal(err)
			}
			if !found {
				t.Fatal("Unexpected failure of contain")
			}
		})
	}
}

func BenchmarkNewBloomFilter(b *testing.B) {
	attData := &ethpb.AttestationData{
		Slot:           4,
		CommitteeIndex: 2,
		Target: &ethpb.Checkpoint{
			Epoch: 4,
			Root:  []byte("haaaaaaaaaaaaaaa"),
		},
		Source: &ethpb.Checkpoint{
			Epoch: 1,
			Root:  []byte("wooooooooooooooooooo"),
		},
		BeaconBlockRoot: []byte("hoooooooooooooooo"),
	}
	dataRoot, err := ssz.HashTreeRoot(attData)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := NewBloomFilter(dataRoot[:])
		if err != nil {
			b.Fatal(err)
		}
	}
}
