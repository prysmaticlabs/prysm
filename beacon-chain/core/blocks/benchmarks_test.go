package blocks_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"math"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var quickRunAmount = 10000
var genesisState16K, deposits16K = createGenesisState(16000)
var genesisState300K, deposits300K = createGenesisState(300000)

// var genesisState4M = createGenesisState(4000000)

func setBenchmarkConfig(conditions string) {
	c := params.DemoBeaconConfig()
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
	block := &pb.BeaconBlock{
		RandaoReveal: []byte{2, 3, 4},
	}

	b.Run("16K", func(b *testing.B) {
		b.N = quickRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = blocks.ProcessBlockRandao(
				genesisState16K,
				block,
				false, /* verify signatures */
				false, /* disable logging */
			)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = quickRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = blocks.ProcessBlockRandao(
				genesisState300K,
				block,
				false, /* verify signatures */
				false, /* disable logging */
			)
		}
	})

	// b.Run("4M Validators", func(b *testing.B) {
	// 	b.N = quickRunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, _ = blocks.ProcessBlockRandao(
	// 			genesisState4M,
	// 			block,
	// 			false, /* verify signatures */
	// 			false, /* disable logging */
	// 		)
	// 	}
	// })
}

func BenchmarkProcessValidatorExits(b *testing.B) {
	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = quickRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessValidatorExits(genesisState16K, block, false)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	block, _ = createFullBlock(b, genesisState300K, deposits300K)

	b.Run("300K", func(b *testing.B) {
		b.N = quickRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessValidatorExits(genesisState300K, block, false)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessProposerSlashings(b *testing.B) {
	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = runAmount(16000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = blocks.ProcessProposerSlashings(
				genesisState16K,
				block,
				false,
			)
		}
	})

	block, _ = createFullBlock(b, genesisState300K, deposits300K)

	b.Run("300K", func(b *testing.B) {
		b.N = runAmount(300000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = blocks.ProcessProposerSlashings(
				genesisState300K,
				block,
				false,
			)
		}
	})
}

func BenchmarkProcessAttesterSlashings(b *testing.B) {
	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = 5
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessAttesterSlashings(
				genesisState16K,
				block,
				false,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	block, _ = createFullBlock(b, genesisState300K, deposits300K)

	b.Run("300K", func(b *testing.B) {
		b.N = 1
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessAttesterSlashings(
				genesisState300K,
				block,
				false,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessBlockAttestations(b *testing.B) {
	block, _ := createFullBlock(b, genesisState16K, deposits16K)

	b.Run("16K", func(b *testing.B) {
		b.N = runAmount(16000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessBlockAttestations(
				genesisState16K,
				block,
				false,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	block, _ = createFullBlock(b, genesisState300K, deposits300K)

	b.Run("300K", func(b *testing.B) {
		b.N = runAmount(300000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessBlockAttestations(
				genesisState300K,
				block,
				false,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessValidatorDeposits(b *testing.B) {
	block, root := createFullBlock(b, genesisState16K, deposits16K)
	genesisState16K.LatestEth1Data = &pb.Eth1Data{
		BlockHash32:       root,
		DepositRootHash32: root,
	}

	b.Run("16K", func(b *testing.B) {
		b.N = runAmount(16000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessValidatorDeposits(
				genesisState16K,
				block,
			)
			genesisState16K.DepositIndex = 16000
			if err != nil {
				b.Fatalf("Expected block deposits to process correctly, received: %v", err)
			}
		}
	})

	block, root = createFullBlock(b, genesisState300K, deposits300K)
	genesisState300K.LatestEth1Data = &pb.Eth1Data{
		BlockHash32:       root,
		DepositRootHash32: root,
	}

	b.Run("300K", func(b *testing.B) {
		b.N = runAmount(300000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := blocks.ProcessValidatorDeposits(
				genesisState300K,
				block,
			)
			genesisState300K.DepositIndex = 300000
			if err != nil {
				b.Fatalf("Expected block deposits to process correctly, received: %v", err)
			}
		}
	})
}

func BenchmarkProcessEth1Data(b *testing.B) {
	block, root := createFullBlock(b, genesisState16K, deposits16K)
	eth1DataVotes := []*pb.Eth1DataVote{
		{
			Eth1Data: &pb.Eth1Data{
				BlockHash32:       root,
				DepositRootHash32: root,
			},
			VoteCount: 5,
		},
	}

	genesisState16K.Eth1DataVotes = eth1DataVotes
	b.Run("16K", func(b *testing.B) {
		b.N = quickRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = blocks.ProcessEth1DataInBlock(genesisState16K, block)
		}
	})

	block, _ = createFullBlock(b, genesisState300K, deposits300K)
	eth1DataVotes = []*pb.Eth1DataVote{
		{
			Eth1Data: &pb.Eth1Data{
				BlockHash32:       root,
				DepositRootHash32: root,
			},
			VoteCount: 5,
		},
	}
	genesisState300K.Eth1DataVotes = eth1DataVotes
	b.Run("300K", func(b *testing.B) {
		b.N = quickRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = blocks.ProcessEth1DataInBlock(genesisState300K, block)
		}
	})

	// genesisState4M.Eth1DataVotes = eth1DataVotes
	// b.Run("4M Validators", func(b *testing.B) {
	// 	b.N = quickRunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_ = blocks.ProcessEth1DataInBlock(genesisState4M, block)
	// 	}
	// })
}

func BenchmarkProcessBlock(b *testing.B) {
	block, root := createFullBlock(b, genesisState16K, deposits16K)

	genesisState16K.LatestEth1Data = &pb.Eth1Data{
		BlockHash32:       root,
		DepositRootHash32: root,
	}

	cfg := &state.TransitionConfig{
		VerifySignatures: false,
		Logging:          false,
	}
	b.Run("16K", func(b *testing.B) {
		b.N = 1
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := state.ProcessBlock(context.Background(), genesisState16K, block, cfg); err != nil {
				b.Fatal(err)
			}
			genesisState16K.DepositIndex = 16000
		}
	})

	block, root = createFullBlock(b, genesisState300K, deposits300K)

	genesisState300K.LatestEth1Data = &pb.Eth1Data{
		BlockHash32:       root,
		DepositRootHash32: root,
	}

	b.Run("300K", func(b *testing.B) {
		b.N = 1
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := state.ProcessBlock(context.Background(), genesisState300K, block, cfg); err != nil {
				b.Fatal(err)
			}
			genesisState300K.DepositIndex = 300000
		}
	})
}

func createFullBlock(b *testing.B, bState *pb.BeaconState, previousDeposits []*pb.Deposit) (*pb.BeaconBlock, []byte) {
	currentSlot := bState.Slot
	currentEpoch := helpers.CurrentEpoch(bState)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	validatorIndices := helpers.ActiveValidatorIndices(bState.ValidatorRegistry, currentEpoch)
	validatorCount := len(validatorIndices)

	committeeSize := math.Ceil(float64(validatorCount) /
		float64(params.BeaconConfig().ShardCount) / float64(slotsPerEpoch))
	byteLength := mathutil.CeilDiv8(int(committeeSize))

	proposerSlashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings; i++ {
		slashing := &pb.ProposerSlashing{
			ProposerIndex: i,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            currentSlot - (i % slotsPerEpoch),
				Shard:           0,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            currentSlot - (i % slotsPerEpoch),
				Shard:           0,
				BlockRootHash32: []byte{0, 1, 0},
			},
		}
		proposerSlashings[i] = slashing
	}

	committeesPerEpoch := helpers.EpochCommitteeCount(uint64(validatorCount))
	if float64(validatorCount)/float64(committeesPerEpoch) > float64(params.BeaconConfig().MaxIndicesPerSlashableVote) {
		committeesPerEpoch = uint64(math.Ceil(float64(validatorCount) / float64(params.BeaconConfig().MaxIndicesPerSlashableVote)))
	}
	splitValidatorIndices := utils.SplitIndices(validatorIndices, committeesPerEpoch)

	maxSlashes := params.BeaconConfig().MaxAttesterSlashings
	attesterSlashings := make([]*pb.AttesterSlashing, maxSlashes)
	for i := uint64(0); i < maxSlashes; i++ {
		indices := splitValidatorIndices[i%maxSlashes]
		att1 := &pb.AttestationData{
			Slot:           currentSlot - (i % slotsPerEpoch),
			JustifiedEpoch: 2,
		}
		att2 := &pb.AttestationData{
			Slot:           currentSlot - (i % slotsPerEpoch),
			JustifiedEpoch: 1,
		}

		slashing := &pb.AttesterSlashing{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: indices,
				CustodyBitfield:  bitutil.FillBitfield(len(indices) - 1),
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: indices,
				CustodyBitfield:  bitutil.FillBitfield(len(indices) - 1),
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
				Slot:                     currentSlot - 1 - (i % (slotsPerEpoch - 1)),
				JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
				LatestCrosslink: &pb.Crosslink{
					Epoch:                   helpers.CurrentEpoch(bState),
					CrosslinkDataRootHash32: []byte{1},
				},
				CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:],
				JustifiedEpoch:          params.BeaconConfig().GenesisEpoch,
			},
			AggregationBitfield: bitutil.SetBitfield(int(i), byteLength),
			CustodyBitfield:     []byte{1},
		}
		attestations[i] = att1
	}

	voluntaryExits := make([]*pb.VoluntaryExit, params.BeaconConfig().MaxVoluntaryExits)
	for i := 0; i < len(voluntaryExits); i++ {
		voluntaryExits[i] = &pb.VoluntaryExit{
			Epoch:          currentEpoch - 1,
			ValidatorIndex: uint64(validatorCount/2 + i),
		}
	}

	previousDepsLen := uint64(len(previousDeposits))

	newData := make([][]byte, params.BeaconConfig().MaxDeposits)
	for i := 0; i < len(newData); i++ {
		pubkey := make([]byte, 32)
		binary.LittleEndian.PutUint64(pubkey, previousDepsLen+uint64(i))
		depositInput := &pb.DepositInput{
			Pubkey: pubkey,
		}

		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			panic(err)
		}

		newData[i] = depositData
	}

	allData := make([][]byte, previousDepsLen)
	for i := 0; i < int(previousDepsLen); i++ {
		allData[i] = previousDeposits[i].DepositData
	}
	allData = append(allData, newData...)

	depositTrie, err := trieutil.GenerateTrieFromItems(allData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Errorf("Could not generate trie: %v", err)
	}

	newDeposits := make([]*pb.Deposit, len(newData))
	for i := 0; i < len(newDeposits); i++ {
		proof, err := depositTrie.MerkleProof(int(previousDepsLen) + i)
		if err != nil {
			b.Errorf("Could not generate proof: %v", err)
		}

		newDeposits[i] = &pb.Deposit{
			DepositData:        newData[i],
			MerkleProofHash32S: proof,
			MerkleTreeIndex:    previousDepsLen + uint64(i),
		}
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

func createGenesisState(numDeposits int) (*pb.BeaconState, []*pb.Deposit) {
	setBenchmarkConfig("SML")
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
			DepositData:     depositData,
			MerkleTreeIndex: uint64(i),
		}
	}
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		panic(err)
	}

	genesisState.Slot = params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch - 1
	genesisState.LatestCrosslinks = []*pb.Crosslink{
		{
			Epoch:                   helpers.CurrentEpoch(genesisState),
			CrosslinkDataRootHash32: []byte{1},
		},
	}

	return genesisState, deposits
}

func runAmount(validatorCount int) int {
	// 33554432
	return 16777216 / validatorCount
}
