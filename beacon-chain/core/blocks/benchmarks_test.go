package blocks_test

import (
	"bytes"
	"context"
	// "crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var ValidatorCount = 131072
var RunAmount = 67108864 / ValidatorCount

func setBenchmarkConfig() {
	c := params.BeaconConfig()
	// From Danny Ryan's "Minimal Config"
	// c.SlotsPerEpoch = 8
	// c.MinAttestationInclusionDelay = 2
	// c.TargetCommitteeSize = 4
	// c.GenesisEpoch = c.GenesisSlot / 8
	// c.LatestRandaoMixesLength = 64
	// c.LatestActiveIndexRootsLength = 64
	// c.LatestSlashedExitLength = 64
	params.OverrideBeaconConfig(c)
}

func BenchmarkProcessBlockRandao(b *testing.B) {
	deposits, _ := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	block := &pb.BeaconBlock{
		RandaoReveal: []byte{2, 3, 4},
	}

	b.N = 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = blocks.ProcessBlockRandao(
			context.Background(),
			beaconState,
			block,
			false, /* verify signatures */
			false, /* disable logging */
		)
	}
}

func BenchmarkProcessEth1Data(b *testing.B) {
	deposits, _ := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}
	beaconState.Eth1DataVotes = []*pb.Eth1DataVote{
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{0},
				BlockHash32:       []byte{1},
			},
			VoteCount: 5,
		}, {
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{2},
				BlockHash32:       []byte{3},
			},
			VoteCount: 2,
		},
	}

	block := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = blocks.ProcessEth1DataInBlock(context.Background(), beaconState, block)
	}
}

func BenchmarkProcessProposerSlashings(b *testing.B) {
	validators := make([]*pb.Validator, ValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:       params.BeaconConfig().GenesisEpoch + 1,
			SlashedEpoch:    params.BeaconConfig().GenesisEpoch + 1,
			WithdrawalEpoch: params.BeaconConfig().GenesisEpoch + 1,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            params.BeaconConfig().GenesisSlot + 1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            params.BeaconConfig().GenesisSlot + 1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
		},
	}
	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch
	beaconState := &pb.BeaconState{
		ValidatorRegistry:     validators,
		Slot:                  currentSlot,
		ValidatorBalances:     validatorBalances,
		LatestSlashedBalances: []uint64{0},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = blocks.ProcessProposerSlashings(
			context.Background(),
			beaconState,
			block,
			false,
		)
	}
}

func BenchmarkProcessAttesterSlashings(b *testing.B) {
	validators := make([]*pb.Validator, ValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:       params.BeaconConfig().GenesisEpoch + 1,
			SlashedEpoch:    params.BeaconConfig().FarFutureEpoch,
			WithdrawalEpoch: params.BeaconConfig().GenesisEpoch + 1*params.BeaconConfig().SlotsPerEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	att1 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch,
		JustifiedEpoch: 5,
	}
	att2 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch,
		JustifiedEpoch: 4,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}

	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch
	beaconState := &pb.BeaconState{
		ValidatorRegistry:     validators,
		Slot:                  currentSlot,
		ValidatorBalances:     validatorBalances,
		LatestSlashedBalances: make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessAttesterSlashings(
			context.Background(),
			beaconState,
			block,
			false,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessBlockAttestations(b *testing.B) {
	deposits, _ := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch + 3
	beaconState.PreviousJustifiedEpoch = params.BeaconConfig().GenesisEpoch
	beaconState.LatestBlockRootHash32S = blockRoots
	beaconState.LatestCrosslinks = stateLatestCrosslinks

	attestations := []*pb.Attestation{}
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations; i++ {
		att1 := &pb.Attestation{
			Data: &pb.AttestationData{
				Shard:                    0,
				Slot:                     params.BeaconConfig().GenesisSlot + 20,
				JustifiedBlockRootHash32: blockRoots[0],
				LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
				CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			},
			AggregationBitfield: []byte{1},
			CustodyBitfield:     []byte{1},
		}
		attestations = append(attestations, att1)
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessBlockAttestations(
			context.Background(),
			beaconState,
			block,
			false,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessValidatorDeposits(b *testing.B) {
	deposits, _ := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot

	newDeposits := []*pb.Deposit{}
	allData := [][]byte{}
	for i := uint64(0); i < params.BeaconConfig().MaxDeposits; i++ {
		pubkey := make([]byte, 32)
		binary.LittleEndian.PutUint64(pubkey, uint64(i))
		depositInput := &pb.DepositInput{
			Pubkey: pubkey,
		}
		wBuf := new(bytes.Buffer)
		if err := ssz.Encode(wBuf, depositInput); err != nil {
			b.Fatalf("failed to encode deposit input: %v", err)
		}
		encodedInput := wBuf.Bytes()

		data := []byte{}
		value := make([]byte, 8)
		depositValue := uint64(1000)
		binary.LittleEndian.PutUint64(value, depositValue)

		timestamp := make([]byte, 8)
		depositTime := time.Unix(1000, 0).Unix()
		binary.LittleEndian.PutUint64(timestamp, uint64(depositTime))

		data = append(data, value...)
		data = append(data, timestamp...)
		data = append(data, encodedInput...)
		allData = append(allData, data)
	}

	for i := uint64(0); i < params.BeaconConfig().MaxDeposits; i++ {
		depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			b.Fatalf("Could not generate trie: %v", err)
		}
		proof, err := depositTrie.MerkleProof(int(i))
		if err != nil {
			b.Fatalf("Could not generate proof: %v", err)
		}

		newDeposits = append(newDeposits, &pb.Deposit{
			DepositData:         allData[i],
			MerkleBranchHash32S: proof,
			MerkleTreeIndex:     i,
		})
	}

	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: newDeposits,
		},
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Fatalf("Could not generate trie: %v", err)
	}
	root := depositTrie.Root()
	beaconState.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: root[:],
		BlockHash32:       root[:],
	}
	beaconState.Slot = currentSlot

	b.N = 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = blocks.ProcessValidatorDeposits(
			context.Background(),
			beaconState,
			block,
		)
		if err != nil {
			b.Fatalf("Expected block deposits to process correctly, received: %v", err)
		}
	}
}

func BenchmarkProcessValidatorExits(b *testing.B) {
	deposits, _ := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	beaconState.Slot = 4
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: []*pb.VoluntaryExit{
				{
					ValidatorIndex: 0,
					Epoch:          0,
				},
			},
		},
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessValidatorExits(context.Background(), beaconState, block, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessBlock(b *testing.B) {
	deposits, _ := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
		},
	}
	att1 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 5,
	}
	att2 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 4,
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRootHash32S = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + 10
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     params.BeaconConfig().GenesisSlot,
			JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			JustifiedBlockRootHash32: blockRoots[0],
			LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
			CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 10,
			Epoch:          params.BeaconConfig().GenesisEpoch,
		},
	}
	// randaoReveal := createRandaoReveal(b, beaconState, privKeys, params.BeaconConfig().GenesisSlot+10)
	block := &pb.BeaconBlock{
		Slot:         params.BeaconConfig().GenesisSlot + 10,
		RandaoReveal: []byte{1, 2, 3},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}

	cfg := &state.TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}
	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ProcessBlock(context.Background(), beaconState, block, cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func setupBenchmarkInitialDeposits(numDeposits int) ([]*pb.Deposit, []*bls.SecretKey) {
	setBenchmarkConfig()
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte(fmt.Sprintf("%d", i)),
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			panic(err)
		}
		deposits[i] = &pb.Deposit{
			DepositData: depositData,
		}
	}
	return deposits, nil
}

func createRandaoReveal(b *testing.B, beaconState *pb.BeaconState, privKeys []*bls.SecretKey, slot uint64) []byte {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState, beaconState.Slot)
	if err != nil {
		b.Fatal(err)
	}
	epoch := helpers.SlotToEpoch(slot)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal()
}
