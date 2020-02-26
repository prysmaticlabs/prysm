package filter

import (
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func (f Filter) String() string {
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
	f, err := NewFilter(dataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	got := f.String()
	want := "...1.....1.1..11"
	if got != want {
		t.Fatalf("%d bits, got %q, want %q", len(f)*8, got, want)
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
	f, err := NewFilter(dataRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	got := f.String()

	for i := uint64(0); i < 1000; i++ {
		attData.Source.Root = []byte(fmt.Sprintf("%d dsadadadd", i))
		dataRoot, err := ssz.HashTreeRoot(attData)
		if err != nil {
			t.Fatal(err)
		}
		filter, err := NewFilter(dataRoot[:])
		if err != nil {
			t.Fatal(err)
		}
		if filter.String() == got {
			t.Fatalf("coliision at %d", i)
		}
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
		_, err := NewFilter(dataRoot[:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestFilter_Output(t *testing.T) {
	testCases := []struct {
		key  string
		want string
	}{
		{key: "eth2", want: "1.1.....1.....11"},
		{key: "is really", want: "...1..1.1.1...1."},
		{key: "awesome", want: "..11..1.....1.1."},
		{key: "and you", want: "...11......1..1."},
		{key: "should", want: "....1..1..1.1.1."},
		{key: "be part", want: "...1.....1..1.11"},
		{key: "of the vision", want: "...1.11....1..1."},
		{key: ":woke:", want: "..1........11.11"},
	}
	for _, tc := range testCases {
		bytes32 := bytesutil.ToBytes32([]byte(tc.key))
		got, err := NewFilter(bytes32[:])
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != tc.want {
			t.Errorf("Unexpected filter result, received %s, expected %s", got, tc.want)
		}
	}
}
