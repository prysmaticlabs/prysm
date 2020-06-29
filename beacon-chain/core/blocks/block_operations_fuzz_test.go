package blocks

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"

	//"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestFuzzProcessAttestationNoVerify_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	ctx := context.Background()
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	att := &eth.Attestation{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(att)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		_, err = ProcessAttestationNoVerify(ctx, s, att)
		_ = err
	}
}

func TestFuzzProcessBlockHeader_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	block := &eth.SignedBeaconBlock{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(block)

		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		_, err = ProcessBlockHeader(s, block)
		_ = err
	}
}

func TestFuzzverifyDepositDataSigningRoot_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	ba := []byte{}
	pubkey := [48]byte{}
	sig := [96]byte{}
	domain := [4]byte{}
	p := []byte{}
	s := []byte{}
	d := []byte{}
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(&ba)
		fuzzer.Fuzz(&pubkey)
		fuzzer.Fuzz(&sig)
		fuzzer.Fuzz(&domain)
		fuzzer.Fuzz(&p)
		fuzzer.Fuzz(&s)
		fuzzer.Fuzz(&d)
		err := verifySignature(ba, pubkey[:], sig[:], domain[:])
		err = verifySignature(ba, p, s, d)
		_ = err
	}
}

func TestFuzzProcessEth1DataInBlock_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	block := &eth.BeaconBlock{}
	state := &stateTrie.BeaconState{}
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(block)
		s, err := ProcessEth1DataInBlock(state, block)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v and block: %v", s, err, state, block)
		}
	}
}

func TestFuzzareEth1DataEqual_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	eth1data := &eth.Eth1Data{}
	eth1data2 := &eth.Eth1Data{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(eth1data)
		fuzzer.Fuzz(eth1data2)
		AreEth1DataEqual(eth1data, eth1data2)
		AreEth1DataEqual(eth1data, eth1data)
	}
}

func TestFuzzEth1DataHasEnoughSupport_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	eth1data := &eth.Eth1Data{}
	stateVotes := []*eth.Eth1Data{}
	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(eth1data)
		fuzzer.Fuzz(&stateVotes)
		s, err := beaconstate.InitializeFromProto(&ethereum_beacon_p2p_v1.BeaconState{
			Eth1DataVotes: stateVotes,
		})
		_, err = Eth1DataHasEnoughSupport(s, eth1data)
		_ = err
	}

}

func TestFuzzProcessBlockHeaderNoVerify_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	block := &eth.BeaconBlock{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(block)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		_, err = ProcessBlockHeaderNoVerify(s, block)
		_ = err
	}
}

func TestFuzzProcessRandao_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessRandao(s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzProcessRandaoNoVerify_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessRandaoNoVerify(s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzProcessProposerSlashings_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessProposerSlashings(ctx, s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzVerifyProposerSlashing_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	proposerSlashing := &eth.ProposerSlashing{}
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(proposerSlashing)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		err = VerifyProposerSlashing(s, proposerSlashing)
		_ = err
	}
}

func TestFuzzProcessAttesterSlashings_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessAttesterSlashings(ctx, s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzVerifyAttesterSlashing_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	attesterSlashing := &eth.AttesterSlashing{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(attesterSlashing)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		err = VerifyAttesterSlashing(ctx, s, attesterSlashing)
		_ = err
	}
}

func TestFuzzIsSlashableAttestationData_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	attestationData := &eth.AttestationData{}
	attestationData2 := &eth.AttestationData{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(attestationData)
		fuzzer.Fuzz(attestationData2)
		IsSlashableAttestationData(attestationData, attestationData2)
	}
}

func TestFuzzslashableAttesterIndices_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	attesterSlashing := &eth.AttesterSlashing{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(attesterSlashing)
		slashableAttesterIndices(attesterSlashing)
	}
}

func TestFuzzProcessAttestations_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessAttestations(ctx, s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzProcessAttestationsNoVerify_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessAttestationsNoVerify(ctx, s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzProcessAttestation_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	attestation := &eth.Attestation{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(attestation)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessAttestation(ctx, s, attestation)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, attestation)
		}
	}
}

func TestFuzzVerifyIndexedAttestationn_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	idxAttestation := &eth.IndexedAttestation{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(idxAttestation)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		err = VerifyIndexedAttestation(ctx, s, idxAttestation)
		_ = err
	}
}

func TestFuzzVerifyAttestation_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	attestation := &eth.Attestation{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(attestation)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		err = VerifyAttestation(ctx, s, attestation)
		_ = err
	}
}

func TestFuzzProcessDeposits_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessDeposits(ctx, s, blockBody.Deposits)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzProcessPreGenesisDeposit_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	deposit := &eth.Deposit{}
	ctx := context.Background()

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(deposit)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessPreGenesisDeposits(ctx, s, []*eth.Deposit{deposit})
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, deposit)
		}
	}
}

func TestFuzzProcessDeposit_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	deposit := &eth.Deposit{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(deposit)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessDeposit(s, deposit, true)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, deposit)
		}
	}
}

func TestFuzzverifyDeposit_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	deposit := &eth.Deposit{}
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(deposit)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		err = verifyDeposit(s, deposit)
		_ = err
	}
}

func TestFuzzProcessVoluntaryExits_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessVoluntaryExits(ctx, s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzProcessVoluntaryExitsNoVerify_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	blockBody := &eth.BeaconBlockBody{}
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(blockBody)
		s, err := beaconstate.InitializeFromProtoUnsafe(state)
		r, err := ProcessVoluntaryExitsNoVerify(s, blockBody)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, blockBody)
		}
	}
}

func TestFuzzVerifyExit_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	ve := &eth.SignedVoluntaryExit{}
	val := &beaconstate.ReadOnlyValidator{}
	fork := &pb.Fork{}
	var slot uint64

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(ve)
		fuzzer.Fuzz(val)
		fuzzer.Fuzz(fork)
		fuzzer.Fuzz(&slot)
		err := VerifyExit(val, slot, fork, ve, params.BeaconConfig().ZeroHash[:])
		_ = err
	}
}
