package state_test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func setupInitialDeposits(t *testing.T, numDeposits uint64) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		depositData := &pb.DepositData{
			Pubkey: priv.PublicKey().Marshal(),
			Amount: params.BeaconConfig().MaxDepositAmount,
		}

		deposits[i] = &pb.Deposit{Data: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func createRandaoReveal(t *testing.T, beaconState *pb.BeaconState, privKeys []*bls.SecretKey) []byte {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	epoch := uint64(0)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.DomainVersion(beaconState, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal()
}

func TestExecuteStateTransition_IncorrectSlot(t *testing.T) {
	beaconState := &pb.BeaconState{
		Slot: 5,
	}
	block := &pb.BeaconBlock{
		Slot: 4,
	}
	want := "expected state.slot"
	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	var slashings []*pb.ProposerSlashing
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings+1; i++ {
		slashings = append(slashings, &pb.ProposerSlashing{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: 0,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: slashings,
		},
	}
	want := "could not verify block proposer slashing"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectAttesterSlashing(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	var attesterSlashings []*pb.AttesterSlashing
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings+1; i++ {
		attesterSlashings = append(attesterSlashings, &pb.AttesterSlashing{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: 0,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: slashings,
			AttesterSlashings: attesterSlashings,
		},
	}
	want := "could not verify block attester slashing"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessBlockAttestations(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}

	var blockAttestations []*pb.Attestation
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations+1; i++ {
		blockAttestations = append(blockAttestations, &pb.Attestation{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: 0,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      blockAttestations,
		},
	}
	want := "could not process block attestations"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessExits(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRoots = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	beaconState.Slot = 10
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     0,
			JustifiedEpoch:           0,
			JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
			LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
			CrosslinkDataRoot:        params.BeaconConfig().ZeroHash[:],
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	var exits []*pb.VoluntaryExit
	for i := uint64(0); i < params.BeaconConfig().MaxVoluntaryExits+1; i++ {
		exits = append(exits, &pb.VoluntaryExit{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: 10,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}
	want := "could not process validator exits"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRoots = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = (params.BeaconConfig().PersistentCommitteePeriod * slotsPerEpoch)
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     beaconState.Slot - params.BeaconConfig().MinAttestationInclusionDelay,
			JustifiedEpoch:           0,
			JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
			LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
			CrosslinkDataRoot:        params.BeaconConfig().ZeroHash[:],
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 10,
			Epoch:          0,
		},
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: beaconState.Slot,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); err != nil {
		t.Errorf("Expected block to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_CantGetTgtAttsPrevEpoch(t *testing.T) {
	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{TargetEpoch: 1}}}
	_, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{CurrentEpochAttestations: atts})
	if !strings.Contains(err.Error(), "could not get target atts prev epoch") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestProcessEpoch_CantGetTgtAttsCurrEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Slot: 100}}}
	_, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{
		Slot:                     e,
		LatestBlockRoots:         make([][]byte, 128),
		LatestRandaoMixes:        make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:   make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		CurrentEpochAttestations: atts})
	if !strings.Contains(err.Error(), "could not get target atts current epoch") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestProcessEpoch_CantGetAttsBalancePrevEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Slot: 1}, AggregationBitfield: []byte{1}}}
	_, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{
		Slot:                      e + 1,
		LatestBlockRoots:          make([][]byte, 128),
		LatestRandaoMixes:         make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:    make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		PreviousEpochAttestations: atts})
	if !strings.Contains(err.Error(), "could not get attesting balance prev epoch") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestProcessEpoch_CantGetAttsBalanceCurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Slot: 1}, AggregationBitfield: []byte{1}}}
	_, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{
		Slot:                     e + 1,
		LatestBlockRoots:         make([][]byte, 128),
		LatestRandaoMixes:        make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:   make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		CurrentEpochAttestations: atts})
	if !strings.Contains(err.Error(), "could not get attesting balance current epoch") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestProcessEpoch_CanProcess(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Slot: 1}}}
	var crosslinks []*pb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &pb.Crosslink{
			Epoch:                   0,
			CrosslinkDataRootHash32: []byte{'A'},
		})
	}
	newState, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{
		Slot:                     e + 1,
		LatestBlockRoots:         make([][]byte, 128),
		LatestSlashedBalances:    []uint64{0, 1e9, 0},
		LatestRandaoMixes:        make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:   make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		CurrentCrosslinks:        crosslinks,
		CurrentEpochAttestations: atts})
	if err != nil {
		t.Fatal(err)
	}

	wanted := uint64(1e9)
	if newState.LatestSlashedBalances[2] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Balances[2])
	}
}
