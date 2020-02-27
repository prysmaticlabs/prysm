package filter

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

func (f BloomFilter) String() string {
	s := make([]byte, 8*len(f))
	for i, x := range f {
		for j := 0; j < 8; j++ {
			if x&(1<<uint(j)) != 0 {
				s[8*i+j] = '1'
			} else {
				s[8*i+j] = '.'
			}
		}
	}
	return string(s)
}

func TestFilter_OK(t *testing.T) {
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
	want := "...1.....1.1.11."
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

func TestFilter_NoCollisions(t *testing.T) {
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

func TestFilter_Output(t *testing.T) {
	testCases := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "large key",
			key:  "A very very large key, almost too large. This key is wayyyy too large.",
			want: "....11....11.1..",
		},
		{
			name: "small key",
			key:  "A small key",
			want: "..........1..111",
		},
		{
			name: "tiny key",
			key:  "Tiny",
			want: ".......1...111..",
		},
		{
			name: "empty key",
			key:  "",
			want: "11.1.1..........",
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

func BenchmarkNewFilter(b *testing.B) {
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
