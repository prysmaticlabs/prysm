package helpers_test

import (
	"bytes"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestAggregateAttestation(t *testing.T) {
	tests := []struct {
		a1   *ethpb.Attestation
		a2   *ethpb.Attestation
		want *ethpb.Attestation
	}{
		{a1: &ethpb.Attestation{AggregationBits: []byte{}},
			a2:   &ethpb.Attestation{AggregationBits: []byte{}},
			want: &ethpb.Attestation{AggregationBits: []byte{}}},
		{a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x03}},
			a2:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x02}},
			want: &ethpb.Attestation{AggregationBits: []byte{0x03}}},
		{a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x02}},
			a2:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x03}},
			want: &ethpb.Attestation{AggregationBits: []byte{0x03}}},
	}
	for _, tt := range tests {
		got, err := helpers.AggregateAttestation(tt.a1, tt.a2)
		if err != nil {
			t.Fatal(err)
		}
		if !ssz.DeepEqual(got, tt.want) {
			t.Errorf("AggregateAttestation() = %v, want %v", got, tt.want)
		}
	}
}

func TestAggregateAttestation_OverlapFails(t *testing.T) {
	tests := []struct {
		a1 *ethpb.Attestation
		a2 *ethpb.Attestation
	}{
		{a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x1F}},
			a2: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x11}}},
		{a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0xFF, 0x85}},
			a2: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x13, 0x8F}}},
	}
	for _, tt := range tests {
		_, err := helpers.AggregateAttestation(tt.a1, tt.a2)
		if err != helpers.ErrAttestationAggregationBitsOverlap {
			t.Error("Did not receive wanted error")
		}
	}
}

func TestAggregateAttestation_DiffLengthFails(t *testing.T) {
	tests := []struct {
		a1 *ethpb.Attestation
		a2 *ethpb.Attestation
	}{
		{a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x0F}},
			a2: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x11}}},
	}
	for _, tt := range tests {
		_, err := helpers.AggregateAttestation(tt.a1, tt.a2)
		if err != helpers.ErrAttestationAggregationBitsDifferentLen {
			t.Error("Did not receive wanted error")
		}
	}
}

func bitlistWithAllBitsSet(length uint64) bitfield.Bitlist {
	b := bitfield.NewBitlist(length)
	for i := uint64(0); i < length; i++ {
		b.SetBitAt(i, true)
	}
	return b
}

func bitlistsWithSingleBitSet(length uint64) []bitfield.Bitlist {
	lists := make([]bitfield.Bitlist, length)
	for i := uint64(0); i < length; i++ {
		b := bitfield.NewBitlist(length)
		b.SetBitAt(i, true)
		lists[i] = b
	}
	return lists
}

func TestAggregateAttestations(t *testing.T) {
	// Each test defines the aggregation bitfield inputs and the wanted output result.
	tests := []struct {
		name   string
		inputs []bitfield.Bitlist
		want   []bitfield.Bitlist
	}{
		{
			name: "two attestations with no overlap",
			inputs: []bitfield.Bitlist{
				{0b00000001, 0b1},
				{0b00000010, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b1},
			},
		},
		{
			name:   "256 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(256),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(256),
			},
		},
		{
			name:   "1024 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(1024),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(1024),
			},
		},
		{
			name: "two attestations with overlap",
			inputs: []bitfield.Bitlist{
				{0b00000101, 0b1},
				{0b00000110, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000101, 0b1},
				{0b00000110, 0b1},
			},
		},
		{
			name: "some attestations overlap",
			inputs: []bitfield.Bitlist{
				{0b00001001, 0b1},
				{0b00010110, 0b1},
				{0b00001010, 0b1},
				{0b00110001, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00111011, 0b1},
				{0b00011111, 0b1},
			},
		},
		{
			name: "some attestations produce duplicates which are removed",
			inputs: []bitfield.Bitlist{
				{0b00000101, 0b1},
				{0b00000110, 0b1},
				{0b00001010, 0b1},
				{0b00001001, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00001111, 0b1}, // both 0&1 and 2&3 produce this bitlist
			},
		},
		{
			name: "two attestations where one is fully contained within the other",
			inputs: []bitfield.Bitlist{
				{0b00000001, 0b1},
				{0b00000011, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b1},
			},
		},
		{
			name: "two attestations where one is fully contained within the other reversed",
			inputs: []bitfield.Bitlist{
				{0b00000011, 0b1},
				{0b00000001, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b1},
			},
		},
		{
			name: "attestations with different bitlist lengths",
			inputs: []bitfield.Bitlist{
				{0b00000011, 0b10},
				{0b00000111, 0b100},
				{0b00000100, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b10},
				{0b00000111, 0b100},
				{0b00000100, 0b1},
			},
		},
	}

	var makeAttestationsFromBitlists = func(bl []bitfield.Bitlist) []*ethpb.Attestation {
		atts := make([]*ethpb.Attestation, len(bl))
		for i, b := range bl {
			sk := bls.RandKey()
			sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)
			atts[i] = &ethpb.Attestation{
				AggregationBits: b,
				Data:            nil,
				Signature:       sig.Marshal(),
			}
		}
		return atts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := helpers.AggregateAttestations(makeAttestationsFromBitlists(tt.inputs))
			if err != nil {
				t.Fatal(err)
			}
			sort.Slice(got, func(i, j int) bool {
				return got[i].AggregationBits.Bytes()[0] < got[j].AggregationBits.Bytes()[0]
			})
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Bytes()[0] < tt.want[j].Bytes()[0]
			})
			if len(got) != len(tt.want) {
				t.Logf("got=%v", got)
				t.Fatalf("Wrong number of responses. Got %d, wanted %d", len(got), len(tt.want))
			}
			for i, w := range tt.want {
				if !bytes.Equal(got[i].AggregationBits.Bytes(), w.Bytes()) {
					t.Errorf("Unexpected bitlist at index %d, got %b, wanted %b", i, got[i].AggregationBits.Bytes(), w.Bytes())
				}
			}
		})
	}
}

func TestSlotSignature_Verify(t *testing.T) {
	priv := bls.RandKey()
	pub := priv.PublicKey()
	state, _ := beaconstate.InitializeFromProto(&pb.BeaconState{
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			Epoch:           0,
		},
		Slot: 100,
	})
	slot := uint64(101)

	sig, err := helpers.SlotSignature(state, slot, priv)
	if err != nil {
		t.Fatal(err)
	}

	domain := helpers.Domain(state.Fork(), helpers.CurrentEpoch(state), params.BeaconConfig().DomainBeaconAttester)
	msg, _ := ssz.HashTreeRoot(slot)
	if !sig.Verify(msg[:], pub, domain) {
		t.Error("Could not verify slot signature")
	}
}

func TestIsAggregator_True(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	committee, err := helpers.BeaconCommitteeFromState(beaconState, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig := privKeys[0].Sign([]byte{}, 0)
	agg, err := helpers.IsAggregator(uint64(len(committee)), sig.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if !agg {
		t.Error("Wanted aggregator true, got false")
	}
}

func TestIsAggregator_False(t *testing.T) {
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 2048)

	committee, err := helpers.BeaconCommitteeFromState(beaconState, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	sig := privKeys[0].Sign([]byte{}, 0)
	agg, err := helpers.IsAggregator(uint64(len(committee)), sig.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if agg {
		t.Error("Wanted aggregator false, got true")
	}
}

func TestAggregateSignature_True(t *testing.T) {
	pubkeys := make([]*bls.PublicKey, 0, 100)
	atts := make([]*ethpb.Attestation, 0, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv := bls.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:], 0)
		pubkeys = append(pubkeys, pub)
		att := &ethpb.Attestation{Signature: sig.Marshal()}
		atts = append(atts, att)
	}
	aggSig, err := helpers.AggregateSignature(atts)
	if err != nil {
		t.Fatal(err)
	}
	if !aggSig.VerifyAggregateCommon(pubkeys, bytesutil.ToBytes32(msg), 0) {
		t.Error("Signature did not verify")
	}
}

func TestAggregateSignature_False(t *testing.T) {
	pubkeys := make([]*bls.PublicKey, 0, 100)
	atts := make([]*ethpb.Attestation, 0, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv := bls.RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:], 0)
		pubkeys = append(pubkeys, pub)
		att := &ethpb.Attestation{Signature: sig.Marshal()}
		atts = append(atts, att)
	}
	aggSig, err := helpers.AggregateSignature(atts[0 : len(atts)-2])
	if err != nil {
		t.Fatal(err)
	}
	if aggSig.VerifyAggregateCommon(pubkeys, bytesutil.ToBytes32(msg), 0) {
		t.Error("Signature not suppose to verify")
	}
}
