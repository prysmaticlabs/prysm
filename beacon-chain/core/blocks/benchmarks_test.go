package blocks_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var ValidatorCount = 16000
var RunAmount = 134217728 / ValidatorCount
var QuickRunAmount = 100000
var conditions = "BIG"

var genesisState = createGenesisState(ValidatorCount)

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
	if conditions == "BIG" {
		c.MaxProposerSlashings = 16
		c.MaxAttesterSlashings = 1
		c.MaxAttestations = 128
		c.MaxDeposits = 16
		c.MaxVoluntaryExits = 16
	} else if conditions == "SML" {
		c.MaxAttesterSlashings = 0
		c.MaxProposerSlashings = 0
		c.MaxAttestations = 16
		c.MaxDeposits = 2
		c.MaxVoluntaryExits = 2
	}
	params.OverrideBeaconConfig(c)
}

func BenchmarkProcessBlockRandao(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	block := &pb.BeaconBlock{
		RandaoReveal: []byte{2, 3, 4},
	}

	b.N = QuickRunAmount
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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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

	b.N = QuickRunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = blocks.ProcessEth1DataInBlock(context.Background(), beaconState, block)
	}
}

func BenchmarkProcessProposerSlashings(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = currentSlot

	block, _ := createFullBlock(b, beaconState.Slot)

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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = currentSlot

	block, _ := createFullBlock(b, beaconState.Slot)

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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}

	beaconState.Slot = params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch + 3
	beaconState.PreviousJustifiedEpoch = params.BeaconConfig().GenesisEpoch

	block, _ := createFullBlock(b, beaconState.Slot)

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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot
	beaconState.Slot = currentSlot

	block, _ := createFullBlock(b, beaconState.Slot)

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessValidatorDeposits(
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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)
	beaconState.Slot = 4

	block, root := createFullBlock(b, beaconState.Slot)
	beaconState.LatestEth1Data = &pb.Eth1Data{
		BlockHash32:       root,
		DepositRootHash32: root,
	}

	b.N = QuickRunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.ProcessValidatorExits(context.Background(), beaconState, block, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessBlock(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch + 6
	beaconState.Slot = currentSlot
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}

	block, root := createFullBlock(b, beaconState.Slot)
	beaconState.LatestEth1Data = &pb.Eth1Data{
		BlockHash32:       root,
		DepositRootHash32: root,
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

func createFullBlock(b *testing.B, currentSlot uint64) (*pb.BeaconBlock, []byte) {
	proposerSlashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings; i++ {
		slashing := &pb.ProposerSlashing{
			ProposerIndex: i,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            currentSlot - 4,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            currentSlot - 4,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
		}
		proposerSlashings[i] = slashing
	}

	attesterSlashings := make([]*pb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
		att1 := &pb.AttestationData{
			Slot:           params.BeaconConfig().GenesisSlot + i,
			JustifiedEpoch: 2,
		}
		att2 := &pb.AttestationData{
			Slot:           params.BeaconConfig().GenesisSlot + i,
			JustifiedEpoch: 1,
		}

		offset := i * 8
		validatorIndices := make([]uint64, 8)
		for r := uint64(0); r < 8; r++ {
			validatorIndices[r] = offset + r
		}

		slashing := &pb.AttesterSlashing{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: validatorIndices,
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: validatorIndices,
				CustodyBitfield:  []byte{0xFF},
			},
		}
		attesterSlashings[i] = slashing
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	attestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations)
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations; i++ {
		att1 := &pb.Attestation{
			Data: &pb.AttestationData{
				Shard:                    0,
				Slot:                     currentSlot - 32,
				JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
				LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
				CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			},
			AggregationBitfield: bitutil.SetBitfield(int(i)),
			CustodyBitfield:     []byte{1},
		}
		attestations[i] = att1
	}

	voluntaryExits := make([]*pb.VoluntaryExit, params.BeaconConfig().MaxVoluntaryExits)
	for i := 0; i < len(voluntaryExits); i++ {
		voluntaryExits[i] = &pb.VoluntaryExit{
			ValidatorIndex: uint64(i + 64),
			Epoch:          helpers.SlotToEpoch(currentSlot),
		}
	}

	allData := make([][]byte, params.BeaconConfig().MaxDeposits)
	for i := 0; i < len(allData); i++ {
		pubkey := make([]byte, 32)
		binary.LittleEndian.PutUint64(pubkey, uint64(i))
		depositInput := &pb.DepositInput{
			Pubkey: pubkey,
		}
		wBuf := new(bytes.Buffer)
		if err := ssz.Encode(wBuf, depositInput); err != nil {
			b.Errorf("failed to encode deposit input: %v", err)
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
		allData[i] = data
	}

	newDeposits := make([]*pb.Deposit, len(allData))
	for i := 0; i < len(newDeposits); i++ {
		depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			b.Errorf("Could not generate trie: %v", err)
		}
		proof, err := depositTrie.MerkleProof(int(i))
		if err != nil {
			b.Errorf("Could not generate proof: %v", err)
		}

		newDeposits[i] = &pb.Deposit{
			DepositData:         allData[i],
			MerkleBranchHash32S: proof,
			MerkleTreeIndex:     uint64(i),
		}
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Errorf("Could not generate trie: %v", err)
	}
	root := depositTrie.Root()

	block := &pb.BeaconBlock{
		Slot:         currentSlot,
		RandaoReveal: []byte{1, 2, 3},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: root[:],
			BlockHash32:       root[:],
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    voluntaryExits,
			Deposits:          newDeposits,
		},
	}

	return block, root[:]
}

func createGenesisState(numDeposits int) *pb.BeaconState {
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
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		panic(err)
	}

	return genesisState
}
